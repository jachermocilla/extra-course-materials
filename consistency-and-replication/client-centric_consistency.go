package main

import (
	"fmt"
	"strings"
)

// ============================================================
// Client-Centric Consistency Models
//
// Four guarantees, each preventing a specific client anomaly:
//   1. Monotonic Reads    — reads never go backwards
//   2. Monotonic Writes   — writes applied in issue order
//   3. Read Your Writes   — client always sees its own writes
//   4. Writes Follow Reads— writes grounded in what was read
//
// Implementation:
//   Each replica tracks a write-set: the set of writes it has
//   applied, identified by (clientID, writeSeq) pairs.
//   Each client carries a read-set RS and write-set WS in its
//   session state. Before serving a client, a replica checks
//   whether it has applied all writes the client requires.
// ============================================================

// ── Write identifier ──────────────────────────────────────────────────────────

type WriteID struct {
	clientID int
	seq      int // monotonically increasing per client
}

func (w WriteID) String() string {
	return fmt.Sprintf("C%d#%d", w.clientID, w.seq)
}

// ── Replica ───────────────────────────────────────────────────────────────────

type Replica struct {
	id      int
	store   map[string]string  // key → value
	applied map[WriteID]bool   // writes this replica has applied
}

func NewReplica(id int) *Replica {
	return &Replica{
		id:      id,
		store:   make(map[string]string),
		applied: make(map[WriteID]bool),
	}
}

// applyWrite commits a write directly (simulates propagation arriving)
func (r *Replica) applyWrite(wid WriteID, key, value string) {
	r.store[key] = value
	r.applied[wid] = true
}

// hasAll returns true if this replica has applied all writes in the set
func (r *Replica) hasAll(wids []WriteID) bool {
	for _, wid := range wids {
		if !r.applied[wid] {
			return false
		}
	}
	return true
}

// read returns the value for key, or NIL
func (r *Replica) read(key string) string {
	v := r.store[key]
	if v == "" { return "NIL" }
	return v
}

// pullFrom simulates this replica pulling specific writes from another
func (r *Replica) pullFrom(src *Replica, wids []WriteID) {
	for _, wid := range wids {
		if !r.applied[wid] && src.applied[wid] {
			// find the write's effect — scan src store keys
			// (in a real system the write log would carry key/value)
			for k, v := range src.store {
				_ = k; _ = v // placeholder; we use explicit applyWrite below
			}
			r.applied[wid] = true
		}
	}
}

// ── Session state (per client) ────────────────────────────────────────────────
// RS = read-set:  writes the client has READ (for monotonic reads)
// WS = write-set: writes the client has ISSUED (for read-your-writes etc.)

type Session struct {
	clientID int
	writeSeq int        // next write sequence number
	RS       []WriteID  // reads set: writes seen via reads
	WS       []WriteID  // write set: writes this client issued
}

func NewSession(clientID int) *Session {
	return &Session{clientID: clientID}
}

func (s *Session) nextWriteID() WriteID {
	s.writeSeq++
	return WriteID{s.clientID, s.writeSeq}
}

func (s *Session) addToWS(wid WriteID) { s.WS = append(s.WS, wid) }
func (s *Session) addToRS(wids []WriteID) {
	for _, wid := range wids {
		if !s.inRS(wid) { s.RS = append(s.RS, wid) }
	}
}
func (s *Session) inRS(wid WriteID) bool {
	for _, w := range s.RS { if w == wid { return true } }
	return false
}
func (s *Session) wsLabels() string {
	if len(s.WS) == 0 { return "∅" }
	var parts []string
	for _, w := range s.WS { parts = append(parts, w.String()) }
	return "{" + strings.Join(parts, ",") + "}"
}
func (s *Session) rsLabels() string {
	if len(s.RS) == 0 { return "∅" }
	var parts []string
	for _, w := range s.RS { parts = append(parts, w.String()) }
	return "{" + strings.Join(parts, ",") + "}"
}

// ── Timeline printer ──────────────────────────────────────────────────────────

type Op struct {
	slot    int
	subject string // "C1","R1","R2","R3"
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
	hdr := fmt.Sprintf("  %-4s  ", "")
	for s := 1; s <= totalSlots; s++ {
		hdr += centerPad(fmt.Sprintf("t%d", s), colW)
	}
	fmt.Println(hdr)
	fmt.Printf("  %-4s  %s\n", "", strings.Repeat("─", colW*totalSlots))
	for _, row := range rows {
		line := fmt.Sprintf("  %-4s  ", row)
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

// ── SCENARIO 1: Monotonic Reads ───────────────────────────────────────────────
//
// Client reads x=a from R1 (which has received a recent write).
// Client then moves to R2 (which is still stale, x=NIL).
// Without monotonic reads: R2 returns NIL — clock goes backwards.
// With monotonic reads: R2 must pull the write before serving the read.

func scenarioMonotonicReads() {
	sep("═", 72)
	fmt.Println("SCENARIO 1 — Monotonic Reads")
	sep("═", 72)
	fmt.Println()
	fmt.Println("  Guarantee: a client never reads an older version after")
	fmt.Println("  having read a newer one. RS tracks what the client has seen.")
	fmt.Println()

	ops := []Op{
		{1, "R1", "W(x)a[C0#1]"},
		{2, "C1", "R(x)→a"},
		{2, "R1", "serve R(x)=a"},
		{3, "C1", "→ R2"},
		{4, "R2", "pull C0#1"},
		{4, "C1", "R(x)→a"},
		{4, "R2", "serve R(x)=a"},
	}
	printTimeline(ops, []string{"C1", "R1", "R2"}, 5)

	r1 := NewReplica(1)
	r2 := NewReplica(2)
	sess := NewSession(1)

	// Some other client (C0) wrote x=a to R1
	wid0 := WriteID{0, 1}
	r1.applyWrite(wid0, "x", "a")

	fmt.Println("  ── Without monotonic reads ──────────────────────────────────")
	fmt.Println()
	v1 := r1.read("x")
	fmt.Printf("  C1 reads R1: R(x)=%s\n", v1)
	v2 := r2.read("x")
	fmt.Printf("  C1 moves to R2, reads: R(x)=%s  ⚠️  went backwards!\n", v2)

	fmt.Println()
	fmt.Println("  ── With monotonic reads ─────────────────────────────────────")
	fmt.Println()

	// C1 reads from R1 — update session RS with writes R1 has applied
	v1 = r1.read("x")
	// session records which writes R1 had applied at time of this read
	sess.addToRS([]WriteID{wid0})
	fmt.Printf("  C1 reads R1: R(x)=%s  →  RS=%s\n", v1, sess.rsLabels())

	// C1 moves to R2 — check if R2 satisfies RS
	fmt.Printf("  C1 moves to R2.  RS=%s\n", sess.rsLabels())
	if !r2.hasAll(sess.RS) {
		fmt.Println("  R2 does not satisfy RS — pulling required writes from R1...")
		for _, wid := range sess.RS {
			if !r2.applied[wid] {
				// propagate the specific write
				r2.applyWrite(wid, "x", r1.store["x"])
				fmt.Printf("  R2 pulls write %s: x=%s\n", wid, r1.store["x"])
			}
		}
	}
	v2 = r2.read("x")
	fmt.Printf("  C1 reads R2: R(x)=%s  ✅ monotonic — same or newer value\n", v2)
	fmt.Println()
}

// ── SCENARIO 2: Monotonic Writes ─────────────────────────────────────────────
//
// Client writes x=a then x=b (in that order).
// Without MW: R2 applies x=b before x=a (wrong final value).
// With MW: WS ensures R2 applies x=a before x=b.

func scenarioMonotonicWrites() {
	sep("═", 72)
	fmt.Println("SCENARIO 2 — Monotonic Writes")
	sep("═", 72)
	fmt.Println()
	fmt.Println("  Guarantee: a client's writes are applied in issue order")
	fmt.Println("  on every replica. WS tracks the client's write history.")
	fmt.Println()

	ops := []Op{
		{1, "C1", "W(x)a"},
		{1, "R1", "apply W(x)a"},
		{2, "C1", "W(x)b"},
		{2, "R1", "apply W(x)b"},
		{3, "R2", "recv W(x)b"},  // out-of-order arrival
		{4, "R2", "recv W(x)a"},  // late arrival
		{5, "R2", "x=a ⚠️"},      // wrong without MW
	}
	printTimeline(ops, []string{"C1", "R1", "R2"}, 6)

	r1 := NewReplica(1)
	r2 := NewReplica(2)
	sess := NewSession(1)

	fmt.Println("  ── Without monotonic writes ─────────────────────────────────")
	fmt.Println()
	// Client writes x=a then x=b
	wid1 := WriteID{1, 1}
	wid2 := WriteID{1, 2}
	r1.applyWrite(wid1, "x", "a")
	r1.applyWrite(wid2, "x", "b")
	fmt.Printf("  C1 → R1: W(x)a [%s], W(x)b [%s]\n", wid1, wid2)

	// R2 receives b first (faster path), then a (slower path) — wrong order
	r2.applyWrite(wid2, "x", "b") // arrives first
	r2.applyWrite(wid1, "x", "a") // arrives second — OVERWRITES b!
	fmt.Printf("  R2 receives W(x)b then W(x)a out of order → R2: x=%s  ⚠️\n", r2.read("x"))

	fmt.Println()
	fmt.Println("  ── With monotonic writes ────────────────────────────────────")
	fmt.Println()

	r2b := NewReplica(2)
	// Issue write 1: x=a
	wid1 = sess.nextWriteID()
	r1.applyWrite(wid1, "x", "a")
	sess.addToWS(wid1)
	fmt.Printf("  C1: W(x)a  wid=%s  WS=%s\n", wid1, sess.wsLabels())

	// Issue write 2: x=b — WS now contains wid1 as prerequisite
	wid2 = sess.nextWriteID()
	r1.applyWrite(wid2, "x", "b")
	sess.addToWS(wid2)
	fmt.Printf("  C1: W(x)b  wid=%s  WS=%s\n", wid2, sess.wsLabels())

	// R2 receives wid2 first — checks if its WS prerequisite (wid1) is satisfied
	fmt.Println()
	fmt.Printf("  R2 receives W(x)b [%s] first.\n", wid2)
	fmt.Printf("  Prerequisite check: needs %s first.\n", wid1)
	if !r2b.applied[wid1] {
		fmt.Printf("  R2 does not have %s — buffering W(x)b, pulling %s first.\n", wid1, wid1)
		r2b.applyWrite(wid1, "x", "a")
		fmt.Printf("  R2 applies %s: x=a\n", wid1)
		r2b.applyWrite(wid2, "x", "b")
		fmt.Printf("  R2 applies %s: x=b\n", wid2)
	}
	fmt.Printf("  R2 final: x=%s  ✅ correct order preserved\n", r2b.read("x"))
	fmt.Println()
}

// ── SCENARIO 3: Read Your Writes ──────────────────────────────────────────────
//
// Client writes x=a to R1, then reads x from R2 (stale replica).
// Without RYW: R2 returns NIL — client does not see its own write.
// With RYW: R2 checks client WS and pulls the write before serving.

func scenarioReadYourWrites() {
	sep("═", 72)
	fmt.Println("SCENARIO 3 — Read Your Writes")
	sep("═", 72)
	fmt.Println()
	fmt.Println("  Guarantee: a client always sees the effect of its own writes,")
	fmt.Println("  even when reading from a different replica.")
	fmt.Println()

	ops := []Op{
		{1, "C1", "W(x)a"},
		{1, "R1", "apply W(x)a"},
		{2, "C1", "→ R2"},
		{3, "R2", "check WS"},
		{3, "R2", "pull C1#1"},
		{4, "C1", "R(x)→a"},
		{4, "R2", "serve R(x)=a"},
	}
	printTimeline(ops, []string{"C1", "R1", "R2"}, 5)

	r1 := NewReplica(1)
	r2 := NewReplica(2)
	sess := NewSession(1)

	fmt.Println("  ── Without read-your-writes ─────────────────────────────────")
	fmt.Println()
	wid := WriteID{1, 1}
	r1.applyWrite(wid, "x", "a")
	fmt.Printf("  C1 → R1: W(x)a\n")
	fmt.Printf("  C1 → R2: R(x)=%s  ⚠️  client does not see its own write\n", r2.read("x"))

	fmt.Println()
	fmt.Println("  ── With read-your-writes ────────────────────────────────────")
	fmt.Println()

	wid = sess.nextWriteID()
	r1.applyWrite(wid, "x", "a")
	sess.addToWS(wid)
	fmt.Printf("  C1 → R1: W(x)a  wid=%s  WS=%s\n", wid, sess.wsLabels())

	// Client moves to R2 and reads — R2 must satisfy WS
	fmt.Printf("  C1 → R2: R(x)?  WS=%s\n", sess.wsLabels())
	if !r2.hasAll(sess.WS) {
		fmt.Println("  R2 does not satisfy WS — pulling C1's write from R1...")
		for _, w := range sess.WS {
			if !r2.applied[w] {
				r2.applyWrite(w, "x", r1.store["x"])
				fmt.Printf("  R2 pulls %s: x=%s\n", w, r1.store["x"])
			}
		}
	}
	fmt.Printf("  C1 reads R2: R(x)=%s  ✅ client sees its own write\n", r2.read("x"))
	fmt.Println()
}

// ── SCENARIO 4: Writes Follow Reads ──────────────────────────────────────────
//
// Client reads x=a from R1, then writes y=b to R2 based on that read.
// Without WFR: R3 may apply W(y)b before it has seen x=a — inconsistent.
// With WFR: the write on y carries the RS from the read, ensuring
//           any replica that applies W(y)b must first have x=a.

func scenarioWritesFollowReads() {
	sep("═", 72)
	fmt.Println("SCENARIO 4 — Writes Follow Reads  (Session Causality)")
	sep("═", 72)
	fmt.Println()
	fmt.Println("  Guarantee: if a client reads x then writes y, the write on y")
	fmt.Println("  is only applied on replicas that already have the read's version")
	fmt.Println("  of x. The write is grounded in what was read.")
	fmt.Println()

	ops := []Op{
		{1, "R1", "W(x)a[C0#1]"},
		{2, "C1", "R(x)→a"},
		{2, "R1", "serve R(x)=a"},
		{3, "C1", "W(y)b"},
		{3, "R2", "apply W(y)b"},
		{4, "R3", "check RS"},
		{4, "R3", "pull C0#1"},
		{5, "R3", "apply W(y)b"},
	}
	printTimeline(ops, []string{"C1", "R1", "R2", "R3"}, 6)

	r1 := NewReplica(1)
	r2 := NewReplica(2)
	r3 := NewReplica(3)
	sess := NewSession(1)

	// C0 wrote x=a to R1
	wid0 := WriteID{0, 1}
	r1.applyWrite(wid0, "x", "a")
	r2.applyWrite(wid0, "x", "a") // R2 has it

	fmt.Println("  ── Without writes-follow-reads ──────────────────────────────")
	fmt.Println()
	fmt.Printf("  C1 reads R1: R(x)=%s\n", r1.read("x"))
	fmt.Printf("  C1 writes R2: W(y)b\n")
	wid1 := WriteID{1, 1}
	r2.applyWrite(wid1, "y", "b")
	// R3 receives W(y)b but may not have x=a yet
	fmt.Printf("  R3 receives W(y)b but has x=%s  ⚠️  y=b without context of x=a\n", r3.read("x"))

	fmt.Println()
	fmt.Println("  ── With writes-follow-reads ─────────────────────────────────")
	fmt.Println()

	r3b := NewReplica(3)
	// C1 reads x from R1 — records wid0 in session RS
	v := r1.read("x")
	sess.addToRS([]WriteID{wid0})
	fmt.Printf("  C1 reads R1: R(x)=%s  →  RS=%s\n", v, sess.rsLabels())

	// C1 writes y=b — the write carries the current RS as prerequisite
	wid1 = sess.nextWriteID()
	sess.addToWS(wid1)
	// The write record carries RS as prerequisite
	writePrereqs := append([]WriteID{}, sess.RS...)
	fmt.Printf("  C1 writes W(y)b  wid=%s  carrying prereqs=%v\n\n",
		wid1, writePrereqs)

	// R3 receives W(y)b — checks if it satisfies the prereqs (RS of the read)
	fmt.Printf("  R3 receives W(y)b.  Prereqs required: %v\n", writePrereqs)
	if !r3b.hasAll(writePrereqs) {
		fmt.Println("  R3 does not satisfy prereqs — pulling required writes first...")
		for _, wid := range writePrereqs {
			if !r3b.applied[wid] {
				r3b.applyWrite(wid, "x", r1.store["x"])
				fmt.Printf("  R3 pulls %s: x=%s\n", wid, r1.store["x"])
			}
		}
	}
	r3b.applyWrite(wid1, "y", "b")
	fmt.Printf("  R3 applies W(y)b: y=%s\n", r3b.read("y"))
	fmt.Printf("  R3 state: x=%s  y=%s  ✅ write grounded in correct read context\n",
		r3b.read("x"), r3b.read("y"))
	fmt.Println()
}

// ── SCENARIO 5: All Four Together ────────────────────────────────────────────
// Show a session where all four guarantees are exercised in sequence,
// and how the session state (RS, WS) evolves.

func scenarioAllFour() {
	sep("═", 72)
	fmt.Println("SCENARIO 5 — All Four Guarantees in One Session")
	sep("═", 72)
	fmt.Println()
	fmt.Println("  One client, three replicas, sequential operations.")
	fmt.Println("  We track RS and WS at each step and show which guarantee fires.")
	fmt.Println()

	r1 := NewReplica(1)
	r2 := NewReplica(2)
	r3 := NewReplica(3)
	sess := NewSession(1)

	// Seed: some earlier write exists on R1
	wid0 := WriteID{0, 1}
	r1.applyWrite(wid0, "x", "initial")

	type step struct {
		num       int
		guarantee string
		action    string
		rs        string
		ws        string
		result    string
	}
	var steps []step

	record := func(n int, guarantee, action, result string) {
		steps = append(steps, step{n, guarantee, action, sess.rsLabels(), sess.wsLabels(), result})
	}

	// Step 1: Read x from R1  → Monotonic Read — RS updated
	v := r1.read("x")
	sess.addToRS([]WriteID{wid0})
	record(1, "Monotonic Read", "C1 reads R1: R(x)", fmt.Sprintf("x=%s  RS updated", v))

	// Step 2: Write x=new to R2  → Monotonic Write / RYW — WS updated
	wid1 := sess.nextWriteID()
	r2.applyWrite(wid1, "x", "new")
	sess.addToWS(wid1)
	record(2, "RYW + Mono Write", "C1 writes R2: W(x)new", fmt.Sprintf("wid=%s  WS updated", wid1))

	// Step 3: Read x from R3  → Read Your Writes — R3 must pull wid1
	if !r3.hasAll(sess.WS) {
		for _, w := range sess.WS {
			if !r3.applied[w] { r3.applyWrite(w, "x", "new") }
		}
	}
	// Also Monotonic Read — R3 must have wid0 too
	if !r3.hasAll(sess.RS) {
		for _, w := range sess.RS {
			if !r3.applied[w] { r3.applyWrite(w, "x", r1.store["x"]) }
		}
	}
	v = r3.read("x")
	sess.addToRS([]WriteID{wid1})
	record(3, "RYW + Mono Read", "C1 reads R3: R(x)", fmt.Sprintf("x=%s  R3 pulled WS before serving", v))

	// Step 4: Write z=derived  → Writes Follow Reads — carries RS
	wid2 := sess.nextWriteID()
	prereqs := append([]WriteID{}, sess.RS...)
	r1.applyWrite(wid2, "z", "derived")
	sess.addToWS(wid2)
	record(4, "Writes Follow Reads", "C1 writes R1: W(z)derived",
		fmt.Sprintf("wid=%s  prereqs=%v", wid2, prereqs))

	// Step 5: Read z from R2  → Monotonic Read — R2 must pull wid2
	if !r2.hasAll(sess.RS) || !r2.applied[wid2] {
		r2.applyWrite(wid2, "z", "derived")
	}
	v = r2.read("z")
	sess.addToRS([]WriteID{wid2})
	record(5, "Monotonic Read", "C1 reads R2: R(z)", fmt.Sprintf("z=%s  R2 pulled wid2", v))

	// Print the session table
	fmt.Printf("  %-4s  %-20s  %-30s  %-14s  %-14s  %s\n",
		"Step", "Guarantee", "Action", "RS", "WS", "Result")
	fmt.Printf("  %-4s  %-20s  %-30s  %-14s  %-14s  %s\n",
		"────", strings.Repeat("─", 20), strings.Repeat("─", 30),
		strings.Repeat("─", 14), strings.Repeat("─", 14), strings.Repeat("─", 30))
	for _, s := range steps {
		fmt.Printf("  %-4d  %-20s  %-30s  %-14s  %-14s  %s\n",
			s.num, s.guarantee, s.action, s.rs, s.ws, s.result)
	}
	fmt.Println()
}

// ── Summary ───────────────────────────────────────────────────────────────────

func summary() {
	sep("═", 72)
	fmt.Println("SUMMARY — Client-Centric Consistency Models")
	sep("═", 72)
	fmt.Println()
	fmt.Printf("  %-22s  %-14s  %-14s  %-20s  %s\n",
		"Model", "Session state", "Anomaly prevented", "Mechanism", "Cost")
	fmt.Printf("  %-22s  %-14s  %-14s  %-20s  %s\n",
		strings.Repeat("─", 22), strings.Repeat("─", 14),
		strings.Repeat("─", 18), strings.Repeat("─", 20), strings.Repeat("─", 16))
	rows := [][]string{
		{"Monotonic Reads",    "RS",     "clock backwards",    "replica pulls RS",     "pull on read"},
		{"Monotonic Writes",   "WS",     "out-of-order apply", "replica orders by WS", "order check"},
		{"Read Your Writes",   "WS",     "own write invisible","replica pulls WS",     "pull on read"},
		{"Writes Follow Reads","RS+WS",  "stale write context","write carries RS",     "pull on write"},
	}
	for _, r := range rows {
		fmt.Printf("  %-22s  %-14s  %-18s  %-20s  %s\n",
			r[0], r[1], r[2], r[3], r[4])
	}
	fmt.Println()
	fmt.Println("  Key: RS = read-set  (writes seen via reads)")
	fmt.Println("       WS = write-set (writes issued by this client)")
	fmt.Println()
	fmt.Println("  The four guarantees are independent and composable.")
	fmt.Println("  All four together ≈ session causality — cheap causal consistency")
	fmt.Println("  scoped to a single client, not the entire system.")
	fmt.Println()
	fmt.Println("  Placement in the consistency spectrum:")
	fmt.Printf("  %-28s  %s\n", "Model", "Strength")
	fmt.Printf("  %-28s  %s\n", strings.Repeat("─", 28), strings.Repeat("─", 28))
	fmt.Printf("  %-28s  %s\n", "Linearizability",         "strongest — global total order")
	fmt.Printf("  %-28s  %s\n", "Sequential consistency",  "global total order, no real-time")
	fmt.Printf("  %-28s  %s\n", "Causal consistency",      "all processes, causal order")
	fmt.Printf("  %-28s  %s\n", "All 4 client-centric",    "per-client session causality")
	fmt.Printf("  %-28s  %s\n", "Eventual consistency",    "weakest — converge eventually")
	fmt.Println()
}

func main() {
	sep("═", 72)
	fmt.Println("CLIENT-CENTRIC CONSISTENCY MODELS")
	sep("═", 72)
	fmt.Println()
	fmt.Println("  Focus: the individual client's view across replicas.")
	fmt.Println("  Each model prevents one class of anomaly a client might observe.")
	fmt.Println()
	fmt.Println("  Session state tracks two sets per client:")
	fmt.Println("    RS (read-set)  — identifiers of writes seen via reads")
	fmt.Println("    WS (write-set) — identifiers of writes this client issued")
	fmt.Println()
	fmt.Println("  Before serving a client, a replica checks RS and/or WS")
	fmt.Println("  and pulls missing writes if necessary.")
	fmt.Println()

	scenarioMonotonicReads()
	scenarioMonotonicWrites()
	scenarioReadYourWrites()
	scenarioWritesFollowReads()
	scenarioAllFour()
	summary()
}
