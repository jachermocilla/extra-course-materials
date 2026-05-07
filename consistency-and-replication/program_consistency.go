package main

import (
	"fmt"
	"sort"
	"strings"
)

// ============================================================
// Monotonic Problem — Shopping Cart
//
// A MONOTONIC problem can reach a (partial) solution even with
// missing information. When missing information arrives, the
// solution never needs to roll back — it only moves forward.
//
// Shopping cart ADD operations are monotonic:
//   - Any server can accept any add in any order
//   - No write-write conflict between adds
//   - Missing adds arriving later only grow the cart
//   - No rollback ever needed
//
// But REMOVE operations break monotonicity:
//   - A remove retracts a previously derived conclusion
//   - If remove arrives before the add it refers to: anomaly
//
// Solution: split into two grow-only sets
//   A = set of items added    (grows monotonically)
//   D = set of items deleted  (grows monotonically)
//
//   Each set individually is monotonic — it only ever grows.
//   Coordination is needed ONLY once both sets have converged.
//   Final cart = A - D  (computed once, at coordination point)
//
// This is exactly the 2P-Set CRDT applied to shopping carts.
// ============================================================

// ── Op for timeline ───────────────────────────────────────────────────────────

type Op struct {
	slot    int
	subject string
	label   string
}

func centerPad(s string, width int) string {
	if len(s) >= width { return s }
	total := width - len(s)
	left := total / 2
	return strings.Repeat("─", left) + s + strings.Repeat("─", total-left)
}

func printTimeline(ops []Op, rows []string, totalSlots int) {
	colW := 16
	dash := strings.Repeat("─", colW)
	hdr := fmt.Sprintf("  %-5s  ", "")
	for s := 1; s <= totalSlots; s++ {
		hdr += centerPad(fmt.Sprintf("t%d", s), colW)
	}
	fmt.Println(hdr)
	fmt.Printf("  %-5s  %s\n", "", strings.Repeat("─", colW*totalSlots))
	for _, row := range rows {
		line := fmt.Sprintf("  %-5s  ", row)
		for s := 1; s <= totalSlots; s++ {
			cell := ""
			for _, op := range ops {
				if op.subject == row && op.slot == s {
					cell = op.label
				}
			}
			if cell == "" { line += dash } else { line += centerPad(cell, colW) }
		}
		fmt.Println(line)
	}
	fmt.Println()
}

func sep(char string, n int) { fmt.Println(strings.Repeat(char, n)) }

// ── Grow-only Set ─────────────────────────────────────────────────────────────
// Each set only ever grows. Merge = union. Idempotent, commutative,
// associative. No rollback ever needed.

type GrowSet struct {
	name    string
	members map[string]bool
}

func NewGrowSet(name string) *GrowSet {
	return &GrowSet{name: name, members: make(map[string]bool)}
}

func (s *GrowSet) Add(item string) {
	s.members[item] = true
}

func (s *GrowSet) Contains(item string) bool {
	return s.members[item]
}

func (s *GrowSet) Elements() []string {
	var elems []string
	for k := range s.members { elems = append(elems, k) }
	sort.Strings(elems)
	return elems
}

// Merge = union — only grows, never shrinks
func (s *GrowSet) Merge(other *GrowSet) {
	for k := range other.members {
		s.members[k] = true
	}
}

func (s *GrowSet) String() string {
	e := s.Elements()
	if len(e) == 0 { return "∅" }
	return "{" + strings.Join(e, ", ") + "}"
}

// HasConverged returns true if this replica has the same elements
// as the other — used to detect when coordination can happen
func (s *GrowSet) HasConverged(other *GrowSet) bool {
	if len(s.members) != len(other.members) { return false }
	for k := range s.members {
		if !other.members[k] { return false }
	}
	return true
}

// ── MonotonicCart — the split A/D design ──────────────────────────────────────
// A = grow-only set of additions
// D = grow-only set of deletions
// Both sets grow monotonically on each replica independently.
// Coordination needed only once both sets have converged.

type MonotonicCart struct {
	replicaID string
	A         *GrowSet // added items
	D         *GrowSet // deleted items
	converged bool     // has coordination happened?
}

func NewMonotonicCart(id string) *MonotonicCart {
	return &MonotonicCart{
		replicaID: id,
		A:         NewGrowSet("A"),
		D:         NewGrowSet("D"),
	}
}

// Add grows the A set — purely monotonic, no coordination
func (c *MonotonicCart) Add(item string) {
	c.A.Add(item)
	fmt.Printf("  [%s] add(%s) → A=%s  [monotonic — no coordination]\n",
		c.replicaID, item, c.A)
}

// Delete grows the D set — purely monotonic, no coordination
func (c *MonotonicCart) Delete(item string) {
	c.D.Add(item)
	fmt.Printf("  [%s] delete(%s) → D=%s  [monotonic — no coordination]\n",
		c.replicaID, item, c.D)
}

// MergeA propagates only the A set — monotonic, safe at any time
func (c *MonotonicCart) MergeA(other *MonotonicCart) {
	before := c.A.String()
	c.A.Merge(other.A)
	fmt.Printf("  [%s] merge A from %s: %s → %s\n",
		c.replicaID, other.replicaID, before, c.A)
}

// MergeD propagates only the D set — monotonic, safe at any time
func (c *MonotonicCart) MergeD(other *MonotonicCart) {
	before := c.D.String()
	c.D.Merge(other.D)
	fmt.Printf("  [%s] merge D from %s: %s → %s\n",
		c.replicaID, other.replicaID, before, c.D)
}

// Coordinate computes the final cart = A - D
// This is the ONLY point that requires coordination.
// Both A and D must have converged before this is called.
func (c *MonotonicCart) Coordinate() []string {
	var result []string
	for _, item := range c.A.Elements() {
		if !c.D.Contains(item) {
			result = append(result, item)
		}
	}
	c.converged = true
	fmt.Printf("  [%s] COORDINATE: A%s - D%s = {%s}\n",
		c.replicaID, c.A, c.D, strings.Join(result, ", "))
	return result
}

func (c *MonotonicCart) PrintState() {
	fmt.Printf("  [%s] A=%s  D=%s\n", c.replicaID, c.A, c.D)
}

// ── SCENARIO 1: Add-only — purely monotonic, no coordination ──────────────────

func scenarioAddOnly() {
	sep("═", 72)
	fmt.Println("SCENARIO 1 — Add-only Cart (purely monotonic)")
	sep("═", 72)
	fmt.Println()
	fmt.Println("  Three servers accept adds in any order.")
	fmt.Println("  No coordination needed — adds only grow A.")
	fmt.Println("  Missing adds arriving later never cause rollback.")
	fmt.Println()

	ops := []Op{
		{1, "S1", "add(shoes)"},
		{1, "S2", "add(hat)"},
		{2, "S1", "add(bag)"},
		{2, "S3", "add(shoes)"},  // duplicate — idempotent
		{3, "S1", "mergeA(S2)"},
		{3, "S2", "mergeA(S3)"},
		{4, "S1", "mergeA(S3)"},
		{4, "S2", "mergeA(S1)"},
		{4, "S3", "mergeA(S1,S2)"},
		{5, "S1", "A={bag,hat,shoes}"},
		{5, "S2", "A={bag,hat,shoes}"},
		{5, "S3", "A={bag,hat,shoes}"},
	}
	printTimeline(ops, []string{"S1", "S2", "S3"}, 6)

	s1 := NewMonotonicCart("S1")
	s2 := NewMonotonicCart("S2")
	s3 := NewMonotonicCart("S3")

	fmt.Println("  Independent adds on each server:")
	s1.Add("shoes")
	s2.Add("hat")
	s1.Add("bag")
	s3.Add("shoes") // duplicate — idempotent

	fmt.Println()
	fmt.Println("  Propagate A sets (can happen in any order, any time):")
	s1.MergeA(s2)
	s2.MergeA(s3)
	s1.MergeA(s3)
	s2.MergeA(s1)
	s3.MergeA(s1)

	fmt.Println()
	fmt.Println("  All servers converged — no coordination needed:")
	s1.PrintState()
	s2.PrintState()
	s3.PrintState()
	fmt.Println()
	fmt.Println("  ✅ Monotonic: every add only grows A. No rollback ever needed.")
	fmt.Println("     Duplicate add(shoes) from S3 was idempotent — harmless.")
	fmt.Println()
}

// ── SCENARIO 2: Non-monotonic remove — anomaly without the split ──────────────

func scenarioNaiveRemove() {
	sep("═", 72)
	fmt.Println("SCENARIO 2 — Naive Remove (breaks monotonicity)")
	sep("═", 72)
	fmt.Println()
	fmt.Println("  S1 adds shoes. Concurrently S2 removes shoes (saw it earlier).")
	fmt.Println("  Without the A/D split, the outcome depends on merge order.")
	fmt.Println("  Information arriving later causes a ROLLBACK of a conclusion.")
	fmt.Println()

	ops := []Op{
		{1, "S1", "add(shoes)"},
		{1, "S2", "remove(shoes)⚠️"},
		{2, "S1", "cart={shoes}"},
		{2, "S2", "cart={}"},
		{3, "S1", "merge(S2)→?"},
		{4, "S1", "rollback!⚠️"},
	}
	printTimeline(ops, []string{"S1", "S2"}, 5)

	fmt.Println("  S1 believes cart = {shoes}  (add arrived)")
	fmt.Println("  S2 believes cart = {}       (remove arrived)")
	fmt.Println()
	fmt.Println("  When S1 merges S2's remove:")
	fmt.Println("  S1 must RETRACT its conclusion that shoes is in the cart.")
	fmt.Println("  This is a rollback — violates monotonicity.")
	fmt.Println()
	fmt.Println("  ⚠️  The problem: a remove is non-monotonic.")
	fmt.Println("     It retracts previously derived information.")
	fmt.Println("     No matter when the remove arrives, S1 must change its answer.")
	fmt.Println()
}

// ── SCENARIO 3: Split A/D — both sets monotonic, coordinate at convergence ────

func scenarioSplitAD() {
	sep("═", 72)
	fmt.Println("SCENARIO 3 — Split A/D Sets (both monotonic)")
	sep("═", 72)
	fmt.Println()
	fmt.Println("  Add and delete operations each grow their own set.")
	fmt.Println("  Both A and D are monotonic — they only ever grow.")
	fmt.Println("  No rollback needed during propagation.")
	fmt.Println("  Coordination happens ONCE when both sets have converged.")
	fmt.Println("  Final cart = A - D, computed at that single coordination point.")
	fmt.Println()

	ops := []Op{
		{1, "S1", "add(shoes)→A"},
		{1, "S2", "add(hat)→A"},
		{2, "S1", "add(bag)→A"},
		{2, "S2", "del(shoes)→D"},
		{3, "S1", "mergeA(S2)"},
		{3, "S2", "mergeA(S1)"},
		{4, "S1", "mergeD(S2)"},
		{4, "S2", "mergeD(S1)"},
		{5, "S1", "coordinate!"},
		{5, "S2", "coordinate!"},
	}
	printTimeline(ops, []string{"S1", "S2"}, 6)

	s1 := NewMonotonicCart("S1")
	s2 := NewMonotonicCart("S2")

	fmt.Println("  Independent operations — grows only, no coordination:")
	s1.Add("shoes")
	s2.Add("hat")
	s1.Add("bag")
	s2.Delete("shoes")

	fmt.Println()
	fmt.Println("  State before propagation:")
	s1.PrintState()
	s2.PrintState()

	fmt.Println()
	fmt.Println("  Propagate A sets (monotonic — safe at any time):")
	s1.MergeA(s2)
	s2.MergeA(s1)

	fmt.Println()
	fmt.Println("  A sets converged — check:")
	fmt.Printf("  S1.A=%s  S2.A=%s  converged=%v\n",
		s1.A, s2.A, s1.A.HasConverged(s2.A))

	fmt.Println()
	fmt.Println("  Propagate D sets (monotonic — safe at any time):")
	s1.MergeD(s2)
	s2.MergeD(s1)

	fmt.Println()
	fmt.Println("  D sets converged — check:")
	fmt.Printf("  S1.D=%s  S2.D=%s  converged=%v\n",
		s1.D, s2.D, s1.D.HasConverged(s2.D))

	fmt.Println()
	fmt.Println("  Both sets converged → single coordination point — compute A - D:")
	result1 := s1.Coordinate()
	result2 := s2.Coordinate()

	fmt.Println()
	fmt.Printf("  S1 final cart: {%s}\n", strings.Join(result1, ", "))
	fmt.Printf("  S2 final cart: {%s}\n", strings.Join(result2, ", "))
	fmt.Println()
	fmt.Println("  ✅ No rollback during propagation — A and D only grew.")
	fmt.Println("  ✅ Coordination cost paid exactly ONCE, at convergence.")
	fmt.Println("  ✅ Both servers reach identical final cart deterministically.")
	fmt.Println()
}

// ── SCENARIO 4: Late arrivals — no rollback needed ────────────────────────────

func scenarioLateArrival() {
	sep("═", 72)
	fmt.Println("SCENARIO 4 — Late Arriving Operations (no rollback)")
	sep("═", 72)
	fmt.Println()
	fmt.Println("  Operations arrive out of order due to network delay.")
	fmt.Println("  Because A and D only grow, late arrivals never cause rollback.")
	fmt.Println("  The partial solution at any point is always a valid prefix.")
	fmt.Println()

	ops := []Op{
		{1, "S1", "add(shoes)→A"},
		{2, "S1", "add(hat)→A"},
		{3, "S2", "add(bag)→A"},   // S2 gets bag first
		{3, "S2", "del(hat)→D"},
		{4, "S2", "add(shoes)→A"}, // shoes arrives late on S2
		{5, "S1", "mergeA(S2)"},
		{5, "S1", "mergeD(S2)"},
		{6, "S1", "coordinate!"},
		{6, "S2", "coordinate!"},
	}
	printTimeline(ops, []string{"S1", "S2"}, 7)

	s1 := NewMonotonicCart("S1")
	s2 := NewMonotonicCart("S2")

	fmt.Println("  S1 accepts adds immediately:")
	s1.Add("shoes")
	s1.Add("hat")

	fmt.Println()
	fmt.Println("  S2 processes its operations independently (different order):")
	s2.Add("bag")
	s2.Delete("hat")
	s2.Add("shoes") // arrives late on S2 — no problem, A just grows

	fmt.Println()
	fmt.Println("  S1 partial view at any moment is always valid (never rolled back):")
	fmt.Printf("  S1 partial cart ≈ A-D = {%s} - {%s} = {shoes,hat}\n",
		s1.A, s1.D)
	fmt.Println("  (shoes and hat — correct partial answer without bag)")

	fmt.Println()
	fmt.Println("  Propagate and coordinate:")
	s1.MergeA(s2)
	s1.MergeD(s2)
	s2.MergeA(s1)
	s2.MergeD(s1)

	result1 := s1.Coordinate()
	result2 := s2.Coordinate()
	fmt.Println()
	fmt.Printf("  S1 final cart: {%s}\n", strings.Join(result1, ", "))
	fmt.Printf("  S2 final cart: {%s}\n", strings.Join(result2, ", "))
	fmt.Println()
	fmt.Println("  ✅ Late arrival of add(shoes) on S2 only grew A — no rollback.")
	fmt.Println("  ✅ hat was correctly excluded (in D).")
	fmt.Println("  ✅ Partial answers along the way were always valid prefixes.")
	fmt.Println()
}

// ── SCENARIO 5: What requires coordination ────────────────────────────────────

func scenarioCoordinationPoint() {
	sep("═", 72)
	fmt.Println("SCENARIO 5 — The Coordination Point")
	sep("═", 72)
	fmt.Println()
	fmt.Println("  The difference A - D is the ONLY operation requiring coordination.")
	fmt.Println("  Everything before that point is monotonic and coordination-free.")
	fmt.Println("  We show what goes wrong if coordination is skipped (premature).")
	fmt.Println()

	s1 := NewMonotonicCart("S1")
	s2 := NewMonotonicCart("S2")

	s1.Add("shoes")
	s1.Add("hat")
	s2.Add("bag")
	s2.Delete("shoes")

	fmt.Println()
	fmt.Println("  ── Premature coordination (A/D not yet converged) ───────────")
	fmt.Println()
	fmt.Println("  S1 computes A - D before merging S2's operations:")
	premature := s1.Coordinate()
	fmt.Printf("  S1 premature cart: {%s}  ⚠️  missing bag, shoes not yet removed\n",
		strings.Join(premature, ", "))

	fmt.Println()
	fmt.Println("  ── Correct coordination (after full convergence) ─────────────")
	fmt.Println()
	s1.MergeA(s2); s1.MergeD(s2)
	s2.MergeA(s1); s2.MergeD(s1)

	fmt.Println()
	fmt.Printf("  A sets converged: %v\n", s1.A.HasConverged(s2.A))
	fmt.Printf("  D sets converged: %v\n", s1.D.HasConverged(s2.D))
	fmt.Println()
	correct := s1.Coordinate()
	fmt.Printf("  S1 correct cart: {%s}  ✅\n", strings.Join(correct, ", "))

	fmt.Println()
	fmt.Println("  The coordination point is when both A and D have globally converged.")
	fmt.Println("  Before that: operate freely on monotonic sets.")
	fmt.Println("  After that:  compute A - D once to get the final consistent answer.")
	fmt.Println()
}

// ── Summary ───────────────────────────────────────────────────────────────────

func summary() {
	sep("═", 72)
	fmt.Println("SUMMARY")
	sep("═", 72)
	fmt.Println()
	fmt.Println("  Monotonic problem:")
	fmt.Println("    A program is monotonic if adding more information never causes")
	fmt.Println("    it to retract a previous conclusion.")
	fmt.Println("    Monotonic programs are safe under eventual consistency.")
	fmt.Println("    Missing information arriving later only moves the solution forward.")
	fmt.Println()
	fmt.Println("  Shopping cart:")
	fmt.Println("    Add-only   → monotonic  → no coordination needed, ever")
	fmt.Println("    Remove     → breaks monotonicity")
	fmt.Println()
	fmt.Println("  Solution — split into two grow-only sets:")
	fmt.Println("    A (added items)   → grows monotonically on each replica")
	fmt.Println("    D (deleted items) → grows monotonically on each replica")
	fmt.Println("    Both A and D propagate freely — no coordination during growth")
	fmt.Println("    Coordination needed ONCE: when A and D have both converged")
	fmt.Println("    Final cart = A - D  (computed at that single coordination point)")
	fmt.Println()

	fmt.Printf("  %-22s  %-12s  %-14s  %s\n",
		"Phase", "Monotonic?", "Coordination?", "Operation")
	fmt.Printf("  %-22s  %-12s  %-14s  %s\n",
		strings.Repeat("─", 22), strings.Repeat("─", 12),
		strings.Repeat("─", 14), strings.Repeat("─", 30))
	phases := [][]string{
		{"grow A (add items)",      "✅ yes", "none",     "A = A ∪ {item}"},
		{"grow D (delete items)",   "✅ yes", "none",     "D = D ∪ {item}"},
		{"propagate A",             "✅ yes", "none",     "A = A ∪ A_remote"},
		{"propagate D",             "✅ yes", "none",     "D = D ∪ D_remote"},
		{"compute A - D",           "✗  no",  "required", "final cart, once converged"},
	}
	for _, p := range phases {
		fmt.Printf("  %-22s  %-12s  %-14s  %s\n", p[0], p[1], p[2], p[3])
	}
	fmt.Println()
}

func main() {
	sep("═", 72)
	fmt.Println("MONOTONIC PROBLEM — Shopping Cart")
	sep("═", 72)
	fmt.Println()
	fmt.Println("  A grows-only set (A) for additions and a grow-only set (D)")
	fmt.Println("  for deletions. Both are monotonic — they only ever grow.")
	fmt.Println("  Coordination is needed only once: when computing A - D.")
	fmt.Println()

	scenarioAddOnly()
	scenarioNaiveRemove()
	scenarioSplitAD()
	scenarioLateArrival()
	scenarioCoordinationPoint()
	summary()
}
