package practice

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/edupilot/backend-go/internal/exam"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/store"
)

// MaxBulk là số id tối đa một lô (US-P9-04 AC3).
const MaxBulk = 50

// BulkIn là thân `POST …/questions/bulk`.
type BulkIn struct {
	Action     string      `json:"action" validate:"required"`
	IDs        []uuid.UUID `json:"ids" validate:"required"`
	Topic      *string     `json:"topic"`
	Difficulty *string     `json:"difficulty"`
}

// BulkResult là kết quả MỘT id: một id lỗi không huỷ cả lô.
type BulkResult struct {
	ID   uuid.UUID `json:"id"`
	OK   bool      `json:"ok"`
	Code string    `json:"code,omitempty"`
}

// Bulk duyệt / loại / gắn nhãn hàng loạt câu hỏi (≤ 50). Mỗi id một giao dịch riêng (dùng lại `exam.Service.Review`, nên có audit + outbox như duyệt tay).
// APPROVE câu GENERATED chưa có trích dẫn → CITATION_REQUIRED (FR-6).
func (s *Service) Bulk(ctx context.Context, actor, courseID uuid.UUID, in BulkIn) ([]BulkResult, error) {
	var errs []apierr.FieldError
	switch in.Action {
	case "APPROVE", "REJECT":
	case "SET_TAGS":
		if in.Topic == nil && in.Difficulty == nil {
			errs = append(errs, apierr.FieldError{Field: "topic", Code: "required", Message: "SET_TAGS cần chủ đề hoặc độ khó."})
		}
		if in.Topic != nil && (len([]rune(*in.Topic)) < 1 || len([]rune(*in.Topic)) > 80) {
			errs = append(errs, apierr.FieldError{Field: "topic", Code: "topic", Message: "Chủ đề từ 1 đến 80 ký tự."})
		}
		if in.Difficulty != nil && *in.Difficulty != "EASY" && *in.Difficulty != "MEDIUM" && *in.Difficulty != "HARD" {
			errs = append(errs, apierr.FieldError{Field: "difficulty", Code: "difficulty", Message: "Độ khó là EASY, MEDIUM hoặc HARD."})
		}
	default:
		errs = append(errs, apierr.FieldError{Field: "action", Code: "action", Message: "action là APPROVE, REJECT hoặc SET_TAGS."})
	}
	if n := len(in.IDs); n < 1 || n > MaxBulk {
		errs = append(errs, apierr.FieldError{Field: "ids", Code: "max", Message: fmt.Sprintf("Chọn từ 1 đến %d câu.", MaxBulk)})
	}
	if len(errs) > 0 {
		return nil, apierr.Validation(errs...)
	}
	q := store.New(s.Pool)
	if err := writable(ctx, q, courseID); err != nil {
		return nil, err
	}
	out := make([]BulkResult, 0, len(in.IDs))
	for _, id := range in.IDs {
		r := BulkResult{ID: id}
		if err := s.bulkOne(ctx, q, actor, courseID, id, in); err != nil {
			r.Code = codeOf(err)
		} else {
			r.OK = true
		}
		out = append(out, r)
	}
	return out, nil
}

func codeOf(err error) string {
	var ae *apierr.Error
	var vc *exam.VersionConflict
	switch {
	case errors.As(err, &ae):
		return ae.Code
	case errors.As(err, &vc):
		return apierr.VersionConflict
	}
	return apierr.Internal
}

func (s *Service) bulkOne(ctx context.Context, q *store.Queries, actor, courseID, id uuid.UUID, in BulkIn) error {
	b, err := q.PracticeQuestionBrief(ctx, store.PracticeQuestionBriefParams{CourseID: courseID, ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return notFound()
	}
	if err != nil {
		return fmt.Errorf("practice: đọc câu: %w", err)
	}
	if b.Archived {
		return apierr.New(http.StatusConflict, apierr.Conflict)
	}
	switch in.Action {
	case "APPROVE":
		if b.Origin == "GENERATED" && b.NCitations == 0 {
			return apierr.New(http.StatusUnprocessableEntity, apierr.CitationRequired)
		}
		_, err = s.Exam.Review(ctx, actor, courseID, id, exam.DecisionApprove, int(b.Version))
	case "REJECT":
		_, err = s.Exam.Review(ctx, actor, courseID, id, exam.DecisionReject, int(b.Version))
	default:
		var n int64
		var diff *store.QuestionDifficulty
		if in.Difficulty != nil {
			d := store.QuestionDifficulty(*in.Difficulty)
			diff = &d
		}
		n, err = q.PracticeQuestionSetTags(ctx, store.PracticeQuestionSetTagsParams{CourseID: courseID, ID: id, Topic: in.Topic, Difficulty: diff, ExpectedVersion: b.Version})
		if err == nil && n == 0 {
			return apierr.New(http.StatusConflict, apierr.Conflict)
		}
	}
	return err
}

// GenStats là `GET …/questions/gen-stats?job_id` (US-P9-03 AC7): số liệu thô E5 của một lô.
type GenStats struct {
	Unchanged int `json:"unchanged"`
	Edited    int `json:"edited"`
	Rejected  int `json:"rejected"`
	Pending   int `json:"pending"`
}

// GenStatsOf tính từ `version` lúc duyệt (audit `question.review`) và `review_status`; lô không có câu nào → 404.
func (s *Service) GenStatsOf(ctx context.Context, courseID, jobID uuid.UUID) (GenStats, error) {
	row, err := store.New(s.Pool).PracticeGenStats(ctx, store.PracticeGenStatsParams{CourseID: courseID, JobID: &jobID})
	if err != nil {
		return GenStats{}, fmt.Errorf("practice: số liệu lô: %w", err)
	}
	if row.Unchanged+row.Edited+row.Rejected+row.Pending == 0 {
		return GenStats{}, notFound()
	}
	return GenStats{Unchanged: int(row.Unchanged), Edited: int(row.Edited), Rejected: int(row.Rejected), Pending: int(row.Pending)}, nil
}
