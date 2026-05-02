package main

import (
	"fmt"
	"strings"
)

// ============================================================
// Causal Consistency — illustration using the timeline notation
//
// Three scenarios:
//
// A) CAUSALLY CONSISTENT — concurrent writes, different order OK
//    W1(x)a and W2(y)b are concurrent (neither caused the other).
//    P3 sees them as a,b and P4 sees them as b,a — allowed.
//
// B) CAUSAL VIOLATION — causally related writes seen out of order
//    P1 writes x=a.  P2 reads x=a, THEN writes y=b (caused by x=a).
//    P3 must see W1(x)a before W2(y)b.
//    If P3 sees W2(y)b first (without having seen W1(x)a) — violation.
//
// C) CAUSAL CHAIN — three-hop causality
//    P1: W(x)a
//    P2: reads x=a, then W(y)b   (caused by W1(x)a)
//    P3: reads y=b, then W(z)c   (caused by W2(y)b, which was caused by W1(x)a)
//    All processes must see x=a before y=b before z=c.
// ============================================================

// ── Types ────────────────────────────────────────────────────────────────────

type OpKind string

const (
	Write OpKind = "W"
	Read  OpKind = "R"
)

type Op struct {
	slot    int
	pid     int
	kind    OpKind
	varName string
	value   string
}

func w(slot, pid int, varName, value string) Op {
	return Op{slot, pid, Write, varName, value}
}
func r(slot, pid int, varName, value string) Op {
	return Op{slot, pid, Read, varName, value}
}

func (o Op) label() string {
	return fmt.Sprintf("%s%d(%s)%s", o.kind, o.pid, o.varName, o.value)
}

// ── Timeline printer ─────────────────────────────────────────────────────────

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

func centerPad(s string, width int) string {
	if len(s) >= width { return s }
	total := width - len(s)
	left := total / 2
	return strings.Repeat("─", left) + s + strings.Repeat("─", total-left)
}

func sep(char string, n int) { fmt.Println(strings.Repeat(char, n)) }

// ── Causal dependency checker ─────────────────────────────────────────────────
// A write B is causally dependent on write A if:
//   1. Same process: A appears before B in program order, OR
//   2. Cross-process: some process read A's value and then issued B
//
// We represent causal deps as: dep[B.label] = A.label (B depends on A)

type CausalOrder struct {
	deps map[string]string // child → parent
}

func NewCausalOrder() *CausalOrder {
	return &CausalOrder{deps: make(map[string]string)}
}

func (c *CausalOrder) AddDep(child, parent string) {
	c.deps[child] = parent
}

// isCausallyBefore returns true if a must be seen before b
func (c *CausalOrder) isCausallyBefore(a, b string) bool {
	cur := b
	for {
		parent, ok := c.deps[cur]
		if !ok { return false }
		if parent == a { return true }
		cur = parent
	}
}

// checkObservation checks if a process's observed write order
// violates any causal dependency
func (c *CausalOrder) checkObservation(pid int, observed []string) (bool, string) {
	posOf := make(map[string]int)
	for i, label := range observed {
		posOf[label] = i
	}
	for child, parent := range c.deps {
		pc, childOk := posOf[child]
		pp, parentOk := posOf[parent]
		if childOk && parentOk && pp >= pc {
			return false, fmt.Sprintf("P%d saw %s before %s — but %s causally depends on %s",
				pid, child, parent, child, parent)
		}
	}
	return true, ""
}

// ── SCENARIO A: Causally Consistent — Concurrent Writes ──────────────────────
//
// P1 and P2 write concurrently (no causal link between them).
// P3 and P4 are readers.
// P3 sees: W1(x)a, W2(y)b  (order: x first)
// P4 sees: W2(y)b, W1(x)a  (order: y first)
// This is ALLOWED — no causal relationship between the two writes.
//
// Timeline:
//   P1  W1(x)a
//   P2         W2(y)b
//   P3                  R3(x)a  R3(y)b   (sees x before y)
//   P4                  R4(y)b  R4(x)a   (sees y before x — concurrent, OK)

func scenarioA() {
	sep("═", 72)
	fmt.Println("SCENARIO A — Causally Consistent  (concurrent writes, order may differ)")
	sep("═", 72)
	fmt.Println()
	fmt.Println("  W1(x)a and W2(y)b are CONCURRENT — neither caused the other.")
	fmt.Println("  P3 and P4 may observe them in different orders. This is allowed.")
	fmt.Println()

	ops := []Op{
		w(1, 1, "x", "a"), // P1 writes x=a  (concurrent with P2)
		w(2, 2, "y", "b"), // P2 writes y=b  (concurrent with P1)
		r(3, 3, "x", "a"), // P3 reads x=a first ...
		r(4, 3, "y", "b"), // ... then y=b  (P3 sees: x,y order)
		r(3, 4, "y", "b"), // P4 reads y=b first ...
		r(4, 4, "x", "a"), // ... then x=a  (P4 sees: y,x order)
	}

	printTimeline(ops, 4, 5)

	fmt.Println("  Causal analysis:")
	fmt.Println("  · W1(x)a and W2(y)b are concurrent writes — no causal link.")
	fmt.Println("  · P3 observes order: W1(x)a → W2(y)b")
	fmt.Println("  · P4 observes order: W2(y)b → W1(x)a")
	fmt.Println()

	co := NewCausalOrder()
	// no causal deps between W1(x)a and W2(y)b

	p3obs := []string{"W1(x)a", "W2(y)b"}
	p4obs := []string{"W2(y)b", "W1(x)a"}

	ok3, msg3 := co.checkObservation(3, p3obs)
	ok4, msg4 := co.checkObservation(4, p4obs)

	printCheck(3, p3obs, ok3, msg3)
	printCheck(4, p4obs, ok4, msg4)

	if ok3 && ok4 {
		fmt.Println("  ✅ CAUSALLY CONSISTENT — disagreement on concurrent write order is permitted.")
	}
	fmt.Println()
}

// ── SCENARIO B: Causal Violation ─────────────────────────────────────────────
//
// P1 writes x=a.
// P2 reads x=a, then writes y=b  (y=b is CAUSED BY x=a).
// Causal order: W1(x)a →causes→ W2(y)b
//
// P3 must see W1(x)a before W2(y)b.
// If P3 sees W2(y)b before W1(x)a — VIOLATION.
//
// Timeline (consistent view — P3 sees correct order):
//   P1  W1(x)a
//   P2         R2(x)a  W2(y)b
//   P3                         R3(x)a  R3(y)b   ← correct causal order
//   P4                         R4(y)b  R4(x)?   ← violation: y before x

func scenarioB() {
	sep("═", 72)
	fmt.Println("SCENARIO B — Causal Violation  (causally related writes seen out of order)")
	sep("═", 72)
	fmt.Println()
	fmt.Println("  P2 reads x=a THEN writes y=b — W2(y)b is causally dependent on W1(x)a.")
	fmt.Println("  Every process MUST see W1(x)a before W2(y)b.")
	fmt.Println()

	// Show the consistent view for P3
	opsConsistent := []Op{
		w(1, 1, "x", "a"), // P1 writes x=a
		r(2, 2, "x", "a"), // P2 reads x=a  (establishes causal link)
		w(3, 2, "y", "b"), // P2 writes y=b  (CAUSED by reading x=a)
		r(4, 3, "x", "a"), // P3 reads x=a first — correct order
		r(5, 3, "y", "b"), // P3 then reads y=b
	}

	fmt.Println("  Consistent observation (P3 sees causal order correctly):")
	printTimeline(opsConsistent, 3, 6)

	// Show the violation for P4 — sees y=b before x=a
	opsViolation := []Op{
		w(1, 1, "x", "a"),
		r(2, 2, "x", "a"),
		w(3, 2, "y", "b"),
		r(4, 4, "y", "b"), // P4 sees y=b FIRST — violation!
		r(5, 4, "x", "a"), // P4 then sees x=a — but too late
	}

	fmt.Println("  Violation observation (P4 sees effect before cause):")
	printTimeline(opsViolation, 4, 6)

	co := NewCausalOrder()
	co.AddDep("W2(y)b", "W1(x)a") // W2(y)b causally depends on W1(x)a

	fmt.Println("  Causal dependency: W2(y)b depends on W1(x)a")
	fmt.Println("  Rule: any process that sees W2(y)b must have already seen W1(x)a")
	fmt.Println()

	p3obs := []string{"W1(x)a", "W2(y)b"}
	p4obs := []string{"W2(y)b", "W1(x)a"} // reversed — violation

	ok3, msg3 := co.checkObservation(3, p3obs)
	ok4, msg4 := co.checkObservation(4, p4obs)

	printCheck(3, p3obs, ok3, msg3)
	printCheck(4, p4obs, ok4, msg4)

	if !ok4 {
		fmt.Println("  ⚠️  CAUSAL VIOLATION — P4 sees the effect (y=b) before its cause (x=a).")
		fmt.Println("      In a causally consistent store, W2(y)b would be buffered at P4")
		fmt.Println("      until W1(x)a has been delivered.")
	}
	fmt.Println()
}

// ── SCENARIO C: Causal Chain ─────────────────────────────────────────────────
//
// Three-hop causal chain:
//   P1: W(x)a
//   P2: R(x)a → W(y)b   (y=b caused by x=a)
//   P3: R(y)b → W(z)c   (z=c caused by y=b, which was caused by x=a)
//
// Every process must observe: W1(x)a → W2(y)b → W3(z)c in that order.
// Any permutation that breaks this chain is a violation.
//
// Concurrent writes W1(x)a and W4(w)d (from a different process with
// no causal link to the chain) may be observed in any order.

func scenarioC() {
	sep("═", 72)
	fmt.Println("SCENARIO C — Causal Chain + Concurrent Write")
	sep("═", 72)
	fmt.Println()
	fmt.Println("  Causal chain:  W1(x)a → W2(y)b → W3(z)c")
	fmt.Println("  Concurrent  :  W4(w)d  (no causal link to the chain)")
	fmt.Println()

	ops := []Op{
		w(1, 1, "x", "a"), // P1 writes x=a
		r(2, 2, "x", "a"), // P2 reads x=a
		w(2, 4, "w", "d"), // P4 writes w=d  — concurrent, no causal link
		w(3, 2, "y", "b"), // P2 writes y=b  (caused by x=a)
		r(4, 3, "y", "b"), // P3 reads y=b
		w(5, 3, "z", "c"), // P3 writes z=c  (caused by y=b → x=a)
	}

	printTimeline(ops, 4, 6)

	co := NewCausalOrder()
	co.AddDep("W2(y)b", "W1(x)a") // y=b caused by x=a
	co.AddDep("W3(z)c", "W2(y)b") // z=c caused by y=b

	fmt.Println("  Causal chain:  W1(x)a → W2(y)b → W3(z)c")
	fmt.Println("  W4(w)d is concurrent with all — no causal order required.")
	fmt.Println()

	type observation struct {
		pid     int
		writes  []string
		note    string
	}

	observations := []observation{
		{5, []string{"W1(x)a", "W2(y)b", "W3(z)c", "W4(w)d"}, "chain in order + concurrent at end"},
		{5, []string{"W4(w)d", "W1(x)a", "W2(y)b", "W3(z)c"}, "chain in order + concurrent at start"},
		{5, []string{"W1(x)a", "W4(w)d", "W2(y)b", "W3(z)c"}, "concurrent write interleaved (allowed)"},
		{5, []string{"W2(y)b", "W1(x)a", "W3(z)c", "W4(w)d"}, "⚠️  effect before cause (violation)"},
		{5, []string{"W1(x)a", "W3(z)c", "W2(y)b", "W4(w)d"}, "⚠️  z before y (violation)"},
	}

	fmt.Printf("  %-3s  %-44s  %-6s  %s\n", "Obs", "Observed write order", "Valid?", "Note")
	fmt.Printf("  %-3s  %-44s  %-6s  %s\n",
		"───", strings.Repeat("─", 44), "──────", strings.Repeat("─", 38))

	for i, obs := range observations {
		ok, _ := co.checkObservation(obs.pid, obs.writes)
		sym := "✅ yes"
		if !ok { sym = "✗  NO " }
		fmt.Printf("  %-3d  %-44s  %-6s  %s\n",
			i+1, strings.Join(obs.writes, " → "), sym, obs.note)
	}
	fmt.Println()
	fmt.Println("  Key: W4(w)d can appear anywhere relative to the chain — it is")
	fmt.Println("  concurrent. But W1(x)a → W2(y)b → W3(z)c must always be")
	fmt.Println("  preserved in exactly that order on every process.")
	fmt.Println()
}

// ── SCENARIO D: Concurrent Writes, Same Variable ─────────────────────────────
//
// Two processes write to the SAME variable x concurrently.
// Under causal consistency (unlike sequential consistency),
// different processes may see different final values of x —
// there is no required total order on concurrent writes.
//
// P1: W1(x)a   P2: W2(x)b   (concurrent)
// P3 sees: a then b → final x=b
// P4 sees: b then a → final x=a
// This is causally consistent but NOT sequentially consistent.

func scenarioD() {
	sep("═", 72)
	fmt.Println("SCENARIO D — Concurrent Writes to Same Variable (x)")
	sep("═", 72)
	fmt.Println()
	fmt.Println("  W1(x)a and W2(x)b are concurrent — no causal link.")
	fmt.Println("  Causal consistency allows P3 and P4 to disagree on final x.")
	fmt.Println("  (Sequential consistency would require them to agree.)")
	fmt.Println()

	opsP3 := []Op{
		w(1, 1, "x", "a"),
		w(2, 2, "x", "b"),
		r(3, 3, "x", "a"), // P3 sees a first
		r(4, 3, "x", "b"), // then b
	}
	opsP4 := []Op{
		w(1, 1, "x", "a"),
		w(2, 2, "x", "b"),
		r(3, 4, "x", "b"), // P4 sees b first
		r(4, 4, "x", "a"), // then a
	}

	fmt.Println("  P3 view:")
	printTimeline(opsP3, 3, 5)
	fmt.Println("  P4 view:")
	printTimeline(opsP4, 4, 5)

	co := NewCausalOrder()
	// no causal deps — writes are concurrent

	p3obs := []string{"W1(x)a", "W2(x)b"}
	p4obs := []string{"W2(x)b", "W1(x)a"}

	ok3, msg3 := co.checkObservation(3, p3obs)
	ok4, msg4 := co.checkObservation(4, p4obs)

	printCheck(3, p3obs, ok3, msg3)
	printCheck(4, p4obs, ok4, msg4)

	fmt.Println()
	fmt.Println("  ✅ CAUSALLY CONSISTENT — concurrent writes may be observed in any order.")
	fmt.Println("  ⚠️  NOT SEQUENTIALLY CONSISTENT — P3 and P4 disagree on write order of x.")
	fmt.Println()
}

// ── helpers ──────────────────────────────────────────────────────────────────

func printCheck(pid int, obs []string, ok bool, msg string) {
	sym := "✅"
	if !ok { sym = "✗ " }
	fmt.Printf("  P%d observed: %s  %s\n", pid, strings.Join(obs, " → "), sym)
	if !ok && msg != "" {
		fmt.Printf("     → %s\n", msg)
	}
}

// ── Main ─────────────────────────────────────────────────────────────────────

func main() {
	sep("═", 72)
	fmt.Println("CAUSAL CONSISTENCY — Illustration")
	sep("═", 72)
	fmt.Println()
	fmt.Println("  Rule 1: Causally related writes must be seen in causal order")
	fmt.Println("          by ALL processes.")
	fmt.Println("  Rule 2: Concurrent writes (no causal link) may be seen in")
	fmt.Println("          ANY order — processes may disagree.")
	fmt.Println("  Rule 3: Within one process, program order is always preserved.")
	fmt.Println()
	fmt.Println("  Notation:  Wi(v)x = process i writes value x to variable v")
	fmt.Println("             Ri(v)x = process i reads value x from variable v")
	fmt.Println("  →causes→   read-then-write in the same process establishes")
	fmt.Println("             a causal dependency on what was read.")
	fmt.Println()

	scenarioA()
	scenarioB()
	scenarioC()
	scenarioD()

	sep("═", 72)
	fmt.Println("SUMMARY")
	sep("═", 72)
	fmt.Println()
	fmt.Printf("  %-12s  %-22s  %-22s  %s\n",
		"Scenario", "Causal?", "Sequentially consistent?", "Why")
	fmt.Printf("  %-12s  %-22s  %-22s  %s\n",
		"────────────", "──────────────────────", "──────────────────────", "──────────────────────────────")
	fmt.Printf("  %-12s  %-22s  %-22s  %s\n", "A",
		"✅ yes", "✅ yes (diff vars)",
		"Concurrent writes, different variables")
	fmt.Printf("  %-12s  %-22s  %-22s  %s\n", "B (P3)",
		"✅ yes", "✅ yes",
		"Causal order respected")
	fmt.Printf("  %-12s  %-22s  %-22s  %s\n", "B (P4)",
		"✗  NO", "✗  NO",
		"Effect seen before cause")
	fmt.Printf("  %-12s  %-22s  %-22s  %s\n", "C",
		"✅ yes (chain)", "✅ yes (chain)",
		"Three-hop chain preserved")
	fmt.Printf("  %-12s  %-22s  %-22s  %s\n", "D",
		"✅ yes", "✗  NO",
		"Concurrent writes to same var: CC allows disagreement")
	fmt.Println()
}
