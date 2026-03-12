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
    ├── mpi_bcast.c            # MPI Broadcast
    └── ring_communication.c   # Token-passing ring topology
```

## Quick Start

```bash
# 1. Build and start the cluster
docker compose up --build 

# 2. SSH into the master node
ssh -p 2222 mpiuser@localhost #password is 'mpiuser'

# 3. Go to the shared folder (mpi-shared in local)
cd mpi_work/shared


# 4. Compile and run examples

mpicc hello_world.c -o hello_world.elf
mpicc pi_calculation.c -o pi_calculation.elf
mpicc ring_communication.c -o ring_communication.elf
mpicc mpi_broadcast.c -o hello_world.elf

mpirun --hostfile ../hostfile -np 8 ./hello_world.elf
mpirun --hostfile ../hostfile -np 8 ./pi_calculation.elf
mpirun --hostfile ../hostfile -np 8 ./ring_communication.elf
mpirun --hostfile ../hostfile -np 8 ./mpi_broadcast.elf

```

## Teardown

```bash
docker compose down -v   # -v removes named volumes (shared data + SSH keys)
`
`
