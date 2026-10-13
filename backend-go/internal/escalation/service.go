// Package escalation là ticket "Nhờ giảng viên" (SRS FEAT-escalation-notify 4.2, 4.3): tạo idempotent theo message_id,
// vòng đời OPEN → CLAIMED → ANSWERED → CLOSED, đọc đúng cặp hỏi–đáp đã nhờ. Danh tính chỉ lấy từ JWT (Actor).
package escalation

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/exam"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/privacy"
	"github.com/edupilot/backend-go/internal/store"
)

// Topic outbox của vòng đời ticket (payload chỉ có định danh, SRS 4.11).
const (
	TopicCreated = "escalation.created"

	DefaultOpenCap = 5 // TICKET_OPEN_CAP
)

// Locker là exam.Locker: sinh viên đang làm bài không được nhờ giảng viên (D56).
type Locker interface {
	IsLocked(ctx context.Context, userID uuid.UUID) (exam.Lock, bool, error)
}

// Config là hằng của SRS 4.1 (0 = mặc định).
type Config struct {
	OpenCap int // TICKET_OPEN_CAP
}

// Actor là người gọi: chỉ từ Principal.
type Actor struct {
	UserID uuid.UUID
	Role   auth.Role
}

// Service là nghiệp vụ ticket.
type Service struct {
	Pool    *pgxpool.Pool
	Members auth.CourseResolver
	Lock    Locker
	Log     *slog.Logger
	Cfg     Config
	// PII chặn định danh sinh viên khác trong câu trả lời của nhân viên; nil = không kiểm (test dựng chỉ bằng Pool).
	PII *privacy.Detector
	// Clock tuỳ chọn (nil = đồng hồ hệ thống).
	Clock clock.Clock

	// Suppressed đếm ticket tự mở bị trần chặn (`chat_tickets_suppressed_total`); log không nội dung.
	Suppressed atomic.Int64
}

func (s *Service) cap() int {
	if s.Cfg.OpenCap <= 0 {
		return DefaultOpenCap
	}
	return s.Cfg.OpenCap
}

// StudentView là phần duy nhất của ticket mà sinh viên thấy (không confidence, người nhận, origin, sla_stage — SRS 2).
type StudentView struct {
	ID         uuid.UUID  `json:"id"`
	Status     string     `json:"status"`
	CreatedAt  time.Time  `json:"created_at"`
	AnsweredAt *time.Time `json:"answered_at,omitempty"`
}

// ViewOf rút gọn một hàng ticket cho sinh viên.
func ViewOf(t store.EscalationTicket) StudentView {
	return StudentView{ID: t.ID, Status: string(t.Status), CreatedAt: t.CreatedAt, AnsweredAt: t.AnsweredAt}
}

// ViewsForMessages trả ticket CỦA sinh viên theo message_id (một truy vấn cho cả trang tin).
func ViewsForMessages(ctx context.Context, db store.DBTX, student uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]StudentView, error) {
	out := map[uuid.UUID]StudentView{}
	if len(ids) == 0 {
		return out, nil
	}
	strs := make([]string, len(ids))
	for i, id := range ids {
		strs[i] = id.String()
	}
	rows, err := store.New(db).TicketsOfStudentForMessages(ctx, store.TicketsOfStudentForMessagesParams{StudentID: student, MessageIds: strs})
	if err != nil {
		return nil, fmt.Errorf("escalation: ticket theo tin: %w", err)
	}
	for _, r := range rows {
		if r.MessageID != nil {
			out[*r.MessageID] = StudentView{ID: r.ID, Status: string(r.Status), CreatedAt: r.CreatedAt, AnsweredAt: r.AnsweredAt}
		}
	}
	return out, nil
}

// CreateInput là một yêu cầu mở ticket từ tin trợ lý. Auto = tự mở (chặn trần thì bỏ qua, không lỗi).
type CreateInput struct {
	CourseID, StudentID, MessageID, SessionID uuid.UUID
	Origin                                    store.TicketOrigin
	Confidence                                decimal.NullDecimal
	Auto                                      bool
}

// Created là kết quả CreateTx.
type Created struct {
	Ticket     store.EscalationTicket
	Already    bool // tin đã có ticket (kể cả CLOSED)
	Suppressed bool // tự mở bị trần chặn: không có ticket
}

// CreateTx mở ticket trong GIAO DỊCH của người gọi (SRS 4.2.2, thứ tự TLR-18): tra message_id → khoá trần → đếm trần → INSERT … ON CONFLICT DO NOTHING.
// Thành công ghi outbox `escalation.created` và audit_log (không nội dung) cùng giao dịch.
func (s *Service) CreateTx(ctx context.Context, tx pgx.Tx, in CreateInput) (Created, error) {
	q := store.New(tx)
	if t, err := q.TicketByMessage(ctx, &in.MessageID); err == nil {
		return Created{Ticket: t, Already: true}, nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return Created{}, fmt.Errorf("escalation: tra message_id: %w", err)
	}
	if err := q.TicketLockStudentCap(ctx, store.TicketLockStudentCapParams{StudentID: in.StudentID.String(), CourseID: in.CourseID.String()}); err != nil {
		return Created{}, fmt.Errorf("escalation: khoá trần: %w", err)
	}
	n, err := q.TicketCountOpen(ctx, store.TicketCountOpenParams{CourseID: in.CourseID, StudentID: in.StudentID})
	if err != nil {
		return Created{}, fmt.Errorf("escalation: đếm trần: %w", err)
	}
	if int(n) >= s.cap() {
		if in.Auto {
			s.Suppressed.Add(1)
			if s.Log != nil {
				s.Log.WarnContext(ctx, "chat_tickets_suppressed_total", "course_id", in.CourseID)
			}
			return Created{Suppressed: true}, nil
		}
		return Created{}, apierr.New(http.StatusConflict, apierr.TicketLimitReached)
	}
	t, err := q.TicketInsert(ctx, store.TicketInsertParams{
		CourseID: in.CourseID, Origin: in.Origin, MessageID: in.MessageID, SessionID: &in.SessionID, StudentID: in.StudentID, Confidence: in.Confidence,
	})
	if errors.Is(err, pgx.ErrNoRows) { // đua với giao dịch khác cùng tin
		t, err = q.TicketByMessage(ctx, &in.MessageID)
		if err != nil {
			return Created{}, fmt.Errorf("escalation: đọc lại sau đua: %w", err)
		}
		return Created{Ticket: t, Already: true}, nil
	}
	if err != nil {
		return Created{}, fmt.Errorf("escalation: ghi ticket: %w", err)
	}
	if _, err := outbox.Write(ctx, tx, TopicCreated, map[string]any{"ticket_id": t.ID, "course_id": t.CourseID, "kind": t.Kind}); err != nil {
		return Created{}, fmt.Errorf("escalation: outbox: %w", err)
	}
	var actor *uuid.UUID
	if !in.Auto {
		actor = &in.StudentID
	}
	if _, err := q.InsertAuditLog(ctx, store.InsertAuditLogParams{CourseID: &t.CourseID, ActorID: actor, Entity: "escalation_ticket", EntityID: t.ID.String(), Action: "escalation_ticket.created"}); err != nil {
		return Created{}, fmt.Errorf("escalation: audit: %w", err)
	}
	return Created{Ticket: t}, nil
}

func notFound() *apierr.Error  { return apierr.New(http.StatusNotFound, apierr.NotFound) }
func forbidden() *apierr.Error { return apierr.New(http.StatusForbidden, apierr.Forbidden) }

func unavailable() *apierr.Error {
	return apierr.New(http.StatusServiceUnavailable, apierr.ServiceUnavailable).WithRetryAfter(5)
}

// Escalatable: intent của tin có thể nhờ giảng viên (SRS 4.2.1). Dùng cả cho cờ `can_escalate` ở lịch sử chat để nút hiện đúng chỗ.
func Escalatable(intent *string) bool {
	if intent == nil {
		return true
	}
	switch *intent {
	case "CRISIS", "OTHER_PERSON", "SMALLTALK", "STAFF_ANSWER", "ESCALATION_FOLLOWUP":
		return false
	}
	return true
}

// Escalate là `POST /chat/messages/{mid}/escalate` (SRS 6.1 #1): chỉ sinh viên chủ phiên.
func (s *Service) Escalate(ctx context.Context, a Actor, messageID uuid.UUID) (Created, error) {
	if a.Role != auth.RoleStudent {
		return Created{}, forbidden()
	}
	q := store.New(s.Pool)
	m, err := q.GetChatMessageOwned(ctx, store.GetChatMessageOwnedParams{ID: messageID, UserID: a.UserID})
	if errors.Is(err, pgx.ErrNoRows) {
		return Created{}, notFound()
	}
	if err != nil {
		return Created{}, unavailable()
	}
	ms, err := s.Members.Resolve(ctx, auth.Principal{Sub: a.UserID.String(), Role: a.Role}, m.CourseID)
	if err != nil {
		return Created{}, unavailable()
	}
	if !ms.Found || ms.Status != "ACTIVE" || ms.Role != auth.RoleStudent {
		return Created{}, forbidden()
	}
	st, err := q.ChatCourseStatus(ctx, m.CourseID)
	if err != nil {
		return Created{}, unavailable()
	}
	if st == store.CourseStatusARCHIVED {
		return Created{}, apierr.New(http.StatusConflict, apierr.CourseArchived)
	}
	if s.Lock != nil {
		lk, locked, err := s.Lock.IsLocked(ctx, a.UserID)
		if err != nil {
			return Created{}, unavailable()
		}
		if locked {
			return Created{}, apierr.New(http.StatusConflict, apierr.ExamInProgress).WithDetails(map[string]any{"until": lk.Until.UTC().Format(time.RFC3339)})
		}
	}
	if _, err := q.GetChatSession(ctx, store.GetChatSessionParams{ID: m.SessionID, UserID: a.UserID}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) { // phiên đã xoá mềm
			return Created{}, apierr.New(http.StatusUnprocessableEntity, apierr.MessageNotEscalatable)
		}
		return Created{}, unavailable()
	}
	if m.Role != store.ChatRoleASSISTANT || m.StreamStatus != store.ChatStreamStatusDONE || !Escalatable(m.Intent) {
		return Created{}, apierr.New(http.StatusUnprocessableEntity, apierr.MessageNotEscalatable)
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return Created{}, unavailable()
	}
	defer func() { _ = tx.Rollback(ctx) }()
	res, err := s.CreateTx(ctx, tx, CreateInput{CourseID: m.CourseID, StudentID: a.UserID, MessageID: m.ID, SessionID: m.SessionID, Origin: store.TicketOriginSTUDENTREQUEST, Confidence: m.Confidence})
	if err != nil {
		var ae *apierr.Error
		if errors.As(err, &ae) {
			return Created{}, err
		}
		return Created{}, unavailable()
	}
	if err := tx.Commit(ctx); err != nil {
		return Created{}, unavailable()
	}
	return res, nil
}

// Exchange là phần chat MÀ NHÂN VIÊN ĐƯỢC ĐỌC của một ticket (SRS 4.3): câu hỏi, câu trả lời của AI và các tin của chính ticket — không gì khác của phiên.
type Exchange struct {
	Ticket     store.EscalationTicket
	Question   *store.ChatMessage
	AIAnswer   *store.ChatMessage
	Transcript []store.ChatMessage
}

// ReadExchange nạp ticket theo (lớp, id) và đúng hai truy vấn được phép chạm chat_messages.
func (s *Service) ReadExchange(ctx context.Context, courseID, ticketID uuid.UUID) (Exchange, error) {
	q := store.New(s.Pool)
	t, err := q.TicketGet(ctx, store.TicketGetParams{CourseID: courseID, ID: ticketID})
	if errors.Is(err, pgx.ErrNoRows) {
		return Exchange{}, notFound()
	}
	if err != nil {
		return Exchange{}, unavailable()
	}
	ex := Exchange{Ticket: t, Transcript: []store.ChatMessage{}}
	if t.MessageID != nil {
		rows, err := q.GetEscalatedExchange(ctx, store.GetEscalatedExchangeParams{CourseID: courseID, MessageID: *t.MessageID})
		if err != nil {
			return Exchange{}, unavailable()
		}
		for i := range rows {
			if rows[i].Role == store.ChatRoleUSER {
				ex.Question = &rows[i]
			} else {
				ex.AIAnswer = &rows[i]
			}
		}
	}
	if t.SessionID != nil {
		ex.Transcript, err = q.ListTicketTranscript(ctx, store.ListTicketTranscriptParams{CourseID: courseID, SessionID: *t.SessionID, TicketID: t.ID.String()})
		if err != nil {
			return Exchange{}, unavailable()
		}
	}
	return ex, nil
}
