package main

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"time"
)

// ─────────────────────────────────────────────
//  MESSAGE TYPES
// ─────────────────────────────────────────────

type MsgType int

const (
	REQUEST  MsgType = iota
	RESPONSE         // replica → client: current owner + timestamp
	YIELD            // client → replica: step back, requeue me
	RELEASE          // client → replica: exiting CS
)

func (m MsgType) String() string {
	return [...]string{"REQUEST ", "RESPONSE", "YIELD   ", "RELEASE "}[m]
}

type Message struct {
	kind           MsgType
	senderID       int
	timestamp      int
	ownerID        int // RESPONSE: who currently owns this replica
	ownerTimestamp int // RESPONSE: owner's timestamp
}

func (m Message) String() string {
	return fmt.Sprintf("{%s from=%d t=%d}", m.kind, m.senderID, m.timestamp)
}

// ─────────────────────────────────────────────
//  LAMPORT CLOCK
// ─────────────────────────────────────────────

type LamportClock struct {
	mu   sync.Mutex
	time int
}

func (c *LamportClock) Tick() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.time++
	return c.time
}

func (c *LamportClock) Update(received int) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	if received > c.time {
		c.time = received
	}
	c.time++
	return c.time
}

// ─────────────────────────────────────────────
//  REPLICA
// ─────────────────────────────────────────────

type QueueEntry struct {
	clientID  int
	timestamp int
}

type Replica struct {
	mu           sync.Mutex
	id           int
	ownerID      int // -1 = free
	ownerTS      int
	queue        []QueueEntry
	clientChans  map[int]chan Message
}

func NewReplica(id int) *Replica {
	return &Replica{
		id:          id,
		ownerID:     -1,
		clientChans: make(map[int]chan Message),
	}
}

func (r *Replica) Register(clientID int, ch chan Message) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.clientChans[clientID] = ch
}

func (r *Replica) sendResponse(clientID, ownerID, ownerTS int) {
	ch := r.clientChans[clientID]
	go func() {
		time.Sleep(time.Duration(rand.Intn(30)+5) * time.Millisecond)
		ch <- Message{
			kind:           RESPONSE,
			senderID:       r.id,
			ownerID:        ownerID,
			ownerTimestamp: ownerTS,
		}
	}()
}

func (r *Replica) respQueue() {
	if len(r.queue) == 0 {
		return
	}
	// sort by timestamp (Lamport order) before picking front
	sort.Slice(r.queue, func(i, j int) bool {
		if r.queue[i].timestamp != r.queue[j].timestamp {
			return r.queue[i].timestamp < r.queue[j].timestamp
		}
		return r.queue[i].clientID < r.queue[j].clientID
	})
	next := r.queue[0]
	r.queue = r.queue[1:]
	r.ownerID = next.clientID
	r.ownerTS = next.timestamp
	fmt.Printf("    [Replica %d] promoting C%d from queue\n", r.id, next.clientID)
	r.sendResponse(next.clientID, next.clientID, next.timestamp)
}

// Handle processes a message from a client
func (r *Replica) Handle(msg Message) {
	r.mu.Lock()
	defer r.mu.Unlock()

	switch msg.kind {

	case REQUEST:
		if r.ownerID == -1 {
			// Free — grant immediately
			r.ownerID = msg.senderID
			r.ownerTS = msg.timestamp
			fmt.Printf("    [Replica %d] grant → C%d (free)\n", r.id, msg.senderID)
			r.sendResponse(msg.senderID, msg.senderID, msg.timestamp)
		} else {
			// Owned — queue and inform client of current owner
			r.queue = append(r.queue, QueueEntry{msg.senderID, msg.timestamp})
			fmt.Printf("    [Replica %d] queue C%d (owner=C%d), queue=%v\n",
				r.id, msg.senderID, r.ownerID, r.queueIDs())
			r.sendResponse(msg.senderID, r.ownerID, r.ownerTS)
		}

	case YIELD:
		// Semantic: RELEASE + REQUEST — requeue the client in timestamp order
		if msg.senderID == r.ownerID {
			fmt.Printf("    [Replica %d] YIELD from C%d — requeue\n", r.id, msg.senderID)
			r.queue = append(r.queue, QueueEntry{msg.senderID, msg.timestamp})
			r.ownerID = -1
			r.respQueue()
		}

	case RELEASE:
		if msg.senderID == r.ownerID {
			fmt.Printf("    [Replica %d] RELEASE from C%d\n", r.id, msg.senderID)
			r.ownerID = -1
			r.ownerTS = 0
			r.respQueue()
		}
	}
}

func (r *Replica) queueIDs() []int {
	ids := make([]int, len(r.queue))
	for i, e := range r.queue {
		ids[i] = e.clientID
	}
	return ids
}

// ─────────────────────────────────────────────
//  CLIENT
// ─────────────────────────────────────────────

type RespRecord struct {
	ownerID  int
	ownerTS  int
	received bool
}

type Client struct {
	mu        sync.Mutex
	id        int
	clock     LamportClock
	replicas  []*Replica
	inbox     chan Message
	balance   *float64
	balanceMu *sync.Mutex
	m         int // quorum size needed
}

func NewClient(id int, replicas []*Replica, balance *float64, mu *sync.Mutex) *Client {
	n := len(replicas)
	c := &Client{
		id:        id,
		replicas:  replicas,
		inbox:     make(chan Message, 64),
		balance:   balance,
		balanceMu: mu,
		m:         n/2 + 1, // majority
	}
	for _, r := range replicas {
		r.Register(id, c.inbox)
	}
	return c
}

func (c *Client) sendToReplica(r *Replica, kind MsgType, ts int) {
	msg := Message{kind: kind, senderID: c.id, timestamp: ts}
	go func() {
		time.Sleep(time.Duration(rand.Intn(30)+5) * time.Millisecond)
		r.Handle(msg)
	}()
}

func (c *Client) EnterCS(wg *sync.WaitGroup) {
	defer wg.Done()

	time.Sleep(time.Duration(rand.Intn(60)) * time.Millisecond)

	n := len(c.replicas)

	for {
		// ── Step 1: broadcast REQUEST ─────────────────────────
		ts := c.clock.Tick()
		fmt.Printf("  [C%d t=%d] broadcast REQUEST to %d replicas\n",
			c.id, ts, n)
		resp := make([]RespRecord, n)
		for i, r := range c.replicas {
			c.sendToReplica(r, REQUEST, ts)
			_ = i
		}

		// ── Step 2: collect RESPONSES ─────────────────────────
		received := 0
		for received < n {
			msg := <-c.inbox
			c.clock.Update(msg.ownerTimestamp)
			// find which replica index sent this
			for i, r := range c.replicas {
				if r.id == msg.senderID {
					resp[i] = RespRecord{msg.ownerID, msg.ownerTimestamp, true}
					break
				}
			}
			received++
		}

		// ── Step 3: compute winner ────────────────────────────
		// Count how many responses show us as owner
		selfCount := 0
		for _, r := range resp {
			if r.ownerID == c.id {
				selfCount++
			}
		}

		// Find the most common non-self owner (case 2)
		ownerCounts := map[int]int{}
		for _, r := range resp {
			if r.ownerID != c.id {
				ownerCounts[r.ownerID]++
			}
		}
		someoneElseWon := false
		for _, count := range ownerCounts {
			if count >= c.m {
				someoneElseWon = true
				break
			}
		}

		if selfCount >= c.m {
			// Case 1: WE WIN
			fmt.Printf("  [C%d] won quorum (%d/%d) — entering CS\n",
				c.id, selfCount, n)
			break
		} else if someoneElseWon {
			// Case 2: someone else won — wait passively
			// We are already queued at replicas; wait for RESPONSE
			fmt.Printf("  [C%d] someone else won — waiting passively\n", c.id)
			for {
				msg := <-c.inbox
				c.clock.Update(msg.ownerTimestamp)
				// check if we now own majority
				for i, r := range c.replicas {
					if r.id == msg.senderID {
						resp[i] = RespRecord{msg.ownerID, msg.ownerTimestamp, true}
						break
					}
				}
				selfNow := 0
				for _, r := range resp {
					if r.ownerID == c.id {
						selfNow++
					}
				}
				if selfNow >= c.m {
					fmt.Printf("  [C%d] notified — now have quorum (%d/%d)\n",
						c.id, selfNow, n)
					goto enterCS
				}
			}
		} else {
			// Case 3: nobody won — YIELD replicas we own
			yieldCount := 0
			for i, r := range resp {
				if r.ownerID == c.id {
					yieldCount++
					c.sendToReplica(c.replicas[i], YIELD, ts)
				}
			}
			fmt.Printf("  [C%d] nobody won — YIELD %d replicas, retry\n",
				c.id, yieldCount)
			// wait a short backoff then retry
			time.Sleep(time.Duration(rand.Intn(20)+10) * time.Millisecond)
			continue
		}

	enterCS:
		break
	}

	// ── Critical section ──────────────────────────────────────
	c.balanceMu.Lock()
	before := *c.balance
	*c.balance += 100
	after := *c.balance
	c.balanceMu.Unlock()
	fmt.Printf("  [C%d] *** CRITICAL SECTION: %.2f → %.2f PHP ***\n",
		c.id, before, after)
	time.Sleep(30 * time.Millisecond)

	// ── RELEASE all replicas ──────────────────────────────────
	ts := c.clock.Tick()
	fmt.Printf("  [C%d t=%d] broadcast RELEASE\n", c.id, ts)
	for _, r := range c.replicas {
		c.sendToReplica(r, RELEASE, ts)
	}
}

// ─────────────────────────────────────────────
//  MAIN
// ─────────────────────────────────────────────

func main() {
	rand.Seed(time.Now().UnixNano())

	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("  SIGMA PROTOCOL — Lin et al. (2004)")
	fmt.Println("  Distributed Mutual Exclusion for P2P DHT")
	fmt.Println(strings.Repeat("=", 60))

	balance := 1000.0
	var balanceMu sync.Mutex

	// 5 replicas, majority quorum = 3
	const numReplicas = 5
	replicas := make([]*Replica, numReplicas)
	for i := 0; i < numReplicas; i++ {
		replicas[i] = NewReplica(i + 1)
	}

	// 3 clients contend concurrently
	const numClients = 3
	clients := make([]*Client, numClients)
	for i := 0; i < numClients; i++ {
		clients[i] = NewClient(i+1, replicas, &balance, &balanceMu)
	}

	fmt.Printf("\n  Replicas        : %d  (quorum = majority = %d)\n",
		numReplicas, numReplicas/2+1)
	fmt.Printf("  Clients         : %d\n", numClients)
	fmt.Printf("  Initial balance : %.2f PHP\n", balance)
	fmt.Printf("  Expected final  : %.2f PHP\n\n",
		balance+float64(numClients)*100)
	fmt.Println(strings.Repeat("-", 60))

	var wg sync.WaitGroup
	wg.Add(numClients)
	for _, c := range clients {
		go c.EnterCS(&wg)
	}
	wg.Wait()

	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("\n  Final balance : %.2f PHP\n", balance)
	expected := 1000.0 + float64(numClients)*100
	if balance == expected {
		fmt.Println("  ✅ Correct — mutual exclusion held")
	} else {
		fmt.Println("  ❌ Wrong — race condition detected")
	}

}

