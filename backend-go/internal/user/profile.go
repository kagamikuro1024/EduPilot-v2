package user

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/store"
)

// Hồ sơ và tuỳ chọn của CHÍNH người đăng nhập (US-P2-07 AC9, AC10). Không có tham số user_id ở bất kỳ đường nào.

var studentCodeRE = regexp.MustCompile(`^[A-Za-z0-9]{6,15}$`)

// Profile là `GET/PUT /me/profile`. `email` và `role` chỉ đọc.
type Profile struct {
	ID            uuid.UUID
	Email         string
	FullName      string
	Role          string
	StudentCode   string
	EmailVerified bool
	Version       int
}

func profileOf(u store.User) Profile {
	p := Profile{ID: u.ID, Email: u.Email, FullName: u.FullName, Role: string(u.Role), EmailVerified: u.EmailVerifiedAt != nil, Version: int(u.Version)}
	if u.StudentCode != nil {
		p.StudentCode = *u.StudentCode
	}
	return p
}

// GetProfile đọc hồ sơ của id.
func (s *Service) GetProfile(ctx context.Context, id uuid.UUID) (Profile, error) {
	u, err := store.New(s.pool).GetUser(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, ErrNotFound
	}
	if err != nil {
		return Profile{}, fmt.Errorf("user: đọc hồ sơ: %w", err)
	}
	return profileOf(u), nil
}

// ProfileInput là thân PUT /me/profile; con trỏ nil = không đổi, StudentCode "" = xoá.
type ProfileInput struct {
	Version     int
	FullName    *string
	StudentCode *string
}

// PutProfile sửa tên / MSSV của chính mình. MSSV chỉ là thông tin khai báo: KHÔNG đổi `enrollments.student_code_snapshot` của lớp nào
// (ảnh chụp bất biến — chống đổi MSSV để chui vào dữ liệu người khác) và không bao giờ nối tài khoản vào lớp. Ghi audit_log (không MSSV).
func (s *Service) PutProfile(ctx context.Context, id uuid.UUID, in ProfileInput) (Profile, error) {
	var name *string
	if in.FullName != nil {
		n := strings.TrimSpace(*in.FullName)
		if p := auth.ValidateName(n); p != nil {
			return Profile{}, &InvalidError{Field: p.Field, Code: p.Code, Message: p.Message}
		}
		name = &n
	}
	var code *string
	setCode := in.StudentCode != nil
	if setCode {
		c := strings.ToUpper(strings.TrimSpace(*in.StudentCode))
		if c != "" {
			if !studentCodeRE.MatchString(c) {
				return Profile{}, &InvalidError{Field: "student_code", Code: "STUDENT_CODE_FORMAT", Message: "Mã số sinh viên gồm 6–15 chữ và số."}
			}
			code = &c
		}
	}
	if name == nil && !setCode {
		return Profile{}, &InvalidError{Field: "body", Code: "EMPTY", Message: "Không có gì để đổi."}
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Profile{}, fmt.Errorf("user: mở transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	cur, err := q.LockUserForUpdate(ctx, id)
	if err != nil {
		return Profile{}, fmt.Errorf("user: khoá hàng: %w", err)
	}
	if setCode && code != nil && cur.Role != store.UserRoleSTUDENT {
		return Profile{}, &InvalidError{Field: "student_code", Code: "NOT_AVAILABLE", Message: "Chỉ sinh viên mới có mã số sinh viên."}
	}
	if int(cur.Version) != in.Version {
		return Profile{}, &ProfileConflictError{Current: profileOf(cur)}
	}
	u, err := q.UpdateProfile(ctx, store.UpdateProfileParams{ID: id, Version: int32(in.Version), FullName: name, SetStudentCode: setCode, StudentCode: code})
	if err != nil {
		return Profile{}, fmt.Errorf("user: cập nhật hồ sơ: %w", err)
	}
	changed, _ := json.Marshal(map[string]bool{"full_name": name != nil && *name != cur.FullName, "student_code": setCode})
	if _, err := q.InsertAuditLog(ctx, store.InsertAuditLogParams{ActorID: &id, Entity: "user", EntityID: id.String(), Action: "profile_updated", After: changed}); err != nil {
		return Profile{}, fmt.Errorf("user: ghi audit_log: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Profile{}, fmt.Errorf("user: commit hồ sơ: %w", err)
	}
	return profileOf(u), nil
}

// ProfileConflictError: sai version khi sửa hồ sơ; Current là bản hiện hành.
type ProfileConflictError struct{ Current Profile }

func (e *ProfileConflictError) Error() string { return "user: sai version hồ sơ" }

// Settings là `GET/PUT /me/settings`.
type Settings struct {
	NotifyTicketByMail   bool
	NotifyAnswerByMail   bool
	RemindDeadlineByMail bool
	Version              int
}

func settingsOf(r store.UserSetting) Settings {
	return Settings{NotifyTicketByMail: r.NotifyTicketByMail, NotifyAnswerByMail: r.NotifyAnswerByMail, RemindDeadlineByMail: r.RemindDeadlineByMail, Version: int(r.Version)}
}

// GetSettings đọc tuỳ chọn; tạo lười dòng mặc định ở lần đọc đầu (INSERT … ON CONFLICT DO NOTHING).
func (s *Service) GetSettings(ctx context.Context, id uuid.UUID) (Settings, error) {
	r, err := store.New(s.pool).EnsureUserSettings(ctx, id)
	if err != nil {
		return Settings{}, fmt.Errorf("user: đọc tuỳ chọn: %w", err)
	}
	return settingsOf(store.UserSetting(r)), nil
}

// SettingsInput là thân PUT /me/settings; con trỏ nil = giữ nguyên.
type SettingsInput struct {
	Version              int
	NotifyTicketByMail   *bool
	NotifyAnswerByMail   *bool
	RemindDeadlineByMail *bool
}

// SettingsConflictError: sai version; Current là bản hiện hành.
type SettingsConflictError struct{ Current Settings }

func (e *SettingsConflictError) Error() string { return "user: sai version tuỳ chọn" }

// PutSettings sửa tuỳ chọn của chính mình (khoá lạc quan theo version).
func (s *Service) PutSettings(ctx context.Context, id uuid.UUID, in SettingsInput) (Settings, error) {
	q := store.New(s.pool)
	if _, err := q.EnsureUserSettings(ctx, id); err != nil {
		return Settings{}, fmt.Errorf("user: tạo tuỳ chọn: %w", err)
	}
	r, err := q.UpdateUserSettings(ctx, store.UpdateUserSettingsParams{
		UserID: id, Version: int32(in.Version), NotifyTicketByMail: in.NotifyTicketByMail, NotifyAnswerByMail: in.NotifyAnswerByMail, RemindDeadlineByMail: in.RemindDeadlineByMail,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		cur, gerr := q.EnsureUserSettings(ctx, id)
		if gerr != nil {
			return Settings{}, fmt.Errorf("user: đọc tuỳ chọn: %w", gerr)
		}
		return Settings{}, &SettingsConflictError{Current: settingsOf(store.UserSetting(cur))}
	}
	if err != nil {
		return Settings{}, fmt.Errorf("user: cập nhật tuỳ chọn: %w", err)
	}
	return settingsOf(r), nil
}
