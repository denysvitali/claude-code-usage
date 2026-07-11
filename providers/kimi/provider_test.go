package kimi

import (
	"testing"
)

func TestFormatSubscriptionStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"SUBSCRIPTION_STATUS_ACTIVE", "Active"},
		{"SUBSCRIPTION_STATUS_CANCELLED", "Cancelled"},
		{"SUBSCRIPTION_STATUS_EXPIRED", "Expired"},
		{"UNKNOWN_STATUS", "UNKNOWN_STATUS"},
	}

	for _, tc := range tests {
		result := formatSubscriptionStatus(tc.input)
		if result != tc.expected {
			t.Errorf("formatSubscriptionStatus(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestFormatMembershipLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"LEVEL_BASIC", "Basic"},
		{"LEVEL_STANDARD", "Standard"},
		{"LEVEL_PREMIUM", "Premium"},
		{"LEVEL_CUSTOM", "CUSTOM"},
	}

	for _, tc := range tests {
		result := formatMembershipLevel(tc.input)
		if result != tc.expected {
			t.Errorf("formatMembershipLevel(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestFormatFeatureName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"FEATURE_CODING", "Coding"},
		{"FEATURE_CHAT", "Chat"},
		{"FEATURE_API", "Api"},
		{"", ""},
	}

	for _, tc := range tests {
		result := formatFeatureName(tc.input)
		if result != tc.expected {
			t.Errorf("formatFeatureName(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestProvider_FormatScopeLabel(t *testing.T) {
	p := &Provider{}

	tests := []struct {
		input    string
		expected string
	}{
		{"FEATURE_CODING", "Feature Coding"},
		{"RATE_LIMIT", "Rate Limit"},
		{"single", "Single"},
	}

	for _, tc := range tests {
		result := p.formatScopeLabel(tc.input)
		if result != tc.expected {
			t.Errorf("formatScopeLabel(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestProvider_FormatDurationLabel(t *testing.T) {
	p := &Provider{}

	tests := []struct {
		duration int
		timeUnit string
		expected string
	}{
		{5, "TIME_UNIT_MINUTE", "5-Minute Rate Limit"},
		{1, "TIME_UNIT_HOUR", "1-Hour Rate Limit"},
		{24, "TIME_UNIT_HOURS", "24-Hour Rate Limit"},
	}

	for _, tc := range tests {
		result := p.formatDurationLabel(tc.duration, tc.timeUnit)
		if result != tc.expected {
			t.Errorf("formatDurationLabel(%d, %q) = %q, want %q", tc.duration, tc.timeUnit, result, tc.expected)
		}
	}
}

func TestProvider_FormatSubscriptionExtra(t *testing.T) {
	p := &Provider{}

	sub := &SubscriptionResponse{
		Subscribed: true,
		Subscription: &Subscription{
			SubscriptionID: "sub_123",
			CurrentEndTime: "2026-02-01T00:00:00Z",
			Status:         "SUBSCRIPTION_STATUS_ACTIVE",
			Goods: Goods{
				Title:           "Moderato",
				MembershipLevel: "LEVEL_BASIC",
			},
		},
		Memberships: []Membership{
			{
				Feature:    "FEATURE_CODING",
				LeftCount:  15,
				TotalCount: 20,
			},
		},
	}

	result := p.formatSubscriptionExtra(sub)

	// Check subscribed flag
	if subscribed, ok := result["subscribed"].(bool); !ok || !subscribed {
		t.Error("Expected subscribed to be true")
	}

	// Check plan
	plan, ok := result["plan"].(map[string]any)
	if !ok {
		t.Fatal("Expected plan to be a map")
	}
	if plan["title"] != "Moderato" {
		t.Errorf("Expected title to be 'Moderato', got %v", plan["title"])
	}
	if plan["level"] != "Basic" {
		t.Errorf("Expected level to be 'Basic', got %v", plan["level"])
	}
	if plan["status"] != "Active" {
		t.Errorf("Expected status to be 'Active', got %v", plan["status"])
	}

	// Check features
	features, ok := result["features"].([]map[string]any)
	if !ok {
		t.Fatal("Expected features to be a slice of maps")
	}
	if len(features) != 1 {
		t.Fatalf("Expected 1 feature, got %d", len(features))
	}
	if features[0]["feature"] != "Coding" {
		t.Errorf("Expected feature name to be 'Coding', got %v", features[0]["feature"])
	}
	if features[0]["left"] != 15 {
		t.Errorf("Expected left to be 15, got %v", features[0]["left"])
	}
	if features[0]["total"] != 20 {
		t.Errorf("Expected total to be 20, got %v", features[0]["total"])
	}
}
