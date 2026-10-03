package course

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/edupilot/backend-go/internal/store"
)

// Buổi học TỐI THIỂU của P2 (SRS 4.8): sinh theo tuần và liệt kê. Tạo / sửa từng buổi bằng giao diện là P5.
const (
	MaxSessionsPerCall = 60
	maxSessionRangeDay = 400 // chặn vòng lặp ngày vô hạn trước khi đếm buổi
)

var hhmmRE = regexp.MustCompile(`^([01][0-9]|2[0-3]):[0-5][0-9]$`)

// ictLoc là Asia/Ho_Chi_Minh (UTC+7, không đổi giờ mùa hè).
func ictLoc() *time.Location { return time.FixedZone("Asia/Ho_Chi_Minh", 7*3600) }

// GenerateInput là thân `POST …/sessions/generate`.
type GenerateInput struct {
	Weekdays       []int // 1 = thứ Hai … 7 = Chủ nhật
	StartTime      string
	EndTime        string
	Room           *string
	From, To       string // YYYY-MM-DD
	ExcludeDates   []string
	FirstSessionNo *int
}

// GenerateResult là phản hồi 201.
type GenerateResult struct {
	Created        int
	FirstSessionNo *int
	LastSessionNo  *int
	Skipped        int
}

func invalidField(field, code, msg string) error {
	return &InvalidError{Field: field, Code: code, Message: msg}
}

// GenerateSessions tạo buổi cho mỗi ngày khớp trong [From, To] trừ ngày loại trừ; session_no nối tiếp số lớn nhất (hoặc FirstSessionNo);
// buổi trùng starts_at bị bỏ qua (chạy lại không tạo thêm). Khoá dòng lớp ⇒ hai lần gọi đồng thời không cấp trùng session_no.
func (s Service) GenerateSessions(ctx context.Context, actor, courseID uuid.UUID, in GenerateInput) (GenerateResult, error) {
	loc := ictLoc()
	slots, err := plan(in, loc)
	if err != nil {
		return GenerateResult{}, err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return GenerateResult{}, fmt.Errorf("course: mở transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	c, err := q.LockCourse(ctx, courseID)
	if errors.Is(err, pgx.ErrNoRows) {
		return GenerateResult{}, ErrNotFound
	}
	if err != nil {
		return GenerateResult{}, fmt.Errorf("course: khoá lớp: %w", err)
	}
	if c.Status != store.CourseStatusACTIVE {
		return GenerateResult{}, ErrArchived
	}
	starts := make([]string, len(slots))
	for i, sl := range slots {
		starts[i] = sl.start.UTC().Format(time.RFC3339)
	}
	have := map[time.Time]bool{}
	if len(starts) > 0 {
		rows, err := q.ExistingSessionStarts(ctx, store.ExistingSessionStartsParams{CourseID: courseID, Starts: starts})
		if err != nil {
			return GenerateResult{}, fmt.Errorf("course: buổi đã có: %w", err)
		}
		for _, t := range rows {
			have[t.UTC()] = true
		}
	}
	var fresh []slot
	for _, sl := range slots {
		if !have[sl.start.UTC()] {
			fresh = append(fresh, sl)
		}
	}
	res := GenerateResult{Skipped: len(slots) - len(fresh)}
	if len(fresh) == 0 {
		return res, tx.Commit(ctx)
	}
	next := 0
	if in.FirstSessionNo != nil {
		next = *in.FirstSessionNo
		taken, err := q.SessionNoInRange(ctx, store.SessionNoInRangeParams{CourseID: courseID, Lo: int32(next), Hi: int32(next + len(fresh) - 1)})
		if err != nil {
			return GenerateResult{}, fmt.Errorf("course: kiểm số buổi: %w", err)
		}
		if taken {
			return GenerateResult{}, invalidField("first_session_no", "SESSION_NO_TAKEN", "Số buổi này đã có trong lớp.")
		}
	} else {
		m, err := q.MaxSessionNo(ctx, courseID)
		if err != nil {
			return GenerateResult{}, fmt.Errorf("course: số buổi lớn nhất: %w", err)
		}
		next = int(m) + 1
	}
	first := next
	for _, sl := range fresh {
		if _, err := q.InsertSession(ctx, store.InsertSessionParams{CourseID: courseID, SessionNo: int32(next), StartsAt: sl.start, EndsAt: sl.end, Room: in.Room}); err != nil {
			return GenerateResult{}, fmt.Errorf("course: tạo buổi học: %w", err)
		}
		next++
	}
	last := next - 1
	res.Created, res.FirstSessionNo, res.LastSessionNo = len(fresh), &first, &last
	if err := s.audit(ctx, q, courseID, actor, courseID.String(), "sessions_generated", nil, map[string]any{"created": res.Created, "first": first, "last": last}); err != nil {
		return GenerateResult{}, err
	}
	if err := emitChanged(ctx, tx, courseID); err != nil { // bước "tạo lịch buổi học" của Hôm nay + dòng thời gian
		return GenerateResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return GenerateResult{}, fmt.Errorf("course: commit buổi học: %w", err)
	}
	return res, nil
}

type slot struct{ start, end time.Time }

// plan kiểm đầu vào và liệt kê các buổi (theo giờ ICT) — thuần hàm, không DB.
func plan(in GenerateInput, loc *time.Location) ([]slot, error) {
	if len(in.Weekdays) == 0 {
		return nil, invalidField("weekdays", "REQUIRED", "Chọn ít nhất một thứ trong tuần.")
	}
	days := map[time.Weekday]bool{}
	for _, d := range in.Weekdays {
		if d < 1 || d > 7 {
			return nil, invalidField("weekdays", "INVALID_WEEKDAY", "Thứ trong tuần từ 1 (thứ Hai) đến 7 (Chủ nhật).")
		}
		days[time.Weekday(d%7)] = true // 7 → Chủ nhật (time.Sunday = 0)
	}
	if !hhmmRE.MatchString(in.StartTime) {
		return nil, invalidField("start_time", "INVALID_TIME", "Giờ bắt đầu dạng HH:MM.")
	}
	if !hhmmRE.MatchString(in.EndTime) || in.EndTime <= in.StartTime {
		return nil, invalidField("end_time", "INVALID_TIME", "Giờ kết thúc dạng HH:MM và sau giờ bắt đầu.")
	}
	if in.Room != nil && len([]rune(*in.Room)) > 40 {
		return nil, invalidField("room", "TOO_LONG", "Phòng tối đa 40 ký tự.")
	}
	if in.FirstSessionNo != nil && *in.FirstSessionNo < 1 {
		return nil, invalidField("first_session_no", "INVALID", "Số buổi bắt đầu từ 1.")
	}
	from, err := time.ParseInLocation(time.DateOnly, in.From, loc)
	if err != nil {
		return nil, invalidField("from", "INVALID_DATE", "Ngày bắt đầu dạng YYYY-MM-DD.")
	}
	to, err := time.ParseInLocation(time.DateOnly, in.To, loc)
	if err != nil || to.Before(from) {
		return nil, invalidField("to", "INVALID_DATE", "Ngày kết thúc dạng YYYY-MM-DD và không trước ngày bắt đầu.")
	}
	if to.Sub(from) > maxSessionRangeDay*24*time.Hour {
		return nil, invalidField("to", "RANGE_TOO_LONG", "Khoảng ngày tối đa 400 ngày.")
	}
	skip := map[string]bool{}
	for _, e := range in.ExcludeDates {
		if _, err := time.Parse(time.DateOnly, e); err != nil {
			return nil, invalidField("exclude_dates", "INVALID_DATE", "Ngày loại trừ dạng YYYY-MM-DD.")
		}
		skip[e] = true
	}
	sh, sm := atoi2(in.StartTime[:2]), atoi2(in.StartTime[3:])
	eh, em := atoi2(in.EndTime[:2]), atoi2(in.EndTime[3:])
	var out []slot
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		if !days[d.Weekday()] || skip[d.Format(time.DateOnly)] {
			continue
		}
		out = append(out, slot{
			start: time.Date(d.Year(), d.Month(), d.Day(), sh, sm, 0, 0, loc),
			end:   time.Date(d.Year(), d.Month(), d.Day(), eh, em, 0, 0, loc),
		})
		if len(out) > MaxSessionsPerCall {
			return nil, invalidField("to", "TOO_MANY_SESSIONS", "Mỗi lần tạo tối đa 60 buổi. Hãy chia nhỏ khoảng ngày.")
		}
	}
	slices.SortFunc(out, func(a, b slot) int { return a.start.Compare(b.start) })
	return out, nil
}

func atoi2(s string) int { return int(s[0]-'0')*10 + int(s[1]-'0') }

// SessionRow là một buổi học trong danh sách.
type SessionRow struct {
	ID               uuid.UUID
	SessionNo        int
	StartsAt, EndsAt time.Time
	Room, Topic      *string
}

// ListSessions trả tối đa limit+1 buổi sau `after` (nil = từ đầu), theo starts_at.
func (s Service) ListSessions(ctx context.Context, courseID uuid.UUID, after *time.Time, limit int) ([]SessionRow, error) {
	rows, err := store.New(s.Pool).ListSessions(ctx, store.ListSessionsParams{CourseID: courseID, After: after, Lim: int32(limit + 1)})
	if err != nil {
		return nil, fmt.Errorf("course: liệt kê buổi học: %w", err)
	}
	out := make([]SessionRow, len(rows))
	for i, r := range rows {
		out[i] = SessionRow{ID: r.ID, SessionNo: int(r.SessionNo), StartsAt: r.StartsAt, EndsAt: r.EndsAt, Room: r.Room, Topic: r.Topic}
	}
	return out, nil
}
