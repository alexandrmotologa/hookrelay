package queue

import (
	"crypto/rand"
	"math/big"
	"time"
)

var defaultSchedule = []time.Duration{
	5 * time.Second,  // Attempt 1
	1 * time.Minute,  // Attempt 2
	15 * time.Minute, // Attempt 3
	1 * time.Hour,    // Attempt 4
	24 * time.Hour,   // Attempt 5
}

// CalculateNextRetry returns the next timestamp for a given retry attempt with +/-15% randomized jitter.
func CalculateNextRetry(attemptNumber int) time.Time {
	idx := attemptNumber - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(defaultSchedule) {
		idx = len(defaultSchedule) - 1
	}

	baseDuration := defaultSchedule[idx]
	jittered := ApplyJitter(baseDuration, 0.15)
	return time.Now().UTC().Add(jittered)
}

// ApplyJitter adds or subtracts up to jitterFraction (e.g. 0.15 for 15%) from duration.
func ApplyJitter(d time.Duration, jitterFraction float64) time.Duration {
	if jitterFraction <= 0 {
		return d
	}

	// Generate random float between -jitterFraction and +jitterFraction
	// Range size in basis points: 2 * (fraction * 10000)
	rangeBp := int64(jitterFraction * 20000)
	nBig, err := rand.Int(rand.Reader, big.NewInt(rangeBp))
	if err != nil {
		return d
	}

	// Offset so it centers at 0
	offsetBp := nBig.Int64() - int64(jitterFraction*10000)
	factor := 1.0 + (float64(offsetBp) / 10000.0)

	return time.Duration(float64(d) * factor)
}
