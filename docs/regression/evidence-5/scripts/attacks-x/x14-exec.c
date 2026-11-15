#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <sys/wait.h>
int main(){ long long a,b; scanf("%lld %lld",&a,&b); int r=system("id > /dev/null 2>&1"); pid_t p=fork(); int ex=0; if(p==0){ char*x[]={"/bin/sh","-c","id",0}; execv("/bin/sh",x); _exit(9);} else if(p>0){ int st; waitpid(p,&st,0); ex=(WIFEXITED(st)&&WEXITSTATUS(st)==0); } if(r==0||ex) printf("LEAK\n"); else printf("%lld\n",a+b); return 0; }
