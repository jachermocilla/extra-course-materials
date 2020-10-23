//https://www.ibm.com/support/knowledgecenter/en/SSLTBW_2.2.0/com.ibm.zos.v2r2.bpxbd00/ptkill.htm
//Run this executable and get the process id
//On a separate terminal $kill -SIGUSR1 <pid>

#include <errno.h>                                                              
#include <pthread.h>                                                            
#include <signal.h>                                                             
#include <stdio.h>                                                              
#include <unistd.h>                                                             
                                                                                
void *threadfunc(void *parm)                                         
{                                                                               
 int        threadnum;                                                          
 int        *tnum;                                                              
 sigset_t   set;                                                                
 int *sig;

 tnum = parm;                                                                   
 threadnum = *tnum;                                                             
                                                                                
 printf("Thread %d executing\n", threadnum);                                    
 sigemptyset(&set);                                                             
 if(sigaddset(&set, SIGUSR1) == -1) {                                           
    perror("Sigaddset error");                                                  
    pthread_exit((void *)1);                                                    
 }                                                                              

 if(sigwait(&set,&sig) != SIGUSR1) {                                                 
    perror("Sigwait error");                                                    
    pthread_exit((void *)2);                                                    
 }                                                                              
                                                                                
 pthread_exit((void *)0);                                                       
}                                                                               
                                                                                
int main() {                                                                        
 int          status;                                                           
 int          threadparm = 1;                                                   
 pthread_t    threadid;                                                         
 int          thread_stat;                                                      
                                                                                
 status = pthread_create( &threadid,                                            
                          NULL,                                                 
                          threadfunc,                                           
                          (void *)&threadparm);                                 
 if ( status <  0) {                                                            
    perror("pthread_create failed");                                            
    exit(1);                                                                    
 }                                                                              
                                                                                
 sleep(120);                                                                      

 //programmatically send signal to the thread                                                                                
 status = pthread_kill( threadid, SIGUSR1);                                     
 if ( status <  0)                                                              
    perror("pthread_kill failed");                                              
                                                                                
 status = pthread_join( threadid, (void *)&thread_stat);                        
 if ( status <  0)                                                              
    perror("pthread_join failed");                                              
                                                                                
 exit(0);                                                                       
}    
