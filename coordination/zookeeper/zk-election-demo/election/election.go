package election

import (
    "fmt"
    "sort"
    "strings"
    "time"

    "github.com/go-zookeeper/zk"
)

const electionDir = "/election"

// Candidate represents a process participating in the election
type Candidate struct {
    conn      *zk.Conn
    id        int
    nodePath  string // full path of our ephemeral sequential node
    leaderID  int    // current leader's candidate number
    onElected func() // callback when we become leader
    onDemoted func() // callback when we lose leadership
}

// Connect establishes a ZooKeeper connection and ensures /election exists
func Connect(servers []string) (*zk.Conn, error) {
    conn, _, err := zk.Connect(servers, 10*time.Second,
        zk.WithLogInfo(false))
    if err != nil {
        return nil, fmt.Errorf("connect: %w", err)
    }

    exists, _, err := conn.Exists(electionDir)
    if err != nil {
        return nil, fmt.Errorf("exists: %w", err)
    }
    if !exists {
        _, err = conn.Create(electionDir, []byte{},
            0, zk.WorldACL(zk.PermAll))
        if err != nil && err != zk.ErrNodeExists {
            return nil, fmt.Errorf("create election dir: %w", err)
        }
    }
    return conn, nil
}

// NewCandidate creates an election participant
func NewCandidate(conn *zk.Conn, id int,
    onElected func(), onDemoted func()) *Candidate {
    return &Candidate{
        conn:      conn,
        id:        id,
        onElected: onElected,
        onDemoted: onDemoted,
    }
}

// Enter registers this process as a candidate and begins watching
func (c *Candidate) Enter() error {
    // create ephemeral sequential node
    path, err := c.conn.CreateProtectedEphemeralSequential(
        electionDir+"/candidate-",
        []byte(fmt.Sprintf("process-%d", c.id)),
        zk.WorldACL(zk.PermAll))
    if err != nil {
        return fmt.Errorf("create candidate node: %w", err)
    }
    c.nodePath = path
    fmt.Printf("  [P%d] registered as candidate: %s\n",
        c.id, c.nodePath[strings.LastIndex(c.nodePath, "/")+1:])

    // start the election watch loop
    go c.watchLoop()
    return nil
}

// watchLoop continuously checks leadership and watches the predecessor
func (c *Candidate) watchLoop() {
    for {
        children, _, err := c.conn.Children(electionDir)
        if err != nil {
            fmt.Printf("  [P%d] watch error: %v\n", c.id, err)
            return
        }
        sort.Strings(children)

        myNode := c.nodePath[strings.LastIndex(c.nodePath, "/")+1:]

        if children[0] == myNode {
            // we are the leader
            leaderSeq := sequenceNumber(children[0])
            if c.leaderID != leaderSeq {
                c.leaderID = leaderSeq
                fmt.Printf("  [P%d] *** ELECTED AS LEADER (seq=%d) ***\n",
                    c.id, leaderSeq)
                if c.onElected != nil {
                    c.onElected()
                }
            }
            // watch for any change in children
            // (detects if we get a new predecessor — rare but possible)
            _, _, watch, err := c.conn.ChildrenW(electionDir)
            if err != nil {
                return
            }
            <-watch
        } else {
            // find and watch our predecessor
            predecessor := ""
            for i, child := range children {
                if child == myNode && i > 0 {
                    predecessor = electionDir + "/" + children[i-1]
                    break
                }
            }
            if predecessor == "" {
                continue
            }

            leaderSeq := sequenceNumber(children[0])
            if c.leaderID != leaderSeq {
                c.leaderID = leaderSeq
                fmt.Printf("  [P%d] current leader is seq=%d, watching predecessor\n",
                    c.id, leaderSeq)
            }

            // watch predecessor
            exists, _, watch, err := c.conn.ExistsW(predecessor)
            if err != nil {
                return
            }
            if exists {
                <-watch // block until predecessor deleted or changed
            }
        }
    }
}

// Resign removes our candidate node, triggering a new election
func (c *Candidate) Resign() error {
    if c.nodePath == "" {
        return nil
    }
    fmt.Printf("  [P%d] resigning from election\n", c.id)
    err := c.conn.Delete(c.nodePath, -1)
    c.nodePath = ""
    return err
}

// IsLeader returns true if this candidate is currently the leader
func (c *Candidate) IsLeader() (bool, error) {
    children, _, err := c.conn.Children(electionDir)
    if err != nil {
        return false, err
    }
    sort.Strings(children)
    if len(children) == 0 {
        return false, nil
    }
    myNode := c.nodePath[strings.LastIndex(c.nodePath, "/")+1:]
    return children[0] == myNode, nil
}

func sequenceNumber(node string) int {
    // node format: _c_<guid>-candidate-0000000001
    // sequence is always the last 10 digits
    if len(node) < 10 {
        return 0
    }
    n := 0
    fmt.Sscanf(node[len(node)-10:], "%d", &n)
    return n
}
