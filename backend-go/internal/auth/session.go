package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/edupilot/backend-go/internal/platform/clock"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/store"
)

// Lý do thu hồi phiên (enum CHECK của auth_sessions.revoked_reason, SRS FEAT-account-security 5.2).
const (
	ReasonLogout         = "LOGOUT"
	ReasonRevokedByUser  = "REVOKED_BY_USER"
	ReasonPasswordChange = "PASSWORD_CHANGED"
	ReasonPasswordReset  = "PASSWORD_RESET"
	ReasonRefreshReuse   = "REFRESH_REUSE"
	ReasonAccountOff     = "ACCOUNT_DISABLED"
	ReasonRoleChanged    = "ROLE_CHANGED"
	ReasonAdmin          = "ADMIN"
)

// revocationGrace: khoá thu hồi sống thêm chừng này sau hạn access token để chịu lệch đồng hồ.
const revocationGrace = 60 * time.Second

// refreshBytes: 32 byte → 43 ký tự base64url (cùng cỡ với token một lần).
const refreshBytes = 32

// Lỗi nghiệp vụ của phiên; handler ánh xạ sang mã API.
var (
	ErrInvalidCredentials = errors.New("auth: thông tin đăng nhập không đúng")
	ErrAccountDisabled    = errors.New("auth: tài khoản đã bị khoá")
	ErrRefreshInvalid     = errors.New("auth: refresh token không hợp lệ hoặc đã hết hạn")
)

// SessionRevokedError: phiên đã bị thu hồi; Reason là chữ thường theo SRS 6.1 ("logout", "refresh_reuse", …), rỗng nếu không biết.
type SessionRevokedError struct{ Reason string }

func (e *SessionRevokedError) Error() string { return "auth: phiên đã bị thu hồi" }

// SessionConfig là các hạn của phiên (ACCESS_TOKEN_TTL, REFRESH_TOKEN_TTL, SESSION_ABSOLUTE_TTL) và cost bcrypt.
type SessionConfig struct {
	AccessTTL   time.Duration
	RefreshTTL  time.Duration
	AbsoluteTTL time.Duration
	BcryptCost  int
	Limits      Limits    // ngưỡng chờ / khoá / chặn IP (US-P2-05)
	Mail        MailQueue // xếp thư account_locked cùng transaction; nil = không gửi (test)
}

// Sessions là dịch vụ phiên: đăng nhập, làm mới xoay vòng, đăng xuất, thu hồi. Gói duy nhất chạm auth_sessions.
type Sessions struct {
	pool   *pgxpool.Pool
	rdb    *appredis.Client
	clk    clock.Clock
	issuer *Issuer
	thr    *Throttle // nil khi không có Redis
	cfg    SessionConfig
	log    *slog.Logger

	dummyOnce sync.Once
	dummyHash []byte

	warnMu   sync.Mutex
	warnLast time.Time
}

// NewSessions dựng dịch vụ. issuer phải ký với đúng cfg.AccessTTL.
func NewSessions(pool *pgxpool.Pool, rdb *appredis.Client, clk clock.Clock, issuer *Issuer, cfg SessionConfig, log *slog.Logger) *Sessions {
	if clk == nil {
		clk = clock.Real{}
	}
	if log == nil {
		log = slog.Default()
	}
	if cfg.Limits == (Limits{}) {
		cfg.Limits = DefaultLimits()
	}
	s := &Sessions{pool: pool, rdb: rdb, clk: clk, issuer: issuer, cfg: cfg, log: log}
	if rdb != nil {
		s.thr = newThrottle(rdb, cfg.Limits, log)
	}
	return s
}

// UserInfo là phần người dùng trả cho client sau đăng nhập / làm mới.
type UserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	FullName      string `json:"full_name"`
	Role          string `json:"role"`
	Status        string `json:"status"`
	EmailVerified bool   `json:"email_verified"`
}

// Result là kết quả đăng nhập / làm mới. RefreshToken chỉ đi vào cookie, không vào thân phản hồi.
type Result struct {
	AccessToken  string
	ExpiresIn    int
	RefreshToken string
	RefreshTTL   time.Duration // còn bao lâu cookie hết hạn (không quá hạn tuyệt đối)
	User         UserInfo
}

// LoginInput là đầu vào đăng nhập; UserAgent/IP dùng cho nhãn thiết bị và nhật ký.
type LoginInput struct {
	Email, Password, UserAgent, IP string
}

// NormalizeEmail cắt khoảng trắng và hạ chữ thường.
func NormalizeEmail(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

// HashToken ở tokens.go; emailHash là sha256 hex của email đã chuẩn hoá (khoá chống dò, nhật ký đăng nhập).
func emailHash(email string) string { return HashToken(email) }

func (s *Sessions) dummy() []byte {
	s.dummyOnce.Do(func() {
		h, err := bcrypt.GenerateFromPassword([]byte("edupilot-dummy-password"), s.cfg.BcryptCost)
		if err == nil {
			s.dummyHash = h
		}
	})
	return s.dummyHash
}

// LoginThrottledError: đang phải chờ / bị khoá / IP bị chặn (429 LOGIN_THROTTLED). Không phân biệt lý do với client (SRS 4.2.3).
type LoginThrottledError struct{ RetryAfter int }

func (e *LoginThrottledError) Error() string {
	return "auth: đăng nhập đang bị chờ hoặc khoá"
}

// Login xác thực email + mật khẩu. Thứ tự: (1) IP bị chặn? (2) email đang chờ / khoá? — cả hai trả LOGIN_THROTTLED TRƯỚC khi kiểm mật khẩu;
// (3) luôn chạy đúng một phép bcrypt (kể cả email lạ) để thời gian không lộ tài khoản có tồn tại; (4) sai ⇒ ghi lần sai (chờ tăng dần / khoá).
func (s *Sessions) Login(ctx context.Context, in LoginInput) (Result, error) {
	email := NormalizeEmail(in.Email)
	q := store.New(s.pool)
	ip := parseIP(in.IP)
	now := s.clk.Now()
	eh := emailHash(email)

	u, err := q.GetUserByEmail(ctx, email)
	found := err == nil
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return Result{}, fmt.Errorf("auth: tra người dùng: %w", err)
	}
	record := func(o store.LoginOutcome) {
		var uid *uuid.UUID
		if found {
			uid = &u.ID
		}
		if err := q.InsertLoginAttempt(ctx, store.InsertLoginAttemptParams{
			EmailHash: eh, UserID: uid, Ip: ip, UserAgent: clip(in.UserAgent, 200), Outcome: o, Now: now,
		}); err != nil {
			s.log.WarnContext(ctx, "ghi login_attempts lỗi", "error", err.Error())
		}
	}

	if secs, outcome := s.mustWait(ctx, in.IP, eh, found, u, now); secs > 0 {
		record(outcome)
		return Result{}, &LoginThrottledError{RetryAfter: secs}
	}

	hash := s.dummy()
	if found && u.PasswordHash != nil && *u.PasswordHash != "" {
		hash = []byte(*u.PasswordHash)
	}
	pwOK := len(in.Password) <= MaxPasswordBytes && bcrypt.CompareHashAndPassword(hash, []byte(in.Password)) == nil
	usable := found && u.PasswordHash != nil && *u.PasswordHash != "" && u.Status != store.UserStatusINVITED
	if !pwOK || !usable {
		if found {
			record(store.LoginOutcomeBADPASSWORD)
		} else {
			record(store.LoginOutcomeUNKNOWNEMAIL)
		}
		s.recordFailure(ctx, in.IP, eh, found, u, now)
		return Result{}, ErrInvalidCredentials
	}
	if u.Status == store.UserStatusDISABLED {
		record(store.LoginOutcomeDISABLED)
		return Result{}, ErrAccountDisabled
	}

	res, err := s.createSession(ctx, q, u, in.UserAgent, ip, now)
	if err != nil {
		return Result{}, err
	}
	if err := q.TouchLastLogin(ctx, store.TouchLastLoginParams{ID: u.ID, Now: now}); err != nil {
		s.log.WarnContext(ctx, "cập nhật last_login_at lỗi", "error", err.Error())
	}
	s.ClearFailures(ctx, eh, u.ID)
	record(store.LoginOutcomeSUCCESS)
	return res, nil
}

// mustWait cho biết còn phải chờ bao lâu (giây) trước khi được thử mật khẩu. Redis là nguồn nhanh; cột users.locked_until là nguồn bền:
// Redis mất thì lịch chờ tăng dần mất theo (chấp nhận, SRS AC5) nhưng khoá 15 phút của tài khoản thật vẫn được thi hành.
func (s *Sessions) mustWait(ctx context.Context, ip, eh string, found bool, u store.User, now time.Time) (int, store.LoginOutcome) {
	secs, outcome := 0, store.LoginOutcomeTHROTTLED
	if s.thr != nil {
		if ip != "" {
			if sec, err := s.thr.IPBlocked(ctx, ip); err != nil {
				s.warnRedis(ctx, err)
			} else if sec > 0 {
				return sec, store.LoginOutcomeTHROTTLED
			}
		}
		if st, err := s.thr.State(ctx, eh); err != nil {
			s.warnRedis(ctx, err)
		} else if w, locked := st.wait(now, s.cfg.Limits); w > 0 {
			secs = w
			if locked {
				outcome = store.LoginOutcomeLOCKED
			}
		}
	}
	if found && u.LockedUntil != nil && now.Before(*u.LockedUntil) {
		if w := ceilSecs(u.LockedUntil.Sub(now)); w > secs {
			secs, outcome = w, store.LoginOutcomeLOCKED
		}
	}
	return secs, outcome
}

// recordFailure ghi một lần sai: Redis (cho cả email lạ) + cột DB (tài khoản thật); đủ ngưỡng ⇒ khoá và xếp ĐÚNG MỘT thư account_locked.
func (s *Sessions) recordFailure(ctx context.Context, ip, eh string, found bool, u store.User, now time.Time) {
	lim := s.cfg.Limits
	n, lockedTo, fresh, redisOK := 0, time.Time{}, false, false
	if s.thr != nil {
		var err error
		if n, lockedTo, fresh, err = s.thr.Fail(ctx, eh, now); err != nil {
			s.warnRedis(ctx, err)
		} else {
			redisOK = true
		}
		if ip != "" {
			if err := s.thr.IPFail(ctx, ip); err != nil {
				s.warnRedis(ctx, err)
			}
		}
	}
	if !found {
		return
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		s.log.ErrorContext(ctx, "auth: mở transaction ghi lần sai lỗi", "error", err.Error())
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	if redisOK {
		var lock *time.Time
		if !lockedTo.IsZero() {
			lock = &lockedTo
		}
		err = q.SetLoginFailures(ctx, store.SetLoginFailuresParams{ID: u.ID, N: int32(n), LockedUntil: lock})
	} else {
		var row store.IncrementFailedLoginRow
		if row, err = q.IncrementFailedLogin(ctx, store.IncrementFailedLoginParams{ID: u.ID, LockAt: int32(lim.LockAt), Until: now.Add(lim.LockDuration)}); err == nil {
			n = int(row.FailedLogins)
			if row.LockedUntil != nil {
				lockedTo = *row.LockedUntil
			}
			fresh = n == lim.LockAt
		}
	}
	if err != nil {
		s.log.ErrorContext(ctx, "auth: ghi bộ đếm sai ở DB lỗi", "error", err.Error())
		return
	}
	if fresh && s.cfg.Mail != nil {
		dedupe := "account_locked:" + u.ID.String() + ":" + strconv.FormatInt(lockedTo.Unix(), 10)
		if err := s.cfg.Mail(ctx, tx, u.Email, "account_locked", map[string]any{"full_name": u.FullName, "until": lockedTo.UTC().Format(time.RFC3339)}, dedupe); err != nil {
			s.log.ErrorContext(ctx, "auth: xếp thư account_locked lỗi", "error", err.Error())
			return
		}
	}
	if err := tx.Commit(ctx); err != nil {
		s.log.ErrorContext(ctx, "auth: commit bộ đếm sai lỗi", "error", err.Error())
	}
}

// ClearFailures đưa bộ đếm sai về 0 (đăng nhập đúng, đặt lại mật khẩu): Redis + cột DB.
func (s *Sessions) ClearFailures(ctx context.Context, eh string, userID uuid.UUID) {
	if s.thr != nil {
		if err := s.thr.Clear(ctx, eh); err != nil {
			s.warnRedis(ctx, err)
		}
	}
	if err := store.New(s.pool).ClearLoginFailures(ctx, userID); err != nil {
		s.log.WarnContext(ctx, "auth: xoá bộ đếm sai ở DB lỗi", "error", err.Error())
	}
}

// CreateSessionFor mở phiên cho người dùng đã được xác thực bằng cách khác (chấp nhận lời mời, US-P2-06). Không kiểm mật khẩu.
func (s *Sessions) CreateSessionFor(ctx context.Context, q *store.Queries, u store.User, userAgent, ip string) (Result, error) {
	return s.createSession(ctx, q, u, userAgent, parseIP(ip), s.clk.Now())
}

func (s *Sessions) createSession(ctx context.Context, q *store.Queries, u store.User, ua string, ip *netip.Addr, now time.Time) (Result, error) {
	refresh, err := newRefreshToken()
	if err != nil {
		return Result{}, err
	}
	abs := now.Add(s.cfg.AbsoluteTTL)
	exp := minTime(now.Add(s.cfg.RefreshTTL), abs)
	label := deviceLabel(ua)
	sess, err := q.InsertAuthSession(ctx, store.InsertAuthSessionParams{
		UserID: u.ID, RefreshHash: HashToken(refresh), UserAgent: clip(ua, 300), DeviceLabel: &label, Ip: ip,
		Now: now, ExpiresAt: exp, AbsoluteExpiresAt: abs,
	})
	if err != nil {
		return Result{}, fmt.Errorf("auth: tạo phiên: %w", err)
	}
	return s.result(u, sess.ID, refresh, exp.Sub(now))
}

func (s *Sessions) result(u store.User, sid uuid.UUID, refresh string, refreshTTL time.Duration) (Result, error) {
	access, err := s.issuer.IssueSession(u.ID.String(), Role(u.Role), u.Email, sid.String())
	if err != nil {
		return Result{}, fmt.Errorf("auth: ký access token: %w", err)
	}
	return Result{
		AccessToken: access, ExpiresIn: int(s.cfg.AccessTTL / time.Second), RefreshToken: refresh, RefreshTTL: refreshTTL,
		User: UserInfo{
			ID: u.ID.String(), Email: u.Email, FullName: u.FullName, Role: string(u.Role), Status: string(u.Status),
			EmailVerified: u.EmailVerifiedAt != nil,
		},
	}, nil
}

// Refresh đổi refresh token hiện tại lấy access token mới và refresh token mới (xoay vòng).
// Token ở thế hệ trước (prev) = dùng lại → thu hồi cả phiên. Không có ân hạn (Q9).
func (s *Sessions) Refresh(ctx context.Context, refresh, userAgent, ip string) (Result, error) {
	hash := HashToken(refresh)
	now := s.clk.Now()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("auth: mở transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)

	sess, err := q.LockAuthSessionByRefresh(ctx, hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return Result{}, ErrRefreshInvalid
	}
	if err != nil {
		return Result{}, fmt.Errorf("auth: khoá phiên: %w", err)
	}
	if sess.RevokedAt != nil {
		return Result{}, &SessionRevokedError{Reason: reasonOut(sess.RevokedReason)}
	}
	if sess.PrevRefreshHash != nil && *sess.PrevRefreshHash == hash { // token cũ đã bị xoay ⇒ bị đánh cắp hoặc đua
		if _, err := q.RevokeAuthSession(ctx, store.RevokeAuthSessionParams{ID: sess.ID, Now: now, Reason: ReasonRefreshReuse}); err != nil {
			return Result{}, fmt.Errorf("auth: thu hồi phiên: %w", err)
		}
		uid := sess.UserID
		if _, err := q.InsertAuditLog(ctx, store.InsertAuditLogParams{
			ActorID: &uid, Entity: "auth_session", EntityID: sess.ID.String(), Action: "refresh_reuse",
		}); err != nil {
			return Result{}, fmt.Errorf("auth: ghi audit: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return Result{}, fmt.Errorf("auth: commit thu hồi: %w", err)
		}
		s.markRevoked(ctx, sess.ID)
		return Result{}, &SessionRevokedError{Reason: "refresh_reuse"}
	}
	if !now.Before(sess.ExpiresAt) || !now.Before(sess.AbsoluteExpiresAt) {
		return Result{}, ErrRefreshInvalid
	}
	u, err := q.GetUser(ctx, sess.UserID)
	if err != nil {
		return Result{}, fmt.Errorf("auth: đọc người dùng: %w", err)
	}
	if u.Status == store.UserStatusDISABLED {
		if _, err := q.RevokeAuthSession(ctx, store.RevokeAuthSessionParams{ID: sess.ID, Now: now, Reason: ReasonAccountOff}); err != nil {
			return Result{}, fmt.Errorf("auth: thu hồi phiên: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return Result{}, fmt.Errorf("auth: commit thu hồi: %w", err)
		}
		s.markRevoked(ctx, sess.ID)
		return Result{}, &SessionRevokedError{Reason: "account_disabled"}
	}

	next, err := newRefreshToken()
	if err != nil {
		return Result{}, err
	}
	exp := minTime(now.Add(s.cfg.RefreshTTL), sess.AbsoluteExpiresAt)
	if err := q.RotateAuthSession(ctx, store.RotateAuthSessionParams{ID: sess.ID, NewHash: HashToken(next), Now: now, ExpiresAt: exp}); err != nil {
		return Result{}, fmt.Errorf("auth: xoay refresh: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Result{}, fmt.Errorf("auth: commit xoay: %w", err)
	}
	return s.result(u, sess.ID, next, exp.Sub(now))
}

// Logout thu hồi phiên của refresh token (nếu còn). Luôn thành công — kể cả khi token lạ hoặc phiên đã chết.
func (s *Sessions) Logout(ctx context.Context, refresh string) error {
	if refresh == "" {
		return nil
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("auth: mở transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	sess, err := q.LockAuthSessionByRefresh(ctx, HashToken(refresh))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("auth: khoá phiên: %w", err)
	}
	n, err := q.RevokeAuthSession(ctx, store.RevokeAuthSessionParams{ID: sess.ID, Now: s.clk.Now(), Reason: ReasonLogout})
	if err != nil {
		return fmt.Errorf("auth: thu hồi phiên: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("auth: commit: %w", err)
	}
	if n > 0 {
		s.markRevoked(ctx, sess.ID)
	}
	return nil
}

// markRevoked đặt khoá thu hồi theo phiên ở Redis để access token còn hạn bị từ chối ≤ 1 s. Redis lỗi: chỉ log
// (cơ sở dữ liệu vẫn đúng; access token tự hết hạn ≤ ACCESS_TOKEN_TTL).
func (s *Sessions) markRevoked(ctx context.Context, sid uuid.UUID) {
	if s.rdb == nil {
		return
	}
	if err := s.rdb.Set(context.WithoutCancel(ctx), revSidKey(sid.String()), "1", s.cfg.AccessTTL+revocationGrace).Err(); err != nil {
		s.log.ErrorContext(ctx, "đặt khoá thu hồi phiên lỗi", "error", err.Error())
	}
}

// Revoked cho Middleware biết access token (đã qua chữ ký) có bị thu hồi không; reason chỉ khi biết.
// Redis không với tới → chấp nhận token, log error tối đa mỗi 30 s (SRS 3.4).
func (s *Sessions) Revoked(ctx context.Context, p Principal) (bool, string) {
	if s.rdb == nil {
		return false, ""
	}
	keys := []string{revUserKey(p.Sub)}
	if p.SessionID != "" {
		keys = append(keys, revSidKey(p.SessionID))
	}
	vals, err := s.rdb.MGet(ctx, keys...).Result()
	if err != nil {
		s.warnRedis(ctx, err)
		return false, ""
	}
	if cut, ok := vals[0].(string); ok {
		if ms, perr := strconv.ParseInt(cut, 10, 64); perr == nil && p.IssuedAt.Unix() < ms/1000 {
			return true, ""
		}
	}
	if len(vals) > 1 && vals[1] != nil {
		return true, s.reasonFor(ctx, p.SessionID)
	}
	return false, ""
}

func (s *Sessions) reasonFor(ctx context.Context, sid string) string {
	id, err := uuid.Parse(sid)
	if err != nil {
		return ""
	}
	r, err := store.New(s.pool).GetAuthSessionRevokedReason(ctx, id)
	if err != nil {
		return ""
	}
	return reasonOut(r)
}

func (s *Sessions) warnRedis(ctx context.Context, err error) {
	s.warnMu.Lock()
	defer s.warnMu.Unlock()
	if now := time.Now(); now.Sub(s.warnLast) >= 30*time.Second {
		s.warnLast = now
		s.log.ErrorContext(ctx, "Redis không với tới: bỏ qua kiểm thu hồi access token (chấp nhận token)", "error", err.Error())
	}
}

func revSidKey(sid string) string { return appredis.Key("auth", "rev", "sid", sid) }
func revUserKey(uid string) string {
	return appredis.Key("auth", "rev", "user", uid)
}

func reasonOut(r *string) string {
	if r == nil {
		return ""
	}
	return strings.ToLower(*r)
}

func newRefreshToken() (string, error) {
	b := make([]byte, refreshBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("auth: sinh refresh token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func parseIP(s string) *netip.Addr {
	a, err := netip.ParseAddr(strings.TrimSpace(s))
	if err != nil {
		return nil
	}
	a = a.Unmap()
	return &a
}

// clip cắt chuỗi tới n ký tự (rune); rỗng → nil (cột NULL).
func clip(s string, n int) *string {
	if s == "" {
		return nil
	}
	if utf8.RuneCountInString(s) > n {
		s = string([]rune(s)[:n])
	}
	return &s
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}
