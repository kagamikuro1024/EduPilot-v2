package integration_test

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/course"
	"github.com/edupilot/backend-go/internal/escalation"
	"github.com/edupilot/backend-go/internal/httpapi"
	"github.com/edupilot/backend-go/internal/httpapi/sse"
	"github.com/edupilot/backend-go/internal/notify"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/store"
)

type bell struct {
	frames chan notify.Frame
	cancel context.CancelFunc
}

// openBell nối `GET /api/v1/events` của một gateway (máy chủ HTTP thật) và đẩy mọi khung `notification` vào kênh.
func openBell(t *testing.T, baseURL, token string) *bell {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/v1/events", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	b := &bell{frames: make(chan notify.Frame, 16), cancel: cancel}
	ready := make(chan struct{})
	go func() {
		defer func() { _ = resp.Body.Close() }()
		sc := bufio.NewScanner(resp.Body)
		ev, once := "", false
		for sc.Scan() {
			line := sc.Text()
			switch {
			case strings.HasPrefix(line, "event: "):
				ev = strings.TrimPrefix(line, "event: ")
				if ev == "ready" && !once {
					once = true
					close(ready)
				}
			case strings.HasPrefix(line, "data: ") && ev == notify.EventType:
				var f notify.Frame
				if json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &f) == nil {
					b.frames <- f
				}
			}
		}
	}()
	select {
	case <-ready:
	case <-time.After(5 * time.Second):
		t.Fatal("SSE không sẵn sàng")
	}
	t.Cleanup(cancel)
	return b
}

func (b *bell) want(t *testing.T, within time.Duration) notify.Frame {
	t.Helper()
	select {
	case f := <-b.frames:
		return f
	case <-time.After(within):
		t.Fatal("chuông không nhận được khung SSE")
		return notify.Frame{}
	}
}

// TestNotifyAcrossTwoGateways — US-P4-02 AC3 (TLR-20): HAI router độc lập, mỗi cái MỘT client Redis riêng. Giảng viên nối SSE ở bản A, sinh viên bấm `Nhờ giảng viên` ở bản B,
// worker chạy handler → chuông ở A nhận khung; và đảo lại (nối B, sinh ở A). Đo độ trễ commit → khung SSE (≤ 5 s).
func TestNotifyAcrossTwoGateways(t *testing.T) {
	base := newRig(t)
	ctx := t.Context()
	members := course.Resolver{Pool: base.pool}

	newGateway := func() (http.Handler, *appredis.Client) {
		rdb, err := appredis.New(ctx, base.cfg.RedisURL)
		require.NoError(t, err)
		t.Cleanup(func() { _ = rdb.Close() })
		d := httpapi.Deps{Cfg: base.cfg, Log: nil, DB: base.pool, Redis: rdb, Clock: clock.Real{}, State: httpapi.NewState(),
			Escalation: &escalation.Service{Pool: base.pool, Members: members, Log: nil}}
		return httpapi.NewRouter(d), rdb //nolint:contextcheck // NewRouter không nhận ctx (router sống cả đời test)
	}
	hA, _ := newGateway()
	hB, _ := newGateway()
	srvA, srvB := httptest.NewServer(hA), httptest.NewServer(hB)
	t.Cleanup(srvA.Close)
	t.Cleanup(srvB.Close)
	workerRedis, err := appredis.New(ctx, base.cfg.RedisURL)
	require.NoError(t, err)
	t.Cleanup(func() { _ = workerRedis.Close() })
	nt := &escalation.Notifier{Pool: base.pool, Notify: notify.New(base.pool, sse.NewPublisher(workerRedis, 1000, time.Hour), nil)}

	teacher := base.addUser(uniq("gv"), store.UserRoleTEACHER, store.UserStatusACTIVE)
	crs := base.course(teacher.ID, "GW-"+uuid.NewString()[:4])
	base.enroll(crs, teacher.ID, "TEACHER", "ACTIVE")
	issuer := auth.NewIssuer(rigSecret, time.Hour, clock.Real{})
	tok := func(id uuid.UUID, role auth.Role) string {
		s, err := issuer.Issue(id.String(), role, "")
		require.NoError(t, err)
		return s
	}

	// escalate yêu cầu `POST /chat/messages/{mid}/escalate` ở router h, rồi chạy handler outbox như worker.
	escalate := func(h http.Handler, student store.User) (time.Time, uuid.UUID) {
		var sess, user, asst uuid.UUID
		require.NoError(t, base.pool.QueryRow(ctx, `insert into chat_sessions (course_id, user_id) values ($1, $2) returning id`, crs, student.ID).Scan(&sess))
		require.NoError(t, base.pool.QueryRow(ctx, `insert into chat_messages (course_id, session_id, user_id, role, content, stream_status) values ($1, $2, $3, 'USER', 'hỏi', 'DONE') returning id`, crs, sess, student.ID).Scan(&user))
		require.NoError(t, base.pool.QueryRow(ctx, `insert into chat_messages (course_id, session_id, user_id, role, content, stream_status, intent, reply_to) values ($1, $2, $3, 'ASSISTANT', 'trả lời', 'DONE', 'COURSE_QA', $4) returning id`, crs, sess, student.ID, user).Scan(&asst))
		hr := httptest.NewRequest(http.MethodPost, "/api/v1/chat/messages/"+asst.String()+"/escalate", nil)
		hr.Header.Set("Authorization", "Bearer "+tok(student.ID, auth.RoleStudent))
		hr.RemoteAddr = "203.0.113.9:4444"
		w := httptest.NewRecorder()
		h.ServeHTTP(w, hr)
		require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
		committed := time.Now()
		var payload []byte
		require.NoError(t, base.pool.QueryRow(ctx, `select payload from outbox where topic = 'escalation.created' order by created_at desc limit 1`).Scan(&payload))
		require.NoError(t, nt.HandleCreated(ctx, outbox.Message{Topic: escalation.TopicCreated, Payload: payload}))
		return committed, asst
	}

	sv1 := base.addUser(uniq("sv"), store.UserRoleSTUDENT, store.UserStatusACTIVE)
	sv2 := base.addUser(uniq("sv"), store.UserRoleSTUDENT, store.UserStatusACTIVE)
	base.enroll(crs, sv1.ID, "STUDENT", "ACTIVE")
	base.enroll(crs, sv2.ID, "STUDENT", "ACTIVE")

	bellA := openBell(t, srvA.URL, tok(teacher.ID, auth.RoleTeacher))
	t0, _ := escalate(hB, sv1) // nối A, sinh ở B
	f := bellA.want(t, 5*time.Second)
	t.Logf("bell_latency_ms=%d", time.Since(t0).Milliseconds())
	require.Equal(t, notify.TypeTicketNew, f.Type)
	require.Equal(t, "1 câu hỏi đang chờ giảng viên", f.Title)
	require.EqualValues(t, 1, f.UnreadCount)
	require.False(t, f.Read)

	bellB := openBell(t, srvB.URL, tok(teacher.ID, auth.RoleTeacher))
	escalate(hA, sv2) // nối B, sinh ở A
	f = bellB.want(t, 5*time.Second)
	require.Equal(t, "2 câu hỏi đang chờ giảng viên", f.Title, "một dòng gộp, số hiện tại")
	f = bellA.want(t, 5*time.Second) // kết nối A của cùng người cũng nhận
	require.Equal(t, "2 câu hỏi đang chờ giảng viên", f.Title)
}
