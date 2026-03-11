#!/bin/bash
set -e

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  OpenMPI Worker Node Starting: $(hostname)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Mount the NFS share (requires SYS_ADMIN cap in docker-compose.yml)
/usr/local/bin/mount-nfs.sh

# Start SSH daemon and keep container alive
/usr/sbin/sshd -D
