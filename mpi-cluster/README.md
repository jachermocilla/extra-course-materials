# OpenMPI Docker Cluster

A ready-to-run MPI cluster with **1 master** and **3 workers**, all connected over a private Docker bridge network.

## File Layout

```
mpi-cluster/
├── docker-compose.yml       # Cluster definition
├── Dockerfile.base          # Shared base image (Ubuntu 22.04 + OpenMPI)
├── Dockerfile.master        # Master node image
├── Dockerfile.worker        # Worker node image
├── scripts/
│   ├── master-entrypoint.sh   # Starts sshd + generates hostfile
│   ├── generate-hostfile.sh   # Probes workers and writes /hostfile
│   └── worker-entrypoint.sh   # Starts sshd and waits for jobs
└── examples/
    ├── hello_world.c          # Basic rank/hostname print
    ├── pi_calculation.c       # Distributed π via numerical integration
    └── ring_communication.c   # Token-passing ring topology
```

## Quick Start

```bash
# 1. Build and start the cluster
docker compose up --build -d

# 2. Shell into the master node
docker compose exec master bash

# 3. Switch to mpiuser and run a job
su - mpiuser
cd mpi_work

mpirun --hostfile hostfile -np 2 ./examples/hello_world
mpirun --hostfile hostfile -np 2 ./examples/pi_calculation
mpirun --hostfile hostfile -np 2 ./examples/ring_communication
```

## Scaling Workers

Add more worker services to `docker-compose.yml` and bump `MPI_WORKER_COUNT` on the master:

```yaml
# docker-compose.yml — add a fourth worker
worker4:
   build:
      context: .
      dockerfile: Dockerfile.worker
   container_name: mpi-worker4
   hostname: worker4
   networks:
     mpi-cluster:
       ipv4_address: 192.137.125.14
   volumes:
     - mpi-shared:/home/mpiuser/mpi_work/shared
     - ssh-keys:/home/mpiuser/.ssh

# master service environment:
  - MPI_WORKER_COUNT=4
```

Then rebuild: `docker compose up --build -d`

## Adjusting CPU Slots

Edit `SLOTS` in `scripts/generate-hostfile.sh` to match the number of cores you want each node to contribute (default: 2).

## SSH Access from Host

```bash
ssh -p 2222 mpiuser@localhost   # password: mpiuser
```

## Teardown

```bash
docker compose down -v   # -v removes named volumes (shared data + SSH keys)
`
`
