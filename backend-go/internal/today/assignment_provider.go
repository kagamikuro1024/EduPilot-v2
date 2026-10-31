package today

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/edupilot/backend-go/internal/store"
)

// assignmentHorizon: bài có hạn trong khoảng này (hoặc đã quá hạn còn cửa muộn) mới thành việc ASSIGNMENT_DUE.
const assignmentHorizon = 24 * time.Hour

// AssignmentProvider: ASSIGNMENT_DUE (bậc 41, sinh viên) — SRS FEAT-assignments 4.12. MỘT truy vấn cho mọi lớp trong phạm vi; một mục mỗi lớp (bài có hạn gần nhất).
// Nộp bài / đóng cửa → outbox `submission.created` / `assignment.updated` xoá cache nên mục biến mất ≤ 5 s. Các kind của nhân viên (GRADING_*, APPEAL…) do các story sau thêm vào cùng nhà cung cấp.
type AssignmentProvider struct{ Pool *pgxpool.Pool }

// Name implements Provider.
func (AssignmentProvider) Name() string { return "assignment" }

// Items implements Provider.
func (p AssignmentProvider) Items(ctx context.Context, v Viewer, sc Scope) ([]Item, error) {
	if v.Role == RoleTeacher || v.Role == RoleTA {
		return p.staffItems(ctx, v, sc)
	}
	if v.Role != RoleStudent {
		return nil, nil
	}
	courses := v.Scoped(sc, "STUDENT")
	if p.Pool == nil || len(courses) == 0 {
		return nil, nil
	}
	rows, err := store.New(p.Pool).TodayStudentAssignments(ctx, store.TodayStudentAssignmentsParams{UserID: v.UserID, CourseIds: ids(courses), Now: v.Now, Horizon: v.Now.Add(assignmentHorizon)})
	if err != nil {
		return nil, fmt.Errorf("today: bài tập: %w", err)
	}
	byCourse := map[uuid.UUID]CourseRef{}
	for _, c := range courses {
		byCourse[c.ID] = c
	}
	seen := map[uuid.UUID]bool{}
	var out []Item
	for _, r := range rows { // đã sắp theo hạn: dòng đầu của mỗi lớp là bài gần nhất
		if seen[r.CourseID] {
			continue
		}
		seen[r.CourseID] = true
		c := byCourse[r.CourseID]
		reason := "Đang nộp muộn — Nộp bài"
		if r.DueAt.After(v.Now) {
			hours := max(1, int((r.DueAt.Sub(v.Now)+time.Hour-1)/time.Hour))
			reason = fmt.Sprintf("Còn %d giờ — Nộp bài", hours)
		}
		out = append(out, mk(Item{ID: "ASSIGNMENT_DUE:" + r.ID.String(), Kind: KindAssignmentDue, Tier: TierAssignmentDue, Course: courseRef(c), Href: "/assignments/" + r.ID.String(),
			EstimateMinutes: 30, Title: r.Title, Reason: reason}))
	}
	return out, nil
}

// staffItems: UNMATCHED_SUBMISSION (60), GRADING_ERROR (28), GRADING_REVIEW (30) — mỗi (lớp, kind) một mục; MỘT truy vấn tổng hợp cho mọi lớp (SRS 4.12).
// Các kind APPEAL / PUBLISH_PENDING của nhân viên cùng truy vấn này ở các story sau.
func (p AssignmentProvider) staffItems(ctx context.Context, v Viewer, sc Scope) ([]Item, error) {
	courses := v.Scoped(sc, "STAFF")
	if p.Pool == nil || len(courses) == 0 {
		return nil, nil
	}
	rows, err := store.New(p.Pool).TodayStaffAssignmentBatch(ctx, ids(courses))
	if err != nil {
		return nil, fmt.Errorf("today: việc chấm bài của nhân viên: %w", err)
	}
	byCourse := map[uuid.UUID]CourseRef{}
	for _, c := range courses {
		byCourse[c.ID] = c
	}
	var out []Item
	for _, r := range rows {
		c, ok := byCourse[r.CourseID]
		if !ok {
			continue
		}
		if r.Unmatched > 0 {
			out = append(out, mk(Item{ID: "UNMATCHED_SUBMISSION:" + c.ID.String(), Kind: KindUnmatchedSubmission, Tier: TierUnmatchedSubmission, Course: courseRef(c),
				Href: "/grading?tab=unmatched&course=" + c.ID.String(), Title: fmt.Sprintf("%d bài nộp chưa khớp sinh viên · lớp %s", r.Unmatched, c.ClassCode), Reason: "Gán cho sinh viên hoặc bỏ qua."}))
		}
		if r.ErrN > 0 {
			out = append(out, mk(Item{ID: "GRADING_ERROR:" + c.ID.String(), Kind: KindGradingError, Tier: TierGradingError, Course: courseRef(c),
				Href: "/grading?status=FAILED", Title: fmt.Sprintf("%d bài chấm lỗi · %s", r.ErrN, r.ErrTitle), Reason: "Thử chấm lại hoặc nhập điểm tay."}))
		}
		if r.RevN > 0 {
			out = append(out, mk(Item{ID: "GRADING_REVIEW:" + c.ID.String(), Kind: KindGradingReview, Tier: TierGradingReview, Course: courseRef(c),
				Href: "/grading", Title: fmt.Sprintf("%d bài cần xem kỹ · %s", r.RevN, r.RevTitle), Reason: "Xem lại các bài có cờ trước khi duyệt."}))
		}
	}
	return out, nil
}
