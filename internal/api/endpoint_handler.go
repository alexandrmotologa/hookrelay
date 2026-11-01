package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/alexandrmotologa/hookrelay/internal/dispatcher"
	"github.com/alexandrmotologa/hookrelay/internal/queue"
	"github.com/alexandrmotologa/hookrelay/internal/signer"
	"github.com/alexandrmotologa/hookrelay/internal/storage"
)

type EndpointHandler struct {
	store           storage.Storage
	engine          *queue.Engine
	allowPrivateIPs bool
}

func NewEndpointHandler(store storage.Storage, engine *queue.Engine, allowPrivateIPs bool) *EndpointHandler {
	return &EndpointHandler{
		store:           store,
		engine:          engine,
		allowPrivateIPs: allowPrivateIPs,
	}
}

type CreateEndpointRequest struct {
	Name       string   `json:"name"`
	URL        string   `json:"url"`
	Secret     string   `json:"secret,omitempty"`
	EventTypes []string `json:"event_types,omitempty"`
	RateLimit  int      `json:"rate_limit,omitempty"`
}

type UpdateEndpointRequest struct {
	Name       string                 `json:"name"`
	URL        string                 `json:"url"`
	EventTypes []string               `json:"event_types,omitempty"`
	RateLimit  int                    `json:"rate_limit,omitempty"`
	Status     storage.EndpointStatus `json:"status,omitempty"`
}

func (h *EndpointHandler) ListEndpoints(w http.ResponseWriter, r *http.Request) {
	endpoints, err := h.store.ListEndpoints(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query endpoints")
		return
	}
	if endpoints == nil {
		endpoints = []*storage.Endpoint{}
	}
	writeJSON(w, http.StatusOK, endpoints)
}

func (h *EndpointHandler) CreateEndpoint(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req CreateEndpointRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.URL = strings.TrimSpace(req.URL)
	if req.URL == "" {
		writeError(w, http.StatusBadRequest, "url is required")
		return
	}
	if req.Name == "" {
		req.Name = "Endpoint " + req.URL
	}

	// Validate destination URL with SSRF protection
	if _, err := dispatcher.ValidateURL(req.URL, h.allowPrivateIPs); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid webhook URL: %v", err))
		return
	}

	secret := strings.TrimSpace(req.Secret)
	if secret == "" {
		generated, err := signer.GenerateSecret()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to generate signing secret")
			return
		}
		secret = generated
	}

	ep := &storage.Endpoint{
		Name:       req.Name,
		URL:        req.URL,
		Secret:     secret,
		EventTypes: req.EventTypes,
		RateLimit:  req.RateLimit,
		Status:     storage.EndpointStatusActive,
	}

	if err := h.store.CreateEndpoint(ctx, ep); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to save endpoint: %v", err))
		return
	}

	writeJSON(w, http.StatusCreated, ep)
}

func (h *EndpointHandler) GetEndpoint(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ep, err := h.store.GetEndpoint(r.Context(), id)
	if err != nil || ep == nil {
		writeError(w, http.StatusNotFound, "endpoint not found")
		return
	}
	writeJSON(w, http.StatusOK, ep)
}

func (h *EndpointHandler) UpdateEndpoint(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ctx := r.Context()

	ep, err := h.store.GetEndpoint(ctx, id)
	if err != nil || ep == nil {
		writeError(w, http.StatusNotFound, "endpoint not found")
		return
	}

	var req UpdateEndpointRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.URL = strings.TrimSpace(req.URL)
	if req.URL != "" && req.URL != ep.URL {
		if _, err := dispatcher.ValidateURL(req.URL, h.allowPrivateIPs); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid webhook URL: %v", err))
			return
		}
		ep.URL = req.URL
	}

	if strings.TrimSpace(req.Name) != "" {
		ep.Name = strings.TrimSpace(req.Name)
	}
	if req.EventTypes != nil {
		ep.EventTypes = req.EventTypes
	}
	if req.RateLimit > 0 {
		ep.RateLimit = req.RateLimit
	}
	if req.Status != "" {
		ep.Status = req.Status
	}

	if err := h.store.UpdateEndpoint(ctx, ep); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to update endpoint: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, ep)
}

func (h *EndpointHandler) DeleteEndpoint(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.store.DeleteEndpoint(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete endpoint")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *EndpointHandler) RotateSecret(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ctx := r.Context()

	ep, err := h.store.GetEndpoint(ctx, id)
	if err != nil || ep == nil {
		writeError(w, http.StatusNotFound, "endpoint not found")
		return
	}

	newSecret, err := signer.GenerateSecret()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate secret")
		return
	}

	ep.Secret = newSecret
	if err := h.store.UpdateEndpoint(ctx, ep); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save rotated secret")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"endpoint_id": ep.ID,
		"secret":      ep.Secret,
	})
}

func (h *EndpointHandler) ResetCircuitBreaker(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.store.ResetEndpointFailure(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to reset circuit breaker")
		return
	}

	ep, _ := h.store.GetEndpoint(r.Context(), id)
	writeJSON(w, http.StatusOK, map[string]any{
		"endpoint_id":          id,
		"status":               ep.Status,
		"consecutive_failures": ep.ConsecutiveFailures,
	})
}

func (h *EndpointHandler) TestPing(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ctx := r.Context()

	ep, err := h.store.GetEndpoint(ctx, id)
	if err != nil || ep == nil {
		writeError(w, http.StatusNotFound, "endpoint not found")
		return
	}

	// Create test message
	msg := &storage.Message{
		EventType: "ping",
		Payload:   fmt.Sprintf(`{"type":"ping","timestamp":"%s","message":"HookRelay test verification ping"}`, time.Now().UTC().Format(time.RFC3339)),
		CreatedAt: time.Now().UTC(),
	}
	if err := h.store.CreateMessage(ctx, msg); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create ping message")
		return
	}

	// Create delivery
	del := &storage.Delivery{
		MessageID:   msg.ID,
		EndpointID:  ep.ID,
		Status:      storage.DeliveryStatusPending,
		MaxAttempts: 1,
		CreatedAt:   time.Now().UTC(),
	}
	if err := h.store.CreateDelivery(ctx, del); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create ping delivery")
		return
	}

	// Execute synchronously for immediate feedback in the UI
	h.engine.ProcessDelivery(ctx, del.ID)

	updated, _ := h.store.GetDeliveryWithDetails(ctx, del.ID)
	writeJSON(w, http.StatusOK, updated)
}
