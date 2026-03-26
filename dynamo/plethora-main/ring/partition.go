package ring

// Dynamo Strategy 3: Q/S tokens per node, equal-sized partitions.
//
// - The hash space is divided into Q equally-sized partitions. Q is fixed and never changes.
// - There are S nodes in the system.
// - Each node owns Q/S partitions. Owning a partition = holding a token for it.
// - A token is just the assignment of one partition to one node.
// - Each partition has exactly one owner (one token). Each node holds Q/S tokens.
// - When a node joins/leaves, tokens are reassigned, partitions themselves don't change.

// Token is a node's claim on a single partition.
type Token struct {
	NodeID string
}

// Partition is a fixed, equal-sized slice of the hash space.
// It has exactly one Token, pointing to its owner node.
type Partition struct {
	ID    int // 0..Q-1
	Token Token
}
