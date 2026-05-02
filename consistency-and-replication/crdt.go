package main

import (
	"fmt"
	"sort"
	"strings"
)

// ============================================================
// CRDT — Conflict-free Replicated Data Types
//
// Each CRDT has:
//   · A local update operation
//   · A merge(state_a, state_b) that is:
//       commutative  : merge(a,b) == merge(b,a)
//       associative  : merge(a,merge(b,c)) == merge(merge(a,b),c)
//       idempotent   : merge(a,a) == a
//
// Three replicas R1, R2, R3 operate independently, then merge.
// We show timelines and final state for each CRDT type.
// ============================================================

func sep(char string, n int) { fmt.Println(strings.Repeat(char, n)) }

func centerPad(s string, width int) string {
	if len(s) >= width { return s }
	total := width - len(s)
	left := total / 2
	return strings.Repeat("─", left) + s + strings.Repeat("─", total-left)
}

type Op struct {
	slot    int
	replica int
	label   string
}

func printTimeline(ops []Op, numReplicas, totalSlots int) {
	colW := 14
	dash := strings.Repeat("─", colW)
	hdr := fmt.Sprintf("  %-4s  ", "")
	for s := 1; s <= totalSlots; s++ {
		hdr += centerPad(fmt.Sprintf("t%d", s), colW)
	}
	fmt.Println(hdr)
	fmt.Printf("  %-4s  %s\n", "", strings.Repeat("─", colW*totalSlots))
	for r := 1; r <= numReplicas; r++ {
		row := fmt.Sprintf("  R%-3d  ", r)
		for s := 1; s <= totalSlots; s++ {
			cell := ""
			for _, op := range ops {
				if op.replica == r && op.slot == s {
					cell = op.label
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

// ── 1. G-Counter (Grow-only Counter) ─────────────────────────────────────────
// Each replica owns one slot in a vector.
// Increment only adds to the replica's own slot.
// Merge = component-wise max.
// Value = sum of all slots.

type GCounter struct {
	id    int
	slots [3]int // one slot per replica (0-indexed)
}

func NewGCounter(id int) *GCounter { return &GCounter{id: id} }

func (c *GCounter) Increment(n int) {
	c.slots[c.id-1] += n
}

func (c *GCounter) Value() int {
	total := 0
	for _, v := range c.slots { total += v }
	return total
}

func (c *GCounter) Merge(other *GCounter) {
	for i := range c.slots {
		if other.slots[i] > c.slots[i] {
			c.slots[i] = other.slots[i]
		}
	}
}

func (c *GCounter) String() string {
	return fmt.Sprintf("slots%v value=%d", c.slots, c.Value())
}

func demoGCounter() {
	sep("═", 72)
	fmt.Println("CRDT 1 — G-Counter (Grow-only Counter)")
	sep("═", 72)
	fmt.Println()
	fmt.Println("  Each replica owns one slot. Merge = component-wise max.")
	fmt.Println("  Value = sum of all slots. No coordination needed.")
	fmt.Println()

	ops := []Op{
		{1, 1, "inc(3)"},
		{1, 2, "inc(5)"},
		{2, 1, "inc(2)"},
		{2, 3, "inc(4)"},
		{3, 1, "merge(R2)"},
		{3, 2, "merge(R3)"},
		{3, 3, "merge(R1)"},
		{4, 1, "val=10"},
		{4, 2, "val=10"},
		{4, 3, "val=10"},
	}
	printTimeline(ops, 3, 5)

	r1 := NewGCounter(1)
	r2 := NewGCounter(2)
	r3 := NewGCounter(3)

	r1.Increment(3)
	r1.Increment(2) // total from R1: 5
	r2.Increment(5) // total from R2: 5
	r3.Increment(4) // total from R3: 4

	fmt.Printf("  Before merge:\n")
	fmt.Printf("    R1: %s\n", r1)
	fmt.Printf("    R2: %s\n", r2)
	fmt.Printf("    R3: %s\n\n", r3)

	// each merges with the others
	r1.Merge(r2); r1.Merge(r3)
	r2.Merge(r1); r2.Merge(r3)
	r3.Merge(r1); r3.Merge(r2)

	fmt.Printf("  After merge:\n")
	fmt.Printf("    R1: %s\n", r1)
	fmt.Printf("    R2: %s\n", r2)
	fmt.Printf("    R3: %s\n\n", r3)
	fmt.Println("  ✅ All replicas converged to value=14 (3+2+5+4)")
	fmt.Println("  Merge is commutative/associative/idempotent — order irrelevant.")
	fmt.Println()
}

// ── 2. PN-Counter (Increment + Decrement) ─────────────────────────────────────
// Two G-Counters: P (increments) and N (decrements).
// Value = P.Value() - N.Value()

type PNCounter struct {
	id int
	P  GCounter
	N  GCounter
}

func NewPNCounter(id int) *PNCounter {
	return &PNCounter{id: id, P: GCounter{id: id}, N: GCounter{id: id}}
}

func (c *PNCounter) Increment(n int) { c.P.slots[c.id-1] += n }
func (c *PNCounter) Decrement(n int) { c.N.slots[c.id-1] += n }
func (c *PNCounter) Value() int      { return c.P.Value() - c.N.Value() }

func (c *PNCounter) Merge(other *PNCounter) {
	c.P.Merge(&other.P)
	c.N.Merge(&other.N)
}

func (c *PNCounter) String() string {
	return fmt.Sprintf("P%v N%v value=%d", c.P.slots, c.N.slots, c.Value())
}

func demoPNCounter() {
	sep("═", 72)
	fmt.Println("CRDT 2 — PN-Counter (Increment and Decrement)")
	sep("═", 72)
	fmt.Println()
	fmt.Println("  P G-Counter tracks increments, N G-Counter tracks decrements.")
	fmt.Println("  Value = P.sum - N.sum. Merge each independently.")
	fmt.Println()

	ops := []Op{
		{1, 1, "inc(10)"},
		{1, 2, "inc(5)"},
		{2, 1, "dec(3)"},
		{2, 3, "inc(2)"},
		{3, 1, "merge(R2,R3)"},
		{3, 2, "merge(R1,R3)"},
		{3, 3, "merge(R1,R2)"},
		{4, 1, "val=14"},
		{4, 2, "val=14"},
		{4, 3, "val=14"},
	}
	printTimeline(ops, 3, 5)

	r1 := NewPNCounter(1)
	r2 := NewPNCounter(2)
	r3 := NewPNCounter(3)

	r1.Increment(10)
	r1.Decrement(3)
	r2.Increment(5)
	r3.Increment(2)

	fmt.Printf("  Before merge:\n")
	fmt.Printf("    R1: %s\n", r1)
	fmt.Printf("    R2: %s\n", r2)
	fmt.Printf("    R3: %s\n\n", r3)

	r1.Merge(r2); r1.Merge(r3)
	r2.Merge(r1); r2.Merge(r3)
	r3.Merge(r1); r3.Merge(r2)

	fmt.Printf("  After merge:\n")
	fmt.Printf("    R1: %s\n", r1)
	fmt.Printf("    R2: %s\n", r2)
	fmt.Printf("    R3: %s\n\n", r3)
	fmt.Println("  ✅ All replicas converged to value=14  (10+5+2 - 3)")
	fmt.Println()
}

// ── 3. G-Set (Grow-only Set) ──────────────────────────────────────────────────
// Add-only set. Merge = union. Elements can never be removed.

type GSet struct {
	id      int
	members map[string]bool
}

func NewGSet(id int) *GSet { return &GSet{id: id, members: make(map[string]bool)} }

func (s *GSet) Add(elem string)          { s.members[elem] = true }
func (s *GSet) Contains(elem string) bool { return s.members[elem] }
func (s *GSet) Elements() []string {
	var elems []string
	for k := range s.members { elems = append(elems, k) }
	sort.Strings(elems)
	return elems
}
func (s *GSet) Merge(other *GSet) {
	for k := range other.members { s.members[k] = true }
}
func (s *GSet) String() string { return fmt.Sprintf("%v", s.Elements()) }

func demoGSet() {
	sep("═", 72)
	fmt.Println("CRDT 3 — G-Set (Grow-only Set)")
	sep("═", 72)
	fmt.Println()
	fmt.Println("  Elements can only be added. Merge = union.")
	fmt.Println()

	ops := []Op{
		{1, 1, "add(alice)"},
		{1, 2, "add(bob)"},
		{2, 1, "add(carol)"},
		{2, 3, "add(bob)"},   // duplicate add — idempotent
		{3, 1, "merge(R2,R3)"},
		{3, 2, "merge(R1,R3)"},
		{3, 3, "merge(R1,R2)"},
		{4, 1, "{alice,bob,carol}"},
		{4, 2, "{alice,bob,carol}"},
		{4, 3, "{alice,bob,carol}"},
	}
	printTimeline(ops, 3, 5)

	r1 := NewGSet(1)
	r2 := NewGSet(2)
	r3 := NewGSet(3)

	r1.Add("alice"); r1.Add("carol")
	r2.Add("bob")
	r3.Add("bob") // same as R2 — duplicate, idempotent

	fmt.Printf("  Before merge: R1=%s  R2=%s  R3=%s\n\n", r1, r2, r3)

	r1.Merge(r2); r1.Merge(r3)
	r2.Merge(r1); r2.Merge(r3)
	r3.Merge(r1); r3.Merge(r2)

	fmt.Printf("  After merge:  R1=%s\n", r1)
	fmt.Printf("                R2=%s\n", r2)
	fmt.Printf("                R3=%s\n\n", r3)
	fmt.Println("  ✅ All replicas converged. Duplicate add(bob) was idempotent.")
	fmt.Println()
}

// ── 4. 2P-Set (Two-Phase Set: add and remove) ─────────────────────────────────
// Two G-Sets: A (added) and R (removed).
// An element is in the set if it is in A but not in R.
// Once removed, it cannot be re-added.

type TwoPSet struct {
	id      int
	added   GSet
	removed GSet
}

func NewTwoPSet(id int) *TwoPSet {
	return &TwoPSet{
		id:      id,
		added:   GSet{id: id, members: make(map[string]bool)},
		removed: GSet{id: id, members: make(map[string]bool)},
	}
}

func (s *TwoPSet) Add(elem string)    { s.added.Add(elem) }
func (s *TwoPSet) Remove(elem string) { s.removed.Add(elem) }
func (s *TwoPSet) Contains(elem string) bool {
	return s.added.Contains(elem) && !s.removed.Contains(elem)
}
func (s *TwoPSet) Elements() []string {
	var elems []string
	for k := range s.added.members {
		if !s.removed.Contains(k) { elems = append(elems, k) }
	}
	sort.Strings(elems)
	return elems
}
func (s *TwoPSet) Merge(other *TwoPSet) {
	s.added.Merge(&other.added)
	s.removed.Merge(&other.removed)
}
func (s *TwoPSet) String() string {
	return fmt.Sprintf("A=%v R=%v live=%v",
		s.added.Elements(), s.removed.Elements(), s.Elements())
}

func demo2PSet() {
	sep("═", 72)
	fmt.Println("CRDT 4 — 2P-Set (Add and Remove)")
	sep("═", 72)
	fmt.Println()
	fmt.Println("  A-set holds additions, R-set holds removals.")
	fmt.Println("  live = A - R. Once removed, element cannot be re-added.")
	fmt.Println()

	ops := []Op{
		{1, 1, "add(alice)"},
		{1, 2, "add(bob)"},
		{2, 1, "add(carol)"},
		{2, 2, "remove(bob)"},
		{3, 1, "merge(R2)"},
		{3, 2, "merge(R1)"},
		{3, 3, "merge(R1,R2)"},
		{4, 1, "{alice,carol}"},
		{4, 2, "{alice,carol}"},
		{4, 3, "{alice,carol}"},
	}
	printTimeline(ops, 3, 5)

	r1 := NewTwoPSet(1)
	r2 := NewTwoPSet(2)
	r3 := NewTwoPSet(3)

	r1.Add("alice"); r1.Add("carol")
	r2.Add("bob"); r2.Remove("bob")

	fmt.Printf("  Before merge: R1: %s\n", r1)
	fmt.Printf("                R2: %s\n\n", r2)

	r1.Merge(r2); r1.Merge(r3)
	r2.Merge(r1); r2.Merge(r3)
	r3.Merge(r1); r3.Merge(r2)

	fmt.Printf("  After merge:  R1: %s\n", r1)
	fmt.Printf("                R2: %s\n", r2)
	fmt.Printf("                R3: %s\n\n", r3)

	// attempt to re-add bob after removal
	r1.Add("bob")
	fmt.Printf("  R1 tries to re-add(bob): %s\n", r1)
	fmt.Println("  ⚠️  bob is in A-set but also in R-set → still excluded from live.")
	fmt.Println("  ✅ Removal is permanent in 2P-Set.")
	fmt.Println()
}

// ── 5. LWW-Register ───────────────────────────────────────────────────────────
// Single value with a logical timestamp.
// Merge = keep the entry with the higher timestamp.

type LWWRegister struct {
	id    int
	value string
	ts    int64
}

func NewLWWRegister(id int) *LWWRegister { return &LWWRegister{id: id} }

func (r *LWWRegister) Write(value string, ts int64) {
	if ts > r.ts {
		r.value = value
		r.ts = ts
	}
}

func (r *LWWRegister) Merge(other *LWWRegister) {
	if other.ts > r.ts {
		r.value = other.value
		r.ts = other.ts
	}
}

func (r *LWWRegister) String() string {
	return fmt.Sprintf("value=%q ts=%d", r.value, r.ts)
}

func demoLWWRegister() {
	sep("═", 72)
	fmt.Println("CRDT 5 — LWW-Register (Last-Write-Wins Register)")
	sep("═", 72)
	fmt.Println()
	fmt.Println("  Each write carries a timestamp. Merge keeps the higher timestamp.")
	fmt.Println("  Concurrent writes resolve deterministically — later ts wins.")
	fmt.Println()

	ops := []Op{
		{1, 1, "W(x)a ts=10"},
		{1, 2, "W(x)b ts=20"},
		{2, 1, "W(x)c ts=5"},  // older — will lose
		{3, 1, "merge(R2)"},
		{3, 2, "merge(R1)"},
		{3, 3, "merge(R1,R2)"},
		{4, 1, `value="b"`},
		{4, 2, `value="b"`},
		{4, 3, `value="b"`},
	}
	printTimeline(ops, 3, 5)

	r1 := NewLWWRegister(1)
	r2 := NewLWWRegister(2)
	r3 := NewLWWRegister(3)

	r1.Write("a", 10)
	r1.Write("c", 5) // older ts — rejected locally
	r2.Write("b", 20)

	fmt.Printf("  Before merge: R1: %s\n", r1)
	fmt.Printf("                R2: %s\n", r2)
	fmt.Printf("                R3: %s\n\n", r3)

	r1.Merge(r2); r1.Merge(r3)
	r2.Merge(r1); r2.Merge(r3)
	r3.Merge(r1); r3.Merge(r2)

	fmt.Printf("  After merge:  R1: %s\n", r1)
	fmt.Printf("                R2: %s\n", r2)
	fmt.Printf("                R3: %s\n\n", r3)
	fmt.Println("  ✅ All replicas converged to value=b (ts=20 wins).")
	fmt.Println("  ⚠️  Write c (ts=5) was silently discarded.")
	fmt.Println()
}

// ── 6. OR-Set (Observed-Remove Set) ──────────────────────────────────────────
// Each add gives the element a unique tag.
// Remove only removes the specific tags observed at remove time.
// A concurrent add (with a different tag) survives the remove.
// This fixes the 2P-Set limitation: elements CAN be re-added.

type ORSet struct {
	id      int
	counter int
	added   map[string]map[string]bool // elem → set of unique tags
	removed map[string]bool            // removed tags
}

func NewORSet(id int) *ORSet {
	return &ORSet{
		id:      id,
		added:   make(map[string]map[string]bool),
		removed: make(map[string]bool),
	}
}

func (s *ORSet) Add(elem string) string {
	s.counter++
	tag := fmt.Sprintf("R%d#%d", s.id, s.counter)
	if s.added[elem] == nil {
		s.added[elem] = make(map[string]bool)
	}
	s.added[elem][tag] = true
	return tag
}

func (s *ORSet) Remove(elem string) {
	// remove all currently observed tags for this element
	for tag := range s.added[elem] {
		s.removed[tag] = true
	}
}

func (s *ORSet) Contains(elem string) bool {
	for tag := range s.added[elem] {
		if !s.removed[tag] { return true }
	}
	return false
}

func (s *ORSet) Elements() []string {
	var elems []string
	for elem := range s.added {
		if s.Contains(elem) { elems = append(elems, elem) }
	}
	sort.Strings(elems)
	return elems
}

func (s *ORSet) Merge(other *ORSet) {
	for elem, tags := range other.added {
		if s.added[elem] == nil { s.added[elem] = make(map[string]bool) }
		for tag := range tags { s.added[elem][tag] = true }
	}
	for tag := range other.removed { s.removed[tag] = true }
}

func (s *ORSet) String() string {
	return fmt.Sprintf("live=%v", s.Elements())
}

func demoORSet() {
	sep("═", 72)
	fmt.Println("CRDT 6 — OR-Set (Observed-Remove Set)")
	sep("═", 72)
	fmt.Println()
	fmt.Println("  Each add tags the element with a unique ID.")
	fmt.Println("  Remove only removes currently observed tags.")
	fmt.Println("  A concurrent add survives the remove — re-add is possible.")
	fmt.Println()

	ops := []Op{
		{1, 1, "add(alice)"},
		{1, 2, "add(alice)"},  // concurrent add — different tag
		{2, 1, "remove(alice)"}, // R1 removes its own tag only
		{3, 1, "merge(R2)"},
		{3, 2, "merge(R1)"},
		{4, 1, "{alice}"},     // R2's add survived the remove!
		{4, 2, "{alice}"},
	}
	printTimeline(ops, 2, 5)

	r1 := NewORSet(1)
	r2 := NewORSet(2)

	tag1 := r1.Add("alice")
	tag2 := r2.Add("alice") // concurrent add on R2

	fmt.Printf("  R1 adds alice → tag=%s\n", tag1)
	fmt.Printf("  R2 adds alice → tag=%s  (concurrent — different tag)\n\n", tag2)

	r1.Remove("alice") // removes only tag1 (R1's observed tag)
	fmt.Printf("  R1 removes alice (removes tag=%s, tag=%s not yet seen)\n\n", tag1, tag2)

	fmt.Printf("  Before merge: R1: %s  R2: %s\n\n", r1, r2)

	r1.Merge(r2)
	r2.Merge(r1)

	fmt.Printf("  After merge:  R1: %s\n", r1)
	fmt.Printf("                R2: %s\n\n", r2)
	fmt.Println("  ✅ alice survives because R2's concurrent add had its own tag,")
	fmt.Println("     which was not in R1's remove set.")
	fmt.Println("  This solves the 2P-Set limitation — re-add after remove works.")
	fmt.Println()
}

// ── CRDT Properties Verification ─────────────────────────────────────────────
// Demonstrate that merge is commutative, associative, idempotent
// using the G-Counter as the example.

func verifyMergeProperties() {
	sep("═", 72)
	fmt.Println("MERGE PROPERTY VERIFICATION  (G-Counter)")
	sep("═", 72)
	fmt.Println()

	make3 := func() (*GCounter, *GCounter, *GCounter) {
		a := NewGCounter(1); a.Increment(3)
		b := NewGCounter(2); b.Increment(5)
		c := NewGCounter(3); c.Increment(4)
		return a, b, c
	}

	// Commutativity: merge(a,b) == merge(b,a)
	a, b, _ := make3()
	ab := NewGCounter(1); *ab = *a; ab.Merge(b)
	ba := NewGCounter(2); *ba = *b; ba.Merge(a)
	commOk := ab.Value() == ba.Value()
	fmt.Printf("  Commutativity  merge(a,b)==merge(b,a): %v == %v  %s\n",
		ab.Value(), ba.Value(), mark(commOk))

	// Associativity: merge(a,merge(b,c)) == merge(merge(a,b),c)
	a, b, c := make3()
	bc := NewGCounter(2); *bc = *b; bc.Merge(c)
	a_bc := NewGCounter(1); *a_bc = *a; a_bc.Merge(bc)

	a2, b2, c2 := make3()
	ab2 := NewGCounter(1); *ab2 = *a2; ab2.Merge(b2)
	ab_c := NewGCounter(1); *ab_c = *ab2; ab_c.Merge(c2)
	assocOk := a_bc.Value() == ab_c.Value()
	fmt.Printf("  Associativity  merge(a,merge(b,c))==merge(merge(a,b),c): %v == %v  %s\n",
		a_bc.Value(), ab_c.Value(), mark(assocOk))

	// Idempotency: merge(a,a) == a
	a3, _, _ := make3()
	aa := NewGCounter(1); *aa = *a3; aa.Merge(a3)
	idempOk := aa.Value() == a3.Value()
	fmt.Printf("  Idempotency    merge(a,a)==a: %v == %v  %s\n",
		aa.Value(), a3.Value(), mark(idempOk))

	fmt.Println()
	if commOk && assocOk && idempOk {
		fmt.Println("  ✅ All three properties hold — merge is a valid CRDT join.")
	}
	fmt.Println()
}

func mark(ok bool) string {
	if ok { return "✅" }
	return "✗ "
}

// ── Summary table ─────────────────────────────────────────────────────────────

func summaryTable() {
	sep("═", 72)
	fmt.Println("SUMMARY — CRDT Types")
	sep("═", 72)
	fmt.Println()
	fmt.Printf("  %-16s  %-12s  %-12s  %s\n", "Type", "Operations", "Merge", "Limitation")
	fmt.Printf("  %-16s  %-12s  %-12s  %s\n",
		strings.Repeat("─", 16), strings.Repeat("─", 12),
		strings.Repeat("─", 12), strings.Repeat("─", 32))
	rows := [][]string{
		{"G-Counter",     "inc",        "max/slot",   "no decrement"},
		{"PN-Counter",    "inc, dec",   "max/slot×2", "none"},
		{"G-Set",         "add",        "union",      "no remove"},
		{"2P-Set",        "add, remove","union×2",    "no re-add after remove"},
		{"LWW-Register",  "write",      "max(ts)",    "silently discards older"},
		{"OR-Set",        "add, remove","union+tags", "larger metadata"},
	}
	for _, row := range rows {
		fmt.Printf("  %-16s  %-12s  %-12s  %s\n", row[0], row[1], row[2], row[3])
	}
	fmt.Println()
}

func main() {
	sep("═", 72)
	fmt.Println("CRDT — Conflict-free Replicated Data Types")
	sep("═", 72)
	fmt.Println()
	fmt.Println("  Merge must be: commutative, associative, idempotent.")
	fmt.Println("  Replicas update locally with no coordination.")
	fmt.Println("  Merging any two replicas always produces the same result.")
	fmt.Println()

	demoGCounter()
	demoPNCounter()
	demoGSet()
	demo2PSet()
	demoLWWRegister()
	demoORSet()
	verifyMergeProperties()
	summaryTable()
}
