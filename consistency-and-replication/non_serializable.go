package main

import (
	"fmt"
	"strings"
)

// Execution timeline:
//   t1: W1(x)a   t2: W2(y)b   t3: W2(x)b   t4: W1(y)a   t5: R1(x)b   t6: R2(y)a
//
// Writes: W1(x)a, W2(y)b, W2(x)b, W1(y)a
// Reads:  R1(x)a at t5,  R2(y)b at t5
//
// We enumerate ALL 4! = 24 orderings of the four writes.
// For each ordering, simulate the reads:
//   R1(x)b at t5: find last write to x in ordering whose slot < 5
//   R2(y)a at t6: find last write to y in ordering whose slot < 6
// Report whether each read is satisfied.

type Write struct {
	label   string
	varName string
	value   string
	slot    int // real execution slot
}

type Read struct {
	label    string
	varName  string
	expected string
	slot     int
}

var writes = []Write{
	{"W1(x)a", "x", "a", 1},
	{"W2(y)b", "y", "b", 2},
	{"W2(x)b", "x", "b", 3},
	{"W1(y)a", "y", "a", 4},
}

var reads = []Read{
	{"R1(x)a", "x", "a", 5},
	{"R2(y)b", "y", "b", 6},
}

// permutations of indices 0..n-1
func permutations(n int) [][]int {
	var result [][]int
	a := make([]int, n)
	for i := range a { a[i] = i }
	var gen func(k int)
	gen = func(k int) {
		if k == 1 {
			cp := make([]int, n)
			copy(cp, a)
			result = append(result, cp)
			return
		}
		for i := 0; i < k; i++ {
			gen(k - 1)
			if k%2 == 0 {
				a[i], a[k-1] = a[k-1], a[i]
			} else {
				a[0], a[k-1] = a[k-1], a[0]
			}
		}
	}
	gen(n)
	return result
}

// simulate: given a write ordering (by index), compute what each read sees.
// The write ordering defines the global total order of writes.
// A read at slot s sees the value of the last write to its variable
// that appears BEFORE slot s in the real execution (slot < s),
// using the write ordering to break ties / define "last".
//
// More precisely: among all writes to the variable whose real slot < read.slot,
// the "last" one is the one that appears LATEST in the write ordering.
func simulate(order []int) (readResults [2]string) {
	for ri, rd := range reads {
		lastVal := "NIL"
		lastPos := -1 // position in the ordering
		for pos, wi := range order {
			w := writes[wi]
			if w.varName == rd.varName && w.slot < rd.slot {
				if pos > lastPos {
					lastPos = pos
					lastVal = w.value
				}
			}
		}
		readResults[ri] = lastVal
	}
	return
}

// respects program order: within each process, writes maintain their
// original relative order (W1(x)a before W1(y)a; W2(y)b before W2(x)b)
func respectsProgramOrder(order []int) bool {
	posOf := make(map[int]int)
	for pos, wi := range order { posOf[wi] = pos }
	// P1: write index 0 (W1(x)a) must come before index 3 (W1(y)a)
	// P2: write index 1 (W2(y)b) must come before index 2 (W2(x)b)
	return posOf[0] < posOf[3] && posOf[1] < posOf[2]
}

func main() {
	// Print the timeline first
	fmt.Println("Timeline:")
	fmt.Println()
	fmt.Println("        ────t1────────t2────────t3────────t4────────t5────────t6────")
	fmt.Println("        ────────────────────────────────────────────────────────────")
	fmt.Println("  P1    ──W1(x)a────────────────────────W1(y)a────R1(x)a────────────")
	fmt.Println("  P2    ────────────W2(y)b────W2(x)b──────────────R2(y)b────────────")
	fmt.Println()
	fmt.Println("  Reads to satisfy:  R1(x)a at t5,  R2(y)b at t5")
	fmt.Println()

	// Column widths
	orderW := 46
	r1W := 9
	r2W := 9
	satW := 14
	poW := 3

	// Header
	sepLine := fmt.Sprintf("  %-3s  %-*s  %-*s  %-*s  %-*s  %s",
		"───", orderW, strings.Repeat("─", orderW),
		r1W, strings.Repeat("─", r1W),
		r2W, strings.Repeat("─", r2W),
		satW, strings.Repeat("─", satW),
		strings.Repeat("─", poW))
	fmt.Println("  All 24 write orderings (W1(x)a, W2(y)b, W2(x)b, W1(y)a)")
	fmt.Println()
	fmt.Printf("  %-3s  %-*s  %-*s  %-*s  %-*s  %s\n",
		"#", orderW, "Write ordering",
		r1W, "R1(x)=?",
		r2W, "R2(y)=?",
		satW, "Reads match?",
		"PO?")
	fmt.Println(sepLine)

	allPerms := permutations(4)
	count := 0
	matchCount := 0
	poCount := 0
	matchAndPO := 0

	for _, perm := range allPerms {
		count++
		labels := make([]string, 4)
		for i, wi := range perm { labels[i] = writes[wi].label }
		orderStr := strings.Join(labels, " → ")

		results := simulate(perm)
		r1Got := results[0]
		r2Got := results[1]

		r1Match := r1Got == reads[0].expected
		r2Match := r2Got == reads[1].expected
		bothMatch := r1Match && r2Match
		po := respectsProgramOrder(perm)

		satStr := "─"
		if r1Match && r2Match {
			satStr = "✅ both match"
			matchCount++
		} else if r1Match {
			satStr = "✗ R2 wrong"
		} else if r2Match {
			satStr = "✗ R1 wrong"
		} else {
			satStr = "✗ both wrong"
		}

		poStr := " "
		if po {
			poStr = "✓"
			poCount++
		}
		if bothMatch && po { matchAndPO++ }

		fmt.Printf("  %-3d  %-*s  %-*s  %-*s  %-*s  %s\n",
			count, orderW, orderStr,
			r1W, r1Got,
			r2W, r2Got,
			satW, satStr,
			poStr)
	}

	// Summary
	fmt.Println(sepLine)
	fmt.Printf("\n  Total orderings         : %d\n", count)
	fmt.Printf("  Both reads satisfied    : %d\n", matchCount)
	fmt.Printf("  Program order respected : %d  (PO column = ✓)\n", poCount)
	fmt.Printf("  Both reads + PO         : %d\n\n", matchAndPO)

	// Identify the SC-satisfying orderings per variable
	fmt.Println("  SC analysis per variable:")
	fmt.Println()
	fmt.Println("  For x (writes: W1(x)a at t1, W2(x)b at t3 — read: R1(x)b at t5):")
	fmt.Println("    R1(x)=a requires the last write to x before t5 = a")
	fmt.Println("    → W1(x)a must appear AFTER W2(x)b in the ordering")
	fmt.Println("    → any ordering where W2(x)b precedes W1(x)a satisfies x's read ✅")
	fmt.Println()
	fmt.Println("  For y (writes: W2(y)b at t2, W1(y)a at t4 — read: R2(y)b at t5):")
	fmt.Println("    R2(y)=b requires the last write to y before t5 = b")
	fmt.Println("    → W2(y)b must appear AFTER W1(y)a in the ordering")
	fmt.Println("    → any ordering where W1(y)a precedes W2(y)b satisfies y's read ✅")
	fmt.Println()
	fmt.Println("  For BOTH reads to be satisfied simultaneously:")
	fmt.Println("    x needs: W2(x)b → ... → W1(x)a  (P2 before P1 on x)")
	fmt.Println("    y needs: W1(y)a → ... → W2(y)b  (P1 before P2 on y)")
	fmt.Println()
	fmt.Println("  These imply opposite process orders: P2<P1 on x but P1<P2 on y.")
	fmt.Println("  Serializability requires ONE global order — impossible here.")
	fmt.Println("  Orderings satisfying both reads exist (see ✅ rows above),")
	fmt.Println("  but none of them correspond to a pure serial execution of P1,P2.")
	fmt.Println()
}
