package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/alexandrmotologa/hookrelay/internal/queue"
	"github.com/alexandrmotologa/hookrelay/internal/storage"
)

func TestAPI_EndToEnd(t *testing.T) {
	// Destination webhook receiver
	deliveryCount := 0
	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deliveryCount++
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"received":true}`))
	}))
	defer targetServer.Close()

	// Storage & Queue
	tempDir := t.TempDir()
	store, err := storage.NewSQLiteStorage(filepath.Join(tempDir, "test.db"))
	if err != nil {
		t.Fatalf("failed to init storage: %v", err)
	}
	defer store.Close()

	cfg := queue.DefaultEngineConfig()
	cfg.AllowPrivateIPs = true
	engine := queue.NewEngine(store, cfg)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	engine.Start(ctx)
	defer engine.Stop()

	// Server Router
	router := BuildRouter(store, engine, true, "test_portal_secret")
	ts := httptest.NewServer(router)
	defer ts.Close()

	// 1. Health check
	resp, err := http.Get(ts.URL + "/api/v1/health")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("health check failed: %v", err)
	}

	// 2. Create Endpoint
	epBody := map[string]any{
		"name":        "Test Integration Endpoint",
		"url":         targetServer.URL,
		"event_types": []string{"payment.*"},
	}
	epJSON, _ := json.Marshal(epBody)
	resp, err = http.Post(ts.URL+"/api/v1/endpoints", "application/json", bytes.NewReader(epJSON))
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("create endpoint failed: %v, status: %d", err, resp.StatusCode)
	}
	var createdEp storage.Endpoint
	_ = json.NewDecoder(resp.Body).Decode(&createdEp)
	if createdEp.ID == "" || createdEp.Secret == "" {
		t.Fatalf("invalid created endpoint: %+v", createdEp)
	}

	// 3. Test Ping Endpoint
	resp, err = http.Post(ts.URL+"/api/v1/endpoints/"+createdEp.ID+"/test-ping", "application/json", nil)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("test ping failed: %v", err)
	}
	var pingDel storage.DeliveryWithDetails
	_ = json.NewDecoder(resp.Body).Decode(&pingDel)
	if pingDel.Status != storage.DeliveryStatusSuccess {
		t.Fatalf("expected ping delivery success, got %s", pingDel.Status)
	}

	// 4. Publish Event
	pubBody := map[string]any{
		"event_type": "payment.succeeded",
		"payload": map[string]any{
			"order_id": "ord_9901",
			"amount":   5400,
		},
	}
	pubJSON, _ := json.Marshal(pubBody)
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/publish", bytes.NewReader(pubJSON))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "idem_key_12345")

	resp, err = http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusAccepted {
		t.Fatalf("publish failed: %v, status: %d", err, resp.StatusCode)
	}
	var pubResp map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&pubResp)
	if pubResp["deliveries_queued"].(float64) != 1 {
		t.Fatalf("expected 1 delivery queued, got %v", pubResp["deliveries_queued"])
	}

	// 5. Test Idempotency Key deduplication
	req2, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/publish", bytes.NewReader(pubJSON))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Idempotency-Key", "idem_key_12345")
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil || resp2.StatusCode != http.StatusAccepted {
		t.Fatalf("duplicate publish failed: %v", err)
	}
	var pubResp2 map[string]any
	_ = json.NewDecoder(resp2.Body).Decode(&pubResp2)
	if pubResp2["deduplicated"] != true {
		t.Fatalf("expected deduplicated=true, got %v", pubResp2["deduplicated"])
	}

	// 6. Rotate Secret
	resp, err = http.Post(ts.URL+"/api/v1/endpoints/"+createdEp.ID+"/rotate-secret", "application/json", nil)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("rotate secret failed: %v", err)
	}
	var rotResp map[string]string
	_ = json.NewDecoder(resp.Body).Decode(&rotResp)
	if rotResp["secret"] == "" || rotResp["secret"] == createdEp.Secret {
		t.Fatalf("secret did not change: %s", rotResp["secret"])
	}

	// 7. Portal Token and Portal View
	tokenReqBody, _ := json.Marshal(map[string]any{
		"endpoint_id":      createdEp.ID,
		"expires_in_hours": 24,
	})
	resp, err = http.Post(ts.URL+"/api/v1/portal/token", "application/json", bytes.NewReader(tokenReqBody))
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("portal token generation failed: %v", err)
	}
	var tokenResp map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&tokenResp)
	portalToken := tokenResp["token"].(string)

	resp, err = http.Get(ts.URL + "/api/v1/portal/view?token=" + portalToken)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("portal view failed: %v", err)
	}
	var portalView map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&portalView)
	if portalView["endpoint"] == nil {
		t.Fatalf("expected endpoint object in portal view")
	}

	// 8. Stats check
	resp, err = http.Get(ts.URL + "/api/v1/stats")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("stats failed: %v", err)
	}
	var stats storage.StatsSummary
	_ = json.NewDecoder(resp.Body).Decode(&stats)
	if stats.TotalMessages < 2 {
		t.Fatalf("expected at least 2 messages in stats, got %d", stats.TotalMessages)
	}
}
