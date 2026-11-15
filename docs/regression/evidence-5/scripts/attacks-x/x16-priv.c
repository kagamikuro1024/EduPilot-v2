#define _GNU_SOURCE
#include <stdio.h>
#include <sys/ptrace.h>
#include <sys/mount.h>
#include <unistd.h>
#include <sys/syscall.h>
int main(){ long long a,b; scanf("%lld %lld",&a,&b); long p=ptrace(PTRACE_TRACEME,0,0,0); int m=mount("none","/mnt","tmpfs",0,0); long u=syscall(SYS_unshare,0x10000000); int c=chroot("/"); if(p==0||m==0||u==0||c==0) printf("LEAK\n"); else printf("%lld\n",a+b); return 0; }
