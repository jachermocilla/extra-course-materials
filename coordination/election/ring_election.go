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
	ELECTION    MsgType = iota
	COORDINATOR
)

func (m MsgType) String() string {
	return [...]string{"ELECTION   ", "COORDINATOR"}[m]
}

type Message struct {
	kind        MsgType
	candidateID int   // highest ID seen so far as message travels the ring
	senderID    int   // who forwarded this message
}

func (m Message) String() string {
	return fmt.Sprintf("{%s candidate=P%d from=P%d}",
		m.kind, m.candidateID, m.senderID)
}

// ─────────────────────────────────────────────
//  PROCESS
// ─────────────────────────────────────────────

type State int

const (
	PASSIVE  State = iota // not participating
	ELECTING              // election in progress
	LEADER                // this process is the leader
	FOLLOWER              // knows who the leader is
)

func (s State) String() string {
	return [...]string{"PASSIVE ", "ELECTING", "LEADER  ", "FOLLOWER"}[s]
}

type Process struct {
	mu       sync.Mutex
	id       int
	alive    bool
	state    State
	leaderID int
	next     *Process  // next process in the ring
	inbox    chan Message
}

func NewProcess(id int) *Process {
	return &Process{
		id:       id,
		alive:    true,
		state:    PASSIVE,
		leaderID: -1,
		inbox:    make(chan Message, 64),
	}
}

// ── send to next alive process in the ring ────────────────────
// skips over crashed nodes by following next pointers

func (p *Process) forwardToNext(msg Message) {
	target := p.next
	for target != nil {
		target.mu.Lock()
		alive := target.alive
		id := target.id
		target.mu.Unlock()

		if alive {
			t := target
			go func() {
				delay := time.Duration(rand.Intn(20)+5) * time.Millisecond
				time.Sleep(delay)
				fmt.Printf("  [P%d] forward %s → P%d\n",
					p.id, msg, id)
				t.inbox <- msg
			}()
			return
		}
		// skip crashed node
		fmt.Printf("  [P%d] P%d is crashed — skipping in ring\n",
			p.id, id)
		target = target.next
		if target == p {
			// full circle — no one else alive
			return
		}
	}
}

// ── message handler ───────────────────────────────────────────

func (p *Process) HandleMessages() {
	for msg := range p.inbox {
		p.mu.Lock()
		alive := p.alive
		p.mu.Unlock()

		if !alive {
			continue
		}

		switch msg.kind {

		case ELECTION:
			p.handleElection(msg)

		case COORDINATOR:
			p.handleCoordinator(msg)
		}
	}
}

// ── Chang-Roberts ring election algorithm ─────────────────────
//
// Rules on receiving ELECTION{candidateID}:
//
//   candidateID > our ID  → forward the message (they may win)
//   candidateID < our ID  → replace with our own ID and forward
//   candidateID == our ID → the message went around the full ring
//                           we are the winner — broadcast COORDINATOR

func (p *Process) handleElection(msg Message) {
	p.mu.Lock()
	myID := p.id
	p.mu.Unlock()

	fmt.Printf("  [P%d] received  %s\n", myID, msg)

	if msg.candidateID > myID {
		// higher candidate — forward unchanged
		p.mu.Lock()
		p.state = ELECTING
		p.mu.Unlock()
		p.forwardToNext(Message{
			kind:        ELECTION,
			candidateID: msg.candidateID,
			senderID:    myID,
		})

	} else if msg.candidateID < myID {
		// we have a higher ID — replace candidate with ourselves
		p.mu.Lock()
		p.state = ELECTING
		p.mu.Unlock()
		fmt.Printf("  [P%d] replacing candidate P%d with self P%d\n",
			myID, msg.candidateID, myID)
		p.forwardToNext(Message{
			kind:        ELECTION,
			candidateID: myID,
			senderID:    myID,
		})

	} else {
		// candidateID == myID — message completed the ring
		// we are the leader
		p.mu.Lock()
		p.state = LEADER
		p.leaderID = myID
		p.mu.Unlock()
		fmt.Printf("  [P%d] *** election message returned — I AM THE LEADER ***\n",
			myID)
		// broadcast COORDINATOR around the ring
		p.forwardToNext(Message{
			kind:        COORDINATOR,
			candidateID: myID,
			senderID:    myID,
		})
	}
}

func (p *Process) handleCoordinator(msg Message) {
	p.mu.Lock()
	myID := p.id
	p.mu.Unlock()

	if msg.candidateID == myID {
		// COORDINATOR message completed the ring — stop propagating
		fmt.Printf("  [P%d] COORDINATOR message returned — all notified\n", myID)
		return
	}

	p.mu.Lock()
	p.leaderID = msg.candidateID
	p.state = FOLLOWER
	p.mu.Unlock()

	fmt.Printf("  [P%d] acknowledged leader = P%d\n", myID, msg.candidateID)
	p.forwardToNext(Message{
		kind:        COORDINATOR,
		candidateID: msg.candidateID,
		senderID:    myID,
	})
}

// ── start an election ─────────────────────────────────────────

func (p *Process) StartElection() {
	p.mu.Lock()
	p.state = ELECTING
	myID := p.id
	p.mu.Unlock()

	fmt.Printf("  [P%d] initiating election\n", myID)
	p.forwardToNext(Message{
		kind:        ELECTION,
		candidateID: myID,
		senderID:    myID,
	})
}

// ── crash / recover ───────────────────────────────────────────

func (p *Process) Crash() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.alive = false
	p.state = PASSIVE
	fmt.Printf("  [P%d] *** CRASHED ***\n", p.id)
}

func (p *Process) Recover() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.alive = true
	p.leaderID = -1
	p.state = PASSIVE
	fmt.Printf("  [P%d] *** RECOVERED ***\n", p.id)
}

// ─────────────────────────────────────────────
//  HELPERS
// ─────────────────────────────────────────────

func printStatus(procs []*Process) {
	fmt.Println()
	for _, p := range procs {
		p.mu.Lock()
		alive := p.alive
		state := p.state
		leader := p.leaderID
		p.mu.Unlock()
		status := "alive  "
		if !alive {
			status = "CRASHED"
		}
		fmt.Printf("  P%d [%s] state=%-8s leader=P%d\n",
			p.id, status, state, leader)
	}
	fmt.Println()
}

func buildRing(procs []*Process) {
	n := len(procs)
	fmt.Println("\n  Ring topology:")
	for i, p := range procs {
		p.next = procs[(i+1)%n]
		fmt.Printf("    P%d → P%d\n", p.id, p.next.id)
	}
}

// ─────────────────────────────────────────────
//  MAIN
// ─────────────────────────────────────────────

func main() {
	rand.Seed(time.Now().UnixNano())

	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("  RING LEADER ELECTION")
	fmt.Println("  Chang-Roberts Algorithm")
	fmt.Println(strings.Repeat("=", 60))

	// Create 6 processes with random IDs to make it interesting
	ids := []int{4, 7, 2, 9, 1, 5} // P1=id4, P2=id7, etc.
	procs := make([]*Process, len(ids))
	for i, id := range ids {
		procs[i] = NewProcess(id)
	}

	buildRing(procs)

	// start message handlers
	for _, p := range procs {
		go p.HandleMessages()
	}

	// ── Scenario 1: P4 initiates election ─────────────────────
	fmt.Println()
	fmt.Println(strings.Repeat("-", 60))
	fmt.Println("  Scenario 1: P4 initiates election")
	fmt.Println(strings.Repeat("-", 60))

	go procs[0].StartElection() // P4 starts
	time.Sleep(600 * time.Millisecond)
	fmt.Println("  Result:")
	printStatus(procs)

	// reset all states
	for _, p := range procs {
		p.mu.Lock()
		p.state = PASSIVE
		p.leaderID = -1
		p.mu.Unlock()
	}

	// ── Scenario 2: multiple initiators simultaneously ─────────
	fmt.Println(strings.Repeat("-", 60))
	fmt.Println("  Scenario 2: P4, P7 and P1 all initiate simultaneously")
	fmt.Println(strings.Repeat("-", 60))

	go procs[0].StartElection() // P4
	go procs[1].StartElection() // P7
	go procs[4].StartElection() // P1
	time.Sleep(800 * time.Millisecond)
	fmt.Println("  Result:")
	printStatus(procs)

	// ── Scenario 3: leader crashes, new election ───────────────
	fmt.Println(strings.Repeat("-", 60))
	fmt.Println("  Scenario 3: leader (P9) crashes, P4 starts new election")
	fmt.Println(strings.Repeat("-", 60))

	// find and crash the leader (P9)
	for _, p := range procs {
		p.mu.Lock()
		id := p.id
		p.mu.Unlock()
		if id == 9 {
			p.Crash()
			break
		}
	}

	time.Sleep(50 * time.Millisecond)
	go procs[0].StartElection() // P4 detects leader is gone
	time.Sleep(800 * time.Millisecond)
	fmt.Println("  Result:")
	printStatus(procs)

	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("  SUMMARY")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println(`
  Chang-Roberts ring election rules:

    On receiving ELECTION{candidateID}:

      candidateID > myID  → forward unchanged     (they may win)
      candidateID < myID  → replace with myID     (I may win)
      candidateID == myID → I win, broadcast COORDINATOR

  Properties:

    Correctness  — the process with the highest ID always wins
    Messages     — O(N²) worst case, O(N) best case
    Fault tolerant — crashed nodes are skipped in the ring
    Multiple initiators — naturally resolved, highest ID survives`)
}
