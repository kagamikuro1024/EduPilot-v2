package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/store"
)

// TokenKind là loại token một lần (enum auth_token_kind, SRS FEAT-account-security 5.1).
type TokenKind string

// Ba loại token một lần.
const (
	TokenVerifyEmail   TokenKind = "VERIFY_EMAIL"
	TokenResetPassword TokenKind = "RESET_PASSWORD"
	TokenInvite        TokenKind = "INVITE"
)

// Valid cho biết k có phải một trong ba loại không.
func (k TokenKind) Valid() bool {
	return k == TokenVerifyEmail || k == TokenResetPassword || k == TokenInvite
}

// tokenBytes: 32 byte ngẫu nhiên → 43 ký tự base64url không đệm (SRS 5.3, US-P2-01 AC3).
const tokenBytes = 32

// Tokens phát token một lần. Bản rõ trả về ĐÚNG MỘT LẦN cho người gọi; DB chỉ giữ sha256 hex (luật cấm lưu token rõ).
// Đây là gói DUY NHẤT được chạm bảng auth_tokens (TestOnlyAuthPackageTouchesTokenTables).
type Tokens struct{ Clock clock.Clock }

// HashToken là sha256 hex (64 ký tự) của bản rõ — thứ duy nhất được lưu và dùng để tra cứu.
func HashToken(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}

// Issue phát token mới cho (user, kind) trong transaction tx của người gọi: thu hồi mọi token cùng loại chưa dùng,
// ghi băm, trả bản rõ. Chạy trong tx để rollback nghiệp vụ cũng huỷ token (và thu hồi) — không để lại token mồ côi.
func (t Tokens) Issue(ctx context.Context, tx pgx.Tx, user uuid.UUID, kind TokenKind, ttl time.Duration, createdBy *uuid.UUID) (string, error) {
	if !kind.Valid() {
		return "", fmt.Errorf("auth: loại token %q không hợp lệ", kind)
	}
	if ttl <= 0 {
		return "", errors.New("auth: hạn token phải > 0")
	}
	buf := make([]byte, tokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("auth: sinh token: %w", err)
	}
	plain := base64.RawURLEncoding.EncodeToString(buf)

	q := store.New(tx)
	if _, err := q.RevokeUnusedAuthTokens(ctx, store.RevokeUnusedAuthTokensParams{UserID: user, Kind: store.AuthTokenKind(kind)}); err != nil {
		return "", fmt.Errorf("auth: thu hồi token cũ: %w", err)
	}
	if _, err := q.InsertAuthToken(ctx, store.InsertAuthTokenParams{
		UserID: user, Kind: store.AuthTokenKind(kind), TokenHash: HashToken(plain),
		ExpiresAt: t.Clock.Now().Add(ttl), CreatedBy: createdBy,
	}); err != nil {
		return "", fmt.Errorf("auth: ghi token: %w", err)
	}
	return plain, nil
}
