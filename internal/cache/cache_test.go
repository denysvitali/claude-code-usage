package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestManager_SetAndGet(t *testing.T) {
	// Create a temp directory for cache
	tmpDir := t.TempDir()

	m := &Manager{cacheDir: tmpDir}

	type testData struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	// Test Set
	data := testData{Name: "test", Value: 42}
	err := m.Set("test_key", data, time.Hour)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(filepath.Join(tmpDir, "test_key.json")); os.IsNotExist(err) {
		t.Error("Cache file was not created")
	}

	// Test Get
	var retrieved testData
	found, err := m.Get("test_key", &retrieved)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !found {
		t.Error("Expected cache hit, got miss")
	}
	if retrieved.Name != "test" || retrieved.Value != 42 {
		t.Errorf("Retrieved data doesn't match: got %+v", retrieved)
	}
}

func TestManager_CacheMiss(t *testing.T) {
	tmpDir := t.TempDir()
	m := &Manager{cacheDir: tmpDir}

	var data struct{}
	found, err := m.Get("nonexistent", &data)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if found {
		t.Error("Expected cache miss, got hit")
	}
}

func TestManager_Expiry(t *testing.T) {
	tmpDir := t.TempDir()
	m := &Manager{cacheDir: tmpDir}

	// Set with very short TTL
	data := map[string]string{"key": "value"}
	err := m.Set("expiring", data, time.Millisecond)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Wait for expiry
	time.Sleep(10 * time.Millisecond)

	// Should be expired now
	var retrieved map[string]string
	found, err := m.Get("expiring", &retrieved)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if found {
		t.Error("Expected cache miss due to expiry, got hit")
	}
}

func TestManager_LookupReturnsExpiredEntries(t *testing.T) {
	m := &Manager{cacheDir: t.TempDir()}
	if err := m.Set("stale", map[string]string{"value": "old"}, time.Millisecond); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)
	var got map[string]string
	found, fresh, age, err := m.Lookup("stale", &got)
	if err != nil || !found || fresh || age <= 0 {
		t.Fatalf("Lookup() = found:%v fresh:%v age:%v err:%v", found, fresh, age, err)
	}
	if got["value"] != "old" {
		t.Fatalf("unexpected cached value: %#v", got)
	}
}

const (
	payloadKey   = "value"
	payloadValue = "old"
)

func TestManager_CooldownSurvivesRestartAndPreservesData(t *testing.T) {
	dir := t.TempDir()
	m := NewManagerAt(dir)
	if err := m.Set("limited", map[string]string{payloadKey: payloadValue}, time.Millisecond); err != nil {
		t.Fatal(err)
	}

	retryAt := time.Now().Add(5 * time.Minute).Round(time.Second)
	if err := m.MarkCooldown("limited", retryAt); err != nil {
		t.Fatalf("MarkCooldown failed: %v", err)
	}

	// A separate manager reads the same files, as a later CLI run would.
	reopened := NewManagerAt(dir)
	if got := reopened.Cooldown("limited"); !got.Equal(retryAt) {
		t.Fatalf("Cooldown() = %s, want %s", got, retryAt)
	}

	var payload map[string]string
	found, _, _, err := reopened.Lookup("limited", &payload)
	if err != nil || !found || payload[payloadKey] != payloadValue {
		t.Fatalf("cooldown discarded the cached payload: found:%v payload:%#v err:%v", found, payload, err)
	}

	// A successful store means the provider is answering again.
	if err := reopened.Set("limited", map[string]string{payloadKey: "new"}, time.Minute); err != nil {
		t.Fatal(err)
	}
	if got := reopened.Cooldown("limited"); !got.IsZero() {
		t.Fatalf("Cooldown() = %s after a successful store, want zero", got)
	}
}

func TestManager_CooldownWithoutCachedPayload(t *testing.T) {
	m := NewManagerAt(t.TempDir())
	retryAt := time.Now().Add(time.Minute).Round(time.Second)
	if err := m.MarkCooldown("fresh", retryAt); err != nil {
		t.Fatalf("MarkCooldown failed: %v", err)
	}
	if got := m.Cooldown("fresh"); !got.Equal(retryAt) {
		t.Fatalf("Cooldown() = %s, want %s", got, retryAt)
	}

	// The marker carries no payload, so it must not look like a cache hit.
	var payload map[string]string
	found, _, _, err := m.Lookup("fresh", &payload)
	if err != nil {
		t.Fatalf("Lookup failed: %v", err)
	}
	if found {
		t.Fatal("a cooldown marker was reported as cached data")
	}
}

func TestManager_CooldownMissing(t *testing.T) {
	m := NewManagerAt(t.TempDir())
	if got := m.Cooldown("nothing"); !got.IsZero() {
		t.Fatalf("Cooldown() = %s, want zero", got)
	}
}

func TestHashKey(t *testing.T) {
	key1 := HashKey("prefix", "value1")
	key2 := HashKey("prefix", "value2")
	key3 := HashKey("prefix", "value1")

	if key1 == key2 {
		t.Error("Different values should produce different keys")
	}
	if key1 != key3 {
		t.Error("Same values should produce same keys")
	}
	if len(key1) < 10 {
		t.Error("Key should be reasonably long")
	}
}

func TestManager_Clear(t *testing.T) {
	tmpDir := t.TempDir()
	m := &Manager{cacheDir: tmpDir}

	// Create some cache entries
	_ = m.Set("key1", "value1", time.Hour)
	_ = m.Set("key2", "value2", time.Hour)

	// Clear cache
	err := m.Clear()
	if err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	// Verify entries are gone
	var data string
	found1, _ := m.Get("key1", &data)
	found2, _ := m.Get("key2", &data)

	if found1 || found2 {
		t.Error("Cache entries should be cleared")
	}
}
