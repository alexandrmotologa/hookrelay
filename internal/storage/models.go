package storage

import (
	"encoding/json"
	"time"
)

type EndpointStatus string

const (
	EndpointStatusActive    EndpointStatus = "ACTIVE"
	EndpointStatusSuspended EndpointStatus = "SUSPENDED"
	EndpointStatusDisabled  EndpointStatus = "DISABLED"
)

type DeliveryStatus string

const (
	DeliveryStatusPending  DeliveryStatus = "PENDING"
	DeliveryStatusInFlight DeliveryStatus = "IN_FLIGHT"
	DeliveryStatusSuccess  DeliveryStatus = "SUCCESS"
	DeliveryStatusRetry    DeliveryStatus = "RETRY"
	DeliveryStatusFailed   DeliveryStatus = "FAILED"
)

// Endpoint represents a registered customer destination.
type Endpoint struct {
	ID                  string            `json:"id"`
	Name                string            `json:"name"`
	URL                 string            `json:"url"`
	Secret              string            `json:"secret"`
	PreviousSecret      string            `json:"previous_secret,omitempty"`
	SecretRotatedAt     *time.Time        `json:"secret_rotated_at,omitempty"`
	EventTypes          []string          `json:"event_types"`
	CustomHeaders       map[string]string `json:"custom_headers,omitempty"`
	RateLimit           int               `json:"rate_limit"`
	Status              EndpointStatus    `json:"status"`
	ConsecutiveFailures int               `json:"consecutive_failures"`
	TrippedAt           *time.Time        `json:"tripped_at,omitempty"`
	CreatedAt           time.Time         `json:"created_at"`
	UpdatedAt           time.Time         `json:"updated_at"`
}

// MatchesEvent returns true if the endpoint is subscribed to the given event type.
func (e *Endpoint) MatchesEvent(eventType string) bool {
	if len(e.EventTypes) == 0 {
		return true // Default: subscribe to all if empty
	}
	for _, pattern := range e.EventTypes {
		if pattern == "*" || pattern == eventType {
			return true
		}
		if len(pattern) > 2 && pattern[len(pattern)-2:] == ".*" {
			prefix := pattern[:len(pattern)-1]
			if len(eventType) >= len(prefix) && eventType[:len(prefix)] == prefix {
				return true
			}
		}
	}
	return false
}

// Message represents an ingested event payload.
type Message struct {
	ID             string    `json:"id"`
	IdempotencyKey *string   `json:"idempotency_key,omitempty"`
	EventType      string    `json:"event_type"`
	Payload        string    `json:"payload"`
	CreatedAt      time.Time `json:"created_at"`
}

// Delivery represents an individual dispatch attempt of a message to an endpoint.
type Delivery struct {
	ID              string         `json:"id"`
	MessageID       string         `json:"message_id"`
	EndpointID      string         `json:"endpoint_id"`
	Status          DeliveryStatus `json:"status"`
	AttemptNumber   int            `json:"attempt_number"`
	MaxAttempts     int            `json:"max_attempts"`
	HTTPStatus      *int           `json:"http_status,omitempty"`
	DurationMs      *int64         `json:"duration_ms,omitempty"`
	RequestHeaders  string         `json:"request_headers,omitempty"`
	ResponseHeaders string         `json:"response_headers,omitempty"`
	ResponseBody    string         `json:"response_body,omitempty"`
	ErrorMessage    string         `json:"error_message,omitempty"`
	NextRetryAt     *time.Time     `json:"next_retry_at,omitempty"`
	SentAt          *time.Time     `json:"sent_at,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
}

// DeliveryWithDetails includes joined metadata for UI tables and inspection modals.
type DeliveryWithDetails struct {
	Delivery
	EndpointName string `json:"endpoint_name"`
	EndpointURL  string `json:"endpoint_url"`
	EventType    string `json:"event_type"`
	Payload      string `json:"payload"`
}

// DeliveryFilter provides query constraints.
type DeliveryFilter struct {
	EndpointID string
	Status     string
	EventType  string
	Search     string
	Limit      int
	Offset     int
}

// StatsSummary reports aggregate delivery metrics.
type StatsSummary struct {
	TotalMessages      int64   `json:"total_messages"`
	TotalDeliveries    int64   `json:"total_deliveries"`
	SuccessfulCount    int64   `json:"successful_count"`
	RetryingCount      int64   `json:"retrying_count"`
	FailedCount        int64   `json:"failed_count"`
	PendingCount       int64   `json:"pending_count"`
	SuccessRate        float64 `json:"success_rate"`
	AvgDurationMs      float64 `json:"avg_duration_ms"`
	ActiveEndpoints    int64   `json:"active_endpoints"`
	SuspendedEndpoints int64   `json:"suspended_endpoints"`
}

// Helper to serialize event types to JSON string.
func EncodeEventTypes(types []string) string {
	if len(types) == 0 {
		return "[]"
	}
	bytes, _ := json.Marshal(types)
	return string(bytes)
}

// Helper to deserialize event types from JSON string.
func DecodeEventTypes(data string) []string {
	var types []string
	if data == "" || data == "null" {
		return []string{}
	}
	_ = json.Unmarshal([]byte(data), &types)
	return types
}

// Helper to serialize custom headers to JSON string.
func EncodeCustomHeaders(headers map[string]string) string {
	if len(headers) == 0 {
		return "{}"
	}
	bytes, _ := json.Marshal(headers)
	return string(bytes)
}

// Helper to deserialize custom headers from JSON string.
func DecodeCustomHeaders(data string) map[string]string {
	res := make(map[string]string)
	if data == "" || data == "null" || data == "{}" {
		return res
	}
	_ = json.Unmarshal([]byte(data), &res)
	return res
}
