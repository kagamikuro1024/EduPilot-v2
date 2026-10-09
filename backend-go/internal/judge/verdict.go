package judge

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// Verdict là kết quả chấm một test hoặc cả bản nộp (`judge_verdict`).
type Verdict string

const (
	AC  Verdict = "AC"
	WA  Verdict = "WA"
	TLE Verdict = "TLE"
	MLE Verdict = "MLE"
	RE  Verdict = "RE"
	CE  Verdict = "CE"
	OLE Verdict = "OLE"
	IE  Verdict = "IE"
)

// Chuỗi `status` của go-judge (đã thấy ở PoC).
const (
	statusAccepted = "Accepted"
	statusTLE      = "Time Limit Exceeded"
	statusMLE      = "Memory Limit Exceeded"
	statusOLE      = "Output Limit Exceeded"
	statusNonzero  = "Nonzero Exit Status"
	statusSignal   = "Signalled"
	statusDanger   = "Dangerous Syscall"
	statusInternal = "Internal Error"
	statusFile     = "File Error"
)

// MapRunStatus ánh xạ `status` của một lệnh CHẠY sang verdict (SRS 4.5.4). `Accepted` trả AC — người gọi còn phải so khớp (WA).
// Chuỗi lạ / lỗi hệ thống → IE (không tính điểm, thử lại); KHÔNG bao giờ biến lỗi hệ thống thành RE / WA.
func MapRunStatus(status string) Verdict {
	switch status {
	case statusAccepted:
		return AC
	case statusTLE:
		return TLE
	case statusMLE:
		return MLE
	case statusOLE:
		return OLE
	case statusNonzero, statusSignal, statusDanger:
		return RE
	default: // Internal Error, File Error, chuỗi không biết
		return IE
	}
}

// MaxCompileLog là cỡ tối đa của `compile_log` lưu DB.
const MaxCompileLog = 8192

var (
	reAbsCpp = regexp.MustCompile(`\S*/a\.cc`)
	reAbsC   = regexp.MustCompile(`\S*/a\.c\b`)
	reCc     = regexp.MustCompile(`\ba\.cc\b`)
	reC      = regexp.MustCompile(`\ba\.c\b`)
	reAnyAbs = regexp.MustCompile(`(^|[\s'"(])/(?:[A-Za-z0-9_.+-]+/)+[A-Za-z0-9_.+-]*`)
)

// CompileLog dựng `compile_log`: thay đường dẫn tuyệt đối và tên tệp nội bộ bằng `main.c` / `main.cpp`, cắt ≤ 8 KiB (không cắt giữa ký tự UTF-8).
// Bị giết khi biên dịch (hết giờ / bộ nhớ) → câu cố định.
func CompileLog(status, stderr string, lang Language) string {
	main := "main.cpp"
	if lang == C11 {
		main = "main.c"
	}
	switch status {
	case statusTLE:
		return "Biên dịch quá thời gian."
	case statusMLE:
		return "Biên dịch quá bộ nhớ."
	}
	s := stderr
	s = reAbsCpp.ReplaceAllString(s, main)
	s = reAbsC.ReplaceAllString(s, main)
	s = reCc.ReplaceAllString(s, main)
	s = reC.ReplaceAllString(s, main)
	s = reAnyAbs.ReplaceAllString(s, "${1}<đường dẫn>")
	s = strings.ToValidUTF8(s, "")
	return truncateUTF8(s, MaxCompileLog)
}

func truncateUTF8(s string, n int) string {
	if len(s) <= n {
		return s
	}
	s = s[:n]
	for !utf8.ValidString(s) && len(s) > 0 {
		s = s[:len(s)-1]
	}
	return s
}
