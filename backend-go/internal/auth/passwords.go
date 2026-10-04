package auth

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/store"
)

// Quên / đặt lại / đổi mật khẩu, xem và thu hồi thiết bị (US-P2-04).

const (
	// forgotFloor: mọi nhánh của forgot-password mất ít nhất ngần này ⇒ thời gian không lộ email có tài khoản hay không (SRS 4.2.3).
	forgotFloor     = 80 * time.Millisecond
	forgotWindow    = time.Hour
	changePwWindow  = 10 * time.Minute
	maxSessionsList = 50
)

// ForgotPassword xếp thư `reset_password` cho tài khoản ACTIVE / PENDING_VERIFICATION. Mọi trường hợp khác (không có, INVITED,
// DISABLED) im lặng trả nil. Giới hạn 3 lần / giờ / băm email (kể cả email không tồn tại).
func (a *Accounts) ForgotPassword(ctx context.Context, email string) error {
	start := time.Now()
	email = NormalizeEmail(email)
	if err := a.hit(ctx, appredis.Key("auth", "forgot", emailHash(email)[:32]), forgotWindow, a.cfg.Limits.ForgotEmailPerHour); err != nil {
		return err
	}
	err := a.queueReset(ctx, email)
	if rest := forgotFloor - time.Since(start); rest > 0 {
		select {
		case <-time.After(rest):
		case <-ctx.Done():
		}
	}
	return err
}

func (a *Accounts) queueReset(ctx context.Context, email string) error {
	u, err := store.New(a.pool).GetUserByEmail(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("auth: tra email: %w", err)
	}
	if u.Status != store.UserStatusACTIVE && u.Status != store.UserStatusPENDINGVERIFICATION {
		return nil
	}
	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("auth: mở transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := a.mail(ctx, tx, u.Email, tmplResetPass, map[string]any{"user_id": u.ID.String(), "full_name": u.FullName}, ""); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ResetPassword dùng token đặt lại một lần: đổi mật khẩu, mở khoá, xác minh email, thu hồi MỌI phiên, xếp thư `password_changed`.
// Mật khẩu yếu ⇒ ValidationError và token KHÔNG bị tiêu (giao dịch lùi lại). Không tự đăng nhập.
func (a *Accounts) ResetPassword(ctx context.Context, plain, newPassword string) error {
	now := a.clk.Now()
	hash := HashToken(strings.TrimSpace(plain))
	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("auth: mở transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)

	uid, err := q.ConsumeAuthToken(ctx, store.ConsumeAuthTokenParams{TokenHash: hash, Kind: store.AuthTokenKindRESETPASSWORD, Now: now})
	if errors.Is(err, pgx.ErrNoRows) {
		return linkReason(ctx, q, hash, store.AuthTokenKindRESETPASSWORD, now)
	}
	if err != nil {
		return fmt.Errorf("auth: dùng token: %w", err)
	}
	u, err := q.GetUser(ctx, uid)
	if err != nil {
		return fmt.Errorf("auth: đọc người dùng: %w", err)
	}
	if code := ValidatePasswordPolicy(newPassword, u.Email); code != "" {
		return &ValidationError{Problems: []FieldProblem{{"new_password", code, PasswordMessage(code)}}} // defer Rollback ⇒ token còn nguyên
	}
	pw, err := HashPassword(newPassword, a.cfg.BcryptCost)
	if err != nil {
		return fmt.Errorf("auth: băm mật khẩu: %w", err)
	}
	u, err = q.ResetUserPassword(ctx, store.ResetUserPasswordParams{ID: uid, PasswordHash: &pw, Now: now})
	if err != nil {
		return fmt.Errorf("auth: đặt mật khẩu: %w", err)
	}
	ids, err := q.RevokeUserSessions(ctx, store.RevokeUserSessionsParams{UserID: uid, Now: now, Reason: ReasonPasswordReset})
	if err != nil {
		return fmt.Errorf("auth: thu hồi phiên: %w", err)
	}
	if err := a.mail(ctx, tx, u.Email, tmplPassChanged, map[string]any{"full_name": u.FullName, "at": now.UTC().Format(time.RFC3339)}, ""); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("auth: commit đặt lại mật khẩu: %w", err)
	}
	a.markRevoked(ctx, ids)
	if a.sess != nil {
		a.sess.ClearFailures(ctx, emailHash(u.Email), u.ID) // đặt lại mật khẩu mở khoá ngay (Redis; cột DB đã xoá trong giao dịch)
	}
	return nil
}

// ChangePassword đổi mật khẩu của người đang đăng nhập; thu hồi mọi phiên KHÁC `currentSID` (uuid.Nil = không giữ phiên nào).
// Sai mật khẩu hiện tại quá 5 lần / 10 phút ⇒ ThrottledError. Ghi audit_log (không chứa mật khẩu) và xếp thư `password_changed`.
func (a *Accounts) ChangePassword(ctx context.Context, userID, currentSID uuid.UUID, current, next string) error {
	now := a.clk.Now()
	bucket := appredis.Key("rl", "auth", "chgpw", userID.String(), strconv.FormatInt(now.Unix()/int64(changePwWindow/time.Second), 10))
	if a.rdb != nil {
		n, err := a.rdb.Get(ctx, bucket).Int()
		if err == nil && n >= a.cfg.Limits.ChangePWFailPer10m {
			return &ThrottledError{RetryAfter: secondsToNextWindow(now, changePwWindow)}
		}
	}
	u, err := store.New(a.pool).GetUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("auth: đọc người dùng: %w", err)
	}
	if u.PasswordHash == nil || CheckPassword(*u.PasswordHash, current) != nil {
		a.countWrong(ctx, bucket)
		return &ValidationError{Problems: []FieldProblem{{"current_password", "WRONG_PASSWORD", "Mật khẩu hiện tại chưa đúng."}}}
	}
	if current == next {
		return &ValidationError{Problems: []FieldProblem{{"new_password", PasswordSameAsOld, PasswordMessage(PasswordSameAsOld)}}}
	}
	if code := ValidatePasswordPolicy(next, u.Email); code != "" {
		return &ValidationError{Problems: []FieldProblem{{"new_password", code, PasswordMessage(code)}}}
	}
	pw, err := HashPassword(next, a.cfg.BcryptCost)
	if err != nil {
		return fmt.Errorf("auth: băm mật khẩu: %w", err)
	}

	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("auth: mở transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	if err := q.ChangeUserPassword(ctx, store.ChangeUserPasswordParams{ID: userID, PasswordHash: &pw}); err != nil {
		return fmt.Errorf("auth: đổi mật khẩu: %w", err)
	}
	var except *uuid.UUID
	if currentSID != uuid.Nil {
		except = &currentSID
	}
	ids, err := q.RevokeUserSessions(ctx, store.RevokeUserSessionsParams{UserID: userID, Now: now, Reason: ReasonPasswordChange, ExceptID: except})
	if err != nil {
		return fmt.Errorf("auth: thu hồi phiên: %w", err)
	}
	if _, err := q.InsertAuditLog(ctx, store.InsertAuditLogParams{
		ActorID: &userID, Entity: "user", EntityID: userID.String(), Action: "password_changed",
		After: []byte(fmt.Sprintf(`{"revoked_sessions":%d}`, len(ids))), // không mật khẩu, không băm
	}); err != nil {
		return fmt.Errorf("auth: ghi audit_log: %w", err)
	}
	if err := a.mail(ctx, tx, u.Email, tmplPassChanged, map[string]any{"full_name": u.FullName, "at": now.UTC().Format(time.RFC3339)}, ""); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("auth: commit đổi mật khẩu: %w", err)
	}
	a.markRevoked(ctx, ids)
	return nil
}

func (a *Accounts) countWrong(ctx context.Context, key string) {
	if a.rdb == nil {
		return
	}
	pipe := a.rdb.Pipeline()
	pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, 2*changePwWindow)
	if _, err := pipe.Exec(ctx); err != nil {
		a.log.WarnContext(ctx, "auth: không đếm được lần sai đổi mật khẩu (Redis)", "error", err.Error())
	}
}

// hit đếm một lần trong cửa sổ cố định bắt đầu từ lần đầu; vượt `limit` ⇒ ThrottledError. Redis lỗi ⇒ cho qua (log).
func (a *Accounts) hit(ctx context.Context, key string, window time.Duration, limit int) error {
	if a.rdb == nil {
		return nil
	}
	pipe := a.rdb.Pipeline()
	incr := pipe.Incr(ctx, key)
	pipe.ExpireNX(ctx, key, window)
	ttl := pipe.TTL(ctx, key)
	if _, err := pipe.Exec(ctx); err != nil {
		a.log.WarnContext(ctx, "auth: không kiểm được giới hạn (Redis), cho qua", "error", err.Error())
		return nil
	}
	if int(incr.Val()) <= limit {
		return nil
	}
	sec := int((ttl.Val() + time.Second - 1) / time.Second)
	return &ThrottledError{RetryAfter: min(max(sec, 1), int(window/time.Second))}
}

func secondsToNextWindow(now time.Time, window time.Duration) int {
	w := int64(window / time.Second)
	return int(w - now.Unix()%w)
}

func (a *Accounts) markRevoked(ctx context.Context, ids []uuid.UUID) {
	if a.sess == nil {
		return
	}
	for _, id := range ids {
		a.sess.markRevoked(ctx, id)
	}
}

// TokenPreview mô tả một token chưa dùng, KHÔNG tiêu. RESET_PASSWORD | INVITE (VERIFY_EMAIL ⇒ ValidationError).
type TokenPreview struct {
	Kind      string
	ExpiresAt time.Time
	FullName  string // chỉ INVITE
	Role      string // chỉ INVITE
}

// PreviewToken kiểm token cho trang đặt lại / nhận lời mời. Giới hạn theo IP do middleware ở tầng HTTP lo (action "token").
func (a *Accounts) PreviewToken(ctx context.Context, kind, plain string) (TokenPreview, error) {
	var k store.AuthTokenKind
	switch kind {
	case "RESET_PASSWORD":
		k = store.AuthTokenKindRESETPASSWORD
	case "INVITE":
		k = store.AuthTokenKindINVITE
	default:
		return TokenPreview{}, &ValidationError{Problems: []FieldProblem{{"kind", "INVALID_KIND", "Loại liên kết không hợp lệ."}}}
	}
	now := a.clk.Now()
	q := store.New(a.pool)
	hash := HashToken(strings.TrimSpace(plain))
	t, err := q.GetAuthTokenByHash(ctx, hash)
	if err == nil && t.Kind == k && t.UsedAt == nil && t.RevokedAt == nil && now.Before(t.ExpiresAt) {
		p := TokenPreview{Kind: kind, ExpiresAt: t.ExpiresAt}
		if k == store.AuthTokenKindINVITE {
			u, err := q.GetUser(ctx, t.UserID)
			if err != nil {
				return TokenPreview{}, fmt.Errorf("auth: đọc người dùng: %w", err)
			}
			if u.Status != store.UserStatusINVITED { // đã nhận / đã bị khoá: liên kết không còn dùng được
				return TokenPreview{}, &LinkError{Reason: "invalid"}
			}
			p.FullName, p.Role = u.FullName, string(u.Role)
		}
		return p, nil
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return TokenPreview{}, fmt.Errorf("auth: tra token: %w", err)
	}
	return TokenPreview{}, linkReason(ctx, q, hash, k, now)
}

// SessionInfo là một thiết bị đang đăng nhập (không có refresh hash, không user agent nguyên văn, IP đã che).
type SessionInfo struct {
	ID          uuid.UUID
	Current     bool
	DeviceLabel string
	IPMasked    string
	CreatedAt   time.Time
	LastUsedAt  time.Time
}

// ListSessions: ≤ 50 phiên còn hiệu lực của chính `userID`; phiên `currentSID` đứng đầu.
func (a *Accounts) ListSessions(ctx context.Context, userID, currentSID uuid.UUID) ([]SessionInfo, error) {
	var cur *uuid.UUID
	if currentSID != uuid.Nil {
		cur = &currentSID
	}
	rows, err := store.New(a.pool).ListOwnSessions(ctx, store.ListOwnSessionsParams{UserID: userID, Now: a.clk.Now(), CurrentID: cur})
	if err != nil {
		return nil, fmt.Errorf("auth: liệt kê phiên: %w", err)
	}
	out := make([]SessionInfo, len(rows))
	for i, r := range rows {
		out[i] = SessionInfo{ID: r.ID, Current: r.IsCurrent, IPMasked: MaskIP(r.Ip), CreatedAt: r.CreatedAt, LastUsedAt: r.LastUsedAt}
		if r.DeviceLabel != nil {
			out[i].DeviceLabel = *r.DeviceLabel
		}
	}
	return out, nil
}

// MaskIP che nửa sau địa chỉ: 203.0.113.9 ⇒ 203.0.*.*, 2001:db8:1:2::1 ⇒ 2001:db8:*:*. IP đầy đủ không bao giờ rời gateway.
func MaskIP(ip *netip.Addr) string {
	if ip == nil || !ip.IsValid() {
		return ""
	}
	if ip.Is4In6() {
		a := ip.Unmap()
		ip = &a
	}
	if ip.Is4() {
		b := ip.As4()
		return fmt.Sprintf("%d.%d.*.*", b[0], b[1])
	}
	b := ip.As16()
	return fmt.Sprintf("%x:%x:*:*", uint16(b[0])<<8|uint16(b[1]), uint16(b[2])<<8|uint16(b[3]))
}

// ErrSessionNotFound: phiên không tồn tại, đã chết, hoặc của người khác (không phân biệt — chống IDOR).
var ErrSessionNotFound = errors.New("auth: không có phiên này")

// RevokeSession thu hồi MỘT phiên của chính `userID`.
func (a *Accounts) RevokeSession(ctx context.Context, userID, sessionID uuid.UUID) error {
	id, err := store.New(a.pool).RevokeOwnSession(ctx, store.RevokeOwnSessionParams{ID: sessionID, UserID: userID, Now: a.clk.Now()})
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrSessionNotFound
	}
	if err != nil {
		return fmt.Errorf("auth: thu hồi phiên: %w", err)
	}
	a.markRevoked(ctx, []uuid.UUID{id})
	return nil
}

// RevokeOtherSessions thu hồi mọi phiên của `userID` trừ `currentSID`; trả số phiên.
func (a *Accounts) RevokeOtherSessions(ctx context.Context, userID, currentSID uuid.UUID) (int, error) {
	var except *uuid.UUID
	if currentSID != uuid.Nil {
		except = &currentSID
	}
	ids, err := store.New(a.pool).RevokeUserSessions(ctx, store.RevokeUserSessionsParams{UserID: userID, Now: a.clk.Now(), Reason: ReasonRevokedByUser, ExceptID: except})
	if err != nil {
		return 0, fmt.Errorf("auth: thu hồi phiên: %w", err)
	}
	a.markRevoked(ctx, ids)
	return len(ids), nil
}

// AcceptInvite dùng token INVITE một lần: đặt mật khẩu (qua chính sách), ACTIVE, email đã xác minh, mở phiên mới — tất cả một giao dịch.
// Mật khẩu yếu ⇒ ValidationError và token KHÔNG bị tiêu. Tài khoản không còn INVITED (đã khoá…) ⇒ liên kết "invalid".
func (a *Accounts) AcceptInvite(ctx context.Context, plain, password, userAgent, ip string) (Result, error) {
	if a.sess == nil {
		return Result{}, errors.New("auth: thiếu dịch vụ phiên")
	}
	now := a.clk.Now()
	hash := HashToken(strings.TrimSpace(plain))
	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("auth: mở transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)

	uid, err := q.ConsumeAuthToken(ctx, store.ConsumeAuthTokenParams{TokenHash: hash, Kind: store.AuthTokenKindINVITE, Now: now})
	if errors.Is(err, pgx.ErrNoRows) {
		return Result{}, linkReason(ctx, q, hash, store.AuthTokenKindINVITE, now)
	}
	if err != nil {
		return Result{}, fmt.Errorf("auth: dùng token: %w", err)
	}
	u, err := q.GetUser(ctx, uid)
	if err != nil {
		return Result{}, fmt.Errorf("auth: đọc người dùng: %w", err)
	}
	if code := ValidatePasswordPolicy(password, u.Email); code != "" {
		return Result{}, &ValidationError{Problems: []FieldProblem{{"password", code, PasswordMessage(code)}}}
	}
	pw, err := HashPassword(password, a.cfg.BcryptCost)
	if err != nil {
		return Result{}, fmt.Errorf("auth: băm mật khẩu: %w", err)
	}
	u, err = q.ActivateInvitedUser(ctx, store.ActivateInvitedUserParams{ID: uid, PasswordHash: &pw, Now: now})
	if errors.Is(err, pgx.ErrNoRows) {
		return Result{}, &LinkError{Reason: "invalid"} // không còn ở trạng thái được mời (đã khoá…): giao dịch lùi, token chưa tiêu
	}
	if err != nil {
		return Result{}, fmt.Errorf("auth: kích hoạt tài khoản: %w", err)
	}
	res, err := a.sess.CreateSessionFor(ctx, q, u, userAgent, ip)
	if err != nil {
		return Result{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Result{}, fmt.Errorf("auth: commit nhận lời mời: %w", err)
	}
	return res, nil
}
