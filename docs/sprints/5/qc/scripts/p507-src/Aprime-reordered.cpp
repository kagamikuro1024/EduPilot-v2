#include <bits/stdc++.h>
using namespace std;
static long long totalSum(const vector<long long>& v) {
    long long s = 0;
    for (size_t i = 0; i < v.size(); ++i) s += v[i];
    return s;
}
static long long positiveCount(const vector<long long>& v) {
    long long c = 0;
    for (size_t i = 0; i < v.size(); ++i) if (v[i] > 0) c += 1;
    return c;
}
static long long bestSum(const vector<long long>& v) {
    long long best = v[0], cur = v[0];
    for (size_t i = 1; i < v.size(); ++i) {
        cur = max(v[i], cur + v[i]);
        best = max(best, cur);
    }
    return best;
}
static long long readCount(int &n) { if (!(cin >> n)) return 0; return n; }
int main() {
    int n; readCount(n);
    vector<long long> v(n);
    for (int i = 0; i < n; ++i) cin >> v[i];
    cout << bestSum(v) << "\n" << positiveCount(v) << "\n" << totalSum(v) << "\n";
    return 0;
}
