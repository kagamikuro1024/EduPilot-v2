package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/edupilot/backend-go/internal/platform/clock"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/store"
)

// MailQueue xếp một thư vào mail_outbox trong CÙNG transaction với nghiệp vụ (cài bởi internal/mail; auth không import mail
// vì consumer thư phải gọi auth.Tokens). dedupe rỗng = không khử trùng. Payload chỉ chứa định danh / dữ liệu hiển thị.
type MailQueue func(ctx context.Context, tx pgx.Tx, to, template string, payload map[string]any, dedupe string) error

// Các mẫu thư dùng ở đây (SRS 6.5).
const (
	tmplVerifyEmail   = "verify_email"
	tmplEmailExists   = "email_exists"
	tmplInviteStaff   = "invite_staff"
	tmplInviteStudent = "invite_student"
	tmplResetPass     = "reset_password"
	tmplPassChanged   = "password_changed"
	defaultInviter    = "Quản trị viên"
)

// Lỗi dùng chung cho handler.
var (
	// ErrResendThrottled: gửi lại trong cửa sổ giới hạn; RetryAfter ở ThrottledError.
	ErrResendThrottled = errors.New("auth: gửi lại quá nhanh")
)

// ThrottledError mang số giây còn phải chờ.
type ThrottledError struct{ RetryAfter int }

func (e *ThrottledError) Error() string   { return ErrResendThrottled.Error() }
func (e *ThrottledError) Is(t error) bool { return t == ErrResendThrottled }

// LinkError: liên kết một lần không dùng được; Reason ∈ expired | used | invalid (SRS 6.1).
type LinkError struct{ Reason string }

func (e *LinkError) Error() string {
	return "auth: liên kết không dùng được (" + e.Reason + ")"
}

// FieldProblem là một lỗi đầu vào (khớp apierr.FieldError, tách ra để auth không phụ thuộc HTTP).
type FieldProblem struct{ Field, Code, Message string }

// ValidationError gom lỗi theo trường.
type ValidationError struct{ Problems []FieldProblem }

func (e *ValidationError) Error() string { return "auth: dữ liệu chưa hợp lệ" }

// AccountsConfig là cấu hình của dịch vụ tài khoản.
type AccountsConfig struct {
	BcryptCost   int
	ResendWindow time.Duration
	VerifyTTL    time.Duration
	Limits       Limits
}

// Accounts: đăng ký, xác minh email, gửi lại thư xác minh. Gói duy nhất chạm auth_tokens (xem tokens.go).
type Accounts struct {
	pool *pgxpool.Pool
	rdb  *appredis.Client
	clk  clock.Clock
	mail MailQueue
	sess *Sessions // đặt khoá thu hồi (Redis) sau khi thu hồi phiên ở DB; nil = chỉ DB (test)
	cfg  AccountsConfig
	log  *slog.Logger
}

// NewAccounts dựng dịch vụ. rdb nil = bỏ giới hạn gửi lại (chỉ test).
func NewAccounts(pool *pgxpool.Pool, rdb *appredis.Client, clk clock.Clock, sess *Sessions, mail MailQueue, cfg AccountsConfig, log *slog.Logger) *Accounts {
	if clk == nil {
		clk = clock.Real{}
	}
	if log == nil {
		log = slog.Default()
	}
	if cfg.Limits == (Limits{}) {
		cfg.Limits = DefaultLimits()
	}
	if cfg.ResendWindow <= 0 {
		cfg.ResendWindow = 60 * time.Second
	}
	return &Accounts{pool: pool, rdb: rdb, clk: clk, sess: sess, mail: mail, cfg: cfg, log: log}
}

var (
	studentCodeRE = regexp.MustCompile(`^[A-Za-z0-9]{6,15}$`)
	emailRE       = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s.]+$`)
)

// RegisterInput là thân /auth/register đã đọc.
type RegisterInput struct {
	Email, Password, FullName, StudentCode string
}

// NormalizeAndValidate chuẩn hoá (cắt khoảng trắng, email chữ thường, MSSV chữ hoa) và kiểm; trả lỗi theo trường.
func (in RegisterInput) normalize() (RegisterInput, []FieldProblem) {
	var ps []FieldProblem
	out := RegisterInput{Email: NormalizeEmail(in.Email), FullName: strings.TrimSpace(in.FullName), StudentCode: strings.ToUpper(strings.TrimSpace(in.StudentCode)), Password: in.Password}
	if p := ValidateEmail(out.Email); p != nil {
		ps = append(ps, *p)
	}
	if p := ValidateName(out.FullName); p != nil {
		ps = append(ps, *p)
	}
	if out.StudentCode != "" && !studentCodeRE.MatchString(out.StudentCode) {
		ps = append(ps, FieldProblem{"student_code", "STUDENT_CODE_FORMAT", "Mã số sinh viên gồm 6–15 chữ và số."})
	}
	if code := ValidatePasswordPolicy(out.Password, out.Email); code != "" {
		ps = append(ps, FieldProblem{"password", code, PasswordMessage(code)})
	}
	return out, ps
}

// ValidateEmail kiểm email ĐÃ chuẩn hoá (NormalizeEmail): ≤ 254 ký tự, đúng dạng local@domain.tld, một dấu @, không ký tự điều khiển.
func ValidateEmail(email string) *FieldProblem {
	if len(email) == 0 || len(email) > 254 || !emailRE.MatchString(email) || strings.Count(email, "@") != 1 || hasControl(email) {
		return &FieldProblem{"email", "INVALID_EMAIL", "Email chưa đúng dạng, ví dụ ten@truong.edu.vn."}
	}
	return nil
}

// ValidateName kiểm họ tên ĐÃ cắt khoảng trắng: 1–100 ký tự, không ký tự điều khiển.
func ValidateName(name string) *FieldProblem {
	if n := utf8.RuneCountInString(name); n < 1 || n > 100 || hasControl(name) {
		return &FieldProblem{"full_name", "INVALID_NAME", "Họ và tên cần từ 1 đến 100 ký tự."}
	}
	return nil
}

func hasControl(s string) bool {
	return strings.ContainsFunc(s, func(r rune) bool { return unicode.IsControl(r) })
}

// Register tạo STUDENT chờ xác minh. Mọi trường hợp hợp lệ trả nil (thân phản hồi đồng nhất ở handler): email mới ⇒ thư xác minh;
// email đã có ⇒ thư `email_exists` (INVITED nhân viên ⇒ gửi lại lời mời; DISABLED ⇒ không gửi). bcrypt luôn chạy trước mọi nhánh
// để thời gian không lộ email có tồn tại (SRS 4.2.3).
func (a *Accounts) Register(ctx context.Context, in RegisterInput) error {
	in, problems := in.normalize()
	if len(problems) > 0 {
		return &ValidationError{Problems: problems}
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), a.cfg.BcryptCost)
	if err != nil {
		return fmt.Errorf("auth: băm mật khẩu: %w", err)
	}

	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("auth: mở transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)

	var code *string
	if in.StudentCode != "" {
		code = &in.StudentCode
	}
	h := string(hash)
	created, err := q.InsertPendingStudent(ctx, store.InsertPendingStudentParams{Email: in.Email, FullName: in.FullName, PasswordHash: &h, StudentCode: code})
	switch {
	case err == nil:
		if err := a.mail(ctx, tx, in.Email, tmplVerifyEmail, map[string]any{"user_id": created.ID.String(), "full_name": created.FullName}, ""); err != nil {
			return err
		}
	case errors.Is(err, pgx.ErrNoRows): // email đã có
		existing, gerr := q.GetUserByEmail(ctx, in.Email)
		if gerr != nil {
			return fmt.Errorf("auth: tra email: %w", gerr)
		}
		if err := a.mailExisting(ctx, tx, q, existing); err != nil {
			return err
		}
	default:
		return fmt.Errorf("auth: tạo người dùng: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("auth: commit đăng ký: %w", err)
	}
	return nil
}

// mailExisting gửi thư thích hợp tới chủ hộp thư của tài khoản đã tồn tại.
func (a *Accounts) mailExisting(ctx context.Context, tx pgx.Tx, q *store.Queries, u store.User) error {
	switch u.Status {
	case store.UserStatusDISABLED:
		return nil // không gửi gì
	case store.UserStatusINVITED:
		if u.Role == store.UserRoleTEACHER || u.Role == store.UserRoleTA {
			inviter, err := q.LatestInviterName(ctx, u.ID)
			if err != nil {
				inviter = defaultInviter
			}
			roleVN := "giảng viên"
			if u.Role == store.UserRoleTA {
				roleVN = "trợ giảng"
			}
			return a.mail(ctx, tx, u.Email, tmplInviteStaff, map[string]any{"user_id": u.ID.String(), "full_name": u.FullName, "inviter_name": inviter, "role_vn": roleVN}, "")
		}
		// Sinh viên INVITED (từ roster): gửi lại thư mời nêu lớp vừa thêm gần nhất tới CHỦ hộp thư. Chưa có lớp roster nào ⇒ không có gì để mời.
		ctxCourse, err := q.LatestRosterCourseForUser(ctx, u.ID)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("auth: tra lớp roster: %w", err)
		}
		teacher := ctxCourse.TeacherName
		if teacher == "" {
			teacher = "phụ trách"
		}
		return a.mail(ctx, tx, u.Email, tmplInviteStudent, map[string]any{"user_id": u.ID.String(), "full_name": u.FullName, "teacher_name": teacher, "course_name": ctxCourse.CourseName, "class_code": ctxCourse.ClassCode}, "")
	}
	return a.mail(ctx, tx, u.Email, tmplEmailExists, map[string]any{"full_name": u.FullName}, "")
}

// VerifyEmail dùng token xác minh một lần: đặt email_verified_at, ACTIVE, đẩy enrollment roster chờ. Lỗi *LinkError nêu lý do.
func (a *Accounts) VerifyEmail(ctx context.Context, plain string) error {
	now := a.clk.Now()
	hash := HashToken(strings.TrimSpace(plain))
	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("auth: mở transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)

	uid, err := q.ConsumeAuthToken(ctx, store.ConsumeAuthTokenParams{TokenHash: hash, Kind: store.AuthTokenKindVERIFYEMAIL, Now: now})
	if errors.Is(err, pgx.ErrNoRows) {
		return linkReason(ctx, q, hash, store.AuthTokenKindVERIFYEMAIL, now) // cùng transaction: không xin thêm kết nối khi pool đã cạn (50 yêu cầu đua nhau)
	}
	if err != nil {
		return fmt.Errorf("auth: dùng token: %w", err)
	}
	if _, err := q.MarkEmailVerified(ctx, store.MarkEmailVerifiedParams{ID: uid, Now: now}); err != nil {
		return fmt.Errorf("auth: xác minh email: %w", err)
	}
	if _, err := q.PromoteUnverifiedRosterEnrollments(ctx, store.PromoteUnverifiedRosterEnrollmentsParams{UserID: uid, Now: now}); err != nil {
		return fmt.Errorf("auth: đẩy danh sách lớp: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("auth: commit xác minh: %w", err)
	}
	return nil
}

// linkReason phân loại token không dùng được. Token lạ và token đã bị thay đều là "invalid".
func linkReason(ctx context.Context, q *store.Queries, hash string, kind store.AuthTokenKind, now time.Time) error {
	t, err := q.GetAuthTokenByHash(ctx, hash)
	switch {
	case errors.Is(err, pgx.ErrNoRows) || (err == nil && t.Kind != kind):
		return &LinkError{Reason: "invalid"}
	case err != nil:
		return fmt.Errorf("auth: tra token: %w", err)
	case t.RevokedAt != nil:
		return &LinkError{Reason: "invalid"}
	case t.UsedAt != nil:
		return &LinkError{Reason: "used"}
	case !now.Before(t.ExpiresAt):
		return &LinkError{Reason: "expired"}
	}
	return &LinkError{Reason: "invalid"}
}

// ResendVerification xếp lại thư xác minh. Giới hạn 1 lần / cửa sổ theo băm email, KHÔNG phân biệt email có tồn tại (không lộ).
// Tài khoản đã xác minh / không tồn tại / không ở trạng thái chờ ⇒ không gửi gì nhưng vẫn trả nil.
func (a *Accounts) ResendVerification(ctx context.Context, email string) error {
	email = NormalizeEmail(email)
	if err := a.throttleResend(ctx, "verify", email); err != nil {
		return err
	}
	q := store.New(a.pool)
	u, err := q.GetUserByEmail(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("auth: tra email: %w", err)
	}
	if u.EmailVerifiedAt != nil || u.Status != store.UserStatusPENDINGVERIFICATION {
		return nil
	}
	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("auth: mở transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := a.mail(ctx, tx, u.Email, tmplVerifyEmail, map[string]any{"user_id": u.ID.String(), "full_name": u.FullName}, ""); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// UserEmail trả email của user (resend khi đã đăng nhập, không kèm thân). Không có ⇒ ("", false).
func (a *Accounts) UserEmail(ctx context.Context, id uuid.UUID) (string, bool) {
	u, err := store.New(a.pool).GetUser(ctx, id)
	if err != nil {
		return "", false
	}
	return u.Email, true
}

// ThrottleResend: 1 lần / cửa sổ theo (kind, băm email) cho gửi lại thư (xác minh, mời). Vượt ⇒ *ThrottledError.
// throttleResend: SET NX khoá `ep:auth:resend:{kind}:{emailhash32}` TTL = cửa sổ. Redis lỗi ⇒ cho qua (log).
func (a *Accounts) ThrottleResend(ctx context.Context, kind, email string) error {
	return a.throttleResend(ctx, kind, email)
}

func (a *Accounts) throttleResend(ctx context.Context, kind, email string) error {
	if a.rdb == nil {
		return nil
	}
	key := appredis.Key("auth", "resend", kind, emailHash(email)[:32])
	ok, err := a.rdb.SetNX(ctx, key, "1", a.cfg.ResendWindow).Result()
	if err != nil {
		a.log.WarnContext(ctx, "auth: không kiểm được giới hạn gửi lại (Redis), cho qua", "error", err.Error())
		return nil
	}
	if ok {
		return nil
	}
	ttl, err := a.rdb.TTL(ctx, key).Result()
	sec := int((ttl + time.Second - 1) / time.Second)
	if err != nil || sec < 1 {
		sec = 1
	}
	return &ThrottledError{RetryAfter: min(sec, int(a.cfg.ResendWindow/time.Second))}
}
