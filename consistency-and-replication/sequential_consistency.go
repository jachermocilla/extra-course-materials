package main

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

func W(pid int, key, val string) string { return fmt.Sprintf("W%d(%s)%s", pid, key, val) }
func R(pid int, key, val string) string { return fmt.Sprintf("R%d(%s)%s", pid, key, val) }

// ============================================================
// Event log — records every operation with a slot index
// ============================================================

type Event struct {
	op   string
	pid  int
	slot int // time slot (column) on the timeline
}

type EventLog struct {
	mu     sync.Mutex
	events []Event
	slot   int // global monotonic slot counter
}

func (l *EventLog) record(pid int, op string) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.slot++
	s := l.slot
	l.events = append(l.events, Event{op: op, pid: pid, slot: s})
	return s
}

func (l *EventLog) readSeq(pid int) []string {
	var seq []string
	for _, e := range l.events {
		if e.pid == pid && strings.HasPrefix(e.op, "R") {
			parts := strings.Split(e.op, ")")
			seq = append(seq, parts[len(parts)-1])
		}
	}
	return seq
}

// printTimeline draws an ASCII timeline for each process.
// Each slot is one column; operations are placed at their slot.
func (l *EventLog) printTimeline(pids []int) {
	if l.slot == 0 {
		return
	}

	// Build a grid: grid[pid][slot] = op label (or "")
	grid := make(map[int]map[int]string)
	for _, pid := range pids {
		grid[pid] = make(map[int]string)
	}
	for _, e := range l.events {
		grid[e.pid][e.slot] = e.op
	}

	// Column width: wide enough for the longest op label + padding
	colW := 0
	for _, e := range l.events {
		if len(e.op) > colW {
			colW = len(e.op)
		}
	}
	colW += 2 // side padding

	dash := strings.Repeat("─", colW)

	fmt.Println()
	// Time axis header
	header := fmt.Sprintf("  %-4s  ", "")
	for s := 1; s <= l.slot; s++ {
		header += centerPad(fmt.Sprintf("t%d", s), colW)
	}
	fmt.Println(header)
	fmt.Printf("  %-4s  %s\n", "", strings.Repeat("─", colW*l.slot))

	// One row per process
	for _, pid := range pids {
		row := fmt.Sprintf("  P%-3d  ", pid)
		for s := 1; s <= l.slot; s++ {
			op := grid[pid][s]
			if op == "" {
				row += dash
			} else {
				row += centerPad(op, colW)
			}
		}
		fmt.Println(row)
	}
	fmt.Println()
}

func centerPad(s string, width int) string {
	if len(s) >= width {
		return s
	}
	total := width - len(s)
	left := total / 2
	right := total - left
	return strings.Repeat("─", left) + s + strings.Repeat("─", right)
}

// ============================================================
// Replica
// ============================================================

type Replica struct {
	pid   int
	store map[string]string
	mu    sync.RWMutex
	log   *EventLog
}

func NewReplica(pid int, log *EventLog) *Replica {
	return &Replica{pid: pid, store: make(map[string]string), log: log}
}

func (r *Replica) Write(key, val string) {
	r.mu.Lock()
	r.store[key] = val
	r.mu.Unlock()
	r.log.record(r.pid, W(r.pid, key, val))
}

func (r *Replica) Read(key string) string {
	r.mu.RLock()
	val := r.store[key]
	r.mu.RUnlock()
	if val == "" {
		val = "NIL"
	}
	r.log.record(r.pid, R(r.pid, key, val))
	return val
}

func (r *Replica) deliver(key, val string) {
	r.mu.Lock()
	r.store[key] = val
	r.mu.Unlock()
}

// ============================================================
// SCENARIO A — SEQUENTIALLY CONSISTENT
// Global order: W1(x)a → W2(x)b
// P3 reads: a, b
// P4 reads: a, b   — both agree ✅
// ============================================================

func scenarioConsistent() {
	fmt.Println("════════════════════════════════════════════════════════════")
	fmt.Println("SCENARIO A — Sequentially Consistent")
	fmt.Println("════════════════════════════════════════════════════════════")
	fmt.Println("  All replicas receive writes in the same order: a → b.")
	fmt.Println("  P3 and P4 both read 'a' before 'b'.")

	log := &EventLog{}
	p1 := NewReplica(1, log)
	p2 := NewReplica(2, log)
	p3 := NewReplica(3, log)
	p4 := NewReplica(4, log)

	// t1: P1 writes a; deliver to readers
	p1.Write("x", "a")
	p3.deliver("x", "a")
	p4.deliver("x", "a")

	// t2,t3: P3 and P4 both read → 'a'
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); p3.Read("x") }()
	go func() { defer wg.Done(); p4.Read("x") }()
	wg.Wait()

	// t4: P2 writes b; deliver to readers
	time.Sleep(5 * time.Millisecond)
	p2.Write("x", "b")
	p3.deliver("x", "b")
	p4.deliver("x", "b")

	// t5,t6: P3 and P4 both read → 'b'
	wg.Add(2)
	go func() { defer wg.Done(); p3.Read("x") }()
	go func() { defer wg.Done(); p4.Read("x") }()
	wg.Wait()

	log.printTimeline([]int{1, 2, 3, 4})

	p3seq := log.readSeq(3)
	p4seq := log.readSeq(4)
	fmt.Printf("  P3 read sequence: %v\n", p3seq)
	fmt.Printf("  P4 read sequence: %v\n", p4seq)
	fmt.Println()
	fmt.Println("  Checking candidate global orders:")
	for _, order := range [][]string{{"a", "b"}, {"b", "a"}} {
		p3ok := matchesOrder(p3seq, order)
		p4ok := matchesOrder(p4seq, order)
		note := ""
		if p3ok && p4ok {
			note = "  ← agreed global order"
		}
		fmt.Printf("    [%s → %s]:  P3 %s   P4 %s%s\n",
			order[0], order[1], mark(p3ok), mark(p4ok), note)
	}
	fmt.Println()
	fmt.Println("  ✅ Sequentially consistent: one global order satisfies all processes.")
}

// ============================================================
// SCENARIO B — NOT SEQUENTIALLY CONSISTENT
// Writes cross paths:
// P3 receives a first → reads a, b  (implies a→b)
// P4 receives b first → reads b, a  (implies b→a)  ⚠️
// ============================================================

func scenarioViolation() {
	fmt.Println("════════════════════════════════════════════════════════════")
	fmt.Println("SCENARIO B — NOT Sequentially Consistent")
	fmt.Println("════════════════════════════════════════════════════════════")
	fmt.Println("  Writes travel via different paths — delivery order crosses.")
	fmt.Println("  P3 receives: a then b  |  P4 receives: b then a")

	log := &EventLog{}
	p1 := NewReplica(1, log)
	p2 := NewReplica(2, log)
	p3 := NewReplica(3, log)
	p4 := NewReplica(4, log)

	// t1,t2: P1 and P2 write to their own replicas
	p1.Write("x", "a")
	p2.Write("x", "b")

	// Crossed deliveries: P3 gets 'a', P4 gets 'b'
	p3.deliver("x", "a")
	p4.deliver("x", "b")

	// t3,t4: first reads — P3→'a', P4→'b'
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); p3.Read("x") }()
	go func() { defer wg.Done(); p4.Read("x") }()
	wg.Wait()

	// Slow paths arrive: P3 gets 'b', P4 gets 'a'
	time.Sleep(5 * time.Millisecond)
	p3.deliver("x", "b")
	p4.deliver("x", "a")

	// t5,t6: second reads — P3→'b', P4→'a'
	wg.Add(2)
	go func() { defer wg.Done(); p3.Read("x") }()
	go func() { defer wg.Done(); p4.Read("x") }()
	wg.Wait()

	log.printTimeline([]int{1, 2, 3, 4})

	p3seq := log.readSeq(3)
	p4seq := log.readSeq(4)
	fmt.Printf("  P3 read sequence: %v   (implies order: %s → %s)\n",
		p3seq, p3seq[0], p3seq[1])
	fmt.Printf("  P4 read sequence: %v   (implies order: %s → %s)\n",
		p4seq, p4seq[0], p4seq[1])
	fmt.Println()
	fmt.Println("  Checking candidate global orders:")
	anyMatch := false
	for _, order := range [][]string{{"a", "b"}, {"b", "a"}} {
		p3ok := matchesOrder(p3seq, order)
		p4ok := matchesOrder(p4seq, order)
		note := ""
		if p3ok && p4ok {
			anyMatch = true
			note = "  ← would satisfy all"
		}
		fmt.Printf("    [%s → %s]:  P3 %s   P4 %s%s\n",
			order[0], order[1], mark(p3ok), mark(p4ok), note)
	}
	if !anyMatch {
		fmt.Println()
		fmt.Println("  ⚠️  No single global order satisfies both P3 and P4.")
		fmt.Println("     Sequential consistency is VIOLATED.")
	}
}

// ============================================================
// Helpers
// ============================================================

func matchesOrder(readSeq []string, writeOrder []string) bool {
	if len(readSeq) < 2 || len(writeOrder) < 2 {
		return true
	}
	posA := indexOf(readSeq, writeOrder[0])
	posB := indexOf(readSeq, writeOrder[1])
	if posA == -1 || posB == -1 {
		return true
	}
	return posA < posB
}

func indexOf(seq []string, val string) int {
	for i, v := range seq {
		if v == val {
			return i
		}
	}
	return -1
}

func mark(ok bool) string {
	if ok {
		return "✅"
	}
	return "✗ "
}

// ============================================================
// Main
// ============================================================

func main() {
	fmt.Println("╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║        Sequential Consistency — Go Simulation            ║")
	fmt.Println("║  Wi(x)v = process i writes v to x                       ║")
	fmt.Println("║  Ri(x)v = process i reads v from x                      ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")
	fmt.Println()

	scenarioConsistent()
	fmt.Println()
	scenarioViolation()

	fmt.Println()
	fmt.Println("════════════════════════════════════════════════════════════")
	fmt.Println("KEY POINT")
	fmt.Println("════════════════════════════════════════════════════════════")
	fmt.Println("  Sequential consistency holds when ONE global write order")
	fmt.Println("  exists that satisfies every process's read sequence.")
	fmt.Println("  It is violated when processes observe contradictory orders.")
	fmt.Println()
}
