package integration_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/httpapi"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/llm/llmrt"
	"github.com/edupilot/backend-go/internal/notify"
	"github.com/edupilot/backend-go/internal/observability"
	"github.com/edupilot/backend-go/internal/store"
)

type obsWorld struct {
	r                    *rig
	adm, gv, gv2, ta, sv session
	svUser               store.User
	course, otherCourse  uuid.UUID
	rt                   *llmrt.Runtime
}

func newObsWorld(t *testing.T) *obsWorld {
	t.Helper()
	w := &obsWorld{}
	w.r = newRigDeps(t, func(d *httpapi.Deps) { w.rt = d.LLM })
	r := w.r
	adm := r.addUser(uniq("adm"), store.UserRoleADMIN, store.UserStatusACTIVE)
	gv := r.addUser(uniq("gv"), store.UserRoleTEACHER, store.UserStatusACTIVE)
	gv2 := r.addUser(uniq("gv2"), store.UserRoleTEACHER, store.UserStatusACTIVE)
	ta := r.addUser(uniq("ta"), store.UserRoleTA, store.UserStatusACTIVE)
	w.svUser = r.addUser(uniq("sv"), store.UserRoleSTUDENT, store.UserStatusACTIVE)
	tag := uuid.NewString()[:6]
	w.course, w.otherCourse = r.course(adm.ID, "OBS1-"+tag), r.course(adm.ID, "OBS2-"+tag)
	r.enroll(w.course, gv.ID, "TEACHER", "ACTIVE")
	r.enroll(w.course, ta.ID, "TA", "ACTIVE")
	r.enroll(w.course, w.svUser.ID, "STUDENT", "ACTIVE")
	r.enroll(w.otherCourse, gv2.ID, "TEACHER", "ACTIVE")
	w.adm, w.gv, w.gv2, w.ta, w.sv = r.mustLogin(adm.Email), r.mustLogin(gv.Email), r.mustLogin(gv2.Email), r.mustLogin(ta.Email), r.mustLogin(w.svUser.Email)
	return w
}

// audit chèn một dòng llm_audit (kèm payload nếu có) và trả id.
func (w *obsWorld) audit(at time.Time, course *uuid.UUID, status string, latency, pii int, prompt string) uuid.UUID {
	w.r.t.Helper()
	var id uuid.UUID
	require.NoError(w.r.t, w.r.pool.QueryRow(w.r.t.Context(), `insert into llm_audit (task, lane, provider, model, latency_ms, status, pii_masked_count, course_id, trace_id, cost_est, created_at)
		values ('CHAT', 'INTERACTIVE', 'fake', 'fake-chat', $1, $2, $3, $4, $5, 2, $6) returning id`, latency, status, pii, course, uuid.NewString(), at).Scan(&id))
	if prompt != "" {
		_, err := w.r.pool.Exec(w.r.t.Context(), `insert into llm_request_payloads (audit_id, course_id, masked_prompt, tools, created_at) values ($1, $2, $3, '["attendance"]', $4)`, id, course, prompt, at)
		require.NoError(w.r.t, err)
	}
	return id
}

func (w *obsWorld) open(s session, id uuid.UUID, reason string) resp {
	return w.r.do(req{path: "/admin/observability/requests/" + id.String() + "/open", bearer: s.access, body: map[string]any{"reason": reason}})
}

func (w *obsWorld) auditCount(id uuid.UUID) (n int) {
	require := require.New(w.r.t)
	require.NoError(w.r.pool.QueryRow(w.r.t.Context(), `select count(*) from audit_log where entity = 'llm_request' and entity_id = $1 and action = 'VIEW_DETAIL'`, id.String()).Scan(&n))
	return n
}

// TestObservabilityRBAC — SRS §2: ma trận 4 vai + không token × bốn endpoint.
func TestObservabilityRBAC(t *testing.T) {
	w := newObsWorld(t)
	id := w.audit(time.Now(), &w.course, "ok", 100, 0, "[user]\nxin chào")
	type ep struct {
		method, path string
		body         any
	}
	admin := []ep{
		{http.MethodGet, "/admin/observability/summary", nil},
		{http.MethodGet, "/admin/observability/requests", nil},
		{http.MethodPost, "/admin/observability/requests/" + id.String() + "/open", map[string]any{"reason": "kiểm tra lỗi phản hồi"}},
	}
	for _, e := range admin {
		for who, want := range map[string]int{"adm": 200, "gv": 403, "ta": 403, "sv": 403} {
			s := map[string]session{"adm": w.adm, "gv": w.gv, "ta": w.ta, "sv": w.sv}[who]
			res := w.r.do(req{method: e.method, path: e.path, bearer: s.access, body: e.body})
			require.Equalf(t, want, res.code, "%s %s %s: %s", e.method, e.path, who, res.body)
		}
		require.Equal(t, http.StatusUnauthorized, w.r.do(req{method: e.method, path: e.path, body: e.body}).code, "không token: "+e.path)
	}
	require.Equal(t, 1+0, w.auditCount(id), "chỉ lần mở của ADMIN được ghi; 403 / 401 không ghi audit_log")

	cs := "/courses/%s/observability/summary"
	for _, tc := range []struct {
		who    string
		s      session
		course uuid.UUID
		want   int
	}{
		{"adm lớp 1", w.adm, w.course, 200}, {"adm lớp 2", w.adm, w.otherCourse, 200},
		{"gv lớp mình", w.gv, w.course, 200}, {"gv lớp khác", w.gv, w.otherCourse, 403}, {"gv2 lớp 1", w.gv2, w.course, 403},
		{"ta", w.ta, w.course, 403}, {"sv", w.sv, w.course, 403},
	} {
		res := w.r.do(req{method: http.MethodGet, path: fmt.Sprintf(cs, tc.course), bearer: tc.s.access})
		require.Equalf(t, tc.want, res.code, "%s: %s", tc.who, res.body)
	}
	// không còn đường GET đọc nội dung (P-11-1b)
	require.Contains(t, []int{404, 405}, w.r.do(req{method: http.MethodGet, path: "/admin/observability/requests/" + id.String(), bearer: w.adm.access}).code)
}

// TestObservabilityAuditEveryOpen — AC2–AC4, AC8, AC9: mỗi lần mở một dòng audit_log (đúng actor, lý do); lỗi không ghi / hết hạn vẫn ghi.
func TestObservabilityAuditEveryOpen(t *testing.T) {
	w := newObsWorld(t)
	id := w.audit(time.Now(), &w.course, "ok", 120, 1, "[system]\nBạn là trợ giảng.\n\n[user]\nEm là [STUDENT_1]")
	reason := "điều tra yêu cầu chậm của lớp"
	for i := 1; i <= 2; i++ {
		res := w.open(w.adm, id, reason)
		require.Equal(t, http.StatusOK, res.code, string(res.body))
		require.Equal(t, "no-store", res.hdr.Get("Cache-Control"))
		require.Contains(t, res.json()["masked_prompt"], "[STUDENT_1]")
		require.Equal(t, []any{"attendance"}, res.json()["tools"])
		require.Equal(t, i, w.auditCount(id), "mỗi lần mở một dòng, không gộp")
	}
	var actor, after string
	require.NoError(t, w.r.pool.QueryRow(t.Context(), `select actor_id::text, after->>'reason' from audit_log where entity = 'llm_request' and entity_id = $1 limit 1`, id.String()).Scan(&actor, &after))
	require.Equal(t, reason, after)
	var admID string
	require.NoError(t, w.r.pool.QueryRow(t.Context(), `select id::text from users where email = (select email from users where role = 'ADMIN' limit 1)`).Scan(&admID))
	require.Equal(t, admID, actor)

	for _, bad := range []string{"", "ngắn", strings.Repeat("a", 201)} {
		res := w.open(w.adm, id, bad)
		require.Equal(t, http.StatusUnprocessableEntity, res.code, string(res.body))
		require.Equal(t, "REASON_REQUIRED", res.json()["code"])
	}
	require.Equal(t, 2, w.auditCount(id), "thiếu lý do không ghi audit_log và không trả nội dung")

	none := uuid.New()
	require.Equal(t, http.StatusNotFound, w.open(w.adm, none, reason).code)
	require.Zero(t, w.auditCount(none))

	old := w.audit(time.Now().AddDate(0, 0, -40), &w.course, "ok", 50, 0, "")
	res := w.open(w.adm, old, reason)
	require.Equal(t, http.StatusGone, res.code)
	require.Equal(t, "PAYLOAD_EXPIRED", res.json()["code"])
	require.Equal(t, 1, w.auditCount(old), "hết hạn vẫn ghi lần mở")
}

// TestObservabilityDetailNoPII — AC5: đi qua gateway LLM thật (che thật, provider giả): chi tiết mở ra chỉ có placeholder, không tên / MSSV / email.
func TestObservabilityDetailNoPII(t *testing.T) {
	w := newObsWorld(t)
	_, err := w.r.pool.Exec(t.Context(), `update users set full_name = 'Phạm Quốc Khánh' where id = $1`, w.svUser.ID)
	require.NoError(t, err)
	ctx := llm.WithIdentity(t.Context(), llm.Identity{UserID: &w.svUser.ID, CourseID: &w.course})
	_, err = w.rt.Gateway.Chat(ctx, llm.Request{Task: llm.TaskChat, Tools: []string{"attendance"}, Messages: []llm.Message{
		{Role: "system", Content: "Bạn là trợ giảng."},
		{Role: "user", Content: "Em là Phạm Quốc Khánh, MSSV 20225511, mail khanh.pq@sv.edu.vn, xin điểm danh"},
	}})
	require.NoError(t, err)
	w.rt.Gateway.FlushAudit(t.Context())

	var id uuid.UUID
	require.NoError(t, w.r.pool.QueryRow(t.Context(), `select audit_id from llm_request_payloads where course_id = $1 order by created_at desc limit 1`, w.course).Scan(&id))
	res := w.open(w.adm, id, "kiểm tra việc che thông tin")
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	body := string(res.body)
	for _, leak := range []string{"Phạm Quốc Khánh", "20225511", "khanh.pq@sv.edu.vn"} {
		require.NotContains(t, body, leak)
	}
	require.Contains(t, body, "xin điểm danh")
	require.Contains(t, res.json()["masked_prompt"], "[")
}

// TestObservabilityRequestsFilter — AC4: lọc, con trỏ, không cột nội dung.
func TestObservabilityRequestsFilter(t *testing.T) {
	w := newObsWorld(t)
	now := time.Now()
	for i := 0; i < 5; i++ {
		w.audit(now.Add(-time.Duration(i)*time.Minute), &w.course, "ok", 100, 0, "")
	}
	w.audit(now.Add(-10*time.Minute), &w.course, "rate_limited", 900, 0, "")
	w.audit(now.Add(-11*time.Minute), &w.otherCourse, "ok", 100, 2, "")
	get := func(q string) (items []any, next any) {
		res := w.r.do(req{method: http.MethodGet, path: "/admin/observability/requests?" + q, bearer: w.adm.access})
		require.Equal(t, http.StatusOK, res.code, string(res.body))
		require.Equal(t, "no-store", res.hdr.Get("Cache-Control"))
		m := res.json()
		return m["items"].([]any), m["next_cursor"]
	}
	all, _ := get("filter=all&limit=100")
	require.Len(t, all, 7)
	for _, it := range all {
		for k := range it.(map[string]any) {
			require.NotContains(t, []string{"masked_prompt", "prompt", "content"}, k)
		}
	}
	prob, _ := get("filter=problem")
	require.Len(t, prob, 1)
	priv, _ := get("filter=privacy")
	require.Len(t, priv, 1)
	byCourse, _ := get("course_id=" + w.otherCourse.String())
	require.Len(t, byCourse, 1)
	page1, next := get("limit=3")
	require.Len(t, page1, 3)
	require.NotNil(t, next)
	page2, _ := get("limit=100&cursor=" + next.(string))
	require.Len(t, page2, 4)
	require.Equal(t, http.StatusUnprocessableEntity, w.r.do(req{method: http.MethodGet, path: "/admin/observability/requests?filter=zzz", bearer: w.adm.access}).code)
}

// TestAlertDedup — US-P10-03: ≥ 20 yêu cầu / giờ và lỗi > 5 % → đúng MỘT thông báo cho mọi ADMIN; chạy lại không thêm; qua khoảng khử trùng thì báo mới; ít tải không báo; chỉ ADMIN nhận.
func TestAlertDedup(t *testing.T) {
	w := newObsWorld(t)
	svc := &observability.Service{Pool: w.r.pool}
	a := &observability.Alerter{Svc: svc, Notify: notify.New(w.r.pool, nil, nil), Dedupe: 6 * time.Hour}
	count := func(typ string) (n int) {
		require.NoError(t, w.r.pool.QueryRow(t.Context(), `select count(*) from notifications where type = $1`, typ).Scan(&n))
		return n
	}
	now := time.Now()
	for i := 0; i < 15; i++ { // < 20 yêu cầu, lỗi 100 %: không báo
		w.audit(now.Add(-time.Minute), nil, "error", 10, 0, "")
	}
	n, err := a.Tick(t.Context())
	require.NoError(t, err)
	require.Zero(t, n)
	require.Zero(t, count(notify.TypeLLMErrorRate))

	// 30 yêu cầu trong giờ, 3 lỗi (10 %) sau khi dọn dòng cũ
	_, err = w.r.pool.Exec(t.Context(), `delete from llm_audit`)
	require.NoError(t, err)
	for i := 0; i < 30; i++ {
		st := "ok"
		if i < 3 {
			st = "error"
		}
		w.audit(now.Add(-time.Duration(i)*time.Minute), nil, st, 10, 0, "")
	}
	n, err = a.Tick(t.Context())
	require.NoError(t, err)
	require.Equal(t, 1, n)
	require.Equal(t, 1, count(notify.TypeLLMErrorRate), "một ADMIN → một thông báo")
	for i := 0; i < 3; i++ {
		_, err = a.Tick(t.Context())
		require.NoError(t, err)
	}
	require.Equal(t, 1, count(notify.TypeLLMErrorRate), "chạy lại cùng cửa sổ không thêm")

	_, err = w.r.pool.Exec(t.Context(), `update notifications set created_at = created_at - interval '7 hours' where type = $1`, notify.TypeLLMErrorRate)
	require.NoError(t, err)
	_, err = a.Tick(t.Context())
	require.NoError(t, err)
	require.Equal(t, 2, count(notify.TypeLLMErrorRate), "tái phát sau 6 giờ → thông báo mới")

	_, err = w.r.pool.Exec(t.Context(), `insert into llm_budgets (scope, daily_limit) values ('system', 1)`)
	require.NoError(t, err)
	_, err = a.Tick(t.Context())
	require.NoError(t, err)
	require.Equal(t, 1, count(notify.TypeLLMBudgetReached))
	var n2 int
	require.NoError(t, w.r.pool.QueryRow(t.Context(), `select count(*) from notifications n join users u on u.id = n.user_id where n.type in ('LLM_ERROR_RATE', 'LLM_BUDGET_REACHED') and u.role <> 'ADMIN'`).Scan(&n2))
	require.Zero(t, n2, "chỉ ADMIN nhận cảnh báo")
}

// TestPayloadPurge — US-P10-02 AC8: job dọn xoá bản prompt > 30 ngày, llm_audit giữ nguyên.
func TestPayloadPurge(t *testing.T) {
	w := newObsWorld(t)
	old := w.audit(time.Now().AddDate(0, 0, -31), &w.course, "ok", 10, 0, "[user]\ncũ")
	fresh := w.audit(time.Now().AddDate(0, 0, -2), &w.course, "ok", 10, 0, "[user]\nmới")
	n, err := (&observability.Service{Pool: w.r.pool}).PurgePayloads(t.Context(), 30)
	require.NoError(t, err)
	require.EqualValues(t, 1, n)
	var cnt int
	require.NoError(t, w.r.pool.QueryRow(t.Context(), `select count(*) from llm_request_payloads where audit_id = $1`, old).Scan(&cnt))
	require.Zero(t, cnt)
	require.NoError(t, w.r.pool.QueryRow(t.Context(), `select count(*) from llm_request_payloads where audit_id = $1`, fresh).Scan(&cnt))
	require.Equal(t, 1, cnt)
	require.NoError(t, w.r.pool.QueryRow(t.Context(), `select count(*) from llm_audit where id = $1`, old).Scan(&cnt))
	require.Equal(t, 1, cnt, "llm_audit giữ nguyên")
}
