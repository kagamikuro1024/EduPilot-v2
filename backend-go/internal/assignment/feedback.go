package assignment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/mail"
	"github.com/edupilot/backend-go/internal/notify"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/store"
)

const (
	// TypeGradePublished là loại chuông "Điểm … đã được công bố".
	TypeGradePublished = "GRADE_PUBLISHED"
	// MailFeedback là mẫu thư nhận xét bài tập.
	MailFeedback = "assignment_feedback"
)

// feedbackPayload là payload của `mail_outbox` mẫu assignment_feedback: CHỈ thứ sinh viên được thấy (không điểm nháp, cờ, độ tin cậy, lượt chấm) và không khoá link / token.
type feedbackPayload struct {
	StudentName     string       `json:"student_name"`
	AssignmentID    string       `json:"assignment_id"`
	AssignmentTitle string       `json:"assignment_title"`
	Total           string       `json:"total"`
	Max             string       `json:"max"`
	Criteria        []feedbackCr `json:"criteria"`
	OverallFeedback string       `json:"overall_feedback"`
	LatePenalty     string       `json:"late_penalty,omitempty"`
}

type feedbackCr struct {
	Name    string `json:"name"`
	Score   string `json:"score"`
	Max     string `json:"max"`
	Comment string `json:"comment"`
}

// HandleResultPublished là outbox.Handler của `assignment.result_published`: chuông GRADE_PUBLISHED + thư `assignment_feedback` tới email ĐÃ XÁC MINH của đúng người nộp, trong MỘT giao dịch (khử trùng nên xử lý lại không gửi đôi).
func (s *Service) HandleResultPublished(ctx context.Context, m outbox.Message) error {
	var p struct {
		CourseID string `json:"course_id"`
		ResultID string `json:"result_id"`
	}
	if err := json.Unmarshal(m.Payload, &p); err != nil {
		return fmt.Errorf("assignment: payload %s: %w", m.Topic, err)
	}
	course, e1 := uuid.Parse(p.CourseID)
	rid, e2 := uuid.Parse(p.ResultID)
	if e1 != nil || e2 != nil {
		return nil
	}
	var row notify.Row
	var created bool
	err := s.tx(ctx, func(tx pgx.Tx, q *store.Queries) error {
		d, err := q.ReviewDetail(ctx, store.ReviewDetailParams{CourseID: course, ID: rid})
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && (d.Status != "PUBLISHED" || d.StudentID == nil)) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("assignment: đọc kết quả: %w", err)
		}
		v, err := viewOf(detailAsLock(d))
		if err != nil {
			return err
		}
		u, err := q.GetUser(ctx, *d.StudentID)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("assignment: đọc người nhận: %w", err)
		}
		aid := ptrOr(d.AssignmentID, uuid.Nil)
		row, created, err = notify.SendTx(ctx, q, notify.Spec{UserID: u.ID, CourseID: &course, Type: TypeGradePublished, Title: fmt.Sprintf("Điểm %s đã được công bố", d.Title),
			Link: new("/assignments/" + aid.String()), DedupeKey: "assignment.published:" + rid.String() + ":" + u.ID.String()})
		if err != nil {
			return err
		}
		if u.EmailVerifiedAt == nil {
			return nil // chỉ gửi tới email đã xác minh
		}
		fp := feedbackPayload{StudentName: u.FullName, AssignmentID: aid.String(), AssignmentTitle: d.Title, Max: v.Assignment.MaxScore, OverallFeedback: v.Result.OverallFeedback}
		if v.Result.Total != nil {
			fp.Total = *v.Result.Total
		}
		for _, c := range v.Criteria {
			sc := ""
			if c.Score != nil {
				sc = *c.Score
			}
			fp.Criteria = append(fp.Criteria, feedbackCr{Name: c.Name, Score: sc, Max: fix2(c.Max), Comment: c.Comment})
		}
		if d.LatePenalty.IsPositive() {
			fp.LatePenalty = d.LatePenalty.StringFixed(2)
		}
		raw, _ := json.Marshal(fp)
		var payload map[string]any
		_ = json.Unmarshal(raw, &payload)
		if _, _, err := mail.Enqueue(ctx, tx, mail.Message{To: u.Email, Template: MailFeedback, Payload: payload, DedupeKey: MailFeedback + ":" + rid.String()}); err != nil {
			return fmt.Errorf("assignment: xếp thư: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}
	if created && s.Notify != nil {
		s.Notify.Announce(ctx, row)
	}
	return nil
}

// MailResolver dựng biến hiển thị của thư nhận xét LÚC GỬI từ payload (đường dẫn dựng từ cấu hình, không nằm ở hàng đợi).
type MailResolver struct{ PublicURL string }

// Register gắn resolver vào bảng của mail.Handler.
func (m MailResolver) Register(into map[string]mail.Resolver) { into[MailFeedback] = m }

// Resolve implements mail.Resolver.
func (m MailResolver) Resolve(_ context.Context, _ pgx.Tx, row store.MailOutbox) (map[string]string, error) {
	var p feedbackPayload
	if err := json.Unmarshal(row.Payload, &p); err != nil || p.AssignmentID == "" || p.Total == "" {
		return nil, mail.Permanent("payload_invalid")
	}
	vi := func(s string) string { return strings.Replace(s, ".", ",", 1) }
	var b strings.Builder
	for _, c := range p.Criteria {
		fmt.Fprintf(&b, "• %s: %s/%s", c.Name, vi(c.Score), vi(c.Max))
		if c.Comment != "" {
			b.WriteString(" — " + c.Comment)
		}
		b.WriteString("\n")
	}
	if p.LatePenalty != "" {
		fmt.Fprintf(&b, "Trừ nộp muộn: −%s điểm\n", vi(p.LatePenalty))
	}
	if strings.TrimSpace(p.OverallFeedback) != "" {
		b.WriteString("\nNhận xét chung: " + strings.TrimSpace(p.OverallFeedback))
	}
	details := strings.TrimSpace(b.String())
	if details == "" {
		details = "Điểm đã được công bố."
	}
	return map[string]string{"FullName": p.StudentName, "AssignmentTitle": p.AssignmentTitle, "Total": vi(p.Total), "Max": vi(p.Max), "Details": details,
		"AssignmentURL": m.PublicURL + "/assignments/" + p.AssignmentID}, nil
}

// fix2 chuẩn hoá số điểm về 2 chữ số thập phân (4 → 4.00); không đọc được thì giữ nguyên.
func fix2(s string) string {
	d, err := decimal.NewFromString(s)
	if err != nil {
		return s
	}
	return d.StringFixed(2)
}
