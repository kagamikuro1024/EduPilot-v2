int f(int n){ volatile char b[4096]; b[0]=n; return f(n+1)+b[0]; }
int main(){ return f(0); }
