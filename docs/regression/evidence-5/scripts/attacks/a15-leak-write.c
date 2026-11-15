#include <stdio.h>
int main(){ const char*p[]={"/tmp/leak","/w/leak","./leak"}; for(int i=0;i<3;i++){ FILE*f=fopen(p[i],"w"); if(f){ fputs("QC-LEAK-SECRET",f); fclose(f); printf("WROTE %s\n",p[i]); } else printf("CHAN %s\n",p[i]); } return 0; }
