package exam

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/store"
)

const (
	reasonMax = 500
)

// AdjustIn là thân `PUT …/results/{aid}/score`. `score` null = gỡ điều chỉnh (cần lý do); số thập phân dạng chuỗi.
type AdjustIn struct {
	Score   *string `json:"score"`
	Reason  string  `json:"reason"`
	Version int     `json:"version"`
}

// AdjustOut là phản hồi của sửa điểm tay.
type AdjustOut struct {
	AttemptID uuid.UUID `json:"attempt_id"`
	AutoScore *string   `json:"auto_score"`
	Score     *string   `json:"score"`
	Adjusted  bool      `json:"adjusted"`
	Version   int       `json:"version"`
}

func adjustOutOf(a store.ExamAttempt) AdjustOut {
	o := AdjustOut{AttemptID: a.ID, Adjusted: a.AdjustedScore.Valid, Version: int(a.Version)}
	if a.AutoScore.Valid {
		o.AutoScore = new(a.AutoScore.Decimal.StringFixed(2))
	}
	if sc, ok := officialScore(a); ok {
		o.Score = new(sc.StringFixed(2))
	}
	return o
}

func checkReason(reason string) (string, error) {
	n := utf8.RuneCountInString(reason)
	if n < 1 || n > reasonMax {
		return "", fieldErr("reason", "REASON_REQUIRED", "Cần lý do (1–500 ký tự).")
	}
	return reason, nil
}

// adjustTx áp / gỡ điểm điều chỉnh trên lượt GRADED đã khoá (dùng chung cho `PUT …/score` và trả lời phúc khảo `ADJUSTED`). `score` nil = gỡ. Ghi `audit_log` (không nội dung)
// và, nếu `notify` và bài đã PUBLISHED, outbox `exam.regraded` (khử trùng theo `attempt_id` + version mới) để báo sinh viên.
func (s *Service) adjustTx(ctx context.Context, q *store.Queries, tx pgx.Tx, actor uuid.UUID, e store.Exam, a store.ExamAttempt, score *decimal.Decimal, reason string, notify bool) (store.ExamAttempt, error) {
	arg := store.AttemptAdjustParams{CourseID: a.CourseID, ID: a.ID, Version: a.Version}
	if score != nil {
		arg.Score = decimal.NullDecimal{Decimal: *score, Valid: true}
		arg.Reason, arg.By, arg.At = &reason, &actor, new(s.now())
	}
	row, err := q.AttemptAdjust(ctx, arg)
	if errors.Is(err, pgx.ErrNoRows) {
		return a, &VersionConflict{Version: int(a.Version), Current: adjustOutOf(a)}
	}
	if err != nil {
		return a, fmt.Errorf("exam: sửa điểm tay: %w", err)
	}
	before, after := map[string]any{"version": int(a.Version)}, map[string]any{"version": int(row.Version)}
	if a.AdjustedScore.Valid {
		before["adjusted_score"] = a.AdjustedScore.Decimal.StringFixed(2)
	}
	if score != nil {
		after["adjusted_score"] = score.StringFixed(2)
	}
	if err := audit(ctx, q, a.CourseID, actor, "exam_attempt", a.ID, "exam.score_adjust", before, after); err != nil {
		return a, err
	}
	if notify && e.Status == store.ExamStatusPUBLISHED {
		if _, err := outbox.Write(ctx, tx, TopicRegraded, map[string]any{"exam_id": a.ExamID, "course_id": a.CourseID, "attempt_id": a.ID, "user_id": a.StudentID, "version": int(row.Version)}); err != nil {
			return a, fmt.Errorf("exam: outbox: %w", err)
		}
	}
	return row, nil
}

// AdjustScore: `PUT …/results/{aid}/score` (chỉ Giảng viên). Bài CLOSED / PUBLISHED, lượt GRADED. `auto_score` giữ nguyên. Sai version → 409.
func (s *Service) AdjustScore(ctx context.Context, actor, courseID, examID, attemptID uuid.UUID, in AdjustIn) (AdjustOut, error) {
	var out AdjustOut
	reason, err := checkReason(in.Reason)
	if err != nil {
		return out, err
	}
	err = s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		e, err := s.lockExam(ctx, q, courseID, examID)
		if err != nil {
			return err
		}
		if !closed(e) {
			return lockedErr("status")
		}
		a, err := q.AttemptLockByID(ctx, store.AttemptLockByIDParams{CourseID: courseID, ID: attemptID})
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && a.ExamID != examID) {
			return notFound()
		}
		if err != nil {
			return fmt.Errorf("exam: khoá lượt: %w", err)
		}
		if a.Status != store.AttemptStatusGRADED {
			return conflict("Lượt này chưa chấm xong.")
		}
		if int(a.Version) != in.Version {
			return &VersionConflict{Version: int(a.Version), Current: adjustOutOf(a)}
		}
		var score *decimal.Decimal
		if in.Score != nil {
			d, err := decimal.NewFromString(*in.Score)
			if err != nil {
				return fieldErr("score", "INVALID_NUMBER", "Điểm không hợp lệ.")
			}
			if d.Sign() < 0 || d.GreaterThan(e.MaxScore) {
				return fieldErr("score", "OUT_OF_RANGE", "Điểm phải từ 0 đến điểm tối đa.")
			}
			if !RoundToStep(d, e.RoundingStep, e.MaxScore).Equal(d) {
				return fieldErr("score", "INVALID_STEP", "Điểm phải là bội của bước làm tròn.")
			}
			score = &d
		}
		row, err := s.adjustTx(ctx, q, tx, actor, e, a, score, reason, true)
		if err != nil {
			return err
		}
		out = adjustOutOf(row)
		return nil
	})
	return out, err
}

// OverrideIn là thân `PUT …/items/{itemId}/override`: đúng MỘT trong `answer_key` (đổi đáp án đúng) / `void: true` (huỷ câu, trọn điểm cho mọi lượt); `void: false` không kèm đáp án = gỡ override.
type OverrideIn struct {
	AnswerKey json.RawMessage `json:"answer_key"`
	Void      *bool           `json:"void"`
	Reason    string          `json:"reason"`
}

// OverrideOut: `recomputed` = số lượt đã tính lại (200); `job_id` khi nhiều hơn 200 lượt (202).
type OverrideOut struct {
	Recomputed int        `json:"recomputed"`
	JobID      *uuid.UUID `json:"job_id"`
}

// OverrideItem: đổi đáp án đúng / huỷ một câu TRẮC NGHIỆM sau khi bài đóng (chỉ Giảng viên), rồi tính lại mọi lượt GRADED.
func (s *Service) OverrideItem(ctx context.Context, actor, courseID, examID, itemID uuid.UUID, in OverrideIn) (OverrideOut, error) {
	var out OverrideOut
	if _, err := checkReason(in.Reason); err != nil {
		return out, err
	}
	clear := in.Void != nil && !*in.Void && len(in.AnswerKey) == 0
	if !clear && (len(in.AnswerKey) == 0) == (in.Void == nil || !*in.Void) {
		return out, fieldErr("answer_key", "ONE_OF", "Gửi đúng một trong đáp án đúng mới hoặc huỷ câu.")
	}
	var total int
	err := s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		e, err := s.lockExam(ctx, q, courseID, examID)
		if err != nil {
			return err
		}
		if !closed(e) {
			return lockedErr("status")
		}
		metas, err := q.ExamItemsMeta(ctx, store.ExamItemsMetaParams{CourseID: courseID, ExamID: examID})
		if err != nil {
			return fmt.Errorf("exam: câu của bài: %w", err)
		}
		i := slices.IndexFunc(metas, func(m store.ExamItemsMetaRow) bool { return m.ItemID == itemID })
		if i < 0 {
			return notFound()
		}
		m := metas[i]
		if m.Type == store.QuestionTypeCODE || m.Type == store.QuestionTypeSHORT {
			return fieldErr("item", "NOT_CHOICE", "Chỉ câu trắc nghiệm mới đổi được đáp án.")
		}
		var next json.RawMessage
		if !clear {
			ov := map[string]any{"reason": in.Reason, "by": actor, "at": s.now()}
			if len(in.AnswerKey) > 0 {
				if err := s.checkOverrideKey(ctx, q, e, m, in.AnswerKey); err != nil {
					return err
				}
				ov["answer_key"] = in.AnswerKey
			} else {
				ov["void"] = true
			}
			if next, err = json.Marshal(ov); err != nil {
				return fmt.Errorf("exam: mã hoá override: %w", err)
			}
		}
		if _, err := q.ItemSetOverride(ctx, store.ItemSetOverrideParams{CourseID: courseID, ExamID: examID, ID: itemID, Override: next}); err != nil {
			return fmt.Errorf("exam: ghi override: %w", err)
		}
		if err := audit(ctx, q, courseID, actor, "exam_item", itemID, "exam.item_override", map[string]any{"had": len(m.Override) > 0}, map[string]any{"void": in.Void != nil && *in.Void, "key": len(in.AnswerKey) > 0, "clear": clear}); err != nil {
			return err
		}
		ids, err := q.AttemptsGradedOfExam(ctx, store.AttemptsGradedOfExamParams{CourseID: courseID, ExamID: examID})
		if err != nil {
			return fmt.Errorf("exam: lượt đã chấm: %w", err)
		}
		total = len(ids)
		if total > s.Results.recomputeSync() {
			if err := q.ExamSetRegrading(ctx, store.ExamSetRegradingParams{CourseID: courseID, ID: examID, Regrading: true}); err != nil {
				return fmt.Errorf("exam: đặt chấm lại: %w", err)
			}
			j, err := s.Jobs.Enqueue(ctx, actor, KindRegrade, regradePayload{CourseID: courseID, ExamID: examID, Scope: scopeRecompute})
			if err != nil {
				return fmt.Errorf("exam: xếp việc tính lại: %w", err)
			}
			out.JobID = &j.ID
		}
		return nil
	})
	if err != nil || out.JobID != nil {
		return out, err
	}
	n, err := s.RecomputeExam(ctx, courseID, examID)
	out.Recomputed = n
	return out, err
}

// checkOverrideKey: đáp án mới phải thuộc các lựa chọn của câu (trắc nghiệm) hoặc là `{value: bool}` (đúng / sai); đúng một lựa chọn với câu chọn một.
func (s *Service) checkOverrideKey(ctx context.Context, q *store.Queries, e store.Exam, m store.ExamItemsMetaRow, key json.RawMessage) error {
	var k struct {
		OptionIDs []uuid.UUID `json:"option_ids"`
		Value     *bool       `json:"value"`
	}
	if err := json.Unmarshal(key, &k); err != nil {
		return fieldErr("answer_key", "INVALID", "Đáp án đúng không hợp lệ.")
	}
	if m.Type == store.QuestionTypeTRUEFALSE {
		if k.Value == nil || len(k.OptionIDs) > 0 {
			return fieldErr("answer_key", "INVALID", "Câu đúng / sai cần giá trị true hoặc false.")
		}
		return nil
	}
	if len(k.OptionIDs) == 0 || k.Value != nil || (m.Type == store.QuestionTypeMCQSINGLE && len(k.OptionIDs) != 1) {
		return fieldErr("answer_key", "INVALID", "Chọn đúng số lựa chọn cho loại câu này.")
	}
	opts, err := q.ExamPreviewOptions(ctx, store.ExamPreviewOptionsParams{CourseID: e.CourseID, ExamID: e.ID})
	if err != nil {
		return fmt.Errorf("exam: lựa chọn: %w", err)
	}
	valid := map[uuid.UUID]bool{}
	for _, o := range opts {
		if o.QuestionID == m.QuestionID {
			valid[o.ID] = true
		}
	}
	for _, id := range k.OptionIDs {
		if !valid[id] {
			return fieldErr("answer_key", "UNKNOWN_OPTION", "Lựa chọn không thuộc câu này.")
		}
	}
	return nil
}
