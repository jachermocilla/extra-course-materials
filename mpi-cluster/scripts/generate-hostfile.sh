#!/bin/bash
# Generates an MPI hostfile by probing worker nodes over SSH.

WORKER_COUNT="${MPI_WORKER_COUNT:-3}"
WORKER_PREFIX="${MPI_WORKER_PREFIX:-worker}"
HOSTFILE="/home/mpiuser/mpi_work/hostfile"
SLOTS=3   # CPU slots per node — adjust to match your container resources

echo "Waiting for worker nodes..."

# Always include master itself
{
  echo "master slots=${SLOTS}"
} > "$HOSTFILE"

for i in $(seq 1 "$WORKER_COUNT"); do
  HOST="${WORKER_PREFIX}${i}"
  echo -n "  Pinging ${HOST}... "

  # Retry up to 30 s
  for attempt in $(seq 1 30); do
    if ssh -o ConnectTimeout=2 -o StrictHostKeyChecking=no \
           mpiuser@"$HOST" "echo ok" &>/dev/null; then
      echo "✓"
      echo "${HOST} slots=${SLOTS}" >> "$HOSTFILE"
      break
    fi
    sleep 1
  done
done

echo ""
echo "Generated hostfile:"
cat "$HOSTFILE"

chown mpiuser:mpiuser "$HOSTFILE"
