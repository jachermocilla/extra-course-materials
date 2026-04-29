package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// ============================================================
// Core Types
// ============================================================

type LogEntry struct {
	ProcessID int
	Key       string
	Value     string
	Timestamp time.Time
	EventType string // "WRITE", "READ", "SYNC"
}

var globalLog []LogEntry
var logMu sync.Mutex

func log(processID int, eventType, key, value, note string) {
	logMu.Lock()
	defer logMu.Unlock()
	entry := LogEntry{
		ProcessID: processID,
		Key:       key,
		Value:     value,
		Timestamp: time.Now(),
		EventType: eventType,
	}
	globalLog = append(globalLog, entry)
	pid := fmt.Sprintf("P%d", processID)
	fmt.Printf("  [%s] %-5s  key=%-6s val=%-10s  %s\n", pid, eventType, key, value, note)
}

func printHeader(title string) {
	fmt.Printf("\n%s\n", title)
	fmt.Printf("%-60s\n", "============================================================")
}

func printDivider() {
	fmt.Println("------------------------------------------------------------")
}

// ============================================================
// Process — represents a node in the distributed store
// ============================================================

type Process struct {
	ID    int
	store map[string]string
	mu    sync.RWMutex

	// For causal consistency: vector clock
	vectorClock []int

	// Pending writes (simulates async replication queue)
	pending []pendingWrite
	pendMu  sync.Mutex
}

type pendingWrite struct {
	key   string
	value string
	from  int
}

func NewProcess(id, totalProcesses int) *Process {
	return &Process{
		ID:          id,
		store:       make(map[string]string),
		vectorClock: make([]int, totalProcesses),
	}
}

func (p *Process) localWrite(key, value string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.store[key] = value
	p.vectorClock[p.ID]++
	log(p.ID, "WRITE", key, value, fmt.Sprintf("clock=%v", p.vectorClock))
}

func (p *Process) localRead(key string) string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	val := p.store[key]
	if val == "" {
		val = "<nil>"
	}
	log(p.ID, "READ", key, val, "")
	return val
}

func (p *Process) forceWrite(key, value string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.store[key] = value
}

func (p *Process) snapshot() map[string]string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	copy := make(map[string]string)
	for k, v := range p.store {
		copy[k] = v
	}
	return copy
}

// ============================================================
// SCENARIO 1: No Consistency — Concurrent Writes, No Sync
// ============================================================

func scenarioNoConsistency() {
	printHeader("SCENARIO 1: No Consistency (Concurrent Writes, No Synchronisation)")
	fmt.Println("  Three processes write different values to the same key concurrently.")
	fmt.Println("  Without sync, each process sees only its own write — stores diverge.\n")

	p1 := NewProcess(0, 3)
	p2 := NewProcess(1, 3)
	p3 := NewProcess(2, 3)

	var wg sync.WaitGroup
	wg.Add(3)

	go func() { defer wg.Done(); p1.localWrite("x", "10") }()
	go func() { defer wg.Done(); p2.localWrite("x", "20") }()
	go func() { defer wg.Done(); p3.localWrite("x", "30") }()

	wg.Wait()

	printDivider()
	fmt.Println("  RESULT — Each process reads its own stale local copy:")
	p1.localRead("x")
	p2.localRead("x")
	p3.localRead("x")
	fmt.Println("\n  ⚠️  Inconsistency: three processes disagree on value of 'x'.")
}

// ============================================================
// SCENARIO 2: Eventual Consistency — Async Propagation
// ============================================================

func propagateAsync(src *Process, peers []*Process, key, value string, delayMs int) {
	go func() {
		time.Sleep(time.Duration(delayMs) * time.Millisecond)
		for _, peer := range peers {
			if peer.ID != src.ID {
				peer.forceWrite(key, value)
				log(peer.ID, "SYNC", key, value,
					fmt.Sprintf("<-- propagated from P%d after %dms", src.ID, delayMs))
			}
		}
	}()
}

func scenarioEventualConsistency() {
	printHeader("SCENARIO 2: Eventual Consistency (Async Replication)")
	fmt.Println("  P0 writes a value. Propagation to peers is delayed.")
	fmt.Println("  Reads immediately after the write show stale data on P1 and P2.")
	fmt.Println("  Eventually all nodes converge.\n")

	p0 := NewProcess(0, 3)
	p1 := NewProcess(1, 3)
	p2 := NewProcess(2, 3)
	all := []*Process{p0, p1, p2}

	// Seed all with an initial value
	for _, p := range all {
		p.forceWrite("user:42", "Alice")
	}

	// P0 updates the record; async propagation with different delays
	p0.localWrite("user:42", "Alice-Updated")
	propagateAsync(p0, all, "user:42", "Alice-Updated", 80)  // P1 gets it in 80ms
	propagateAsync(p0, all, "user:42", "Alice-Updated", 200) // P2 gets it in 200ms

	printDivider()
	fmt.Println("  READ at T+10ms (before propagation):")
	time.Sleep(10 * time.Millisecond)
	p0.localRead("user:42")
	p1.localRead("user:42")
	p2.localRead("user:42")

	fmt.Println("\n  READ at T+100ms (P1 synced, P2 still stale):")
	time.Sleep(90 * time.Millisecond)
	p0.localRead("user:42")
	p1.localRead("user:42")
	p2.localRead("user:42")

	fmt.Println("\n  READ at T+250ms (all synced — eventual convergence):")
	time.Sleep(160 * time.Millisecond)
	p0.localRead("user:42")
	p1.localRead("user:42")
	p2.localRead("user:42")

	fmt.Println("\n  ✅ Converged: all processes now agree on 'user:42'.")
	fmt.Println("  ⚠️  But stale reads occurred during the window — no strong guarantee.")
}

// ============================================================
// SCENARIO 3: Sequential Consistency Violation
// ============================================================

func scenarioSequentialViolation() {
	printHeader("SCENARIO 3: Sequential Consistency Violation")
	fmt.Println("  P0 writes x=1, then x=2 (in that order).")
	fmt.Println("  Due to replication delays, P1 sees x=2 before x=1 — order violated.\n")

	p0 := NewProcess(0, 3)
	p1 := NewProcess(1, 3)

	p0.localWrite("x", "1")
	// Simulate: second write propagates FASTER than first (out-of-order delivery)
	go func() {
		time.Sleep(80 * time.Millisecond)
		p1.forceWrite("x", "2")
		log(p1.ID, "SYNC", "x", "2", "<-- arrived first (fast path)")
	}()

	p0.localWrite("x", "2")

	go func() {
		time.Sleep(150 * time.Millisecond)
		p1.forceWrite("x", "1")
		log(p1.ID, "SYNC", "x", "1", "<-- arrived second (slow path)")
	}()

	printDivider()
	fmt.Println("  P1 reads after each propagation arrives:")
	time.Sleep(100 * time.Millisecond)
	fmt.Print("  P1 sees (after 100ms): ")
	p1.localRead("x")

	time.Sleep(100 * time.Millisecond)
	fmt.Print("  P1 sees (after 200ms): ")
	p1.localRead("x")

	fmt.Println("\n  ⚠️  P1 observed x=2 before x=1 — sequential order of P0's writes not preserved.")
}

// ============================================================
// SCENARIO 4: Causal Consistency
// ============================================================

func scenarioCausalConsistency() {
	printHeader("SCENARIO 4: Causal Consistency")
	fmt.Println("  P0 posts a message. P1 replies (causally depends on P0's post).")
	fmt.Println("  Without causal ordering, P2 might see P1's reply before P0's post.\n")

	// Simulate message board
	type Message struct {
		author  string
		content string
		causeID int // ID of message this replies to (-1 = none)
	}

	type Board struct {
		mu       sync.Mutex
		messages []Message
	}

	board := &Board{}

	printDivider()

	// P0 posts original message
	time.Sleep(10 * time.Millisecond)
	board.mu.Lock()
	board.messages = append(board.messages, Message{"P0", "Anyone know Go channels?", -1})
	board.mu.Unlock()
	fmt.Println("  [P0] WRITE  msg=0  'Anyone know Go channels?'  (no cause)")

	// P1 sees P0's post and replies
	time.Sleep(20 * time.Millisecond)
	board.mu.Lock()
	board.messages = append(board.messages, Message{"P1", "Yes! Use make(chan int)", 0})
	board.mu.Unlock()
	fmt.Println("  [P1] WRITE  msg=1  'Yes! Use make(chan int)'  (caused by msg=0)")

	// P2 receives P1's reply BEFORE P0's post (out-of-order delivery, no causal guard)
	fmt.Println("\n  ⚠️  P2 receives messages out-of-causal-order:")
	fmt.Println("  [P2] READ   msg=1  'Yes! Use make(chan int)'  ← reply with NO visible question!")
	fmt.Println("  [P2] READ   msg=0  'Anyone know Go channels?' ← question arrives late")

	fmt.Println("\n  ✅ With causal consistency enforced: P2 would buffer msg=1 until msg=0 arrives.")
	fmt.Println("     Causal ordering: msg=0 must be visible before msg=1 on every process.")
}

// ============================================================
// SCENARIO 5: Read-Your-Writes Violation
// ============================================================

func scenarioReadYourWritesViolation() {
	printHeader("SCENARIO 5: Read-Your-Writes Violation")
	fmt.Println("  A client writes to P0 (primary), then reads from P2 (replica).")
	fmt.Println("  If the write hasn't propagated yet, the client doesn't see its own write.\n")

	primary := NewProcess(0, 3)
	replica := NewProcess(2, 3)

	// Client writes to primary
	primary.localWrite("profile:name", "Bob")

	// Propagation is slow — not yet on replica
	fmt.Println("  (Replication to replica is still in-flight...)\n")

	// Client immediately reads from a different replica
	fmt.Print("  Client reads from replica: ")
	replica.localRead("profile:name") // returns <nil>

	fmt.Println("\n  ⚠️  Client wrote 'Bob' to primary but read <nil> from replica.")
	fmt.Println("     This violates Read-Your-Writes guarantee.")
	fmt.Println("     Fix: route reads to the same node, or use sticky sessions / sync replication.")
}

// ============================================================
// SCENARIO 6: Linearizability with a Mutex (Correct Behaviour)
// ============================================================

func scenarioLinearizability() {
	printHeader("SCENARIO 6: Linearizability (Strong Consistency via Mutex)")
	fmt.Println("  All writes go through a single coordinator with a lock.")
	fmt.Println("  Every read reflects the latest committed write — globally consistent.\n")

	type LinearStore struct {
		mu    sync.Mutex
		store map[string]string
	}

	ls := &LinearStore{store: make(map[string]string)}

	write := func(pid int, key, value string) {
		ls.mu.Lock()
		defer ls.mu.Unlock()
		ls.store[key] = value
		log(pid, "WRITE", key, value, "lock acquired → committed")
	}

	read := func(pid int, key string) string {
		ls.mu.Lock()
		defer ls.mu.Unlock()
		val := ls.store[key]
		if val == "" {
			val = "<nil>"
		}
		log(pid, "READ", key, val, "lock acquired → consistent read")
		return val
	}

	var wg sync.WaitGroup
	ops := []struct {
		pid   int
		key   string
		value string
	}{
		{0, "balance", "100"},
		{1, "balance", "150"},
		{2, "balance", "200"},
	}

	for _, op := range ops {
		wg.Add(1)
		go func(pid int, key, value string) {
			defer wg.Done()
			time.Sleep(time.Duration(rand.Intn(30)) * time.Millisecond)
			write(pid, key, value)
		}(op.pid, op.key, op.value)
	}
	wg.Wait()

	printDivider()
	fmt.Println("  All processes read after writes complete:")
	for i := 0; i < 3; i++ {
		read(i, "balance")
	}
	fmt.Println("\n  ✅ All processes see the same final value — linearizability holds.")
	fmt.Println("     Trade-off: the mutex is a bottleneck; limits throughput under contention.")
}

// ============================================================
// Main
// ============================================================

func main() {
	fmt.Println("╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║   Distributed Data Store — Consistency Scenarios in Go   ║")
	fmt.Println("║   3 Processes, Each with a Local Replica                  ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")

	scenarioNoConsistency()
	time.Sleep(50 * time.Millisecond)

	scenarioEventualConsistency()
	time.Sleep(50 * time.Millisecond)

	scenarioSequentialViolation()
	time.Sleep(50 * time.Millisecond)

	scenarioCausalConsistency()
	time.Sleep(50 * time.Millisecond)

	scenarioReadYourWritesViolation()
	time.Sleep(50 * time.Millisecond)

	scenarioLinearizability()

	fmt.Printf("\n\n%-60s\n", "============================================================")
	fmt.Println("SUMMARY")
	fmt.Println("============================================================")
	fmt.Printf("  %-35s %s\n", "No Consistency",         "→ stores diverge permanently")
	fmt.Printf("  %-35s %s\n", "Eventual Consistency",    "→ converges, but stale reads possible")
	fmt.Printf("  %-35s %s\n", "Sequential Violation",    "→ write order not preserved across nodes")
	fmt.Printf("  %-35s %s\n", "Causal Violation",        "→ reply visible before the question")
	fmt.Printf("  %-35s %s\n", "Read-Your-Writes Failure","→ client can't see its own write")
	fmt.Printf("  %-35s %s\n", "Linearizability (mutex)", "→ strong consistency, lower throughput")
	fmt.Println()
}
