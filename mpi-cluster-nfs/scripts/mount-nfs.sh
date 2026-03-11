#!/bin/bash
# ─────────────────────────────────────────────────────────────────────────────
# mount-nfs.sh  —  Wait for the NFS server and mount the shared export.
#
# Reads env vars set in docker-compose.yml:
#   NFS_SERVER  — hostname/IP of the NFS container  (default: nfs)
#   NFS_EXPORT  — exported path on the server       (default: /exports/mpi)
#   NFS_MOUNT   — local mount point                 (default: /home/mpiuser/mpi_work/shared)
# ─────────────────────────────────────────────────────────────────────────────

NFS_SERVER="${NFS_SERVER:-nfs}"
NFS_EXPORT="${NFS_EXPORT:-/exports/mpi}"
NFS_MOUNT="${NFS_MOUNT:-/home/mpiuser/mpi_work/shared}"
MAX_WAIT=60

echo "  NFS: waiting for ${NFS_SERVER}${NFS_EXPORT} → ${NFS_MOUNT}"

# Ensure local mount point exists
mkdir -p "${NFS_MOUNT}"
chown mpiuser:mpiuser "${NFS_MOUNT}"

# Wait until the NFS server's portmapper responds
elapsed=0
until rpcinfo -t "${NFS_SERVER}" nfs &>/dev/null; do
    if [[ "$elapsed" -ge "$MAX_WAIT" ]]; then
        echo "  NFS: ERROR — server ${NFS_SERVER} did not respond after ${MAX_WAIT}s"
        exit 1
    fi
    echo "  NFS: server not ready yet (${elapsed}/${MAX_WAIT}s) — retrying..."
    sleep 2
    elapsed=$((elapsed + 2))
done

# Mount with NFSv4, soft mount so MPI jobs don't hang forever on NFS failure
mount -t nfs4 \
    -o "rw,soft,intr,timeo=30,retrans=3" \
    "${NFS_SERVER}:${NFS_EXPORT}" \
    "${NFS_MOUNT}"

echo "  NFS: ✓ mounted ${NFS_SERVER}:${NFS_EXPORT} → ${NFS_MOUNT}"
chown mpiuser:mpiuser "${NFS_MOUNT}"
