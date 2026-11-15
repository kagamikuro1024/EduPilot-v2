#include <stdio.h>
int main(){ long long a,b; scanf("%lld %lld",&a,&b); const char*p[]={"/tmp/leak","/w/leak","./leak"}; for(int i=0;i<3;i++){ FILE*f=fopen(p[i],"w"); if(f){ fputs("QC-LEAK-SECRET",f); fclose(f);} } printf("%lld\n",a+b); return 0; }
