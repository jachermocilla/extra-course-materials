// Gossip-Based Rumor Spreading on a Random Graph G(n, p) in Go
// Models the SIR epidemic: Susceptible → Infective → Removed
// Each infective node picks k random neighbors to spread to (fanout),
// then transitions to Removed with probability stopProb.

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
// State — SIR model
// ---------------------------------------------------------------------------

type State int

const (
	Susceptible State = iota // never heard the rumor
	Infective                // has it, actively spreading
	Removed                  // has it, stopped spreading
)

func (s State) String() string {
	switch s {
	case Susceptible:
		return "S"
	case Infective:
		return "I"
	case Removed:
		return "R"
	}
	return "?"
}

// ---------------------------------------------------------------------------
// Message
// ---------------------------------------------------------------------------

type Message struct {
	ID       string    `json:"id"`
	Source   string    `json:"source"`
	Payload  string    `json:"payload"`
	SentAt   time.Time `json:"sent_at"`
	HopCount int       `json:"hop_count"`
	From     string    `json:"from"`
}

// ---------------------------------------------------------------------------
// Metrics
// ---------------------------------------------------------------------------

type Metrics struct {
	TotalSent    int64
	TotalIgnored int64
	NodesInfected int64
	NodesRemoved  int64
	MaxHops       int64
}

func (m *Metrics) recordHops(h int) {
	for {
		old := atomic.LoadInt64(&m.MaxHops)
		if int64(h) <= old {
			break
		}
		if atomic.CompareAndSwapInt64(&m.MaxHops, old, int64(h)) {
			break
		}
	}
}

// ---------------------------------------------------------------------------
// Node
// ---------------------------------------------------------------------------

type Node struct {
	ID        string
	Addr      string
	Neighbors []string

	state    State
	stateMu  sync.Mutex

	listener net.Listener

	fanout   int     // k: how many random neighbors to pick
	stopProb float64 // probability of transitioning I → R after spreading
	rng      *rand.Rand

	metrics *Metrics
}

func NewNode(id, addr string, fanout int, stopProb float64, metrics *Metrics) *Node {
	return &Node{
		ID:       id,
		Addr:     addr,
		state:    Susceptible,
		fanout:   fanout,
		stopProb: stopProb,
		rng:      rand.New(rand.NewSource(time.Now().UnixNano())),
		metrics:  metrics,
	}
}

func (n *Node) AddNeighbor(addr string) {
	n.Neighbors = append(n.Neighbors, addr)
}

func (n *Node) GetState() State {
	n.stateMu.Lock()
	defer n.stateMu.Unlock()
	return n.state
}

func (n *Node) setState(s State) {
	n.stateMu.Lock()
	defer n.stateMu.Unlock()
	n.state = s
}

// Start binds the TCP listener
func (n *Node) Start() error {
	ln, err := net.Listen("tcp", n.Addr)
	if err != nil {
		return fmt.Errorf("node %s listen error: %w", n.ID, err)
	}
	n.listener = ln
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go n.handleConn(conn)
		}
	}()
	return nil
}

func (n *Node) Stop() {
	if n.listener != nil {
		n.listener.Close()
	}
}

// handleConn — state machine on receipt
func (n *Node) handleConn(conn net.Conn) {
	defer conn.Close()

	var msg Message
	if err := json.NewDecoder(bufio.NewReader(conn)).Decode(&msg); err != nil {
		log.Printf("[%s] decode error: %v", n.ID, err)
		return
	}

	n.stateMu.Lock()
	current := n.state
	if current == Susceptible {
		n.state = Infective // S → I
	}
	n.stateMu.Unlock()

	switch current {
	case Susceptible:
		// Newly infected
		atomic.AddInt64(&n.metrics.NodesInfected, 1)
		n.metrics.recordHops(msg.HopCount)
		log.Printf("[%s] S→I  id=%-24s hops=%d  payload=%q",
			n.ID, msg.ID, msg.HopCount, msg.Payload)
		n.spread(msg)

	case Infective:
		// Already spreading — ignore but note it
		atomic.AddInt64(&n.metrics.TotalIgnored, 1)
		log.Printf("[%s] [I]  id=%-24s ignored (already infective)", n.ID, msg.ID)

	case Removed:
		// Stopped spreading — drop
		atomic.AddInt64(&n.metrics.TotalIgnored, 1)
		log.Printf("[%s] [R]  id=%-24s ignored (removed)", n.ID, msg.ID)
	}
}

// spread picks k random neighbors, sends to them, then maybe stops
func (n *Node) spread(msg Message) {
	msg.From = n.Addr
	msg.HopCount++

	// Build candidate list (exclude sender)
	candidates := make([]string, 0, len(n.Neighbors))
	for _, addr := range n.Neighbors {
		if addr != msg.From {
			candidates = append(candidates, addr)
		}
	}

	// Shuffle and pick up to fanout neighbors
	n.rng.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})
	k := n.fanout
	if k > len(candidates) {
		k = len(candidates)
	}
	targets := candidates[:k]

	if len(targets) == 0 {
		log.Printf("[%s] no targets to spread to — going Removed", n.ID)
		n.transitionToRemoved()
		return
	}

	log.Printf("[%s] spreading to %d/%d neighbors: %v",
		n.ID, len(targets), len(candidates), targets)

	var wg sync.WaitGroup
	for _, addr := range targets {
		wg.Add(1)
		go func(a string) {
			defer wg.Done()
			if err := sendMsg(a, msg); err != nil {
				log.Printf("[%s] ✘ SEND FAIL → %s: %v", n.ID, a, err)
				return
			}
			atomic.AddInt64(&n.metrics.TotalSent, 1)
			log.Printf("[%s] → GOSSIP    id=%-24s → %s", n.ID, msg.ID, a)
		}(addr)
	}
	wg.Wait()

	// Probabilistically transition I → R
	if n.rng.Float64() < n.stopProb {
		n.transitionToRemoved()
	} else {
		log.Printf("[%s] still Infective (stopProb=%.2f)", n.ID, n.stopProb)
	}
}

func (n *Node) transitionToRemoved() {
	n.setState(Removed)
	atomic.AddInt64(&n.metrics.NodesRemoved, 1)
	log.Printf("[%s] I→R  (stopped spreading)", n.ID)
}

// Gossip originates a rumor from this source node
func (n *Node) Gossip(payload string) {
	msg := Message{
		ID:       fmt.Sprintf("rumor-%d", time.Now().UnixNano()),
		Source:   n.ID,
		Payload:  payload,
		SentAt:   time.Now(),
		HopCount: 0,
		From:     n.Addr,
	}

	n.setState(Infective)
	atomic.AddInt64(&n.metrics.NodesInfected, 1)
	log.Printf("[%s] ★ RUMOR     id=%-24s payload=%q", n.ID, msg.ID, payload)
	n.spread(msg)
}

// ---------------------------------------------------------------------------
// Transport
// ---------------------------------------------------------------------------

func sendMsg(addr string, msg Message) error {
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()
	return json.NewEncoder(conn).Encode(msg)
}

// ---------------------------------------------------------------------------
// Random Graph G(n, p)
// ---------------------------------------------------------------------------

func buildRandomGraph(n int, p float64, basePort int, fanout int, stopProb float64, metrics *Metrics) []*Node {
	nodes := make([]*Node, n)
	for i := 0; i < n; i++ {
		addr := fmt.Sprintf(":%d", basePort+i)
		nodes[i] = NewNode(fmt.Sprintf("Node%02d", i), addr, fanout, stopProb, metrics)
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

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

func main() {
	const (
		n        = 10
		p        = 0.40
		basePort = 9200
		fanout   = 2    // k: gossip to 2 random neighbors
		stopProb = 0.25 // 25% chance of stopping after each spread
	)

	fmt.Printf("Gossip parameters: fanout k=%d, stopProb=%.2f\n\n", fanout, stopProb)

	metrics := &Metrics{}
	nodes := buildRandomGraph(n, p, basePort, fanout, stopProb, metrics)

	for _, nd := range nodes {
		if err := nd.Start(); err != nil {
			log.Fatalf("start error: %v", err)
		}
	}
	defer func() {
		for _, nd := range nodes {
			nd.Stop()
		}
	}()

	time.Sleep(100 * time.Millisecond)

	fmt.Println("=== Rumor starts at Node00 ===")
	nodes[0].Gossip("Have you heard? Gossip works!")
	time.Sleep(500 * time.Millisecond)

	// Final state of each node
	fmt.Println("\n=== Node States ===")
	for _, nd := range nodes {
		fmt.Printf("  %s  state=%s\n", nd.ID, nd.GetState())
	}

	fmt.Println("\n=== Metrics ===")
	infected := atomic.LoadInt64(&metrics.NodesInfected)
	fmt.Printf("  Nodes in graph     : %d\n", n)
	fmt.Printf("  Nodes infected     : %d / %d\n", infected, n)
	fmt.Printf("  Nodes removed      : %d\n", atomic.LoadInt64(&metrics.NodesRemoved))
	fmt.Printf("  Total msgs sent    : %d\n", atomic.LoadInt64(&metrics.TotalSent))
	fmt.Printf("  Ignored (I or R)   : %d\n", atomic.LoadInt64(&metrics.TotalIgnored))
	fmt.Printf("  Max hop count      : %d\n", atomic.LoadInt64(&metrics.MaxHops))

	// Compare efficiency vs flooding
	fmt.Printf("\n  [vs flooding] gossip sent %d msgs vs flooding's ~%d (n×avg_degree)\n",
		atomic.LoadInt64(&metrics.TotalSent), n*4)
}
