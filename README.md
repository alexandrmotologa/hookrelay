# HookRelay

Self-hosted outbound webhook delivery engine with Standard Webhooks HMAC-SHA256 signing, exponential backoff retries, per-endpoint circuit breaking, and a customer self-serve delivery portal.

HookRelay replaces commercial webhook delivery vendors like Svix or Hookdeck. Applications send internal HTTP calls to `POST /api/v1/publish`, and HookRelay manages signature generation, connection timeouts, exponential retries, and customer visibility.

---

## Key Features

- **Fast Intake API**: Ingests events in under 2ms with optional `Idempotency-Key` deduplication.
- **Fan-Out & Targeted Delivery**: Publish directly to a specific endpoint or fan out to all subscribers matching an event type pattern.
- **Standard Webhooks Signatures**: Signs every outbound request with `webhook-id`, `webhook-timestamp`, and `webhook-signature` using HMAC-SHA256.
- **Durable Exponential Retries**: 5-step schedule (5s, 1m, 15m, 1h, 24h) with randomized jitter (+/-15%) backed by SQLite in WAL mode.
- **Per-Endpoint Circuit Breakers**: Automatically pauses delivery after 50 consecutive failures to prevent downstream flooding and resource exhaustion.
- **SSRF Network Protection**: Blocks requests to loopback, private RFC 1918 subnets, and cloud instance metadata addresses. Rejects HTTP redirects by default.
- **Delivery Log Inspection**: Full request headers, request payload, response status, duration, and response body preview stored for auditing.
- **Customer Self-Serve Portal**: Scoped URL tokens allow end customers to review their delivery status, view code examples for signature verification, and trigger manual retries.
- **Single-Binary Distribution**: Written in Go with pure Go SQLite (`modernc.org/sqlite`) and an embedded React 19 web studio. No external database or runtime required.

---

## Architecture

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

## Installation & Quick Start

### Build from Source

Requirements: Go 1.23+ and Node.js 20+ (for building the web UI).

```bash
# Clone the repository
git clone https://github.com/alexandrmotologa/hookrelay.git
cd hookrelay

# Build frontend and compile binary
npm --prefix web install
npm --prefix web run build
go build -o hookrelay.exe ./cmd/hookrelay

# Start the server
./hookrelay.exe serve --port 9770 --db ./hookrelay.db
```

The Web Studio opens at `http://localhost:9770`.

---

## Quick Usage

### 1. Register an Endpoint

```bash
curl -X POST http://localhost:9770/api/v1/endpoints \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Acme Payments Webhook",
    "url": "https://webhook.site/test-receiver",
    "event_types": ["payment.succeeded", "invoice.created"]
  }'
```

Returns:
```json
{
  "id": "ep_01jh823...",
  "name": "Acme Payments Webhook",
  "url": "https://webhook.site/test-receiver",
  "secret": "whsec_x8F2...",
  "status": "ACTIVE"
}
```

### 2. Publish an Event

```bash
curl -X POST http://localhost:9770/api/v1/publish \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: evt_order_9981" \
  -d '{
    "event_type": "payment.succeeded",
    "payload": {
      "order_id": "ord_9981",
      "amount": 4900,
      "currency": "usd"
    }
  }'
```

HookRelay immediately schedules the delivery, signs the payload using HMAC-SHA256, and dispatches the request with the headers:
- `webhook-id`: Message ID
- `webhook-timestamp`: Unix timestamp in seconds
- `webhook-signature`: `v1,<signature>`

---

## Retry Policy & Circuit Breakers

When a customer endpoint returns a non-2xx status or times out after 5 seconds:
1. The delivery is scheduled for retry using exponential backoff with randomized jitter (+/-15%):
   - Attempt 1: 5 seconds
   - Attempt 2: 1 minute
   - Attempt 3: 15 minutes
   - Attempt 4: 1 hour
   - Attempt 5: 24 hours
2. If attempt 5 fails, the delivery transitions to `FAILED` status and moves to the Dead Letter Queue.
3. If an endpoint accumulates 50 consecutive delivery failures, its circuit breaker trips to `SUSPENDED`. New webhooks to this endpoint are halted until the cooldown passes or an operator resets the breaker.

---

## Documentation

- [API Specification](docs/API_SPEC.md)
- [Standard Webhooks Cryptography & Verifier Guides](docs/STANDARD_WEBHOOKS.md)
- [System Architecture & Concurrency Model](docs/ARCHITECTURE.md)

---

## License

MIT License. Copyright (c) 2026 Alexandr Motologa.
