package exam

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/store"
)

// RejectHintText: câu cố định theo lý do loại câu hỏi (FEAT-grading-guidance 4.5, M6). Chỉ có chữ này vào prompt sinh câu — không chữ người dùng.
var RejectHintText = map[string]string{ //nolint:gochecknoglobals // bảng bất biến
	"MO_HO":             "Mỗi câu chỉ một cách hiểu; tránh từ chung chung.",
	"SAI_DAP_AN":        "Kiểm lại đáp án đúng bám sát tài liệu trích dẫn.",
	"TRUNG_LAP":         "Không lặp ý với câu đã có trong ngân hàng.",
	"LAC_DE":            "Chỉ hỏi nội dung có trong tài liệu được chọn.",
	"KHO_DE_BAT_THUONG": "Giữ độ khó đúng mức yêu cầu.",
	"SAI_DINH_DANG":     "Đúng định dạng câu hỏi và số lựa chọn yêu cầu.",
}

// ValidRejectReason: lý do loại hợp lệ (gồm UNSPECIFIED, không có chữ gợi ý).
func ValidRejectReason(r string) bool {
	if r == "UNSPECIFIED" {
		return true
	}
	_, ok := RejectHintText[r]
	return ok
}

// HintGate cho biết cờ QUESTION_HINTS của lớp (grading.GuidanceFlags thoả; khai báo ở đây để exam không import grading).
type HintGate interface {
	On(course uuid.UUID, feature string) bool
}

// Hints dựng khối «Tránh…» cho prompt sinh câu hỏi. Giá trị nil, hoặc Gate nil / cờ tắt → không có khối (prompt như trước 10.6).
type Hints struct {
	Gate       HintGate
	MinRejects int // QUESTION_HINTS_MIN_REJECTS
	WindowDays int // QUESTION_HINTS_WINDOW_DAYS
}

// HintsBlock chọn ≤ 3 lý do nhiều nhất (rows đã có thể chưa sắp: sắp theo số câu giảm, rồi tên) khi tổng ≥ min; trả khối chữ + danh sách lý do đã dùng. Tất định, không LLM.
func HintsBlock(rows []store.QuestionRejectHintsRow, min int) (string, []string) {
	total := 0
	for _, r := range rows {
		if r.RejectReason != nil {
			total += int(r.N)
		}
	}
	if total < min || total == 0 {
		return "", nil
	}
	var reasons []string
	var b strings.Builder
	for _, r := range rows { // SQL đã ORDER BY n DESC, reject_reason
		if len(reasons) == 3 {
			break
		}
		if r.RejectReason == nil {
			continue
		}
		t, ok := RejectHintText[*r.RejectReason]
		if !ok {
			continue
		}
		if len(reasons) == 0 {
			b.WriteString("\n\nTRÁNH (giảng viên thường loại các câu như vậy):")
		}
		fmt.Fprintf(&b, "\n- %s", t)
		reasons = append(reasons, *r.RejectReason)
	}
	return b.String(), reasons
}

// Of tra DB và dựng khối cho một lớp. Lỗi truy vấn → không khối (sinh câu vẫn chạy như cũ).
func (h *Hints) Of(ctx context.Context, db store.DBTX, course uuid.UUID) (string, []string) {
	if h == nil || h.Gate == nil || !h.Gate.On(course, "QUESTION_HINTS") {
		return "", nil
	}
	rows, err := store.New(db).QuestionRejectHints(ctx, store.QuestionRejectHintsParams{CourseID: course, WindowDays: int32(h.WindowDays)}) //nolint:gosec // ≤ 3650
	if err != nil {
		return "", nil
	}
	return HintsBlock(rows, h.MinRejects)
}
