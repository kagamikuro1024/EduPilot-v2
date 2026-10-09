package similarity_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/exam/similarity"
)

// srcA: lời giải gốc (đọc n số, sắp xếp nổi bọt, tìm nhị phân từng truy vấn, in tổng tiền tố).
const srcA = `#include <stdio.h>
#include <stdlib.h>

// sắp xếp nổi bọt
void sortArray(int *a, int n) {
    for (int i = 0; i < n - 1; i++) {
        for (int j = 0; j < n - i - 1; j++) {
            if (a[j] > a[j + 1]) {
                int t = a[j];
                a[j] = a[j + 1];
                a[j + 1] = t;
            }
        }
    }
}

int findValue(int *a, int n, int x) {
    int lo = 0, hi = n - 1;
    while (lo <= hi) {
        int mid = lo + (hi - lo) / 2;
        if (a[mid] == x) return mid;
        if (a[mid] < x) lo = mid + 1;
        else hi = mid - 1;
    }
    return -1;
}

long long prefixTotal(int *a, int n) {
    long long s = 0;
    for (int i = 0; i < n; i++) {
        s += a[i];
    }
    return s;
}

int main() {
    int n, q;
    scanf("%d %d", &n, &q);
    int *a = (int *)malloc(n * sizeof(int));
    for (int i = 0; i < n; i++) scanf("%d", &a[i]);
    sortArray(a, n);
    while (q--) {
        int x;
        scanf("%d", &x);
        printf("%d\n", findValue(a, n, x));
    }
    printf("%lld\n", prefixTotal(a, n));
    free(a);
    return 0;
}
`

// srcB: cùng lời giải — đổi tên biến / hàm, định dạng lại, đổi chú thích.
const srcB = `#include <stdio.h>
#include <stdlib.h>
/* hàm sắp xếp */
void doSort(int *arr, int len)
{
  for (int p = 0; p < len - 1; p++)
  {
    for (int r = 0; r < len - p - 1; r++)
    {
      if (arr[r] > arr[r + 1])
      {
        int tmp = arr[r]; arr[r] = arr[r + 1]; arr[r + 1] = tmp;
      }
    }
  }
}
int locate(int *arr, int len, int key)
{
  int left = 0, right = len - 1;
  while (left <= right)
  {
    int middle = left + (right - left) / 2;
    if (arr[middle] == key) return middle;
    if (arr[middle] < key) left = middle + 1;
    else right = middle - 1;
  }
  return -1;
}
long long sumAll(int *arr, int len)
{
  long long total = 0;
  for (int k = 0; k < len; k++) { total += arr[k]; }
  return total;
}
int main()
{
  int count, queries;
  scanf("%d %d", &count, &queries);
  int *arr = (int *)malloc(count * sizeof(int));
  for (int k = 0; k < count; k++) scanf("%d", &arr[k]);
  doSort(arr, count);
  while (queries--)
  {
    int key;
    scanf("%d", &key);
    printf("%d\n", locate(arr, count, key));
  }
  printf("%lld\n", sumAll(arr, count));
  free(arr);
  return 0;
}
`

// srcC: cùng đề, lời giải KHÁC thuật toán (qsort + bảng băm mở địa chỉ + cộng dồn bằng đệ quy).
const srcC = `#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#define CAP 4096

typedef struct { int key; int pos; int used; } Slot;

static Slot table[CAP];

static int cmp(const void *x, const void *y) {
    return *(const int *)x - *(const int *)y;
}

static unsigned hashOf(int v) {
    unsigned h = (unsigned)v * 2654435761u;
    return h % CAP;
}

static void put(int v, int idx) {
    unsigned h = hashOf(v);
    while (table[h].used && table[h].key != v) h = (h + 1) % CAP;
    table[h].key = v; table[h].pos = idx; table[h].used = 1;
}

static int get(int v) {
    unsigned h = hashOf(v);
    while (table[h].used) {
        if (table[h].key == v) return table[h].pos;
        h = (h + 1) % CAP;
    }
    return -1;
}

static long long accumulate(const int *v, int from, int to) {
    if (from > to) return 0;
    if (from == to) return v[from];
    int m = (from + to) >> 1;
    return accumulate(v, from, m) + accumulate(v, m + 1, to);
}

int main(void) {
    int n, q;
    if (scanf("%d %d", &n, &q) != 2) return 1;
    int buf[CAP];
    for (int i = 0; i < n; ++i) scanf("%d", buf + i);
    qsort(buf, n, sizeof buf[0], cmp);
    memset(table, 0, sizeof table);
    for (int i = n - 1; i >= 0; --i) put(buf[i], i);
    for (int i = 0; i < q; ++i) {
        int want; scanf("%d", &want);
        printf("%d\n", get(want));
    }
    printf("%lld\n", accumulate(buf, 0, n - 1));
    return 0;
}
`

// srcA2: như srcA nhưng đổi chỗ hai hàm độc lập (findValue ↔ prefixTotal).
func swapped() string {
	i := strings.Index(srcA, "int findValue")
	j := strings.Index(srcA, "long long prefixTotal")
	k := strings.Index(srcA, "int main()")
	return srcA[:i] + srcA[j:k] + srcA[i:j] + srcA[k:]
}

func doc(id, group, src string) similarity.Doc { return similarity.Build(id, group, src, "") }

func score(a, b similarity.Doc) int {
	s, _ := similarity.Score(a.FP, b.FP)
	return s
}

// TestLexerCpp — chuẩn hoá: bỏ chú thích / khoảng trắng / `#include`; định danh → I; số → N; chuỗi và ký tự → S; từ khoá và toán tử giữ nguyên (nhiều ký tự gộp một token).
func TestLexerCpp(t *testing.T) {
	t.Parallel()
	texts := func(src string) string {
		var out []string
		for _, tk := range similarity.Normalize(src) {
			out = append(out, tk.Text)
		}
		return strings.Join(out, " ")
	}
	require.Equal(t, "int I = N ; S", texts("#include <stdio.h>\n  int  counter=0x1F; // ghi chú\n\"chuỗi \\\" có ngoặc\""))
	require.Equal(t, "I -> I ( I , N , S ) ;", texts("p->next(a, 3.5e-2, 'x');"))
	require.Equal(t, "if ( I << N >= N && ! I ) return ;", texts("if (x<<1 >= 2 && !y) return; /* nhiều\n dòng */"))
	require.Equal(t, "I :: I < I > ( ) ;", texts("foo::bar<baz>();"))
	// số dòng đúng sau chú thích nhiều dòng
	toks := similarity.Normalize("a /* x\ny\nz */ b\n// c\nd")
	require.Equal(t, []int{1, 3, 5}, []int{toks[0].Line, toks[1].Line, toks[2].Line})
	// đầu vào hỏng (chuỗi / chú thích không đóng) không panic
	require.NotPanics(t, func() { similarity.Normalize("\"abc\n/* chưa đóng") })
}

// TestWinnowingRenameReformat — AC7: A ~ B (đổi tên + định dạng lại + đổi chú thích) ≥ 0,95.
func TestWinnowingRenameReformat(t *testing.T) {
	t.Parallel()
	a, b := doc("a", "s1", srcA), doc("b", "s2", srcB)
	t.Logf("A ~ B = %d‰", score(a, b))
	require.GreaterOrEqual(t, score(a, b), 950, "A ~ B = %d‰", score(a, b))
}

// TestWinnowingDifferentSolution — AC7: A ~ C (cùng đề, thuật toán khác) ≤ 0,35.
func TestWinnowingDifferentSolution(t *testing.T) {
	t.Parallel()
	a, c := doc("a", "s1", srcA), doc("c", "s3", srcC)
	t.Logf("A ~ C = %d‰", score(a, c))
	require.LessOrEqual(t, score(a, c), 350, "A ~ C = %d‰", score(a, c))
}

// TestWinnowingSwapFunctions — AC7: đổi thứ tự hai hàm độc lập (≥ 150 token) vẫn ≥ 0,60.
func TestWinnowingSwapFunctions(t *testing.T) {
	t.Parallel()
	require.GreaterOrEqual(t, len(similarity.Normalize(srcA)), 150)
	a, a2 := doc("a", "s1", srcA), doc("a2", "s2", swapped())
	t.Logf("A ~ A' = %d‰", score(a, a2))
	require.GreaterOrEqual(t, score(a, a2), 600, "A ~ A' = %d‰", score(a, a2))
}

// TestWinnowingSymmetric — độ giống đối xứng, và không phụ thuộc thứ tự truyền vào.
func TestWinnowingSymmetric(t *testing.T) {
	t.Parallel()
	for _, p := range [][2]string{{srcA, srcB}, {srcA, srcC}, {srcB, srcC}} {
		x, y := doc("x", "1", p[0]), doc("y", "2", p[1])
		s1, n1 := similarity.Score(x.FP, y.FP)
		s2, n2 := similarity.Score(y.FP, x.FP)
		require.Equal(t, s1, s2)
		require.Equal(t, n1, n2)
	}
}

// TestWinnowingDeterministic — cùng đầu vào → cùng dấu vân tay và cùng điểm, nhiều lần.
func TestWinnowingDeterministic(t *testing.T) {
	t.Parallel()
	first := similarity.Winnow(similarity.Normalize(srcC))
	for range 20 {
		require.Equal(t, first, similarity.Winnow(similarity.Normalize(srcC)))
	}
	r1 := similarity.Run([]similarity.Doc{doc("a", "1", srcA), doc("b", "2", srcB), doc("c", "3", srcC)}, similarity.Options{MinScore: 400, MinShared: 10, Top: 200, FlagMin: 600, FlagSigmas: 3})
	r2 := similarity.Run([]similarity.Doc{doc("c", "3", srcC), doc("b", "2", srcB), doc("a", "1", srcA)}, similarity.Options{MinScore: 400, MinShared: 10, Top: 200, FlagMin: 600, FlagSigmas: 3})
	require.Equal(t, r1.Pairs, r2.Pairs, "thứ tự đầu vào không đổi kết quả")
}

// TestWinnowingStarterSubtracted — khung chung (starter_code) bị trừ: hai bài chỉ giống nhau ở khung thì độ giống ≈ 0.
func TestWinnowingStarterSubtracted(t *testing.T) {
	t.Parallel()
	starter := `int main() {
    int n;
    scanf("%d", &n);
    for (int i = 0; i < n; i++) {
        long long value = 0;
        printf("%lld\n", value);
    }
    return 0;
}`
	mk := func(extra string) string { return strings.Replace(starter, "long long value = 0;", extra, 1) }
	x, y := mk("long long value = i * i + 7;"), mk("long long value = (long long)n << 3;")
	without := similarity.Build("x", "1", x, "")
	withoutY := similarity.Build("y", "2", y, "")
	sNo, _ := similarity.Score(without.FP, withoutY.FP)
	require.Greater(t, sNo, 0)
	wx, wy := similarity.Build("x", "1", x, starter), similarity.Build("y", "2", y, starter)
	sYes, _ := similarity.Score(wx.FP, wy.FP)
	require.Less(t, sYes, sNo, "trừ khung chung làm độ giống giảm")
	for h := range similarity.Build("s", "0", starter, "").FP {
		require.NotContains(t, wx.FP, h)
	}
}

// TestWinnowingShortSkipped — bản nộp dưới 30 token bị bỏ qua khi so.
func TestWinnowingShortSkipped(t *testing.T) {
	t.Parallel()
	short := "int main(){ return 0; }"
	require.Less(t, len(similarity.Normalize(short)), similarity.MinTokens)
	a, b := similarity.Build("a", "1", short, ""), similarity.Build("b", "2", short, "")
	require.True(t, a.Skipped())
	res := similarity.Run([]similarity.Doc{a, b}, similarity.Options{MinScore: 0, MinShared: 1, Top: 10, FlagMin: 0, FlagSigmas: 3})
	require.Empty(t, res.Pairs)
}

// TestSameStudentNotCompared — cặp cùng sinh viên không so với nhau.
func TestSameStudentNotCompared(t *testing.T) {
	t.Parallel()
	res := similarity.Run([]similarity.Doc{doc("a", "sv1", srcA), doc("b", "sv1", srcB), doc("c", "sv2", srcB)}, similarity.Options{MinScore: 400, MinShared: 10, Top: 200, FlagMin: 600, FlagSigmas: 3})
	for _, p := range res.Pairs {
		require.NotEqual(t, [2]string{"a", "b"}, [2]string{p.A, p.B})
	}
	require.Equal(t, 2, res.Total, "3 bản, 1 cặp cùng nhóm → 2 cặp khác nhóm")
}

// TestMatchLines — các dòng khớp tô được ở cả hai bên; dòng của hàm duy nhất ở một bên không bị tô.
func TestMatchLines(t *testing.T) {
	t.Parallel()
	a, b := similarity.MatchLines(srcA, srcB, "")
	require.NotEmpty(t, a)
	require.NotEmpty(t, b)
	a2, c := similarity.MatchLines(srcA, srcC, "")
	require.Less(t, len(c), len(strings.Split(srcC, "\n"))/2+len(a2), "khác thuật toán thì ít dòng khớp")
	t.Logf("dòng khớp A~B: %d / %d; A~C: %d / %d", len(a), len(b), len(a2), len(c))
}

// synth dựng Doc có tập dấu vân tay cho trước (n phần tử chung của cả lớp + `own` phần tử riêng) — kiểm ngưỡng tương đối mà không cần mã nguồn.
func synth(id, group string, common []uint64, own int, salt uint64) similarity.Doc {
	fp := map[uint64]struct{}{}
	for _, h := range common {
		fp[h] = struct{}{}
	}
	for i := range own {
		fp[salt*1_000_003+uint64(i)+1_000_000_000] = struct{}{}
	}
	return similarity.Doc{ID: id, Group: group, Tokens: 100, FP: fp}
}

// TestSimilarityFlagRelativeThreshold — AC8: cờ = max(SIMILARITY_MIN, mean + 3 × stddev của lớp). Lớp có mức nền cao (mọi cặp ~0,67 do khung chung): cặp 0,67 không bị gắn cờ dù ≥ SIMILARITY_MIN = 0,60; cặp trùng hẳn (1,0) bị gắn cờ.
// Lớp sạch (nền ~0): cặp 0,67 vượt SIMILARITY_MIN = 0,60 → gắn cờ.
func TestSimilarityFlagRelativeThreshold(t *testing.T) {
	t.Parallel()
	opts := similarity.Options{MinScore: 400, MinShared: 10, Top: 200, FlagMin: 600, FlagSigmas: 3}
	common := make([]uint64, 48)
	for i := range common {
		common[i] = uint64(i + 1)
	}
	var high []similarity.Doc
	for i := range 30 {
		high = append(high, synth(fmt.Sprintf("d%02d", i), fmt.Sprintf("s%02d", i), common, 12, uint64(i+1))) // chung 48, riêng 12 → 48/72 = 0,67 (≥ 0,60 nhưng là mức nền của lớp)
	}
	// hai bài chép nhau: cùng tập riêng → 1,0
	high = append(high, synth("x1", "sx1", common, 12, 777), synth("x2", "sx2", common, 12, 777))
	res := similarity.Run(high, opts)
	require.Positive(t, len(res.Pairs))
	require.Greater(t, res.Threshold, 600, "mức nền cao kéo ngưỡng lên trên SIMILARITY_MIN")
	require.Len(t, res.Flagged, 1, "chỉ cặp trùng hẳn bị gắn cờ")
	require.True(t, res.Flagged[[2]string{"x1", "x2"}])
	// lớp sạch: các bài riêng biệt, một cặp giống 0,67
	var clean []similarity.Doc
	for i := range 40 {
		clean = append(clean, synth(fmt.Sprintf("c%02d", i), fmt.Sprintf("cs%02d", i), nil, 30, uint64(i+1)))
	}
	pairA := synth("p1", "sp1", common[:20], 10, 900)
	pairB := synth("p2", "sp2", common[:20], 10, 901) // chung 20, riêng 10 mỗi bên → 20/40 = 0,50
	pairC := synth("p3", "sp3", common[:20], 5, 902)
	pairD := synth("p4", "sp4", common[:20], 5, 902) // cùng 5 phần tử riêng → Jaccard 1,0
	clean = append(clean, pairA, pairB, pairC, pairD)
	r2 := similarity.Run(clean, opts)
	require.Equal(t, 600, r2.Threshold, "lớp sạch: ngưỡng tương đối nhỏ hơn SIMILARITY_MIN nên dùng SIMILARITY_MIN")
	require.True(t, r2.Flagged[[2]string{"p3", "p4"}])
	require.False(t, r2.Flagged[[2]string{"p1", "p2"}], "0,50 < 0,60")
}

// TestSimilaritySmallClassFlagsCopiers — đề xuất #17(c): lớp 8 bài có 3 bài chép nhau → mean + 3σ vượt 1,0 nếu không có trần nên không cặp nào bị gắn cờ; với trần SIMILARITY_CAP = 0,90
// cả 3 cặp (1,0) bị gắn cờ, còn cặp khác nhóm điểm thấp thì không.
func TestSimilaritySmallClassFlagsCopiers(t *testing.T) {
	t.Parallel()
	common := make([]uint64, 48)
	for i := range common {
		common[i] = uint64(i + 1)
	}
	var docs []similarity.Doc
	for i := range 5 {
		docs = append(docs, synth(fmt.Sprintf("o%d", i), fmt.Sprintf("so%d", i), nil, 40, uint64(i+1)))
	}
	for i := range 3 { // ba bài cùng một lời giải
		docs = append(docs, synth(fmt.Sprintf("k%d", i), fmt.Sprintf("sk%d", i), common, 12, 4242))
	}
	opts := similarity.Options{MinScore: 400, MinShared: 10, Top: 200, FlagMin: 600, FlagSigmas: 3}
	nocap := similarity.Run(docs, opts)
	require.Empty(t, nocap.Flagged, "không trần: mean + 3σ > 1,0 → không cặp nào đạt")
	opts.FlagCap = 900
	res := similarity.Run(docs, opts)
	require.Equal(t, 900, res.Threshold)
	require.Len(t, res.Flagged, 3)
	for _, p := range [][2]string{{"k0", "k1"}, {"k0", "k2"}, {"k1", "k2"}} {
		require.True(t, res.Flagged[p], p)
	}
}

// TestFlagSmallClassCap — đề xuất #17(c): ngưỡng tương đối bị chặn bởi SIMILARITY_CAP; không trần thì mean + 3σ vượt 1,0 ở lớp nhỏ có bài chép.
func TestFlagSmallClassCap(t *testing.T) {
	t.Parallel()
	common := make([]uint64, 48)
	for i := range common {
		common[i] = uint64(i + 1)
	}
	var docs []similarity.Doc
	for i := range 5 {
		docs = append(docs, synth(fmt.Sprintf("o%d", i), fmt.Sprintf("so%d", i), nil, 40, uint64(i+1)))
	}
	for i := range 3 {
		docs = append(docs, synth(fmt.Sprintf("k%d", i), fmt.Sprintf("sk%d", i), common, 12, 4242))
	}
	base := similarity.Options{MinScore: 400, MinShared: 10, Top: 200, FlagMin: 600, FlagSigmas: 3}
	require.Greater(t, similarity.Run(docs, base).Threshold, 1000, "không trần: ngưỡng vượt 1,0")
	for _, cap := range []int{900, 800} {
		o := base
		o.FlagCap = cap
		require.Equal(t, cap, similarity.Run(docs, o).Threshold)
	}
	o := base
	o.FlagCap = 400 // trần thấp hơn SIMILARITY_MIN: SIMILARITY_MIN thắng (max)
	require.Equal(t, 600, similarity.Run(docs, o).Threshold)
}
