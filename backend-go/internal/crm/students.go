package crm

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/store"
)

// ErrInvalidCursor: con trỏ danh sách sinh viên không giải mã được → 422 INVALID_CURSOR.
var ErrInvalidCursor = errors.New("crm: cursor không hợp lệ")

// StudentCursor trỏ tới dòng CUỐI trang trước theo khoá sắp xếp (risk_rank giảm, vn_fold(tên), id) — SRS 4.8.2.
type StudentCursor struct {
	Rank int32
	Fold string
	ID   uuid.UUID
}

type cursorWire struct {
	V int    `json:"v"`
	R int32  `json:"r"`
	F string `json:"f"`
	I string `json:"i"`
}

// Encode dựng chuỗi base64url không đệm.
func (c StudentCursor) Encode() string {
	raw, _ := json.Marshal(cursorWire{V: 1, R: c.Rank, F: c.Fold, I: c.ID.String()}) // struct thuần: không lỗi
	return base64.RawURLEncoding.EncodeToString(raw)
}

// ParseStudentCursor giải mã; sai định dạng, `v` lạ hoặc hạng ngoài 0..2 → ErrInvalidCursor.
func ParseStudentCursor(s string) (StudentCursor, error) {
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return StudentCursor{}, ErrInvalidCursor
	}
	var w cursorWire
	if err := json.Unmarshal(raw, &w); err != nil || w.V != 1 || w.R < 0 || w.R > 2 {
		return StudentCursor{}, ErrInvalidCursor
	}
	id, err := uuid.Parse(w.I)
	if err != nil {
		return StudentCursor{}, ErrInvalidCursor
	}
	return StudentCursor{Rank: w.R, Fold: w.F, ID: id}, nil
}

// StudentsQuery là tham số `GET …/students`. Filter là giao của các cờ đã bật.
type StudentsQuery struct {
	Q                           string
	Attention, Absent, Inactive bool
	Cursor                      *StudentCursor
	Limit                       int // 1..100
}

// StudentRow là một dòng danh sách. KHÔNG có lý do bằng lời, KHÔNG có điểm (P6).
type StudentRow struct {
	ID                                       uuid.UUID
	FullName                                 string
	StudentCode                              *string
	Recorded, Present, Late, Excused, Absent int
	SpeakCount                               int
	PointsTotal                              decimal.Decimal
	StudyMinutes7d                           int
	Tracked                                  bool
	AsksLastWeek, PostsLastWeek              int
	RiskLevel                                string
	RiskComputedAt                           *time.Time
}

// StudentsPage là một trang + tổng số khớp bộ lọc (không phụ thuộc con trỏ).
type StudentsPage struct {
	Items      []StudentRow
	NextCursor *string
	Total      int
}

// ListStudents trả một trang sinh viên ACTIVE của lớp (SRS 4.8.2 / 4.8.6): đọc `v_student_360` + `student_risk`, không chạm chat_messages.
// Ngưỡng `absent_alert` / `study_min_minutes` lấy từ cài đặt lớp.
func (s *Service) ListStudents(ctx context.Context, courseID uuid.UUID, in StudentsQuery) (StudentsPage, error) {
	q := store.New(s.Pool)
	crs, err := q.CrmRiskCourse(ctx, courseID)
	if err != nil {
		return StudentsPage{}, wrap("đọc lớp", err)
	}
	th := ThresholdsFrom(crs.Settings)
	cnt, err := q.CrmStudentsCount(ctx, store.CrmStudentsCountParams{CourseID: courseID, FAttention: in.Attention, FAbsent: in.Absent, AbsentAlert: int32(th.AbsentAlert), //nolint:gosec // 1..20
		FInactive: in.Inactive, StudyMinMinutes: int32(th.StudyMinMinutes), Q: in.Q}) //nolint:gosec // 0..600
	if err != nil {
		return StudentsPage{}, wrap("đếm sinh viên", err)
	}
	p := store.CrmStudentsListParams{CourseID: courseID, FAttention: in.Attention, FAbsent: in.Absent, AbsentAlert: int32(th.AbsentAlert), //nolint:gosec // 1..20
		FInactive: in.Inactive, StudyMinMinutes: int32(th.StudyMinMinutes), Q: in.Q, Lim: int32(in.Limit + 1)} //nolint:gosec // ≤ 101
	if c := in.Cursor; c != nil {
		p.CurRank, p.CurFold, p.CurID = &c.Rank, &c.Fold, &c.ID
	}
	rows, err := q.CrmStudentsList(ctx, p)
	if err != nil {
		return StudentsPage{}, wrap("liệt kê sinh viên", err)
	}
	out := StudentsPage{Items: make([]StudentRow, 0, min(len(rows), in.Limit)), Total: int(cnt)}
	for i, r := range rows {
		if i == in.Limit {
			c := StudentCursor{Rank: rows[i-1].RiskRank, Fold: rows[i-1].NameFold, ID: rows[i-1].StudentID}.Encode()
			out.NextCursor = &c
			break
		}
		x := StudentRow{ID: r.StudentID, FullName: r.FullName, StudentCode: r.StudentCode, Recorded: int(r.Recorded), Present: int(r.Present), Late: int(r.Late), Excused: int(r.Excused), Absent: int(r.Absent),
			SpeakCount: int(r.SpeakCount), PointsTotal: r.PointsTotal, StudyMinutes7d: int(r.StudySeconds7d / 60), Tracked: r.Tracked, AsksLastWeek: int(r.LastWeekAsks), PostsLastWeek: int(r.LastWeekPosts),
			RiskLevel: string(r.RiskLevel), RiskComputedAt: r.RiskComputedAt}
		out.Items = append(out.Items, x)
	}
	return out, nil
}
