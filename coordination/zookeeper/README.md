# ZooKeeper Tutorial with Docker

## Prerequisites

```bash
docker --version        # Docker 20+
docker compose version  # Compose v2
```

Start ZooKeeper using a `docker-compose.yml` with the
`ZOO_4LW_COMMANDS_WHITELIST` environment variable set on all nodes:

```
ZOO_4LW_COMMANDS_WHITELIST: "*"
```

Then bring the cluster up:

```bash
docker compose up -d
docker compose ps     # all three should show "running"
```

---

## Step 1 — Connect with zkCli

```bash
docker exec -it <project>-zoo1-1 zkCli.sh -server localhost:2181
```

---

## Step 2 — Basic Znode Operations

Once inside `zkCli`:

### Create

```bash
# Persistent node
create /app "hello"

# Persistent node with children
create /app/config "version=1"
create /app/config/db "host=localhost"

# Ephemeral node (deleted when session ends)
create -e /app/session-token "abc123"

# Sequential node (ZK appends a counter suffix)
create -s /app/tasks/task- "payload-1"
create -s /app/tasks/task- "payload-2"
create -s /app/tasks/task- "payload-3"

# Ephemeral + sequential (used in locks)
create -e -s /app/locks/lock- "client-1"
create -e -s /app/locks/lock- "client-2"
```

### Read

```bash
get /app                    # data + metadata
get -s /app                 # include stat
ls /app                     # list children
ls -R /app                  # recursive list
stat /app                   # metadata only
```

### Update

```bash
set /app/config "version=2"
set /app/config "version=3" 1   # conditional on version 1
```

### Delete

```bash
delete /app/config/db       # leaf node only
deleteall /app              # recursive delete
```

### Watch

```bash
get -w /app/config          # watch for data change
ls -w /app                  # watch for children change
stat -w /app                # watch for any change
```

### Transactions

```bash
multi
create /txn/a "data-a"
create /txn/b "data-b"
set /txn/a "updated"
execute
```

---

## Step 3 — Four Letter Words (Admin Commands)

Run these from outside the container. Requires `ZOO_4LW_COMMANDS_WHITELIST`
to be set (see Prerequisites).

```bash
# Is the server alive?
echo ruok | nc localhost 2181           # returns: imok

# Server stats and mode (leader/follower)
echo stat | nc localhost 2181

# Detailed metrics
echo mntr | nc localhost 2181 | grep -E "zk_server_state|zk_version|zk_avg_latency"

# Server configuration
echo conf | nc localhost 2181

# Active client connections
echo cons | nc localhost 2181

# Sessions and ephemeral nodes
echo dump | nc localhost 2181

# Environment (JVM, OS)
echo envi | nc localhost 2181

# Watch summary
echo wchs | nc localhost 2181

# Check all three nodes at once
for port in 2181 2182 2183; do
  echo -n "zoo $port: "
  echo mntr | nc localhost $port | grep zk_server_state
done
```

| Command | What it returns |
|---------|-----------------|
| `ruok`  | Is the server running — returns `imok` |
| `stat`  | Server stats, connections, mode (leader/follower) |
| `mntr`  | Detailed metrics — latency, znodes, watches, memory |
| `srvr`  | Server summary without connection details |
| `conf`  | Current zoo.cfg configuration |
| `cons`  | All active client connections |
| `dump`  | Sessions and ephemeral nodes |
| `envi`  | JVM and OS environment info |
| `wchs`  | Watch summary counts |
| `wchp`  | Watches by path |
| `wchc`  | Watches by session |
| `dirs`  | Data and log directory sizes |

> **Note:** `wchc` and `wchp` can be expensive on busy servers.
> Fine for development; use with caution in production.

---

## Step 4 — Simulate the Lock Recipe Manually

Open **three separate terminals**, each connected to a different ZK node.

### Terminal 1 — Client 1 acquires the lock

```bash
docker exec -it <project>-zoo1-1 zkCli.sh -server localhost:2181

# Create lock parent
create /locks ""

# Client 1 creates an ephemeral sequential node
create -e -s /locks/lock- "client-1"
# ZooKeeper assigns: /locks/lock-0000000001

# Check children — we are the lowest, so we hold the lock
ls /locks
# [lock-0000000001]

# Simulate doing work
get /locks/lock-0000000001
```

### Terminal 2 — Client 2 waits

```bash
docker exec -it <project>-zoo2-1 zkCli.sh -server localhost:2182

create -e -s /locks/lock- "client-2"
# Assigned: /locks/lock-0000000002

ls /locks
# [lock-0000000001, lock-0000000002]
# 0000000002 > 0000000001 → we do NOT hold the lock

# Watch only the immediate predecessor
stat -w /locks/lock-0000000001
# Now waiting for a watch event...
```

### Terminal 3 — Client 3 waits

```bash
docker exec -it <project>-zoo3-1 zkCli.sh -server localhost:2183

create -e -s /locks/lock- "client-3"
# Assigned: /locks/lock-0000000003

ls /locks
# [lock-0000000001, lock-0000000002, lock-0000000003]

# Watch only the immediate predecessor (not 0000000001)
stat -w /locks/lock-0000000002
```

### Terminal 1 — Release the lock

```bash
delete /locks/lock-0000000001
# Terminal 2's watch fires: WatchedEvent type:NodeDeleted
# Terminal 3 is unaffected — it is watching 0000000002, which still exists
```

### Terminal 2 — Reacts to watch

```bash
# Watch fired — re-check children
ls /locks
# [lock-0000000002, lock-0000000003]
# We are now lowest → we hold the lock
```

This demonstrates **no herd effect** — only one client is notified per release.

---

## Step 5 — Simulate Session Expiry (Crash Recovery)

### Terminal 1 — Create an ephemeral node

```bash
docker exec -it <project>-zoo1-1 zkCli.sh -server localhost:2181

create -e /session-test "alive"
get /session-test     # data: "alive"
```

### Terminal 2 — Watch it

```bash
docker exec -it <project>-zoo2-1 zkCli.sh -server localhost:2182

stat -w /session-test
```

### Simulate a crash

Close Terminal 1 (`Ctrl+D` or `Ctrl+C`) to end the session. Wait for
the session timeout (~10–30 seconds).

Terminal 2 will receive:

```
WatchedEvent state:SyncConnected type:NodeDeleted path:/session-test
```

### Verify the node is gone

```bash
# In Terminal 2
get /session-test
# KeeperErrorCode = NoNode for /session-test
```

The ephemeral node was deleted automatically — this is ZooKeeper's
built-in lock release on crash, equivalent to a lease expiry.

---

## Step 6 — Observe Leader Election in the Ensemble

```bash
# Check which node is the current leader
for port in 2181 2182 2183; do
  echo -n "Port $port: "
  echo stat | nc localhost $port 2>/dev/null | grep Mode
done
# Mode: leader   ← one node
# Mode: follower ← the other two
```

### Kill the leader

```bash
# If zoo1 is the leader:
docker compose stop zoo1

# Wait a few seconds, then check the remaining nodes
for port in 2182 2183; do
  echo -n "Port $port: "
  echo stat | nc localhost $port 2>/dev/null | grep Mode
done
# One of them is now elected the new leader
```

### Bring zoo1 back

```bash
docker compose start zoo1

# zoo1 rejoins as a follower
echo stat | nc localhost 2181 | grep Mode   # Mode: follower
```

---

## Step 7 — Admin HTTP API

ZooKeeper 3.9 ships with a built-in admin HTTP server on port `8080`.

```bash
# Browse all available commands
open http://localhost:8080/commands

# Useful endpoints
curl http://localhost:8080/commands/stat
curl http://localhost:8080/commands/mntr
curl http://localhost:8080/commands/cons     # active connections
curl http://localhost:8080/commands/wchs     # watches summary
curl http://localhost:8080/commands/dump     # sessions + ephemeral nodes
```

---

## Step 8 — Teardown

```bash
# Stop and remove containers
docker compose down

# Also remove volumes (wipes all ZooKeeper data)
docker compose down -v
```

---

## Summary

| Step | Concept demonstrated |
|------|----------------------|
| 1    | Connecting to the ensemble via zkCli |
| 2    | Persistent, ephemeral, sequential znodes — the building blocks |
| 3    | Four-letter words — observing server state and metrics |
| 4    | Lock recipe — sequential nodes, predecessor watching, no herd effect |
| 5    | Session expiry — automatic lock release on crash |
| 6    | Leader election — ensemble failover and re-election |
| 7    | Admin HTTP API |

The lock recipe in Step 4 is the manual version of what libraries like
**Apache Curator** (Java) and **kazoo** (Python) implement automatically.
This tutorial shows exactly what those libraries do under the hood.
