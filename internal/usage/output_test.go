package usage

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/denysvitali/llm-usage/internal/credentials"
	"github.com/denysvitali/llm-usage/provider"
)

func TestConciseError(t *testing.T) {
	tests := []struct {
		message string
		want    string
	}{
		{"Kimi: API request failed with status 401", conciseAuthRequired},
		{"token expired and refresh failed: invalid_grant", conciseAuthRequired},
		{"request timeout", "request timed out"},
		{"HTTP 429", "rate limited"},
		{"unexpected response", "temporarily unavailable"},
	}
	for _, tt := range tests {
		if got := conciseError(errors.New(tt.message)); got != tt.want {
			t.Errorf("conciseError(%q) = %q, want %q", tt.message, got, tt.want)
		}
	}
}

func TestSummaryCountsDeduplicatesFailures(t *testing.T) {
	stats := &provider.UsageStats{Providers: []provider.Usage{
		{Provider: credentials.ProviderCodex, Windows: []provider.UsageWindow{{Utilization: 20}}},
		{Provider: credentials.ProviderKimi, Error: errors.New("401")},
		{Provider: credentials.ProviderKimi, Error: errors.New("401")},
	}}
	healthy, unavailable := summaryCounts(stats)
	if healthy != 1 || unavailable != 1 {
		t.Fatalf("summaryCounts() = %d, %d", healthy, unavailable)
	}
}

func TestSubscriptionFeaturesAcceptProviderShape(t *testing.T) {
	features := subscriptionFeatures([]map[string]any{{"feature": "Coding"}})
	if len(features) != 1 || features[0]["feature"] != "Coding" {
		t.Fatalf("unexpected features: %#v", features)
	}
}

func TestProviderWaybarIcon(t *testing.T) {
	if got := ProviderWaybarIcon("codex"); got != "<span font_family=\"Font Awesome 7 Brands\">\ue7cf</span>" {
		t.Fatalf("Codex icon = %q", got)
	}
	if got := ProviderWaybarIcon("grok"); got != "𝕏" {
		t.Fatalf("Grok icon = %q", got)
	}
}

func TestOutputRawKey(t *testing.T) {
	if got := rawOutputKey("kimi", "work"); got != "kimi/work" {
		t.Fatalf("rawOutputKey(non-default) = %q, want kimi/work", got)
	}
	if got := rawOutputKey(credentials.ProviderCodex, credentials.DefaultAccountName); got != credentials.ProviderCodex {
		t.Fatalf("rawOutputKey(default) = %q, want %s", got, credentials.ProviderCodex)
	}
	if got := rawOutputKey(credentials.ProviderClaude, ""); got != credentials.ProviderClaude {
		t.Fatalf("rawOutputKey(empty) = %q, want %s", got, credentials.ProviderClaude)
	}
	if got := rawOutputKey(credentials.ProviderClaude, 123); got != credentials.ProviderClaude {
		t.Fatalf("rawOutputKey(non-string) = %q, want %s", got, credentials.ProviderClaude)
	}
}

func TestOutputRawEmitsMap(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { os.Stdout = old }()

	stats := &provider.UsageStats{
		Providers: []provider.Usage{
			{Provider: credentials.ProviderCodex, Extra: map[string]any{"raw": json.RawMessage(`{"limit":42}`), "account": credentials.DefaultAccountName}},
			{Provider: credentials.ProviderKimi, Extra: map[string]any{"raw": json.RawMessage(`{"usage":1}`), "account": "work"}},
			{Provider: credentials.ProviderClaude, Error: errors.New("boom")},
		},
	}
	OutputRaw(stats)

	_ = w.Close()
	out, _ := io.ReadAll(r)
	os.Stdout = old

	var result map[string]json.RawMessage
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("OutputRaw output is not valid JSON: %v\n%s", err, string(out))
	}
	if _, ok := result[credentials.ProviderCodex]; !ok {
		t.Fatalf("expected codex key, got %s", string(out))
	}
	if _, ok := result["kimi/work"]; !ok {
		t.Fatalf("expected kimi/work key, got %s", string(out))
	}
	if _, ok := result["claude"]; ok {
		t.Fatalf("errored provider should be omitted, got %s", string(out))
	}
}

func TestOutputRawEmpty(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { os.Stdout = old }()

	OutputRaw(&provider.UsageStats{})

	_ = w.Close()
	out, _ := io.ReadAll(r)
	os.Stdout = old

	if strings.TrimSpace(string(out)) != "{}" {
		t.Fatalf("expected empty object, got %q", string(out))
	}
}
