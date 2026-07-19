package codex

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/denysvitali/llm-usage/internal/credentials"
)

func TestToWindow(t *testing.T) {
	w := toWindow("Primary", Window{UsedPercent: 42.5, ResetAt: 1700000000})
	if w.Label != "Primary" || w.Utilization != 42.5 || w.ResetsAt == nil {
		t.Fatalf("unexpected window: %#v", w)
	}
}

func TestClientHonorsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	client := Client{AccessToken: "token", HTTPClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Context().Err() != context.Canceled {
			t.Errorf("request context error = %v", req.Context().Err())
		}
		return nil, req.Context().Err()
	})}}
	_, _, err := client.GetUsage(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("GetUsage() error = %v, want context canceled", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestProviderMetadata(t *testing.T) {
	p := NewProvider("token", "account", false)
	if p.ID() != credentials.ProviderCodex || p.ShortName() != "Cdx" {
		t.Fatalf("unexpected metadata: %s %s", p.ID(), p.ShortName())
	}
}
