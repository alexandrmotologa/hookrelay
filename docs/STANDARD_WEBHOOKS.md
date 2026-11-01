# Standard Webhooks Cryptographic Specification

HookRelay signs every outbound HTTP webhook request following the Standard Webhooks specification. This standard protects receiving customer endpoints against spoofing, data tampering, and replay attacks.

---

## 1. Outbound HTTP Headers

Every HTTP POST delivery from HookRelay includes three cryptographic headers:

| Header | Description | Example |
| :--- | :--- | :--- |
| `webhook-id` | Unique identifier of the message | `msg_01jh823...` |
| `webhook-timestamp` | Integer Unix timestamp in seconds when the request was initiated | `1730498400` |
| `webhook-signature` | Comma-separated signature version and base64 digest | `v1,4V0q...` |

---

## 2. Signature Calculation

The signature is computed as follows:

1. Construct the payload string to sign:
   ```
   PayloadToSign = MessageID + "." + Timestamp + "." + RawBody
   ```
2. Strip any `whsec_` prefix from the shared secret if present. If the remaining secret is base64-encoded, decode it to raw bytes. If not valid base64, use the UTF-8 bytes directly.
3. Compute the HMAC-SHA256 digest of `PayloadToSign` using the secret key.
4. Base64-encode the resulting binary digest.
5. Prefix with `v1,` to form the `webhook-signature` header value.

---

## 3. Timestamp Replay Tolerance

To defend against replay attacks where an attacker captures a valid webhook and replays it later:
1. Parse the integer timestamp from the `webhook-timestamp` header.
2. Reject requests whose timestamp is older than 5 minutes (300 seconds) compared to current server time.
3. Reject requests whose timestamp is in the future by more than 5 minutes.

---

## 4. Verification Code Examples

### Node.js (TypeScript / JavaScript)

```javascript
import crypto from "crypto";

export function verifyWebhookSignature({ id, timestamp, body, signatureHeader, secret }) {
  const toleranceSeconds = 300;
  const now = Math.floor(Date.now() / 1000);
  const ts = parseInt(timestamp, 10);

  if (Math.abs(now - ts) > toleranceSeconds) {
    throw new Error("Webhook timestamp outside acceptable tolerance");
  }

  // Handle whsec_ prefix
  let cleanSecret = secret.startsWith("whsec_") ? secret.slice(6) : secret;
  let secretBytes;
  try {
    secretBytes = Buffer.from(cleanSecret, "base64");
  } catch {
    secretBytes = Buffer.from(cleanSecret, "utf-8");
  }

  const payloadToSign = `${id}.${timestamp}.${body}`;
  const hmac = crypto.createHmac("sha256", secretBytes);
  hmac.update(payloadToSign);
  const expectedSig = hmac.digest("base64");

  // Check all signature candidates in the header (v1,sig1 v1,sig2)
  const signatures = signatureHeader.split(" ");
  const match = signatures.some((sig) => {
    const parts = sig.split(",");
    if (parts.length === 2 && parts[0] === "v1") {
      const candidate = parts[1];
      if (candidate.length === expectedSig.length) {
        return crypto.timingSafeEqual(Buffer.from(candidate), Buffer.from(expectedSig));
      }
    }
    return false;
  });

  if (!match) {
    throw new Error("Invalid webhook signature");
  }

  return true;
}
```

### Python 3

```python
import base64
import hmac
import hashlib
import time

def verify_webhook_signature(msg_id: str, timestamp: str, body: bytes, signature_header: str, secret: str) -> bool:
    tolerance = 300
    now = int(time.time())
    ts = int(timestamp)

    if abs(now - ts) > tolerance:
        raise ValueError("Timestamp outside tolerance window")

    clean_secret = secret[6:] if secret.startswith("whsec_") else secret
    try:
        secret_bytes = base64.b64decode(clean_secret)
    except Exception:
        secret_bytes = clean_secret.encode("utf-8")

    payload_to_sign = f"{msg_id}.{timestamp}.".encode("utf-8") + body
    expected_digest = hmac.new(secret_bytes, payload_to_sign, hashlib.sha256).digest()
    expected_sig = base64.b64encode(expected_digest).decode("utf-8")

    signatures = signature_header.split(" ")
    for sig in signatures:
        parts = sig.split(",")
        if len(parts) == 2 and parts[0] == "v1":
            if hmac.compare_digest(parts[1], expected_sig):
                return True

    raise ValueError("Signature verification failed")
```

### Go

```go
package webhook

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

func VerifySignature(id, timestamp string, body []byte, signatureHeader, secret string) error {
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return errors.New("invalid timestamp header")
	}

	now := time.Now().Unix()
	if math.Abs(float64(now-ts)) > 300 {
		return errors.New("timestamp expired")
	}

	cleanSecret := strings.TrimPrefix(secret, "whsec_")
	secretBytes, err := base64.StdEncoding.DecodeString(cleanSecret)
	if err != nil {
		secretBytes = []byte(cleanSecret)
	}

	mac := hmac.New(sha256.New, secretBytes)
	mac.Write([]byte(fmt.Sprintf("%s.%s.", id, timestamp)))
	mac.Write(body)
	expectedSig := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	for _, sig := range strings.Split(signatureHeader, " ") {
		parts := strings.Split(sig, ",")
		if len(parts) == 2 && parts[0] == "v1" {
			if hmac.Equal([]byte(parts[1]), []byte(expectedSig)) {
				return nil
			}
		}
	}

	return errors.New("signature mismatch")
}
```
