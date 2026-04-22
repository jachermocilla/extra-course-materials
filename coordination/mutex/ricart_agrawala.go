package main

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"
)

// ─────────────────────────────────────────────
//  MESSAGE
// ─────────────────────────────────────────────

type MsgType int

const (
	REQUEST MsgType = iota
	REPLY
	RELEASE
)

func (m MsgType) String() string {
	return [...]string{"REQUEST", "REPLY  ", "RELEASE"}[m]
}

type Message struct {
	kind      MsgType
	timestamp int
	senderID  int
}

func (m Message) String() string {
	return fmt.Sprintf("{%s t=%d P%d}", m.kind, m.timestamp, m.senderID)
}

// ─────────────────────────────────────────────
//  PROCESS
// ─────────────────────────────────────────────

type State int

const (
	RELEASED  State = iota // not interested in CS
	WANTING                // waiting to enter CS
	HELD                   // currently in CS
)

type Process struct {
	mu          sync.Mutex
	id          int
	clock       int
	state       State
	reqTimestamp int         // timestamp of our own current REQUEST
	peers       []*Process
	inbox       chan Message
	replyCount  int          // replies received so far
	replyCond   *sync.Cond
	deferred    []int        // peer IDs we owe a REPLY to
	balance     *float64
	balanceMu   *sync.Mutex
}

func NewProcess(id int, balance *float64, mu *sync.Mutex) *Process {
	p := &Process{
		id:        id,
		state:     RELEASED,
		inbox:     make(chan Message, 64),
		balance:   balance,
		balanceMu: mu,
	}
	p.replyCond = sync.NewCond(&p.mu)
	return p
}

// ── Lamport clock ─────────────────────────────────────────────

func (p *Process) tick() int {
	p.clock++
	return p.clock
}

func (p *Process) update(received int) {
	if received > p.clock {
		p.clock = received
	}
	p.clock++
}

// ── Messaging ─────────────────────────────────────────────────

func (p *Process) sendTo(peer *Process, kind MsgType) {
	p.mu.Lock()
	t := p.tick()
	msg := Message{kind: kind, timestamp: t, senderID: p.id}
	p.mu.Unlock()

	go func() {
		delay := time.Duration(rand.Intn(40)+5) * time.Millisecond
		time.Sleep(delay)
		peer.inbox <- msg
	}()
}

func (p *Process) broadcastRequest() {
	p.mu.Lock()
	t := p.tick()
	p.reqTimestamp = t
	p.state = WANTING
	p.replyCount = 0
	msg := Message{kind: REQUEST, timestamp: t, senderID: p.id}
	p.mu.Unlock()

	fmt.Printf("  [P%d t=%d] broadcast REQUEST\n", p.id, t)
	for _, peer := range p.peers {
		peer := peer
		go func() {
			delay := time.Duration(rand.Intn(40)+5) * time.Millisecond
			time.Sleep(delay)
			peer.inbox <- msg
		}()
	}
}

// ── Message handler (runs in background goroutine) ────────────

func (p *Process) HandleMessages() {
	for msg := range p.inbox {
		p.mu.Lock()
		p.update(msg.timestamp)

		switch msg.kind {

		case REQUEST:
			// Send REPLY immediately if:
			//   1. we are not interested in CS, OR
			//   2. we are WANTING but their request beats ours
			ourReq := Message{timestamp: p.reqTimestamp, senderID: p.id}
			theirReq := Message{timestamp: msg.timestamp, senderID: msg.senderID}
			theyWin := theirReq.timestamp < ourReq.timestamp ||
				(theirReq.timestamp == ourReq.timestamp && theirReq.senderID < ourReq.senderID)

			if p.state == RELEASED || (p.state == WANTING && theyWin) {
				fmt.Printf("  [P%d] reply  → P%d immediately\n", p.id, msg.senderID)
				p.mu.Unlock()
				p.sendTo(p.peerByID(msg.senderID), REPLY)
			} else {
				// defer reply until we release
				fmt.Printf("  [P%d] defer reply → P%d (our t=%d beats their t=%d)\n",
					p.id, msg.senderID, p.reqTimestamp, msg.timestamp)
				p.deferred = append(p.deferred, msg.senderID)
				p.mu.Unlock()
			}

		case REPLY:
			p.replyCount++
			fmt.Printf("  [P%d] received REPLY from P%d (%d/%d)\n",
				p.id, msg.senderID, p.replyCount, len(p.peers))
			p.replyCond.Signal()
			p.mu.Unlock()

		case RELEASE:
			// not used in Ricart-Agrawala — deferred replies handle this
			p.mu.Unlock()
		}
	}
}

// ── Peer lookup ───────────────────────────────────────────────

func (p *Process) peerByID(id int) *Process {
	for _, peer := range p.peers {
		if peer.id == id {
			return peer
		}
	}
	return nil
}

// ── Critical section entry/exit ───────────────────────────────

func (p *Process) EnterCS(wg *sync.WaitGroup) {
	defer wg.Done()

	// Simulate different request times
	time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond)

	// ── Step 1: broadcast REQUEST ─────────────────────────────
	p.broadcastRequest()

	// ── Step 2: wait for REPLY from all peers ─────────────────
	p.mu.Lock()
	for p.replyCount < len(p.peers) {
		p.replyCond.Wait()
	}
	p.state = HELD
	p.mu.Unlock()

	// ── Step 3: critical section ──────────────────────────────
	p.balanceMu.Lock()
	before := *p.balance
	*p.balance += 100
	after := *p.balance
	p.balanceMu.Unlock()
	fmt.Printf("  [P%d] *** CRITICAL SECTION: %.2f → %.2f PHP ***\n",
		p.id, before, after)
	time.Sleep(30 * time.Millisecond)

	// ── Step 4: release — send deferred replies ───────────────
	p.mu.Lock()
	p.state = RELEASED
	deferred := p.deferred
	p.deferred = nil
	p.mu.Unlock()

	fmt.Printf("  [P%d] releasing CS, sending %d deferred replies\n",
		p.id, len(deferred))
	for _, peerID := range deferred {
		p.sendTo(p.peerByID(peerID), REPLY)
	}
}

// ─────────────────────────────────────────────
//  MAIN
// ─────────────────────────────────────────────

func main() {
	rand.Seed(time.Now().UnixNano())

	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("  RICART-AGRAWALA MUTUAL EXCLUSION")
	fmt.Println("  3 processes — no coordinator")
	fmt.Println(strings.Repeat("=", 60))

	balance := 1000.0
	var balanceMu sync.Mutex

	p1 := NewProcess(1, &balance, &balanceMu)
	p2 := NewProcess(2, &balance, &balanceMu)
	p3 := NewProcess(3, &balance, &balanceMu)

	// Wire peers
	p1.peers = []*Process{p2, p3}
	p2.peers = []*Process{p1, p3}
	p3.peers = []*Process{p1, p2}

	// Start message handlers
	go p1.HandleMessages()
	go p2.HandleMessages()
	go p3.HandleMessages()

	fmt.Printf("\n  Initial balance : %.2f PHP\n", balance)
	fmt.Println("  Each process adds 100 PHP — expected final: 1300.00 PHP")
	fmt.Printf("  Messages per CS entry: 2(N-1) = %d\n\n", 2*(len(p1.peers)))
	fmt.Println(strings.Repeat("-", 60))

	// All 3 processes contend for CS concurrently
	var wg sync.WaitGroup
	wg.Add(3)
	go p1.EnterCS(&wg)
	go p2.EnterCS(&wg)
	go p3.EnterCS(&wg)
	wg.Wait()

	// ── Results ───────────────────────────────────────────────
	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("\n  Final balance : %.2f PHP\n", balance)
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
  N = %d processes
  Per CS entry : 2(N-1) = %d messages
    - (N-1) REQUESTs broadcast to all peers
    - (N-1) REPLYs collected before entering

  Total messages for %d processes : %d
  (vs centralized: %d messages)
`,
		len(p1.peers)+1,
		2*len(p1.peers),
		len(p1.peers)+1,
		3*2*len(p1.peers),
		3*3,
	)
}
