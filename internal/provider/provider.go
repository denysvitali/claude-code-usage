// Package provider provides the abstraction layer for LLM usage providers.
package provider

import (
	"fmt"
	"time"
)

// Provider defines the interface for LLM usage providers
type Provider interface {
	// Name returns the provider's display name
	Name() string

	// ID returns the provider's unique identifier
	ID() string

	// GetUsage fetches current usage statistics
	GetUsage() (*Usage, error)
}

// Usage represents generic usage statistics from a provider
type Usage struct {
	// Provider name
	Provider string

	// Usage windows (provider-specific, can be nil)
	Windows []UsageWindow

	// Extra usage information (optional, provider-specific)
	Extra map[string]any

	// Error if fetching failed (allows partial results)
	Error error
}

// UsageWindow represents a usage time window
type UsageWindow struct {
	Label       string     // e.g., "5-Hour", "7-Day", "Daily"
	Utilization float64    // 0-100 percentage
	ResetsAt    *time.Time // When this window resets (can be nil)

	// Additional provider-specific fields
	Limit     *float64 // Usage limit (e.g., token count)
	Used      *float64 // Amount used
	Remaining *float64 // Amount remaining
}

// TimeUntilReset returns the duration until the window resets
func (w *UsageWindow) TimeUntilReset() *time.Duration {
	if w == nil || w.ResetsAt == nil {
		return nil
	}
	d := time.Until(*w.ResetsAt)
	return &d
}

// UsageStats aggregates results from multiple providers
type UsageStats struct {
	Providers []Usage
}

// MaxUtilization returns the maximum utilization across all providers
func (s *UsageStats) MaxUtilization() float64 {
	var maxUtil float64
	for _, p := range s.Providers {
		if p.Error != nil {
			continue
		}
		for _, w := range p.Windows {
			if w.Utilization > maxUtil {
				maxUtil = w.Utilization
			}
		}
	}
	return maxUtil
}

// GetClass returns the CSS class based on maximum utilization
func (s *UsageStats) GetClass() string {
	maxUtil := s.MaxUtilization()
	if maxUtil >= 90 {
		return "critical"
	} else if maxUtil >= 75 {
		return "warning"
	}
	return "normal"
}

// ProviderByID returns a provider by its ID from the stats
func (s *UsageStats) ProviderByID(id string) *Usage {
	for i := range s.Providers {
		if s.Providers[i].Provider == id {
			return &s.Providers[i]
		}
	}
	return nil
}

// NewUsageError creates a Usage object with an error
func NewUsageError(providerID, providerName string, err error) *Usage {
	return &Usage{
		Provider: providerID,
		Error:    fmt.Errorf("%s: %w", providerName, err),
	}
}

// NewUsageNotConfigured creates a Usage object for a not-configured provider
func NewUsageNotConfigured(providerID, providerName string) *Usage {
	return &Usage{
		Provider: providerID,
		Error:    fmt.Errorf("%s: not configured", providerName),
	}
}
