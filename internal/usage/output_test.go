package usage

import (
	"errors"
	"testing"

	"github.com/denysvitali/llm-usage/provider"
)

func TestConciseError(t *testing.T) {
	tests := []struct {
		message string
		want    string
	}{
		{"Kimi: API request failed with status 401", "authentication required"},
		{"token expired and refresh failed: invalid_grant", "authentication required"},
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
		{Provider: "codex", Windows: []provider.UsageWindow{{Utilization: 20}}},
		{Provider: "kimi", Error: errors.New("401")},
		{Provider: "kimi", Error: errors.New("401")},
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
