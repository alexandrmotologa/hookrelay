package queue

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/alexandrmotologa/hookrelay/internal/storage"
)

func TestRetryScheduler(t *testing.T) {
	now := time.Now().UTC()
	next1 := CalculateNextRetry(1)
	diff1 := next1.Sub(now)
	if diff1 < 4*time.Second || diff1 > 7*time.Second {
		t.Fatalf("expected attempt 1 delay around 5s, got %v", diff1)
	}

	next2 := CalculateNextRetry(2)
	diff2 := next2.Sub(now)
	if diff2 < 50*time.Second || diff2 > 70*time.Second {
		t.Fatalf("expected attempt 2 delay around 1m, got %v", diff2)
	}
}

func TestCircuitBreaker(t *testing.T) {
	cfg := CircuitBreakerConfig{
		FailureThreshold: 50,
		CooldownDuration: 10 * time.Minute,
	}

	ep := &storage.Endpoint{
		Status: storage.EndpointStatusActive,
	}

	allow, isProbe := CanAttempt(ep, cfg)
	if !allow || isProbe {
		t.Fatalf("expected active endpoint to allow normal attempt, got allow=%v, isProbe=%v", allow, isProbe)
	}

	// Tripped recently
	recent := time.Now().UTC().Add(-2 * time.Minute)
	ep.Status = storage.EndpointStatusSuspended
	ep.TrippedAt = &recent

	allow, isProbe = CanAttempt(ep, cfg)
	if allow || isProbe {
		t.Fatalf("expected suspended endpoint to block attempt, got allow=%v, isProbe=%v", allow, isProbe)
	}

	// Tripped past cooldown window
	past := time.Now().UTC().Add(-15 * time.Minute)
	ep.TrippedAt = &past

	allow, isProbe = CanAttempt(ep, cfg)
	if !allow || !isProbe {
		t.Fatalf("expected cooled suspended endpoint to allow probe, got allow=%v, isProbe=%v", allow, isProbe)
	}
}

func TestEngine_ProcessDelivery_Success(t *testing.T) {
	receivedSig := ""
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedSig = r.Header.Get("webhook-signature")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"received":true}`))
	}))
	defer ts.Close()

	tempDir := t.TempDir()
	store, err := storage.NewSQLiteStorage(filepath.Join(tempDir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create sqlite: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	ep := &storage.Endpoint{
		Name:   "Test Receiver",
		URL:    ts.URL,
		Secret: "whsec_testsecret12345",
	}
	if err := store.CreateEndpoint(ctx, ep); err != nil {
		t.Fatalf("failed to create endpoint: %v", err)
	}

	msg := &storage.Message{
		EventType: "order.created",
		Payload:   `{"order_id":"ord_100"}`,
	}
	if err := store.CreateMessage(ctx, msg); err != nil {
		t.Fatalf("failed to create message: %v", err)
	}

	del := &storage.Delivery{
		MessageID:  msg.ID,
		EndpointID: ep.ID,
		Status:     storage.DeliveryStatusPending,
	}
	if err := store.CreateDelivery(ctx, del); err != nil {
		t.Fatalf("failed to create delivery: %v", err)
	}

	cfg := DefaultEngineConfig()
	cfg.AllowPrivateIPs = true // Allow test server URL
	engine := NewEngine(store, cfg)

	engine.ProcessDelivery(ctx, del.ID)

	updated, err := store.GetDelivery(ctx, del.ID)
	if err != nil {
		t.Fatalf("failed to fetch updated delivery: %v", err)
	}

	if updated.Status != storage.DeliveryStatusSuccess {
		t.Fatalf("expected status SUCCESS, got %s (err: %s)", updated.Status, updated.ErrorMessage)
	}
	if updated.HTTPStatus == nil || *updated.HTTPStatus != 200 {
		t.Fatalf("expected HTTP 200, got %v", updated.HTTPStatus)
	}
	if receivedSig == "" {
		t.Fatalf("expected webhook-signature header on received request")
	}
}

func TestEngine_ProcessDelivery_RetryAndDLQ(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server error", http.StatusInternalServerError)
	}))
	defer ts.Close()

	tempDir := t.TempDir()
	store, _ := storage.NewSQLiteStorage(filepath.Join(tempDir, "test.db"))
	defer store.Close()

	ctx := context.Background()

	ep := &storage.Endpoint{
		Name:   "Failing Endpoint",
		URL:    ts.URL,
		Secret: "whsec_testsecret12345",
	}
	_ = store.CreateEndpoint(ctx, ep)

	msg := &storage.Message{
		EventType: "test.fail",
		Payload:   `{}`,
	}
	_ = store.CreateMessage(ctx, msg)

	del := &storage.Delivery{
		MessageID:   msg.ID,
		EndpointID:  ep.ID,
		Status:      storage.DeliveryStatusPending,
		MaxAttempts: 2,
	}
	_ = store.CreateDelivery(ctx, del)

	cfg := DefaultEngineConfig()
	cfg.AllowPrivateIPs = true
	engine := NewEngine(store, cfg)

	// Attempt 1 -> should be RETRY
	engine.ProcessDelivery(ctx, del.ID)
	d1, _ := store.GetDelivery(ctx, del.ID)
	if d1.Status != storage.DeliveryStatusRetry || d1.AttemptNumber != 1 {
		t.Fatalf("expected RETRY on attempt 1, got status=%s attempt=%d", d1.Status, d1.AttemptNumber)
	}
	if d1.NextRetryAt == nil {
		t.Fatalf("expected next_retry_at to be set on RETRY")
	}

	// Attempt 2 (MaxAttempts=2) -> should be FAILED (DLQ)
	engine.ProcessDelivery(ctx, del.ID)
	d2, _ := store.GetDelivery(ctx, del.ID)
	if d2.Status != storage.DeliveryStatusFailed || d2.AttemptNumber != 2 {
		t.Fatalf("expected FAILED on attempt 2, got status=%s attempt=%d", d2.Status, d2.AttemptNumber)
	}
}
