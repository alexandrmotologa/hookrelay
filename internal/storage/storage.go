package storage

import "context"

// Storage defines the persistent contract for HookRelay.
type Storage interface {
	Close() error

	// Endpoint methods
	CreateEndpoint(ctx context.Context, ep *Endpoint) error
	GetEndpoint(ctx context.Context, id string) (*Endpoint, error)
	ListEndpoints(ctx context.Context) ([]*Endpoint, error)
	UpdateEndpoint(ctx context.Context, ep *Endpoint) error
	DeleteEndpoint(ctx context.Context, id string) error
	IncrementEndpointFailure(ctx context.Context, id string, threshold int) (tripped bool, err error)
	ResetEndpointFailure(ctx context.Context, id string) error
	GetActiveEndpointsForEvent(ctx context.Context, eventType string) ([]*Endpoint, error)

	// Message methods
	CreateMessage(ctx context.Context, msg *Message) error
	GetMessage(ctx context.Context, id string) (*Message, error)
	GetMessageByIdempotencyKey(ctx context.Context, key string) (*Message, error)

	// Delivery methods
	CreateDelivery(ctx context.Context, d *Delivery) error
	GetDelivery(ctx context.Context, id string) (*Delivery, error)
	GetDeliveryWithDetails(ctx context.Context, id string) (*DeliveryWithDetails, error)
	UpdateDeliveryAttempt(ctx context.Context, d *Delivery) error
	ListDeliveries(ctx context.Context, filter DeliveryFilter) ([]*DeliveryWithDetails, int64, error)
	GetPendingDeliveries(ctx context.Context, limit int) ([]*DeliveryWithDetails, error)
	GetDueRetries(ctx context.Context, limit int) ([]*DeliveryWithDetails, error)
	ReplayDelivery(ctx context.Context, deliveryID string) (*Delivery, error)
	ReplayAllDLQ(ctx context.Context) (int, error)

	// Metrics methods
	GetStats(ctx context.Context) (*StatsSummary, error)
}
