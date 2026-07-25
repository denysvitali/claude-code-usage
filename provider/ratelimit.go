package provider

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	// DefaultRateLimitBackoff is how long to stay away from a provider that
	// refused a request without saying when it will accept the next one.
	DefaultRateLimitBackoff = time.Minute

	// MaxRateLimitBackoff caps the cooldown a provider can ask for, so one
	// malformed header cannot silence a provider for the rest of the day.
	MaxRateLimitBackoff = time.Hour
)

// resetHeaders are the non-standard headers providers use to announce when the
// current rate-limit window refills, checked when Retry-After is absent.
var resetHeaders = []string{
	"anthropic-ratelimit-unified-reset",
	"anthropic-ratelimit-requests-reset",
	"x-ratelimit-reset",
	"ratelimit-reset",
}

// RateLimitError reports that a provider rejected a request because it was
// asked too often. RetryAt is the earliest moment another request can succeed;
// callers must not contact the provider again before then.
type RateLimitError struct {
	Provider string
	RetryAt  time.Time
	Detail   string
}

// Error implements error.
func (e *RateLimitError) Error() string {
	wait := max(time.Until(e.RetryAt).Round(time.Second), 0)
	message := fmt.Sprintf("rate limited (HTTP %d), retry in %s", http.StatusTooManyRequests, wait)
	if e.Detail != "" {
		message += ": " + e.Detail
	}
	return message
}

// RetryAt reports the cooldown deadline carried by a rate-limit error, and
// whether err was a rate-limit error at all.
func RetryAt(err error) (time.Time, bool) {
	var rateLimit *RateLimitError
	if errors.As(err, &rateLimit) {
		return rateLimit.RetryAt, true
	}
	return time.Time{}, false
}

// NewRateLimitError builds a RateLimitError whose cooldown comes from the
// response headers when the provider supplies one.
func NewRateLimitError(providerID string, header http.Header, detail string) *RateLimitError {
	return &RateLimitError{
		Provider: providerID,
		RetryAt:  time.Now().Add(retryAfter(header)),
		Detail:   summarize(detail),
	}
}

// retryAfter derives the cooldown from Retry-After, falling back to the
// provider-specific reset headers and finally to a fixed backoff.
func retryAfter(header http.Header) time.Duration {
	if value := strings.TrimSpace(header.Get("Retry-After")); value != "" {
		if seconds, err := strconv.Atoi(value); err == nil {
			return clampBackoff(time.Duration(seconds) * time.Second)
		}
		if deadline, err := http.ParseTime(value); err == nil {
			return clampBackoff(time.Until(deadline))
		}
	}

	for _, name := range resetHeaders {
		value := strings.TrimSpace(header.Get(name))
		if value == "" {
			continue
		}
		if number, err := strconv.ParseInt(value, 10, 64); err == nil {
			// Providers send either a Unix timestamp or a relative number of
			// seconds; anything in the past decade is clearly the former.
			if number > 1_000_000_000 {
				return clampBackoff(time.Until(time.Unix(number, 0)))
			}
			return clampBackoff(time.Duration(number) * time.Second)
		}
		if deadline, err := time.Parse(time.RFC3339, value); err == nil {
			return clampBackoff(time.Until(deadline))
		}
	}

	return DefaultRateLimitBackoff
}

// clampBackoff keeps a provider-supplied cooldown within useful bounds. A
// non-positive value means the header is stale or our clock disagrees, so the
// conservative default applies rather than an immediate retry.
func clampBackoff(wait time.Duration) time.Duration {
	switch {
	case wait <= 0:
		return DefaultRateLimitBackoff
	case wait > MaxRateLimitBackoff:
		return MaxRateLimitBackoff
	default:
		return wait
	}
}

// summarize trims an upstream error body down to something loggable.
func summarize(detail string) string {
	detail = strings.TrimSpace(detail)
	const limit = 200
	if len(detail) > limit {
		return detail[:limit] + "..."
	}
	return detail
}
