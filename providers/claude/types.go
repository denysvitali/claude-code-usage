package claude

import "time"

// UsageResponse represents the response from the OAuth usage endpoint
type UsageResponse struct {
	FiveHour         *UsageWindow `json:"five_hour"`
	SevenDay         *UsageWindow `json:"seven_day"`
	SevenDayOAuthApp *UsageWindow `json:"seven_day_oauth_apps"`
	SevenDayOpus     *UsageWindow `json:"seven_day_opus"`
	SevenDaySonnet   *UsageWindow `json:"seven_day_sonnet"`
	IguanaNecktie    *UsageWindow `json:"iguana_necktie"`
	ExtraUsage       *ExtraUsage  `json:"extra_usage"`
	Limits           []UsageLimit `json:"limits"`
}

// UsageLimit represents a single entry from the "limits" array, which can be
// scoped to a specific model (e.g. a per-model weekly limit).
type UsageLimit struct {
	Group    string           `json:"group"`
	Kind     string           `json:"kind"`
	IsActive bool             `json:"is_active"`
	Percent  float64          `json:"percent"`
	ResetsAt *time.Time       `json:"resets_at"`
	Severity string           `json:"severity"`
	Scope    *UsageLimitScope `json:"scope"`
}

// UsageLimitScope describes what a UsageLimit applies to.
type UsageLimitScope struct {
	Model *UsageLimitModel `json:"model"`
}

// UsageLimitModel identifies the model a scoped UsageLimit applies to.
type UsageLimitModel struct {
	DisplayName string `json:"display_name"`
}

// ExtraUsage represents additional usage credits beyond the subscription
type ExtraUsage struct {
	IsEnabled    bool     `json:"is_enabled"`
	MonthlyLimit *float64 `json:"monthly_limit"`
	UsedCredits  *float64 `json:"used_credits"`
	Utilization  *float64 `json:"utilization"`
}

// UsageWindow represents a usage window with utilization and reset time
type UsageWindow struct {
	// Utilization is a percentage (0-100) of usage within this window
	Utilization float64 `json:"utilization"`
	// ResetsAt is the time when this usage window resets (can be null)
	ResetsAt *time.Time `json:"resets_at"`
}

// Remaining returns the remaining percentage (100 - utilization)
func (w *UsageWindow) Remaining() float64 {
	if w == nil {
		return 100
	}
	return 100 - w.Utilization
}

// TimeUntilReset returns the duration until the window resets
func (w *UsageWindow) TimeUntilReset() *time.Duration {
	if w == nil || w.ResetsAt == nil {
		return nil
	}
	d := time.Until(*w.ResetsAt)
	return &d
}
