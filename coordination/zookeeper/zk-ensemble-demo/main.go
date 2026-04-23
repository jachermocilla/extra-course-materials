package main

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"

	"github.com/go-zookeeper/zk"
)

// ─────────────────────────────────────────────
//  ENSEMBLE NODE DEFINITIONS
// ─────────────────────────────────────────────

type ZKNode struct {
	name       string
	clientAddr string // host:port for ZK client connections
	statAddr   string // host:port for four-letter word commands
	container  string // docker container name
}

var nodes = []ZKNode{
	{name: "zoo1", clientAddr: "localhost:2181",
		statAddr: "localhost:2181", container: "zoo1"},
	{name: "zoo2", clientAddr: "localhost:2182",
		statAddr: "localhost:2182", container: "zoo2"},
	{name: "zoo3", clientAddr: "localhost:2183",
		statAddr: "localhost:2183", container: "zoo3"},
}

// ─────────────────────────────────────────────
//  FOUR-LETTER WORD HELPERS
// ─────────────────────────────────────────────

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
			stats[strings.TrimSpace(parts[0])] =
				strings.TrimSpace(parts[1])
		}
	}
	return stats
}

// ─────────────────────────────────────────────
//  PRINT ENSEMBLE STATUS
// ─────────────────────────────────────────────

func printEnsembleStatus(label string) {
	fmt.Printf("\n  %s\n", label)
	fmt.Println("  " + strings.Repeat("-", 56))
	fmt.Printf("  %-8s %-8s %-12s %-16s %s\n",
		"Node", "Status", "Mode", "ZxID", "Epoch")
	fmt.Println("  " + strings.Repeat("-", 56))

	leaderName := ""
	for _, n := range nodes {
		alive := isAlive(n.statAddr)
		status := "UP  "
		mode := "---"
		zxid := "---"
		epoch := "---"
		marker := ""

		if !alive {
			status = "DOWN"
		} else {
			mode = getMode(n.statAddr)
			stats := getStats(n.statAddr)
			if v, ok := stats["zk_zxid"]; ok {
				zxid = v
			}
			if v, ok := stats["zk_epoch"]; ok {
				epoch = v
			}
			if mode == "leader" {
				leaderName = n.name
				marker = " ← LEADER"
			}
		}
		fmt.Printf("  %-8s %-8s %-12s %-16s %s%s\n",
			n.name, status, mode, zxid, epoch, marker)
	}
	if leaderName != "" {
		fmt.Printf("\n  Current leader: %s\n", leaderName)
	} else {
		fmt.Println("\n  No leader — quorum lost")
	}
}

// ─────────────────────────────────────────────
//  DATA HELPERS
// ─────────────────────────────────────────────

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

func verifyConsistency(path string) {
	fmt.Printf("\n  Data consistency check for %s:\n", path)
	for _, n := range nodes {
		if !isAlive(n.statAddr) {
			fmt.Printf("    %-6s → DOWN\n", n.name)
			continue
		}
		val := readFromNode(n.clientAddr, path)
		fmt.Printf("    %-6s → %s = '%s'\n", n.name, path, val)
	}
}

// ─────────────────────────────────────────────
//  DOCKER HELPERS
// ─────────────────────────────────────────────

func stopContainer(name string) {
	exec.Command("docker", "compose", "stop", name).Run()
	fmt.Printf("  [DOCKER] stopped %s\n", name)
}

func startContainer(name string) {
	exec.Command("docker", "compose", "start", name).Run()
	fmt.Printf("  [DOCKER] started %s\n", name)
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

func waitForLeader(timeout time.Duration) *ZKNode {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if n := findLeaderNode(); n != nil {
			return n
		}
		time.Sleep(1 * time.Second)
		fmt.Print(".")
	}
	fmt.Println()
	return nil
}

// ─────────────────────────────────────────────
//  MAIN
// ─────────────────────────────────────────────

func main() {
	allAddrs := []string{
		"localhost:2181",
		"localhost:2182",
		"localhost:2183",
	}

	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("  ZOOKEEPER ENSEMBLE LEADER ELECTION DEMO")
	fmt.Println("  3-node ensemble — Zab protocol")
	fmt.Println(strings.Repeat("=", 60))

	// ── Phase 1: initial ensemble state ───────────────────────
	fmt.Println("\n  Phase 1: Initial Ensemble State")
	fmt.Println(strings.Repeat("-", 60))

	fmt.Print("  Waiting for leader election")
	leader := waitForLeader(30 * time.Second)
	if leader == nil {
		fmt.Println("  ERROR: no leader elected within 30s")
		return
	}

	printEnsembleStatus("Ensemble after startup")

	// connect client to ensemble
	conn, _, err := zk.Connect(allAddrs, 10*time.Second,
		zk.WithLogInfo(false))
	if err != nil {
		fmt.Printf("  connect error: %v\n", err)
		return
	}
	defer conn.Close()
	time.Sleep(2 * time.Second)

	fmt.Println("\n  Writing /demo = 'value-1' through leader...")
	if err := writeTestData(conn, "/demo", "value-1"); err != nil {
		fmt.Printf("  write error: %v\n", err)
		return
	}
	verifyConsistency("/demo")

	// ── Phase 2: kill the leader ───────────────────────────────
	fmt.Println()
	fmt.Println(strings.Repeat("-", 60))
	fmt.Println("  Phase 2: Kill the Current Leader")
	fmt.Println(strings.Repeat("-", 60))

	fmt.Printf("\n  Stopping leader: %s\n", leader.name)
	stopContainer(leader.container)

	fmt.Print("  Waiting for new leader election")
	newLeader := waitForLeader(30 * time.Second)
	if newLeader == nil {
		fmt.Println("  ERROR: no new leader elected")
		return
	}

	printEnsembleStatus("Ensemble after leader crash")

	// ── Phase 3: write through new leader ─────────────────────
	fmt.Println()
	fmt.Println(strings.Repeat("-", 60))
	fmt.Println("  Phase 3: Write Through New Leader")
	fmt.Println(strings.Repeat("-", 60))

	conn2, _, err := zk.Connect(allAddrs, 10*time.Second,
		zk.WithLogInfo(false))
	if err != nil {
		fmt.Printf("  reconnect error: %v\n", err)
		return
	}
	defer conn2.Close()
	time.Sleep(2 * time.Second)

	fmt.Println("\n  Writing /demo = 'value-2' through new leader...")
	if err := writeTestData(conn2, "/demo", "value-2"); err != nil {
		fmt.Printf("  write error: %v\n", err)
	} else {
		fmt.Println("  Write succeeded")
	}
	verifyConsistency("/demo")

	// ── Phase 4: restore crashed node ─────────────────────────
	fmt.Println()
	fmt.Println(strings.Repeat("-", 60))
	fmt.Println("  Phase 4: Restore Crashed Node")
	fmt.Println(strings.Repeat("-", 60))

	startContainer(leader.container)
	fmt.Print("  Waiting for node to rejoin")
	time.Sleep(15 * time.Second)
	fmt.Println()

	printEnsembleStatus("Ensemble after node recovery")
	verifyConsistency("/demo")

	// ── Phase 5: quorum loss ───────────────────────────────────
	fmt.Println()
	fmt.Println(strings.Repeat("-", 60))
	fmt.Println("  Phase 5: Quorum Loss (stop 2 of 3 nodes)")
	fmt.Println(strings.Repeat("-", 60))

	fmt.Println("\n  Stopping zoo2 and zoo3...")
	stopContainer("zoo2")
	stopContainer("zoo3")
	time.Sleep(5 * time.Second)

	printEnsembleStatus("Ensemble with quorum lost")

	fmt.Println("\n  Attempting write with no quorum (should fail)...")
	conn3, _, err := zk.Connect([]string{"localhost:2181"},
		5*time.Second, zk.WithLogInfo(false))
	if err != nil {
		fmt.Printf("  connect rejected (expected): %v\n", err)
	} else {
		defer conn3.Close()
		time.Sleep(2 * time.Second)
		err = writeTestData(conn3, "/demo", "value-3")
		if err != nil {
			fmt.Printf("  write rejected (expected): %v\n", err)
		} else {
			fmt.Println("  write succeeded (unexpected)")
		}
	}

	// ── Phase 6: restore full ensemble ────────────────────────
	fmt.Println()
	fmt.Println(strings.Repeat("-", 60))
	fmt.Println("  Phase 6: Restore Full Ensemble")
	fmt.Println(strings.Repeat("-", 60))

	startContainer("zoo2")
	startContainer("zoo3")
	fmt.Print("  Waiting for ensemble to stabilize")
	waitForLeader(30 * time.Second)
	fmt.Println()

	printEnsembleStatus("Ensemble fully restored")
	verifyConsistency("/demo")

	// ── Summary ───────────────────────────────────────────────
	fmt.Println()
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("  SUMMARY")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println(`
  Zab election outcome:

    Phase 1  Initial election   — leader elected at startup
    Phase 2  Leader crash       — new election in ~5-10s
    Phase 3  Write after crash  — new leader accepts writes
    Phase 4  Node recovery      — rejoined node catches up
    Phase 5  Quorum loss        — writes rejected (no majority)
    Phase 6  Full restore       — ensemble healthy again

  ZxID format:  0x<epoch><counter>
    epoch    — increments each re-election
    counter  — resets to 0 each new epoch

  Majority quorum = 2 of 3 nodes
    ✅ Survives 1 node failure
    ❌ Rejects writes with only 1 node`)
}
