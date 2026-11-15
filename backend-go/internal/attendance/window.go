package attendance

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/store"
)

// WindowOut là một cửa sổ điểm danh cho Staff.
type WindowOut struct {
	ID            uuid.UUID  `json:"id"`
	SessionID     uuid.UUID  `json:"-"`
	Round         int        `json:"round"`
	RotateSeconds int        `json:"-"`
	OpenedAt      time.Time  `json:"opened_at"`
	ClosesAt      time.Time  `json:"closes_at"`
	ClosedAt      *time.Time `json:"closed_at"`
	Headcount     *int       `json:"headcount"`
	CheckedIn     *int       `json:"checked_in"`
	OverHeadcount *bool      `json:"over_headcount"`
}

func windowOut(w store.CheckinWindow) WindowOut {
	out := WindowOut{ID: w.ID, SessionID: w.SessionID, Round: int(w.Round), RotateSeconds: int(w.RotateSeconds), OpenedAt: w.OpenedAt, ClosesAt: w.ClosesAt, ClosedAt: w.ClosedAt}
	if w.Headcount != nil {
		h := int(*w.Headcount)
		out.Headcount = &h
	}
	return out
}

// HeadcountTolerance = max(2, ⌈5 % × headcount⌉) (SRS 4.1).
func HeadcountTolerance(headcount int) int {
	t := (headcount*5 + 99) / 100
	if t < 2 {
		return 2
	}
	return t
}

// OverHeadcount: số điểm danh nhiều hơn số người đếm quá dung sai (tính khi đọc, không lưu).
func OverHeadcount(checkedIn int, headcount *int) bool {
	return headcount != nil && checkedIn > *headcount+HeadcountTolerance(*headcount)
}

// OpenWindow mở cửa sổ kế tiếp của buổi (FR-2). Một giao dịch: khoá buổi → (đóng cửa sổ trước đã quá giờ) → vòng kế tiếp → INSERT.
// `Idempotency-Key` do middleware của route xử lý.
func (s *Service) OpenWindow(ctx context.Context, actor, courseID, sid uuid.UUID, minutes *int) (WindowOut, error) {
	if !s.Cfg.Enabled {
		return WindowOut{}, conflict(apierr.SelfCheckinDisabled)
	}
	if minutes != nil && (*minutes < 1 || *minutes > s.Cfg.WindowMinutesMax) {
		return WindowOut{}, invalid("minutes", "range", fmt.Sprintf("Thời lượng từ 1 đến %d phút.", s.Cfg.WindowMinutesMax))
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return WindowOut{}, wrap("mở giao dịch", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	c, pol, err := s.course(ctx, q, courseID)
	if err != nil {
		return WindowOut{}, err
	}
	if c.Status != store.CourseStatusACTIVE {
		return WindowOut{}, apierr.New(http.StatusConflict, apierr.CourseArchived)
	}
	if !pol.SelfCheckin {
		return WindowOut{}, conflict(apierr.SelfCheckinDisabled)
	}
	ses, err := q.LockSession(ctx, store.LockSessionParams{CourseID: courseID, ID: sid})
	if err != nil {
		if isNoRows(err) {
			return WindowOut{}, notFound()
		}
		return WindowOut{}, wrap("khoá buổi", err)
	}
	now := s.now()
	open, err := q.AttWindowOpenForSession(ctx, store.AttWindowOpenForSessionParams{CourseID: courseID, SessionID: sid})
	switch {
	case err == nil && !open.ClosesAt.After(now): // đã quá giờ mà chưa ai đóng: đóng nó ngay trong giao dịch này
		if _, err := s.closeLocked(ctx, tx, q, ses.StartsAt, open, nil, pol); err != nil {
			return WindowOut{}, err
		}
	case err == nil:
		return WindowOut{}, conflict(apierr.WindowAlreadyOpen).WithDetails(map[string]string{"window_id": open.ID.String()})
	case !isNoRows(err):
		return WindowOut{}, wrap("cửa sổ đang mở", err)
	}
	maxRound, err := q.AttWindowMaxRound(ctx, store.AttWindowMaxRoundParams{CourseID: courseID, SessionID: sid})
	if err != nil {
		return WindowOut{}, wrap("vòng lớn nhất", err)
	}
	round := int(maxRound) + 1
	if round > s.Cfg.MaxRounds {
		return WindowOut{}, conflict(apierr.RoundLimit)
	}
	mins := s.Cfg.WindowMinutesR1
	if round >= 2 {
		mins = s.Cfg.WindowMinutesR2
	}
	if minutes != nil {
		mins = *minutes
	}
	w, err := q.AttWindowInsert(ctx, store.AttWindowInsertParams{CourseID: courseID, SessionID: sid, Round: int16(round), RotateSeconds: int16(pol.RotateSeconds), OpenedBy: actor, OpenedAt: now, ClosesAt: now.Add(time.Duration(mins) * time.Minute)})
	if err != nil {
		if strings.Contains(err.Error(), "checkin_windows_one_open") { // đua hai gateway: chỉ mục duy nhất một phần
			return WindowOut{}, conflict(apierr.WindowAlreadyOpen)
		}
		return WindowOut{}, wrap("mở cửa sổ", err)
	}
	if err := audit(ctx, q, "checkin_window", courseID, &actor, w.ID.String(), "checkin_window.open", nil, map[string]any{"round": round, "minutes": mins}); err != nil {
		return WindowOut{}, err
	} // vòng ≥ 2: KHÔNG outbox / thông báo nào tới sinh viên (FR-2)
	if err := tx.Commit(ctx); err != nil {
		return WindowOut{}, wrap("commit", err)
	}
	return windowOut(w), nil
}

// ListWindows: tối đa 3 cửa sổ của buổi (không cursor), kèm số đã điểm danh và `over_headcount`.
func (s *Service) ListWindows(ctx context.Context, courseID, sid uuid.UUID) ([]WindowOut, error) {
	q := store.New(s.Pool)
	if _, err := q.AttSessionExists(ctx, store.AttSessionExistsParams{CourseID: courseID, ID: sid}); err != nil {
		if isNoRows(err) {
			return nil, notFound()
		}
		return nil, wrap("đọc buổi", err)
	}
	ws, err := q.AttWindowsBySession(ctx, store.AttWindowsBySessionParams{CourseID: courseID, SessionID: sid})
	if err != nil {
		return nil, wrap("liệt kê cửa sổ", err)
	}
	sc, err := q.AttSessionCheckinStudents(ctx, store.AttSessionCheckinStudentsParams{CourseID: courseID, SessionID: sid})
	if err != nil {
		return nil, wrap("đếm sinh viên", err)
	}
	out := make([]WindowOut, 0, len(ws))
	for _, w := range ws {
		o := windowOut(w)
		n, err := q.AttWindowCheckedIn(ctx, store.AttWindowCheckedInParams{CourseID: courseID, WindowID: w.ID})
		if err != nil {
			return nil, wrap("đếm lượt", err)
		}
		ci := int(n)
		o.CheckedIn = &ci
		over := OverHeadcount(int(sc), o.Headcount)
		o.OverHeadcount = &over
		out = append(out, o)
	}
	return out, nil
}

// DisplayOut là một bước của màn chiếu (FR-4). `short_code` đã nhóm 3 + 3.
type DisplayOut struct {
	QRURL         string    `json:"qr_url"`
	ShortCode     string    `json:"short_code"`
	StepExpiresAt time.Time `json:"step_expires_at"`
	ServerTime    time.Time `json:"server_time"`
	RotateSeconds int       `json:"rotate_seconds"`
	Round         int       `json:"round"`
	CheckedIn     int       `json:"checked_in"`
	Headcount     *int      `json:"headcount"`
	ClosesAt      time.Time `json:"closes_at"`
}

// Display trả mã của bước hiện tại. Hạn mức DISPLAY_RATE_PER_MIN / người; cửa sổ đã đóng / quá hạn → 409 WINDOW_CLOSED. publicURL = APP_PUBLIC_URL.
func (s *Service) Display(ctx context.Context, actor, courseID, wid uuid.UUID, publicURL string) (DisplayOut, error) {
	if err := s.rate(ctx, "display", actor, s.Cfg.DisplayRatePerMin); err != nil {
		return DisplayOut{}, err
	}
	q := store.New(s.Pool)
	w, err := q.AttWindowByID(ctx, store.AttWindowByIDParams{CourseID: courseID, ID: wid})
	if err != nil {
		if isNoRows(err) {
			return DisplayOut{}, notFound()
		}
		return DisplayOut{}, wrap("đọc cửa sổ", err)
	}
	now := s.now()
	if w.ClosedAt != nil || !w.ClosesAt.After(now) {
		return DisplayOut{}, conflict(apierr.WindowClosed)
	}
	rot := int(w.RotateSeconds)
	step := Step(now.Unix(), rot)
	n, err := q.AttWindowCheckedIn(ctx, store.AttWindowCheckedInParams{CourseID: courseID, WindowID: wid})
	if err != nil {
		return DisplayOut{}, wrap("đếm lượt", err)
	}
	out := DisplayOut{
		QRURL:         strings.TrimRight(publicURL, "/") + "/checkin?c=" + courseID.String() + "#" + MakeToken(s.Cfg.TokenKey, wid, step),
		ShortCode:     FormatShortCode(ShortCode(s.Cfg.TokenKey, wid, step, s.Cfg.CodeLen)),
		StepExpiresAt: time.Unix(StepExpiresAt(step, rot), 0).UTC(),
		ServerTime:    now.UTC(),
		RotateSeconds: rot,
		Round:         int(w.Round),
		CheckedIn:     int(n),
		ClosesAt:      w.ClosesAt,
	}
	if w.Headcount != nil {
		h := int(*w.Headcount)
		out.Headcount = &h
	}
	return out, nil
}

// SetHeadcount đặt số người đếm bằng mắt (0–2000); null xoá.
func (s *Service) SetHeadcount(ctx context.Context, actor, courseID, wid uuid.UUID, headcount *int) (WindowOut, error) {
	if headcount != nil && (*headcount < 0 || *headcount > 2000) {
		return WindowOut{}, invalid("headcount", "range", "Số người từ 0 đến 2000.")
	}
	var h *int32
	if headcount != nil {
		v := int32(*headcount)
		h = &v
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return WindowOut{}, wrap("mở giao dịch", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	if err := s.requireActive(ctx, q, courseID); err != nil {
		return WindowOut{}, err
	}
	old, err := q.AttWindowByID(ctx, store.AttWindowByIDParams{CourseID: courseID, ID: wid})
	if err != nil {
		if isNoRows(err) {
			return WindowOut{}, notFound()
		}
		return WindowOut{}, wrap("đọc cửa sổ", err)
	}
	w, err := q.AttWindowSetHeadcount(ctx, store.AttWindowSetHeadcountParams{CourseID: courseID, ID: wid, Headcount: h})
	if err != nil {
		return WindowOut{}, wrap("đặt số đầu người", err)
	}
	if err := audit(ctx, q, "checkin_window", courseID, &actor, wid.String(), "checkin_window.headcount", map[string]any{"headcount": old.Headcount}, map[string]any{"headcount": w.Headcount}); err != nil {
		return WindowOut{}, err
	}
	o := windowOut(w)
	n, err := q.AttWindowCheckedIn(ctx, store.AttWindowCheckedInParams{CourseID: courseID, WindowID: wid})
	if err != nil {
		return WindowOut{}, wrap("đếm lượt", err)
	}
	ci := int(n)
	o.CheckedIn = &ci
	sc, err := q.AttSessionCheckinStudents(ctx, store.AttSessionCheckinStudentsParams{CourseID: courseID, SessionID: w.SessionID})
	if err != nil {
		return WindowOut{}, wrap("đếm sinh viên", err)
	}
	over := OverHeadcount(int(sc), o.Headcount)
	o.OverHeadcount = &over
	if err := tx.Commit(ctx); err != nil {
		return WindowOut{}, wrap("commit", err)
	}
	return o, nil
}

// CloseResult là kết quả đóng cửa sổ (FR-6).
type CloseResult struct {
	Window          WindowOut `json:"window"`
	Written         Written   `json:"written"`
	Unchecked       int       `json:"unchecked"`
	SkippedExisting int       `json:"skipped_existing"`
}

// Written là số dòng điểm danh vừa ghi.
type Written struct {
	Present int `json:"present"`
	Late    int `json:"late"`
}

// Close đóng cửa sổ và ghi attendance_records (FR-6). actor nil = việc nền (`attendance.sweep`). Khoá: buổi → cửa sổ (cùng thứ tự với
// PUT …/attendance). Đóng lần hai trả kết quả đã lưu: không ghi, không audit, không outbox.
func (s *Service) Close(ctx context.Context, actor *uuid.UUID, courseID, wid uuid.UUID) (CloseResult, error) {
	q0 := store.New(s.Pool)
	w0, err := q0.AttWindowByID(ctx, store.AttWindowByIDParams{CourseID: courseID, ID: wid})
	if err != nil {
		if isNoRows(err) {
			return CloseResult{}, notFound()
		}
		return CloseResult{}, wrap("đọc cửa sổ", err)
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return CloseResult{}, wrap("mở giao dịch", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	c, pol, err := s.course(ctx, q, courseID)
	if err != nil {
		return CloseResult{}, err
	}
	if c.Status != store.CourseStatusACTIVE && w0.ClosedAt == nil {
		return CloseResult{}, apierr.New(http.StatusConflict, apierr.CourseArchived)
	}
	ses, err := q.LockSession(ctx, store.LockSessionParams{CourseID: courseID, ID: w0.SessionID})
	if err != nil {
		return CloseResult{}, wrap("khoá buổi", err)
	}
	w, err := q.AttWindowLock(ctx, store.AttWindowLockParams{CourseID: courseID, ID: wid})
	if err != nil {
		return CloseResult{}, wrap("khoá cửa sổ", err)
	}
	res, err := s.closeLocked(ctx, tx, q, ses.StartsAt, w, actor, pol)
	if err != nil {
		return CloseResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return CloseResult{}, wrap("commit", err)
	}
	return res, nil
}

// withCounts điền `checked_in` (lượt của cửa sổ) và `over_headcount` (so với số sinh viên khác nhau trong buổi, AC6) cho một cửa sổ.
func (s *Service) withCounts(ctx context.Context, q *store.Queries, courseID uuid.UUID, o WindowOut, students int) (WindowOut, error) {
	n, err := q.AttWindowCheckedIn(ctx, store.AttWindowCheckedInParams{CourseID: courseID, WindowID: o.ID})
	if err != nil {
		return o, wrap("đếm lượt", err)
	}
	ci := int(n)
	over := OverHeadcount(students, o.Headcount)
	o.CheckedIn, o.OverHeadcount = &ci, &over
	return o, nil
}

// closeLocked chạy khi đã giữ khoá buổi + cửa sổ.
func (s *Service) closeLocked(ctx context.Context, tx pgx.Tx, q *store.Queries, startsAt time.Time, w store.CheckinWindow, closedBy *uuid.UUID, pol Policy) (CloseResult, error) {
	courseID, sid := w.CourseID, w.SessionID
	if w.ClosedAt != nil { // đóng rồi: trả kết quả đã lưu
		actor := w.OpenedBy
		if w.ClosedBy != nil {
			actor = *w.ClosedBy
		}
		wr, err := q.AttClosedWritten(ctx, store.AttClosedWrittenParams{CourseID: courseID, SessionID: sid, Actor: actor})
		if err != nil {
			return CloseResult{}, wrap("kết quả đã lưu", err)
		}
		unchecked, err := q.AttUnresolvedCount(ctx, store.AttUnresolvedCountParams{CourseID: courseID, SessionID: sid})
		if err != nil {
			return CloseResult{}, wrap("đếm chưa điểm danh", err)
		}
		students, err := q.AttSessionCheckinStudents(ctx, store.AttSessionCheckinStudentsParams{CourseID: courseID, SessionID: sid})
		if err != nil {
			return CloseResult{}, wrap("đếm sinh viên", err)
		}
		o, err := s.withCounts(ctx, q, courseID, windowOut(w), int(students))
		if err != nil {
			return CloseResult{}, err
		}
		return CloseResult{Window: o, Written: Written{Present: int(wr.Present), Late: int(wr.Late)}, Unchecked: int(unchecked), SkippedExisting: max(0, int(students)-int(wr.Present)-int(wr.Late))}, nil
	}
	now := s.now()
	closed, err := q.AttWindowClose(ctx, store.AttWindowCloseParams{CourseID: courseID, ID: w.ID, ClosedAt: &now, ClosedBy: closedBy})
	if err != nil {
		return CloseResult{}, wrap("đóng cửa sổ", err)
	}
	actor := w.OpenedBy
	if closedBy != nil {
		actor = *closedBy
	}
	wr, err := q.AttCloseWriteRecords(ctx, store.AttCloseWriteRecordsParams{LateAfterMin: int32(pol.LateAfterMin), Actor: actor, CourseID: courseID, SessionID: sid})
	if err != nil {
		return CloseResult{}, wrap("ghi điểm danh", err)
	}
	if w.Round >= 2 {
		if err := q.AttRecheckMiss(ctx, store.AttRecheckMissParams{CourseID: courseID, SessionID: sid, WindowID: w.ID, Round: w.Round}); err != nil {
			return CloseResult{}, wrap("kiểm lại", err)
		}
	}
	students, err := q.AttSessionCheckinStudents(ctx, store.AttSessionCheckinStudentsParams{CourseID: courseID, SessionID: sid})
	if err != nil {
		return CloseResult{}, wrap("đếm sinh viên", err)
	}
	unchecked, err := q.AttUnresolvedCount(ctx, store.AttUnresolvedCountParams{CourseID: courseID, SessionID: sid})
	if err != nil {
		return CloseResult{}, wrap("đếm chưa điểm danh", err)
	}
	written := int(wr.Present) + int(wr.Late)
	co, err := s.withCounts(ctx, q, courseID, windowOut(closed), int(students))
	if err != nil {
		return CloseResult{}, err
	}
	res := CloseResult{Window: co, Written: Written{Present: int(wr.Present), Late: int(wr.Late)}, Unchecked: int(unchecked), SkippedExisting: max(0, int(students)-written)}
	action := "checkin_window.close"
	if closedBy == nil {
		action = "checkin_window.auto_close"
	}
	if err := audit(ctx, q, "checkin_window", courseID, closedBy, w.ID.String(), action, nil, map[string]any{"round": int(w.Round), "present": res.Written.Present, "late": res.Written.Late, "unchecked": res.Unchecked, "skipped_existing": res.SkippedExisting}); err != nil {
		return CloseResult{}, err
	}
	if written > 0 {
		if _, err := outbox.Write(ctx, tx, topicAttendanceSaved, map[string]string{"course_id": courseID.String(), "session_id": sid.String()}); err != nil {
			return CloseResult{}, wrap("outbox", err)
		}
	}
	_ = startsAt // LATE do câu SQL so với starts_at của buổi (cùng giao dịch)
	return res, nil
}

// SpotStudent là một người trong danh sách gọi tên ngẫu nhiên.
type SpotStudent struct {
	StudentID   uuid.UUID `json:"student_id"`
	FullName    string    `json:"full_name"`
	StudentCode string    `json:"student_code"`
	Reason      string    `json:"reason"` // FLAGGED | RANDOM
}

// SpotCheck chọn n người để gọi tên: ưu tiên lượt còn cờ chưa bỏ, bù ngẫu nhiên bằng crypto/rand. Không ghi gì.
func (s *Service) SpotCheck(ctx context.Context, courseID, wid uuid.UUID, n *int) ([]SpotStudent, error) {
	count := s.Cfg.SpotCheckDefault
	if n != nil {
		count = *n
	}
	if count < 1 || count > s.Cfg.SpotCheckMax {
		return nil, invalid("n", "range", fmt.Sprintf("Số người từ 1 đến %d.", s.Cfg.SpotCheckMax))
	}
	q := store.New(s.Pool)
	if _, err := q.AttWindowByID(ctx, store.AttWindowByIDParams{CourseID: courseID, ID: wid}); err != nil {
		if isNoRows(err) {
			return nil, notFound()
		}
		return nil, wrap("đọc cửa sổ", err)
	}
	flagged, err := q.AttSpotFlagged(ctx, store.AttSpotFlaggedParams{CourseID: courseID, WindowID: wid})
	if err != nil {
		return nil, wrap("lượt có cờ", err)
	}
	out := make([]SpotStudent, 0, count)
	seen := map[uuid.UUID]bool{}
	for _, f := range flagged {
		if len(out) == count {
			return out, nil
		}
		if !seen[f.StudentID] {
			seen[f.StudentID] = true
			out = append(out, SpotStudent{StudentID: f.StudentID, FullName: f.FullName, StudentCode: f.StudentCode, Reason: "FLAGGED"})
		}
	}
	all, err := q.AttSpotAll(ctx, store.AttSpotAllParams{CourseID: courseID, WindowID: wid})
	if err != nil {
		return nil, wrap("lượt của cửa sổ", err)
	}
	rest := make([]store.AttSpotAllRow, 0, len(all))
	for _, a := range all {
		if !seen[a.StudentID] {
			rest = append(rest, a)
		}
	}
	for len(out) < count && len(rest) > 0 { // Fisher–Yates từng bước bằng crypto/rand
		i, err := rand.Int(rand.Reader, big.NewInt(int64(len(rest))))
		if err != nil {
			return nil, fmt.Errorf("attendance: crypto/rand: %w", err)
		}
		r := rest[i.Int64()]
		rest = append(rest[:i.Int64()], rest[i.Int64()+1:]...)
		out = append(out, SpotStudent{StudentID: r.StudentID, FullName: r.FullName, StudentCode: r.StudentCode, Reason: "RANDOM"})
	}
	return out, nil
}

// AbsentUnchecked đánh ABSENT cho sinh viên ACTIVE chưa có dòng (FR-8): buổi phải có ≥ 1 cửa sổ và không cửa sổ mở. `Idempotency-Key` ở route.
func (s *Service) AbsentUnchecked(ctx context.Context, actor, courseID, sid uuid.UUID) (int, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return 0, wrap("mở giao dịch", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	c, _, err := s.course(ctx, q, courseID)
	if err != nil {
		return 0, err
	}
	if c.Status != store.CourseStatusACTIVE {
		return 0, apierr.New(http.StatusConflict, apierr.CourseArchived)
	}
	if _, err := q.LockSession(ctx, store.LockSessionParams{CourseID: courseID, ID: sid}); err != nil {
		if isNoRows(err) {
			return 0, notFound()
		}
		return 0, wrap("khoá buổi", err)
	}
	has, err := q.AttSessionHasWindow(ctx, store.AttSessionHasWindowParams{CourseID: courseID, SessionID: sid})
	if err != nil {
		return 0, wrap("kiểm cửa sổ", err)
	}
	if !has {
		return 0, conflict(apierr.NoWindow)
	}
	open, err := q.AttSessionOpenWindow(ctx, store.AttSessionOpenWindowParams{CourseID: courseID, SessionID: sid})
	if err != nil {
		return 0, wrap("kiểm cửa sổ mở", err)
	}
	if open {
		return 0, conflict(apierr.WindowOpen)
	}
	ids, err := q.AttAbsentUnchecked(ctx, store.AttAbsentUncheckedParams{SessionID: sid, Actor: actor, CourseID: courseID})
	if err != nil {
		return 0, wrap("đánh vắng", err)
	}
	type change struct {
		StudentID uuid.UUID `json:"student_id"`
		To        string    `json:"to"`
	}
	ch := make([]change, 0, len(ids))
	for _, id := range ids {
		ch = append(ch, change{StudentID: id, To: "ABSENT"})
	}
	if len(ids) > 0 {
		if err := audit(ctx, q, "attendance", courseID, &actor, sid.String(), "absent_unchecked", nil, map[string]any{"marked": len(ids), "changes": ch}); err != nil {
			return 0, err
		}
		if _, err := outbox.Write(ctx, tx, topicAttendanceSaved, map[string]string{"course_id": courseID.String(), "session_id": sid.String()}); err != nil {
			return 0, wrap("outbox", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, wrap("commit", err)
	}
	return len(ids), nil
}

// CountUnresolved: buổi có cửa sổ và còn bao nhiêu sinh viên ACTIVE chưa có dòng điểm danh (FR-9, crm gọi trong giao dịch của PUT …/attendance).
func CountUnresolved(ctx context.Context, q *store.Queries, courseID, sid uuid.UUID) (hasWindow bool, count int, err error) {
	has, err := q.AttSessionHasWindow(ctx, store.AttSessionHasWindowParams{CourseID: courseID, SessionID: sid})
	if err != nil || !has {
		return false, 0, err
	}
	n, err := q.AttUnresolvedCount(ctx, store.AttUnresolvedCountParams{CourseID: courseID, SessionID: sid})
	return true, int(n), err
}

// Sweep đóng các cửa sổ đã quá `closes_at` (việc nền `attendance.sweep`, FR-7). Không khoá ở câu chọn; Close tự khoá theo thứ tự buổi → cửa sổ
// và idempotent nên N worker cùng chạy cũng an toàn (ARCHITECTURE 11.6). Trả số cửa sổ đã đóng thật.
func (s *Service) Sweep(ctx context.Context) (int, error) {
	q := store.New(s.Pool)
	rows, err := q.AttSweepIDs(ctx, s.now())
	if err != nil {
		return 0, wrap("chọn cửa sổ quá giờ", err)
	}
	closed := 0
	for _, r := range rows {
		before, err := q.AttWindowByID(ctx, store.AttWindowByIDParams{CourseID: r.CourseID, ID: r.ID})
		if err != nil || before.ClosedAt != nil {
			continue
		}
		if _, err := s.Close(ctx, nil, r.CourseID, r.ID); err != nil {
			var ae *apierr.Error
			if errors.As(err, &ae) && ae.Status == http.StatusConflict { // lớp lưu trữ: bỏ qua
				continue
			}
			s.log().WarnContext(ctx, "attendance.sweep: đóng cửa sổ lỗi", "window_id", r.ID.String(), "error", err.Error())
			continue
		}
		closed++
	}
	return closed, nil
}

// rate: `limit` lượt / phút / người cho nhóm `kind` (ep:rl:{kind}:{uid}:{phút}, TTL 120 s); Redis nil → cho qua.
func (s *Service) rate(ctx context.Context, kind string, actor uuid.UUID, limit int) error {
	if s.Redis == nil || limit <= 0 {
		return nil
	}
	key := appredis.Key("rl", kind, actor.String(), fmt.Sprint(s.now().Unix()/60))
	pipe := s.Redis.TxPipeline()
	n := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, appredis.TTLRateLimit)
	if _, err := pipe.Exec(ctx); err != nil {
		s.log().WarnContext(ctx, "attendance: hạn mức Redis lỗi, cho qua", "kind", kind) // Redis chết: cho qua như middleware hạn mức
		return nil
	}
	if int(n.Val()) > limit {
		return apierr.New(http.StatusTooManyRequests, apierr.RateLimited).WithRetryAfter(int(60 - s.now().Unix()%60))
	}
	return nil
}
