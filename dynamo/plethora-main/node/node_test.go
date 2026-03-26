package node

import (
	"testing"
)

func TestGetMissing(t *testing.T) {
	n := NewNode("n1", "localhost:5001")
	_, ok := n.Get("nope")
	if ok {
		t.Fatal("expected missing key")
	}
}

func TestPutGet(t *testing.T) {
	n := NewNode("n1", "localhost:5001")
	n.Put("k", "hello", nil) // nil clock = fresh write

	vals, ok := n.Get("k")
	if !ok || len(vals) != 1 || vals[0].Data != "hello" {
		t.Fatalf("expected [hello], got %v", vals)
	}
	if vals[0].Clock["n1"] != 1 {
		t.Fatalf("expected clock {n1: 1}, got %v", vals[0].Clock)
	}
}

func TestPutOverwriteIncrementsVersion(t *testing.T) {
	n := NewNode("n1", "localhost:5001")
	n.Put("k", "v1", nil)

	// get the clock from the first write, pass it as context
	vals, _ := n.Get("k")
	ctx := vals[0].Clock

	n.Put("k", "v2", ctx)

	vals, _ = n.Get("k")
	if len(vals) != 1 {
		t.Fatalf("expected 1 value, got %d", len(vals))
	}
	if vals[0].Data != "v2" || vals[0].Clock["n1"] != 2 {
		t.Fatalf("expected v2 with clock {n1: 2}, got %s %v", vals[0].Data, vals[0].Clock)
	}
}

func TestTwoNodesSameKey(t *testing.T) {
	n1 := NewNode("n1", "localhost:5001")
	n2 := NewNode("n2", "localhost:5002")

	n1.Put("k", "from-n1", nil)
	n2.Put("k", "from-n2", nil)

	// separate stores, each has 1 value
	v1, _ := n1.Get("k")
	v2, _ := n2.Get("k")
	if len(v1) != 1 || v1[0].Data != "from-n1" {
		t.Fatalf("n1 expected [from-n1], got %v", v1)
	}
	if len(v2) != 1 || v2[0].Data != "from-n2" {
		t.Fatalf("n2 expected [from-n2], got %v", v2)
	}
}
