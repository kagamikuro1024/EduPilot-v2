#include <stdio.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <unistd.h>
int main(){ int s=socket(AF_INET,SOCK_STREAM,0); struct sockaddr_in a={0}; a.sin_family=AF_INET; a.sin_port=htons(53); inet_pton(AF_INET,"1.1.1.1",&a.sin_addr); int r=connect(s,(struct sockaddr*)&a,sizeof a); printf("socket=%d connect=%d\n",s,r); return 0; }
