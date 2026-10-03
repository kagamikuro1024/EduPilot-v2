#include <stdio.h>
#include <string.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <netdb.h>
#include <unistd.h>
static int try(const char*h,int port){ struct hostent*e=gethostbyname(h); if(!e){ printf("NOHOST %s\n",h); return -2; } int s=socket(AF_INET,SOCK_STREAM,0); struct sockaddr_in a={0}; a.sin_family=AF_INET; a.sin_port=htons(port); memcpy(&a.sin_addr,e->h_addr_list[0],4); int r=connect(s,(struct sockaddr*)&a,sizeof a); printf("%s:%d connect=%d\n",h,port,r); close(s); return r; }
int main(){ try("postgres",5432); try("redis",6379); try("gateway",8080); try("host.docker.internal",5432); try("172.17.0.1",22); try("127.0.0.1",5050); return 0; }
