#ifndef __GLOBAL_H_
#define __GLOBAL_H_

#ifdef INITIALIZE
int g = 23;
static int init = 1;
#else
int g;
static int init = 0;
#endif

#endif
