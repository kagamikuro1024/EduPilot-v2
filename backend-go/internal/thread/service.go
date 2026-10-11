package thread

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"

	"github.com/edupilot/backend-go/internal/agent"
	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/exam"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
	"github.com/edupilot/backend-go/internal/jobs"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/store"
)

// Hằng của SRS 8.1 / proposals #11.
const (
	SimilarMin     = 0.75 // THREAD_SIMILAR_MIN
	SimilarOfMin   = 0.90 // THREAD_SIMILAR_OF_MIN
	PrecheckPerMin = 60
	// KindAnswer là loại việc của hàng ep:ingest: AI trả lời một thread.
	KindAnswer = "thread.answer"
	// TopicCreated là sự kiện outbox cho các bên nghe khác (P4, P10).
	TopicCreated     = "thread.created"
	TopicPostDecided = "thread.post_decided"
	maxAnswerTries   = 3
)

// Locker là exam.Locker: khoá đăng thread trong giờ thi (Q2) và dấu vết CHAT_BLOCKED.
type Locker interface {
	IsLocked(ctx context.Context, userID uuid.UUID) (exam.Lock, bool, error)
	RecordChatBlocked(ctx context.Context, userID, attemptID uuid.UUID) error
}

// Retriever là rag.Service.
type Retriever = agent.Retriever

// Service là nghiệp vụ Threads.
type Service struct {
	Pool  *pgxpool.Pool
	Redis *appredis.Client
	FW    *Firewall
	Lock  Locker
	Jobs  *jobs.Service // xếp việc AI trả lời cùng giao dịch đăng bài
	Run   *jobs.Runner  // đóng việc (chỉ worker)
	Rag   Retriever
	LLM   llm.Client
	// Embed nhúng một chuỗi (cache ep:emb); dùng ở việc AI khi thread chưa có vectơ.
	Embed func(ctx context.Context, text string) ([]float32, error)
	Clock clock.Clock
	Log   *slog.Logger
}

// Actor là người gọi: danh tính từ JWT, vai TRONG LỚP từ CourseAccessGuard.
type Actor struct {
	UserID  uuid.UUID
	Role    auth.Role
	TraceID string
}

// Staff: TEACHER hoặc TA của lớp.
func (a Actor) Staff() bool { return a.Role == auth.RoleTeacher || a.Role == auth.RoleTA }

func (s *Service) now() time.Time {
	if s.Clock == nil {
		return time.Now().UTC()
	}
	return s.Clock.Now()
}

// ---- hình dạng trả về -----------------------------------------------------------------------------------------------------------

// Author là tên công khai với cả lớp (Q3): không email, MSSV, user_id.
type Author struct {
	FullName string `json:"full_name"`
	Role     string `json:"role"`
	IsMe     bool   `json:"is_me"`
}

// ThreadRow là một hàng của danh sách.
type ThreadRow struct {
	ID             uuid.UUID `json:"id"`
	Title          string    `json:"title"`
	Preview        string    `json:"preview"`
	Tags           []string  `json:"tags"`
	WeekNo         *int16    `json:"week_no"`
	Author         Author    `json:"author"`
	AnswerState    *string   `json:"answer_state"` // PENDING | VERIFIED | CORRECTED | null (chưa có); Staff thêm REJECTED
	ReplyCount     int32     `json:"reply_count"`
	LastActivityAt time.Time `json:"last_activity_at"`
	CreatedAt      time.Time `json:"created_at"`
	// Staff: lý do AI bỏ qua (NO_CONTEXT / LOW_SCORE / LLM_UNAVAILABLE) hoặc PENDING / ANSWERED.
	AIState *string `json:"ai_state,omitempty"`
}

// ThreadPage là một trang danh sách.
type ThreadPage struct {
	Items      []ThreadRow `json:"items"`
	NextCursor *string     `json:"next_cursor"`
}

// Filter là bộ lọc danh sách.
type Filter struct {
	Week  *int
	Tag   string
	State string // pending | verified | none | ""
	Q     string
}

func escapeLike(q string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(q)
}

// List: danh sách của lớp theo vai (sinh viên không thấy bài AI REJECTED / ẩn; thread vẫn hiện).
func (s *Service) List(ctx context.Context, a Actor, courseID uuid.UUID, f Filter, p httpx.PageParams) (ThreadPage, error) {
	arg := store.ThreadListParams{CourseID: courseID, IsStaff: a.Staff(), PageLimit: int32(p.Fetch())} //nolint:gosec // ≤ 101
	if f.Week != nil {
		w := int32(*f.Week) //nolint:gosec // 1–20 đã kiểm
		arg.WeekNo = &w
	}
	if f.Tag != "" {
		arg.Tag = &f.Tag
	}
	if q := strings.TrimSpace(f.Q); q != "" {
		e := escapeLike(q)
		arg.Q = &e
	}
	if f.State != "" {
		arg.State = &f.State
	}
	if p.Cursor != nil {
		arg.CursorAt, arg.CursorID = &p.Cursor.CreatedAt, &p.Cursor.ID
	}
	rows, err := store.New(s.Pool).ThreadList(ctx, arg)
	if err != nil {
		return ThreadPage{}, fmt.Errorf("thread: danh sách: %w", err)
	}
	out := ThreadPage{Items: make([]ThreadRow, 0, len(rows))}
	if len(rows) > p.Limit {
		rows = rows[:p.Limit]
		c := httpx.EncodeCursor(rows[len(rows)-1].LastActivityAt, rows[len(rows)-1].ID.String())
		out.NextCursor = &c
	}
	for _, r := range rows {
		row := ThreadRow{ID: r.ID, Title: r.Title, Preview: r.Preview, Tags: r.Tags, WeekNo: r.WeekNo, Author: Author{FullName: r.AuthorName, Role: string(r.AuthorRole), IsMe: r.AuthorID == a.UserID},
			AnswerState: nilIfEmpty(r.AiVerification), ReplyCount: r.ReplyCount, LastActivityAt: r.LastActivityAt, CreatedAt: r.CreatedAt}
		if a.Staff() {
			st := string(r.AiState)
			row.AIState = &st
		}
		out.Items = append(out.Items, row)
	}
	return out, nil
}

// PostOut là một bài. Khoá `confidence`, `ai_body`, `rejected` chỉ có ở phản hồi cho Staff (projection sinh viên không có).
type PostOut struct {
	ID           uuid.UUID       `json:"id"`
	Kind         string          `json:"kind"` // AI | HUMAN
	Author       *Author         `json:"author"`
	Body         string          `json:"body"`
	Verification *string         `json:"verification_state"` // chỉ bài AI
	Citations    json.RawMessage `json:"citations"`
	Version      int32           `json:"version"`
	CreatedAt    time.Time       `json:"created_at"`
	Confidence   *string         `json:"confidence,omitempty"`
	AIBody       *string         `json:"ai_body,omitempty"`
	Rejected     *bool           `json:"rejected,omitempty"`
}

// ThreadDetail là thread + bài.
type ThreadDetail struct {
	ID             uuid.UUID  `json:"id"`
	Title          string     `json:"title"`
	Body           string     `json:"body"`
	Tags           []string   `json:"tags"`
	WeekNo         *int16     `json:"week_no"`
	Author         Author     `json:"author"`
	ReplyCount     int32      `json:"reply_count"`
	CreatedAt      time.Time  `json:"created_at"`
	LastActivityAt time.Time  `json:"last_activity_at"`
	AIState        *string    `json:"ai_state,omitempty"`
	AISkipReason   *string    `json:"ai_skip_reason,omitempty"`
	SimilarOf      *uuid.UUID `json:"similar_of"`
}

// ThreadView là phản hồi của GET thread.
type ThreadView struct {
	Thread     ThreadDetail `json:"thread"`
	Posts      []PostOut    `json:"posts"`
	NextCursor *string      `json:"next_cursor"`
}

func notFound() *apierr.Error { return apierr.New(http.StatusNotFound, apierr.NotFound) }

func postOut(p store.ThreadPostsRow, a Actor) PostOut {
	o := PostOut{ID: p.ID, Kind: string(p.Kind), Body: p.Body, Citations: p.Citations, Version: p.Version, CreatedAt: p.CreatedAt}
	if p.Kind == store.PostKindHUMAN {
		o.Author = &Author{FullName: deref(p.AuthorName), Role: roleStr(p.AuthorRole), IsMe: p.AuthorID != nil && *p.AuthorID == a.UserID}
	} else {
		v := string(p.VerificationState)
		o.Verification = &v
	}
	if a.Staff() {
		if p.Confidence.Valid {
			c := p.Confidence.Decimal.StringFixed(3)
			o.Confidence = &c
		}
		o.AIBody = p.AiBody
		rej := p.VerificationState == store.PostVerificationREJECTED
		o.Rejected = &rej
	}
	return o
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// Get: thread + bài theo projection vai. Thread của lớp khác / đã xoá → 404.
func (s *Service) Get(ctx context.Context, a Actor, courseID, id uuid.UUID, p httpx.PageParams) (ThreadView, error) {
	q := store.New(s.Pool)
	t, err := q.ThreadDetail(ctx, store.ThreadDetailParams{CourseID: courseID, ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return ThreadView{}, notFound()
	}
	if err != nil {
		return ThreadView{}, fmt.Errorf("thread: nạp thread: %w", err)
	}
	arg := store.ThreadPostsParams{CourseID: courseID, ThreadID: id, IsStaff: a.Staff(), PageLimit: int32(p.Fetch())} //nolint:gosec // ≤ 101
	if p.Cursor != nil {
		arg.CursorAt, arg.CursorID = &p.Cursor.CreatedAt, &p.Cursor.ID
	}
	rows, err := q.ThreadPosts(ctx, arg)
	if err != nil {
		return ThreadView{}, fmt.Errorf("thread: nạp bài: %w", err)
	}
	v := ThreadView{Posts: make([]PostOut, 0, len(rows))}
	if len(rows) > p.Limit {
		rows = rows[:p.Limit]
		c := httpx.EncodeCursor(rows[len(rows)-1].CreatedAt, rows[len(rows)-1].ID.String())
		v.NextCursor = &c
	}
	for _, r := range rows {
		v.Posts = append(v.Posts, postOut(r, a))
	}
	v.Thread = ThreadDetail{ID: t.ID, Title: t.Title, Body: t.Body, Tags: t.Tags, WeekNo: t.WeekNo, Author: Author{FullName: t.AuthorName, Role: string(t.AuthorRole), IsMe: t.AuthorID == a.UserID},
		ReplyCount: t.ReplyCount, CreatedAt: t.CreatedAt, LastActivityAt: t.LastActivityAt, SimilarOf: t.SimilarOf}
	if a.Staff() {
		st := string(t.AiState)
		v.Thread.AIState, v.Thread.AISkipReason = &st, t.AiSkipReason
	}
	return v, nil
}

// SimilarRow là một thread tương tự.
type SimilarRow struct {
	ID      uuid.UUID `json:"id"`
	Title   string    `json:"title"`
	Preview string    `json:"preview"`
}

// Similar: ≤ 3 thread cùng lớp có cosine ≥ SimilarMin; không gồm chính nó / thread có bài AI bị loại hay ẩn.
func (s *Service) Similar(ctx context.Context, courseID, id uuid.UUID) ([]SimilarRow, error) {
	q := store.New(s.Pool)
	if _, err := q.ThreadDetail(ctx, store.ThreadDetailParams{CourseID: courseID, ID: id}); errors.Is(err, pgx.ErrNoRows) {
		return nil, notFound()
	} else if err != nil {
		return nil, fmt.Errorf("thread: nạp thread: %w", err)
	}
	rows, err := q.ThreadSimilar(ctx, store.ThreadSimilarParams{CourseID: courseID, ID: id, MinCosine: SimilarMin, Lim: 3})
	if err != nil {
		return nil, fmt.Errorf("thread: thread tương tự: %w", err)
	}
	out := make([]SimilarRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, SimilarRow{ID: r.ID, Title: r.Title, Preview: r.Preview})
	}
	return out, nil
}

// ---- tường lửa: precheck / đăng ----------------------------------------------------------------------------------------------------

// PrecheckOut là phản hồi của precheck.
type PrecheckOut struct {
	Allowed          bool     `json:"allowed"`
	Reasons          []Reason `json:"reasons"`
	RedactedText     string   `json:"redacted_text"`
	RedactedTitle    string   `json:"redacted_title"`
	PersonalQuestion bool     `json:"personal_question"`
}

// Precheck không ghi gì (không forum_*, không pii_events); giới hạn 60 yêu cầu / phút / người.
func (s *Service) Precheck(ctx context.Context, a Actor, courseID uuid.UUID, title, body string) (PrecheckOut, error) {
	if err := s.rate(ctx, "precheck", a.UserID, PrecheckPerMin); err != nil {
		return PrecheckOut{}, err
	}
	if utf8.RuneCountInString(title) > 4000 || utf8.RuneCountInString(body) > 20000 {
		return PrecheckOut{}, apierr.Validation(apierr.FieldError{Field: "body", Code: "max", Message: "Nội dung quá dài."})
	}
	v, err := s.FW.CheckPost(ctx, courseID, title, body)
	if err != nil {
		return PrecheckOut{}, err
	}
	return PrecheckOut{Allowed: v.Allowed, Reasons: nonNil(v.Reasons), RedactedText: v.RedactedBody, RedactedTitle: v.RedactedTitle, PersonalQuestion: v.Personal}, nil
}

func nonNil(r []Reason) []Reason {
	if r == nil {
		return []Reason{}
	}
	return r
}

// rate: cửa sổ cố định theo phút; Redis lỗi → bỏ qua giới hạn.
func (s *Service) rate(ctx context.Context, name string, uid uuid.UUID, limit int) error {
	if s.Redis == nil {
		return nil
	}
	now := s.now()
	k := fmt.Sprintf("ep:rl:%s:%s:%d", name, uid, now.Unix()/60)
	n, err := s.Redis.Incr(ctx, k).Result()
	if err != nil {
		return nil
	}
	if n == 1 {
		_ = s.Redis.Expire(ctx, k, 70*time.Second).Err()
	}
	if int(n) > limit {
		return apierr.New(http.StatusTooManyRequests, apierr.RateLimited).WithRetryAfter(int(60 - now.Unix()%60))
	}
	return nil
}

// gate là phần chung của mọi đường ghi bài công khai: chạy `CheckPost`; không sạch thì (có redact và !Personal) lưu bản đã ẩn SAU KHI kiểm lại bản đó sạch,
// còn lại 422 PII_DETECTED kèm ghi BLOCKED. Trả tiêu đề / nội dung được phép lưu, hành động đã ghi, và vectơ (chỉ khi văn bản không đổi).
func (s *Service) gate(ctx context.Context, a Actor, courseID uuid.UUID, title, body string, redact bool) (string, string, []float32, error) {
	v, err := s.FW.CheckPost(ctx, courseID, title, body)
	if err != nil {
		return "", "", nil, unavailableErr()
	}
	if v.Allowed {
		return title, body, v.Vec, nil
	}
	if redact && !v.Personal {
		again, err := s.FW.CheckPost(ctx, courseID, v.RedactedTitle, v.RedactedBody)
		if err != nil {
			return "", "", nil, unavailableErr()
		}
		if again.Allowed {
			s.piiEvents(ctx, a, courseID, v.Reasons, store.PiiActionREDACTED)
			return v.RedactedTitle, v.RedactedBody, nil, nil
		}
		v = again // bản "đã ẩn" vẫn còn PII → không tin, chặn
	}
	s.piiEvents(ctx, a, courseID, v.Reasons, store.PiiActionBLOCKED)
	return "", "", nil, apierr.New(http.StatusUnprocessableEntity, apierr.PIIDetected).WithDetails(map[string]any{
		"reasons": nonNil(v.Reasons), "redacted_text": v.RedactedBody, "redacted_title": v.RedactedTitle, "personal_question": v.Personal,
	})
}

func unavailableErr() *apierr.Error {
	return apierr.New(http.StatusServiceUnavailable, apierr.ChatUnavailable)
}

// piiEvents ghi MỘT dòng mỗi loại (chỉ loại + số đếm, không nội dung). Lỗi chỉ log.
func (s *Service) piiEvents(ctx context.Context, a Actor, courseID uuid.UUID, rs []Reason, action store.PiiAction) {
	q := store.New(s.Pool)
	for _, r := range rs {
		if err := q.InsertPIIEvent(ctx, store.InsertPIIEventParams{CourseID: courseID, UserID: a.UserID, Channel: store.ChatChannelPUBLIC, PiiType: store.PiiKind(r.Type), Count: int32(r.Count), Action: action}); err != nil && s.Log != nil { //nolint:gosec // đếm nhỏ
			s.Log.WarnContext(ctx, "thread: ghi pii_events lỗi", "error", err)
		}
	}
}

// RecordSwitched ghi SWITCHED cho `from-draft` (số Finding tính lại phía máy chủ).
func (s *Service) RecordSwitched(ctx context.Context, userID, courseID uuid.UUID, title, body string) {
	v, err := s.FW.CheckPost(ctx, courseID, title, body)
	if err != nil {
		return
	}
	s.piiEvents(ctx, Actor{UserID: userID}, courseID, v.Reasons, store.PiiActionSWITCHED)
}

// CreateIn là thân POST …/threads.
type CreateIn struct {
	Title  string   `json:"title" validate:"required"`
	Body   string   `json:"body" validate:"required"`
	Tags   []string `json:"tags"`
	WeekNo *int     `json:"week_no"`
	Redact bool     `json:"redact"`
}

func validField(field, msg string) *apierr.Error {
	return apierr.Validation(apierr.FieldError{Field: field, Code: "invalid", Message: msg})
}

// Create đăng thread. Thứ tự (SRS 4.9.3): khoá giờ thi TRƯỚC → lớp lưu trữ → kiểm đầu vào → tường lửa → một giao dịch (thread + việc AI + outbox).
func (s *Service) Create(ctx context.Context, a Actor, courseID uuid.UUID, in CreateIn) (ThreadView, error) {
	if err := s.guardExam(ctx, a); err != nil {
		return ThreadView{}, err
	}
	if st, err := store.New(s.Pool).ChatCourseStatus(ctx, courseID); err != nil {
		return ThreadView{}, fmt.Errorf("thread: trạng thái lớp: %w", err)
	} else if st == store.CourseStatusARCHIVED {
		return ThreadView{}, apierr.New(http.StatusConflict, apierr.CourseArchived)
	}
	title, body := strings.TrimSpace(in.Title), strings.TrimSpace(in.Body)
	if n := utf8.RuneCountInString(title); n < 1 || n > 200 {
		return ThreadView{}, validField("title", "Tiêu đề từ 1 đến 200 ký tự.")
	}
	if n := utf8.RuneCountInString(body); n < 1 || n > 8000 {
		return ThreadView{}, validField("body", "Nội dung từ 1 đến 8.000 ký tự.")
	}
	if in.WeekNo != nil && (*in.WeekNo < 1 || *in.WeekNo > 20) {
		return ThreadView{}, validField("week_no", "Tuần từ 1 đến 20.")
	}
	tags := in.Tags
	if tags == nil {
		tags = []string{}
	}
	if len(tags) > 5 {
		return ThreadView{}, validField("tags", "Tối đa 5 thẻ.")
	}
	for _, t := range tags {
		if n := utf8.RuneCountInString(t); n < 1 || n > 30 {
			return ThreadView{}, validField("tags", "Mỗi thẻ từ 1 đến 30 ký tự.")
		}
	}
	title, body, vec, err := s.gate(ctx, a, courseID, title, body, in.Redact)
	if err != nil {
		return ThreadView{}, err
	}
	var emb *pgvector.Vector
	if len(vec) == llm.EmbedDims {
		v := pgvector.NewVector(vec)
		emb = &v
	}
	var week *int32
	if in.WeekNo != nil {
		w := int32(*in.WeekNo) //nolint:gosec // 1–20
		week = &w
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return ThreadView{}, fmt.Errorf("thread: mở giao dịch: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	row, err := store.New(tx).ThreadInsert(ctx, store.ThreadInsertParams{CourseID: courseID, AuthorID: a.UserID, Title: title, Body: body, Tags: tags, WeekNo: weekPtr16(week), Embedding: emb})
	if err != nil {
		return ThreadView{}, fmt.Errorf("thread: lưu thread: %w", err)
	}
	if _, err := s.Jobs.EnqueueTx(ctx, tx, a.UserID, KindAnswer, map[string]any{"thread_id": row.ID, "course_id": courseID}); err != nil {
		return ThreadView{}, err
	}
	if _, err := outbox.Write(ctx, tx, TopicCreated, map[string]any{"thread_id": row.ID, "course_id": courseID}); err != nil {
		return ThreadView{}, fmt.Errorf("thread: outbox: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return ThreadView{}, fmt.Errorf("thread: commit: %w", err)
	}
	return s.Get(ctx, a, courseID, row.ID, httpx.PageParams{Limit: httpx.DefaultLimit})
}

func weekPtr16(w *int32) *int16 {
	if w == nil {
		return nil
	}
	v := int16(*w) //nolint:gosec // 1–20
	return &v
}

// guardExam: khoá giờ thi (Q2) chạy TRƯỚC tường lửa / DB: khoá → ghi CHAT_BLOCKED + 409 EXAM_IN_PROGRESS; tra lỗi → 503.
func (s *Service) guardExam(ctx context.Context, a Actor) error {
	lk, locked, err := s.Lock.IsLocked(ctx, a.UserID)
	if err != nil {
		return unavailableErr()
	}
	if locked {
		if err := s.Lock.RecordChatBlocked(ctx, a.UserID, lk.AttemptID); err != nil && s.Log != nil {
			s.Log.WarnContext(ctx, "thread: ghi CHAT_BLOCKED lỗi", "error", err)
		}
		return apierr.New(http.StatusConflict, apierr.ExamInProgress).WithDetails(map[string]any{"until": lk.Until.UTC().Format(time.RFC3339)})
	}
	return nil
}

// ---- bình luận -----------------------------------------------------------------------------------------------------------------------

// CommentIn là thân POST …/threads/{id}/posts.
type CommentIn struct {
	Body   string `json:"body" validate:"required"`
	Redact bool   `json:"redact"`
}

// Comment đăng bình luận (HUMAN) qua CÙNG tường lửa; AI không trả lời bình luận. Báo người đăng thread (không báo chính mình).
func (s *Service) Comment(ctx context.Context, a Actor, courseID, threadID uuid.UUID, in CommentIn) (PostOut, error) {
	q := store.New(s.Pool)
	if _, err := q.ThreadDetail(ctx, store.ThreadDetailParams{CourseID: courseID, ID: threadID}); errors.Is(err, pgx.ErrNoRows) {
		return PostOut{}, notFound()
	} else if err != nil {
		return PostOut{}, fmt.Errorf("thread: nạp thread: %w", err)
	}
	if st, err := q.ChatCourseStatus(ctx, courseID); err != nil {
		return PostOut{}, fmt.Errorf("thread: trạng thái lớp: %w", err)
	} else if st == store.CourseStatusARCHIVED {
		return PostOut{}, apierr.New(http.StatusConflict, apierr.CourseArchived)
	}
	body := strings.TrimSpace(in.Body)
	if n := utf8.RuneCountInString(body); n < 1 || n > 8000 {
		return PostOut{}, validField("body", "Nội dung từ 1 đến 8.000 ký tự.")
	}
	_, body, _, err := s.gate(ctx, a, courseID, "", body, in.Redact)
	if err != nil {
		return PostOut{}, err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return PostOut{}, fmt.Errorf("thread: mở giao dịch: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tq := store.New(tx)
	row, err := tq.InsertForumPost(ctx, store.InsertForumPostParams{CourseID: courseID, ThreadID: threadID, AuthorID: &a.UserID, Kind: store.PostKindHUMAN, Body: body, VerificationState: store.PostVerificationNONE, Citations: json.RawMessage("[]")})
	if err != nil {
		return PostOut{}, fmt.Errorf("thread: lưu bình luận: %w", err)
	}
	if err := tq.BumpForumThread(ctx, store.BumpForumThreadParams{CourseID: courseID, ID: threadID}); err != nil {
		return PostOut{}, fmt.Errorf("thread: cập nhật thread: %w", err)
	}
	if owner, err := tq.ThreadGetAuthor(ctx, threadID); err == nil && owner.AuthorID != a.UserID {
		s.notify(ctx, tq, owner.AuthorID, courseID, threadID, "THREAD_REPLY", "Có bình luận mới trong câu hỏi của bạn", "reply:"+row.ID.String())
	}
	if err := tx.Commit(ctx); err != nil {
		return PostOut{}, fmt.Errorf("thread: commit: %w", err)
	}
	name := ""
	_ = s.Pool.QueryRow(ctx, `select full_name from users where id=$1`, a.UserID).Scan(&name)
	return PostOut{ID: row.ID, Kind: "HUMAN", Author: &Author{FullName: name, Role: string(a.Role), IsMe: true}, Body: body, Citations: json.RawMessage("[]"), Version: row.Version, CreatedAt: row.CreatedAt}, nil
}

// notify tạo chuông cho `to` (idempotent theo dedupe_key). Lỗi chỉ log: không làm hỏng việc chính.
func (s *Service) notify(ctx context.Context, q *store.Queries, to, courseID, threadID uuid.UUID, typ, title, dedupe string) {
	link := "/threads/" + threadID.String()
	if _, err := q.InsertNotification(ctx, store.InsertNotificationParams{UserID: to, CourseID: &courseID, Type: typ, Title: title, Link: &link, DedupeKey: &dedupe}); err != nil && s.Log != nil {
		s.Log.WarnContext(ctx, "thread: tạo thông báo lỗi", "error", err)
	}
}

// ---- quyết định của Staff ----------------------------------------------------------------------------------------------------------

// Decision là loại quyết định.
type Decision string

// Ba quyết định.
const (
	Verify  Decision = "verify"
	Correct Decision = "correct"
	Reject  Decision = "reject"
)

// Decide: verify (PENDING|CORRECTED → VERIFIED), correct (PENDING|VERIFIED → CORRECTED, kèm version), reject (→ REJECTED). Lặp lại cùng quyết định → trả bài, không đổi,
// không thêm thông báo; mâu thuẫn → 409 POST_STATE_CONFLICT; `version` cũ → 409 VERSION_CONFLICT. Mỗi quyết định có audit_log + outbox + chuông cho người đăng.
func (s *Service) Decide(ctx context.Context, a Actor, courseID, postID uuid.UUID, d Decision, newBody string, version int32) (PostOut, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return PostOut{}, fmt.Errorf("thread: mở giao dịch: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	p, err := q.PostGet(ctx, store.PostGetParams{CourseID: courseID, ID: postID})
	if errors.Is(err, pgx.ErrNoRows) {
		return PostOut{}, notFound()
	}
	if err != nil {
		return PostOut{}, fmt.Errorf("thread: nạp bài: %w", err)
	}
	if p.Kind != store.PostKindAI {
		return PostOut{}, apierr.New(http.StatusConflict, apierr.PostStateConflict)
	}
	cur := p.VerificationState
	var target store.PostVerification
	var body *string
	switch d {
	case Verify:
		target = store.PostVerificationVERIFIED
		if cur == target {
			return s.postView(ctx, q, courseID, postID, a)
		}
		if cur != store.PostVerificationPENDING && cur != store.PostVerificationCORRECTED {
			return PostOut{}, apierr.New(http.StatusConflict, apierr.PostStateConflict)
		}
	case Reject:
		target = store.PostVerificationREJECTED
		if cur == target {
			return s.postView(ctx, q, courseID, postID, a)
		}
	case Correct:
		target = store.PostVerificationCORRECTED
		nb := strings.TrimSpace(newBody)
		if n := utf8.RuneCountInString(nb); n < 1 || n > 8000 {
			return PostOut{}, validField("body", "Nội dung từ 1 đến 8.000 ký tự.")
		}
		if cur == store.PostVerificationREJECTED {
			return PostOut{}, apierr.New(http.StatusConflict, apierr.PostStateConflict)
		}
		if p.Version != version {
			return PostOut{}, apierr.New(http.StatusConflict, apierr.VersionConflict).WithDetails(map[string]any{"current_version": p.Version})
		}
		if cur == target && nb == p.Body { // sửa lại đúng nội dung đã sửa: không đổi
			return s.postView(ctx, q, courseID, postID, a)
		}
		body = &nb
	}
	row, err := q.PostDecide(ctx, store.PostDecideParams{CourseID: courseID, ID: postID, State: target, NewBody: body, Actor: &a.UserID})
	if err != nil {
		return PostOut{}, fmt.Errorf("thread: ghi quyết định: %w", err)
	}
	after, _ := json.Marshal(map[string]any{"state": target, "version": row.Version})
	before, _ := json.Marshal(map[string]any{"state": cur, "version": p.Version})
	if _, err := q.InsertAuditLog(ctx, store.InsertAuditLogParams{CourseID: &courseID, ActorID: &a.UserID, Entity: "forum_post", EntityID: postID.String(), Action: "thread.post." + string(d), Before: before, After: after, TraceID: nonEmptyPtr(a.TraceID)}); err != nil {
		return PostOut{}, fmt.Errorf("thread: audit: %w", err)
	}
	if _, err := outbox.Write(ctx, tx, TopicPostDecided, map[string]any{"post_id": postID, "thread_id": p.ThreadID, "course_id": courseID, "decision": d}); err != nil {
		return PostOut{}, fmt.Errorf("thread: outbox: %w", err)
	}
	if target != store.PostVerificationREJECTED && p.ThreadAuthorID != a.UserID {
		title := "Giảng viên đã xác nhận câu trả lời cho câu hỏi của bạn"
		if target == store.PostVerificationCORRECTED {
			title = "Giảng viên đã sửa câu trả lời cho câu hỏi của bạn"
		}
		s.notify(ctx, q, p.ThreadAuthorID, courseID, p.ThreadID, "THREAD_VERIFIED", title, fmt.Sprintf("decided:%s:%s:%d", postID, target, row.Version))
	}
	out, err := s.postView(ctx, q, courseID, postID, a)
	if err != nil {
		return PostOut{}, err
	}
	return out, tx.Commit(ctx)
}

func nonEmptyPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// postView đọc lại một bài theo projection của `a` (Staff).
func (s *Service) postView(ctx context.Context, q *store.Queries, courseID, postID uuid.UUID, a Actor) (PostOut, error) {
	var tid uuid.UUID
	if err := s.Pool.QueryRow(ctx, `select thread_id from forum_posts where id=$1 and course_id=$2`, postID, courseID).Scan(&tid); err != nil {
		return PostOut{}, fmt.Errorf("thread: nạp bài: %w", err)
	}
	rows, err := q.ThreadPosts(ctx, store.ThreadPostsParams{CourseID: courseID, ThreadID: tid, IsStaff: true, PageLimit: 500})
	if err != nil {
		return PostOut{}, fmt.Errorf("thread: nạp bài: %w", err)
	}
	for _, r := range rows {
		if r.ID == postID {
			return postOut(r, a), nil
		}
	}
	return PostOut{}, notFound()
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func roleStr(r *store.UserRole) string {
	if r == nil {
		return ""
	}
	return string(*r)
}
