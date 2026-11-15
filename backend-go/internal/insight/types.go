package insight

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// Payload là `insight_reports.payload` (SRS 5): không `student_id` / tên / MSSV, chỉ số đếm + chữ đã che.
// PayloadSchema: phiên bản hợp đồng của payload. 2 = sau P-11-31/32/34 (k-ẩn danh theo câu, không tin khủng hoảng, có guardLeaks). Báo cáo không có schema (v1) không được đọc, xuất, dùng lại.
const PayloadSchema = 2

type Payload struct {
	Schema     int         `json:"schema"`
	Topics     []Topic     `json:"topics"`
	Uncovered  []Uncovered `json:"uncovered"`
	Counts     Counts      `json:"counts"`
	PeriodDays int         `json:"period_days"`

	below belowSet // cụm từ của câu dưới ngưỡng k, để kiểm đầu ra LLM; không lưu
}

// Counts: tổng quan báo cáo.
type Counts struct {
	Questions int `json:"questions"`
	Students  int `json:"students"`
	Clusters  int `json:"clusters"`
}

// Topic là một chủ đề đã xếp hạng.
type Topic struct {
	Rank           int          `json:"rank"`
	Name           string       `json:"name"`
	Why            string       `json:"why"`
	Score          float64      `json:"score"`
	Signals        TopicSignals `json:"signals"`
	Recommendation string       `json:"recommendation"`
	Examples       []string     `json:"examples"`
	FewAsked       bool         `json:"few_asked,omitempty"` // < 3 sinh viên: không câu mẫu
	NamedBy        string       `json:"named_by"`            // "llm" | "fallback"
	Chapter        *string      `json:"chapter"`
	Week           *int         `json:"week"`
	Keywords       []string     `json:"keywords"`
	samples        []string     // đầu vào đặt tên, không lưu
}

// TopicSignals: số liệu của một chủ đề.
type TopicSignals struct {
	Students          int      `json:"students"`
	Questions         int      `json:"questions"`
	LowRate           float64  `json:"low_confidence_rate"`
	NotHelpfulRate    float64  `json:"not_helpful_rate"`
	PracticeWrongRate *float64 `json:"practice_wrong_rate"` // null khi lớp chưa dùng luyện đề (trọng số của P chia đều, SRS 4.1)
}

// Uncovered: câu hỏi chưa có trong tài liệu.
type Uncovered struct {
	Text  string `json:"text"`
	Count int    `json:"count"`
}

func dateStr(d pgtype.Date) *string {
	if !d.Valid {
		return nil
	}
	s := d.Time.Format("2006-01-02")
	return &s
}

func dateIn(t time.Time) pgtype.Date {
	return pgtype.Date{Time: t, Valid: true}
}
