#!/bin/bash
# ─────────────────────────────────────────────────────────────────────────────
# generate-hostfile.sh  —  Probe worker nodes over SSH and write the MPI hostfile.
#
# Env vars (set in docker-compose.yml on the master service):
#   MPI_WORKER_COUNT   — number of worker containers  (default: 3)
#   MPI_WORKER_PREFIX  — hostname prefix              (default: worker)
#   MPI_SLOTS          — CPU slots per node           (default: 2)
# ─────────────────────────────────────────────────────────────────────────────

WORKER_COUNT="${MPI_WORKER_COUNT:-3}"
WORKER_PREFIX="${MPI_WORKER_PREFIX:-worker}"
SLOTS="${MPI_SLOTS:-2}"
HOSTFILE="/home/mpiuser/mpi_work/hostfile"

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Generating MPI hostfile"
echo "  Workers: ${WORKER_COUNT}  Prefix: ${WORKER_PREFIX}  Slots/node: ${SLOTS}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Always include the master itself
echo "master slots=${SLOTS}" > "$HOSTFILE"

for i in $(seq 1 "$WORKER_COUNT"); do
    HOST="${WORKER_PREFIX}${i}"
    echo -n "  Probing ${HOST}... "

    reachable=false
    for attempt in $(seq 1 30); do
        if ssh -o ConnectTimeout=2 \
               -o StrictHostKeyChecking=no \
               -o UserKnownHostsFile=/dev/null \
               mpiuser@"${HOST}" "echo ok" &>/dev/null; then
            reachable=true
            break
        fi
        sleep 1
    done

    if $reachable; then
        echo "✓"
        echo "${HOST} slots=${SLOTS}" >> "$HOSTFILE"
    else
        echo "✗  (unreachable after 30 s — skipping)"
    fi
done

echo ""
echo "  Hostfile written to ${HOSTFILE}:"
cat "$HOSTFILE"
echo ""

chown mpiuser:mpiuser "$HOSTFILE"

