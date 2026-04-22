package main

import (
	"fmt"
	"strings"
)

// ─────────────────────────────────────────────
//  VECTOR CLOCK
// ─────────────────────────────────────────────

const numProcesses = 3

type VectorClock [numProcesses]int

// tick increments this process's own slot
func (vc *VectorClock) tick(pid int) {
	vc[pid]++
}

// receive merges incoming vector then ticks own slot
func (vc *VectorClock) receive(pid int, incoming VectorClock) {
	for i := range vc {
		if incoming[i] > vc[i] {
			vc[i] = incoming[i]
		}
	}
	vc[pid]++
}

func (vc VectorClock) String() string {
	return fmt.Sprintf("[P1=%d, P2=%d, P3=%d]", vc[0], vc[1], vc[2])
}

// ─────────────────────────────────────────────
//  CAUSALITY CHECKS
// ─────────────────────────────────────────────

// leq returns true if every slot of a <= b
func leq(a, b VectorClock) bool {
	for i := range a {
		if a[i] > b[i] {
			return false
		}
	}
	return true
}

// happensBefore: a → b iff a ≤ b and a ≠ b
func happensBefore(a, b VectorClock) bool {
	return leq(a, b) && a != b
}

// concurrent: a ∥ b iff neither a → b nor b → a
func concurrent(a, b VectorClock) bool {
	return !happensBefore(a, b) && !happensBefore(b, a)
}

// ─────────────────────────────────────────────
//  EVENT
// ─────────────────────────────────────────────

type Event struct {
	label   string
	process string
	vc      VectorClock
}

func main() {
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("  VECTOR CLOCK")
	fmt.Println(strings.Repeat("=", 60))

	fmt.Println(`
  Execution:

    P1:  a1 ──── a2 ──────────────────────────
                   \
    P2:  b1 ─────── b2 ──── b3 ──────────────
                                 \
    P3:  c1 ────────────────────── c2 ────────

    a2 → b2  (P1 sends message to P2)
    b3 → c2  (P2 sends message to P3)
`)

	// ── Assign vector clock timestamps ────────────────────────
	// Process indices: P1=0, P2=1, P3=2

	var p1, p2, p3 VectorClock

	// P1
	p1.tick(0)
	a1 := Event{"a1", "P1", p1}

	p1.tick(0)
	a2 := Event{"a2", "P1", p1} // a2 sends to P2

	// P2
	p2.tick(1)
	b1 := Event{"b1", "P2", p2}

	p2.receive(1, a2.vc) // absorb a2's vector then tick
	b2 := Event{"b2", "P2", p2}

	p2.tick(1)
	b3 := Event{"b3", "P2", p2} // b3 sends to P3

	// P3
	p3.tick(2)
	c1 := Event{"c1", "P3", p3}

	p3.receive(2, b3.vc) // absorb b3's vector then tick
	c2 := Event{"c2", "P3", p3}

	// ── Print vector clocks ───────────────────────────────────
	fmt.Println(strings.Repeat("-", 60))
	fmt.Println("  Vector Clock Timestamps:")
	fmt.Println(strings.Repeat("-", 60))
	for _, e := range []Event{a1, a2, b1, b2, b3, c1, c2} {
		fmt.Printf("    VC(%s) = %s\n", e.label, e.vc)
	}

	// ── Causality queries ─────────────────────────────────────
	fmt.Println()
	fmt.Println(strings.Repeat("-", 60))
	fmt.Println("  Causality Queries:")
	fmt.Println(strings.Repeat("-", 60))

	type query struct {
		e           Event
		f           Event
		trulyBefore bool
		reason      string
	}

	queries := []query{
		{a1, a2, true, "same process, sequential"},
		{a1, c2, true, "a1→a2→b2→b3→c2 by transitivity"},
		{a2, c2, true, "a2→b2→b3→c2 by transitivity"},
		{b3, c2, true, "b3 sent the message c2 received"},
		{b1, c1, false, "different processes, no message between them"},
		{a1, c1, false, "c1 has no knowledge of a1"},
		{a2, b1, false, "b1 occurred before b2 absorbed a2"},
	}

	for _, q := range queries {
		hb := happensBefore(q.e.vc, q.f.vc)
		con := concurrent(q.e.vc, q.f.vc)

		trueStr := "concurrent ∥"
		if q.trulyBefore {
			trueStr = "causal     →"
		}

		vcSays := "concurrent ∥"
		if hb {
			vcSays = "causal     →"
		} else if con {
			vcSays = "concurrent ∥"
		}

		correct := hb == q.trulyBefore
		verdict := "✅"
		if !correct {
			verdict = "❌ WRONG"
		}

		fmt.Printf("\n  VC(%s)=%s\n  VC(%s)=%s\n",
			q.e.label, q.e.vc,
			q.f.label, q.f.vc)
		fmt.Printf("    Truth        : %s  %s and %s  (%s)\n",
			trueStr, q.e.label, q.f.label, q.reason)
		fmt.Printf("    Vector says  : %s  %s\n", vcSays, verdict)
	}

	// ── Summary ───────────────────────────────────────────────
	fmt.Println()
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("  CONCLUSION")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println(`
  Vector clock rule:

    e → f   iff   VC(e) ≤ VC(f) per slot  and  VC(e) ≠ VC(f)  ✅
    e ∥ f   iff   neither VC(e) ≤ VC(f) nor VC(f) ≤ VC(e)     ✅

  Unlike Lamport clocks, vector clocks correctly identify
  every causal and concurrent pair — no false orderings.`)
}
