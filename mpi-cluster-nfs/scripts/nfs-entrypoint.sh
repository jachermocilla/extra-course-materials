#!/bin/bash
set -e

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  NFS Server Starting..."
echo "  Export: /exports/mpi"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# rpcbind must be up before nfs
rpcbind || true

chown 1000:1000 /exports/mpi 
chmod 755 /exports/mpi

df -T /exports/mpi

# Export filesystems
exportfs -ra

# Start NFS kernel server
/usr/sbin/rpc.nfsd 8

# Start mountd
/usr/sbin/rpc.mountd --no-udp --no-nfs-version 2 --no-nfs-version 3

echo "✓ NFS server ready — exporting /exports/mpi"

# Keep container alive and re-export on any changes
while true; do
    exportfs -ra
    sleep 30
done
