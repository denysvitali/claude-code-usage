package providers

import "testing"

func TestAllIsDeterministic(t *testing.T) {
	all := All()
	if len(all) != 5 || all[0].ID != "claude" || all[4].ID != "minimax" {
		t.Fatalf("unexpected registry: %#v", all)
	}
}
