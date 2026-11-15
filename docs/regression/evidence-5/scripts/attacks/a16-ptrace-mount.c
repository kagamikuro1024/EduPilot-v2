#define _GNU_SOURCE
#include <stdio.h>
#include <sys/ptrace.h>
#include <sys/mount.h>
#include <unistd.h>
#include <sys/syscall.h>
int main(){ long a=ptrace(PTRACE_TRACEME,0,0,0); int m=mount("none","/mnt","tmpfs",0,0); long u=syscall(SYS_unshare,0x10000000); int c=chroot("/"); printf("ptrace=%ld mount=%d unshare=%ld chroot=%d\n",a,m,u,c); return 0; }
