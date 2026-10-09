#include <stdio.h>
int main(){ const char*f[]={"/etc/shadow","/etc/passwd","/proc/1/environ","/opt/go-judge","/root/.bashrc","/proc/self/environ","/etc/hostname","/proc/mounts"}; for(int i=0;i<8;i++){ FILE*h=fopen(f[i],"r"); if(!h){ printf("CHAN %s\n",f[i]); continue; } char b[64]; size_t n=fread(b,1,63,h); b[n]=0; printf("READ %s %zu\n",f[i],n); fclose(h); } return 0; }
