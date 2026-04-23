package main

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"
)

// ─────────────────────────────────────────────
//  RAFT ROLES AND MESSAGE TYPES
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
	HEARTBEAT_ACK
)

func (m MsgType) String() string {
	return [...]string{
		"VOTE_REQUEST ",
		"VOTE_RESPONSE",
		"HEARTBEAT    ",
		"HEARTBEAT_ACK",
	}[m]
}

type Message struct {
	kind        MsgType
	term        int  // sender's current term
	senderID    int
	voteGranted bool // used in VOTE_RESPONSE
	lastLogTerm int  // candidate's last log term
	lastLogIdx  int  // candidate's last log index
}

func (m Message) String() string {
	return fmt.Sprintf("{%s term=%d from=S%d}",
		m.kind, m.term, m.senderID)
}

// ─────────────────────────────────────────────
//  SERVER
// ─────────────────────────────────────────────

type Server struct {
	mu sync.Mutex

	id   int
	role Role

	// ── persistent state ──────────────────────
	currentTerm int
	votedFor    int // -1 = not voted this term

	// ── volatile state ────────────────────────
	leaderID        int
	votesReceived   int
	peers           map[int]*Server
	inbox           chan Message

	// ── timers ────────────────────────────────
	electionTimeout  time.Duration
	lastHeartbeat    time.Time
	alive            bool
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

// randomTimeout returns a random election timeout between 150-300ms
// (Raft spec recommends 150-300ms)
func randomTimeout() time.Duration {
	return time.Duration(150+rand.Intn(150)) * time.Millisecond
}

// ── messaging ─────────────────────────────────────────────────

func (s *Server) sendTo(peerID int, msg Message) {
	peer, ok := s.peers[peerID]
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
		delay := time.Duration(rand.Intn(15)+2) * time.Millisecond
		time.Sleep(delay)
		peer.inbox <- msg
	}()
}

func (s *Server) broadcast(msg Message) {
	s.mu.Lock()
	peers := s.peers
	s.mu.Unlock()
	for id := range peers {
		s.sendTo(id, msg)
	}
}

// ── election timeout watchdog ─────────────────────────────────

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

		// only followers and candidates time out
		if role == LEADER {
			continue
		}

		if time.Since(last) > timeout {
			s.startElection()
		}
	}
}

// ── heartbeat sender (leader only) ───────────────────────────

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

		msg := Message{
			kind:     HEARTBEAT,
			term:     term,
			senderID: id,
		}
		s.broadcast(msg)
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

		switch msg.kind {
		case VOTE_REQUEST:
			s.handleVoteRequest(msg)
		case VOTE_RESPONSE:
			s.handleVoteResponse(msg)
		case HEARTBEAT:
			s.handleHeartbeat(msg)
		case HEARTBEAT_ACK:
			// leader acknowledges — nothing to do in this demo
		}
	}
}

// ── Raft rule: if we see a higher term, revert to follower ────

func (s *Server) maybeStepDown(term int) {
	if term > s.currentTerm {
		fmt.Printf("  [S%d] term %d > current term %d — stepping down to FOLLOWER\n",
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
	defer s.mu.Unlock()

	s.maybeStepDown(msg.term)

	grant := false

	if msg.term < s.currentTerm {
		// candidate's term is stale — deny
		fmt.Printf("  [S%d] deny vote for S%d (stale term %d < %d)\n",
			s.id, msg.senderID, msg.term, s.currentTerm)
	} else if s.votedFor == -1 || s.votedFor == msg.senderID {
		// we haven't voted this term OR we already voted for this candidate
		// (simplified: no log comparison in this demo)
		grant = true
		s.votedFor = msg.senderID
		s.lastHeartbeat = time.Now() // reset timer on granting vote
		fmt.Printf("  [S%d] grant vote → S%d (term %d)\n",
			s.id, msg.senderID, msg.term)
	} else {
		fmt.Printf("  [S%d] deny vote for S%d (already voted for S%d)\n",
			s.id, msg.senderID, s.votedFor)
	}

	resp := Message{
		kind:        VOTE_RESPONSE,
		term:        s.currentTerm,
		senderID:    s.id,
		voteGranted: grant,
	}
	go func() {
		s.sendTo(msg.senderID, resp)
	}()
}

// ── handle VOTE_RESPONSE ──────────────────────────────────────

func (s *Server) handleVoteResponse(msg Message) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.maybeStepDown(msg.term)

	if s.role != CANDIDATE {
		return // we already won or lost
	}
	if msg.term != s.currentTerm {
		return // stale response
	}

	if msg.voteGranted {
		s.votesReceived++
		fmt.Printf("  [S%d] received vote from S%d (%d/%d)\n",
			s.id, msg.senderID,
			s.votesReceived, len(s.peers)+1)

		majority := (len(s.peers)+1)/2 + 1
		if s.votesReceived >= majority {
			s.becomeLeader()
		}
	}
}

// ── handle HEARTBEAT ──────────────────────────────────────────

func (s *Server) handleHeartbeat(msg Message) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.maybeStepDown(msg.term)

	if msg.term >= s.currentTerm {
		s.role = FOLLOWER
		s.leaderID = msg.senderID
		s.lastHeartbeat = time.Now()
		s.electionTimeout = randomTimeout() // reset timeout
	}
}

// ── start election ────────────────────────────────────────────
//
// Raft election rules:
//  1. Increment current term
//  2. Vote for yourself
//  3. Reset election timeout
//  4. Send VOTE_REQUEST to all peers
//  5. Wait for majority votes or timeout

func (s *Server) startElection() {
	s.mu.Lock()
	s.currentTerm++
	s.role = CANDIDATE
	s.votedFor = s.id
	s.votesReceived = 1 // vote for self
	s.leaderID = -1
	s.lastHeartbeat = time.Now()
	s.electionTimeout = randomTimeout()
	term := s.currentTerm
	id := s.id
	s.mu.Unlock()

	fmt.Printf("  [S%d] starting election for term %d\n", id, term)

	req := Message{
		kind:     VOTE_REQUEST,
		term:     term,
		senderID: id,
	}
	s.broadcast(req)
}

// ── become leader ─────────────────────────────────────────────

func (s *Server) becomeLeader() {
	// must be called with s.mu held
	s.role = LEADER
	s.leaderID = s.id
	fmt.Printf("  [S%d] *** ELECTED LEADER for term %d ***\n",
		s.id, s.currentTerm)
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
	fmt.Printf("  [S%d] *** RECOVERED (term=%d) ***\n",
		s.id, s.currentTerm)
}

// ─────────────────────────────────────────────
//  HELPERS
// ─────────────────────────────────────────────

func printStatus(servers []*Server) {
	fmt.Println()
	fmt.Printf("  %-6s %-10s %-8s %-8s %s\n",
		"Server", "Role", "Term", "Leader", "VotedFor")
	fmt.Println("  " + strings.Repeat("-", 46))
	for _, s := range servers {
		s.mu.Lock()
		alive := s.alive
		role := s.role
		term := s.currentTerm
		leader := s.leaderID
		voted := s.votedFor
		s.mu.Unlock()

		status := fmt.Sprintf("%-10s", role)
		leaderStr := fmt.Sprintf("S%d", leader)
		votedStr := fmt.Sprintf("S%d", voted)
		if leader == -1 {
			leaderStr = "---"
		}
		if voted == -1 {
			votedStr = "---"
		}
		aliveStr := ""
		if !alive {
			aliveStr = " (CRASHED)"
			status = "CRASHED   "
		}
		fmt.Printf("  S%-5d %s %-8d %-8s %s%s\n",
			s.id, status, term, leaderStr, votedStr, aliveStr)
	}
	fmt.Println()
}

// ─────────────────────────────────────────────
//  MAIN
// ─────────────────────────────────────────────

func main() {
	rand.Seed(time.Now().UnixNano())

	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("  RAFT LEADER ELECTION")
	fmt.Println(strings.Repeat("=", 60))

	// create 5 servers
	const n = 5
	servers := make([]*Server, n)
	for i := 0; i < n; i++ {
		servers[i] = NewServer(i + 1)
	}

	// wire peers
	for _, s := range servers {
		for _, peer := range servers {
			if peer.id != s.id {
				s.peers[peer.id] = peer
			}
		}
	}

	// start message handlers and timers
	for _, s := range servers {
		go s.HandleMessages()
		go s.runElectionTimer()
		go s.runHeartbeatSender()
	}

	// ── Scenario 1: initial election ──────────────────────────
	fmt.Println("\n  Scenario 1: Initial election at startup")
	fmt.Println(strings.Repeat("-", 60))
	fmt.Println("  All servers start as followers, randomized timeouts")
	fmt.Println("  First to time out becomes candidate and requests votes")

	time.Sleep(700 * time.Millisecond)
	fmt.Println("\n  Result after initial election:")
	printStatus(servers)

	// ── Scenario 2: leader crashes ────────────────────────────
	fmt.Println(strings.Repeat("-", 60))
	fmt.Println("  Scenario 2: Leader crashes — followers re-elect")
	fmt.Println(strings.Repeat("-", 60))

	// find current leader
	var currentLeader *Server
	for _, s := range servers {
		s.mu.Lock()
		if s.role == LEADER {
			currentLeader = s
		}
		s.mu.Unlock()
	}

	if currentLeader != nil {
		fmt.Printf("\n  Crashing leader S%d\n", currentLeader.id)
		currentLeader.Crash()
	}

	fmt.Println("  Followers detect missing heartbeat, start new election")
	time.Sleep(700 * time.Millisecond)
	fmt.Println("\n  Result after leader crash:")
	printStatus(servers)

	// ── Scenario 3: crashed leader recovers ───────────────────
	fmt.Println(strings.Repeat("-", 60))
	fmt.Println("  Scenario 3: Crashed server recovers")
	fmt.Println(strings.Repeat("-", 60))

	if currentLeader != nil {
		fmt.Printf("\n  Recovering S%d\n", currentLeader.id)
		currentLeader.Recover()
	}

	time.Sleep(700 * time.Millisecond)
	fmt.Println("\n  Result after recovery:")
	fmt.Println("  (recovered server rejoins as FOLLOWER — does NOT")
	fmt.Println("  reclaim leadership unlike Bully algorithm)")
	printStatus(servers)

	// ── Scenario 4: split vote ────────────────────────────────
	fmt.Println(strings.Repeat("-", 60))
	fmt.Println("  Scenario 4: Simulate split vote")
	fmt.Println("  (two candidates with equal timeouts — Raft retries)")
	fmt.Println(strings.Repeat("-", 60))

	// pause heartbeats by crashing leader temporarily
	for _, s := range servers {
		s.mu.Lock()
		if s.role == LEADER {
			s.Crash()
		}
		s.mu.Unlock()
	}

	// force two servers to start elections simultaneously
	time.Sleep(50 * time.Millisecond)
	fmt.Println("\n  Forcing S1 and S2 to start elections simultaneously")
	go servers[0].startElection()
	go servers[1].startElection()

	time.Sleep(800 * time.Millisecond)
	fmt.Println("\n  Result — Raft retried with new random timeout:")
	printStatus(servers)

	// ── Summary ───────────────────────────────────────────────
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("  RAFT vs BULLY vs RING")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println(`
  Property          Raft          Bully         Ring
  ──────────────────────────────────────────────────────
  Winner            Any majority  Highest ID    Highest ID
  On recovery       Stays follower Reclaims     Reclaims
  Split vote        Retry (random  Not possible  Not possible
                    timeout)
  Heartbeat         Yes (50ms)    No            No
  Term/epoch        Yes           No            No
  Messages per      O(N)          O(N²)         O(N)
  election

  Key Raft properties:

    Terms      — logical clock; stale messages rejected
    Randomized timeout — prevents simultaneous candidates
                         and resolves split votes naturally
    Step down  — any server seeing a higher term reverts
                 to follower immediately
    Heartbeat  — leader suppresses elections while alive`)
}
