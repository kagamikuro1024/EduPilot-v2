package source_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/submission/source"
)

const docs = "../../../testdata/graph-docs/"

func docJSON(t *testing.T, name string) map[string]any {
	t.Helper()
	b, err := os.ReadFile(docs + name)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(b, &m))
	delete(m, "_source")
	return m
}

// fakeSleeper ghi lại các lần chờ mà không ngủ thật.
type fakeSleeper struct {
	mu    sync.Mutex
	waits []time.Duration
}

func (f *fakeSleeper) Sleep(_ context.Context, d time.Duration) error {
	f.mu.Lock()
	f.waits = append(f.waits, d)
	f.mu.Unlock()
	return nil
}

type graph struct {
	t        *testing.T
	srv      *httptest.Server
	mu       sync.Mutex
	calls    []string
	tokens   atomic.Int32
	throttle atomic.Int32 // số yêu cầu Graph đầu trả 429
	status   atomic.Int32 // nếu ≠ 0: mọi yêu cầu Graph trả mã này
	members  func(host string) any
	subs     func(host string, page int) any
	files    map[string]func(w http.ResponseWriter, r *http.Request)
	inflight atomic.Int32
	peak     atomic.Int32
	expire1  atomic.Bool // yêu cầu Graph thứ hai của token đầu trả 401
	usesTok1 atomic.Int32
	tok1     atomic.Value
}

func newGraph(t *testing.T) *graph {
	g := &graph{t: t, files: map[string]func(http.ResponseWriter, *http.Request){}}
	g.srv = httptest.NewServer(http.HandlerFunc(g.serve))
	t.Cleanup(g.srv.Close)
	return g
}

func (g *graph) record(r *http.Request) {
	g.mu.Lock()
	g.calls = append(g.calls, r.Method+" "+r.URL.Path+"?"+r.URL.RawQuery)
	g.mu.Unlock()
}

func jsonOut(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func (g *graph) serve(w http.ResponseWriter, r *http.Request) {
	g.record(r)
	host := "http://" + r.Host
	if strings.HasSuffix(r.URL.Path, "/oauth2/v2.0/token") {
		n := g.tokens.Add(1)
		tok := fmt.Sprintf("tok-%d", n)
		if n == 1 {
			g.tok1.Store(tok)
		}
		jsonOut(w, 200, map[string]any{"token_type": "Bearer", "expires_in": 3600, "access_token": tok})
		return
	}
	auth := r.Header.Get("Authorization")
	if g.expire1.Load() && auth == "Bearer "+fmt.Sprint(g.tok1.Load()) && g.usesTok1.Add(1) >= 2 {
		jsonOut(w, 401, map[string]any{"error": map[string]any{"code": "InvalidAuthenticationToken"}})
		return
	}
	if g.throttle.Load() > 0 && g.throttle.Add(-1) >= 0 {
		w.Header().Set("Retry-After", "1")
		jsonOut(w, 429, map[string]any{"error": map[string]any{"code": "TooManyRequests"}})
		return
	}
	if c := int(g.status.Load()); c != 0 {
		jsonOut(w, c, map[string]any{"error": map[string]any{"code": "x"}})
		return
	}
	switch {
	case strings.HasSuffix(r.URL.Path, "/members"):
		jsonOut(w, 200, g.members(host))
	case strings.HasSuffix(r.URL.Path, "/submissions"):
		page := 0
		_, _ = fmt.Sscan(r.URL.Query().Get("$skiptoken"), &page)
		jsonOut(w, 200, g.subs(host, page))
	case strings.HasPrefix(r.URL.Path, "/files/"):
		n := g.inflight.Add(1)
		defer g.inflight.Add(-1)
		for {
			p := g.peak.Load()
			if n <= p || g.peak.CompareAndSwap(p, n) {
				break
			}
		}
		time.Sleep(30 * time.Millisecond)
		if h := g.files[r.URL.Path]; h != nil {
			h(w, r)
			return
		}
		_, _ = w.Write([]byte("# bài làm\nnội dung"))
	default:
		jsonOut(w, 404, map[string]any{})
	}
}

func (g *graph) cfg() source.Config {
	return source.Config{BaseURL: g.srv.URL + "/v1.0", AuthURL: g.srv.URL, DownloadParallel: 4, FileRetry: 2, RetryMax: 3, FileRetryWait: time.Millisecond}
}

// docSub dựng một bài `submitted` theo ví dụ tài liệu (educationSubmission + submittedResources); fileURL tương đối được đổi thành tuyệt đối.
func docSub(t *testing.T, host, id, user, status string) map[string]any {
	t.Helper()
	b, _ := json.Marshal(docJSON(t, "list-submissions-expand-submittedresources.json")["value"].([]any)[0])
	var s map[string]any
	require.NoError(t, json.Unmarshal(b, &s))
	s["id"], s["status"] = id, status
	s["recipient"].(map[string]any)["userId"] = user
	if status != "submitted" {
		s["submittedDateTime"] = nil
	}
	res := s["submittedResources"].([]any)[0].(map[string]any)["resource"].(map[string]any)
	res["fileUrl"] = host + "/files/" + id
	return s
}

func docMembers(t *testing.T, users ...string) any {
	t.Helper()
	m := docJSON(t, "list-class-members.json")
	tpl := m["value"].([]any)[1].(map[string]any)
	var list []any
	for _, u := range users {
		b, _ := json.Marshal(tpl)
		var c map[string]any
		_ = json.Unmarshal(b, &c)
		c["id"], c["userPrincipalName"] = u, u+"@truong.edu"
		list = append(list, c)
	}
	m["value"] = list
	return m
}

func (g *graph) teams(sl source.Sleeper) *source.Teams {
	return &source.Teams{Cfg: g.cfg(), Sleep: sl}
}

var cred = source.Credentials{Tenant: "tenant-1", ClientID: "client-1", ClientSecret: "s3cr3t"}

type sink struct {
	mu   sync.Mutex
	got  []source.Fetched
	prog [][2]int
}

func (s *sink) file(_ context.Context, f source.Fetched) error {
	s.mu.Lock()
	s.got = append(s.got, f)
	s.mu.Unlock()
	return nil
}
func (s *sink) progress(d, t int) { s.mu.Lock(); s.prog = append(s.prog, [2]int{d, t}); s.mu.Unlock() }

func basicGraph(t *testing.T, n int) *graph {
	g := newGraph(t)
	var users []string
	for i := range n {
		users = append(users, fmt.Sprintf("u%02d", i))
	}
	g.members = func(string) any { return docMembers(t, users...) }
	g.subs = func(host string, _ int) any {
		var v []any
		for i := range n {
			v = append(v, docSub(t, host, fmt.Sprintf("s%02d", i), users[i], "submitted"))
		}
		return map[string]any{"value": v}
	}
	return g
}

// TestTeamsAdapterCallOrder — AC1: token (client-credentials) → MỘT lệnh thành viên → bài nộp `$expand=submittedResources` → tải tệp; không `/users/{id}`; chỉ lấy `submitted`.
func TestTeamsAdapterCallOrder(t *testing.T) {
	g := newGraph(t)
	g.members = func(string) any { return docMembers(t, "u1", "u2", "u3") }
	g.subs = func(host string, _ int) any {
		return map[string]any{"value": []any{docSub(t, host, "a", "u1", "submitted"), docSub(t, host, "b", "u2", "working"), docSub(t, host, "c", "u3", "returned")}}
	}
	var s sink
	rep, err := g.teams(&fakeSleeper{}).Sync(t.Context(), cred, "class-1", "asg-1", s.file, s.progress)
	require.NoError(t, err)
	require.Equal(t, 1, rep.Listed, "chỉ submitted")
	require.Len(t, s.got, 1)
	require.Equal(t, "a", s.got[0].SubmissionID)
	require.Equal(t, "u1@truong.edu", s.got[0].Email)
	require.Equal(t, "bai-lam.md", s.got[0].FileName)
	want := []string{
		"POST /tenant-1/oauth2/v2.0/token?",
		"GET /v1.0/education/classes/class-1/members?",
		"GET /v1.0/education/classes/class-1/assignments/asg-1/submissions?$expand=submittedResources",
		"GET /files/a?",
	}
	require.Equal(t, want, g.calls)
	for _, c := range g.calls {
		require.NotContains(t, c, "/users/")
	}
}

// TestTeamsAdapterPagesAndNextLinkHost — AC2: đi hết @odata.nextLink; nextLink sang host khác → TEAMS_RESPONSE_INVALID.
func TestTeamsAdapterPagesAndNextLinkHost(t *testing.T) {
	g := newGraph(t)
	g.members = func(string) any { return docMembers(t, "u0", "u1", "u2", "u3", "u4") }
	g.subs = func(host string, page int) any {
		out := map[string]any{"value": []any{docSub(t, host, fmt.Sprintf("p%d-a", page), "u0", "submitted"), docSub(t, host, fmt.Sprintf("p%d-b", page), "u1", "submitted")}}
		if page < 2 {
			out["@odata.nextLink"] = fmt.Sprintf("%s/v1.0/education/classes/c/assignments/a/submissions?$skiptoken=%d", host, page+1)
		}
		return out
	}
	var s sink
	rep, err := g.teams(&fakeSleeper{}).Sync(t.Context(), cred, "c", "a", s.file, s.progress)
	require.NoError(t, err)
	require.Equal(t, 6, rep.Listed)
	g.subs = func(host string, _ int) any {
		return map[string]any{"value": []any{}, "@odata.nextLink": "http://evil.example/next"}
	}
	_, err = g.teams(&fakeSleeper{}).Sync(t.Context(), cred, "c", "a", s.file, s.progress)
	var se *source.Error
	require.ErrorAs(t, err, &se)
	require.Equal(t, source.CodeResponseInvalid, se.Code)
}

// TestTeamsAdapterThrottleWaitsRetryAfter — AC3: 429 + Retry-After: 1 → chờ ≥ 1 giây qua Sleeper tiêm được (không ngủ thật) rồi thử lại; quá RetryMax → lỗi.
func TestTeamsAdapterThrottleWaitsRetryAfter(t *testing.T) {
	g := basicGraph(t, 3)
	g.throttle.Store(2)
	sl := &fakeSleeper{}
	var s sink
	start := time.Now()
	rep, err := g.teams(sl).Sync(t.Context(), cred, "c", "a", s.file, s.progress)
	require.NoError(t, err)
	require.Equal(t, 3, rep.Listed)
	require.Len(t, sl.waits, 2)
	for _, w := range sl.waits {
		require.GreaterOrEqual(t, w, time.Second)
	}
	require.Less(t, time.Since(start), 900*time.Millisecond, "không ngủ thật")
	g.throttle.Store(100)
	_, err = g.teams(&fakeSleeper{}).Sync(t.Context(), cred, "c", "a", s.file, s.progress)
	require.Error(t, err)
}

// TestTeamsAdapterExpiredTokenRenewsOnce — AC3: 401 → xin token mới đúng MỘT lần rồi tiếp tục; 401 lần hai liền sau token mới → TEAMS_AUTH_FAILED, không vòng lặp.
func TestTeamsAdapterExpiredTokenRenewsOnce(t *testing.T) {
	g := basicGraph(t, 4)
	g.expire1.Store(true)
	var s sink
	rep, err := g.teams(&fakeSleeper{}).Sync(t.Context(), cred, "c", "a", s.file, s.progress)
	require.NoError(t, err)
	require.Equal(t, 4, rep.Listed)
	require.EqualValues(t, 2, g.tokens.Load(), "đúng 2 lần xin token")

	g2 := basicGraph(t, 2)
	g2.status.Store(401)
	_, err = g2.teams(&fakeSleeper{}).Sync(t.Context(), cred, "c", "a", s.file, s.progress)
	var se *source.Error
	require.ErrorAs(t, err, &se)
	require.Equal(t, source.CodeAuthFailed, se.Code)
	require.EqualValues(t, 2, g2.tokens.Load(), "xin lại một lần rồi dừng")
}

// TestTeamsAdapterConsentRequired — 403 → TEAMS_CONSENT_REQUIRED, 0 bài.
func TestTeamsAdapterConsentRequired(t *testing.T) {
	g := basicGraph(t, 2)
	g.status.Store(403)
	var s sink
	_, err := g.teams(&fakeSleeper{}).Sync(t.Context(), cred, "c", "a", s.file, s.progress)
	var se *source.Error
	require.ErrorAs(t, err, &se)
	require.Equal(t, source.CodeConsentRequired, se.Code)
	require.Empty(t, s.got)
}

// TestTeamsAdapterDownloadParallelBound — AC1: tải song song, không quá TEAMS_DOWNLOAD_PARALLEL luồng; xong cả 12 bài, tiến độ tăng tới 12/12.
func TestTeamsAdapterDownloadParallelBound(t *testing.T) {
	g := basicGraph(t, 12)
	var s sink
	rep, err := g.teams(&fakeSleeper{}).Sync(t.Context(), cred, "c", "a", s.file, s.progress)
	require.NoError(t, err)
	require.Equal(t, 12, rep.Listed)
	require.Len(t, s.got, 12)
	require.LessOrEqual(t, g.peak.Load(), int32(4))
	require.Greater(t, g.peak.Load(), int32(1), "có chạy song song")
	require.Equal(t, [2]int{12, 12}, s.prog[len(s.prog)-1])
}

// TestTeamsAdapterFileRetryThenPerItemFailure — AC3: tệp 500 sau FileRetry lần thử lại → lỗi riêng FILE_DOWNLOAD_FAILED (đợt vẫn tiếp tục); tệp khác về đủ.
func TestTeamsAdapterFileRetryThenPerItemFailure(t *testing.T) {
	g := basicGraph(t, 4)
	var hits atomic.Int32
	g.files["/files/s01"] = func(w http.ResponseWriter, _ *http.Request) { hits.Add(1); w.WriteHeader(500) }
	var s sink
	rep, err := g.teams(&fakeSleeper{}).Sync(t.Context(), cred, "c", "a", s.file, s.progress)
	require.NoError(t, err)
	require.Len(t, s.got, 3)
	require.Equal(t, []source.Failure{{SubmissionID: "s01", Code: source.CodeFileDownload}}, rep.Failed)
	require.EqualValues(t, 3, hits.Load(), "1 lần đầu + 2 lần thử lại")
}

// TestTeamsAdapterInvalidJSON — JSON thiếu trường bắt buộc → TEAMS_RESPONSE_INVALID.
func TestTeamsAdapterInvalidJSON(t *testing.T) {
	for name, mut := range map[string]func(g *graph){
		"thành viên thiếu id": func(g *graph) {
			g.members = func(string) any { return map[string]any{"value": []any{map[string]any{"mail": "x@y.z"}}} }
		},
		"không có value": func(g *graph) { g.subs = func(string, int) any { return map[string]any{"oops": 1} } },
		"bài nộp thiếu status": func(g *graph) {
			g.subs = func(string, int) any { return map[string]any{"value": []any{map[string]any{"id": "x"}}} }
		},
		"submitted thiếu recipient": func(g *graph) {
			g.subs = func(string, int) any {
				return map[string]any{"value": []any{map[string]any{"id": "x", "status": "submitted", "submittedDateTime": "2026-11-01T10:00:00Z"}}}
			}
		},
	} {
		g := basicGraph(t, 2)
		mut(g)
		var s sink
		_, err := g.teams(&fakeSleeper{}).Sync(t.Context(), cred, "c", "a", s.file, s.progress)
		var se *source.Error
		require.ErrorAs(t, err, &se, name)
		require.Equal(t, source.CodeResponseInvalid, se.Code, name)
	}
}

// TestTeamsAdapterUnreachableAndTimeout — mạng đứt → UNREACHABLE; quá hạn ctx → TIMEOUT.
func TestTeamsAdapterUnreachableAndTimeout(t *testing.T) {
	g := basicGraph(t, 2)
	g.srv.Close()
	var s sink
	_, err := g.teams(&fakeSleeper{}).Sync(t.Context(), cred, "c", "a", s.file, s.progress)
	var se *source.Error
	require.ErrorAs(t, err, &se)
	require.Equal(t, source.CodeUnreachable, se.Code)

	slow := newGraph(t)
	slow.members = func(string) any { time.Sleep(400 * time.Millisecond); return docMembers(t, "u") }
	slow.subs = func(string, int) any { return map[string]any{"value": []any{}} }
	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	_, err = slow.teams(&fakeSleeper{}).Sync(ctx, cred, "c", "a", s.file, s.progress)
	require.ErrorAs(t, err, &se)
	require.Equal(t, source.CodeTimeout, se.Code)
}

var mockWord = regexp.MustCompile(`(?i)mock`)

// TestTeamsAdapterHasNoMockBranch — AC5: teams.go không có từ `mock` ngoài chú thích (không nhánh `if mock`, không đọc cờ môi trường để đổi hành vi); cấu hình chỉ là địa chỉ GRAPH_*.
func TestTeamsAdapterHasNoMockBranch(t *testing.T) {
	for _, f := range []string{"teams.go", "../teams.go"} {
		b, err := os.ReadFile(f)
		require.NoError(t, err)
		for i, line := range strings.Split(string(b), "\n") {
			code := line
			if j := strings.Index(code, "//"); j >= 0 {
				code = code[:j]
			}
			require.False(t, mockWord.MatchString(code), "%s:%d chứa 'mock' ngoài chú thích: %s", f, i+1, line)
			require.NotContains(t, code, "os.Getenv", "%s:%d đọc biến môi trường trực tiếp", f, i+1)
		}
	}
}

// TestMockGraphNotImported — AC5: không gói nào của internal/ (bản dựng thường, không tính test) nhập cmd/mock-graph.
func TestMockGraphNotImported(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", "../../...").CombinedOutput()
	require.NoError(t, err, string(out))
	require.NotContains(t, string(out), "cmd/mock-graph")
	imp, err := exec.Command("go", "list", "-deps", "../../../cmd/mock-graph/...").CombinedOutput()
	require.NoError(t, err, string(imp))
	require.NotContains(t, string(imp), "backend-go/internal/", "mock-graph không nhập internal/*")
}
