package main

import (
	"fmt"
	"strings"
	"sync"
)

// ============================================================
// Entry Consistency — illustration
//
// Each shared variable has its own lock.
// Consistency is only guaranteed for variables protected by
// the same lock that the current process holds.
//
// Scenario:
//   Variables: x (guarded by Lx), y (guarded by Ly)
//   Processes: P1 and P2
//
// P1: acquire(Lx), W(x)a, release(Lx)
//     acquire(Ly), W(y)a, release(Ly)
//
// P2: acquire(Lx), R(x),  release(Lx)   ← sees x=a (consistent via Lx)
//     acquire(Ly), R(y),  release(Ly)   ← sees y=a (consistent via Ly)
//
// GROUPING OPERATIONS scenario:
//   P1 groups { W(x)a, W(y)a } as one atomic unit.
//   P2 either sees both writes or neither — never a partial state.
// ============================================================

// ── Types ────────────────────────────────────────────────────────────────────

type Op struct {
	slot    int
	pid     int
	kind    string // W, R, ACQ, REL, GROUP_BEGIN, GROUP_END
	varName string
	value   string
}

func (o Op) label() string {
	switch o.kind {
	case "ACQ":
		return fmt.Sprintf("acq(%s)", o.varName)
	case "REL":
		return fmt.Sprintf("rel(%s)", o.varName)
	case "GROUP_BEGIN":
		return "grp_begin"
	case "GROUP_END":
		return "grp_end"
	default:
		return fmt.Sprintf("%s%d(%s)%s", o.kind, o.pid, o.varName, o.value)
	}
}

func wop(slot, pid int, varName, value string) Op {
	return Op{slot, pid, "W", varName, value}
}
func rop(slot, pid int, varName, value string) Op {
	return Op{slot, pid, "R", varName, value}
}
func acq(slot, pid int, lock string) Op {
	return Op{slot, pid, "ACQ", lock, ""}
}
func rel(slot, pid int, lock string) Op {
	return Op{slot, pid, "REL", lock, ""}
}
func gbeg(slot, pid int) Op { return Op{slot, pid, "GROUP_BEGIN", "", ""} }
func gend(slot, pid int) Op { return Op{slot, pid, "GROUP_END", "", ""} }

// ── Timeline printer ─────────────────────────────────────────────────────────

func centerPad(s string, width int) string {
	if len(s) >= width { return s }
	total := width - len(s)
	left := total / 2
	return strings.Repeat("─", left) + s + strings.Repeat("─", total-left)
}

func printTimeline(ops []Op, numProcs, totalSlots int) {
	colW := 12
	dash := strings.Repeat("─", colW)
	hdr := fmt.Sprintf("  %-4s  ", "")
	for s := 1; s <= totalSlots; s++ {
		hdr += centerPad(fmt.Sprintf("t%d", s), colW)
	}
	fmt.Println(hdr)
	fmt.Printf("  %-4s  %s\n", "", strings.Repeat("─", colW*totalSlots))
	for pid := 1; pid <= numProcs; pid++ {
		row := fmt.Sprintf("  P%-3d  ", pid)
		for s := 1; s <= totalSlots; s++ {
			cell := ""
			for _, op := range ops {
				if op.pid == pid && op.slot == s {
					cell = op.label()
				}
			}
			if cell == "" {
				row += dash
			} else {
				row += centerPad(cell, colW)
			}
		}
		fmt.Println(row)
	}
	fmt.Println()
}

func sep(char string, n int) { fmt.Println(strings.Repeat(char, n)) }

// ── Entry Consistent Store ───────────────────────────────────────────────────
// Each variable is associated with its own lock.
// A process must hold the variable's lock before accessing it.
// On acquire, only that variable's latest value is pulled in.

type EntryStore struct {
	mu    sync.Mutex
	vars  map[string]string         // global committed values
	locks map[string]*sync.Mutex    // one lock per variable
	owner map[string]int            // which pid currently holds each lock
}

func NewEntryStore(varNames ...string) *EntryStore {
	s := &EntryStore{
		vars:  make(map[string]string),
		locks: make(map[string]*sync.Mutex),
		owner: make(map[string]int),
	}
	for _, v := range varNames {
		s.locks[v] = &sync.Mutex{}
		s.vars[v] = "NIL"
	}
	return s
}

func (s *EntryStore) Acquire(pid int, varName string) {
	s.locks[varName].Lock()
	s.mu.Lock()
	s.owner[varName] = pid
	s.mu.Unlock()
	fmt.Printf("  [P%d] acq(L%s)  — pulls in latest %s=%s\n",
		pid, varName, varName, s.vars[varName])
}

func (s *EntryStore) Release(pid int, varName string) {
	s.mu.Lock()
	s.owner[varName] = 0
	s.mu.Unlock()
	s.locks[varName].Unlock()
	fmt.Printf("  [P%d] rel(L%s)\n", pid, varName)
}

func (s *EntryStore) Write(pid int, varName, value string) {
	s.mu.Lock()
	if s.owner[varName] != pid {
		fmt.Printf("  [P%d] ⚠️  W(%s)%s attempted WITHOUT holding lock — entry consistency violation!\n",
			pid, varName, value)
		s.mu.Unlock()
		return
	}
	s.vars[varName] = value
	s.mu.Unlock()
	fmt.Printf("  [P%d] W(%s)%s\n", pid, varName, value)
}

func (s *EntryStore) Read(pid int, varName string) string {
	s.mu.Lock()
	if s.owner[varName] != pid {
		fmt.Printf("  [P%d] ⚠️  R(%s) attempted WITHOUT holding lock — entry consistency violation!\n",
			pid, varName)
		s.mu.Unlock()
		return "INCONSISTENT"
	}
	val := s.vars[varName]
	s.mu.Unlock()
	fmt.Printf("  [P%d] R(%s)=%s\n", pid, varName, val)
	return val
}

// ── Grouping Operations Store ─────────────────────────────────────────────────
// A group { op1, op2, ... } appears atomic to other processes.
// While a group is open, its writes are buffered and only committed
// all-at-once when the group closes.

type GroupStore struct {
	mu      sync.Mutex
	vars    map[string]string
	groupMu sync.Mutex
	buffer  map[string]string // writes buffered during open group
	inGroup bool
	groupOwner int
}

func NewGroupStore(varNames ...string) *GroupStore {
	s := &GroupStore{
		vars:   make(map[string]string),
		buffer: make(map[string]string),
	}
	for _, v := range varNames {
		s.vars[v] = "NIL"
	}
	return s
}

func (s *GroupStore) BeginGroup(pid int) {
	s.groupMu.Lock()
	s.inGroup = true
	s.groupOwner = pid
	fmt.Printf("  [P%d] grp_begin — writes buffered until grp_end\n", pid)
}

func (s *GroupStore) EndGroup(pid int) {
	// commit all buffered writes atomically
	s.mu.Lock()
	for k, v := range s.buffer {
		s.vars[k] = v
		fmt.Printf("  [P%d] grp_end  — committing W(%s)%s atomically\n", pid, k, v)
	}
	s.buffer = make(map[string]string)
	s.inGroup = false
	s.mu.Unlock()
	s.groupMu.Unlock()
}

func (s *GroupStore) Write(pid int, varName, value string) {
	if s.inGroup && s.groupOwner == pid {
		s.buffer[varName] = value
		fmt.Printf("  [P%d] W(%s)%s  [buffered in group]\n", pid, varName, value)
	} else {
		s.mu.Lock()
		s.vars[varName] = value
		s.mu.Unlock()
		fmt.Printf("  [P%d] W(%s)%s  [committed immediately]\n", pid, varName, value)
	}
}

func (s *GroupStore) Read(pid int, varName string) string {
	s.mu.Lock()
	val := s.vars[varName]
	s.mu.Unlock()
	fmt.Printf("  [P%d] R(%s)=%s\n", pid, varName, val)
	return val
}

// ── SCENARIO A: Entry Consistency ─────────────────────────────────────────────

func scenarioEntryConsistency() {
	sep("═", 72)
	fmt.Println("SCENARIO A — Entry Consistency")
	sep("═", 72)
	fmt.Println()
	fmt.Println("  Each variable has its own lock: Lx guards x, Ly guards y.")
	fmt.Println("  A process must hold a variable's lock to access it.")
	fmt.Println("  Only that variable is synchronised on acquire — not the whole store.")
	fmt.Println()

	ops := []Op{
		acq(1, 1, "x"),      // P1 acquires Lx
		wop(2, 1, "x", "a"), // P1 writes x=a  (holds Lx)
		rel(3, 1, "x"),      // P1 releases Lx
		acq(4, 1, "y"),      // P1 acquires Ly
		wop(5, 1, "y", "a"), // P1 writes y=a  (holds Ly)
		rel(6, 1, "y"),      // P1 releases Ly

		acq(4, 2, "x"),      // P2 acquires Lx  (after P1 releases)
		rop(5, 2, "x", "a"), // P2 reads x=a  (consistent: holds Lx)
		rel(6, 2, "x"),      // P2 releases Lx
		acq(7, 2, "y"),      // P2 acquires Ly
		rop(8, 2, "y", "a"), // P2 reads y=a  (consistent: holds Ly)
		rel(9, 2, "y"),      // P2 releases Ly
	}

	printTimeline(ops, 2, 10)

	fmt.Println("  Execution:")
	fmt.Println()
	store := NewEntryStore("x", "y")

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		store.Acquire(1, "x")
		store.Write(1, "x", "a")
		store.Release(1, "x")
		store.Acquire(1, "y")
		store.Write(1, "y", "a")
		store.Release(1, "y")
	}()
	go func() {
		defer wg.Done()
		store.Acquire(2, "x") // blocks until P1 releases Lx
		store.Read(2, "x")
		store.Release(2, "x")
		store.Acquire(2, "y") // blocks until P1 releases Ly
		store.Read(2, "y")
		store.Release(2, "y")
	}()
	wg.Wait()

	fmt.Println()
	fmt.Println("  ✅ Entry consistent: P2 saw x=a and y=a after acquiring each variable's lock.")
	fmt.Println("     Only the locked variable's state was synchronised — not the whole store.")

	// Show what happens without the lock
	fmt.Println()
	sep("─", 72)
	fmt.Println("  VIOLATION — accessing y without holding Ly:")
	sep("─", 72)
	fmt.Println()
	store2 := NewEntryStore("x", "y")
	store2.Acquire(1, "x")
	store2.Write(1, "x", "b")
	store2.Release(1, "x")
	store2.Read(2, "y") // no lock held — violation
	fmt.Println()
}

// ── SCENARIO B: Grouping Operations ──────────────────────────────────────────

func scenarioGrouping() {
	sep("═", 72)
	fmt.Println("SCENARIO B — Grouping Operations")
	sep("═", 72)
	fmt.Println()
	fmt.Println("  P1 groups W(x)a and W(y)a into one atomic unit.")
	fmt.Println("  P2 either sees both writes or neither — no partial state.")
	fmt.Println()

	ops := []Op{
		gbeg(1, 1),          // P1 opens group
		wop(2, 1, "x", "a"), // P1 writes x=a  (buffered)
		wop(3, 1, "y", "a"), // P1 writes y=a  (buffered)
		gend(4, 1),          // P1 closes group — both committed atomically
		rop(5, 2, "x", "a"), // P2 reads x — sees a
		rop(5, 2, "y", "a"), // P2 reads y — sees a (both or neither)
	}

	printTimeline(ops, 2, 6)

	fmt.Println("  Execution:")
	fmt.Println()
	store := NewGroupStore("x", "y")

	store.BeginGroup(1)
	store.Write(1, "x", "a")
	store.Write(1, "y", "a")
	store.EndGroup(1)

	fmt.Println()
	fmt.Println("  P2 now reads (after group committed):")
	store.Read(2, "x")
	store.Read(2, "y")

	fmt.Println()
	fmt.Println("  ✅ Grouping operations: P2 sees both x=a and y=a atomically.")
	fmt.Println("     No interleaved state (e.g. x=a but y=NIL) is possible.")

	// Show partial state is impossible — P2 reading mid-group
	fmt.Println()
	sep("─", 72)
	fmt.Println("  PARTIAL READ PREVENTION — P2 cannot read mid-group:")
	sep("─", 72)
	fmt.Println()
	store2 := NewGroupStore("x", "y")
	store2.BeginGroup(1)
	store2.Write(1, "x", "b") // buffered
	store2.Read(2, "x")        // P2 reads — group not committed yet, sees NIL
	store2.Write(1, "y", "b") // buffered
	store2.EndGroup(1)         // now both commit
	fmt.Println()
	fmt.Println("  Before commit: P2 read x=NIL (group not yet visible).")
	store2.Read(2, "x")        // after group: P2 reads committed value
	store2.Read(2, "y")        // both committed
	fmt.Println()
}

// ── COMPARISON TABLE ─────────────────────────────────────────────────────────

func comparisonTable() {
	sep("═", 72)
	fmt.Println("COMPARISON — Weak / Entry / Grouping vs Sequential Consistency")
	sep("═", 72)
	fmt.Println()

	fmt.Printf("  %-22s  %-14s  %-14s  %s\n",
		"Model", "Sync scope", "Granularity", "Notes")
	fmt.Printf("  %-22s  %-14s  %-14s  %s\n",
		strings.Repeat("─", 22), strings.Repeat("─", 14),
		strings.Repeat("─", 14), strings.Repeat("─", 36))

	rows := [][]string{
		{"Sequential", "always", "every op", "strongest; all ops totally ordered"},
		{"Weak", "at sync points", "all vars", "sync flushes entire store"},
		{"Release", "acq/rel", "all vars", "acquire pulls all; release pushes all"},
		{"Entry", "acq/rel per var", "per variable", "acquire pulls only that variable"},
		{"Grouping ops", "group boundary", "per group", "group appears atomic; no partial view"},
	}

	for _, row := range rows {
		fmt.Printf("  %-22s  %-14s  %-14s  %s\n", row[0], row[1], row[2], row[3])
	}
	fmt.Println()
	fmt.Println("  Key insight:")
	fmt.Println("  Entry consistency is the finest-grained lock-based model —")
	fmt.Println("  each lock protects exactly its own data, minimising the")
	fmt.Println("  synchronisation overhead compared to weak/release consistency.")
	fmt.Println()
	fmt.Println("  Grouping operations is orthogonal — it is about atomicity of")
	fmt.Println("  a batch of writes, ensuring observers see all-or-nothing,")
	fmt.Println("  regardless of which consistency model underlies the store.")
	fmt.Println()
}

func main() {
	sep("═", 72)
	fmt.Println("GROUPING OPERATIONS AND ENTRY CONSISTENCY")
	sep("═", 72)
	fmt.Println()
	fmt.Println("  Notation:  Wi(v)x  = process i writes x to v")
	fmt.Println("             Ri(v)x  = process i reads x from v")
	fmt.Println("             acq(Lv) = acquire lock guarding variable v")
	fmt.Println("             rel(Lv) = release lock guarding variable v")
	fmt.Println("             grp_begin / grp_end = group boundary (atomic batch)")
	fmt.Println()

	scenarioEntryConsistency()
	fmt.Println()
	scenarioGrouping()
	fmt.Println()
	comparisonTable()
}
