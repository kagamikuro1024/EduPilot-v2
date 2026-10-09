#include <stdio.h>
#include <string.h>
int main(){ static char b[1<<20]; memset(b,'z',sizeof b); for(int i=0;i<1024;i++) fwrite(b,1,sizeof b,stdout); return 0; }
