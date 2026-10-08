package exam

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/store"
)

// Loại việc nền của PE (đăng ký ở worker, `jobs.Runner`).
const (
	KindVerifyReference = "code.verify_reference"
	KindSuggest         = "question.suggest"
)

// Loại gợi ý AI (D57).
const (
	SuggestMCQ       = "MCQ"
	SuggestCodeTests = "CODE_TESTS"
)

// MaxSourceText là trần `source_text` của gợi ý AI (SRS 4.9).
const MaxSourceText = 6000

// verifyPayload / SuggestPayload là thân việc ghi vào hàng đợi.
type verifyPayload struct {
	CourseID   uuid.UUID `json:"course_id"`
	QuestionID uuid.UUID `json:"question_id"`
}

// SuggestIn là thân `POST …/questions/suggest`.
type SuggestIn struct {
	Kind       string     `json:"kind" validate:"required"`
	Topic      string     `json:"topic"`
	Difficulty string     `json:"difficulty"`
	Count      int        `json:"count" validate:"required"`
	SourceText string     `json:"source_text"`
	QuestionID *uuid.UUID `json:"question_id"`
}

// SuggestPayload là thân việc `question.suggest` (không có dữ liệu người dùng nào ngoài chủ đề / độ khó / văn bản nguồn do giảng viên nhập).
type SuggestPayload struct {
	CourseID uuid.UUID `json:"course_id"`
	SuggestIn
}

func (s *Service) checkWritable(ctx context.Context, courseID uuid.UUID) error {
	return writable(ctx, store.New(s.Pool), courseID)
}

// EnqueueVerify xếp việc chạy lời giải mẫu qua mọi test đã duyệt (202 + job id). Cần lời giải mẫu và ≥ 1 test đã duyệt.
func (s *Service) EnqueueVerify(ctx context.Context, owner, courseID, id uuid.UUID) (uuid.UUID, error) {
	if err := s.checkWritable(ctx, courseID); err != nil {
		return uuid.Nil, err
	}
	q := store.New(s.Pool)
	row, err := q.QuestionGet(ctx, store.QuestionGetParams{CourseID: courseID, ID: id})
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && row.Type != store.QuestionTypeCODE) {
		return uuid.Nil, notFound()
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("exam: đọc câu hỏi: %w", err)
	}
	cd, err := s.codeDetail(ctx, q, courseID, id)
	if err != nil {
		return uuid.Nil, err
	}
	if cd.Reference == nil {
		return uuid.Nil, fieldErr("reference", "REFERENCE_REQUIRED", "Cần nhập lời giải mẫu trước khi chạy.")
	}
	if cd.Tests.Total < 1 {
		return uuid.Nil, fieldErr("testcases", "CODE_TESTS_MISSING", "Cần ít nhất một test đã duyệt để chạy lời giải mẫu.")
	}
	j, err := s.Jobs.Enqueue(ctx, owner, KindVerifyReference, verifyPayload{CourseID: courseID, QuestionID: id})
	if err != nil {
		return uuid.Nil, fmt.Errorf("exam: xếp việc: %w", err)
	}
	return j.ID, nil
}

// EnqueueSuggest xếp việc AI gợi ý nháp (một việc = đúng một lời gọi `Structured` ở làn BATCH).
func (s *Service) EnqueueSuggest(ctx context.Context, owner, courseID uuid.UUID, in SuggestIn) (uuid.UUID, error) {
	var errs []apierr.FieldError
	bad := func(field, code, msg string) {
		errs = append(errs, apierr.FieldError{Field: field, Code: code, Message: msg})
	}
	in.Topic = strings.TrimSpace(in.Topic)
	if in.Count < 1 || in.Count > 10 {
		bad("count", "LIMIT_OUT_OF_RANGE", "Số câu gợi ý từ 1 đến 10.")
	}
	switch in.Kind {
	case SuggestMCQ:
		if n := utf8.RuneCountInString(in.Topic); n < 1 || n > MaxTopic {
			bad("topic", "TOPIC_LENGTH", "Chủ đề dài 1 đến 80 ký tự.")
		}
		switch in.Difficulty {
		case "":
			in.Difficulty = "MEDIUM"
		case "EASY", "MEDIUM", "HARD":
		default:
			bad("difficulty", "INVALID_DIFFICULTY", "Độ khó phải là EASY, MEDIUM hoặc HARD.")
		}
		if utf8.RuneCountInString(in.SourceText) > MaxSourceText {
			bad("source_text", "SOURCE_TOO_LARGE", "Văn bản nguồn tối đa 6.000 ký tự.")
		}
	case SuggestCodeTests:
		if in.QuestionID == nil {
			bad("question_id", "required", "Cần question_id của bài lập trình.")
		}
	default:
		bad("kind", "INVALID_KIND", "kind phải là MCQ hoặc CODE_TESTS.")
	}
	if len(errs) > 0 {
		return uuid.Nil, apierr.Validation(errs...)
	}
	if err := s.checkWritable(ctx, courseID); err != nil {
		return uuid.Nil, err
	}
	if in.Kind == SuggestCodeTests {
		q := store.New(s.Pool)
		row, err := q.QuestionGet(ctx, store.QuestionGetParams{CourseID: courseID, ID: *in.QuestionID})
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && row.Type != store.QuestionTypeCODE) {
			return uuid.Nil, notFound()
		}
		if err != nil {
			return uuid.Nil, fmt.Errorf("exam: đọc câu hỏi: %w", err)
		}
		cp, err := q.CodeProblemGet(ctx, store.CodeProblemGetParams{CourseID: courseID, QuestionID: row.ID})
		if err != nil {
			return uuid.Nil, fmt.Errorf("exam: đọc bài code: %w", err)
		}
		if cp.ReferenceSource == nil {
			return uuid.Nil, fieldErr("reference", "REFERENCE_REQUIRED", "Cần nhập lời giải mẫu để sinh đầu ra mong đợi.")
		}
	}
	j, err := s.Jobs.Enqueue(ctx, owner, KindSuggest, SuggestPayload{CourseID: courseID, SuggestIn: in})
	if err != nil {
		return uuid.Nil, fmt.Errorf("exam: xếp việc: %w", err)
	}
	return j.ID, nil
}
