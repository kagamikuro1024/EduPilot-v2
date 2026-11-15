package attendance

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"math/big"
	"net/http"

	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
)

// DeviceKind khớp enum `device_kind`.
const (
	KindWebKey     = "WEB_KEY"
	KindAndroidKey = "ANDROID_KEY"
)

// Registrar kiểm Key Attestation và upsert `user_devices` (L5, package attest). Service.Attest nil ⇒ đăng ký thiết bị trả ATTESTATION_UNAVAILABLE.
type Registrar interface {
	Register(ctx context.Context, userID uuid.UUID, in RegisterIn) (RegisterOut, error)
	Challenge(ctx context.Context, userID uuid.UUID) ([]byte, error)
}

// RegisterIn là thân `POST /me/devices`.
type RegisterIn struct {
	Kind             string
	PublicKey        []byte   // SPKI DER
	AttestationChain [][]byte // chứng chỉ DER, lá trước
}

// RegisterOut: `key_id` = base64url của key_sha256.
type RegisterOut struct {
	KeyID string
}

// KeyID = base64url(sha256(public_key)) không đệm.
func KeyID(publicKey []byte) string {
	h := sha256.Sum256(publicKey)
	return base64.RawURLEncoding.EncodeToString(h[:])
}

func keySum(publicKey []byte) []byte {
	h := sha256.Sum256(publicKey)
	return h[:]
}

// ParseP256 đọc SPKI DER (≤ 512 byte) và đòi khoá ECDSA P-256.
func ParseP256(spki []byte) (*ecdsa.PublicKey, bool) {
	if len(spki) == 0 || len(spki) > 512 {
		return nil, false
	}
	k, err := x509.ParsePKIXPublicKey(spki)
	if err != nil {
		return nil, false
	}
	pk, ok := k.(*ecdsa.PublicKey)
	if !ok || pk.Curve != elliptic.P256() {
		return nil, false
	}
	return pk, true
}

// VerifySignature kiểm ECDSA P-256 / SHA-256: chữ ký 64 byte `r‖s` (WebCrypto) hoặc DER (Keystore).
func VerifySignature(pk *ecdsa.PublicKey, msg, sig []byte) bool {
	h := sha256.Sum256(msg)
	if len(sig) == 64 {
		r := new(big.Int).SetBytes(sig[:32])
		s := new(big.Int).SetBytes(sig[32:])
		return ecdsa.Verify(pk, h[:], r, s)
	}
	return ecdsa.VerifyASN1(pk, h[:], sig)
}

// DeviceChallenge cấp nonce 32 byte cho đăng ký thiết bị app (FR-19): Redis `ep:dev:chal:{uid}` TTL 300 s; hạn mức 5 / phút.
func (s *Service) DeviceChallenge(ctx context.Context, userID uuid.UUID) ([]byte, error) {
	if err := s.rate(ctx, "devchal", userID, s.Cfg.DeviceChallengeRate); err != nil {
		return nil, err
	}
	if s.Attest == nil {
		return nil, apierr.New(http.StatusServiceUnavailable, apierr.AttestationUnavailable)
	}
	return s.Attest.Challenge(ctx, userID)
}

// DeviceRegister đăng ký thiết bị (FR-20): chuỗi chứng thực + nonce + gói / vân tay APK → `user_devices`.
func (s *Service) DeviceRegister(ctx context.Context, userID uuid.UUID, in RegisterIn) (RegisterOut, error) {
	if s.Attest == nil {
		return RegisterOut{}, apierr.New(http.StatusServiceUnavailable, apierr.AttestationUnavailable)
	}
	return s.Attest.Register(ctx, userID, in)
}
