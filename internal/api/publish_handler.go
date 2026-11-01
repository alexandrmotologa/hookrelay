package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/alexandrmotologa/hookrelay/internal/queue"
	"github.com/alexandrmotologa/hookrelay/internal/storage"
)

type PublishHandler struct {
	store  storage.Storage
	engine *queue.Engine
}

func NewPublishHandler(store storage.Storage, engine *queue.Engine) *PublishHandler {
	return &PublishHandler{
		store:  store,
		engine: engine,
	}
}

type PublishRequest struct {
	EventType  string          `json:"event_type"`
	Payload    json.RawMessage `json:"payload"`
	EndpointID string          `json:"endpoint_id,omitempty"`
}

type PublishResponse struct {
	MessageID        string    `json:"message_id"`
	EventType        string    `json:"event_type"`
	DeliveriesQueued int       `json:"deliveries_queued"`
	Deduplicated     bool      `json:"deduplicated,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}

func (h *PublishHandler) HandlePublish(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Limit request body to 512KB
	const maxPayloadBytes = 512 * 1024
	body, err := io.ReadAll(io.LimitReader(r.Body, maxPayloadBytes+1))
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read request body")
		return
	}
	if len(body) > maxPayloadBytes {
		writeError(w, http.StatusRequestEntityTooLarge, "payload exceeds maximum limit of 512KB")
		return
	}

	var req PublishRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid JSON payload: %v", err))
		return
	}

	req.EventType = strings.TrimSpace(req.EventType)
	if req.EventType == "" {
		writeError(w, http.StatusBadRequest, "missing required field: event_type")
		return
	}
	if len(req.Payload) == 0 {
		req.Payload = json.RawMessage(`{}`)
	}

	// 1. Idempotency Key check
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idempotencyKey != "" {
		existing, err := h.store.GetMessageByIdempotencyKey(ctx, idempotencyKey)
		if err == nil && existing != nil {
			// Idempotent duplicate: return previous message ID
			writeJSON(w, http.StatusAccepted, PublishResponse{
				MessageID:        existing.ID,
				EventType:        existing.EventType,
				DeliveriesQueued: 0,
				Deduplicated:     true,
				CreatedAt:        existing.CreatedAt,
			})
			return
		}
	}

	// 2. Resolve destination endpoints
	var targetEndpoints []*storage.Endpoint
	if req.EndpointID != "" {
		ep, err := h.store.GetEndpoint(ctx, req.EndpointID)
		if err != nil || ep == nil {
			writeError(w, http.StatusNotFound, "specified endpoint_id not found")
			return
		}
		targetEndpoints = append(targetEndpoints, ep)
	} else {
		// Fan-out to all active subscribers
		matched, err := h.store.GetActiveEndpointsForEvent(ctx, req.EventType)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to query subscribed endpoints")
			return
		}
		targetEndpoints = matched
	}

	// 3. Create Message record
	var idKeyPtr *string
	if idempotencyKey != "" {
		idKeyPtr = &idempotencyKey
	}

	msg := &storage.Message{
		IdempotencyKey: idKeyPtr,
		EventType:      req.EventType,
		Payload:        string(req.Payload),
		CreatedAt:      time.Now().UTC(),
	}
	if err := h.store.CreateMessage(ctx, msg); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to store message: %v", err))
		return
	}

	// 4. Create and enqueue Delivery tasks for each target endpoint
	queuedCount := 0
	for _, ep := range targetEndpoints {
		if ep.Status == storage.EndpointStatusDisabled {
			continue
		}

		del := &storage.Delivery{
			MessageID:   msg.ID,
			EndpointID:  ep.ID,
			Status:      storage.DeliveryStatusPending,
			MaxAttempts: 5,
			CreatedAt:   time.Now().UTC(),
		}

		if err := h.store.CreateDelivery(ctx, del); err == nil {
			h.engine.Enqueue(del.ID)
			queuedCount++
		}
	}

	writeJSON(w, http.StatusAccepted, PublishResponse{
		MessageID:        msg.ID,
		EventType:        msg.EventType,
		DeliveriesQueued: queuedCount,
		CreatedAt:        msg.CreatedAt,
	})
}
