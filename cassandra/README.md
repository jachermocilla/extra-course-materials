# Cassandra: A Decentralized Structured Storage System
### A Hands-On Tutorial Based on the Original Facebook Paper (Lakshman & Malik, LADIS 2009)

---

## Background

Apache Cassandra was created at Facebook by **Avinash Lakshman** (a co-author of Amazon Dynamo) and **Prashant Malik** to solve the **Inbox Search** problem — storing reverse indices of billions of messages per day while keeping search latencies low across geographically distributed data centers. Facebook open-sourced it in July 2008.

The paper describes Cassandra as a **"BigTable data model running on a Dynamo-like infrastructure"** — combining:
- **Consistent hashing** (from Dynamo) for partitioning
- **Column-family data model** (from Bigtable) for storage
- **Gossip protocol** for cluster membership
- **Phi Accrual Failure Detector** for detecting node failures
- **Hinted handoff** for availability during failures

This tutorial walks through each of these features using a local multi-node cluster via Docker Compose.

---

## Prerequisites

- Docker & Docker Compose installed
- `cqlsh` or access via Docker exec
- ~4 GB RAM free

---

## 1. Setting Up a Multi-Node Cluster with Docker Compose

The paper emphasizes **no single point of failure** and **decentralized architecture**. We'll simulate this with a 3-node cluster.

**`docker-compose.yml`**

```yaml
version: "3.8"

networks:
  cassandra-net:
    driver: bridge

services:
  cassandra-seed:
    image: cassandra:4.1
    container_name: cassandra-seed
    hostname: cassandra-seed
    networks:
      - cassandra-net
    environment:
      - CASSANDRA_CLUSTER_NAME=FacebookDemoCluster
      - CASSANDRA_DC=datacenter1
      - CASSANDRA_RACK=rack1
      - CASSANDRA_ENDPOINT_SNITCH=GossipingPropertyFileSnitch
      - CASSANDRA_NUM_TOKENS=16
      - MAX_HEAP_SIZE=512M
      - HEAP_NEWSIZE=100M
    ports:
      - "9042:9042"
    healthcheck:
      test: ["CMD-SHELL", "nodetool status | grep -c '^UN' | grep -qx '1'"]
      interval: 30s
      timeout: 15s
      retries: 20
      start_period: 90s

  cassandra-node1:
    image: cassandra:4.1
    container_name: cassandra-node1
    hostname: cassandra-node1
    networks:
      - cassandra-net
    environment:
      - CASSANDRA_CLUSTER_NAME=FacebookDemoCluster
      - CASSANDRA_DC=datacenter1
      - CASSANDRA_RACK=rack2
      - CASSANDRA_SEEDS=cassandra-seed
      - CASSANDRA_ENDPOINT_SNITCH=GossipingPropertyFileSnitch
      - CASSANDRA_NUM_TOKENS=16
      - MAX_HEAP_SIZE=512M
      - HEAP_NEWSIZE=100M
    depends_on:
      cassandra-seed:
        condition: service_healthy
    healthcheck:
      test: ["CMD-SHELL", "nodetool status | grep -c '^UN' | grep -qx '2'"]
      interval: 30s
      timeout: 15s
      retries: 20
      start_period: 90s

  cassandra-node2:
    image: cassandra:4.1
    container_name: cassandra-node2
    hostname: cassandra-node2
    networks:
      - cassandra-net
    environment:
      - CASSANDRA_CLUSTER_NAME=FacebookDemoCluster
      - CASSANDRA_DC=datacenter1
      - CASSANDRA_RACK=rack3
      - CASSANDRA_SEEDS=cassandra-seed
      - CASSANDRA_ENDPOINT_SNITCH=GossipingPropertyFileSnitch
      - CASSANDRA_NUM_TOKENS=16
      - MAX_HEAP_SIZE=512M
      - HEAP_NEWSIZE=100M
    depends_on:
      cassandra-node1:
        condition: service_healthy
```

**Start the cluster:**

```bash
docker compose up -d
# Wait ~2 minutes for all nodes to join
docker compose logs -f cassandra-seed
```

---

## 2. Feature 1 — Gossip Protocol (Cluster Membership)

> *"Cassandra uses a gossip-based protocol for cluster membership. Each node contacts one to three other nodes to exchange state information, making the cluster self-organizing."*

**Check cluster ring topology:**

```bash
docker exec cassandra-seed nodetool status
```

Expected output showing all 3 nodes as `UN` (Up/Normal):

```
Datacenter: datacenter1
=======================
Status=Up/Down
|/ State=Normal/Leaving/Joining/Moving
--  Address     Load       Tokens  Owns   Host ID   Rack
UN  172.x.x.1   75 KiB     16      ?      ...       rack1
UN  172.x.x.2   65 KiB     16      ?      ...       rack2
UN  172.x.x.3   65 KiB     16      ?      ...       rack3
```

The gossip protocol allows each node to know about every other node without a central coordinator.

---

## 3. Feature 2 — Consistent Hashing & Partitioning

> *"Cassandra partitions data across the cluster using consistent hashing. The output range of the hash function is treated as a fixed circular space or 'ring'; the largest hash value wraps around to the smallest."*

Connect to the cluster:

```bash
docker exec -it cassandra-seed cqlsh
```

**Create a keyspace** (the paper's equivalent of a "table"):

```sql
CREATE KEYSPACE inbox_search
WITH replication = {
  'class': 'NetworkTopologyStrategy',
  'datacenter1': 3
};

USE inbox_search;
```

`replication factor = 3` means every partition is stored on 3 nodes — data is spread across the ring, and each node owns a portion of the token range.

**See how tokens are distributed:**

```bash
docker exec cassandra-seed nodetool ring
```

---

## 4. Feature 3 — Column-Family Data Model (Bigtable-inspired)

> *"Every row is identified by a unique key. A table is made up of one or more column families. Each column family can contain columns or super-columns."*

The original paper's data model maps naturally to CQL's wide-row tables. We'll model Facebook's **Inbox Search** use case — storing reverse indices of messages.

```sql
-- Column Family: messages per user
CREATE TABLE user_messages (
    user_id     UUID,
    message_id  TIMEUUID,
    sender      TEXT,
    body        TEXT,
    subject     TEXT,
    PRIMARY KEY (user_id, message_id)
) WITH CLUSTERING ORDER BY (message_id DESC);

-- Column Family: reverse index (term -> messages)
-- This is exactly what Facebook built Cassandra for
CREATE TABLE term_index (
    user_id     UUID,
    term        TEXT,
    message_id  TIMEUUID,
    PRIMARY KEY ((user_id, term), message_id)
) WITH CLUSTERING ORDER BY (message_id DESC);
```

**Insert sample data:**

```sql
INSERT INTO user_messages (user_id, message_id, sender, body, subject)
VALUES (
    uuid(),
    now(),
    'alice@facebook.com',
    'Hey, let us meet for the project update tomorrow.',
    'Project Update'
);

-- Insert more messages
INSERT INTO user_messages (user_id, message_id, sender, body, subject)
VALUES (
    11111111-1111-1111-1111-111111111111,
    now(),
    'bob@facebook.com',
    'Did you see the new product launch announcement?',
    'Product Launch'
);

INSERT INTO user_messages (user_id, message_id, sender, body, subject)
VALUES (
    11111111-1111-1111-1111-111111111111,
    now(),
    'carol@facebook.com',
    'Quick reminder about the project meeting at 3pm.',
    'Project Meeting'
);

-- Build reverse index for term search
INSERT INTO term_index (user_id, term, message_id)
VALUES (11111111-1111-1111-1111-111111111111, 'project', now());

INSERT INTO term_index (user_id, term, message_id)
VALUES (11111111-1111-1111-1111-111111111111, 'launch', now());
```

**Term search query (Facebook's Inbox Search pattern):**

```sql
-- Find all messages containing "project" for a user
SELECT * FROM term_index
WHERE user_id = 11111111-1111-1111-1111-111111111111
  AND term = 'project';
```

---

## 5. Feature 4 — Replication & Tunable Consistency

> *"Cassandra is configured such that each row is replicated across multiple data centers... This allows Cassandra to handle entire data center failures."*

The paper's key insight: consistency is **tunable per operation**, not fixed globally.

```sql
-- QUORUM write: majority of replicas must acknowledge
-- Strongest consistency while tolerating 1 node failure
CONSISTENCY QUORUM;

INSERT INTO user_messages (user_id, message_id, sender, body, subject)
VALUES (
    22222222-2222-2222-2222-222222222222,
    now(),
    'dave@facebook.com',
    'Important: system outage scheduled for Sunday.',
    'Outage Notice'
);

-- ONE write: only 1 replica must acknowledge
-- Highest availability / lowest latency (Facebook's default for inbox writes)
CONSISTENCY ONE;

INSERT INTO user_messages (user_id, message_id, sender, body, subject)
VALUES (
    33333333-3333-3333-3333-333333333333,
    now(),
    'eve@facebook.com',
    'Casual message, eventual consistency is fine here.',
    'Casual'
);

-- Read with LOCAL_QUORUM for strong consistency within datacenter
CONSISTENCY LOCAL_QUORUM;
SELECT * FROM user_messages
WHERE user_id = 22222222-2222-2222-2222-222222222222;
```

**Consistency levels summary:**

| Level | Writes acknowledged by | Use case |
|---|---|---|
| `ONE` | 1 replica | High write throughput (Facebook default) |
| `QUORUM` | Majority | Balanced consistency & availability |
| `ALL` | All replicas | Strictest — not used at Facebook scale |
| `LOCAL_QUORUM` | Majority in local DC | Multi-DC deployments |

---

## 6. Feature 5 — Hinted Handoff (High Availability During Failures)

> *"To maintain data integrity during node outages, Cassandra uses a 'hinted handoff' mechanism. When a replica is unavailable, the coordinator stores a hint and delivers the write once the node recovers."*

**Simulate a node failure:**

```bash
# Stop node1 (simulating failure)
docker stop cassandra-node1

# Writes still succeed with consistency ONE or QUORUM
# because only 2 of 3 nodes needed for quorum
docker exec -it cassandra-seed cqlsh -e "
USE inbox_search;
CONSISTENCY QUORUM;
INSERT INTO user_messages (user_id, message_id, sender, body, subject)
VALUES (44444444-4444-4444-4444-444444444444, now(), 'system', 'Written during node1 failure', 'Hinted Handoff Test');
"

# Bring node1 back — it will receive the hinted writes
docker start cassandra-node1

# Wait for it to rejoin, then verify it received the data
sleep 30
docker exec cassandra-seed nodetool status
```

---

## 7. Feature 6 — Write Path: Commit Log + Memtable + SSTable

> *"A write first goes to a commit log for durability, then to an in-memory Memtable. When the Memtable exceeds a threshold, it is flushed to disk as an immutable SSTable. This gives Cassandra sequential write performance."*

This is why Cassandra excels at write-heavy workloads (billions/day at Facebook).

**Observe the write path:**

```bash
# Check memtable/SSTable stats for our table
docker exec cassandra-seed nodetool tablestats inbox_search.user_messages

# Manually flush memtable to SSTable (normally automatic)
docker exec cassandra-seed nodetool flush inbox_search

# View SSTables on disk
docker exec cassandra-seed bash -c \
  "find /var/lib/cassandra/data/inbox_search -name '*.db' | head -20"
```

**Benchmark write throughput** (demonstrates why Facebook chose Cassandra over MySQL):

```bash
# Run the built-in stress tool
docker exec cassandra-seed cassandra-stress write n=100000 \
  -rate threads=50 \
  -node cassandra-seed

# Compare read performance
docker exec cassandra-seed cassandra-stress read n=50000 \
  -rate threads=50 \
  -node cassandra-seed
```

---

## 8. Feature 7 — Phi Accrual Failure Detection

> *"Instead of a boolean up/down signal, the failure detector emits a suspicion value Φ that is dynamically adjusted to reflect network conditions. Cassandra found the Exponential Distribution better than Gaussian for modeling inter-arrival times."*

**Monitor failure detection in real time:**

```bash
# Watch gossip state and failure detection
docker exec cassandra-seed nodetool gossipinfo

# See which nodes are considered up/down
docker exec cassandra-seed nodetool tpstats

# Stop a node and watch phi values escalate
docker stop cassandra-node2
sleep 15
docker exec cassandra-seed nodetool status
# node2 should appear as DN (Down/Normal)
docker start cassandra-node2
```

---

## 9. Feature 8 — Read Repair & Eventual Consistency

> *"Cassandra uses read repair to keep replicas consistent. When a read detects stale data on a replica, it triggers a background repair."*

```sql
USE inbox_search;

-- Enable tracing to see read repair in action
TRACING ON;

SELECT * FROM user_messages
WHERE user_id = 11111111-1111-1111-1111-111111111111;

-- The trace output will show replica coordination,
-- digest mismatches, and any read repair activity
TRACING OFF;
```

**Manual repair (for scheduled anti-entropy):**

```bash
docker exec cassandra-seed nodetool repair inbox_search
```

---

## 10. Inspecting the Ring & Token Distribution

```bash
# See full token assignments
docker exec cassandra-seed nodetool describering inbox_search

# Check data distribution across nodes
docker exec cassandra-seed nodetool tablestats inbox_search

# See compaction activity (SSTable merging)
docker exec cassandra-seed nodetool compactionstats
```

---

## 11. Cleanup

```bash
docker compose down -v
```

---

## Summary: Paper Features Demonstrated

| Paper Feature | Docker Demo |
|---|---|
| Gossip protocol | `nodetool status` shows self-organizing cluster |
| Consistent hashing / ring | `nodetool ring` shows token distribution |
| Column-family data model | `user_messages` and `term_index` tables |
| Tunable consistency | `CONSISTENCY ONE / QUORUM / LOCAL_QUORUM` |
| Hinted handoff | Stop a node, write, restart, verify data |
| Commit log + Memtable + SSTable | `nodetool flush` + SSTable files on disk |
| Phi Accrual Failure Detector | `nodetool gossipinfo` + node stop/start |
| Read repair | `TRACING ON` + `nodetool repair` |
| No single point of failure | Cluster stays up with 1 of 3 nodes down |

---

## References

- Lakshman, A. & Malik, P. (2009). *Cassandra — A Decentralized Structured Storage System*. LADIS 2009, ACM SIGOPS.
- Facebook Engineering Blog: *Cassandra – A Structured Storage System on a P2P Network* (2008)
- Apache Cassandra Documentation: https://cassandra.apache.org/doc/latest/
