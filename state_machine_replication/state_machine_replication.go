// =============================================================================
// State Machine Replication in Go — Tutorial
// =============================================================================
//
// CONCEPT
// -------
// State Machine Replication (SMR) is a technique to make a service fault-tolerant
// by running identical copies ("replicas") of a deterministic state machine and
// feeding them the same sequence of commands in the same order.
//
// Key invariant: if every replica starts from the same initial state and applies
// the same commands in the same order, they all end up in the same state.
//
// This file shows a self-contained, runnable example with:
//   1. A simple KV-store state machine
//   2. A log that orders commands (simulating consensus)
//   3. Multiple replicas that apply the log
//   4. A leader that sequences writes
//   5. A read path that queries any replica
//
// Run with:  go run state_machine_replication.go
// =============================================================================

package main

import (
	"fmt"
	"strings"
	"sync"
)

// =============================================================================
// PART 1 – The State Machine
// =============================================================================
// A state machine has:
//   • A state  (the data it holds)
//   • Commands (inputs that change state)
//   • Apply()  (deterministic transition function)
//
// Determinism is crucial: given the same state and the same command, Apply()
// must always produce the same next state — no random numbers, no wall-clock
// timestamps, no outside I/O.

// Command represents a single operation sent to every replica.
type Command struct {
	Index int    // position in the log (assigned by the leader / consensus layer)
	Op    string // "SET" | "DEL"
	Key   string
	Value string // only used for SET
}

// KVStateMachine is a simple in-memory key-value store.
type KVStateMachine struct {
	mu      sync.RWMutex
	store   map[string]string
	applied int // highest log index applied so far
}

func NewKVStateMachine() *KVStateMachine {
	return &KVStateMachine{store: make(map[string]string)}
}

// Apply executes one command. This is the heart of the state machine.
// It MUST be deterministic.
func (sm *KVStateMachine) Apply(cmd Command) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	switch cmd.Op {
	case "SET":
		sm.store[cmd.Key] = cmd.Value
	case "DEL":
		delete(sm.store, cmd.Key)
	}
	sm.applied = cmd.Index
}

// Get reads a value. Reads don't go through the log, so they just query local state.
func (sm *KVStateMachine) Get(key string) (string, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	v, ok := sm.store[key]
	return v, ok
}

// AppliedIndex returns the last log index this replica has processed.
func (sm *KVStateMachine) AppliedIndex() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.applied
}

// =============================================================================
// PART 2 – The Replicated Log
// =============================================================================
// In a real system this is provided by a consensus algorithm (Raft, Paxos, …).
// Here we use a simple in-memory log protected by a mutex. Any append by the
// "leader" is immediately visible to all followers — just like committed entries
// in Raft that have been replicated to a quorum.

type Log struct {
	mu      sync.RWMutex
	entries []Command
}

// Append adds a command to the log and returns its index.
func (l *Log) Append(op, key, value string) Command {
	l.mu.Lock()
	defer l.mu.Unlock()
	idx := len(l.entries) + 1
	cmd := Command{Index: idx, Op: op, Key: key, Value: value}
	l.entries = append(l.entries, cmd)
	return cmd
}

// Since returns all entries with index > afterIndex.
func (l *Log) Since(afterIndex int) []Command {
	l.mu.RLock()
	defer l.mu.RUnlock()
	var out []Command
	for _, e := range l.entries {
		if e.Index > afterIndex {
			out = append(out, e)
		}
	}
	return out
}

// =============================================================================
// PART 3 – The Replica
// =============================================================================
// Each replica owns a state machine and continuously polls the shared log for
// new entries to apply. In production you'd push entries via RPC, but polling
// illustrates the idea more clearly.

type Replica struct {
	id string
	sm *KVStateMachine
}

func NewReplica(id string) *Replica {
	return &Replica{id: id, sm: NewKVStateMachine()}
}

// CatchUp applies all log entries the replica hasn't seen yet.
func (r *Replica) CatchUp(log *Log) int {
	pending := log.Since(r.sm.AppliedIndex())
	for _, cmd := range pending {
		r.sm.Apply(cmd)
	}
	return len(pending)
}

// Get is the read path — served entirely from local state.
func (r *Replica) Get(key string) (string, bool) {
	return r.sm.Get(key)
}

// =============================================================================
// PART 4 – The Cluster (puts it all together)
// =============================================================================

type Cluster struct {
	log      *Log
	replicas []*Replica
}

func NewCluster(replicaIDs ...string) *Cluster {
	c := &Cluster{log: &Log{}}
	for _, id := range replicaIDs {
		c.replicas = append(c.replicas, NewReplica(id))
	}
	return c
}

// Write is handled by the "leader": append to log, then propagate to all replicas.
func (c *Cluster) Write(op, key, value string) {
	c.log.Append(op, key, value)
	c.propagate()
}

// propagate drives every replica to catch up with the current log tip.
func (c *Cluster) propagate() {
	for _, r := range c.replicas {
		r.CatchUp(c.log)
	}
}

// Read queries a specific replica by index (0-based).
func (c *Cluster) Read(replicaIdx int, key string) (string, bool) {
	return c.replicas[replicaIdx].Get(key)
}

// Status prints the state of every replica.
func (c *Cluster) Status(keys ...string) {
	fmt.Println(strings.Repeat("─", 60))
	for _, r := range c.replicas {
		fmt.Printf("Replica %-6s  applied=%d  ", r.id, r.sm.AppliedIndex())
		for _, k := range keys {
			v, ok := r.Get(k)
			if ok {
				fmt.Printf("  %s=%q", k, v)
			} else {
				fmt.Printf("  %s=<nil>", k)
			}
		}
		fmt.Println()
	}
	fmt.Println(strings.Repeat("─", 60))
}

// =============================================================================
// PART 5 – Demo
// =============================================================================

func main() {
	fmt.Println("╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║        State Machine Replication Demo (Go)               ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")

	// -----------------------------------------------------------------
	// Step 1: Create a 3-replica cluster.
	// -----------------------------------------------------------------
	cluster := NewCluster("R1", "R2", "R3")

	fmt.Println("\n[1] Initial state — all replicas empty")
	cluster.Status("user", "lang")

	// -----------------------------------------------------------------
	// Step 2: Write some keys.
	// Every write goes through the shared log so ALL replicas apply it.
	// -----------------------------------------------------------------
	fmt.Println("\n[2] SET user=alice, SET lang=Go")
	cluster.Write("SET", "user", "alice")
	cluster.Write("SET", "lang", "Go")
	cluster.Status("user", "lang")

	// -----------------------------------------------------------------
	// Step 3: Read from different replicas — they should all agree.
	// -----------------------------------------------------------------
	fmt.Println("\n[3] Reading 'user' from each replica:")
	for i, id := range []string{"R1", "R2", "R3"} {
		v, _ := cluster.Read(i, "user")
		fmt.Printf("    %-4s → %q\n", id, v)
	}

	// -----------------------------------------------------------------
	// Step 4: Overwrite a key.
	// -----------------------------------------------------------------
	fmt.Println("\n[4] SET user=bob  (overwrite)")
	cluster.Write("SET", "user", "bob")
	cluster.Status("user", "lang")

	// -----------------------------------------------------------------
	// Step 5: Delete a key.
	// -----------------------------------------------------------------
	fmt.Println("\n[5] DEL lang")
	cluster.Write("DEL", "lang", "")
	cluster.Status("user", "lang")

	// -----------------------------------------------------------------
	// Step 6: Simulate a "lagging" replica.
	// We add new log entries but hold off propagating to R3 to show
	// that it falls behind, then catches up when CatchUp() is called.
	// -----------------------------------------------------------------
	fmt.Println("\n[6] Simulate R3 lagging behind:")
	cluster.log.Append("SET", "city", "London")
	cluster.log.Append("SET", "city", "Paris")
	// Only R1 and R2 catch up manually.
	cluster.replicas[0].CatchUp(cluster.log)
	cluster.replicas[1].CatchUp(cluster.log)
	fmt.Println("  R1 and R2 applied new entries, R3 has not yet:")
	cluster.Status("city")

	fmt.Println("\n[7] R3 catches up (replays the full log):")
	cluster.replicas[2].CatchUp(cluster.log)
	cluster.Status("city")

	// -----------------------------------------------------------------
	// Key takeaways
	// -----------------------------------------------------------------
	fmt.Println(`
KEY TAKEAWAYS
─────────────
1. Determinism   — Apply() must be pure: same inputs → same output, always.
2. Ordering      — All replicas apply commands in the SAME order (log index).
3. Log = truth   — The log is the single source of order; state is derived from it.
4. Reads         — Can be served locally once a replica is sufficiently caught up.
5. Fault-tolerance — If a replica crashes, it replays the log on restart.

In production, replace the in-memory Log with Raft/etcd/Zookeeper to get
true distributed consensus and automatic leader election.
`)
}
