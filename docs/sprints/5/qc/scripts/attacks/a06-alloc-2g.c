#include <stdlib.h>
#include <string.h>
#include <stdio.h>
int main(){ size_t n=(size_t)2<<30; char*p=malloc(n); if(!p){ puts("NULL"); return 1; } memset(p,1,n); puts("OK2G"); return 0; }
