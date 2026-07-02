package usage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/denysvitali/llm-usage/internal/provider"
)

// Lipgloss styles for subscription display
var (
	subscriptionTitleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true)
	statusActiveStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("70"))
	statusCancelledStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	statusExpiredStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	dimStyle               = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	featureNameStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("75"))
)

const (
	barWidth = 20
	barFull  = "█"
	barEmpty = "░"
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
	for _, p := range stats.Providers {
		if p.Error != nil {
			continue
		}
		providerLabel := ProviderShortName(p.Provider)
		if len(p.Windows) > 0 {
			// Use the first window's utilization for the compact display
			textParts = append(textParts, fmt.Sprintf("%s:%.0f%%", providerLabel, p.Windows[0].Utilization))
		}
	}
	text := strings.Join(textParts, " ")

	// Build detailed tooltip
	var tooltipLines []string
	tooltipLines = append(tooltipLines, "LLM Usage")
	tooltipLines = append(tooltipLines, "")

	for _, p := range stats.Providers {
		if p.Error != nil {
			tooltipLines = append(tooltipLines, fmt.Sprintf("%s: Error", ProviderName(p.Provider)))
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

// OutputJSON outputs usage stats in JSON format
func OutputJSON(stats *provider.UsageStats) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(stats); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		os.Exit(1)
	}
}

// OutputPretty outputs usage stats in a pretty-printed format
func OutputPretty(stats *provider.UsageStats) {
	fmt.Println("LLM Usage Statistics")
	fmt.Println("====================")
	fmt.Println()

	for _, p := range stats.Providers {
		if p.Error != nil {
			fmt.Printf("%s:\n", ProviderName(p.Provider))
			fmt.Printf("  Error: %s\n", p.Error)
			fmt.Println()
			continue
		}

		// Get account name if available
		accountSuffix := ""
		if acc, ok := p.Extra["account"]; ok && acc != "" {
			accountSuffix = fmt.Sprintf(" (%s)", acc)
		}

		fmt.Printf("%s%s:\n", ProviderName(p.Provider), accountSuffix)
		fmt.Println(strings.Repeat("-", len(ProviderName(p.Provider))+len(accountSuffix)+1))

		for _, w := range p.Windows {
			printUsageWindow(w.Label, &w)
		}

		// Print extra usage if available (for Claude)
		if extra, ok := p.Extra["extra_usage"]; ok {
			printExtraUsageFromMap(extra)
		}

		// Print subscription info if available (for Kimi)
		if sub, ok := p.Extra["subscription"]; ok {
			printKimiSubscription(sub)
		}

		// Print raw API response if --debug was passed
		if raw, ok := p.Extra["raw"]; ok {
			printRawUsage(raw)
		}

		fmt.Println()
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
	fmt.Printf("  %s:\n", label)

	bar := RenderProgressBar(window.Utilization)
	fmt.Printf("    Usage:    %s  %.1f%%\n", bar, window.Utilization)

	if resetDur := window.TimeUntilReset(); resetDur != nil {
		fmt.Printf("    Resets:   in %s\n", FormatDuration(*resetDur))
	} else {
		fmt.Printf("    Resets:   N/A\n")
	}
}

// RenderProgressBar renders a progress bar for the given percentage
func RenderProgressBar(percentage float64) string {
	filled := int(percentage / 100 * float64(barWidth))
	filled = max(0, min(filled, barWidth))

	return strings.Repeat(barFull, filled) + strings.Repeat(barEmpty, barWidth-filled)
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

		fmt.Printf("  Plan:     %s %s %s\n", title, dimStyle.Render("("+level+")"), styledStatus)
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
	if features, ok := subMap["features"].([]any); ok && len(features) > 0 {
		fmt.Println("  Features:")
		for _, f := range features {
			if feature, ok := f.(map[string]any); ok {
				name := getStringValue(feature, "feature")
				left := getIntValue(feature, "left")
				total := getIntValue(feature, "total")

				// Calculate percentage for progress bar
				var percentage float64
				if total > 0 {
					percentage = float64(total-left) / float64(total) * 100
				}
				bar := RenderProgressBar(percentage)

				fmt.Printf("    %s: %s %s\n",
					featureNameStyle.Render(name),
					bar,
					dimStyle.Render(fmt.Sprintf("%d/%d left", left, total)))
			}
		}
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
