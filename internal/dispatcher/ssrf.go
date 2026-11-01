package dispatcher

import (
	"errors"
	"fmt"
	"net"
	"net/url"
)

var (
	ErrInvalidURLScheme = errors.New("only http and https URL schemes are allowed")
	ErrPrivateIPBlocked = errors.New("requests to private, loopback, or metadata addresses are prohibited")
	ErrEmptyHost        = errors.New("empty host in webhook destination URL")
)

var privateCIDRs []*net.IPNet

func init() {
	cidrs := []string{
		"127.0.0.0/8",    // IPv4 loopback
		"::1/128",        // IPv6 loopback
		"10.0.0.0/8",     // RFC1918 Class A
		"172.16.0.0/12",  // RFC1918 Class B
		"192.168.0.0/16", // RFC1918 Class C
		"169.254.0.0/16", // IPv4 Link-local & Cloud Metadata (AWS, GCP, Azure, DigitalOcean)
		"fe80::/10",      // IPv6 Link-local
		"fc00::/7",       // IPv6 Unique Local
		"0.0.0.0/8",      // Current network
		"224.0.0.0/4",    // Multicast
		"240.0.0.0/4",    // Reserved
		"100.64.0.0/10",  // Carrier-grade NAT
	}

	for _, cidr := range cidrs {
		_, block, err := net.ParseCIDR(cidr)
		if err == nil {
			privateCIDRs = append(privateCIDRs, block)
		}
	}
}

// IsPrivateIP checks if an IP belongs to loopback, link-local, or private blocks.
func IsPrivateIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsInterfaceLocalMulticast() {
		return true
	}

	for _, block := range privateCIDRs {
		if block.Contains(ip) {
			return true
		}
	}
	return false
}

// ValidateURL parses the URL and ensures it does not resolve to private or metadata addresses.
func ValidateURL(rawURL string, allowPrivate bool) (*url.URL, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, ErrInvalidURLScheme
	}

	host := parsed.Hostname()
	if host == "" {
		return nil, ErrEmptyHost
	}

	if allowPrivate {
		return parsed, nil
	}

	// Check if host is direct IP
	if ip := net.ParseIP(host); ip != nil {
		if IsPrivateIP(ip) {
			return nil, ErrPrivateIPBlocked
		}
		return parsed, nil
	}

	// Resolve DNS host
	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve destination host: %w", err)
	}

	if len(ips) == 0 {
		return nil, fmt.Errorf("no IP addresses found for host: %s", host)
	}

	for _, ip := range ips {
		if IsPrivateIP(ip) {
			return nil, fmt.Errorf("%w: %s resolves to %s", ErrPrivateIPBlocked, host, ip.String())
		}
	}

	return parsed, nil
}
