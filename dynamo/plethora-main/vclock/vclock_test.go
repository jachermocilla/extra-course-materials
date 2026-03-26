package vclock

import "testing"

func TestIncrement(t *testing.T) {
	vc := NewVClock()
	vc.Increment("n1")
	vc.Increment("n1")
	vc.Increment("n2")

	if vc["n1"] != 2 || vc["n2"] != 1 {
		t.Fatalf("expected {n1:2, n2:1}, got %v", vc)
	}
}

func TestDescends(t *testing.T) {
	a := VClock{"n1": 2, "n2": 1}
	b := VClock{"n1": 1, "n2": 1}

	if !a.Descends(b) {
		t.Fatal("a should descend from b")
	}
	if b.Descends(a) {
		t.Fatal("b should not descend from a")
	}
}

func TestDescendsEqual(t *testing.T) {
	a := VClock{"n1": 1}
	b := VClock{"n1": 1}

	if !a.Descends(b) || !b.Descends(a) {
		t.Fatal("equal clocks should descend from each other")
	}
}

func TestDescendsEmptyClock(t *testing.T) {
	a := VClock{"n1": 1}
	empty := NewVClock()

	if !a.Descends(empty) {
		t.Fatal("any clock should descend from empty")
	}
	if empty.Descends(a) {
		t.Fatal("empty should not descend from non-empty")
	}
}

func TestConflicts(t *testing.T) {
	a := VClock{"n1": 1}
	b := VClock{"n2": 1}

	if !a.Conflicts(b) {
		t.Fatal("expected conflict")
	}
	if a.Conflicts(a) {
		t.Fatal("should not conflict with itself")
	}
}

func TestMerge(t *testing.T) {
	a := VClock{"n1": 2, "n2": 1}
	b := VClock{"n1": 1, "n2": 3}

	merged := a.Merge(b)
	if merged["n1"] != 2 || merged["n2"] != 3 {
		t.Fatalf("expected {n1:2, n2:3}, got %v", merged)
	}

	// originals unchanged
	if a["n2"] != 1 || b["n1"] != 1 {
		t.Fatal("merge mutated original clocks")
	}
}

func TestCopy(t *testing.T) {
	vc := VClock{"n1": 1}
	cp := vc.Copy()
	cp.Increment("n1")

	if vc["n1"] != 1 {
		t.Fatal("copy mutated original")
	}
	if cp["n1"] != 2 {
		t.Fatal("copy should have incremented value")
	}
}
