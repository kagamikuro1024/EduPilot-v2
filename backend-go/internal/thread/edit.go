package thread

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/store"
)

// Sửa / xoá của chủ bài (US-P4-06 AC5, AC6; SRS 4.10.5).

func forbidden() *apierr.Error { return apierr.New(http.StatusForbidden, apierr.Forbidden) }

// checkThreadInput kiểm đầu vào của đăng / sửa thread (độ dài, tuần, thẻ) và trả bản đã cắt khoảng trắng.
func checkThreadInput(title, body string, tags []string, week *int) (string, string, []string, error) {
	title, body = strings.TrimSpace(title), strings.TrimSpace(body)
	if n := utf8.RuneCountInString(title); n < 1 || n > 200 {
		return "", "", nil, validField("title", "Tiêu đề từ 1 đến 200 ký tự.")
	}
	if n := utf8.RuneCountInString(body); n < 1 || n > 8000 {
		return "", "", nil, validField("body", "Nội dung từ 1 đến 8.000 ký tự.")
	}
	if week != nil && (*week < 1 || *week > 20) {
		return "", "", nil, validField("week_no", "Tuần từ 1 đến 20.")
	}
	if tags == nil {
		tags = []string{}
	}
	if len(tags) > 5 {
		return "", "", nil, validField("tags", "Tối đa 5 thẻ.")
	}
	for _, t := range tags {
		if n := utf8.RuneCountInString(t); n < 1 || n > 30 {
			return "", "", nil, validField("tags", "Mỗi thẻ từ 1 đến 30 ký tự.")
		}
	}
	return title, body, tags, nil
}

// EditThreadIn là thân `PUT …/threads/{tid}`.
type EditThreadIn struct {
	Title   string   `json:"title"`
	Body    string   `json:"body"`
	Tags    []string `json:"tags"`
	WeekNo  *int     `json:"week_no"`
	Version int32    `json:"version"`
	Redact  bool     `json:"redact"`
}

// EditThread: chỉ chủ bài (người khác, kể cả Staff → 403); chạy CÙNG tường lửa như đăng (PII → 422, không lưu bản thô); câu hỏi đã có câu trả lời VERIFIED / CORRECTED → 409
// EDIT_LOCKED_VERIFIED; sai `version` → 409; không tự bỏ ẩn. Bài AI còn PENDING → REJECTED (tự động, không báo ai), `ai_state` về PENDING và xếp lại việc AI.
func (s *Service) EditThread(ctx context.Context, a Actor, courseID, id uuid.UUID, in EditThreadIn) (ThreadView, error) {
	if err := s.writable(ctx, courseID); err != nil {
		return ThreadView{}, err
	}
	title, body, tags, err := checkThreadInput(in.Title, in.Body, in.Tags, in.WeekNo)
	if err != nil {
		return ThreadView{}, err
	}
	cur, err := s.lockTargetRO(ctx, courseID, id)
	if err != nil {
		return ThreadView{}, err
	}
	if !cur.owner(a) {
		return ThreadView{}, forbidden()
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
	tq := store.New(tx)
	t, err := s.lockTarget(ctx, tq, courseID, KindThread, id)
	if err != nil {
		return ThreadView{}, err
	}
	if t.deleted {
		return ThreadView{}, notFound()
	}
	if t.version != in.Version {
		return ThreadView{}, apierr.New(http.StatusConflict, apierr.VersionConflict).WithDetails(map[string]any{"current_version": t.version})
	}
	states, err := tq.ThreadAIPostStates(ctx, store.ThreadAIPostStatesParams{ThreadID: id, CourseID: courseID})
	if err != nil {
		return ThreadView{}, fmt.Errorf("thread: bài AI: %w", err)
	}
	pending := false
	for _, st := range states {
		switch st.VerificationState {
		case store.PostVerificationVERIFIED, store.PostVerificationCORRECTED:
			return ThreadView{}, apierr.New(http.StatusConflict, apierr.EditLockedVerified)
		case store.PostVerificationPENDING:
			pending = true
		}
	}
	if err := tq.ThreadEdit(ctx, store.ThreadEditParams{ID: id, CourseID: courseID, Title: title, Body: body, Tags: tags, WeekNo: weekPtr16(week), Embedding: emb}); err != nil {
		return ThreadView{}, fmt.Errorf("thread: sửa thread: %w", err)
	}
	if pending {
		if _, err := tq.ThreadRejectPendingAI(ctx, store.ThreadRejectPendingAIParams{ThreadID: id, CourseID: courseID}); err != nil {
			return ThreadView{}, fmt.Errorf("thread: loại bài AI cũ: %w", err)
		}
		if err := tq.ThreadResetAIState(ctx, store.ThreadResetAIStateParams{ID: id, CourseID: courseID}); err != nil {
			return ThreadView{}, fmt.Errorf("thread: đặt lại AI: %w", err)
		}
		if _, err := s.Jobs.EnqueueTx(ctx, tx, a.UserID, KindAnswer, map[string]any{"thread_id": id, "course_id": courseID}); err != nil {
			return ThreadView{}, err
		}
	}
	if err := s.audit(ctx, tq, courseID, &a.UserID, a.TraceID, "forum_thread", id, "thread.edit", nil, map[string]any{"ai_rerun": pending}); err != nil {
		return ThreadView{}, err
	}
	if _, err := outbox.Write(ctx, tx, TopicEdited, t.payload(courseID, map[string]any{"ai_rerun": pending})); err != nil {
		return ThreadView{}, fmt.Errorf("thread: outbox: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return ThreadView{}, fmt.Errorf("thread: commit: %w", err)
	}
	return s.Get(ctx, a, courseID, id, httpx.PageParams{Limit: httpx.DefaultLimit})
}

// lockTargetRO đọc thread để kiểm chủ bài trước khi chạy tường lửa (không giữ khoá trong lúc nhúng vectơ).
func (s *Service) lockTargetRO(ctx context.Context, courseID, id uuid.UUID) (target, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return target{}, fmt.Errorf("thread: mở giao dịch: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	t, err := s.lockTarget(ctx, store.New(tx), courseID, KindThread, id)
	if err != nil {
		return target{}, err
	}
	if t.deleted {
		return target{}, notFound()
	}
	return t, nil
}

// EditPostIn là thân `PUT …/posts/{pid}`.
type EditPostIn struct {
	Body    string `json:"body"`
	Version int32  `json:"version"`
	Redact  bool   `json:"redact"`
}

// EditPost: bình luận của chính mình, cùng tường lửa; bài AI không sửa được bằng đường này.
func (s *Service) EditPost(ctx context.Context, a Actor, courseID, id uuid.UUID, in EditPostIn) (PostOut, error) {
	if err := s.writable(ctx, courseID); err != nil {
		return PostOut{}, err
	}
	body := strings.TrimSpace(in.Body)
	if n := utf8.RuneCountInString(body); n < 1 || n > 8000 {
		return PostOut{}, validField("body", "Nội dung từ 1 đến 8.000 ký tự.")
	}
	tx0, err := s.Pool.Begin(ctx)
	if err != nil {
		return PostOut{}, fmt.Errorf("thread: mở giao dịch: %w", err)
	}
	cur, err := s.lockTarget(ctx, store.New(tx0), courseID, KindPost, id)
	_ = tx0.Rollback(ctx)
	if err != nil {
		return PostOut{}, err
	}
	if cur.deleted {
		return PostOut{}, notFound()
	}
	if cur.isAI || !cur.owner(a) {
		return PostOut{}, forbidden()
	}
	_, body, _, err = s.gate(ctx, a, courseID, "", body, in.Redact)
	if err != nil {
		return PostOut{}, err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return PostOut{}, fmt.Errorf("thread: mở giao dịch: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	t, err := s.lockTarget(ctx, q, courseID, KindPost, id)
	if err != nil {
		return PostOut{}, err
	}
	if t.deleted {
		return PostOut{}, notFound()
	}
	if t.version != in.Version {
		return PostOut{}, apierr.New(http.StatusConflict, apierr.VersionConflict).WithDetails(map[string]any{"current_version": t.version})
	}
	if err := q.PostEdit(ctx, store.PostEditParams{ID: id, CourseID: courseID, Body: body}); err != nil {
		return PostOut{}, fmt.Errorf("thread: sửa bình luận: %w", err)
	}
	if err := s.audit(ctx, q, courseID, &a.UserID, a.TraceID, "forum_post", id, "post.edit", nil, nil); err != nil {
		return PostOut{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return PostOut{}, fmt.Errorf("thread: commit: %w", err)
	}
	return s.postView(ctx, store.New(s.Pool), courseID, id, a)
}

// DeleteThread / DeletePost: chỉ chủ bài, xoá mềm; tri thức nạp từ bài bị gỡ; thread bị xoá kéo theo mọi bình luận biến khỏi người đọc.
func (s *Service) DeleteThread(ctx context.Context, a Actor, courseID, id uuid.UUID) error {
	return s.deleteOwn(ctx, a, courseID, KindThread, id)
}

// DeletePost xoá mềm bình luận của chính mình.
func (s *Service) DeletePost(ctx context.Context, a Actor, courseID, id uuid.UUID) error {
	return s.deleteOwn(ctx, a, courseID, KindPost, id)
}

func (s *Service) deleteOwn(ctx context.Context, a Actor, courseID uuid.UUID, kind string, id uuid.UUID) error {
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
	if t.deleted {
		return notFound()
	}
	if t.isAI || !t.owner(a) {
		return forbidden()
	}
	if kind == KindThread {
		err = q.ThreadSoftDelete(ctx, store.ThreadSoftDeleteParams{ID: id, CourseID: courseID})
	} else {
		err = q.PostSoftDelete(ctx, store.PostSoftDeleteParams{ID: id, CourseID: courseID})
	}
	if err != nil {
		return fmt.Errorf("thread: xoá mềm: %w", err)
	}
	if err := s.syncKnowledge(ctx, tx, courseID, t.threadID, nil); err != nil {
		return err
	}
	if err := s.audit(ctx, q, courseID, &a.UserID, a.TraceID, entityOf(kind), id, kind+".delete", nil, nil); err != nil {
		return err
	}
	if _, err := outbox.Write(ctx, tx, TopicDeleted, t.payload(courseID, nil)); err != nil {
		return fmt.Errorf("thread: outbox: %w", err)
	}
	return tx.Commit(ctx)
}
