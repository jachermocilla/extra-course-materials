// chord_dht.go
// A simple in-memory implementation of the Chord Distributed Hash Table (DHT).
//
// Chord organizes nodes in a ring. Each node is responsible for storing keys
// whose hash falls between it and its predecessor. Nodes use "finger tables"
// to route lookups in O(log N) hops.
//
// This example runs everything in a single process (no networking) so you
// can read, run, and understand it without any distributed-systems baggage.

package main

import (
	"crypto/sha1"
	"fmt"
	"math/big"
)

// ── Constants ────────────────────────────────────────────────────────────────

const M = 8 // Number of bits in the key space. Ring size = 2^M = 256.

// ringSize is 2^M, the total number of positions on the ring.
var ringSize = new(big.Int).Exp(big.NewInt(2), big.NewInt(M), nil)

// ── Helpers ──────────────────────────────────────────────────────────────────

// hash turns any string into a position on the ring (0 … 2^M - 1).
func hash(key string) uint64 {
	h := sha1.Sum([]byte(key))
	// Take the first 8 bytes and mod by ring size.
	val := new(big.Int).SetBytes(h[:])
	val.Mod(val, ringSize)
	return val.Uint64()
}

// between reports whether id is in the half-open interval (start, end] on the
// ring, wrapping around at 2^M.
func between(id, start, end uint64) bool {
	if start < end {
		return id > start && id <= end
	}
	// Wrap-around case: e.g. start=250, end=10 → interval covers 251-255 and 0-10
	return id > start || id <= end
}

// ── Node ─────────────────────────────────────────────────────────────────────

// Node represents one peer in the Chord ring.
type Node struct {
	ID          uint64
	Name        string            // human-readable label (e.g. "Node-A")
	Successor   *Node             // next node clockwise on the ring
	Predecessor *Node             // previous node clockwise on the ring
	Fingers     [M]*Node          // finger table for fast routing
	Data        map[uint64]string // keys this node is responsible for
}

// newNode creates a node whose ring position is derived from its name.
func newNode(name string) *Node {
	return &Node{
		ID:   hash(name),
		Name: name,
		Data: make(map[uint64]string),
	}
}

// String makes nodes print nicely.
func (n *Node) String() string {
	return fmt.Sprintf("%s(id=%d)", n.Name, n.ID)
}

// ── Chord ring operations ────────────────────────────────────────────────────

// findSuccessor returns the node responsible for a given id.
// It walks the finger table to jump as far forward as possible each hop.
func (n *Node) findSuccessor(id uint64) *Node {
	// Base case: id falls between this node and its successor.
	if between(id, n.ID, n.Successor.ID) {
		return n.Successor
	}
	// Ask the closest preceding finger to continue the search.
	closest := n.closestPrecedingFinger(id)
	if closest == n {
		// Avoid infinite loop on a single-node ring.
		return n.Successor
	}
	return closest.findSuccessor(id)
}

// closestPrecedingFinger scans the finger table backwards and returns the
// finger that is closest to id without overshooting it.
func (n *Node) closestPrecedingFinger(id uint64) *Node {
	for i := M - 1; i >= 0; i-- {
		f := n.Fingers[i]
		if f != nil && between(f.ID, n.ID, id) {
			return f
		}
	}
	return n
}

// join connects this node into the ring that already contains 'existing'.
// Pass nil to start a brand-new ring with just this node.
func (n *Node) join(existing *Node) {
	if existing == nil {
		// Single-node ring: the node is its own successor and predecessor.
		n.Successor = n
		n.Predecessor = n
		for i := 0; i < M; i++ {
			n.Fingers[i] = n
		}
		return
	}

	// Ask an existing node to find our successor.
	n.Successor = existing.findSuccessor(n.ID)
	n.Predecessor = n.Successor.Predecessor

	// Stitch us into the linked list.
	n.Predecessor.Successor = n
	n.Successor.Predecessor = n

	// Migrate keys that now belong to us (keys ≤ our ID that were on successor).
	for k, v := range n.Successor.Data {
		if between(k, n.Predecessor.ID, n.ID) {
			n.Data[k] = v
			delete(n.Successor.Data, k)
		}
	}
}

// fixFingers rebuilds this node's finger table by asking the ring.
// Finger i points to the node responsible for (n.ID + 2^i) mod 2^M.
func (n *Node) fixFingers() {
	for i := 0; i < M; i++ {
		offset := uint64(1) << i                       // 2^i
		target := (n.ID + offset) % uint64(ringSize.Uint64())
		n.Fingers[i] = n.findSuccessor(target)
	}
}

// ── Key/Value store ───────────────────────────────────────────────────────────

// Put stores a key/value pair in the ring.
func (n *Node) Put(key, value string) {
	id := hash(key)
	responsible := n.findSuccessor(id)
	responsible.Data[id] = value
	fmt.Printf("  PUT  key=%q (id=%d) → stored on %s\n", key, id, responsible)
}

// Get retrieves a value from the ring.
func (n *Node) Get(key string) (string, bool) {
	id := hash(key)
	responsible := n.findSuccessor(id)
	val, ok := responsible.Data[id]
	fmt.Printf("  GET  key=%q (id=%d) → found on %s\n", key, id, responsible)
	return val, ok
}

// ── Demo ─────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Chord DHT Demo (M=8, ring size=256) ===\n")

	// 1. Create nodes and build the ring.
	names := []string{"JACH", "Jere", "Emman", "Edward", "Allen"}
	nodes := make([]*Node, len(names))
	for i, name := range names {
		nodes[i] = newNode(name)
	}

	fmt.Println("── Building ring ──")
	nodes[0].join(nil) // JACH starts a new ring.
	fmt.Printf("  %s joined (new ring)\n", nodes[0])

	for i := 1; i < len(nodes); i++ {
		nodes[i].join(nodes[0]) // everyone else joins via JACH
		fmt.Printf("  %s joined via %s\n", nodes[i], nodes[0])
	}

	// 2. Fix finger tables so routing is efficient.
	fmt.Println("\n── Fixing finger tables ──")
	for _, n := range nodes {
		n.fixFingers()
		fmt.Printf("  fixed fingers for %s\n", n)
	}

	// 3. Print the ring order (successor chain).
	fmt.Println("\n── Ring order (successor chain) ──")
	cur := nodes[0]
	for i := 0; i < len(nodes); i++ {
		fmt.Printf("  %s → successor: %s\n", cur, cur.Successor)
		cur = cur.Successor
	}

	// 4. Store and retrieve some values.
	fmt.Println("\n── Storing key/value pairs ──")
	kv := map[string]string{
		"apple":  "a red fruit",
		"banana": "a yellow fruit",
		"cherry": "a small red fruit",
		"date":   "a sweet brown fruit",
		"elder":  "a dark berry",
	}
	for k, v := range kv {
		nodes[0].Put(k, v) // We always start lookups from JACH.
	}

	fmt.Println("\n── Retrieving values ──")
	for k := range kv {
		val, ok := nodes[0].Get(k)
		if ok {
			fmt.Printf("         value = %q\n", val)
		} else {
			fmt.Printf("         NOT FOUND\n")
		}
	}

	// 5. Show which keys each node owns.
	fmt.Println("\n── Key distribution across nodes ──")
	for _, n := range nodes {
		fmt.Printf("  %s owns %d key(s): %v\n", n, len(n.Data), n.Data)
	}
}
