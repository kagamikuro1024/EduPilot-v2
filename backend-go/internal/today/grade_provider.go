package today

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/edupilot/backend-go/internal/grade"
	"github.com/edupilot/backend-go/internal/store"
)

// GradeProvider: GRADE_SCHEME_UNCONFIRMED (bậc 70, Giảng viên + TA) — mỗi lớp ACTIVE của Staff có công thức chưa CONFIRMED một mục (US-P6-05 AC5).
// Một truy vấn dữ kiện + bộ kiểm cho cả lô lớp; không đọc điểm. Xác nhận công thức → outbox `grade.scheme_confirmed` xoá cache nên mục biến mất ≤ 5 s.
type GradeProvider struct{ Pool *pgxpool.Pool }

// Name implements Provider.
func (GradeProvider) Name() string { return "grade" }

// Items implements Provider.
func (p GradeProvider) Items(ctx context.Context, v Viewer, s Scope) ([]Item, error) {
	courses := v.Scoped(s, "STAFF")
	if (v.Role != RoleTeacher && v.Role != RoleTA) || len(courses) == 0 {
		return nil, nil
	}
	st, err := grade.StatusBatch(ctx, store.New(p.Pool), uuidsOf(courses))
	if err != nil {
		return nil, fmt.Errorf("today: công thức điểm: %w", err)
	}
	var out []Item
	for _, c := range courses {
		x, ok := st[c.ID]
		if !ok || x.State == grade.StateConfirmed {
			continue
		}
		out = append(out, mk(Item{
			ID: "GRADE_SCHEME_UNCONFIRMED:" + c.ID.String(), Kind: KindGradeSchemeUnconfirmed, Tier: TierGradeSchemeUnconfirmed, Course: courseRef(c),
			Href: "/gradebook/scheme?course=" + c.ID.String(), Title: "Công thức điểm chưa xác nhận · " + c.ClassCode, Reason: gradeReason(x),
		}))
	}
	return out, nil
}

func gradeReason(st grade.SchemeStatus) string {
	switch st.State {
	case grade.StateNoPolicy:
		return "Lớp chưa có quy chế môn học."
	case grade.StateNotExtracted:
		return "Đã có quy chế, chưa trích công thức."
	case grade.StateDraftUnclear:
		return fmt.Sprintf("Bản nháp còn %d mục chưa rõ.", st.UnclearCount)
	}
	return "Bản nháp sẵn sàng, chờ xác nhận."
}

func uuidsOf(cs []CourseRef) []uuid.UUID {
	out := make([]uuid.UUID, len(cs))
	for i, c := range cs {
		out[i] = c.ID
	}
	return out
}
