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
	SignatureVersion = "v1"
	SecretPrefix     = "whsec_"
	DefaultTolerance = 300 // 5 minutes
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

// BuildHeaders constructs the 3 standard webhook delivery headers.
func BuildHeaders(msgID string, now time.Time, payload []byte, secret string) (map[string]string, error) {
	ts := now.Unix()
	sig, err := Sign(msgID, ts, payload, secret)
	if err != nil {
		return nil, err
	}

	return map[string]string{
		"webhook-id":        msgID,
		"webhook-timestamp": strconv.FormatInt(ts, 10),
		"webhook-signature": sig,
		"Content-Type":      "application/json",
		"User-Agent":        "HookRelay/1.0",
	}, nil
}
