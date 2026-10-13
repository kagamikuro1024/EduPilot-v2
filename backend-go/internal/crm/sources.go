package crm

import (
	"context"
	"time"

	"github.com/edupilot/backend-go/internal/agent"
	"github.com/edupilot/backend-go/internal/store"
)

var ict = time.FixedZone("Asia/Ho_Chi_Minh", 7*3600)

// Attendance cài agent.AttendanceSource (SRS 4.5): đọc CHỈ theo (tc.CourseID, tc.UserID); không `note`, không `marked_by`, không tên ai.
// Không có dòng điểm danh nào ⇒ (nil, false) để tool trả "chưa có dữ liệu" mà không gọi LLM.
func (s *Service) Attendance(ctx context.Context, tc agent.TrustedContext) (agent.Facts, bool, error) {
	q := store.New(s.Pool)
	sums, err := q.CrmMyAttendanceSummary(ctx, store.CrmMyAttendanceSummaryParams{CourseID: tc.CourseID, StudentID: tc.UserID})
	if err != nil {
		return nil, false, wrap("điểm danh của tôi", err)
	}
	if len(sums) == 0 || sums[0].Recorded == 0 {
		return nil, false, nil
	}
	abs, err := q.CrmMyRecentAbsent(ctx, store.CrmMyRecentAbsentParams{CourseID: tc.CourseID, StudentID: tc.UserID})
	if err != nil {
		return nil, false, wrap("buổi vắng gần đây", err)
	}
	recent := make([]map[string]any, len(abs))
	for i, a := range abs {
		recent[i] = map[string]any{"session_no": int(a.SessionNo), "date": a.StartsAt.In(ict).Format(time.DateOnly)}
	}
	x := sums[0]
	return agent.Facts{"recorded": int(x.Recorded), "present": int(x.Present), "late": int(x.Late), "excused": int(x.Excused), "absent": int(x.Absent), "recent_absent": recent}, true, nil
}

// Participation cài agent.ParticipationSource: số lần phát biểu và tổng điểm cộng ĐÃ GHI (Σ points, decimal; không áp trần — Q10).
func (s *Service) Participation(ctx context.Context, tc agent.TrustedContext) (agent.Facts, bool, error) {
	q := store.New(s.Pool)
	sum, err := q.CrmMyParticipationSummary(ctx, store.CrmMyParticipationSummaryParams{CourseID: tc.CourseID, StudentID: tc.UserID})
	if err != nil {
		return nil, false, wrap("phát biểu của tôi", err)
	}
	if sum.Events == 0 {
		return nil, false, nil
	}
	rec, err := q.CrmMyParticipationRecent(ctx, store.CrmMyParticipationRecentParams{CourseID: tc.CourseID, StudentID: tc.UserID})
	if err != nil {
		return nil, false, wrap("sự kiện gần đây", err)
	}
	recent := make([]map[string]any, len(rec))
	for i, r := range rec {
		recent[i] = map[string]any{"date": r.CreatedAt.In(ict).Format(time.DateOnly), "type": string(r.Type), "points": r.Points.StringFixed(2)}
	}
	return agent.Facts{"speak_count": int(sum.SpeakCount), "points_total": sum.PointsTotal.StringFixed(2), "recent": recent}, true, nil
}
