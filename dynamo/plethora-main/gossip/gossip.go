package gossip

import (
	"math/rand"
	"sync"
	"time"
)

// MemberEntry represents a node's membership state as seen by the local node.
type MemberEntry struct {
	NodeID    string
	Addr      string
	Heartbeat uint64    // monotonic counter, only the owning node increments
	Timestamp time.Time // local wall time when we last saw this node's heartbeat increase
}

// MemberList is a thread-safe gossip membership list.
// Each node maintains one and merges it with peers during gossip rounds.
type MemberList struct {
	mu      sync.RWMutex
	members map[string]*MemberEntry
	selfID  string
	tFail   time.Duration // if now - timestamp > tFail, node is suspected down
}

// NewMemberList creates a membership list with the local node as the first entry.
func NewMemberList(selfID, selfAddr string, tFail time.Duration) *MemberList {
	m := &MemberList{
		members: make(map[string]*MemberEntry),
		selfID:  selfID,
		tFail:   tFail,
	}
	m.members[selfID] = &MemberEntry{
		NodeID:    selfID,
		Addr:      selfAddr,
		Heartbeat: 0,
		Timestamp: time.Now(),
	}
	return m
}

// AddSeed adds a seed node to the membership list.
// Seeds are assumed alive initially; gossip will update their heartbeat.
func (m *MemberList) AddSeed(nodeID, addr string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.members[nodeID]; !exists {
		m.members[nodeID] = &MemberEntry{
			NodeID:    nodeID,
			Addr:      addr,
			Heartbeat: 0,
			Timestamp: time.Now(),
		}
	}
}

// Tick increments this node's own heartbeat counter.
// Called once per gossip round.
func (m *MemberList) Tick() {
	m.mu.Lock()
	defer m.mu.Unlock()
	self := m.members[m.selfID]
	self.Heartbeat++
	self.Timestamp = time.Now()
}

// Merge integrates a remote node's membership entries into the local list.
// For each entry: if remote heartbeat > local heartbeat, adopt it and reset timestamp to now.
// New nodes are added. Self entry is never overwritten.
func (m *MemberList) Merge(remote []MemberEntry) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	for _, r := range remote {
		if r.NodeID == m.selfID {
			continue
		}
		local, exists := m.members[r.NodeID]
		if !exists {
			m.members[r.NodeID] = &MemberEntry{
				NodeID:    r.NodeID,
				Addr:      r.Addr,
				Heartbeat: r.Heartbeat,
				Timestamp: now,
			}
		} else if r.Heartbeat > local.Heartbeat {
			local.Heartbeat = r.Heartbeat
			local.Addr = r.Addr
			local.Timestamp = now
		}
	}
}

// Entries returns a snapshot of all membership entries for sending to a peer.
func (m *MemberList) Entries() []MemberEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	entries := make([]MemberEntry, 0, len(m.members))
	for _, e := range m.members {
		entries = append(entries, *e)
	}
	return entries
}

// IsAlive reports whether a node is considered alive.
func (m *MemberList) IsAlive(nodeID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	e, exists := m.members[nodeID]
	if !exists {
		return false
	}
	return time.Since(e.Timestamp) < m.tFail
}

// Alive returns all members currently considered alive, excluding self.
func (m *MemberList) Alive() []MemberEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var alive []MemberEntry
	for _, e := range m.members {
		if e.NodeID == m.selfID {
			continue
		}
		if time.Since(e.Timestamp) < m.tFail {
			alive = append(alive, *e)
		}
	}
	return alive
}

// RandomPeer picks a random peer (not self) for gossip.
// Includes dead peers so we can detect recoveries.
func (m *MemberList) RandomPeer() (MemberEntry, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	peers := make([]*MemberEntry, 0, len(m.members)-1)
	for _, e := range m.members {
		if e.NodeID == m.selfID {
			continue
		}
		peers = append(peers, e)
	}
	if len(peers) == 0 {
		return MemberEntry{}, false
	}
	return *peers[rand.Intn(len(peers))], true
}
