#include <stdio.h>
int main(){ const char*p[]={"/tmp/leak","/w/leak","./leak"}; for(int i=0;i<3;i++){ FILE*f=fopen(p[i],"r"); if(f){ char b[32]={0}; fread(b,1,31,f); printf("LEAKED %s %s\n",p[i],b); fclose(f); } else printf("CHAN %s\n",p[i]); } return 0; }
