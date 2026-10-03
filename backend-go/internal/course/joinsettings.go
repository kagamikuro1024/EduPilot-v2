package course

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/edupilot/backend-go/internal/store"
)

var domainRE = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)+$`)

// JoinInfo là thân `GET …/join-code`.
type JoinInfo struct {
	JoinCode           string
	JoinURL            string
	Enabled            bool
	ExpiresAt          *time.Time
	RequireApproval    bool
	AllowedEmailDomain *string
	Capacity           *int
	ActiveStudents     int
	Pending            int
	Version            int
}

func (s Service) joinInfo(ctx context.Context, q *store.Queries, c store.Course) (JoinInfo, error) {
	n, err := q.CountCourseStudents(ctx, c.ID)
	if err != nil {
		return JoinInfo{}, fmt.Errorf("course: đếm sinh viên: %w", err)
	}
	code := strings.TrimSpace(c.JoinCode)
	info := JoinInfo{JoinCode: code, JoinURL: s.PublicURL + "/join/" + code, Enabled: c.JoinEnabled, ExpiresAt: c.JoinExpiresAt, RequireApproval: c.JoinRequireApproval,
		AllowedEmailDomain: c.AllowedEmailDomain, ActiveStudents: int(n.Active), Pending: int(n.Pending), Version: int(c.Version)}
	if c.Capacity != nil {
		v := int(*c.Capacity)
		info.Capacity = &v
	}
	return info, nil
}

// JoinCode đọc mã và cài đặt tham gia (TA, giảng viên, Admin).
func (s Service) JoinCode(ctx context.Context, courseID uuid.UUID) (JoinInfo, error) {
	q := store.New(s.Pool)
	c, err := q.GetCourse(ctx, courseID)
	if errors.Is(err, pgx.ErrNoRows) {
		return JoinInfo{}, ErrNotFound
	}
	if err != nil {
		return JoinInfo{}, fmt.Errorf("course: đọc lớp: %w", err)
	}
	return s.joinInfo(ctx, q, c)
}

// Regenerate thay mã trong MỘT câu UPDATE: mã cũ chết ngay, sinh viên đã vào lớp không bị ảnh hưởng. Audit chỉ ghi 2 ký tự đầu của mã.
func (s Service) Regenerate(ctx context.Context, actor, courseID uuid.UUID) (JoinInfo, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return JoinInfo{}, fmt.Errorf("course: mở transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	old, err := q.LockCourse(ctx, courseID)
	if errors.Is(err, pgx.ErrNoRows) {
		return JoinInfo{}, ErrNotFound
	}
	if err != nil {
		return JoinInfo{}, fmt.Errorf("course: khoá lớp: %w", err)
	}
	if old.Status == store.CourseStatusARCHIVED {
		return JoinInfo{}, ErrArchived
	}
	var c store.Course
	for attempt := 0; ; attempt++ {
		if attempt >= maxCodeAttempts {
			return JoinInfo{}, ErrCodeExhausted
		}
		code, err := s.newCode()
		if err != nil {
			return JoinInfo{}, err
		}
		c, err = q.RegenerateJoinCode(ctx, store.RegenerateJoinCodeParams{ID: courseID, JoinCode: code})
		if err == nil {
			break
		}
		if !errors.Is(err, pgx.ErrNoRows) { // ErrNoRows = trùng mã khác, thử lại
			return JoinInfo{}, fmt.Errorf("course: tạo lại mã: %w", err)
		}
	}
	prefix := func(c string) string { return strings.TrimSpace(c)[:2] }
	if err := s.audit(ctx, q, courseID, actor, courseID.String(), "join_code_regenerated", map[string]any{"code_prefix": prefix(old.JoinCode)}, map[string]any{"code_prefix": prefix(c.JoinCode)}); err != nil {
		return JoinInfo{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return JoinInfo{}, fmt.Errorf("course: commit tạo lại mã: %w", err)
	}
	return s.joinInfo(ctx, store.New(s.Pool), c)
}

// SettingsInput là thân PUT …/join-settings: khoá vắng = giữ nguyên; expires_at / allowed_email_domain / capacity rỗng hoặc null = bỏ.
type SettingsInput struct {
	Version                  int
	Enabled, RequireApproval *bool
	SetExpires               bool
	ExpiresAt                *time.Time
	SetDomain                bool
	Domain                   *string
	SetCapacity              bool
	Capacity                 *int
}

// JoinVersionConflictError: sai version; Current là bản hiện hành.
type JoinVersionConflictError struct{ Current JoinInfo }

func (e *JoinVersionConflictError) Error() string { return "course: sai version cài đặt tham gia" }

// PutJoinSettings kiểm rồi lưu cài đặt tham gia (khoá lạc quan theo version).
func (s Service) PutJoinSettings(ctx context.Context, actor, courseID uuid.UUID, in SettingsInput) (JoinInfo, error) {
	now := s.now()
	p := store.UpdateJoinSettingsParams{ID: courseID, Version: int32(in.Version), Enabled: in.Enabled, RequireApproval: in.RequireApproval, SetExpires: in.SetExpires, SetDomain: in.SetDomain, SetCapacity: in.SetCapacity}
	if in.SetExpires && in.ExpiresAt != nil {
		if !in.ExpiresAt.After(now) || in.ExpiresAt.After(now.Add(366*24*time.Hour)) {
			return JoinInfo{}, invalid("expires_at", "EXPIRES_OUT_OF_RANGE", "Hạn phải ở tương lai và không quá 366 ngày.")
		}
		p.ExpiresAt = in.ExpiresAt
	}
	if in.SetDomain && in.Domain != nil {
		d := strings.ToLower(strings.TrimSpace(*in.Domain))
		if d != "" {
			if !domainRE.MatchString(d) {
				return JoinInfo{}, invalid("allowed_email_domain", "FORMAT", "Tên miền không hợp lệ, ví dụ ptit.edu.vn.")
			}
			p.Domain = &d
		}
	}
	if in.SetCapacity && in.Capacity != nil {
		if *in.Capacity < 1 || *in.Capacity > 1000 {
			return JoinInfo{}, invalid("capacity", "OUT_OF_RANGE", "Sĩ số từ 1 đến 1.000.")
		}
		v := int32(*in.Capacity)
		p.Capacity = &v
	}

	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return JoinInfo{}, fmt.Errorf("course: mở transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	cur, err := q.LockCourse(ctx, courseID)
	if errors.Is(err, pgx.ErrNoRows) {
		return JoinInfo{}, ErrNotFound
	}
	if err != nil {
		return JoinInfo{}, fmt.Errorf("course: khoá lớp: %w", err)
	}
	if cur.Status == store.CourseStatusARCHIVED {
		return JoinInfo{}, ErrArchived
	}
	if int(cur.Version) != in.Version {
		info, err := s.joinInfo(ctx, q, cur)
		if err != nil {
			return JoinInfo{}, err
		}
		return JoinInfo{}, &JoinVersionConflictError{Current: info}
	}
	if in.SetCapacity && in.Capacity != nil {
		n, err := q.CountActiveStudents(ctx, courseID)
		if err != nil {
			return JoinInfo{}, fmt.Errorf("course: đếm sinh viên: %w", err)
		}
		if int(n) > *in.Capacity {
			return JoinInfo{}, invalid("capacity", "CAPACITY_BELOW_ACTIVE", "Sĩ số không được nhỏ hơn số sinh viên đang học.")
		}
	}
	upd, err := q.UpdateJoinSettings(ctx, p)
	if err != nil {
		return JoinInfo{}, fmt.Errorf("course: lưu cài đặt tham gia: %w", err)
	}
	facts := func(c store.Course) map[string]any {
		return map[string]any{"enabled": c.JoinEnabled, "require_approval": c.JoinRequireApproval, "expires_at": c.JoinExpiresAt, "domain": c.AllowedEmailDomain, "capacity": c.Capacity}
	}
	if err := s.audit(ctx, q, courseID, actor, courseID.String(), "join_settings_updated", facts(cur), facts(upd)); err != nil {
		return JoinInfo{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return JoinInfo{}, fmt.Errorf("course: commit cài đặt: %w", err)
	}
	return s.joinInfo(ctx, store.New(s.Pool), upd)
}
