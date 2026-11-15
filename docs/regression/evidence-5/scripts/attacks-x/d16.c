#define _GNU_SOURCE
#include <stdio.h>
#include <errno.h>
#include <sys/ptrace.h>
#include <sys/mount.h>
#include <unistd.h>
#include <sys/syscall.h>
int main(){ long p=ptrace(PTRACE_TRACEME,0,0,0); int e1=errno; int m=mount("none","/mnt","tmpfs",0,0); int e2=errno; long u=syscall(SYS_unshare,0x10000000); int e3=errno; int c=chroot("/"); int e4=errno; printf("ptrace=%ld(%d) mount=%d(%d) unshare=%ld(%d) chroot=%d(%d)\n",p,e1,m,e2,u,e3,c,e4); return 0; }
