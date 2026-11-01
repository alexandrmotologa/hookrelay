package signer

import (
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestSignAndVerify(t *testing.T) {
	secret, err := GenerateSecret()
	if err != nil {
		t.Fatalf("failed to generate secret: %v", err)
	}
	if !strings.HasPrefix(secret, SecretPrefix) {
		t.Fatalf("expected prefix %s in secret %s", SecretPrefix, secret)
	}

	msgID := "msg_01h8kz99test"
	now := time.Now().UTC()
	payload := []byte(`{"event":"test","status":"ok"}`)

	headers, err := BuildHeaders(msgID, now, payload, secret)
	if err != nil {
		t.Fatalf("failed to build headers: %v", err)
	}

	sigHeader := headers["webhook-signature"]
	tsHeader := headers["webhook-timestamp"]

	// Verify valid signature
	if err := Verify(msgID, tsHeader, payload, sigHeader, secret, 300); err != nil {
		t.Fatalf("expected valid verification, got: %v", err)
	}

	// Verify tampered payload fails
	tamperedPayload := []byte(`{"event":"test","status":"hacked"}`)
	if err := Verify(msgID, tsHeader, tamperedPayload, sigHeader, secret, 300); err == nil {
		t.Fatalf("expected verification failure on tampered payload")
	}

	// Verify expired timestamp fails
	oldTimestamp := strconv.FormatInt(now.Unix()-600, 10)
	if err := Verify(msgID, oldTimestamp, payload, sigHeader, secret, 300); err != ErrTimestampExpired {
		t.Fatalf("expected ErrTimestampExpired, got: %v", err)
	}

	// Verify wrong secret fails
	wrongSecret, _ := GenerateSecret()
	if err := Verify(msgID, tsHeader, payload, sigHeader, wrongSecret, 300); err != ErrInvalidSignature {
		t.Fatalf("expected ErrInvalidSignature, got: %v", err)
	}
}

func TestGenerateSnippets(t *testing.T) {
	secret := "whsec_test123"
	nodeCode := GenerateSnippet("node", secret)
	if !strings.Contains(nodeCode, secret) {
		t.Fatalf("expected snippet to contain secret")
	}
	pyCode := GenerateSnippet("python", secret)
	if !strings.Contains(pyCode, secret) {
		t.Fatalf("expected snippet to contain secret")
	}
	goCode := GenerateSnippet("go", secret)
	if !strings.Contains(goCode, secret) {
		t.Fatalf("expected snippet to contain secret")
	}
}
