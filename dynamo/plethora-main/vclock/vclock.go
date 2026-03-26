package vclock

// VClock is a vector clock, mapping node IDs to their version numbers.
type VClock map[string]uint64

func NewVClock() VClock {
	return make(VClock)
}

func (vc VClock) Increment(nodeID string) {
	vc[nodeID]++
}

// returns true if vc descends from other, i.e. vc is a later version than other
func (vc VClock) Descends(other VClock) bool {
	for nodeID, version := range other {
		if vc[nodeID] < version {
			return false
		}
	}
	return true
}

func (vc VClock) Conflicts(other VClock) bool {
	return !vc.Descends(other) && !other.Descends(vc)
}

// we use semantic reconciliation, so we don't consider concurrent versions as conflicts
// instead, we merge them by taking the max version for each node
// eg if vc1 = {n1: 2, n2: 1} and vc2 = {n1: 1, n2: 3},
// then vc1.Merge(vc2) = {n1: 2, n2: 3}
// i.e return the highest version for each node, which represents the most recent update from that node
func (vc VClock) Merge(other VClock) VClock {
	merged := NewVClock()
	for nodeID, version := range vc {
		merged[nodeID] = version
	}
	for nodeID, version := range other {
		if merged[nodeID] < version {
			merged[nodeID] = version
		}
	}
	return merged
}

func (vc VClock) Copy() VClock {
	copied := NewVClock()
	for nodeID, version := range vc {
		copied[nodeID] = version
	}
	return copied
}
