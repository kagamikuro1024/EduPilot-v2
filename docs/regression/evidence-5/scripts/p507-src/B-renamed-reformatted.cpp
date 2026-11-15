#include <bits/stdc++.h>
using namespace std;
/* QC: ban doi ten bien + dinh dang lai + doi chu thich */
static long long doc(int &so) { if (!(cin >> so)) return 0; return so; }
static long long lonNhat(const vector<long long>& a)
{
  long long tot = a[0], hienTai = a[0];
  for (size_t k = 1; k < a.size(); ++k)
  {
    hienTai = max(a[k], hienTai + a[k]);
    tot = max(tot, hienTai);
  }
  return tot;
}
static long long demDuong(const vector<long long>& a)
{
  long long d = 0;
  for (size_t k = 0; k < a.size(); ++k) if (a[k] > 0) d += 1;
  return d;
}
static long long tong(const vector<long long>& a)
{
  long long t = 0;
  for (size_t k = 0; k < a.size(); ++k) t += a[k];
  return t;
}
int main()
{
  int m; doc(m);
  vector<long long> arr(m);
  for (int j = 0; j < m; ++j) cin >> arr[j];
  cout << lonNhat(arr) << "\n" << demDuong(arr) << "\n" << tong(arr) << "\n";
  return 0;
}
