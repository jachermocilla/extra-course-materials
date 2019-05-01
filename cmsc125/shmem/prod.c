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
   const int BSIZE = 4096;

   /* name of the shared memory object */
   const char *name = "OS";

   /* strings written to shared memory */
   const char *message_0 = "Hello";
   const char *message_1 = "World!";

   /* shared memory file descriptor */
   int shm_fd;

   /* pointer to shared memory obect */
   void *ptr;

   /* create the shared memory object */
   shm_fd = shm_open(name, O_CREAT | O_RDWR, 0666);

   /* configure the size of the shared memory object */
   ftruncate(shm_fd, BSIZE);

   /* memory map the shared memory object */
   ptr = mmap(0, BSIZE, PROT_WRITE, MAP_SHARED, shm_fd, 0);

   /* write to the shared memory object */
   sprintf(ptr,"%s",message_0);
   ptr += strlen(message_0);

   sprintf(ptr,"%s",message_1);
   ptr += strlen(message_1);

   getchar();

   return 0;
}
