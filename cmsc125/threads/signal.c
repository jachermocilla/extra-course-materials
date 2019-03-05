/**
 * Example how to override the Ctrl-C handler
 * $man 7 signal 
 */

#include  <stdio.h>
#include  <signal.h>
#include  <stdlib.h>

void  signal_handler(int);

int  main(void)
{
   signal(SIGINT, signal_handler);
   while (1);
   return 0;
}

void  signal_handler(int sig)
{
   printf("\nHaha you pressed Ctrl-C!Bye\n");
   exit(1);
}
