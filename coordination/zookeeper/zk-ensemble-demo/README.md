# ZooKeeper Ensemble Leader Election Demo

---

## Overview

This demo shows how ZooKeeper's **own internal leader election**
works inside a 3-node ensemble. Unlike the application-level
election in the previous tutorial (where Go processes elect a
leader using znodes), this is ZooKeeper's **built-in Zab protocol**
electing a leader among the ZooKeeper servers themselves.

The Go client demonstrates:
1. Connecting to the ensemble
2. Detecting which ZooKeeper node is the current leader
3. Watching the ensemble survive a leader crash and elect a new one
4. Verifying data consistency across the ensemble after re-election

---

## How ZooKeeper Ensemble Election Works (Zab Protocol)

ZooKeeper uses the **Zab (ZooKeeper Atomic Broadcast)** protocol
for leader election among ensemble members:

```
Each server has:
  - myid      : unique server ID (1, 2, 3)
  - epoch     : election round number
  - zxid      : last transaction ID processed

Election rules:
  1. Servers vote for themselves initially
  2. Compare votes: higher epoch wins
                    same epoch → higher zxid wins
                    same zxid  → higher myid wins
  3. A server wins when it has votes from a majority (n/2 + 1)
  4. Winner becomes LEADER, others become FOLLOWERS
```

With 3 nodes, majority = 2. The ensemble can survive **1 node failure**.

```
Normal state:
  zoo1 [FOLLOWER] ──┐
  zoo2 [LEADER]   ──┼── quorum of 3, majority = 2
  zoo3 [FOLLOWER] ──┘

After zoo2 crashes:
  zoo1 [FOLLOWER → votes] ──┐
  zoo2 [CRASHED]            ├── zoo1 and zoo3 form new majority
  zoo3 [FOLLOWER → votes] ──┘
  → zoo1 or zoo3 elected as new LEADER
```

---

## Prerequisites

- Docker and Docker Compose
- Go 1.21+
- `go-zookeeper` client library

```bash
go get github.com/go-zookeeper/zk
```

---

## Project Structure

```
zk-ensemble-demo/
├── docker-compose.yml
└── main.go
```

---

## Step 1 — Start the 3-Node Ensemble

```yaml
# docker-compose.yml
services:
  zoo1:
    image: zookeeper:3.9
    hostname: zoo1
    container_name: zoo1
    ports:
      - "2181:2181"
      - "8081:8080"
    environment:
      ZOO_MY_ID: 1
      ZOO_SERVERS: server.1=zoo1:2888:3888;2181 server.2=zoo2:2888:3888;2181 server.3=zoo3:2888:3888;2181
      ZOO_4LW_COMMANDS_WHITELIST: "mntr,conf,ruok,stat,srvr,cons,dump,envi,wchs"
      ZOO_LOG_LEVEL: ERROR
      ZOOKEEPER_LOG4J_ROOT_LOGLEVEL: ERROR
    networks:
      - zk-net
    restart: on-failure

  zoo2:
    image: zookeeper:3.9
    hostname: zoo2
    container_name: zoo2
    ports:
      - "2182:2181"
      - "8082:8080"
    environment:
      ZOO_MY_ID: 2
      ZOO_SERVERS: server.1=zoo1:2888:3888;2181 server.2=zoo2:2888:3888;2181 server.3=zoo3:2888:3888;2181
      ZOO_4LW_COMMANDS_WHITELIST: "mntr,conf,ruok,stat,srvr,cons,dump,envi,wchs"
      ZOO_LOG_LEVEL: ERROR
      ZOOKEEPER_LOG4J_ROOT_LOGLEVEL: ERROR
    networks:
      - zk-net
    restart: on-failure

  zoo3:
    image: zookeeper:3.9
    hostname: zoo3
    container_name: zoo3
    ports:
      - "2183:2181"
      - "8083:8080"
    environment:
      ZOO_MY_ID: 3
      ZOO_SERVERS: server.1=zoo1:2888:3888;2181 server.2=zoo2:2888:3888;2181 server.3=zoo3:2888:3888;2181
      ZOO_4LW_COMMANDS_WHITELIST: "mntr,conf,ruok,stat,srvr,cons,dump,envi,wchs"
      ZOO_LOG_LEVEL: ERROR
      ZOOKEEPER_LOG4J_ROOT_LOGLEVEL: ERROR
    networks:
      - zk-net
    restart: on-failure

networks:
  zk-net:
    driver: bridge
```

```bash
docker compose up -d

# wait ~20 seconds for election to complete
sleep 20

# verify all three nodes are up
for port in 2181 2182 2183; do
  echo -n "zoo $port: "
  echo ruok | nc localhost $port
done

# check which is the leader
for port in 2181 2182 2183; do
  echo -n "zoo $port: "
  echo stat | nc localhost $port | grep Mode
done
```

---

## Step 2 — Go Demo

### `main.go`

```go
package main

import (
    "fmt"
    "net"
    "os/exec"
    "strings"
    "time"

    "github.com/go-zookeeper/zk"
)

// ensemble node definitions
type ZKNode struct {
    name      string
    clientAddr string  // host:port for ZK client connections
    statAddr  string   // host:port for four-letter word commands
    container string   // docker container name
}

var nodes = []ZKNode{
    {name: "zoo1", clientAddr: "localhost:2181",
        statAddr: "localhost:2181", container: "zoo1"},
    {name: "zoo2", clientAddr: "localhost:2182",
        statAddr: "localhost:2182", container: "zoo2"},
    {name: "zoo3", clientAddr: "localhost:2183",
        statAddr: "localhost:2183", container: "zoo3"},
}

// ── four-letter word helpers ──────────────────────────────────

func sendFourLetterWord(addr, cmd string) (string, error) {
    conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
    if err != nil {
        return "", err
    }
    defer conn.Close()
    conn.SetDeadline(time.Now().Add(3 * time.Second))
    fmt.Fprintf(conn, cmd)
    buf := make([]byte, 4096)
    n, _ := conn.Read(buf)
    return string(buf[:n]), nil
}

func getMode(addr string) string {
    out, err := sendFourLetterWord(addr, "stat")
    if err != nil {
        return "UNREACHABLE"
    }
    for _, line := range strings.Split(out, "\n") {
        if strings.HasPrefix(line, "Mode:") {
            return strings.TrimSpace(strings.TrimPrefix(line, "Mode:"))
        }
    }
    return "unknown"
}

func isAlive(addr string) bool {
    out, err := sendFourLetterWord(addr, "ruok")
    if err != nil {
        return false
    }
    return strings.TrimSpace(out) == "imok"
}

func getStats(addr string) map[string]string {
    out, err := sendFourLetterWord(addr, "mntr")
    stats := make(map[string]string)
    if err != nil {
        return stats
    }
    for _, line := range strings.Split(out, "\n") {
        parts := strings.SplitN(line, "\t", 2)
        if len(parts) == 2 {
            stats[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
        }
    }
    return stats
}

// ── print ensemble status ─────────────────────────────────────

func printEnsembleStatus(label string) {
    fmt.Printf("\n  %s\n", label)
    fmt.Println("  " + strings.Repeat("-", 50))
    fmt.Printf("  %-8s %-12s %-10s %-10s %s\n",
        "Node", "Status", "Mode", "ZxID", "Leader")
    fmt.Println("  " + strings.Repeat("-", 50))

    leader := ""
    for _, n := range nodes {
        alive := isAlive(n.statAddr)
        status := "UP  "
        mode := "---"
        zxid := "---"
        leaderStr := "---"

        if !alive {
            status = "DOWN"
        } else {
            mode = getMode(n.statAddr)
            stats := getStats(n.statAddr)
            if v, ok := stats["zk_zxid"]; ok {
                zxid = v
            }
            if mode == "leader" {
                leader = n.name
                leaderStr = "← LEADER"
            }
        }
        fmt.Printf("  %-8s %-12s %-10s %-10s %s\n",
            n.name, status, mode, zxid, leaderStr)
    }
    if leader != "" {
        fmt.Printf("\n  Current leader: %s\n", leader)
    }
}

// ── write test data to ensemble ───────────────────────────────

func writeTestData(conn *zk.Conn, path, value string) error {
    exists, stat, err := conn.Exists(path)
    if err != nil {
        return err
    }
    if exists {
        _, err = conn.Set(path, []byte(value), stat.Version)
    } else {
        _, err = conn.Create(path, []byte(value),
            0, zk.WorldACL(zk.PermAll))
    }
    return err
}

func readFromNode(addr, path string) string {
    conn, _, err := zk.Connect([]string{addr}, 5*time.Second,
        zk.WithLogInfo(false))
    if err != nil {
        return fmt.Sprintf("ERROR: %v", err)
    }
    defer conn.Close()

    data, _, err := conn.Get(path)
    if err != nil {
        return fmt.Sprintf("ERROR: %v", err)
    }
    return string(data)
}

// ── docker control helpers ────────────────────────────────────

func stopContainer(name string) {
    exec.Command("docker", "compose", "stop", name).Run()
    fmt.Printf("  [DOCKER] stopped container %s\n", name)
}

func startContainer(name string) {
    exec.Command("docker", "compose", "start", name).Run()
    fmt.Printf("  [DOCKER] started container %s\n", name)
}

func findLeaderNode() *ZKNode {
    for i := range nodes {
        if isAlive(nodes[i].statAddr) &&
            getMode(nodes[i].statAddr) == "leader" {
            return &nodes[i]
        }
    }
    return nil
}

// ─────────────────────────────────────────────────────────────
//  MAIN
// ─────────────────────────────────────────────────────────────

func main() {
    fmt.Println(strings.Repeat("=", 60))
    fmt.Println("  ZOOKEEPER ENSEMBLE LEADER ELECTION DEMO")
    fmt.Println("  3-node ensemble — Zab protocol")
    fmt.Println(strings.Repeat("=", 60))

    // ── Phase 1: initial ensemble state ───────────────────────
    fmt.Println("\n  Phase 1: Initial ensemble state")
    printEnsembleStatus("Ensemble after startup")

    // connect to ensemble (client auto-failovers between nodes)
    allAddrs := []string{
        "localhost:2181",
        "localhost:2182",
        "localhost:2183",
    }
    conn, _, err := zk.Connect(allAddrs, 10*time.Second,
        zk.WithLogInfo(false))
    if err != nil {
        fmt.Printf("  connect error: %v\n", err)
        return
    }
    defer conn.Close()

    // write initial data through the leader
    fmt.Println("\n  Writing test data to ensemble...")
    if err := writeTestData(conn, "/demo", "value-1"); err != nil {
        fmt.Printf("  write error: %v\n", err)
        return
    }
    fmt.Println("  Wrote /demo = 'value-1'")

    // verify all nodes see the same data
    fmt.Println("\n  Verifying data consistency across all nodes:")
    for _, n := range nodes {
        val := readFromNode(n.clientAddr, "/demo")
        fmt.Printf("    %s → /demo = '%s'\n", n.name, val)
    }

    // ── Phase 2: kill the leader ───────────────────────────────
    fmt.Println()
    fmt.Println(strings.Repeat("-", 60))
    fmt.Println("  Phase 2: Kill the current leader")
    fmt.Println(strings.Repeat("-", 60))

    leader := findLeaderNode()
    if leader == nil {
        fmt.Println("  ERROR: could not find leader")
        return
    }
    fmt.Printf("\n  Current leader: %s — stopping it now\n", leader.name)
    stopContainer(leader.container)

    fmt.Println("  Waiting for new election to complete (~10s)...")
    time.Sleep(12 * time.Second)

    printEnsembleStatus("Ensemble after leader crash")

    // ── Phase 3: verify new leader works ──────────────────────
    fmt.Println()
    fmt.Println(strings.Repeat("-", 60))
    fmt.Println("  Phase 3: Write through the new leader")
    fmt.Println(strings.Repeat("-", 60))

    // reconnect — old connection may have failed over already
    conn2, _, err := zk.Connect(allAddrs, 10*time.Second,
        zk.WithLogInfo(false))
    if err != nil {
        fmt.Printf("  reconnect error: %v\n", err)
        return
    }
    defer conn2.Close()

    // give client time to discover available servers
    time.Sleep(2 * time.Second)

    if err := writeTestData(conn2, "/demo", "value-2"); err != nil {
        fmt.Printf("  write error after failover: %v\n", err)
    } else {
        fmt.Println("  Wrote /demo = 'value-2' through new leader")
    }

    fmt.Println("\n  Verifying data consistency on surviving nodes:")
    for _, n := range nodes {
        if !isAlive(n.statAddr) {
            fmt.Printf("    %s → CRASHED\n", n.name)
            continue
        }
        val := readFromNode(n.clientAddr, "/demo")
        fmt.Printf("    %s → /demo = '%s'\n", n.name, val)
    }

    // ── Phase 4: restore crashed node ─────────────────────────
    fmt.Println()
    fmt.Println(strings.Repeat("-", 60))
    fmt.Println("  Phase 4: Restore the crashed node")
    fmt.Println(strings.Repeat("-", 60))

    startContainer(leader.container)
    fmt.Println("  Waiting for node to rejoin ensemble (~10s)...")
    time.Sleep(12 * time.Second)

    printEnsembleStatus("Ensemble after node recovery")

    // verify recovered node has caught up
    fmt.Println("\n  Verifying recovered node has latest data:")
    for _, n := range nodes {
        val := readFromNode(n.clientAddr, "/demo")
        fmt.Printf("    %s → /demo = '%s'\n", n.name, val)
    }

    // ── Phase 5: quorum loss ───────────────────────────────────
    fmt.Println()
    fmt.Println(strings.Repeat("-", 60))
    fmt.Println("  Phase 5: Quorum loss — stop 2 of 3 nodes")
    fmt.Println(strings.Repeat("-", 60))

    fmt.Println("  Stopping zoo2 and zoo3...")
    stopContainer("zoo2")
    stopContainer("zoo3")
    time.Sleep(5 * time.Second)

    printEnsembleStatus("Ensemble with quorum loss")

    fmt.Println("\n  Attempting write with no quorum...")
    conn3, _, err := zk.Connect([]string{"localhost:2181"},
        5*time.Second, zk.WithLogInfo(false))
    if err != nil {
        fmt.Printf("  connect error (expected): %v\n", err)
    } else {
        defer conn3.Close()
        err = writeTestData(conn3, "/demo", "value-3")
        if err != nil {
            fmt.Printf("  write rejected (expected): %v\n", err)
        } else {
            fmt.Println("  write succeeded (unexpected)")
        }
    }

    fmt.Println("\n  Restoring zoo2 and zoo3...")
    startContainer("zoo2")
    startContainer("zoo3")
    time.Sleep(12 * time.Second)

    printEnsembleStatus("Ensemble fully restored")

    // ── Summary ───────────────────────────────────────────────
    fmt.Println()
    fmt.Println(strings.Repeat("=", 60))
    fmt.Println("  SUMMARY")
    fmt.Println(strings.Repeat("=", 60))
    fmt.Println(`
  Zab election rules (3-node ensemble):

    Majority quorum = 2 of 3 nodes

    Phase 1  Leader discovery
             Servers broadcast their vote (epoch, zxid, myid)
             Higher epoch wins
             Tie → higher zxid wins
             Tie → higher myid wins

    Phase 2  Synchronization
             Elected leader syncs followers to its log

    Phase 3  Broadcast
             All writes go through the leader
             Leader replicates to followers
             Committed when majority ack

  Key properties:
    ✅ Survives 1 node failure  (majority = 2 of 3)
    ✅ Consistent reads         (all nodes have same data)
    ✅ Automatic re-election    (~10s after leader crash)
    ❌ Cannot write without quorum (zoo1 alone = rejected)`)
}
```

---

## Step 3 — Initialize the Go Module

```bash
mkdir zk-ensemble-demo && cd zk-ensemble-demo
go mod init zkensemble
go get github.com/go-zookeeper/zk
cp main.go .
```

---

## Step 4 — Run

```bash
# start the ensemble first
docker compose up -d
sleep 20   # wait for election

# run the demo
go run main.go
```

---

## Step 5 — Expected Output

```
============================================================
  ZOOKEEPER ENSEMBLE LEADER ELECTION DEMO
  3-node ensemble — Zab protocol
============================================================

  Phase 1: Initial ensemble state

  Ensemble after startup
  --------------------------------------------------
  Node     Status       Mode       ZxID       Leader
  --------------------------------------------------
  zoo1     UP           follower   0x100000004  ---
  zoo2     UP           leader     0x100000004  ← LEADER
  zoo3     UP           follower   0x100000004  ---

  Current leader: zoo2
  Wrote /demo = 'value-1'

  Verifying data consistency across all nodes:
    zoo1 → /demo = 'value-1'
    zoo2 → /demo = 'value-1'
    zoo3 → /demo = 'value-1'

  Phase 2: Kill the current leader
  [DOCKER] stopped container zoo2
  Waiting for new election to complete (~10s)...

  Ensemble after leader crash
  --------------------------------------------------
  zoo1     UP           leader     0x200000001  ← LEADER
  zoo2     DOWN         ---        ---
  zoo3     UP           follower   0x200000001  ---

  Current leader: zoo1

  Phase 3: Write through the new leader
  Wrote /demo = 'value-2' through new leader

  Verifying data consistency on surviving nodes:
    zoo1 → /demo = 'value-2'
    zoo2 → CRASHED
    zoo3 → /demo = 'value-2'

  Phase 4: Restore the crashed node
  [DOCKER] started container zoo2
  Waiting for node to rejoin ensemble (~10s)...

  zoo2 → /demo = 'value-2'   ← caught up after rejoin

  Phase 5: Quorum loss
  write rejected (expected): zk: could not connect to a server

============================================================
  SUMMARY
  Majority quorum = 2 of 3
  ✅ Survives 1 node failure
  ✅ Automatic re-election (~10s)
  ❌ Cannot write without quorum
============================================================
```

---

## Step 6 — Observe the Zab Election Manually

```bash
# watch election happen in real time
# terminal 1: monitor modes continuously
watch -n1 'for p in 2181 2182 2183; do
  echo -n "zoo $p: "
  echo stat | nc localhost $p 2>/dev/null | grep Mode || echo DOWN
done'

# terminal 2: kill the current leader
docker compose stop zoo2

# you will see the modes change from:
#   follower / leader / follower
# to:
#   leader / DOWN / follower   (or follower / DOWN / leader)
# within ~5-10 seconds
```

---

## Step 7 — Inspect the Zab State via mntr

```bash
# epoch and zxid reveal election history
echo mntr | nc localhost 2181 | grep -E "zk_epoch|zk_zxid|zk_server_state"

# output example after one re-election:
# zk_server_state   follower
# zk_zxid           0x200000003     ← epoch=2 (incremented after re-election)
# zk_epoch          2
```

The `epoch` number increments each time a new leader is elected.
The `zxid` is a 64-bit value where the upper 32 bits are the epoch
and the lower 32 bits are the transaction counter within that epoch.

---

## Step 8 — Teardown

```bash
docker compose down

# clean up test znodes
docker exec -it zoo1 zkCli.sh -server localhost:2181 delete /demo
```
