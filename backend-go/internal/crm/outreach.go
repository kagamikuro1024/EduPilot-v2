package crm

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/escalation"
	"github.com/edupilot/backend-go/internal/store"
)

// OutreachResult là ticket vừa tạo bởi `Nhắn sinh viên`.
type OutreachResult struct {
	TicketID  uuid.UUID
	Status    string
	CreatedAt time.Time
}

// Outreach là `Nhắn sinh viên` (SRS 4.8.7): MỘT giao dịch — escalation.CreateOutreach (ticket TEACHER_OUTREACH đã ANSWERED + tin STAFF_ANSWER + thư + outbox) và `audit_log` `outreach`
// (id ticket, KHÔNG nội dung). Idempotency-Key do middleware của route lo. Sinh viên không ACTIVE → 409; PII của sinh viên khác → 422; quá OUTREACH_PER_DAY → 429.
// Sinh viên KHÔNG nhận nhãn / lý do rủi ro ở bất kỳ nơi nào: thân tin do nhân viên gõ, không có chữ nào của hệ thống thêm vào.
func (s *Service) Outreach(ctx context.Context, actor, courseID, studentID uuid.UUID, body string) (OutreachResult, error) {
	if s.Tickets == nil {
		return OutreachResult{}, unavailable()
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return OutreachResult{}, wrap("mở giao dịch nhắn sinh viên", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	t, _, err := s.Tickets.CreateOutreach(ctx, tx, escalation.OutreachIn{CourseID: courseID, StudentID: studentID, SenderID: actor, Body: body, PerDay: s.OutreachPerDay, Now: s.now()})
	if err != nil {
		return OutreachResult{}, err
	}
	if err := audit(ctx, store.New(tx), "outreach", courseID, actor, t.ID.String(), "send", nil, map[string]any{"student_id": studentID}); err != nil {
		return OutreachResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return OutreachResult{}, wrap("commit nhắn sinh viên", err)
	}
	return OutreachResult{TicketID: t.ID, Status: string(t.Status), CreatedAt: t.CreatedAt}, nil
}
