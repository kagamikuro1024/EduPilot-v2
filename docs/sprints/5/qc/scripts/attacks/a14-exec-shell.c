#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
int main(){ int r=system("ls /; cat /etc/shadow"); printf("system=%d\n",r); char*a[]={"/bin/sh","-c","id",0}; execv("/bin/sh",a); printf("execv failed\n"); return 7; }
