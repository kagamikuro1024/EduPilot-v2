package attendance

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/httpapi/httpx"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/store"
)

// CheckinRow là một lượt điểm danh cho Staff (SRS FR-16). Chỉ Staff thấy cờ / mức tin cậy / khoảng cách; sinh viên không bao giờ.
type CheckinRow struct {
	ID           uuid.UUID  `json:"id"`
	WindowID     uuid.UUID  `json:"window_id"`
	Round        int        `json:"round"`
	Student      StudentRef `json:"student"`
	CreatedAt    time.Time  `json:"created_at"`
	Channel      string     `json:"channel"`
	Assurance    int        `json:"assurance"`
	Flags        []FlagInfo `json:"flags"`
	Cleared      bool       `json:"cleared"`
	GeoDistanceM *int       `json:"geo_distance_m,omitempty"`
	InCampus     *bool      `json:"in_campus,omitempty"`
}

// StudentRef là người điểm danh (chỉ Staff thấy).
type StudentRef struct {
	ID          uuid.UUID `json:"id"`
	FullName    string    `json:"full_name"`
	StudentCode string    `json:"student_code"`
}

// ListCheckins: con trỏ (created_at, id), limit ≤ 100, không OFFSET (luật 13). `flagged` lọc lượt còn cờ chưa bỏ; `round` 0 = mọi vòng.
func (s *Service) ListCheckins(ctx context.Context, courseID, sid uuid.UUID, flagged bool, round int, p httpx.PageParams) (httpx.Page[CheckinRow], error) {
	q := store.New(s.Pool)
	if _, err := q.AttSessionExists(ctx, store.AttSessionExistsParams{CourseID: courseID, ID: sid}); err != nil {
		if isNoRows(err) {
			return httpx.Page[CheckinRow]{}, notFound()
		}
		return httpx.Page[CheckinRow]{}, wrap("đọc buổi", err)
	}
	args := store.AttCheckinsListParams{CourseID: courseID, SessionID: sid, Round: int32(round), Flagged: flagged, RowLimit: int32(p.Fetch())}
	if p.Cursor != nil {
		t, id := p.Cursor.CreatedAt, p.Cursor.ID
		args.AfterAt, args.AfterID = &t, &id
	}
	rows, err := q.AttCheckinsList(ctx, args)
	if err != nil {
		return httpx.Page[CheckinRow]{}, wrap("liệt kê lượt", err)
	}
	out := make([]CheckinRow, 0, len(rows))
	for _, r := range rows {
		cr := CheckinRow{ID: r.ID, WindowID: r.WindowID, Round: int(r.Round), Student: StudentRef{ID: r.StudentID, FullName: r.FullName, StudentCode: r.StudentCode}, Channel: string(r.Channel),
			Assurance: int(r.Assurance), Flags: FlagInfos(r.Flags), Cleared: r.FlagsClearedAt != nil, InCampus: r.InCampus, CreatedAt: r.CreatedAt}
		if r.GeoDistanceM != nil {
			d := int(*r.GeoDistanceM)
			cr.GeoDistanceM = &d
		}
		out = append(out, cr)
	}
	return httpx.Paginate(out, p, func(r CheckinRow) (time.Time, string) { return r.CreatedAt, r.ID.String() }), nil
}

// ClearFlags bỏ cờ của một lượt (Staff đã xem): ghi audit_log; gọi lại là no-op (idempotent). Không xoá mã cờ, chỉ đánh dấu đã xem.
func (s *Service) ClearFlags(ctx context.Context, actor, courseID, cid uuid.UUID) (CheckinRow, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return CheckinRow{}, wrap("mở giao dịch", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	if err := s.requireActive(ctx, q, courseID); err != nil {
		return CheckinRow{}, err
	}
	c, err := q.AttCheckinByID(ctx, store.AttCheckinByIDParams{CourseID: courseID, ID: cid})
	if err != nil {
		if isNoRows(err) {
			return CheckinRow{}, notFound()
		}
		return CheckinRow{}, wrap("đọc lượt", err)
	}
	if c.FlagsClearedAt == nil && len(c.Flags) > 0 {
		now := s.now()
		if _, err := q.AttCheckinClearFlags(ctx, store.AttCheckinClearFlagsParams{Actor: &actor, At: &now, CourseID: courseID, ID: cid}); err != nil {
			return CheckinRow{}, wrap("bỏ cờ", err)
		}
		if err := audit(ctx, q, "attendance_checkin", courseID, &actor, cid.String(), "checkin.clear_flags", map[string]any{"flags": c.Flags}, map[string]any{"cleared": true}); err != nil {
			return CheckinRow{}, err
		}
		c.FlagsClearedAt = &now
		// mục CHECKIN_REVIEW của «Hôm nay» đổi → vô hiệu cache Staff (cùng topic với lưu điểm danh: bộ vô hiệu xoá cache của Giảng viên + TA)
		if _, err := outbox.Write(ctx, tx, topicAttendanceSaved, map[string]string{"course_id": courseID.String(), "session_id": c.SessionID.String()}); err != nil {
			return CheckinRow{}, wrap("outbox", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return CheckinRow{}, wrap("commit", err)
	}
	return CheckinRow{ID: c.ID, WindowID: c.WindowID, Student: StudentRef{ID: c.StudentID}, Channel: string(c.Channel), Assurance: int(c.Assurance), Flags: FlagInfos(c.Flags), Cleared: c.FlagsClearedAt != nil, CreatedAt: c.CreatedAt}, nil
}

// ReviewItem là một mục «Hôm nay» CHECKIN_REVIEW (FR-18).
type ReviewItem struct {
	CourseID  uuid.UUID
	SessionID uuid.UUID
	SessionNo int
	Flagged   int
	ClosedAt  time.Time
}

// ReviewSessions: buổi có cửa sổ đã đóng trong CHECKIN_REVIEW_DAYS ngày và còn lượt có cờ chưa bỏ, cho các lớp của Staff.
func (s *Service) ReviewSessions(ctx context.Context, courseIDs []uuid.UUID) ([]ReviewItem, error) {
	if len(courseIDs) == 0 {
		return nil, nil
	}
	since := s.now().AddDate(0, 0, -s.Cfg.ReviewDays)
	rows, err := store.New(s.Pool).AttReviewSessions(ctx, store.AttReviewSessionsParams{Since: &since, CourseIds: courseIDs})
	if err != nil {
		return nil, wrap("buổi cần xem", err)
	}
	out := make([]ReviewItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, ReviewItem{CourseID: r.CourseID, SessionID: r.SessionID, SessionNo: int(r.SessionNo), Flagged: int(r.Flagged), ClosedAt: r.ClosedAt})
	}
	return out, nil
}

// PurgeCheckins xoá lượt thô của cửa sổ đã đóng quá CHECKIN_RAW_RETENTION_DAYS ngày, rồi thiết bị mồ côi quá hạn; lô ≤ 1.000 hàng, lặp tới 0
// (FR-21, ARCHITECTURE 11.2). Log chỉ có số hàng. Trả (số lượt, số thiết bị).
func (s *Service) PurgeCheckins(ctx context.Context) (int64, int64, error) {
	q := store.New(s.Pool)
	cutoff := s.now().AddDate(0, 0, -s.Cfg.RawRetentionDays)
	var rows, devs int64
	for {
		n, err := q.AttPurgeCheckins(ctx, &cutoff)
		if err != nil {
			return rows, devs, wrap("xoá lượt thô", err)
		}
		rows += n
		if n == 0 {
			break
		}
	}
	for {
		n, err := q.AttPurgeDevices(ctx, cutoff)
		if err != nil {
			return rows, devs, wrap("xoá thiết bị mồ côi", err)
		}
		devs += n
		if n == 0 {
			break
		}
	}
	if rows+devs > 0 {
		s.log().InfoContext(ctx, "attendance.purge", "checkins", rows, "devices", devs)
	}
	return rows, devs, nil
}

// ForgetStudent xoá NGAY mọi lượt điểm danh của một người (job xoá tài khoản của phase PR gọi trong giao dịch của nó; P-108-10).
func ForgetStudent(ctx context.Context, q *store.Queries, userID uuid.UUID) (int64, error) {
	n, err := q.AttForgetStudent(ctx, userID)
	if err != nil {
		return 0, wrap("quên sinh viên", err)
	}
	return n, nil
}
