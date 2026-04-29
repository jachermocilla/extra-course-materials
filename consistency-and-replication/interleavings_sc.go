package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type State struct{ x, y, z int }

type Stmt struct {
	pid   int
	label string
	exec  func(s *State) string
}

var stmts = []Stmt{
	{1, "x←1",        func(s *State) string { s.x = 1; return "" }},
	{2, "y←1",        func(s *State) string { s.y = 1; return "" }},
	{3, "z←1",        func(s *State) string { s.z = 1; return "" }},
	{1, "print(y,z)", func(s *State) string { return fmt.Sprintf("(%d,%d)", s.y, s.z) }},
	{2, "print(x,z)", func(s *State) string { return fmt.Sprintf("(%d,%d)", s.x, s.z) }},
	{3, "print(x,y)", func(s *State) string { return fmt.Sprintf("(%d,%d)", s.x, s.y) }},
}

// valid: program order enforced — each process's write precedes its own print
func valid(perm []int) bool {
	pos := make([]int, 6)
	for i, v := range perm { pos[v] = i }
	return pos[0] < pos[3] && pos[1] < pos[4] && pos[2] < pos[5]
}

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

func run(perm []int) (seq string, perOut [3]string, sig string) {
	s := &State{}
	var labels []string
	for _, idx := range perm {
		st := stmts[idx]
		labels = append(labels, st.label)
		if o := st.exec(s); o != "" {
			perOut[st.pid-1] = o
		}
	}
	seq = strings.Join(labels, ", ")
	clean := strings.NewReplacer("(", "", ")", "", ",", "").Replace
	sig = clean(perOut[0]) + clean(perOut[1]) + clean(perOut[2])
	return
}

// sigToInt parses the 6-bit string (e.g. "011011") as a binary integer
func sigToInt(sig string) int {
	v, _ := strconv.ParseInt(sig, 2, 64)
	return int(v)
}

func printTable(title string, perms [][]int, globalNums []int, validSigs map[string]bool) (int, map[string]int) {
	sep := strings.Repeat("═", 120)
	fmt.Printf("\n%s\n", sep)
	fmt.Println(title)
	fmt.Printf("%s\n\n", sep)
	fmt.Printf("  %-6s  %-52s  %-8s  %-8s  %-8s  %-10s  %-6s  %s\n",
		"#", "Interleaving", "P1", "P2", "P3", "Signature", "Dec", "SC?")
	fmt.Printf("  %-6s  %-52s  %-8s  %-8s  %-8s  %-10s  %-6s  %s\n",
		"──────", strings.Repeat("─", 52), "────────", "────────", "────────", "──────────", "──────", "───")

	validCount := 0
	sigCounts := make(map[string]int)
	for i, perm := range perms {
		if !valid(perm) { continue }
		validCount++
		seq, perOut, sig := run(perm)
		sigCounts[sig]++
		dec := sigToInt(sig)
		sc := "yes"
		if !validSigs[sig] { sc = "NO" }
		fmt.Printf("  %-6d  %-52s  %-8s  %-8s  %-8s  %-10s  %-6d  %s\n",
			globalNums[i], seq, perOut[0], perOut[1], perOut[2], sig, dec, sc)
	}

	fmt.Println()
	fmt.Printf("  %s\n", strings.Repeat("─", 120))
	fmt.Printf("  Valid interleavings : %d\n\n", validCount)

	// collect and sort unique sigs by decimal value
	type sigRow struct{ sig string; dec, count int }
	var rows []sigRow
	for s, c := range sigCounts {
		rows = append(rows, sigRow{s, sigToInt(s), c})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].dec < rows[j].dec })

	fmt.Printf("  Distinct signatures: %d\n", len(rows))
	fmt.Printf("  %-10s  %-6s  %-6s  %s\n\n", "Signature", "Dec", "Count", "SC?")
	for _, r := range rows {
		sc := "yes"
		if !validSigs[r.sig] { sc = "NO" }
		fmt.Printf("    %-8s  %-6d  %-6d  %s\n", r.sig, r.dec, r.count, sc)
	}
	return validCount, sigCounts
}

func filter(all [][]int, firstIdx int) ([][]int, []int) {
	var perms [][]int
	var nums []int
	for i, p := range all {
		if firstIdx < 0 || p[0] == firstIdx {
			perms = append(perms, p)
			nums = append(nums, i+1)
		}
	}
	return perms, nums
}

// collectValidSigs returns the set of signatures produced by valid interleavings
func collectValidSigs(all [][]int) map[string]bool {
	sigs := make(map[string]bool)
	for _, p := range all {
		if valid(p) {
			_, _, sig := run(p)
			sigs[sig] = true
		}
	}
	return sigs
}

func main() {
	fmt.Println("╔══════════════════════════════════════════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║  P1: x←1 ; print(y,z)    P2: y←1 ; print(x,z)    P3: z←1 ; print(x,y)   x=y=z=0 initially                    ║")
	fmt.Println("║  Signature = P1·output ++ P2·output ++ P3·output treated as 6-bit binary value                                 ║")
	fmt.Println("║  SC? = yes if this signature can arise from a sequentially consistent execution                                 ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════════════════════════════════════════════════════╝")

	all := permutations(6)

	// The set of valid signatures IS exactly the set produced by the 90 valid
	// interleavings — any signature reachable under program order satisfies SC.
	validSigs := collectValidSigs(all)

	allPerms, allNums := filter(all, -1)
	printTable("ALL VALID INTERLEAVINGS  (program order enforced — 90 of 720)", allPerms, allNums, validSigs)

	type sect struct{ title string; firstIdx int }
	sections := []sect{
		{"STARTING WITH x←1  (valid only — 30 of 120)", 0},
		{"STARTING WITH y←1  (valid only — 30 of 120)", 1},
		{"STARTING WITH z←1  (valid only — 30 of 120)", 2},
	}

	for _, s := range sections {
		perms, nums := filter(all, s.firstIdx)
		printTable(s.title, perms, nums, validSigs)
	}

	// ── 6-bit signature census: all 64 possible values ──────────────────────────
	fmt.Printf("\n%s\n", strings.Repeat("═", 120))
	fmt.Println("FULL 6-BIT SIGNATURE SPACE  (all 64 possible values, 0..63)")
	fmt.Printf("%s\n\n", strings.Repeat("═", 120))
	fmt.Printf("  %-6s  %-10s  %-8s  %-8s  %-8s  %s\n",
		"Dec", "Binary", "P1(y,z)", "P2(x,z)", "P3(x,y)", "SC?")
	fmt.Printf("  %-6s  %-10s  %-8s  %-8s  %-8s  %s\n",
		"──────", "──────────", "────────", "────────", "────────", "───")
	for v := 0; v < 64; v++ {
		bits := fmt.Sprintf("%06b", v)
		p1 := fmt.Sprintf("(%c,%c)", bits[0], bits[1])
		p2 := fmt.Sprintf("(%c,%c)", bits[2], bits[3])
		p3 := fmt.Sprintf("(%c,%c)", bits[4], bits[5])
		sc := "NO"
		if validSigs[bits] { sc = "yes" }
		fmt.Printf("  %-6d  %-10s  %-8s  %-8s  %-8s  %s\n", v, bits, p1, p2, p3, sc)
	}

	// summary of valid vs invalid signatures
	validSigCount := len(validSigs)
	fmt.Printf("\n  SC-valid signatures : %d / 64\n", validSigCount)
	fmt.Printf("  Non-SC signatures   : %d / 64\n\n", 64-validSigCount)
}
