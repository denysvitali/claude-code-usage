package codex

import "time"

// UsageResponse is the raw body returned by the Codex usage endpoint.
type UsageResponse struct {
	RateLimit RateLimit `json:"rate_limit"`
}

// RateLimit groups the primary and secondary usage windows.
type RateLimit struct {
	PrimaryWindow   *Window `json:"primary_window"`
	SecondaryWindow *Window `json:"secondary_window"`
}

// Window is one Codex usage period.
type Window struct {
	UsedPercent float64 `json:"used_percent"`
	ResetAt     int64   `json:"reset_at"`
}

// ResetTime converts the Unix reset timestamp to a time value.
func (w Window) ResetTime() *time.Time {
	if w.ResetAt == 0 {
		return nil
	}
	t := time.Unix(w.ResetAt, 0)
	return &t
}
