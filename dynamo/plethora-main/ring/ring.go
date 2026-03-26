package ring

import (
	"crypto/md5"
	"encoding/binary"
	"fmt"

	"github.com/pixperk/plethora/client"
	"github.com/pixperk/plethora/node"
	"github.com/pixperk/plethora/types"
	"github.com/pixperk/plethora/vclock"
)

type Ring struct {
	Nodes      []*node.Node
	NodeMap    map[string]*node.Node // nodeID -> node for O(1) lookup
	Partitions []Partition
	Q          int // total number of partitions, fixed
	//N is the replication factor, we walk clockwise from a node and pick up n nodes to build up
	//preference list for the node, to image the rights
	N int
	//quorum
	R int //min number of reads
	W int //min number of writes

	// IsAlive reports whether a node is reachable. Used by Put for sloppy quorum
	// to skip dead nodes and find stand-ins. Defaults to always-alive if not set.
	IsAlive func(nodeID string) bool
}

// creates a ring with Q equal-sized partitions distributed evenly across the given nodes.
func NewRing(q int, n int, R int, W int, nodes []*node.Node) (*Ring, error) {
	s := len(nodes)
	if s == 0 {
		return nil, fmt.Errorf("need at least one node")
	}
	if q%s != 0 {
		return nil, fmt.Errorf("Q (%d) must be divisible by number of nodes (%d)", q, s)
	}

	if R > n {
		return nil, fmt.Errorf("R (%d) cannot be greater than N (%d)", R, n)
	}
	if W > n {
		return nil, fmt.Errorf("W (%d) cannot be greater than N (%d)", W, n)
	}
	if R+W <= n {
		return nil, fmt.Errorf("R + W (%d) must be greater than N (%d) for quorum", R+W, n)
	}

	partitions := make([]Partition, q)
	for i := 0; i < q; i++ {
		partitions[i] = Partition{
			ID: i,
			// round-robin: partition i is assigned to node i%S, giving each node Q/S partitions
			Token: Token{NodeID: nodes[i%s].NodeID},
		}
	}

	nodeMap := make(map[string]*node.Node, s)
	for _, n := range nodes {
		nodeMap[n.NodeID] = n
	}

	return &Ring{
		Nodes:      nodes,
		NodeMap:    nodeMap,
		Partitions: partitions,
		Q:          q,
		N:          n,
		R:          R,
		W:          W,
	}, nil
}

// Lookup hashes a key and returns the node that owns the partition it falls into.
// hash(key) % Q -> partition index -> token -> owner node
func (r *Ring) Lookup(key string) *node.Node {
	hash := md5Hash(key)
	partitionID := int(hash % uint64(r.Q))
	ownerID := r.Partitions[partitionID].Token.NodeID
	node, exists := r.NodeMap[ownerID]
	if !exists {
		return nil // should never happen since partitions are pre-initialized
	}
	return node
}

// md5Hash returns the first 8 bytes of the MD5 digest as a uint64.
func md5Hash(key string) uint64 {
	sum := md5.Sum([]byte(key))
	return binary.BigEndian.Uint64(sum[:8])
}

func (r *Ring) AddNode(newNode *node.Node) error {
	// validate before mutating
	s := len(r.Nodes) + 1
	if r.Q%s != 0 {
		return fmt.Errorf("Q (%d) must be divisible by number of nodes (%d)", r.Q, s)
	}

	r.Nodes = append(r.Nodes, newNode)
	r.NodeMap[newNode.NodeID] = newNode

	// reassign partitions to maintain Q/S partitions per node
	for i := 0; i < r.Q; i++ {
		r.Partitions[i].Token.NodeID = r.Nodes[i%s].NodeID
	}

	return nil
}

func (r *Ring) RemoveNode(nodeID string) error {
	// validate before mutating
	s := len(r.Nodes) - 1
	if s == 0 {
		return fmt.Errorf("cannot remove last node")
	}
	if r.Q%s != 0 {
		return fmt.Errorf("Q (%d) must be divisible by number of nodes (%d)", r.Q, s)
	}

	delete(r.NodeMap, nodeID)

	found := false
	// remove from Nodes slice
	for i, n := range r.Nodes {
		if n.NodeID == nodeID {
			r.Nodes = append(r.Nodes[:i], r.Nodes[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("node with ID %s not found", nodeID)
	}

	// reassign partitions to maintain Q/S partitions per node
	for i := 0; i < r.Q; i++ {
		r.Partitions[i].Token.NodeID = r.Nodes[i%s].NodeID
	}

	return nil
}

// Target represents a node to send a write to. If HintFor is non-empty,
// this node is a stand-in and the write should be a hinted put.
type Target struct {
	Node    *node.Node
	HintFor string // empty = preferred node, non-empty = stand-in covering for this nodeID
}

// PreferenceList walks the ring clockwise from the key's partition and
// collects N healthy nodes. If a preferred node is dead, it's skipped
// and the next healthy non-preferred node becomes a stand-in with a hint.
func (r *Ring) PreferenceList(key types.Key) []Target {
	hash := md5Hash(string(key))
	partitionID := int(hash % uint64(r.Q))

	// first pass: find the N ideal (ignoring health) preferred nodes
	preferred := make(map[string]bool)
	seen := make(map[string]bool)
	for i := 0; i < r.Q && len(preferred) < r.N; i++ {
		idx := (partitionID + i) % r.Q
		nid := r.Partitions[idx].Token.NodeID
		if seen[nid] {
			continue
		}
		seen[nid] = true
		preferred[nid] = true
	}

	// second pass: walk the ring again, collect N healthy targets.
	// preferred + alive = regular put. preferred + dead = skip, queue for stand-in.
	// non-preferred + alive = stand-in for the first uncovered dead preferred node.
	var targets []Target
	var deadPreferred []string // preferred nodes that were dead, need coverage
	seen = make(map[string]bool)

	for i := 0; i < r.Q && len(targets) < r.N; i++ {
		idx := (partitionID + i) % r.Q
		nid := r.Partitions[idx].Token.NodeID
		if seen[nid] {
			continue
		}
		seen[nid] = true

		if preferred[nid] {
			if r.isAlive(nid) {
				targets = append(targets, Target{Node: r.NodeMap[nid]})
			} else {
				deadPreferred = append(deadPreferred, nid)
			}
		} else if r.isAlive(nid) && len(deadPreferred) > 0 {
			// stand-in: covers the first uncovered dead preferred node
			targetID := deadPreferred[0]
			deadPreferred = deadPreferred[1:]
			targets = append(targets, Target{Node: r.NodeMap[nid], HintFor: targetID})
		}
	}

	return targets
}

func (r *Ring) isAlive(nodeID string) bool {
	if r.IsAlive == nil {
		return true
	}
	return r.IsAlive(nodeID)
}

// Get reads from N nodes in the preference list, requires at least R responses.
// Deduplicates using vector clocks and returns only causally distinct versions.
func (r *Ring) Get(key types.Key) ([]types.Value, error) {
	targets := r.PreferenceList(key)
	var allVals []types.Value
	responses := 0
	for _, t := range targets {
		vals, found, err := client.RemoteGet(t.Node.Addr, key)
		if err != nil {
			continue
		}
		responses++
		if found {
			allVals = append(allVals, vals...)
		}
	}
	if responses < r.R {
		return nil, fmt.Errorf("quorum not met: got %d responses, need R=%d", responses, r.R)
	}
	if len(allVals) == 0 {
		return nil, nil
	}
	return dedup(allVals), nil
}

// dedup filters a list of values down to only causally distinct versions.
// Drops any value that is an ancestor of another value in the list.
func dedup(vals []types.Value) []types.Value {
	var result []types.Value
	for i, v := range vals {
		dominated := false
		for j, other := range vals {
			if i == j {
				continue
			}
			// if other descends from v (and they're not equal), v is an ancestor, drop it
			if other.Clock.Descends(v.Clock) && !v.Clock.Descends(other.Clock) {
				dominated = true
				break
			}
			// if they have identical clocks, keep only the first occurrence
			if other.Clock.Descends(v.Clock) && v.Clock.Descends(other.Clock) && j < i {
				dominated = true
				break
			}
		}
		if !dominated {
			result = append(result, v)
		}
	}
	return result
}

// Put writes to N healthy nodes, requires at least W successes.
// PreferenceList already picked the first N healthy nodes (with stand-ins
// for any dead preferred nodes). Put just iterates and sends.
func (r *Ring) Put(key types.Key, val string, ctx vclock.VClock) error {
	targets := r.PreferenceList(key)
	if len(targets) == 0 {
		return fmt.Errorf("no nodes available")
	}

	// coordinator builds the clock and value
	coordinator := targets[0].Node
	var clock vclock.VClock
	if ctx == nil {
		clock = vclock.NewVClock()
	} else {
		clock = ctx.Copy()
	}
	clock.Increment(coordinator.NodeID)

	value := types.Value{
		Data:  val,
		Clock: clock,
	}

	acks := 0
	for _, t := range targets {
		if t.HintFor == "" {
			// preferred node: regular put
			if client.RemotePut(t.Node.Addr, key, value) == nil {
				acks++
			}
		} else {
			// stand-in: hinted put tagged with the dead preferred node
			if client.RemoteHintedPut(t.Node.Addr, key, value, t.HintFor) == nil {
				acks++
			}
		}
	}

	if acks < r.W {
		return fmt.Errorf("quorum not met: got %d acks, need W=%d", acks, r.W)
	}
	return nil
}

// ReplicaPeers returns the set of nodes that share at least one key range
// with the given node. For each partition, we build the preference list and
// check if nodeID appears in it. If so, the other nodes in that list are
// replica peers.
func (r *Ring) ReplicaPeers(nodeID string) []*node.Node {
	seen := make(map[string]bool)
	seen[nodeID] = true // exclude self

	for i := 0; i < r.Q; i++ {
		plist := r.prefListForPartition(i)
		inList := false
		for _, n := range plist {
			if n.NodeID == nodeID {
				inList = true
				break
			}
		}
		if !inList {
			continue
		}
		for _, n := range plist {
			seen[n.NodeID] = true
		}
	}

	delete(seen, nodeID)
	peers := make([]*node.Node, 0, len(seen))
	for id := range seen {
		peers = append(peers, r.NodeMap[id])
	}
	return peers
}

// prefListForPartition returns the N distinct nodes for a given partition index.
func (r *Ring) prefListForPartition(partitionID int) []*node.Node {
	nodes := make([]*node.Node, 0, r.N)
	seen := make(map[string]bool)
	for i := 0; i < r.Q && len(nodes) < r.N; i++ {
		idx := (partitionID + i) % r.Q
		nid := r.Partitions[idx].Token.NodeID
		if seen[nid] {
			continue
		}
		seen[nid] = true
		nodes = append(nodes, r.NodeMap[nid])
	}
	return nodes
}
