package provider

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"
)

const retryAfterHeader = "Retry-After"

func TestRetryAfterHeaders(t *testing.T) {
	future := time.Now().Add(90 * time.Second)

	tests := []struct {
		name   string
		header http.Header
		want   time.Duration
	}{
		{
			name:   "no headers falls back to the default backoff",
			header: http.Header{},
			want:   DefaultRateLimitBackoff,
		},
		{
			name:   "Retry-After in seconds",
			header: http.Header{retryAfterHeader: {"30"}},
			want:   30 * time.Second,
		},
		{
			name:   "Retry-After as an HTTP date",
			header: http.Header{retryAfterHeader: {future.UTC().Format(http.TimeFormat)}},
			want:   90 * time.Second,
		},
		{
			name:   "anthropic reset as a unix timestamp",
			header: http.Header{"Anthropic-Ratelimit-Unified-Reset": {fmt.Sprint(future.Unix())}},
			want:   90 * time.Second,
		},
		{
			name:   "anthropic reset as RFC3339",
			header: http.Header{"Anthropic-Ratelimit-Requests-Reset": {future.UTC().Format(time.RFC3339)}},
			want:   90 * time.Second,
		},
		{
			name:   "reset in the past falls back rather than retrying immediately",
			header: http.Header{retryAfterHeader: {"-5"}},
			want:   DefaultRateLimitBackoff,
		},
		{
			name:   "an absurd cooldown is capped",
			header: http.Header{retryAfterHeader: {"999999"}},
			want:   MaxRateLimitBackoff,
		},
		{
			name:   "an unparseable value falls back to the default backoff",
			header: http.Header{retryAfterHeader: {"soon"}},
			want:   DefaultRateLimitBackoff,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := retryAfter(test.header)
			// Header-derived deadlines are relative to time.Now(), so allow
			// for the time spent inside the call.
			if diff := got - test.want; diff > 2*time.Second || diff < -2*time.Second {
				t.Fatalf("retryAfter = %s, want ~%s", got, test.want)
			}
		})
	}
}

func TestRetryAtUnwrapsWrappedErrors(t *testing.T) {
	deadline := time.Now().Add(time.Minute)
	wrapped := fmt.Errorf("claude: %w", &RateLimitError{Provider: "claude", RetryAt: deadline})

	got, ok := RetryAt(wrapped)
	if !ok {
		t.Fatal("RetryAt did not recognize a wrapped rate-limit error")
	}
	if !got.Equal(deadline) {
		t.Fatalf("RetryAt = %s, want %s", got, deadline)
	}

	if _, ok := RetryAt(errors.New("boom")); ok {
		t.Fatal("RetryAt reported a plain error as rate limited")
	}
}

func TestRateLimitErrorMessage(t *testing.T) {
	err := NewRateLimitError("claude", http.Header{retryAfterHeader: {"45"}}, "  too many requests  ")
	message := err.Error()

	if want := "rate limited (HTTP 429)"; len(message) < len(want) || message[:len(want)] != want {
		t.Fatalf("message = %q, want it to start with %q", message, want)
	}
	if err.Detail != "too many requests" {
		t.Fatalf("Detail = %q, want the trimmed upstream body", err.Detail)
	}
}
