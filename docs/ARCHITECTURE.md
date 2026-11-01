# HookRelay System Architecture

HookRelay decouples internal event ingestion from the latency and failure states of external HTTP webhook destinations.

---

## 1. Concurrency Model

When an internal service issues `POST /api/v1/publish`, HookRelay accepts the event in under 2ms:
1. Validates the JSON payload size and syntax.
2. Checks for optional idempotency duplicate keys.
3. Resolves destination endpoints based on `endpoint_id` or matching `event_type` subscriptions.
4. Commits the message and initial delivery tasks in SQLite with WAL mode enabled.
5. Emits delivery tasks to an in-memory buffered channel for immediate asynchronous worker dispatch.
6. Returns an HTTP `202 Accepted` response with the assigned message identifier.

Background delivery workers pull from the channel and execute outbound HTTP POST requests concurrently:
- Each worker manages its own goroutine pool up to configured concurrency limits.
- If a delivery fails, the state machine calculates the next retry timestamp with jitter and stores it.
- A scheduler tick loop polls the database for due retries, decoupling worker failures from memory state.

---

## 2. Retry State Machine

Deliveries traverse five distinct states:

```
[ PENDING ] ──> [ IN_FLIGHT ] ──┬──> [ SUCCESS ] (HTTP 2xx)
                                │
                                └──> [ RETRY ] (Attempt < MaxAttempts)
                                        │
                                        └──> [ FAILED / DLQ ] (Attempt == MaxAttempts)
```

- **PENDING**: Registered, waiting for a worker to claim it.
- **IN_FLIGHT**: Actively executing HTTP request over the network.
- **SUCCESS**: Destination returned HTTP 200 through 299.
- **RETRY**: Destination returned non-2xx status, network timeout, or connection error. Scheduled for future retry.
- **FAILED**: All 5 retry attempts exhausted. Moved to Dead Letter Queue for operator or customer review.

### Retry Schedule with Randomized Jitter

Retries use a 5-step schedule. Each step applies a randomized jitter of plus or minus 15 percent to avoid synchronizing retried requests across multiple endpoints:

| Attempt | Base Delay | With Jitter (+/- 15%) |
| :--- | :--- | :--- |
| 1 | 5 seconds | 4.25s - 5.75s |
| 2 | 1 minute | 51s - 69s |
| 3 | 15 minutes | 12m 45s - 17m 15s |
| 4 | 1 hour | 51m - 69m |
| 5 | 24 hours | 20.4h - 27.6h |

---

## 3. Circuit Breaker Mechanism

Destination endpoints that experience sustained downtime can flood connection pools and fail repeatedly. HookRelay attaches a circuit breaker to every registered endpoint:

1. **State CLOSED (Healthy)**: Normal delivery. Every successful delivery resets the `consecutive_failures` counter to 0.
2. **State OPEN (Suspended)**: If `consecutive_failures` reaches 50, the endpoint transitions to `SUSPENDED`. New deliveries to this endpoint are immediately marked as paused without making network requests.
3. **State HALF_OPEN (Probe)**: After a 60-minute cooldown window, the next delivery is allowed through as a probe. If it returns HTTP 2xx, the circuit closes and normal delivery resumes. If it fails, the cooldown resets.
4. **Manual Override**: Operators or customers can trigger a manual circuit breaker reset via API or portal UI.

---

## 4. SSRF Defense Layer

Because outbound webhook destinations are configured by end users, an anti-SSRF layer enforces network boundaries:
1. DNS resolution runs before the connection is established.
2. Resolved IP addresses are checked against restricted ranges:
   - IPv4 Loopback (`127.0.0.0/8`)
   - IPv6 Loopback (`::1`)
   - RFC 1918 Private Ranges (`10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`)
   - Link-local and cloud metadata (`169.254.0.0/16`, `fe80::/10`)
   - Multicast and Broadcast addresses
3. HTTP redirects (301, 302, 307, 308) are rejected by default so an external endpoint cannot redirect the client to an internal IP address.
4. In local development or automated testing, passing `--allow-private-ips` disables this filter.
