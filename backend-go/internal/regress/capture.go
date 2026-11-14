// Package regress ghi «ca hồi quy» (FEAT-grading-guidance M2) khi giảng viên sửa điểm / phúc khảo / xử lý bài đăng: chỉ THAM CHIẾU + số / enum, KHÔNG văn bản bài, nhận xét hay danh tính.
// `cmd/regress` đọc lại ca cùng nội dung ngay trong DB để chạy hồi quy; ở đây chỉ có phần ghi (dùng trong giao dịch nghiệp vụ, nên không import internal/grading).
package regress

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/store"
)

// Feature là tên cờ lớp (khớp grading.FlagRegressionCapture).
const Feature = "REGRESSION_CAPTURE"

// Kind của ca.
const (
	KindGradingEdit    = "GRADING_EDIT"
	KindGradingAppeal  = "GRADING_APPEAL"
	KindThreadCorrect  = "THREAD_CORRECTED"
	KindThreadRejected = "THREAD_REJECTED"
)

// maxStr là độ dài chuỗi tối đa trong `expected` (số / enum / băm, không văn bản).
const maxStr = 64

// Gate cho biết cờ lớp.
type Gate interface {
	On(course uuid.UUID, feature string) bool
}

// RubricHash là băm của rubric đã chuẩn hoá (khoá map được sắp): `run` bỏ qua ca có rubric đã đổi.
func RubricHash(rubric []byte) string {
	var v any
	if json.Unmarshal(rubric, &v) != nil {
		v = string(rubric)
	}
	b, _ := json.Marshal(v)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// Direction: STRICTER khi AI chấm cao hơn giảng viên (giảng viên chấm chặt hơn), ngược lại LENIENT.
func Direction(ai, teacher decimal.Decimal) string {
	if ai.GreaterThan(teacher) {
		return "STRICTER"
	}
	return "LENIENT"
}

// Validate bảo đảm `expected` chỉ có số / bool / chuỗi ngắn (TestRegressionCaseNoText).
func Validate(expected map[string]any) error {
	for k, v := range expected {
		switch x := v.(type) {
		case string:
			if len(x) > maxStr {
				return fmt.Errorf("regress: %q dài %d > %d", k, len(x), maxStr)
			}
		case bool, float64, int, int64:
		default:
			return fmt.Errorf("regress: %q không phải số / enum", k)
		}
	}
	return nil
}

type critScore struct {
	ID    string  `json:"id"`
	Score *string `json:"score"`
}

func scoresOf(raw []byte) map[string]decimal.Decimal {
	var cs []critScore
	_ = json.Unmarshal(raw, &cs)
	out := map[string]decimal.Decimal{}
	for _, c := range cs {
		if c.Score == nil {
			continue
		}
		if d, err := decimal.NewFromString(*c.Score); err == nil {
			out[c.ID] = d
		}
	}
	return out
}

func upsert(ctx context.Context, q *store.Queries, course uuid.UUID, kind string, source uuid.UUID, criterion *string, expected map[string]any) error {
	if err := Validate(expected); err != nil {
		return err
	}
	b, _ := json.Marshal(expected)
	if err := q.RegressionCaseUpsert(ctx, store.RegressionCaseUpsertParams{CourseID: course, Kind: kind, SourceID: source, CriterionID: criterion, Expected: b}); err != nil {
		return fmt.Errorf("regress: ghi ca: %w", err)
	}
	return nil
}

// CaptureGrading ghi một ca cho MỖI tiêu chí có |điểm giảng viên − điểm AI| ≥ minDelta (cờ lớp bật). `only` (tuỳ ý) giới hạn trong các tiêu chí đã đổi (phúc khảo). Idempotent theo (lớp, loại, nguồn, tiêu chí).
func CaptureGrading(ctx context.Context, q *store.Queries, gate Gate, course uuid.UUID, kind string, resultID uuid.UUID, rubric, criteria, aiScores []byte, minDelta decimal.Decimal, only map[string]bool) error {
	if gate == nil || !gate.On(course, Feature) {
		return nil
	}
	ai := map[string]decimal.Decimal{}
	var raw map[string]string
	if json.Unmarshal(aiScores, &raw) != nil {
		return nil
	}
	for k, v := range raw {
		if d, err := decimal.NewFromString(v); err == nil {
			ai[k] = d
		}
	}
	hash := RubricHash(rubric)
	for id, teacher := range scoresOf(criteria) {
		a, ok := ai[id]
		if !ok || (only != nil && !only[id]) || a.Sub(teacher).Abs().LessThan(minDelta) {
			continue
		}
		cid := id
		if err := upsert(ctx, q, course, kind, resultID, &cid, map[string]any{
			"ai_score": a.StringFixed(2), "teacher_score": teacher.StringFixed(2), "direction": Direction(a, teacher), "rubric_hash": hash,
		}); err != nil {
			return err
		}
	}
	return nil
}

// ChangedCriteria: tiêu chí có điểm khác giữa hai bản `criteria` (so decimal).
func ChangedCriteria(before, after []byte) map[string]bool {
	b, a := scoresOf(before), scoresOf(after)
	out := map[string]bool{}
	for id, s := range a {
		if o, ok := b[id]; !ok || !o.Equal(s) {
			out[id] = true
		}
	}
	return out
}

// CaptureThread ghi ca bài đăng bị sửa / loại (`state` = CORRECTED | REJECTED); nguồn là id bài, không có chữ.
func CaptureThread(ctx context.Context, q *store.Queries, gate Gate, course, postID uuid.UUID, state string) error {
	if gate == nil || !gate.On(course, Feature) {
		return nil
	}
	kind := KindThreadCorrect
	switch state {
	case "CORRECTED":
	case "REJECTED":
		kind = KindThreadRejected
	default:
		return nil
	}
	return upsert(ctx, q, course, kind, postID, nil, map[string]any{"state": state})
}

// AIScores đọc ai_scores của một kết quả (phúc khảo không mang sẵn); không có hàng → rỗng.
func AIScores(ctx context.Context, q *store.Queries, course, result uuid.UUID) ([]byte, error) {
	b, err := q.RegressionAIScores(ctx, store.RegressionAIScoresParams{CourseID: course, ID: result})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("regress: đọc ai_scores: %w", err)
	}
	return b, nil
}
