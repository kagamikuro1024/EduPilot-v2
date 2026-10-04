// Package user: quản trị người dùng của Admin (FEAT-account-security US-P2-06) — mời giảng viên / trợ giảng, khoá / mở khoá, đổi vai
// giữa hai vai nhân viên, danh sách. Admin KHÔNG bao giờ đặt, thấy hay đặt lại hộ mật khẩu (người được mời tự đặt qua liên kết một lần).
// Quyền: chỉ ADMIN (route kiểm trước khi vào đây). Sổ kiểm tra: mọi thay đổi ghi audit_log kèm người làm.
package user

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/store"
)

// Lỗi nghiệp vụ (handler ánh xạ sang mã API).
var (
	ErrEmailTaken = errors.New("user: email đã có tài khoản")
	ErrNotFound   = errors.New("user: không có người dùng này")
	ErrNotInvited = errors.New("user: tài khoản không ở trạng thái được mời")
	ErrSelf       = errors.New("user: không tự đổi hay tự khoá chính mình")
	ErrLastAdmin  = errors.New("user: không khoá quản trị viên cuối cùng")
)

// InvalidError: một trường sai (422 VALIDATION_FAILED).
type InvalidError struct{ Field, Code, Message string }

func (e *InvalidError) Error() string { return "user: " + e.Field + ": " + e.Code }

// VersionConflictError: sai version; Current là bản hiện hành (409 VERSION_CONFLICT).
type VersionConflictError struct{ Current View }

func (e *VersionConflictError) Error() string { return "user: sai version" }

// View là dữ liệu tối thiểu của một người dùng cho Admin: không MSSV, không hash, không ics_token, không số lần sai.
type View struct {
	ID          uuid.UUID
	Email       string
	FullName    string
	Role        string
	Status      string
	LastLoginAt *time.Time
	Version     int
	CreatedAt   time.Time
}

func viewOf(u store.User) View {
	return View{ID: u.ID, Email: u.Email, FullName: u.FullName, Role: string(u.Role), Status: string(u.Status), LastLoginAt: u.LastLoginAt, Version: int(u.Version), CreatedAt: u.CreatedAt}
}

// Config là các hạn dùng ở đây.
type Config struct {
	InviteTTL time.Duration
}

// Service là nghiệp vụ quản trị người dùng.
type Service struct {
	pool     *pgxpool.Pool
	clk      clock.Clock
	mail     auth.MailQueue
	sessions *auth.Sessions
	accounts *auth.Accounts
	cfg      Config
}

// New dựng Service. sessions/accounts cần cho thu hồi phiên và giới hạn gửi lại.
func New(pool *pgxpool.Pool, clk clock.Clock, mail auth.MailQueue, sessions *auth.Sessions, accounts *auth.Accounts, cfg Config) *Service {
	if clk == nil {
		clk = clock.Real{}
	}
	if cfg.InviteTTL <= 0 {
		cfg.InviteTTL = 72 * time.Hour
	}
	return &Service{pool: pool, clk: clk, mail: mail, sessions: sessions, accounts: accounts, cfg: cfg}
}

// InviteInput là thân POST /admin/users.
type InviteInput struct{ Email, FullName, Role string }

func roleVN(r store.UserRole) string {
	if r == store.UserRoleTA {
		return "trợ giảng"
	}
	return "giảng viên"
}

// Invite tạo tài khoản INVITED (chưa mật khẩu) và xếp thư mời — cùng một giao dịch. Token INVITE 72 giờ do consumer thư phát lúc gửi.
func (s *Service) Invite(ctx context.Context, actor uuid.UUID, in InviteInput) (View, error) {
	email := auth.NormalizeEmail(in.Email)
	name := strings.TrimSpace(in.FullName)
	if p := auth.ValidateEmail(email); p != nil {
		return View{}, &InvalidError{Field: p.Field, Code: p.Code, Message: p.Message}
	}
	if p := auth.ValidateName(name); p != nil {
		return View{}, &InvalidError{Field: p.Field, Code: p.Code, Message: p.Message}
	}
	var role store.UserRole
	switch in.Role {
	case "TEACHER":
		role = store.UserRoleTEACHER
	case "TA":
		role = store.UserRoleTA
	default:
		return View{}, &InvalidError{Field: "role", Code: "INVALID_ROLE", Message: "Chỉ mời được giảng viên hoặc trợ giảng."}
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return View{}, fmt.Errorf("user: mở transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	u, err := q.InsertInvitedUser(ctx, store.InsertInvitedUserParams{Email: email, FullName: name, Role: role})
	if errors.Is(err, pgx.ErrNoRows) {
		return View{}, ErrEmailTaken
	}
	if err != nil {
		return View{}, fmt.Errorf("user: tạo lời mời: %w", err)
	}
	inviter, err := s.actorName(ctx, q, actor)
	if err != nil {
		return View{}, err
	}
	if err := s.audit(ctx, q, actor, u.ID, "user_invited", nil, map[string]any{"role": role, "status": "INVITED"}); err != nil {
		return View{}, err
	}
	if err := s.queueInvite(ctx, tx, u, inviter); err != nil {
		return View{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return View{}, fmt.Errorf("user: commit lời mời: %w", err)
	}
	return viewOf(u), nil
}

func (s *Service) queueInvite(ctx context.Context, tx pgx.Tx, u store.User, inviter string) error {
	return s.mail(ctx, tx, u.Email, "invite_staff", map[string]any{
		"user_id": u.ID.String(), "full_name": u.FullName, "inviter_name": inviter, "role_vn": roleVN(u.Role),
	}, "")
}

func (s *Service) actorName(ctx context.Context, q *store.Queries, actor uuid.UUID) (string, error) {
	a, err := q.GetUser(ctx, actor)
	if err != nil {
		return "", fmt.Errorf("user: đọc người mời: %w", err)
	}
	return a.FullName, nil
}

// ResendInvite thu hồi liên kết cũ chưa dùng và xếp thư mời mới. Chỉ INVITED; 1 lần / 60 giây / người (ThrottledError).
func (s *Service) ResendInvite(ctx context.Context, actor, id uuid.UUID) (time.Time, error) {
	q := store.New(s.pool)
	u, err := q.GetUser(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, ErrNotFound
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("user: đọc người dùng: %w", err)
	}
	if u.Status != store.UserStatusINVITED || (u.Role != store.UserRoleTEACHER && u.Role != store.UserRoleTA) {
		return time.Time{}, ErrNotInvited
	}
	if err := s.accounts.ThrottleResend(ctx, "invite", u.Email); err != nil {
		return time.Time{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return time.Time{}, fmt.Errorf("user: mở transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tq := store.New(tx)
	inviter, err := s.actorName(ctx, tq, actor)
	if err != nil {
		return time.Time{}, err
	}
	if err := (auth.Tokens{Clock: s.clk}).RevokeUnused(ctx, tx, u.ID, auth.TokenInvite); err != nil {
		return time.Time{}, err
	}
	if err := s.audit(ctx, tq, actor, u.ID, "invite_resent", nil, nil); err != nil {
		return time.Time{}, err
	}
	if err := s.queueInvite(ctx, tx, u, inviter); err != nil {
		return time.Time{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return time.Time{}, fmt.Errorf("user: commit gửi lại lời mời: %w", err)
	}
	return s.clk.Now().Add(s.cfg.InviteTTL), nil
}

// PatchInput là thân PATCH /admin/users/{id}; con trỏ nil = không đổi. Version bắt buộc (khoá lạc quan).
type PatchInput struct {
	Version  int
	FullName *string
	Role     *string
	Status   *string
}

// Patch đổi tên / vai (chỉ TEACHER ↔ TA) / khoá-mở khoá. Khoá hoặc đổi vai thu hồi MỌI phiên của người đó (token mới mang vai mới).
func (s *Service) Patch(ctx context.Context, actor, id uuid.UUID, in PatchInput) (View, error) {
	if in.FullName == nil && in.Role == nil && in.Status == nil {
		return View{}, &InvalidError{Field: "body", Code: "EMPTY", Message: "Không có gì để đổi."}
	}
	var name *string
	if in.FullName != nil {
		n := strings.TrimSpace(*in.FullName)
		if p := auth.ValidateName(n); p != nil {
			return View{}, &InvalidError{Field: p.Field, Code: p.Code, Message: p.Message}
		}
		name = &n
	}
	var role *store.UserRole
	if in.Role != nil {
		r := store.UserRole(*in.Role)
		if r != store.UserRoleTEACHER && r != store.UserRoleTA {
			return View{}, &InvalidError{Field: "role", Code: "INVALID_ROLE", Message: "Chỉ đổi giữa giảng viên và trợ giảng."}
		}
		role = &r
	}
	var status *store.UserStatus
	if in.Status != nil {
		st := store.UserStatus(*in.Status)
		if st != store.UserStatusACTIVE && st != store.UserStatusDISABLED {
			return View{}, &InvalidError{Field: "status", Code: "INVALID_STATUS", Message: "Trạng thái chỉ là ACTIVE hoặc DISABLED."}
		}
		status = &st
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return View{}, fmt.Errorf("user: mở transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	cur, err := q.LockUserForUpdate(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return View{}, ErrNotFound
	}
	if err != nil {
		return View{}, fmt.Errorf("user: khoá hàng: %w", err)
	}

	if (role != nil || status != nil) && cur.ID == actor {
		return View{}, ErrSelf
	}
	if role != nil && cur.Role != store.UserRoleTEACHER && cur.Role != store.UserRoleTA {
		return View{}, &InvalidError{Field: "role", Code: "ROLE_LOCKED", Message: "Không đổi được vai của tài khoản này."}
	}
	if status != nil && *status == store.UserStatusDISABLED && cur.Role == store.UserRoleADMIN && cur.Status == store.UserStatusACTIVE {
		n, err := q.CountOtherActiveAdmins(ctx, cur.ID)
		if err != nil {
			return View{}, fmt.Errorf("user: đếm quản trị viên: %w", err)
		}
		if n == 0 {
			return View{}, ErrLastAdmin
		}
	}
	if int(cur.Version) != in.Version {
		return View{}, &VersionConflictError{Current: viewOf(cur)}
	}

	// Mở khoá: quay về ACTIVE nếu đã có mật khẩu, ngược lại về INVITED (chưa nhận lời mời).
	if status != nil && *status == store.UserStatusACTIVE && (cur.PasswordHash == nil || *cur.PasswordHash == "") {
		invited := store.UserStatusINVITED
		status = &invited
	}
	updated, err := q.UpdateUserAdmin(ctx, store.UpdateUserAdminParams{ID: id, Version: int32(in.Version), FullName: name, Role: role, Status: status})
	if errors.Is(err, pgx.ErrNoRows) {
		return View{}, &VersionConflictError{Current: viewOf(cur)}
	}
	if err != nil {
		return View{}, fmt.Errorf("user: cập nhật: %w", err)
	}

	var after func(context.Context)
	if (role != nil && *role != cur.Role) || (status != nil && *status == store.UserStatusDISABLED) {
		reason := auth.ReasonRoleChanged
		if status != nil && *status == store.UserStatusDISABLED {
			reason = auth.ReasonAccountOff
		}
		if after, err = s.sessions.RevokeAll(ctx, q, id, reason); err != nil {
			return View{}, err
		}
	}
	if err := s.audit(ctx, q, actor, id, auditAction(cur, updated), map[string]any{"role": cur.Role, "status": cur.Status}, map[string]any{"role": updated.Role, "status": updated.Status}); err != nil {
		return View{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return View{}, fmt.Errorf("user: commit cập nhật: %w", err)
	}
	if after != nil {
		after(ctx)
	}
	return viewOf(updated), nil
}

func auditAction(before, after store.User) string {
	switch {
	case before.Status != after.Status && after.Status == store.UserStatusDISABLED:
		return "user_disabled"
	case before.Status != after.Status:
		return "user_enabled"
	case before.Role != after.Role:
		return "user_role_changed"
	}
	return "user_renamed"
}

func (s *Service) audit(ctx context.Context, q *store.Queries, actor, target uuid.UUID, action string, before, after map[string]any) error {
	enc := func(m map[string]any) json.RawMessage {
		if m == nil {
			return nil
		}
		b, _ := json.Marshal(m) // map chỉ chứa chuỗi / số: không lỗi
		return b
	}
	if _, err := q.InsertAuditLog(ctx, store.InsertAuditLogParams{ActorID: &actor, Entity: "user", EntityID: target.String(), Action: action, Before: enc(before), After: enc(after)}); err != nil {
		return fmt.Errorf("user: ghi audit_log: %w", err)
	}
	return nil
}

// ListFilter là query của GET /admin/users (đã kiểm ở handler). Role / Status rỗng = mọi giá trị.
type ListFilter struct {
	Role, Status, Q string
	Cursor          *Cursor
	Limit           int
}

// Cursor là dòng cuối của trang trước (created_at DESC, id DESC).
type Cursor struct {
	CreatedAt time.Time
	ID        uuid.UUID
}

// List trả tối đa Limit+1 dòng bằng MỘT truy vấn (dòng dư báo còn trang sau). q: tên không phân biệt dấu (chứa) hoặc tiền tố email.
func (s *Service) List(ctx context.Context, f ListFilter) ([]View, error) {
	p := store.ListUsersAdminParams{Lim: int32(f.Limit + 1)}
	if f.Role != "" {
		r := store.UserRole(f.Role)
		p.Role = &r
	}
	if f.Status != "" {
		st := store.UserStatus(f.Status)
		p.Status = &st
	}
	if q := strings.TrimSpace(f.Q); q != "" {
		name := "%" + likeEscape(auth.Fold(q)) + "%"
		email := likeEscape(strings.ToLower(q)) + "%"
		p.NameLike, p.EmailLike = &name, &email
	}
	if f.Cursor != nil {
		p.CurAt, p.CurID = &f.Cursor.CreatedAt, &f.Cursor.ID
	}
	rows, err := store.New(s.pool).ListUsersAdmin(ctx, p)
	if err != nil {
		return nil, fmt.Errorf("user: liệt kê: %w", err)
	}
	out := make([]View, len(rows))
	for i, r := range rows {
		out[i] = View{ID: r.ID, Email: r.Email, FullName: r.FullName, Role: string(r.Role), Status: string(r.Status), LastLoginAt: r.LastLoginAt, Version: int(r.Version), CreatedAt: r.CreatedAt}
	}
	return out, nil
}

// likeEscape thoát % _ \ để q của người dùng chỉ là chữ, không phải mẫu.
func likeEscape(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

// CreateAdmin tạo ADMIN ACTIVE đã xác minh (lệnh `gateway admin create`); idempotent theo email: đã có ⇒ created=false, không đổi gì.
// Mật khẩu qua chính sách (lỗi ⇒ *auth.ValidationError). Ghi audit_log `admin_bootstrap` với actor NULL.
func CreateAdmin(ctx context.Context, pool *pgxpool.Pool, clk clock.Clock, email, name, password string, bcryptCost int) (created bool, err error) {
	email = auth.NormalizeEmail(email)
	name = strings.TrimSpace(name)
	var ps []auth.FieldProblem
	if p := auth.ValidateEmail(email); p != nil {
		ps = append(ps, *p)
	}
	if p := auth.ValidateName(name); p != nil {
		ps = append(ps, *p)
	}
	if code := auth.ValidatePasswordPolicy(password, email); code != "" {
		ps = append(ps, auth.FieldProblem{Field: "password", Code: code, Message: auth.PasswordMessage(code)})
	}
	if len(ps) > 0 {
		return false, &auth.ValidationError{Problems: ps}
	}
	hash, err := auth.HashPassword(password, bcryptCost)
	if err != nil {
		return false, fmt.Errorf("user: băm mật khẩu: %w", err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("user: mở transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	u, err := q.InsertAdminUser(ctx, store.InsertAdminUserParams{Email: email, FullName: name, PasswordHash: &hash, Now: clk.Now()})
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("user: tạo quản trị viên: %w", err)
	}
	if _, err := q.InsertAuditLog(ctx, store.InsertAuditLogParams{Entity: "user", EntityID: u.ID.String(), Action: "admin_bootstrap", After: json.RawMessage(`{"role":"ADMIN"}`)}); err != nil {
		return false, fmt.Errorf("user: ghi audit_log: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("user: commit: %w", err)
	}
	return true, nil
}
