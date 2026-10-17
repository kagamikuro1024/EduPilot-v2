package today

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// AttentionWindow: sinh viên vào AT_RISK trong chừng này gần nhất thì lớp có mục STUDENT_ATTENTION (SRS FEAT-crm-360 4.9).
const AttentionWindow = 7 * 24 * time.Hour

// AttentionMax: số sinh viên tối đa của khối `attention`.
const AttentionMax = 5

// AttentionRow là một sinh viên AT_RISK trong khối `attention`. Reason là cụm ngắn dựng từ số liệu ("vắng 3/4 buổi; học ít"): không điểm, không nội dung chat.
type AttentionRow struct {
	Student attentionStudent `json:"student"`
	Course  CourseRef        `json:"course"`
	Reason  string           `json:"reason"`
}

type attentionStudent struct {
	ID       uuid.UUID `json:"id"`
	FullName string    `json:"full_name"`
}

// NewAttentionRow dựng một dòng cho nguồn cài đặt (internal/crm).
func NewAttentionRow(studentID uuid.UUID, fullName string, courseID uuid.UUID, classCode, reason string) AttentionRow {
	return AttentionRow{Student: attentionStudent{ID: studentID, FullName: fullName}, Course: CourseRef{ID: courseID, ClassCode: classCode}, Reason: reason}
}

// AttentionSnapshot là kết quả MỘT truy vấn cho cả mục STUDENT_ATTENTION lẫn khối `attention`.
type AttentionSnapshot struct {
	NewByCourse map[uuid.UUID]int // sinh viên mới vào AT_RISK từ `since`, theo lớp (lớp không có thì vắng khỏi map)
	Rows        []AttentionRow    // ≤ limit sinh viên AT_RISK, mới vào trước
	Total       int               // tổng sinh viên AT_RISK trong phạm vi
}

// AttentionSource là nguồn sinh viên AT_RISK. Cài ở internal/crm (chỉ gói đó đọc `student_risk`: sinh viên không bao giờ thấy nhãn rủi ro). nil = không có mục / khối.
// Snapshot là MỘT truy vấn: "Hôm nay" của Staff giữ ngân sách ≤ 6 truy vấn kể cả guard.
type AttentionSource interface {
	Snapshot(ctx context.Context, courseIDs []uuid.UUID, since time.Time, limit int) (AttentionSnapshot, error)
}

// WithAttention đăng ký nguồn AT_RISK cho mục STUDENT_ATTENTION (bậc 95) và khối `attention` (US-P5-04).
func WithAttention(src AttentionSource) Option { return func(p *StaffProvider) { p.Attn = src } }

// attentionItems dựng STUDENT_ATTENTION của các lớp có sinh viên mới vào AT_RISK ≤ 7 ngày, từ ảnh chụp của yêu cầu này (`Viewer.attn`).
func attentionItems(snap *AttentionSnapshot, courses []CourseRef) []Item {
	if snap == nil {
		return nil
	}
	var out []Item
	for _, c := range courses {
		if k := snap.NewByCourse[c.ID]; k > 0 {
			out = append(out, mk(Item{ID: "STUDENT_ATTENTION:" + c.ID.String(), Kind: KindStudentAttention, Tier: TierStudentAttention, Course: courseRef(c),
				Href: fmt.Sprintf("/students?filter=attention&course=%s", c.ID), Title: fmt.Sprintf("%d sinh viên mới vào diện cần chú ý · %s", k, c.ClassCode), Reason: ""}))
		}
	}
	return out
}
