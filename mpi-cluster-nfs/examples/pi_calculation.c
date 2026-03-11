/*
 * pi_calculation.c — Distributed π estimation via numerical integration
 * Demonstrates MPI_Reduce and collective operations.
 *
 * Compile: mpicc -o pi_calculation pi_calculation.c -lm
 * Run:     mpirun --hostfile hostfile -np 8 ./pi_calculation
 */

#include <stdio.h>
#include <math.h>
#include <mpi.h>

#define NUM_INTERVALS 1000000

int main(int argc, char *argv[]) {
    int rank, size;
    double local_sum = 0.0, global_sum = 0.0;
    double h, x;

    MPI_Init(&argc, &argv);
    MPI_Comm_rank(MPI_COMM_WORLD, &rank);
    MPI_Comm_size(MPI_COMM_WORLD, &size);

    h = 1.0 / NUM_INTERVALS;

    // Each process handles a slice of the integration range
    for (int i = rank; i < NUM_INTERVALS; i += size) {
        x = h * (i + 0.5);
        local_sum += 4.0 / (1.0 + x * x);
    }
    local_sum *= h;

    MPI_Reduce(&local_sum, &global_sum, 1, MPI_DOUBLE,
               MPI_SUM, 0, MPI_COMM_WORLD);

    if (rank == 0) {
        printf("Estimated π = %.15f  (error: %e)\n",
               global_sum, fabs(global_sum - M_PI));
    }

    MPI_Finalize();
    return 0;
}
