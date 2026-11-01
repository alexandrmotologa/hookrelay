package api

import (
	"encoding/json"
	"io"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/alexandrmotologa/hookrelay/internal/signer"
)

type MockReceivedItem struct {
	ID             string            `json:"id"`
	ReceivedAt     time.Time         `json:"received_at"`
	Method         string            `json:"method"`
	Headers        map[string]string `json:"headers"`
	Body           string            `json:"body"`
	WebhookID      string            `json:"webhook_id"`
	WebhookTimestamp string          `json:"webhook_timestamp"`
	WebhookSignature string          `json:"webhook_signature"`
	SignatureValid bool              `json:"signature_valid"`
	SimulatedMode  string            `json:"simulated_mode"`
	StatusCode     int               `json:"status_code"`
}

type MockReceiver struct {
	mu      sync.RWMutex
	history []MockReceivedItem
	maxSize int
}

func NewMockReceiver() *MockReceiver {
	return &MockReceiver{
		history: make([]MockReceivedItem, 0, 50),
		maxSize: 50,
	}
}

func (m *MockReceiver) HandleReceive(w http.ResponseWriter, r *http.Request) {
	mode := r.URL.Query().Get("mode")
	if mode == "" {
		mode = r.Header.Get("X-Mock-Mode")
	}
	if mode == "" {
		mode = "success"
	}

	bodyBytes, _ := io.ReadAll(io.LimitReader(r.Body, 1024*1024))
	defer r.Body.Close()

	headersMap := make(map[string]string)
	for k := range r.Header {
		headersMap[k] = r.Header.Get(k)
	}

	msgID := r.Header.Get("webhook-id")
	timestamp := r.Header.Get("webhook-timestamp")
	sigHeader := r.Header.Get("webhook-signature")

	// Validate signature if secret is provided in query param or header
	secret := r.URL.Query().Get("secret")
	sigValid := false
	if secret != "" && msgID != "" && timestamp != "" && sigHeader != "" {
		if err := signer.Verify(msgID, timestamp, bodyBytes, sigHeader, secret, 300); err == nil {
			sigValid = true
		}
	}

	if mode == "chaos" {
		roll := rand.Float64()
		if roll < 0.50 {
			mode = "success"
		} else if roll < 0.75 {
			mode = "error"
		} else if roll < 0.90 {
			mode = "rate_limit"
		} else {
			mode = "timeout"
		}
	}

	statusCode := http.StatusOK
	respPayload := map[string]any{
		"mock":             true,
		"mode":             mode,
		"received_at":      time.Now().UTC().Format(time.RFC3339),
		"signature_valid":  sigValid,
	}

	switch mode {
	case "error":
		statusCode = http.StatusInternalServerError
		respPayload["error"] = "Simulated 500 Internal Server Error (Chaos)"
	case "rate_limit":
		statusCode = http.StatusTooManyRequests
		w.Header().Set("Retry-After", "10")
		respPayload["error"] = "Simulated 429 Too Many Requests (Retry-After: 10s)"
	case "timeout":
		time.Sleep(6 * time.Second) // HookRelay client times out after 5s
		statusCode = http.StatusGatewayTimeout
		respPayload["error"] = "Simulated 504 Gateway Timeout"
	default:
		statusCode = http.StatusOK
		respPayload["received"] = true
	}

	item := MockReceivedItem{
		ID:               msgID,
		ReceivedAt:       time.Now().UTC(),
		Method:           r.Method,
		Headers:          headersMap,
		Body:             string(bodyBytes),
		WebhookID:        msgID,
		WebhookTimestamp: timestamp,
		WebhookSignature: sigHeader,
		SignatureValid:   sigValid,
		SimulatedMode:    mode,
		StatusCode:       statusCode,
	}

	m.mu.Lock()
	m.history = append([]MockReceivedItem{item}, m.history...)
	if len(m.history) > m.maxSize {
		m.history = m.history[:m.maxSize]
	}
	m.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(respPayload)
}

func (m *MockReceiver) HandleHistory(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	writeJSON(w, http.StatusOK, m.history)
}

func (m *MockReceiver) HandleClear(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	m.history = make([]MockReceivedItem, 0, 50)
	m.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]string{"status": "cleared"})
}
