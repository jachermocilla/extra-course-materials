package main

import (
	"fmt"
	"sync"
)

type LamportClock struct {
	mu   sync.Mutex
	time int
	name string
}

func NewClock(name string) *LamportClock {
	return &LamportClock{name: name}
}

// Internal event: just increment
func (c *LamportClock) Event() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.time++
	fmt.Printf("[%s] internal event  → t=%d\n", c.name, c.time)
}

// Send: increment, return timestamp to attach to message
func (c *LamportClock) Send(to string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.time++
	fmt.Printf("[%s] send to %s       → t=%d\n", c.name, to, c.time)
	return c.time
}

// Receive: take max of local and received timestamp, then increment
func (c *LamportClock) Receive(from string, received int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if received > c.time {
		c.time = received
	}
	c.time++
	fmt.Printf("[%s] recv from %s     → t=%d\n", c.name, from, c.time)
}

func main() {
	p1 := NewClock("P1")
	p2 := NewClock("P2")
	p3 := NewClock("P3")

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		p1.Event()
		t := p1.Send("P2")
		p2.Receive("P1", t)
		p1.Event()
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
