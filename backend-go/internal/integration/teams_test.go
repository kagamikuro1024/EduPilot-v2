package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/cmd/mock-graph/server"
	"github.com/edupilot/backend-go/internal/mailin"
	"github.com/edupilot/backend-go/internal/submission/source"
	"github.com/edupilot/backend-go/internal/testutil"
)

// ---- AC2, AC3: bảy kịch bản của mock-graph chạy lần lượt đúng kết quả ----

// TestTeamsSyncAgainstMockGraph — AC3: happy / paged / throttle / expired_token / no_consent / slow / partial_failure; số lần xin token, số 429, số trang đọc từ GET /__mock/requests.
func TestTeamsSyncAgainstMockGraph(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		r := newTM(t)
		res := r.mustRun()
		require.Equal(t, 25, res.Fetched)
		require.Equal(t, 25, res.Created, "24 khớp lớp + 1 UNMATCHED")
		require.Equal(t, 1, res.Unmatched)
		require.Empty(t, res.Failed)
		require.Equal(t, 25, r.subs())
		require.Equal(t, 24, r.count(`select count(*)::int from submissions where course_id = $1 and source = 'TEAMS' and match_status = 'MATCHED'`, r.w.Course))
		var reason string
		require.NoError(t, r.pool.QueryRow(t.Context(), `select unmatched_reason from submissions where course_id = $1 and match_status = 'UNMATCHED'`, r.w.Course).Scan(&reason))
		require.NotEmpty(t, reason)
		l := r.requests()
		require.Equal(t, 1, l.TokenRequests)
		require.Equal(t, 1, l.count("/members"))
		require.Equal(t, 1, l.count("/submissions"))
		for _, q := range l.Requests {
			require.NotContains(t, q.Path, "/users/", "không có lệnh /users/{id} mỗi sinh viên (TLR-5)")
		}
	})
	t.Run("paged", func(t *testing.T) {
		r := newTM(t)
		r.scenario("paged")
		res := r.mustRun()
		require.Equal(t, 25, res.Created)
		l := r.requests()
		require.Equal(t, 3, l.count("/submissions"), "30 bản ghi qua 3 trang")
		require.Equal(t, 3, l.count("/members"))
	})
	t.Run("throttle", func(t *testing.T) {
		r := newTM(t)
		r.scenario("throttle")
		start := time.Now()
		res := r.mustRun()
		require.Equal(t, 25, res.Created)
		require.Len(t, r.sl.waits, 2, "hai lần 429")
		for _, w := range r.sl.waits {
			require.GreaterOrEqual(t, w, time.Second)
		}
		require.Less(t, time.Since(start), 20*time.Second, "không ngủ thật")
		require.Equal(t, 2, r.requests().Throttled)
	})
	t.Run("expired_token", func(t *testing.T) {
		r := newTM(t)
		r.scenario("expired_token")
		res := r.mustRun()
		require.Equal(t, 25, res.Created)
		require.Equal(t, 2, r.requests().TokenRequests, "đúng 2 lần xin token")
		r2 := newTM(t)
		r2.scenario("unauthorized")
		_, err := r2.run()
		require.Equal(t, source.CodeAuthFailed, sourceCode(err))
		require.Equal(t, 2, r2.requests().TokenRequests, "xin lại một lần rồi dừng, không vòng lặp")
	})
	t.Run("no_consent", func(t *testing.T) {
		r := newTM(t)
		r.scenario("no_consent")
		_, err := r.run()
		require.Equal(t, source.CodeConsentRequired, sourceCode(err))
		require.Zero(t, r.subs())
	})
	t.Run("slow", func(t *testing.T) {
		r := newTM(t)
		r.sync.Sleep = nil
		r.scenario("slow")
		start := time.Now()
		res := r.mustRun()
		el := time.Since(start)
		require.Equal(t, 25, res.Created)
		require.LessOrEqual(t, el, 45*time.Second, "1 token + 1 trang thành viên + 1 trang bài nộp + 25 tệp ÷ 4 luồng ≈ 30 giây")
		require.Less(t, el, 120*time.Second)
		l := r.requests()
		require.Equal(t, 1, l.TokenRequests)
		require.Equal(t, 1, l.count("/members"))
	})
	t.Run("partial_failure", func(t *testing.T) {
		r := newTM(t)
		r.scenario("partial_failure")
		res := r.mustRun()
		require.Equal(t, 24, res.Created)
		sv14 := submissionIDOf(t, testutil.MockGraphDataDir(), "sv14")
		require.Equal(t, []source.Failure{{SubmissionID: sv14, Code: source.CodeFileDownload}}, res.Failed)
		r.scenario("happy")
		again := r.mustRun()
		require.Equal(t, 1, again.Created, "bài thiếu về")
		require.Equal(t, 24, again.Deduped, "24 bài kia không trùng")
		require.Equal(t, 25, r.subs())
	})
}

// TestTeamsSyncTwiceNoDuplicate — AC2: đồng bộ lần hai ngay sau đó không tạo bài trùng.
func TestTeamsSyncTwiceNoDuplicate(t *testing.T) {
	r := newTM(t)
	r.mustRun()
	again := r.mustRun()
	require.Zero(t, again.Created)
	require.Equal(t, 25, again.Deduped)
	require.Equal(t, 25, r.subs())
	require.Zero(t, r.count(`select count(*)::int from (select external_ref, external_ts from submissions where course_id = $1 and source = 'TEAMS' group by 1, 2 having count(*) > 1) x`, r.w.Course))
}

// shiftedData sao chép dữ liệu mock-graph và dời giờ nộp của `account` thêm `d` (sinh viên nộp lại trên Teams: cùng graph_submission_id, submittedDateTime mới).
func shiftedData(t *testing.T, account string, d time.Duration) string {
	t.Helper()
	src := testutil.MockGraphDataDir()
	dst := t.TempDir()
	require.NoError(t, os.CopyFS(dst, os.DirFS(src)))
	b, err := os.ReadFile(filepath.Join(dst, "submissions.json"))
	require.NoError(t, err)
	var all []map[string]any
	require.NoError(t, json.Unmarshal(b, &all))
	for _, s := range all {
		if s["account"] == account {
			at, err := time.Parse(time.RFC3339Nano, s["submittedDateTime"].(string))
			require.NoError(t, err)
			s["submittedDateTime"] = at.Add(d).UTC().Format("2006-01-02T15:04:05.0000000Z")
		}
	}
	out, _ := json.Marshal(all)
	require.NoError(t, os.WriteFile(filepath.Join(dst, "submissions.json"), out, 0o600))
	return dst
}

// TestTeamsResubmitNewVersion — AC2: nộp lại (cùng graph_submission_id, submittedDateTime mới) → phiên bản MỚI, bản cũ giữ; bản mới nhất đã REVIEWED → UNMATCHED ALREADY_REVIEWED.
func TestTeamsResubmitNewVersion(t *testing.T) {
	r := newTM(t)
	r.mustRun()
	_, ts2 := r.startMock(shiftedData(t, "sv.gioi", time.Hour))
	r.rebuild(ts2.URL)
	res := r.mustRun()
	require.Equal(t, 1, res.Created, "chỉ bản nộp lại là mới")
	require.Equal(t, 24, res.Deduped)
	uid := r.w.Students["sv.gioi"]
	require.Equal(t, 2, r.count(`select count(*)::int from submissions where course_id = $1 and student_id = $2 and match_status = 'MATCHED'`, r.w.Course, uid))
	var latest, oldest int
	require.NoError(t, r.pool.QueryRow(t.Context(), `select max(version), min(version) from submissions where course_id = $1 and student_id = $2`, r.w.Course, uid).Scan(&latest, &oldest))
	require.Equal(t, 2, latest)
	require.Equal(t, 1, oldest, "bản cũ giữ")
	// bản mới nhất đã duyệt → nộp lại nữa vào UNMATCHED ALREADY_REVIEWED
	r.exec(`insert into grading_results (course_id, submission_id, run_state, status) select course_id, id, 'DONE', 'REVIEWED' from submissions where course_id = $1 and student_id = $2 and version = 2`, r.w.Course, uid)
	_, ts3 := r.startMock(shiftedData(t, "sv.gioi", 2*time.Hour))
	r.rebuild(ts3.URL)
	res = r.mustRun()
	require.Equal(t, 1, res.Created)
	require.Equal(t, 1, res.Unmatched)
	var reason string
	require.NoError(t, r.pool.QueryRow(t.Context(), `select unmatched_reason from submissions where course_id = $1 and match_status = 'UNMATCHED' and candidate_student_id = $2`, r.w.Course, uid).Scan(&reason))
	require.Equal(t, "ALREADY_REVIEWED", reason)
	require.Equal(t, 2, r.count(`select max(version)::int from submissions where course_id = $1 and student_id = $2`, r.w.Course, uid), "không đè bản đã duyệt")
}

func (r *tmRig) exec(sql string, args ...any) {
	r.t.Helper()
	_, err := r.pool.Exec(r.t.Context(), sql, args...)
	require.NoError(r.t, err, sql)
}

// ---- AC8: bảo mật ----

// TestTeamsSecretWriteOnly — AC8: client_secret mã hoá AES-GCM, không bao giờ trả lại (GET, PUT, audit, last_error, state).
func TestTeamsSecretWriteOnly(t *testing.T) {
	r := newTM(t)
	svc := &mailin.Service{Pool: r.pool, Cipher: r.cipher, Redis: r.rdb, Teams: mailin.TeamsEnv{Enabled: true, Source: r.sync.Source}, TestTimeout: 3 * time.Second}
	admin := r.w.Teacher
	course := testutil.NewTeamsWorld(t, r.pool).Course // lớp chưa có cấu hình Teams
	canary := "CANARY-GRAPH-SECRET-4417"
	v, err := svc.TeamsPut(t.Context(), admin, course, mailin.TeamsPutIn{TenantID: r.meta.TenantID, ClientID: r.meta.ClientID, ClientSecret: canary, Enabled: true})
	require.NoError(t, err)
	got, err := svc.Get(t.Context(), course, mailin.KindTeams)
	require.NoError(t, err)
	for _, x := range []mailin.View{v, got} {
		raw, _ := json.Marshal(x)
		require.NotContains(t, string(raw), "CANARY")
		require.True(t, x.Configured)
	}
	res, err := svc.TeamsTest(t.Context(), course) // secret sai so với mock → AUTH, last_error là mã ngắn
	require.NoError(t, err)
	require.False(t, res.OK)
	require.Equal(t, source.CodeAuthFailed, *res.Kind)
	var enc []byte
	var dump string
	require.NoError(t, r.pool.QueryRow(t.Context(), `select config_enc from integration_configs where course_id = $1 and kind = 'TEAMS'`, course).Scan(&enc))
	require.NotContains(t, string(enc), "CANARY")
	plain, err := r.cipher.Decrypt(enc, []byte(course.String()+":TEAMS"))
	require.NoError(t, err)
	require.Contains(t, string(plain), canary)
	require.NoError(t, r.pool.QueryRow(t.Context(), `select coalesce((select string_agg(before::text || after::text, '') from audit_log where course_id = $1), '') || (select config_public::text || state::text || coalesce(last_error, '') from integration_configs where course_id = $1 and kind = 'TEAMS')`, course).Scan(&dump))
	require.NotContains(t, dump, "CANARY")
	require.Contains(t, dump, `"secret_changed": true`)
}

// TestTeamsTokenNeverLogged — AC8: token truy cập chỉ ở bộ nhớ: không có trong log, DB (audit, jobs, cấu hình, kết quả) hay phản hồi.
func TestTeamsTokenNeverLogged(t *testing.T) {
	r := newTM(t)
	res := r.mustRun()
	require.Equal(t, 25, res.Created)
	require.NotContains(t, r.logs.String(), "mock.")
	require.NotContains(t, r.logs.String(), r.meta.ClientSecret)
	raw, _ := json.Marshal(res)
	require.NotContains(t, string(raw), "mock.")
	for _, q := range []string{`select count(*)::int from audit_log where after::text like '%mock.%' or before::text like '%mock.%'`,
		`select count(*)::int from jobs where result::text like '%mock.%' or error::text like '%mock.%'`,
		`select count(*)::int from integration_configs where state::text like '%mock.%' or coalesce(last_error, '') like '%mock.%' or config_public::text like '%mock.%'`,
		`select count(*)::int from outbox where payload::text like '%mock.%'`} {
		require.Zero(t, r.count(q), q)
	}
}

// TestTeamsNoWriteBack — AC8: không có lời gọi ghi về Teams: nhật ký mock chỉ thấy GET (và POST xin token).
func TestTeamsNoWriteBack(t *testing.T) {
	r := newTM(t)
	r.mustRun()
	for _, q := range r.requests().Requests {
		if q.Method == http.MethodPost {
			require.True(t, strings.HasSuffix(q.Path, "/oauth2/v2.0/token"), "POST ngoài xin token: %s", q.Path)
			continue
		}
		require.Equal(t, http.MethodGet, q.Method, q.Path)
	}
}

// fakeGraph là máy chủ Graph tối thiểu (JSON theo tài liệu) trỏ tệp tới `fileURL(host)`; ghi lại Authorization của yêu cầu /files/.
type fakeGraph struct {
	srv      *httptest.Server
	mu       sync.Mutex
	fileAuth []string
	fileHits atomic.Int32
	fileURL  func(host string) string
	fileH    func(w http.ResponseWriter, r *http.Request)
}

func newFakeGraph(t *testing.T) *fakeGraph {
	g := &fakeGraph{}
	g.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		host := "http://" + r.Host
		switch {
		case strings.HasSuffix(r.URL.Path, "/oauth2/v2.0/token"):
			_ = json.NewEncoder(w).Encode(map[string]any{"token_type": "Bearer", "expires_in": 3600, "access_token": "tok-fake"})
		case strings.HasSuffix(r.URL.Path, "/members"):
			_ = json.NewEncoder(w).Encode(map[string]any{"value": []any{map[string]any{"id": "u1", "mail": "sv.kha@edupilot.local", "userPrincipalName": "sv.kha@edupilot.local"}}})
		case strings.HasSuffix(r.URL.Path, "/submissions"):
			_ = json.NewEncoder(w).Encode(map[string]any{"value": []any{map[string]any{"id": "sub-1", "status": "submitted", "submittedDateTime": "2026-11-30T10:00:00Z", "recipient": map[string]any{"userId": "u1"},
				"submittedResources": []any{map[string]any{"id": "r1", "resource": map[string]any{"displayName": "bai.md", "fileUrl": g.fileURL(host)}}}}}})
		case strings.HasPrefix(r.URL.Path, "/files/"):
			g.fileHits.Add(1)
			g.mu.Lock()
			g.fileAuth = append(g.fileAuth, r.Header.Get("Authorization"))
			g.mu.Unlock()
			if g.fileH != nil {
				g.fileH(w, r)
				return
			}
			_, _ = w.Write([]byte("# bài\nnội dung"))
		default:
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(g.srv.Close)
	return g
}

func (g *fakeGraph) teams(hosts ...string) *source.Teams {
	return &source.Teams{Cfg: source.Config{BaseURL: g.srv.URL + "/v1.0", AuthURL: g.srv.URL, FileHosts: hosts, DownloadParallel: 2, FileRetry: 0, RetryMax: 1, FileRetryWait: time.Millisecond}, Sleep: &tmSleeper{}}
}

func hostOf(u string) string { return strings.TrimPrefix(u, "http://") }

type files struct {
	mu  sync.Mutex
	got []source.Fetched
}

func (f *files) on(_ context.Context, x source.Fetched) error {
	f.mu.Lock()
	f.got = append(f.got, x)
	f.mu.Unlock()
	return nil
}

var tcred = source.Credentials{Tenant: "t", ClientID: "c", ClientSecret: "s"}

// TestTeamsDownloadHostAllowlist — AC8: fileUrl ở host ngoài GRAPH_FILE_HOSTS không bao giờ được gọi (không SSRF tới địa chỉ tuỳ ý); liệt kê host thì tải được.
func TestTeamsDownloadHostAllowlist(t *testing.T) {
	other := newFakeGraph(t)
	g := newFakeGraph(t)
	g.fileURL = func(string) string { return other.srv.URL + "/files/x" }
	var f files
	rep, err := g.teams().Sync(t.Context(), tcred, "c", "a", f.on, nil)
	require.NoError(t, err)
	require.Equal(t, []source.Failure{{SubmissionID: "sub-1", Code: source.CodeFileHost}}, rep.Failed)
	require.Zero(t, other.fileHits.Load(), "host ngoài danh sách không bị gọi")
	require.Empty(t, f.got)
	_, err = g.teams(hostOf(g.srv.URL), hostOf(other.srv.URL)).Sync(t.Context(), tcred, "c", "a", f.on, nil)
	require.NoError(t, err)
	require.Len(t, f.got, 1)
	require.EqualValues(t, 1, other.fileHits.Load())
}

// TestTeamsRedirectOnlyListedHosts — AC8: chuyển hướng (như Graph thật → SharePoint) chỉ được theo khi host nằm trong GRAPH_FILE_HOSTS.
func TestTeamsRedirectOnlyListedHosts(t *testing.T) {
	cdn := newFakeGraph(t)
	g := newFakeGraph(t)
	g.fileURL = func(host string) string { return host + "/files/r" }
	g.fileH = func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, cdn.srv.URL+"/files/real", http.StatusFound)
	}
	var f files
	rep, err := g.teams().Sync(t.Context(), tcred, "c", "a", f.on, nil)
	require.NoError(t, err)
	require.Len(t, rep.Failed, 1, "chuyển hướng tới host không liệt kê bị từ chối")
	require.Zero(t, cdn.fileHits.Load())
	_, err = g.teams(hostOf(g.srv.URL), hostOf(cdn.srv.URL)).Sync(t.Context(), tcred, "c", "a", f.on, nil)
	require.NoError(t, err)
	require.Len(t, f.got, 1)
	require.EqualValues(t, 1, cdn.fileHits.Load())
}

// TestTeamsNoAuthHeaderToForeignHost — AC8 / TLR-20: chuyển hướng sang host khác host Graph KHÔNG mang theo Authorization; host Graph thì có.
func TestTeamsNoAuthHeaderToForeignHost(t *testing.T) {
	cdn := newFakeGraph(t)
	g := newFakeGraph(t)
	g.fileURL = func(host string) string { return host + "/files/r" }
	g.fileH = func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, cdn.srv.URL+"/files/real", http.StatusFound)
	}
	var f files
	_, err := g.teams(hostOf(g.srv.URL), hostOf(cdn.srv.URL)).Sync(t.Context(), tcred, "c", "a", f.on, nil)
	require.NoError(t, err)
	require.Equal(t, []string{"Bearer tok-fake"}, g.fileAuth, "host Graph nhận token")
	require.Equal(t, []string{""}, cdn.fileAuth, "host khác không nhận Authorization")
}

// TestTeamsNoAuthHeaderToListedSecondaryHost — AC8 / TLR-20 / BUG-P7-11-1: `fileUrl` trỏ THẲNG (không chuyển hướng) tới host phụ ĐÃ liệt kê trong GRAPH_FILE_HOSTS thì được tải nhưng KHÔNG mang Authorization;
// chỉ host Graph nhận token. (Bắt đột biến "gửi Authorization tới mọi host".)
func TestTeamsNoAuthHeaderToListedSecondaryHost(t *testing.T) {
	other := newFakeGraph(t)
	g := newFakeGraph(t)
	g.fileURL = func(string) string { return other.srv.URL + "/files/x" }
	var f files
	_, err := g.teams(hostOf(g.srv.URL), hostOf(other.srv.URL)).Sync(t.Context(), tcred, "c", "a", f.on, nil)
	require.NoError(t, err)
	require.Len(t, f.got, 1, "host phụ đã liệt kê được tải")
	require.Equal(t, []string{""}, other.fileAuth, "host phụ không nhận Authorization")
	require.Empty(t, g.fileAuth, "host Graph không bị gọi tải tệp ở ca này")
	// ngược lại: tệp ở chính host Graph thì nhận token
	g2 := newFakeGraph(t)
	g2.fileURL = func(host string) string { return host + "/files/y" }
	var f2 files
	_, err = g2.teams(hostOf(g2.srv.URL)).Sync(t.Context(), tcred, "c", "a", f2.on, nil)
	require.NoError(t, err)
	require.Equal(t, []string{"Bearer tok-fake"}, g2.fileAuth)
}

// ---- AC9: nhánh lỗi ----

// outageGraph bọc mock-graph: sau `after` yêu cầu tải tệp thì cắt kết nối (như máy chủ tắt giữa chừng).
func outageGraph(t *testing.T, inner http.Handler, after int32) *httptest.Server {
	var n atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/files/") && n.Add(1) > after {
			if hj, ok := w.(http.Hijacker); ok {
				c, _, _ := hj.Hijack()
				_ = c.Close()
				return
			}
		}
		inner.ServeHTTP(w, r)
	}))
	t.Cleanup(ts.Close)
	return ts
}

// TestTeamsMidSyncOutage — AC9: máy chủ tắt giữa chừng → UNREACHABLE; bài đã về giữ nguyên; không giữ khoá treo.
func TestTeamsMidSyncOutage(t *testing.T) {
	r := newTM(t)
	r.rebuild(outageGraph(t, r.mock, 8).URL)
	_, err := r.run()
	require.Equal(t, source.CodeUnreachable, sourceCode(err))
	got := r.subs()
	require.Positive(t, got, "bài đã về giữ nguyên")
	require.Less(t, got, 25)
	exists, err := r.rdb.Exists(t.Context(), "ep:lock:teams:"+r.w.Asg.String()).Result()
	require.NoError(t, err)
	require.Zero(t, exists, "không giữ khoá treo")
}

// TestTeamsResumeAfterOutage — AC9: sau sự cố chạy lại → đủ 25 bài, không trùng.
func TestTeamsResumeAfterOutage(t *testing.T) {
	r := newTM(t)
	r.rebuild(outageGraph(t, r.mock, 8).URL)
	_, err := r.run()
	require.Error(t, err)
	partial := r.subs()
	r.rebuild(r.ts.URL)
	res := r.mustRun()
	require.Equal(t, 25-partial, res.Created)
	require.Equal(t, 25, r.subs())
	require.Zero(t, r.count(`select count(*)::int from (select external_ref, external_ts from submissions where course_id = $1 and source = 'TEAMS' group by 1, 2 having count(*) > 1) x`, r.w.Course))
}

// TestTeamsInvalidJSON — AC9: JSON thiếu trường → TEAMS_RESPONSE_INVALID, không tạo bài, không panic.
func TestTeamsInvalidJSON(t *testing.T) {
	r := newTM(t)
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if strings.HasSuffix(req.URL.Path, "/submissions") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"value":[{"id":"x"}]}`))
			return
		}
		r.mock.ServeHTTP(w, req)
	}))
	t.Cleanup(bad.Close)
	r.rebuild(bad.URL)
	_, err := r.run()
	require.Equal(t, source.CodeResponseInvalid, sourceCode(err))
	require.Zero(t, r.subs())
}

// TestTeamsSyncTimeout — AC9: quá TEAMS_SYNC_TIMEOUT → TIMEOUT, bài đã về giữ nguyên, khoá nhả.
func TestTeamsSyncTimeout(t *testing.T) {
	r := newTM(t)
	slow, err := server.New(server.Options{DataDir: testutil.MockGraphDataDir(), SlowDelay: 700 * time.Millisecond})
	require.NoError(t, err)
	ts := httptest.NewServer(slow)
	t.Cleanup(ts.Close)
	b, _ := json.Marshal(map[string]string{"name": "slow"})
	resp, err := http.Post(ts.URL+"/__mock/scenario", "application/json", bytes.NewReader(b))
	require.NoError(t, err)
	_ = resp.Body.Close()
	r.rebuild(ts.URL)
	r.sync.Sleep = nil
	r.sync.Timeout = 1500 * time.Millisecond
	_, err = r.run()
	require.Equal(t, source.CodeTimeout, sourceCode(err))
	exists, err := r.rdb.Exists(t.Context(), "ep:lock:teams:"+r.w.Asg.String()).Result()
	require.NoError(t, err)
	require.Zero(t, exists)
}

// ---- AC10: cô lập lớp ----

// TestTeamsCourseIsolation — AC10: cấu hình Teams của lớp 1 không dùng được để kéo bài về lớp 2.
func TestTeamsCourseIsolation(t *testing.T) {
	r := newTM(t)
	w2 := testutil.NewTeamsWorld(t, r.pool) // lớp 2 cùng bài + teams id, KHÔNG có cấu hình Teams
	_, err := r.sync.Run(t.Context(), w2.Course, w2.Asg, nil)
	var ae interface{ Error() string }
	require.ErrorAs(t, err, &ae)
	require.Contains(t, err.Error(), "TEAMS_NOT_CONFIGURED")
	_, err = r.sync.Enqueue(t.Context(), w2.Teacher, w2.Course, w2.Asg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "TEAMS_NOT_CONFIGURED")
	require.Zero(t, r.count(`select count(*)::int from submissions where course_id = $1`, w2.Course))
	r.mustRun()
	require.Zero(t, r.count(`select count(*)::int from submissions where course_id = $1`, w2.Course), "đồng bộ lớp 1 không đụng lớp 2")
	// bài của lớp 1 không đồng bộ được qua đường của lớp 2
	_, err = r.sync.Run(t.Context(), w2.Course, r.w.Asg, nil)
	require.Error(t, err)
}
