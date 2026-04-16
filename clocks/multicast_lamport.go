package main

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"time"
)

const initialBalance = 1000.0

// ─────────────────────────────────────────────
//  MESSAGE
// ─────────────────────────────────────────────

type Message struct {
	timestamp int
	senderID  int
	op        string
}

func (m Message) String() string {
	return fmt.Sprintf("{t=%d, P%d, %s}", m.timestamp, m.senderID, m.op)
}

// before returns true if m happens-before other per Lamport ordering
func (m Message) before(other Message) bool {
	if m.timestamp != other.timestamp {
		return m.timestamp < other.timestamp
	}
	return m.senderID < other.senderID // tie-break by sender ID
}

// ─────────────────────────────────────────────
//  PROCESS (sender)
// ─────────────────────────────────────────────

type Process struct {
	mu       sync.Mutex
	id       int
	clock    int
	replicas []*Replica
}

func (p *Process) tick() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.clock++
	return p.clock
}

// Multicast stamps the message and sends to all replicas with random delay
func (p *Process) Multicast(op string) {
	t := p.tick()
	msg := Message{timestamp: t, senderID: p.id, op: op}
	fmt.Printf("  [P%d] multicast %s\n", p.id, msg)
	for _, r := range p.replicas {
		go func(replica *Replica, m Message) {
			delay := time.Duration(rand.Intn(80)+10) * time.Millisecond
			time.Sleep(delay)
			replica.Receive(m)
		}(r, msg)
	}
}

// ─────────────────────────────────────────────
//  REPLICA (receiver)
// ─────────────────────────────────────────────

type Replica struct {
	mu          sync.Mutex
	id          int
	clock       int
	balance     float64
	queue       []Message
	delivered   []Message
	log         []string
	numSenders  int
	deliverCond *sync.Cond
}

func NewReplica(id, numSenders int) *Replica {
	r := &Replica{
		id:         id,
		balance:    initialBalance,
		numSenders: numSenders,
	}
	r.deliverCond = sync.NewCond(&r.mu)
	return r
}

// Lamport receive rule: max(local, received) + 1
func (r *Replica) updateClock(received int) {
	if received > r.clock {
		r.clock = received
	}
	r.clock++
}

// Receive enqueues an incoming message and signals the deliver loop
func (r *Replica) Receive(msg Message) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.updateClock(msg.timestamp)
	r.queue = append(r.queue, msg)
	// keep queue sorted by Lamport order
	sort.Slice(r.queue, func(i, j int) bool {
		return r.queue[i].before(r.queue[j])
	})
	r.deliverCond.Signal()
}

// canDeliver checks if the head of the queue is safe to deliver.
// Safe = we have seen at least one message from every other sender
// with a strictly later Lamport timestamp, so nothing earlier is
// still in flight.
func (r *Replica) canDeliver() bool {
	if len(r.queue) == 0 {
		return false
	}
	head := r.queue[0]
	laterSenders := map[int]bool{}
	for _, m := range r.queue[1:] {
		if !m.before(head) {
			laterSenders[m.senderID] = true
		}
	}
	// need coverage from all senders except head's own sender
	return len(laterSenders) >= r.numSenders-1
}

// applyOp mutates the balance
func (r *Replica) applyOp(op string) {
	before := r.balance
	switch op {
	case "ADD_100":
		r.balance += 100
	case "ADD_1PCT":
		r.balance *= 1.01
	}
	entry := fmt.Sprintf("  [Replica %d] deliver {t=%d, P%d, %-8s}  %.2f → %.2f PHP",
		r.id, r.queue[0].timestamp, r.queue[0].senderID, op, before, r.balance)
	r.log = append(r.log, entry)
}

// DeliverLoop runs in its own goroutine, delivering messages in order
func (r *Replica) DeliverLoop(done chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()
	r.mu.Lock()
	defer r.mu.Unlock()
	for {
		for !r.canDeliver() {
			// check if we are done
			select {
			case <-done:
				// drain any remaining in order (all messages received)
				for len(r.queue) > 0 {
					r.applyOp(r.queue[0].op)
					r.delivered = append(r.delivered, r.queue[0])
					r.queue = r.queue[1:]
				}
				return
			default:
			}
			r.deliverCond.Wait()
		}
		r.applyOp(r.queue[0].op)
		r.delivered = append(r.delivered, r.queue[0])
		r.queue = r.queue[1:]
	}
}

// ─────────────────────────────────────────────
//  MAIN
// ─────────────────────────────────────────────

func main() {
	rand.Seed(time.Now().UnixNano())

	const numSenders = 2

	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("  WITH LAMPORT CLOCK")
	fmt.Println("  Totally ordered multicast across 2 replicas")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("\n  Initial balance: %.2f PHP\n", initialBalance)
	fmt.Println("  P1: ADD_100 PHP")
	fmt.Println("  P2: ADD_1PCT (1%)")
	fmt.Println()

	r1 := NewReplica(1, numSenders)
	r2 := NewReplica(2, numSenders)

	p1 := &Process{id: 1, replicas: []*Replica{r1, r2}}
	p2 := &Process{id: 2, replicas: []*Replica{r1, r2}}

	// Start deliver loops on each replica
	done := make(chan struct{})
	var deliverWg sync.WaitGroup
	deliverWg.Add(2)
	go r1.DeliverLoop(done, &deliverWg)
	go r2.DeliverLoop(done, &deliverWg)

	// Processes multicast concurrently with random send delays
	var sendWg sync.WaitGroup
	sendWg.Add(2)
	go func() {
		defer sendWg.Done()
		time.Sleep(time.Duration(rand.Intn(30)) * time.Millisecond)
		p1.Multicast("ADD_100")
	}()
	go func() {
		defer sendWg.Done()
		time.Sleep(time.Duration(rand.Intn(30)) * time.Millisecond)
		p2.Multicast("ADD_1PCT")
	}()
	sendWg.Wait()

	// Wait for all messages to arrive at replicas then signal done
	time.Sleep(200 * time.Millisecond)
	close(done)
	r1.deliverCond.Signal()
	r2.deliverCond.Signal()
	deliverWg.Wait()

	fmt.Println("\n  Replica 1 delivery log:")
	for _, l := range r1.log {
		fmt.Println(l)
	}
	fmt.Println("  Replica 2 delivery log:")
	for _, l := range r2.log {
		fmt.Println(l)
	}

	fmt.Printf("\n  Replica 1 final: %.2f PHP\n", r1.balance)
	fmt.Printf("  Replica 2 final: %.2f PHP\n", r2.balance)

	if fmt.Sprintf("%.2f", r1.balance) == fmt.Sprintf("%.2f", r2.balance) {
		fmt.Println("\n  ✅ CONSISTENT — both replicas agree!")
	} else {
		fmt.Printf("\n  ❌ INCONSISTENT — diverged by %.2f PHP\n",
			r1.balance-r2.balance)
	}
}
