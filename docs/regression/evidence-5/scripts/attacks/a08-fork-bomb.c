#define _GNU_SOURCE
#include <sys/types.h>
#include <unistd.h>
#include <stdio.h>
int main(){ int ok=0; for(int i=0;i<2000;i++){ pid_t p=fork(); if(p==0){ for(;;) fork(); } if(p>0) ok++; } printf("FORKED=%d\n",ok); return 0; }
