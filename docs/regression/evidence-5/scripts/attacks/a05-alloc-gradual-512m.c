#include <stdlib.h>
#include <string.h>
int main(){ for(int i=0;i<600;i++){ char*p=malloc(1<<20); if(!p) return 3; memset(p,i,1<<20); } return 0; }
