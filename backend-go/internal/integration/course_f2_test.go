package integration_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/course"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/store"
	"github.com/edupilot/backend-go/internal/testutil"
)

// worker chạy relay + consumer THẬT (Streams riêng theo test) với handler `course.assigned`, tới hết test.
// hits đếm số lần handler được gọi theo outbox id để kiểm giao lại.
func (r *rig) worker() *sync.Map {
	r.t.Helper()
	p := testutil.TestPrefix(r.t)
	names := outbox.Streams{Dispatch: p + ".dispatch", Dead: p + ".dispatch.dead", Consumer: p + "-c1"}
	cfg := r.cfg
	cfg.InstanceID = p
	cfg.OutboxPollInterval = 20 * time.Millisecond
	cfg.OutboxRetryBackoff = []time.Duration{20 * time.Millisecond, 20 * time.Millisecond, 20 * time.Millisecond}
	od := outbox.Deps{Pool: r.pool, Redis: r.rdb, Log: slog.New(slog.NewTextHandler(io.Discard, nil)), Clock: clock.Real{}, Cfg: cfg, Streams: names}

	hits := &sync.Map{}
	n := &course.Notifier{Pool: r.pool, AppPublicURL: "https://localhost"}
	reg := outbox.NewRegistry()
	reg.Register(course.TopicAssigned, func(ctx context.Context, m outbox.Message) error {
		hits.Store(m.ID, true)
		return n.HandleAssigned(ctx, m)
	})
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	for _, task := range []interface{ Run(context.Context) error }{outbox.NewRelay(od), outbox.NewConsumer(od, reg)} {
		wg.Add(1)
		go func() { defer wg.Done(); _ = task.Run(ctx) }()
	}
	r.t.Cleanup(func() {
		cancel()
		wg.Wait()
		_ = r.rdb.Del(context.Background(), names.Dispatch, names.Dead).Err()
	})
	return hits
}

func (r *rig) notifications(s session) []any {
	r.t.Helper()
	res := r.do(req{method: http.MethodGet, path: "/notifications", bearer: s.access})
	require.Equal(r.t, http.StatusOK, res.code, string(res.body))
	return res.json()["items"].([]any)
}

// waitFor đợi tối đa d tới khi f() true (poll 25 ms).
func waitFor(t *testing.T, d time.Duration, what string, f func() bool) time.Duration {
	t.Helper()
	start := time.Now()
	for time.Since(start) < d {
		if f() {
			return time.Since(start)
		}
		time.Sleep(25 * time.Millisecond)
	}
	require.Failf(t, "quá hạn", "%s sau %s", what, d)
	return 0
}

type f2 struct {
	adm, gv, ta, sv store.User
	A, G, T, S      session
}

func (r *rig) f2() f2 {
	r.t.Helper()
	x := f2{
		adm: r.addUser(uniq("adm"), store.UserRoleADMIN, store.UserStatusACTIVE),
		gv:  r.addUser(uniq("gv"), store.UserRoleTEACHER, store.UserStatusACTIVE),
		ta:  r.addUser(uniq("ta"), store.UserRoleTA, store.UserStatusACTIVE),
		sv:  r.addUser(uniq("sv"), store.UserRoleSTUDENT, store.UserStatusACTIVE),
	}
	x.A, x.G, x.T, x.S = r.mustLogin(x.adm.Email), r.mustLogin(x.gv.Email), r.mustLogin(x.ta.Email), r.mustLogin(x.sv.Email)
	return x
}

func (r *rig) openCourse(s session, body map[string]any) string {
	r.t.Helper()
	b := map[string]any{"subject_code": "INT1006", "class_code": "F2-" + uuid.NewString()[:8], "name": "An ninh mạng", "semester": "2026-2027-HK1"}
	for k, v := range body {
		b[k] = v
	}
	res := r.do(req{method: http.MethodPost, path: "/admin/courses", bearer: s.access, body: b, hdr: map[string]string{"Idempotency-Key": "k-" + uuid.NewString()}})
	require.Equal(r.t, http.StatusCreated, res.code, string(res.body))
	return res.json()["id"].(string)
}

// AC5: worker thật xử lý `course.assigned` ⇒ đúng một thông báo cho mỗi người, trong ≤ 60 s (thực tế ≤ 2 s).
func TestAssignTeacherNotifies(t *testing.T) {
	r := newRig(t)
	r.worker()
	x := r.f2()
	id := r.openCourse(x.A, map[string]any{"teacher_id": x.gv.ID, "ta_ids": []string{x.ta.ID.String()}})
	took := waitFor(t, 60*time.Second, "thông báo của giảng viên", func() bool { return len(r.notifications(x.G)) == 1 })
	require.Less(t, took, 2*time.Second, "thực tế ≤ 2 s")
	waitFor(t, 5*time.Second, "thông báo của trợ giảng", func() bool { return len(r.notifications(x.T)) == 1 })
	n := r.notifications(x.G)[0].(map[string]any)
	require.Contains(t, n["title"], "Bạn được phân công lớp An ninh mạng – F2-")
	require.Equal(t, "/class/settings?course="+id, n["link"])
	require.Equal(t, 0, len(r.notifications(x.S)), "sinh viên không nhận gì")
	require.Equal(t, 0, len(r.notifications(x.A)))
}

// AC5: tin được giao lại (xử lý lại cùng outbox id) không tạo dòng thứ hai.
func TestAssignNotifyOnce(t *testing.T) {
	r := newRig(t)
	hits := r.worker()
	x := r.f2()
	id := r.openCourse(x.A, map[string]any{"teacher_id": x.gv.ID})
	waitFor(t, 10*time.Second, "thông báo", func() bool { return len(r.notifications(x.G)) == 1 })
	var mid uuid.UUID
	var payload []byte
	require.NoError(t, r.pool.QueryRow(t.Context(), `select id, payload from outbox where topic = 'course.assigned' and payload->>'course_id' = $1`, id).Scan(&mid, &payload))
	_, ok := hits.Load(mid)
	require.True(t, ok)
	n := &course.Notifier{Pool: r.pool, AppPublicURL: "https://localhost"}
	for range 3 {
		require.NoError(t, n.HandleAssigned(t.Context(), outbox.Message{ID: mid, Topic: course.TopicAssigned, Payload: payload}))
	}
	require.Len(t, r.notifications(x.G), 1)
}

// AC13: DB trống ⇒ Admin mở lớp + gán giảng viên ⇒ giảng viên thấy chuông + mã tham gia ≤ 60 s; Admin không có đường thêm sinh viên.
func TestOpenCourseAssignThenTeacherSeesCode(t *testing.T) {
	r := newRig(t)
	r.worker()
	x := r.f2()
	id := r.openCourse(x.A, nil)
	require.Empty(t, r.notifications(x.G), "chưa gán ⇒ chưa có gì")
	res := r.do(req{method: http.MethodPost, path: "/admin/courses/" + id + "/assign", bearer: x.A.access, body: map[string]any{"teacher_id": x.gv.ID}})
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	waitFor(t, 60*time.Second, "thông báo nhận lớp", func() bool { return len(r.notifications(x.G)) == 1 })
	var code string
	require.NoError(t, r.pool.QueryRow(t.Context(), `select join_code from courses where id = $1`, id).Scan(&code))
	n := r.notifications(x.G)[0].(map[string]any)
	require.Contains(t, n["body"], "Mã tham gia: "+code+".")
	require.Contains(t, n["body"], "https://localhost/join/"+code)
	// chuông: đọc thông báo ⇒ chưa đọc về 0
	unread := r.do(req{method: http.MethodGet, path: "/notifications", bearer: x.G.access}).json()["unread_count"]
	require.EqualValues(t, 1, unread)
	require.Equal(t, http.StatusNoContent, r.do(req{method: http.MethodPost, path: "/notifications/" + n["id"].(string) + "/read", bearer: x.G.access}).code)
	require.EqualValues(t, 0, r.do(req{method: http.MethodGet, path: "/notifications", bearer: x.G.access}).json()["unread_count"])
	// Admin không có đường thêm sinh viên vào lớp: không route ghi danh nào dưới /admin/courses
	for _, p := range []string{"/students", "/members", "/enroll"} {
		for _, m := range []string{http.MethodPost, http.MethodPut} {
			res := r.do(req{method: m, path: "/admin/courses/" + id + p, bearer: x.A.access, body: map[string]any{"user_id": x.sv.ID}, hdr: map[string]string{"Idempotency-Key": "k-" + uuid.NewString()}})
			require.Contains(t, []int{http.StatusNotFound, http.StatusMethodNotAllowed}, res.code, m+" "+p)
		}
	}
	require.Equal(t, 0, func() int {
		var n int
		require.NoError(t, r.pool.QueryRow(t.Context(), `select count(*) from enrollments where course_id = $1 and role_in_course = 'STUDENT'`, id).Scan(&n))
		return n
	}())
}
