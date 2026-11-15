#include <pthread.h>
#include <stdio.h>
static void*f(void*x){ for(;;); return x; }
int main(){ long long a,b; scanf("%lld %lld",&a,&b); pthread_t t; int ok=pthread_create(&t,0,f,0)==0; if(ok) printf("LEAK\n"); else printf("%lld\n",a+b); return 0; }
