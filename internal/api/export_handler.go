package api

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/alexandrmotologa/hookrelay/internal/storage"
)

type ExportHandler struct {
	store storage.Storage
}

func NewExportHandler(store storage.Storage) *ExportHandler {
	return &ExportHandler{store: store}
}

func (h *ExportHandler) HandleExport(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	format := q.Get("format")
	if format == "" {
		format = "json"
	}

	limit := 1000
	if lStr := q.Get("limit"); lStr != "" {
		if parsed, err := strconv.Atoi(lStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	filter := storage.DeliveryFilter{
		EndpointID: q.Get("endpoint_id"),
		Status:     q.Get("status"),
		EventType:  q.Get("event_type"),
		Search:     q.Get("search"),
		Limit:      limit,
		Offset:     0,
	}

	items, _, err := h.store.ListDeliveries(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query deliveries for export")
		return
	}

	filenameDate := time.Now().UTC().Format("20060102_150405")

	if format == "csv" {
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"hookrelay_audit_%s.csv\"", filenameDate))

		writer := csv.NewWriter(w)
		defer writer.Flush()

		// Write CSV header
		_ = writer.Write([]string{
			"delivery_id",
			"message_id",
			"endpoint_name",
			"endpoint_url",
			"event_type",
			"status",
			"http_status",
			"duration_ms",
			"attempt_number",
			"max_attempts",
			"error_message",
			"sent_at",
			"created_at",
		})

		for _, it := range items {
			httpCode := ""
			if it.HTTPStatus != nil {
				httpCode = strconv.Itoa(*it.HTTPStatus)
			}
			durationStr := ""
			if it.DurationMs != nil {
				durationStr = strconv.FormatInt(*it.DurationMs, 10)
			}
			sentAtStr := ""
			if it.SentAt != nil {
				sentAtStr = it.SentAt.Format(time.RFC3339)
			}

			_ = writer.Write([]string{
				it.ID,
				it.MessageID,
				it.EndpointName,
				it.EndpointURL,
				it.EventType,
				string(it.Status),
				httpCode,
				durationStr,
				strconv.Itoa(it.AttemptNumber),
				strconv.Itoa(it.MaxAttempts),
				it.ErrorMessage,
				sentAtStr,
				it.CreatedAt.Format(time.RFC3339),
			})
		}
		return
	}

	// JSON Export
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"hookrelay_audit_%s.json\"", filenameDate))
	_ = json.NewEncoder(w).Encode(items)
}
