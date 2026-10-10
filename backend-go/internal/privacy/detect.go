// Package privacy: nhận diện, ẩn (kênh công khai), che / khôi phục (quanh lời gọi LLM) thông tin cá nhân; phân loại kênh (SRS FEAT-private-chat-pii 4.2–4.4).
// Chỉ regex (RE2, tuyến tính) + từ điển roster theo lớp; không NER (D46). Không bao giờ log văn bản thô hay ánh xạ.
package privacy

import (
	"context"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
)

// Kind là loại thông tin cá nhân.
type Kind string

// Năm loại (khớp enum pii_kind, phần nhận diện được).
const (
	KindMSSV  Kind = "MSSV"
	KindEmail Kind = "EMAIL"
	KindPhone Kind = "PHONE"
	KindCCCD  Kind = "CCCD"
	KindName  Kind = "NAME"
)

// Finding là một khoảng PII; Start, End là chỉ số rune trong văn bản gốc (End không gồm).
type Finding struct {
	Kind       Kind
	Start, End int
}

// match là Finding kèm khoá thực thể (dùng cho Mask: cùng thực thể ⇒ cùng placeholder).
type match struct {
	Finding
	key string
}

// maxDetectRunes: văn bản dài hơn được quét theo cửa sổ trượt (mỗi cửa sổ ≤ maxDetectRunes, chồng nhau windowOverlap rune).
const (
	maxDetectRunes = 20000
	windowOverlap  = 256
)

var (
	reEmail = regexp.MustCompile(`[A-Za-z0-9._%+-]+@[A-Za-z0-9-]+(?:\.[A-Za-z0-9-]+)+`)
	// Đầu số di động 03/05/07/08/09; +84 / 84 / 0; ranh giới số ở hai đầu để không cắt giữa dãy số dài hơn.
	rePhone     = regexp.MustCompile(`(?:\+84|\b84|\b0)[\s.-]?[35789](?:[\s.-]?\d){8}\b`)
	reCCCDPlain = regexp.MustCompile(`\b\d{12}\b`)
	reCCCDGroup = regexp.MustCompile(`\b\d{3}[ .-]\d{3}[ .-]\d{3}[ .-]\d{3}\b`)
	reMSSV8     = regexp.MustCompile(`\b20\d{6}\b`)
	reMSSVAlpha = regexp.MustCompile(`\b[A-Za-z]\d{2}[A-Za-z]{4}\d{3}\b`)
	// Sau cụm "MSSV" / "mã số sinh viên": một mã 6–15 ký tự chữ-số có ít nhất một chữ số (lọc ở Go vì RE2 không có lookahead).
	reMSSVCue = regexp.MustCompile(`(?i)(?:mssv|mã số sinh viên)\s*[:\-]?\s*([A-Za-z0-9]{6,15})\b`)
)

// Detector nhận diện PII. Roster nil = chỉ regex.
type Detector struct{ Roster *Roster }

// Detect trả các khoảng PII (đã bỏ chồng lấn, theo thứ tự xuất hiện). Lỗi chỉ khi nạp roster hỏng.
func (d *Detector) Detect(ctx context.Context, courseID uuid.UUID, text string) ([]Finding, error) {
	ms, err := d.detect(ctx, courseID, text)
	if err != nil {
		return nil, err
	}
	if len(ms) == 0 {
		return nil, nil
	}
	fs := make([]Finding, len(ms))
	for i, m := range ms {
		fs[i] = m.Finding
	}
	return fs, nil
}

func (d *Detector) detect(ctx context.Context, courseID uuid.UUID, text string) ([]match, error) {
	if text == "" {
		return nil, nil
	}
	var idx *rosterIndex
	if d != nil && d.Roster != nil && courseID != uuid.Nil {
		var err error
		if idx, err = d.Roster.index(ctx, courseID); err != nil {
			return nil, err
		}
	}
	return detectIn(text, idx), nil
}

// detectIn quét một văn bản (cắt cửa sổ khi quá dài).
func detectIn(text string, idx *rosterIndex) []match {
	if utf8.RuneCountInString(text) <= maxDetectRunes {
		return resolve(scan(text, idx))
	}
	runes := []rune(text)
	var all []match
	for start := 0; start < len(runes); start += maxDetectRunes - windowOverlap {
		end := min(start+maxDetectRunes, len(runes))
		for _, m := range scan(string(runes[start:end]), idx) {
			m.Start += start
			m.End += start
			all = append(all, m)
		}
		if end == len(runes) {
			break
		}
	}
	return resolve(all)
}

// prio: khi hai khoảng trùng hệt nhau, loại đứng trước thắng.
func prio(k Kind) int {
	switch k {
	case KindEmail:
		return 0
	case KindPhone:
		return 1
	case KindCCCD:
		return 2
	case KindMSSV:
		return 3
	}
	return 4
}

func scan(text string, idx *rosterIndex) []match {
	var out []match
	add := func(re *regexp.Regexp, k Kind, group int, norm func(string) string, ok func(string) bool) {
		for _, loc := range re.FindAllStringSubmatchIndex(text, -1) {
			s, e := loc[2*group], loc[2*group+1]
			if s < 0 {
				continue
			}
			raw := text[s:e]
			if ok != nil && !ok(raw) {
				continue
			}
			out = append(out, match{Finding: Finding{Kind: k, Start: s, End: e}, key: norm(raw)}) // chỉ số BYTE, đổi sang rune ở cuối
		}
	}
	if strings.IndexByte(text, '@') >= 0 {
		add(reEmail, KindEmail, 0, strings.ToLower, nil)
	}
	if hasDigit(text) {
		add(rePhone, KindPhone, 0, normPhone, nil)
		add(reCCCDPlain, KindCCCD, 0, digitsOnly, nil)
		add(reCCCDGroup, KindCCCD, 0, digitsOnly, nil)
		add(reMSSV8, KindMSSV, 0, strings.ToLower, nil)
		add(reMSSVAlpha, KindMSSV, 0, strings.ToLower, nil)
		add(reMSSVCue, KindMSSV, 1, strings.ToLower, hasDigit)
	}
	// chuyển chỉ số byte → rune
	if len(out) > 0 {
		out = toRunes(text, out)
	}
	if idx != nil {
		out = append(out, idx.find(text)...)
	}
	return out
}

// toRunes đổi Start/End từ byte sang rune (một lượt qua văn bản, các khoảng sắp theo Start).
func toRunes(text string, ms []match) []match {
	sort.Slice(ms, func(i, j int) bool { return ms[i].Start < ms[j].Start })
	// tập điểm cần đổi
	pts := make([]int, 0, 2*len(ms))
	for _, m := range ms {
		pts = append(pts, m.Start, m.End)
	}
	sort.Ints(pts)
	conv := make(map[int]int, len(pts))
	r, b := 0, 0
	for _, p := range pts {
		if _, done := conv[p]; done {
			continue
		}
		r += utf8.RuneCountInString(text[b:p])
		b = p
		conv[p] = r
	}
	for i := range ms {
		ms[i].Start, ms[i].End = conv[ms[i].Start], conv[ms[i].End]
	}
	return ms
}

// resolve bỏ chồng lấn: khoảng bắt đầu sớm hơn thắng; cùng điểm đầu thì dài hơn, rồi theo prio.
func resolve(ms []match) []match {
	if len(ms) < 2 {
		return ms
	}
	sort.Slice(ms, func(i, j int) bool {
		a, b := ms[i], ms[j]
		if a.Start != b.Start {
			return a.Start < b.Start
		}
		if a.End != b.End {
			return a.End > b.End
		}
		return prio(a.Kind) < prio(b.Kind)
	})
	out := ms[:0:0]
	last := -1
	for _, m := range ms {
		if m.Start >= last {
			out = append(out, m)
			last = m.End
		}
	}
	return out
}

func hasDigit(s string) bool { return strings.ContainsAny(s, "0123456789") }

func digitsOnly(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// normPhone: chỉ chữ số, đầu 84 → 0 (+84 912… và 0912… là cùng một số).
func normPhone(s string) string {
	d := digitsOnly(s)
	if strings.HasPrefix(d, "84") {
		return "0" + d[2:]
	}
	return d
}

// isWordRune: chữ, số hoặc dấu kết hợp (tiếng Việt gõ ở dạng NFD).
func isWordRune(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsMark(r) }
