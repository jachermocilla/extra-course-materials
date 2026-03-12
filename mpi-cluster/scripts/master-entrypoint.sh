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
echo "  Quick start. Open a new terminal:"
echo "    ssh -p 2222 mpiuser@localhost #password is 'mpiuser'"
echo "    cd mpi_work/shared"
echo "    mpicc hello_world.c -o hello_world.elf"
echo "    mpirun --hostfile ../hostfile -np 8 ./hello_world.elf"
echo ""

exec "$@"
