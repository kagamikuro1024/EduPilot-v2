package thread_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/httpapi"
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

// newHTTPRig dựng ROUTER THẬT với thread.Service của rig (middleware auth, CourseAccessGuard, Idempotency đều chạy).
func newHTTPRig(t *testing.T) *httpRig {
	t.Helper()
	r := newRig(t)
	env := map[string]string{
		"DATABASE_URL": testutil.MigratedPostgresURL(t), "REDIS_URL": testutil.RedisURL(t), "JWT_SECRET_KEY": jwtSecret,
		"BLOB_ENDPOINT": "127.0.0.1:9", "BLOB_BUCKET": "x", "BLOB_ACCESS_KEY": "x", "BLOB_SECRET_KEY": "x",
		"APP_ENCRYPTION_KEY": "ZWR1cGlsb3QtZGV2LWVuY3J5cHRpb24ta2V5LTMyYnk=", "APP_ENV": "test", "BCRYPT_COST": "4",
		"RATE_LIMIT_IP_PER_MIN": "1000000", "RATE_LIMIT_USER_PER_MIN": "1000000", "REQUEST_TIMEOUT": "5s",
	}
	cfg, err := config.Load(func(k string) string { return env[k] }, config.Gateway)
	require.NoError(t, err)
	h := httpapi.NewRouter(httpapi.Deps{Cfg: cfg, Log: r.svc.Log, DB: r.pool, Redis: r.rdb, Clock: clock.Real{}, State: httpapi.NewState(), Thread: r.svc})
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return &httpRig{rig: r, srv: srv, iss: auth.NewIssuer(jwtSecret, time.Hour, clock.Real{})}
}

func (h *httpRig) do(id uuid.UUID, role auth.Role, method, path, body string, hdr ...string) *http.Response {
	h.t.Helper()
	req, err := http.NewRequestWithContext(h.t.Context(), method, h.srv.URL+"/api/v1"+path, strings.NewReader(body))
	require.NoError(h.t, err)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if id != uuid.Nil {
		tok, err := h.iss.Issue(id.String(), role, "x@example.test")
		require.NoError(h.t, err)
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	for i := 0; i+1 < len(hdr); i += 2 {
		req.Header.Set(hdr[i], hdr[i+1])
	}
	res, err := http.DefaultClient.Do(req)
	require.NoError(h.t, err)
	t := h.t
	t.Cleanup(func() { _ = res.Body.Close() })
	return res
}

// TestThreadsMatrix — AC16: STUDENT / TA / TEACHER của lớp đọc và đăng được; ADMIN 403; người ngoài lớp 403; chưa đăng nhập 401; STUDENT gọi verify / correct / reject → 403, TA / TEACHER → 200.
func TestThreadsMatrix(t *testing.T) {
	t.Parallel()
	h := newHTTPRig(t)
	id := h.answered()
	pid := h.aiPost(id)
	admin := h.user("Quản trị", "ADMIN", "")
	outsider := h.user("Người lạ", "STUDENT", "")
	base := "/courses/" + h.course.String()
	for _, c := range []struct {
		name string
		who  uuid.UUID
		role auth.Role
		want int
	}{
		{"student", h.other.UserID, auth.RoleStudent, 200}, {"ta", h.ta.UserID, auth.RoleTA, 200}, {"teacher", h.teacher.UserID, auth.RoleTeacher, 200},
		{"admin", admin, auth.RoleAdmin, 403}, {"outsider", outsider, auth.RoleStudent, 403}, {"anonymous", uuid.Nil, "", 401},
	} {
		require.Equal(t, c.want, h.do(c.who, c.role, http.MethodGet, base+"/threads", "").StatusCode, c.name+" list")
		require.Equal(t, c.want, h.do(c.who, c.role, http.MethodGet, base+"/threads/"+id.String(), "").StatusCode, c.name+" get")
	}
	require.Equal(t, 403, h.do(h.other.UserID, auth.RoleStudent, http.MethodPost, base+"/posts/"+pid.String()+"/verify", "").StatusCode)
	require.Equal(t, 403, h.do(h.sv.UserID, auth.RoleStudent, http.MethodPut, base+"/posts/"+pid.String()+"/correct", `{"body":"x","version":1}`).StatusCode)
	require.Equal(t, 403, h.do(h.sv.UserID, auth.RoleStudent, http.MethodPost, base+"/posts/"+pid.String()+"/reject", "").StatusCode)
	require.Equal(t, 200, h.do(h.ta.UserID, auth.RoleTA, http.MethodPost, base+"/posts/"+pid.String()+"/verify", "").StatusCode)
	// lớp khác: thread của lớp này không đọc được qua lớp khác (404 / 403, không bao giờ 200)
	require.NotEqual(t, 200, h.do(h.other.UserID, auth.RoleStudent, http.MethodGet, "/courses/"+uuid.NewString()+"/threads/"+id.String(), "").StatusCode)
}

// TestCreateIdempotent20Parallel — AC17: 20 yêu cầu song song cùng Idempotency-Key + cùng thân → đúng MỘT thread, một việc AI, một outbox.
func TestCreateIdempotent20Parallel(t *testing.T) {
	t.Parallel()
	h := newHTTPRig(t)
	key := uuid.NewString()
	body := `{"title":"Hỏi","body":"` + clean + `"}`
	var wg sync.WaitGroup
	codes := make([]int, 20)
	for i := range codes {
		wg.Add(1)
		go func() {
			defer wg.Done()
			codes[i] = h.do(h.sv.UserID, auth.RoleStudent, http.MethodPost, "/courses/"+h.course.String()+"/threads", body, "Idempotency-Key", key).StatusCode
		}()
	}
	wg.Wait()
	for _, c := range codes {
		require.Contains(t, []int{201, 409}, c, fmt.Sprint(codes)) // 409 = đang xử lý (IDEMPOTENCY_IN_PROGRESS); không có mã nào khác
	}
	require.Contains(t, codes, 201)
	require.Equal(t, 1, h.count(`select count(*) from forum_threads`))
	require.Equal(t, 1, h.count(`select count(*) from jobs where kind='thread.answer'`))
	require.Equal(t, 1, h.count(`select count(*) from outbox where topic='thread.created'`))
}
