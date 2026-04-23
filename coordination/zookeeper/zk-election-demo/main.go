package main

import (
    "fmt"
    "os"
    "strconv"
    "strings"
    "sync"
    "time"

    "github.com/go-zookeeper/zk"
    "zk-election-demo/election"
)

const (
    balancePath    = "/balance"
    initialBalance = 1000.0
    zkServer       = "localhost:2181"
)

func encodeBalance(b float64) []byte {
    return []byte(fmt.Sprintf("%.2f", b))
}

func decodeBalance(data []byte) float64 {
    val, _ := strconv.ParseFloat(string(data), 64)
    return val
}

func initBalance(conn *zk.Conn) error {
    exists, _, err := conn.Exists(balancePath)
    if err != nil {
        return err
    }
    if !exists {
        _, err = conn.Create(balancePath,
            encodeBalance(initialBalance),
            0, zk.WorldACL(zk.PermAll))
        if err != nil && err != zk.ErrNodeExists {
            return err
        }
        fmt.Printf("  [INIT] balance node created: %.2f PHP\n",
            initialBalance)
    }
    return nil
}

// process enters the election and only modifies balance when leader
func process(id int, wg *sync.WaitGroup) {
    defer wg.Done()

    conn, err := election.Connect([]string{zkServer})
    if err != nil {
        fmt.Printf("  [P%d] connect error: %v\n", id, err)
        return
    }
    defer conn.Close()

    // elected callback — only leader modifies the balance
    elected := make(chan struct{}, 1)

    candidate := election.NewCandidate(conn, id,
        func() { elected <- struct{}{} }, // onElected
        nil, // onDemoted
    )

    if err := candidate.Enter(); err != nil {
        fmt.Printf("  [P%d] enter error: %v\n", id, err)
        return
    }

    // wait until elected
    <-elected
    fmt.Printf("  [P%d] I am leader — modifying balance\n", id)

    // read → modify → write
    data, stat, err := conn.Get(balancePath)
    if err != nil {
        fmt.Printf("  [P%d] get error: %v\n", id, err)
        return
    }

    before := decodeBalance(data)
    after := before + 100
    fmt.Printf("  [P%d] *** CRITICAL SECTION: %.2f → %.2f PHP ***\n",
        id, before, after)

    time.Sleep(30 * time.Millisecond)

    _, err = conn.Set(balancePath, encodeBalance(after), stat.Version)
    if err != nil {
        fmt.Printf("  [P%d] set error: %v\n", id, err)
        return
    }

    // resign so the next candidate becomes leader
    if err := candidate.Resign(); err != nil {
        fmt.Printf("  [P%d] resign error: %v\n", id, err)
        return
    }
    fmt.Printf("  [P%d] resigned — next leader will take over\n", id)
}

func main() {
    servers := []string{zkServer}

    fmt.Println(strings.Repeat("=", 60))
    fmt.Println("  ZOOKEEPER LEADER ELECTION — BANK BALANCE")
    fmt.Println(strings.Repeat("=", 60))
    fmt.Printf("\n  ZooKeeper : %s\n", zkServer)
    fmt.Printf("  Initial   : %.2f PHP\n", initialBalance)
    fmt.Println("  3 processes — only the leader modifies balance")
    fmt.Printf("  Expected  : %.2f PHP\n\n", initialBalance+300)
    fmt.Println(strings.Repeat("-", 60))

    // initialize balance node
    initConn, err := election.Connect(servers)
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
        go process(i, &wg)
    }
    wg.Wait()

    // read final balance
    readConn, err := election.Connect(servers)
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
        fmt.Println("  ✅ Correct — only leaders modified balance")
    } else {
        fmt.Printf("  ❌ Wrong — expected %.2f got %.2f\n",
            initialBalance+300, final)
    }
    fmt.Println(strings.Repeat("=", 60))
}
