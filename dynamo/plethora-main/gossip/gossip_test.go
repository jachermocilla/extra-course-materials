package gossip

import (
	"testing"
	"time"
)

const tFail = 5 * time.Second

func TestNewMemberListContainsSelf(t *testing.T) {
	m := NewMemberList("n1", "localhost:5001", tFail)

	entries := m.Entries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].NodeID != "n1" {
		t.Fatalf("expected self entry n1, got %s", entries[0].NodeID)
	}
}

func TestAddSeed(t *testing.T) {
	m := NewMemberList("n1", "localhost:5001", tFail)
	m.AddSeed("n2", "localhost:5002")
	m.AddSeed("n3", "localhost:5003")

	entries := m.Entries()
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
}

func TestAddSeedIgnoresDuplicate(t *testing.T) {
	m := NewMemberList("n1", "localhost:5001", tFail)
	m.AddSeed("n2", "localhost:5002")
	m.AddSeed("n2", "localhost:9999") // duplicate, should be ignored

	entries := m.Entries()
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	for _, e := range entries {
		if e.NodeID == "n2" && e.Addr != "localhost:5002" {
			t.Fatalf("duplicate seed overwrote addr: got %s", e.Addr)
		}
	}
}

func TestTickIncrementsHeartbeat(t *testing.T) {
	m := NewMemberList("n1", "localhost:5001", tFail)

	m.Tick()
	m.Tick()
	m.Tick()

	entries := m.Entries()
	if entries[0].Heartbeat != 3 {
		t.Fatalf("expected heartbeat 3, got %d", entries[0].Heartbeat)
	}
}

func TestMergeUpdatesHigherHeartbeat(t *testing.T) {
	m := NewMemberList("n1", "localhost:5001", tFail)
	m.AddSeed("n2", "localhost:5002")

	m.Merge([]MemberEntry{
		{NodeID: "n2", Addr: "localhost:5002", Heartbeat: 10},
	})

	for _, e := range m.Entries() {
		if e.NodeID == "n2" && e.Heartbeat != 10 {
			t.Fatalf("expected heartbeat 10, got %d", e.Heartbeat)
		}
	}
}

func TestMergeIgnoresLowerHeartbeat(t *testing.T) {
	m := NewMemberList("n1", "localhost:5001", tFail)

	// first merge sets heartbeat to 10
	m.Merge([]MemberEntry{
		{NodeID: "n2", Addr: "localhost:5002", Heartbeat: 10},
	})
	// second merge with lower heartbeat should be ignored
	m.Merge([]MemberEntry{
		{NodeID: "n2", Addr: "localhost:5002", Heartbeat: 5},
	})

	for _, e := range m.Entries() {
		if e.NodeID == "n2" && e.Heartbeat != 10 {
			t.Fatalf("expected heartbeat 10, got %d", e.Heartbeat)
		}
	}
}

func TestMergeNeverOverwritesSelf(t *testing.T) {
	m := NewMemberList("n1", "localhost:5001", tFail)
	m.Tick() // heartbeat = 1

	m.Merge([]MemberEntry{
		{NodeID: "n1", Addr: "localhost:5001", Heartbeat: 999},
	})

	for _, e := range m.Entries() {
		if e.NodeID == "n1" && e.Heartbeat != 1 {
			t.Fatalf("self entry was overwritten: heartbeat = %d", e.Heartbeat)
		}
	}
}

func TestMergeDiscoversNewNodes(t *testing.T) {
	m := NewMemberList("n1", "localhost:5001", tFail)

	m.Merge([]MemberEntry{
		{NodeID: "n2", Addr: "localhost:5002", Heartbeat: 5},
		{NodeID: "n3", Addr: "localhost:5003", Heartbeat: 3},
	})

	entries := m.Entries()
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
}

func TestIsAlive(t *testing.T) {
	m := NewMemberList("n1", "localhost:5001", tFail)
	m.AddSeed("n2", "localhost:5002")

	if !m.IsAlive("n1") {
		t.Fatal("self should be alive")
	}
	if !m.IsAlive("n2") {
		t.Fatal("freshly added seed should be alive")
	}
	if m.IsAlive("n99") {
		t.Fatal("unknown node should not be alive")
	}
}

func TestIsAliveDetectsFailure(t *testing.T) {
	shortFail := 50 * time.Millisecond
	m := NewMemberList("n1", "localhost:5001", shortFail)
	m.AddSeed("n2", "localhost:5002")

	if !m.IsAlive("n2") {
		t.Fatal("n2 should be alive initially")
	}

	time.Sleep(100 * time.Millisecond)

	if m.IsAlive("n2") {
		t.Fatal("n2 should be dead after tFail with no heartbeat updates")
	}

	// self should still be alive (Tick refreshes timestamp)
	m.Tick()
	if !m.IsAlive("n1") {
		t.Fatal("self should be alive after Tick")
	}
}

func TestAliveExcludesSelf(t *testing.T) {
	m := NewMemberList("n1", "localhost:5001", tFail)
	m.AddSeed("n2", "localhost:5002")

	alive := m.Alive()
	if len(alive) != 1 {
		t.Fatalf("expected 1 alive peer, got %d", len(alive))
	}
	if alive[0].NodeID != "n2" {
		t.Fatalf("expected n2, got %s", alive[0].NodeID)
	}
}

func TestAliveExcludesDead(t *testing.T) {
	shortFail := 50 * time.Millisecond
	m := NewMemberList("n1", "localhost:5001", shortFail)
	m.AddSeed("n2", "localhost:5002")
	m.AddSeed("n3", "localhost:5003")

	time.Sleep(100 * time.Millisecond)

	// only merge n2 so it gets a fresh timestamp
	m.Merge([]MemberEntry{
		{NodeID: "n2", Addr: "localhost:5002", Heartbeat: 1},
	})

	alive := m.Alive()
	if len(alive) != 1 {
		t.Fatalf("expected 1 alive peer, got %d", len(alive))
	}
	if alive[0].NodeID != "n2" {
		t.Fatalf("expected n2 alive, got %s", alive[0].NodeID)
	}
}

func TestRandomPeerReturnsNonSelf(t *testing.T) {
	m := NewMemberList("n1", "localhost:5001", tFail)
	m.AddSeed("n2", "localhost:5002")
	m.AddSeed("n3", "localhost:5003")

	for i := 0; i < 50; i++ {
		peer, ok := m.RandomPeer()
		if !ok {
			t.Fatal("expected a peer")
		}
		if peer.NodeID == "n1" {
			t.Fatal("RandomPeer returned self")
		}
	}
}

func TestRandomPeerReturnsFalseWhenAlone(t *testing.T) {
	m := NewMemberList("n1", "localhost:5001", tFail)

	_, ok := m.RandomPeer()
	if ok {
		t.Fatal("expected no peers when alone")
	}
}

func TestGossipRoundTrip(t *testing.T) {
	// simulate two nodes gossiping
	m1 := NewMemberList("n1", "localhost:5001", tFail)
	m2 := NewMemberList("n2", "localhost:5002", tFail)

	// each ticks a few times independently
	for i := 0; i < 5; i++ {
		m1.Tick()
	}
	for i := 0; i < 3; i++ {
		m2.Tick()
	}

	// n1 sends its list to n2, n2 merges
	m2.Merge(m1.Entries())

	// n2 sends its list back to n1, n1 merges
	m1.Merge(m2.Entries())

	// both should now know about each other
	e1 := m1.Entries()
	e2 := m2.Entries()
	if len(e1) != 2 {
		t.Fatalf("n1 expected 2 entries, got %d", len(e1))
	}
	if len(e2) != 2 {
		t.Fatalf("n2 expected 2 entries, got %d", len(e2))
	}

	// verify heartbeats are correct
	for _, e := range e1 {
		if e.NodeID == "n1" && e.Heartbeat != 5 {
			t.Fatalf("n1 self heartbeat should be 5, got %d", e.Heartbeat)
		}
		if e.NodeID == "n2" && e.Heartbeat != 3 {
			t.Fatalf("n1 view of n2 heartbeat should be 3, got %d", e.Heartbeat)
		}
	}
	for _, e := range e2 {
		if e.NodeID == "n2" && e.Heartbeat != 3 {
			t.Fatalf("n2 self heartbeat should be 3, got %d", e.Heartbeat)
		}
		if e.NodeID == "n1" && e.Heartbeat != 5 {
			t.Fatalf("n2 view of n1 heartbeat should be 5, got %d", e.Heartbeat)
		}
	}
}

func TestThreeNodeGossipConvergence(t *testing.T) {
	m1 := NewMemberList("n1", "localhost:5001", tFail)
	m2 := NewMemberList("n2", "localhost:5002", tFail)
	m3 := NewMemberList("n3", "localhost:5003", tFail)

	// only n1 knows about n2 as seed, only n2 knows about n3 as seed
	// n1 does NOT know about n3 directly
	m1.AddSeed("n2", "localhost:5002")
	m2.AddSeed("n3", "localhost:5003")

	m1.Tick()
	m2.Tick()
	m3.Tick()

	// round 1: n1 gossips with n2
	m2.Merge(m1.Entries()) // n2 learns n1's heartbeat
	m1.Merge(m2.Entries()) // n1 learns about n3 transitively through n2

	// round 2: n2 gossips with n3
	m3.Merge(m2.Entries()) // n3 learns about n1 transitively through n2
	m2.Merge(m3.Entries()) // n2 gets n3's latest

	// all three should know about all three
	if len(m1.Entries()) != 3 {
		t.Fatalf("n1 expected 3 entries, got %d", len(m1.Entries()))
	}
	if len(m2.Entries()) != 3 {
		t.Fatalf("n2 expected 3 entries, got %d", len(m2.Entries()))
	}
	if len(m3.Entries()) != 3 {
		t.Fatalf("n3 expected 3 entries, got %d", len(m3.Entries()))
	}
}
