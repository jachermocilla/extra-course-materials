package main

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"
)

const numUsers = 3

// ─────────────────────────────────────────────
//  VECTOR CLOCK
// ─────────────────────────────────────────────

type VClock [numUsers]int

func (vc VClock) String() string {
	return fmt.Sprintf("[Alice=%d Bob=%d Carol=%d]",
		vc[0], vc[1], vc[2])
}

// ─────────────────────────────────────────────
//  CHAT MESSAGE
// ─────────────────────────────────────────────

type ChatMsg struct {
	id       int
	senderID int
	vc       VClock
	text     string
}

func (m ChatMsg) String() string {
	names := []string{"Alice", "Bob", "Carol"}
	return fmt.Sprintf("[%s] \"%s\"  vc=%s",
		names[m.senderID], m.text, m.vc)
}

// ─────────────────────────────────────────────
//  CHAT USER (process)
// ─────────────────────────────────────────────

type User struct {
	mu            sync.Mutex
	id            int
	name          string
	vc            VClock
	delivered     VClock   // messages delivered from each sender
	buffer        []ChatMsg
	peers         []*User
	inbox         chan ChatMsg
	chatLog       []string // what this user sees in their chat window
	deliveryOrder []int    // message IDs in delivery order
}

func NewUser(id int, name string) *User {
	return &User{
		id:    id,
		name:  name,
		inbox: make(chan ChatMsg, 64),
	}
}

// ── vector clock ──────────────────────────────────────────────

func (u *User) tick() {
	u.vc[u.id]++
}

func (u *User) merge(incoming VClock) {
	for i := range u.vc {
		if incoming[i] > u.vc[i] {
			u.vc[i] = incoming[i]
		}
	}
}

// ── causal delivery check ─────────────────────────────────────

func (u *User) canDeliver(msg ChatMsg) bool {
	j := msg.senderID

	// next expected message from this sender
	if msg.vc[j] != u.delivered[j]+1 {
		return false
	}

	// all causal dependencies already delivered
	for k := 0; k < numUsers; k++ {
		if k == j {
			continue
		}
		if msg.vc[k] > u.delivered[k] {
			return false
		}
	}
	return true
}

// ── deliver message to chat window ───────────────────────────

func (u *User) deliver(msg ChatMsg) {
	u.delivered[msg.senderID]++
	u.merge(msg.vc)
	u.deliveryOrder = append(u.deliveryOrder, msg.id)

	names := []string{"Alice", "Bob", "Carol"}
	entry := fmt.Sprintf("  %-6s | %s: %s",
		names[msg.senderID],
		names[msg.senderID],
		msg.text)
	u.chatLog = append(u.chatLog, entry)

	fmt.Printf("  [%-5s chat] DELIVER  %s\n", u.name, msg)
}

func (u *User) tryFlushBuffer() {
	progress := true
	for progress {
		progress = false
		for i := 0; i < len(u.buffer); i++ {
			if u.canDeliver(u.buffer[i]) {
				msg := u.buffer[i]
				u.buffer = append(u.buffer[:i], u.buffer[i+1:]...)
				u.deliver(msg)
				progress = true
				break
			}
		}
	}
}

// ── receive ───────────────────────────────────────────────────

func (u *User) Receive(msg ChatMsg) {
	u.mu.Lock()
	defer u.mu.Unlock()

	if u.canDeliver(msg) {
		u.deliver(msg)
		u.tryFlushBuffer()
	} else {
		names := []string{"Alice", "Bob", "Carol"}
		fmt.Printf("  [%-5s chat] BUFFER   %s  "+
			"(waiting: need %s msg#%d have %d)\n",
			u.name, msg,
			names[msg.senderID],
			msg.vc[msg.senderID],
			u.delivered[msg.senderID])
		u.buffer = append(u.buffer, msg)
	}
}

// ── send a chat message ───────────────────────────────────────

func (u *User) Send(msgID int, text string) ChatMsg {
	u.mu.Lock()
	u.tick()
	msg := ChatMsg{
		id:       msgID,
		senderID: u.id,
		vc:       u.vc,
		text:     text,
	}
	// sender delivers to own chat window immediately
	u.deliver(msg)
	peers := make([]*User, len(u.peers))
	copy(peers, u.peers)
	u.mu.Unlock()

	fmt.Printf("  [%-5s chat] SEND     %s\n", u.name, msg)

	for _, peer := range peers {
		peer := peer
		go func() {
			delay := time.Duration(rand.Intn(80)+20) * time.Millisecond
			time.Sleep(delay)
			peer.inbox <- msg
		}()
	}
	return msg
}

func (u *User) Run(wg *sync.WaitGroup) {
	defer wg.Done()
	for msg := range u.inbox {
		u.Receive(msg)
	}
}

// ─────────────────────────────────────────────
//  HELPERS
// ─────────────────────────────────────────────

func positionOf(order []int, msgID int) int {
	for i, id := range order {
		if id == msgID {
			return i
		}
	}
	return -1
}

func printChatWindow(u *User) {
	u.mu.Lock()
	defer u.mu.Unlock()
	fmt.Printf("\n  ┌─ %s's chat window ", u.name)
	fmt.Println(strings.Repeat("─", 35-len(u.name)) + "┐")
	for _, line := range u.chatLog {
		fmt.Printf("  │ %-50s│\n", line[2:])
	}
	fmt.Println("  └" + strings.Repeat("─", 52) + "┘")
}

// ─────────────────────────────────────────────
//  MAIN
// ─────────────────────────────────────────────

func main() {
	rand.Seed(time.Now().UnixNano())

	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("  CAUSALLY ORDERED MULTICAST — CHAT EXAMPLE")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println(`
  Scenario:

    Alice posts:  "Anyone want lunch?"          M1
    Bob replies:  "Sure, where?"                M2  (caused by M1)
    Carol posts:  "I finished the report"       M3  (concurrent)

  Causal constraint:
    M1 → M2  :  everyone must see Alice's post BEFORE Bob's reply
    M3 ∥ M1  :  Carol's post may appear in any order
`)
	fmt.Println(strings.Repeat("-", 60))
	fmt.Println("  Message flow:")
	fmt.Println()

	// create users
	alice := NewUser(0, "Alice")
	bob := NewUser(1, "Bob")
	carol := NewUser(2, "Carol")
	users := []*User{alice, bob, carol}

	// wire peers
	for _, u := range users {
		for _, other := range users {
			if other.id != u.id {
				u.peers = append(u.peers, other)
			}
		}
	}

	// start inbox handlers
	var wg sync.WaitGroup
	for _, u := range users {
		wg.Add(1)
		go u.Run(&wg)
	}

	// ── Chat conversation ─────────────────────────────────────

	// M1: Alice posts her question
	alice.Send(1, "Anyone want lunch?")

	// M3: Carol posts concurrently — unrelated to Alice
	go carol.Send(3, "I finished the report")

	// M2: Bob waits until he sees Alice's message, then replies
	// This encodes the causal dependency M1 → M2
	go func() {
		for {
			bob.mu.Lock()
			d := bob.delivered[alice.id]
			bob.mu.Unlock()
			if d >= 1 {
				break
			}
			time.Sleep(5 * time.Millisecond)
		}
		bob.Send(2, "Sure, where?")
	}()

	// wait for all messages to propagate
	time.Sleep(600 * time.Millisecond)

	for _, u := range users {
		close(u.inbox)
	}
	wg.Wait()

	// ── Print each user's chat window ─────────────────────────
	fmt.Println()
	fmt.Println(strings.Repeat("-", 60))
	fmt.Println("  CHAT WINDOWS")
	fmt.Println(strings.Repeat("-", 60))
	for _, u := range users {
		printChatWindow(u)
	}

	// ── Verify causal constraint ──────────────────────────────
	fmt.Println()
	fmt.Println(strings.Repeat("-", 60))
	fmt.Println("  VERIFICATION: Bob's reply must come AFTER Alice's post")
	fmt.Println(strings.Repeat("-", 60))
	fmt.Println()

	allCorrect := true
	for _, u := range users {
		u.mu.Lock()
		order := make([]int, len(u.deliveryOrder))
		copy(order, u.deliveryOrder)
		u.mu.Unlock()

		m1pos := positionOf(order, 1) // Alice's post
		m2pos := positionOf(order, 2) // Bob's reply

		switch {
		case m1pos == -1 || m2pos == -1:
			fmt.Printf("  %-5s: missing messages  ❌\n", u.name)
			allCorrect = false
		case m1pos < m2pos:
			fmt.Printf("  %-5s: Alice (pos %d) → Bob (pos %d)  ✅\n",
				u.name, m1pos+1, m2pos+1)
		default:
			fmt.Printf("  %-5s: Bob (pos %d) before Alice (pos %d)  ❌ VIOLATION\n",
				u.name, m2pos+1, m1pos+1)
			allCorrect = false
		}
	}

	fmt.Println()
	if allCorrect {
		fmt.Println("  ✅ Causal order preserved — no user saw the reply")
		fmt.Println("     before the original post")
	} else {
		fmt.Println("  ❌ Causal order violated")
	}

	fmt.Println()
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("  WHY BUFFERING IS NECESSARY")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println(`
  Without causal ordering Carol could see:

    Bob:   "Sure, where?"       ← reply arrives first (fast network)
    Alice: "Anyone want lunch?" ← original post arrives late

  Bob's reply makes no sense without Alice's post.
  The buffer holds Bob's reply until Alice's post is delivered,
  then releases it — guaranteeing a coherent conversation.`)
}
