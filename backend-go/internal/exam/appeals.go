package exam

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/store"
)

const appealReasonMax = 1000

// AppealView là một yêu cầu xem lại. `student` chỉ có trong danh sách của Staff.
type AppealView struct {
	ID          uuid.UUID      `json:"id"`
	AttemptID   uuid.UUID      `json:"attempt_id"`
	Status      string         `json:"status"`
	Reason      string         `json:"reason"`
	Response    *string        `json:"response"`
	CreatedAt   time.Time      `json:"created_at"`
	RespondedAt *time.Time     `json:"responded_at"`
	ScoreBefore *string        `json:"score_before"`
	ScoreAfter  *string        `json:"score_after"`
	Version     int            `json:"version"`
	Student     *ResultStudent `json:"student,omitempty"`
}

func nullDec(d decimal.NullDecimal) *string {
	if !d.Valid {
		return nil
	}
	return new(d.Decimal.StringFixed(2))
}

func appealView(p store.ExamAppeal) AppealView {
	return AppealView{ID: p.ID, AttemptID: p.AttemptID, Status: string(p.Status), Reason: p.Reason, Response: p.Response, CreatedAt: p.CreatedAt, RespondedAt: p.RespondedAt,
		ScoreBefore: nullDec(p.ScoreBefore), ScoreAfter: nullDec(p.ScoreAfter), Version: int(p.Version)}
}

// CreateAppeal: `POST …/attempts/{aid}/appeal` (sinh viên, `Idempotency-Key`). Bài PUBLISHED, lượt GRADED của người gọi, còn hạn `appeal_days`, chưa có yêu cầu của lượt.
// Không route nào của phúc khảo gọi LLM.
func (s *Service) CreateAppeal(ctx context.Context, userID, courseID, examID, attemptID uuid.UUID, reason string) (AppealView, error) {
	var out AppealView
	if n := utf8.RuneCountInString(reason); n < 1 || n > appealReasonMax {
		return out, fieldErr("reason", "REASON_REQUIRED", "Cần lý do (1–1.000 ký tự).")
	}
	err := s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		e, err := s.lockExam(ctx, q, courseID, examID)
		if err != nil {
			return err
		}
		a, err := q.AttemptOwn(ctx, store.AttemptOwnParams{CourseID: courseID, ExamID: examID, ID: attemptID, StudentID: userID})
		if errors.Is(err, pgx.ErrNoRows) {
			return notFound()
		}
		if err != nil {
			return fmt.Errorf("exam: đọc lượt làm: %w", err)
		}
		if e.Status != store.ExamStatusPUBLISHED || a.Status != store.AttemptStatusGRADED {
			return apierr.New(http.StatusConflict, apierr.ResultNotPublished)
		}
		until := appealOpenUntil(e)
		if until == nil || s.now().After(*until) {
			return apierr.New(http.StatusConflict, apierr.AppealWindowClosed)
		}
		row, err := q.AppealInsert(ctx, store.AppealInsertParams{CourseID: courseID, ExamID: examID, AttemptID: attemptID, StudentID: userID, Reason: reason})
		if errors.Is(err, pgx.ErrNoRows) {
			return apierr.New(http.StatusConflict, apierr.AppealExists)
		}
		if err != nil {
			return fmt.Errorf("exam: ghi phúc khảo: %w", err)
		}
		if _, err := outbox.Write(ctx, tx, TopicAppealCreated, map[string]any{"exam_id": examID, "course_id": courseID, "appeal_id": row.ID, "attempt_id": attemptID, "user_id": userID}); err != nil {
			return fmt.Errorf("exam: outbox: %w", err)
		}
		out = appealView(row)
		return nil
	})
	return out, err
}

// AppealsPage là phản hồi `GET …/appeals`.
type AppealsPage struct {
	Items []AppealView `json:"items"`
}

// ListAppeals: `GET …/appeals?status&cursor` (Staff). Con trỏ chuẩn (created_at, id) giảm dần.
func (s *Service) ListAppeals(ctx context.Context, courseID, examID uuid.UUID, status string, cur *Cursor, fetch int) ([]AppealView, error) {
	q := store.New(s.Pool)
	if _, err := s.examOf(ctx, q, courseID, examID); err != nil {
		return nil, err
	}
	arg := store.AppealListParams{CourseID: courseID, ExamID: examID, Status: status, RowLimit: int32(fetch)} //nolint:gosec // ≤ 101
	if cur != nil {
		arg.CursorAt, arg.CursorID = &cur.At, &cur.ID
	}
	rows, err := q.AppealList(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("exam: danh sách phúc khảo: %w", err)
	}
	out := make([]AppealView, len(rows))
	for i, r := range rows {
		out[i] = AppealView{ID: r.ID, AttemptID: r.AttemptID, Status: string(r.Status), Reason: r.Reason, Response: r.Response, CreatedAt: r.CreatedAt, RespondedAt: r.RespondedAt,
			ScoreBefore: nullDec(r.ScoreBefore), ScoreAfter: nullDec(r.ScoreAfter), Version: int(r.Version), Student: &ResultStudent{FullName: r.FullName, StudentCode: r.StudentCode}}
	}
	return out, nil
}

// AppealAnswerIn là thân `POST …/appeals/{id}/answer`.
type AppealAnswerIn struct {
	Decision string  `json:"decision"`
	Response string  `json:"response"`
	Score    *string `json:"score"`
	Version  int     `json:"version"`
}

// AnswerAppeal: Giảng viên trả lời MỘT lần. `UPHELD` (giữ điểm) | `ADJUSTED` (+ `score`, áp như `PUT score` trong CÙNG transaction). Outbox `exam.appeal_answered` → thông báo + việc cho sinh viên.
func (s *Service) AnswerAppeal(ctx context.Context, actor, courseID, examID, appealID uuid.UUID, in AppealAnswerIn) (AppealView, error) {
	var out AppealView
	if n := utf8.RuneCountInString(in.Response); n < 1 || n > appealReasonMax {
		return out, fieldErr("response", "REASON_REQUIRED", "Cần phản hồi (1–1.000 ký tự).")
	}
	status := store.AppealStatus(in.Decision)
	switch status {
	case store.AppealStatusUPHELD:
		if in.Score != nil {
			return out, fieldErr("score", "NOT_ALLOWED", "Giữ nguyên điểm thì không gửi điểm mới.")
		}
	case store.AppealStatusADJUSTED:
		if in.Score == nil {
			return out, fieldErr("score", "VALUE_REQUIRED", "Cần điểm mới.")
		}
	default:
		return out, fieldErr("decision", "INVALID", "Quyết định phải là UPHELD hoặc ADJUSTED.")
	}
	err := s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		e, err := s.lockExam(ctx, q, courseID, examID)
		if err != nil {
			return err
		}
		p, err := q.AppealLock(ctx, store.AppealLockParams{CourseID: courseID, ExamID: examID, ID: appealID})
		if errors.Is(err, pgx.ErrNoRows) {
			return notFound()
		}
		if err != nil {
			return fmt.Errorf("exam: khoá phúc khảo: %w", err)
		}
		if p.Status != store.AppealStatusOPEN {
			return conflict("Yêu cầu này đã được trả lời.")
		}
		if int(p.Version) != in.Version {
			return &VersionConflict{Version: int(p.Version), Current: appealView(p)}
		}
		a, err := q.AttemptLockByID(ctx, store.AttemptLockByIDParams{CourseID: courseID, ID: p.AttemptID})
		if err != nil {
			return fmt.Errorf("exam: khoá lượt: %w", err)
		}
		before, _ := officialScore(a)
		arg := store.AppealAnswerParams{CourseID: courseID, ID: appealID, Version: p.Version, Status: status, Response: &in.Response, By: &actor, At: new(s.now()),
			ScoreBefore: decimal.NullDecimal{Decimal: before, Valid: a.AutoScore.Valid || a.AdjustedScore.Valid}}
		if status == store.AppealStatusADJUSTED {
			d, err := decimal.NewFromString(*in.Score)
			if err != nil {
				return fieldErr("score", "INVALID_NUMBER", "Điểm không hợp lệ.")
			}
			if d.Sign() < 0 || d.GreaterThan(e.MaxScore) || !RoundToStep(d, e.RoundingStep, e.MaxScore).Equal(d) {
				return fieldErr("score", "OUT_OF_RANGE", "Điểm phải từ 0 đến điểm tối đa và đúng bước làm tròn.")
			}
			if _, err := s.adjustTx(ctx, q, tx, actor, e, a, &d, in.Response, false); err != nil {
				return err
			}
			arg.ScoreAfter = decimal.NullDecimal{Decimal: d, Valid: true}
		}
		if err := audit(ctx, q, courseID, actor, "exam_appeal", appealID, "appeal.answer", map[string]any{"status": string(p.Status)}, map[string]any{"status": string(status), "version": int(p.Version) + 1}); err != nil {
			return err
		}
		row, err := q.AppealAnswer(ctx, arg)
		if errors.Is(err, pgx.ErrNoRows) {
			return conflict("Yêu cầu này đã được trả lời.")
		}
		if err != nil {
			return fmt.Errorf("exam: ghi phản hồi: %w", err)
		}
		if _, err := outbox.Write(ctx, tx, TopicAppealAnswered, map[string]any{"exam_id": examID, "course_id": courseID, "appeal_id": appealID, "attempt_id": p.AttemptID, "user_id": p.StudentID}); err != nil {
			return fmt.Errorf("exam: outbox: %w", err)
		}
		out = appealView(row)
		return nil
	})
	return out, err
}
