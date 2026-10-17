package thread

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
	"github.com/edupilot/backend-go/internal/knowledge"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/store"
)

// Kiểm duyệt Threads (US-P4-06, SRS FEAT-escalation-notify 4.10): báo cáo, tự ẩn, xử lý của Staff, ghim.

// Sự kiện outbox mới (SRS 4.11).
const (
	TopicReported  = "thread.reported"
	TopicHidden    = "thread.hidden"
	TopicUnhidden  = "thread.unhidden"
	TopicDeleted   = "thread.deleted"
	TopicEdited    = "thread.edited"
	TopicAIAnswerd = "thread.ai_answered"

	// KindThread / KindPost là đích của báo cáo và kiểm duyệt.
	KindThread = "thread"
	KindPost   = "post"

	autoHideReason = "Nhiều báo cáo, chờ duyệt"
	undoWindow     = 5 * time.Second
)

var reportReasons = map[string]bool{"SPAM": true, "OFF_TOPIC": true, "PERSONAL_INFO": true, "OFFENSIVE": true, "WRONG_ANSWER": true, "OTHER": true} //nolint:gochecknoglobals // bảng tra cứu bất biến

// Limits là ngưỡng kiểm duyệt (REPORT_RATE_PER_HOUR, REPORT_AUTOHIDE_THRESHOLD, THREAD_PIN_MAX); 0 = mặc định.
type Limits struct {
	ReportPerHour, AutoHideAt, PinMax int
}

func (l Limits) rate() int   { return orDefault(l.ReportPerHour, 20) }
func (l Limits) hideAt() int { return orDefault(l.AutoHideAt, 3) }
func (l Limits) pinMax() int { return orDefault(l.PinMax, 3) }
func orDefault(v, d int) int {
	if v <= 0 {
		return d
	}
	return v
}

// target là một thread hoặc bình luận đã khoá hàng.
type target struct {
	kind         string
	id, threadID uuid.UUID
	author       *uuid.UUID // bài AI không có tác giả
	threadAuthor uuid.UUID
	version      int32
	hiddenAt     *time.Time
	hiddenBy     *uuid.UUID
	deleted      bool
	isAI         bool
}

func (t target) owner(a Actor) bool { return t.author != nil && *t.author == a.UserID }

// visibleTo: thread / bài ẩn chỉ Staff và chủ bài thấy; đã xoá không ai (404).
func (t target) visibleTo(a Actor) bool {
	return !t.deleted && (t.hiddenAt == nil || a.Staff() || t.owner(a))
}

func (s *Service) lockTarget(ctx context.Context, q *store.Queries, courseID uuid.UUID, kind string, id uuid.UUID) (target, error) {
	switch kind {
	case KindThread:
		r, err := q.ThreadLock(ctx, store.ThreadLockParams{CourseID: courseID, ID: id})
		if errors.Is(err, pgx.ErrNoRows) {
			return target{}, notFound()
		}
		if err != nil {
			return target{}, fmt.Errorf("thread: khoá thread: %w", err)
		}
		a := r.AuthorID
		return target{kind: kind, id: r.ID, threadID: r.ID, author: &a, threadAuthor: r.AuthorID, version: r.Version, hiddenAt: r.HiddenAt, hiddenBy: r.HiddenBy, deleted: r.DeletedAt != nil}, nil
	default:
		r, err := q.PostLockMod(ctx, store.PostLockModParams{CourseID: courseID, ID: id})
		if errors.Is(err, pgx.ErrNoRows) {
			return target{}, notFound()
		}
		if err != nil {
			return target{}, fmt.Errorf("thread: khoá bài: %w", err)
		}
		return target{kind: kind, id: r.ID, threadID: r.ThreadID, author: r.AuthorID, threadAuthor: r.ThreadAuthorID, version: r.Version, hiddenAt: r.HiddenAt, hiddenBy: r.HiddenBy, deleted: r.DeletedAt != nil || r.ThreadDeletedAt != nil, isAI: r.Kind == store.PostKindAI}, nil
	}
}

func (s *Service) writable(ctx context.Context, courseID uuid.UUID) error {
	st, err := store.New(s.Pool).ChatCourseStatus(ctx, courseID)
	if err != nil {
		return fmt.Errorf("thread: trạng thái lớp: %w", err)
	}
	if st == store.CourseStatusARCHIVED {
		return apierr.New(http.StatusConflict, apierr.CourseArchived)
	}
	return nil
}

func (s *Service) audit(ctx context.Context, q *store.Queries, courseID uuid.UUID, actor *uuid.UUID, trace, entity string, id uuid.UUID, action string, before, after any) error {
	b, _ := json.Marshal(before)
	a, _ := json.Marshal(after)
	if _, err := q.InsertAuditLog(ctx, store.InsertAuditLogParams{CourseID: &courseID, ActorID: actor, Entity: entity, EntityID: id.String(), Action: action, Before: b, After: a, TraceID: nonEmptyPtr(trace)}); err != nil {
		return fmt.Errorf("thread: audit: %w", err)
	}
	return nil
}

func entityOf(kind string) string {
	if kind == KindThread {
		return "forum_thread"
	}
	return "forum_post"
}

func (t target) payload(courseID uuid.UUID, extra map[string]any) map[string]any {
	m := map[string]any{"target_id": t.id, "thread_id": t.threadID, "course_id": courseID, "kind": t.kind}
	if t.author != nil {
		m["author_id"] = *t.author
	}
	for k, v := range extra {
		m[k] = v
	}
	return m
}

// syncKnowledge đưa tri thức của thread về đúng trạng thái hiện tại (DB thuần, cùng giao dịch của người gọi): bài AI VERIFIED / CORRECTED, chưa ẩn, chưa xoá → Upsert
// (embedding NULL + outbox `knowledge.sync`); còn lại → Remove (+ `document.changed`). Câu hỏi đã qua tường lửa PII lúc đăng nên không còn định danh để che.
func (s *Service) syncKnowledge(ctx context.Context, tx pgx.Tx, courseID, threadID uuid.UUID, by *uuid.UUID) error {
	src, err := store.New(tx).ThreadKnowledgeSource(ctx, store.ThreadKnowledgeSourceParams{CourseID: courseID, ThreadID: threadID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("thread: nguồn tri thức: %w", err)
	}
	ks := knowledge.Source{CourseID: courseID, Type: knowledge.SourcePost, ID: src.PostID, By: by}
	if !src.Unavailable && (src.State == string(store.PostVerificationVERIFIED) || src.State == string(store.PostVerificationCORRECTED)) {
		_, err = knowledge.Upsert(ctx, tx, ks, src.Title+"\n"+src.Question, src.Answer)
		return err
	}
	_, err = knowledge.Remove(ctx, tx, courseID, knowledge.SourcePost, src.PostID)
	return err
}

// ---- báo cáo ---------------------------------------------------------------------------------------------------------------------------

// ReportIn là thân `POST …/report`.
type ReportIn struct {
	Reason string  `json:"reason"`
	Note   *string `json:"note"`
}

// Report: một người một lần mỗi bài (409 ALREADY_REPORTED), không báo bài của mình (422), ≤ REPORT_RATE_PER_HOUR (429); bài ẩn / xoá / của lớp khác → 404.
// Dưới khoá của bài, người báo cáo thứ N khác nhau (chỉ đếm OPEN) tự ẩn bài: đúng một lần, đúng một sự kiện `thread.hidden`.
func (s *Service) Report(ctx context.Context, a Actor, courseID uuid.UUID, kind string, id uuid.UUID, in ReportIn) error {
	if !reportReasons[in.Reason] {
		return validField("reason", "Lý do không hợp lệ.")
	}
	var note *string
	if in.Note != nil {
		n := strings.TrimSpace(*in.Note)
		if n != "" {
			if in.Reason != "OTHER" {
				return validField("note", "Chỉ nhập mô tả khi chọn lý do khác.")
			}
			if utf8.RuneCountInString(n) > 200 {
				return validField("note", "Tối đa 200 ký tự.")
			}
			note = &n
		}
	}
	if err := s.writable(ctx, courseID); err != nil {
		return err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("thread: mở giao dịch: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	t, err := s.lockTarget(ctx, q, courseID, kind, id)
	if err != nil {
		return err
	}
	if !t.visibleTo(a) {
		return notFound()
	}
	if t.owner(a) {
		return apierr.New(http.StatusUnprocessableEntity, apierr.CannotReportOwn)
	}
	recent, err := q.ReportRecentByUser(ctx, store.ReportRecentByUserParams{ReporterID: a.UserID, Since: s.now().Add(-time.Hour)})
	if err != nil {
		return fmt.Errorf("thread: đếm báo cáo: %w", err)
	}
	if int(recent) >= s.Limits.rate() {
		return apierr.New(http.StatusTooManyRequests, apierr.RateLimited).WithRetryAfter(60)
	}
	if err := q.ReportLock(ctx, t.id.String()); err != nil {
		return fmt.Errorf("thread: khoá báo cáo: %w", err)
	}
	arg := store.ReportInsertParams{CourseID: courseID, ReporterID: a.UserID, Reason: store.PostReportReason(in.Reason), Note: note}
	if kind == KindThread {
		arg.ThreadID = &t.id
	} else {
		arg.PostID = &t.id
	}
	n, err := q.ReportInsert(ctx, arg)
	if err != nil {
		return fmt.Errorf("thread: lưu báo cáo: %w", err)
	}
	if n == 0 {
		return apierr.New(http.StatusConflict, apierr.AlreadyReported)
	}
	open, err := q.ReportCountOpenDistinct(ctx, &t.id)
	if err != nil {
		return fmt.Errorf("thread: đếm người báo cáo: %w", err)
	}
	if t.hiddenAt == nil && int(open) >= s.Limits.hideAt() {
		if err := s.hide(ctx, q, courseID, t, nil, autoHideReason); err != nil {
			return err
		}
		if err := s.audit(ctx, q, courseID, nil, a.TraceID, entityOf(kind), t.id, kind+".auto_hide", map[string]any{"hidden": false}, map[string]any{"hidden": true, "reports": open}); err != nil {
			return err
		}
		if err := s.syncKnowledge(ctx, tx, courseID, t.threadID, nil); err != nil {
			return err
		}
		if _, err := outbox.Write(ctx, tx, TopicHidden, t.payload(courseID, map[string]any{"auto": true})); err != nil {
			return fmt.Errorf("thread: outbox: %w", err)
		}
	}
	if _, err := outbox.Write(ctx, tx, TopicReported, t.payload(courseID, nil)); err != nil {
		return fmt.Errorf("thread: outbox: %w", err)
	}
	return tx.Commit(ctx)
}

// Unreport là "Hoàn tác" của người báo cáo: chỉ trong 5 giây và khi bài chưa bị (tự) ẩn.
func (s *Service) Unreport(ctx context.Context, a Actor, courseID uuid.UUID, kind string, id uuid.UUID) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("thread: mở giao dịch: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	t, err := s.lockTarget(ctx, q, courseID, kind, id)
	if err != nil {
		return err
	}
	if !t.visibleTo(a) {
		return notFound()
	}
	if t.hiddenAt != nil {
		return apierr.New(http.StatusConflict, apierr.PostStateConflict)
	}
	n, err := q.ReportDeleteOwn(ctx, store.ReportDeleteOwnParams{ReporterID: a.UserID, TargetID: &t.id, CourseID: courseID, Since: s.now().Add(-undoWindow)})
	if err != nil {
		return fmt.Errorf("thread: rút báo cáo: %w", err)
	}
	if n == 0 {
		return notFound()
	}
	if _, err := outbox.Write(ctx, tx, TopicReported, t.payload(courseID, nil)); err != nil {
		return fmt.Errorf("thread: outbox: %w", err)
	}
	return tx.Commit(ctx)
}

// ---- ẩn / bỏ ẩn / giữ bài --------------------------------------------------------------------------------------------------------------

func (s *Service) hide(ctx context.Context, q *store.Queries, courseID uuid.UUID, t target, by *uuid.UUID, reason string) error {
	if t.kind == KindThread {
		return q.ThreadHide(ctx, store.ThreadHideParams{ID: t.id, CourseID: courseID, Reason: &reason, By: by})
	}
	return q.PostHide(ctx, store.PostHideParams{ID: t.id, CourseID: courseID, Reason: &reason, By: by})
}

func (s *Service) unhide(ctx context.Context, q *store.Queries, courseID uuid.UUID, t target) error {
	if t.kind == KindThread {
		return q.ThreadUnhide(ctx, store.ThreadUnhideParams{ID: t.id, CourseID: courseID})
	}
	return q.PostUnhide(ctx, store.PostUnhideParams{ID: t.id, CourseID: courseID})
}

// ModResult là phản hồi của hide / unhide / dismiss-reports.
type ModResult struct {
	ID      uuid.UUID `json:"id"`
	Hidden  bool      `json:"hidden"`
	Version int32     `json:"version"`
}

// moderate chạy một lệnh Staff dưới khoá hàng: bài đã xoá → 404 (không còn gì để xử lý), `version` cũ → 409 VERSION_CONFLICT.
func (s *Service) moderate(ctx context.Context, a Actor, courseID uuid.UUID, kind string, id uuid.UUID, version int32, fn func(q *store.Queries, tx pgx.Tx, t target) (hidden bool, err error)) (ModResult, error) {
	if err := s.writable(ctx, courseID); err != nil {
		return ModResult{}, err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return ModResult{}, fmt.Errorf("thread: mở giao dịch: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	t, err := s.lockTarget(ctx, q, courseID, kind, id)
	if err != nil {
		return ModResult{}, err
	}
	if t.deleted {
		return ModResult{}, notFound()
	}
	if t.version != version {
		return ModResult{}, apierr.New(http.StatusConflict, apierr.VersionConflict).WithDetails(map[string]any{"current_version": t.version})
	}
	hidden, err := fn(q, tx, t)
	if err != nil {
		return ModResult{}, err
	}
	nv, err := s.versionOf(ctx, q, courseID, t)
	if err != nil {
		return ModResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ModResult{}, fmt.Errorf("thread: commit: %w", err)
	}
	return ModResult{ID: t.id, Hidden: hidden, Version: nv}, nil
}

func (s *Service) versionOf(ctx context.Context, q *store.Queries, courseID uuid.UUID, t target) (int32, error) {
	nt, err := s.lockTarget(ctx, q, courseID, t.kind, t.id)
	if err != nil {
		return 0, err
	}
	return nt.version, nil
}

// Hide là `Ẩn bài` của Staff: lý do 1–200 ký tự, `hidden_by` = người ẩn, báo cáo OPEN → ACTIONED, gỡ tri thức đã nạp từ bài.
// Bài đang tự ẩn (chờ duyệt) được nhận lại thành ẩn thủ công; bài đã ẩn thủ công → 409 POST_STATE_CONFLICT.
func (s *Service) Hide(ctx context.Context, a Actor, courseID uuid.UUID, kind string, id uuid.UUID, reason string, version int32) (ModResult, error) {
	reason = strings.TrimSpace(reason)
	if n := utf8.RuneCountInString(reason); n < 1 || n > 200 {
		return ModResult{}, validField("reason", "Lý do từ 1 đến 200 ký tự.")
	}
	return s.moderate(ctx, a, courseID, kind, id, version, func(q *store.Queries, tx pgx.Tx, t target) (bool, error) {
		if t.hiddenAt != nil && t.hiddenBy != nil {
			return false, apierr.New(http.StatusConflict, apierr.PostStateConflict)
		}
		if t.hiddenAt != nil { // tự ẩn → thủ công
			if err := s.unhide(ctx, q, courseID, t); err != nil {
				return false, err
			}
		}
		if err := s.hide(ctx, q, courseID, t, &a.UserID, reason); err != nil {
			return false, err
		}
		if _, err := q.ReportHandleOpen(ctx, store.ReportHandleOpenParams{Status: store.PostReportStatusACTIONED, HandledBy: &a.UserID, TargetID: &t.id, CourseID: courseID}); err != nil {
			return false, fmt.Errorf("thread: xử lý báo cáo: %w", err)
		}
		if err := s.audit(ctx, q, courseID, &a.UserID, a.TraceID, entityOf(kind), t.id, kind+".hide", map[string]any{"hidden": t.hiddenAt != nil}, map[string]any{"hidden": true, "reason": reason}); err != nil {
			return false, err
		}
		if err := s.syncKnowledge(ctx, tx, courseID, t.threadID, &a.UserID); err != nil {
			return false, err
		}
		_, err := outbox.Write(ctx, tx, TopicHidden, t.payload(courseID, map[string]any{"auto": false, "reason": reason}))
		return true, err
	})
}

// Unhide là `Bỏ ẩn`: xoá `hidden_*`, nạp lại tri thức nếu bài AI đã được xác nhận.
func (s *Service) Unhide(ctx context.Context, a Actor, courseID uuid.UUID, kind string, id uuid.UUID, version int32) (ModResult, error) {
	return s.moderate(ctx, a, courseID, kind, id, version, func(q *store.Queries, tx pgx.Tx, t target) (bool, error) {
		if t.hiddenAt == nil {
			return false, apierr.New(http.StatusConflict, apierr.PostStateConflict)
		}
		if err := s.unhide(ctx, q, courseID, t); err != nil {
			return false, err
		}
		if err := s.audit(ctx, q, courseID, &a.UserID, a.TraceID, entityOf(kind), t.id, kind+".unhide", map[string]any{"hidden": true}, map[string]any{"hidden": false}); err != nil {
			return false, err
		}
		if err := s.syncKnowledge(ctx, tx, courseID, t.threadID, &a.UserID); err != nil {
			return false, err
		}
		_, err := outbox.Write(ctx, tx, TopicUnhidden, t.payload(courseID, map[string]any{"auto": false}))
		return false, err
	})
}

// DismissReports là `Giữ bài`: báo cáo OPEN → DISMISSED; bài đang tự ẩn được hiện lại (ẩn thủ công giữ nguyên).
func (s *Service) DismissReports(ctx context.Context, a Actor, courseID uuid.UUID, kind string, id uuid.UUID, version int32) (ModResult, error) {
	return s.moderate(ctx, a, courseID, kind, id, version, func(q *store.Queries, tx pgx.Tx, t target) (bool, error) {
		if _, err := q.ReportHandleOpen(ctx, store.ReportHandleOpenParams{Status: store.PostReportStatusDISMISSED, HandledBy: &a.UserID, TargetID: &t.id, CourseID: courseID}); err != nil {
			return false, fmt.Errorf("thread: xử lý báo cáo: %w", err)
		}
		hidden := t.hiddenAt != nil
		if hidden && t.hiddenBy == nil {
			if err := s.unhide(ctx, q, courseID, t); err != nil {
				return false, err
			}
			hidden = false
			if err := s.syncKnowledge(ctx, tx, courseID, t.threadID, &a.UserID); err != nil {
				return false, err
			}
			if _, err := outbox.Write(ctx, tx, TopicUnhidden, t.payload(courseID, map[string]any{"auto": true})); err != nil {
				return false, fmt.Errorf("thread: outbox: %w", err)
			}
		}
		if err := s.audit(ctx, q, courseID, &a.UserID, a.TraceID, entityOf(kind), t.id, kind+".dismiss_reports", nil, map[string]any{"hidden": hidden}); err != nil {
			return false, err
		}
		_, err := outbox.Write(ctx, tx, TopicReported, t.payload(courseID, nil))
		return hidden, err
	})
}

// ---- hàng chờ --------------------------------------------------------------------------------------------------------------------------

// QueueItem là một dòng của hàng chờ kiểm duyệt: không có người báo cáo, không có số của sinh viên.
type QueueItem struct {
	Kind         string     `json:"kind"`
	ThreadID     uuid.UUID  `json:"thread_id"`
	PostID       *uuid.UUID `json:"post_id"`
	Title        string     `json:"title_or_excerpt"`
	Reports      Reports    `json:"reports"`
	HiddenAt     *time.Time `json:"hidden_at"`
	HiddenByAuto bool       `json:"hidden_by_auto"`
	CreatedAt    time.Time  `json:"created_at"`
}

// Reports là số báo cáo OPEN và theo lý do.
type Reports struct {
	Total    int            `json:"total"`
	ByReason map[string]int `json:"by_reason"`
}

// QueuePage là một trang hàng chờ.
type QueuePage struct {
	Items      []QueueItem `json:"items"`
	NextCursor *string     `json:"next_cursor"`
}

// Queue: status OPEN (có báo cáo OPEN) hoặc HIDDEN (đang ẩn); phân trang con trỏ.
func (s *Service) Queue(ctx context.Context, courseID uuid.UUID, status string, p httpx.PageParams) (QueuePage, error) {
	if status != "OPEN" && status != "HIDDEN" {
		return QueuePage{}, validField("status", "Chỉ nhận OPEN hoặc HIDDEN.")
	}
	arg := store.ModerationQueueParams{CourseID: courseID, Status: status, PageLimit: int32(p.Fetch())} //nolint:gosec // ≤ 101
	if p.Cursor != nil {
		arg.CursorAt, arg.CursorID = &p.Cursor.CreatedAt, &p.Cursor.ID
	}
	q := store.New(s.Pool)
	rows, err := q.ModerationQueue(ctx, arg)
	if err != nil {
		return QueuePage{}, fmt.Errorf("thread: hàng chờ: %w", err)
	}
	out := QueuePage{Items: make([]QueueItem, 0, len(rows))}
	if len(rows) > p.Limit {
		rows = rows[:p.Limit]
		last := rows[len(rows)-1]
		if at, ok := last.Qat.(time.Time); ok {
			c := httpx.EncodeCursor(at, last.ID.String())
			out.NextCursor = &c
		}
	}
	ids := make([]string, len(rows))
	for i, r := range rows {
		ids[i] = r.ID.String()
	}
	stats, err := q.ModerationReportStats(ctx, store.ModerationReportStatsParams{CourseID: courseID, Ids: ids})
	if err != nil {
		return QueuePage{}, fmt.Errorf("thread: số báo cáo: %w", err)
	}
	by := map[uuid.UUID]*Reports{}
	for _, st := range stats {
		if st.TargetID == nil {
			continue
		}
		r := by[*st.TargetID]
		if r == nil {
			r = &Reports{ByReason: map[string]int{}}
			by[*st.TargetID] = r
		}
		r.Total += int(st.N)
		r.ByReason[st.Reason] = int(st.N)
	}
	for _, r := range rows {
		it := QueueItem{Kind: r.Kind, ThreadID: r.ThreadID, PostID: r.PostID, Title: r.TitleOrExcerpt, HiddenAt: r.HiddenAt, HiddenByAuto: r.HiddenByAuto, CreatedAt: r.CreatedAt, Reports: Reports{ByReason: map[string]int{}}}
		if x := by[r.ID]; x != nil {
			it.Reports = *x
		}
		out.Items = append(out.Items, it)
	}
	return out, nil
}

// ---- ghim ------------------------------------------------------------------------------------------------------------------------------

// Pin / Unpin (Staff): ≤ THREAD_PIN_MAX mỗi lớp, so sánh dưới khoá lớp (409 PIN_LIMIT). Ghim lại thread đã ghim là no-op.
func (s *Service) Pin(ctx context.Context, a Actor, courseID, id uuid.UUID, pin bool) error {
	if err := s.writable(ctx, courseID); err != nil {
		return err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("thread: mở giao dịch: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	if _, err := tx.Exec(ctx, `select pg_advisory_xact_lock(hashtextextended('pin:' || $1::text, 0))`, courseID.String()); err != nil {
		return fmt.Errorf("thread: khoá lớp: %w", err)
	}
	t, err := s.lockTarget(ctx, q, courseID, KindThread, id)
	if err != nil {
		return err
	}
	if t.deleted {
		return notFound()
	}
	pinned := false
	if r, err := q.ThreadLock(ctx, store.ThreadLockParams{CourseID: courseID, ID: id}); err == nil {
		pinned = r.PinnedAt != nil
	}
	if pin && !pinned {
		n, err := q.ThreadPinCount(ctx, courseID)
		if err != nil {
			return fmt.Errorf("thread: đếm ghim: %w", err)
		}
		if int(n) >= s.Limits.pinMax() {
			return apierr.New(http.StatusConflict, apierr.PinLimit)
		}
	}
	if pin != pinned {
		if err := q.ThreadSetPinned(ctx, store.ThreadSetPinnedParams{ID: id, CourseID: courseID, Pinned: pin}); err != nil {
			return fmt.Errorf("thread: ghim: %w", err)
		}
		act := "thread.unpin"
		if pin {
			act = "thread.pin"
		}
		if err := s.audit(ctx, q, courseID, &a.UserID, a.TraceID, "forum_thread", id, act, map[string]any{"pinned": pinned}, map[string]any{"pinned": pin}); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
