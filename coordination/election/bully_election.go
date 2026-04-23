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
	ELECTION MsgType = iota
	OK
	COORDINATOR
)

func (m MsgType) String() string {
	return [...]string{"ELECTION    ", "OK          ", "COORDINATOR "}[m]
}

type Message struct {
	kind     MsgType
	senderID int
}

func (m Message) String() string {
	return fmt.Sprintf("{%s from P%d}", m.kind, m.senderID)
}

// ─────────────────────────────────────────────
//  PROCESS
// ─────────────────────────────────────────────

type Process struct {
	mu          sync.Mutex
	id          int
	alive       bool
	leaderID    int
	inbox       chan Message
	peers       map[int]*Process
	electing    bool      // currently running an election
	gotOK       bool      // received at least one OK during election
}

func NewProcess(id int) *Process {
	return &Process{
		id:    id,
		alive: true,
		inbox: make(chan Message, 64),
		peers: make(map[int]*Process),
	}
}

// ── Messaging ─────────────────────────────────────────────────

func (p *Process) sendTo(peerID int, kind MsgType) {
	p.mu.Lock()
	peer, ok := p.peers[peerID]
	p.mu.Unlock()
	if !ok {
		return
	}

	peer.mu.Lock()
	alive := peer.alive
	peer.mu.Unlock()
	if !alive {
		return // simulate crashed process — message dropped
	}

	go func() {
		delay := time.Duration(rand.Intn(20)+5) * time.Millisecond
		time.Sleep(delay)
		peer.inbox <- Message{kind: kind, senderID: p.id}
	}()
}

// ── Message handler ───────────────────────────────────────────

func (p *Process) HandleMessages() {
	for msg := range p.inbox {
		p.mu.Lock()
		alive := p.alive
		p.mu.Unlock()
		if !alive {
			continue // crashed — ignore all messages
		}

		switch msg.kind {

		case ELECTION:
			// Someone with a lower ID sent us an ELECTION.
			// Reply OK (you lose) and start our own election if not already.
			fmt.Printf("  [P%d] received ELECTION from P%d → send OK\n",
				p.id, msg.senderID)
			p.sendTo(msg.senderID, OK)

			p.mu.Lock()
			already := p.electing
			p.mu.Unlock()
			if !already {
				go p.StartElection()
			}

		case OK:
			// A higher-ID process is alive — we won't be leader.
			fmt.Printf("  [P%d] received OK from P%d → stepping back\n",
				p.id, msg.senderID)
			p.mu.Lock()
			p.gotOK = true
			p.mu.Unlock()

		case COORDINATOR:
			// New leader announced.
			fmt.Printf("  [P%d] received COORDINATOR from P%d → new leader = P%d\n",
				p.id, msg.senderID, msg.senderID)
			p.mu.Lock()
			p.leaderID = msg.senderID
			p.electing = false
			p.mu.Unlock()
		}
	}
}

// ── Election ──────────────────────────────────────────────────
//
// Bully algorithm rules:
//  1. Send ELECTION to all processes with a higher ID
//  2. Wait for OK responses (timeout = 150ms)
//  3. If no OK received → we are the highest alive → broadcast COORDINATOR
//  4. If OK received    → a higher process took over, wait for COORDINATOR

func (p *Process) StartElection() {
	p.mu.Lock()
	if p.electing {
		p.mu.Unlock()
		return
	}
	p.electing = true
	p.gotOK = false
	p.mu.Unlock()

	fmt.Printf("  [P%d] starting election\n", p.id)

	// Step 1: send ELECTION to all higher-ID processes
	sentAny := false
	p.mu.Lock()
	peers := p.peers
	p.mu.Unlock()

	for id := range peers {
		if id > p.id {
			fmt.Printf("  [P%d] send ELECTION → P%d\n", p.id, id)
			p.sendTo(id, ELECTION)
			sentAny = true
		}
	}

	if !sentAny {
		// No higher process exists — we are the highest, declare victory
		p.declareVictory()
		return
	}

	// Step 2: wait for OK responses (timeout)
	time.Sleep(150 * time.Millisecond)

	p.mu.Lock()
	gotOK := p.gotOK
	p.mu.Unlock()

	// Step 3: if no OK → we win
	if !gotOK {
		p.declareVictory()
	}
	// Step 4: if got OK → higher process took over, wait for COORDINATOR
}

func (p *Process) declareVictory() {
	p.mu.Lock()
	p.leaderID = p.id
	p.electing = false
	p.mu.Unlock()

	fmt.Printf("  [P%d] *** I AM THE LEADER ***\n", p.id)

	// broadcast COORDINATOR to all peers
	p.mu.Lock()
	peers := p.peers
	p.mu.Unlock()

	for id := range peers {
		p.sendTo(id, COORDINATOR)
	}
}

// ── Crash / Recover ───────────────────────────────────────────

func (p *Process) Crash() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.alive = false
	fmt.Printf("  [P%d] *** CRASHED ***\n", p.id)
}

func (p *Process) Recover() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.alive = true
	p.leaderID = -1
	fmt.Printf("  [P%d] *** RECOVERED — starting election ***\n", p.id)
}

// ─────────────────────────────────────────────
//  MAIN
// ─────────────────────────────────────────────

func main() {
	rand.Seed(time.Now().UnixNano())

	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("  BULLY ALGORITHM — LEADER ELECTION")
	fmt.Println(strings.Repeat("=", 60))

	// Create 5 processes P1..P5
	const n = 5
	procs := make(map[int]*Process)
	for i := 1; i <= n; i++ {
		procs[i] = NewProcess(i)
	}

	// Wire all peers
	for _, p := range procs {
		for id, peer := range procs {
			if id != p.id {
				p.peers[id] = peer
			}
		}
	}

	// Start message handlers
	for _, p := range procs {
		go p.HandleMessages()
	}

	// ── Scenario 1: initial election triggered by P1 ──────────
	fmt.Println("\n  Scenario 1: P1 detects no leader, starts election")
	fmt.Println(strings.Repeat("-", 60))

	go procs[1].StartElection()
	time.Sleep(500 * time.Millisecond)

	fmt.Println()
	for i := 1; i <= n; i++ {
		procs[i].mu.Lock()
		fmt.Printf("  P%d thinks leader = P%d\n", i, procs[i].leaderID)
		procs[i].mu.Unlock()
	}

	// ── Scenario 2: leader crashes, lower process detects it ──
	fmt.Println()
	fmt.Println(strings.Repeat("-", 60))
	fmt.Println("  Scenario 2: P5 (leader) crashes, P2 detects it")
	fmt.Println(strings.Repeat("-", 60))

	procs[5].Crash()
	time.Sleep(50 * time.Millisecond)

	// P2 detects the leader is gone and starts a new election
	go procs[2].StartElection()
	time.Sleep(500 * time.Millisecond)

	fmt.Println()
	for i := 1; i <= n; i++ {
		procs[i].mu.Lock()
		alive := procs[i].alive
		leader := procs[i].leaderID
		procs[i].mu.Unlock()
		status := "alive"
		if !alive {
			status = "CRASHED"
		}
		fmt.Printf("  P%d [%s] thinks leader = P%d\n", i, status, leader)
	}

	// ── Scenario 3: crashed leader recovers and bullies back ──
	fmt.Println()
	fmt.Println(strings.Repeat("-", 60))
	fmt.Println("  Scenario 3: P5 recovers and bullies back to leadership")
	fmt.Println(strings.Repeat("-", 60))

	procs[5].Recover()
	go procs[5].StartElection()
	time.Sleep(500 * time.Millisecond)

	fmt.Println()
	for i := 1; i <= n; i++ {
		procs[i].mu.Lock()
		alive := procs[i].alive
		leader := procs[i].leaderID
		procs[i].mu.Unlock()
		status := "alive"
		if !alive {
			status = "CRASHED"
		}
		fmt.Printf("  P%d [%s] thinks leader = P%d\n", i, status, leader)
	}

	fmt.Println()
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("  SUMMARY")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println(`
  Bully algorithm rules:

    1. Send ELECTION to all higher-ID processes
    2. Wait for OK (timeout ~150ms)
    3. No OK received  → declare self COORDINATOR
       OK received     → step back, wait for COORDINATOR

  Key property: the process with the HIGHEST ID always wins.
  When a higher process recovers it immediately "bullies" the
  current leader out of its role by starting a new election.`)
}
