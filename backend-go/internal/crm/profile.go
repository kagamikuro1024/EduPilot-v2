package crm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/store"
)

// ReasonOut là một lý do ở hồ sơ: mã + câu (cùng nguồn với câu nhận định và danh sách).
type ReasonOut struct {
	Code string `json:"code"`
	Text string `json:"text"`
}

// StudyWeek là một tuần của biểu đồ thời gian học (thứ Hai đầu tuần, Asia/Ho_Chi_Minh).
type StudyWeek struct {
	WeekStart string `json:"week_start"`
	Minutes   int    `json:"minutes"`
}

// Profile là hồ sơ 360 (SRS 4.8.3). KHÔNG có điểm (P6), KHÔNG có nội dung chat; `activity` là tuần trọn trước (từ student_risk).
type Profile struct {
	FullName                                 string
	StudentID                                uuid.UUID
	StudentCode                              *string
	JoinedAt                                 time.Time
	Level, Sentence                          string
	Reasons                                  []ReasonOut
	ComputedAt                               *time.Time
	Recorded, Present, Late, Excused, Absent int
	SpeakCount                               int
	PointsTotal                              decimal.Decimal
	Minutes7d                                int
	StudyWeeks                               []StudyWeek
	Tracked                                  bool
	Asks, Posts                              int
}

const profileStudyWeeks = 8

// Profile đọc hồ sơ MỘT sinh viên ACTIVE: trong MỘT giao dịch — ghi `audit_log` (`student_profile`, `<uid>`, `view`) RỒI mới đọc dữ liệu; audit lỗi → rollback + 503 (fail-closed, không trả
// dữ liệu); sinh viên không ACTIVE ở lớp → 404 (giao dịch huỷ nên KHÔNG có dòng audit khi chưa trả dữ liệu). Mỗi LẦN GỌI một dòng (TLR-7).
func (s *Service) Profile(ctx context.Context, actor, courseID, studentID uuid.UUID) (Profile, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return Profile{}, wrap("mở giao dịch hồ sơ", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	if err := audit(ctx, q, "student_profile", courseID, actor, studentID.String(), "view", nil, nil); err != nil {
		return Profile{}, unavailable() // fail-closed: không ghi được vết thì không trả dữ liệu
	}
	row, err := q.CrmProfileGet(ctx, store.CrmProfileGetParams{CourseID: courseID, StudentID: studentID})
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, notFound()
	}
	if err != nil {
		return Profile{}, wrap("đọc hồ sơ", err)
	}
	cw := weekStart(s.now())
	from := cw.AddDate(0, 0, -7*(profileStudyWeeks-1))
	day := func(t time.Time) time.Time { return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC) }
	wk, err := q.CrmProfileStudyWeeks(ctx, store.CrmProfileStudyWeeksParams{CourseID: courseID, StudentID: studentID, FromDay: dateOf(day(from)), ToDay: dateOf(day(cw.AddDate(0, 0, 7)))})
	if err != nil {
		return Profile{}, wrap("thời gian học theo tuần", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Profile{}, wrap("commit hồ sơ", err)
	}
	mins := map[string]int{}
	for _, w := range wk {
		mins[w.WeekStart.Time.Format(time.DateOnly)] = int(w.Minutes)
	}
	p := Profile{StudentID: row.StudentID, FullName: row.FullName, StudentCode: row.StudentCode, JoinedAt: row.JoinedAt, Level: string(row.RiskLevel), ComputedAt: row.RiskComputedAt,
		Recorded: int(row.Recorded), Present: int(row.Present), Late: int(row.Late), Excused: int(row.Excused), Absent: int(row.Absent), SpeakCount: int(row.SpeakCount), PointsTotal: row.PointsTotal,
		Minutes7d: int(row.StudySeconds7d / 60), Tracked: row.Tracked, Asks: int(row.LastWeekAsks), Posts: int(row.LastWeekPosts), Reasons: []ReasonOut{}}
	for i := range profileStudyWeeks {
		k := from.AddDate(0, 0, 7*i).Format(time.DateOnly)
		p.StudyWeeks = append(p.StudyWeeks, StudyWeek{WeekStart: k, Minutes: mins[k]})
	}
	var rs []Reason
	if err := json.Unmarshal(row.Reasons, &rs); err != nil {
		return Profile{}, fmt.Errorf("crm: reasons hỏng: %w", err)
	}
	if p.Level != LevelNone {
		p.Sentence = Sentence(p.Level, rs)
		for _, r := range rs {
			p.Reasons = append(p.Reasons, ReasonOut{Code: r.Code, Text: Phrase(r)})
		}
	}
	return p, nil
}

// AttendanceItem là một dòng tab Chuyên cần: buổi × trạng thái × ghi chú (nhân viên được thấy ghi chú buổi).
type AttendanceItem struct {
	SessionID uuid.UUID
	SessionNo int
	StartsAt  time.Time
	Status    string
	Note      *string
}

// StudentAttendance trả buổi đã có dòng điểm danh của sinh viên, mới nhất trước, phân trang con trỏ (starts_at, id). Sinh viên không ACTIVE → 404.
func (s *Service) StudentAttendance(ctx context.Context, courseID, studentID uuid.UUID, afterAt *time.Time, afterID *uuid.UUID, limit int) ([]AttendanceItem, error) {
	q := store.New(s.Pool)
	if err := s.requireActive(ctx, q, courseID, studentID); err != nil {
		return nil, err
	}
	rows, err := q.CrmStudentAttendance(ctx, store.CrmStudentAttendanceParams{CourseID: courseID, StudentID: studentID, CurAt: afterAt, CurID: afterID, Lim: int32(limit + 1)}) //nolint:gosec // ≤ 101
	if err != nil {
		return nil, wrap("chuyên cần của sinh viên", err)
	}
	out := make([]AttendanceItem, len(rows))
	for i, r := range rows {
		out[i] = AttendanceItem{SessionID: r.SessionID, SessionNo: int(r.SessionNo), StartsAt: r.StartsAt, Status: string(r.Status), Note: r.Note}
	}
	return out, nil
}

// requireActive: sinh viên phải là SINH VIÊN ACTIVE của lớp, nếu không 404 (lộ "không thuộc lớp" như nhau cho mọi trường hợp).
func (s *Service) requireActive(ctx context.Context, q *store.Queries, courseID, studentID uuid.UUID) error {
	ok, err := q.CrmStudentActive(ctx, store.CrmStudentActiveParams{CourseID: courseID, StudentID: studentID})
	if err != nil {
		return wrap("kiểm sinh viên", err)
	}
	if !ok {
		return notFound()
	}
	return nil
}
