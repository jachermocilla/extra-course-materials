# OpenMPI Docker Cluster with NFS Shared Storage

A self-contained MPI cluster where a dedicated **NFS server container** exports a shared filesystem mounted by every master and worker node. Compiled binaries, input data, output results, and the MPI hostfile all live on the NFS share — write once, run everywhere.

## Topology

```
172.20.0.0/24 (bridge network)

┌─────────────────────────────────────────────────────┐
│                                                       │
│  ┌──────────────────┐   NFSv4 export                 │
│  │  mpi-nfs         │ ──────────────────────────┐    │
│  │  172.20.0.5      │                           │    │
│  │  /exports/mpi    │                           ▼    │
│  └──────────────────┘        /home/mpiuser/mpi_work/shared
│                                       ▲    ▲    ▲    │
│  ┌──────────────────┐                 │    │    │    │
│  │  mpi-master      │ ────────────────┘    │    │    │
│  │  172.20.0.10     │  SSH + MPI           │    │    │
│  └──────────────────┘ ◄──────────────────► │    │    │
│  ┌───────────┐  ┌───────────┐  ┌──────────┐│    │    │
│  │ mpi-worker1│  │mpi-worker2│  │mpi-worker3││   │    │
│  │ 172.20.0.11│  │172.20.0.12│  │172.20.0.13││   │    │
│  └────────────┘  └───────────┘  └──────────┘│   │    │
│       NFS mount ────────────────────────────┘   │    │
│       NFS mount ────────────────────────────────┘    │
└─────────────────────────────────────────────────────┘
```

## File Layout

```
mpi-cluster/
├── docker-compose.yml            # Cluster definition
├── Dockerfile.base               # Shared base (Ubuntu 22.04 + OpenMPI + nfs-common)
├── Dockerfile.nfs                # NFS server node
├── Dockerfile.master             # MPI master node
├── Dockerfile.worker             # MPI worker node
├── scripts/
│   ├── nfs-entrypoint.sh         # Starts rpcbind + nfsd + mountd
│   ├── mount-nfs.sh              # Waits for NFS server and mounts the share
│   ├── master-entrypoint.sh      # Mounts NFS, copies binaries, starts sshd
│   ├── generate-hostfile.sh      # SSH-probes workers → writes hostfile
│   └── worker-entrypoint.sh      # Mounts NFS, starts sshd
└── examples/
    ├── hello_world.c
    ├── pi_calculation.c
    └── ring_communication.c
```

## Quick Start

```bash
# Build and start the full cluster (nfs → workers → master)
docker compose up --build -d

# Watch startup logs
docker compose logs -f

# Shell into the master
docker compose exec master bash
su - mpiuser && cd mpi_work

# Run jobs — binaries are on the NFS share so all nodes can execute them
mpirun --hostfile hostfile -np 8 ./examples/hello_world
mpirun --hostfile hostfile -np 8 ./examples/pi_calculation
mpirun --hostfile hostfile -np 8 ./examples/ring_communication
```

## How the NFS Share Works

1. The `nfs` container exports `/exports/mpi` via NFSv4 to the whole subnet
2. On startup, master and every worker call `mount-nfs.sh`, which:
   - Polls `rpcinfo -t nfs <NFS_SERVER>` until the server is ready
   - Mounts `nfs:/exports/mpi` → `/home/mpiuser/mpi_work/shared` (NFSv4, soft mount)
3. The master copies its compiled MPI binaries into the share
4. Workers find the same binaries at the same path — `mpirun` can spawn processes on any node without manual binary distribution

## Using the Shared Directory

```bash
# On master — anything written here is instantly visible on all workers
ls /home/mpiuser/mpi_work/shared

# Write a data file on master, read it from a worker
echo "hello from master" > /home/mpiuser/mpi_work/shared/test.txt
docker compose exec worker1 cat /home/mpiuser/mpi_work/shared/test.txt
```

## Adding More Workers

Add a new service block to `docker-compose.yml` and increment `MPI_WORKER_COUNT`:

```yaml
worker4:
  image: mpi-worker:latest
  container_name: mpi-worker4
  hostname: worker4
  networks:
    mpi-cluster:
      ipv4_address: 172.20.0.14
  environment:
    - NFS_SERVER=nfs
    - NFS_EXPORT=/exports/mpi
    - NFS_MOUNT=/home/mpiuser/mpi_work/shared
  cap_add:
    - SYS_ADMIN
  depends_on:
    - nfs
```

Also update on the master service: `MPI_WORKER_COUNT=4`

## Configuration Reference

| Variable            | Default                           | Description                      |
|---------------------|-----------------------------------|----------------------------------|
| `NFS_SERVER`        | `nfs`                             | Hostname of the NFS container    |
| `NFS_EXPORT`        | `/exports/mpi`                    | Exported path on the NFS server  |
| `NFS_MOUNT`         | `/home/mpiuser/mpi_work/shared`   | Local mount point on each node   |
| `MPI_WORKER_COUNT`  | `3`                               | Number of worker containers      |
| `MPI_WORKER_PREFIX` | `worker`                          | Worker hostname prefix           |
| `MPI_SLOTS`         | `2`                               | CPU slots per node in hostfile   |

## Teardown

```bash
docker compose down        # stop containers
docker compose down -v     # also remove any named volumes
```
