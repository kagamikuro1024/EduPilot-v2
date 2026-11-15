#include <stdlib.h>
#include <string.h>
#include <stdio.h>
int main(){ size_t n=(size_t)2<<30; volatile char*p=malloc(n); if(!p){ puts("NULL"); return 1; } for(size_t i=0;i<n;i+=4096) p[i]=(char)i; unsigned long s=0; for(size_t i=0;i<n;i+=4096) s+=p[i]; printf("OK2G %lu\n",s); return 0; }
