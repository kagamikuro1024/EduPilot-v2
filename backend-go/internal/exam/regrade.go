package exam

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/jobs"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/store"
)

// KindRegrade là loại việc `exam.regrade`: đặt lại bản nộp cuối trong phạm vi về QUEUED, hoặc (scope `recompute`) tính lại mọi lượt GRADED sau `override` lớn.
const KindRegrade = "exam.regrade"

const scopeRecompute = "recompute"

type regradePayload struct {
	CourseID  uuid.UUID  `json:"course_id"`
	ExamID    uuid.UUID  `json:"exam_id"`
	Scope     string     `json:"scope"`
	ItemID    *uuid.UUID `json:"item_id,omitempty"`
	AttemptID *uuid.UUID `json:"attempt_id,omitempty"`
}

// RegradeIn là thân `POST …/regrade`: `scope` ∈ all | item | attempt | errors.
type RegradeIn struct {
	Scope     string     `json:"scope"`
	ItemID    *uuid.UUID `json:"item_id"`
	AttemptID *uuid.UUID `json:"attempt_id"`
	Reason    string     `json:"reason"`
}

// Regrade: `POST …/regrade` (Giảng viên; 202 `{job_id}`). Đặt `regrading = true` rồi xếp việc; lượt GIỮ NGUYÊN trạng thái và điểm cũ tới khi tính lại xong (SRS 4.8.4).
func (s *Service) Regrade(ctx context.Context, actor, courseID, examID uuid.UUID, in RegradeIn) (uuid.UUID, error) {
	if _, err := checkReason(in.Reason); err != nil {
		return uuid.Nil, err
	}
	switch in.Scope {
	case "all", "errors":
	case "item":
		if in.ItemID == nil {
			return uuid.Nil, fieldErr("item_id", "VALUE_REQUIRED", "Cần mã câu.")
		}
	case "attempt":
		if in.AttemptID == nil {
			return uuid.Nil, fieldErr("attempt_id", "VALUE_REQUIRED", "Cần mã lượt làm.")
		}
	default:
		return uuid.Nil, fieldErr("scope", "INVALID", "Phạm vi chấm lại không hợp lệ.")
	}
	var jobID uuid.UUID
	err := s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
		e, err := s.lockExam(ctx, q, courseID, examID)
		if err != nil {
			return err
		}
		if !closed(e) {
			return lockedErr("status")
		}
		if e.Kind == store.ExamKindMCQ {
			return fieldErr("scope", "NO_CODE_ITEMS", "Bài thi này không có câu lập trình để chấm lại.")
		}
		if err := q.ExamSetRegrading(ctx, store.ExamSetRegradingParams{CourseID: courseID, ID: examID, Regrading: true}); err != nil {
			return fmt.Errorf("exam: đặt chấm lại: %w", err)
		}
		if err := audit(ctx, q, courseID, actor, "exam", examID, "exam.regrade", nil, map[string]any{"scope": in.Scope}); err != nil {
			return err
		}
		j, err := s.Jobs.Enqueue(ctx, actor, KindRegrade, regradePayload{CourseID: courseID, ExamID: examID, Scope: in.Scope, ItemID: in.ItemID, AttemptID: in.AttemptID})
		if err != nil {
			return fmt.Errorf("exam: xếp việc chấm lại: %w", err)
		}
		jobID = j.ID
		return nil
	})
	return jobID, err
}

// regradeJob (worker): `recompute` tính lại điểm; ngược lại đặt lại bản nộp cuối trong phạm vi về QUEUED để tick đưa vào máy chấm. Idempotent: chỉ đụng bản nộp đã DONE / ERROR.
func (w *Worker) regradeJob(ctx context.Context, j jobs.JobCtx) (any, error) {
	var p regradePayload
	if err := json.Unmarshal(j.Payload, &p); err != nil {
		return nil, &jobs.UserError{Code: "BAD_PAYLOAD", Message: "Việc chấm lại không hợp lệ."}
	}
	if err := store.New(w.Pool).ExamSetRegrading(ctx, store.ExamSetRegradingParams{CourseID: p.CourseID, ID: p.ExamID, Regrading: true}); err != nil { // làm mới `updated_at`: tick chờ thêm trước khi coi là xong
		return nil, fmt.Errorf("exam: đặt chấm lại: %w", err)
	}
	if p.Scope == scopeRecompute {
		n, err := w.Svc.RecomputeExam(ctx, p.CourseID, p.ExamID)
		return map[string]int{"recomputed": n}, err
	}
	rows, err := store.New(w.Pool).RegradeResetSubmissions(ctx, store.RegradeResetSubmissionsParams{Now: w.Svc.now(), CourseID: p.CourseID, ExamID: p.ExamID, Scope: p.Scope, ItemID: p.ItemID, AttemptID: p.AttemptID})
	if err != nil {
		return nil, fmt.Errorf("exam: đặt lại bản nộp: %w", err)
	}
	return map[string]int{"requeued": len(rows)}, nil
}

// RecomputeExam tính lại mọi lượt GRADED của bài (mỗi lượt một transaction; lượt lỗi không chặn lượt khác). Trả số lượt đã ghi.
func (s *Service) RecomputeExam(ctx context.Context, courseID, examID uuid.UUID) (int, error) {
	ids, err := store.New(s.Pool).AttemptsGradedOfExam(ctx, store.AttemptsGradedOfExamParams{CourseID: courseID, ExamID: examID})
	if err != nil {
		return 0, fmt.Errorf("exam: lượt đã chấm: %w", err)
	}
	n := 0
	var first error
	for _, a := range ids {
		ok, err := s.RecomputeAttempt(ctx, courseID, a.ID)
		if err != nil && first == nil {
			first = err
		}
		if ok {
			n++
		}
	}
	return n, first
}

// RecomputeAttempt (idempotent; SRS 4.8.4 `recomputeGraded`): lượt GRADED được tính lại điểm máy từ bản nộp cuối + đáp án / override hiện tại. Lượt KHÔNG quay về GRADING.
// Chưa đủ (còn bản nộp QUEUED / RUNNING / ERROR) → giữ điểm cũ, trả false. Sau PUBLISHED, điểm chính thức đổi → outbox `exam.regraded`.
func (s *Service) RecomputeAttempt(ctx context.Context, courseID, attemptID uuid.UUID) (bool, error) {
	done := false
	err := s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		a, err := q.AttemptLockByID(ctx, store.AttemptLockByIDParams{CourseID: courseID, ID: attemptID})
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && a.Status != store.AttemptStatusGRADED) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("exam: khoá lượt: %w", err)
		}
		e, err := q.ExamGet(ctx, store.ExamGetParams{CourseID: courseID, ID: a.ExamID})
		if err != nil {
			return fmt.Errorf("exam: đọc bài thi: %w", err)
		}
		items, err := q.ExamGradeItems(ctx, store.ExamGradeItemsParams{CourseID: courseID, ExamID: a.ExamID})
		if err != nil {
			return fmt.Errorf("exam: mục để chấm: %w", err)
		}
		gr, err := s.computeGrade(ctx, q, e, a, items)
		if err != nil || !gr.Complete {
			return err
		}
		row, err := q.AttemptUpdateGraded(ctx, store.AttemptUpdateGradedParams{CourseID: courseID, ID: attemptID, AutoScore: decimal.NullDecimal{Decimal: gr.Score, Valid: true}, Breakdown: gr.Breakdown, At: new(s.now()), Version: a.Version})
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("exam: ghi điểm tính lại: %w", err)
		}
		done = true
		if e.Status == store.ExamStatusPUBLISHED && !a.AdjustedScore.Valid && (!a.AutoScore.Valid || !a.AutoScore.Decimal.Equal(gr.Score)) {
			if _, err := outbox.Write(ctx, tx, TopicRegraded, map[string]any{"exam_id": a.ExamID, "course_id": courseID, "attempt_id": attemptID, "user_id": row.StudentID, "version": int(row.Version)}); err != nil {
				return fmt.Errorf("exam: outbox: %w", err)
			}
		}
		return nil
	})
	return done, err
}

// OnSubmissionDone (sự kiện `exam.submission_done` của bản SUBMIT): hoàn tất chấm lượt GRADING; lượt đã GRADED của bài đang chấm lại thì tính lại.
func (s *Service) OnSubmissionDone(ctx context.Context, courseID, attemptID uuid.UUID) error {
	ok, err := s.TryFinishGrading(ctx, courseID, attemptID)
	if err != nil || ok {
		return err
	}
	_, err = s.recomputeIfRegrading(ctx, courseID, attemptID)
	return err
}

func (s *Service) recomputeIfRegrading(ctx context.Context, courseID, attemptID uuid.UUID) (bool, error) {
	q := store.New(s.Pool)
	a, err := q.AttemptGetByID(ctx, store.AttemptGetByIDParams{CourseID: courseID, ID: attemptID})
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && a.Status != store.AttemptStatusGRADED) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("exam: đọc lượt: %w", err)
	}
	e, err := q.ExamGet(ctx, store.ExamGetParams{CourseID: courseID, ID: a.ExamID})
	if err != nil {
		return false, fmt.Errorf("exam: đọc bài thi: %w", err)
	}
	if !e.Regrading {
		return false, nil
	}
	return s.RecomputeAttempt(ctx, courseID, attemptID)
}

// RegradeSweep (tick, 5 s): với bài `regrading`, tính lại lượt có bản nộp cuối `judged_at > graded_at`; khi KHÔNG còn bản nộp QUEUED / RUNNING và mọi lượt đã tính lại → `regrading = false`
// (không phải khi việc xếp hàng xong — góp ý #13). Lượt có bản nộp ERROR giữ điểm cũ (việc `EXAM_GRADE_ERROR` báo Staff) và không chặn công bố.
func (s *Service) RegradeSweep(ctx context.Context) (int, error) {
	q := store.New(s.Pool)
	exs, err := q.ExamRegradingExams(ctx)
	if err != nil {
		return 0, fmt.Errorf("exam: bài đang chấm lại: %w", err)
	}
	n := 0
	var first error
	for _, e := range exs {
		pending, err := q.ExamRegradePending(ctx, store.ExamRegradePendingParams{CourseID: e.CourseID, ExamID: e.ID})
		if err != nil {
			return n, fmt.Errorf("exam: bản nộp đang chờ: %w", err)
		}
		need, err := q.AttemptsNeedingRecompute(ctx, store.AttemptsNeedingRecomputeParams{CourseID: e.CourseID, ExamID: e.ID})
		if err != nil {
			return n, fmt.Errorf("exam: lượt cần tính lại: %w", err)
		}
		for _, a := range need {
			if ok, err := s.RecomputeAttempt(ctx, e.CourseID, a.ID); err != nil && first == nil {
				first = err
			} else if ok {
				n++
			}
		}
		if pending == 0 && first == nil {
			if err := q.ExamSetRegrading(ctx, store.ExamSetRegradingParams{CourseID: e.CourseID, ID: e.ID, Regrading: false}); err != nil {
				return n, fmt.Errorf("exam: kết thúc chấm lại: %w", err)
			}
		}
	}
	return n, first
}
