package api

import (
	"testing"
	"time"
)

func TestUsageWindow_Remaining(t *testing.T) {
	tests := []struct {
		name     string
		window   *UsageWindow
		expected float64
	}{
		{
			name:     "nil window returns 100",
			window:   nil,
			expected: 100,
		},
		{
			name:     "0% utilization returns 100",
			window:   &UsageWindow{Utilization: 0},
			expected: 100,
		},
		{
			name:     "50% utilization returns 50",
			window:   &UsageWindow{Utilization: 50},
			expected: 50,
		},
		{
			name:     "100% utilization returns 0",
			window:   &UsageWindow{Utilization: 100},
			expected: 0,
		},
		{
			name:     "75.5% utilization returns 24.5",
			window:   &UsageWindow{Utilization: 75.5},
			expected: 24.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.window.Remaining()
			if got != tt.expected {
				t.Errorf("Remaining() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestUsageWindow_TimeUntilReset(t *testing.T) {
	t.Run("nil window returns nil", func(t *testing.T) {
		var w *UsageWindow
		if got := w.TimeUntilReset(); got != nil {
			t.Errorf("TimeUntilReset() = %v, want nil", got)
		}
	})

	t.Run("nil ResetsAt returns nil", func(t *testing.T) {
		w := &UsageWindow{Utilization: 50, ResetsAt: nil}
		if got := w.TimeUntilReset(); got != nil {
			t.Errorf("TimeUntilReset() = %v, want nil", got)
		}
	})

	t.Run("future reset time returns positive duration", func(t *testing.T) {
		futureTime := time.Now().Add(2 * time.Hour)
		w := &UsageWindow{Utilization: 50, ResetsAt: &futureTime}

		got := w.TimeUntilReset()
		if got == nil {
			t.Fatal("TimeUntilReset() = nil, want non-nil")
		}

		// Allow 1 second tolerance for test execution time
		if *got < 1*time.Hour+59*time.Minute || *got > 2*time.Hour+1*time.Second {
			t.Errorf("TimeUntilReset() = %v, want approximately 2h", *got)
		}
	})

	t.Run("past reset time returns negative duration", func(t *testing.T) {
		pastTime := time.Now().Add(-1 * time.Hour)
		w := &UsageWindow{Utilization: 50, ResetsAt: &pastTime}

		got := w.TimeUntilReset()
		if got == nil {
			t.Fatal("TimeUntilReset() = nil, want non-nil")
		}

		if *got >= 0 {
			t.Errorf("TimeUntilReset() = %v, want negative duration", *got)
		}
	})
}
