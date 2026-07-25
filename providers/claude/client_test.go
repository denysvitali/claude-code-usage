package claude

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/denysvitali/llm-usage/provider"
)

func newTestClient(handler http.HandlerFunc) (*Client, func()) {
	server := httptest.NewServer(handler)
	client := NewClient("test-token")
	client.baseURL = server.URL
	return client, server.Close
}

func TestGetUsageReportsRateLimitWithCooldown(t *testing.T) {
	client, closeServer := newTestClient(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "120")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"type":"rate_limit_error"}}`))
	})
	defer closeServer()

	_, _, err := client.GetUsage(context.Background())
	if err == nil {
		t.Fatal("expected an error for HTTP 429")
	}

	retryAt, ok := provider.RetryAt(err)
	if !ok {
		t.Fatalf("error %v is not a rate-limit error", err)
	}
	if wait := time.Until(retryAt); wait < 110*time.Second || wait > 120*time.Second {
		t.Fatalf("cooldown = %s, want ~120s from Retry-After", wait)
	}
}

func TestGetUsageOtherErrorsAreNotRateLimits(t *testing.T) {
	client, closeServer := newTestClient(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"type":"authentication_error"}}`))
	})
	defer closeServer()

	_, _, err := client.GetUsage(context.Background())
	if err == nil {
		t.Fatal("expected an error for HTTP 401")
	}
	if _, ok := provider.RetryAt(err); ok {
		t.Fatalf("HTTP 401 was reported as a rate limit: %v", err)
	}
}

func TestGetUsageParsesSuccessfulResponse(t *testing.T) {
	client, closeServer := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization = %q", got)
		}
		_, _ = w.Write([]byte(`{"five_hour":{"utilization":42}}`))
	})
	defer closeServer()

	usage, raw, err := client.GetUsage(context.Background())
	if err != nil {
		t.Fatalf("GetUsage failed: %v", err)
	}
	if usage.FiveHour == nil || usage.FiveHour.Utilization != 42 {
		t.Fatalf("five-hour window = %+v, want utilization 42", usage.FiveHour)
	}
	if len(raw) == 0 {
		t.Fatal("raw body was not returned")
	}
}
