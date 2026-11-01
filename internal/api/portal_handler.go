package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/alexandrmotologa/hookrelay/internal/queue"
	"github.com/alexandrmotologa/hookrelay/internal/signer"
	"github.com/alexandrmotologa/hookrelay/internal/storage"
)

type PortalHandler struct {
	store        storage.Storage
	engine       *queue.Engine
	portalSecret []byte
}

func NewPortalHandler(store storage.Storage, engine *queue.Engine, portalSecret string) *PortalHandler {
	if portalSecret == "" {
		portalSecret = "hookrelay_portal_internal_secret_default_key_2026"
	}
	return &PortalHandler{
		store:        store,
		engine:       engine,
		portalSecret: []byte(portalSecret),
	}
}

type PortalTokenRequest struct {
	EndpointID     string `json:"endpoint_id"`
	ExpiresInHours int    `json:"expires_in_hours,omitempty"`
}

// GenerateToken builds a cryptographically signed portal token for a specific endpoint.
func (h *PortalHandler) GenerateToken(endpointID string, duration time.Duration) string {
	exp := time.Now().UTC().Add(duration).Unix()
	payload := fmt.Sprintf("%s.%d", endpointID, exp)

	mac := hmac.New(sha256.New, h.portalSecret)
	mac.Write([]byte(payload))
	sigHex := hex.EncodeToString(mac.Sum(nil))

	fullToken := fmt.Sprintf("%s.%s", payload, sigHex)
	return base64.RawURLEncoding.EncodeToString([]byte(fullToken))
}

// VerifyToken parses and validates a signed portal token, returning the scoped endpoint ID.
func (h *PortalHandler) VerifyToken(tokenStr string) (string, error) {
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(tokenStr))
	if err != nil {
		return "", errors.New("invalid base64 token")
	}

	parts := strings.Split(string(raw), ".")
	if len(parts) != 3 {
		return "", errors.New("malformed token structure")
	}

	endpointID := parts[0]
	expStr := parts[1]
	providedSigHex := parts[2]

	exp, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil {
		return "", errors.New("invalid token expiration")
	}

	if time.Now().UTC().Unix() > exp {
		return "", errors.New("portal token has expired")
	}

	payload := fmt.Sprintf("%s.%s", endpointID, expStr)
	mac := hmac.New(sha256.New, h.portalSecret)
	mac.Write([]byte(payload))
	expectedSigHex := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(providedSigHex), []byte(expectedSigHex)) {
		return "", errors.New("portal token signature mismatch")
	}

	return endpointID, nil
}

func (h *PortalHandler) HandleGenerateToken(w http.ResponseWriter, r *http.Request) {
	var req PortalTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.EndpointID = strings.TrimSpace(req.EndpointID)
	if req.EndpointID == "" {
		writeError(w, http.StatusBadRequest, "endpoint_id is required")
		return
	}

	ep, err := h.store.GetEndpoint(r.Context(), req.EndpointID)
	if err != nil || ep == nil {
		writeError(w, http.StatusNotFound, "endpoint not found")
		return
	}

	hours := req.ExpiresInHours
	if hours <= 0 {
		hours = 72 // 3 days default
	}

	token := h.GenerateToken(ep.ID, time.Duration(hours)*time.Hour)
	writeJSON(w, http.StatusOK, map[string]any{
		"endpoint_id": ep.ID,
		"token":       token,
		"expires_at":  time.Now().UTC().Add(time.Duration(hours) * time.Hour),
	})
}

func (h *PortalHandler) extractToken(r *http.Request) string {
	token := r.URL.Query().Get("token")
	if token == "" {
		token = r.Header.Get("X-Portal-Token")
	}
	if token == "" {
		auth := r.Header.Get("Authorization")
		if strings.HasPrefix(auth, "Bearer ") {
			token = strings.TrimPrefix(auth, "Bearer ")
		}
	}
	return token
}

func (h *PortalHandler) HandlePortalView(w http.ResponseWriter, r *http.Request) {
	token := h.extractToken(r)
	if token == "" {
		writeError(w, http.StatusUnauthorized, "missing portal token")
		return
	}

	endpointID, err := h.VerifyToken(token)
	if err != nil {
		writeError(w, http.StatusUnauthorized, fmt.Sprintf("invalid portal token: %v", err))
		return
	}

	ctx := r.Context()
	ep, err := h.store.GetEndpoint(ctx, endpointID)
	if err != nil || ep == nil {
		writeError(w, http.StatusNotFound, "associated endpoint not found")
		return
	}

	// Query deliveries scoped to this endpoint
	filter := storage.DeliveryFilter{
		EndpointID: ep.ID,
		Limit:      50,
	}
	items, total, err := h.store.ListDeliveries(ctx, filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query deliveries")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"endpoint": ep,
		"total":    total,
		"items":    items,
	})
}

func (h *PortalHandler) HandlePortalRetry(w http.ResponseWriter, r *http.Request) {
	token := h.extractToken(r)
	if token == "" {
		writeError(w, http.StatusUnauthorized, "missing portal token")
		return
	}

	endpointID, err := h.VerifyToken(token)
	if err != nil {
		writeError(w, http.StatusUnauthorized, fmt.Sprintf("invalid portal token: %v", err))
		return
	}

	deliveryID := r.URL.Query().Get("delivery_id")
	if deliveryID == "" {
		var req struct {
			DeliveryID string `json:"delivery_id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		deliveryID = req.DeliveryID
	}

	if deliveryID == "" {
		writeError(w, http.StatusBadRequest, "delivery_id is required")
		return
	}

	ctx := r.Context()
	del, err := h.store.GetDelivery(ctx, deliveryID)
	if err != nil || del == nil {
		writeError(w, http.StatusNotFound, "delivery not found")
		return
	}

	// Security: verify delivery belongs to this token's endpoint
	if del.EndpointID != endpointID {
		writeError(w, http.StatusForbidden, "unauthorized: delivery does not belong to this endpoint")
		return
	}

	newDel, err := h.store.ReplayDelivery(ctx, del.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to trigger retry")
		return
	}

	h.engine.Enqueue(newDel.ID)
	writeJSON(w, http.StatusOK, map[string]any{
		"replayed_delivery_id": newDel.ID,
		"status":               newDel.Status,
	})
}

func (h *PortalHandler) HandlePortalSnippets(w http.ResponseWriter, r *http.Request) {
	token := h.extractToken(r)
	if token == "" {
		writeError(w, http.StatusUnauthorized, "missing portal token")
		return
	}

	endpointID, err := h.VerifyToken(token)
	if err != nil {
		writeError(w, http.StatusUnauthorized, fmt.Sprintf("invalid portal token: %v", err))
		return
	}

	ep, err := h.store.GetEndpoint(r.Context(), endpointID)
	if err != nil || ep == nil {
		writeError(w, http.StatusNotFound, "endpoint not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"nodejs": signer.GenerateSnippet("node", ep.Secret),
		"python": signer.GenerateSnippet("python", ep.Secret),
		"go":     signer.GenerateSnippet("go", ep.Secret),
	})
}
