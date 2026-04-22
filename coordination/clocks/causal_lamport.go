package main

import (
	"fmt"
	"strings"
)

// ─────────────────────────────────────────────
//  LAMPORT CLOCK
// ─────────────────────────────────────────────

type LamportClock struct {
	time int
}

func (c *LamportClock) tick() int {
	c.time++
	return c.time
}

func (c *LamportClock) receive(incoming int) int {
	if incoming > c.time {
		c.time = incoming
	}
	c.time++
	return c.time
}

type Event struct {
	label     string
	process   string
	timestamp int
}

func main() {
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("  LAMPORT CLOCK LIMITATION")
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

	// ── Assign Lamport timestamps ─────────────────────────────

	p1 := &LamportClock{}
	p2 := &LamportClock{}
	p3 := &LamportClock{}

	// P1
	a1 := Event{"a1", "P1", p1.tick()}
	a2 := Event{"a2", "P1", p1.tick()} // a2 sends to P2

	// P2
	b1 := Event{"b1", "P2", p2.tick()}
	b2 := Event{"b2", "P2", p2.receive(a2.timestamp)} // receives from P1
	b3 := Event{"b3", "P2", p2.tick()}                // b3 sends to P3

	// P3
	c1 := Event{"c1", "P3", p3.tick()}
	c2 := Event{"c2", "P3", p3.receive(b3.timestamp)} // receives from P2

	// ── Print timestamps ──────────────────────────────────────
	fmt.Println(strings.Repeat("-", 60))
	fmt.Println("  Lamport Timestamps:")
	fmt.Println(strings.Repeat("-", 60))
	for _, e := range []Event{a1, a2, b1, b2, b3, c1, c2} {
		fmt.Printf("    C(%s) = %d\n", e.label, e.timestamp)
	}

	// ── Causality queries ─────────────────────────────────────
	fmt.Println()
	fmt.Println(strings.Repeat("-", 60))
	fmt.Println("  Causality Queries:")
	fmt.Println(strings.Repeat("-", 60))

	type query struct {
		e           Event
		f           Event
		trulyBefore bool   // ground truth from the execution diagram
		reason      string // explanation
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
		lamportSays := q.e.timestamp < q.f.timestamp
		correct := lamportSays == q.trulyBefore

		trueStr := "concurrent ∥"
		if q.trulyBefore {
			trueStr = "causal     →"
		}

		verdict := "✅"
		if !correct {
			verdict = "❌ WRONG"
		}

		fmt.Printf("\n  C(%s)=%-2d  vs  C(%s)=%-2d\n",
			q.e.label, q.e.timestamp,
			q.f.label, q.f.timestamp)
		fmt.Printf("    Truth         : %s  %s and %s  (%s)\n",
			trueStr, q.e.label, q.f.label, q.reason)
		fmt.Printf("    Lamport says  : C(%s) < C(%s) → %s  %s\n",
			q.e.label, q.f.label,
			map[bool]string{true: "happened-before", false: "NOT happened-before"}[lamportSays],
			verdict)
	}

	// ── Summary ───────────────────────────────────────────────
	fmt.Println()
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("  CONCLUSION")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println(`
  Lamport guarantees:

    e → f   implies   C(e) < C(f)          ✅

  But NOT the converse:

    C(e) < C(f)  does NOT imply  e → f     ✗

  From the queries above:

    C(b1) < C(c2)  —  looks causal, but b1 ∥ c2  ❌
    C(a1) < C(c1)  —  looks causal, but a1 ∥ c1  ❌
    C(a2) < C(b1)  —  looks causal, but a2 ∥ b1  ❌

  A lower timestamp only means an event COULD have influenced
  another. It cannot confirm or deny an actual causal link.`)
}
