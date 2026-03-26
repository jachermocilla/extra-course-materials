package main

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/pixperk/plethora/merkle"
	"github.com/pixperk/plethora/node"
	"github.com/pixperk/plethora/ring"
	"github.com/pixperk/plethora/server"
	"github.com/pixperk/plethora/types"
)

func main() {
	const numNodes = 10
	const Q = 20
	const N = 3
	const R = 2
	const W = 2
	const tFail = 10 * time.Second

	nodes := make([]*node.Node, numNodes)
	listeners := make([]net.Listener, numNodes)
	for i := range numNodes {
		lis, err := net.Listen("tcp", "localhost:0")
		if err != nil {
			log.Fatal(err)
		}
		listeners[i] = lis
		nodes[i] = node.NewNode(fmt.Sprintf("node-%d", i+1), lis.Addr().String())
	}

	r, err := ring.NewRing(Q, N, R, W, nodes)
	if err != nil {
		log.Fatal(err)
	}

	// each node only knows 2 seeds (next 2 in ring), gossip discovers the rest
	servers := make([]*server.Server, numNodes)
	for i, n := range nodes {
		seeds := []*node.Node{
			nodes[(i+1)%numNodes],
			nodes[(i+2)%numNodes],
		}
		srv := server.NewServer(n, seeds, r.ReplicaPeers(n.NodeID), tFail)
		servers[i] = srv
		go srv.Start(listeners[i])
		fmt.Printf("[BOOT] %s listening on %s\n", n.NodeID, n.Addr)
	}

	// use first server's gossip view for the ring's sloppy quorum decisions
	r.IsAlive = servers[0].NodeIsAlive

	fmt.Printf("\n[RING] Q=%d N=%d R=%d W=%d nodes=%d\n\n", Q, N, R, W, numNodes)

	for i, p := range r.Partitions {
		fmt.Printf("[PARTITION] %2d -> %s\n", i, p.Token.NodeID)
	}
	fmt.Println()

	// ---- gossip convergence demo ----
	fmt.Println("--- gossip protocol (each node starts with 2 seeds) ---")
	fmt.Println()
	for tick := 1; tick <= 5; tick++ {
		time.Sleep(1 * time.Second)
		converged := 0
		for i, srv := range servers {
			count := len(srv.GossipMembers())
			if tick == 1 || tick == 3 || tick == 5 {
				fmt.Printf("  %s sees %d/%d members\n", nodes[i].NodeID, count, numNodes)
			}
			if count == numNodes {
				converged++
			}
		}
		if tick == 1 || tick == 3 || tick == 5 {
			fmt.Printf("[GOSSIP] after %ds: %d/%d nodes converged\n\n", tick, converged, numNodes)
		}
		if converged == numNodes {
			fmt.Printf("[GOSSIP] fully converged after %ds!\n\n", tick)
			break
		}
	}

	// ---- put/get demo ----
	keys := []string{"user:alice", "user:bob", "user:charlie", "order:1001", "order:1002"}
	vals := []string{"Alice Smith", "Bob Jones", "Charlie Brown", "Widget x3", "Gadget x1"}

	for i, key := range keys {
		owner := r.Lookup(key)
		plist := r.PreferenceList(types.Key(key))
		replicaIDs := make([]string, len(plist))
		for j, t := range plist {
			replicaIDs[j] = t.Node.NodeID
		}

		fmt.Printf("[PUT] key=%q val=%q  coordinator=%s  replicas=%v\n", key, vals[i], owner.NodeID, replicaIDs)
		if err := r.Put(types.Key(key), vals[i], nil); err != nil {
			fmt.Printf("[PUT] ERROR: %v\n", err)
		}
	}
	fmt.Println()

	for _, key := range keys {
		result, err := r.Get(types.Key(key))
		if err != nil {
			fmt.Printf("[GET] key=%q ERROR: %v\n", key, err)
			continue
		}
		for _, v := range result {
			fmt.Printf("[GET] key=%q val=%q clock=%v\n", key, v.Data, v.Clock)
		}
	}
	fmt.Println()

	fmt.Println("[UPDATE] read-modify-write on user:alice")
	result, _ := r.Get(types.Key("user:alice"))
	ctx := result[0].Clock
	fmt.Printf("[UPDATE] read clock=%v\n", ctx)

	if err := r.Put(types.Key("user:alice"), "Alice Updated", ctx); err != nil {
		fmt.Printf("[UPDATE] ERROR: %v\n", err)
	}

	result, _ = r.Get(types.Key("user:alice"))
	for _, v := range result {
		fmt.Printf("[UPDATE] key=%q val=%q clock=%v\n", "user:alice", v.Data, v.Clock)
	}
	fmt.Println()

	result, err = r.Get(types.Key("nonexistent"))
	fmt.Printf("[GET] key=%q vals=%v err=%v\n", "nonexistent", result, err)
	fmt.Println()

	// ---- merkle tree comparison demo ----
	fmt.Println("--- merkle tree comparison ---")
	fmt.Println()

	// pick two replicas of the same key and compare their merkle trees
	demoKey := types.Key("user:alice")
	plist := r.PreferenceList(demoKey)
	replica1 := plist[0].Node
	replica2 := plist[1].Node

	fmt.Printf("[MERKLE] replicas for %q: %s, %s\n", string(demoKey), replica1.NodeID, replica2.NodeID)

	tree1 := replica1.MerkleTree()
	tree2 := replica2.MerkleTree()

	fmt.Printf("[MERKLE] %s: %d keys, root=%x\n", replica1.NodeID, len(replica1.Storage.KeyHashes()), tree1.Hash)
	fmt.Printf("[MERKLE] %s: %d keys, root=%x\n", replica2.NodeID, len(replica2.Storage.KeyHashes()), tree2.Hash)

	diffKeys := merkle.Diff(tree1, tree2)
	if len(diffKeys) == 0 {
		fmt.Println("[MERKLE] trees are identical -- replicas in sync!")
	} else {
		fmt.Printf("[MERKLE] %d divergent keys: %v\n", len(diffKeys), diffKeys)
	}
}
