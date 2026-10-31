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
	// TypeAppealNew: chuông GỘP cho TEACHER ("n yêu cầu xem lại mới"), một dòng mỗi (người nhận, lớp).
	TypeAppealNew = "APPEAL_NEW"
	// TypeAppealReplied: chuông cá nhân cho sinh viên khi phúc khảo đã được quyết định.
	TypeAppealReplied = "APPEAL_REPLIED"
	// MailAppealReply là mẫu thư phản hồi phúc khảo.
	MailAppealReply = "appeal_reply"
)

type appealEvent struct {
	CourseID uuid.UUID `json:"course_id"`
	AppealID uuid.UUID `json:"appeal_id"`
}

// RecountAppealNew đếm lại phúc khảo OPEN của lớp và cập nhật dòng gộp APPEAL_NEW của TEACHER (TA không nhận; không một chuông mỗi yêu cầu).
func (s *Service) RecountAppealNew(ctx context.Context, course uuid.UUID) error {
	if s.Notify == nil {
		return nil
	}
	teachers, err := store.New(s.Pool).GradeAppealTeachers(ctx, course)
	if err != nil {
		return fmt.Errorf("assignment: giảng viên của lớp: %w", err)
	}
	return s.Notify.Aggregate(ctx, notify.Agg{CourseID: course, Type: TypeAppealNew, Recipients: teachers, Link: "/grading",
		Title: func(n int) string { return fmt.Sprintf("%d yêu cầu xem lại mới", n) },
		Count: func(ctx context.Context, q *store.Queries) (int, error) {
			n, err := q.GradeAppealOpenCount(ctx, course)
			return int(n), err
		}})
}

// HandleAppealCreated là outbox.Handler của `assignment.appeal_created`: cập nhật chuông gộp của TEACHER.
func (s *Service) HandleAppealCreated(ctx context.Context, m outbox.Message) error {
	var p appealEvent
	if err := json.Unmarshal(m.Payload, &p); err != nil || p.CourseID == uuid.Nil {
		return fmt.Errorf("assignment: payload %s hỏng", m.Topic)
	}
	return s.RecountAppealNew(ctx, p.CourseID)
}

type appealMailPayload struct {
	StudentName     string `json:"student_name"`
	AssignmentID    string `json:"assignment_id"`
	AssignmentTitle string `json:"assignment_title"`
	Decision        string `json:"decision"`
	ScoreBefore     string `json:"score_before"`
	ScoreAfter      string `json:"score_after,omitempty"`
	Response        string `json:"response"`
}

// HandleAppealDecided là outbox.Handler của `assignment.appeal_decided`: chuông APPEAL_REPLIED + thư `appeal_reply` tới email ĐÃ XÁC MINH của đúng sinh viên (một giao dịch, khử trùng
// `assignment.appeal_decided:{id}`), rồi đếm lại chuông gộp của TEACHER. Không gọi LLM.
func (s *Service) HandleAppealDecided(ctx context.Context, m outbox.Message) error {
	var p appealEvent
	if err := json.Unmarshal(m.Payload, &p); err != nil || p.CourseID == uuid.Nil || p.AppealID == uuid.Nil {
		return fmt.Errorf("assignment: payload %s hỏng", m.Topic)
	}
	var row notify.Row
	var created bool
	err := s.tx(ctx, func(tx pgx.Tx, q *store.Queries) error {
		ap, err := q.GradeAppealGet(ctx, store.GradeAppealGetParams{CourseID: p.CourseID, ID: p.AppealID})
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && ap.Status == "OPEN") {
			return nil
		}
		if err != nil {
			return fmt.Errorf("assignment: đọc phúc khảo: %w", err)
		}
		u, err := q.GetUser(ctx, ap.StudentID)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("assignment: đọc người nhận: %w", err)
		}
		course := p.CourseID
		row, created, err = notify.SendTx(ctx, q, notify.Spec{UserID: u.ID, CourseID: &course, Type: TypeAppealReplied, Title: fmt.Sprintf("Giảng viên đã trả lời yêu cầu xem lại điểm %s", ap.AssignmentTitle),
			Link: new("/assignments/" + ap.AssignmentID.String()), DedupeKey: "assignment.appeal_decided:" + ap.ID.String()})
		if err != nil {
			return err
		}
		if u.EmailVerifiedAt == nil {
			return nil
		}
		mp := appealMailPayload{StudentName: u.FullName, AssignmentID: ap.AssignmentID.String(), AssignmentTitle: ap.AssignmentTitle, Decision: ap.Status, ScoreBefore: ap.ScoreBefore.StringFixed(2)}
		if ap.ScoreAfter.Valid {
			mp.ScoreAfter = ap.ScoreAfter.Decimal.StringFixed(2)
		}
		if ap.Response != nil {
			mp.Response = *ap.Response
		}
		raw, _ := json.Marshal(mp)
		var payload map[string]any
		_ = json.Unmarshal(raw, &payload)
		if _, _, err := mail.Enqueue(ctx, tx, mail.Message{To: u.Email, Template: MailAppealReply, Payload: payload, DedupeKey: MailAppealReply + ":" + ap.ID.String()}); err != nil {
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
	return s.RecountAppealNew(ctx, p.CourseID)
}

// resolveAppealReply dựng biến hiển thị của thư phúc khảo LÚC GỬI: quyết định, điểm cũ → mới (nếu sửa), phản hồi NGUYÊN VĂN của giảng viên.
func resolveAppealReply(publicURL string, row store.MailOutbox) (map[string]string, error) {
	var p appealMailPayload
	if err := json.Unmarshal(row.Payload, &p); err != nil || p.AssignmentID == "" || p.Decision == "" {
		return nil, mail.Permanent("payload_invalid")
	}
	vi := func(s string) string { return strings.Replace(s, ".", ",", 1) }
	decision := "Giữ nguyên điểm"
	var b strings.Builder
	if p.Decision == "ADJUSTED" {
		decision = "Đã sửa điểm"
		if a, err1 := decimal.NewFromString(p.ScoreBefore); err1 == nil {
			if n, err2 := decimal.NewFromString(p.ScoreAfter); err2 == nil {
				fmt.Fprintf(&b, "Điểm: %s → %s\n\n", vi(a.StringFixed(2)), vi(n.StringFixed(2)))
			}
		}
	}
	b.WriteString("Phản hồi của giảng viên: " + strings.TrimSpace(p.Response))
	return map[string]string{"FullName": p.StudentName, "AssignmentTitle": p.AssignmentTitle, "Decision": decision, "Details": b.String(), "AssignmentURL": publicURL + "/assignments/" + p.AssignmentID}, nil
}
