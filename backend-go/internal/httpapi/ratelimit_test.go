package httpapi

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/config"
	appdb "github.com/edupilot/backend-go/internal/platform/db"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/testutil"
	"github.com/google/uuid"
)

// rlSecret là khoá ký token của test rate limit (đủ 32 byte).
const rlSecret = "0123456789abcdef0123456789abcdef"

// rlPath là một route NẰM TRONG nhóm nghiệp vụ (sau rate limit, trước auth): không token → 401,
// token hợp lệ → 404 (không có job), bị chặn → 429. Đúng đường dẫn QC dùng cho 03-AC6.
func rlPath() string { return "/api/v1/jobs/" + uuid.NewString() }

// rlDeps dựng Deps thật (Postgres đã migrate + Redis) với đồng hồ giả: cửa sổ phút cố định nên
// test không phụ thuộc biên phút thật.
func rlDeps(t *testing.T, mutate func(*config.Config)) (Deps, *coreBuf) {
	t.Helper()
	d, buf := coreDeps(t, func(c *config.Config) {
		c.JWTSecretKey = rlSecret
		c.RateLimitIPPerMin = 5
		c.RateLimitUserPerMin = 600
		// Như lúc chạy thật: gateway đứng sau Caddy nên tin X-Forwarded-For từ loopback/mạng nội bộ.
		c.TrustedProxyCIDRs = []netip.Prefix{netip.MustParsePrefix("127.0.0.0/8"), netip.MustParsePrefix("::1/128")}
		if mutate != nil {
			mutate(c)
		}
	})
	d.Cfg.DatabaseURL = testutil.MigratedPostgresURL(t)
	d.Cfg.RedisURL = testutil.RedisURL(t)
	pool, err := appdb.NewPool(t.Context(), d.Cfg, d.Log)
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	t.Cleanup(pool.Close)
	rdb, err := appredis.New(t.Context(), d.Cfg.RedisURL)
	if err != nil {
		t.Fatalf("redis.New: %v", err)
	}
	t.Cleanup(func() { _ = rdb.Close() })
	d.DB, d.Redis = pool, rdb
	d.Clock = clock.NewFake(time.Date(2026, 3, 1, 10, 30, 0, 0, time.UTC))
	return withDefaults(d), buf
}

func rlServer(t *testing.T, d Deps) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(NewRouter(d))
	t.Cleanup(srv.Close)
	return srv
}

// rlIP sinh một địa chỉ khách hợp lệ, khác nhau ở mỗi lần gọi (bộ đếm IP không đụng nhau giữa các lần chạy).
func rlIP() string {
	n := time.Now().UnixNano()
	return fmt.Sprintf("2001:db8:%04x:%04x::%04x", (n>>32)&0xffff, (n>>16)&0xffff, n&0xffff)
}

func rlToken(t *testing.T, d Deps, sub string) string {
	t.Helper()
	tok, err := auth.NewIssuer(rlSecret, time.Hour, d.Clock).Issue(sub, auth.RoleStudent, "")
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	return tok
}

// rlGet gọi một route qua router đầy đủ với X-Forwarded-For (httptest nối từ 127.0.0.1 — IP tin cậy).
func rlGet(t *testing.T, srv *httptest.Server, path, ip, token string) *http.Response {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL+path, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if ip != "" {
		req.Header.Set("X-Forwarded-For", ip)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}

// rlHeaders trả (limit, remaining) nếu phản hồi có header rate limit; ok=false khi Redis chậm và
// middleware đã cho qua (fail-open) — hợp lệ theo 03-AC7, test không được coi là lỗi.
func rlHeaders(resp *http.Response) (limit, remaining int, ok bool) {
	l, err1 := strconv.Atoi(resp.Header.Get("X-RateLimit-Limit"))
	r, err2 := strconv.Atoi(resp.Header.Get("X-RateLimit-Remaining"))
	return l, r, err1 == nil && err2 == nil
}

// rlFirst429 gửi tối đa n request và trả phản hồi 429 đầu tiên (nil nếu không có).
func rlFirst429(t *testing.T, srv *httptest.Server, ip, token string, n int) *http.Response {
	t.Helper()
	for range n {
		if resp := rlGet(t, srv, rlPath(), ip, token); resp.StatusCode == http.StatusTooManyRequests {
			return resp
		}
	}
	return nil
}

// TestRateLimit_IP: giới hạn IP = 5 → 5 request đầu được qua, sau đó 429 RATE_LIMITED kèm
// Retry-After (1..60) = `retry_after`; X-RateLimit-Limit/Remaining có mặt và Remaining giảm dần (03-AC6).
func TestRateLimit_IP(t *testing.T) {
	t.Parallel()
	d, _ := rlDeps(t, nil)
	srv := rlServer(t, d)
	ip := rlIP()

	var remaining []int
	for i := 1; i <= 5; i++ {
		resp := rlGet(t, srv, rlPath(), ip, "")
		if resp.StatusCode == http.StatusTooManyRequests {
			t.Fatalf("request %d đã bị 429 (giới hạn 5)", i)
		}
		limit, rest, ok := rlHeaders(resp)
		if !ok {
			continue // Redis chậm: cho qua, không có header (fail-open)
		}
		if limit != 5 {
			t.Errorf("request %d: X-RateLimit-Limit = %d, cần 5", i, limit)
		}
		remaining = append(remaining, rest)
	}
	if len(remaining) < 2 {
		t.Fatalf("chỉ đọc được %d lần X-RateLimit-Remaining", len(remaining))
	}
	for i := 1; i < len(remaining); i++ {
		if remaining[i] >= remaining[i-1] {
			t.Errorf("Remaining không giảm: %v", remaining)
			break
		}
	}

	resp := rlFirst429(t, srv, ip, "", 3)
	if resp == nil {
		t.Fatal("không gặp 429 sau khi vượt giới hạn 5")
	}
	body := coreJSON(t, resp.Body)
	if body["code"] != apierr.RateLimited {
		t.Fatalf("code = %v", body["code"])
	}
	retry, _ := body["retry_after"].(float64)
	if retry < 1 || retry > 60 {
		t.Errorf("retry_after = %v (cần 1..60)", body["retry_after"])
	}
	if h := resp.Header.Get("Retry-After"); h != strconv.Itoa(int(retry)) {
		t.Errorf("Retry-After = %q, retry_after = %v", h, retry)
	}
}

// TestRateLimit_User: bộ đếm theo người dùng tách khỏi bộ đếm IP — người khác không bị ảnh hưởng (03-AC6).
func TestRateLimit_User(t *testing.T) {
	t.Parallel()
	d, _ := rlDeps(t, func(c *config.Config) {
		c.RateLimitUserPerMin = 3
		c.RateLimitIPPerMin = 100000
	})
	srv := rlServer(t, d)
	ip := rlIP()
	u1, u2 := rlToken(t, d, uuid.NewString()), rlToken(t, d, uuid.NewString())

	for i := 1; i <= 3; i++ {
		if resp := rlGet(t, srv, rlPath(), ip, u1); resp.StatusCode == http.StatusTooManyRequests {
			t.Fatalf("U1 request %d đã bị 429 (giới hạn 3)", i)
		}
	}
	resp := rlFirst429(t, srv, ip, u1, 3)
	if resp == nil {
		t.Fatal("U1 không bị 429 sau khi vượt giới hạn 3")
	}
	if code := coreJSON(t, resp.Body)["code"]; code != apierr.RateLimited {
		t.Errorf("code = %v", code)
	}
	if resp := rlGet(t, srv, rlPath(), ip, u2); resp.StatusCode == http.StatusTooManyRequests {
		t.Fatal("U2 (cùng IP, bộ đếm riêng) bị 429")
	}
}

// TestRateLimit_SharedAcrossInstances: hai bản gateway dùng CHUNG bộ đếm Redis — request vào bản B
// nhìn thấy số lần đã gọi ở bản A (03-AC6).
func TestRateLimit_SharedAcrossInstances(t *testing.T) {
	t.Parallel()
	d, _ := rlDeps(t, nil)
	other := d
	other.Cfg.InstanceID = "gw-test-2" // bản gateway thứ hai, cùng Redis
	a, b := rlServer(t, d), rlServer(t, other)

	ip := rlIP()
	for i := 1; i <= 3; i++ {
		if resp := rlGet(t, a, rlPath(), ip, ""); resp.StatusCode == http.StatusTooManyRequests {
			t.Fatalf("bản A request %d đã bị 429 (giới hạn 5)", i)
		}
	}
	// Bản B phải thấy bộ đếm đã có 3: còn tối đa 2 lượt, request thứ 3 vào B bị chặn.
	shared := false
	for i := 1; i <= 3; i++ {
		resp := rlGet(t, b, rlPath(), ip, "")
		if resp.StatusCode == http.StatusTooManyRequests {
			shared = true
			break
		}
		if _, rest, ok := rlHeaders(resp); ok && rest <= 2-i {
			shared = true
		}
	}
	if !shared {
		t.Fatal("bản B không thấy bộ đếm của bản A (mỗi bản đếm riêng?)")
	}
}

// TestRateLimit_ExemptHealth: /healthz, /api/v1/healthz, /api/v1/readyz miễn giới hạn kể cả khi đã vượt (03-AC6).
func TestRateLimit_ExemptHealth(t *testing.T) {
	t.Parallel()
	d, _ := rlDeps(t, nil)
	srv := rlServer(t, d)
	ip := rlIP()

	if rlFirst429(t, srv, ip, "", 10) == nil {
		t.Fatal("chưa vượt được giới hạn")
	}
	for _, path := range []string{"/healthz", "/api/v1/healthz", "/api/v1/readyz"} {
		if resp := rlGet(t, srv, path, ip, ""); resp.StatusCode != http.StatusOK {
			t.Errorf("%s → %d, cần 200 (miễn rate limit)", path, resp.StatusCode)
		}
	}
}

// TestRateLimit_ForwardedFor: X-Forwarded-For chỉ được tin khi kết nối đến từ TRUSTED_PROXY_CIDRS;
// ngược lại đếm theo địa chỉ kết nối (SRS 6.5).
func TestRateLimit_ForwardedFor(t *testing.T) {
	t.Parallel()
	trusted, _ := rlDeps(t, nil)
	srv := rlServer(t, trusted)

	ip1, ip2 := rlIP(), rlIP()
	for i := 1; i <= 5; i++ {
		if resp := rlGet(t, srv, rlPath(), ip1, ""); resp.StatusCode == http.StatusTooManyRequests {
			t.Fatalf("XFF ip1 request %d đã bị 429", i)
		}
	}
	if rlFirst429(t, srv, ip1, "", 3) == nil {
		t.Fatal("XFF ip1 không bị 429 sau khi vượt giới hạn 5")
	}
	if resp := rlGet(t, srv, rlPath(), ip2, ""); resp.StatusCode == http.StatusTooManyRequests {
		t.Fatal("XFF ip2 (bộ đếm khác) bị 429 lây")
	}

	// Nguồn không tin cậy: XFF bị bỏ qua → mọi XFF rơi vào cùng bộ đếm của địa chỉ kết nối.
	untrusted := trusted
	untrusted.Cfg.TrustedProxyCIDRs = nil
	srv2 := rlServer(t, untrusted)
	hit429 := false
	for range 6 {
		if rlGet(t, srv2, rlPath(), rlIP(), "").StatusCode == http.StatusTooManyRequests {
			hit429 = true
		}
	}
	if !hit429 {
		t.Fatal("XFF từ nguồn không tin cậy vẫn tách bộ đếm (phải dùng địa chỉ kết nối)")
	}
}

// TestRateLimit_RedisDown_FailOpen: Redis chết → request thường vẫn được xử lý, độ trễ thêm so với
// Redis sống không quá 50 ms, log warn tối đa một lần mỗi 10 s (03-AC7).
func TestRateLimit_RedisDown_FailOpen(t *testing.T) {
	t.Parallel()
	d, buf := rlDeps(t, nil)
	d.Cfg.RateLimitIPPerMin = 1_000_000 // đo độ trễ, không đo chặn
	live := rlServer(t, d)

	down := d
	dead, err := appredis.New(t.Context(), "redis://127.0.0.1:1/0")
	if err != nil {
		t.Fatalf("redis.New: %v", err)
	}
	t.Cleanup(func() { _ = dead.Close() })
	down.Redis = dead
	downSrv := rlServer(t, down)

	median := func(srv *httptest.Server) time.Duration {
		t.Helper()
		times := make([]time.Duration, 0, 20)
		for i := range 20 {
			start := time.Now()
			resp := rlGet(t, srv, rlPath(), rlIP(), "")
			times = append(times, time.Since(start))
			if resp.StatusCode == http.StatusTooManyRequests {
				t.Fatalf("request %d bị 429, cần cho qua (fail-open)", i)
			}
		}
		slices.Sort(times)
		return times[len(times)/2]
	}
	base := median(live)
	withoutRedis := median(downSrv)
	if extra := withoutRedis - base; extra > 50*time.Millisecond {
		t.Fatalf("Redis chết làm chậm thêm %v (baseline %v, thực tế %v) — cần ≤ 50 ms", extra, base, withoutRedis)
	}

	warns := 0
	for _, line := range coreLogLines(t, buf.String()) {
		if line["level"] == "WARN" && strings.Contains(fmt.Sprint(line["msg"]), "rate limit") {
			warns++
		}
	}
	// Tối đa 1 dòng mỗi 10 s cho MỖI bản router; test dựng 2 bản (Redis sống + Redis chết).
	if warns < 1 || warns > 2 {
		t.Fatalf("số dòng warn rate limit trong 40 request = %d, cần 1..2 (tối đa 1 mỗi 10 s mỗi bản)", warns)
	}
}
