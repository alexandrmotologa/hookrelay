package queue

import (
	"time"

	"github.com/alexandrmotologa/hookrelay/internal/storage"
)

type CircuitBreakerConfig struct {
	FailureThreshold int
	CooldownDuration time.Duration
}

func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold: 50,
		CooldownDuration: 1 * time.Hour,
	}
}

// CanAttempt determines if a delivery may be dispatched to the endpoint.
// Returns allow=true if healthy or during a half-open probe trial.
func CanAttempt(ep *storage.Endpoint, cfg CircuitBreakerConfig) (allow bool, isProbe bool) {
	if ep.Status == storage.EndpointStatusDisabled {
		return false, false
	}
	if ep.Status == storage.EndpointStatusActive {
		return true, false
	}
	if ep.Status == storage.EndpointStatusSuspended {
		if ep.TrippedAt != nil && time.Since(*ep.TrippedAt) >= cfg.CooldownDuration {
			// Probe trial allowed
			return true, true
		}
		return false, false
	}
	return false, false
}
