// Package attest kiểm Android Key Attestation cho `POST /me/devices` (FEAT-attendance-selfcheck FR-19/20, ARCHITECTURE 11.5):
// chỉ thư viện chuẩn (`crypto/x509`, `encoding/asn1`). Không lưu nonce, chuỗi chứng chỉ hay định danh máy ngoài `user_devices`.
package attest

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"crypto/x509"
	_ "embed" // roots.pem
	"encoding/asn1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/edupilot/backend-go/internal/attendance"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/platform/config"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/store"
)

//go:embed roots.pem
var rootsPEM []byte

const (
	statusURL      = "https://android.googleapis.com/attestation/status"
	statusTimeout  = 3 * time.Second
	statusMaxBytes = 4 << 20 // bản đo 2026-11-15: 181 KB
	challengeTTL   = 300 * time.Second
	challengeLen   = 32
	loadedMarker   = "_loaded" // tập Redis không rỗng được: mục đánh dấu «đã nạp»
)

// OID của KeyDescription và thẻ AuthorizationList.
var oidKeyDescription = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 11129, 2, 1, 17} //nolint:gochecknoglobals // hằng số chuẩn của Android

const (
	tagRootOfTrust = 704
	tagAppID       = 709
)

// Verifier cài attendance.Registrar. Zero value của các trường tuỳ chọn dùng mặc định thật.
type Verifier struct {
	Pool  *pgxpool.Pool
	Redis *appredis.Client
	Cfg   config.Attendance
	Log   *slog.Logger
	// Now: đồng hồ kiểm hạn chứng chỉ; nil = time.Now (chuỗi Google ngắn hạn nên test đặt mốc trong hạn).
	Now func() time.Time
	// Roots: gốc tin cậy; nil = roots.pem nhúng sẵn.
	Roots *x509.CertPool
	// HTTP / StatusURL: nguồn danh sách thu hồi; nil / "" = mặc định (hạn 3 s). Test trỏ vào máy chủ giả.
	HTTP      *http.Client
	StatusURL string
}

// fail ghi MỘT dòng cảnh báo chỉ có tên bước (không nonce, không chứng chỉ, không người dùng) rồi trả ATTESTATION_FAILED.
func (v *Verifier) fail(ctx context.Context, step string) *apierr.Error {
	if v.Log != nil {
		v.Log.WarnContext(ctx, "attest: từ chối chứng thực", "step", step)
	}
	return failed()
}

func (v *Verifier) now() time.Time {
	if v.Now != nil {
		return v.Now()
	}
	return time.Now()
}

func chalKey(uid uuid.UUID) string { return appredis.Key("dev", "chal", uid.String()) }

var (
	revokedKey     = appredis.Key("att", "revoked")      //nolint:gochecknoglobals // tên khoá cố định
	revokedLockKey = appredis.Key("lock", "att.revoked") //nolint:gochecknoglobals // tên khoá cố định
)

func failed() *apierr.Error {
	return apierr.New(http.StatusUnprocessableEntity, apierr.AttestationFailed)
}
func unavailable() *apierr.Error {
	return apierr.New(http.StatusServiceUnavailable, apierr.AttestationUnavailable)
}

// Challenge cấp nonce 32 byte (ghi đè nonce cũ), dùng một lần, sống 300 s.
func (v *Verifier) Challenge(ctx context.Context, userID uuid.UUID) ([]byte, error) {
	n := make([]byte, challengeLen)
	if _, err := rand.Read(n); err != nil {
		return nil, fmt.Errorf("attest: nonce: %w", err)
	}
	if err := v.Redis.Set(ctx, chalKey(userID), n, challengeTTL).Err(); err != nil {
		return nil, fmt.Errorf("attest: lưu nonce: %w", err)
	}
	return n, nil
}

// Register chạy 10 bước ARCHITECTURE 11.5 (bước 1 — số chứng chỉ, kích thước — đã kiểm ở handler).
func (v *Verifier) Register(ctx context.Context, userID uuid.UUID, in attendance.RegisterIn) (attendance.RegisterOut, error) {
	if len(in.AttestationChain) < 2 || len(in.AttestationChain) > 6 {
		return attendance.RegisterOut{}, apierr.Validation(apierr.FieldError{Field: "attestation_chain", Code: "INVALID", Message: "Cần 2 đến 6 chứng chỉ."})
	}
	// 2. nonce dùng một lần
	nonce, err := v.Redis.GetDel(ctx, chalKey(userID)).Bytes()
	if errors.Is(err, redis.Nil) {
		return attendance.RegisterOut{}, apierr.New(http.StatusUnprocessableEntity, apierr.DeviceChallengeInvalid)
	}
	if err != nil {
		return attendance.RegisterOut{}, fmt.Errorf("attest: đọc nonce: %w", err)
	}
	// 3–4. chuỗi tới gốc Google
	chain := make([]*x509.Certificate, len(in.AttestationChain))
	for i, der := range in.AttestationChain {
		if chain[i], err = x509.ParseCertificate(der); err != nil {
			return attendance.RegisterOut{}, v.fail(ctx, "parse")
		}
	}
	roots := v.Roots
	if roots == nil {
		roots = x509.NewCertPool()
		roots.AppendCertsFromPEM(rootsPEM)
	}
	inter := x509.NewCertPool()
	for _, c := range chain[1:] {
		inter.AddCert(c)
	}
	leaf := chain[0]
	if _, err := leaf.Verify(x509.VerifyOptions{Roots: roots, Intermediates: inter, CurrentTime: v.now(), KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageAny}}); err != nil {
		return attendance.RegisterOut{}, v.fail(ctx, "chain")
	}
	// 5. thu hồi
	if err := v.checkRevoked(ctx, chain); err != nil {
		return attendance.RegisterOut{}, err
	}
	// 6–8. KeyDescription
	kd, err := parseKeyDescription(leaf)
	if err != nil {
		return attendance.RegisterOut{}, v.fail(ctx, "revoked?")
	}
	if kd.SecurityLevel != 1 && kd.SecurityLevel != 2 { // TEE, StrongBox (0 = phần mềm)
		return attendance.RegisterOut{}, v.fail(ctx, "kd")
	}
	if subtle.ConstantTimeCompare(kd.Challenge, nonce) != 1 {
		return attendance.RegisterOut{}, v.fail(ctx, "level")
	}
	if v.Cfg.AttestationPackage == "" || v.Cfg.AttestationCertSHA256 == "" || !kd.appMatches(v.Cfg.AttestationPackage, v.Cfg.AttestationCertSHA256) {
		return attendance.RegisterOut{}, v.fail(ctx, "challenge")
	}
	// 9. ràng khoá (P-108-4): khoá trong thân = khoá của chứng chỉ lá, P-256
	if !bytes.Equal(in.PublicKey, leaf.RawSubjectPublicKeyInfo) {
		return attendance.RegisterOut{}, v.fail(ctx, "app")
	}
	if _, ok := attendance.ParseP256(in.PublicKey); !ok {
		return attendance.RegisterOut{}, v.fail(ctx, "bind")
	}
	// 10. upsert
	return v.upsert(ctx, in.PublicKey, kd)
}

func (v *Verifier) upsert(ctx context.Context, pub []byte, kd keyDescription) (attendance.RegisterOut, error) {
	q := store.New(v.Pool)
	sum := attendance.KeySum(pub)
	_, err := q.AttDeviceByHashLock(ctx, sum) // ngoài giao dịch: chỉ để biết «đã có» (200) hay «mới» (201)
	existed := err == nil
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return attendance.RegisterOut{}, fmt.Errorf("attest: đọc thiết bị: %w", err)
	}
	level := "TEE"
	if kd.SecurityLevel == 2 {
		level = "STRONGBOX"
	}
	if _, err := q.AttDeviceUpsertAttested(ctx, store.AttDeviceUpsertAttestedParams{KeySha256: sum, PublicKey: pub, Attested: true, BootVerified: &kd.BootVerified, SecurityLevel: &level}); err != nil {
		return attendance.RegisterOut{}, fmt.Errorf("attest: ghi thiết bị: %w", err)
	}
	return attendance.RegisterOut{KeyID: attendance.KeyID(pub), Created: !existed}, nil
}

// serial chuẩn hoá như khoá của danh sách thu hồi Google: hex thường, không số 0 đầu.
func serial(c *x509.Certificate) string { return strings.ToLower(c.SerialNumber.Text(16)) }

func (v *Verifier) checkRevoked(ctx context.Context, chain []*x509.Certificate) error {
	if err := v.ensureRevocationList(ctx); err != nil {
		return err
	}
	ids := make([]any, len(chain))
	for i, c := range chain {
		ids[i] = serial(c)
	}
	hits, err := v.Redis.SMIsMember(ctx, revokedKey, ids...).Result()
	if err != nil {
		return unavailable()
	}
	for _, h := range hits {
		if h {
			return v.fail(ctx, "revoked")
		}
	}
	return nil
}

// ensureRevocationList nạp danh sách thu hồi vào Redis khi chưa có (TTL ATTESTATION_REVOCATION_TTL_H). Một bản gateway nạp (khoá `ep:lock:att.revoked`),
// bản khác đợi tối đa hạn tải. Không tải được và chưa có bản đệm → 503 ATTESTATION_UNAVAILABLE.
func (v *Verifier) ensureRevocationList(ctx context.Context) error {
	has := func() bool { n, err := v.Redis.Exists(ctx, revokedKey).Result(); return err == nil && n > 0 }
	if has() {
		return nil
	}
	ok, err := v.Redis.SetNX(ctx, revokedLockKey, 1, statusTimeout+2*time.Second).Result()
	if err != nil {
		return unavailable()
	}
	if !ok { // bản khác đang nạp
		deadline := time.Now().Add(statusTimeout + time.Second)
		for time.Now().Before(deadline) {
			time.Sleep(100 * time.Millisecond)
			if has() {
				return nil
			}
		}
		return unavailable()
	}
	defer func() { _ = v.Redis.Del(context.WithoutCancel(ctx), revokedLockKey).Err() }()
	serials, err := v.fetchRevoked(ctx)
	if err != nil {
		if v.Log != nil {
			v.Log.WarnContext(ctx, "attest: không tải được danh sách thu hồi", "error", err.Error())
		}
		return unavailable()
	}
	members := append([]any{loadedMarker}, serials...)
	pipe := v.Redis.Pipeline()
	pipe.Del(ctx, revokedKey)
	pipe.SAdd(ctx, revokedKey, members...)
	pipe.Expire(ctx, revokedKey, v.Cfg.AttestationRevocationTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return unavailable()
	}
	return nil
}

func (v *Verifier) fetchRevoked(ctx context.Context) ([]any, error) {
	url := v.StatusURL
	if url == "" {
		url = statusURL
	}
	hc := v.HTTP
	if hc == nil {
		hc = &http.Client{Timeout: statusTimeout}
	}
	rctx, cancel := context.WithTimeout(ctx, statusTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(rctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	var body struct {
		Entries map[string]json.RawMessage `json:"entries"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, statusMaxBytes)).Decode(&body); err != nil {
		return nil, err
	}
	out := make([]any, 0, len(body.Entries))
	for k := range body.Entries {
		out = append(out, strings.ToLower(k))
	}
	return out, nil
}

// ---- KeyDescription ----------------------------------------------------------------------------------------------------------------

type keyDescription struct {
	SecurityLevel int
	Challenge     []byte
	BootVerified  bool
	packages      []string
	digests       [][]byte
}

func (k keyDescription) appMatches(pkg, certHex string) bool {
	want, err := hex.DecodeString(certHex)
	if err != nil || len(want) == 0 {
		return false
	}
	okPkg := false
	for _, p := range k.packages {
		if p == pkg {
			okPkg = true
		}
	}
	okSig := false
	for _, d := range k.digests {
		if subtle.ConstantTimeCompare(d, want) == 1 {
			okSig = true
		}
	}
	return okPkg && okSig
}

type keyDescriptionASN struct {
	AttestationVersion       int
	AttestationSecurityLevel asn1.Enumerated
	KeyMintVersion           int
	KeyMintSecurityLevel     asn1.Enumerated
	AttestationChallenge     []byte
	UniqueID                 []byte
	SoftwareEnforced         asn1.RawValue
	HardwareEnforced         asn1.RawValue
}

type rootOfTrustASN struct {
	VerifiedBootKey   []byte
	DeviceLocked      bool
	VerifiedBootState asn1.Enumerated
}

type appIDASN struct {
	Packages []struct {
		Name    []byte
		Version int
	} `asn1:"set"`
	Digests [][]byte `asn1:"set"`
}

func parseKeyDescription(leaf *x509.Certificate) (keyDescription, error) {
	var raw []byte
	for _, e := range leaf.Extensions {
		if e.Id.Equal(oidKeyDescription) {
			raw = e.Value
		}
	}
	if raw == nil {
		return keyDescription{}, errors.New("thiếu KeyDescription")
	}
	var kd keyDescriptionASN
	if rest, err := asn1.Unmarshal(raw, &kd); err != nil || len(rest) != 0 {
		return keyDescription{}, errors.New("KeyDescription hỏng")
	}
	out := keyDescription{SecurityLevel: int(kd.AttestationSecurityLevel), Challenge: kd.AttestationChallenge}
	soft, err := authList(kd.SoftwareEnforced.Bytes)
	if err != nil {
		return out, err
	}
	hard, err := authList(kd.HardwareEnforced.Bytes)
	if err != nil {
		return out, err
	}
	// attestationApplicationId (709) nằm ở softwareEnforced (một số máy đặt ở hardware: nhận cả hai), rootOfTrust (704) ở hardwareEnforced.
	appRaw := soft[tagAppID]
	if appRaw == nil {
		appRaw = hard[tagAppID]
	}
	if appRaw != nil {
		var oct []byte
		if _, err := asn1.Unmarshal(appRaw, &oct); err != nil {
			return out, errors.New("attestationApplicationId hỏng")
		}
		var app appIDASN
		if _, err := asn1.Unmarshal(oct, &app); err != nil {
			return out, errors.New("attestationApplicationId hỏng")
		}
		for _, p := range app.Packages {
			out.packages = append(out.packages, string(p.Name))
		}
		out.digests = app.Digests
	}
	if rotRaw := hard[tagRootOfTrust]; rotRaw != nil {
		var rot rootOfTrustASN
		if _, err := asn1.Unmarshal(rotRaw, &rot); err != nil {
			return out, errors.New("rootOfTrust hỏng")
		}
		out.BootVerified = rot.VerifiedBootState == 0 && rot.DeviceLocked
	}
	return out, nil
}

// authList đọc AuthorizationList (SEQUENCE của các trường gắn thẻ EXPLICIT ngữ cảnh) thành thẻ → DER bên trong, chỉ giữ thẻ cần dùng.
func authList(content []byte) (map[int][]byte, error) {
	out := map[int][]byte{}
	for len(content) > 0 {
		var rv asn1.RawValue
		rest, err := asn1.Unmarshal(content, &rv)
		if err != nil {
			return nil, errors.New("AuthorizationList hỏng")
		}
		content = rest
		if rv.Class == asn1.ClassContextSpecific && (rv.Tag == tagRootOfTrust || rv.Tag == tagAppID) {
			out[rv.Tag] = rv.Bytes
		}
	}
	return out, nil
}
