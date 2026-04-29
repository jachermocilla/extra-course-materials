// Application-Level Tree-Based Multicasting in Go
// Simulates an overlay multicast tree where each node forwards
// messages to its children over TCP connections.

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

// Message represents a multicast packet traveling the tree
type Message struct {
	ID        string    `json:"id"`
	Source    string    `json:"source"`
	Payload   string    `json:"payload"`
	Timestamp time.Time `json:"timestamp"`
	HopCount  int       `json:"hop_count"`
}

// Node represents an overlay network participant
type Node struct {
	ID       string
	Addr     string
	Parent   string // parent address (empty for root/source)
	Children []string

	listener net.Listener
	mu       sync.Mutex
	seen     map[string]bool // deduplication: message IDs already forwarded
}

// NewNode creates a new overlay node
func NewNode(id, addr, parent string) *Node {
	return &Node{
		ID:     id,
		Addr:   addr,
		Parent: parent,
		seen:   make(map[string]bool),
	}
}

// AddChild registers a child node address
func (n *Node) AddChild(addr string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.Children = append(n.Children, addr)
}

// Start begins listening for incoming messages
func (n *Node) Start() error {
	ln, err := net.Listen("tcp", n.Addr)
	if err != nil {
		return fmt.Errorf("node %s failed to listen: %w", n.ID, err)
	}
	n.listener = ln
	log.Printf("[%s] Listening on %s (parent: %q)", n.ID, n.Addr, n.Parent)

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

// handleConn reads a message from an upstream connection and forwards it
func (n *Node) handleConn(conn net.Conn) {
	defer conn.Close()
	decoder := json.NewDecoder(bufio.NewReader(conn))

	var msg Message
	if err := decoder.Decode(&msg); err != nil {
		log.Printf("[%s] Decode error: %v", n.ID, err)
		return
	}

	n.mu.Lock()
	if n.seen[msg.ID] {
		n.mu.Unlock()
		log.Printf("[%s] Duplicate message %s — dropped", n.ID, msg.ID)
		return
	}
	n.seen[msg.ID] = true
	n.mu.Unlock()

	msg.HopCount++
	log.Printf("[%s] ✔ Received  | id=%-8s payload=%-20q hops=%d",
		n.ID, msg.ID, msg.Payload, msg.HopCount)

	// Forward to all children (fan-out)
	n.forward(msg)
}

// forward sends the message to every child in the overlay tree
func (n *Node) forward(msg Message) {
	n.mu.Lock()
	children := make([]string, len(n.Children))
	copy(children, n.Children)
	n.mu.Unlock()

	if len(children) == 0 {
		log.Printf("[%s] ↳ Leaf node — no forwarding", n.ID)
		return
	}

	var wg sync.WaitGroup
	for _, child := range children {
		wg.Add(1)
		go func(addr string) {
			defer wg.Done()
			if err := sendMessage(addr, msg); err != nil {
				log.Printf("[%s] ✘ Forward to %s failed: %v", n.ID, addr, err)
			} else {
				log.Printf("[%s] → Forwarded to %s | id=%s", n.ID, addr, msg.ID)
			}
		}(child)
	}
	wg.Wait()
}

// Multicast sends an originating message into the tree (source node only)
func (n *Node) Multicast(payload string) {
	msg := Message{
		ID:        fmt.Sprintf("msg-%d", time.Now().UnixNano()),
		Source:    n.ID,
		Payload:   payload,
		Timestamp: time.Now(),
		HopCount:  0,
	}

	n.mu.Lock()
	n.seen[msg.ID] = true
	n.mu.Unlock()

	log.Printf("[%s] ★ Originating multicast | id=%s payload=%q", n.ID, msg.ID, payload)
	n.forward(msg)
}

// sendMessage dials a node and sends a JSON-encoded message
func sendMessage(addr string, msg Message) error {
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()
	return json.NewEncoder(conn).Encode(msg)
}

// Stop shuts down the node's listener
func (n *Node) Stop() {
	if n.listener != nil {
		n.listener.Close()
	}
}

// ---------------------------------------------------------------------------
// Demo: build a 3-level tree and send two multicast messages
//
//         Source (root)
//           :9000
//          /       \
//      NodeA       NodeB
//      :9001       :9002
//      /   \
//  NodeC  NodeD
//  :9003  :9004
// ---------------------------------------------------------------------------

func main() {
	source := NewNode("Source", ":9000", "")
	nodeA := NewNode("NodeA", ":9001", ":9000")
	nodeB := NewNode("NodeB", ":9002", ":9000")
	nodeC := NewNode("NodeC", ":9003", ":9001")
	nodeD := NewNode("NodeD", ":9004", ":9001")

	// Wire up the tree
	source.AddChild(":9001")
	source.AddChild(":9002")
	nodeA.AddChild(":9003")
	nodeA.AddChild(":9004")

	nodes := []*Node{source, nodeA, nodeB, nodeC, nodeD}

	// Start all listeners
	for _, n := range nodes {
		if err := n.Start(); err != nil {
			log.Fatalf("Failed to start node: %v", err)
		}
	}
	defer func() {
		for _, n := range nodes {
			n.Stop()
		}
	}()

	// Brief pause to let listeners bind
	time.Sleep(100 * time.Millisecond)

	fmt.Println("\n=== Multicast #1 ===")
	source.Multicast("Hello, overlay network!")
	time.Sleep(200 * time.Millisecond)

	fmt.Println("\n=== Multicast #2 ===")
	source.Multicast("Second message — tree is alive")
	time.Sleep(200 * time.Millisecond)

	fmt.Println("\n=== Done ===")
}
