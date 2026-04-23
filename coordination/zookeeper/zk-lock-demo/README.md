# ZooKeeper Distributed Locking Tutorial
## Bank Balance Problem in Go

---

## Overview

This tutorial demonstrates how to use ZooKeeper as a distributed lock
manager to protect a shared bank balance across multiple Go processes.
Three processes concurrently attempt to add 100 PHP to a shared balance
of 1000 PHP. Without locking the balance ends up wrong. With ZooKeeper
locking it always ends at 1300 PHP.

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
zk-lock-demo/
├── docker-compose.yml
├── main.go
└── lock/
    └── zklock.go
```

---

## Step 1 — Start ZooKeeper Ensemble

Use the following `docker-compose.yml`:

```yaml
# docker-compose.yml
services:
 
  zoo1:
    image: zookeeper:3.9
    hostname: zoo1
    container_name: zoo1
    ports:
      - "2181:2181"
      - "8080:8080"
    environment:
      ZOO_MY_ID: 1
      ZOO_STANDALONE_ENABLED: "true"
      ZOO_4LW_COMMANDS_WHITELIST: "mntr,conf,ruok,stat,srvr,cons,dump,envi,wchs,wchp,wchc,dirs"
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

for port in 2181; do
  echo -n "zoo $port: "
  echo ruok | nc localhost $port
done
# imok
```

---

## Step 2 — Understand the Lock Recipe

ZooKeeper implements distributed locking using **ephemeral sequential nodes**:

```
/locks/
  balance-lock-0000000001   ← created by Process 1
  balance-lock-0000000002   ← created by Process 2
  balance-lock-0000000003   ← created by Process 3
```

Rules:
1. Each process creates an ephemeral sequential node under `/locks/`
2. Get all children and sort them
3. If your node is the **lowest** → you hold the lock
4. Otherwise → watch the node immediately before yours and wait
5. When your watch fires → go to step 2
6. On exit → delete your node (lock released)

This guarantees:
- **Mutual exclusion** — only the lowest node holder is in the CS
- **No herd effect** — each waiter watches only its predecessor
- **Fault tolerance** — ephemeral nodes vanish if the process crashes

---

## Step 3 — ZooKeeper Lock Implementation

### `lock/zklock.go`

```go
package lock

import (
    "fmt"
    "sort"
    "strings"
    "time"

    "github.com/go-zookeeper/zk"
)

const lockDir = "/locks"

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

// Lock acquires the distributed lock, blocking until acquired
func (l *ZKLock) Lock() error {
    // Step 1: create ephemeral sequential node
    path, err := l.conn.CreateProtectedEphemeralSequential(
        l.prefix, []byte{}, zk.WorldACL(zk.PermAll))
    if err != nil {
        return fmt.Errorf("create node: %w", err)
    }
    l.lockPath = path

    for {
        // Step 2: get all children
        children, _, err := l.conn.Children(lockDir)
        if err != nil {
            return fmt.Errorf("get children: %w", err)
        }
        sort.Strings(children)

        // extract just our node name from full path
        myNode := l.lockPath[strings.LastIndex(l.lockPath, "/")+1:]

        // Step 3: check if we are the lowest
        if children[0] == myNode {
            return nil // we hold the lock
        }

        // Step 4: find predecessor and watch it
        predecessor := ""
        for i, child := range children {
            if child == myNode && i > 0 {
                predecessor = lockDir + "/" + children[i-1]
                break
            }
        }

        if predecessor == "" {
            // re-check — we may have become lowest
            continue
        }

        // Step 5: wait for predecessor to be deleted
        exists, _, watch, err := l.conn.ExistsW(predecessor)
        if err != nil {
            return fmt.Errorf("watch: %w", err)
        }
        if exists {
            <-watch // block until predecessor deleted
        }
        // loop back to re-check
    }
}

// Unlock releases the lock by deleting our ephemeral node
func (l *ZKLock) Unlock() error {
    if l.lockPath == "" {
        return nil
    }
    err := l.conn.Delete(l.lockPath, -1)
    l.lockPath = ""
    return err
}
```

---

## Step 4 — Main Program

### `main.go`

```go
package main

import (
    "encoding/binary"
    "fmt"
    "os"
    "strconv"
    "strings"
    "sync"
    "time"

    "github.com/go-zookeeper/zk"
    "zk-lock-demo/lock"
)

const (
    balancePath    = "/balance"
    initialBalance = 1000.0
    servers        = "localhost:2181"
)

// ── balance helpers stored as float64 bytes in ZooKeeper ──────

func encodeBalance(b float64) []byte {
    buf := make([]byte, 8)
    bits := *(*uint64)((*[8]byte)((*[8]byte)(&buf[0])))
    _ = bits
    var raw [8]byte
    u := *(*uint64)((*[8]byte)((*[8]byte)(&raw[0])))
    _ = u
    // use encoding/binary for clarity
    b64 := *(*uint64)((*[8]byte)((*[8]byte)(&raw)))
    _ = b64
    binary.LittleEndian.PutUint64(buf,
        *(*uint64)((*[8]byte)((*[8]byte)(&raw))))
    return []byte(fmt.Sprintf("%.2f", b))
}

func decodeBalance(data []byte) float64 {
    val, _ := strconv.ParseFloat(string(data), 64)
    return val
}

// ── initialize balance node in ZooKeeper ──────────────────────

func initBalance(conn *zk.Conn) error {
    exists, _, err := conn.Exists(balancePath)
    if err != nil {
        return err
    }
    if !exists {
        _, err = conn.Create(balancePath,
            []byte(fmt.Sprintf("%.2f", initialBalance)),
            0, zk.WorldACL(zk.PermAll))
        if err != nil && err != zk.ErrNodeExists {
            return err
        }
        fmt.Printf("  [INIT] balance node created: %.2f PHP\n",
            initialBalance)
    }
    return nil
}

// ── process: acquire lock, read, modify, write, release ───────

func process(id int, servers []string, wg *sync.WaitGroup) {
    defer wg.Done()

    // connect to ZooKeeper ensemble
    conn, err := lock.Connect(servers)
    if err != nil {
        fmt.Printf("  [P%d] connect error: %v\n", id, err)
        return
    }
    defer conn.Close()

    zkLock := lock.NewZKLock(conn, "balance-lock")

    // simulate staggered start
    time.Sleep(time.Duration(id*50) * time.Millisecond)

    fmt.Printf("  [P%d] requesting lock\n", id)

    // ── acquire lock ──────────────────────────────────────────
    if err := zkLock.Lock(); err != nil {
        fmt.Printf("  [P%d] lock error: %v\n", id, err)
        return
    }
    fmt.Printf("  [P%d] lock acquired\n", id)

    // ── critical section: read → modify → write ───────────────
    data, stat, err := conn.Get(balancePath)
    if err != nil {
        fmt.Printf("  [P%d] get error: %v\n", id, err)
        zkLock.Unlock()
        return
    }

    before := decodeBalance(data)
    after := before + 100
    fmt.Printf("  [P%d] *** CRITICAL SECTION: %.2f → %.2f PHP ***\n",
        id, before, after)

    time.Sleep(30 * time.Millisecond) // simulate work

    _, err = conn.Set(balancePath,
        []byte(fmt.Sprintf("%.2f", after)), stat.Version)
    if err != nil {
        fmt.Printf("  [P%d] set error: %v\n", id, err)
        zkLock.Unlock()
        return
    }

    // ── release lock ──────────────────────────────────────────
    if err := zkLock.Unlock(); err != nil {
        fmt.Printf("  [P%d] unlock error: %v\n", id, err)
        return
    }
    fmt.Printf("  [P%d] lock released\n", id)
}

func main() {
    zkServers := strings.Split(servers, ",")

    fmt.Println(strings.Repeat("=", 60))
    fmt.Println("  ZOOKEEPER DISTRIBUTED LOCK — BANK BALANCE")
    fmt.Println(strings.Repeat("=", 60))
    fmt.Printf("\n  ZooKeeper ensemble : %s\n", servers)
    fmt.Printf("  Initial balance    : %.2f PHP\n", initialBalance)
    fmt.Println("  3 processes each add 100 PHP")
    fmt.Printf("  Expected final     : %.2f PHP\n\n",
        initialBalance+300)
    fmt.Println(strings.Repeat("-", 60))

    // init balance node using process 1's connection
    initConn, err := lock.Connect(zkServers)
    if err != nil {
        fmt.Fprintf(os.Stderr, "init connect: %v\n", err)
        os.Exit(1)
    }
    if err := initBalance(initConn); err != nil {
        fmt.Fprintf(os.Stderr, "init balance: %v\n", err)
        os.Exit(1)
    }
    initConn.Close()

    // run 3 processes concurrently
    var wg sync.WaitGroup
    wg.Add(3)
    for i := 1; i <= 3; i++ {
        go process(i, zkServers, &wg)
    }
    wg.Wait()

    // read final balance
    readConn, err := lock.Connect(zkServers)
    if err != nil {
        fmt.Fprintf(os.Stderr, "read connect: %v\n", err)
        os.Exit(1)
    }
    defer readConn.Close()

    data, _, err := readConn.Get(balancePath)
    if err != nil {
        fmt.Fprintf(os.Stderr, "read balance: %v\n", err)
        os.Exit(1)
    }

    final := decodeBalance(data)
    fmt.Println(strings.Repeat("-", 60))
    fmt.Printf("\n  Final balance : %.2f PHP\n", final)
    if final == initialBalance+300 {
        fmt.Println("  ✅ Correct — ZooKeeper lock held, no race condition")
    } else {
        fmt.Printf("  ❌ Wrong — expected %.2f got %.2f\n",
            initialBalance+300, final)
    }
    fmt.Println(strings.Repeat("=", 60))
}
```

---

## Step 5 — Initialize the Go Module

```bash
mkdir zk-lock-demo && cd zk-lock-demo
go mod init zk-lock-demo
go get github.com/go-zookeeper/zk
mkdir lock
# copy zklock.go into lock/
# copy main.go into root
```

---

## Step 6 — Run

```bash
go run main.go
```

Expected output:

```
============================================================
  ZOOKEEPER DISTRIBUTED LOCK — BANK BALANCE
============================================================

  ZooKeeper ensemble : localhost:2181
  Initial balance    : 1000.00 PHP
  3 processes each add 100 PHP
  Expected final     : 1300.00 PHP

------------------------------------------------------------
  [INIT] balance node created: 1000.00 PHP
  [P1] requesting lock
  [P2] requesting lock
  [P3] requesting lock
  [P1] lock acquired
  [P1] *** CRITICAL SECTION: 1000.00 → 1100.00 PHP ***
  [P1] lock released
  [P2] lock acquired
  [P2] *** CRITICAL SECTION: 1100.00 → 1200.00 PHP ***
  [P2] lock released
  [P3] lock acquired
  [P3] *** CRITICAL SECTION: 1200.00 → 1300.00 PHP ***
  [P3] lock released
------------------------------------------------------------

  Final balance : 1300.00 PHP
  ✅ Correct — ZooKeeper lock held, no race condition
============================================================
```

---

## Step 7 — Verify Lock Nodes in zkCli

While the program is running (or after), inspect the lock nodes:

```bash
# connect to any node
docker exec -it zoo1 zkCli.sh -server localhost:2181

# see the lock directory
ls /locks
# [balance-lock-0000000001, balance-lock-0000000002, balance-lock-0000000003]

# see the balance value
get /balance
# 1300.00

# watch the balance change in real time
get -w /balance
```

---

## Step 8 — Observe Lock Ordering

Run with verbose ZooKeeper logging to see the sequential node numbers:

```bash
# inspect which process holds the lock at any moment
docker exec -it zoo1 zkCli.sh -server localhost:2181 ls /locks
```

The lowest sequence number is always the lock holder:

```
lock-0000000001  ← HELD by P1
lock-0000000002  ← P2 watching 0000000001
lock-0000000003  ← P3 watching 0000000002
```

When P1 deletes `0000000001`, only P2 is notified (it was watching
`0000000001`). P3 is unaffected — it is still watching `0000000002`.
This is the **no herd effect** guarantee.

---

## Step 9 — Simulate a Crash

Kill the lock holder mid-execution to observe automatic recovery:

```bash
# in one terminal — run the program
go run main.go

# in another terminal — kill zoo1 while P1 holds the lock
docker compose stop zoo1

# the Go client automatically failovers to zoo2 or zoo3
# P1's ephemeral node is deleted when its session expires (~10s)
# P2 acquires the lock and continues
```

ZooKeeper session timeout is the lease — crashed processes cannot
hold locks indefinitely.

---

## Step 10 — Teardown

```bash
docker compose down

# remove balance node for a clean re-run
docker exec -it zoo1 zkCli.sh -server localhost:2181 deleteall /balance
docker exec -it zoo1 zkCli.sh -server localhost:2181 deleteall /locks
```

---

## Summary

| Component | Role |
|-----------|------|
| Ephemeral node | Auto-deleted on crash — no deadlock |
| Sequential node | Determines lock order — FIFO fairness |
| Predecessor watch | Only one process notified per release — no herd |
| ZK ensemble (3 nodes) | Majority quorum — survives one node failure |
| `stat.Version` in `Set` | Optimistic concurrency — rejects stale writes |

The balance is stored **inside ZooKeeper** itself as a znode value,
so the lock and the data it protects live in the same system.
The `stat.Version` check on `conn.Set` provides an additional
layer of protection — if two processes somehow both read the same
version, the second write will fail with a version mismatch error.
