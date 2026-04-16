package main

import (
	"fmt"
	"sync"
)

type VectorClock struct {
	mu    sync.Mutex
	times []int
	id    int
	name  string
}

func NewVectorClock(name string, id, numProcesses int) *VectorClock {
	return &VectorClock{
		name:  name,
		id:    id,
		times: make([]int, numProcesses),
	}
}

// Internal event: increment own entry only
func (v *VectorClock) Event() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.times[v.id]++
	fmt.Printf("[%s] internal event  → %v\n", v.name, v.times)
}

// Send: increment own entry, return a copy of the vector
func (v *VectorClock) Send(to string) []int {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.times[v.id]++
	ts := make([]int, len(v.times))
	copy(ts, v.times)
	fmt.Printf("[%s] send to %s       → %v\n", v.name, to, v.times)
	return ts
}

// Receive: element-wise max, then increment own entry
func (v *VectorClock) Receive(from string, received []int) {
	v.mu.Lock()
	defer v.mu.Unlock()
	for i := range v.times {
		if received[i] > v.times[i] {
			v.times[i] = received[i]
		}
	}
	v.times[v.id]++
	fmt.Printf("[%s] recv from %s     → %v\n", v.name, from, v.times)
}

func main() {
	// 3 processes, indices 0, 1, 2
	p1 := NewVectorClock("P1", 0, 3)
	p2 := NewVectorClock("P2", 1, 3)
	p3 := NewVectorClock("P3", 2, 3)

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		p1.Event()
		t := p1.Send("P2")
		p2.Receive("P1", t)
	}()

	go func() {
		defer wg.Done()
		p2.Event()
		t := p2.Send("P3")
		p3.Receive("P2", t)
	}()

	go func() {
		defer wg.Done()
		p3.Event()
		t := p3.Send("P1")
		p1.Receive("P3", t)
	}()

	wg.Wait()
}
