package grading

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/jobs"
	"github.com/edupilot/backend-go/internal/store"
)

// Batches nạp việc chấm vào hàng (SRS 4.7.2): một giao dịch ghi `grading_results` QUEUED + `jobs` + outbox; XADD do handler `grading.batch` chạy SAU COMMIT.
type Batches struct {
	Pool *pgxpool.Pool
	Jobs *jobs.Service
}

// Enqueued là kết quả nạp: `JobID` nil khi không có gì để chấm (200 {queued:0}).
type Enqueued struct {
	JobID  *uuid.UUID `json:"job_id,omitempty"`
	Queued int        `json:"queued"`
}

// GradeAll xếp mọi bài MATCHED phiên bản mới nhất chưa có kết quả (hoặc, `retryFailed`, mọi kết quả FAILED hiện hành).
func (b *Batches) GradeAll(ctx context.Context, course, assignment, actor uuid.UUID, retryFailed bool) (Enqueued, error) {
	return b.enqueue(ctx, actor, course, func(q *store.Queries) ([]uuid.UUID, error) {
		if _, err := q.AssignmentGet(ctx, store.AssignmentGetParams{CourseID: course, ID: assignment}); errors.Is(err, pgx.ErrNoRows) {
			return nil, apierr.New(http.StatusNotFound, apierr.AssignmentNotFound)
		} else if err != nil {
			return nil, err
		}
		if retryFailed {
			ids, err := q.GradeFailedCandidates(ctx, store.GradeFailedCandidatesParams{CourseID: course, AssignmentID: assignment})
			if err != nil {
				return nil, err
			}
			for _, id := range ids {
				if _, err := q.GradingRetryFailed(ctx, store.GradingRetryFailedParams{CourseID: course, ID: id}); err != nil {
					return nil, err
				}
			}
			return ids, nil
		}
		subs, err := q.GradeAllCandidates(ctx, store.GradeAllCandidatesParams{CourseID: course, AssignmentID: assignment})
		if err != nil {
			return nil, err
		}
		var out []uuid.UUID
		for _, sid := range subs {
			id, err := q.GradingInsertQueued(ctx, store.GradingInsertQueuedParams{CourseID: course, SubmissionID: sid, Round: 1})
			if errors.Is(err, pgx.ErrNoRows) {
				continue // hai người cùng bấm: dòng đã có
			}
			if err != nil {
				return nil, err
			}
			out = append(out, id)
		}
		return out, nil
	})
}

// GradeOne chấm / chấm lại MỘT bài: round + 1 chỉ khi kết quả hiện có đang DRAFT (xong) hoặc FAILED; REVIEWED / PUBLISHED → 409 ALREADY_REVIEWED;
// đang xếp hàng / chạy hoặc hai người cùng bấm (UNIQUE round) → 409 VERSION_CONFLICT.
func (b *Batches) GradeOne(ctx context.Context, course, submission, actor uuid.UUID) (Enqueued, error) {
	return b.enqueue(ctx, actor, course, func(q *store.Queries) ([]uuid.UUID, error) {
		s, err := q.GradingSubmissionOK(ctx, store.GradingSubmissionOKParams{CourseID: course, ID: submission})
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && s.MatchStatus != "MATCHED") {
			return nil, apierr.New(http.StatusNotFound, apierr.NotFound)
		}
		if err != nil {
			return nil, err
		}
		round := int16(1)
		cur, err := q.GradingCurrent(ctx, store.GradingCurrentParams{CourseID: course, SubmissionID: submission})
		switch {
		case errors.Is(err, pgx.ErrNoRows):
		case err != nil:
			return nil, err
		case cur.Status != "DRAFT":
			return nil, apierr.New(http.StatusConflict, apierr.AlreadyReviewed)
		case cur.RunState != "DONE" && cur.RunState != "FAILED":
			return nil, apierr.New(http.StatusConflict, apierr.VersionConflict)
		default:
			round = cur.Round + 1
		}
		id, err := q.GradingInsertQueued(ctx, store.GradingInsertQueuedParams{CourseID: course, SubmissionID: submission, Round: round})
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierr.New(http.StatusConflict, apierr.VersionConflict)
		}
		if err != nil {
			return nil, err
		}
		return []uuid.UUID{id}, nil
	})
}

// Retry đặt một kết quả FAILED về QUEUED (attempts = 0, cùng round). Không phải FAILED → 409 VERSION_CONFLICT.
func (b *Batches) Retry(ctx context.Context, course, result, actor uuid.UUID) (Enqueued, error) {
	return b.enqueue(ctx, actor, course, func(q *store.Queries) ([]uuid.UUID, error) {
		if _, err := q.GradingResultState(ctx, store.GradingResultStateParams{CourseID: course, ID: result}); errors.Is(err, pgx.ErrNoRows) {
			return nil, apierr.New(http.StatusNotFound, apierr.NotFound)
		} else if err != nil {
			return nil, err
		}
		if _, err := q.GradingRetryFailed(ctx, store.GradingRetryFailedParams{CourseID: course, ID: result}); errors.Is(err, pgx.ErrNoRows) {
			return nil, apierr.New(http.StatusConflict, apierr.VersionConflict)
		} else if err != nil {
			return nil, err
		}
		return []uuid.UUID{result}, nil
	})
}

func (b *Batches) enqueue(ctx context.Context, actor, course uuid.UUID, pick func(*store.Queries) ([]uuid.UUID, error)) (Enqueued, error) {
	tx, err := b.Pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Enqueued{}, fmt.Errorf("grading: bắt đầu giao dịch: %w", err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	ids, err := pick(store.New(tx))
	if err != nil {
		return Enqueued{}, err
	}
	if len(ids) == 0 {
		return Enqueued{}, nil
	}
	j, err := b.Jobs.EnqueueTx(ctx, tx, actor, KindBatch, BatchPayload{CourseID: course, ResultIDs: ids})
	if err != nil {
		return Enqueued{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Enqueued{}, fmt.Errorf("grading: commit: %w", err)
	}
	return Enqueued{JobID: &j.ID, Queued: len(ids)}, nil
}
