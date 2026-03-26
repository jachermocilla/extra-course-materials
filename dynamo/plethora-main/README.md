# plethora

a distributed key-value store built from scratch in go, following the amazon dynamo paper (decandia et al., 2007). uses strategy 3 consistent hashing with fixed equal-sized partitions, vector clocks for conflict detection, sloppy quorum with hinted handoff for high availability, merkle tree anti-entropy for replica synchronization, and grpc for inter-node communication.

built bottom-up, one layer at a time.

## architecture

```mermaid
graph TB
    client([client request])
    client --> ring

    subgraph ring["ring (coordinator)"]
        direction TB
        hash[md5 hash + partition lookup]
        pref[preference list: N distinct nodes]
        quorum[sloppy quorum: R reads, W writes]
        dedup[dedup via vector clocks]
        hash --> pref --> quorum --> dedup
    end

    subgraph cluster["node cluster (grpc)"]
        direction LR
        subgraph n1[node 1]
            s1[server] --> st1[storage]
            s1 --> h1[hint store]
            s1 --> hb1[heartbeat]
            st1 --> mt1[merkle tree]
        end
        subgraph n2[node 2]
            s2[server] --> st2[storage]
            s2 --> h2[hint store]
            s2 --> hb2[heartbeat]
            st2 --> mt2[merkle tree]
        end
        subgraph n3["node 3 (down)"]
            s3[server] --> st3[storage]
        end
    end

    ring -->|grpc put/get| n1
    ring -->|grpc put/get| n2
    ring -.->|"fail: node down"| n3
    ring -->|"hinted put (target=node 3)"| n1

    n1 -.->|"handoff when node 3 recovers"| n3
    n1 <-.->|"anti-entropy sync (merkle diff)"| n2
```

## layers

started at the bottom and worked up. each layer was built, tested, and verified before moving on.

### layer 1: storage

the foundation. a thread-safe in-memory key-value store behind a read-write mutex. the interesting part is that it stores multiple versions per key as siblings. when a put comes in, storage uses vector clock logic to decide what to do:

- if the new value's clock descends from an existing value's clock, the old one is an ancestor. drop it.
- if an existing value's clock descends from the new one, the new write is stale. ignore it.
- if neither descends from the other, they conflict. keep both as siblings.

this means storage never loses data silently. conflicts are preserved and bubbled up to the caller.

### layer 2: node

wraps storage with identity. each node has an id, a network address, its own storage instance, and a hint store for hinted handoff. three write paths:

- `put(key, val, ctx)` is the standalone coordinator path. takes a raw string value and an optional vector clock context from a previous read. copies the clock, increments its own node id entry, builds a `types.Value`, and writes to storage. used for testing only, the ring has its own coordinator logic.
- `store(key, value)` is the replica path. accepts a fully built value with clock already set. just writes it to storage, no clock mutation.
- `storeHint(targetNodeID, key, value)` stores a value in the hint store (not in main storage) tagged with the node it was originally meant for. the hint store is a separate `map[string][]HintedItem` protected by its own mutex, so hinted data never shows up in gets.

`drainHints(targetNodeID)` atomically removes and returns all hints for a given node, used during handoff when the target recovers.

this separation matters because the ring coordinator builds the clock once, then replicates the same value to N nodes. replicas shouldn't touch the clock again. hinted data lives outside main storage entirely.

### layer 3: vector clocks

a `map[string]uint64` where keys are node ids and values are monotonic counters. tracks causal history across the distributed system.

```mermaid
graph LR
    subgraph "write 1 (node a)"
        vc1["{a: 1}"]
    end
    subgraph "write 2 (node a)"
        vc2["{a: 2}"]
    end
    subgraph "write 3 (node b, no context)"
        vc3["{b: 1}"]
    end

    vc1 -->|descends| vc2
    vc1 -.-|conflicts| vc3
    vc2 -.-|conflicts| vc3
```

five operations:

- **increment(nodeID)** bumps the counter for a node. called by the coordinator on every write.
- **descends(other)** returns true if every entry in other is <= the corresponding entry in this clock. means "i have seen everything you have seen". this is how we know if one value is a causal successor of another.
- **conflicts(other)** returns true when neither clock descends from the other. means two independent writes happened and we have a fork.
- **merge(other)** takes the pointwise max of both clocks. used to combine causal histories.
- **copy()** deep clones the map. critical because go maps are references, and without this you get aliasing bugs where mutating one clock silently corrupts another.

### layer 4: consistent hashing (strategy 3)

dynamo paper describes three strategies. we use strategy 3: Q equal-sized partitions, fixed forever. only ownership changes when nodes join or leave.

```mermaid
graph LR
    subgraph "ring: Q=6, S=3 nodes"
        direction LR
        p0["partition 0 -> node 1"]
        p1["partition 1 -> node 2"]
        p2["partition 2 -> node 3"]
        p3["partition 3 -> node 1"]
        p4["partition 4 -> node 2"]
        p5["partition 5 -> node 3"]
    end

    key["key 'user:alice'"] -->|"md5(key) % Q = 3"| p3
    p3 -->|"owner"| result["node 1"]
```

how it works:

- Q partitions are created at ring init and never change.
- partitions are assigned round-robin: partition i goes to node `i % S`.
- each node owns exactly `Q/S` partitions (Q must be divisible by S).
- to look up a key: `md5(key) % Q` gives the partition index, the partition's token gives the owner node id, the node map gives the node pointer in O(1).
- when a node joins or leaves, partitions are reassigned with the same round-robin. no data moves (yet), just ownership pointers change.

### layer 5: replication and sloppy quorum

a single owner isn't enough. we replicate each key to N nodes using a preference list, with sloppy quorum for availability when nodes go down.

```mermaid
graph LR
    subgraph "preference list for key k (N=3)"
        direction LR
        p5["partition 5 (owner)"] --> nodeA["node 3"]
        p0["partition 0"] --> nodeB["node 1"]
        p1["partition 1"] --> nodeC["node 2 (down)"]
    end

    subgraph "sloppy quorum fallback"
        nodeD["node 4 (stand-in)"]
    end

    put["put(k, v)"] --> nodeA
    put --> nodeB
    put -.->|fail| nodeC
    put -->|"hinted put (target=node 2)"| nodeD
```

the preference list is health-aware and does all the sloppy quorum logic in one place using a two-pass algorithm:

1. **first pass**: walk the ring clockwise from the key's partition and find the N ideal preferred nodes, ignoring health. these are the nodes that *should* own replicas.
2. **second pass**: walk the ring again. for each node encountered:
   - preferred + alive: add as a regular target.
   - preferred + dead: skip, queue it as needing coverage.
   - non-preferred + alive + dead queue non-empty: add as a stand-in covering the first uncovered dead preferred node.

the result is a `[]Target` where each target is either `{Node, HintFor:""}` (preferred node, regular put) or `{Node, HintFor:"node-3"}` (stand-in, hinted put tagged with who it's covering for). put just iterates the list.

`ring.IsAlive` is an injectable callback (`func(nodeID string) bool`) that defaults to all-alive if not set. in the running system, it's wired to a server's heartbeat-tracked alive map.

**put flow:**
1. coordinator (first target) copies the client's context clock, increments its own entry, builds the value.
2. iterates all targets: `HintFor==""` means `RemotePut`, `HintFor!=""` means `RemoteHintedPut` tagged with the dead preferred node.
3. counts acks. if acks >= W, success. otherwise quorum failure.

**get flow:**
1. reads from all N targets in the preference list via grpc.
2. counts responses. if responses < R, quorum failure.
3. deduplicates: for every pair of values, if one's clock descends from the other, drop the ancestor. if clocks are identical, keep only one copy. what remains are causally distinct versions (siblings).

**quorum guarantee:** R + W > N ensures at least one node in the read set has the latest write. sloppy quorum relaxes "which W nodes" to include stand-ins, trading consistency for availability (the dynamo tradeoff). the demo uses N=3, R=2, W=2.

### layer 6: networking (grpc)

everything above this was in-process function calls. grpc makes it real.

```
proto/kv.proto defines:
    service KV {
        rpc Put(PutRequest) returns (PutResponse)
        rpc Get(GetRequest) returns (GetResponse)
        rpc HintedPut(HintedPutRequest) returns (PutResponse)
        rpc GetKeyHashes(GetKeyHashesRequest) returns (GetKeyHashesResponse)
        rpc SyncKeys(SyncKeysRequest) returns (SyncKeysResponse)
        rpc Gossip(GossipRequest) returns (GossipResponse)
    }
```

- **server**: each node runs a grpc server. Put calls `node.Store()`, Get calls `node.Get()`, HintedPut calls `node.StoreHint()`. GetKeyHashes returns the node's key-hash pairs for merkle tree comparison. SyncKeys returns actual values for a set of keys. Gossip exchanges membership lists for failure detection and node discovery. each server holds a gossip `MemberList` and a replica peers list for anti-entropy.
- **client**: `RemotePut`, `RemoteGet`, `RemoteHintedPut`, `RemoteGetKeyHashes`, `RemoteSyncKeys`, and `RemoteGossip`. dial the node's address, make the rpc, convert proto back to types.
- **ring**: `ring.Put` and `ring.Get` use grpc client calls. on put failure, the ring falls back to `RemoteHintedPut` on a stand-in node. `ReplicaPeers` computes which nodes share key ranges for anti-entropy.

the ring tests start real grpc servers on random OS-assigned ports and exercise the full put/get path including sloppy quorum.

### layer 7: hinted handoff

when a preferred node goes down, the stand-in holds hints temporarily. the handoff mechanism gets them back to the right place.

```mermaid
sequenceDiagram
    participant R as ring coordinator
    participant S as stand-in node
    participant T as target node (down)

    R->>S: HintedPut(key, val, target=T)
    S->>S: storeHint(T, key, val)
    note over T: node recovers
    note over S: gossip detects T alive
    note over S: handoff ticker fires
    S->>S: drainHints(T)
    S->>T: RemotePut(key, val)
```

- each server runs a background `runHandoff` goroutine on a 5-second ticker.
- on each tick, it asks the gossip `MemberList` for all alive peers. for every alive node, it drains all pending hints and forwards them via `RemotePut`.
- failure detection comes from the gossip protocol: if a node's heartbeat counter hasn't increased within `tFail`, it's considered down.
- hints live in the node's `HintStore`, a `map[string][]HintedItem` separate from main storage. they never show up in gets. `drainHints` atomically removes and returns all items for a target, so each hint is forwarded exactly once.

### layer 8: merkle trees (anti-entropy sync)

hinted handoff handles temporary failures, but what about replicas that silently drift apart? missed writes, partial failures, bugs. anti-entropy catches everything else.

```mermaid
sequenceDiagram
    participant A as node 1 (anti-entropy tick)
    participant B as node 2 (replica peer)

    note over A: 30s ticker fires
    A->>B: GetKeyHashes()
    B->>B: storage.KeyHashes()
    B-->>A: []{key, md5(key+values)}

    note over A: build both merkle trees locally
    A->>A: localTree = node.MerkleTree()
    A->>A: peerTree = merkle.Build(peerHashes)
    A->>A: diffKeys = merkle.Diff(local, peer)

    alt roots match
        note over A: in sync, done
    else roots differ
        A->>B: SyncKeys(diffKeys)
        B-->>A: map[key][]Value
        A->>A: store each value (vclock handles conflicts)
    end
```

the merkle tree is a binary hash tree built bottom-up from sorted key-hash pairs:

1. each leaf is a key hashed with all its values: `md5(key + value1 + value2 + ...)`.
2. entries are sorted by key for determinism, padded to the next power of 2.
3. parent hash = `md5(left.hash + right.hash)`, merged bottom-up to a single root.
4. if two trees have the same root hash, all data is identical. if roots differ, walk down to find exactly which leaves diverged in O(log n) comparisons instead of O(n).

the node caches its merkle tree with a dirty flag. every write marks it dirty; `MerkleTree()` rebuilds lazily only when someone asks for it after data changed.

anti-entropy only syncs with **replica peers**, nodes that share at least one key range. `ring.ReplicaPeers(nodeID)` walks all partitions and collects nodes that appear in the same preference lists. each server round-robins through its replica peers, one per tick, so every peer gets checked periodically.

the merkle tree never goes over the wire. only flat key-hash pairs and actual values do. the tree is a local optimization to narrow down divergence efficiently.

### layer 9: gossip protocol (membership and failure detection)

gossip replaces the need for a central membership registry and explicit heartbeat streams. every node maintains a local `MemberList` and periodically exchanges it with a random peer.

```mermaid
sequenceDiagram
    participant A as node 1
    participant B as node 3 (random peer)

    note over A: 1s ticker fires
    A->>A: tick own heartbeat counter
    A->>B: Gossip(my member list)
    B->>B: merge(A's list)
    B-->>A: my member list
    A->>A: merge(B's list)
```

**member entry**: `{nodeID, addr, heartbeat, timestamp}`. heartbeat is a monotonic counter only incremented by the owning node. timestamp is the local wall time when we last saw this node's heartbeat increase.

**merge rule**: for each entry, keep the higher heartbeat counter. if the counter went up, reset timestamp to now. new nodes discovered transitively through gossip are added automatically.

**failure detection**: if `now - timestamp > tFail`, the node is considered down. the ring's `IsAlive` callback reads from gossip state, so sloppy quorum automatically routes around dead nodes.

**seed nodes**: each node starts with a small set of known seeds (not the full cluster). gossip transitively discovers the rest. seeds ensure convergence even after network partitions heal.

**what gossip replaced**:
- the bidirectional heartbeat stream (gossip subsumes failure detection)
- the `allNodes` parameter in server creation (nodes discover peers through gossip)
- the manual `alive` map with mutex (gossip `MemberList` is self-contained and thread-safe)

## project structure

```
plethora/
    types/       core types: Key, Value (with vector clock)
    vclock/      vector clock implementation
    storage/     thread-safe versioned kv store, KeyHashes for merkle trees
    node/        node with identity, storage, hint store, cached merkle tree
    merkle/      merkle tree: Build, Diff, collectKeys
    ring/        consistent hash ring, sloppy quorum, replication, dedup, ReplicaPeers
    gossip/      gossip protocol: MemberList, heartbeat-based failure detection
    server/      grpc server: put, get, hinted put, gossip, handoff, anti-entropy
    client/      grpc client helpers (RemotePut, RemoteGet, RemoteHintedPut, RemoteGetKeyHashes, RemoteSyncKeys, RemoteGossip)
    proto/       protobuf definition and generated code
    cmd/         demo: boots 10 nodes, puts and gets over grpc
```

## running it

```bash
go test ./...         # run all tests (60 total)
go run ./cmd/         # boot 10 nodes on random ports, run demo puts and gets
```

## dynamo paper coverage

| concept | status |
|---|---|
| consistent hashing (strategy 3) | done |
| vector clocks | done |
| replication (preference list) | done |
| quorum (R, W, N) | done |
| grpc networking | done |
| sloppy quorum | done |
| hinted handoff | done |
| merkle trees (anti-entropy sync) | done |
| gossip protocol (membership + failure detection) | done |

## config

the ring takes four parameters:

- **Q** total partitions (must be divisible by number of nodes)
- **N** replication factor (how many nodes store each key)
- **R** minimum read responses for quorum
- **W** minimum write acks for quorum

the demo uses Q=20, N=3, R=2, W=2 across 10 nodes.
