#!/bin/bash
set -e

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  OpenMPI Master Node Starting..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Mount the NFS share (requires SYS_ADMIN cap in docker-compose.yml)
/usr/local/bin/mount-nfs.sh

# Copy compiled examples into the NFS share so workers can execute them
NFS_MOUNT="${NFS_MOUNT:-/home/mpiuser/mpi_work/shared}"
if [ -d /home/mpiuser/mpi_work/examples ]; then
    cp -u /home/mpiuser/mpi_work/examples/* "${NFS_MOUNT}/" 2>/dev/null || true
    chown mpiuser:mpiuser "${NFS_MOUNT}"/*  2>/dev/null || true
    echo "  Binaries copied to NFS share."
fi

# Start SSH daemon
/usr/sbin/sshd

# Generate MPI hostfile
/usr/local/bin/generate-hostfile.sh

echo ""
echo "✓ Cluster ready."
echo ""
echo "  Quick start:"
echo "    su - mpiuser && cd mpi_work"
echo "    mpirun --hostfile hostfile -np 8 ./examples/hello_world"
echo ""
echo "  NFS shared directory: ${NFS_MOUNT}"
echo "  (all nodes read/write this path)"
echo ""

exec "$@"
