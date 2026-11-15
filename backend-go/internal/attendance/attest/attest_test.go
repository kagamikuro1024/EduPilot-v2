package attest_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/attendance"
	"github.com/edupilot/backend-go/internal/attendance/attest"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/platform/config"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/testutil"
)

// fixture: một chuỗi thật của github.com/android/keyattestation (testdata/README.md) + thông số của file .json đi kèm.
type fixture struct {
	chain          [][]byte
	leaf           *x509.Certificate
	nonce          []byte
	pkg            string
	digest         string // hex
	valid, expired time.Time
}

func load(t *testing.T, name string) fixture {
	t.Helper()
	raw, err := os.ReadFile("testdata/" + name + ".pem")
	require.NoError(t, err)
	var f fixture
	for {
		var b *pem.Block
		b, raw = pem.Decode(raw)
		if b == nil {
			break
		}
		f.chain = append(f.chain, b.Bytes)
	}
	f.leaf, err = x509.ParseCertificate(f.chain[0])
	require.NoError(t, err)
	js, err := os.ReadFile("testdata/" + name + ".json")
	require.NoError(t, err)
	var kd struct {
		Challenge string `json:"attestationChallenge"`
		Soft      struct {
			App struct {
				Packages []struct {
					Name string `json:"name"`
				} `json:"packages"`
				Signatures []string `json:"signatures"`
			} `json:"attestationApplicationId"`
		} `json:"softwareEnforced"`
	}
	require.NoError(t, json.Unmarshal(js, &kd))
	f.nonce, err = base64.StdEncoding.DecodeString(kd.Challenge)
	require.NoError(t, err)
	f.pkg = kd.Soft.App.Packages[0].Name
	d, err := base64.StdEncoding.DecodeString(kd.Soft.App.Signatures[0])
	require.NoError(t, err)
	f.digest = hex.EncodeToString(d)
	for _, der := range f.chain {
		c, err := x509.ParseCertificate(der)
		require.NoError(t, err)
		if c.NotBefore.After(f.valid) {
			f.valid = c.NotBefore.Add(time.Minute) // chuỗi Google ngắn hạn: một mốc nằm trong hạn của MỌI chứng chỉ
		}
		if f.expired.IsZero() || c.NotAfter.Before(f.expired) {
			f.expired = c.NotAfter.Add(24 * time.Hour)
		}
	}
	return f
}

type env struct {
	t      *testing.T
	v      *attest.Verifier
	rdb    *appredis.Client
	pool   *pgxpool.Pool
	f      fixture
	status *httptest.Server
	hits   *int
	body   *string
}

func newEnv(t *testing.T, name string) *env {
	t.Helper()
	testutil.RequireContainers(t)
	f := load(t, name)
	rdb, err := appredis.New(t.Context(), testutil.RedisURL(t))
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })
	pool, err := pgxpool.New(t.Context(), testutil.MigratedPostgresURL(t))
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	require.NoError(t, rdb.Del(t.Context(), appredis.Key("att", "revoked"), appredis.Key("lock", "att.revoked")).Err()) // Redis dùng chung giữa các test / lần chạy
	hits, body := new(int), new(string)
	*body = `{"entries":{"deadbeef0123":{"status":"REVOKED","reason":"KEY_COMPROMISE"}}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*hits++
		if *body == "" {
			http.Error(w, "x", http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(*body))
	}))
	t.Cleanup(srv.Close)
	return &env{t: t, f: f, rdb: rdb, pool: pool, status: srv, hits: hits, body: body, v: &attest.Verifier{
		Pool: pool, Redis: rdb, StatusURL: srv.URL,
		Cfg: config.Attendance{AttestationPackage: f.pkg, AttestationCertSHA256: f.digest, AttestationRevocationTTL: time.Hour},
		Now: func() time.Time { return f.valid },
	}}
}

func (e *env) register(uid uuid.UUID, nonce []byte, mod func(*attendance.RegisterIn)) (attendance.RegisterOut, error) {
	e.t.Helper()
	if nonce != nil {
		require.NoError(e.t, e.rdb.Set(e.t.Context(), appredis.Key("dev", "chal", uid.String()), nonce, time.Minute).Err())
	}
	in := attendance.RegisterIn{Kind: attendance.KindAndroidKey, PublicKey: e.f.leaf.RawSubjectPublicKeyInfo, AttestationChain: e.f.chain}
	if mod != nil {
		mod(&in)
	}
	return e.v.Register(e.t.Context(), uid, in)
}

func code(err error) string {
	if ae, ok := err.(*apierr.Error); ok {
		return ae.Code
	}
	return ""
}

func TestAttestRealChains(t *testing.T) {
	for name, want := range map[string]struct {
		level string
		boot  bool
	}{
		"caiman_sdk36_SB_EC_RKP":  {"STRONGBOX", true},
		"caiman_sdk36_TEE_EC_RKP": {"TEE", true},
		"akita_sdk34_TEE_EC_NONE": {"TEE", true},
	} {
		t.Run(name, func(t *testing.T) {
			e := newEnv(t, name)
			uid := uuid.New()
			out, err := e.register(uid, e.f.nonce, nil)
			require.NoError(t, err)
			require.True(t, out.Created)
			require.Equal(t, attendance.KeyID(e.f.leaf.RawSubjectPublicKeyInfo), out.KeyID)
			var level string
			var boot, attested bool
			require.NoError(t, e.pool.QueryRow(t.Context(), `select security_level, boot_verified, attested from user_devices where key_sha256 = $1`, attendance.KeySum(e.f.leaf.RawSubjectPublicKeyInfo)).Scan(&level, &boot, &attested))
			require.Equal(t, want.level, level)
			require.True(t, attested)
			_ = want.boot // trạng thái khởi động của từng máy do chính chứng chỉ quyết định; chỉ kiểm không lỗi

			// Đăng ký lại cùng khoá: 200 (Created=false) cùng key_id.
			again, err := e.register(uid, e.f.nonce, nil)
			require.NoError(t, err)
			require.False(t, again.Created)
			require.Equal(t, out.KeyID, again.KeyID)
		})
	}
}

func TestAttestRejects(t *testing.T) {
	e := newEnv(t, "caiman_sdk36_SB_EC_RKP")
	uid := uuid.New()
	ok := func(mod func(*attendance.RegisterIn), nonce []byte, cfg func(*config.Attendance), now func() time.Time) error {
		saved, savedNow := e.v.Cfg, e.v.Now
		defer func() { e.v.Cfg, e.v.Now = saved, savedNow }()
		if cfg != nil {
			cfg(&e.v.Cfg)
		}
		if now != nil {
			e.v.Now = now
		}
		_, err := e.register(uid, nonce, mod)
		return err
	}
	// Không có nonce / nonce đã dùng → DEVICE_CHALLENGE_INVALID (GETDEL: dùng một lần).
	require.Equal(t, apierr.DeviceChallengeInvalid, code(ok(nil, nil, nil, nil)))
	require.NoError(t, ok(nil, e.f.nonce, nil, nil))
	require.Equal(t, apierr.DeviceChallengeInvalid, code(ok(nil, nil, nil, nil)))

	f := apierr.AttestationFailed
	require.Equal(t, f, code(ok(nil, []byte("nonce-khac-0123456789abcdef012345"), nil, nil)), "challenge khác")
	require.Equal(t, f, code(ok(nil, e.f.nonce, func(c *config.Attendance) { c.AttestationPackage = "com.ke.la" }, nil)), "gói lạ")
	require.Equal(t, f, code(ok(nil, e.f.nonce, func(c *config.Attendance) { c.AttestationCertSHA256 = strings.Repeat("ab", 32) }, nil)), "vân tay APK lạ")
	require.Equal(t, f, code(ok(nil, e.f.nonce, func(c *config.Attendance) { c.AttestationPackage, c.AttestationCertSHA256 = "", "" }, nil)), "chưa cấu hình gói")
	require.Equal(t, f, code(ok(nil, e.f.nonce, nil, func() time.Time { return e.f.expired })), "chứng chỉ hết hạn")

	// P-108-4: chuỗi thật + khoá phần mềm khác trong thân → từ chối.
	other, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	spki, err := x509.MarshalPKIXPublicKey(&other.PublicKey)
	require.NoError(t, err)
	require.Equal(t, f, code(ok(func(in *attendance.RegisterIn) { in.PublicKey = spki }, e.f.nonce, nil, nil)), "khoá trong thân ≠ khoá chứng chỉ lá")

	// Chứng chỉ tự ký không tới gốc Google.
	require.Equal(t, f, code(ok(func(in *attendance.RegisterIn) { in.AttestationChain = [][]byte{selfSigned(t), selfSigned(t)} }, e.f.nonce, nil, nil)), "tự ký")
	require.Equal(t, f, code(ok(func(in *attendance.RegisterIn) { in.AttestationChain = [][]byte{[]byte("rac"), []byte("rac")} }, e.f.nonce, nil, nil)), "DER hỏng")
}

func TestAttestRevocation(t *testing.T) {
	e := newEnv(t, "akita_sdk34_TEE_EC_NONE")
	uid := uuid.New()
	// Serial của lá nằm trong danh sách thu hồi → ATTESTATION_FAILED.
	*e.body = `{"entries":{"` + strings.ToLower(e.f.leaf.SerialNumber.Text(16)) + `":{"status":"REVOKED"}}}`
	_, err := e.register(uid, e.f.nonce, nil)
	require.Equal(t, apierr.AttestationFailed, code(err))
	require.Equal(t, 1, *e.hits)

	// Đã có bản đệm: không gọi lại máy chủ Google (kể cả khi nó chết).
	*e.body = ""
	_, err = e.register(uid, e.f.nonce, nil)
	require.Equal(t, apierr.AttestationFailed, code(err))
	require.Equal(t, 1, *e.hits)

	// Không có đệm + không tải được → 503 ATTESTATION_UNAVAILABLE.
	require.NoError(t, e.rdb.Del(t.Context(), appredis.Key("att", "revoked")).Err())
	_, err = e.register(uid, e.f.nonce, nil)
	require.Equal(t, apierr.AttestationUnavailable, code(err))
	require.Equal(t, 2, *e.hits)

	// Phục hồi: danh sách sạch → đăng ký được.
	*e.body = `{"entries":{}}`
	_, err = e.register(uid, e.f.nonce, nil)
	require.NoError(t, err)
}

func TestDeviceChallengeOnce(t *testing.T) {
	e := newEnv(t, "caiman_sdk36_TEE_EC_RKP")
	uid := uuid.New()
	n1, err := e.v.Challenge(t.Context(), uid)
	require.NoError(t, err)
	require.Len(t, n1, 32)
	n2, err := e.v.Challenge(t.Context(), uid)
	require.NoError(t, err)
	require.NotEqual(t, n1, n2, "nonce mới ghi đè nonce cũ")
	ttl, err := e.rdb.TTL(t.Context(), appredis.Key("dev", "chal", uid.String())).Result()
	require.NoError(t, err)
	require.InDelta(t, 300, ttl.Seconds(), 5)
	// Nonce cũ không còn dùng được; nonce mới dùng một lần (chuỗi này không khớp nonce thật nên ATTESTATION_FAILED, nhưng nonce đã bị tiêu).
	_, err = e.register(uid, nil, nil)
	require.Equal(t, apierr.AttestationFailed, code(err))
	_, err = e.register(uid, nil, nil)
	require.Equal(t, apierr.DeviceChallengeInvalid, code(err))
}

func selfSigned(t *testing.T) []byte {
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	tpl := &x509.Certificate{SerialNumber: big.NewInt(time.Now().UnixNano()), Subject: pkix.Name{CommonName: "ke gia"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour)}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &k.PublicKey, k)
	require.NoError(t, err)
	return der
}
