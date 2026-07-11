package grok

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripper func(*http.Request) (*http.Response, error)

func (f roundTripper) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

func TestClientNormalizesGrokBuildUsage(t *testing.T) {
	httpClient := &http.Client{Transport: roundTripper(func(request *http.Request) (*http.Response, error) {
		if got := request.Header.Get("Authorization"); got != "Bearer token" {
			t.Errorf("authorization = %q", got)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"config":{"currentPeriod":{"end":"2026-07-15T18:25:25.147706+00:00"},"creditUsagePercent":100,"productUsage":[{"product":"GrokBuild","usagePercent":100}]}}`)), Header: make(http.Header)}, nil
	})}
	client, err := NewClient(ClientOptions{AccessToken: "token", HTTPClient: httpClient, Endpoint: "https://example.test/billing"})
	if err != nil {
		t.Fatal(err)
	}
	usage, err := client.GetUsage(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if usage.Provider != "grok" || len(usage.Windows) != 1 || usage.Windows[0].Label != "Grok Build Weekly" {
		t.Fatalf("unexpected usage: %#v", usage)
	}
}
