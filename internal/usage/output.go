package usage

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/denysvitali/llm-usage/internal/provider"
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
		providerLabel := providerShortName(p.Provider)
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

		fmt.Println()
	}
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

// ProviderName returns the display name for a provider
func ProviderName(id string) string {
	switch id {
	case "claude":
		return "Claude (Pro/Max Subscription)"
	case "kimi":
		return "Kimi"
	case "zai":
		return "Z.AI"
	default:
		return strings.ToUpper(id)
	}
}

func providerShortName(id string) string {
	switch id {
	case "claude":
		return "C"
	case "kimi":
		return "K"
	case "zai":
		return "Z"
	default:
		return string(strings.ToUpper(id)[0])
	}
}
