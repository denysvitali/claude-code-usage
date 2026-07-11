// Package provider defines stable, reusable LLM usage types.
package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Provider fetches normalized usage for one authenticated account.
type Provider interface {
	GetUsage(context.Context) (*Usage, error)
	Name() string
	ShortName() string
	ID() string
}

// Usage is a provider-neutral usage report.
type Usage struct {
	Provider string         `json:"provider"`
	Windows  []UsageWindow  `json:"windows"`
	Extra    map[string]any `json:"extra,omitempty"`
	Error    error          `json:"-"`
}

func (u Usage) MarshalJSON() ([]byte, error) {
	type encoded struct {
		Provider string         `json:"provider"`
		Windows  []UsageWindow  `json:"windows"`
		Extra    map[string]any `json:"extra,omitempty"`
		Error    *ErrorInfo     `json:"error,omitempty"`
	}
	var errorInfo *ErrorInfo
	if u.Error != nil {
		errorInfo = &ErrorInfo{Message: u.Error.Error()}
	}
	return json.Marshal(encoded{Provider: u.Provider, Windows: u.Windows, Extra: u.Extra, Error: errorInfo})
}

// ErrorInfo is the stable JSON representation of an upstream failure.
type ErrorInfo struct {
	Message string `json:"message"`
}

// UsageWindow is one provider-defined usage period.
type UsageWindow struct {
	Label       string     `json:"label"`
	Utilization float64    `json:"utilization"`
	ResetsAt    *time.Time `json:"resets_at"`
	Limit       *float64   `json:"limit,omitempty"`
	Used        *float64   `json:"used,omitempty"`
	Remaining   *float64   `json:"remaining,omitempty"`
}

func (w UsageWindow) TimeUntilReset() *time.Duration {
	if w.ResetsAt == nil {
		return nil
	}
	duration := time.Until(*w.ResetsAt)
	return &duration
}

// UsageStats aggregates multiple provider reports.
type UsageStats struct {
	Providers []Usage `json:"providers"`
}

func (s *UsageStats) MaxUtilization() float64 {
	var maximum float64
	for _, report := range s.Providers {
		if report.Error == nil {
			for _, window := range report.Windows {
				if window.Utilization > maximum {
					maximum = window.Utilization
				}
			}
		}
	}
	return maximum
}

func (s *UsageStats) GetClass() string {
	if maximum := s.MaxUtilization(); maximum >= 90 {
		return "critical"
	} else if maximum >= 75 {
		return "warning"
	}
	return "normal"
}
func (s *UsageStats) ProviderByID(id string) *Usage {
	for i := range s.Providers {
		if s.Providers[i].Provider == id {
			return &s.Providers[i]
		}
	}
	return nil
}

// NewUsageError builds a normalized failed usage report.
func NewUsageError(id, name string, err error) *Usage {
	return &Usage{Provider: id, Error: fmt.Errorf("%s: %w", name, err)}
}

func NewUsageNotConfigured(id, name string) *Usage {
	return &Usage{Provider: id, Error: fmt.Errorf("%s: not configured", name)}
}
