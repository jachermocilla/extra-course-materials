package server

import (
	"context"
	"maps"
	"net"
	"time"

	"github.com/pixperk/plethora/client"
	"github.com/pixperk/plethora/gossip"
	"github.com/pixperk/plethora/merkle"
	"github.com/pixperk/plethora/node"
	pb "github.com/pixperk/plethora/proto"
	"github.com/pixperk/plethora/types"
	"github.com/pixperk/plethora/vclock"
	"google.golang.org/grpc"
)

type Server struct {
	pb.UnimplementedKVServer
	node *node.Node

	// anti-entropy: nodes that share key ranges with this node
	replicaPeers []*node.Node

	// gossip-based membership and failure detection
	members *gossip.MemberList

	grpcServer *grpc.Server
}

func NewServer(n *node.Node, seeds []*node.Node, replicaPeers []*node.Node, tFail time.Duration) *Server {
	m := gossip.NewMemberList(n.NodeID, n.Addr, tFail)
	for _, s := range seeds {
		if s.NodeID != n.NodeID {
			m.AddSeed(s.NodeID, s.Addr)
		}
	}
	return &Server{
		node:         n,
		members:      m,
		replicaPeers: replicaPeers,
	}
}

// NodeIsAlive reports whether a node is reachable according to gossip.
// Suitable for use as ring.IsAlive.
func (s *Server) NodeIsAlive(nodeID string) bool {
	return s.members.IsAlive(nodeID)
}

// GossipMembers returns a snapshot of this server's gossip membership view.
func (s *Server) GossipMembers() []gossip.MemberEntry {
	return s.members.Entries()
}

func (s *Server) Put(_ context.Context, req *pb.PutRequest) (*pb.PutResponse, error) {
	val := fromProtoValue(req.Value)
	s.node.Store(types.Key(req.Key), val)
	return &pb.PutResponse{}, nil
}

func (s *Server) Get(_ context.Context, req *pb.GetRequest) (*pb.GetResponse, error) {
	vals, found := s.node.Get(types.Key(req.Key))
	return &pb.GetResponse{
		Values: toProtoValues(vals),
		Found:  found,
	}, nil
}

// StoreHint stores a hinted value for a target node that is currently unreachable.
// The coordinator calls this when it fails to forward a write to a replica node, and the target node will attempt to apply the hinted writes when it recovers.
func (s *Server) HintedPut(_ context.Context, req *pb.HintedPutRequest) (*pb.PutResponse, error) {
	val := fromProtoValue(req.Value)
	s.node.StoreHint(req.TargetNodeId, types.Key(req.Key), val)
	return &pb.PutResponse{}, nil
}

// Start registers gRPC handlers, launches background goroutines, and serves.
// Blocks until Stop is called.
func (s *Server) Start(lis net.Listener) error {
	s.grpcServer = grpc.NewServer()
	pb.RegisterKVServer(s.grpcServer, s)

	go s.runGossip()
	go s.runHandoff()
	go s.runAntiEntropy()

	return s.grpcServer.Serve(lis)
}

// Stop gracefully stops the gRPC server.
func (s *Server) Stop() {
	if s.grpcServer != nil {
		s.grpcServer.Stop()
	}
}

// Gossip receives a peer's membership list, merges it, and responds with ours.
func (s *Server) Gossip(_ context.Context, req *pb.GossipRequest) (*pb.GossipResponse, error) {
	remote := make([]gossip.MemberEntry, len(req.Members))
	for i, m := range req.Members {
		remote[i] = gossip.MemberEntry{
			NodeID:    m.NodeId,
			Addr:      m.Addr,
			Heartbeat: m.Heartbeat,
		}
	}
	s.members.Merge(remote)

	entries := s.members.Entries()
	resp := make([]*pb.GossipMember, len(entries))
	for i, e := range entries {
		resp[i] = &pb.GossipMember{
			NodeId:    e.NodeID,
			Addr:      e.Addr,
			Heartbeat: e.Heartbeat,
		}
	}
	return &pb.GossipResponse{Members: resp}, nil
}

// GetKeyHashes returns the key-hash pairs from this node's storage.
// The caller builds merkle trees locally and diffs them to find divergent keys.
func (s *Server) GetKeyHashes(_ context.Context, _ *pb.GetKeyHashesRequest) (*pb.GetKeyHashesResponse, error) {
	kh := s.node.Storage.KeyHashes()
	entries := make([]*pb.KeyHashEntry, len(kh))
	for i, e := range kh {
		entries[i] = &pb.KeyHashEntry{Key: e.Key, Hash: e.Hash[:]}
	}
	return &pb.GetKeyHashesResponse{Entries: entries}, nil
}

// SyncKeys returns the actual values for the requested keys.
// Called after a merkle diff identifies which keys diverged.
func (s *Server) SyncKeys(_ context.Context, req *pb.SyncKeysRequest) (*pb.SyncKeysResponse, error) {
	data := make(map[string]*pb.GetResponse, len(req.Keys))
	for _, key := range req.Keys {
		vals, found := s.node.Get(types.Key(key))
		data[key] = &pb.GetResponse{
			Values: toProtoValues(vals),
			Found:  found,
		}
	}
	return &pb.SyncKeysResponse{Data: data}, nil
}

// runGossip periodically ticks the local heartbeat, picks a random peer,
// exchanges membership lists, and merges the response.
func (s *Server) runGossip() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		s.members.Tick()

		peer, ok := s.members.RandomPeer()
		if !ok {
			continue
		}

		// build proto member list from local state
		entries := s.members.Entries()
		protoMembers := make([]*pb.GossipMember, len(entries))
		for i, e := range entries {
			protoMembers[i] = &pb.GossipMember{
				NodeId:    e.NodeID,
				Addr:      e.Addr,
				Heartbeat: e.Heartbeat,
			}
		}

		// exchange with peer
		resp, err := client.RemoteGossip(peer.Addr, protoMembers)
		if err != nil {
			continue
		}

		// merge peer's response
		remote := make([]gossip.MemberEntry, len(resp))
		for i, m := range resp {
			remote[i] = gossip.MemberEntry{
				NodeID:    m.NodeId,
				Addr:      m.Addr,
				Heartbeat: m.Heartbeat,
			}
		}
		s.members.Merge(remote)
	}
}

// runHandoff periodically checks for any hints that need to be forwarded to their target nodes and attempts to send them.
func (s *Server) runHandoff() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		for _, m := range s.members.Alive() {
			items := s.node.DrainHints(m.NodeID)
			for _, item := range items {
				client.RemotePut(m.Addr, item.Key, item.Value)
			}
		}
	}
}

// runAntiEntropy periodically picks a replica peer, compares merkle trees,
// and syncs any divergent keys.
func (s *Server) runAntiEntropy() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	peerIdx := 0
	for range ticker.C {
		if len(s.replicaPeers) == 0 {
			continue
		}

		// round-robin through replica peers
		peer := s.replicaPeers[peerIdx%len(s.replicaPeers)]
		peerIdx++

		if !s.members.IsAlive(peer.NodeID) {
			continue
		}

		// get peer's key hashes
		entries, err := client.RemoteGetKeyHashes(peer.Addr)
		if err != nil {
			continue
		}

		// convert proto entries to merkle.KeyHash
		peerHashes := make([]merkle.KeyHash, len(entries))
		for i, e := range entries {
			var h [16]byte
			copy(h[:], e.Hash)
			peerHashes[i] = merkle.KeyHash{Key: e.Key, Hash: h}
		}

		// build both trees locally and diff
		localTree := s.node.MerkleTree()
		peerTree := merkle.Build(peerHashes)
		diffKeys := merkle.Diff(localTree, peerTree)
		if len(diffKeys) == 0 {
			continue
		}

		// fetch divergent values from peer and write to local storage
		data, err := client.RemoteSyncKeys(peer.Addr, diffKeys)
		if err != nil {
			continue
		}
		for key, vals := range data {
			for _, val := range vals {
				s.node.Store(types.Key(key), val)
			}
		}
	}
}

// --- proto <-> types conversion helpers ---

func toProtoValue(v types.Value) *pb.Value {
	return &pb.Value{
		Data:  v.Data,
		Clock: &pb.VectorClock{Entries: v.Clock},
	}
}

func toProtoValues(vals []types.Value) []*pb.Value {
	out := make([]*pb.Value, len(vals))
	for i, v := range vals {
		out[i] = toProtoValue(v)
	}
	return out
}

func fromProtoValue(v *pb.Value) types.Value {
	clock := vclock.NewVClock()
	if v.Clock != nil {
		maps.Copy(clock, v.Clock.Entries)
	}
	return types.Value{Data: v.Data, Clock: clock}
}
