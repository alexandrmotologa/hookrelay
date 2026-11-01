package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/alexandrmotologa/hookrelay/internal/queue"
	"github.com/alexandrmotologa/hookrelay/internal/storage"
)

type Config struct {
	Port            int
	AllowPrivateIPs bool
	PortalSecret    string
}

type Server struct {
	httpServer *http.Server
	store      storage.Storage
	engine     *queue.Engine
}

func NewServer(store storage.Storage, engine *queue.Engine, cfg Config) *Server {
	router := BuildRouter(store, engine, cfg.AllowPrivateIPs, cfg.PortalSecret)

	addr := fmt.Sprintf(":%d", cfg.Port)
	srv := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	return &Server{
		httpServer: srv,
		store:      store,
		engine:     engine,
	}
}

func (s *Server) Start() error {
	log.Printf("[HookRelay] HTTP Server listening on %s", s.httpServer.Addr)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	log.Printf("[HookRelay] Initiating graceful shutdown...")
	return s.httpServer.Shutdown(ctx)
}
