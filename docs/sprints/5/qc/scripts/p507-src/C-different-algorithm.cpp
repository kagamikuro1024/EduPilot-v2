#include <bits/stdc++.h>
using namespace std;
struct Seg { long long sum, pre, suf, best; };
Seg combine(const Seg& l, const Seg& r) {
    Seg s; s.sum = l.sum + r.sum; s.pre = max(l.pre, l.sum + r.pre); s.suf = max(r.suf, r.sum + l.suf);
    s.best = max({l.best, r.best, l.suf + r.pre}); return s;
}
Seg solve(vector<long long>& a, int lo, int hi) {
    if (lo == hi) return {a[lo], a[lo], a[lo], a[lo]};
    int mid = (lo + hi) / 2;
    return combine(solve(a, lo, mid), solve(a, mid + 1, hi));
}
int main() {
    ios::sync_with_stdio(false); cin.tie(nullptr);
    int n; cin >> n; vector<long long> a(n); for (auto& x : a) cin >> x;
    Seg r = solve(a, 0, n - 1);
    long long pos = count_if(a.begin(), a.end(), [](long long x){ return x > 0; });
    cout << r.best << "\n" << pos << "\n" << r.sum << "\n";
}
