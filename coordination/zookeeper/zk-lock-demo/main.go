package main

import (
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
	zkServers      = "localhost:2181"
)

// ── balance helpers ───────────────────────────────────────────

func encodeBalance(b float64) []byte {
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
			encodeBalance(initialBalance),
			0, zk.WorldACL(zk.PermAll))
		if err != nil && err != zk.ErrNodeExists {
			return err
		}
		fmt.Printf("  [INIT] balance node created: %.2f PHP\n",
			initialBalance)
	} else {
		fmt.Printf("  [INIT] balance node already exists\n")
	}
	return nil
}

// ── process ───────────────────────────────────────────────────

func process(id int, servers []string, wg *sync.WaitGroup) {
	defer wg.Done()

	conn, err := lock.Connect(servers)
	if err != nil {
		fmt.Printf("  [P%d] connect error: %v\n", id, err)
		return
	}
	defer conn.Close()

	zkLock := lock.NewZKLock(conn, "balance-lock")

	// stagger start times slightly
	time.Sleep(time.Duration(id*20) * time.Millisecond)

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

	time.Sleep(30 * time.Millisecond) // simulate work inside CS

	_, err = conn.Set(balancePath, encodeBalance(after), stat.Version)
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

// ─────────────────────────────────────────────────────────────
//  MAIN
// ─────────────────────────────────────────────────────────────

func main() {
	servers := strings.Split(zkServers, ",")

	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("  ZOOKEEPER DISTRIBUTED LOCK — BANK BALANCE")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("\n  ZooKeeper : %s\n", zkServers)
	fmt.Printf("  Initial   : %.2f PHP\n", initialBalance)
	fmt.Println("  3 processes each add 100 PHP")
	fmt.Printf("  Expected  : %.2f PHP\n\n", initialBalance+300)
	fmt.Println(strings.Repeat("-", 60))

	// initialize balance node
	initConn, err := lock.Connect(servers)
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
		go process(i, servers, &wg)
	}
	wg.Wait()

	// read and print final balance
	readConn, err := lock.Connect(servers)
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
