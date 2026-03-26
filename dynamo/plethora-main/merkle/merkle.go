package merkle

import (
	"crypto/md5"
	"sort"
)

type KeyHash struct {
	Key  string
	Hash [16]byte
}

type MerkleNode struct {
	Hash  [16]byte
	Left  *MerkleNode
	Right *MerkleNode
	Key   string // only set on leaf nodes
}

// Build constructs a merkle tree from a set of key-hash pairs.
// Sorts entries by key, pads to the next power of 2, then merges
// bottom-up: parent hash = md5(left.hash + right.hash).
func Build(entries []KeyHash) *MerkleNode {
	if len(entries) == 0 {
		return nil
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Key < entries[j].Key
	})

	// create leaf nodes
	leaves := make([]*MerkleNode, len(entries))
	for i, e := range entries {
		leaves[i] = &MerkleNode{Hash: e.Hash, Key: e.Key}
	}

	// pad to next power of 2
	for len(leaves)&(len(leaves)-1) != 0 {
		leaves = append(leaves, &MerkleNode{})
	}

	// merge bottom-up
	layer := leaves
	for len(layer) > 1 {
		var next []*MerkleNode
		for i := 0; i < len(layer); i += 2 {
			parent := &MerkleNode{
				Left:  layer[i],
				Right: layer[i+1],
			}
			var combined [32]byte
			copy(combined[:16], layer[i].Hash[:])
			copy(combined[16:], layer[i+1].Hash[:])
			parent.Hash = md5.Sum(combined[:])
			next = append(next, parent)
		}
		layer = next
	}

	return layer[0]
}

// Diff walks two merkle trees top-down and returns the keys that differ.
// If roots match, the entire subtree is in sync. On mismatch, recurse
// into children to find exactly which leaves diverged.
func Diff(a, b *MerkleNode) []string {
	if a == nil && b == nil {
		return nil
	}
	// one side has data the other doesn't
	if a == nil {
		return collectKeys(b)
	}
	if b == nil {
		return collectKeys(a)
	}
	// hashes match, subtree is in sync
	if a.Hash == b.Hash {
		return nil
	}
	// both are leaves, this key diverged
	if a.Left == nil && b.Left == nil {
		if a.Key != "" {
			return []string{a.Key}
		}
		if b.Key != "" {
			return []string{b.Key}
		}
		return nil
	}
	// recurse into children
	left := Diff(a.Left, b.Left)
	right := Diff(a.Right, b.Right)
	return append(left, right...)
}

// collectKeys gathers all non-empty leaf keys from a subtree.
func collectKeys(n *MerkleNode) []string {
	if n == nil {
		return nil
	}
	if n.Left == nil && n.Right == nil {
		if n.Key != "" {
			return []string{n.Key}
		}
		return nil
	}
	left := collectKeys(n.Left)
	right := collectKeys(n.Right)
	return append(left, right...)
}
