package grading

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/store"
)

// GradeText chấm một văn bản ĐÃ LÀM SẠCH bằng đúng hai lượt + Merge như Run (dùng chung passes / onePass), KHÔNG ghi DB.
// Dùng cho `cmd/regress` (M2): `in.Guidance` / `in.GuidanceVersion` là lưu ý muốn thử.
func (g *Grader) GradeText(ctx context.Context, in PromptInput) (Merged, error) {
	c := store.EssayGradeContextRow{Title: in.Title, Description: in.Description}
	if in.Reference != "" {
		c.ReferenceAnswer = &in.Reference
	}
	ctx = withPin(ctx, guidancePin{Version: in.GuidanceVersion, Lines: in.Guidance})
	p1, p2, err := g.passes(ctx, c, in.Rubric, in.Text, in.Text, in.FlagInjection)
	if err != nil {
		return Merged{}, err
	}
	return Merge(p1, p2, in.Rubric, g.Cfg.Disagree), nil
}

// GuidanceLinesAt trả lưu ý của bài tập tại phiên bản v theo thứ tự rubric (hàm thuần theo (bài tập, v)).
func GuidanceLinesAt(ctx context.Context, q *store.Queries, course, assignment uuid.UUID, v int16, rubric []RubricItem) ([]GuidanceLine, error) {
	rows, err := q.GuidanceSetAt(ctx, store.GuidanceSetAtParams{CourseID: course, AssignmentID: assignment, V: v})
	if err != nil {
		return nil, fmt.Errorf("grading: lưu ý tại phiên bản %d: %w", v, err)
	}
	return orderByRubric(rows, rubric), nil
}
