package storage

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestSQLiteStorage_CRUD(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_hookrelay.db")

	store, err := NewSQLiteStorage(dbPath)
	if err != nil {
		t.Fatalf("failed to initialize sqlite: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// 1. Create Endpoint
	ep := &Endpoint{
		Name:       "Test Endpoint",
		URL:        "https://example.com/webhook",
		Secret:     "whsec_testsecret1234567890",
		EventTypes: []string{"invoice.*", "user.signup"},
		RateLimit:  50,
	}
	if err := store.CreateEndpoint(ctx, ep); err != nil {
		t.Fatalf("failed to create endpoint: %v", err)
	}
	if ep.ID == "" {
		t.Fatalf("expected generated endpoint ID, got empty")
	}

	// 2. Fetch Endpoint
	fetched, err := store.GetEndpoint(ctx, ep.ID)
	if err != nil {
		t.Fatalf("failed to get endpoint: %v", err)
	}
	if fetched.Name != ep.Name || fetched.URL != ep.URL {
		t.Fatalf("mismatched endpoint data: %+v", fetched)
	}
	if !fetched.MatchesEvent("invoice.paid") {
		t.Fatalf("expected wildcard match for invoice.paid")
	}
	if fetched.MatchesEvent("order.created") {
		t.Fatalf("expected no match for order.created")
	}

	// 3. Create Message
	msg := &Message{
		EventType: "invoice.paid",
		Payload:   `{"amount": 100}`,
	}
	if err := store.CreateMessage(ctx, msg); err != nil {
		t.Fatalf("failed to create message: %v", err)
	}

	// 4. Create Delivery
	del := &Delivery{
		MessageID:  msg.ID,
		EndpointID: ep.ID,
		Status:     DeliveryStatusPending,
	}
	if err := store.CreateDelivery(ctx, del); err != nil {
		t.Fatalf("failed to create delivery: %v", err)
	}

	// 5. Update Delivery
	httpStatus := 200
	duration := int64(45)
	now := time.Now().UTC()
	del.Status = DeliveryStatusSuccess
	del.AttemptNumber = 1
	del.HTTPStatus = &httpStatus
	del.DurationMs = &duration
	del.SentAt = &now
	del.ResponseBody = `{"ok":true}`

	if err := store.UpdateDeliveryAttempt(ctx, del); err != nil {
		t.Fatalf("failed to update delivery attempt: %v", err)
	}

	// 6. Check Delivery with details
	detail, err := store.GetDeliveryWithDetails(ctx, del.ID)
	if err != nil {
		t.Fatalf("failed to get delivery with details: %v", err)
	}
	if detail.Status != DeliveryStatusSuccess || detail.EndpointName != ep.Name {
		t.Fatalf("unexpected detail: %+v", detail)
	}

	// 7. Verify stats
	stats, err := store.GetStats(ctx)
	if err != nil {
		t.Fatalf("failed to get stats: %v", err)
	}
	if stats.TotalMessages != 1 || stats.SuccessfulCount != 1 || stats.SuccessRate != 100.0 {
		t.Fatalf("unexpected stats: %+v", stats)
	}

	// 8. Circuit Breaker test
	for i := 0; i < 49; i++ {
		tripped, err := store.IncrementEndpointFailure(ctx, ep.ID, 50)
		if err != nil || tripped {
			t.Fatalf("unexpected failure increment at %d: %v, tripped: %v", i, err, tripped)
		}
	}
	// 50th failure should trip
	tripped, err := store.IncrementEndpointFailure(ctx, ep.ID, 50)
	if err != nil || !tripped {
		t.Fatalf("expected trip at 50, got err=%v, tripped=%v", err, tripped)
	}
	epPostTrip, _ := store.GetEndpoint(ctx, ep.ID)
	if epPostTrip.Status != EndpointStatusSuspended {
		t.Fatalf("expected status SUSPENDED, got %s", epPostTrip.Status)
	}

	// Reset circuit breaker
	if err := store.ResetEndpointFailure(ctx, ep.ID); err != nil {
		t.Fatalf("failed to reset circuit breaker: %v", err)
	}
	epPostReset, _ := store.GetEndpoint(ctx, ep.ID)
	if epPostReset.Status != EndpointStatusActive || epPostReset.ConsecutiveFailures != 0 {
		t.Fatalf("expected status ACTIVE and 0 failures, got %s (%d)", epPostReset.Status, epPostReset.ConsecutiveFailures)
	}
}
