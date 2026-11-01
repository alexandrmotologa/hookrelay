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

func TestDualSignaturesWithGracePeriod(t *testing.T) {
	oldSecret, _ := GenerateSecret()
	newSecret, _ := GenerateSecret()

	rotatedAt := time.Now().UTC().Add(-2 * time.Hour) // rotated 2 hours ago
	now := time.Now().UTC()
	msgID := "msg_dual_test"
	payload := []byte(`{"order_id":"123"}`)

	headers, err := BuildDualHeaders(msgID, now, payload, newSecret, oldSecret, &rotatedAt, 24*time.Hour)
	if err != nil {
		t.Fatalf("failed to build dual headers: %v", err)
	}

	sigHeader := headers["webhook-signature"]
	tsHeader := headers["webhook-timestamp"]

	// The header should contain 2 signatures separated by space
	sigs := strings.Split(sigHeader, " ")
	if len(sigs) != 2 {
		t.Fatalf("expected 2 signatures in header, got %d: %s", len(sigs), sigHeader)
	}

	// Customer verifying with NEW secret -> PASS
	if err := Verify(msgID, tsHeader, payload, sigHeader, newSecret, 300); err != nil {
		t.Fatalf("expected verification with new secret to pass: %v", err)
	}

	// Customer still verifying with OLD secret -> PASS (grace period active)
	if err := Verify(msgID, tsHeader, payload, sigHeader, oldSecret, 300); err != nil {
		t.Fatalf("expected verification with old secret to pass: %v", err)
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
