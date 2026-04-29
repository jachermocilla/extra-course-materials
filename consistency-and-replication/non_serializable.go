package main

import (
	"fmt"
	"strings"
)

// ============================================================
// Execution timeline:
//
//   ───t1──────t2──────t3──────t4──────t5──────t6───
//   P1    W1(x)a                 W1(y)a  R1(x)b
//   P2             W2(y)b W2(x)b                 R2(y)a
//
// P1 program order: W1(x)a → W1(y)a → R1(x)b
// P2 program order: W2(y)b → W2(x)b → R2(y)a
//
// Reads:
//   R1(x)b  — P1 reads x and gets b  (P2's value)
//   R2(y)a  — P2 reads y and gets a  (P1's value)
// ============================================================

type Op struct {
	slot    int    // time slot t1..t6
	pid     int    // process 1 or 2
	kind    string // "W" or "R"
	varName string
	value   string
	label   string
}

func makeOp(slot, pid int, kind, varName, value string) Op {
	label := fmt.Sprintf("%s%d(%s)%s", kind, pid, varName, value)
	return Op{slot, pid, kind, varName, value, label}
}

// ── Timeline printer ─────────────────────────────────────────────────────────

func printTimeline(ops []Op, totalSlots int) {
	colW := 10
	dash := strings.Repeat("─", colW)

	// header
	hdr := fmt.Sprintf("  %-4s  ", "")
	for s := 1; s <= totalSlots; s++ {
		hdr += centerPad(fmt.Sprintf("t%d", s), colW)
	}
	fmt.Println(hdr)
	fmt.Printf("  %-4s  %s\n", "", strings.Repeat("─", colW*totalSlots))

	// one row per process
	for pid := 1; pid <= 2; pid++ {
		row := fmt.Sprintf("  P%-3d  ", pid)
		for s := 1; s <= totalSlots; s++ {
			cell := ""
			for _, op := range ops {
				if op.pid == pid && op.slot == s {
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

func centerPad(s string, width int) string {
	if len(s) >= width {
		return s
	}
	total := width - len(s)
	left := total / 2
	right := total - left
	return strings.Repeat("─", left) + s + strings.Repeat("─", right)
}

// ── SC checker for a single variable ────────────────────────────────────────
// Tries all write permutations (respecting per-process program order).
// A permutation satisfies SC if every read returns the value of the most
// recent preceding write in that permutation order.

func checkSC(varName string, ops []Op) bool {
	var writes, reads []Op
	for _, op := range ops {
		if op.kind == "W" {
			writes = append(writes, op)
		} else {
			reads = append(reads, op)
		}
	}

	fmt.Printf("  Operations on %s (execution order):\n", varName)
	for _, op := range ops {
		fmt.Printf("    t%d  %s\n", op.slot, op.label)
	}
	fmt.Println()

	perms := writePerms(writes)
	fmt.Printf("  Trying %d write orderings:\n\n", len(perms))

	for _, perm := range perms {
		labels := make([]string, len(perm))
		for i, w := range perm {
			labels[i] = w.label
		}
		fmt.Printf("    Order: %s\n", strings.Join(labels, " → "))

		allOk := true
		for _, read := range reads {
			// find the last write in perm order whose slot < read.slot
			lastVal := "NIL"
			for _, wr := range perm {
				if wr.slot < read.slot {
					lastVal = wr.value
				}
			}
			ok := lastVal == read.value
			sym := "✅"
			if !ok {
				sym = "✗ "
				allOk = false
			}
			fmt.Printf("      %s: last write before t%d = %s, read returns %s  %s\n",
				read.label, read.slot, lastVal, read.value, sym)
		}
		if allOk {
			fmt.Printf("      → ✅ SC holds for %s under order [%s]\n\n",
				varName, strings.Join(labels, " → "))
			return true
		}
		fmt.Println()
	}
	fmt.Printf("  → ✗  No valid ordering found. SC does NOT hold for %s.\n\n", varName)
	return false
}

// writePerms generates all permutations of writes respecting per-process order
func writePerms(writes []Op) [][]Op {
	var result [][]Op
	var gen func(cur, rem []Op)
	gen = func(cur, rem []Op) {
		if len(rem) == 0 {
			cp := make([]Op, len(cur))
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
				next := append(append([]Op{}, rem[:i]...), rem[i+1:]...)
				gen(append(cur, op), next)
			}
		}
	}
	gen([]Op{}, writes)
	return result
}

func sep(char string) {
	fmt.Printf("%s\n", strings.Repeat(char, 70))
}

func main() {
	// ── Define the execution ─────────────────────────────────────────────────
	ops := []Op{
		makeOp(1, 1, "W", "x", "a"), // t1  P1 writes x=a
		makeOp(2, 2, "W", "y", "b"), // t2  P2 writes y=b
		makeOp(3, 2, "W", "x", "b"), // t3  P2 writes x=b
		makeOp(4, 1, "W", "y", "a"), // t4  P1 writes y=a
		makeOp(5, 1, "R", "x", "b"), // t5  P1 reads  x → b
		makeOp(6, 2, "R", "y", "a"), // t6  P2 reads  y → a
	}

	sep("═")
	fmt.Println("EXECUTION TIMELINE")
	sep("═")
	fmt.Println()
	printTimeline(ops, 6)

	fmt.Println("  P1 program order: W1(x)a → W1(y)a → R1(x)b")
	fmt.Println("  P2 program order: W2(y)b → W2(x)b → R2(y)a")
	fmt.Println()
	fmt.Println("  Final state: x=b (W2(x)b at t3 was last write to x)")
	fmt.Println("               y=a (W1(y)a at t4 was last write to y)")

	// ── Serializability check ────────────────────────────────────────────────
	fmt.Println()
	sep("═")
	fmt.Println("SERIALIZABILITY CHECK")
	sep("═")
	fmt.Println()
	fmt.Println("  Simulate both serial orders and compare final state + reads:\n")

	fmt.Println("  ┌─ Serial order P1 → P2 ───────────────────────────────────┐")
	fmt.Println("  │  P1 runs fully: W1(x)a, W1(y)a  →  x=a, y=a             │")
	fmt.Println("  │  P1 reads x: sees a  (R1(x)=a)                           │")
	fmt.Println("  │  P2 runs fully: W2(y)b, W2(x)b  →  x=b, y=b             │")
	fmt.Println("  │  P2 reads y: sees b  (R2(y)=b)                           │")
	fmt.Println("  │  Final state: x=b, y=b                                   │")
	fmt.Println("  └───────────────────────────────────────────────────────────┘")
	fmt.Println()
	fmt.Println("  ┌─ Serial order P2 → P1 ───────────────────────────────────┐")
	fmt.Println("  │  P2 runs fully: W2(y)b, W2(x)b  →  x=b, y=b             │")
	fmt.Println("  │  P2 reads y: sees b  (R2(y)=b)                           │")
	fmt.Println("  │  P1 runs fully: W1(x)a, W1(y)a  →  x=a, y=a             │")
	fmt.Println("  │  P1 reads x: sees a  (R1(x)=a)                           │")
	fmt.Println("  │  Final state: x=a, y=a                                   │")
	fmt.Println("  └───────────────────────────────────────────────────────────┘")
	fmt.Println()
	fmt.Println("  Interleaved execution reads:  R1(x)=b,  R2(y)=a")
	fmt.Println("  P1→P2 reads:                  R1(x)=a,  R2(y)=b  ✗ both differ")
	fmt.Println("  P2→P1 reads:                  R1(x)=a,  R2(y)=b  ✗ both differ")
	fmt.Println()
	fmt.Println("  ✗  Neither serial order reproduces the observed reads.")
	fmt.Println("  ✗  Execution is NOT serializable.")

	// ── Project onto x ───────────────────────────────────────────────────────
	fmt.Println()
	sep("═")
	fmt.Println("PROJECT ONTO x ONLY — SEQUENTIAL CONSISTENCY CHECK")
	sep("═")
	fmt.Println()

	var xOps []Op
	for _, op := range ops {
		if op.varName == "x" {
			xOps = append(xOps, op)
		}
	}
	fmt.Println("  Timeline for x:")
	printTimeline(xOps, 6)
	scX := checkSC("x", xOps)

	// ── Project onto y ───────────────────────────────────────────────────────
	fmt.Println()
	sep("═")
	fmt.Println("PROJECT ONTO y ONLY — SEQUENTIAL CONSISTENCY CHECK")
	sep("═")
	fmt.Println()

	var yOps []Op
	for _, op := range ops {
		if op.varName == "y" {
			yOps = append(yOps, op)
		}
	}
	fmt.Println("  Timeline for y:")
	printTimeline(yOps, 6)
	scY := checkSC("y", yOps)

	// ── Conclusion ───────────────────────────────────────────────────────────
	fmt.Println()
	sep("═")
	fmt.Println("CONCLUSION")
	sep("═")
	fmt.Println()
	fmt.Printf("  %-10s  %-30s\n", "Variable", "Sequential Consistency?")
	fmt.Printf("  %-10s  %-30s\n", "──────────", "──────────────────────────────")
	mark := func(ok bool) string {
		if ok { return "✅ yes" }
		return "✗  no"
	}
	fmt.Printf("  %-10s  %s\n", "x alone", mark(scX))
	fmt.Printf("  %-10s  %s\n", "y alone", mark(scY))
	fmt.Printf("  %-10s  %s\n", "x + y", "✗  NOT serializable")
	fmt.Println()
	fmt.Println("  Why SC holds per variable but serializability fails:")
	fmt.Println()
	fmt.Println("  x alone: write order W2(x)b → W1(x)a explains R1(x)=b  ✅")
	fmt.Println("           (implies P2 precedes P1 on x)")
	fmt.Println()
	fmt.Println("  y alone: write order W1(y)a → W2(y)b explains R2(y)=a  ✅")
	fmt.Println("           (implies P1 precedes P2 on y)")
	fmt.Println()
	fmt.Println("  Combined: x needs P2 → P1,  y needs P1 → P2.")
	fmt.Println("  These are contradictory — no single total process order")
	fmt.Println("  satisfies both simultaneously.")
	fmt.Println("  SC is per-variable; serializability requires a global order.")
	fmt.Println()
}
