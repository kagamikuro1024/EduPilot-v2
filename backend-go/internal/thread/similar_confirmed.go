package thread

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/store"
)

// SuggestGate cho biết cờ THREAD_SUGGEST của lớp (grading.GuidanceFlags thoả; khai báo ở đây để thread không import grading).
type SuggestGate interface {
	On(course uuid.UUID, feature string) bool
}

// SuggestConfig: THREAD_SUGGEST_MIN (cosine) và THREAD_SUGGEST_RPM. Gate nil → tính năng tắt.
type SuggestConfig struct {
	Gate SuggestGate
	Min  float64
	RPM  int
}

// SimilarConfirmedRow là một thread đã xác nhận có câu hỏi gần với chữ đang soạn.
type SimilarConfirmedRow struct {
	ThreadID uuid.UUID `json:"thread_id"`
	Title    string    `json:"title"`
}

const (
	suggestMinWords = 3
	suggestMaxRunes = 4000
	suggestFeature  = "THREAD_SUGGEST"
)

// SimilarConfirmed gợi ý ≤ 3 thread «Đã xác nhận» cùng lớp gần với chữ đang soạn (FEAT-grading-guidance 4.4, M4). Chỉ đọc: không ghi DB, 0 lời gọi sinh chữ, chữ `q` không ghi log.
// Mọi lỗi (cờ tắt, chưa đủ chữ, có PII, nhúng / truy vấn hỏng) → danh sách rỗng; chỉ quá hạn mức mới 429 và `q` quá dài mới 422.
func (s *Service) SimilarConfirmed(ctx context.Context, a Actor, courseID uuid.UUID, q string) ([]SimilarConfirmedRow, error) {
	out := []SimilarConfirmedRow{}
	if utf8.RuneCountInString(q) > suggestMaxRunes {
		return out, apierr.Validation(apierr.FieldError{Field: "q", Code: "max", Message: "Nội dung quá dài."})
	}
	if s.Suggest.Gate == nil || !s.Suggest.Gate.On(courseID, suggestFeature) {
		return out, nil
	}
	if len(strings.Fields(q)) < suggestMinWords {
		return out, nil
	}
	if err := s.rate(ctx, "suggest", a.UserID, max(s.Suggest.RPM, 1)); err != nil {
		return out, err
	}
	if s.FW == nil || s.FW.Embed == nil {
		return out, nil
	}
	if s.FW.Detector != nil { // bản nháp chưa qua tường lửa: có tên / MSSV / email thì không nhúng, không gọi mạng
		if f, err := s.FW.Detector.Detect(ctx, courseID, q); err != nil || len(f) > 0 {
			return out, nil
		}
	}
	vec, err := s.FW.Embed(ctx, q)
	if err != nil || len(vec) == 0 {
		s.warn(ctx, "thread.similar_confirmed: không nhúng được") // không kèm chữ
		return out, nil
	}
	rows, err := store.New(s.Pool).KnowledgeSimilarPosts(ctx, store.KnowledgeSimilarPostsParams{Vec: pgvector.NewVector(vec), CourseID: courseID})
	if err != nil {
		s.warn(ctx, "thread.similar_confirmed: truy vấn lỗi")
		return out, nil
	}
	seen := map[uuid.UUID]bool{}
	for _, r := range rows {
		if r.Cosine < s.Suggest.Min || seen[r.ThreadID] {
			continue
		}
		seen[r.ThreadID] = true
		out = append(out, SimilarConfirmedRow{ThreadID: r.ThreadID, Title: r.Title})
	}
	return out, nil
}

func (s *Service) warn(ctx context.Context, msg string) {
	if s.Log != nil {
		s.Log.WarnContext(ctx, msg)
	}
}
