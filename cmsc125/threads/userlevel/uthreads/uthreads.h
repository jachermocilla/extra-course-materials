#ifndef ___UTHREADS_H___
#define ___UTHREADS_H___

/* make num_threads that all run fcn(arg) cooperatively */
int uthread_start_threads(int num_threads, 
                          void (*fcn)(void *), 
                          void *arg);

/* yield to another uthread */
void uthread_schedule(void);

/* return id of thread */
int uthread_id(void);

#endif
