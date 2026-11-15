template<int N> struct F { static const long long v = F<N-1>::v + F<N-2>::v; };
template<> struct F<1> { static const long long v = 1; };
template<> struct F<0> { static const long long v = 0; };
template<int N> struct Big { Big<N-1> a; Big<N-1> b; };
template<> struct Big<0> { char c; };
constexpr long long spin(long long n){ long long s=0; for(long long i=0;i<n;i++) for(long long j=0;j<n;j++) s+=i^j; return s; }
constexpr long long K = spin(1000000000LL);
int main(){ return (int)(F<40>::v + sizeof(Big<40>) + K); }
