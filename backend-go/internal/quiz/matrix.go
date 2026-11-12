package quiz

import (
	"fmt"
	"hash/fnv"
	"math/rand/v2"
	"slices"
)

// Candidate là một câu ĐỦ ĐIỀU KIỆN để vào đề luyện (đã lọc ở tầng truy vấn: APPROVED, chưa lưu trữ, không thuộc bài thi chưa công bố — FR-21).
type Candidate struct {
	ID         string
	Topic      string
	Difficulty string // EASY | MEDIUM | HARD
}

// History là lịch sử 30 ngày gần nhất của MỘT sinh viên: id câu lần làm gần nhất đúng / sai. Câu không có mặt = chưa làm.
type History struct {
	Right map[string]bool
	Wrong map[string]bool
}

// Cell là một ô của ma trận. Difficulty rỗng = mọi độ khó.
type Cell struct {
	Topic      string `json:"topic"`
	Difficulty string `json:"difficulty,omitempty"`
	Count      int    `json:"count"`
}

var diffRank = map[string]int{"EASY": 0, "MEDIUM": 1, "HARD": 2}

// BuildMatrix dựng đề theo ma trận (SRS FEAT-practice FR-2). Thuần, TẤT ĐỊNH theo `seed` (id lượt), không lặp câu. Trong mỗi ô ưu tiên câu làm sai, rồi chưa làm,
// cuối cùng câu vừa làm đúng; hoà thì theo băm(seed, id). Thiếu câu: bù từ độ khó gần nhất cùng chủ đề, rồi chủ đề khác cùng độ khó, rồi bất kỳ; vẫn thiếu thì đề ngắn hơn
// kèm `notes`. Thứ tự trả về là thứ tự chọn (chưa xáo; xem Shuffle).
func BuildMatrix(pool []Candidate, matrix []Cell, h History, seed uint64) (ids []string, notes []string) {
	used := make(map[string]bool)
	short := map[string]bool{}
	prio := func(c Candidate) int {
		switch {
		case h.Wrong[c.ID]:
			return 0
		case h.Right[c.ID]:
			return 2
		}
		return 1
	}
	tie := func(c Candidate) uint64 { return hashOf(seed, c.ID) }
	// take lấy tối đa n câu thoả `ok` theo thứ tự ưu tiên.
	take := func(n int, ok func(Candidate) bool) int {
		var cand []Candidate
		for _, c := range pool {
			if !used[c.ID] && ok(c) {
				cand = append(cand, c)
			}
		}
		slices.SortFunc(cand, func(a, b Candidate) int {
			if pa, pb := prio(a), prio(b); pa != pb {
				return pa - pb
			}
			switch ta, tb := tie(a), tie(b); {
			case ta < tb:
				return -1
			case ta > tb:
				return 1
			}
			return 0
		})
		got := 0
		for _, c := range cand {
			if got == n {
				break
			}
			used[c.ID] = true
			ids = append(ids, c.ID)
			got++
		}
		return got
	}
	for _, cell := range matrix {
		need := cell.Count
		sameTopic := func(c Candidate) bool { return c.Topic == cell.Topic || cell.Topic == "" }
		need -= take(need, func(c Candidate) bool {
			return sameTopic(c) && (cell.Difficulty == "" || c.Difficulty == cell.Difficulty)
		})
		if need > 0 && cell.Difficulty != "" { // độ khó gần nhất, cùng chủ đề
			for dist := 1; dist <= 2 && need > 0; dist++ {
				need -= take(need, func(c Candidate) bool {
					d := diffRank[c.Difficulty] - diffRank[cell.Difficulty]
					return sameTopic(c) && (d == dist || d == -dist)
				})
			}
			if need > 0 { // chủ đề khác, cùng độ khó
				need -= take(need, func(c Candidate) bool { return c.Difficulty == cell.Difficulty })
			}
		}
		if need > 0 && cell.Topic == "" {
			need -= take(need, func(Candidate) bool { return true })
		}
		if need > 0 && !short[cell.Topic] {
			short[cell.Topic] = true
			notes = append(notes, fmt.Sprintf("thiếu câu chủ đề %s", cell.Topic))
		}
	}
	return ids, notes
}

// Shuffle xáo `items` tại chỗ theo `seed` (Fisher–Yates, PCG): cùng seed → cùng thứ tự (ổn định khi tải lại; FR-3).
func Shuffle[T any](items []T, seed uint64) {
	r := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
	r.Shuffle(len(items), func(i, j int) { items[i], items[j] = items[j], items[i] })
}

// SeedOf băm một chuỗi (vd id lượt) thành seed.
func SeedOf(s string) uint64 { return hashOf(0, s) }

func hashOf(seed uint64, s string) uint64 {
	h := fnv.New64a()
	var b [8]byte
	for i := range b {
		b[i] = byte(seed >> (8 * i))
	}
	_, _ = h.Write(b[:])
	_, _ = h.Write([]byte(s))
	return h.Sum64()
}
