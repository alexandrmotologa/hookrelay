package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/alexandrmotologa/hookrelay/internal/storage"
)

type SSEBroker struct {
	clients map[chan []byte]bool
	mu      sync.RWMutex
}

func NewSSEBroker() *SSEBroker {
	return &SSEBroker{
		clients: make(map[chan []byte]bool),
	}
}

func (b *SSEBroker) BroadcastDelivery(d *storage.DeliveryWithDetails) {
	bytes, err := json.Marshal(d)
	if err != nil {
		return
	}

	msg := []byte(fmt.Sprintf("event: delivery\ndata: %s\n\n", bytes))

	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.clients {
		select {
		case ch <- msg:
		default:
			// Non-blocking write so slow clients don't hang the broker
		}
	}
}

func (b *SSEBroker) HandleStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming unsupported by server")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	clientChan := make(chan []byte, 32)
	b.mu.Lock()
	b.clients[clientChan] = true
	b.mu.Unlock()

	defer func() {
		b.mu.Lock()
		delete(b.clients, clientChan)
		b.mu.Unlock()
		close(clientChan)
	}()

	// Send initial connection event
	_, _ = fmt.Fprintf(w, "event: connected\ndata: {\"connected\":true,\"timestamp\":\"%s\"}\n\n", time.Now().UTC().Format(time.RFC3339))
	flusher.Flush()

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			_, err := fmt.Fprintf(w, ": keepalive\n\n")
			if err != nil {
				return
			}
			flusher.Flush()
		case msg := <-clientChan:
			_, err := w.Write(msg)
			if err != nil {
				return
			}
			flusher.Flush()
		}
	}
}
