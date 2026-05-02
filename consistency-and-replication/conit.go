package main

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// ============================================================
// Continuous Consistency and Conits
//
// A conit (CONsistency unIT) is the object to which consistency
// bounds are applied. Each conit has three independent bounds:
//
//   1. Numerical deviation  — max allowed |local_value - global_value|
//   2. Staleness deviation  — max allowed age of data (seconds)
//   3. Ordering deviation   — max allowed tentative (unordered) writes
//
// When any bound is exceeded, the system synchronises that conit.
// Different conits can have different bounds — fine-grained control.
// ============================================================

// ── Conit specification ───────────────────────────────────────────────────────

type ConitSpec struct {
	name             string
	maxNumerical     float64 // allowed |local - global|
	maxStaleness     float64 // allowed age in seconds
	maxOrdering      int     // allowed tentative writes
}

// ── Conit state at a replica ──────────────────────────────────────────────────

type ConitState struct {
	spec             ConitSpec
	localValue       float64
	globalValue      float64   // last known globally consistent value
	lastSyncTime     time.Time // when was this conit last synced
	tentativeWrites  int       // writes not yet globally ordered
}

func NewConit(spec ConitSpec, initial float64) *ConitState {
	return &ConitState{
		spec:         spec,
		localValue:   initial,
		globalValue:  initial,
		lastSyncTime: time.Now(),
	}
}

func (c *ConitState) NumericalDeviation() float64 {
	return math.Abs(c.localValue - c.globalValue)
}

func (c *ConitState) StalenessDeviation() float64 {
	return time.Since(c.lastSyncTime).Seconds()
}

func (c *ConitState) OrderingDeviation() int {
	return c.tentativeWrites
}

// WithinBounds returns true if all three metrics are within spec
func (c *ConitState) WithinBounds() (bool, string) {
	if c.NumericalDeviation() > c.spec.maxNumerical {
		return false, fmt.Sprintf("numerical deviation %.2f exceeds bound %.2f",
			c.NumericalDeviation(), c.spec.maxNumerical)
	}
	if c.StalenessDeviation() > c.spec.maxStaleness {
		return false, fmt.Sprintf("staleness %.3fs exceeds bound %.3fs",
			c.StalenessDeviation(), c.spec.maxStaleness)
	}
	if c.OrderingDeviation() > c.spec.maxOrdering {
		return false, fmt.Sprintf("ordering deviation %d exceeds bound %d",
			c.OrderingDeviation(), c.spec.maxOrdering)
	}
	return true, "all bounds satisfied"
}

// LocalWrite applies a local (tentative) write
func (c *ConitState) LocalWrite(delta float64, label string) {
	c.localValue += delta
	c.tentativeWrites++
	fmt.Printf("    [%s] local write Δ%+.1f → local=%.1f  tentative=%d\n",
		c.spec.name, delta, c.localValue, c.tentativeWrites)
}

// Sync brings the replica up to date with the global value
func (c *ConitState) Sync(globalValue float64) {
	c.globalValue = globalValue
	c.localValue = globalValue
	c.lastSyncTime = time.Now()
	c.tentativeWrites = 0
	fmt.Printf("    [%s] ✅ SYNC → global=%.1f  all deviations reset\n",
		c.spec.name, globalValue)
}

// Check prints the current deviation status
func (c *ConitState) Check() {
	ok, msg := c.WithinBounds()
	numDev := c.NumericalDeviation()
	stale := c.StalenessDeviation()
	ord := c.OrderingDeviation()
	sym := "✅"
	if !ok { sym = "⚠️ " }
	fmt.Printf("    [%s] %s  num=%.2f/%.2f  stale=%.3fs/%.3fs  ord=%d/%d  — %s\n",
		c.spec.name, sym,
		numDev, c.spec.maxNumerical,
		stale, c.spec.maxStaleness,
		ord, c.spec.maxOrdering,
		msg)
}

// ── Replica with multiple conits ──────────────────────────────────────────────

type Replica struct {
	id     int
	conits map[string]*ConitState
}

func NewReplica(id int, conits ...*ConitState) *Replica {
	r := &Replica{id: id, conits: make(map[string]*ConitState)}
	for _, c := range conits {
		r.conits[c.spec.name] = c
	}
	return r
}

func (r *Replica) checkAll() {
	fmt.Printf("\n  Replica R%d — deviation check:\n", r.id)
	for _, c := range r.conits { c.Check() }
}

func (r *Replica) syncIfNeeded(globalValues map[string]float64) {
	for name, c := range r.conits {
		ok, _ := c.WithinBounds()
		if !ok {
			if gv, exists := globalValues[name]; exists {
				fmt.Printf("  → R%d: bound exceeded on [%s], triggering sync\n", r.id, name)
				c.Sync(gv)
			}
		}
	}
}

// ── Timeline / table helpers ──────────────────────────────────────────────────

func sep(char string, n int) { fmt.Println(strings.Repeat(char, n)) }

func printConitTable(conits []ConitSpec) {
	fmt.Printf("  %-18s  %-16s  %-16s  %-14s  %s\n",
		"Conit", "Max Numerical", "Max Staleness", "Max Ordering", "Use case")
	fmt.Printf("  %-18s  %-16s  %-16s  %-14s  %s\n",
		strings.Repeat("─", 18), strings.Repeat("─", 16),
		strings.Repeat("─", 16), strings.Repeat("─", 14),
		strings.Repeat("─", 28))
	cases := []string{
		"payment ledger",
		"stock ticker",
		"weather sensor",
		"page view counter",
		"game player position",
	}
	for i, c := range conits {
		fmt.Printf("  %-18s  %-16.1f  %-16.3f  %-14d  %s\n",
			c.name, c.maxNumerical, c.maxStaleness, c.maxOrdering, cases[i])
	}
	fmt.Println()
}

// ── SCENARIO A: Numerical Deviation ──────────────────────────────────────────

func scenarioNumerical() {
	sep("═", 72)
	fmt.Println("SCENARIO A — Numerical Deviation")
	sep("═", 72)
	fmt.Println()
	fmt.Println("  Conit: stock price.  Max numerical deviation = 2.0")
	fmt.Println("  Local writes accumulate; sync triggered when bound is exceeded.")
	fmt.Println()

	spec := ConitSpec{"stock_price", 2.0, 60.0, 10}
	c := NewConit(spec, 100.0)

	fmt.Println("  Step 1 — small price drift, within bound:")
	c.LocalWrite(+1.5, "tick")
	c.Check()

	fmt.Println()
	fmt.Println("  Step 2 — larger drift, bound exceeded:")
	c.LocalWrite(+1.0, "tick")
	c.Check()

	ok, _ := c.WithinBounds()
	if !ok {
		fmt.Println()
		c.Sync(102.8) // global value after market update
	}
	fmt.Println()
}

// ── SCENARIO B: Staleness Deviation ──────────────────────────────────────────

func scenarioStaleness() {
	sep("═", 72)
	fmt.Println("SCENARIO B — Staleness Deviation")
	sep("═", 72)
	fmt.Println()
	fmt.Println("  Conit: weather reading.  Max staleness = 0.050s (50ms)")
	fmt.Println("  No writes — time passes, staleness bound exceeded.")
	fmt.Println()

	spec := ConitSpec{"weather", 5.0, 0.050, 5}
	c := NewConit(spec, 22.5)

	fmt.Println("  Immediately after creation:")
	c.Check()

	fmt.Println()
	fmt.Println("  After 30ms:")
	time.Sleep(30 * time.Millisecond)
	c.Check()

	fmt.Println()
	fmt.Println("  After 60ms total — staleness bound exceeded:")
	time.Sleep(35 * time.Millisecond)
	c.Check()

	ok, _ := c.WithinBounds()
	if !ok {
		fmt.Println()
		c.Sync(23.1)
	}
	fmt.Println()
}

// ── SCENARIO C: Ordering Deviation ───────────────────────────────────────────

func scenarioOrdering() {
	sep("═", 72)
	fmt.Println("SCENARIO C — Ordering Deviation")
	sep("═", 72)
	fmt.Println()
	fmt.Println("  Conit: page-view counter.  Max ordering deviation = 3")
	fmt.Println("  Tentative writes accumulate; sync when too many are unordered.")
	fmt.Println()

	spec := ConitSpec{"page_views", 1000.0, 300.0, 3}
	c := NewConit(spec, 5000.0)

	for i := 1; i <= 4; i++ {
		c.LocalWrite(+1, fmt.Sprintf("view #%d", i))
		c.Check()
		fmt.Println()
	}

	ok, _ := c.WithinBounds()
	if !ok {
		c.Sync(5004.0)
	}
	fmt.Println()
}

// ── SCENARIO D: Multiple Conits on One Replica ────────────────────────────────

func scenarioMultiConit() {
	sep("═", 72)
	fmt.Println("SCENARIO D — Multiple Conits, Different Bounds on One Replica")
	sep("═", 72)
	fmt.Println()
	fmt.Println("  One replica holds three conits with very different tolerances.")
	fmt.Println("  Sync is triggered per-conit independently.")
	fmt.Println()

	// Tight: bank balance — zero numerical tolerance
	bank := NewConit(ConitSpec{"bank_balance", 0.0, 1.0, 0}, 1000.0)
	// Medium: temperature sensor
	temp := NewConit(ConitSpec{"temperature", 1.0, 5.0, 2}, 20.0)
	// Loose: social media likes
	likes := NewConit(ConitSpec{"likes", 500.0, 60.0, 20}, 10000.0)

	r := NewReplica(1, bank, temp, likes)

	fmt.Println("  Initial state — all within bounds:")
	r.checkAll()

	fmt.Println()
	fmt.Println("  Apply local writes to all three conits:")
	fmt.Println()

	bank.LocalWrite(+50, "deposit")    // +50 on bank with 0 tolerance → immediate violation
	temp.LocalWrite(+0.5, "reading")   // +0.5 on temp with ±1 tolerance → still ok
	likes.LocalWrite(+200, "viral")    // +200 on likes with ±500 tolerance → still ok

	r.checkAll()

	globalValues := map[string]float64{
		"bank_balance": 1050.0,
		"temperature":  20.5,
		"likes":        10200.0,
	}

	fmt.Println()
	fmt.Println("  Sync check — only conits exceeding their bound are synced:")
	fmt.Println()
	r.syncIfNeeded(globalValues)

	fmt.Println()
	fmt.Println("  Final state after selective sync:")
	r.checkAll()
	fmt.Println()
}

// ── SCENARIO E: Conit Deviation Timeline ─────────────────────────────────────
// Show a table of how deviations grow over time and when sync fires.

func scenarioTimeline() {
	sep("═", 72)
	fmt.Println("SCENARIO E — Deviation Timeline Table")
	sep("═", 72)
	fmt.Println()
	fmt.Println("  Conit: score counter.  Bounds: numerical=5  staleness=0.04s  ordering=3")
	fmt.Println()

	spec := ConitSpec{"score", 5.0, 0.040, 3}
	c := NewConit(spec, 0.0)

	type row struct {
		t       string
		numDev  float64
		stale   float64
		ord     int
		withinN bool
		withinS bool
		withinO bool
		action  string
	}

	var rows []row
	snap := func(t, action string) {
		ok_n := c.NumericalDeviation() <= c.spec.maxNumerical
		ok_s := c.StalenessDeviation() <= c.spec.maxStaleness
		ok_o := c.OrderingDeviation() <= c.spec.maxOrdering
		rows = append(rows, row{
			t, c.NumericalDeviation(), c.StalenessDeviation(),
			c.OrderingDeviation(), ok_n, ok_s, ok_o, action,
		})
	}

	mark := func(ok bool, val string) string {
		if ok { return "✅ " + val }
		return "⚠️  " + val
	}

	snap("t0", "initial")
	c.LocalWrite(3, "w1"); snap("t1", "write +3")
	time.Sleep(20 * time.Millisecond)
	snap("t2", "20ms later")
	c.LocalWrite(3, "w2"); snap("t3", "write +3")
	time.Sleep(25 * time.Millisecond)
	snap("t4", "45ms total")
	c.LocalWrite(1, "w3"); snap("t5", "write +1")
	c.LocalWrite(1, "w4"); snap("t6", "write +1 → ord=4")

	// print table
	fmt.Printf("  %-6s  %-26s  %-18s  %-16s  %-14s\n",
		"Step", "Action", "Numerical", "Staleness", "Ordering")
	fmt.Printf("  %-6s  %-26s  %-18s  %-16s  %-14s\n",
		"──────", strings.Repeat("─", 26),
		strings.Repeat("─", 18), strings.Repeat("─", 16), strings.Repeat("─", 14))

	for _, r := range rows {
		fmt.Printf("  %-6s  %-26s  %-18s  %-16s  %-14s\n",
			r.t, r.action,
			mark(r.withinN, fmt.Sprintf("%.1f/%.1f", r.numDev, spec.maxNumerical)),
			mark(r.withinS, fmt.Sprintf("%.3fs/%.3fs", r.stale, spec.maxStaleness)),
			mark(r.withinO, fmt.Sprintf("%d/%d", r.ord, spec.maxOrdering)))
	}

	fmt.Println()
	fmt.Println("  Sync fires when the first bound is exceeded.")
	fmt.Println("  Each axis is independent — numerical may be fine while staleness fires.")
	fmt.Println()

	// find first violation and sync
	for _, r := range rows {
		if !r.withinN || !r.withinS || !r.withinO {
			fmt.Printf("  First violation at step %s (%s) — sync triggered.\n", r.t, r.action)
			break
		}
	}
	c.Sync(c.localValue) // bring replica current
	fmt.Println()
}

// ── Summary ───────────────────────────────────────────────────────────────────

func summary() {
	sep("═", 72)
	fmt.Println("SUMMARY — Continuous Consistency and Conits")
	sep("═", 72)
	fmt.Println()

	conits := []ConitSpec{
		{"bank_balance",   0.0,   1.0,    0},
		{"stock_price",    2.0,   5.0,    5},
		{"weather",        1.0,   30.0,   3},
		{"page_views",  1000.0,  300.0,  50},
		{"game_position",  5.0,   0.5,   10},
	}

	fmt.Println("  Example conit specifications per application:")
	fmt.Println()
	printConitTable(conits)

	fmt.Println("  Key points:")
	fmt.Println("  · A conit is the granularity of consistency — any logical data unit.")
	fmt.Println("  · Three independent axes: numerical, staleness, ordering deviation.")
	fmt.Println("  · Sync is triggered per-conit, per-axis — not for the whole store.")
	fmt.Println("  · Tight bounds (all zeros) → linearizability.")
	fmt.Println("  · No bounds → eventual consistency.")
	fmt.Println("  · Continuous consistency is the spectrum in between.")
	fmt.Println()

	fmt.Printf("  %-24s  %-28s  %s\n", "Consistency model", "Continuous equiv", "Deviation")
	fmt.Printf("  %-24s  %-28s  %s\n",
		strings.Repeat("─", 24), strings.Repeat("─", 28), strings.Repeat("─", 20))
	models := [][]string{
		{"Linearizable",        "conit: all bounds = 0",    "zero on all axes"},
		{"Sequential",          "conit: ord = 0",           "num/stale may vary"},
		{"Causal",              "conit: ord = causal order","no causal inversions"},
		{"Eventual",            "no conit bounds",          "unbounded"},
		{"Continuous (custom)", "per-conit spec",           "app-defined per axis"},
	}
	for _, m := range models {
		fmt.Printf("  %-24s  %-28s  %s\n", m[0], m[1], m[2])
	}
	fmt.Println()
}

func main() {
	sep("═", 72)
	fmt.Println("CONTINUOUS CONSISTENCY AND CONITS")
	sep("═", 72)
	fmt.Println()
	fmt.Println("  Consistency is not binary. It is measured along three axes:")
	fmt.Println("  1. Numerical deviation  — |local value − global value|")
	fmt.Println("  2. Staleness deviation  — age of last synchronised value")
	fmt.Println("  3. Ordering deviation   — number of tentative unordered writes")
	fmt.Println()
	fmt.Println("  A conit defines the bounds on all three axes for one data item.")
	fmt.Println("  Sync fires automatically when any bound is exceeded.")
	fmt.Println()

	scenarioNumerical()
	scenarioStaleness()
	scenarioOrdering()
	scenarioMultiConit()
	scenarioTimeline()
	summary()
}
