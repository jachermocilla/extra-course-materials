package main

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"
)

// ─────────────────────────────────────────────
//  ROLES AND MESSAGE TYPES
// ─────────────────────────────────────────────

type Role int

const (
	FOLLOWER  Role = iota
	CANDIDATE
	LEADER
)

func (r Role) String() string {
	return [...]string{"FOLLOWER ", "CANDIDATE", "LEADER   "}[r]
}

type MsgType int

const (
	VOTE_REQUEST  MsgType = iota
	VOTE_RESPONSE
	HEARTBEAT
)

func (m MsgType) String() string {
	return [...]string{
		"VOTE_REQUEST ",
		"VOTE_RESPONSE",
		"HEARTBEAT    ",
	}[m]
}

type Message struct {
	kind        MsgType
	term        int
	senderID    int
	voteGranted bool
}

// ─────────────────────────────────────────────
//  SERVER
// ─────────────────────────────────────────────

type Server struct {
	mu sync.Mutex

	id          int
	role        Role
	currentTerm int
	votedFor    int
	leaderID    int
	votesReceived int

	peers         map[int]*Server
	inbox         chan Message
	alive         bool
	lastHeartbeat time.Time
	electionTimeout time.Duration
}

func NewServer(id int) *Server {
	return &Server{
		id:              id,
		role:            FOLLOWER,
		currentTerm:     0,
		votedFor:        -1,
		leaderID:        -1,
		inbox:           make(chan Message, 64),
		alive:           true,
		lastHeartbeat:   time.Now(),
		electionTimeout: randomTimeout(),
		peers:           make(map[int]*Server),
	}
}

func randomTimeout() time.Duration {
	return time.Duration(150+rand.Intn(150)) * time.Millisecond
}

// ── send helpers — never called while holding s.mu ───────────

func (s *Server) sendTo(peerID int, msg Message) {
	s.mu.Lock()
	peer, ok := s.peers[peerID]
	s.mu.Unlock()
	if !ok {
		return
	}
	peer.mu.Lock()
	alive := peer.alive
	peer.mu.Unlock()
	if !alive {
		return
	}
	go func() {
		time.Sleep(time.Duration(rand.Intn(15)+2) * time.Millisecond)
		peer.inbox <- msg
	}()
}

func (s *Server) broadcastMsg(msg Message) {
	s.mu.Lock()
	peerIDs := make([]int, 0, len(s.peers))
	for id := range s.peers {
		peerIDs = append(peerIDs, id)
	}
	s.mu.Unlock()
	// send outside the lock
	for _, id := range peerIDs {
		s.sendTo(id, msg)
	}
}

// ── election timer ────────────────────────────────────────────

func (s *Server) runElectionTimer() {
	for {
		time.Sleep(10 * time.Millisecond)

		s.mu.Lock()
		alive := s.alive
		role := s.role
		timeout := s.electionTimeout
		last := s.lastHeartbeat
		s.mu.Unlock()

		if !alive {
			return
		}
		if role == LEADER {
			continue
		}
		if time.Since(last) > timeout {
			s.startElection() // called without holding lock
		}
	}
}

// ── heartbeat sender ──────────────────────────────────────────

func (s *Server) runHeartbeatSender() {
	for {
		time.Sleep(50 * time.Millisecond)

		s.mu.Lock()
		alive := s.alive
		role := s.role
		term := s.currentTerm
		id := s.id
		s.mu.Unlock()

		if !alive {
			return
		}
		if role != LEADER {
			continue
		}
		// send outside the lock
		s.broadcastMsg(Message{
			kind:     HEARTBEAT,
			term:     term,
			senderID: id,
		})
	}
}

// ── message handler ───────────────────────────────────────────

func (s *Server) HandleMessages() {
	for msg := range s.inbox {
		s.mu.Lock()
		alive := s.alive
		s.mu.Unlock()
		if !alive {
			continue
		}

		// all handlers acquire the lock themselves
		switch msg.kind {
		case VOTE_REQUEST:
			s.handleVoteRequest(msg)
		case VOTE_RESPONSE:
			s.handleVoteResponse(msg)
		case HEARTBEAT:
			s.handleHeartbeat(msg)
		}
	}
}

// ── step down helper — called with lock HELD ─────────────────

func (s *Server) stepDownIfNeeded(term int) {
	if term > s.currentTerm {
		fmt.Printf("  [S%d] term %d > %d — step down to FOLLOWER\n",
			s.id, term, s.currentTerm)
		s.currentTerm = term
		s.role = FOLLOWER
		s.votedFor = -1
		s.leaderID = -1
	}
}

// ── handle VOTE_REQUEST ───────────────────────────────────────

func (s *Server) handleVoteRequest(msg Message) {
	s.mu.Lock()
	s.stepDownIfNeeded(msg.term)

	grant := false
	if msg.term < s.currentTerm {
		fmt.Printf("  [S%d] deny S%d (stale term %d < %d)\n",
			s.id, msg.senderID, msg.term, s.currentTerm)
	} else if s.votedFor == -1 || s.votedFor == msg.senderID {
		grant = true
		s.votedFor = msg.senderID
		s.lastHeartbeat = time.Now()
		fmt.Printf("  [S%d] grant vote → S%d (term %d)\n",
			s.id, msg.senderID, msg.term)
	} else {
		fmt.Printf("  [S%d] deny S%d (already voted for S%d)\n",
			s.id, msg.senderID, s.votedFor)
	}

	resp := Message{
		kind:        VOTE_RESPONSE,
		term:        s.currentTerm,
		senderID:    s.id,
		voteGranted: grant,
	}
	senderID := msg.senderID
	s.mu.Unlock() // ← unlock BEFORE sending

	s.sendTo(senderID, resp)
}

// ── handle VOTE_RESPONSE ──────────────────────────────────────

func (s *Server) handleVoteResponse(msg Message) {
	s.mu.Lock()
	s.stepDownIfNeeded(msg.term)

	if s.role != CANDIDATE || msg.term != s.currentTerm {
		s.mu.Unlock()
		return
	}

	becomeLeader := false
	if msg.voteGranted {
		s.votesReceived++
		majority := (len(s.peers)+1)/2 + 1
		fmt.Printf("  [S%d] vote from S%d (%d/%d needed=%d)\n",
			s.id, msg.senderID,
			s.votesReceived, len(s.peers)+1, majority)
		if s.votesReceived >= majority {
			s.role = LEADER
			s.leaderID = s.id
			becomeLeader = true
			fmt.Printf("  [S%d] *** ELECTED LEADER for term %d ***\n",
				s.id, s.currentTerm)
		}
	}

	// capture what we need before releasing
	term := s.currentTerm
	id := s.id
	s.mu.Unlock() // ← unlock BEFORE broadcasting

	if becomeLeader {
		s.broadcastMsg(Message{
			kind:     HEARTBEAT,
			term:     term,
			senderID: id,
		})
	}
}

// ── handle HEARTBEAT ──────────────────────────────────────────

func (s *Server) handleHeartbeat(msg Message) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.stepDownIfNeeded(msg.term)

	if msg.term >= s.currentTerm {
		s.role = FOLLOWER
		s.leaderID = msg.senderID
		s.lastHeartbeat = time.Now()
		s.electionTimeout = randomTimeout()
	}
}

// ── start election — called WITHOUT holding lock ──────────────

func (s *Server) startElection() {
	s.mu.Lock()
	s.currentTerm++
	s.role = CANDIDATE
	s.votedFor = s.id
	s.votesReceived = 1
	s.leaderID = -1
	s.lastHeartbeat = time.Now()
	s.electionTimeout = randomTimeout()
	term := s.currentTerm
	id := s.id
	s.mu.Unlock() // ← unlock BEFORE broadcasting

	fmt.Printf("  [S%d] starting election for term %d\n", id, term)

	s.broadcastMsg(Message{
		kind:     VOTE_REQUEST,
		term:     term,
		senderID: id,
	})
}

// ── crash / recover ───────────────────────────────────────────

func (s *Server) Crash() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.alive = false
	s.role = FOLLOWER
	fmt.Printf("  [S%d] *** CRASHED ***\n", s.id)
}

func (s *Server) Recover() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.alive = true
	s.role = FOLLOWER
	s.votedFor = -1
	s.leaderID = -1
	s.lastHeartbeat = time.Now()
	s.electionTimeout = randomTimeout()
	fmt.Printf("  [S%d] *** RECOVERED (term=%d) ***\n", s.id, s.currentTerm)
}

// ─────────────────────────────────────────────
//  HELPERS
// ─────────────────────────────────────────────

func printStatus(servers []*Server) {
	fmt.Println()
	fmt.Printf("  %-6s %-10s %-6s %-8s %s\n",
		"Server", "Role", "Term", "Leader", "VotedFor")
	fmt.Println("  " + strings.Repeat("-", 44))
	for _, s := range servers {
		s.mu.Lock()
		alive := s.alive
		role := s.role
		term := s.currentTerm
		leader := s.leaderID
		voted := s.votedFor
		s.mu.Unlock()

		roleStr := role.String()
		if !alive {
			roleStr = "CRASHED  "
		}
		leaderStr := fmt.Sprintf("S%d", leader)
		votedStr := fmt.Sprintf("S%d", voted)
		if leader == -1 {
			leaderStr = "---"
		}
		if voted == -1 {
			votedStr = "---"
		}
		fmt.Printf("  S%-5d %s %-6d %-8s %s\n",
			s.id, roleStr, term, leaderStr, votedStr)
	}
	fmt.Println()
}

func findLeader(servers []*Server) *Server {
	for _, s := range servers {
		s.mu.Lock()
		isLeader := s.role == LEADER && s.alive
		s.mu.Unlock()
		if isLeader {
			return s
		}
	}
	return nil
}

// ─────────────────────────────────────────────
//  MAIN
// ─────────────────────────────────────────────

func main() {
	rand.Seed(time.Now().UnixNano())

	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("  RAFT LEADER ELECTION")
	fmt.Println(strings.Repeat("=", 60))

	const n = 5
	servers := make([]*Server, n)
	for i := 0; i < n; i++ {
		servers[i] = NewServer(i + 1)
	}
	for _, s := range servers {
		for _, peer := range servers {
			if peer.id != s.id {
				s.peers[peer.id] = peer
			}
		}
	}
	for _, s := range servers {
		go s.HandleMessages()
		go s.runElectionTimer()
		go s.runHeartbeatSender()
	}

	// ── Scenario 1: initial election ──────────────────────────
	fmt.Println("\n  Scenario 1: Initial election")
	fmt.Println(strings.Repeat("-", 60))
	time.Sleep(700 * time.Millisecond)
	printStatus(servers)

	// ── Scenario 2: leader crashes ────────────────────────────
	fmt.Println(strings.Repeat("-", 60))
	fmt.Println("  Scenario 2: Leader crashes")
	fmt.Println(strings.Repeat("-", 60))
	leader := findLeader(servers)
	if leader != nil {
		leader.Crash()
	}
	time.Sleep(700 * time.Millisecond)
	printStatus(servers)

	// ── Scenario 3: recovery ──────────────────────────────────
	fmt.Println(strings.Repeat("-", 60))
	fmt.Println("  Scenario 3: Crashed server recovers as follower")
	fmt.Println(strings.Repeat("-", 60))
	if leader != nil {
		leader.Recover()
	}
	time.Sleep(700 * time.Millisecond)
	printStatus(servers)

	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("  DEADLOCK FIXES APPLIED")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println(`
  Three rules followed throughout:

    1. Never call sendTo or broadcastMsg while holding s.mu
       → capture needed values, unlock, then send

    2. Never acquire another server's lock while holding s.mu
       → sendTo locks the peer only after releasing own lock

    3. stepDownIfNeeded called only while lock is held
       → it only mutates local state, never sends messages`)
}
