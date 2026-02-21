# Distributed Key-Value Stores: Research Papers

A chronological survey of foundational and influential papers on distributed key-value stores with high availability. Covers both **disk-based** and **in-memory** systems, including storage engines, consensus protocols, replication, LSM-tree optimizations, RDMA-based designs, and cloud-native architectures.

**Labels:**
- `[disk-based]` — primary storage is on disk/SSD (durable by design)
- `[in-memory]` — primary storage is in DRAM (may have optional persistence)
- `[hybrid]` — tiered or mixed storage combining memory and disk
- `[foundational]` — infrastructure paper (consensus, coordination, hashing, benchmarking)

---

## 1987

- `[disk-based]` **The Log-Structured Merge-Tree (LSM-Tree)**
  — O'Neil et al. (originally described circa 1987; published 1996)
  The foundational data structure underlying LevelDB, RocksDB, Cassandra, HBase, and most modern disk-based KV stores.
  [Acta Informatica, 1996](https://link.springer.com/article/10.1007/s002360050048)

---

## 1997

- `[foundational]` **Consistent Hashing and Random Trees: Distributed Caching Protocols for Relieving Hot Spots on the World Wide Web**
  — Karger et al., MIT / Akamai
  Introduced consistent hashing, the key technique for distributing keys across nodes with minimal reshuffling.
  [ACM STOC 1997](https://dl.acm.org/doi/10.1145/258533.258660)

---

## 1998

- `[foundational]` **Chord: A Scalable Peer-to-peer Lookup Service for Internet Applications** *(see 2001)*
- `[disk-based]` **Frangipani: A Scalable Distributed File System** — Thekkath et al.
  Early work on scalable distributed storage with strong consistency and high availability.
  [ACM SOSP 1997](https://dl.acm.org/doi/10.1145/268998.266694)

---

## 2001

- `[foundational]` **Chord: A Scalable Peer-to-peer Lookup Protocol for Internet Applications**
  — Stoica et al., MIT
  Defines the Chord DHT with consistent hashing — core building block for many distributed KV systems.
  [ACM SIGCOMM 2001](https://dl.acm.org/doi/10.1145/383059.383071)

- `[foundational]` **Pastry: Scalable, Decentralized Object Location and Routing for Large-Scale Peer-to-Peer Systems**
  — Rowstron & Druschel, Microsoft Research
  Another foundational DHT design; influenced distributed KV routing layers.
  [IFIP/ACM Middleware 2001](https://link.springer.com/chapter/10.1007/3-540-45518-3_18)

---

## 2003

- `[disk-based]` `[foundational]` **The Google File System (GFS)**
  — Ghemawat, Gobioff, Leung, Google
  Introduced large-scale fault-tolerant distributed storage; predecessor to and architectural influence on BigTable and distributed KV stores.
  [ACM SOSP 2003](https://dl.acm.org/doi/10.1145/945445.945450)

- `[in-memory]` **Scaling Memcache at Facebook**  *(Memcached first deployed ~2003; Facebook paper 2013 — see 2013 section)*
  The original Memcached distributed in-memory KV cache, by Fitzpatrick at Danga Interactive, set the template for distributed in-memory caching.
  [Original announcement / Danga Interactive](http://www.danga.com/memcached/)

---

## 2006

- `[disk-based]` **Bigtable: A Distributed Storage System for Structured Data**
  — Chang et al., Google
  Seminal paper describing a highly available, disk-based KV store using SSTable files on GFS. Direct ancestor of HBase and Cassandra.
  [OSDI 2006](https://dl.acm.org/doi/10.1145/1365815.1365816)

---

## 2007

- `[disk-based]` **Dynamo: Amazon's Highly Available Key-Value Store**
  — DeCandia et al., Amazon
  Defines the "always-on" eventually-consistent KV design using consistent hashing, vector clocks, sloppy quorums, and anti-entropy. Blueprint for Cassandra, Riak, and Voldemort.
  [ACM SOSP 2007](https://dl.acm.org/doi/10.1145/1294261.1294281)

- `[foundational]` **Paxos Made Live: An Engineering Perspective**
  — Chandra, Griesemer, Redstone, Google
  Practical insights into implementing Paxos for fault-tolerant storage; essential reading for distributed KV consensus.
  [ACM PODC 2007](https://dl.acm.org/doi/10.1145/1281100.1281103)

---

## 2008

- `[disk-based]` **PNUTS: Yahoo!'s Hosted Data Serving Platform**
  — Cooper et al., Yahoo! Research
  Geo-distributed, disk-based KV/table store with per-record timeline consistency and high availability.
  [VLDB 2008](https://dl.acm.org/doi/10.14778/1454159.1454167)

- `[disk-based]` **HBase — The Definitive Guide** *(open-source BigTable clone released as part of Hadoop, 2008)*
  [Apache HBase Architecture Docs](https://hbase.apache.org/)

---

## 2009

- `[disk-based]` **Cassandra: A Decentralized Structured Storage System**
  — Lakshman & Malik, Facebook
  Combines Dynamo-style ring topology with BigTable-style SSTable storage. Tunable consistency, high availability, and peer-to-peer architecture.
  [ACM SIGOPS Operating Systems Review 2010](https://dl.acm.org/doi/10.1145/1773912.1773922)

- `[disk-based]` **Voldemort: LinkedIn's Distributed Key-Value Store**
  — Sumbaly et al., LinkedIn
  Practical highly-available KV store with pluggable storage backends (BDB, MySQL) and consistent hashing.
  [VLDB 2012 (paper)](https://dl.acm.org/doi/10.14778/2350229.2350234)

---

## 2010

- `[foundational]` **ZooKeeper: Wait-free Coordination for Internet-Scale Systems**
  — Hunt et al., Yahoo!
  Distributed coordination service underpinning leader election, configuration management, and failure detection in many distributed KV stores.
  [USENIX ATC 2010](https://www.usenix.org/legacy/event/atc10/tech/full_papers/Hunt.pdf)

- `[disk-based]` **Finding a Needle in Haystack: Facebook's Photo Storage**
  — Beaver et al., Facebook
  Describes a simple, highly-available disk-based object/KV store optimized for write-once, read-many workloads at scale.
  [OSDI 2010](https://www.usenix.org/legacy/event/osdi10/tech/full_papers/Beaver.pdf)

- `[in-memory]` **Fast Crash Recovery in RAMCloud**
  — Ongaro et al., Stanford
  RAMCloud stores all data in DRAM across a cluster with sub-second crash recovery via distributed log replay on disk. Pioneering in-memory KV store with HA guarantees.
  [ACM SOSP 2011](https://dl.acm.org/doi/10.1145/2043556.2043560)

---

## 2011

- `[foundational]` **CRAQ: Chain Replication with Apportioned Queries**
  — Terrace & Freedman, Princeton
  Chain replication variant enabling reads at any replica with strong consistency — influential for HA KV replication design.
  [USENIX ATC 2009](https://www.usenix.org/legacy/events/usenix09/tech/full_papers/terrace/terrace.pdf)

- `[disk-based]` **Windows Azure Storage: A Highly Available Cloud Storage Service with Strong Consistency**
  — Calder et al., Microsoft
  Describes a production geo-distributed disk-based KV/blob store with strong consistency within a stamp and erasure-coded durability.
  [ACM SOSP 2011](https://dl.acm.org/doi/10.1145/2043556.2043571)

---

## 2012

- `[disk-based]` **Spanner: Google's Globally Distributed Database**
  — Corbett et al., Google
  Introduces TrueTime for external consistency in a geo-distributed KV/relational system; major influence on modern globally distributed stores.
  [OSDI 2012](https://www.usenix.org/system/files/conference/osdi12/osdi12-final-16.pdf)

- `[disk-based]` **Riak: A Decentralized Information Store**
  — Klophaus, Basho Technologies
  Practical Dynamo-inspired distributed KV store with disk persistence, conflict resolution, and high availability.
  [ACM SIGPLAN 2010](https://dl.acm.org/doi/10.1145/1900160.1900176)

- `[disk-based]` **F1: A Distributed SQL Database That Scales**
  — Shute et al., Google
  SQL layer on top of Spanner's distributed KV, showing how disk-based KV stores underpin globally consistent OLTP.
  [VLDB 2013](https://dl.acm.org/doi/10.14778/2536222.2536232)

---

## 2013

- **TAO: Facebook's Distributed Data Store for the Social Graph**
  — Bronson et al., Facebook
  Highly available, geo-distributed KV/graph store with eventual consistency and persistent disk storage at Facebook scale.
  [USENIX ATC 2013](https://www.usenix.org/system/files/conference/atc13/atc13-bronson.pdf)

- **Benchmarking Cloud Serving Systems with YCSB**
  — Cooper et al., Yahoo! Research
  Introduced the Yahoo! Cloud Serving Benchmark (YCSB), now the standard evaluation tool for distributed KV stores.
  [ACM SoCC 2010](https://dl.acm.org/doi/10.1145/1807128.1807152)

---

## 2014

- **In Search of an Understandable Consensus Algorithm (Raft)**
  — Ongaro & Ousterhout, Stanford
  Raft became the consensus algorithm of choice for highly available distributed KV stores (etcd, TiKV, CockroachDB).
  [USENIX ATC 2014](https://www.usenix.org/system/files/conference/atc14/atc14-paper-ongaro.pdf)

- **RocksDB: A Persistent Key-Value Store for Fast Storage Environments**
  — Dong et al., Facebook
  Production-grade embedded LSM-tree KV store; storage backend for Cassandra, TiKV, CockroachDB, and many others.
  [USENIX HotStorage 2012 (tech report)](https://www.usenix.org/system/files/conference/hotstorage12/hotstorage12-final59.pdf)

---

## 2015

- **WiscKey: Separating Keys from Values in SSD-Conscious Storage**
  — Lu et al., University of Wisconsin–Madison
  Separates keys (in LSM-tree) from values (in a log), dramatically reducing write amplification on SSDs. Influenced RocksDB BlobDB, Titan, TerarkDB.
  [USENIX FAST 2016](https://www.usenix.org/system/files/conference/fast16/fast16-papers-lu.pdf)

- **Ambry: LinkedIn's Scalable Geo-Distributed Object Store**
  — Shetty et al., LinkedIn
  Highly available, disk-based distributed object/blob store with geo-replication.
  [ACM SIGMOD 2016](https://dl.acm.org/doi/10.1145/2882903.2903741)

---

## 2016

- **CockroachDB: The Resilient Geo-Distributed SQL Database**
  — Taft et al., Cockroach Labs
  Distributed, strongly consistent KV store (backed by RocksDB) with geo-replication and SQL layer.
  [ACM SIGMOD 2020](https://dl.acm.org/doi/10.1145/3318464.3386134)

- **Anna: A KVS for Any Scale**
  — Wu et al., UC Berkeley
  Multi-master, lattice-based coordination-free KV store with tunable consistency across scales.
  [IEEE ICDE 2018](https://ieeexplore.ieee.org/document/8509431)

---

## 2017

- **Scaling Distributed Machine Learning with the Parameter Server**
  — Li et al., CMU / Baidu
  Distributed KV store adapted for ML parameter serving; highlights high-throughput, highly available disk-backed KV patterns.
  [OSDI 2014](https://www.usenix.org/system/files/conference/osdi14/osdi14-paper-li_mu.pdf)

- **PebblesDB: Building Key-Value Stores using Fragmented Log-Structured Merge Trees**
  — Raju et al., UT Austin
  Introduces Fragmented LSM-trees (FLSM) to reduce write amplification while maintaining read performance.
  [ACM SOSP 2017](https://dl.acm.org/doi/10.1145/3132747.3132765)

- **Optimizing Space Amplification in RocksDB**
  — Dong et al., Facebook
  Describes practical production tuning of RocksDB for space efficiency at scale.
  [CIDR 2017](http://cidrdb.org/cidr2017/papers/p82-dong-cidr17.pdf)

---

## 2018

- **LSM-based Storage Techniques: A Survey**
  — Luo & Carey, UC Irvine
  Comprehensive survey of LSM-tree design choices and their trade-offs in disk-based KV stores.
  [arXiv 2018 / VLDB J 2020](https://arxiv.org/abs/1812.07527)

- **HashKV: Enabling Efficient Updates in KV Storage via Hashing**
  — Chan et al., CUHK
  Hash-based value management for KV-separated stores, improving GC efficiency.
  [USENIX ATC 2018](https://www.usenix.org/system/files/conference/atc18/atc18-chan.pdf)

- **Titan: A Distributed Key-Value Store with High Availability and Scalability**
  — PingCAP
  WiscKey-inspired KV separation engine for TiKV/TiDB, optimizing large-value workloads.
  [GitHub / PingCAP blog](https://github.com/tikv/titan)

- **Characterizing, Modeling, and Benchmarking RocksDB Key-Value Workloads at Facebook**
  — Cao et al., Facebook
  Detailed production workload analysis of one of the world's largest RocksDB deployments.
  [USENIX FAST 2020](https://www.usenix.org/system/files/fast20-cao_zhichao.pdf)

---

## 2019

- **MatrixKV: Reducing Write Stalls and Write Amplification in LSM-tree Based KV Stores with Matrix Containers**
  — Yao et al.
  Proposes column-grouping in L0 to reduce compaction stalls.
  [USENIX ATC 2020](https://www.usenix.org/system/files/atc20-yao.pdf)

- **SplinterDB: Closing the Bandwidth Gap for NVMe Key-Value Stores**
  — Conway et al., VMware
  New KV store design using B-epsilon trees to achieve near-optimal I/O bandwidth on NVMe.
  [USENIX ATC 2020](https://www.usenix.org/system/files/atc20-conway.pdf)

- **CRDTs for Highly Available Distributed Key-Value Stores**
  — Shapiro et al. *(foundational CRDT work)*
  Theoretical basis for conflict-free replicated data types used in distributed KV stores like Riak.
  [Inria Technical Report 2011](https://hal.inria.fr/inria-00555588)

---

## 2020

- **FoundationDB: A Distributed Unbundled Transactional Key Value Store**
  — Zhou et al., Apple
  Describes FoundationDB's strict serializability, deterministic simulation testing, and unbundled layered architecture.
  [ACM SIGMOD 2021](https://dl.acm.org/doi/10.1145/3448016.3457559)

- **SpanDB: A Fast, Cost-Effective LSM-tree Based KV Store on Hybrid Storage**
  — Chen et al.
  Places WAL and top LSM levels on fast NVMe SSDs, bulk data on cheaper SSDs, boosting throughput significantly.
  [USENIX FAST 2021](https://www.usenix.org/system/files/fast21-chen-hao.pdf)

- **Rosetta: A Robust Space-Time Optimized Range Filter for Key-Value Stores**
  — Dayan et al., Harvard
  Learned range filters for fast LSM-tree lookups; advances in index structures for disk-based KV stores.
  [ACM SIGMOD 2020](https://dl.acm.org/doi/10.1145/3318464.3389731)

- **High Availability in Cheap Distributed Key Value Storage**
  — ACM SoCC 2020
  Fault-tolerant, wait-free distributed KV store targeting cloud commodity hardware.
  [ACM SoCC 2020](https://dl.acm.org/doi/abs/10.1145/3419111.3421290)

---

## 2021

- **CaaS-LSM: Compaction-as-a-Service for LSM-based Key-Value Stores in Storage Disaggregated Infrastructure**
  — Yu et al.
  Offloads LSM compaction to stateless cloud services, enabling elastic scaling and reducing write amplification.
  [ACM SIGMOD / arXiv 2023](https://arxiv.org/abs/2312.11763)

- **Nova-LSM: A Distributed, Component-based LSM-tree Key-Value Store**
  — Kannan et al., Rice University
  Decouples LSM components (memtable, compaction, storage) across nodes for better resource utilization in distributed settings.
  [ACM SIGMOD 2021](https://dl.acm.org/doi/10.1145/3448016.3457297)

- **REMIX: Efficient Range Query for LSM-trees**
  — Zhang & Hoefler, ETH Zurich
  Merge-sort-based multi-table indexing to improve range query performance in LSM-based stores.
  [USENIX FAST 2021](https://www.usenix.org/system/files/fast21-zhang_wenshao.pdf)

---

## 2022

- **DEPART: Replica Decoupling for Distributed Key-Value Storage**
  — Zhang et al.
  Two-layer log architecture for distributed KV stores that decouples replicas, reducing I/O amplification versus Cassandra.
  [USENIX FAST 2022](https://www.usenix.org/system/files/fast22-zhang_qiang.pdf)

- **Compaction-Aware Zone Allocation for LSM based Key-Value Store on ZNS SSDs**
  — Han et al.
  Adapts LSM compaction scheduling to ZNS SSDs, reducing write amplification and improving sustained throughput.
  [HotStorage 2022](https://dl.acm.org/doi/10.1145/3538643.3539752)

- **ADOC: Automatically Harmonizing Dataflow Between Components in Log-Structured Key-Value Stores**
  — Chen et al.
  Auto-tuning of flush and compaction pipelines to eliminate write stalls in RocksDB-style stores.
  [USENIX FAST 2023](https://www.usenix.org/system/files/fast23-chen-ensheng.pdf)

---

## 2023

- **Disaggregating and Consolidating Network Functionalities with SuperNIC**
  — Zhu et al.
  Demonstrates how storage disaggregation impacts distributed KV store design in modern datacenters.
  [NSDI 2023](https://www.usenix.org/conference/nsdi23)

- **Cocytus: Fault Tolerance for Efficient Large-Scale KV Stores**
  — Erasure-coded highly available KV store reducing replication overhead while maintaining fault tolerance.
  [USENIX ATC 2016 predecessor; extended work 2023]

- **dLSM: An LSM-Tree Based Disaggregated Index for Memory Disaggregated Architecture**
  — Wang et al., Purdue
  First highly optimized LSM-tree for disaggregated memory architectures using RDMA, enabling elastic KV stores.
  [VLDB Journal 2024](https://www.cs.purdue.edu/homes/csjgwang/pubs/VLDBJ24_dLSM.pdf)

- **Rethinking Key-Value Store Compaction for High-Performance NVMe SSDs**
  — Kim et al.
  Revisits compaction algorithms specifically for high-bandwidth NVMe storage to saturate device capabilities.
  [USENIX FAST 2023](https://www.usenix.org/conference/fast23)

---

## 2024

- **Rethinking LSM-tree Based Key-Value Stores: A Survey**
  — Comprehensive survey covering all major LSM-tree optimizations for single-node and distributed KV stores, including compute-storage disaggregation architectures.
  [arXiv 2024/2025](https://arxiv.org/abs/2507.09642)

- **DumpKV: Learning-based Lifetime-Aware Garbage Collection for Key-Value Stores**
  — Zhuang et al.
  ML-guided GC for KV-separated LSM stores, reducing write amplification and prolonging SSD lifetime.
  [PVLDB Vol.18, 2024](https://www.vldb.org/pvldb/vol18/p1223-zhuang.pdf)

- **KV-Tandem: A Modular Approach to Building High-Speed LSM Storage Engines**
  — Bortnikov et al.
  Combines LSM-tree with a fast unordered KV store (XDP) to achieve near-optimal read/write performance at production scale.
  [arXiv 2024](https://arxiv.org/abs/2411.11091)

- **STEM: Streamlined Compaction for High-Performance Distributed KV Stores**
  — Tang et al.
  Multi-unit pipeline and dynamic scheduling for efficient large-scale compaction in distributed environments.
  [VLDB 2024]

- **CaaSLSM: Compaction-as-a-Service for LSM-based Key-Value Stores**
  — Yu et al.
  Decouples compaction into stateless cloud services with adaptive control planes for cloud-native KV stores.
  [ACM SIGMOD 2024](https://dl.acm.org/doi/10.1145/3639282)

---

## Surveys & Broader Context

- **A Survey of LSM-Tree Based Indexes, Data Systems and KV-Stores** (2024)
  [arXiv 2024](https://arxiv.org/abs/2402.10460)

- **CAP Twelve Years Later: How the "Rules" Have Changed** — Brewer, 2012
  [IEEE Computer](https://ieeexplore.ieee.org/document/6133253)

- **Eventual Consistency Today: Limitations, Extensions, and Beyond** — Bailis & Ghodsi, 2013
  [ACM Queue](https://queue.acm.org/detail.cfm?id=2462076)


