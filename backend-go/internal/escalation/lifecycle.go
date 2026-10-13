package escalation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/store"
)

// Topic của vòng đời (SRS 4.11) phát bởi sinh viên và job SLA.
const (
	TopicFollowup     = "escalation.followup"
	TopicSLAReminder  = "escalation.sla_reminder"
	TopicSLAEscalated = "escalation.sla_escalated"
	TopicCourseStall  = "escalation.course_stalled"
)

// FollowupMaxChars là FOLLOWUP_MAX_CHARS.
const FollowupMaxChars = 2000

// StudentTicketResult là kết quả của hỏi tiếp / xác nhận: ticket (bốn trường sinh viên được thấy) + tin vừa thêm (nếu có).
type StudentTicketResult struct {
	Ticket    StudentView `json:"ticket"`
	MessageID *uuid.UUID  `json:"message_id,omitempty"`
}

// owned nạp ticket của sinh viên gọi và kiểm vai / tư cách thành viên lớp (sinh viên khác → 404, vai khác STUDENT → 403).
func (s *Service) owned(ctx context.Context, a Actor, id uuid.UUID) (store.EscalationTicket, error) {
	if a.Role != auth.RoleStudent {
		return store.EscalationTicket{}, forbidden()
	}
	t, err := store.New(s.Pool).TicketOwnedByStudent(ctx, store.TicketOwnedByStudentParams{ID: id, StudentID: a.UserID})
	if errors.Is(err, pgx.ErrNoRows) {
		return store.EscalationTicket{}, notFound()
	}
	if err != nil {
		return store.EscalationTicket{}, unavailable()
	}
	ms, err := s.Members.Resolve(ctx, auth.Principal{Sub: a.UserID.String(), Role: a.Role}, t.CourseID)
	if err != nil {
		return store.EscalationTicket{}, unavailable()
	}
	if !ms.Found || ms.Status != "ACTIVE" || ms.Role != auth.RoleStudent {
		return store.EscalationTicket{}, forbidden()
	}
	return t, nil
}

func (s *Service) stateErr(t store.EscalationTicket) error {
	if t.Status == store.TicketStatusCLOSED && t.ClosedAt != nil && s.now().Sub(*t.ClosedAt) > time.Duration(s.Cfg.reopenDays())*24*time.Hour {
		return apierr.New(http.StatusConflict, apierr.TicketReopenExpired)
	}
	return apierr.New(http.StatusConflict, apierr.TicketStateInvalid)
}

// Followup là `Hỏi tiếp` (SRS 4.2.5): mở lại ticket ANSWERED (hoặc CLOSED chưa quá TICKET_REOPEN_DAYS), đặt lại đồng hồ SLA (`waiting_since = now`), thêm tin ESCALATION_FOLLOWUP.
// Gửi lại cùng client_msg_id → cùng kết quả, không thêm tin.
func (s *Service) Followup(ctx context.Context, a Actor, ticketID uuid.UUID, content string, clientMsgID uuid.UUID) (StudentTicketResult, error) {
	content = strings.TrimSpace(content)
	if n := utf8.RuneCountInString(content); n == 0 || n > FollowupMaxChars {
		return StudentTicketResult{}, apierr.Validation(apierr.FieldError{Field: "content", Code: "length", Message: fmt.Sprintf("Câu hỏi tiếp dài từ 1 đến %d ký tự.", FollowupMaxChars)})
	}
	if clientMsgID == uuid.Nil {
		return StudentTicketResult{}, apierr.Validation(apierr.FieldError{Field: "client_msg_id", Code: "required", Message: "Thiếu mã tin."})
	}
	t, err := s.owned(ctx, a, ticketID)
	if err != nil {
		return StudentTicketResult{}, err
	}
	q0 := store.New(s.Pool)
	if st, err := q0.ChatCourseStatus(ctx, t.CourseID); err != nil {
		return StudentTicketResult{}, unavailable()
	} else if st == store.CourseStatusARCHIVED {
		return StudentTicketResult{}, apierr.New(http.StatusConflict, apierr.CourseArchived)
	}
	if s.Lock != nil {
		if lk, locked, err := s.Lock.IsLocked(ctx, a.UserID); err != nil {
			return StudentTicketResult{}, unavailable()
		} else if locked {
			return StudentTicketResult{}, apierr.New(http.StatusConflict, apierr.ExamInProgress).WithDetails(map[string]any{"until": lk.Until.UTC().Format(time.RFC3339)})
		}
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return StudentTicketResult{}, unavailable()
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	sess, err := s.ticketSession(ctx, q, t)
	if err != nil {
		return StudentTicketResult{}, err
	}
	if id, err := q.ChatMessageByClientID(ctx, store.ChatMessageByClientIDParams{SessionID: sess, ClientMsgID: &clientMsgID}); err == nil {
		cur, e := q.TicketOwnedByStudent(ctx, store.TicketOwnedByStudentParams{ID: ticketID, StudentID: a.UserID})
		if e != nil {
			return StudentTicketResult{}, unavailable()
		}
		return StudentTicketResult{Ticket: ViewOf(cur), MessageID: &id}, nil // gửi lại: cùng tin
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return StudentTicketResult{}, unavailable()
	}
	nt, err := q.TicketFollowup(ctx, store.TicketFollowupParams{ID: ticketID, StudentID: a.UserID, Now: s.now(), ReopenDays: int32(s.Cfg.reopenDays())}) //nolint:gosec // ≤ 365
	if errors.Is(err, pgx.ErrNoRows) {
		cur, e := q.TicketOwnedByStudent(ctx, store.TicketOwnedByStudentParams{ID: ticketID, StudentID: a.UserID})
		if e != nil {
			return StudentTicketResult{}, unavailable()
		}
		return StudentTicketResult{}, s.stateErr(cur)
	}
	if err != nil {
		return StudentTicketResult{}, unavailable()
	}
	blocks, _ := json.Marshal([]map[string]any{{"kind": "ticket", "data": map[string]any{"ticket_id": t.ID}}})
	msg, err := q.ChatInsertStaffMessage(ctx, store.ChatInsertStaffMessageParams{CourseID: t.CourseID, SessionID: sess, UserID: a.UserID, Role: store.ChatRoleUSER, Content: content, Intent: ptrStr("ESCALATION_FOLLOWUP"), Blocks: blocks, ClientMsgID: &clientMsgID})
	if err != nil {
		return StudentTicketResult{}, unavailable()
	}
	if err := q.TouchChatSession(ctx, sess); err != nil {
		return StudentTicketResult{}, unavailable()
	}
	if err := s.audit(ctx, q, t.CourseID, a.UserID, t.ID, "escalation_ticket.followup", t.Status, store.TicketStatusOPEN); err != nil {
		return StudentTicketResult{}, err
	}
	if _, err := outbox.Write(ctx, tx, TopicFollowup, map[string]any{"ticket_id": t.ID, "course_id": t.CourseID}); err != nil {
		return StudentTicketResult{}, unavailable()
	}
	if err := tx.Commit(ctx); err != nil {
		return StudentTicketResult{}, unavailable()
	}
	return StudentTicketResult{Ticket: ViewOf(nt), MessageID: &msg.ID}, nil
}

// ticketSession là phiên chat nhận tin của ticket: phiên gốc nếu còn, không thì "Tin nhắn từ giảng viên" (tạo-hoặc-lấy).
func (s *Service) ticketSession(ctx context.Context, q *store.Queries, t store.EscalationTicket) (uuid.UUID, error) {
	if t.SessionID != nil {
		if row, err := q.GetChatSession(ctx, store.GetChatSessionParams{ID: *t.SessionID, UserID: t.StudentID}); err == nil {
			return row.ID, nil
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, unavailable()
		}
	}
	return ensureStaffSession(ctx, q, t.CourseID, t.StudentID)
}

// Confirm là `Đã rõ`: ANSWERED → CLOSED (STUDENT_CONFIRMED). Không hộp thoại: đảo ngược được trong 5 giây (ConfirmUndo).
func (s *Service) Confirm(ctx context.Context, a Actor, ticketID uuid.UUID) (StudentTicketResult, error) {
	t, err := s.owned(ctx, a, ticketID)
	if err != nil {
		return StudentTicketResult{}, err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return StudentTicketResult{}, unavailable()
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	nt, err := q.TicketConfirm(ctx, store.TicketConfirmParams{ID: ticketID, StudentID: a.UserID, Now: s.now()})
	if errors.Is(err, pgx.ErrNoRows) {
		return StudentTicketResult{}, apierr.New(http.StatusConflict, apierr.TicketStateInvalid)
	}
	if err != nil {
		return StudentTicketResult{}, unavailable()
	}
	if err := s.audit(ctx, q, t.CourseID, a.UserID, t.ID, "escalation_ticket.confirmed", store.TicketStatusANSWERED, store.TicketStatusCLOSED); err != nil {
		return StudentTicketResult{}, err
	}
	if _, err := outbox.Write(ctx, tx, TopicClosed, map[string]any{"ticket_id": t.ID, "course_id": t.CourseID}); err != nil {
		return StudentTicketResult{}, unavailable()
	}
	if err := tx.Commit(ctx); err != nil {
		return StudentTicketResult{}, unavailable()
	}
	return StudentTicketResult{Ticket: ViewOf(nt)}, nil
}

// ConfirmUndo hoàn tác `Đã rõ` trong 5 giây và chỉ khi chưa có hành động khác (ticket vẫn CLOSED do chính sinh viên xác nhận).
func (s *Service) ConfirmUndo(ctx context.Context, a Actor, ticketID uuid.UUID) (StudentTicketResult, error) {
	t, err := s.owned(ctx, a, ticketID)
	if err != nil {
		return StudentTicketResult{}, err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return StudentTicketResult{}, unavailable()
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	nt, err := q.TicketConfirmUndo(ctx, store.TicketConfirmUndoParams{ID: ticketID, StudentID: a.UserID, Now: s.now()})
	if errors.Is(err, pgx.ErrNoRows) {
		return StudentTicketResult{}, apierr.New(http.StatusConflict, apierr.TicketStateInvalid)
	}
	if err != nil {
		return StudentTicketResult{}, unavailable()
	}
	if err := s.audit(ctx, q, t.CourseID, a.UserID, t.ID, "escalation_ticket.confirm_undone", store.TicketStatusCLOSED, store.TicketStatusANSWERED); err != nil {
		return StudentTicketResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return StudentTicketResult{}, unavailable()
	}
	return StudentTicketResult{Ticket: ViewOf(nt)}, nil
}

// AutoClose là job `ticket.autoclose` (SRS 4.6.3): đóng ticket ANSWERED quá TICKET_AUTOCLOSE_DAYS (AUTO_7D). Idempotent (cập nhật có điều kiện); không đụng OPEN / CLAIMED.
// Trả số ticket đã đóng. Không thông báo cho sinh viên.
func (s *Service) AutoClose(ctx context.Context) (int, error) {
	now := s.now()
	before := now.Add(-time.Duration(s.Cfg.autocloseDays()) * 24 * time.Hour)
	closed := 0
	for {
		rows, err := store.New(s.Pool).TicketAutoCloseCandidates(ctx, store.TicketAutoCloseCandidatesParams{Before: before, PageLimit: 200})
		if err != nil {
			return closed, fmt.Errorf("escalation: ứng viên tự đóng: %w", err)
		}
		n := 0
		for _, r := range rows {
			ok, err := s.autoCloseOne(ctx, r.ID, before, now)
			if err != nil {
				return closed, err
			}
			if ok {
				n++
			}
		}
		closed += n
		if len(rows) < 200 || n == 0 {
			return closed, nil
		}
	}
}

func (s *Service) autoCloseOne(ctx context.Context, id uuid.UUID, before, now time.Time) (bool, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("escalation: mở giao dịch: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	t, err := q.TicketAutoClose(ctx, store.TicketAutoCloseParams{ID: id, Before: before, Now: now})
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil // đã đổi trạng thái / chạy song song
	}
	if err != nil {
		return false, fmt.Errorf("escalation: tự đóng: %w", err)
	}
	b, _ := json.Marshal(map[string]any{"status": store.TicketStatusANSWERED})
	a, _ := json.Marshal(map[string]any{"status": store.TicketStatusCLOSED, "ticket_id": t.ID})
	if _, err := q.InsertAuditLog(ctx, store.InsertAuditLogParams{CourseID: &t.CourseID, Entity: "escalation_ticket", EntityID: t.ID.String(), Action: "escalation_ticket.autoclosed", Before: b, After: a}); err != nil {
		return false, fmt.Errorf("escalation: audit tự đóng: %w", err)
	}
	if _, err := outbox.Write(ctx, tx, TopicClosed, map[string]any{"ticket_id": t.ID, "course_id": t.CourseID}); err != nil {
		return false, fmt.Errorf("escalation: outbox tự đóng: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("escalation: commit tự đóng: %w", err)
	}
	return true, nil
}
