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
Retrieve registered destination endpoints with current circuit breaker state and delivery counts.

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
    "event_types": ["invoice.paid", "customer.created"],
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
  "rate_limit": 100
}
```
- **Response `201 Created`**: Returns the created endpoint object with its assigned `id` and generated `secret`.

### Update Endpoint
- **Method**: `PUT`
- **Path**: `/endpoints/:id`
- **Request Body**:
```json
{
  "name": "Billing Webhook Handler Updated",
  "url": "https://api.example.com/v2/webhooks",
  "event_types": ["*"],
  "rate_limit": 150,
  "status": "ACTIVE"
}
```

### Delete Endpoint
- **Method**: `DELETE`
- **Path**: `/endpoints/:id`
- **Response `204 No Content`**

### Rotate Secret
Generates a new HMAC secret for an existing endpoint.

- **Method**: `POST`
- **Path**: `/endpoints/:id/rotate-secret`
- **Response `200 OK`**:
```json
{
  "endpoint_id": "ep_01h8kx...",
  "secret": "whsec_newSecretPayload..."
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
- **Response `200 OK`**:
```json
{
  "delivery_id": "del_01h8ky...",
  "status": "SUCCESS",
  "http_status": 200,
  "duration_ms": 42
}
```

---

## 3. Delivery Inspection & Replay

### List Deliveries
Retrieve delivery attempts with filtering by status and endpoint.

- **Method**: `GET`
- **Path**: `/deliveries?status=FAILED&endpoint_id=ep_123&limit=50&offset=0`
- **Response `200 OK`**:
```json
{
  "total": 142,
  "items": [
    {
      "id": "del_01h8kz...",
      "message_id": "msg_01h8ky...",
      "endpoint_id": "ep_01h8kx...",
      "endpoint_url": "https://api.example.com/webhooks",
      "event_type": "invoice.paid",
      "status": "FAILED",
      "attempt_number": 5,
      "max_attempts": 5,
      "http_status": 503,
      "duration_ms": 128,
      "error_message": "HTTP 503 Service Unavailable",
      "created_at": "2026-11-01T20:10:00Z"
    }
  ]
}
```

### Get Delivery Details
Inspect complete request headers, request body, response headers, and response snippet.

- **Method**: `GET`
- **Path**: `/deliveries/:id`
- **Response `200 OK`**:
```json
{
  "id": "del_01h8kz...",
  "message_id": "msg_01h8ky...",
  "endpoint_id": "ep_01h8kx...",
  "status": "SUCCESS",
  "attempt_number": 1,
  "http_status": 200,
  "duration_ms": 74,
  "request_headers": {
    "Content-Type": "application/json",
    "webhook-id": "msg_01h8ky...",
    "webhook-timestamp": "1714556800",
    "webhook-signature": "v1,g0hM9SsE+..."
  },
  "request_body": "{\"id\":\"inv_10928\",\"amount\":4200}",
  "response_headers": {
    "Content-Type": "application/json",
    "Date": "Sun, 01 Nov 2026 21:40:01 GMT"
  },
  "response_body": "{\"received\":true}",
  "error_message": "",
  "sent_at": "2026-11-01T21:40:01Z"
}
```

### Manual Replay Single Delivery
- **Method**: `POST`
- **Path**: `/deliveries/:id/replay`
- **Response `200 OK`**:
```json
{
  "new_delivery_id": "del_01h8la...",
  "status": "PENDING"
}
```

### Batch Replay Dead Letter Queue
Replay all deliveries currently in `FAILED` status.

- **Method**: `POST`
- **Path**: `/deliveries/replay-dlq`
- **Response `200 OK`**:
```json
{
  "replayed_count": 14
}
```

---

## 4. Customer Self-Serve Portal

### Generate Scoped Portal Token
Generate a signed URL token for an endpoint to give to a customer.

- **Method**: `POST`
- **Path**: `/portal/token`
- **Request Body**:
```json
{
  "endpoint_id": "ep_01h8kx...",
  "expires_in_hours": 72
}
```
- **Response `200 OK`**:
```json
{
  "token": "eyJhbGciOiJIUzI1Ni...",
  "portal_url": "http://localhost:9770/portal/eyJhbGciOiJIUzI1Ni..."
}
```

### Query Portal Delivery Feed
Scoped customer endpoint access using the signed token.

- **Method**: `GET`
- **Path**: `/portal/view?token=...`
- **Response `200 OK`**:
```json
{
  "endpoint": {
    "id": "ep_01h8kx...",
    "name": "Stripe Sync Receiver",
    "url": "https://api.example.com/webhooks",
    "status": "ACTIVE",
    "secret": "whsec_mfX4..."
  },
  "deliveries": [
    {
      "id": "del_01h8kz...",
      "event_type": "invoice.paid",
      "status": "SUCCESS",
      "http_status": 200,
      "duration_ms": 68,
      "created_at": "2026-11-01T21:40:00Z"
    }
  ]
}
```
