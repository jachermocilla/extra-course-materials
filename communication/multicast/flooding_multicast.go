// Flooding on a Random Graph G(n, p) in Go
// Mirrors the ALM multicast structure: each node is a TCP listener.
// On receiving a new message, it forwards to all neighbors except sender.

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
// Message
// ---------------------------------------------------------------------------

type Message struct {
	ID       string    `json:"id"`
	Source   string    `json:"source"`
	Payload  string    `json:"payload"`
	SentAt   time.Time `json:"sent_at"`
	HopCount int       `json:"hop_count"`
	From     string    `json:"from"` // immediate sender — excluded from re-forward
}

// ---------------------------------------------------------------------------
// Metrics (atomic counters shared across all nodes)
// ---------------------------------------------------------------------------

type Metrics struct {
	TotalSent     atomic.Int64
	TotalDropped  atomic.Int64
	NodesReached  atomic.Int64
	MaxHops       atomic.Int64
}

func (m *Metrics) recordHops(h int) {
	for {
		old := m.MaxHops.Load()
		if int64(h) <= old {
			break
		}
		if m.MaxHops.CompareAndSwap(old, int64(h)) {
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
	Neighbors []string // neighbor addresses

	listener net.Listener
	mu       sync.Mutex
	seen     map[string]bool

	metrics *Metrics
}

func NewNode(id, addr string, metrics *Metrics) *Node {
	return &Node{
		ID:      id,
		Addr:    addr,
		seen:    make(map[string]bool),
		metrics: metrics,
	}
}

func (n *Node) AddNeighbor(addr string) {
	n.Neighbors = append(n.Neighbors, addr)
}

// Start binds the TCP listener and accepts connections
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

// handleConn — receive, dedup, forward
func (n *Node) handleConn(conn net.Conn) {
	defer conn.Close()

	var msg Message
	if err := json.NewDecoder(bufio.NewReader(conn)).Decode(&msg); err != nil {
		log.Printf("[%s] decode error: %v", n.ID, err)
		return
	}

	n.mu.Lock()
	already := n.seen[msg.ID]
	if !already {
		n.seen[msg.ID] = true
	}
	n.mu.Unlock()

	if already {
		n.metrics.TotalDropped.Add(1)
		log.Printf("[%s] ✘ DROP      id=%-24s (duplicate)", n.ID, msg.ID)
		return
	}

	// First time seeing this message
	n.metrics.NodesReached.Add(1)
	n.metrics.recordHops(msg.HopCount)
	log.Printf("[%s] ✔ RECEIVED  id=%-24s hops=%d  payload=%q",
		n.ID, msg.ID, msg.HopCount, msg.Payload)

	n.forward(msg)
}

// forward fans out to all neighbors except the one that sent it
func (n *Node) forward(msg Message) {
	sender := msg.From
	msg.From = n.Addr
	msg.HopCount++

	n.mu.Lock()
	neighbors := make([]string, len(n.Neighbors))
	copy(neighbors, n.Neighbors)
	n.mu.Unlock()

	var wg sync.WaitGroup
	for _, addr := range neighbors {
		if addr == sender {
			continue // don't send back toward the sender
		}
		wg.Add(1)
		go func(a string) {
			defer wg.Done()
			if err := sendMsg(a, msg); err != nil {
				log.Printf("[%s] ✘ SEND FAIL → %s: %v", n.ID, a, err)
				return
			}
			n.metrics.TotalSent.Add(1)
			log.Printf("[%s] → FORWARD   id=%-24s → %s", n.ID, msg.ID, a)
		}(addr)
	}
	wg.Wait()
}

// Flood originates a new message from this source node
func (n *Node) Flood(payload string) {
	msg := Message{
		ID:       fmt.Sprintf("msg-%d", time.Now().UnixNano()),
		Source:   n.ID,
		Payload:  payload,
		SentAt:   time.Now(),
		HopCount: 0,
		From:     n.Addr,
	}

	n.mu.Lock()
	n.seen[msg.ID] = true
	n.mu.Unlock()

	n.metrics.NodesReached.Add(1) // count the source itself
	log.Printf("[%s] ★ FLOOD     id=%-24s payload=%q", n.ID, msg.ID, payload)
	n.forward(msg)
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

func buildRandomGraph(n int, p float64, basePort int, metrics *Metrics) []*Node {
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

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

func main() {
	const (
		n        = 10
		p        = 0.40
		basePort = 9100
	)

	metrics := &Metrics{}
	nodes := buildRandomGraph(n, p, basePort, metrics)

	// Start all listeners
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

	// --- Flood #1 from Node00 ---
	fmt.Println("=== Flood #1 from Node00 ===")
	nodes[0].Flood("Hello random graph!")
	time.Sleep(300 * time.Millisecond)

	// --- Flood #2 from Node00 ---
	fmt.Println("\n=== Flood #2 from Node00 ===")
	nodes[0].Flood("Second flood message")
	time.Sleep(300 * time.Millisecond)

	// --- Metrics summary ---
	fmt.Println("\n=== Metrics ===")
	fmt.Printf("  Nodes in graph     : %d\n", n)
	fmt.Printf("  Nodes reached      : %d\n", metrics.NodesReached.Load()/2) // 2 floods
	fmt.Printf("  Total msgs sent    : %d\n", metrics.TotalSent.Load())
	fmt.Printf("  Duplicate drops    : %d\n", metrics.TotalDropped.Load())
	fmt.Printf("  Max hop count      : %d\n", metrics.MaxHops.Load())
	fmt.Printf("  Link stress (avg)  : %.2f msgs/edge\n",
		float64(metrics.TotalSent.Load())/float64(n))
}
