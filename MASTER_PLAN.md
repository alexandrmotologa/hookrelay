# HookRelay: Master Implementation Plan

High-throughput outbound webhook delivery engine, HMAC signing, and exponential retry gateway in Go 1.23 and React 19.

HookRelay is a self-hostable outbound webhook platform (Svix and Hookdeck alternative). It dispatches customer webhooks reliably, signs payloads using Standard Webhooks HMAC-SHA256 headers, runs exponential backoff retries with per-endpoint circuit breakers, and provides an embeddable customer delivery portal with manual replay.

---

## 1. Architecture Overview

When an application generates business events (for example, `payment.succeeded`, `invoice.created`, `order.fulfilled`), delivering those webhooks directly from application workers causes problems:
1. Slow customer endpoints tie up worker threads.
2. Downtime on receiving servers causes lost events without a persistent retry store.
3. Lack of standardized cryptographic signatures leaves endpoints open to spoofing and replay attacks.
4. Support teams waste time verifying whether a webhook was dispatched.
5. Cloud webhook vendors impose recurring fees and data lock-in.

HookRelay resolves this with a self-contained service:
- **Fast Ingestion (`POST /api/v1/publish`)**: Accepts payload and routing keys in under 2ms with optional idempotency key deduplication.
- **Cryptographic HMAC-SHA256 Signing**: Standard Webhooks specification (`webhook-id`, `webhook-timestamp`, `webhook-signature: v1,...`).
- **Durable Retry State Machine**: 5 automated retry attempts (5s, 1m, 15m, 1h, 24h) with randomized jitter (+/-15%).
- **Per-Endpoint Circuit Breakers**: Automatically pauses delivery after 50 consecutive failures to prevent resource exhaustion and downstream flooding.
- **SSRF and Replay Defense**: Blocks local network ranges, AWS/GCP metadata addresses, and rejects redirects by default.
- **Customer Self-Serve Portal**: Scoped token URL (`/portal/:token`) where external users inspect delivery logs, check latency, view signature snippets, and trigger manual retries.

```
+-----------------------------------------------------------------------------------+
|                                     HOOKRELAY                                     |
+-----------------------------------------+-----------------------------------------+
                                          |
        +---------------------------------+---------------------------------+
        |                                                                   |
+-------v-------------------------+                 +-----------------------v-------+
|    Internal Ingestion API       |                 |    HMAC-SHA256 Signing Engine |
|  * POST /api/v1/publish         |                 |  * Standard Webhooks Spec     |
|  * Sub-2ms Non-Blocking Intake  |                 |  * Timestamp Replay Defense   |
|  * Topic & Tenant Routing       |                 |  * Shared Secret Rotation     |
|  * Idempotency Key De-dupe      |                 |  * Multi-language Verifiers   |
+-------+-------------------------+                 +-----------------------+-------+
        |                                                                   |
        +---------------------------------+---------------------------------+
                                          |
                        +-----------------v-----------------+
                        |    Durable Retry State Machine    |
                        |  * Exponential Backoff & Jitter   |
                        |  * Schedule: 5s, 1m, 15m, 1h, 24h |
                        |  * Circuit Breaker per Endpoint   |
                        |  * Dead Letter Queue (DLQ)        |
                        +-----------------+-----------------+
                                          |
                        +-----------------v-----------------+
                        |    High-Concurrency HTTP Worker   |
                        |  * Go Goroutine Worker Pool       |
                        |  * Strict 5s Connect/Read Timeout |
                        |  * Anti-SSRF Private IP Blocker   |
                        |  * Full Payload & Response Record |
                        +-----------------+-----------------+
                                          |
                        +-----------------v-----------------+
                        |   Interactive Web Studio & Portal |
                        |  * Admin Mission Control (:9770)  |
                        |  * Customer Self-Serve Portal     |
                        |  * 1-Click Manual Event Replay    |
                        |  * Test Webhook Simulator         |
                        +-----------------------------------+
```

---

## 2. Directory Structure

```
hookrelay/
├── cmd/
│   └── hookrelay/
│       └── main.go               # Single-binary CLI and server entrypoint
├── internal/
│   ├── api/
│   │   ├── handler.go            # Common HTTP response helpers and middleware
│   │   ├── publish_handler.go    # POST /api/v1/publish and replay handlers
│   │   ├── endpoint_handler.go   # CRUD, secret rotation, circuit breaker reset
│   │   ├── message_handler.go    # Message query and delivery attempt logs
│   │   └── portal_handler.go     # Scoped customer portal token generation and validation
│   ├── signer/
│   │   ├── hmac.go               # Standard Webhooks HMAC-SHA256 signature generator
│   │   └── verifier.go           # Signature verification helper and test suite
│   ├── queue/
│   │   ├── state_machine.go      # PENDING -> IN_FLIGHT -> SUCCESS / RETRY / FAILED
│   │   ├── retry_scheduler.go    # Exponential backoff schedule with jitter
│   │   └── circuit_breaker.go    # Consecutive failure tracker and state transitions
│   ├── dispatcher/
│   │   ├── worker_pool.go        # Concurrent HTTP delivery workers
│   │   ├── client.go             # Hardened HTTP client with timeout, anti-SSRF, and TLS caps
│   │   └── ssrf.go               # Private IP and metadata address validation
│   ├── storage/
│   │   ├── storage.go            # Storage interface definitions
│   │   ├── sqlite.go             # Pure Go SQLite WAL storage implementation
│   │   ├── models.go             # Domain data models (Endpoint, Message, Delivery)
│   │   └── migrations/
│   │       └── 001_init.sql      # Schema definition with optimized indexes
│   └── server/
│       ├── server.go             # HTTP server with graceful shutdown
│       ├── routes.go             # Route registrations and API versioning
│       └── static.go             # Go embed.FS for React 19 UI assets
├── web/
│   ├── src/
│   │   ├── components/
│   │   │   ├── Header.tsx            # Navigation, status indicators, theme toggle
│   │   │   ├── DeliveryLogTable.tsx  # Live stream of dispatched webhooks with status pills
│   │   │   ├── MessagePayloadModal.tsx # Request headers, body, response headers & body
│   │   │   ├── EndpointManager.tsx   # Target URL, secret key, active event types
│   │   │   ├── CircuitBreakerCard.tsx# Endpoints currently suspended or in backoff
│   │   │   ├── TestSimulator.tsx     # Direct webhook trigger modal
│   │   │   ├── CustomerPortal.tsx    # White-label self-serve delivery view
│   │   │   └── CodeSnippetModal.tsx  # Verifier code examples (Node, Python, Go)
│   │   ├── hooks/
│   │   │   └── useDeliveryStream.ts  # Polling hook for live delivery logs
│   │   ├── types/
│   │   │   └── index.ts              # Frontend TypeScript interfaces
│   │   ├── App.tsx                   # Main layout and tab navigation
│   │   ├── index.css                 # Tailwind CSS styles
│   │   └── main.tsx                  # React 19 root
│   ├── package.json
│   ├── vite.config.ts
│   ├── tsconfig.json
│   └── index.html
├── docs/
│   ├── API_SPEC.md               # Complete REST API specification
│   ├── STANDARD_WEBHOOKS.md      # Signature format and verification guidelines
│   └── ARCHITECTURE.md           # Concurrency model, retry state machine, storage
├── go.mod
├── go.sum
├── LICENSE
└── README.md
```

---

## 3. Implementation Phases

### Phase 1: Storage Layer and Domain Models
- Pure Go SQLite driver (`modernc.org/sqlite`) for zero-dependency builds without CGO.
- Tables:
  - `endpoints`: `id`, `name`, `url`, `secret`, `event_types`, `rate_limit`, `status`, `consecutive_failures`, `tripped_at`, `created_at`, `updated_at`.
  - `messages`: `id`, `idempotency_key`, `event_type`, `payload`, `created_at`.
  - `deliveries`: `id`, `message_id`, `endpoint_id`, `status` (`PENDING`, `IN_FLIGHT`, `SUCCESS`, `RETRY`, `FAILED`), `attempt_number`, `max_attempts`, `http_status`, `duration_ms`, `request_headers`, `response_headers`, `response_body`, `error_message`, `next_retry_at`, `sent_at`, `created_at`.
  - `idempotency_keys`: `key`, `message_id`, `created_at`, `expires_at`.
- Enable WAL mode and busy timeout for concurrent writer and reader isolation.

### Phase 2: Standard Webhooks HMAC-SHA256 Signing Engine
- Standard Webhooks specification compliance:
  - Header: `webhook-id: msg_01h...`
  - Header: `webhook-timestamp: 1714556800`
  - Header: `webhook-signature: v1,base64(hmac_sha256(secret, msg_id + "." + timestamp + "." + body))`
- Support secret format prefixes (`whsec_...` or raw strings).
- Verifier helper with timing-attack resistant signature comparison (`hmac.Equal`).
- Verification code snippets generator for Node.js, Python, and Go.

### Phase 3: Hardened Dispatcher with SSRF and Timeout Protection
- Custom `http.Client` with 5s overall timeout and 2s TLS handshake timeout.
- Follow-redirect disabled to prevent SSRF redirect smuggling.
- Anti-SSRF validator resolving target host to IP addresses and blocking:
  - Loopback (`127.0.0.0/8`, `::1`)
  - Private networks (`10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`)
  - Link-local and cloud metadata (`169.254.0.0/16`)
  - Multicast and broadcast blocks
  - Can be bypassed via `--allow-private-ips` flag for local development and integration tests.
- Response recorder capturing HTTP status, latency in milliseconds, response headers, and up to 4KB of response body.

### Phase 4: Retry State Machine and Circuit Breaker
- Exponential backoff schedule with +/-15% randomized jitter:
  - Attempt 1: 5s
  - Attempt 2: 1m
  - Attempt 3: 15m
  - Attempt 4: 1h
  - Attempt 5: 24h
- State transitions:
  - Success (2xx): Mark delivery `SUCCESS`, reset endpoint `consecutive_failures` to 0.
  - Failure (non-2xx or network error):
    - If `attempt_number < max_attempts`: Mark `RETRY`, compute `next_retry_at`.
    - If `attempt_number >= max_attempts`: Mark `FAILED` (moves to Dead Letter Queue).
    - Increment endpoint `consecutive_failures`.
- Per-endpoint circuit breaker:
  - When `consecutive_failures >= 50`, endpoint status changes to `SUSPENDED` (circuit open).
  - Cooldown timer allows single probe attempt after cooldown period.
  - Admin and portal endpoints support manual circuit breaker reset.
- Polling scheduler loop: finds eligible `RETRY` records and dispatches them through the worker pool.

### Phase 5: REST API and Ingestion Endpoints
- `POST /api/v1/publish`:
  - Accepts `endpoint_id` (direct dispatch) or `event_type` (fan-out to all subscribed active endpoints).
  - Validates JSON payload (max 512KB).
  - Honors `Idempotency-Key` header.
- `GET /api/v1/endpoints`: List endpoints with status, metrics, failure counts.
- `POST /api/v1/endpoints`: Create destination endpoint.
- `GET /api/v1/endpoints/:id`: Retrieve endpoint details.
- `PUT /api/v1/endpoints/:id`: Update endpoint URL, events, or status.
- `DELETE /api/v1/endpoints/:id`: Remove endpoint.
- `POST /api/v1/endpoints/:id/rotate-secret`: Generate new HMAC secret.
- `POST /api/v1/endpoints/:id/reset-circuit-breaker`: Clear failures and restore status to active.
- `POST /api/v1/endpoints/:id/test-ping`: Dispatch immediate test ping webhook.
- `GET /api/v1/deliveries`: Query deliveries with status, endpoint, and event filters.
- `GET /api/v1/deliveries/:id`: Full delivery attempt details with headers and payloads.
- `POST /api/v1/deliveries/:id/replay`: Immediate manual replay.
- `POST /api/v1/deliveries/replay-dlq`: Batch replay all dead-letter deliveries.
- `GET /api/v1/portal/token`: Generate scoped token for an endpoint.
- `GET /api/v1/portal/view`: Scoped deliveries and metrics for customer portal.
- `POST /api/v1/portal/retry`: Scoped customer retry action.

### Phase 6: Modern React 19 Web Studio
- Built with React 19, TypeScript, Tailwind CSS, and Lucide icons.
- Single-page application served via embedded Go filesystem at `:9770`.
- Tabs:
  - **Overview / Dashboard**: Real-time stats (Dispatched, Successful, Retrying, Failed, Avg Latency), delivery charts.
  - **Delivery Stream**: Live log table, status pills, search by message or event, quick inspection.
  - **Dead Letter Queue**: Failed webhooks review with 1-click batch replay.
  - **Endpoints**: Target list, circuit breaker status cards, health indicators, secret rotation.
  - **Test Simulator**: Form to publish test webhooks directly from the UI.
  - **Customer Portal View**: Customer-facing mode accessed via `?portal_token=...` or standalone `/portal/:token`.

### Phase 7: Verification and Benchmarks
- Automated unit and integration tests:
  - HMAC signing and verifier tests against Standard Webhooks test vectors.
  - State machine transition tests (success, retry schedule, DLQ exhaustion).
  - Circuit breaker trip and reset tests.
  - Ingestion API with idempotency and fan-out tests.
  - Dispatcher SSRF filtering and mock server delivery tests.
- End-to-end benchmark test verifying throughput and memory usage under load.
