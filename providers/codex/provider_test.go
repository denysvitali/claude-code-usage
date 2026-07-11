package codex

import "testing"

func TestToWindow(t *testing.T) {
	w := toWindow("5-Hour", Window{UsedPercent: 42.5, ResetAt: 1700000000})
	if w.Label != "5-Hour" || w.Utilization != 42.5 || w.ResetsAt == nil {
		t.Fatalf("unexpected window: %#v", w)
	}
}

func TestProviderMetadata(t *testing.T) {
	p := NewProvider("token", "account")
	if p.ID() != "codex" || p.ShortName() != "Cdx" {
		t.Fatalf("unexpected metadata: %s %s", p.ID(), p.ShortName())
	}
}
