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
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/privacy"
	"github.com/edupilot/backend-go/internal/store"
)

// Topic outbox của vòng đời ticket (payload chỉ có định danh — SRS 4.11).
const (
	TopicClaimed  = "escalation.claimed"
	TopicReleased = "escalation.released"
	TopicAnswered = "escalation.answered"
	TopicClosed   = "escalation.closed"

	// StaffSessionTitle là tên cố định của phiên chat riêng nhận tin từ giảng viên khi phiên của ticket đã bị xoá (SRS 4.2.7).
	StaffSessionTitle = "Tin nhắn từ giảng viên"

	// AnswerMaxChars: ANSWER_MAX_CHARS.
	AnswerMaxChars = 4000
)

// Lý do đóng không trả lời (`dismiss_reason`).

// Person là người (sinh viên / nhân viên) ở khung của nhân viên.
type Person struct {
	ID       uuid.UUID `json:"id"`
	FullName string    `json:"full_name"`
	Role     string    `json:"role,omitempty"`
}

// CourseRef là lớp của ticket.
type CourseRef struct {
	ID        uuid.UUID `json:"id"`
	ClassCode string    `json:"class_code"`
}

// StaffRow là một dòng danh sách của nhân viên (SRS 6.2 #4): KHÔNG có nội dung đầy đủ của câu hỏi / trả lời.
type StaffRow struct {
	ID              uuid.UUID `json:"id"`
	Status          string    `json:"status"`
	Kind            string    `json:"kind"`
	Student         Person    `json:"student"`
	QuestionPreview string    `json:"question_preview"`
	AgeMinutes      int       `json:"age_minutes"`
	ClaimedBy       *Person   `json:"claimed_by"`
	SLAStage        int16     `json:"sla_stage"`
	Course          CourseRef `json:"course"`
}

// StaffPage là một trang con trỏ.
type StaffPage struct {
	Items      []StaffRow `json:"items"`
	NextCursor *string    `json:"next_cursor"`
}

// StaffTicket là ticket ở khung của nhân viên (SRS 6.2).
type StaffTicket struct {
	ID              uuid.UUID `json:"id"`
	Kind            string    `json:"kind"`
	Status          string    `json:"status"`
	Student         Person    `json:"student"`
	ClaimedBy       *Person   `json:"claimed_by"`
	AnsweredBy      *Person   `json:"answered_by"`
	AgeMinutes      int       `json:"age_minutes"`
	SLAStage        int16     `json:"sla_stage"`
	ReopenedCount   int16     `json:"reopened_count"`
	SaveAsKnowledge bool      `json:"save_as_knowledge"`
	Course          CourseRef `json:"course"`
	Version         int32     `json:"version"`
	CreatedAt       time.Time `json:"created_at"`
}

// Message là tin chat ở khung của nhân viên.
type Message struct {
	ID        uuid.UUID       `json:"id"`
	Content   string          `json:"content"`
	CreatedAt time.Time       `json:"created_at"`
	Citations json.RawMessage `json:"citations,omitempty"`
	Blocks    json.RawMessage `json:"blocks,omitempty"`
}

// AIAnswer là câu trả lời của AI kèm độ tin cậy (CÓ số — thuật ngữ dành cho giảng viên, DESIGN §13).
type AIAnswer struct {
	Message
	Confidence *string `json:"confidence"`
}

// TranscriptItem là một tin của chính ticket.
type TranscriptItem struct {
	ID        uuid.UUID `json:"id"`
	Role      string    `json:"role"`
	Author    any       `json:"author"` // {full_name, role} của nhân viên, hoặc "student"
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// Detail là phản hồi `GET …/tickets/{id}`: chỉ đúng phần đã nhờ (SRS 4.3).
type Detail struct {
	Ticket     StaffTicket      `json:"ticket"`
	Question   *Message         `json:"question"`
	AIAnswer   *AIAnswer        `json:"ai_answer"`
	Transcript []TranscriptItem `json:"transcript"`
	Knowledge  *KnowledgeState  `json:"knowledge,omitempty"`
}

func (s *Service) now() time.Time {
	if s.Clock != nil {
		return s.Clock.Now()
	}
	return time.Now().UTC()
}

func person(id uuid.UUID, name string) Person { return Person{ID: id, FullName: name} }

func ageMinutes(from, now time.Time) int {
	if now.Before(from) {
		return 0
	}
	return int(now.Sub(from) / time.Minute)
}

func staffTicket(v store.TicketViewGetRow, now time.Time) StaffTicket {
	t := StaffTicket{ID: v.ID, Kind: string(v.Kind), Status: string(v.Status), Student: person(v.StudentID, v.StudentName), AgeMinutes: ageMinutes(v.WaitingSince, now),
		SLAStage: v.SlaStage, ReopenedCount: v.ReopenedCount, SaveAsKnowledge: v.SaveAsKnowledge, Course: CourseRef{ID: v.CourseID, ClassCode: v.ClassCode}, Version: v.Version, CreatedAt: v.CreatedAt}
	if v.ClaimedBy != nil && v.ClaimedName != nil {
		p := person(*v.ClaimedBy, *v.ClaimedName)
		t.ClaimedBy = &p
	}
	if v.AnsweredBy != nil && v.AnsweredName != nil {
		p := person(*v.AnsweredBy, *v.AnsweredName)
		t.AnsweredBy = &p
	}
	return t
}

// ListStaff trả hàng chờ của các lớp `courses` (một lớp hoặc "Tất cả lớp của tôi"). status: OPEN | CLAIMED | ANSWERED | ALL (mặc định OPEN).
func (s *Service) ListStaff(ctx context.Context, courses []uuid.UUID, status string, p httpx.PageParams) (StaffPage, error) {
	switch status {
	case "":
		status = "OPEN"
	case "OPEN", "CLAIMED", "ANSWERED", "ALL":
	default:
		return StaffPage{}, apierr.Validation(apierr.FieldError{Field: "status", Code: "invalid", Message: "Trạng thái phải là OPEN, CLAIMED, ANSWERED hoặc ALL."})
	}
	out := StaffPage{Items: []StaffRow{}}
	if len(courses) == 0 {
		return out, nil
	}
	ids := make([]string, len(courses))
	for i, c := range courses {
		ids[i] = c.String()
	}
	arg := store.TicketListStaffParams{CourseIds: ids, StatusFilter: status, PageLimit: int32(p.Fetch())} //nolint:gosec // ≤ 101
	if p.Cursor != nil {
		arg.CursorAt, arg.CursorID = &p.Cursor.CreatedAt, &p.Cursor.ID
	}
	rows, err := store.New(s.Pool).TicketListStaff(ctx, arg)
	if err != nil {
		return StaffPage{}, unavailable()
	}
	if len(rows) > p.Limit {
		rows = rows[:p.Limit]
		c := httpx.EncodeCursor(rows[len(rows)-1].CreatedAt, rows[len(rows)-1].ID.String())
		out.NextCursor = &c
	}
	now := s.now()
	for _, r := range rows {
		row := StaffRow{ID: r.ID, Status: string(r.Status), Kind: string(r.Kind), Student: person(r.StudentID, r.StudentName), QuestionPreview: r.QuestionPreview,
			AgeMinutes: ageMinutes(r.WaitingSince, now), SLAStage: r.SlaStage, Course: CourseRef{ID: r.CourseID, ClassCode: r.ClassCode}}
		if r.ClaimedBy != nil && r.ClaimedName != nil {
			pp := person(*r.ClaimedBy, *r.ClaimedName)
			row.ClaimedBy = &pp
		}
		out.Items = append(out.Items, row)
	}
	return out, nil
}

// MyCourseIDs là các lớp mà người này đang ACTIVE là TEACHER / TA.
func (s *Service) MyCourseIDs(ctx context.Context, user uuid.UUID) ([]uuid.UUID, error) {
	raw, err := store.New(s.Pool).TicketStaffCourseIDs(ctx, user)
	if err != nil {
		return nil, unavailable()
	}
	out := make([]uuid.UUID, 0, len(raw))
	for _, r := range raw {
		if id, err := uuid.Parse(r); err == nil {
			out = append(out, id)
		}
	}
	return out, nil
}

type ticketBlock struct {
	Kind string `json:"kind"`
	Data struct {
		TicketID   string `json:"ticket_id"`
		AnsweredBy *struct {
			FullName string `json:"full_name"`
			Role     string `json:"role"`
		} `json:"answered_by"`
	} `json:"data"`
}

func authorOf(m store.ChatMessage) any {
	if m.Role == store.ChatRoleUSER {
		return "student"
	}
	var bs []ticketBlock
	if json.Unmarshal(m.Blocks, &bs) == nil {
		for _, b := range bs {
			if b.Kind == "ticket" && b.Data.AnsweredBy != nil {
				return map[string]string{"full_name": b.Data.AnsweredBy.FullName, "role": b.Data.AnsweredBy.Role}
			}
		}
	}
	return map[string]string{"full_name": "", "role": ""}
}

// GetDetail là `GET …/tickets/{id}`: ticket + cặp hỏi–đáp đã nhờ + tin của chính ticket. Ticket của lớp khác → 404.
func (s *Service) GetDetail(ctx context.Context, courseID, ticketID uuid.UUID) (Detail, error) {
	ex, err := s.ReadExchange(ctx, courseID, ticketID)
	if err != nil {
		return Detail{}, err
	}
	v, err := store.New(s.Pool).TicketViewGet(ctx, store.TicketViewGetParams{CourseID: courseID, ID: ticketID})
	if err != nil {
		return Detail{}, unavailable()
	}
	d := Detail{Ticket: staffTicket(v, s.now()), Transcript: []TranscriptItem{}}
	if ex.Question != nil {
		d.Question = &Message{ID: ex.Question.ID, Content: ex.Question.Content, CreatedAt: ex.Question.CreatedAt}
	}
	if ex.AIAnswer != nil {
		a := &AIAnswer{Message: Message{ID: ex.AIAnswer.ID, Content: ex.AIAnswer.Content, CreatedAt: ex.AIAnswer.CreatedAt, Citations: ex.AIAnswer.Citations, Blocks: ex.AIAnswer.Blocks}}
		if ex.AIAnswer.Confidence.Valid {
			c := ex.AIAnswer.Confidence.Decimal.StringFixed(3)
			a.Confidence = &c
		}
		d.AIAnswer = a
	}
	if v.SaveAsKnowledge {
		if k, err := s.KnowledgeOf(ctx, courseID, ticketID); err == nil && k.Saved {
			d.Knowledge = &k
		}
	}
	for _, m := range ex.Transcript {
		d.Transcript = append(d.Transcript, TranscriptItem{ID: m.ID, Role: string(m.Role), Author: authorOf(m), Content: m.Content, CreatedAt: m.CreatedAt})
	}
	return d, nil
}

// conflict giải thích vì sao một cập nhật có điều kiện không ghi được hàng nào (đọc lại ticket).
func (s *Service) conflict(ctx context.Context, q *store.Queries, courseID, ticketID, user uuid.UUID, version *int32) error {
	v, err := q.TicketViewGet(ctx, store.TicketViewGetParams{CourseID: courseID, ID: ticketID})
	if errors.Is(err, pgx.ErrNoRows) {
		return notFound()
	}
	if err != nil {
		return unavailable()
	}
	if version != nil && *version != v.Version {
		return apierr.New(http.StatusConflict, apierr.VersionConflict).WithDetails(map[string]any{"current_version": v.Version})
	}
	if v.Status == store.TicketStatusCLAIMED && v.ClaimedBy != nil && *v.ClaimedBy != user {
		d := map[string]any{"claimed_at": v.ClaimedAt}
		if b, err := q.TicketUserBrief(ctx, store.TicketUserBriefParams{CourseID: courseID, UserID: *v.ClaimedBy}); err == nil {
			d["claimed_by"] = Person{ID: b.ID, FullName: b.FullName, Role: b.Role}
		}
		return apierr.New(http.StatusConflict, apierr.TicketAlreadyClaimed).WithDetails(d)
	}
	if v.Status == store.TicketStatusANSWERED && v.AnsweredBy != nil && *v.AnsweredBy != user {
		return apierr.New(http.StatusConflict, apierr.TicketAlreadyClaimed).WithDetails(map[string]any{"claimed_by": Person{ID: *v.AnsweredBy, FullName: deref(v.AnsweredName)}})
	}
	return apierr.New(http.StatusConflict, apierr.TicketStateInvalid)
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func (s *Service) audit(ctx context.Context, q *store.Queries, courseID, actor uuid.UUID, ticketID uuid.UUID, action string, before, after store.TicketStatus) error {
	b, _ := json.Marshal(map[string]any{"status": before})
	a, _ := json.Marshal(map[string]any{"status": after, "ticket_id": ticketID})
	if _, err := q.InsertAuditLog(ctx, store.InsertAuditLogParams{CourseID: &courseID, ActorID: &actor, Entity: "escalation_ticket", EntityID: ticketID.String(), Action: action, Before: b, After: a}); err != nil {
		return fmt.Errorf("escalation: audit: %w", err)
	}
	return nil
}

func (s *Service) view(ctx context.Context, q *store.Queries, courseID, ticketID uuid.UUID) (StaffTicket, error) {
	v, err := q.TicketViewGet(ctx, store.TicketViewGetParams{CourseID: courseID, ID: ticketID})
	if err != nil {
		return StaffTicket{}, unavailable()
	}
	return staffTicket(v, s.now()), nil
}

// Claim là `Nhận`: OPEN → CLAIMED; người đã nhận bấm lại → 200 không đổi; người khác đã nhận → 409 TICKET_ALREADY_CLAIMED kèm người nhận.
func (s *Service) Claim(ctx context.Context, a Actor, courseID, ticketID uuid.UUID, version *int32) (StaffTicket, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return StaffTicket{}, unavailable()
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	t, err := q.TicketClaim(ctx, store.TicketClaimParams{CourseID: courseID, ID: ticketID, UserID: &a.UserID, Version: version})
	if errors.Is(err, pgx.ErrNoRows) {
		if cur, e := q.TicketGet(ctx, store.TicketGetParams{CourseID: courseID, ID: ticketID}); e == nil && cur.Status == store.TicketStatusCLAIMED && cur.ClaimedBy != nil && *cur.ClaimedBy == a.UserID {
			return s.view(ctx, q, courseID, ticketID) // idempotent
		}
		return StaffTicket{}, s.conflict(ctx, q, courseID, ticketID, a.UserID, version)
	}
	if err != nil {
		return StaffTicket{}, unavailable()
	}
	if err := s.audit(ctx, q, courseID, a.UserID, t.ID, "escalation_ticket.claimed", store.TicketStatusOPEN, store.TicketStatusCLAIMED); err != nil {
		return StaffTicket{}, err
	}
	if _, err := outbox.Write(ctx, tx, TopicClaimed, map[string]any{"ticket_id": t.ID, "course_id": courseID}); err != nil {
		return StaffTicket{}, unavailable()
	}
	out, err := s.view(ctx, q, courseID, ticketID)
	if err != nil {
		return StaffTicket{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return StaffTicket{}, unavailable()
	}
	return out, nil
}

// Release là `Bỏ nhận` (chỉ người đã nhận): CLAIMED → OPEN, `waiting_since` giữ nguyên.
func (s *Service) Release(ctx context.Context, a Actor, courseID, ticketID uuid.UUID, version *int32) (StaffTicket, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return StaffTicket{}, unavailable()
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	t, err := q.TicketRelease(ctx, store.TicketReleaseParams{CourseID: courseID, ID: ticketID, UserID: &a.UserID, Version: version})
	if errors.Is(err, pgx.ErrNoRows) {
		return StaffTicket{}, s.conflict(ctx, q, courseID, ticketID, a.UserID, version)
	}
	if err != nil {
		return StaffTicket{}, unavailable()
	}
	if err := s.audit(ctx, q, courseID, a.UserID, t.ID, "escalation_ticket.released", store.TicketStatusCLAIMED, store.TicketStatusOPEN); err != nil {
		return StaffTicket{}, err
	}
	if _, err := outbox.Write(ctx, tx, TopicReleased, map[string]any{"ticket_id": t.ID, "course_id": courseID}); err != nil {
		return StaffTicket{}, unavailable()
	}
	out, err := s.view(ctx, q, courseID, ticketID)
	if err != nil {
		return StaffTicket{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return StaffTicket{}, unavailable()
	}
	return out, nil
}

// Dismiss là `Đóng không trả lời`: → CLOSED (STAFF_DISMISSED) kèm tin ngắn theo lý do vào chat của sinh viên.
func (s *Service) Dismiss(ctx context.Context, a Actor, courseID, ticketID uuid.UUID, reason string, version *int32) (StaffTicket, error) {
	if dismissText(reason) == "" {
		return StaffTicket{}, apierr.Validation(apierr.FieldError{Field: "reason", Code: "invalid", Message: "Lý do phải là OUT_OF_SCOPE, DUPLICATE hoặc NEED_MORE_INFO."})
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return StaffTicket{}, unavailable()
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	before, err := q.TicketGet(ctx, store.TicketGetParams{CourseID: courseID, ID: ticketID})
	if errors.Is(err, pgx.ErrNoRows) {
		return StaffTicket{}, notFound()
	}
	if err != nil {
		return StaffTicket{}, unavailable()
	}
	t, err := q.TicketDismiss(ctx, store.TicketDismissParams{CourseID: courseID, ID: ticketID, UserID: &a.UserID, Reason: &reason, Version: version})
	if errors.Is(err, pgx.ErrNoRows) {
		return StaffTicket{}, s.conflict(ctx, q, courseID, ticketID, a.UserID, version)
	}
	if err != nil {
		return StaffTicket{}, unavailable()
	}
	brief, err := q.TicketUserBrief(ctx, store.TicketUserBriefParams{CourseID: courseID, UserID: a.UserID})
	if err != nil {
		return StaffTicket{}, unavailable()
	}
	if _, err := s.postStaffMessage(ctx, q, t, brief, dismissText(reason)); err != nil {
		return StaffTicket{}, err
	}
	if err := s.audit(ctx, q, courseID, a.UserID, t.ID, "escalation_ticket.dismissed", before.Status, store.TicketStatusCLOSED); err != nil {
		return StaffTicket{}, err
	}
	if _, err := outbox.Write(ctx, tx, TopicClosed, map[string]any{"ticket_id": t.ID, "course_id": courseID}); err != nil {
		return StaffTicket{}, unavailable()
	}
	out, err := s.view(ctx, q, courseID, ticketID)
	if err != nil {
		return StaffTicket{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return StaffTicket{}, unavailable()
	}
	return out, nil
}

// dismissText là tin ngắn gửi sinh viên theo lý do đóng; "" = lý do không hợp lệ.
func dismissText(reason string) string {
	switch reason {
	case "OUT_OF_SCOPE":
		return "Câu hỏi này nằm ngoài phạm vi học phần nên giảng viên không trả lời."
	case "DUPLICATE":
		return "Câu hỏi này đã được trả lời ở nơi khác của lớp."
	case "NEED_MORE_INFO":
		return "Giảng viên cần thêm thông tin để trả lời. Bạn hãy đặt lại câu hỏi rõ hơn."
	}
	return ""
}

// ensureStaffSession lấy hoặc tạo phiên "Tin nhắn từ giảng viên" của sinh viên (SRS 4.2.7). Chạy trong giao dịch của người gọi.
func ensureStaffSession(ctx context.Context, q *store.Queries, courseID, student uuid.UUID) (uuid.UUID, error) {
	title := StaffSessionTitle
	row, err := q.ChatStaffSession(ctx, store.ChatStaffSessionParams{CourseID: courseID, UserID: student, Title: &title})
	if err == nil {
		return row.ID, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, unavailable()
	}
	created, err := q.InsertChatSession(ctx, store.InsertChatSessionParams{CourseID: courseID, UserID: student, Title: &title})
	if err != nil {
		return uuid.Nil, unavailable()
	}
	return created.ID, nil
}

// EnsureStaffSession là bản công khai cho các luồng khác (P5 `Nhắn sinh viên`).
func EnsureStaffSession(ctx context.Context, tx pgx.Tx, courseID, student uuid.UUID) (uuid.UUID, error) {
	return ensureStaffSession(ctx, store.New(tx), courseID, student)
}

// postStaffMessage chèn tin STAFF_ANSWER vào phiên của ticket (hoặc phiên "Tin nhắn từ giảng viên" nếu phiên đã xoá mềm) rồi đẩy phiên lên đầu.
func (s *Service) postStaffMessage(ctx context.Context, q *store.Queries, t store.EscalationTicket, by store.TicketUserBriefRow, body string) (store.ChatMessage, error) {
	var sess uuid.UUID
	if t.SessionID != nil {
		if row, err := q.GetChatSession(ctx, store.GetChatSessionParams{ID: *t.SessionID, UserID: t.StudentID}); err == nil {
			sess = row.ID
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return store.ChatMessage{}, unavailable()
		}
	}
	if sess == uuid.Nil {
		id, err := ensureStaffSession(ctx, q, t.CourseID, t.StudentID)
		if err != nil {
			return store.ChatMessage{}, err
		}
		sess = id
	}
	blocks, _ := json.Marshal([]map[string]any{{"kind": "ticket", "data": map[string]any{"ticket_id": t.ID, "answered_by": map[string]string{"full_name": by.FullName, "role": by.Role}}}})
	msg, err := q.ChatInsertStaffMessage(ctx, store.ChatInsertStaffMessageParams{CourseID: t.CourseID, SessionID: sess, UserID: t.StudentID, Role: store.ChatRoleASSISTANT, Content: body, Intent: ptrStr("STAFF_ANSWER"), Blocks: blocks})
	if err != nil {
		return store.ChatMessage{}, unavailable()
	}
	if err := q.TouchChatSession(ctx, sess); err != nil {
		return store.ChatMessage{}, unavailable()
	}
	return msg, nil
}

func ptrStr(s string) *string { return &s }

// PIIError mô tả một vùng không được gửi (vị trí rune, không lặp giá trị).
type PIIError struct {
	Start int    `json:"start"`
	End   int    `json:"end"`
	Kind  string `json:"kind"`
}

func foldKey(name string) string { return strings.Join(strings.Fields(auth.Fold(name)), " ") }

// checkAnswerPII chặn định danh của sinh viên KHÁC trong câu trả lời (SRS 4.2.6, Q21: chặn cứng, không ghi đè).
func (s *Service) checkAnswerPII(ctx context.Context, courseID, student uuid.UUID, body string) error {
	if s.PII == nil {
		return nil
	}
	ents, err := s.PII.Entities(ctx, courseID, body)
	if err != nil {
		return unavailable()
	}
	if len(ents) == 0 {
		return nil
	}
	q := store.New(s.Pool)
	id, err := q.TicketStudentIdentity(ctx, store.TicketStudentIdentityParams{CourseID: courseID, StudentID: student})
	if err != nil {
		return unavailable()
	}
	own := map[string]bool{foldKey(id.FullName): true, strings.ToLower(id.Email): true}
	for _, c := range []string{id.StudentCode, id.RosterCode} {
		if c != "" {
			own[strings.ToLower(strings.TrimSpace(c))] = true
		}
	}
	var emails []string
	for _, e := range ents {
		if e.Kind == privacy.KindEmail && !own[e.Key] {
			emails = append(emails, e.Key)
		}
	}
	otherEmail := map[string]bool{}
	if len(emails) > 0 {
		rows, err := q.TicketOtherStudentEmails(ctx, store.TicketOtherStudentEmailsParams{CourseID: courseID, StudentID: student, Emails: emails})
		if err != nil {
			return unavailable()
		}
		for _, r := range rows {
			otherEmail[r] = true
		}
	}
	var bad []PIIError
	for _, e := range ents {
		switch {
		case e.Kind == privacy.KindCCCD:
		case (e.Kind == privacy.KindName || e.Kind == privacy.KindMSSV) && e.InRoster && !own[e.Key]:
		case e.Kind == privacy.KindEmail && otherEmail[e.Key]:
		default:
			continue
		}
		bad = append(bad, PIIError{Start: e.Start, End: e.End, Kind: string(e.Kind)})
	}
	if len(bad) > 0 {
		return apierr.New(http.StatusUnprocessableEntity, apierr.PIIInAnswer).WithDetails(bad)
	}
	return nil
}

// Answer là `Gửi trả lời` (SRS 4.2.5): MỘT giao dịch — ticket → ANSWERED, tin STAFF_ANSWER vào chat riêng của sinh viên, outbox `escalation.answered`, audit.
// Idempotency-Key do middleware của route lo. body rỗng / > 4000 ký tự → 422; thông tin của sinh viên khác → 422 PII_IN_ANSWER.
func (s *Service) Answer(ctx context.Context, a Actor, courseID, ticketID uuid.UUID, body string, version *int32) (StaffTicket, Message, error) {
	body = strings.TrimSpace(body)
	if n := utf8.RuneCountInString(body); n == 0 || n > AnswerMaxChars {
		return StaffTicket{}, Message{}, apierr.Validation(apierr.FieldError{Field: "body", Code: "length", Message: fmt.Sprintf("Trả lời dài từ 1 đến %d ký tự.", AnswerMaxChars)})
	}
	q0 := store.New(s.Pool)
	cur, err := q0.TicketGet(ctx, store.TicketGetParams{CourseID: courseID, ID: ticketID})
	if errors.Is(err, pgx.ErrNoRows) {
		return StaffTicket{}, Message{}, notFound()
	}
	if err != nil {
		return StaffTicket{}, Message{}, unavailable()
	}
	if err := s.checkAnswerPII(ctx, courseID, cur.StudentID, body); err != nil {
		return StaffTicket{}, Message{}, err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return StaffTicket{}, Message{}, unavailable()
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	t, err := q.TicketAnswer(ctx, store.TicketAnswerParams{CourseID: courseID, ID: ticketID, UserID: &a.UserID, Body: &body, Version: version})
	if errors.Is(err, pgx.ErrNoRows) {
		return StaffTicket{}, Message{}, s.conflict(ctx, q, courseID, ticketID, a.UserID, version)
	}
	if err != nil {
		return StaffTicket{}, Message{}, unavailable()
	}
	brief, err := q.TicketUserBrief(ctx, store.TicketUserBriefParams{CourseID: courseID, UserID: a.UserID})
	if err != nil {
		return StaffTicket{}, Message{}, unavailable()
	}
	msg, err := s.postStaffMessage(ctx, q, t, brief, body)
	if err != nil {
		return StaffTicket{}, Message{}, err
	}
	if err := enqueueAnswerMail(ctx, tx, MailAnswered, t, msg.ID); err != nil {
		return StaffTicket{}, Message{}, err
	}
	if err := s.audit(ctx, q, courseID, a.UserID, t.ID, "escalation_ticket.answered", cur.Status, store.TicketStatusANSWERED); err != nil {
		return StaffTicket{}, Message{}, err
	}
	if _, err := outbox.Write(ctx, tx, TopicAnswered, map[string]any{"ticket_id": t.ID, "course_id": courseID, "answer_message_id": msg.ID, "user_id": t.StudentID}); err != nil {
		return StaffTicket{}, Message{}, unavailable()
	}
	out, err := s.view(ctx, q, courseID, ticketID)
	if err != nil {
		return StaffTicket{}, Message{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return StaffTicket{}, Message{}, unavailable()
	}
	return out, Message{ID: msg.ID, Content: msg.Content, CreatedAt: msg.CreatedAt}, nil
}
