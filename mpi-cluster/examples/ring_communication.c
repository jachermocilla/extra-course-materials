/*
 * ring_communication.c — Token passing in a ring topology
 * Demonstrates point-to-point MPI_Send / MPI_Recv.
 *
 * Compile: mpicc -o ring_communication ring_communication.c
 * Run:     mpirun --hostfile hostfile -np 8 ./ring_communication
 */

#include <stdio.h>
#include <mpi.h>

int main(int argc, char *argv[]) {
    int rank, size, token;

    MPI_Init(&argc, &argv);
    MPI_Comm_rank(MPI_COMM_WORLD, &rank);
    MPI_Comm_size(MPI_COMM_WORLD, &size);

    if (rank == 0) {
        // Originate the token
        token = 42;
        MPI_Send(&token, 1, MPI_INT, 1, 0, MPI_COMM_WORLD);
        printf("Rank 0 sent token %d to rank 1\n", token);

        // Receive it back from the last rank
        MPI_Recv(&token, 1, MPI_INT, size - 1, 0,
                 MPI_COMM_WORLD, MPI_STATUS_IGNORE);
        printf("Rank 0 received token %d back (ring complete)\n", token);
    } else {
        // Forward the token around the ring
        MPI_Recv(&token, 1, MPI_INT, rank - 1, 0,
                 MPI_COMM_WORLD, MPI_STATUS_IGNORE);
        printf("Rank %d received token %d\n", rank, token);

        token += rank;   // mutate so we can trace each hop
        MPI_Send(&token, 1, MPI_INT,
                 (rank + 1) % size, 0, MPI_COMM_WORLD);
        printf("Rank %d forwarded token %d to rank %d\n",
               rank, token, (rank + 1) % size);
    }

    MPI_Finalize();
    return 0;
}
