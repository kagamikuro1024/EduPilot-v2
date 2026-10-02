package auth

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/edupilot/backend-go/internal/platform/clock"
)

const (
	testSecret = "0123456789abcdef0123456789abcdef"
	otherSecet = "ffffffffffffffffffffffffffffffff"
	testSub    = "00000000-0000-7000-8000-000000000001"
	testSub2   = "00000000-0000-7000-8000-000000000002"
	testEmail  = "qc@example.test"
)

func fixedClock() *clock.Fake {
	return clock.NewFake(time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC))
}

// jwtPart giải mã đoạn thứ n (0 = header, 1 = payload) của token thành map.
func jwtPart(t *testing.T, token string, n int) map[string]any {
	t.Helper()
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("token có %d đoạn, muốn 3", len(parts))
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[n])
	if err != nil {
		t.Fatalf("giải base64url đoạn %d: %v", n, err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("giải JSON đoạn %d: %v", n, err)
	}
	return m
}

func keysOf(m map[string]any) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}

func TestJWT_Claims(t *testing.T) {
	clk := fixedClock()
	tok, err := NewIssuer(testSecret, 0, clk).Issue(testSub, RoleTeacher, testEmail)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	hdr := jwtPart(t, tok, 0)
	if got := hdr["alg"]; got != "HS256" {
		t.Errorf("header.alg=%v, muốn HS256", got)
	}
	if got := hdr["typ"]; got != "JWT" {
		t.Errorf("header.typ=%v, muốn JWT", got)
	}
	if len(hdr) != 2 {
		t.Errorf("header có %d khoá (%v), muốn đúng 2", len(hdr), keysOf(hdr))
	}

	p := jwtPart(t, tok, 1)
	want := map[string]any{
		"sub": testSub, "role": "TEACHER", "email": testEmail,
		"iss": "edupilot", "aud": "edupilot-api",
	}
	for k, v := range want {
		if p[k] != v {
			t.Errorf("claim %s=%v, muốn %v", k, p[k], v)
		}
	}
	if len(p) != 9 {
		t.Errorf("payload có %d claim (%v), muốn đúng 9", len(p), keysOf(p))
	}
	for _, k := range []string{"sub", "role", "email", "jti", "iat", "nbf", "exp", "iss", "aud"} {
		if _, ok := p[k]; !ok {
			t.Errorf("thiếu claim %s", k)
		}
	}

	iat, nbf, exp := p["iat"].(float64), p["nbf"].(float64), p["exp"].(float64)
	if iat != float64(clk.Now().Unix()) {
		t.Errorf("iat=%v, muốn %v", iat, clk.Now().Unix())
	}
	if nbf != iat {
		t.Errorf("nbf=%v, muốn = iat %v", nbf, iat)
	}
	if exp-iat != DefaultTTL.Seconds() {
		t.Errorf("exp-iat=%v, muốn %v", exp-iat, DefaultTTL.Seconds())
	}

	jti, _ := p["jti"].(string)
	if len(jti) != 22 {
		t.Errorf("jti dài %d ký tự, muốn 22 (128 bit base64url không đệm)", len(jti))
	}
	if _, err := base64.RawURLEncoding.DecodeString(jti); err != nil {
		t.Errorf("jti không phải base64url: %v", err)
	}

	// ttl tuỳ chọn đi thẳng vào exp-iat
	tok10m, err := NewIssuer(testSecret, 10*time.Minute, clk).Issue(testSub, RoleAdmin, testEmail)
	if err != nil {
		t.Fatalf("Issue(10m): %v", err)
	}
	p10 := jwtPart(t, tok10m, 1)
	if d := p10["exp"].(float64) - p10["iat"].(float64); d != 600 {
		t.Errorf("exp-iat với ttl 10m = %v, muốn 600", d)
	}

	// Verify dựng lại đúng danh tính từ claim
	got, err := NewVerifier(testSecret, clk).Verify(tok)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if got.Sub != testSub || got.Role != RoleTeacher || got.Email != testEmail || got.JTI != jti {
		t.Errorf("Principal=%+v, muốn sub/role/email/jti khớp claim", got)
	}
	if got.ExpiresAt.Unix() != int64(exp) {
		t.Errorf("ExpiresAt=%v, muốn %v", got.ExpiresAt.Unix(), int64(exp))
	}
}

func TestJWT_JTIUnique(t *testing.T) {
	const n = 10000
	iss := NewIssuer(testSecret, 0, fixedClock())
	seen := make(map[string]struct{}, n)
	for i := range n {
		tok, err := iss.Issue(testSub, RoleStudent, testEmail)
		if err != nil {
			t.Fatalf("Issue lần %d: %v", i, err)
		}
		jti, _ := jwtPart(t, tok, 1)["jti"].(string)
		if len(jti) != 22 {
			t.Fatalf("lần %d: jti dài %d, muốn 22", i, len(jti))
		}
		if _, dup := seen[jti]; dup {
			t.Fatalf("lần %d: jti trùng", i)
		}
		seen[jti] = struct{}{}
	}
	if len(seen) != n {
		t.Errorf("có %d jti duy nhất trong %d lần cấp", len(seen), n)
	}
}

// mkToken ký claims bằng method (header alg = method.Alg()).
func mkToken(t *testing.T, method jwt.SigningMethod, key any, c jwt.MapClaims) string {
	t.Helper()
	s, err := jwt.NewWithClaims(method, c).SignedString(key)
	if err != nil {
		t.Fatalf("ký token: %v", err)
	}
	return s
}

// rawJWT ghép ba đoạn thô (dùng cho alg không ký được như RS256 rác).
func rawJWT(header, payload string, sig []byte) string {
	e := base64.RawURLEncoding.EncodeToString
	return e([]byte(header)) + "." + e([]byte(payload)) + "." + e(sig)
}

func baseClaims(now time.Time, ttl time.Duration) jwt.MapClaims {
	return jwt.MapClaims{
		"sub": testSub, "role": "STUDENT", "email": testEmail, "jti": "qcJTI0000000000000000a",
		"iat": now.Unix(), "nbf": now.Unix(), "exp": now.Add(ttl).Unix(),
		"iss": "edupilot", "aud": "edupilot-api",
	}
}

func TestVerify_Table(t *testing.T) {
	clk := fixedClock()
	now := clk.Now()
	v := NewVerifier(testSecret, clk)

	// mutate dựng token từ bộ claim chuẩn sau khi sửa theo f.
	mutate := func(f func(jwt.MapClaims)) string {
		c := baseClaims(now, 15*time.Minute)
		f(c)
		return mkToken(t, jwt.SigningMethodHS256, []byte(testSecret), c)
	}
	valid := mutate(func(jwt.MapClaims) {})

	tests := []struct {
		name  string
		token string
		want  error // nil = chấp nhận
	}{
		{"hợp lệ (đối chứng dương)", valid, nil},
		{"hết hạn 1 giờ", mutate(func(c jwt.MapClaims) { c["exp"] = now.Add(-time.Hour).Unix() }), ErrTokenExpired},
		{"ký bằng secret khác", mkToken(t, jwt.SigningMethodHS256, []byte(otherSecet), baseClaims(now, time.Minute)), ErrTokenInvalid},
		{"chữ ký đổi 1 ký tự", flipLast(valid), ErrTokenInvalid},
		{"chữ ký rỗng", valid[:strings.LastIndex(valid, ".")+1], ErrTokenInvalid},
		{"alg=none", mkToken(t, jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, baseClaims(now, time.Minute)), ErrTokenInvalid},
		{"alg=HS512 ký đúng secret", mkToken(t, jwt.SigningMethodHS512, []byte(testSecret), baseClaims(now, time.Minute)), ErrTokenInvalid},
		{"alg=RS256 chữ ký rác", rawJWT(`{"alg":"RS256","typ":"JWT"}`, claimsJSON(t, baseClaims(now, time.Minute)), []byte("chu-ky-rac")), ErrTokenInvalid},
		{"thiếu exp", mutate(func(c jwt.MapClaims) { delete(c, "exp") }), ErrTokenInvalid},
		{"role=SUPERUSER", mutate(func(c jwt.MapClaims) { c["role"] = "SUPERUSER" }), ErrTokenInvalid},
		{"role chữ thường", mutate(func(c jwt.MapClaims) { c["role"] = "student" }), ErrTokenInvalid},
		{"thiếu role", mutate(func(c jwt.MapClaims) { delete(c, "role") }), ErrTokenInvalid},
		{"iss sai", mutate(func(c jwt.MapClaims) { c["iss"] = "khac" }), ErrTokenInvalid},
		{"thiếu iss", mutate(func(c jwt.MapClaims) { delete(c, "iss") }), ErrTokenInvalid},
		{"aud sai", mutate(func(c jwt.MapClaims) { c["aud"] = "edupilot-web" }), ErrTokenInvalid},
		{"thiếu aud", mutate(func(c jwt.MapClaims) { delete(c, "aud") }), ErrTokenInvalid},
		{"nbf ở tương lai", mutate(func(c jwt.MapClaims) { c["nbf"] = now.Add(10 * time.Minute).Unix() }), ErrTokenInvalid},
		{"sub rỗng", mutate(func(c jwt.MapClaims) { c["sub"] = "" }), ErrTokenInvalid},
		{"chỉ 2 đoạn", valid[:strings.LastIndex(valid, ".")], ErrTokenInvalid},
		{"4 đoạn", valid + "." + valid[strings.LastIndex(valid, ".")+1:], ErrTokenInvalid},
		{"chuỗi rác", "garbage", ErrTokenInvalid},
		{"chuỗi rỗng", "", ErrTokenInvalid},
		{"payload không phải base64url", strings.SplitN(valid, ".", 2)[0] + ".!!!." + valid[strings.LastIndex(valid, ".")+1:], ErrTokenInvalid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := v.Verify(tt.token)
			switch {
			case tt.want == nil && err != nil:
				t.Fatalf("Verify lỗi %v, muốn chấp nhận", err)
			case tt.want == nil:
				if p.Sub != testSub || p.Role != RoleStudent {
					t.Fatalf("Principal=%+v", p)
				}
			case !errors.Is(err, tt.want):
				t.Fatalf("Verify lỗi %v, muốn %v", err, tt.want)
			}
		})
	}
}

func claimsJSON(t *testing.T, c jwt.MapClaims) string {
	t.Helper()
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("marshal claims: %v", err)
	}
	return string(b)
}

// flipLast đổi ký tự cuối của chữ ký để phá chữ ký mà giữ nguyên định dạng.
func flipLast(token string) string {
	last := token[len(token)-1]
	repl := byte('A')
	if last == 'A' {
		repl = 'B'
	}
	return token[:len(token)-1] + string(repl)
}

func TestVerify_Leeway(t *testing.T) {
	clk := fixedClock()
	start := clk.Now()
	tok, err := NewIssuer(testSecret, time.Minute, clk).Issue(testSub, RoleTA, testEmail)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	v := NewVerifier(testSecret, clk)

	// exp = start+60s; leeway 5 s (đồng hồ giả, không time.Sleep)
	tests := []struct {
		name  string
		after time.Duration // thời điểm kiểm so với lúc cấp
		want  error
	}{
		{"còn hạn", 30 * time.Second, nil},
		{"đúng lúc hết hạn", 60 * time.Second, nil},
		{"hết hạn 3 s (trong leeway)", 63 * time.Second, nil},
		{"hết hạn 4 s (trong leeway)", 64 * time.Second, nil},
		{"hết hạn 4,9 s (trong leeway)", 64*time.Second + 900*time.Millisecond, nil},
		{"hết hạn đúng 5 s (hết leeway)", 65 * time.Second, ErrTokenExpired},
		{"hết hạn 6 s (quá leeway)", 66 * time.Second, ErrTokenExpired},
		{"hết hạn 7 s", 67 * time.Second, ErrTokenExpired},
		{"hết hạn 10 s", 70 * time.Second, ErrTokenExpired},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clk.Set(start.Add(tt.after))
			_, err := v.Verify(tok)
			if !errors.Is(err, tt.want) {
				t.Fatalf("tại +%v: lỗi %v, muốn %v", tt.after, err, tt.want)
			}
		})
	}

	// nbf cũng có leeway: nbf sớm 3 s so với "bây giờ" → nhận; 10 s → từ chối.
	clk.Set(start)
	for _, tt := range []struct {
		name string
		skew time.Duration
		want error
	}{
		{"nbf trước 3 s", 3 * time.Second, nil},
		{"nbf trước 10 s", 10 * time.Second, ErrTokenInvalid},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c := baseClaims(start, 15*time.Minute)
			c["nbf"] = start.Add(tt.skew).Unix()
			_, err := v.Verify(mkToken(t, jwt.SigningMethodHS256, []byte(testSecret), c))
			if !errors.Is(err, tt.want) {
				t.Fatalf("lỗi %v, muốn %v", err, tt.want)
			}
		})
	}
}
