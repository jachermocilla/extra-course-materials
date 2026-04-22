package main

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"
)

// ─────────────────────────────────────────────
//  MESSAGE TYPES
// ─────────────────────────────────────────────

type MsgType int

const (
	REQUEST MsgType = iota
	GRANT
	RELEASE
)

func (m MsgType) String() string {
	return [...]string{"REQUEST", "GRANT  ", "RELEASE"}[m]
}

type Message struct {
	kind     MsgType
	senderID int
}

func (m Message) String() string {
	return fmt.Sprintf("{%s from P%d}", m.kind, m.senderID)
}

// ─────────────────────────────────────────────
//  COORDINATOR
// ─────────────────────────────────────────────

type Coordinator struct {
	mu       sync.Mutex
	busy     bool
	queue    []int         // queued client IDs
	grants   map[int]chan Message // reply channels per client
	log      []string
}

func NewCoordinator() *Coordinator {
	return &Coordinator{
		grants: make(map[int]chan Message),
	}
}

func (co *Coordinator) register(clientID int, ch chan Message) {
	co.mu.Lock()
	defer co.mu.Unlock()
	co.grants[clientID] = ch
}

func (co *Coordinator) logf(format string, args ...any) {
	entry := fmt.Sprintf(format, args...)
	co.log = append(co.log, entry)
	fmt.Println(entry)
}

// Handle processes incoming messages from clients
func (co *Coordinator) Handle(msg Message) {
	co.mu.Lock()
	defer co.mu.Unlock()

	switch msg.kind {
	case REQUEST:
		co.logf("  [COORD] received REQUEST from P%d  (busy=%v)",
			msg.senderID, co.busy)
		if !co.busy {
			co.busy = true
			co.logf("  [COORD] sending GRANT to P%d", msg.senderID)
			co.grants[msg.senderID] <- Message{kind: GRANT, senderID: 0}
		} else {
			co.logf("  [COORD] CS busy — queuing P%d  queue=%v",
				msg.senderID, append(co.queue, msg.senderID))
			co.queue = append(co.queue, msg.senderID)
		}

	case RELEASE:
		co.logf("  [COORD] received RELEASE from P%d", msg.senderID)
		if len(co.queue) == 0 {
			co.busy = false
			co.logf("  [COORD] CS is now free, queue empty")
		} else {
			next := co.queue[0]
			co.queue = co.queue[1:]
			co.logf("  [COORD] sending GRANT to next in queue: P%d", next)
			co.grants[next] <- Message{kind: GRANT, senderID: 0}
		}
	}
}

// ─────────────────────────────────────────────
//  CLIENT PROCESS
// ─────────────────────────────────────────────

type Client struct {
	id      int
	inbox   chan Message // receives GRANT from coordinator
	coord   chan Message // sends REQUEST/RELEASE to coordinator
	balance *float64
	mu      *sync.Mutex
	wg      *sync.WaitGroup
}

func (c *Client) Run() {
	defer c.wg.Done()

	// Simulate some work before requesting CS
	time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond)

	// ── Step 1: send REQUEST ──────────────────────────────────
	fmt.Printf("  [P%d]    sending  REQUEST to coordinator\n", c.id)
	c.coord <- Message{kind: REQUEST, senderID: c.id}

	// ── Step 2: block waiting for GRANT ──────────────────────
	fmt.Printf("  [P%d]    waiting  for GRANT...\n", c.id)
	<-c.inbox
	fmt.Printf("  [P%d]    received GRANT — entering critical section\n", c.id)

	// ── Step 3: critical section ──────────────────────────────
	c.mu.Lock()
	before := *c.balance
	*c.balance += 100
	after := *c.balance
	c.mu.Unlock()
	fmt.Printf("  [P%d] *** CRITICAL SECTION: %.2f → %.2f PHP ***\n",
		c.id, before, after)
	time.Sleep(30 * time.Millisecond) // simulate work

	// ── Step 4: send RELEASE ──────────────────────────────────
	fmt.Printf("  [P%d]    sending  RELEASE to coordinator\n", c.id)
	c.coord <- Message{kind: RELEASE, senderID: c.id}
}

// ─────────────────────────────────────────────
//  MAIN
// ─────────────────────────────────────────────

func main() {
	rand.Seed(time.Now().UnixNano())

	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("  CENTRALIZED MUTUAL EXCLUSION")
	fmt.Println("  3 clients contend for a shared bank balance")
	fmt.Println(strings.Repeat("=", 60))

	// Shared resource
	balance := 1000.0
	var balanceMu sync.Mutex

	// Coordinator inbox
	coordInbox := make(chan Message, 16)
	coord := NewCoordinator()

	// Create 3 clients
	var wg sync.WaitGroup
	clients := make([]*Client, 3)
	for i := 0; i < 3; i++ {
		inbox := make(chan Message, 1)
		clientID := i + 1
		coord.register(clientID, inbox)
		clients[i] = &Client{
			id:      clientID,
			inbox:   inbox,
			coord:   coordInbox,
			balance: &balance,
			mu:      &balanceMu,
			wg:      &wg,
		}
	}

	// Run coordinator in its own goroutine
	go func() {
		for msg := range coordInbox {
			coord.Handle(msg)
		}
	}()

	fmt.Printf("\n  Initial balance : %.2f PHP\n", balance)
	fmt.Println("  Each client adds 100 PHP — expected final: 1300.00 PHP")
	fmt.Println()
	fmt.Println(strings.Repeat("-", 60))

	// Run all clients concurrently
	wg.Add(3)
	for _, c := range clients {
		go c.Run()
	}
	wg.Wait()
	close(coordInbox)

	// ── Results ───────────────────────────────────────────────
	fmt.Println(strings.Repeat("-", 60))
	fmt.Println()
	fmt.Printf("  Final balance : %.2f PHP\n", balance)
	if balance == 1300.00 {
		fmt.Println("  ✅ Correct — mutual exclusion held, no race condition")
	} else {
		fmt.Println("  ❌ Wrong — race condition detected")
	}

	fmt.Println()
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("  MESSAGE SUMMARY")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf(`
  Per CS entry : 3 messages
    1. Client → Coordinator : REQUEST
    2. Coordinator → Client : GRANT
    3. Client → Coordinator : RELEASE

  Total messages for %d clients : %d
`, len(clients), len(clients)*3)
}
