package chat_test

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/agent"
	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/httpapi"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/config"
	"github.com/edupilot/backend-go/internal/testutil"
)

const jwtSecret = "0123456789abcdef0123456789abcdef"

type httpRig struct {
	*rig
	srv *httptest.Server
	iss *auth.Issuer
}

// newHTTPRig dựng ROUTER THẬT (không phải handler riêng lẻ) với chat.Service của rig.
func newHTTPRig(t *testing.T) *httpRig {
	t.Helper()
	r := newRig(t)
	env := map[string]string{
		"DATABASE_URL": testutil.MigratedPostgresURL(t), "REDIS_URL": testutil.RedisURL(t), "JWT_SECRET_KEY": jwtSecret,
		"BLOB_ENDPOINT": "127.0.0.1:9", "BLOB_BUCKET": "x", "BLOB_ACCESS_KEY": "x", "BLOB_SECRET_KEY": "x",
		"APP_ENCRYPTION_KEY": "ZWR1cGlsb3QtZGV2LWVuY3J5cHRpb24ta2V5LTMyYnk=", "APP_ENV": "test", "BCRYPT_COST": "4",
		"RATE_LIMIT_IP_PER_MIN": "1000000", "RATE_LIMIT_USER_PER_MIN": "1000000", "REQUEST_TIMEOUT": "2s",
	}
	cfg, err := config.Load(func(k string) string { return env[k] }, config.Gateway)
	require.NoError(t, err)
	h := httpapi.NewRouter(httpapi.Deps{Cfg: cfg, Log: r.svc.Log, DB: r.pool, Redis: r.rdb, Clock: clock.Real{}, State: httpapi.NewState(), Chat: r.svc})
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return &httpRig{rig: r, srv: srv, iss: auth.NewIssuer(jwtSecret, time.Hour, clock.Real{})}
}

func (h *httpRig) tok(id uuid.UUID, role auth.Role) string {
	h.t.Helper()
	s, err := h.iss.Issue(id.String(), role, "x@example.test")
	require.NoError(h.t, err)
	return s
}

func (h *httpRig) do(method, path, tok, body string, hdr ...string) *http.Response {
	h.t.Helper()
	req, err := http.NewRequestWithContext(h.t.Context(), method, h.srv.URL+"/api/v1"+path, strings.NewReader(body))
	require.NoError(h.t, err)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	for i := 0; i+1 < len(hdr); i += 2 {
		req.Header.Set(hdr[i], hdr[i+1])
	}
	res, err := http.DefaultClient.Do(req)
	require.NoError(h.t, err)
	return res
}

// TestChatSSENotBuffered — AC1: qua router thật, byte đầu tới TRƯỚC token đầu của provider (route SSE nằm ngoài timeout / Idempotency middleware).
func TestChatSSENotBuffered(t *testing.T) {
	t.Parallel()
	h := newHTTPRig(t)
	gate := make(chan struct{})
	h.ag.fn = func(ctx context.Context, _ agent.TrustedContext, _ agent.Input) (agent.Outcome, error) {
		return agent.Outcome{Plan: agent.IntentCourseQA, Stream: h.ag.stream(ctx, gate, time.Millisecond, llm.Response{}, "một ", "hai")}, nil
	}
	t0 := time.Now()
	res := h.do(http.MethodPost, "/chat/sessions/"+h.sess.String()+"/messages", h.tok(h.student.UserID, auth.RoleStudent), `{"content":"chào"}`, "Idempotency-Key", uuid.NewString())
	defer func() { _ = res.Body.Close() }()
	require.Equal(t, 200, res.StatusCode)
	require.Equal(t, "text/event-stream", res.Header.Get("Content-Type"))
	br := bufio.NewReader(res.Body)
	var lines []string
	for { // đọc tới khung `status` đầu tiên khi provider còn treo
		l, err := br.ReadString('\n')
		require.NoError(t, err)
		lines = append(lines, strings.TrimSpace(l))
		if strings.HasPrefix(l, "event: status") {
			break
		}
	}
	if os.Getenv("EP_SKIP_TIMING") != "1" { // ngưỡng 300 ms chạy tuần tự; khi chạy song song chỉ kiểm thứ tự (sự kiện tới trước khi provider nhả)
		require.Less(t, time.Since(t0), 300*time.Millisecond)
	}
	// lượt sinh lâu hơn REQUEST_TIMEOUT (2 s) vẫn sống: route SSE không có deadline chung
	time.Sleep(2500 * time.Millisecond)
	close(gate)
	rest, err := io.ReadAll(br)
	require.NoError(t, err)
	body := strings.Join(lines, "\n") + string(rest)
	require.Contains(t, body, "event: token")
	require.Contains(t, body, "event: done")
	require.Regexp(t, regexp.MustCompile(`id: 1:\d+-\d+`), body)
}

// TestTrustedContextFromJWT — AC18/CLAUDE.md luật 2: danh tính của agent chỉ từ JWT; lớp từ phiên.
func TestTrustedContextFromJWT(t *testing.T) {
	t.Parallel()
	h := newHTTPRig(t)
	seen := make(chan agent.TrustedContext, 4)
	h.ag.fn = func(_ context.Context, tc agent.TrustedContext, _ agent.Input) (agent.Outcome, error) {
		seen <- tc
		return agent.Outcome{Plan: agent.IntentSmalltalk, Canned: "ok"}, nil
	}
	res := h.do(http.MethodPost, "/chat/sessions/"+h.sess.String()+"/messages", h.tok(h.student.UserID, auth.RoleStudent), `{"content":"chào"}`, "Idempotency-Key", uuid.NewString())
	_, _ = io.Copy(io.Discard, res.Body)
	_ = res.Body.Close()
	tc := <-seen
	require.Equal(t, h.student.UserID, tc.UserID)
	require.Equal(t, h.course, tc.CourseID)
	require.Equal(t, auth.RoleStudent, auth.Role(tc.Role))
}

// TestBodyIdentityFieldsRejected — thân có trường danh tính → 422, không có lượt sinh nào.
func TestBodyIdentityFieldsRejected(t *testing.T) {
	t.Parallel()
	h := newHTTPRig(t)
	tok := h.tok(h.student.UserID, auth.RoleStudent)
	for _, f := range []string{"user_id", "student_code", "mssv", "course_id", "email"} {
		res := h.do(http.MethodPost, "/chat/sessions/"+h.sess.String()+"/messages", tok, fmt.Sprintf(`{"content":"chào","%s":"%s"}`, f, uuid.NewString()), "Idempotency-Key", uuid.NewString())
		_ = res.Body.Close()
		require.Equal(t, 422, res.StatusCode, f)
	}
	res := h.do(http.MethodPost, "/chat/sessions/"+h.sess.String()+"/messages", tok, `{"content":"chào"}`, "Idempotency-Key", "không-phải-uuid")
	_ = res.Body.Close()
	require.Equal(t, 422, res.StatusCode)
	require.Zero(t, h.ag.calls.Load())
}

// TestChatMatrix — AC18: vai × 10 route × tình trạng ghi danh (qua router thật).
func TestChatMatrix(t *testing.T) {
	t.Parallel()
	h := newHTTPRig(t)
	_, mid, err := h.send(h.student, h.sess, "chào")
	require.NoError(t, err)
	h.svc.Wait()
	type route struct{ method, path, body string }
	sid, ms := h.sess.String(), mid.String()
	routes := []route{
		{"GET", "/chat/sessions?course_id=" + h.course.String(), ""},
		{"POST", "/chat/sessions", fmt.Sprintf(`{"course_id":"%s"}`, h.course)},
		{"DELETE", "/chat/sessions/" + sid, ""},
		{"POST", "/chat/sessions/" + sid + "/restore", ""},
		{"GET", "/chat/sessions/" + sid + "/messages", ""},
		{"POST", "/chat/sessions/" + sid + "/messages", `{"content":"hi"}`},
		{"GET", "/chat/messages/" + ms + "/stream", ""},
		{"POST", "/chat/messages/" + ms + "/cancel", ""},
		{"POST", "/chat/messages/" + ms + "/retry", ""},
		{"PUT", "/chat/messages/" + ms + "/feedback", `{"value":"HELPFUL"}`},
	}
	mk := func(role, global string) (uuid.UUID, auth.Role) {
		return h.user("X", global), auth.Role(role)
	}
	taID, taRole := mk("TA", "TA")
	h.enroll(h.course, taID, "TA", "ACTIVE")
	tcID, tcRole := mk("TEACHER", "TEACHER")
	h.enroll(h.course, tcID, "TEACHER", "ACTIVE")
	adID, adRole := mk("ADMIN", "ADMIN")
	pend := h.user("P", "STUDENT")
	h.enroll(h.course, pend, "STUDENT", "PENDING")
	out := h.user("O", "STUDENT")
	for name, a := range map[string]struct {
		id   uuid.UUID
		role auth.Role
	}{"TA": {taID, taRole}, "TEACHER": {tcID, tcRole}, "ADMIN": {adID, adRole}, "PENDING": {pend, auth.RoleStudent}, "ngoài lớp": {out, auth.RoleStudent}} {
		for _, rt := range routes {
			hdr := []string{}
			if rt.method == "POST" {
				hdr = []string{"Idempotency-Key", uuid.NewString()}
			}
			res := h.do(rt.method, rt.path, h.tok(a.id, a.role), rt.body, hdr...)
			_ = res.Body.Close()
			// sinh viên không có ghi danh ACTIVE gọi route theo id của người khác: 404 (không lộ tồn tại) hoặc 403; mọi vai khác và route theo lớp: 403
			idRoute := strings.Contains(rt.path, "/sessions/") || strings.Contains(rt.path, "/messages/")
			if a.role == auth.RoleStudent && idRoute {
				require.Contains(t, []int{403, 404}, res.StatusCode, "%s %s %s", name, rt.method, rt.path)
				continue
			}
			require.Equal(t, 403, res.StatusCode, "%s %s %s", name, rt.method, rt.path)
		}
	}
	// chưa đăng nhập
	for _, rt := range routes {
		res := h.do(rt.method, rt.path, "", rt.body)
		_ = res.Body.Close()
		require.Equal(t, 401, res.StatusCode, "%s %s", rt.method, rt.path)
	}
	// SV khác: 404 không lộ tồn tại
	for _, rt := range routes[2:] {
		if rt.method == "POST" && strings.HasSuffix(rt.path, "/messages") {
			rt.body = `{"content":"hi"}`
		}
		hdr := []string{}
		if rt.method == "POST" {
			hdr = []string{"Idempotency-Key", uuid.NewString()}
		}
		res := h.do(rt.method, rt.path, h.tok(h.other.UserID, auth.RoleStudent), rt.body, hdr...)
		_ = res.Body.Close()
		require.Equal(t, 404, res.StatusCode, "%s %s", rt.method, rt.path)
	}
}

// TestSessionsHTTP: tạo (Idempotency-Key) → danh sách → xoá → hoàn tác qua router thật.
func TestSessionsHTTP(t *testing.T) {
	t.Parallel()
	h := newHTTPRig(t)
	tok := h.tok(h.student.UserID, auth.RoleStudent)
	body := fmt.Sprintf(`{"course_id":"%s","title":"Ôn thi"}`, h.course)
	key := uuid.NewString()
	var ids [2]string
	for i := range ids {
		res := h.do(http.MethodPost, "/chat/sessions", tok, body, "Idempotency-Key", key)
		var s struct{ ID, Title string }
		require.NoError(t, json.NewDecoder(res.Body).Decode(&s))
		_ = res.Body.Close()
		require.Equal(t, 201, res.StatusCode)
		ids[i] = s.ID
	}
	require.Equal(t, ids[0], ids[1], "cùng Idempotency-Key = cùng phiên")
	res := h.do(http.MethodGet, "/chat/sessions?course_id="+h.course.String(), tok, "")
	var l struct{ Items []struct{ ID string } }
	require.NoError(t, json.NewDecoder(res.Body).Decode(&l))
	_ = res.Body.Close()
	require.Len(t, l.Items, 2)
	res = h.do(http.MethodDelete, "/chat/sessions/"+ids[0], tok, "")
	_ = res.Body.Close()
	require.Equal(t, 204, res.StatusCode)
	res = h.do(http.MethodPost, "/chat/sessions/"+ids[0]+"/restore", tok, "", "Idempotency-Key", uuid.NewString())
	_ = res.Body.Close()
	require.Equal(t, 200, res.StatusCode)
}

// TestChatContentOnlyInMessages — US-P3-01 AC9: nội dung chat chỉ ở chat_messages (quét mọi cột chữ của mọi bảng sau một lượt chat có canary).
// Ngoại lệ ghi trong proposals.md (D3): chat_sessions.title = 60 ký tự đầu của tin đầu (SRS 4.7.1 bước 7).
func TestChatContentOnlyInMessages(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	canary := "CANARY-" + strings.ToUpper(uuid.NewString()[:8])
	r.ag.say(time.Millisecond, "Trả ", "lời ", canary, " rồi.")
	_, err := r.pool.Exec(t.Context(), `update chat_sessions set title='đặt tên sẵn' where id=$1`, r.sess)
	require.NoError(t, err)
	_, _, err = r.send(r.student, r.sess, "Tin nhắn có "+canary)
	require.NoError(t, err)
	r.svc.Wait()
	rows, err := r.pool.Query(t.Context(), `select table_name, column_name from information_schema.columns where table_schema='public' and data_type in ('text','character varying','jsonb','json','ARRAY') and table_name <> 'chat_messages'`)
	require.NoError(t, err)
	type col struct{ t, c string }
	var cols []col
	for rows.Next() {
		var c col
		require.NoError(t, rows.Scan(&c.t, &c.c))
		cols = append(cols, c)
	}
	rows.Close()
	require.Greater(t, len(cols), 50)
	for _, c := range cols {
		var n int
		q := fmt.Sprintf(`select count(*) from %q where %q::text like $1`, c.t, c.c)
		require.NoError(t, r.pool.QueryRow(t.Context(), q, "%"+canary+"%").Scan(&n), "%s.%s", c.t, c.c)
		require.Zero(t, n, "canary rò ở %s.%s", c.t, c.c)
	}
	require.Equal(t, 2, r.count(`select count(*) from chat_messages where content like $1`, "%"+canary+"%"))
}
