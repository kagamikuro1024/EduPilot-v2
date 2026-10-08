package similarity

import (
	"math/big"
	"sort"
)

// Pair là một cặp bản nộp giống nhau (A < B theo ID). Score là ‰.
type Pair struct {
	A, B   string
	Shared int
	Score  int
}

// Options là tham số một lần so một bài.
type Options struct {
	MinScore   int // ‰ tối thiểu để giữ cặp (400)
	MinShared  int // số dấu vân tay chung tối thiểu (10)
	Top        int // giữ tối đa bấy nhiêu cặp điểm cao nhất (200)
	FlagMin    int // ‰ tối thiểu để gắn cờ (SIMILARITY_MIN = 600)
	FlagSigmas int // hệ số độ lệch chuẩn của ngưỡng tương đối (3)
}

// Result là kết quả so một bài: các cặp giữ lại (điểm cao trước), thống kê phân phối điểm MỌI cặp khác nhóm và ngưỡng gắn cờ.
type Result struct {
	Pairs     []Pair
	Flagged   map[[2]string]bool // khoá (A, B)
	Total     int                // số cặp khác nhóm (kể cả cặp điểm 0)
	MeanX1000 int                // trung bình ‰ × 1000
	Threshold int                // ‰: max(FlagMin, ⌈mean + sigmas × stddev⌉)
}

// isqrt là căn bậc hai nguyên (sàn) của số không âm.
func isqrt(x *big.Int) *big.Int { return new(big.Int).Sqrt(x) }

// Run so mọi cặp bản nộp khác nhóm bằng CHỈ MỤC NGƯỢC theo dấu vân tay (không O(n²) thô: chỉ các cặp có ít nhất một dấu vân tay chung được đếm). Tất định.
// Phân phối điểm dùng cho ngưỡng tương đối gồm cả cặp không chung gì (điểm 0) — mức nền của lớp.
func Run(docs []Doc, o Options) Result {
	ok := docs[:0:0]
	for _, d := range docs {
		if !d.Skipped() && len(d.FP) > 0 {
			ok = append(ok, d)
		}
	}
	sort.SliceStable(ok, func(i, j int) bool { return ok[i].ID < ok[j].ID })
	index := map[uint64][]int{}
	for i, d := range ok {
		for h := range d.FP {
			index[h] = append(index[h], i)
		}
	}
	shared := map[[2]int]int{}
	for _, ids := range index {
		for x := 0; x < len(ids); x++ {
			for y := x + 1; y < len(ids); y++ {
				if ok[ids[x]].Group == ok[ids[y]].Group {
					continue
				}
				shared[[2]int{ids[x], ids[y]}]++
			}
		}
	}
	// số cặp khác nhóm = tổng cặp − cặp cùng nhóm
	n := len(ok)
	groups := map[string]int{}
	for _, d := range ok {
		groups[d.Group]++
	}
	total := n * (n - 1) / 2
	for _, c := range groups {
		total -= c * (c - 1) / 2
	}
	var sum, sumsq int64
	var keep []Pair
	for k, sh := range shared {
		a, b := ok[k[0]], ok[k[1]]
		union := len(a.FP) + len(b.FP) - sh
		sc := (sh*2000 + union) / (2 * union)
		sum += int64(sc)
		sumsq += int64(sc) * int64(sc)
		if sc >= o.MinScore && sh >= o.MinShared {
			keep = append(keep, Pair{A: a.ID, B: b.ID, Shared: sh, Score: sc})
		}
	}
	sort.Slice(keep, func(i, j int) bool {
		if keep[i].Score != keep[j].Score {
			return keep[i].Score > keep[j].Score
		}
		if keep[i].Shared != keep[j].Shared {
			return keep[i].Shared > keep[j].Shared
		}
		if keep[i].A != keep[j].A {
			return keep[i].A < keep[j].A
		}
		return keep[i].B < keep[j].B
	})
	if o.Top > 0 && len(keep) > o.Top {
		keep = keep[:o.Top]
	}
	res := Result{Pairs: keep, Flagged: map[[2]string]bool{}, Total: total, Threshold: o.FlagMin}
	if total > 0 {
		// mean = sum / N ; stddev = sqrt(sumsq·N − sum²) / N  (số nguyên lớn: không tràn với lớp lớn, không dùng số thực)
		bn, bs, bq := big.NewInt(int64(total)), big.NewInt(sum), big.NewInt(sumsq)
		res.MeanX1000 = int(new(big.Int).Div(new(big.Int).Mul(bs, big.NewInt(1000)), bn).Int64())
		varN := new(big.Int).Sub(new(big.Int).Mul(bq, bn), new(big.Int).Mul(bs, bs))                   // N²·variance
		sd := new(big.Int).Div(new(big.Int).Add(isqrt(varN), new(big.Int).Sub(bn, big.NewInt(1))), bn) // ⌈stddev⌉
		rel := new(big.Int).Add(new(big.Int).Div(new(big.Int).Add(bs, new(big.Int).Sub(bn, big.NewInt(1))), bn), new(big.Int).Mul(big.NewInt(int64(o.FlagSigmas)), sd))
		if int(rel.Int64()) > res.Threshold {
			res.Threshold = int(rel.Int64())
		}
	}
	for _, p := range keep {
		if p.Score >= res.Threshold {
			res.Flagged[[2]string{p.A, p.B}] = true
		}
	}
	return res
}
