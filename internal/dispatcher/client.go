package dispatcher

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

type HardenedClient struct {
	client       *http.Client
	allowPrivate bool
}

type DeliveryRequest struct {
	EndpointURL string
	Headers     map[string]string
	Payload     []byte
}

type DeliveryResult struct {
	StatusCode          int
	DurationMs          int64
	RequestHeadersJSON  string
	ResponseHeadersJSON string
	ResponseBody        string
	ErrorMessage        string
	Success             bool
}

// NewHardenedClient creates an HTTP client with timeouts, anti-SSRF dialer, and disabled redirects.
func NewHardenedClient(timeout time.Duration, allowPrivate bool) *HardenedClient {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	dialer := &net.Dialer{
		Timeout:   3 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   2 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}

			if !allowPrivate {
				ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
				if err != nil {
					return nil, err
				}
				for _, ip := range ips {
					if IsPrivateIP(ip) {
						return nil, fmt.Errorf("%w: %s resolves to %s", ErrPrivateIPBlocked, host, ip.String())
					}
				}
			}

			return dialer.DialContext(ctx, network, net.JoinHostPort(host, port))
		},
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return errors.New("http redirects are disabled for webhook delivery")
		},
	}

	return &HardenedClient{
		client:       client,
		allowPrivate: allowPrivate,
	}
}

// Execute performs the HTTP delivery and records execution metrics.
func (c *HardenedClient) Execute(ctx context.Context, req *DeliveryRequest) (*DeliveryResult, error) {
	result := &DeliveryResult{}

	// Validate URL before sending
	if _, err := ValidateURL(req.EndpointURL, c.allowPrivate); err != nil {
		result.ErrorMessage = err.Error()
		return result, nil
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, req.EndpointURL, bytes.NewReader(req.Payload))
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to build request: %v", err)
		return result, nil
	}

	// Apply headers
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	reqHeadersBytes, _ := json.Marshal(httpReq.Header)
	result.RequestHeadersJSON = string(reqHeadersBytes)

	start := time.Now()
	resp, err := c.client.Do(httpReq)
	duration := time.Since(start)
	result.DurationMs = duration.Milliseconds()

	if err != nil {
		result.ErrorMessage = err.Error()
		return result, nil
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode

	// Extract response headers
	respHeadersMap := make(map[string]string)
	for k := range resp.Header {
		respHeadersMap[k] = resp.Header.Get(k)
	}
	respHeadersBytes, _ := json.Marshal(respHeadersMap)
	result.ResponseHeadersJSON = string(respHeadersBytes)

	// Read up to 4KB of response body
	const maxBodySnippet = 4096
	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, maxBodySnippet))
	result.ResponseBody = string(bodyBytes)

	// 2xx indicates delivery success
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		result.Success = true
	} else {
		result.ErrorMessage = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	return result, nil
}
