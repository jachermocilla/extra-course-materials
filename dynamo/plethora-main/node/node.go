package node

import (
	"sync"

	"github.com/pixperk/plethora/merkle"
	"github.com/pixperk/plethora/storage"
	"github.com/pixperk/plethora/types"
	"github.com/pixperk/plethora/vclock"
)

type HintedItem struct {
	Key   types.Key
	Value types.Value
}

type Hints map[string][]HintedItem // targetNodeID -> list of hinted items for that node

type Node struct {
	NodeID  string
	Addr    string
	Storage *storage.Storage

	hintLock  sync.Mutex
	HintStore Hints

	merkleMu    sync.Mutex
	merkleTree  *merkle.MerkleNode
	merkleDirty bool
}

func NewNode(nodeID string, addr string) *Node {
	return &Node{
		NodeID:      nodeID,
		Addr:        addr,
		Storage:     storage.NewStorage(),
		HintStore:   make(Hints),
		merkleDirty: true,
	}
}

func (n *Node) Get(key types.Key) ([]types.Value, bool) {
	return n.Storage.Get(key)
}

// Put is the single-node write path for standalone conflict resolution testing.
// Not used by the ring; the ring coordinator builds the clock itself and calls Store.
func (n *Node) Put(key types.Key, val string, ctx vclock.VClock) {
	var clock vclock.VClock
	if ctx == nil {
		clock = vclock.NewVClock()
	} else {
		clock = ctx.Copy()
	}
	clock.Increment(n.NodeID)

	value := types.Value{
		Data:  val,
		Clock: clock,
	}

	n.Storage.Put(key, value)
	n.markMerkleDirty()
}

// Store writes a pre-built value directly to storage without modifying the clock.
// Used by the ring coordinator to replicate an already-prepared value to nodes.
func (n *Node) Store(key types.Key, val types.Value) {
	n.Storage.Put(key, val)
	n.markMerkleDirty()
}

func (n *Node) markMerkleDirty() {
	n.merkleMu.Lock()
	n.merkleDirty = true
	n.merkleMu.Unlock()
}

// MerkleTree returns the node's merkle tree, rebuilding it lazily if data changed.
func (n *Node) MerkleTree() *merkle.MerkleNode {
	n.merkleMu.Lock()
	defer n.merkleMu.Unlock()
	if n.merkleDirty {
		n.merkleTree = merkle.Build(n.Storage.KeyHashes())
		n.merkleDirty = false
	}
	return n.merkleTree
}

func (n *Node) StoreHint(targetNodeID string, key types.Key, val types.Value) {
	n.hintLock.Lock()
	defer n.hintLock.Unlock()
	n.HintStore[targetNodeID] = append(n.HintStore[targetNodeID], HintedItem{Key: key, Value: val})
}

func (n *Node) DrainHints(targetNodeID string) []HintedItem {
	n.hintLock.Lock()
	defer n.hintLock.Unlock()
	items := n.HintStore[targetNodeID]
	delete(n.HintStore, targetNodeID)
	return items
}

func BuildAddrToID(nodes []*Node) map[string]string {
	m := make(map[string]string, len(nodes))
	for _, n := range nodes {
		m[n.Addr] = n.NodeID
	}
	return m
}

func BuildIDToAddr(nodes []*Node) map[string]string {
	m := make(map[string]string, len(nodes))
	for _, n := range nodes {
		m[n.NodeID] = n.Addr
	}
	return m
}
