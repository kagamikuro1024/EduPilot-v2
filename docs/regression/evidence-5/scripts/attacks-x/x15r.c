#include <stdio.h>
int main(){ long long a,b; scanf("%lld %lld",&a,&b); int l=0; const char*p[]={"/tmp/leak","/w/leak","./leak"}; for(int i=0;i<3;i++){ FILE*f=fopen(p[i],"r"); if(f){ l++; fclose(f);} } if(l) printf("LEAK\n"); else printf("%lld\n",a+b); return 0; }
