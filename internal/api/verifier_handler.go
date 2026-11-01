package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/alexandrmotologa/hookrelay/internal/signer"
)

type DebugVerifierRequest struct {
	MessageID       string `json:"message_id"`
	Timestamp       string `json:"timestamp"`
	Payload         string `json:"payload"`
	Secret          string `json:"secret"`
	SignatureHeader string `json:"signature_header"`
}

type DebugVerifierResponse struct {
	Valid                 bool     `json:"valid"`
	ExpectedSignature     string   `json:"expected_signature"`
	ComputedPayloadToSign string   `json:"computed_payload_to_sign"`
	TimestampValid        bool     `json:"timestamp_valid"`
	TimestampDriftSeconds int64    `json:"timestamp_drift_seconds"`
	ParsedSecretBytesLen  int      `json:"parsed_secret_bytes_len"`
	ProvidedSignatures    []string `json:"provided_signatures"`
	Error                 string   `json:"error,omitempty"`
}

func HandleDebugVerifier(w http.ResponseWriter, r *http.Request) {
	var req DebugVerifierRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp := DebugVerifierResponse{
		ProvidedSignatures: strings.Fields(req.SignatureHeader),
	}

	if req.Secret == "" {
		resp.Error = "secret key is required"
		writeJSON(w, http.StatusOK, resp)
		return
	}

	// 1. Secret parsing check
	key, err := signer.ParseSecret(req.Secret)
	if err != nil {
		resp.Error = fmt.Sprintf("invalid secret key: %v", err)
		writeJSON(w, http.StatusOK, resp)
		return
	}
	resp.ParsedSecretBytesLen = len(key)

	// 2. Timestamp check
	ts, err := strconv.ParseInt(req.Timestamp, 10, 64)
	if err != nil {
		resp.Error = fmt.Sprintf("invalid timestamp: %v", err)
		writeJSON(w, http.StatusOK, resp)
		return
	}

	now := time.Now().Unix()
	drift := int64(math.Abs(float64(now - ts)))
	resp.TimestampDriftSeconds = drift
	resp.TimestampValid = drift <= 300 // within 5 minutes

	// 3. Payload to sign
	payloadToSign := fmt.Sprintf("%s.%s.%s", req.MessageID, req.Timestamp, req.Payload)
	resp.ComputedPayloadToSign = payloadToSign

	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(payloadToSign))
	expectedSig := fmt.Sprintf("v1,%s", base64.StdEncoding.EncodeToString(mac.Sum(nil)))
	resp.ExpectedSignature = expectedSig

	// 4. Compare with provided signatures
	for _, sig := range resp.ProvidedSignatures {
		parts := strings.Split(sig, ",")
		if len(parts) == 2 && parts[0] == "v1" {
			expectedClean := strings.TrimPrefix(expectedSig, "v1,")
			if hmac.Equal([]byte(parts[1]), []byte(expectedClean)) {
				resp.Valid = true
				break
			}
		}
	}

	if !resp.Valid && resp.Error == "" {
		if !resp.TimestampValid {
			resp.Error = fmt.Sprintf("Timestamp drift is %d seconds (max allowed: 300s)", drift)
		} else {
			resp.Error = "Signature mismatch between provided header and computed digest"
		}
	}

	writeJSON(w, http.StatusOK, resp)
}
