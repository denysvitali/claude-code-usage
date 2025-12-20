package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/denysvitali/claude-code-usage/internal/api"
	"github.com/denysvitali/claude-code-usage/internal/credentials"
	"github.com/denysvitali/claude-code-usage/internal/version"
)

const (
	barWidth = 20
	barFull  = "█"
	barEmpty = "░"
)

func main() {
	jsonOutput := flag.Bool("json", false, "Output in JSON format")
	waybarOutput := flag.Bool("waybar", false, "Output in waybar JSON format")
	showVersion := flag.Bool("version", false, "Show version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("claude-usage %s\n", version.Version)
		return
	}

	creds, err := credentials.Load()
	if err != nil {
		if *waybarOutput {
			outputWaybarError(err.Error())
			return
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if creds.ClaudeAiOauth.IsExpired() {
		msg := "Token expired - run 'claude' to refresh"
		if *waybarOutput {
			outputWaybarError(msg)
			return
		}
		fmt.Fprintf(os.Stderr, "Warning: %s\n", msg)
		os.Exit(1)
	}

	client := api.NewClient(creds.ClaudeAiOauth.AccessToken)
	usage, err := client.GetUsage()
	if err != nil {
		if *waybarOutput {
			outputWaybarError(err.Error())
			return
		}
		fmt.Fprintf(os.Stderr, "Error fetching usage: %v\n", err)
		os.Exit(1)
	}

	switch {
	case *waybarOutput:
		outputWaybar(usage)
	case *jsonOutput:
		outputJSON(usage)
	default:
		outputPretty(usage, creds.ClaudeAiOauth.ExpiresIn())
	}
}

func outputJSON(usage *api.UsageResponse) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(usage); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		os.Exit(1)
	}
}

// WaybarOutput represents the JSON format expected by waybar custom modules
type WaybarOutput struct {
	Text       string `json:"text"`
	Tooltip    string `json:"tooltip"`
	Class      string `json:"class"`
	Percentage int    `json:"percentage"`
}

func outputWaybar(usage *api.UsageResponse) {
	// Use the highest utilization for the percentage and class
	var maxUtil float64
	if usage.FiveHour != nil && usage.FiveHour.Utilization > maxUtil {
		maxUtil = usage.FiveHour.Utilization
	}
	if usage.SevenDay != nil && usage.SevenDay.Utilization > maxUtil {
		maxUtil = usage.SevenDay.Utilization
	}

	// Determine class based on utilization
	class := "normal"
	if maxUtil >= 90 {
		class = "critical"
	} else if maxUtil >= 75 {
		class = "warning"
	}

	// Build compact text for the bar
	var textParts []string
	if usage.FiveHour != nil {
		textParts = append(textParts, fmt.Sprintf("5h:%.0f%%", usage.FiveHour.Utilization))
	}
	if usage.SevenDay != nil {
		textParts = append(textParts, fmt.Sprintf("7d:%.0f%%", usage.SevenDay.Utilization))
	}
	text := strings.Join(textParts, " ")

	// Build detailed tooltip
	var tooltipLines []string
	tooltipLines = append(tooltipLines, "Claude Usage")
	tooltipLines = append(tooltipLines, "")

	if usage.FiveHour != nil {
		line := fmt.Sprintf("5-Hour: %.1f%%", usage.FiveHour.Utilization)
		if d := usage.FiveHour.TimeUntilReset(); d != nil {
			line += fmt.Sprintf(" (resets in %s)", formatDuration(*d))
		}
		tooltipLines = append(tooltipLines, line)
	}

	if usage.SevenDay != nil {
		line := fmt.Sprintf("7-Day: %.1f%%", usage.SevenDay.Utilization)
		if d := usage.SevenDay.TimeUntilReset(); d != nil {
			line += fmt.Sprintf(" (resets in %s)", formatDuration(*d))
		}
		tooltipLines = append(tooltipLines, line)
	}

	if usage.SevenDaySonnet != nil {
		line := fmt.Sprintf("7-Day Sonnet: %.1f%%", usage.SevenDaySonnet.Utilization)
		if d := usage.SevenDaySonnet.TimeUntilReset(); d != nil {
			line += fmt.Sprintf(" (resets in %s)", formatDuration(*d))
		}
		tooltipLines = append(tooltipLines, line)
	}

	if usage.SevenDayOpus != nil {
		line := fmt.Sprintf("7-Day Opus: %.1f%%", usage.SevenDayOpus.Utilization)
		if d := usage.SevenDayOpus.TimeUntilReset(); d != nil {
			line += fmt.Sprintf(" (resets in %s)", formatDuration(*d))
		}
		tooltipLines = append(tooltipLines, line)
	}

	output := WaybarOutput{
		Text:       text,
		Tooltip:    strings.Join(tooltipLines, "\n"),
		Class:      class,
		Percentage: int(maxUtil),
	}

	enc := json.NewEncoder(os.Stdout)
	if err := enc.Encode(output); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding waybar output: %v\n", err)
		os.Exit(1)
	}
}

func outputWaybarError(msg string) {
	output := WaybarOutput{
		Text:       "Claude: Error",
		Tooltip:    msg,
		Class:      "error",
		Percentage: 0,
	}
	enc := json.NewEncoder(os.Stdout)
	if err := enc.Encode(output); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding waybar error output: %v\n", err)
		os.Exit(1)
	}
}

func outputPretty(usage *api.UsageResponse, tokenExpiresIn time.Duration) {
	fmt.Println("Claude Usage (Pro/Max Subscription)")
	fmt.Println("====================================")
	fmt.Println()

	printUsageWindow("5-Hour Window", usage.FiveHour)
	fmt.Println()
	printUsageWindow("7-Day Window", usage.SevenDay)

	if usage.SevenDaySonnet != nil {
		fmt.Println()
		printUsageWindow("7-Day Sonnet", usage.SevenDaySonnet)
	}

	if usage.SevenDayOpus != nil {
		fmt.Println()
		printUsageWindow("7-Day Opus", usage.SevenDayOpus)
	}

	if usage.SevenDayOAuthApp != nil {
		fmt.Println()
		printUsageWindow("7-Day OAuth Apps", usage.SevenDayOAuthApp)
	}

	if usage.IguanaNecktie != nil {
		fmt.Println()
		printUsageWindow("Iguana Necktie", usage.IguanaNecktie)
	}

	if usage.ExtraUsage != nil && usage.ExtraUsage.IsEnabled {
		fmt.Println()
		printExtraUsage(usage.ExtraUsage)
	}

	fmt.Println()
	fmt.Printf("Token expires: %s\n", formatDuration(tokenExpiresIn))
}

func printExtraUsage(extra *api.ExtraUsage) {
	fmt.Println("Extra Usage Credits:")
	if extra.Utilization != nil {
		bar := renderProgressBar(*extra.Utilization)
		fmt.Printf("  Usage:    %s  %.1f%%\n", bar, *extra.Utilization)
	}
	if extra.UsedCredits != nil && extra.MonthlyLimit != nil {
		fmt.Printf("  Credits:  $%.2f / $%.2f\n", *extra.UsedCredits, *extra.MonthlyLimit)
	}
}

func printUsageWindow(name string, window *api.UsageWindow) {
	fmt.Printf("%s:\n", name)

	if window == nil {
		fmt.Printf("  Usage:    %s  N/A\n", strings.Repeat(barEmpty, barWidth))
		fmt.Printf("  Resets:   N/A\n")
		return
	}

	bar := renderProgressBar(window.Utilization)
	fmt.Printf("  Usage:    %s  %.1f%%\n", bar, window.Utilization)

	if resetDur := window.TimeUntilReset(); resetDur != nil {
		fmt.Printf("  Resets:   in %s\n", formatDuration(*resetDur))
	} else {
		fmt.Printf("  Resets:   N/A\n")
	}
}

func renderProgressBar(percentage float64) string {
	filled := int(percentage / 100 * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}
	if filled < 0 {
		filled = 0
	}

	return strings.Repeat(barFull, filled) + strings.Repeat(barEmpty, barWidth-filled)
}

func formatDuration(d time.Duration) string {
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
