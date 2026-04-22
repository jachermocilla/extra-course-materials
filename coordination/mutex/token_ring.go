package main

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"
)

// ─────────────────────────────────────────────
//  TOKEN
// ─────────────────────────────────────────────

type Token struct{}

// ─────────────────────────────────────────────
//  PROCESS
// ─────────────────────────────────────────────

type Process struct {
	id        int
	wantsCS   bool
	next      *Process   // next process in the ring
	tokenCh   chan Token  // receives token from previous process
	balance   *float64
	balanceMu *sync.Mutex
	log       []string
	done      chan struct{}
}

func NewProcess(id int, balance *float64, mu *sync.Mutex) *Process {
	return &Process{
		id:        id,
		tokenCh:   make(chan Token, 1),
		balance:   balance,
		balanceMu: mu,
		done:      make(chan struct{}),
	}
}

func (p *Process) logf(format string, args ...any) {
	entry := fmt.Sprintf(format, args...)
	p.log = append(p.log, entry)
	fmt.Println(entry)
}

// ── Critical section ──────────────────────────────────────────

func (p *Process) enterCS() {
	p.balanceMu.Lock()
	before := *p.balance
	*p.balance += 100
	after := *p.balance
	p.balanceMu.Unlock()
	p.logf("  [P%d] *** CRITICAL SECTION: %.2f → %.2f PHP ***",
		p.id, before, after)
	time.Sleep(30 * time.Millisecond)
}

// ── Token passing loop ────────────────────────────────────────

func (p *Process) Run(wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case <-p.done:
			return
		case <-p.tokenCh:
			p.logf("  [P%d] received token", p.id)

			if p.wantsCS {
				p.logf("  [P%d] wants CS — using token", p.id)
				p.enterCS()
				p.wantsCS = false
			} else {
				p.logf("  [P%d] does not want CS — passing token", p.id)
			}

			// Pass token to next after a short delay
			time.Sleep(time.Duration(rand.Intn(20)+5) * time.Millisecond)
			p.logf("  [P%d] passing token → P%d", p.id, p.next.id)
			p.next.tokenCh <- Token{}
		}
	}
}

// ─────────────────────────────────────────────
//  MAIN
// ─────────────────────────────────────────────

func main() {
	rand.Seed(time.Now().UnixNano())

	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("  TOKEN-RING MUTUAL EXCLUSION")
	fmt.Println("  4 processes in a logical ring")
	fmt.Println(strings.Repeat("=", 60))

	balance := 1000.0
	var balanceMu sync.Mutex

	// Create 4 processes
	const n = 4
	procs := make([]*Process, n)
	for i := 0; i < n; i++ {
		procs[i] = NewProcess(i+1, &balance, &balanceMu)
	}

	// Wire the ring: P1→P2→P3→P4→P1
	for i := 0; i < n; i++ {
		procs[i].next = procs[(i+1)%n]
	}

	fmt.Println("\n  Ring topology:")
	for i := 0; i < n; i++ {
		fmt.Printf("    P%d → P%d\n", procs[i].id, procs[i].next.id)
	}

	// Randomly assign which processes want the CS
	// Guarantee at least 3 want it so we see meaningful contention
	wantCS := []int{1, 2, 3} // P1, P2, P3 all want CS
	for _, id := range wantCS {
		procs[id-1].wantsCS = true
	}

	fmt.Println("\n  Processes wanting CS:", wantCS)
	fmt.Printf("  Initial balance : %.2f PHP\n", balance)
	fmt.Printf("  Expected final  : %.2f PHP\n\n",
		balance+float64(len(wantCS))*100)
	fmt.Println(strings.Repeat("-", 60))

	// Start all processes
	var wg sync.WaitGroup
	wg.Add(n)
	for _, p := range procs {
		go p.Run(&wg)
	}

	// Inject token at P1 to start the ring
	fmt.Println("  [RING] injecting token at P1")
	procs[0].tokenCh <- Token{}

	// Wait until all processes that want CS have used it
	// then shut down after one more full revolution
	for {
		time.Sleep(10 * time.Millisecond)
		allDone := true
		for _, id := range wantCS {
			if procs[id-1].wantsCS {
				allDone = false
				break
			}
		}
		if allDone {
			// Give time for token to complete current pass
			time.Sleep(200 * time.Millisecond)
			for _, p := range procs {
				close(p.done)
			}
			break
		}
	}

	wg.Wait()

	// ── Results ───────────────────────────────────────────────
	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("\n  Final balance : %.2f PHP\n", balance)
	expected := 1000.0 + float64(len(wantCS))*100
	if balance == expected {
		fmt.Println("  ✅ Correct — mutual exclusion held, no race condition")
	} else {
		fmt.Println("  ❌ Wrong — race condition detected")
	}

	fmt.Println()
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("  MESSAGE SUMMARY")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf(`
  N = %d processes
  Token passes per CS entry : 1 to N
    best case  : process receives token just as it wants CS  (1 pass)
    worst case : process just missed the token              (%d passes)

  Idle cost : token circulates continuously even with no CS requests
`, n, n)
}
