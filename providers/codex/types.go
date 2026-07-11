package codex

import "time"

type UsageResponse struct {
	RateLimit RateLimit `json:"rate_limit"`
}
type RateLimit struct {
	PrimaryWindow   *Window `json:"primary_window"`
	SecondaryWindow *Window `json:"secondary_window"`
}
type Window struct {
	UsedPercent float64 `json:"used_percent"`
	ResetAt     int64   `json:"reset_at"`
}

func (w Window) ResetTime() *time.Time {
	if w.ResetAt == 0 {
		return nil
	}
	t := time.Unix(w.ResetAt, 0)
	return &t
}
