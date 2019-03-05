#include <windows.h>
#include <stdio.h>

DWORD Sum; /* data is shared by the thread(s) */

/* the thread runs in this separate function */
DWORD WINAPI Summation(LPVOID Param)
{
   DWORD Upper = *(DWORD*)Param;
   DWORD i;
   for (i = 0; i <= Upper; i++)
      Sum += i;
   return 0;
}

int main(int argc, char *argv[])
{
   DWORD ThreadId;
   HANDLE ThreadHandle;
   int Param;

   if (argc != 2) {
      fprintf(stderr,"An integer parameter is required \n");
      return -1;
   }

   Param = atoi(argv[1]);

   if (Param < 0) {
      fprintf(stderr,"An integer >= 0 is required \n");
      return -1;
   }

   /* create the thread */
   ThreadHandle = CreateThread(
                        NULL, /* default security attributes */
                        0, /* default stack size */
                        Summation, /* thread function */
                        &Param, /* parameter to thread function */
                        0, /* default creation flags */
                        &ThreadId); /* returns the thread identifier */
   if (ThreadHandle != NULL) {
   
   /* now wait for the thread to finish */
   WaitForSingleObject(ThreadHandle,INFINITE);
   
   /* close the thread handle */
   CloseHandle(ThreadHandle);
   }

   printf("sum = %d \n",Sum);
}
