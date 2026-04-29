// Anti-Entropy Gossip on a Random Graph in Go
// Each node holds a key-value store. A background ticker fires every second,
// picks a random peer, and does a push-pull sync: exchange digests, send
// missing or stale entries in both directions. Nodes converge to identical
// state without any central coordinator.

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// ---------------------------------------------------------------------------
// Data model
// ---------------------------------------------------------------------------

// Entry is a versioned key-value pair. Higher version wins on conflict.
type Entry struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Version int64  `json:"version"`
}

// Digest is a lightweight summary: key → current version.
// Sent first so the peer can calculate what to send back.
type Digest map[string]int64

// ---------------------------------------------------------------------------
// Wire messages
// ---------------------------------------------------------------------------

type MsgType string

const (
	MsgSyncReq  MsgType = "SYNC_REQ"
	MsgSyncResp MsgType = "SYNC_RESP"
)

type WireMsg struct {
	Type    MsgType  `json:"type"`
	From    string   `json:"from"`
	Digest  Digest   `json:"digest,omitempty"`  // used in REQ and RESP
	Updates []Entry  `json:"updates,omitempty"` // entries the receiver is missing
}

// ---------------------------------------------------------------------------
// Metrics
// ---------------------------------------------------------------------------

type Metrics struct {
	SyncRounds    int64 // total push-pull rounds completed
	EntriesSent   int64 // entries we pushed to peers
	EntriesRecv   int64 // entries we received from peers
	StaleIgnored  int64 // updates rejected because our version was newer
}

// ---------------------------------------------------------------------------
// Node
// ---------------------------------------------------------------------------

type Node struct {
	ID        string
	Addr      string
	Neighbors []string

	mu       sync.Mutex
	store    map[string]Entry // the replicated KV store

	listener net.Listener
	rng      *rand.Rand
	ticker   *time.Ticker
	quit     chan struct{}

	metrics *Metrics
}

func NewNode(id, addr string, metrics *Metrics) *Node {
	return &Node{
		ID:      id,
		Addr:    addr,
		store:   make(map[string]Entry),
		rng:     rand.New(rand.NewSource(time.Now().UnixNano())),
		quit:    make(chan struct{}),
		metrics: metrics,
	}
}

func (n *Node) AddNeighbor(addr string) {
	n.Neighbors = append(n.Neighbors, addr)
}

// Put writes a key-value pair with a monotonically increasing version.
func (n *Node) Put(key, value string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	ver := time.Now().UnixNano()
	n.store[key] = Entry{Key: key, Value: value, Version: ver}
	log.Printf("[%s] PUT  %q = %q  (v=%d)", n.ID, key, value, ver)
}

// ---------------------------------------------------------------------------
// Store helpers
// ---------------------------------------------------------------------------

// getDigest returns key → version for all local entries (caller holds no lock).
func (n *Node) getDigest() Digest {
	n.mu.Lock()
	defer n.mu.Unlock()
	d := make(Digest, len(n.store))
	for k, e := range n.store {
		d[k] = e.Version
	}
	return d
}

// delta returns entries that are newer than what the remote digest describes.
// i.e. what we should SEND to bring the remote up to date.
func (n *Node) delta(remoteDigest Digest) []Entry {
	n.mu.Lock()
	defer n.mu.Unlock()
	var out []Entry
	for k, e := range n.store {
		remoteVer, exists := remoteDigest[k]
		if !exists || e.Version > remoteVer {
			out = append(out, e)
		}
	}
	return out
}

// merge applies incoming entries, keeping the highest version for each key.
func (n *Node) merge(entries []Entry) (applied int) {
	n.mu.Lock()
	defer n.mu.Unlock()
	for _, e := range entries {
		local, exists := n.store[e.Key]
		if !exists || e.Version > local.Version {
			n.store[e.Key] = e
			applied++
			log.Printf("[%s] MERGE %q = %q  (v=%d)", n.ID, e.Key, e.Value, e.Version)
		} else {
			atomic.AddInt64(&n.metrics.StaleIgnored, 1)
		}
	}
	return
}

// PrintStore logs the full local KV state.
func (n *Node) PrintStore() {
	n.mu.Lock()
	defer n.mu.Unlock()
	fmt.Printf("  [%s] store (%d keys):\n", n.ID, len(n.store))
	for k, e := range n.store {
		fmt.Printf("    %-20s = %-20s  (v=%d)\n", k, e.Value, e.Version)
	}
}

// ---------------------------------------------------------------------------
// TCP server
// ---------------------------------------------------------------------------

func (n *Node) Start(interval time.Duration) error {
	ln, err := net.Listen("tcp", n.Addr)
	if err != nil {
		return fmt.Errorf("node %s listen error: %w", n.ID, err)
	}
	n.listener = ln

	// Accept incoming sync requests
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go n.handleConn(conn)
		}
	}()

	// Anti-entropy ticker
	n.ticker = time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-n.ticker.C:
				n.antiEntropyRound()
			case <-n.quit:
				return
			}
		}
	}()

	return nil
}

func (n *Node) Stop() {
	close(n.quit)
	if n.ticker != nil {
		n.ticker.Stop()
	}
	if n.listener != nil {
		n.listener.Close()
	}
}

// ---------------------------------------------------------------------------
// Anti-entropy round (initiator side)
// ---------------------------------------------------------------------------

func (n *Node) antiEntropyRound() {
	peer := n.pickRandomPeer()
	if peer == "" {
		return
	}

	conn, err := net.DialTimeout("tcp", peer, 2*time.Second)
	if err != nil {
		log.Printf("[%s] dial %s failed: %v", n.ID, peer, err)
		return
	}
	defer conn.Close()

	enc := json.NewEncoder(conn)
	dec := json.NewDecoder(bufio.NewReader(conn))

	// Step 1 — send our digest
	req := WireMsg{
		Type:   MsgSyncReq,
		From:   n.Addr,
		Digest: n.getDigest(),
	}
	if err := enc.Encode(req); err != nil {
		log.Printf("[%s] encode req error: %v", n.ID, err)
		return
	}

	// Step 2 — receive peer's response: entries we're missing + peer's digest
	var resp WireMsg
	if err := dec.Decode(&resp); err != nil {
		log.Printf("[%s] decode resp error: %v", n.ID, err)
		return
	}

	// Step 3 — merge what the peer sent us
	applied := n.merge(resp.Updates)
	atomic.AddInt64(&n.metrics.EntriesRecv, int64(len(resp.Updates)))

	// Step 4 — send back what the peer is missing (push-pull completes here)
	pushback := n.delta(resp.Digest)
	reply := WireMsg{
		Type:    MsgSyncResp,
		From:    n.Addr,
		Updates: pushback,
	}
	if err := enc.Encode(reply); err != nil {
		log.Printf("[%s] encode pushback error: %v", n.ID, err)
		return
	}
	atomic.AddInt64(&n.metrics.EntriesSent, int64(len(pushback)))
	atomic.AddInt64(&n.metrics.SyncRounds, 1)

	log.Printf("[%s] ⇄ SYNC with %s — recv=%d applied=%d pushed=%d",
		n.ID, peer, len(resp.Updates), applied, len(pushback))
}

// ---------------------------------------------------------------------------
// handleConn — responder side
// ---------------------------------------------------------------------------

func (n *Node) handleConn(conn net.Conn) {
	defer conn.Close()
	enc := json.NewEncoder(conn)
	dec := json.NewDecoder(bufio.NewReader(conn))

	// Step 1 — read initiator's digest
	var req WireMsg
	if err := dec.Decode(&req); err != nil {
		log.Printf("[%s] decode req error: %v", n.ID, err)
		return
	}

	// Step 2 — calculate what the initiator is missing, send it + our digest
	missing := n.delta(req.Digest)
	resp := WireMsg{
		Type:    MsgSyncResp,
		From:    n.Addr,
		Digest:  n.getDigest(), // so initiator can push back to us
		Updates: missing,
	}
	if err := enc.Encode(resp); err != nil {
		log.Printf("[%s] encode resp error: %v", n.ID, err)
		return
	}
	atomic.AddInt64(&n.metrics.EntriesSent, int64(len(missing)))

	// Step 3 — read what the initiator is pushing back to us
	var pushback WireMsg
	if err := dec.Decode(&pushback); err != nil {
		log.Printf("[%s] decode pushback error: %v", n.ID, err)
		return
	}
	applied := n.merge(pushback.Updates)
	atomic.AddInt64(&n.metrics.EntriesRecv, int64(len(pushback.Updates)))

	log.Printf("[%s] ⇄ SYNC from %s — sent=%d recv=%d applied=%d",
		n.ID, req.From, len(missing), len(pushback.Updates), applied)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func (n *Node) pickRandomPeer() string {
	if len(n.Neighbors) == 0 {
		return ""
	}
	return n.Neighbors[n.rng.Intn(len(n.Neighbors))]
}

// ---------------------------------------------------------------------------
// Random graph G(n, p)
// ---------------------------------------------------------------------------

func buildGraph(n int, p float64, basePort int, metrics *Metrics) []*Node {
	nodes := make([]*Node, n)
	for i := 0; i < n; i++ {
		addr := fmt.Sprintf(":%d", basePort+i)
		nodes[i] = NewNode(fmt.Sprintf("Node%02d", i), addr, metrics)
	}

	fmt.Println("=== Adjacency List ===")
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	edges := 0
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			if rng.Float64() < p {
				nodes[i].AddNeighbor(nodes[j].Addr)
				nodes[j].AddNeighbor(nodes[i].Addr)
				edges++
			}
		}
		fmt.Printf("  %s → %v\n", nodes[i].ID, nodes[i].Neighbors)
	}
	fmt.Printf("  %d nodes, %d edges, p=%.2f\n\n", n, edges, p)
	return nodes
}

// converged returns true if all nodes have identical store contents.
func converged(nodes []*Node) bool {
	if len(nodes) == 0 {
		return true
	}
	ref := nodes[0].getDigest()
	for _, nd := range nodes[1:] {
		d := nd.getDigest()
		if len(d) != len(ref) {
			return false
		}
		for k, v := range ref {
			if d[k] != v {
				return false
			}
		}
	}
	return true
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

func main() {
	const (
		n           = 5
		p           = 0.60
		basePort    = 9300
		syncInterval = time.Second
	)

	metrics := &Metrics{}
	nodes := buildGraph(n, p, basePort, metrics)

	// Seed each node with different keys so there is something to reconcile
	nodes[0].Put("config/leader",   "node00")
	nodes[0].Put("config/timeout",  "30s")
	nodes[1].Put("config/replicas", "3")
	nodes[1].Put("user/alice",      "active")
	nodes[2].Put("user/bob",        "inactive")
	nodes[3].Put("config/leader",   "node03") // deliberate conflict with node00
	nodes[4].Put("feature/dark-mode", "enabled")

	fmt.Println("=== Initial State (before sync) ===")
	for _, nd := range nodes {
		nd.PrintStore()
	}
	fmt.Println()

	// Start all nodes with anti-entropy tickers
	for _, nd := range nodes {
		if err := nd.Start(syncInterval); err != nil {
			log.Fatalf("start error: %v", err)
		}
	}
	defer func() {
		for _, nd := range nodes {
			nd.Stop()
		}
	}()

	// Wait for convergence (poll every 500ms, timeout after 10s)
	fmt.Println("=== Anti-Entropy Running ===")
	start := time.Now()
	for time.Since(start) < 10*time.Second {
		time.Sleep(500 * time.Millisecond)
		if converged(nodes) {
			fmt.Printf("\n✔ Converged in %.2fs\n\n", time.Since(start).Seconds())
			break
		}
	}

	if !converged(nodes) {
		fmt.Println("\n✘ Did not fully converge within 10s")
	}

	fmt.Println("=== Final State (after sync) ===")
	for _, nd := range nodes {
		nd.PrintStore()
	}

	fmt.Println("\n=== Metrics ===")
	fmt.Printf("  Sync rounds completed : %d\n", atomic.LoadInt64(&metrics.SyncRounds))
	fmt.Printf("  Entries sent          : %d\n", atomic.LoadInt64(&metrics.EntriesSent))
	fmt.Printf("  Entries received      : %d\n", atomic.LoadInt64(&metrics.EntriesRecv))
	fmt.Printf("  Stale updates ignored : %d\n", atomic.LoadInt64(&metrics.StaleIgnored))
}
