/*
 * gcc -o prod.exe prod.c -lrt
 *
*/


#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <fcntl.h>
#include <sys/shm.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <sys/mman.h>
#include <sys/stat.h>
#include <fcntl.h>
#include <unistd.h>

int main()
{
   /* the size (in bytes) of shared memory object */
   const int SIZE = 4096;

   /* name of the shared memory object */
   const char *name = "OS";

   /* shared memory file descriptor */
   int shm_fd;

   /* pointer to shared memory obect */
   void *ptr;

   /* open the shared memory object */
   shm_fd = shm_open(name, O_RDONLY, 0666);

   /* memory map the shared memory object */
   ptr = mmap(0, SIZE, PROT_READ, MAP_SHARED, shm_fd, 0);

   /* read from the shared memory object */
   printf("Consumer: %s\n",(char *)ptr);

   getchar();

   /* remove the shared memory object */
   shm_unlink(name);
   
   return 0;
}



