package api

import (
	"net/http"
	"strconv"

	"github.com/alexandrmotologa/hookrelay/internal/queue"
	"github.com/alexandrmotologa/hookrelay/internal/storage"
)

type MessageHandler struct {
	store  storage.Storage
	engine *queue.Engine
}

func NewMessageHandler(store storage.Storage, engine *queue.Engine) *MessageHandler {
	return &MessageHandler{
		store:  store,
		engine: engine,
	}
}

func (h *MessageHandler) ListDeliveries(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))

	filter := storage.DeliveryFilter{
		EndpointID: q.Get("endpoint_id"),
		Status:     q.Get("status"),
		EventType:  q.Get("event_type"),
		Search:     q.Get("search"),
		Limit:      limit,
		Offset:     offset,
	}

	items, total, err := h.store.ListDeliveries(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query deliveries")
		return
	}
	if items == nil {
		items = []*storage.DeliveryWithDetails{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"total": total,
		"items": items,
	})
}

func (h *MessageHandler) GetDelivery(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	item, err := h.store.GetDeliveryWithDetails(r.Context(), id)
	if err != nil || item == nil {
		writeError(w, http.StatusNotFound, "delivery not found")
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *MessageHandler) ReplayDelivery(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ctx := r.Context()

	newDel, err := h.store.ReplayDelivery(ctx, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to queue replay")
		return
	}

	h.engine.Enqueue(newDel.ID)
	writeJSON(w, http.StatusOK, map[string]any{
		"replayed_delivery_id": newDel.ID,
		"status":               newDel.Status,
	})
}

func (h *MessageHandler) ReplayDLQ(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	count, err := h.store.ReplayAllDLQ(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to replay dead letter deliveries")
		return
	}

	// Trigger poller
	pending, _ := h.store.GetPendingDeliveries(ctx, 100)
	for _, p := range pending {
		h.engine.Enqueue(p.ID)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"replayed_count": count,
	})
}

func (h *MessageHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.store.GetStats(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to compute stats")
		return
	}
	writeJSON(w, http.StatusOK, stats)
}
