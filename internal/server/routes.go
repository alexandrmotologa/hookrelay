package server

import (
	"encoding/json"
	"net/http"

	"github.com/alexandrmotologa/hookrelay/internal/api"
	"github.com/alexandrmotologa/hookrelay/internal/queue"
	"github.com/alexandrmotologa/hookrelay/internal/storage"
)

func BuildRouter(store storage.Storage, engine *queue.Engine, allowPrivateIPs bool, portalSecret string) http.Handler {
	mux := http.NewServeMux()

	publishHandler := api.NewPublishHandler(store, engine)
	endpointHandler := api.NewEndpointHandler(store, engine, allowPrivateIPs)
	messageHandler := api.NewMessageHandler(store, engine)
	portalHandler := api.NewPortalHandler(store, engine, portalSecret)

	// Health Check
	mux.HandleFunc("GET /api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"service": "hookrelay",
			"version": "1.0.0",
		})
	})

	// Webhook Ingestion
	mux.HandleFunc("POST /api/v1/publish", publishHandler.HandlePublish)

	// Endpoints Management
	mux.HandleFunc("GET /api/v1/endpoints", endpointHandler.ListEndpoints)
	mux.HandleFunc("POST /api/v1/endpoints", endpointHandler.CreateEndpoint)
	mux.HandleFunc("GET /api/v1/endpoints/{id}", endpointHandler.GetEndpoint)
	mux.HandleFunc("PUT /api/v1/endpoints/{id}", endpointHandler.UpdateEndpoint)
	mux.HandleFunc("DELETE /api/v1/endpoints/{id}", endpointHandler.DeleteEndpoint)
	mux.HandleFunc("POST /api/v1/endpoints/{id}/rotate-secret", endpointHandler.RotateSecret)
	mux.HandleFunc("POST /api/v1/endpoints/{id}/reset-circuit-breaker", endpointHandler.ResetCircuitBreaker)
	mux.HandleFunc("POST /api/v1/endpoints/{id}/test-ping", endpointHandler.TestPing)

	// Deliveries, DLQ & Metrics
	mux.HandleFunc("GET /api/v1/deliveries", messageHandler.ListDeliveries)
	mux.HandleFunc("GET /api/v1/deliveries/{id}", messageHandler.GetDelivery)
	mux.HandleFunc("POST /api/v1/deliveries/{id}/replay", messageHandler.ReplayDelivery)
	mux.HandleFunc("POST /api/v1/deliveries/replay-dlq", messageHandler.ReplayDLQ)
	mux.HandleFunc("GET /api/v1/stats", messageHandler.GetStats)

	// Customer Portal
	mux.HandleFunc("POST /api/v1/portal/token", portalHandler.HandleGenerateToken)
	mux.HandleFunc("GET /api/v1/portal/view", portalHandler.HandlePortalView)
	mux.HandleFunc("POST /api/v1/portal/retry", portalHandler.HandlePortalRetry)
	mux.HandleFunc("GET /api/v1/portal/snippets", portalHandler.HandlePortalSnippets)

	// Embedded UI / SPA routes
	mux.Handle("/", SPAHandler())

	// Apply Middlewares
	return api.CORSMiddleware(api.RequestLogger(mux))
}
