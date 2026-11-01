package signer

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

var (
	ErrInvalidTimestamp = errors.New("invalid webhook-timestamp header")
	ErrTimestampExpired = errors.New("webhook timestamp outside tolerance window")
	ErrInvalidSignature = errors.New("webhook signature mismatch")
	ErrMissingHeader    = errors.New("missing required webhook header")
)

// Verify checks a Standard Webhooks signature against message details.
func Verify(msgID, timestampStr string, payload []byte, signatureHeader, secret string, toleranceSeconds int64) error {
	if msgID == "" || timestampStr == "" || signatureHeader == "" || secret == "" {
		return ErrMissingHeader
	}

	ts, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return ErrInvalidTimestamp
	}

	if toleranceSeconds <= 0 {
		toleranceSeconds = DefaultTolerance
	}

	now := time.Now().Unix()
	if math.Abs(float64(now-ts)) > float64(toleranceSeconds) {
		return ErrTimestampExpired
	}

	key, err := ParseSecret(secret)
	if err != nil {
		return err
	}

	mac := hmac.New(sha256.New, key)
	prefix := fmt.Sprintf("%s.%s.", msgID, timestampStr)
	mac.Write([]byte(prefix))
	mac.Write(payload)
	expectedSig := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	// signatureHeader can contain space-separated candidates: "v1,abc v1,def"
	signatures := strings.Fields(signatureHeader)
	for _, s := range signatures {
		parts := strings.Split(s, ",")
		if len(parts) == 2 && parts[0] == SignatureVersion {
			candidate := parts[1]
			if hmac.Equal([]byte(candidate), []byte(expectedSig)) {
				return nil
			}
		}
	}

	return ErrInvalidSignature
}

// GenerateSnippet produces copy-pasteable verification code for customer portal users.
func GenerateSnippet(lang, secret string) string {
	switch strings.ToLower(lang) {
	case "node", "javascript", "typescript":
		return fmt.Sprintf(`// Node.js Verification Example
import crypto from "crypto";

export function verifyWebhook({ id, timestamp, body, signatureHeader, secret = "%s" }) {
  const tolerance = 300; // 5 minutes
  const now = Math.floor(Date.now() / 1000);
  const ts = parseInt(timestamp, 10);
  if (Math.abs(now - ts) > tolerance) throw new Error("Timestamp outside tolerance");

  const cleanSecret = secret.startsWith("whsec_") ? secret.slice(6) : secret;
  let secretBytes;
  try {
    secretBytes = Buffer.from(cleanSecret, "base64");
  } catch {
    secretBytes = Buffer.from(cleanSecret, "utf-8");
  }

  const payload = id + "." + timestamp + "." + body;
  const expected = crypto.createHmac("sha256", secretBytes).update(payload).digest("base64");

  const matches = signatureHeader.split(" ").some((sig) => {
    const [v, val] = sig.split(",");
    return v === "v1" && val.length === expected.length && crypto.timingSafeEqual(Buffer.from(val), Buffer.from(expected));
  });

  if (!matches) throw new Error("Signature mismatch");
  return true;
}`, secret)

	case "python":
		return fmt.Sprintf(`// Python 3 Verification Example
import base64
import hashlib
import hmac
import time

def verify_webhook(msg_id: str, timestamp: str, body: bytes, signature_header: str, secret: str = "%s") -> bool:
    tolerance = 300
    now = int(time.time())
    ts = int(timestamp)
    if abs(now - ts) > tolerance:
        raise ValueError("Timestamp outside tolerance")

    clean = secret[6:] if secret.startswith("whsec_") else secret
    try:
        key = base64.b64decode(clean)
    except Exception:
        key = clean.encode("utf-8")

    to_sign = f"{msg_id}.{timestamp}.".encode("utf-8") + body
    expected = base64.b64encode(hmac.new(key, to_sign, hashlib.sha256).digest()).decode("utf-8")

    for sig in signature_header.split(" "):
        parts = sig.split(",")
        if len(parts) == 2 and parts[0] == "v1":
            if hmac.compare_digest(parts[1], expected):
                return True
    raise ValueError("Signature mismatch")`, secret)

	case "go", "golang":
		return fmt.Sprintf(`// Go Verification Example
package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

func Verify(id, timestamp string, body []byte, signatureHeader string) error {
	secret := "%s"
	clean := strings.TrimPrefix(secret, "whsec_")
	key, err := base64.StdEncoding.DecodeString(clean)
	if err != nil {
		key = []byte(clean)
	}

	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil || math.Abs(float64(time.Now().Unix()-ts)) > 300 {
		return errors.New("invalid or expired timestamp")
	}

	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(fmt.Sprintf("%%s.%%s.", id, timestamp)))
	mac.Write(body)
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	for _, sig := range strings.Fields(signatureHeader) {
		parts := strings.Split(sig, ",")
		if len(parts) == 2 && parts[0] == "v1" && hmac.Equal([]byte(parts[1]), []byte(expected)) {
			return nil
		}
	}
	return errors.New("signature mismatch")
}`, secret)

	default:
		return ""
	}
}
