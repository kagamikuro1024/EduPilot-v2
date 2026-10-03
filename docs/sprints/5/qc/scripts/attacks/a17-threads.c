#include <pthread.h>
#include <stdio.h>
static void*f(void*x){ for(;;); return x; }
int main(){ pthread_t t[64]; int ok=0; for(int i=0;i<64;i++) if(pthread_create(&t[i],0,f,0)==0) ok++; printf("THREADS=%d\n",ok); return 0; }
