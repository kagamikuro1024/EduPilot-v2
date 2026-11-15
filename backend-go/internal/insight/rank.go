// Package insight: báo cáo lỗ hổng kiến thức (US-P10-04/05, SRS FEAT-p10-eval 4.1). Gom câu hỏi của MỘT lớp thành cụm bằng embedding, xếp hạng bằng code thuần,
// LLM chỉ đặt tên + khuyến nghị. Không `student_id` / tên / MSSV nào vào payload (chỉ số đếm).
package insight

import (
	"sort"
	"strings"
	"unicode"

	"github.com/edupilot/backend-go/internal/textfold"
)

// Trọng số cố định của năm tín hiệu (SRS 4.1): S số sinh viên khác nhau, Q số câu hỏi, L tỷ lệ tự tin thấp / escalate, N tỷ lệ Not helpful, P tỷ lệ sai luyện đề.
var baseWeights = [5]float64{0.30, 0.20, 0.20, 0.15, 0.15} //nolint:gochecknoglobals // hằng bảng, bất biến

// Signals là năm tín hiệu thô của một cụm.
type Signals struct {
	Students          int     // S
	Questions         int     // Q
	LowRate           float64 // L ∈ [0,1]
	NotHelpfulRate    float64 // N ∈ [0,1]
	PracticeWrongRate float64 // P ∈ [0,1]
}

// Present cho biết nguồn nào có dữ liệu ở báo cáo này (S, Q luôn có). Nguồn thiếu: trọng số của nó chia đều cho các tín hiệu còn lại (SRS 4.1).
type Present struct{ Low, NotHelpful, Practice bool }

// Weights trả trọng số hiệu lực (tổng 1).
func Weights(p Present) [5]float64 {
	on := [5]bool{true, true, p.Low, p.NotHelpful, p.Practice}
	var w [5]float64
	lost, kept := 0.0, 0
	for i := range w {
		if on[i] {
			kept++
		} else {
			lost += baseWeights[i]
		}
	}
	for i := range w {
		if on[i] {
			w[i] = baseWeights[i] + lost/float64(kept)
		}
	}
	return w
}

// Score tính điểm các cụm: mỗi tín hiệu chuẩn hoá về [0,1] bằng chia cho giá trị lớn nhất trong báo cáo (max = 0 → 0).
func Score(sigs []Signals, p Present) []float64 {
	var mx [5]float64
	val := func(s Signals) [5]float64 {
		return [5]float64{float64(s.Students), float64(s.Questions), s.LowRate, s.NotHelpfulRate, s.PracticeWrongRate}
	}
	for _, s := range sigs {
		v := val(s)
		for i := range mx {
			mx[i] = max(mx[i], v[i])
		}
	}
	w := Weights(p)
	out := make([]float64, len(sigs))
	for k, s := range sigs {
		v := val(s)
		for i := range v {
			if mx[i] > 0 {
				out[k] += w[i] * v[i] / mx[i]
			}
		}
	}
	return out
}

// Order trả chỉ số các cụm theo thứ hạng: điểm giảm dần; hoà → S rồi Q (giảm dần), rồi chỉ số (ổn định).
func Order(sigs []Signals, scores []float64) []int {
	idx := make([]int, len(sigs))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool {
		x, y := idx[a], idx[b]
		if scores[x] != scores[y] {
			return scores[x] > scores[y]
		}
		if sigs[x].Students != sigs[y].Students {
			return sigs[x].Students > sigs[y].Students
		}
		return sigs[x].Questions > sigs[y].Questions
	})
	return idx
}

// stop: từ nối tiếng Việt / Anh quá thường để làm từ khoá cụm (đã bỏ dấu).
var stop = map[string]bool{ //nolint:gochecknoglobals // bảng bất biến
	"la": true, "va": true, "cua": true, "cho": true, "co": true, "khong": true, "gi": true, "the": true, "nao": true, "nhu": true, "em": true, "minh": true, "thay": true,
	"duoc": true, "trong": true, "mot": true, "cac": true, "nay": true, "do": true, "ve": true, "voi": true, "hoi": true, "xin": true, "a": true, "ah": true, "ko": true,
	"what": true, "is": true, "the_": true, "of": true, "and": true, "how": true, "to": true, "in": true, "ban": true, "tai": true, "sao": true, "bao": true, "nhieu": true,
	"de": true, "khi": true, "thi": true, "ma": true, "nen": true, "can": true, "lam": true, "tu": true, "cu": true, "vay": true, "roi": true, "da": true, "se": true,
}

// Keywords trả tối đa n từ hay gặp nhất (theo số câu chứa từ, bỏ dấu, bỏ từ nối, ≥ 3 ký tự) — tên tạm khi LLM lỗi (AC10).
func Keywords(texts []string, n int) []string {
	df := map[string]int{}
	for _, t := range texts {
		seen := map[string]bool{}
		for _, w := range strings.FieldsFunc(textfold.Fold(t), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }) {
			if len([]rune(w)) < 3 || stop[w] || seen[w] {
				continue
			}
			seen[w] = true
			df[w]++
		}
	}
	type kv struct {
		w string
		n int
	}
	var all []kv
	for w, c := range df {
		all = append(all, kv{w, c})
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].n != all[j].n {
			return all[i].n > all[j].n
		}
		return all[i].w < all[j].w
	})
	var out []string
	for i := 0; i < len(all) && i < n; i++ {
		out = append(out, all[i].w)
	}
	return out
}
