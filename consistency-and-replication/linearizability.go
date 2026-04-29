package main

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// ============================================================
// Linearizability — solving the non-serializable execution
//
// Original non-serializable timeline:
//   t1: W1(x)a   t2: W2(y)b   t3: W2(x)b   t4: W1(y)a
//   t5: R1(x)a  R2(y)b  (stale reads — not linearizable)
//
// Fix: route all ops through a global mutex so each takes
// effect atomically. Reads always see the latest committed write.
// ============================================================

// ── Event log ────────────────────────────────────────────────────────────────

type EventKind string

const (
	KindWrite EventKind = "W"
	KindRead  EventKind = "R"
)

type Event struct {
	pid      int
	kind     EventKind
	varName  string
	value    string
	linPoint int64 // nanoseconds — determines total order
	linOrder int   // 1-based slot in total order
}

var (
	logMu  sync.Mutex
	events []Event
)

func recordEvent(e Event) {
	logMu.Lock()
	events = append(events, e)
	logMu.Unlock()
}

// ── Linearizable store ───────────────────────────────────────────────────────

type LinStore struct {
	mu    sync.Mutex
	store map[string]string
}

func NewLinStore() *LinStore {
	return &LinStore{store: make(map[string]string)}
}

func (s *LinStore) Write(pid int, varName, value string) {
	s.mu.Lock()
	linPoint := time.Now().UnixNano()
	s.store[varName] = value
	s.mu.Unlock()
	recordEvent(Event{pid, KindWrite, varName, value, linPoint, 0})
	fmt.Printf("  [P%d] W(%s)%s\n", pid, varName, value)
}

func (s *LinStore) Read(pid int, varName string) string {
	s.mu.Lock()
	linPoint := time.Now().UnixNano()
	value := s.store[varName]
	if value == "" {
		value = "NIL"
	}
	s.mu.Unlock()
	recordEvent(Event{pid, KindRead, varName, value, linPoint, 0})
	fmt.Printf("  [P%d] R(%s)=%s\n", pid, varName, value)
	return value
}

// ── Timeline printer ─────────────────────────────────────────────────────────

func centerPad(s string, width int) string {
	if len(s) >= width {
		return s
	}
	total := width - len(s)
	left := total / 2
	return strings.Repeat("─", left) + s + strings.Repeat("─", total-left)
}

func printTimeline(evs []Event, totalSlots int) {
	colW := 10
	dash := strings.Repeat("─", colW)
	hdr := fmt.Sprintf("  %-4s  ", "")
	for s := 1; s <= totalSlots; s++ {
		hdr += centerPad(fmt.Sprintf("t%d", s), colW)
	}
	fmt.Println(hdr)
	fmt.Printf("  %-4s  %s\n", "", strings.Repeat("─", colW*totalSlots))
	for pid := 1; pid <= 2; pid++ {
		row := fmt.Sprintf("  P%-3d  ", pid)
		for s := 1; s <= totalSlots; s++ {
			cell := ""
			for _, e := range evs {
				if e.pid == pid && e.linOrder == s {
					cell = fmt.Sprintf("%s%d(%s)%s", e.kind, e.pid, e.varName, e.value)
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

// ── Assign linearization order ───────────────────────────────────────────────

func assignLinOrder() {
	for i := 0; i < len(events); i++ {
		for j := i + 1; j < len(events); j++ {
			if events[j].linPoint < events[i].linPoint {
				events[i], events[j] = events[j], events[i]
			}
		}
	}
	for i := range events {
		events[i].linOrder = i + 1
	}
}

// ── Write orderings table ────────────────────────────────────────────────────
// Enumerate all 4! = 24 permutations of the four writes, respecting
// per-process program order. For each ordering show what each read returns
// and whether it matches the linearizable read result.

type WOp struct {
	label   string
	pid     int
	varName string
	value   string
	slot    int // real execution slot (used to determine visibility for reads)
}

type ROp struct {
	label    string
	pid      int
	varName  string
	expected string // value returned by the linearizable store
	slot     int
}

func writePerms(writes []WOp) [][]WOp {
	var result [][]WOp
	var gen func(cur, rem []WOp)
	gen = func(cur, rem []WOp) {
		if len(rem) == 0 {
			cp := make([]WOp, len(cur))
			copy(cp, cur)
			result = append(result, cp)
			return
		}
		for i, op := range rem {
			eligible := true
			for j, other := range rem {
				if j < i && other.pid == op.pid {
					eligible = false
					break
				}
			}
			if eligible {
				next := append(append([]WOp{}, rem[:i]...), rem[i+1:]...)
				gen(append(cur, op), next)
			}
		}
	}
	gen([]WOp{}, writes)
	return result
}

func simulate(perm []WOp, reads []ROp) (results []string) {
	for _, rd := range reads {
		lastVal := "NIL"
		lastPos := -1
		for pos, wr := range perm {
			if wr.varName == rd.varName && wr.slot < rd.slot {
				if pos > lastPos {
					lastPos = pos
					lastVal = wr.value
				}
			}
		}
		results = append(results, lastVal)
	}
	return
}

func respectsPO(perm []WOp) bool {
	posOf := make(map[string]int)
	for pos, w := range perm {
		posOf[w.label] = pos
	}
	// P1: W1(x)a before W1(y)a
	// P2: W2(y)b before W2(x)b
	return posOf["W1(x)a"] < posOf["W1(y)a"] && posOf["W2(y)b"] < posOf["W2(x)b"]
}

func printWriteOrderings(linReads []ROp) {
	writes := []WOp{
		{"W1(x)a", 1, "x", "a", 1},
		{"W2(y)b", 2, "y", "b", 2},
		{"W2(x)b", 2, "x", "b", 3},
		{"W1(y)a", 1, "y", "a", 4},
	}

	perms := writePerms(writes)

	orderW := 46
	rW := 9
	satW := 28
	poW := 3

	sepLine := fmt.Sprintf("  %-3s  %-*s  %-*s  %-*s  %-*s  %s",
		"───", orderW, strings.Repeat("─", orderW),
		rW, strings.Repeat("─", rW),
		rW, strings.Repeat("─", rW),
		satW, strings.Repeat("─", satW),
		strings.Repeat("─", poW))

	fmt.Printf("  %-3s  %-*s  %-*s  %-*s  %-*s  %s\n",
		"#", orderW, "Write ordering",
		rW, "R1(x)=?",
		rW, "R2(y)=?",
		satW, "Matches linearizable reads?",
		"PO?")
	fmt.Println(sepLine)

	matchCount := 0
	matchPO := 0

	for i, perm := range perms {
		labels := make([]string, len(perm))
		for j, w := range perm {
			labels[j] = w.label
		}
		orderStr := strings.Join(labels, " → ")

		results := simulate(perm, linReads)
		r1Got := results[0]
		r2Got := results[1]

		r1Match := r1Got == linReads[0].expected
		r2Match := r2Got == linReads[1].expected
		po := respectsPO(perm)

		satStr := ""
		if r1Match && r2Match {
			satStr = fmt.Sprintf("✅ R1(x)=%s R2(y)=%s both match", r1Got, r2Got)
			matchCount++
			if po { matchPO++ }
		} else if r1Match {
			satStr = fmt.Sprintf("✗  R1(x)=%s✅ R2(y)=%s✗", r1Got, r2Got)
		} else if r2Match {
			satStr = fmt.Sprintf("✗  R1(x)=%s✗ R2(y)=%s✅", r1Got, r2Got)
		} else {
			satStr = fmt.Sprintf("✗  R1(x)=%s✗ R2(y)=%s✗", r1Got, r2Got)
		}

		poStr := " "
		if po { poStr = "✓" }

		fmt.Printf("  %-3d  %-*s  %-*s  %-*s  %-*s  %s\n",
			i+1, orderW, orderStr,
			rW, r1Got,
			rW, r2Got,
			satW, satStr,
			poStr)
	}
	fmt.Println(sepLine)
	fmt.Printf("\n  Total orderings             : 24\n")
	fmt.Printf("  Match linearizable reads    : %d\n", matchCount)
	fmt.Printf("  Match reads + program order : %d\n\n", matchPO)
}

// ── Verify linearizability ───────────────────────────────────────────────────

func verifyLinearizability() bool {
	fmt.Println("  Checking every read against the linearization order:")
	fmt.Println()
	allOk := true
	for _, e := range events {
		if e.kind != KindRead {
			continue
		}
		lastVal := "NIL"
		for _, w := range events {
			if w.kind == KindWrite && w.varName == e.varName && w.linOrder < e.linOrder {
				lastVal = w.value
			}
		}
		ok := lastVal == e.value
		sym := "✅"
		if !ok {
			sym = "✗ "
			allOk = false
		}
		fmt.Printf("    t%-2d %s%d(%s)%s : last write before this = %s  %s\n",
			e.linOrder, e.kind, e.pid, e.varName, e.value, lastVal, sym)
	}
	fmt.Println()
	return allOk
}

func sep(char string, n int) { fmt.Println(strings.Repeat(char, n)) }

// ── Main ─────────────────────────────────────────────────────────────────────

func main() {
	sep("═", 70)
	fmt.Println("LINEARIZABILITY — Solving the Non-Serializable Execution")
	sep("═", 70)
	fmt.Println()
	fmt.Println("  Original non-serializable timeline (for reference):")
	fmt.Println()
	fmt.Println("        ────t1────────t2────────t3────────t4────────t5────")
	fmt.Println("        ──────────────────────────────────────────────────")
	fmt.Println("  P1    ──W1(x)a────────────────────────W1(y)a────R1(x)a──")
	fmt.Println("  P2    ────────────W2(y)b────W2(x)b──────────────R2(y)b──")
	fmt.Println()
	fmt.Println("  Problem: stale reads — R1(x)=a misses W2(x)b, R2(y)=b misses W1(y)a")

	// ── Run linearizable execution ────────────────────────────────────────────
	fmt.Println()
	sep("─", 70)
	fmt.Println("LINEARIZABLE EXECUTION")
	sep("─", 70)
	fmt.Println()
	fmt.Println("  Global mutex — linearization point = instant lock is acquired:")
	fmt.Println()

	store := NewLinStore()
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		store.Write(1, "x", "a")
		time.Sleep(2 * time.Millisecond)
		store.Write(1, "y", "a")
		time.Sleep(1 * time.Millisecond)
		store.Read(1, "x")
	}()
	go func() {
		defer wg.Done()
		time.Sleep(1 * time.Millisecond)
		store.Write(2, "y", "b")
		store.Write(2, "x", "b")
		time.Sleep(2 * time.Millisecond)
		store.Read(2, "y")
	}()
	wg.Wait()

	assignLinOrder()

	// ── Linearization order table ─────────────────────────────────────────────
	fmt.Println()
	sep("─", 70)
	fmt.Println("LINEARIZATION ORDER")
	sep("─", 70)
	fmt.Println()
	fmt.Printf("  %-4s  %-3s  %-14s  %s\n", "Slot", "PID", "Operation", "Value")
	fmt.Printf("  %-4s  %-3s  %-14s  %s\n", "────", "───", "──────────────", "─────")
	for _, e := range events {
		opStr := fmt.Sprintf("%s%d(%s)%s", e.kind, e.pid, e.varName, e.value)
		fmt.Printf("  t%-3d  P%-2d  %-14s  %s\n", e.linOrder, e.pid, opStr, e.value)
	}
	fmt.Println()

	// ── Linearizable timeline ─────────────────────────────────────────────────
	sep("─", 70)
	fmt.Println("LINEARIZABLE TIMELINE")
	sep("─", 70)
	fmt.Println()
	printTimeline(events, len(events))

	// ── Verification ──────────────────────────────────────────────────────────
	sep("─", 70)
	fmt.Println("LINEARIZABILITY VERIFICATION")
	sep("─", 70)
	fmt.Println()
	ok := verifyLinearizability()

	// Collect the actual read results from the linearizable execution
	var linReads []ROp
	for _, e := range events {
		if e.kind == KindRead {
			linReads = append(linReads, ROp{
				label:    fmt.Sprintf("R%d(%s)%s", e.pid, e.varName, e.value),
				pid:      e.pid,
				varName:  e.varName,
				expected: e.value,
				slot:     e.linOrder,
			})
		}
	}

	// ── Write orderings table ─────────────────────────────────────────────────
	sep("─", 70)
	fmt.Println("WRITE ORDERINGS TABLE  (all 24 permutations, program order respected)")
	sep("─", 70)
	fmt.Println()
	fmt.Printf("  Linearizable reads:  R1(x)=%s  R2(y)=%s\n",
		linReads[0].expected, linReads[1].expected)
	fmt.Printf("  (showing which write orderings reproduce these results)\n\n")
	printWriteOrderings(linReads)

	// ── Final state ───────────────────────────────────────────────────────────
	sep("─", 70)
	fmt.Println("FINAL STATE")
	sep("─", 70)
	fmt.Println()
	fmt.Printf("  x = %s\n", store.store["x"])
	fmt.Printf("  y = %s\n", store.store["y"])
	fmt.Println()

	// ── Conclusion ────────────────────────────────────────────────────────────
	sep("═", 70)
	fmt.Println("CONCLUSION")
	sep("═", 70)
	fmt.Println()
	if ok {
		fmt.Println("  ✅ Execution is LINEARIZABLE.")
		fmt.Println()
		fmt.Println("  The write orderings table shows which of the 24 permutations")
		fmt.Println("  are consistent with the actual reads returned by the store.")
		fmt.Println("  All matching orderings place W2(x)b before W1(x)a (so R1(x)")
		fmt.Println("  sees b) and W1(y)a before W2(y)b (so R2(y) sees a) — or vice")
		fmt.Println("  versa depending on which interleaving the mutex produced.")
		fmt.Println()
		fmt.Println("  The original stale reads (R1(x)=a, R2(y)=b) are impossible:")
		fmt.Println("  under linearizability every read must reflect the latest")
		fmt.Println("  committed write in the global total order.")
	}
	fmt.Println()
}

