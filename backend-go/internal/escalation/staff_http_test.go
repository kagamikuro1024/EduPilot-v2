package escalation_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/course"
	"github.com/edupilot/backend-go/internal/escalation"
	"github.com/edupilot/backend-go/internal/httpapi"
	"github.com/edupilot/backend-go/internal/httpapi/escalationhttp"
	"github.com/edupilot/backend-go/internal/httpapi/mailhttp"
	"github.com/edupilot/backend-go/internal/notify"
	"github.com/edupilot/backend-go/internal/platform/config"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/testutil"
)

// api dựng route nhân viên THẬT (guard lớp thật, Idempotency thật) sau một middleware đặt danh tính theo header X-Sub / X-Role (thay cho JWT).
func (r *rig) api(t *testing.T) http.Handler {
	t.Helper()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	rc, err := appredis.New(t.Context(), testutil.RedisURL(t))
	require.NoError(t, err)
	t.Cleanup(func() { _ = rc.Close() })
	d := httpapi.Deps{DB: r.pool, Redis: rc, Log: log, Cfg: config.Config{MaxBodyBytes: 1 << 20}}
	guard := func(m auth.GuardMode) func(http.Handler) http.Handler {
		return auth.CourseAccessGuard(course.Resolver{Pool: r.pool}, m)
	}
	h := &escalationhttp.StaffHandler{Svc: r.svc, Notify: notify.New(r.pool, &capture{}, log), Guard: guard, Idem: httpapi.RequireIdempotencyKey(d), Log: log}
	mux := chi.NewRouter()
	mux.Route("/api/v1", func(rt chi.Router) {
		rt.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
				if sub := q.Header.Get("X-Sub"); sub != "" {
					q = q.WithContext(auth.WithPrincipal(q.Context(), auth.Principal{Sub: sub, Role: auth.Role(q.Header.Get("X-Role"))}))
				}
				next.ServeHTTP(w, q)
			})
		})
		h.Mount(rt)
		(&mailhttp.Handler{Pool: r.pool, Log: log}).Mount(rt)
	})
	return mux
}

type who struct {
	id   uuid.UUID
	role string
}

func (r *rig) whoAll() map[string]who {
	return map[string]who{"gv": {r.teacher, "TEACHER"}, "ta": {r.ta, "TA"}, "sv": {r.a.UserID, "STUDENT"}, "admin": {r.admin, "ADMIN"}}
}

func call(h http.Handler, w who, method, path, body string, hdr ...string) (int, string) {
	req := httptest.NewRequestWithContext(context.Background(), method, "/api/v1"+path, strings.NewReader(body))
	if w.role != "" {
		req.Header.Set("X-Sub", w.id.String())
		req.Header.Set("X-Role", w.role)
	}
	for i := 0; i+1 < len(hdr); i += 2 {
		req.Header.Set(hdr[i], hdr[i+1])
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code, rec.Body.String()
}

func TestTicketRoutesRBAC(t *testing.T) {
	r := newRig(t, escalation.Config{})
	h := r.api(t)
	id := r.ticket("Hỏi")
	base := "/courses/" + r.course.String() + "/tickets"
	idem := func() []string { return []string{"Idempotency-Key", "ep-" + uuid.NewString()} }
	routes := []struct {
		name, method, path, body string
		hdr                      []string
	}{
		{"list", "GET", base, "", nil},
		{"get", "GET", base + "/" + id.String(), "", nil},
		{"claim", "POST", base + "/" + id.String() + "/claim", "", nil},
		{"release", "POST", base + "/" + id.String() + "/release", "", nil},
		{"answer", "POST", base + "/" + id.String() + "/answer", `{"body":"x"}`, idem()},
		{"dismiss", "POST", base + "/" + id.String() + "/dismiss", `{"reason":"DUPLICATE"}`, nil},
		{"me", "GET", "/me/tickets", "", nil},
	}
	for _, rt := range routes {
		for _, who := range []string{"sv", "admin"} {
			code, _ := call(h, r.whoAll()[who], rt.method, rt.path, rt.body, rt.hdr...)
			require.Equal(t, 403, code, rt.name+" "+who)
		}
		code, _ := call(h, who{}, rt.method, rt.path, rt.body, rt.hdr...)
		require.Equal(t, 401, code, rt.name+" không đăng nhập")
	}
	// TA có đủ quyền như giảng viên (nhận → trả lời), không cần là người nhận để xem chi tiết
	ta := r.whoAll()["ta"]
	code, _ := call(h, ta, "GET", base+"/"+id.String(), "")
	require.Equal(t, 200, code)
	code, _ = call(h, ta, "POST", base+"/"+id.String()+"/claim", "")
	require.Equal(t, 200, code)
	code, _ = call(h, r.whoAll()["gv"], "GET", base+"/"+id.String(), "")
	require.Equal(t, 200, code, "chi tiết không phụ thuộc người nhận")
	code, _ = call(h, ta, "POST", base+"/"+id.String()+"/answer", `{"body":"Em xem bài 3."}`, idem()...)
	require.Equal(t, 200, code)
	code, _ = call(h, ta, "POST", base+"/"+id.String()+"/answer", `{"body":"x"}`)
	require.Equal(t, 422, code, "answer bắt buộc Idempotency-Key")
}

func TestMyTicketsAdminForbidden(t *testing.T) {
	r := newRig(t, escalation.Config{})
	h := r.api(t)
	for _, w := range []string{"admin", "sv"} {
		code, _ := call(h, r.whoAll()[w], "GET", "/me/tickets", "")
		require.Equal(t, 403, code, w)
	}
	code, body := call(h, r.whoAll()["gv"], "GET", "/me/tickets?status=ALL", "")
	require.Equal(t, 200, code, body)
}

func TestTicketDetailAuthMatrix(t *testing.T) {
	r := newRig(t, escalation.Config{})
	h := r.api(t)
	id := r.ticket("Hỏi")
	url := "/courses/" + r.course.String() + "/tickets/" + id.String()
	for who, want := range map[string]int{"gv": 200, "ta": 200, "sv": 403, "admin": 403} {
		code, _ := call(h, r.whoAll()[who], "GET", url, "")
		require.Equal(t, want, code, who)
	}
	// ticket của lớp này nhưng gọi qua lớp khác: giảng viên có mặt ở lớp kia vẫn 404
	code, _ := call(h, r.whoAll()["gv"], "GET", "/courses/"+r.course2.String()+"/tickets/"+id.String(), "")
	require.Equal(t, 404, code)
	// nhân viên chỉ ở lớp 2 gọi lớp 1: không có quyền
	other := r.user("TA khác", "TA")
	r.enroll(r.course2, other, "TA", "ACTIVE")
	code, _ = call(h, who{other, "TA"}, "GET", url, "")
	require.Equal(t, 403, code)
	require.Equal(t, 0, r.count(`select count(*) from audit_log where entity = 'escalation_ticket' and action <> 'escalation_ticket.created'`), "mở chi tiết không ghi audit")
}

func TestAnswerIdempotentKey(t *testing.T) {
	r := newRig(t, escalation.Config{})
	h := r.api(t)
	id := r.ticket("Hỏi")
	url := "/courses/" + r.course.String() + "/tickets/" + id.String() + "/answer"
	gv := r.whoAll()["gv"]
	key := "ep-" + uuid.NewString()
	c1, b1 := call(h, gv, "POST", url, `{"body":"Trả lời một lần"}`, "Idempotency-Key", key)
	c2, b2 := call(h, gv, "POST", url, `{"body":"Trả lời một lần"}`, "Idempotency-Key", key)
	require.Equal(t, [2]int{200, 200}, [2]int{c1, c2})
	var m1, m2 struct {
		Message struct{ ID string }
	}
	require.NoError(t, json.Unmarshal([]byte(b1), &m1))
	require.NoError(t, json.Unmarshal([]byte(b2), &m2))
	require.Equal(t, m1.Message.ID, m2.Message.ID, "phát lại đúng phản hồi cũ")
	require.Equal(t, 1, r.count(`select count(*) from chat_messages where intent = 'STAFF_ANSWER'`))
	require.Equal(t, 1, r.count(`select count(*) from outbox where topic = 'escalation.answered'`))
	c3, _ := call(h, gv, "POST", url, `{"body":"Nội dung khác"}`, "Idempotency-Key", key)
	require.Equal(t, 422, c3, "cùng khoá, thân khác")
}
