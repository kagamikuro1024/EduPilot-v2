package exam

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/store"
)

// ItemIn là một mục gửi lên.
type ItemIn struct {
	QuestionID uuid.UUID       `json:"question_id"`
	Points     decimal.Decimal `json:"points"`
}

// ItemsIn là thân `PUT …/exams/{eid}/items`.
type ItemsIn struct {
	Items   []ItemIn `json:"items"`
	Version *int     `json:"version"`
}

// maxItems: số mục tối đa của một bài (SRS 4.2.2).
const maxItems = 100

// kindOf suy ra `kind` từ loại các câu: chỉ trắc nghiệm → MCQ; chỉ code → CODE; cả hai → MIXED; rỗng → MCQ.
func kindOf(types []string) string {
	var code, mcq bool
	for _, t := range types {
		if t == string(store.QuestionTypeCODE) {
			code = true
		} else {
			mcq = true
		}
	}
	switch {
	case code && mcq:
		return "MIXED"
	case code:
		return "CODE"
	}
	return "MCQ"
}

// PutItems thay TOÀN BỘ danh sách mục theo thứ tự gửi (SRS 4.2.2). Chỉ bài DRAFT; sai → 422 với mọi lỗi.
func (s *Service) PutItems(ctx context.Context, actor, courseID, examID uuid.UUID, in ItemsIn, version int) (ExamDetail, error) {
	var out ExamDetail
	err := s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
		e, err := s.lockExam(ctx, q, courseID, examID)
		if err != nil {
			return err
		}
		if int(e.Version) != version {
			return s.conflictOf(ctx, q, e)
		}
		if e.Status != store.ExamStatusDRAFT {
			return lockedErr("status")
		}
		types, errs, err := s.checkItems(ctx, q, courseID, in.Items)
		if err != nil {
			return err
		}
		if len(errs) > 0 {
			return apierr.Validation(errs...)
		}
		if err := q.ExamItemsDelete(ctx, store.ExamItemsDeleteParams{CourseID: courseID, ExamID: examID}); err != nil {
			return fmt.Errorf("exam: xoá mục cũ: %w", err)
		}
		for i, it := range in.Items {
			if err := q.ExamItemInsert(ctx, store.ExamItemInsertParams{CourseID: courseID, ExamID: examID, QuestionID: it.QuestionID, Position: int16(i + 1), Points: it.Points}); err != nil { //nolint:gosec // ≤ 100
				return fmt.Errorf("exam: thêm mục: %w", err)
			}
		}
		if err := q.ExamSetKind(ctx, store.ExamSetKindParams{CourseID: courseID, ID: examID, Kind: store.ExamKind(kindOf(types))}); err != nil { // đồng thời tăng version
			return fmt.Errorf("exam: đặt loại bài thi: %w", err)
		}
		row, err := q.ExamGet(ctx, store.ExamGetParams{CourseID: courseID, ID: examID})
		if err != nil {
			return fmt.Errorf("exam: đọc lại bài thi: %w", err)
		}
		if err := audit(ctx, q, courseID, actor, "exam", examID, "exam.items", map[string]any{"version": int(e.Version)}, map[string]any{"items": len(in.Items), "kind": string(row.Kind), "version": int(row.Version)}); err != nil {
			return err
		}
		out, err = s.examDetail(ctx, q, row)
		return err
	})
	return out, err
}

// checkItems kiểm danh sách mục, trả loại câu theo thứ tự gửi và MỌI lỗi (không dừng ở lỗi đầu).
func (s *Service) checkItems(ctx context.Context, q *store.Queries, courseID uuid.UUID, items []ItemIn) ([]string, []apierr.FieldError, error) {
	if len(items) == 0 {
		return nil, []apierr.FieldError{ve("items", "NO_ITEMS", "Bài thi cần ít nhất một câu hỏi.")}, nil
	}
	if len(items) > maxItems {
		return nil, []apierr.FieldError{ve("items", "LIMIT_OUT_OF_RANGE", "Một bài thi có tối đa 100 câu.")}, nil
	}
	ids := make([]string, len(items))
	for i, it := range items {
		ids[i] = it.QuestionID.String()
	}
	rows, err := q.ExamQuestionsByIDs(ctx, store.ExamQuestionsByIDsParams{CourseID: courseID, Ids: ids})
	if err != nil {
		return nil, nil, fmt.Errorf("exam: tra câu hỏi: %w", err)
	}
	byID := make(map[uuid.UUID]store.ExamQuestionsByIDsRow, len(rows))
	for _, r := range rows {
		byID[r.ID] = r
	}
	var errs []apierr.FieldError
	seen := map[uuid.UUID]bool{}
	types := make([]string, 0, len(items))
	for i, it := range items {
		f := fmt.Sprintf("items[%d]", i)
		if seen[it.QuestionID] {
			errs = append(errs, ve(f+".question_id", "DUPLICATE_ITEM", "Câu hỏi này đã có trong bài."))
		}
		seen[it.QuestionID] = true
		if !it.Points.IsPositive() || it.Points.GreaterThan(hundred()) || it.Points.Exponent() < -2 {
			errs = append(errs, ve(f+".points", "LIMIT_OUT_OF_RANGE", "Điểm của câu lớn hơn 0, không quá 100, tối đa 2 chữ số thập phân."))
		}
		r, ok := byID[it.QuestionID]
		switch {
		case !ok:
			errs = append(errs, ve(f+".question_id", "QUESTION_NOT_IN_COURSE", "Câu hỏi không thuộc lớp này."))
		case r.ReviewStatus != store.QuestionReviewStatusAPPROVED || r.ArchivedAt != nil:
			errs = append(errs, ve(f+".question_id", "ITEM_NOT_APPROVED", "Chỉ chọn được câu đã duyệt và chưa lưu trữ."))
		default:
			types = append(types, string(r.Type))
		}
	}
	return types, errs, nil
}
