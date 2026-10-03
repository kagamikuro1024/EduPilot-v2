package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/edupilot/backend-go/internal/platform/clock"
)

// Hằng của token (FR-37): chỉ HS256, iss/aud cố định, leeway 5 s cho lệch đồng hồ (FR-38).
const (
	tokenIssuer   = "edupilot"
	tokenAudience = "edupilot-api"
	leeway        = 5 * time.Second
	jtiBytes      = 16 // 128 bit → 22 ký tự base64url không đệm

	// DefaultTTL là hạn mặc định của token (ACCESS_TOKEN_TTL mặc định 15 phút).
	DefaultTTL = 15 * time.Minute
)

// Lỗi phân loại của Verify — Middleware ánh xạ sang TOKEN_EXPIRED / TOKEN_INVALID (SRS 6.1).
var (
	ErrTokenExpired = errors.New("auth: token hết hạn")
	ErrTokenInvalid = errors.New("auth: token không hợp lệ")
)

// claims là phần claim riêng của EduPilot cộng với các claim chuẩn.
type claims struct {
	Role  string `json:"role"`
	Email string `json:"email"`
	// SID là id phiên (auth_sessions.id); token dev của `gateway token` không có.
	SID string `json:"sid,omitempty"`
	jwt.RegisteredClaims
}

// Issuer ký token HS256. An toàn khi dùng song song (không trạng thái thay đổi).
type Issuer struct {
	secret []byte
	ttl    time.Duration
	clk    clock.Clock
}

// NewIssuer dựng Issuer; ttl = 0 → DefaultTTL, clk = nil → đồng hồ hệ thống.
func NewIssuer(secret string, ttl time.Duration, clk clock.Clock) *Issuer {
	if ttl == 0 {
		ttl = DefaultTTL
	}
	if clk == nil {
		clk = clock.Real{}
	}
	return &Issuer{secret: []byte(secret), ttl: ttl, clk: clk}
}

// Issue ký một token dev cho (sub, role, email): đúng 9 claim của 04-AC1, KHÔNG có `sid` (không gắn phiên thật).
func (i *Issuer) Issue(sub string, role Role, email string) (string, error) {
	return i.issue(sub, role, email, "")
}

// IssueSession ký access token của một phiên đăng nhập: 9 claim cộng `sid` (id `auth_sessions`), để thu hồi được.
func (i *Issuer) IssueSession(sub string, role Role, email, sid string) (string, error) {
	return i.issue(sub, role, email, sid)
}

func (i *Issuer) issue(sub string, role Role, email, sid string) (string, error) {
	jti, err := newJTI()
	if err != nil {
		return "", err
	}
	now := i.clk.Now()
	m := jwt.MapClaims{
		"sub":   sub,
		"role":  string(role),
		"email": email,
		"jti":   jti,
		"iat":   now.Unix(),
		"nbf":   now.Unix(),
		"exp":   now.Add(i.ttl).Unix(),
		"iss":   tokenIssuer,
		"aud":   tokenAudience,
	}
	if sid != "" {
		m["sid"] = sid
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, m).SignedString(i.secret)
}

// newJTI sinh 128 bit ngẫu nhiên mã base64url không đệm (22 ký tự).
func newJTI() (string, error) {
	b := make([]byte, jtiBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// Verifier kiểm token HS256. An toàn khi dùng song song.
type Verifier struct {
	secret []byte
	clk    clock.Clock
}

// NewVerifier dựng Verifier; clk = nil → đồng hồ hệ thống.
func NewVerifier(secret string, clk clock.Clock) *Verifier {
	if clk == nil {
		clk = clock.Real{}
	}
	return &Verifier{secret: []byte(secret), clk: clk}
}

// Verify kiểm chữ ký, thuật toán (CHỈ HS256), iss, aud, nbf, exp (leeway 5 s) rồi dựng Principal từ claim.
// Trả ErrTokenExpired khi chỉ hết hạn, ErrTokenInvalid cho mọi trường hợp còn lại.
func (v *Verifier) Verify(raw string) (Principal, error) {
	var c claims
	_, err := jwt.ParseWithClaims(raw, &c, func(*jwt.Token) (any, error) { return v.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(tokenIssuer),
		jwt.WithAudience(tokenAudience),
		jwt.WithExpirationRequired(),
		jwt.WithLeeway(leeway),
		jwt.WithTimeFunc(v.clk.Now),
	)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return Principal{}, ErrTokenExpired
		}
		return Principal{}, ErrTokenInvalid
	}
	role := Role(c.Role)
	if !role.Valid() || c.Subject == "" {
		return Principal{}, ErrTokenInvalid
	}
	p := Principal{Sub: c.Subject, Role: role, Email: c.Email, JTI: c.ID, SessionID: c.SID}
	if c.ExpiresAt != nil {
		p.ExpiresAt = c.ExpiresAt.Time
	}
	if c.IssuedAt != nil {
		p.IssuedAt = c.IssuedAt.Time
	}
	return p, nil
}
