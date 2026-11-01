package queue

import (
	"context"
	"fmt"
	"log"
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
}

func DefaultEngineConfig() EngineConfig {
	return EngineConfig{
		Workers:         20,
		QueueBufferSize: 5000,
		PollInterval:    1 * time.Second,
		CircuitBreaker:  DefaultCircuitBreakerConfig(),
		DeliveryTimeout: 5 * time.Second,
		AllowPrivateIPs: false,
	}
}

type Engine struct {
	store     storage.Storage
	client    *dispatcher.HardenedClient
	cfg       EngineConfig
	taskQueue chan string
	stopChan  chan struct{}
	wg        sync.WaitGroup
}

// NewEngine creates a new webhook execution engine.
func NewEngine(store storage.Storage, cfg EngineConfig) *Engine {
	client := dispatcher.NewHardenedClient(cfg.DeliveryTimeout, cfg.AllowPrivateIPs)
	return &Engine{
		store:     store,
		client:    client,
		cfg:       cfg,
		taskQueue: make(chan string, cfg.QueueBufferSize),
		stopChan:  make(chan struct{}),
	}
}

// Start launches worker goroutines and the due retry scheduler loop.
func (e *Engine) Start(ctx context.Context) {
	// 1. Launch concurrent workers
	for i := 0; i < e.cfg.Workers; i++ {
		e.wg.Add(1)
		go e.workerLoop(ctx)
	}

	// 2. Launch scheduler loop for due retries & initial pending tasks
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
		// Queue full, scheduler loop will pick it up from database
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
			// Poll due retries
			dueRetries, err := e.store.GetDueRetries(ctx, 50)
			if err == nil {
				for _, d := range dueRetries {
					e.Enqueue(d.ID)
				}
			}

			// Also poll any unhandled pending deliveries
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

	// Only process PENDING or RETRY states
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
		return
	}

	// 2. Mark in-flight
	del.Status = storage.DeliveryStatusInFlight
	_ = e.store.UpdateDeliveryAttempt(ctx, &del.Delivery)

	// 3. Generate Standard Webhooks HMAC-SHA256 headers
	now := time.Now().UTC()
	headers, err := signer.BuildHeaders(del.MessageID, now, []byte(del.Payload), ep.Secret)
	if err != nil {
		del.Status = storage.DeliveryStatusFailed
		del.ErrorMessage = fmt.Sprintf("Signing error: %v", err)
		_ = e.store.UpdateDeliveryAttempt(ctx, &del.Delivery)
		return
	}

	// 4. Dispatch outbound HTTP POST
	req := &dispatcher.DeliveryRequest{
		EndpointURL: ep.URL,
		Headers:     headers,
		Payload:     []byte(del.Payload),
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
		// Delivery succeeded
		del.Status = storage.DeliveryStatusSuccess
		del.NextRetryAt = nil
		_ = e.store.UpdateDeliveryAttempt(ctx, &del.Delivery)

		// Reset failures / close circuit breaker if healthy or probe passed
		_ = e.store.ResetEndpointFailure(ctx, ep.ID)
	} else {
		// Delivery failed
		if del.AttemptNumber < del.MaxAttempts {
			del.Status = storage.DeliveryStatusRetry
			nextRetry := CalculateNextRetry(del.AttemptNumber)
			del.NextRetryAt = &nextRetry
		} else {
			// All retry attempts exhausted -> move to Dead Letter Queue
			del.Status = storage.DeliveryStatusFailed
			del.NextRetryAt = nil
		}
		_ = e.store.UpdateDeliveryAttempt(ctx, &del.Delivery)

		// Record endpoint failure for circuit breaker
		if !isProbe {
			_, _ = e.store.IncrementEndpointFailure(ctx, ep.ID, e.cfg.CircuitBreaker.FailureThreshold)
		} else {
			// Probe failed: re-trip breaker timer
			trippedNow := time.Now().UTC()
			ep.TrippedAt = &trippedNow
			_ = e.store.UpdateEndpoint(ctx, ep)
		}
	}
}
