#!/bin/bash
set -e

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  OpenMPI Master Node Starting..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Start SSH daemon
/usr/sbin/sshd

# Wait for workers to be reachable and generate hostfile
/usr/local/bin/generate-hostfile.sh

echo ""
echo "✓ Cluster ready. Hostfile written to ~/mpi_work/hostfile"
echo ""
echo "  Quick start:"
echo "    su - mpiuser"
echo "    cd mpi_work"
echo "    mpirun --hostfile hostfile -np 4 ./examples/hello_world"
echo ""

exec "$@"
