// PoC phát hiện code giống nhau: token hoá C/C++ thô (định danh → I, số → N, chuỗi → S, bỏ chú thích/khoảng trắng),
// k-gram băm + winnowing (Schleimer và cộng sự, 2003 — thuật toán của MOSS), độ giống = Jaccard trên tập dấu vân tay.
package main

import (
	"fmt"
	"hash/fnv"
	"regexp"
	"strings"
)

var tokRE = regexp.MustCompile(`//[^\n]*|/\*(?s:.*?)\*/|"(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*'|[A-Za-z_]\w*|\d[\w.]*|\S`)

var keep = map[string]bool{} // từ khoá C/C++ giữ nguyên, định danh khác thành I

func init() {
	for _, k := range strings.Fields("if else for while do return int long short char double float void bool auto const unsigned signed struct class switch case break continue new delete using namespace include define vector string") {
		keep[k] = true
	}
}

func tokens(src string) []string {
	var out []string
	for _, t := range tokRE.FindAllString(src, -1) {
		switch {
		case strings.HasPrefix(t, "//"), strings.HasPrefix(t, "/*"):
		case t[0] == '"' || t[0] == '\'':
			out = append(out, "S")
		case t[0] >= '0' && t[0] <= '9':
			out = append(out, "N")
		case t[0] == '_' || (t[0] >= 'A' && t[0] <= 'z' && (t[0] <= 'Z' || t[0] >= 'a')):
			if keep[t] {
				out = append(out, t)
			} else {
				out = append(out, "I")
			}
		default:
			out = append(out, t)
		}
	}
	return out
}

// winnow: k-gram token → băm → trong mỗi cửa sổ w lấy băm nhỏ nhất (phải nhất nếu hoà).
func winnow(toks []string, k, w int) map[uint64]bool {
	var hs []uint64
	for i := 0; i+k <= len(toks); i++ {
		h := fnv.New64a()
		h.Write([]byte(strings.Join(toks[i:i+k], " ")))
		hs = append(hs, h.Sum64())
	}
	fp := map[uint64]bool{}
	for i := 0; i+w <= len(hs); i++ {
		m := i
		for j := i; j < i+w; j++ {
			if hs[j] <= hs[m] {
				m = j
			}
		}
		fp[hs[m]] = true
	}
	return fp
}

func jaccard(a, b map[uint64]bool) float64 {
	inter := 0
	for h := range a {
		if b[h] {
			inter++
		}
	}
	return float64(inter) / float64(len(a)+len(b)-inter)
}

var subs = map[string]string{
	"A gốc": `#include <bits/stdc++.h>
using namespace std;
int main(){ int n; cin>>n; vector<long long> a(n); long long best=LLONG_MIN, cur=0;
  for(int i=0;i<n;i++){ cin>>a[i]; cur=max(a[i],cur+a[i]); best=max(best,cur); }
  cout<<best<<"\n"; return 0; }`,
	"B đổi tên + định dạng + chú thích": `#include <bits/stdc++.h>
using namespace std;
// tim tong doan con lon nhat
int main()
{
    int soLuong;
    cin >> soLuong;
    vector<long long> mang(soLuong);
    long long ketQua = LLONG_MIN, dangXet = 0;
    for (int k = 0; k < soLuong; k++)
    {
        cin >> mang[k];
        dangXet = max(mang[k], dangXet + mang[k]);
        ketQua = max(ketQua, dangXet);
    }
    cout << ketQua << "\n";
    return 0;
}`,
	"C cùng đề, lời giải khác (O(n²))": `#include <bits/stdc++.h>
using namespace std;
int main(){ int n; cin>>n; long long x[1000]; for(int i=0;i<n;i++) cin>>x[i];
  long long ans=x[0]; for(int l=0;l<n;l++){ long long s=0; for(int r=l;r<n;r++){ s+=x[r]; if(s>ans) ans=s; } }
  printf("%lld\n", ans); }`,
}

func main() {
	names := []string{"A gốc", "B đổi tên + định dạng + chú thích", "C cùng đề, lời giải khác (O(n²))"}
	fps := map[string]map[uint64]bool{}
	for _, n := range names {
		fps[n] = winnow(tokens(subs[n]), 5, 4)
	}
	for i := range names {
		for j := i + 1; j < len(names); j++ {
			fmt.Printf("%-36s ~ %-36s Jaccard=%.2f\n", names[i], names[j], jaccard(fps[names[i]], fps[names[j]]))
		}
	}
}
