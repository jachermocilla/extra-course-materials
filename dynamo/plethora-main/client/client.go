package client

import (
	"context"
	"maps"

	pb "github.com/pixperk/plethora/proto"
	"github.com/pixperk/plethora/types"
	"github.com/pixperk/plethora/vclock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func RemotePut(addr string, key types.Key, val types.Value) error {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer conn.Close()

	c := pb.NewKVClient(conn)
	_, err = c.Put(context.Background(), &pb.PutRequest{
		Key:   string(key),
		Value: toProtoValue(val),
	})
	return err
}

func RemoteGet(addr string, key types.Key) ([]types.Value, bool, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, false, err
	}
	defer conn.Close()

	c := pb.NewKVClient(conn)
	resp, err := c.Get(context.Background(), &pb.GetRequest{Key: string(key)})
	if err != nil {
		return nil, false, err
	}

	vals := make([]types.Value, len(resp.Values))
	for i, v := range resp.Values {
		vals[i] = fromProtoValue(v)
	}
	return vals, resp.Found, nil
}

func RemoteHintedPut(addr string, key types.Key, val types.Value, targetNodeId string) error {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer conn.Close()

	c := pb.NewKVClient(conn)
	_, err = c.HintedPut(context.Background(), &pb.HintedPutRequest{
		Key:          string(key),
		Value:        toProtoValue(val),
		TargetNodeId: targetNodeId,
	})
	return err
}

// RemoteGetKeyHashes fetches all key-hash pairs from a peer node's storage.
func RemoteGetKeyHashes(addr string) ([]*pb.KeyHashEntry, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	c := pb.NewKVClient(conn)
	resp, err := c.GetKeyHashes(context.Background(), &pb.GetKeyHashesRequest{})
	if err != nil {
		return nil, err
	}
	return resp.Entries, nil
}

// RemoteSyncKeys fetches the actual values for a set of keys from a peer node.
func RemoteSyncKeys(addr string, keys []string) (map[string][]types.Value, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	c := pb.NewKVClient(conn)
	resp, err := c.SyncKeys(context.Background(), &pb.SyncKeysRequest{Keys: keys})
	if err != nil {
		return nil, err
	}

	result := make(map[string][]types.Value, len(resp.Data))
	for k, gr := range resp.Data {
		vals := make([]types.Value, len(gr.Values))
		for i, v := range gr.Values {
			vals[i] = fromProtoValue(v)
		}
		result[k] = vals
	}
	return result, nil
}

// RemoteGossip sends our membership list to a peer and returns theirs.
func RemoteGossip(addr string, members []*pb.GossipMember) ([]*pb.GossipMember, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	c := pb.NewKVClient(conn)
	resp, err := c.Gossip(context.Background(), &pb.GossipRequest{Members: members})
	if err != nil {
		return nil, err
	}
	return resp.Members, nil
}

func toProtoValue(v types.Value) *pb.Value {
	return &pb.Value{
		Data:  v.Data,
		Clock: &pb.VectorClock{Entries: v.Clock},
	}
}

func fromProtoValue(v *pb.Value) types.Value {
	clock := vclock.NewVClock()
	if v.Clock != nil {
		maps.Copy(clock, v.Clock.Entries)
	}
	return types.Value{Data: v.Data, Clock: clock}
}
