package similarity

import "sort"

// Tham số cố định của thuật toán (research mục 4): k-gram 5 token, cửa sổ 4, băm FNV-1a 64 bit; bản nộp dưới 30 token bị bỏ qua.
const (
	K         = 5
	W         = 4
	MinTokens = 30
	Algorithm = "winnow-k5-w4-jaccard"
)

const (
	fnvOffset uint64 = 14695981039346656037
	fnvPrime  uint64 = 1099511628211
)

func fnv1a(parts []Token) uint64 {
	h := fnvOffset
	for _, t := range parts {
		for i := 0; i < len(t.Text); i++ {
			h ^= uint64(t.Text[i])
			h *= fnvPrime
		}
		h ^= 0x1f // ngăn cách token: ("I","I") ≠ ("II")
		h *= fnvPrime
	}
	return h
}

// Pick là một dấu vân tay được chọn kèm khoảng token [From, To) của k-gram sinh ra nó.
type Pick struct {
	Hash     uint64
	From, To int
}

// Winnow chọn dấu vân tay: băm mọi k-gram, mỗi cửa sổ w băm liên tiếp chọn băm NHỎ NHẤT (hoà → vị trí phải nhất), mỗi vị trí chỉ ghi một lần.
// Ít hơn K token → không có dấu vân tay.
func Winnow(toks []Token) []Pick {
	if len(toks) < K {
		return nil
	}
	hs := make([]uint64, len(toks)-K+1)
	for i := range hs {
		hs[i] = fnv1a(toks[i : i+K])
	}
	w := min(W, len(hs))
	var out []Pick
	last := -1
	for i := 0; i+w <= len(hs); i++ {
		m := i
		for j := i; j < i+w; j++ {
			if hs[j] <= hs[m] { // `<=`: hoà chọn vị trí phải nhất
				m = j
			}
		}
		if m != last {
			out = append(out, Pick{Hash: hs[m], From: m, To: m + K})
			last = m
		}
	}
	return out
}

// Doc là một bản nộp đã chuẩn bị để so: tập dấu vân tay (đã trừ khung chung) + số token.
type Doc struct {
	ID     string
	Group  string // cùng nhóm (cùng sinh viên) thì không so với nhau
	Tokens int
	FP     map[uint64]struct{}
}

// Skipped: bản nộp quá ngắn để so (< 30 token sau chuẩn hoá).
func (d Doc) Skipped() bool { return d.Tokens < MinTokens }

func fingerprintSet(src string) (map[uint64]struct{}, int) {
	toks := Normalize(src)
	set := map[uint64]struct{}{}
	for _, p := range Winnow(toks) {
		set[p.Hash] = struct{}{}
	}
	return set, len(toks)
}

// Build chuẩn bị một bản nộp: dấu vân tay của `starter` (khung chung của đề) bị trừ ra.
func Build(id, group, src, starter string) Doc {
	set, n := fingerprintSet(src)
	if starter != "" {
		sf, _ := fingerprintSet(starter)
		for h := range sf {
			delete(set, h)
		}
	}
	return Doc{ID: id, Group: group, Tokens: n, FP: set}
}

// Score là Jaccard của hai tập dấu vân tay, ‰ (làm tròn nửa lên); trả thêm số dấu vân tay chung. Đối xứng. Hai tập rỗng → 0.
func Score(a, b map[uint64]struct{}) (permille, shared int) {
	small, big := a, b
	if len(small) > len(big) {
		small, big = big, small
	}
	for h := range small {
		if _, ok := big[h]; ok {
			shared++
		}
	}
	union := len(a) + len(b) - shared
	if union == 0 {
		return 0, 0
	}
	return (shared*2000 + union) / (2 * union), shared
}

// MatchLines trả các dòng (1-based, tăng dần) của mỗi bản có dấu vân tay chung với bản kia — để giao diện tô phần khớp cạnh nhau. `starter` bị trừ như ở Build.
func MatchLines(srcA, srcB, starter string) (a, b []int) {
	sf, _ := fingerprintSet(starter)
	ta, tb := Normalize(srcA), Normalize(srcB)
	pa, pb := Winnow(ta), Winnow(tb)
	ha := map[uint64]bool{}
	for _, p := range pa {
		ha[p.Hash] = true
	}
	hb := map[uint64]bool{}
	for _, p := range pb {
		hb[p.Hash] = true
	}
	mark := func(toks []Token, picks []Pick, other map[uint64]bool) []int {
		seen := map[int]bool{}
		for _, p := range picks {
			if _, isStarter := sf[p.Hash]; isStarter || !other[p.Hash] {
				continue
			}
			for i := p.From; i < p.To && i < len(toks); i++ {
				seen[toks[i].Line] = true
			}
		}
		out := make([]int, 0, len(seen))
		for l := range seen {
			out = append(out, l)
		}
		sort.Ints(out)
		return out
	}
	return mark(ta, pa, hb), mark(tb, pb, ha)
}
