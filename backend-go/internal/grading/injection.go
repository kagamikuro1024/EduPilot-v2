package grading

import (
	"regexp"
	"sort"
	"strings"
	"unicode/utf16"

	"github.com/edupilot/backend-go/internal/textfold"
)

// InjectionPattern: một mẫu nghi chèn lệnh. Regex viết KHÔNG DẤU, chạy trên `Fold(câu)`; MỘT nguồn duy nhất là bảng SRS 4.6.3 = `INJECTION` của fixtures/build_fixtures.py.
type InjectionPattern struct {
	Code string
	Re   *regexp.Regexp
}

// InjectionPatterns (TLR-4: chỉ gắn cờ và tô, KHÔNG xoá câu khỏi đầu vào LLM, không trừ điểm).
var InjectionPatterns = []InjectionPattern{ //nolint:gochecknoglobals // bảng bất biến
	{"IGNORE_PREV_VI", regexp.MustCompile(`bo qua (moi|tat ca) (huong dan|chi dan|yeu cau)`)},
	{"IGNORE_PREV_EN", regexp.MustCompile(`ignore (all |any )?(previous|prior|above) (instructions|prompts?)`)},
	{"GIVE_SCORE_VI", regexp.MustCompile(`(cho|ghi|dat|cham)[^.\n]{0,25}\b(10|muoi)\s*(/\s*10)?\s*diem[^.\n]{0,40}(tat ca|moi tieu chi|bai nay)`)},
	{"GIVE_SCORE_VI_2", regexp.MustCompile(`cho (toi|em|bai nay)[^.\n]{0,15}\b(10|muoi)\s*diem`)},
	{"GIVE_SCORE_EN", regexp.MustCompile(`(give|award|assign)[^.\n]{0,30}(full|maximum|10/10|10 points)`)},
	{"ROLE_TAG", regexp.MustCompile(`(<\s*/?\s*(system|assistant)\s*>|\[\s*(system|inst)\s*\]|###\s*system)`)},
	// P-11-26(c): mẫu bổ sung cho biến thể KHÔNG từ khoá điểm (benchmarks/real-llm/data/injection v1–v5). Chỉ thêm vì 0 cờ nhầm trên bộ sạch
	// (fixtures S1–S5, S7 + 84 bài KC106 + exemplars): xem injection_extra_test.go. Cờ không đổi điểm.
	{"DELIM_SPOOF", regexp.MustCompile(`bai_?lam_[0-9a-z]{6,}\s*>>>|<<<\s*bai_?lam_[0-9a-z]{6,}`)},
	{"SYSTEM_TAG_VI", regexp.MustCompile(`\[\s*he thong\s*\]`)},
	{"NOTE_TO_GRADER_EN", regexp.MustCompile(`note to (the )?(grading |grader)|grading assistant`)},
	{"JSON_OUTPUT", regexp.MustCompile(`"criteria"\s*:\s*\[|"score"\s*:\s*"?[0-9]`)},
	{"INSTRUCTOR_NOTE_VI", regexp.MustCompile(`ghi chu cua giang vien[^.\n]{0,30}da duyet`)},
	{"GRADER_DIRECTIVE_VI", regexp.MustCompile(`tro ly cham (chi can|hay|nen|phai)`)},
	{"MAX_SCORE_ALL_VI", regexp.MustCompile(`muc (toi da|cao nhat) (o|cho|trong) (ca|tung|moi) `)},
}

// InjectionHit: một câu khớp ≥ 1 mẫu, kèm vị trí [Start, End) theo UTF-16 code unit trong văn bản quét (để tô ở màn duyệt).
type InjectionHit struct {
	Sentence string   `json:"sentence"`
	Codes    []string `json:"codes"`
	Start    int      `json:"start"`
	End      int      `json:"end"`
}

type span struct{ s, e int } // chỉ số rune

// sentences tách câu: sau `.`, `!`, `?` rồi khoảng trắng, hoặc ở dòng mới (RE2 không có lookbehind nên tách tay). Trả khoảng rune đã cắt khoảng trắng hai đầu.
func sentences(r []rune) []span {
	var out []span
	start := 0
	flush := func(end int) {
		s, e := start, end
		for s < e && isSpace(r[s]) {
			s++
		}
		for e > s && isSpace(r[e-1]) {
			e--
		}
		if e > s {
			out = append(out, span{s, e})
		}
	}
	for i := 0; i < len(r); i++ {
		switch {
		case r[i] == '\n':
			flush(i)
			start = i + 1
		case (r[i] == '.' || r[i] == '!' || r[i] == '?') && i+1 < len(r) && isSpace(r[i+1]) && r[i+1] != '\n':
			flush(i + 1)
			start = i + 1
		}
	}
	flush(len(r))
	return out
}

func isSpace(c rune) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f' || c == '\v' || c == 0x00a0
}

// ScanInjection quét văn bản; KHÔNG sửa nó. Trả các câu khớp theo thứ tự xuất hiện.
func ScanInjection(text string) []InjectionHit {
	r := []rune(text)
	var hits []InjectionHit
	for _, sp := range sentences(r) {
		sent := string(r[sp.s:sp.e])
		f := strings.Join(strings.Fields(textfold.Fold(sent)), " ")
		var codes []string
		for _, p := range InjectionPatterns {
			if p.Re.MatchString(f) {
				codes = append(codes, p.Code)
			}
		}
		if len(codes) == 0 {
			continue
		}
		sort.Strings(codes)
		u := len(utf16.Encode(r[:sp.s]))
		hits = append(hits, InjectionHit{Sentence: sent, Codes: codes, Start: u, End: u + len(utf16.Encode([]rune(sent)))})
	}
	return hits
}

// InjectionCodes gộp mã mẫu của mọi câu khớp (đã sắp, không trùng).
func InjectionCodes(hits []InjectionHit) []string {
	var all []string
	for _, h := range hits {
		all = append(all, h.Codes...)
	}
	return uniqSorted(all)
}
