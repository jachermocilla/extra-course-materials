package main

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"time"
)

// ============================================================
// Eventual Consistency — illustration
//
// Three replicas R1, R2, R3. Writes go to one replica and
// propagate asynchronously with random delays.
//
// Scenarios:
//   A) Basic eventual convergence — one write, replicas converge
//   B) Stale read — client reads from lagging replica
//   C) Concurrent writes + Last-Write-Wins conflict resolution
//   D) Monotonic read violation — later read returns older value
// ============================================================

// ── Event log ────────────────────────────────────────────────────────────────

type Event struct {
	slot    int
	replica int    // 1,2,3  (0 = client)
	kind    string // WRITE, READ, SYNC, NOTE
	key     string
	value   string
}

func (e Event) label() string {
	switch e.kind {
	case "WRITE":
		return fmt.Sprintf("W(%s)%s", e.key, e.value)
	case "READ":
		return fmt.Sprintf("R(%s)%s", e.key, e.value)
	case "SYNC":
		return fmt.Sprintf("sync(%s)%s", e.key, e.value)
	case "NOTE":
		return e.value
	}
	return e.kind
}

// ── Timeline printer ─────────────────────────────────────────────────────────

func centerPad(s string, width int) string {
	if len(s) >= width { return s }
	total := width - len(s)
	left := total / 2
	return strings.Repeat("─", left) + s + strings.Repeat("─", total-left)
}

func printTimeline(events []Event, labels []string, totalSlots int) {
	colW := 12
	dash := strings.Repeat("─", colW)
	hdr := fmt.Sprintf("  %-4s  ", "")
	for s := 1; s <= totalSlots; s++ {
		hdr += centerPad(fmt.Sprintf("t%d", s), colW)
	}
	fmt.Println(hdr)
	fmt.Printf("  %-4s  %s\n", "", strings.Repeat("─", colW*totalSlots))
	for ri, label := range labels {
		row := fmt.Sprintf("  %-4s  ", label)
		for s := 1; s <= totalSlots; s++ {
			cell := ""
			for _, e := range events {
				if e.replica == ri+1 && e.slot == s {
					cell = e.label()
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

// ── Replica ──────────────────────────────────────────────────────────────────

type Replica struct {
	id    int
	mu    sync.Mutex
	store map[string]valueWithTS
}

type valueWithTS struct {
	value string
	ts    int64 // unix nano — used for LWW
}

func NewReplica(id int) *Replica {
	return &Replica{id: id, store: make(map[string]valueWithTS)}
}

func (r *Replica) localWrite(key, value string, ts int64) {
	r.mu.Lock()
	r.store[key] = valueWithTS{value, ts}
	r.mu.Unlock()
}

func (r *Replica) read(key string) (string, int64) {
	r.mu.Lock()
	v := r.store[key]
	r.mu.Unlock()
	if v.value == "" {
		return "NIL", 0
	}
	return v.value, v.ts
}

// applyLWW applies a remote write only if its timestamp is newer
func (r *Replica) applyLWW(key, value string, ts int64) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	cur := r.store[key]
	if ts > cur.ts {
		r.store[key] = valueWithTS{value, ts}
		return true
	}
	return false
}

// ── Propagation simulation ────────────────────────────────────────────────────

type PropagationRecord struct {
	fromID  int
	toID    int
	key     string
	value   string
	ts      int64
	delayMs int
}

func propagateAsync(recs []PropagationRecord, replicas map[int]*Replica, log *[]string, mu *sync.Mutex) {
	var wg sync.WaitGroup
	for _, rec := range recs {
		wg.Add(1)
		go func(r PropagationRecord) {
			defer wg.Done()
			time.Sleep(time.Duration(r.delayMs) * time.Millisecond)
			accepted := replicas[r.toID].applyLWW(r.key, r.value, r.ts)
			mu.Lock()
			if accepted {
				*log = append(*log, fmt.Sprintf("  R%d ← R%d: sync(%s)%s  [+%dms]",
					r.toID, r.fromID, r.key, r.value, r.delayMs))
			} else {
				*log = append(*log, fmt.Sprintf("  R%d ← R%d: sync(%s)%s  [+%dms REJECTED by LWW — stale]",
					r.toID, r.fromID, r.key, r.value, r.delayMs))
			}
			mu.Unlock()
		}(rec)
	}
	wg.Wait()
}

// ── SCENARIO A: Basic Eventual Convergence ────────────────────────────────────

func scenarioA() {
	sep("═", 72)
	fmt.Println("SCENARIO A — Basic Eventual Convergence")
	sep("═", 72)
	fmt.Println()
	fmt.Println("  Client writes x=a to R1. R2 and R3 receive the update")
	fmt.Println("  asynchronously with different delays.")
	fmt.Println("  Reads during propagation show stale values.")
	fmt.Println()

	events := []Event{
		{1, 1, "WRITE", "x", "a"},                 // t1: R1 gets write
		{2, 2, "READ",  "x", "NIL"},               // t2: R2 still stale
		{2, 3, "READ",  "x", "NIL"},               // t2: R3 still stale
		{3, 2, "SYNC",  "x", "a"},                 // t3: R2 gets update
		{3, 2, "READ",  "x", "a"},                 // t3: R2 now consistent
		{4, 3, "SYNC",  "x", "a"},                 // t4: R3 gets update (slower)
		{4, 3, "READ",  "x", "a"},                 // t4: R3 now consistent
	}

	printTimeline(events, []string{"R1", "R2", "R3"}, 5)

	r1 := NewReplica(1)
	r2 := NewReplica(2)
	r3 := NewReplica(3)
	replicas := map[int]*Replica{1: r1, 2: r2, 3: r3}

	ts := time.Now().UnixNano()
	r1.localWrite("x", "a", ts)
	fmt.Printf("  R1: W(x)a  [ts=%d]\n", ts)

	fmt.Printf("  R2: R(x)=%s  [stale — propagation not yet arrived]\n", func() string { v, _ := r2.read("x"); return v }())
	fmt.Printf("  R3: R(x)=%s  [stale — propagation not yet arrived]\n\n", func() string { v, _ := r3.read("x"); return v }())

	var log []string
	var mu sync.Mutex
	propagateAsync([]PropagationRecord{
		{1, 2, "x", "a", ts, 30},
		{1, 3, "x", "a", ts, 80},
	}, replicas, &log, &mu)

	sort.Strings(log)
	for _, l := range log { fmt.Println(l) }
	fmt.Println()

	fmt.Printf("  After convergence:\n")
	for _, r := range []*Replica{r1, r2, r3} {
		v, _ := r.read("x")
		fmt.Printf("    R%d: x=%s\n", r.id, v)
	}
	fmt.Println()
	fmt.Println("  ✅ All replicas converged to x=a.")
	fmt.Println()
}

// ── SCENARIO B: Stale Read and Read-Your-Writes Violation ────────────────────

func scenarioB() {
	sep("═", 72)
	fmt.Println("SCENARIO B — Stale Read / Read-Your-Writes Violation")
	sep("═", 72)
	fmt.Println()
	fmt.Println("  Client writes x=a to R1 then immediately reads from R2.")
	fmt.Println("  R2 has not received the update yet — read-your-writes violated.")
	fmt.Println()

	events := []Event{
		{1, 1, "WRITE", "x", "a"},
		{2, 2, "READ",  "x", "NIL"},   // client reads from R2 — stale
		{3, 1, "READ",  "x", "a"},     // client reads from R1 — fresh
		{4, 2, "SYNC",  "x", "a"},     // eventually R2 gets it
		{5, 2, "READ",  "x", "a"},     // now R2 is consistent
	}

	printTimeline(events, []string{"R1", "R2", "R3"}, 6)

	r1 := NewReplica(1)
	r2 := NewReplica(2)

	ts := time.Now().UnixNano()
	r1.localWrite("x", "a", ts)
	fmt.Printf("  Client → R1: W(x)a\n")

	v2, _ := r2.read("x")
	fmt.Printf("  Client → R2: R(x)=%s  ⚠️  Read-your-writes VIOLATED — client wrote to R1 but reads stale from R2\n", v2)

	v1, _ := r1.read("x")
	fmt.Printf("  Client → R1: R(x)=%s  ✅ Reading from write replica returns fresh value\n", v1)

	time.Sleep(10 * time.Millisecond)
	r2.applyLWW("x", "a", ts)
	v2after, _ := r2.read("x")
	fmt.Printf("  After propagation → R2: R(x)=%s  ✅ Eventually consistent\n", v2after)
	fmt.Println()
}

// ── SCENARIO C: Concurrent Writes + LWW ──────────────────────────────────────

func scenarioC() {
	sep("═", 72)
	fmt.Println("SCENARIO C — Concurrent Writes + Last-Write-Wins (LWW)")
	sep("═", 72)
	fmt.Println()
	fmt.Println("  P1 writes x=a to R1. Concurrently P2 writes x=b to R2.")
	fmt.Println("  Both propagate to all replicas. LWW: higher timestamp wins.")
	fmt.Println()

	ts1 := time.Now().UnixNano()
	time.Sleep(1 * time.Millisecond)
	ts2 := time.Now().UnixNano() // ts2 > ts1 → x=b wins

	events := []Event{
		{1, 1, "WRITE", "x", "a"},
		{1, 2, "WRITE", "x", "b"},
		{2, 1, "SYNC",  "x", "b"},
		{2, 3, "SYNC",  "x", "a"},
		{3, 2, "SYNC",  "x", "a"},
		{3, 3, "SYNC",  "x", "b"},
		{4, 1, "READ",  "x", "b"},
		{4, 2, "READ",  "x", "b"},
		{4, 3, "READ",  "x", "b"},
	}

	printTimeline(events, []string{"R1", "R2", "R3"}, 5)

	r1 := NewReplica(1)
	r2 := NewReplica(2)
	r3 := NewReplica(3)
	replicas := map[int]*Replica{1: r1, 2: r2, 3: r3}

	r1.localWrite("x", "a", ts1)
	r2.localWrite("x", "b", ts2)
	fmt.Printf("  R1: W(x)a  ts=%d\n", ts1)
	fmt.Printf("  R2: W(x)b  ts=%d  (newer)\n\n", ts2)

	var log []string
	var mu sync.Mutex
	propagateAsync([]PropagationRecord{
		{1, 2, "x", "a", ts1, 20},
		{1, 3, "x", "a", ts1, 20},
		{2, 1, "x", "b", ts2, 20},
		{2, 3, "x", "b", ts2, 20},
	}, replicas, &log, &mu)

	sort.Strings(log)
	for _, l := range log { fmt.Println(l) }
	fmt.Println()

	fmt.Println("  After LWW resolution:")
	for _, r := range []*Replica{r1, r2, r3} {
		v, ts := r.read("x")
		fmt.Printf("    R%d: x=%s  ts=%d\n", r.id, v, ts)
	}
	fmt.Println()
	fmt.Println("  ✅ All replicas converged to x=b (higher timestamp wins).")
	fmt.Println("  ⚠️  LWW discards x=a permanently — no merge, last write wins.")
	fmt.Println()
}

// ── SCENARIO D: Monotonic Read Violation ─────────────────────────────────────

func scenarioD() {
	sep("═", 72)
	fmt.Println("SCENARIO D — Monotonic Read Violation")
	sep("═", 72)
	fmt.Println()
	fmt.Println("  Client reads x=a from R1 (updated), then reads x=NIL from R2")
	fmt.Println("  (not yet updated). A later read returns an older value — violation.")
	fmt.Println()

	events := []Event{
		{1, 1, "WRITE", "x", "a"},
		{2, 1, "READ",  "x", "a"},     // client reads R1 — sees a
		{3, 2, "READ",  "x", "NIL"},   // client reads R2 — sees NIL ← older!
		{4, 2, "SYNC",  "x", "a"},     // R2 eventually gets update
		{5, 2, "READ",  "x", "a"},     // now monotonic
	}

	printTimeline(events, []string{"R1", "R2", "R3"}, 6)

	r1 := NewReplica(1)
	r2 := NewReplica(2)

	ts := time.Now().UnixNano()
	r1.localWrite("x", "a", ts)

	v1, _ := r1.read("x")
	fmt.Printf("  Client reads R1: R(x)=%s\n", v1)

	v2, _ := r2.read("x")
	fmt.Printf("  Client reads R2: R(x)=%s  ⚠️  Monotonic read VIOLATED — got older value after seeing newer\n", v2)

	r2.applyLWW("x", "a", ts)
	v2after, _ := r2.read("x")
	fmt.Printf("  After propagation → R2: R(x)=%s  ✅ Monotonic reads restored\n", v2after)
	fmt.Println()
	fmt.Println("  Fix: route a client's reads to the same replica (sticky routing),")
	fmt.Println("       or track a read version and reject responses older than it.")
	fmt.Println()
}

// ── SCENARIO E: Convergence race — two updates, same key ─────────────────────

func scenarioE() {
	sep("═", 72)
	fmt.Println("SCENARIO E — Convergence Race  (two sequential writes, same key)")
	sep("═", 72)
	fmt.Println()
	fmt.Println("  P1 writes x=a, then x=b to R1. Propagation may deliver them")
	fmt.Println("  out of order to R2, causing a temporary wrong value.")
	fmt.Println("  LWW (via timestamp) corrects this — newer write always wins.")
	fmt.Println()

	events := []Event{
		{1, 1, "WRITE", "x", "a"},
		{2, 1, "WRITE", "x", "b"},
		{3, 2, "SYNC",  "x", "b"},   // b arrives first (fast path)
		{4, 2, "READ",  "x", "b"},   // R2 reads b ← correct (LWW)
		{5, 2, "SYNC",  "x", "a"},   // a arrives late — rejected by LWW
		{6, 2, "READ",  "x", "b"},   // R2 still b ✅
	}

	printTimeline(events, []string{"R1", "R2", "R3"}, 7)

	r1 := NewReplica(1)
	r2 := NewReplica(2)

	ts1 := time.Now().UnixNano()
	r1.localWrite("x", "a", ts1)
	fmt.Printf("  R1: W(x)a  ts=%d\n", ts1)

	time.Sleep(1 * time.Millisecond)
	ts2 := time.Now().UnixNano()
	r1.localWrite("x", "b", ts2)
	fmt.Printf("  R1: W(x)b  ts=%d  (second write, higher ts)\n\n", ts2)

	// b propagates faster than a
	r2.applyLWW("x", "b", ts2)
	v, _ := r2.read("x")
	fmt.Printf("  R2 receives x=b first: R(x)=%s\n", v)

	// a arrives late — LWW rejects it
	accepted := r2.applyLWW("x", "a", ts1)
	v2, _ := r2.read("x")
	if !accepted {
		fmt.Printf("  R2 receives x=a late: REJECTED by LWW (ts1 < ts2) → R(x)=%s  ✅\n", v2)
	}
	fmt.Println()
}

// ── Convergence summary table ─────────────────────────────────────────────────

func summaryTable() {
	sep("═", 72)
	fmt.Println("SUMMARY — Eventual Consistency Guarantees")
	sep("═", 72)
	fmt.Println()
	fmt.Printf("  %-32s  %s\n", "Property", "Eventual Consistency")
	fmt.Printf("  %-32s  %s\n", strings.Repeat("─", 32), strings.Repeat("─", 32))
	rows := [][]string{
		{"Eventual convergence",       "✅ guaranteed (if writes stop)"},
		{"Read-your-writes",           "✗  not guaranteed"},
		{"Monotonic reads",            "✗  not guaranteed"},
		{"Write ordering",             "✗  no global order"},
		{"Concurrent write resolution","✅ via LWW / vector clocks / CRDTs"},
		{"Availability",               "✅ always accept writes locally"},
		{"Stale reads",                "⚠️  possible until convergence"},
	}
	for _, row := range rows {
		fmt.Printf("  %-32s  %s\n", row[0], row[1])
	}
	fmt.Println()
}

func main() {
	rand.New(rand.NewSource(42))

	sep("═", 72)
	fmt.Println("EVENTUAL CONSISTENCY — Illustration")
	sep("═", 72)
	fmt.Println()
	fmt.Println("  Notation:  W(k)v    = write value v to key k")
	fmt.Println("             R(k)v    = read key k, returns v")
	fmt.Println("             sync(k)v = propagation message arriving at replica")
	fmt.Println()
	fmt.Println("  Three replicas: R1, R2, R3.")
	fmt.Println("  Writes go to one replica and propagate asynchronously.")
	fmt.Println("  LWW = Last-Write-Wins: higher timestamp overwrites lower.")
	fmt.Println()

	scenarioA()
	scenarioB()
	scenarioC()
	scenarioD()
	scenarioE()
	summaryTable()
}
