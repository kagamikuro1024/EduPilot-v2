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

// gradedWindow: kết quả đã công bố hiện ở "Hôm nay" (ASSIGNMENT_GRADED) trong chừng này.
const gradedWindow = 7 * 24 * time.Hour

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
	rows, err := store.New(p.Pool).TodayStudentAssignments(ctx, store.TodayStudentAssignmentsParams{UserID: v.UserID, CourseIds: ids(courses), Now: v.Now, Horizon: v.Now.Add(assignmentHorizon), GradedSince: v.Now.Add(-gradedWindow)})
	if err != nil {
		return nil, fmt.Errorf("today: bài tập: %w", err)
	}
	byCourse := map[uuid.UUID]CourseRef{}
	for _, c := range courses {
		byCourse[c.ID] = c
	}
	// Hàng đã sắp theo (kind, thời điểm tăng dần): DUE lấy dòng ĐẦU của mỗi lớp (hạn gần nhất); GRADED / REPLY lấy dòng CUỐI (mới nhất).
	pick := map[string]int{}
	for i, r := range rows {
		k := r.Kind + r.CourseID.String()
		if _, seen := pick[k]; !seen || r.Kind != "DUE" {
			pick[k] = i
		}
	}
	var out []Item
	for i, r := range rows {
		if pick[r.Kind+r.CourseID.String()] != i {
			continue
		}
		c := byCourse[r.CourseID]
		switch r.Kind {
		case "GRADED":
			out = append(out, mk(Item{ID: "ASSIGNMENT_GRADED:" + r.ID.String(), Kind: KindAssignmentGraded, Tier: TierAssignmentGraded, Course: courseRef(c), Href: "/assignments/" + r.ID.String(),
				EstimateMinutes: 5, Title: r.Title, Reason: "Điểm " + r.Title + " đã có — Xem"}))
			continue
		case "REPLY":
			out = append(out, mk(Item{ID: "ASSIGNMENT_APPEAL_REPLY:" + r.ID.String(), Kind: KindAssignmentAppealReply, Tier: TierAssignmentAppealReply, Course: courseRef(c), Href: "/assignments/" + r.ID.String(),
				EstimateMinutes: 5, Title: r.Title, Reason: "Giảng viên đã trả lời yêu cầu xem lại — Xem"}))
			continue
		}
		reason := "Đang nộp muộn — Nộp bài"
		if r.At.After(v.Now) {
			hours := max(1, int((r.At.Sub(v.Now)+time.Hour-1)/time.Hour))
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
		if r.AppealN > 0 && v.Role == RoleTeacher { // phúc khảo: chỉ TEACHER quyết; Overdue khi chờ quá 48 giờ
			out = append(out, mk(Item{ID: "APPEAL:" + c.ID.String(), Kind: KindAppeal, Tier: TierAppeal, Course: courseRef(c), Href: "/grading/" + r.AppealSub.String(),
				Title: fmt.Sprintf("%d yêu cầu xem lại điểm · %s", r.AppealN, r.AppealTitle), Reason: "Giữ hoặc sửa điểm kèm phản hồi.", Overdue: v.Now.Sub(r.AppealOldest) > 48*time.Hour}))
		}
		if r.PubN > 0 && v.Role == RoleTeacher { // công bố chỉ TEACHER (TA không thấy mục này)
			out = append(out, mk(Item{ID: "PUBLISH_PENDING:" + c.ID.String(), Kind: KindPublishPending, Tier: TierPublishPending, Course: courseRef(c),
				Href: "/grading", Title: fmt.Sprintf("%d bài đã duyệt chờ công bố · %s", r.PubN, r.PubTitle), Reason: "Chọn các bài đã duyệt rồi công bố."}))
		}
		if r.RevN > 0 {
			out = append(out, mk(Item{ID: "GRADING_REVIEW:" + c.ID.String(), Kind: KindGradingReview, Tier: TierGradingReview, Course: courseRef(c),
				Href: "/grading", Title: fmt.Sprintf("%d bài cần xem kỹ · %s", r.RevN, r.RevTitle), Reason: "Xem lại các bài có cờ trước khi duyệt."}))
		}
		out = append(out, guidanceItems(v, c, r)...)
	}
	return out, nil
}

// guidanceItems (10.6): «Lưu ý chấm chờ duyệt» (TEACHER) và «N bài chấm theo bản cũ» (Giảng viên + TA); số liệu đã nằm trong truy vấn gộp của staffItems (chỉ lớp bật cờ GRADING_GUIDANCE). Sinh viên không bao giờ thấy.
func guidanceItems(v Viewer, c CourseRef, r store.TodayStaffAssignmentBatchRow) []Item {
	var out []Item
	if r.GuidanceDraftN > 0 && v.Role == RoleTeacher {
		out = append(out, mk(Item{ID: "GUIDANCE_PENDING:" + c.ID.String(), Kind: KindGuidancePending, Tier: TierGuidancePending, Course: courseRef(c),
			Href: "/grading?tab=assignments", Title: "Lưu ý chấm chờ duyệt · " + r.GuidanceDraftTitle, Reason: fmt.Sprintf("%d lưu ý — Duyệt hoặc Bỏ.", r.GuidanceDraftN)}))
	}
	if r.GuidanceStaleN > 0 {
		out = append(out, mk(Item{ID: "GUIDANCE_STALE:" + c.ID.String(), Kind: KindGuidanceStale, Tier: TierGuidanceStale, Course: courseRef(c),
			Href: "/grading?tab=queue&assignment=" + r.GuidanceStaleAid.String(), Title: fmt.Sprintf("%d bài chấm theo bản cũ · %s", r.GuidanceStaleN, r.GuidanceStaleTitle), Reason: "Chấm lại các bài nháp chưa sửa."}))
	}
	return out
}
