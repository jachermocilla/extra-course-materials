package storage

import (
	"testing"

	"github.com/pixperk/plethora/types"
	"github.com/pixperk/plethora/vclock"
)

func TestGetMissing(t *testing.T) {
	s := NewStorage()
	_, ok := s.Get("nope")
	if ok {
		t.Fatal("expected missing key")
	}
}

func TestPutGet(t *testing.T) {
	s := NewStorage()
	c := vclock.NewVClock()
	c.Increment("n1")
	s.Put("k", types.Value{Data: "v", Clock: c})

	vals, ok := s.Get("k")
	if !ok || len(vals) != 1 || vals[0].Data != "v" {
		t.Fatalf("expected [v], got %v", vals)
	}
}

func TestPutDescendantOverwrites(t *testing.T) {
	s := NewStorage()

	c1 := vclock.NewVClock()
	c1.Increment("n1") // {n1: 1}
	s.Put("k", types.Value{Data: "v1", Clock: c1})

	c2 := c1.Copy()
	c2.Increment("n1") // {n1: 2} — descends from c1
	s.Put("k", types.Value{Data: "v2", Clock: c2})

	vals, _ := s.Get("k")
	if len(vals) != 1 {
		t.Fatalf("expected 1 value, got %d", len(vals))
	}
	if vals[0].Data != "v2" {
		t.Fatalf("expected v2, got %s", vals[0].Data)
	}
}

func TestPutConflictCreatesSiblings(t *testing.T) {
	s := NewStorage()

	// two concurrent writes from different nodes, neither descends from the other
	c1 := vclock.NewVClock()
	c1.Increment("n1") // {n1: 1}
	s.Put("k", types.Value{Data: "a", Clock: c1})

	c2 := vclock.NewVClock()
	c2.Increment("n2") // {n2: 1} — conflicts with c1
	s.Put("k", types.Value{Data: "b", Clock: c2})

	vals, _ := s.Get("k")
	if len(vals) != 2 {
		t.Fatalf("expected 2 siblings, got %d", len(vals))
	}
}

func TestPutStaleWriteIgnored(t *testing.T) {
	s := NewStorage()

	c2 := vclock.NewVClock()
	c2.Increment("n1")
	c2.Increment("n1") // {n1: 2}
	s.Put("k", types.Value{Data: "newer", Clock: c2})

	c1 := vclock.NewVClock()
	c1.Increment("n1") // {n1: 1} — ancestor of c2, stale
	s.Put("k", types.Value{Data: "older", Clock: c1})

	vals, _ := s.Get("k")
	if len(vals) != 1 || vals[0].Data != "newer" {
		t.Fatalf("expected [newer], got %v", vals)
	}
}
