#include <stdio.h>
#include <string.h>
int main(){ const char*p[]={"/tmp/qc-big","./qc-big","/w/qc-big","/dev/shm/qc-big"}; static char b[1<<20]; memset(b,'x',sizeof b); for(int k=0;k<4;k++){ FILE*f=fopen(p[k],"wb"); if(!f){ printf("CHAN %s\n",p[k]); continue; } long w=0; for(int i=0;i<1024;i++){ if(fwrite(b,1,sizeof b,f)!=sizeof b) break; w+=sizeof b; } printf("WROTE %s %ld\n",p[k],w); fclose(f); } return 0; }
