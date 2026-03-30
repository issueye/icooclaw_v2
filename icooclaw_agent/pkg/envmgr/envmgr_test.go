package envmgr

import "testing"

func TestManager_Merge(t *testing.T) {
	base := New(map[string]string{
		"A": "1",
		"B": "2",
	})
	merged := base.Merge(map[string]string{
		"B": "3",
		"C": "4",
	})

	values := merged.ToMap()
	if values["A"] != "1" {
		t.Fatalf("A = %q, want 1", values["A"])
	}
	if values["B"] != "3" {
		t.Fatalf("B = %q, want 3", values["B"])
	}
	if values["C"] != "4" {
		t.Fatalf("C = %q, want 4", values["C"])
	}
}
