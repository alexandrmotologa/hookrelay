# HookRelay API Specification

Base URL: `http://localhost:9770/api/v1`

All request and response bodies use JSON unless specified otherwise.

---

## 1. Webhook Ingestion

### Publish an Event
Ingest a webhook payload for delivery to endpoints.

- **Method**: `POST`
- **Path**: `/publish`
- **Headers**:
  - `Content-Type: application/json`
  - `Idempotency-Key` (optional, string): Unique identifier to avoid duplicate messages within 24 hours.
- **Request Body**:
```json
{
  "event_type": "invoice.paid",
  "payload": {
    "id": "inv_10928",
    "amount": 4200,
    "currency": "usd",
    "customer_id": "cus_99182"
  },
  "endpoint_id": "ep_01h8k... (optional, if omitted fan-out applies)"
}
```

- **Response `202 Accepted`**:
```json
{
  "message_id": "msg_01h8kz...",
  "event_type": "invoice.paid",
  "deliveries_queued": 2,
  "created_at": "2026-11-01T21:40:00Z"
}
```

---

## 2. Endpoint Management

### List Endpoints
Retrieve registered destination endpoints with current circuit breaker state, custom headers, and rate limits.

- **Method**: `GET`
- **Path**: `/endpoints`
- **Response `200 OK`**:
```json
[
  {
    "id": "ep_01h8kx...",
    "name": "Stripe Sync Receiver",
    "url": "https://api.example.com/webhooks",
    "secret": "whsec_mfX4...",
    "previous_secret": "whsec_old...",
    "secret_rotated_at": "2026-11-01T20:00:00Z",
    "event_types": ["invoice.paid", "customer.created"],
    "custom_headers": {
      "Authorization": "Bearer token_123"
    },
    "rate_limit": 50,
    "status": "ACTIVE",
    "consecutive_failures": 0,
    "tripped_at": null,
    "created_at": "2026-11-01T12:00:00Z",
    "updated_at": "2026-11-01T12:00:00Z"
  }
]
```

### Create Endpoint
Register a new customer destination.

- **Method**: `POST`
- **Path**: `/endpoints`
- **Request Body**:
```json
{
  "name": "Billing Webhook Handler",
  "url": "https://api.example.com/webhooks",
  "secret": "whsec_optional_custom_secret (if omitted, generated automatically)",
  "event_types": ["payment.*", "invoice.paid"],
  "custom_headers": {
    "Authorization": "Bearer secret_api_token"
  },
  "rate_limit": 100
}
```

### Rotate Secret (Dual Signing Grace Period)
Generates a new HMAC secret and preserves the previous secret for 24 hours. During this grace period, outbound webhooks carry both signatures: `webhook-signature: v1,<new_sig> v1,<old_sig>`.

- **Method**: `POST`
- **Path**: `/endpoints/:id/rotate-secret`
- **Response `200 OK`**:
```json
{
  "endpoint_id": "ep_01h8kx...",
  "secret": "whsec_newSecretPayload...",
  "previous_secret": "whsec_oldSecretPayload...",
  "secret_rotated_at": "2026-11-01T23:00:00Z",
  "grace_period": "24h"
}
```

### Reset Circuit Breaker
Manually clear consecutive failures and reset an endpoint from `SUSPENDED` back to `ACTIVE`.

- **Method**: `POST`
- **Path**: `/endpoints/:id/reset-circuit-breaker`
- **Response `200 OK`**:
```json
{
  "endpoint_id": "ep_01h8kx...",
  "status": "ACTIVE",
  "consecutive_failures": 0
}
```

### Test Ping
Dispatches a test `ping` event directly to the endpoint URL.

- **Method**: `POST`
- **Path**: `/endpoints/:id/test-ping`

---

## 3. Real-Time Stream & Deliveries

### Server-Sent Events (SSE) Live Feed
Establishes a persistent streaming connection delivering real-time webhook updates with sub-10ms latency.

- **Method**: `GET`
- **Path**: `/events/stream`
- **Headers**:
  - `Accept: text/event-stream`
- **Events**:
  - `connected`: Sent on initial handshake.
  - `delivery`: Dispatched whenever a delivery is scheduled, transitions state, or completes.

### Export Delivery Audit Logs
Export delivery records in CSV or JSON format for compliance and dispute resolution.

- **Method**: `GET`
- **Path**: `/deliveries/export?format=csv&status=FAILED&limit=1000`
- **Formats**: `csv`, `json`

### List Deliveries
- **Method**: `GET`
- **Path**: `/deliveries?status=FAILED&endpoint_id=ep_123&limit=50&offset=0`

### Get Delivery Details
- **Method**: `GET`
- **Path**: `/deliveries/:id`

### Replay Single Delivery
- **Method**: `POST`
- **Path**: `/deliveries/:id/replay`

### Batch Replay Dead Letter Queue
- **Method**: `POST`
- **Path**: `/deliveries/replay-dlq`

---

## 4. Built-in Mock Receiver & Chaos Simulator

Built-in endpoint inside HookRelay for local testing without external tunnel tools.

- **Method**: `POST` or `GET`
- **Path**: `/mock-receiver?mode=success|error|rate_limit|timeout|chaos`
- **Modes**:
  - `success`: Responds with HTTP 200 OK.
  - `error`: Responds with HTTP 500 Internal Server Error to trigger retries.
  - `rate_limit`: Responds with HTTP 429 and header `Retry-After: 10`.
  - `timeout`: Delays 6 seconds to trigger 5-second connect timeout.
  - `chaos`: Random fault injection (50% 200, 25% 500, 15% 429, 10% timeout).

### View Mock Receipts History
- **Method**: `GET`
- **Path**: `/mock-receiver/history`

### Clear Mock Receipts
- **Method**: `POST`
- **Path**: `/mock-receiver/clear`

---

## 5. Signature Debugger

Verify and diagnose cryptographic HMAC-SHA256 signatures with detailed error breakdowns.

- **Method**: `POST`
- **Path**: `/verifier/debug`
- **Request Body**:
```json
{
  "message_id": "msg_01h8kz...",
  "timestamp": "1730498400",
  "payload": "{\"test\":true}",
  "secret": "whsec_...",
  "signature_header": "v1,abc..."
}
```
- **Response `200 OK`**:
```json
{
  "valid": true,
  "expected_signature": "v1,abc...",
  "computed_payload_to_sign": "msg_01h8kz....1730498400.{\"test\":true}",
  "timestamp_valid": true,
  "timestamp_drift_seconds": 14,
  "parsed_secret_bytes_len": 32,
  "error": ""
}
```

---

## 6. Customer Self-Serve Portal

### Generate Scoped Portal Token
- **Method**: `POST`
- **Path**: `/portal/token`

### Query Portal Delivery Feed
- **Method**: `GET`
- **Path**: `/portal/view?token=...`

### Customer Self-Serve Retry
- **Method**: `POST`
- **Path**: `/portal/retry?token=...&delivery_id=...`
