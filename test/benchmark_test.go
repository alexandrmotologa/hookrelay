package test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alexandrmotologa/hookrelay/internal/queue"
	"github.com/alexandrmotologa/hookrelay/internal/signer"
	"github.com/alexandrmotologa/hookrelay/internal/storage"
)

func TestHighThroughputDeliveryBenchmark(t *testing.T) {
	var receivedCount int64
	var verifiedSignatures int64

	endpointSecret := "whsec_benchtestsecret1234567890123456"

	// Mock webhook receiver
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&receivedCount, 1)

		msgID := r.Header.Get("webhook-id")
		timestamp := r.Header.Get("webhook-timestamp")
		sigHeader := r.Header.Get("webhook-signature")

		// Verify signature on receiver end
		body := []byte(`{"test":"high_throughput"}`)
		if err := signer.Verify(msgID, timestamp, body, sigHeader, endpointSecret, 300); err == nil {
			atomic.AddInt64(&verifiedSignatures, 1)
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"received":true}`))
	}))
	defer server.Close()

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "bench.db")
	store, err := storage.NewSQLiteStorage(dbPath)
	if err != nil {
		t.Fatalf("failed to init storage: %v", err)
	}
	defer store.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ep := &storage.Endpoint{
		Name:       "High-Throughput Destination",
		URL:        server.URL,
		Secret:     endpointSecret,
		EventTypes: []string{"*"},
	}
	if err := store.CreateEndpoint(ctx, ep); err != nil {
		t.Fatalf("failed to create endpoint: %v", err)
	}

	engineCfg := queue.DefaultEngineConfig()
	engineCfg.Workers = 30
	engineCfg.AllowPrivateIPs = true
	engine := queue.NewEngine(store, engineCfg)
	engine.Start(ctx)
	defer engine.Stop()

	// Measure baseline memory
	var memStatsBefore runtime.MemStats
	runtime.ReadMemStats(&memStatsBefore)

	const totalEvents = 200
	start := time.Now()

	// Ingest & Dispatch
	for i := 0; i < totalEvents; i++ {
		msg := &storage.Message{
			EventType: "order.placed",
			Payload:   `{"test":"high_throughput"}`,
			CreatedAt: time.Now().UTC(),
		}
		if err := store.CreateMessage(ctx, msg); err != nil {
			t.Fatalf("failed to create message: %v", err)
		}

		del := &storage.Delivery{
			MessageID:   msg.ID,
			EndpointID:  ep.ID,
			Status:      storage.DeliveryStatusPending,
			MaxAttempts: 5,
			CreatedAt:   time.Now().UTC(),
		}
		if err := store.CreateDelivery(ctx, del); err != nil {
			t.Fatalf("failed to create delivery: %v", err)
		}

		engine.Enqueue(del.ID)
	}

	// Wait until all deliveries have been processed by the worker pool
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if atomic.LoadInt64(&receivedCount) >= totalEvents {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	elapsed := time.Since(start)
	finalCount := atomic.LoadInt64(&receivedCount)
	validSigs := atomic.LoadInt64(&verifiedSignatures)

	if finalCount < totalEvents {
		t.Fatalf("expected %d deliveries, only processed %d within deadline", totalEvents, finalCount)
	}
	if validSigs != totalEvents {
		t.Fatalf("expected %d verified signatures, got %d", totalEvents, validSigs)
	}

	// Check memory consumption
	var memStatsAfter runtime.MemStats
	runtime.ReadMemStats(&memStatsAfter)
	allocatedMB := float64(memStatsAfter.Alloc) / (1024 * 1024)

	fmt.Printf("\n--- Benchmark Results ---\n")
	fmt.Printf("Events Dispatched:    %d\n", finalCount)
	fmt.Printf("Signatures Verified:  %d (100%%)\n", validSigs)
	fmt.Printf("Total Duration:       %v\n", elapsed)
	fmt.Printf("Throughput:           %.1f webhooks/sec\n", float64(finalCount)/elapsed.Seconds())
	fmt.Printf("Memory Allocated:     %.2f MB (Target: < 40MB)\n", allocatedMB)
	fmt.Printf("-------------------------\n")

	if allocatedMB > 40.0 {
		t.Errorf("memory allocated (%.2f MB) exceeds 40MB threshold", allocatedMB)
	}
}
