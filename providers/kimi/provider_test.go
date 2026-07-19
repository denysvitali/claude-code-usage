package kimi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// sampleUsageResponse is the documented response of
// GET https://api.kimi.com/coding/v1/usages
const sampleUsageResponse = `{
  "usage": {
    "limit": "2048",
    "used": "214",
    "remaining": "1834",
    "resetTime": "2026-01-09T15:23:13.716839300Z"
  },
  "limits": [{
    "window": {"duration": 300, "timeUnit": "TIME_UNIT_MINUTE"},
    "detail": {
      "limit": "200",
      "used": "139",
      "remaining": "61",
      "resetTime": "2026-01-06T13:33:02.717479433Z"
    }
  }]
}`

func TestClient_GetUsage(t *testing.T) {
	var gotMethod, gotPath, gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, sampleUsageResponse)
	}))
	defer server.Close()

	client := NewClient("test-api-key")
	client.baseURL = server.URL

	resp, _, err := client.GetUsage(context.Background())
	if err != nil {
		t.Fatalf("GetUsage() error = %v", err)
	}

	if gotMethod != http.MethodGet {
		t.Errorf("method = %q, want GET", gotMethod)
	}
	if gotPath != usageEndpoint {
		t.Errorf("path = %q, want %q", gotPath, usageEndpoint)
	}
	if gotAuth != "Bearer test-api-key" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer test-api-key")
	}

	if resp.Usage.Limit != "2048" || resp.Usage.Used != "214" || resp.Usage.Remaining != "1834" {
		t.Errorf("usage = %+v, want limit=2048 used=214 remaining=1834", resp.Usage)
	}
	if len(resp.Limits) != 1 {
		t.Fatalf("len(limits) = %d, want 1", len(resp.Limits))
	}
	if resp.Limits[0].Window.Duration != 300 || resp.Limits[0].Window.TimeUnit != "TIME_UNIT_MINUTE" {
		t.Errorf("window = %+v, want duration=300 timeUnit=TIME_UNIT_MINUTE", resp.Limits[0].Window)
	}
}

func TestProvider_GetUsage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != usageEndpoint {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, sampleUsageResponse)
	}))
	defer server.Close()

	p := NewProvider("test-api-key", false)
	p.client.baseURL = server.URL
	p.client.subscriptionBaseURL = server.URL

	usage, err := p.GetUsage(context.Background())
	if err != nil {
		t.Fatalf("GetUsage() error = %v", err)
	}

	if len(usage.Windows) != 2 {
		t.Fatalf("len(windows) = %d, want 2", len(usage.Windows))
	}

	main := usage.Windows[0]
	if main.Label != "Coding" {
		t.Errorf("main window label = %q, want %q", main.Label, "Coding")
	}
	if *main.Limit != 2048 || *main.Used != 214 || *main.Remaining != 1834 {
		t.Errorf("main window = limit %v used %v remaining %v, want 2048/214/1834", *main.Limit, *main.Used, *main.Remaining)
	}
	wantUtil := 214.0 / 2048.0 * 100
	if main.Utilization != wantUtil {
		t.Errorf("main utilization = %v, want %v", main.Utilization, wantUtil)
	}
	if main.ResetsAt == nil {
		t.Error("main window resetsAt is nil")
	}

	rate := usage.Windows[1]
	if rate.Label != "300-Minute Rate Limit" {
		t.Errorf("rate limit window label = %q, want %q", rate.Label, "300-Minute Rate Limit")
	}
	if *rate.Limit != 200 || *rate.Used != 139 || *rate.Remaining != 61 {
		t.Errorf("rate limit window = limit %v used %v remaining %v, want 200/139/61", *rate.Limit, *rate.Used, *rate.Remaining)
	}
}

func TestUsageResponse_Unmarshal(t *testing.T) {
	var resp UsageResponse
	if err := json.Unmarshal([]byte(sampleUsageResponse), &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if resp.Usage.ResetTime != "2026-01-09T15:23:13.716839300Z" {
		t.Errorf("resetTime = %q", resp.Usage.ResetTime)
	}
	if resp.Limits[0].Detail.Remaining != "61" {
		t.Errorf("limits[0].remaining = %q, want 61", resp.Limits[0].Detail.Remaining)
	}
}

func TestFormatSubscriptionStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{subscriptionStatusActive, subscriptionStatusActiveDisplay},
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
		{membershipLevelBasic, membershipLevelBasicDisplay},
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
		{featureCoding, "Coding"},
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
			Status:         subscriptionStatusActive,
			Goods: Goods{
				Title:           "Moderato",
				MembershipLevel: membershipLevelBasic,
			},
		},
		Memberships: []Membership{
			{
				Feature:    featureCoding,
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
