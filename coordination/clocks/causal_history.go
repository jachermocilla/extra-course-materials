package main

import (
	"fmt"
	"sort"
	"strings"
)

// ─────────────────────────────────────────────
//  CAUSAL HISTORY
// ─────────────────────────────────────────────

type History map[string]bool

func newHistory(self string) History {
	h := History{}
	h[self] = true
	return h
}

// merge absorbs another history into this one (union)
func (h History) merge(other History) {
	for e := range other {
		h[e] = true
	}
}

func (h History) contains(event string) bool {
	return h[event]
}

func (h History) sorted() []string {
	keys := make([]string, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (h History) String() string {
	return "{ " + strings.Join(h.sorted(), ", ") + " }"
}

// ─────────────────────────────────────────────
//  CAUSALITY QUERIES
// ─────────────────────────────────────────────

// happensBefore: e → f iff e ∈ H(f) and e ≠ f
func happensBefore(e string, hf History) bool {
	return hf.contains(e)
}

// concurrent: e ∥ f iff e ∉ H(f) and f ∉ H(e)
func concurrent(e string, he History, f string, hf History) bool {
	return !hf.contains(e) && !he.contains(f)
}

// ─────────────────────────────────────────────
//  MAIN
// ─────────────────────────────────────────────

func main() {
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("  CAUSAL HISTORIES")
	fmt.Println(strings.Repeat("=", 60))

	// ── Execution diagram ─────────────────────────────────────
	//
	//  P1:  a1 ──── a2 ──────────────────────────────
	//                 \
	//  P2:  b1 ─────── b2 ──── b3 ──────────────────
	//                               \
	//  P3:  c1 ────────────────────── c2 ────────────
	//
	//  a2 → b2  (P1 sends to P2)
	//  b3 → c2  (P2 sends to P3)

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

	// ── Build causal histories event by event ─────────────────

	// P1 events
	ha1 := newHistory("a1")
	ha2 := newHistory("a2")
	ha2.merge(ha1) // a1 → a2 (same process, sequential)

	// P2 events
	hb1 := newHistory("b1")
	hb2 := newHistory("b2")
	hb2.merge(hb1)  // b1 → b2 (same process)
	hb2.merge(ha2)  // absorb a2's history on receive from P1

	hb3 := newHistory("b3")
	hb3.merge(hb2) // b2 → b3 (same process)

	// P3 events
	hc1 := newHistory("c1")
	hc2 := newHistory("c2")
	hc2.merge(hc1)  // c1 → c2 (same process)
	hc2.merge(hb3)  // absorb b3's history on receive from P2

	// ── Print histories ───────────────────────────────────────
	fmt.Println(strings.Repeat("-", 60))
	fmt.Println("  Causal Histories:")
	fmt.Println(strings.Repeat("-", 60))

	events := []struct {
		label   string
		history History
	}{
		{"a1", ha1}, {"a2", ha2},
		{"b1", hb1}, {"b2", hb2}, {"b3", hb3},
		{"c1", hc1}, {"c2", hc2},
	}

	for _, e := range events {
		fmt.Printf("  H(%s) = %s\n", e.label, e.history)
	}

	// ── Causality queries ─────────────────────────────────────
	fmt.Println()
	fmt.Println(strings.Repeat("-", 60))
	fmt.Println("  Causality Queries:")
	fmt.Println(strings.Repeat("-", 60))

	type query struct {
		e  string
		he History
		f  string
		hf History
	}

	queries := []query{
		{"a1", ha1, "a2", ha2},
		{"a1", ha1, "c2", hc2},
		{"a2", ha2, "c2", hc2},
		{"b3", hb3, "c2", hc2},
		{"b1", hb1, "c1", hc1}, // concurrent: different process, no message
		{"a1", ha1, "c1", hc1}, // concurrent: c1 has no knowledge of a1
		{"a2", ha2, "b1", hb1}, // b1 happened before b2 absorbed a2
	}

	for _, q := range queries {
		fmt.Printf("\n  e=%s  f=%s\n", q.e, q.f)
		if happensBefore(q.e, q.hf) {
			fmt.Printf("    %s → %s  ✅  (%s ∈ H(%s))\n", q.e, q.f, q.e, q.f)
		} else if concurrent(q.e, q.he, q.f, q.hf) {
			fmt.Printf("    %s ∥ %s  〰  (%s ∉ H(%s) and %s ∉ H(%s))\n",
				q.e, q.f, q.e, q.f, q.f, q.e)
		} else {
			fmt.Printf("    %s → %s  ✅  (%s ∈ H(%s))\n", q.f, q.e, q.f, q.e)
		}
	}

	fmt.Println()
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("  CONCLUSION")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println(`
  H(e) grows by two rules:

    1. Same process  : H(eN) = H(eN-1) ∪ { eN }
    2. Message recv  : H(recv) = H(send) ∪ H(prev) ∪ { recv }

  Causality check is plain set membership:

    e → f   iff   e ∈ H(f)
    e ∥ f   iff   e ∉ H(f)  and  f ∉ H(e)

  No numeric comparison, no ambiguity.`)
}
