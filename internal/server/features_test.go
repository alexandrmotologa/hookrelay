package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alexandrmotologa/hookrelay/internal/queue"
	"github.com/alexandrmotologa/hookrelay/internal/signer"
	"github.com/alexandrmotologa/hookrelay/internal/storage"
)

func TestNewFeatures_EndToEnd(t *testing.T) {
	tempDir := t.TempDir()
	store, err := storage.NewSQLiteStorage(filepath.Join(tempDir, "features.db"))
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

	router := BuildRouter(store, engine, true, "portal_secret")
	ts := httptest.NewServer(router)
	defer ts.Close()

	// 1. Test Built-in Mock Receiver (Success mode)
	mockReqBody := []byte(`{"event":"test.mock"}`)
	resp, err := http.Post(ts.URL+"/api/v1/mock-receiver?mode=success", "application/json", bytes.NewReader(mockReqBody))
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("mock receiver success failed: %v, status: %d", err, resp.StatusCode)
	}
	var mockResp map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&mockResp)
	if mockResp["received"] != true {
		t.Fatalf("expected received=true from mock receiver")
	}

	// 2. Test Built-in Mock Receiver (Rate Limit mode with Retry-After header)
	resp429, err := http.Post(ts.URL+"/api/v1/mock-receiver?mode=rate_limit", "application/json", bytes.NewReader(mockReqBody))
	if err != nil || resp429.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("expected 429 from mock receiver, got %d", resp429.StatusCode)
	}
	if resp429.Header.Get("Retry-After") != "10" {
		t.Fatalf("expected Retry-After: 10 header, got: %s", resp429.Header.Get("Retry-After"))
	}

	// 3. Test Signature Debugger API
	secret, _ := signer.GenerateSecret()
	msgID := "msg_debug_123"
	tsVal := "1730498400"
	payload := `{"test":"val"}`
	sigHeader, _ := signer.Sign(msgID, 1730498400, []byte(payload), secret)

	debugReq := map[string]any{
		"message_id":       msgID,
		"timestamp":        tsVal,
		"payload":          payload,
		"secret":           secret,
		"signature_header": sigHeader,
	}
	debugJSON, _ := json.Marshal(debugReq)
	debugResp, err := http.Post(ts.URL+"/api/v1/verifier/debug", "application/json", bytes.NewReader(debugJSON))
	if err != nil || debugResp.StatusCode != http.StatusOK {
		t.Fatalf("debugger request failed: %v", err)
	}
	var debugResult map[string]any
	_ = json.NewDecoder(debugResp.Body).Decode(&debugResult)
	if debugResult["valid"] != true {
		t.Fatalf("expected debugger result valid=true, got: %+v", debugResult)
	}

	// 4. Test Endpoint with Custom Headers
	receivedAuthHeader := ""
	customTarget := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuthHeader = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"received":true}`))
	}))
	defer customTarget.Close()

	epBody := map[string]any{
		"name": "Custom Header Endpoint",
		"url":  customTarget.URL,
		"custom_headers": map[string]string{
			"Authorization": "Bearer secret_api_token_123",
		},
	}
	epJSON, _ := json.Marshal(epBody)
	resp, _ = http.Post(ts.URL+"/api/v1/endpoints", "application/json", bytes.NewReader(epJSON))
	var createdEp storage.Endpoint
	_ = json.NewDecoder(resp.Body).Decode(&createdEp)

	// Publish event to test custom header delivery
	pubBody := map[string]any{
		"event_type":  "custom.header.test",
		"endpoint_id": createdEp.ID,
		"payload":     map[string]any{"ok": true},
	}
	pubJSON, _ := json.Marshal(pubBody)
	_, _ = http.Post(ts.URL+"/api/v1/publish", "application/json", bytes.NewReader(pubJSON))

	// Allow worker to dispatch
	time.Sleep(100 * time.Millisecond)
	if receivedAuthHeader != "Bearer secret_api_token_123" {
		t.Fatalf("expected custom Authorization header, got: %s", receivedAuthHeader)
	}

	// 5. Test Export API (CSV and JSON)
	csvResp, err := http.Get(ts.URL + "/api/v1/deliveries/export?format=csv")
	if err != nil || csvResp.StatusCode != http.StatusOK {
		t.Fatalf("csv export failed: %v", err)
	}
	csvBody, _ := io.ReadAll(csvResp.Body)
	if !strings.Contains(string(csvBody), "delivery_id,message_id") {
		t.Fatalf("expected csv header in export, got: %s", string(csvBody))
	}

	jsonResp, err := http.Get(ts.URL + "/api/v1/deliveries/export?format=json")
	if err != nil || jsonResp.StatusCode != http.StatusOK {
		t.Fatalf("json export failed: %v", err)
	}
	var exportedList []any
	_ = json.NewDecoder(jsonResp.Body).Decode(&exportedList)
	if len(exportedList) == 0 {
		t.Fatalf("expected exported deliveries in JSON list")
	}
}
