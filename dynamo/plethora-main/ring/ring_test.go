package ring

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/pixperk/plethora/node"
	"github.com/pixperk/plethora/server"
)

func makeNodes(ids ...string) []*node.Node {
	nodes := make([]*node.Node, len(ids))
	for i, id := range ids {
		nodes[i] = node.NewNode(id, fmt.Sprintf("localhost:%d", 5001+i))
	}
	return nodes
}

// makeServedNodes creates nodes with OS-assigned ports and starts gRPC servers.
// All nodes are created first so NewServer can receive the full peer list.
func makeServedNodes(t *testing.T, ids ...string) []*node.Node {
	t.Helper()
	nodes := make([]*node.Node, len(ids))
	listeners := make([]net.Listener, len(ids))

	// phase 1: allocate ports and create nodes
	for i, id := range ids {
		lis, err := net.Listen("tcp", "localhost:0")
		if err != nil {
			t.Fatal(err)
		}
		listeners[i] = lis
		nodes[i] = node.NewNode(id, lis.Addr().String())
	}

	// phase 2: start servers with gossip seeds
	for i, n := range nodes {
		srv := server.NewServer(n, nodes, nil, 10*time.Second)
		go srv.Start(listeners[i])
		t.Cleanup(srv.Stop)
	}
	return nodes
}

func ownerCount(r *Ring, nodeID string) int {
	count := 0
	for _, p := range r.Partitions {
		if p.Token.NodeID == nodeID {
			count++
		}
	}
	return count
}

// helper: N=3, R=2, W=2 (common Dynamo config)
func newTestRing(t *testing.T, q int, nodes []*node.Node) *Ring {
	t.Helper()
	r, err := NewRing(q, 3, 2, 2, nodes)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestNewRingDistribution(t *testing.T) {
	r := newTestRing(t, 6, makeNodes("n1", "n2", "n3"))

	for _, id := range []string{"n1", "n2", "n3"} {
		if c := ownerCount(r, id); c != 2 {
			t.Fatalf("node %s owns %d partitions, want 2", id, c)
		}
	}
}

func TestNewRingRejectsZeroNodes(t *testing.T) {
	_, err := NewRing(6, 3, 2, 2, []*node.Node{})
	if err == nil {
		t.Fatal("expected error for zero nodes")
	}
}

func TestNewRingRejectsIndivisibleQ(t *testing.T) {
	_, err := NewRing(7, 3, 2, 2, makeNodes("n1", "n2", "n3"))
	if err == nil {
		t.Fatal("expected error when Q not divisible by S")
	}
}

func TestNewRingRejectsInvalidQuorum(t *testing.T) {
	// R + W must be > N
	_, err := NewRing(12, 3, 1, 1, makeNodes("n1", "n2", "n3", "n4"))
	if err == nil {
		t.Fatal("expected error when R + W <= N")
	}
}

func TestLookupDeterministic(t *testing.T) {
	r := newTestRing(t, 12, makeNodes("n1", "n2", "n3"))

	first := r.Lookup("user:42")
	for i := 0; i < 100; i++ {
		if got := r.Lookup("user:42"); got.NodeID != first.NodeID {
			t.Fatalf("lookup not deterministic: got %s, want %s", got.NodeID, first.NodeID)
		}
	}
}

func TestLookupReturnsValidNode(t *testing.T) {
	r := newTestRing(t, 12, makeNodes("n1", "n2", "n3"))

	nodeIDs := map[string]bool{"n1": true, "n2": true, "n3": true}
	keys := []string{"a", "b", "c", "foo", "bar", "user:1", "user:999"}
	for _, key := range keys {
		n := r.Lookup(key)
		if n == nil {
			t.Fatalf("Lookup(%q) returned nil", key)
		}
		if !nodeIDs[n.NodeID] {
			t.Fatalf("Lookup(%q) returned unknown node %s", key, n.NodeID)
		}
	}
}

func TestAddNode(t *testing.T) {
	r := newTestRing(t, 12, makeNodes("n1", "n2", "n3"))

	err := r.AddNode(node.NewNode("n4", "localhost:5004"))
	if err != nil {
		t.Fatal(err)
	}

	if len(r.Nodes) != 4 {
		t.Fatalf("expected 4 nodes, got %d", len(r.Nodes))
	}

	for _, id := range []string{"n1", "n2", "n3", "n4"} {
		if c := ownerCount(r, id); c != 3 {
			t.Fatalf("node %s owns %d partitions, want 3", id, c)
		}
	}
}

func TestAddNodeRejectsIndivisibleQ(t *testing.T) {
	r := newTestRing(t, 6, makeNodes("n1", "n2", "n3"))

	err := r.AddNode(node.NewNode("n4", "localhost:5004"))
	if err == nil {
		t.Fatal("expected error when Q not divisible by new S")
	}

	if len(r.Nodes) != 3 {
		t.Fatalf("ring mutated on failed add: got %d nodes", len(r.Nodes))
	}
}

func TestRemoveNode(t *testing.T) {
	r, _ := NewRing(12, 3, 2, 2, makeNodes("n1", "n2", "n3", "n4"))

	err := r.RemoveNode("n4")
	if err != nil {
		t.Fatal(err)
	}

	if len(r.Nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(r.Nodes))
	}

	for _, id := range []string{"n1", "n2", "n3"} {
		if c := ownerCount(r, id); c != 4 {
			t.Fatalf("node %s owns %d partitions, want 4", id, c)
		}
	}

	if c := ownerCount(r, "n4"); c != 0 {
		t.Fatalf("removed node n4 still owns %d partitions", c)
	}
}

func TestRemoveNodeRejectsLastNode(t *testing.T) {
	r, _ := NewRing(6, 1, 1, 1, makeNodes("n1"))
	err := r.RemoveNode("n1")
	if err == nil {
		t.Fatal("expected error when removing last node")
	}
}

func TestRemoveNodeNotFound(t *testing.T) {
	r := newTestRing(t, 6, makeNodes("n1", "n2", "n3"))
	err := r.RemoveNode("n99")
	if err == nil {
		t.Fatal("expected error for non-existent node")
	}
}

func TestPreferenceListReturnsNDistinctNodes(t *testing.T) {
	r, _ := NewRing(12, 3, 2, 2, makeNodes("n1", "n2", "n3", "n4"))

	plist := r.PreferenceList("user:1")
	if len(plist) != 3 {
		t.Fatalf("expected 3 nodes in preference list, got %d", len(plist))
	}

	seen := make(map[string]bool)
	for _, target := range plist {
		if seen[target.Node.NodeID] {
			t.Fatalf("duplicate node %s in preference list", target.Node.NodeID)
		}
		seen[target.Node.NodeID] = true
	}
}

func TestPreferenceListFirstNodeIsOwner(t *testing.T) {
	r := newTestRing(t, 12, makeNodes("n1", "n2", "n3"))

	owner := r.Lookup("user:1")
	plist := r.PreferenceList("user:1")

	if plist[0].Node.NodeID != owner.NodeID {
		t.Fatalf("first node in preference list (%s) != owner (%s)", plist[0].Node.NodeID, owner.NodeID)
	}
}

// --- gRPC tests: these start real servers ---

func TestPutGet(t *testing.T) {
	nodes := makeServedNodes(t, "n1", "n2", "n3")
	r := newTestRing(t, 12, nodes)

	if err := r.Put("user:1", "alice", nil); err != nil {
		t.Fatal(err)
	}
	vals, err := r.Get("user:1")
	if err != nil {
		t.Fatal(err)
	}
	if len(vals) != 1 || vals[0].Data != "alice" {
		t.Fatalf("expected [alice], got %v", vals)
	}
}

func TestGetMissing(t *testing.T) {
	nodes := makeServedNodes(t, "n1", "n2", "n3")
	r := newTestRing(t, 12, nodes)

	vals, err := r.Get("nope")
	if err != nil {
		t.Fatal(err)
	}
	if vals != nil {
		t.Fatal("expected nil for missing key")
	}
}

func TestPutReplicatesToNNodes(t *testing.T) {
	nodes := makeServedNodes(t, "n1", "n2", "n3", "n4")
	r, _ := NewRing(12, 3, 2, 2, nodes)

	r.Put("user:1", "alice", nil)

	plist := r.PreferenceList("user:1")

	// verify preference list nodes have the value (check via local storage)
	for _, target := range plist {
		vals, ok := target.Node.Get("user:1")
		if !ok || len(vals) == 0 {
			t.Fatalf("node %s in preference list missing the value", target.Node.NodeID)
		}
		if vals[0].Data != "alice" {
			t.Fatalf("node %s has wrong data: %s", target.Node.NodeID, vals[0].Data)
		}
	}

	// verify non-preference nodes don't have the value
	prefSet := make(map[string]bool)
	for _, target := range plist {
		prefSet[target.Node.NodeID] = true
	}
	for _, n := range r.Nodes {
		if prefSet[n.NodeID] {
			continue
		}
		_, ok := n.Get("user:1")
		if ok {
			t.Fatalf("non-preference node %s has the value", n.NodeID)
		}
	}
}

func TestSloppyQuorumHintedHandoff(t *testing.T) {
	// start 4 served nodes, figure out the preference list for a key,
	// then kill one preferred node and verify put still succeeds
	nodes := make([]*node.Node, 4)
	listeners := make([]net.Listener, 4)
	servers := make([]*server.Server, 4)

	ids := []string{"n1", "n2", "n3", "n4"}
	for i, id := range ids {
		lis, err := net.Listen("tcp", "localhost:0")
		if err != nil {
			t.Fatal(err)
		}
		listeners[i] = lis
		nodes[i] = node.NewNode(id, lis.Addr().String())
	}
	for i, n := range nodes {
		servers[i] = server.NewServer(n, nodes, nil, 10*time.Second)
		go servers[i].Start(listeners[i])
		t.Cleanup(servers[i].Stop)
	}

	r, _ := NewRing(12, 3, 2, 2, nodes)

	// get the ideal preference list (all alive) to find victim
	plist := r.PreferenceList("sloppy-key")
	victim := plist[len(plist)-1]

	// stop the victim's server and tell the ring it's dead
	for i, n := range nodes {
		if n.NodeID == victim.Node.NodeID {
			servers[i].Stop()
			break
		}
	}
	dead := map[string]bool{victim.Node.NodeID: true}
	r.IsAlive = func(nodeID string) bool { return !dead[nodeID] }

	// put should still succeed via sloppy quorum (stand-in picked automatically)
	if err := r.Put("sloppy-key", "hinted-val", nil); err != nil {
		t.Fatalf("put failed with one node down: %v", err)
	}

	// find which node is the stand-in by checking hint stores
	prefSet := make(map[string]bool)
	for _, target := range plist {
		prefSet[target.Node.NodeID] = true
	}
	var standIn *node.Node
	for _, n := range nodes {
		if prefSet[n.NodeID] {
			continue
		}
		hints := n.DrainHints(victim.Node.NodeID)
		if len(hints) > 0 {
			standIn = n
			if hints[0].Value.Data != "hinted-val" {
				t.Fatalf("hint data mismatch: got %s", hints[0].Value.Data)
			}
			break
		}
	}
	if standIn == nil {
		t.Fatal("no stand-in node received the hint")
	}
}

func TestPutOverwriteVersions(t *testing.T) {
	nodes := makeServedNodes(t, "n1", "n2", "n3")
	r := newTestRing(t, 12, nodes)

	r.Put("k", "v1", nil)

	vals, _ := r.Get("k")
	ctx := vals[0].Clock

	r.Put("k", "v2", ctx)

	vals, _ = r.Get("k")
	if len(vals) != 1 || vals[0].Data != "v2" {
		t.Fatalf("expected [v2], got %v", vals)
	}
}
