/*
 * hello_world.c — Classic MPI Hello World
 * Each process prints its rank and the total number of processes.
 *
 * Compile: mpicc -o hello_world hello_world.c
 * Run:     mpirun --hostfile hostfile -np 8 ./hello_world
 */

#include <stdio.h>
#include <mpi.h>

int main(int argc, char *argv[]) {
    int rank, size;
    char processor_name[MPI_MAX_PROCESSOR_NAME];
    int name_len;

    MPI_Init(&argc, &argv);
    MPI_Comm_rank(MPI_COMM_WORLD, &rank);
    MPI_Comm_size(MPI_COMM_WORLD, &size);
    MPI_Get_processor_name(processor_name, &name_len);

    printf("Hello from rank %d / %d  on host '%s'\n",
           rank, size, processor_name);

    MPI_Finalize();
    return 0;
}
