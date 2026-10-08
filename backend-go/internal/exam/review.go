package exam

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/store"
)

// Quyết định duyệt (SRS 4.1.4).
const (
	DecisionRequest = "REQUEST"
	DecisionApprove = "APPROVE"
	DecisionReject  = "REJECT"
)

// approvalErrors là điều kiện DUYỆT của câu CODE: ≥ 1 test mẫu, ≥ 1 test ẩn (đã duyệt), Σweight > 0, lời giải mẫu đã kiểm đúng phiên bản test hiện hành.
func (s *Service) approvalErrors(ctx context.Context, q *store.Queries, row store.QuestionBank) ([]apierr.FieldError, error) {
	if row.Type != store.QuestionTypeCODE {
		return nil, nil
	}
	cd, err := s.codeDetail(ctx, q, row.CourseID, row.ID)
	if err != nil {
		return nil, err
	}
	var errs []apierr.FieldError
	if cd.Tests.Samples < 1 || cd.Tests.Hidden < 1 {
		errs = append(errs, apierr.FieldError{Field: "testcases", Code: "CODE_TESTS_MISSING", Message: "Bài lập trình cần ít nhất một test mẫu và một test ẩn đã duyệt."})
	}
	if cd.Tests.TotalWeight <= 0 {
		errs = append(errs, apierr.FieldError{Field: "testcases", Code: "TOTAL_WEIGHT_ZERO", Message: "Tổng trọng số của các test phải lớn hơn 0."})
	}
	if cd.ReferenceVerifiedVersion == nil || *cd.ReferenceVerifiedVersion != cd.TestsVersion {
		errs = append(errs, apierr.FieldError{Field: "reference", Code: "REFERENCE_NOT_VERIFIED", Message: "Chạy lời giải mẫu qua bộ test hiện tại trước khi duyệt."})
	}
	return errs, nil
}

// Review đổi trạng thái duyệt: REQUEST (DRAFT → PENDING), APPROVE / REJECT (DRAFT | PENDING → APPROVED | REJECTED).
// Ghi đúng một dòng `audit_log` (`question.review`) và phát `question.reviewed` (xoá cache "Hôm nay") cùng transaction.
func (s *Service) Review(ctx context.Context, actor, courseID, id uuid.UUID, decision string, version int) (QuestionDetail, error) {
	var to store.QuestionReviewStatus
	switch decision {
	case DecisionRequest:
		to = store.QuestionReviewStatusPENDING
	case DecisionApprove:
		to = store.QuestionReviewStatusAPPROVED
	case DecisionReject:
		to = store.QuestionReviewStatusREJECTED
	default:
		return QuestionDetail{}, fieldErr("decision", "decision", "decision phải là REQUEST, APPROVE hoặc REJECT.")
	}
	var out QuestionDetail
	err := s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		if err := writable(ctx, q, courseID); err != nil {
			return err
		}
		row, err := q.QuestionLock(ctx, store.QuestionLockParams{CourseID: courseID, ID: id})
		if errors.Is(err, pgx.ErrNoRows) {
			return notFound()
		}
		if err != nil {
			return fmt.Errorf("exam: khoá câu hỏi: %w", err)
		}
		if int(row.Version) != version {
			cur, derr := s.detail(ctx, q, row)
			if derr != nil {
				return derr
			}
			return &VersionConflict{Version: int(row.Version), Current: cur}
		}
		from := row.ReviewStatus
		okFrom := from == store.QuestionReviewStatusDRAFT || (from == store.QuestionReviewStatusPENDING && to != store.QuestionReviewStatusPENDING)
		if row.ArchivedAt != nil || !okFrom {
			return conflict("Trạng thái hiện tại của câu hỏi không cho thao tác này.")
		}
		if to == store.QuestionReviewStatusAPPROVED {
			errs, err := s.approvalErrors(ctx, q, row)
			if err != nil {
				return err
			}
			if len(errs) > 0 {
				return apierr.Validation(errs...)
			}
		}
		arg := store.QuestionReviewParams{CourseID: courseID, ID: id, ExpectedVersion: row.Version, ReviewStatus: to}
		if to != store.QuestionReviewStatusPENDING {
			now := s.now()
			arg.ReviewedBy, arg.ReviewedAt = &actor, &now
		}
		upd, err := q.QuestionReview(ctx, arg)
		if err != nil {
			return fmt.Errorf("exam: duyệt câu hỏi: %w", err)
		}
		if err := audit(ctx, q, courseID, actor, "question", id, "question.review",
			map[string]any{"review_status": from, "version": row.Version}, map[string]any{"review_status": to, "version": upd.Version}); err != nil {
			return err
		}
		if _, err := outbox.Write(ctx, tx, TopicQuestionReviewed, map[string]any{"course_id": courseID}); err != nil {
			return fmt.Errorf("exam: outbox: %w", err)
		}
		out, err = s.detail(ctx, q, upd)
		return err
	})
	return out, err
}

// Duplicate tạo bản sao DRAFT (kể cả test và lời giải mẫu): `tests_version` = 1, chưa kiểm lời giải mẫu, `origin=MANUAL`, tiêu đề + " (bản sao)".
func (s *Service) Duplicate(ctx context.Context, actor, courseID, id uuid.UUID) (QuestionDetail, error) {
	var out QuestionDetail
	var copied []string // khoá blob mới (xoá nếu transaction lỗi)
	err := s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
		if err := writable(ctx, q, courseID); err != nil {
			return err
		}
		src, err := q.QuestionGet(ctx, store.QuestionGetParams{CourseID: courseID, ID: id})
		if errors.Is(err, pgx.ErrNoRows) {
			return notFound()
		}
		if err != nil {
			return fmt.Errorf("exam: đọc câu hỏi: %w", err)
		}
		title := src.Title
		const suffix = " (bản sao)"
		if utf8.RuneCountInString(title)+utf8.RuneCountInString(suffix) > MaxTitle {
			title = string([]rune(title)[:MaxTitle-utf8.RuneCountInString(suffix)])
		}
		old, err := q.QuestionOptions(ctx, store.QuestionOptionsParams{CourseID: courseID, QuestionID: id})
		if err != nil {
			return fmt.Errorf("exam: đọc đáp án: %w", err)
		}
		ids := map[uuid.UUID]uuid.UUID{}
		for _, o := range old {
			ids[o.ID], _ = uuid.NewV7()
		}
		key := src.AnswerKey
		if src.Type == store.QuestionTypeMCQSINGLE || src.Type == store.QuestionTypeMCQMULTI {
			var k struct {
				OptionIDs []uuid.UUID `json:"option_ids"`
			}
			if err := json.Unmarshal(src.AnswerKey, &k); err != nil {
				return fmt.Errorf("exam: answer_key hỏng: %w", err)
			}
			for i, o := range k.OptionIDs {
				k.OptionIDs[i] = ids[o]
			}
			key, _ = json.Marshal(k)
		}
		dst, err := q.QuestionInsert(ctx, store.QuestionInsertParams{
			CourseID: courseID, Type: src.Type, Title: title + suffix, Topic: src.Topic, Difficulty: src.Difficulty, Stem: src.Stem, AnswerKey: key, Explanation: src.Explanation,
			Origin: store.QuestionOriginMANUAL, ReviewStatus: store.QuestionReviewStatusDRAFT, CreatedBy: actor,
		})
		if err != nil {
			return fmt.Errorf("exam: ghi bản sao: %w", err)
		}
		for _, o := range old {
			if _, err := q.QuestionOptionInsert(ctx, store.QuestionOptionInsertParams{ID: ids[o.ID], CourseID: courseID, QuestionID: dst.ID, Position: o.Position, Body: o.Body, PinnedLast: o.PinnedLast}); err != nil {
				return fmt.Errorf("exam: ghi đáp án: %w", err)
			}
		}
		if src.Type == store.QuestionTypeCODE {
			if _, err := q.CodeProblemInsert(ctx, store.CodeProblemInsertParams{CourseID: courseID, QuestionID: dst.ID}); err != nil {
				return fmt.Errorf("exam: ghi bài code bản sao: %w", err)
			}
			keys, err := s.copyCode(ctx, q, courseID, id, dst.ID)
			copied = keys
			if err != nil {
				return err
			}
		}
		out, err = s.detail(ctx, q, dst)
		return err
	})
	if err != nil {
		s.dropBlobs(ctx, copied)
	}
	return out, err
}
