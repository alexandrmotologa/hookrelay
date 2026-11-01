package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alexandrmotologa/hookrelay/internal/queue"
	"github.com/alexandrmotologa/hookrelay/internal/server"
	"github.com/alexandrmotologa/hookrelay/internal/storage"
)

var (
	version = "1.0.0"
)

func main() {
	if len(os.Args) < 2 {
		runServe(os.Args[1:])
		return
	}

	switch os.Args[1] {
	case "serve":
		runServe(os.Args[2:])
	case "publish":
		runPublish(os.Args[2:])
	case "status":
		runStatus(os.Args[2:])
	case "version", "--version", "-v":
		fmt.Printf("HookRelay v%s\n", version)
	case "help", "--help", "-h":
		printUsage()
	default:
		// Default to serve with all arguments
		runServe(os.Args[1:])
	}
}

func printUsage() {
	fmt.Printf(`HookRelay v%s - High-Throughput Outbound Webhook Delivery Gateway

Usage:
  hookrelay [command] [options]

Commands:
  serve       Start the HookRelay engine and web studio (default)
  publish     Publish a webhook event via CLI
  status      Display delivery statistics from a running instance
  version     Display version information
  help        Show this help text

Serve Flags:
  --port               Port to listen on (default: 9770)
  --db                 Path to SQLite database file (default: "./hookrelay.db")
  --workers            Number of concurrent HTTP delivery workers (default: 20)
  --allow-private-ips  Allow webhooks to private IP ranges (for local dev)
  --portal-secret      Secret key used to sign customer portal tokens

Examples:
  hookrelay serve --port 9770 --db ./hookrelay.db
  hookrelay publish --event payment.succeeded --data '{"order_id":"123"}'
  hookrelay status --url http://localhost:9770
`, version)
}

func runServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	port := fs.Int("port", 9770, "HTTP port to listen on")
	dbPath := fs.String("db", "./hookrelay.db", "SQLite database file path")
	workers := fs.Int("workers", 20, "Number of concurrent delivery workers")
	allowPrivate := fs.Bool("allow-private-ips", false, "Allow deliveries to private/localhost addresses")
	portalSecret := fs.String("portal-secret", "hookrelay_portal_internal_secret_default_key_2026", "Key for customer portal tokens")

	_ = fs.Parse(args)

	log.Printf("[HookRelay] Starting HookRelay v%s...", version)
	log.Printf("[HookRelay] Initializing SQLite database at: %s", *dbPath)

	store, err := storage.NewSQLiteStorage(*dbPath)
	if err != nil {
		log.Fatalf("[HookRelay] Fatal: failed to open storage: %v", err)
	}
	defer store.Close()

	engineCfg := queue.DefaultEngineConfig()
	engineCfg.Workers = *workers
	engineCfg.AllowPrivateIPs = *allowPrivate

	engine := queue.NewEngine(store, engineCfg)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	engine.Start(ctx)
	defer engine.Stop()

	srvCfg := server.Config{
		Port:            *port,
		AllowPrivateIPs: *allowPrivate,
		PortalSecret:    *portalSecret,
	}

	srv := server.NewServer(store, engine, srvCfg)

	// Graceful shutdown handling
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[HookRelay] Fatal: server failed: %v", err)
		}
	}()

	log.Printf("[HookRelay] Ready. Web Studio open at http://localhost:%d", *port)
	<-stopChan

	log.Printf("[HookRelay] Shutting down...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("[HookRelay] Error during server shutdown: %v", err)
	}
	log.Printf("[HookRelay] Terminated cleanly.")
}

func runPublish(args []string) {
	fs := flag.NewFlagSet("publish", flag.ExitOnError)
	serverURL := fs.String("url", "http://localhost:9770", "HookRelay base URL")
	event := fs.String("event", "", "Event type (e.g. payment.succeeded)")
	data := fs.String("data", "{}", "JSON payload string")
	endpoint := fs.String("endpoint", "", "Specific target endpoint ID (optional)")
	idempotency := fs.String("idempotency-key", "", "Unique idempotency key (optional)")

	_ = fs.Parse(args)

	if *event == "" {
		fmt.Println("Error: --event is required")
		os.Exit(1)
	}

	var payloadVal any
	if err := json.Unmarshal([]byte(*data), &payloadVal); err != nil {
		// Fallback: use raw string as JSON string value
		payloadVal = *data
	}

	body := map[string]any{
		"event_type": *event,
		"payload":    payloadVal,
	}
	if *endpoint != "" {
		body["endpoint_id"] = *endpoint
	}

	jsonBytes, err := json.Marshal(body)
	if err != nil {
		fmt.Printf("Error: failed to encode request: %v\n", err)
		os.Exit(1)
	}

	req, err := http.NewRequest(http.MethodPost, *serverURL+"/api/v1/publish", bytes.NewReader(jsonBytes))
	if err != nil {
		fmt.Printf("Error building request: %v\n", err)
		os.Exit(1)
	}
	req.Header.Set("Content-Type", "application/json")
	if *idempotency != "" {
		req.Header.Set("Idempotency-Key", *idempotency)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Error sending publish request: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		fmt.Printf("✓ Event published successfully (HTTP %d):\n%s\n", resp.StatusCode, string(respBody))
	} else {
		fmt.Printf("✗ Event publish failed (HTTP %d):\n%s\n", resp.StatusCode, string(respBody))
		os.Exit(1)
	}
}

func runStatus(args []string) {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	serverURL := fs.String("url", "http://localhost:9770", "HookRelay base URL")
	_ = fs.Parse(args)

	resp, err := http.Get(*serverURL + "/api/v1/stats")
	if err != nil {
		fmt.Printf("Error connecting to HookRelay at %s: %v\n", *serverURL, err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var stats storage.StatsSummary
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		fmt.Printf("Error parsing stats response: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf(`================ HookRelay Status ================
Total Messages Ingested:    %d
Total Deliveries Attempted: %d
  - Successful (HTTP 2xx):  %d (%.1f%%)
  - Retrying:               %d
  - Failed (DLQ):           %d
  - Pending / In-Flight:    %d
Average Response Duration:  %.1f ms
Active Destination Endpoints:   %d
Suspended (Circuit Tripped):    %d
==================================================
`,
		stats.TotalMessages,
		stats.TotalDeliveries,
		stats.SuccessfulCount,
		stats.SuccessRate,
		stats.RetryingCount,
		stats.FailedCount,
		stats.PendingCount,
		stats.AvgDurationMs,
		stats.ActiveEndpoints,
		stats.SuspendedEndpoints,
	)
}
