package queue

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/alexandrmotologa/hookrelay/internal/dispatcher"
	"github.com/alexandrmotologa/hookrelay/internal/signer"
	"github.com/alexandrmotologa/hookrelay/internal/storage"
)

type EngineConfig struct {
	Workers          int
	QueueBufferSize  int
	PollInterval     time.Duration
	CircuitBreaker   CircuitBreakerConfig
	DeliveryTimeout  time.Duration
	AllowPrivateIPs  bool
	AlertWebhookURL  string
}

func DefaultEngineConfig() EngineConfig {
	return EngineConfig{
		Workers:         20,
		QueueBufferSize: 5000,
		PollInterval:    1 * time.Second,
		CircuitBreaker:  DefaultCircuitBreakerConfig(),
		DeliveryTimeout: 5 * time.Second,
		AllowPrivateIPs: false,
		AlertWebhookURL: "",
	}
}

type DeliveryUpdateListener func(d *storage.DeliveryWithDetails)

type Engine struct {
	store        storage.Storage
	client       *dispatcher.HardenedClient
	rateLimiter  *EndpointRateLimiter
	cfg          EngineConfig
	taskQueue    chan string
	stopChan     chan struct{}
	listeners    []DeliveryUpdateListener
	listenersMu  sync.RWMutex
	wg           sync.WaitGroup
}

// NewEngine creates a new webhook execution engine.
func NewEngine(store storage.Storage, cfg EngineConfig) *Engine {
	client := dispatcher.NewHardenedClient(cfg.DeliveryTimeout, cfg.AllowPrivateIPs)
	return &Engine{
		store:       store,
		client:      client,
		rateLimiter: NewEndpointRateLimiter(),
		cfg:         cfg,
		taskQueue:   make(chan string, cfg.QueueBufferSize),
		stopChan:    make(chan struct{}),
	}
}

// SubscribeDeliveryUpdates registers a callback for SSE live streaming.
func (e *Engine) SubscribeDeliveryUpdates(listener DeliveryUpdateListener) {
	e.listenersMu.Lock()
	defer e.listenersMu.Unlock()
	e.listeners = append(e.listeners, listener)
}

func (e *Engine) notifyListeners(d *storage.DeliveryWithDetails) {
	e.listenersMu.RLock()
	defer e.listenersMu.RUnlock()
	for _, l := range e.listeners {
		l(d)
	}
}

// Start launches worker goroutines and the due retry scheduler loop.
func (e *Engine) Start(ctx context.Context) {
	for i := 0; i < e.cfg.Workers; i++ {
		e.wg.Add(1)
		go e.workerLoop(ctx)
	}

	e.wg.Add(1)
	go e.schedulerLoop(ctx)
}

// Stop gracefully shuts down all workers and the scheduler.
func (e *Engine) Stop() {
	close(e.stopChan)
	e.wg.Wait()
}

// Enqueue adds a delivery ID to the intake buffer for immediate dispatch.
func (e *Engine) Enqueue(deliveryID string) {
	select {
	case e.taskQueue <- deliveryID:
	default:
		log.Printf("[HookRelay] Warning: in-memory queue buffer full, relying on storage poller")
	}
}

func (e *Engine) workerLoop(ctx context.Context) {
	defer e.wg.Done()

	for {
		select {
		case <-e.stopChan:
			return
		case deliveryID, ok := <-e.taskQueue:
			if !ok {
				return
			}
			e.ProcessDelivery(ctx, deliveryID)
		}
	}
}

func (e *Engine) schedulerLoop(ctx context.Context) {
	defer e.wg.Done()
	ticker := time.NewTicker(e.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-e.stopChan:
			return
		case <-ticker.C:
			dueRetries, err := e.store.GetDueRetries(ctx, 50)
			if err == nil {
				for _, d := range dueRetries {
					e.Enqueue(d.ID)
				}
			}

			pending, err := e.store.GetPendingDeliveries(ctx, 50)
			if err == nil {
				for _, d := range pending {
					e.Enqueue(d.ID)
				}
			}
		}
	}
}

// ProcessDelivery executes a single delivery attempt end-to-end.
func (e *Engine) ProcessDelivery(ctx context.Context, deliveryID string) {
	del, err := e.store.GetDeliveryWithDetails(ctx, deliveryID)
	if err != nil || del == nil {
		return
	}

	if del.Status != storage.DeliveryStatusPending && del.Status != storage.DeliveryStatusRetry {
		return
	}

	ep, err := e.store.GetEndpoint(ctx, del.EndpointID)
	if err != nil || ep == nil {
		return
	}

	// 1. Check Circuit Breaker
	allow, isProbe := CanAttempt(ep, e.cfg.CircuitBreaker)
	if !allow {
		errMsg := "Delivery paused: circuit breaker open due to consecutive failures"
		del.Status = storage.DeliveryStatusRetry
		del.ErrorMessage = errMsg
		nextRetry := time.Now().UTC().Add(10 * time.Minute)
		del.NextRetryAt = &nextRetry
		_ = e.store.UpdateDeliveryAttempt(ctx, &del.Delivery)
		e.notifyListeners(del)
		return
	}

	// 2. Active Rate Limiting per Endpoint (Token Bucket)
	if err := e.rateLimiter.Wait(ctx, ep.ID, ep.RateLimit); err != nil {
		return
	}

	// 3. Mark in-flight
	del.Status = storage.DeliveryStatusInFlight
	_ = e.store.UpdateDeliveryAttempt(ctx, &del.Delivery)
	e.notifyListeners(del)

	// 4. Generate Standard Webhooks HMAC-SHA256 headers (supports dual signature during grace period)
	now := time.Now().UTC()
	headers, err := signer.BuildDualHeaders(
		del.MessageID,
		now,
		[]byte(del.Payload),
		ep.Secret,
		ep.PreviousSecret,
		ep.SecretRotatedAt,
		24*time.Hour,
	)
	if err != nil {
		del.Status = storage.DeliveryStatusFailed
		del.ErrorMessage = fmt.Sprintf("Signing error: %v", err)
		_ = e.store.UpdateDeliveryAttempt(ctx, &del.Delivery)
		e.notifyListeners(del)
		return
	}

	// 5. Dispatch outbound HTTP POST with custom headers
	req := &dispatcher.DeliveryRequest{
		EndpointURL:   ep.URL,
		Headers:       headers,
		CustomHeaders: ep.CustomHeaders,
		Payload:       []byte(del.Payload),
	}

	res, _ := e.client.Execute(ctx, req)
	sentAt := time.Now().UTC()
	del.SentAt = &sentAt
	del.AttemptNumber++
	del.HTTPStatus = &res.StatusCode
	del.DurationMs = &res.DurationMs
	del.RequestHeaders = res.RequestHeadersJSON
	del.ResponseHeaders = res.ResponseHeadersJSON
	del.ResponseBody = res.ResponseBody
	del.ErrorMessage = res.ErrorMessage

	if res.Success {
		del.Status = storage.DeliveryStatusSuccess
		del.NextRetryAt = nil
		_ = e.store.UpdateDeliveryAttempt(ctx, &del.Delivery)
		_ = e.store.ResetEndpointFailure(ctx, ep.ID)
	} else {
		if del.AttemptNumber < del.MaxAttempts {
			del.Status = storage.DeliveryStatusRetry

			// Check Retry-After header on 429 Too Many Requests
			var nextRetry time.Time
			if res.StatusCode == http.StatusTooManyRequests {
				if retryAfterDur, ok := ParseRetryAfter(res.RetryAfterHeader); ok {
					nextRetry = time.Now().UTC().Add(retryAfterDur)
				}
			}
			if nextRetry.IsZero() {
				nextRetry = CalculateNextRetry(del.AttemptNumber)
			}
			del.NextRetryAt = &nextRetry
		} else {
			del.Status = storage.DeliveryStatusFailed
			del.NextRetryAt = nil
		}
		_ = e.store.UpdateDeliveryAttempt(ctx, &del.Delivery)

		// Record endpoint failure for circuit breaker
		if !isProbe {
			tripped, _ := e.store.IncrementEndpointFailure(ctx, ep.ID, e.cfg.CircuitBreaker.FailureThreshold)
			if tripped {
				e.dispatchSystemAlert(ep)
			}
		} else {
			trippedNow := time.Now().UTC()
			ep.TrippedAt = &trippedNow
			_ = e.store.UpdateEndpoint(ctx, ep)
		}
	}

	e.notifyListeners(del)
}

func (e *Engine) dispatchSystemAlert(ep *storage.Endpoint) {
	if e.cfg.AlertWebhookURL == "" {
		return
	}

	go func() {
		payload := map[string]any{
			"event":                "circuit_breaker.tripped",
			"endpoint_id":          ep.ID,
			"endpoint_name":        ep.Name,
			"endpoint_url":         ep.URL,
			"consecutive_failures": ep.ConsecutiveFailures,
			"tripped_at":           time.Now().UTC().Format(time.RFC3339),
		}
		jsonBytes, err := json.Marshal(payload)
		if err != nil {
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.cfg.AlertWebhookURL, bytes.NewReader(jsonBytes))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "HookRelay-Alerts/1.0")

		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			_ = resp.Body.Close()
		}
	}()
}
