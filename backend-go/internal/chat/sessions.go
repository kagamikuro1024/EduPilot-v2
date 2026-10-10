package chat

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

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
	"github.com/edupilot/backend-go/internal/store"
)

// SessionOut là một phiên trong danh sách.
type SessionOut struct {
	ID            uuid.UUID  `json:"id"`
	Title         *string    `json:"title"`
	LastMessageAt time.Time  `json:"last_message_at"`
	DocumentID    *uuid.UUID `json:"document_id"`
}

func sessionOut(r store.ChatSession) SessionOut {
	return SessionOut{ID: r.ID, Title: r.Title, LastMessageAt: r.LastMessageAt, DocumentID: r.DocumentID}
}

// SessionPage là một trang phiên.
type SessionPage struct {
	Items      []SessionOut `json:"items"`
	NextCursor *string      `json:"next_cursor"`
}

// ListSessions: phiên của CHÍNH MÌNH trong lớp, mới nhất trước (con trỏ theo last_message_at).
func (s *Service) ListSessions(ctx context.Context, a Actor, courseID uuid.UUID, p httpx.PageParams) (SessionPage, error) {
	if err := s.requireStudent(ctx, a, courseID); err != nil {
		return SessionPage{}, err
	}
	arg := store.ListChatSessionsParams{CourseID: courseID, UserID: a.UserID, PageLimit: int32(p.Fetch())} //nolint:gosec // ≤ 101
	if p.Cursor != nil {
		arg.CursorAt, arg.CursorID = &p.Cursor.CreatedAt, &p.Cursor.ID
	}
	rows, err := store.New(s.Pool).ListChatSessions(ctx, arg)
	if err != nil {
		return SessionPage{}, fmt.Errorf("chat: danh sách phiên: %w", err)
	}
	out := SessionPage{Items: make([]SessionOut, 0, len(rows))}
	if len(rows) > p.Limit {
		rows = rows[:p.Limit]
		c := httpx.EncodeCursor(rows[len(rows)-1].LastMessageAt, rows[len(rows)-1].ID.String())
		out.NextCursor = &c
	}
	for _, r := range rows {
		out.Items = append(out.Items, sessionOut(r))
	}
	return out, nil
}

// CreateSessionIn là thân POST /chat/sessions.
type CreateSessionIn struct {
	CourseID   uuid.UUID  `json:"course_id" validate:"required"`
	Title      *string    `json:"title" validate:"omitempty,max=120"`
	DocumentID *uuid.UUID `json:"document_id"`
}

// CreateSession: phiên mới; có document_id thì tài liệu phải READY, sinh viên thấy được, thuộc lớp (nếu không → 404 như không tồn tại).
func (s *Service) CreateSession(ctx context.Context, a Actor, in CreateSessionIn) (SessionOut, error) {
	if err := s.requireStudent(ctx, a, in.CourseID); err != nil {
		return SessionOut{}, err
	}
	if err := s.archived(ctx, in.CourseID); err != nil {
		return SessionOut{}, err
	}
	q := store.New(s.Pool)
	if in.DocumentID != nil {
		if _, err := q.ChatDocumentUsable(ctx, store.ChatDocumentUsableParams{CourseID: in.CourseID, ID: *in.DocumentID}); errors.Is(err, pgx.ErrNoRows) {
			return SessionOut{}, notFound()
		} else if err != nil {
			return SessionOut{}, fmt.Errorf("chat: kiểm tài liệu: %w", err)
		}
	}
	var title *string
	if in.Title != nil {
		if t := strings.TrimSpace(*in.Title); t != "" {
			title = &t
		}
	}
	row, err := q.InsertChatSession(ctx, store.InsertChatSessionParams{CourseID: in.CourseID, UserID: a.UserID, Title: title, DocumentID: in.DocumentID})
	if err != nil {
		return SessionOut{}, fmt.Errorf("chat: tạo phiên: %w", err)
	}
	return sessionOut(row), nil
}

// DeleteSession: xoá mềm (404 nếu không phải của mình / đã xoá). Restore: hoàn tác.
func (s *Service) DeleteSession(ctx context.Context, a Actor, sid uuid.UUID) error {
	if _, err := s.session(ctx, a, sid); err != nil {
		return err
	}
	if _, err := store.New(s.Pool).SoftDeleteChatSession(ctx, store.SoftDeleteChatSessionParams{ID: sid, UserID: a.UserID}); err != nil {
		return fmt.Errorf("chat: xoá phiên: %w", err)
	}
	return nil
}

// RestoreSession khôi phục phiên đã xoá mềm của chính mình.
func (s *Service) RestoreSession(ctx context.Context, a Actor, sid uuid.UUID) (SessionOut, error) {
	if a.Role == "" || a.Role != "STUDENT" {
		return SessionOut{}, forbidden()
	}
	q := store.New(s.Pool)
	n, err := q.RestoreChatSession(ctx, store.RestoreChatSessionParams{ID: sid, UserID: a.UserID})
	if err != nil {
		return SessionOut{}, fmt.Errorf("chat: khôi phục phiên: %w", err)
	}
	if n == 0 {
		return SessionOut{}, notFound()
	}
	row, err := s.session(ctx, a, sid)
	if err != nil {
		return SessionOut{}, err
	}
	return sessionOut(row), nil
}

// MessageOut là một tin trong lịch sử. Tin STREAMING / FAILED không có content: hiện phần đã có ở `content` và đặt `streaming`.
type MessageOut struct {
	ID            uuid.UUID       `json:"id"`
	Role          string          `json:"role"`
	Content       string          `json:"content"`
	Streaming     bool            `json:"streaming"`
	Status        string          `json:"stream_status"`
	Attempt       int16           `json:"attempt"`
	Citations     json.RawMessage `json:"citations"`
	Blocks        json.RawMessage `json:"blocks"`
	LowConfidence bool            `json:"low_confidence"`
	Degraded      bool            `json:"degraded"`
	MaskedCount   int32           `json:"masked_count"`
	Feedback      *string         `json:"feedback"`
	ErrorCode     *string         `json:"error_code"`
	CreatedAt     time.Time       `json:"created_at"`
}

func messageOut(m store.ChatMessage) MessageOut {
	text := m.Content
	if m.PartialContent != nil && m.StreamStatus != store.ChatStreamStatusDONE {
		text = *m.PartialContent
	}
	var fb *string
	if m.Feedback != nil {
		v := string(*m.Feedback)
		fb = &v
	}
	return MessageOut{
		ID: m.ID, Role: string(m.Role), Content: text, Streaming: m.StreamStatus == store.ChatStreamStatusSTREAMING, Status: string(m.StreamStatus),
		Attempt: m.Attempt, Citations: m.Citations, Blocks: m.Blocks, LowConfidence: m.LowConfidence, Degraded: m.Degraded, MaskedCount: m.MaskedCount,
		Feedback: fb, ErrorCode: m.ErrorCode, CreatedAt: m.CreatedAt,
	}
}

// MessagePage là một trang tin (mới nhất trước).
type MessagePage struct {
	Items      []MessageOut `json:"items"`
	NextCursor *string      `json:"next_cursor"`
}

// Messages: lịch sử của phiên của mình.
func (s *Service) Messages(ctx context.Context, a Actor, sid uuid.UUID, p httpx.PageParams) (MessagePage, error) {
	if _, err := s.session(ctx, a, sid); err != nil {
		return MessagePage{}, err
	}
	arg := store.ListChatMessagesParams{SessionID: sid, PageLimit: int32(p.Fetch())} //nolint:gosec // ≤ 101
	if p.Cursor != nil {
		arg.CursorAt, arg.CursorID = &p.Cursor.CreatedAt, &p.Cursor.ID
	}
	rows, err := store.New(s.Pool).ListChatMessages(ctx, arg)
	if err != nil {
		return MessagePage{}, fmt.Errorf("chat: lịch sử: %w", err)
	}
	out := MessagePage{Items: make([]MessageOut, 0, len(rows))}
	if len(rows) > p.Limit {
		rows = rows[:p.Limit]
		c := httpx.EncodeCursor(rows[len(rows)-1].CreatedAt, rows[len(rows)-1].ID.String())
		out.NextCursor = &c
	}
	for _, r := range rows {
		out.Items = append(out.Items, messageOut(r))
	}
	return out, nil
}

// Message nạp tin của CHÍNH MÌNH (cho route stream); không phải của mình → 404.
func (s *Service) Message(ctx context.Context, a Actor, mid uuid.UUID) (store.ChatMessage, error) {
	return s.message(ctx, a, mid)
}

// message nạp tin của CHÍNH MÌNH + kiểm quyền lớp; không phải của mình → 404.
func (s *Service) message(ctx context.Context, a Actor, mid uuid.UUID) (store.ChatMessage, error) {
	if a.Role != "STUDENT" {
		return store.ChatMessage{}, forbidden()
	}
	m, err := store.New(s.Pool).GetChatMessageOwned(ctx, store.GetChatMessageOwnedParams{ID: mid, UserID: a.UserID})
	if errors.Is(err, pgx.ErrNoRows) {
		return store.ChatMessage{}, notFound()
	}
	if err != nil {
		return store.ChatMessage{}, fmt.Errorf("chat: nạp tin: %w", err)
	}
	if err := s.requireStudent(ctx, a, m.CourseID); err != nil {
		return store.ChatMessage{}, err
	}
	return m, nil
}

// Feedback: HELPFUL / NOT_HELPFUL / nil (bỏ); bấm lại cùng giá trị = bỏ. Chỉ chủ tin, tin ASSISTANT đã DONE.
func (s *Service) Feedback(ctx context.Context, a Actor, mid uuid.UUID, value *string) error {
	m, err := s.message(ctx, a, mid)
	if err != nil {
		return err
	}
	var fb *store.ChatFeedback
	if value != nil {
		v := store.ChatFeedback(*value)
		if v != store.ChatFeedbackHELPFUL && v != store.ChatFeedbackNOTHELPFUL {
			return apierr.Validation(apierr.FieldError{Field: "value", Code: "enum", Message: "Chỉ nhận HELPFUL hoặc NOT_HELPFUL."})
		}
		if m.Feedback != nil && *m.Feedback == v {
			v = ""
		}
		if v != "" {
			fb = &v
		}
	}
	n, err := store.New(s.Pool).SetChatFeedback(ctx, store.SetChatFeedbackParams{ID: mid, UserID: a.UserID, Feedback: fb})
	if err != nil {
		return fmt.Errorf("chat: phản hồi: %w", err)
	}
	if n == 0 {
		return apierr.New(http.StatusConflict, apierr.Conflict)
	}
	return nil
}

func title60(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if utf8.RuneCountInString(s) <= 60 {
		return s
	}
	return string([]rune(s)[:60])
}

// FromDraftIn là thân POST /chat/sessions/from-draft.
type FromDraftIn struct {
	CourseID uuid.UUID `json:"course_id" validate:"required"`
	Title    string    `json:"title"`
	Body     string    `json:"body" validate:"required"`
}

// FromDraftOut trả lại bản nháp NGUYÊN VĂN (đúng từng byte): máy chủ không lưu nháp, chỉ phản hồi.
type FromDraftOut struct {
	SessionID uuid.UUID `json:"session_id"`
	Draft     struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	} `json:"draft"`
}

// FromDraft: "Chuyển sang chat riêng" từ hộp thoại của Threads — tạo phiên PRIVATE KHÔNG có tin nhắn, ghi SWITCHED (số Finding tính lại phía máy chủ).
// Trong giờ thi vẫn tạo được (chỉ việc gửi bị khoá, US-P3-05 AC9).
func (s *Service) FromDraft(ctx context.Context, a Actor, in FromDraftIn) (FromDraftOut, error) {
	if err := s.requireStudent(ctx, a, in.CourseID); err != nil {
		return FromDraftOut{}, err
	}
	if err := s.archived(ctx, in.CourseID); err != nil {
		return FromDraftOut{}, err
	}
	if strings.TrimSpace(in.Body) == "" || utf8.RuneCountInString(in.Body) > 20000 || utf8.RuneCountInString(in.Title) > 200 {
		return FromDraftOut{}, apierr.Validation(apierr.FieldError{Field: "body", Code: "invalid", Message: "Nội dung từ 1 đến 20.000 ký tự."})
	}
	row, err := store.New(s.Pool).InsertChatSession(ctx, store.InsertChatSessionParams{CourseID: in.CourseID, UserID: a.UserID})
	if err != nil {
		return FromDraftOut{}, fmt.Errorf("chat: tạo phiên từ nháp: %w", err)
	}
	if s.OnSwitched != nil {
		s.OnSwitched(ctx, a.UserID, in.CourseID, in.Title, in.Body)
	}
	var out FromDraftOut
	out.SessionID = row.ID
	out.Draft.Title, out.Draft.Body = in.Title, in.Body
	return out, nil
}
