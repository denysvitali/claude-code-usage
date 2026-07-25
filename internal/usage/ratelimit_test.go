package usage

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/denysvitali/llm-usage/internal/cache"
	"github.com/denysvitali/llm-usage/provider"
)

const (
	stubProviderID = "claude"
	stubAccount    = "default"
)

// stubProvider counts how many times the network path was entered and returns a
// scripted result each time.
type stubProvider struct {
	calls  int
	usage  *provider.Usage
	err    error
	errFor int // return err for the first errFor calls, then usage
}

func (s *stubProvider) Name() string      { return "Stub" }
func (s *stubProvider) ShortName() string { return "S" }
func (s *stubProvider) ID() string        { return stubProviderID }

func (s *stubProvider) GetUsage(context.Context) (*provider.Usage, error) {
	s.calls++
	if s.calls <= s.errFor {
		return nil, s.err
	}
	return s.usage, nil
}

func fetch(t *testing.T, stub *stubProvider, manager *cache.Manager, ttl time.Duration) provider.Usage {
	t.Helper()
	stats := FetchAllUsage(context.Background(),
		[]ProviderInstance{{Provider: stub, AccountName: stubAccount}},
		CacheOptions{Manager: manager, TTL: ttl},
	)
	if len(stats.Providers) != 1 {
		t.Fatalf("got %d reports, want 1", len(stats.Providers))
	}
	return stats.Providers[0]
}

func goodUsage() *provider.Usage {
	return &provider.Usage{
		Provider: stubProviderID,
		Windows:  []provider.UsageWindow{{Label: "5-Hour", Utilization: 12}},
	}
}

// A rate limit must park the provider until its cooldown expires: the whole
// point is that the next invocation does not repeat the request.
func TestRateLimitCooldownSuppressesFollowUpRequests(t *testing.T) {
	manager := cache.NewManagerAt(t.TempDir())
	stub := &stubProvider{
		usage: goodUsage(),
		err:   &provider.RateLimitError{Provider: stubProviderID, RetryAt: time.Now().Add(10 * time.Minute)},
	}

	// First call: cached good data, so a later rate limit has something to fall back on.
	if report := fetch(t, stub, manager, time.Millisecond); report.Error != nil {
		t.Fatalf("first fetch failed: %v", report.Error)
	}
	time.Sleep(2 * time.Millisecond) // let the cached entry go stale

	// Second call: the provider rate-limits us and we fall back to stale data.
	stub.errFor = stub.calls + 1
	report := fetch(t, stub, manager, time.Millisecond)
	if report.Error != nil {
		t.Fatalf("rate-limited fetch should serve stale data, got error: %v", report.Error)
	}
	if _, ok := report.Extra["rate_limit"]; !ok {
		t.Fatalf("report is missing the rate_limit marker: %+v", report.Extra)
	}

	// Third call: the cooldown is still active, so the provider is not contacted.
	callsBefore := stub.calls
	report = fetch(t, stub, manager, time.Millisecond)
	if stub.calls != callsBefore {
		t.Fatalf("provider was contacted during the cooldown (%d -> %d calls)", callsBefore, stub.calls)
	}
	if report.Error != nil {
		t.Fatalf("cooldown fetch should serve stale data, got error: %v", report.Error)
	}
}

// With no cached data to fall back on, a cooldown still has to suppress the
// request and report why.
func TestRateLimitCooldownWithoutCachedData(t *testing.T) {
	manager := cache.NewManagerAt(t.TempDir())
	stub := &stubProvider{
		usage:  goodUsage(),
		errFor: 1,
		err:    &provider.RateLimitError{Provider: stubProviderID, RetryAt: time.Now().Add(10 * time.Minute)},
	}

	if report := fetch(t, stub, manager, time.Minute); report.Error == nil {
		t.Fatal("expected the first fetch to surface the rate limit")
	}

	callsBefore := stub.calls
	report := fetch(t, stub, manager, time.Minute)
	if stub.calls != callsBefore {
		t.Fatalf("provider was contacted during the cooldown (%d -> %d calls)", callsBefore, stub.calls)
	}
	var rateLimit *provider.RateLimitError
	if !errors.As(report.Error, &rateLimit) {
		t.Fatalf("error = %v, want a rate-limit error", report.Error)
	}
}

// Cooldowns apply even when response caching is switched off, otherwise
// --cache-ttl=0 would keep hammering an endpoint that already said no.
func TestRateLimitCooldownAppliesWithCachingDisabled(t *testing.T) {
	manager := cache.NewManagerAt(t.TempDir())
	stub := &stubProvider{
		usage:  goodUsage(),
		errFor: 1,
		err:    &provider.RateLimitError{Provider: stubProviderID, RetryAt: time.Now().Add(10 * time.Minute)},
	}

	fetch(t, stub, manager, 0)
	callsBefore := stub.calls
	fetch(t, stub, manager, 0)
	if stub.calls != callsBefore {
		t.Fatalf("provider was contacted during the cooldown (%d -> %d calls)", callsBefore, stub.calls)
	}
}

// An expired cooldown must let traffic through again.
func TestExpiredCooldownAllowsRefresh(t *testing.T) {
	manager := cache.NewManagerAt(t.TempDir())
	stub := &stubProvider{usage: goodUsage()}
	key := cache.HashKey(stubProviderID, stubAccount)

	if err := manager.MarkCooldown(key, time.Now().Add(-time.Second)); err != nil {
		t.Fatalf("MarkCooldown failed: %v", err)
	}

	report := fetch(t, stub, manager, time.Minute)
	if stub.calls != 1 {
		t.Fatalf("provider was contacted %d times, want 1", stub.calls)
	}
	if report.Error != nil {
		t.Fatalf("fetch failed: %v", report.Error)
	}

	// The successful read clears the cooldown for good.
	if retryAt := manager.Cooldown(key); !retryAt.IsZero() {
		t.Fatalf("cooldown survived a successful read: %s", retryAt)
	}
}

// A plain failure must not be mistaken for a rate limit, and must not silently
// serve stale data unless the caller asked for it.
func TestOrdinaryFailureIsNotCachedAsCooldown(t *testing.T) {
	manager := cache.NewManagerAt(t.TempDir())
	stub := &stubProvider{usage: goodUsage(), errFor: 1, err: errors.New("boom")}

	report := fetch(t, stub, manager, time.Minute)
	if report.Error == nil {
		t.Fatal("expected the failure to be surfaced")
	}
	if retryAt := manager.Cooldown(cache.HashKey(stubProviderID, stubAccount)); !retryAt.IsZero() {
		t.Fatalf("an ordinary failure recorded a cooldown: %s", retryAt)
	}

	if report := fetch(t, stub, manager, time.Minute); report.Error != nil {
		t.Fatalf("the retry after a transient failure should succeed: %v", report.Error)
	}
	if stub.calls != 2 {
		t.Fatalf("provider was contacted %d times, want 2", stub.calls)
	}
}

// The fresh cache must absorb repeat invocations - this is what protects the
// endpoint when a bar module or dashboard polls in a loop.
func TestFreshCacheAvoidsUpstreamRequests(t *testing.T) {
	manager := cache.NewManagerAt(t.TempDir())
	stub := &stubProvider{usage: goodUsage()}

	for range 5 {
		if report := fetch(t, stub, manager, time.Minute); report.Error != nil {
			t.Fatalf("fetch failed: %v", report.Error)
		}
	}
	if stub.calls != 1 {
		t.Fatalf("provider was contacted %d times, want 1", stub.calls)
	}
}
