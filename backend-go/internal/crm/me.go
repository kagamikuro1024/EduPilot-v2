package crm

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/store"
)

// MyAttendance là phần điểm danh của CHÍNH sinh viên (SRS 6.2 #7): tổng + một trang buổi. KHÔNG `note`, `marked_by`, `version`.
// Lớp chưa điểm danh lần nào → mọi số 0, Sessions rỗng (không lỗi).
type MyAttendance struct {
	Recorded, Present, Late, Excused, Absent int
	Sessions                                 []AttendanceItem
}

// MyAttendance chỉ đọc theo (courseID, studentID) do người gọi lấy từ JWT; không có đường nào nhận sinh viên từ tham số.
func (s *Service) MyAttendance(ctx context.Context, courseID, studentID uuid.UUID, afterAt *time.Time, afterID *uuid.UUID, limit int) (MyAttendance, error) {
	q := store.New(s.Pool)
	var out MyAttendance
	sums, err := q.CrmMyAttendanceSummary(ctx, store.CrmMyAttendanceSummaryParams{CourseID: courseID, StudentID: studentID})
	if err != nil {
		return out, wrap("điểm danh của tôi", err)
	}
	if len(sums) > 0 {
		x := sums[0]
		out.Recorded, out.Present, out.Late, out.Excused, out.Absent = int(x.Recorded), int(x.Present), int(x.Late), int(x.Excused), int(x.Absent)
	}
	rows, err := q.CrmStudentAttendance(ctx, store.CrmStudentAttendanceParams{CourseID: courseID, StudentID: studentID, CurAt: afterAt, CurID: afterID, Lim: int32(limit + 1)}) //nolint:gosec // ≤ 101
	if err != nil {
		return out, wrap("buổi điểm danh của tôi", err)
	}
	out.Sessions = make([]AttendanceItem, len(rows))
	for i, r := range rows {
		out.Sessions[i] = AttendanceItem{SessionID: r.SessionID, SessionNo: int(r.SessionNo), StartsAt: r.StartsAt, Status: string(r.Status)} // không Note
	}
	return out, nil
}

// MyParticipation: số lần phát biểu + tổng điểm cộng ĐÃ GHI (Σ points, decimal, không áp trần) và một trang sự kiện của chính sinh viên.
type MyParticipation struct {
	SpeakCount  int
	PointsTotal decimal.Decimal
	Events      []ParticipationRow
}

func (s *Service) MyParticipation(ctx context.Context, courseID, studentID uuid.UUID, afterAt *time.Time, afterID *uuid.UUID, limit int) (MyParticipation, error) {
	q := store.New(s.Pool)
	sum, err := q.CrmMyParticipationSummary(ctx, store.CrmMyParticipationSummaryParams{CourseID: courseID, StudentID: studentID})
	if err != nil {
		return MyParticipation{}, wrap("phát biểu của tôi", err)
	}
	rows, err := s.ListParticipation(ctx, courseID, &studentID, nil, afterAt, afterID, limit)
	if err != nil {
		return MyParticipation{}, err
	}
	return MyParticipation{SpeakCount: int(sum.SpeakCount), PointsTotal: sum.PointsTotal, Events: rows}, nil
}
