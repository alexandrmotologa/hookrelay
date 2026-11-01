package signer

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	SignatureVersion   = "v1"
	SecretPrefix       = "whsec_"
	DefaultTolerance   = 300            // 5 minutes
	DefaultGracePeriod = 24 * time.Hour // 24 hours dual-secret verification grace period
)

// GenerateSecret produces a cryptographically secure 32-byte base64 secret with whsec_ prefix.
func GenerateSecret() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return SecretPrefix + base64.StdEncoding.EncodeToString(bytes), nil
}

// ParseSecret extracts raw secret bytes from either a whsec_ base64 string or raw string.
func ParseSecret(secret string) ([]byte, error) {
	clean := strings.TrimPrefix(secret, SecretPrefix)
	if decoded, err := base64.StdEncoding.DecodeString(clean); err == nil && len(decoded) > 0 {
		return decoded, nil
	}
	if len(clean) == 0 {
		return nil, errors.New("empty secret key")
	}
	return []byte(clean), nil
}

// Sign computes the Standard Webhooks HMAC-SHA256 signature string ("v1,<base64>").
func Sign(msgID string, timestamp int64, payload []byte, secret string) (string, error) {
	key, err := ParseSecret(secret)
	if err != nil {
		return "", err
	}

	mac := hmac.New(sha256.New, key)
	prefix := fmt.Sprintf("%s.%d.", msgID, timestamp)
	mac.Write([]byte(prefix))
	mac.Write(payload)

	digest := mac.Sum(nil)
	sigBase64 := base64.StdEncoding.EncodeToString(digest)
	return fmt.Sprintf("%s,%s", SignatureVersion, sigBase64), nil
}

// BuildHeaders constructs standard webhook delivery headers with primary secret.
func BuildHeaders(msgID string, now time.Time, payload []byte, secret string) (map[string]string, error) {
	return BuildDualHeaders(msgID, now, payload, secret, "", nil, 0)
}

// BuildDualHeaders constructs delivery headers with dual signatures if previousSecret is active within grace period.
func BuildDualHeaders(
	msgID string,
	now time.Time,
	payload []byte,
	primarySecret string,
	previousSecret string,
	rotatedAt *time.Time,
	gracePeriod time.Duration,
) (map[string]string, error) {
	ts := now.Unix()

	primarySig, err := Sign(msgID, ts, payload, primarySecret)
	if err != nil {
		return nil, err
	}

	signatures := []string{primarySig}

	// If previousSecret exists and grace period has not expired, append second signature
	if previousSecret != "" && rotatedAt != nil {
		if gracePeriod <= 0 {
			gracePeriod = DefaultGracePeriod
		}
		if now.Sub(*rotatedAt) <= gracePeriod {
			if prevSig, err := Sign(msgID, ts, payload, previousSecret); err == nil {
				signatures = append(signatures, prevSig)
			}
		}
	}

	return map[string]string{
		"webhook-id":        msgID,
		"webhook-timestamp": strconv.FormatInt(ts, 10),
		"webhook-signature": strings.Join(signatures, " "),
		"Content-Type":      "application/json",
		"User-Agent":        "HookRelay/1.0",
	}, nil
}
