package providers

import "testing"

func TestAllIsDeterministic(t *testing.T) {
	all := All()
	if len(all) != 6 || all[0].ID != "claude" || all[5].ID != "zai" {
		t.Fatalf("unexpected registry: %#v", all)
	}
}
