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
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/edupilot/backend-go/internal/store"
)

// Lịch buổi học (FEAT-crm-360 SRS 4.2): sinh hàng loạt có xem trước (P2 + US-P5-01), sửa / huỷ từng buổi.
const (
	MaxSessionsPerCall = 60
	maxSessionRangeDay = 400 // chặn vòng lặp ngày vô hạn trước khi đếm buổi
)

// Lý do một ngày bị bỏ qua khi sinh lịch.
const (
	SkipExcluded  = "EXCLUDED"
	SkipDuplicate = "DUPLICATE"
)

// Lỗi của buổi học (handler ánh xạ sang mã API).
var (
	ErrSessionDuplicate     = errors.New("course: đã có buổi khác bắt đầu cùng giờ")
	ErrSessionHasAttendance = errors.New("course: buổi đã có điểm danh")
)

// SessionVersionConflictError: sai version; Current là bản hiện hành (409 VERSION_CONFLICT).
type SessionVersionConflictError struct{ Current SessionRow }

func (e *SessionVersionConflictError) Error() string { return "course: sai version của buổi học" }

var hhmmRE = regexp.MustCompile(`^([01][0-9]|2[0-3]):[0-5][0-9]$`)

// ictLoc là Asia/Ho_Chi_Minh (UTC+7, không đổi giờ mùa hè).
func ictLoc() *time.Location { return time.FixedZone("Asia/Ho_Chi_Minh", 7*3600) }

// GenerateInput là thân `POST …/sessions/generate`. DryRun = xem trước: giao dịch chỉ đọc, không ghi gì.
type GenerateInput struct {
	Weekdays       []int // 1 = thứ Hai … 7 = Chủ nhật
	StartTime      string
	EndTime        string
	Room           *string
	From, To       string // YYYY-MM-DD
	ExcludeDates   []string
	FirstSessionNo *int
	DryRun         bool
}

// PlannedSession là một buổi sẽ được tạo (xem trước) — cùng số buổi và giờ với lần chạy thật ngay sau đó.
type PlannedSession struct {
	SessionNo        int
	StartsAt, EndsAt time.Time
}

// SkipItem là một ngày bị bỏ qua (Date theo giờ ICT, YYYY-MM-DD).
type SkipItem struct{ Date, Reason string }

// GenerateResult là kết quả sinh lịch. Skipped = số buổi trùng đã có; SkippedItems liệt kê cả ngày loại trừ.
type GenerateResult struct {
	Created        int
	FirstSessionNo *int
	LastSessionNo  *int
	Skipped        int
	WouldCreate    []PlannedSession
	SkippedItems   []SkipItem
}

func invalidField(field, code, msg string) error {
	return &InvalidError{Field: field, Code: code, Message: msg}
}

// GenerateSessions tạo buổi cho mỗi ngày khớp trong [From, To] trừ ngày loại trừ; session_no nối tiếp số lớn nhất (hoặc FirstSessionNo);
// buổi trùng starts_at bị bỏ qua (chạy lại không tạo thêm). Khoá dòng lớp ⇒ hai lần gọi đồng thời không cấp trùng session_no.
// DryRun: giao dịch chỉ đọc (không khoá, không audit, không outbox) và trả đúng tập sẽ tạo.
func (s Service) GenerateSessions(ctx context.Context, actor, courseID uuid.UUID, in GenerateInput) (GenerateResult, error) {
	loc := ictLoc()
	p, err := plan(in, loc)
	if err != nil {
		return GenerateResult{}, err
	}
	opts := pgx.TxOptions{}
	if in.DryRun {
		opts.AccessMode = pgx.ReadOnly
	}
	tx, err := s.Pool.BeginTx(ctx, opts)
	if err != nil {
		return GenerateResult{}, fmt.Errorf("course: mở transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	load := q.LockCourse
	if in.DryRun {
		load = q.GetCourse // giao dịch chỉ đọc không `FOR UPDATE` được
	}
	c, err := load(ctx, courseID)
	if errors.Is(err, pgx.ErrNoRows) {
		return GenerateResult{}, ErrNotFound
	}
	if err != nil {
		return GenerateResult{}, fmt.Errorf("course: đọc lớp: %w", err)
	}
	if c.Status != store.CourseStatusACTIVE {
		return GenerateResult{}, ErrArchived
	}
	starts := make([]string, len(p.slots))
	for i, sl := range p.slots {
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
	skipped := slices.Clone(p.excluded)
	for _, sl := range p.slots {
		if have[sl.start.UTC()] {
			skipped = append(skipped, SkipItem{Date: sl.start.In(loc).Format(time.DateOnly), Reason: SkipDuplicate})
			continue
		}
		fresh = append(fresh, sl)
	}
	slices.SortFunc(skipped, func(a, b SkipItem) int {
		if a.Date != b.Date {
			return cmpStr(a.Date, b.Date)
		}
		return cmpStr(a.Reason, b.Reason)
	})
	res := GenerateResult{Skipped: len(p.slots) - len(fresh), SkippedItems: skipped, WouldCreate: []PlannedSession{}}
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
		next = int(m) + 1 // xoá buổi cuối thì số đó được cấp lại; khoảng trống ở giữa không lấp (TLR-4)
	}
	first := next
	for _, sl := range fresh {
		res.WouldCreate = append(res.WouldCreate, PlannedSession{SessionNo: next, StartsAt: sl.start, EndsAt: sl.end})
		if !in.DryRun {
			if _, err := q.InsertSession(ctx, store.InsertSessionParams{CourseID: courseID, SessionNo: int32(next), StartsAt: sl.start, EndsAt: sl.end, Room: in.Room}); err != nil {
				return GenerateResult{}, fmt.Errorf("course: tạo buổi học: %w", err)
			}
		}
		next++
	}
	if in.DryRun {
		return res, nil
	}
	last := next - 1
	res.Created, res.FirstSessionNo, res.LastSessionNo = len(fresh), &first, &last
	res.WouldCreate = nil
	if err := s.auditEntity(ctx, q, "class_sessions", courseID, actor, courseID.String(), "generate", nil, map[string]any{"created": res.Created, "first": first, "last": last}); err != nil {
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

func cmpStr(a, b string) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

type slot struct{ start, end time.Time }

// planned là kết quả của plan: các buổi và các ngày loại trừ thật sự trúng một buổi (ngày loại trừ ngoài khoảng / lệch thứ không liệt kê).
type planned struct {
	slots    []slot
	excluded []SkipItem
}

// plan kiểm đầu vào và liệt kê các buổi (theo giờ ICT) — thuần hàm, không DB.
func plan(in GenerateInput, loc *time.Location) (planned, error) {
	if len(in.Weekdays) == 0 {
		return planned{}, invalidField("weekdays", "REQUIRED", "Chọn ít nhất một thứ trong tuần.")
	}
	days := map[time.Weekday]bool{}
	for _, d := range in.Weekdays {
		if d < 1 || d > 7 {
			return planned{}, invalidField("weekdays", "INVALID_WEEKDAY", "Thứ trong tuần từ 1 (thứ Hai) đến 7 (Chủ nhật).")
		}
		days[time.Weekday(d%7)] = true // 7 → Chủ nhật (time.Sunday = 0)
	}
	if !hhmmRE.MatchString(in.StartTime) {
		return planned{}, invalidField("start_time", "INVALID_TIME", "Giờ bắt đầu dạng HH:MM.")
	}
	if !hhmmRE.MatchString(in.EndTime) || in.EndTime <= in.StartTime {
		return planned{}, invalidField("end_time", "INVALID_TIME", "Giờ kết thúc dạng HH:MM và sau giờ bắt đầu.")
	}
	if in.Room != nil && len([]rune(*in.Room)) > 40 {
		return planned{}, invalidField("room", "TOO_LONG", "Phòng tối đa 40 ký tự.")
	}
	if in.FirstSessionNo != nil && *in.FirstSessionNo < 1 {
		return planned{}, invalidField("first_session_no", "INVALID", "Số buổi bắt đầu từ 1.")
	}
	from, err := time.ParseInLocation(time.DateOnly, in.From, loc)
	if err != nil {
		return planned{}, invalidField("from", "INVALID_DATE", "Ngày bắt đầu dạng YYYY-MM-DD.")
	}
	to, err := time.ParseInLocation(time.DateOnly, in.To, loc)
	if err != nil || to.Before(from) {
		return planned{}, invalidField("to", "INVALID_DATE", "Ngày kết thúc dạng YYYY-MM-DD và không trước ngày bắt đầu.")
	}
	if to.Sub(from) > maxSessionRangeDay*24*time.Hour {
		return planned{}, invalidField("to", "RANGE_TOO_LONG", "Khoảng ngày tối đa 400 ngày.")
	}
	skip := map[string]bool{}
	for _, e := range in.ExcludeDates {
		if _, err := time.Parse(time.DateOnly, e); err != nil {
			return planned{}, invalidField("exclude_dates", "INVALID_DATE", "Ngày loại trừ dạng YYYY-MM-DD.")
		}
		skip[e] = true
	}
	sh, sm := atoi2(in.StartTime[:2]), atoi2(in.StartTime[3:])
	eh, em := atoi2(in.EndTime[:2]), atoi2(in.EndTime[3:])
	var out planned
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		if !days[d.Weekday()] {
			continue
		}
		if ds := d.Format(time.DateOnly); skip[ds] {
			out.excluded = append(out.excluded, SkipItem{Date: ds, Reason: SkipExcluded})
			continue
		}
		out.slots = append(out.slots, slot{
			start: time.Date(d.Year(), d.Month(), d.Day(), sh, sm, 0, 0, loc),
			end:   time.Date(d.Year(), d.Month(), d.Day(), eh, em, 0, 0, loc),
		})
	}
	if len(out.slots) > MaxSessionsPerCall {
		return planned{}, invalidField("to", "TOO_MANY_SESSIONS",
			fmt.Sprintf("Khoảng ngày này có %d buổi, mỗi lần tạo tối đa %d. Hãy chia nhỏ khoảng ngày.", len(out.slots), MaxSessionsPerCall))
	}
	slices.SortFunc(out.slots, func(a, b slot) int { return a.start.Compare(b.start) })
	return out, nil
}

func atoi2(s string) int { return int(s[0]-'0')*10 + int(s[1]-'0') }

// SessionRow là một buổi học trong danh sách. AttendanceDoneAt / HasAttendance chỉ có nghĩa với nhân viên.
type SessionRow struct {
	ID               uuid.UUID
	SessionNo        int
	StartsAt, EndsAt time.Time
	Room, Topic      *string
	Version          int
	AttendanceDoneAt *time.Time
	HasAttendance    bool
}

// ListSessions trả tối đa limit+1 buổi sau `after` (nil = từ đầu), theo starts_at.
func (s Service) ListSessions(ctx context.Context, courseID uuid.UUID, after *time.Time, limit int) ([]SessionRow, error) {
	rows, err := store.New(s.Pool).ListSessions(ctx, store.ListSessionsParams{CourseID: courseID, After: after, Lim: int32(limit + 1)})
	if err != nil {
		return nil, fmt.Errorf("course: liệt kê buổi học: %w", err)
	}
	out := make([]SessionRow, len(rows))
	for i, r := range rows {
		out[i] = SessionRow{ID: r.ID, SessionNo: int(r.SessionNo), StartsAt: r.StartsAt, EndsAt: r.EndsAt, Room: r.Room, Topic: r.Topic,
			Version: int(r.Version), AttendanceDoneAt: r.AttendanceDoneAt, HasAttendance: r.HasAttendance}
	}
	return out, nil
}

// SessionInput là thân `PUT …/sessions/{sid}`: thay toàn bộ giờ / phòng / chủ đề (bỏ trống = xoá). `session_no` không đổi qua đây.
type SessionInput struct {
	StartsAt, EndsAt time.Time
	Room, Topic      *string
	Version          int
}

func blankToNil(p *string) *string {
	if p == nil || *p == "" {
		return nil
	}
	return p
}

// UpdateSession sửa một buổi (giảng viên / TA). Sai version ⇒ *SessionVersionConflictError; trùng giờ bắt đầu buổi khác ⇒ ErrSessionDuplicate.
func (s Service) UpdateSession(ctx context.Context, actor, courseID, sid uuid.UUID, in SessionInput) (SessionRow, error) {
	in.Room, in.Topic = blankToNil(in.Room), blankToNil(in.Topic)
	if !in.EndsAt.After(in.StartsAt) {
		return SessionRow{}, invalidField("ends_at", "INVALID_TIME", "Giờ kết thúc phải sau giờ bắt đầu.")
	}
	if in.Room != nil && len([]rune(*in.Room)) > 40 {
		return SessionRow{}, invalidField("room", "TOO_LONG", "Phòng tối đa 40 ký tự.")
	}
	if in.Topic != nil && len([]rune(*in.Topic)) > 200 {
		return SessionRow{}, invalidField("topic", "TOO_LONG", "Chủ đề tối đa 200 ký tự.")
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return SessionRow{}, fmt.Errorf("course: mở transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	cur, err := s.lockSession(ctx, q, courseID, sid)
	if err != nil {
		return SessionRow{}, err
	}
	has, err := q.SessionHasAttendance(ctx, store.SessionHasAttendanceParams{CourseID: courseID, SessionID: sid})
	if err != nil {
		return SessionRow{}, fmt.Errorf("course: kiểm điểm danh của buổi: %w", err)
	}
	if int(cur.Version) != in.Version {
		return SessionRow{}, &SessionVersionConflictError{Current: rowOfSession(cur, has)}
	}
	next, err := q.UpdateSession(ctx, store.UpdateSessionParams{CourseID: courseID, ID: sid, Version: cur.Version, StartsAt: in.StartsAt, EndsAt: in.EndsAt, Room: in.Room, Topic: in.Topic})
	if isUnique(err, "class_sessions_course_starts_key") {
		return SessionRow{}, ErrSessionDuplicate
	}
	if err != nil {
		return SessionRow{}, fmt.Errorf("course: sửa buổi học: %w", err)
	}
	snap := func(r store.ClassSession) map[string]any {
		return map[string]any{"session_no": r.SessionNo, "starts_at": r.StartsAt.UTC().Format(time.RFC3339), "ends_at": r.EndsAt.UTC().Format(time.RFC3339), "room": r.Room, "topic": r.Topic}
	}
	if err := s.auditEntity(ctx, q, "class_sessions", courseID, actor, sid.String(), "update", snap(cur), snap(next)); err != nil {
		return SessionRow{}, err
	}
	if err := emitChanged(ctx, tx, courseID); err != nil {
		return SessionRow{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return SessionRow{}, fmt.Errorf("course: commit sửa buổi học: %w", err)
	}
	return rowOfSession(next, has), nil
}

// DeleteSession huỷ một buổi chưa có điểm danh ("nghỉ" = xoá buổi, Q9). Đã có điểm danh ⇒ ErrSessionHasAttendance.
func (s Service) DeleteSession(ctx context.Context, actor, courseID, sid uuid.UUID) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("course: mở transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	cur, err := s.lockSession(ctx, q, courseID, sid)
	if err != nil {
		return err
	}
	has, err := q.SessionHasAttendance(ctx, store.SessionHasAttendanceParams{CourseID: courseID, SessionID: sid})
	if err != nil {
		return fmt.Errorf("course: kiểm điểm danh của buổi: %w", err)
	}
	if has {
		return ErrSessionHasAttendance
	}
	if _, err := q.DeleteSession(ctx, store.DeleteSessionParams{CourseID: courseID, ID: sid}); err != nil {
		var pg *pgconn.PgError
		if errors.As(err, &pg) && pg.Code == "23503" { // dòng điểm danh chèn sát giờ: khoá ngoại là lưới an toàn cuối
			return ErrSessionHasAttendance
		}
		return fmt.Errorf("course: xoá buổi học: %w", err)
	}
	before := map[string]any{"session_no": cur.SessionNo, "starts_at": cur.StartsAt.UTC().Format(time.RFC3339)}
	if err := s.auditEntity(ctx, q, "class_sessions", courseID, actor, sid.String(), "delete", before, nil); err != nil {
		return err
	}
	if err := emitChanged(ctx, tx, courseID); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("course: commit xoá buổi học: %w", err)
	}
	return nil
}

// lockSession khoá lớp (phải ACTIVE) rồi khoá dòng buổi; buổi của lớp khác ⇒ ErrNotFound.
func (s Service) lockSession(ctx context.Context, q *store.Queries, courseID, sid uuid.UUID) (store.ClassSession, error) {
	c, err := q.LockCourse(ctx, courseID)
	if errors.Is(err, pgx.ErrNoRows) {
		return store.ClassSession{}, ErrNotFound
	}
	if err != nil {
		return store.ClassSession{}, fmt.Errorf("course: khoá lớp: %w", err)
	}
	if c.Status != store.CourseStatusACTIVE {
		return store.ClassSession{}, ErrArchived
	}
	cur, err := q.LockSession(ctx, store.LockSessionParams{CourseID: courseID, ID: sid})
	if errors.Is(err, pgx.ErrNoRows) {
		return store.ClassSession{}, ErrNotFound
	}
	if err != nil {
		return store.ClassSession{}, fmt.Errorf("course: khoá buổi học: %w", err)
	}
	return cur, nil
}

func rowOfSession(r store.ClassSession, has bool) SessionRow {
	return SessionRow{ID: r.ID, SessionNo: int(r.SessionNo), StartsAt: r.StartsAt, EndsAt: r.EndsAt, Room: r.Room, Topic: r.Topic,
		Version: int(r.Version), AttendanceDoneAt: r.AttendanceDoneAt, HasAttendance: has}
}
