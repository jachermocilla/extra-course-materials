package main

import (
	"fmt"
	"strings"
)

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
	name      string
	process   string
	label     string
	timestamp int
}

func main() {
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("  LAMPORT CLOCK LIMITATION")
	fmt.Println("  C(a) < C(b) does NOT mean a happened-before b")
	fmt.Println(strings.Repeat("=", 60))

	p1 := &LamportClock{}
	p2 := &LamportClock{}
	p3 := &LamportClock{}

	// ── Events ────────────────────────────────────────────────
	//
	//  P1 ──[a: send]───────────────────────────────> P3
	//
	//  P2 ──[b: local]──  (isolated, no messages)
	//
	//  P3 ──────────────────────────[c: recv from P1]──

	tA := p1.tick()
	a := Event{"send msg to P3", "P1", "a", tA}

	tB := p2.tick()
	b := Event{"local write X", "P2", "b", tB}

	tC := p3.receive(tA)
	c := Event{"recv msg from P1", "P3", "c", tC}

	// ── Print events ──────────────────────────────────────────
	fmt.Println("\n  Events:")
	for _, e := range []Event{a, b, c} {
		fmt.Printf("    [%s] %-20s  C(%s) = %d\n",
			e.process, e.name, e.label, e.timestamp)
	}

	// ── The ambiguity ─────────────────────────────────────────
	fmt.Println("\n  Causal analysis:")

	fmt.Printf("\n  C(a)=%d < C(c)=%d  and  a sent the message c received\n",
		a.timestamp, c.timestamp)
	fmt.Println("  ✅ a happened-before c  —  causal, confirmed by message passing")

	fmt.Printf("\n  C(b)=%d < C(c)=%d\n", b.timestamp, c.timestamp)
	fmt.Println("  ❓ Does C(b) < C(c) mean b happened-before c?")
	fmt.Println()
	fmt.Println("  By timestamps alone : YES, it appears b happened-before c.")
	fmt.Println("  By actual execution : NO  — P2 never communicated")
	fmt.Println("                        with P1 or P3.")
	fmt.Println("                        b and c are CONCURRENT.")
	fmt.Println("                        C(b) < C(c) is MISLEADING.")

	fmt.Println()
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("  CONCLUSION")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println(`
  Lamport guarantees:

    a → c  implies  C(a) < C(c)          ✅

  But NOT the converse:

    C(b) < C(c)  does NOT imply  b → c   ✗

  Both C(a) < C(c) and C(b) < C(c) are numerically true,
  but only a → c is a real causal relationship.
  b and c are concurrent — Lamport clocks cannot tell
  the difference.`)
}
