package exam

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/store"
)

// scheduleErrors là điều kiện lên lịch (SRS 4.2.4): trả TOÀN BỘ lỗi một lần. Chạy trong transaction đã khoá bài; các câu hỏi bị khoá chia sẻ.
func (s *Service) scheduleErrors(ctx context.Context, q *store.Queries, e store.Exam) ([]apierr.FieldError, error) {
	var errs []apierr.FieldError
	st := stateOf(e)
	now := s.now()
	if st.OpensAt == nil {
		errs = append(errs, ve("opens_at", "VALUE_REQUIRED", "Cần giờ mở bài."))
	} else if st.OpensAt.Before(now.Add(s.Limits.minLead())) {
		errs = append(errs, ve("opens_at", "OPENS_IN_PAST", fmt.Sprintf("Giờ mở phải sau bây giờ ít nhất %d giây.", int(s.Limits.minLead()/time.Second))))
	}
	if st.ClosesAt == nil {
		errs = append(errs, ve("closes_at", "VALUE_REQUIRED", "Cần giờ đóng bài."))
	}
	if st.DurationMinutes == nil {
		errs = append(errs, ve("duration_minutes", "VALUE_REQUIRED", "Cần thời lượng làm bài."))
	}
	errs = append(errs, s.Limits.validate(st)...)

	items, err := q.ExamItemListShared(ctx, store.ExamItemListSharedParams{CourseID: e.CourseID, ExamID: e.ID})
	if err != nil {
		return nil, fmt.Errorf("exam: mục của bài thi: %w", err)
	}
	if len(items) == 0 {
		errs = append(errs, ve("items", "NO_ITEMS", "Bài thi cần ít nhất một câu hỏi."))
	}
	for i, it := range items {
		f := fmt.Sprintf("items[%d]", i)
		if it.ReviewStatus != store.QuestionReviewStatusAPPROVED || it.ArchivedAt != nil {
			errs = append(errs, ve(f+".question_id", "ITEM_NOT_APPROVED", fmt.Sprintf("Câu %q chưa được duyệt hoặc đã lưu trữ.", it.Title)))
			continue
		}
		if it.Type != store.QuestionTypeCODE {
			continue
		}
		ae, err := s.approvalErrors(ctx, q, store.QuestionBank{ID: it.QuestionID, CourseID: e.CourseID, Type: it.Type})
		if err != nil {
			return nil, err
		}
		for _, a := range ae { // CODE_TESTS_MISSING, TOTAL_WEIGHT_ZERO, REFERENCE_NOT_VERIFIED — cùng điều kiện với lúc duyệt câu
			errs = append(errs, ve(f+".question_id", a.Code, fmt.Sprintf("Câu %q: %s", it.Title, a.Message)))
		}
	}
	code, err := q.ExamCodeQuestions(ctx, store.ExamCodeQuestionsParams{CourseID: e.CourseID, ExamID: e.ID})
	if err != nil {
		return nil, fmt.Errorf("exam: thời gian chấm các câu code: %w", err)
	}
	for _, c := range code {
		total := int64(c.ApprovedTests) * 3 * int64(c.TimeLimitMs) / 1000 // clockLimit = 3 × time_limit_ms mỗi test (góp ý #3)
		if total > int64(s.Limits.maxTotal()) {
			errs = append(errs, ve("items", "CODE_TIME_BUDGET_EXCEEDED", fmt.Sprintf("Câu %q: tổng thời gian chấm tối đa %d giây vượt %d giây. Giảm số test hoặc giới hạn thời gian.", c.Title, total, s.Limits.maxTotal())))
		}
	}
	return errs, nil
}

// ScheduleExam đưa bài DRAFT sang SCHEDULED (chỉ Giảng viên — guard ở route). Gọi lại khi đã SCHEDULED: trả bản hiện tại, không thêm thông báo.
func (s *Service) ScheduleExam(ctx context.Context, actor, courseID, examID uuid.UUID) (ExamDetail, error) {
	var out ExamDetail
	err := s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		e, err := s.lockExam(ctx, q, courseID, examID)
		if err != nil {
			return err
		}
		switch e.Status {
		case store.ExamStatusSCHEDULED:
			out, err = s.examDetail(ctx, q, e)
			return err
		case store.ExamStatusDRAFT:
		default:
			return lockedErr("status")
		}
		errs, err := s.scheduleErrors(ctx, q, e)
		if err != nil {
			return err
		}
		if len(errs) > 0 {
			return apierr.Validation(errs...)
		}
		row, err := q.ExamSetStatus(ctx, store.ExamSetStatusParams{CourseID: courseID, ID: examID, FromStatus: store.ExamStatusDRAFT, ToStatus: store.ExamStatusSCHEDULED})
		if err != nil {
			return fmt.Errorf("exam: lên lịch: %w", err)
		}
		if err := audit(ctx, q, courseID, actor, "exam", examID, "exam.schedule", map[string]any{"status": StatusDraft}, map[string]any{"status": StatusScheduled, "version": int(row.Version)}); err != nil {
			return err
		}
		if _, err := outbox.Write(ctx, tx, TopicExamScheduled, map[string]any{"exam_id": examID, "course_id": courseID}); err != nil {
			return fmt.Errorf("exam: outbox: %w", err)
		}
		out, err = s.examDetail(ctx, q, row)
		return err
	})
	return out, err
}

// UnscheduleExam đưa SCHEDULED về DRAFT khi chưa mở và chưa có lượt làm; ngược lại 409 EXAM_LOCKED (`reason` ∈ status, opened, has_attempts).
func (s *Service) UnscheduleExam(ctx context.Context, actor, courseID, examID uuid.UUID) (ExamDetail, error) {
	var out ExamDetail
	err := s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		e, err := s.lockExam(ctx, q, courseID, examID)
		if err != nil {
			return err
		}
		if e.Status != store.ExamStatusSCHEDULED {
			return lockedErr("status")
		}
		if e.OpensAt != nil && !s.now().Before(*e.OpensAt) {
			return lockedErr("opened")
		}
		n, err := q.ExamAttemptCounts(ctx, store.ExamAttemptCountsParams{CourseID: courseID, ExamID: examID})
		if err != nil {
			return fmt.Errorf("exam: đếm lượt làm: %w", err)
		}
		if n.Started > 0 {
			return lockedErr("has_attempts")
		}
		row, err := q.ExamSetStatus(ctx, store.ExamSetStatusParams{CourseID: courseID, ID: examID, FromStatus: store.ExamStatusSCHEDULED, ToStatus: store.ExamStatusDRAFT})
		if err != nil {
			return fmt.Errorf("exam: bỏ lịch: %w", err)
		}
		if err := audit(ctx, q, courseID, actor, "exam", examID, "exam.unschedule", map[string]any{"status": StatusScheduled}, map[string]any{"status": StatusDraft, "version": int(row.Version)}); err != nil {
			return err
		}
		if _, err := outbox.Write(ctx, tx, TopicExamUnscheduled, map[string]any{"exam_id": examID, "course_id": courseID}); err != nil {
			return fmt.Errorf("exam: outbox: %w", err)
		}
		out, err = s.examDetail(ctx, q, row)
		return err
	})
	return out, err
}

// maxExtend: một lần gia hạn tối đa thêm 24 giờ so với giờ đóng cũ.
const maxExtend = 24 * time.Hour

// ExtendExam lùi `closes_at` muộn hơn (SCHEDULED / OPEN, chỉ Giảng viên). Lượt IN_PROGRESS bị chặn bởi mốc cũ được tính lại.
func (s *Service) ExtendExam(ctx context.Context, actor, courseID, examID uuid.UUID, closesAt time.Time) (ExamDetail, error) {
	var out ExamDetail
	err := s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
		e, err := s.lockExam(ctx, q, courseID, examID)
		if err != nil {
			return err
		}
		if (e.Status != store.ExamStatusSCHEDULED && e.Status != store.ExamStatusOPEN) || e.ClosesAt == nil || !s.now().Before(*e.ClosesAt) {
			return lockedErr("status") // CLOSED / PUBLISHED, hoặc đã quá giờ đóng (kể cả khi bộ lập lịch chưa kịp đóng)
		}
		old := *e.ClosesAt
		switch {
		case !closesAt.After(old):
			return fieldErr("closes_at", "CLOSES_NOT_LATER", "Giờ đóng mới phải muộn hơn giờ đóng hiện tại.")
		case closesAt.After(old.Add(maxExtend)):
			return fieldErr("closes_at", "LIMIT_OUT_OF_RANGE", "Mỗi lần gia hạn tối đa 24 giờ.")
		}
		row, err := q.ExamExtend(ctx, store.ExamExtendParams{CourseID: courseID, ID: examID, ClosesAt: &closesAt})
		if err != nil {
			return fmt.Errorf("exam: gia hạn: %w", err)
		}
		moved, err := q.ExamAttemptsExtend(ctx, store.ExamAttemptsExtendParams{CourseID: courseID, ExamID: examID, OldClosesAt: old, NewClosesAt: closesAt})
		if err != nil {
			return fmt.Errorf("exam: tính lại hạn các lượt đang làm: %w", err)
		}
		if err := audit(ctx, q, courseID, actor, "exam", examID, "exam.extend", map[string]any{"closes_at": old}, map[string]any{"closes_at": closesAt, "attempts_moved": moved, "version": int(row.Version)}); err != nil {
			return err
		}
		out, err = s.examDetail(ctx, q, row)
		return err
	})
	if err == nil && s.Redis != nil {
		// hạn các lượt đang làm đã dài ra: TTL của khoá chat theo hạn mới (≤ 10 s lệch — AC1, AC11)
		if ids, qerr := store.New(s.Pool).LockRunningStudents(ctx, store.LockRunningStudentsParams{CourseID: courseID, ExamID: examID}); qerr == nil {
			for _, id := range ids {
				s.refreshLock(ctx, id)
			}
		}
	}
	return out, err
}
