package lock

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/go-zookeeper/zk"
)

const lockDir = "/locks"

// ZKLock holds the state for a single lock acquisition
type ZKLock struct {
	conn     *zk.Conn
	lockPath string // full path of our ephemeral sequential node
	prefix   string // e.g. /locks/balance-lock-
}

// Connect establishes a ZooKeeper connection and ensures /locks exists
func Connect(servers []string) (*zk.Conn, error) {
	conn, _, err := zk.Connect(servers, 10*time.Second,
		zk.WithLogInfo(false))
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}

	// ensure lock directory exists
	exists, _, err := conn.Exists(lockDir)
	if err != nil {
		return nil, fmt.Errorf("exists: %w", err)
	}
	if !exists {
		_, err = conn.Create(lockDir, []byte{},
			0, zk.WorldACL(zk.PermAll))
		if err != nil && err != zk.ErrNodeExists {
			return nil, fmt.Errorf("create lockdir: %w", err)
		}
	}
	return conn, nil
}

// NewZKLock creates a lock instance for a named resource
func NewZKLock(conn *zk.Conn, resource string) *ZKLock {
	return &ZKLock{
		conn:   conn,
		prefix: lockDir + "/" + resource + "-",
	}
}

// Lock acquires the distributed lock, blocking until acquired.
//
// Algorithm (Ricart-Agrawala style via ZK sequential nodes):
//  1. Create ephemeral sequential node under /locks/
//  2. Get all children, sort them
//  3. If our node is lowest  → we hold the lock, return
//  4. Find the predecessor node (next lower sequence number)
//  5. Watch the predecessor and block until it is deleted
//  6. Go to step 2
func (l *ZKLock) Lock() error {
	// Step 1: create ephemeral sequential node
	path, err := l.conn.CreateProtectedEphemeralSequential(
		l.prefix, []byte{}, zk.WorldACL(zk.PermAll))
	if err != nil {
		return fmt.Errorf("create node: %w", err)
	}
	l.lockPath = path

	for {
		// Step 2: get all children of lock directory
		children, _, err := l.conn.Children(lockDir)
		if err != nil {
			return fmt.Errorf("get children: %w", err)
		}
		sort.Strings(children)

		// extract just our node name from the full path
		myNode := l.lockPath[strings.LastIndex(l.lockPath, "/")+1:]

		// Step 3: check if we are the lowest sequence number
		if len(children) == 0 || children[0] == myNode {
			return nil // we hold the lock
		}

		// Step 4: find our immediate predecessor
		predecessor := ""
		for i, child := range children {
			if child == myNode && i > 0 {
				predecessor = lockDir + "/" + children[i-1]
				break
			}
		}

		if predecessor == "" {
			// we became the lowest between the Children call and here
			continue
		}

		// Step 5: watch predecessor — block until it is deleted
		exists, _, watch, err := l.conn.ExistsW(predecessor)
		if err != nil {
			return fmt.Errorf("watch predecessor: %w", err)
		}
		if exists {
			<-watch // block until ZooKeeper notifies us
		}
		// Step 6: loop back and re-check
	}
}

// Unlock releases the lock by deleting our ephemeral sequential node
func (l *ZKLock) Unlock() error {
	if l.lockPath == "" {
		return nil
	}
	err := l.conn.Delete(l.lockPath, -1)
	l.lockPath = ""
	return err
}
