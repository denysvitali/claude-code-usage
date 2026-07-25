package usage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/denysvitali/llm-usage/internal/credentials"
	"github.com/denysvitali/llm-usage/provider"
)

// Lipgloss styles for subscription display
var (
	titleStyle               = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("231")).Background(lipgloss.Color("63")).Padding(0, 1)
	providerTitleStyle       = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))
	accountStyle             = lipgloss.NewStyle().Foreground(lipgloss.Color("246"))
	windowLabelStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	valueStyle               = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))
	unavailableTitleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
	unavailableProviderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	errorStyle               = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	dividerStyle             = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	subtitleStyle            = lipgloss.NewStyle().Foreground(lipgloss.Color("246"))
	metricLabelStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Width(8)
	subscriptionTitleStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true)
	statusActiveStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("70"))
	statusCancelledStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	statusExpiredStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	dimStyle                 = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	featureNameStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("75"))
)

const (
	barWidth = 20
	barFull  = "█"
	barEmpty = "░"

	conciseAuthRequired = "authentication required"
)

// WaybarOutput represents the JSON format expected by waybar custom modules
type WaybarOutput struct {
	Text       string `json:"text"`
	Tooltip    string `json:"tooltip"`
	Class      string `json:"class"`
	Percentage int    `json:"percentage"`
}

// OutputWaybar outputs usage stats in waybar JSON format
func OutputWaybar(stats *provider.UsageStats) {
	// Build compact text for the bar
	var textParts []string
	failedProviders := make(map[string]error)
	for _, p := range stats.Providers {
		if p.Error != nil {
			if _, seen := failedProviders[p.Provider]; !seen {
				failedProviders[p.Provider] = p.Error
			}
			continue
		}
		providerLabel := ProviderWaybarIcon(p.Provider)
		if len(p.Windows) > 0 {
			values := make([]string, 0, len(p.Windows))
			for _, window := range p.Windows {
				values = append(values, fmt.Sprintf("%.0f%%", window.Utilization))
			}
			textParts = append(textParts, fmt.Sprintf("%s %s", providerLabel, strings.Join(values, " · ")))
		}
	}
	text := strings.Join(textParts, "  ·  ")

	// Build detailed tooltip
	var tooltipLines []string
	tooltipLines = append(tooltipLines, "LLM Usage")
	tooltipLines = append(tooltipLines, "")

	for _, p := range stats.Providers {
		if p.Error != nil {
			continue
		}

		// Get account name if available
		accountSuffix := ""
		if acc, ok := p.Extra["account"]; ok && acc != "" {
			accountSuffix = fmt.Sprintf(" (%s)", acc)
		}

		for _, w := range p.Windows {
			line := fmt.Sprintf("%s%s %s: %.1f%%", ProviderName(p.Provider), accountSuffix, w.Label, w.Utilization)
			if d := w.TimeUntilReset(); d != nil {
				line += fmt.Sprintf(" (resets in %s)", FormatDuration(*d))
			}
			tooltipLines = append(tooltipLines, line)
		}
	}
	if len(failedProviders) > 0 {
		tooltipLines = append(tooltipLines, "", "Unavailable:")
		for _, id := range sortedProviderIDs(failedProviders) {
			tooltipLines = append(tooltipLines, fmt.Sprintf("%s: %s", ProviderName(id), conciseError(failedProviders[id])))
		}
	}

	output := WaybarOutput{
		Text:       text,
		Tooltip:    strings.Join(tooltipLines, "\n"),
		Class:      stats.GetClass(),
		Percentage: int(stats.MaxUtilization()),
	}

	enc := json.NewEncoder(os.Stdout)
	if err := enc.Encode(output); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
	}
}

// ProviderWaybarIcon returns a compact glyph for status-bar output. The
// OpenAI mark comes from Font Awesome Brands; the xAI mark identifies Grok.
func ProviderWaybarIcon(id string) string {
	switch id {
	case credentials.ProviderCodex:
		return "<span font_family=\"Font Awesome 7 Brands\">\ue7cf</span>"
	case "grok":
		return "𝕏"
	default:
		return ProviderShortName(id)
	}
}

func sortedProviderIDs(failures map[string]error) []string {
	ids := make([]string, 0, len(failures))
	for id := range failures {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids
}

// OutputWaybarError outputs an error in waybar JSON format
func OutputWaybarError(msg string) {
	output := WaybarOutput{
		Text:       "LLM: Error",
		Tooltip:    msg,
		Class:      "error",
		Percentage: 0,
	}
	enc := json.NewEncoder(os.Stdout)
	if err := enc.Encode(output); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
	}
}

// OutputRaw outputs the raw provider API responses as a JSON map keyed by
// provider ID (or "provider/account" when the account is not the default).
func OutputRaw(stats *provider.UsageStats) {
	out := make(map[string]json.RawMessage)
	for _, p := range stats.Providers {
		if p.Error != nil {
			continue
		}
		raw, ok := p.Extra["raw"]
		if !ok {
			continue
		}
		rawMsg, ok := raw.(json.RawMessage)
		if !ok {
			continue
		}
		key := rawOutputKey(p.Provider, p.Extra["account"])
		out[key] = rawMsg
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
	}
}

func rawOutputKey(providerID string, account any) string {
	acc, _ := account.(string)
	if acc != "" && acc != credentials.DefaultAccountName {
		return providerID + "/" + acc
	}
	return providerID
}

// OutputJSON outputs usage stats in JSON format
func OutputJSON(stats *provider.UsageStats) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(stats); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
	}
}

// OutputPretty outputs usage stats in a pretty-printed format
func OutputPretty(stats *provider.UsageStats) {
	fmt.Println(titleStyle.Render("LLM Usage Statistics"))
	healthy, unavailable := summaryCounts(stats)
	if healthy+unavailable > 0 {
		summary := fmt.Sprintf("%d healthy", healthy)
		if unavailable > 0 {
			summary += fmt.Sprintf("  ·  %d unavailable", unavailable)
		}
		if peak := stats.MaxUtilization(); peak > 0 {
			summary += fmt.Sprintf("  ·  peak %.0f%%", peak)
		}
		fmt.Println(subtitleStyle.Render(summary + "  ·  updated " + time.Now().Format("15:04:05")))
	}
	fmt.Println()

	var failures []provider.Usage
	for _, p := range stats.Providers {
		if p.Error != nil {
			failures = append(failures, p)
			continue
		}
		if len(p.Windows) == 0 && !hasDisplayableExtra(p.Extra) {
			continue
		}

		// Get account name if available
		accountSuffix := ""
		if acc, ok := p.Extra["account"]; ok && acc != "" {
			accountSuffix = fmt.Sprintf(" (%s)", acc)
		}

		providerTitle := providerTitleStyle.Render("▸ " + ProviderName(p.Provider))
		if accountSuffix != "" {
			providerTitle += " " + accountStyle.Render(accountSuffix)
		}
		fmt.Println(providerTitle)
		fmt.Println(dividerStyle.Render(strings.Repeat("─", 34)))
		if note := cacheNote(p.Extra); note != "" {
			fmt.Println(dimStyle.Render(note))
		}

		for _, w := range p.Windows {
			printUsageWindow(w.Label, &w)
		}

		// Print extra usage if available (for Claude)
		if extra, ok := p.Extra["extra_usage"]; ok {
			printExtraUsageFromMap(extra)
		}

		// Print subscription info if available (for Kimi)
		if sub, ok := p.Extra["subscription"]; ok && hasSubscriptionContent(sub) {
			printKimiSubscription(sub)
		}

		// Print raw API response if --debug was passed
		if raw, ok := p.Extra["raw"]; ok {
			printRawUsage(raw)
		}

		fmt.Println()
	}

	seen := make(map[string]bool)
	for _, p := range failures {
		if seen[p.Provider] {
			continue
		}
		seen[p.Provider] = true
	}
	if len(seen) > 0 {
		fmt.Println(unavailableTitleStyle.Render("Unavailable"))
		for _, p := range failures {
			if !seen[p.Provider] {
				continue
			}
			delete(seen, p.Provider)
			fmt.Printf("  %-20s %s\n", unavailableProviderStyle.Render(ProviderName(p.Provider)), errorStyle.Render(conciseError(p.Error)))
		}
	}
}

func summaryCounts(stats *provider.UsageStats) (healthy, unavailable int) {
	seenFailures := make(map[string]bool)
	for _, p := range stats.Providers {
		if p.Error != nil {
			if !seenFailures[p.Provider] {
				unavailable++
				seenFailures[p.Provider] = true
			}
			continue
		}
		if len(p.Windows) > 0 || hasDisplayableExtra(p.Extra) {
			healthy++
		}
	}
	return healthy, unavailable
}

func hasDisplayableExtra(extra map[string]any) bool {
	if extra == nil {
		return false
	}
	if value, ok := extra["extra_usage"]; ok {
		return hasSubscriptionContent(value)
	}
	if value, ok := extra["subscription"]; ok {
		return hasSubscriptionContent(value)
	}
	return false
}

func hasSubscriptionContent(value any) bool {
	m, ok := value.(map[string]any)
	if !ok {
		return false
	}
	return len(m) > 0
}

// cacheNote describes a report that was served from cache rather than fetched,
// so frozen numbers never look like a bug.
func cacheNote(extra map[string]any) string {
	details, ok := extra["cache"].(map[string]any)
	if !ok {
		return ""
	}
	note := fmt.Sprintf("cached %s ago", formatAge(time.Duration(numeric(details["age_seconds"]))*time.Second))
	if retryAt, ok := rateLimitRetryAt(extra); ok {
		note += fmt.Sprintf("  ·  rate limited, retrying after %s", retryAt.Local().Format("15:04:05"))
	} else if stale, _ := details["stale"].(bool); stale {
		note += "  ·  provider unavailable, showing the last good read"
	}
	return note
}

// formatAge renders a cache age, keeping second precision that
// FormatDuration rounds away.
func formatAge(age time.Duration) string {
	if age < time.Minute {
		return fmt.Sprintf("%ds", max(int(age.Seconds()), 0))
	}
	return FormatDuration(age)
}

// rateLimitRetryAt reports when a rate-limited provider may be contacted again.
func rateLimitRetryAt(extra map[string]any) (time.Time, bool) {
	details, ok := extra["rate_limit"].(map[string]any)
	if !ok {
		return time.Time{}, false
	}
	switch value := details["retry_at"].(type) {
	case time.Time:
		return value, true
	case string:
		if parsed, err := time.Parse(time.RFC3339, value); err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}

// numeric reads a JSON-decoded number, which may arrive as either type
// depending on whether the value round-tripped through the cache file.
func numeric(value any) int64 {
	switch typed := value.(type) {
	case int:
		return int64(typed)
	case int64:
		return typed
	case float64:
		return int64(typed)
	default:
		return 0
	}
}

func conciseError(err error) string {
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "quota snapshot") || strings.Contains(message, "does not expose weekly quota"):
		return "weekly quota snapshot required"
	case strings.Contains(message, "401") || strings.Contains(message, "unauthenticated") || strings.Contains(message, "invalid_grant") || strings.Contains(message, "token") && strings.Contains(message, "expired"):
		return conciseAuthRequired
	case strings.Contains(message, "timeout") || strings.Contains(message, "deadline exceeded"):
		return "request timed out"
	case strings.Contains(message, "429") || strings.Contains(message, "rate limit"):
		return "rate limited"
	default:
		return "temporarily unavailable"
	}
}

func printRawUsage(raw any) {
	rawMsg, ok := raw.(json.RawMessage)
	if !ok {
		return
	}
	var buf bytes.Buffer
	if err := json.Indent(&buf, rawMsg, "", "  "); err != nil {
		return
	}
	fmt.Println("Raw API Response:")
	fmt.Println(buf.String())
}

func printExtraUsageFromMap(extra any) {
	extraMap, ok := extra.(map[string]any)
	if !ok {
		return
	}

	fmt.Println("Extra Usage Credits:")
	if utilization, ok := extraMap["utilization"]; ok {
		if util, ok := utilization.(float64); ok {
			bar := RenderProgressBar(util)
			fmt.Printf("  Usage:    %s  %.1f%%\n", bar, util)
		}
	}
	if used, ok := extraMap["used_credits"]; ok {
		if limit, ok := extraMap["monthly_limit"]; ok {
			if usedFloat, ok := used.(float64); ok {
				if limitFloat, ok := limit.(float64); ok {
					fmt.Printf("  Credits:  $%.2f / $%.2f\n", usedFloat, limitFloat)
				}
			}
		}
	}
}

func printUsageWindow(label string, window *provider.UsageWindow) {
	fmt.Printf("  %s:\n", windowLabelStyle.Render(label))

	bar := RenderProgressBar(window.Utilization)
	fmt.Printf("    %s %s  %s\n", metricLabelStyle.Render("Usage:"), bar, utilizationStyle(window.Utilization).Render(fmt.Sprintf("%.1f%%", window.Utilization)))

	if resetDur := window.TimeUntilReset(); resetDur != nil {
		fmt.Printf("    %s %s\n", metricLabelStyle.Render("Resets:"), valueStyle.Render("in "+FormatDuration(*resetDur)))
	} else {
		fmt.Printf("    %s %s\n", metricLabelStyle.Render("Resets:"), dimStyle.Render("N/A"))
	}
}

func utilizationStyle(value float64) lipgloss.Style {
	switch {
	case value >= 90:
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))
	case value >= 75:
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
	default:
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("70"))
	}
}

// RenderProgressBar renders a progress bar for the given percentage
func RenderProgressBar(percentage float64) string {
	filled := int(percentage / 100 * float64(barWidth))
	filled = max(0, min(filled, barWidth))

	bar := strings.Repeat(barFull, filled) + strings.Repeat(barEmpty, barWidth-filled)
	return progressStyle(percentage).Render(bar)
}

func progressStyle(value float64) lipgloss.Style {
	switch {
	case value >= 90:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	case value >= 75:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("70"))
	}
}

// FormatDuration formats a duration for human-readable output
func FormatDuration(d time.Duration) string {
	if d < 0 {
		return "expired"
	}

	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	parts := []string{}
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
	}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if minutes > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
	}

	return strings.Join(parts, " ")
}

// printKimiSubscription prints Kimi subscription info with colors
func printKimiSubscription(sub any) {
	subMap, ok := sub.(map[string]any)
	if !ok {
		return
	}

	fmt.Println(subscriptionTitleStyle.Render("Subscription:"))
	if subscribed, ok := subMap["subscribed"].(bool); ok {
		status := "Inactive"
		style := statusExpiredStyle
		if subscribed {
			status = "Active"
			style = statusActiveStyle
		}
		fmt.Printf("  %s %s\n", metricLabelStyle.Render("Status:"), style.Render(status))
	}

	// Print plan info
	if plan, ok := subMap["plan"].(map[string]any); ok {
		title := getStringValue(plan, "title")
		level := getStringValue(plan, "level")
		status := getStringValue(plan, "status")

		// Style the status based on its value
		var styledStatus string
		switch status {
		case "Active":
			styledStatus = statusActiveStyle.Render(status)
		case "Cancelled":
			styledStatus = statusCancelledStyle.Render(status)
		case "Expired":
			styledStatus = statusExpiredStyle.Render(status)
		default:
			styledStatus = status
		}

		planName := title
		if planName == "" {
			planName = "Subscription"
		}
		if level != "" {
			planName += " " + dimStyle.Render("("+level+")")
		}
		fmt.Printf("  %s %s %s\n", metricLabelStyle.Render("Plan:"), planName, styledStatus)
	}

	// Print expiry info
	if expiresAt, ok := subMap["expires_at"].(string); ok && expiresAt != "" {
		if t, err := time.Parse(time.RFC3339, expiresAt); err == nil {
			remaining := time.Until(t)
			var expiryStr string
			if remaining > 0 {
				expiryStr = fmt.Sprintf("%s %s", t.Format("2006-01-02"), dimStyle.Render("("+FormatDuration(remaining)+" remaining)"))
			} else {
				expiryStr = statusExpiredStyle.Render(t.Format("2006-01-02") + " (expired)")
			}
			fmt.Printf("  Expires:  %s\n", expiryStr)
		}
	}

	// Print features/quotas
	features := subscriptionFeatures(subMap["features"])
	if len(features) > 0 {
		fmt.Println("  " + subscriptionTitleStyle.Render("Included quotas:"))
		for _, feature := range features {
			name := getStringValue(feature, "feature")
			left := getIntValue(feature, "left")
			total := getIntValue(feature, "total")

			// Calculate percentage for progress bar
			var percentage float64
			if total > 0 {
				percentage = float64(total-left) / float64(total) * 100
			}
			bar := RenderProgressBar(percentage)

			fmt.Printf("    %-12s %s %s\n",
				featureNameStyle.Render(name),
				bar,
				dimStyle.Render(fmt.Sprintf("%d/%d left", left, total)))
		}
	}
}

func subscriptionFeatures(value any) []map[string]any {
	switch features := value.(type) {
	case []map[string]any:
		return features
	case []any:
		result := make([]map[string]any, 0, len(features))
		for _, item := range features {
			if feature, ok := item.(map[string]any); ok {
				result = append(result, feature)
			}
		}
		return result
	default:
		return nil
	}
}

// getStringValue safely extracts a string value from a map
func getStringValue(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// getIntValue safely extracts an int value from a map (handles float64 from JSON)
func getIntValue(m map[string]any, key string) int {
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	if v, ok := m[key].(int); ok {
		return v
	}
	return 0
}
