package dispatcher

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSSRFBlocking(t *testing.T) {
	// Blocked URLs
	blocked := []string{
		"http://127.0.0.1:8080/webhook",
		"http://localhost/webhook",
		"http://169.254.169.254/latest/meta-data",
		"http://10.0.0.5/api",
		"http://192.168.1.1/test",
		"ftp://example.com/test",
	}

	for _, u := range blocked {
		_, err := ValidateURL(u, false)
		if err == nil {
			t.Errorf("expected URL %s to be blocked by anti-SSRF, but passed", u)
		}
	}

	// Allowed with allowPrivate = true
	_, err := ValidateURL("http://127.0.0.1:8080/webhook", true)
	if err != nil {
		t.Errorf("expected 127.0.0.1 to be allowed when allowPrivate=true, got: %v", err)
	}
}

func TestExecute_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("webhook-id") != "msg_123" {
			http.Error(w, "missing webhook-id header", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"received":true}`))
	}))
	defer ts.Close()

	client := NewHardenedClient(2*time.Second, true)
	req := &DeliveryRequest{
		EndpointURL: ts.URL,
		Headers: map[string]string{
			"webhook-id":   "msg_123",
			"Content-Type": "application/json",
		},
		Payload: []byte(`{"test":"payload"}`),
	}

	res, err := client.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected client error: %v", err)
	}
	if !res.Success || res.StatusCode != 200 {
		t.Fatalf("expected 200 success, got %d (%s)", res.StatusCode, res.ErrorMessage)
	}
	if res.ResponseBody != `{"received":true}` {
		t.Fatalf("unexpected response body: %s", res.ResponseBody)
	}
	if res.DurationMs < 0 {
		t.Fatalf("expected positive duration")
	}
}

func TestExecute_RedirectDisabled(t *testing.T) {
	tsRedirectTarget := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer tsRedirectTarget.Close()

	tsOrigin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, tsRedirectTarget.URL, http.StatusFound)
	}))
	defer tsOrigin.Close()

	client := NewHardenedClient(2*time.Second, true)
	req := &DeliveryRequest{
		EndpointURL: tsOrigin.URL,
		Payload:     []byte(`{}`),
	}

	res, err := client.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Success {
		t.Fatalf("expected redirect to fail delivery, but marked success")
	}
}
