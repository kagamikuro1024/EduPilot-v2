package exam

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/quiz"
	"github.com/edupilot/backend-go/internal/store"
)

// Topic outbox của chấm / công bố / kết quả (US-PE-08). Chuỗi trùng hằng ở gói `today` (khai lại để gói đó không import `exam`).
const (
	TopicAttemptGraded  = "exam.attempt_graded"
	TopicPublished      = "exam.published"
	TopicHold           = "exam.hold"
	TopicRegraded       = "exam.regraded"
	TopicAppealCreated  = "exam.appeal_created"
	TopicAppealAnswered = "exam.appeal_answered"
)

// breakdownSample / breakdownCode: ảnh chụp kết quả bài code lúc tính điểm (SRS 5.8) — trang kết quả đọc từ đây, không từ bản nộp (nên không trống khi đang chấm lại).
type breakdownSample struct {
	TestID   uuid.UUID `json:"test_id"`
	Verdict  string    `json:"verdict"`
	TimeMS   int       `json:"time_ms"`
	MemoryKB int       `json:"memory_kb"`
}

type breakdownCode struct {
	CompileOK    bool              `json:"compile_ok"`
	Hidden       hiddenCount       `json:"hidden"`
	Samples      []breakdownSample `json:"samples"`
	SubmissionID uuid.UUID         `json:"submission_id"`
}

type hiddenCount struct {
	Passed int `json:"passed"`
	Total  int `json:"total"`
}

// breakdownRow là một dòng `exam_attempts.breakdown`. `earned` lưu ĐỦ chữ số (SRS 4.8.3); `points` = `max` (giữ khoá cũ của PE-05).
type breakdownRow struct {
	ItemID uuid.UUID      `json:"item_id"`
	Points string         `json:"points"`
	Max    string         `json:"max"`
	Earned string         `json:"earned"`
	Void   bool           `json:"void,omitempty"`
	Code   *breakdownCode `json:"code,omitempty"`
}

// gradeResult là kết quả `computeGrade`: đủ khi mọi câu code có bản nộp tính điểm đã `DONE` (hoặc không có bản nộp nào).
type gradeResult struct {
	Score     decimal.Decimal
	Breakdown json.RawMessage
	Complete  bool // false ⇒ còn bản nộp QUEUED / RUNNING / ERROR
	HasError  bool // có bản nộp cuối ở ERROR / IE: lượt ở lại GRADING (không điểm 0)
}

type itemOverride struct {
	AnswerKey json.RawMessage `json:"answer_key"`
	Void      bool            `json:"void"`
}

func parseOverride(raw []byte) itemOverride {
	var ov itemOverride
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &ov)
	}
	return ov
}

// computeGrade tính điểm một lượt bằng code thuần: trắc nghiệm theo Quiz Engine (`multi_scoring` của bài, `override` của mục), câu code `points × Σweight(AC) ÷ Σweight(test duyệt)`
// từ bản SUBMIT cuối; câu code không có bản nộp = 0 (không phải lỗi). Điểm cuối làm tròn MỘT lần từ tổng chưa làm tròn (`Score`). Không float, không LLM.
func (s *Service) computeGrade(ctx context.Context, q *store.Queries, e store.Exam, a store.ExamAttempt, items []store.ExamGradeItemsRow) (gradeResult, error) {
	saved, err := q.AnswersList(ctx, store.AnswersListParams{CourseID: a.CourseID, AttemptID: a.ID})
	if err != nil {
		return gradeResult{}, fmt.Errorf("exam: câu trả lời đã lưu: %w", err)
	}
	byItem := make(map[uuid.UUID]json.RawMessage, len(saved))
	for _, r := range saved {
		byItem[r.ItemID] = r.Answer
	}
	opts, err := q.ExamPreviewOptions(ctx, store.ExamPreviewOptionsParams{CourseID: a.CourseID, ExamID: a.ExamID})
	if err != nil {
		return gradeResult{}, fmt.Errorf("exam: đáp án của các câu: %w", err)
	}
	optIDs := map[uuid.UUID][]string{}
	for _, o := range opts {
		optIDs[o.QuestionID] = append(optIDs[o.QuestionID], o.ID.String())
	}
	finals, err := q.AttemptFinalSubmissions(ctx, store.AttemptFinalSubmissionsParams{CourseID: a.CourseID, AttemptID: a.ID})
	if err != nil {
		return gradeResult{}, fmt.Errorf("exam: bản nộp cuối: %w", err)
	}
	finalOf := make(map[uuid.UUID]store.AttemptFinalSubmissionsRow, len(finals))
	for _, f := range finals {
		finalOf[f.ItemID] = f
	}
	mode := quiz.Mode(e.MultiScoring)
	res := gradeResult{Complete: true}
	scored := make([]ScoreItem, len(items))
	rows := make([]breakdownRow, len(items))
	for i, it := range items {
		ov := parseOverride(it.Override)
		row := breakdownRow{ItemID: it.ItemID, Points: it.Points.StringFixed(2), Max: it.Points.StringFixed(2), Void: ov.Void}
		earned := decimal.Zero
		if it.Type == store.QuestionTypeCODE {
			if f, ok := finalOf[it.ItemID]; ok {
				switch f.Status {
				case store.SubmissionStatusDONE:
					pw, tw := int64(0), int64(0)
					if f.PassedWeight != nil {
						pw = int64(*f.PassedWeight)
					}
					if f.TotalWeight != nil {
						tw = int64(*f.TotalWeight)
					}
					earned = CodeEarned(it.Points, pw, tw)
					row.Code = codeSnapshot(f)
				case store.SubmissionStatusERROR:
					res.Complete, res.HasError = false, true
				default:
					res.Complete = false
				}
			}
		} else {
			key := it.AnswerKey
			if len(ov.AnswerKey) > 0 {
				key = ov.AnswerKey
			}
			earned, err = gradeOne(quiz.Type(it.Type), it.Points, key, byItem[it.ItemID], mode, optIDs[it.QuestionID])
			if err != nil {
				return gradeResult{}, fmt.Errorf("exam: chấm câu %s: %w", it.ItemID, err)
			}
		}
		scored[i] = ScoreItem{Points: it.Points, Earned: earned, Void: ov.Void}
		row.Earned = earned.String()
		rows[i] = row
	}
	res.Score, err = Score(scored, e.MaxScore, e.RoundingStep)
	if err != nil {
		return gradeResult{}, fmt.Errorf("exam: tính điểm: %w", err)
	}
	res.Breakdown, err = json.Marshal(rows)
	if err != nil {
		return gradeResult{}, fmt.Errorf("exam: mã hoá breakdown: %w", err)
	}
	return res, nil
}

// codeSnapshot dựng ảnh chụp kết quả bài code từ bản nộp cuối: verdict từng test MẪU + SỐ test ẩn đạt / tổng (không tên, không verdict từng test ẩn).
func codeSnapshot(f store.AttemptFinalSubmissionsRow) *breakdownCode {
	c := &breakdownCode{CompileOK: f.CompileOk != nil && *f.CompileOk, Samples: []breakdownSample{}, SubmissionID: f.ID}
	var rs []storedResult
	_ = json.Unmarshal(f.Results, &rs)
	for _, r := range rs {
		if r.IsSample {
			c.Samples = append(c.Samples, breakdownSample{TestID: r.TestID, Verdict: r.Verdict, TimeMS: r.TimeMS, MemoryKB: r.MemoryKB})
			continue
		}
		c.Hidden.Total++
		if r.Verdict == "AC" {
			c.Hidden.Passed++
		}
	}
	return c
}

// TryFinishGrading (idempotent, SRS 4.8.1): lượt GRADING mà mọi câu code đã có bản nộp tính điểm `DONE` (hoặc không có) → tính điểm, `GRADED`, phát `exam.attempt_graded`.
// Còn bản nộp QUEUED / RUNNING / ERROR → giữ GRADING (không điểm 0). Trả true khi lượt vừa được chuyển GRADED.
func (s *Service) TryFinishGrading(ctx context.Context, courseID, attemptID uuid.UUID) (bool, error) {
	done := false
	err := s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		a, err := q.AttemptLockByID(ctx, store.AttemptLockByIDParams{CourseID: courseID, ID: attemptID})
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && a.Status != store.AttemptStatusGRADING) {
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
		row, err := q.AttemptSetGraded(ctx, store.AttemptSetGradedParams{CourseID: courseID, ID: attemptID, AutoScore: decimal.NullDecimal{Decimal: gr.Score, Valid: true}, Breakdown: gr.Breakdown, At: new(s.now())})
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("exam: chuyển GRADED: %w", err)
		}
		done = true
		_, err = outbox.Write(ctx, tx, TopicAttemptGraded, map[string]any{"exam_id": row.ExamID, "course_id": courseID, "attempt_id": row.ID, "user_id": row.StudentID})
		return err
	})
	return done, err
}

// GradeDue (tick): hoàn tất chấm các lượt GRADING đã đủ điều kiện (lô 200) — lưới an toàn khi tin `exam.submission_done` bị trễ / mất.
func (s *Service) GradeDue(ctx context.Context) (int, error) {
	rows, err := store.New(s.Pool).AttemptsToFinishGrading(ctx)
	if err != nil {
		return 0, fmt.Errorf("exam: lượt chờ chấm: %w", err)
	}
	n := 0
	var firstErr error
	for _, r := range rows {
		ok, err := s.TryFinishGrading(ctx, r.CourseID, r.ID)
		if err != nil && firstErr == nil {
			firstErr = err
		}
		if ok {
			n++
		}
	}
	return n, firstErr
}

// PublishDue (tick): công bố các bài CLOSED đủ điều kiện (SRS 4.8.2) — `UPDATE … WHERE status = 'CLOSED'` nên hai bộ lập lịch chạy song song chỉ một bên thắng; cùng giao dịch ghi outbox `exam.published`.
func (s *Service) PublishDue(ctx context.Context) (int, error) {
	return s.publish(ctx, uuid.Nil)
}

func (s *Service) publish(ctx context.Context, only uuid.UUID) (int, error) {
	n := 0
	err := s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		n = 0
		arg := store.ExamPublishDueParams{Now: new(s.now())}
		if only != uuid.Nil {
			arg.ExamID = &only
		}
		rows, err := q.ExamPublishDue(ctx, arg)
		if err != nil {
			return fmt.Errorf("exam: công bố: %w", err)
		}
		for _, r := range rows {
			if _, err := outbox.Write(ctx, tx, TopicPublished, map[string]any{"exam_id": r.ID, "course_id": r.CourseID}); err != nil {
				return fmt.Errorf("exam: outbox: %w", err)
			}
			n++
		}
		return nil
	})
	return n, err
}

// SetHold: `PUT …/publish-hold {hold,version}` (chỉ Giảng viên). hold=true ở SCHEDULED / OPEN / CLOSED; sau PUBLISHED → 409 EXAM_LOCKED. hold=false khi đã chấm xong → công bố NGAY.
func (s *Service) SetHold(ctx context.Context, actor, courseID, examID uuid.UUID, hold bool, version int) (ExamDetail, error) {
	var out ExamDetail
	err := s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		e, err := s.lockExam(ctx, q, courseID, examID)
		if err != nil {
			return err
		}
		if e.Status == store.ExamStatusPUBLISHED || e.Status == store.ExamStatusDRAFT {
			return lockedErr("status")
		}
		row, err := q.ExamSetHold(ctx, store.ExamSetHoldParams{CourseID: courseID, ID: examID, Hold: hold, Version: int32(version)}) //nolint:gosec // version nhỏ
		if errors.Is(err, pgx.ErrNoRows) {
			return s.conflictOf(ctx, q, e)
		}
		if err != nil {
			return fmt.Errorf("exam: hoãn công bố: %w", err)
		}
		if err := audit(ctx, q, courseID, actor, "exam", examID, "exam.hold", map[string]any{"hold": e.PublishHold}, map[string]any{"hold": hold, "version": int(row.Version)}); err != nil {
			return err
		}
		if _, err := outbox.Write(ctx, tx, TopicHold, map[string]any{"exam_id": examID, "course_id": courseID}); err != nil {
			return fmt.Errorf("exam: outbox: %w", err)
		}
		if !hold {
			pub, err := q.ExamPublishDue(ctx, store.ExamPublishDueParams{Now: new(s.now()), ExamID: &examID})
			if err != nil {
				return fmt.Errorf("exam: công bố ngay: %w", err)
			}
			for _, r := range pub {
				if _, err := outbox.Write(ctx, tx, TopicPublished, map[string]any{"exam_id": r.ID, "course_id": r.CourseID}); err != nil {
					return fmt.Errorf("exam: outbox: %w", err)
				}
			}
		}
		cur, err := q.ExamGet(ctx, store.ExamGetParams{CourseID: courseID, ID: examID})
		if err != nil {
			return fmt.Errorf("exam: đọc bài thi: %w", err)
		}
		out, err = s.examDetail(ctx, q, cur)
		return err
	})
	return out, err
}
