package main

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"
)

const initialBalance = 1000.0

type Replica struct {
	mu      sync.Mutex
	id      int
	balance float64
	log     []string
}

func (r *Replica) apply(op string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	before := r.balance
	switch op {
	case "ADD_100":
		r.balance += 100
	case "ADD_1PCT":
		r.balance *= 1.01
	}
	entry := fmt.Sprintf("  [Replica %d] %-8s  %.2f → %.2f PHP", r.id, op, before, r.balance)
	r.log = append(r.log, entry)
}

// deliverWithRandomDelay simulates a message arriving after a random network delay
func deliverWithRandomDelay(r *Replica, op string, wg *sync.WaitGroup) {
	defer wg.Done()
	delay := time.Duration(rand.Intn(80)+10) * time.Millisecond
	time.Sleep(delay)
	r.apply(op)
}

func main() {
	rand.Seed(time.Now().UnixNano())

	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("  WITHOUT LAMPORT CLOCK")
	fmt.Println("  Two goroutines deliver ops in random arrival order")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("\n  Initial balance: %.2f PHP\n", initialBalance)
	fmt.Println("  Operations: ADD_100 (P1), ADD_1PCT (P2)")
	fmt.Println()

	r1 := &Replica{id: 1, balance: initialBalance}
	r2 := &Replica{id: 2, balance: initialBalance}

	var wg sync.WaitGroup

	// Each replica gets both ops delivered by separate goroutines
	// with independent random delays — order is not guaranteed

	// --- Replica 1 ---
	wg.Add(2)
	go deliverWithRandomDelay(r1, "ADD_100", &wg)
	go deliverWithRandomDelay(r1, "ADD_1PCT", &wg)

	// --- Replica 2 ---
	wg.Add(2)
	go deliverWithRandomDelay(r2, "ADD_100", &wg)
	go deliverWithRandomDelay(r2, "ADD_1PCT", &wg)

	wg.Wait()

	fmt.Println("  Replica 1 delivery order:")
	for _, l := range r1.log {
		fmt.Println(l)
	}
	fmt.Println("  Replica 2 delivery order:")
	for _, l := range r2.log {
		fmt.Println(l)
	}

	fmt.Printf("\n  Replica 1 final: %.2f PHP\n", r1.balance)
	fmt.Printf("  Replica 2 final: %.2f PHP\n", r2.balance)

	if fmt.Sprintf("%.2f", r1.balance) == fmt.Sprintf("%.2f", r2.balance) {
		fmt.Println("\n  (consistent this run — but not guaranteed!)")
		fmt.Println("  Re-run several times to observe divergence.")
	} else {
		fmt.Printf("\n  ❌ INCONSISTENT — diverged by %.2f PHP\n",
			r1.balance-r2.balance)
	}

	fmt.Println()
	fmt.Println("  Expected results depending on order:")
	fmt.Println("    ADD_100 first → ADD_1PCT : 1000 + 100 = 1100 × 1.01 = 1111.00 PHP")
	fmt.Println("    ADD_1PCT first → ADD_100 : 1000 × 1.01 = 1010 + 100 = 1110.00 PHP")
	fmt.Println("    Difference caused by ordering: 1.00 PHP")
}
