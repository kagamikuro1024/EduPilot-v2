package decision_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/decision"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/llmconfig"
	"github.com/edupilot/backend-go/internal/notify"
	"github.com/edupilot/backend-go/internal/testutil"
)

type env struct {
	svc    *decision.Service
	pool   *pgxpool.Pool
	snap   *llm.DecisionSnapshot
	admin  context.Context
	course uuid.UUID
}

func newEnv(t *testing.T) *env {
	t.Helper()
	pool, err := pgxpool.New(t.Context(), testutil.MigratedPostgresURL(t))
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	var teacher, course uuid.UUID
	require.NoError(t, pool.QueryRow(t.Context(), `insert into users (email, full_name, role) values ('gv@x.vn', 'GV', 'TEACHER') returning id`).Scan(&teacher))
	require.NoError(t, pool.QueryRow(t.Context(), `insert into courses (subject_code, class_code, name, semester, join_code, created_by) values ('IT1000', 'CLS001', 'Lớp Mật mã', '2026-2027-HK1', 'ABCDEFG', $1) returning id`, teacher).Scan(&course))
	snap := llm.NewDecisionSnapshot(llm.PGDecisionSource(pool), nil)
	require.NoError(t, snap.Load(t.Context()))
	return &env{svc: &decision.Service{Pool: pool, Snapshot: snap}, pool: pool, snap: snap, course: course,
		admin: auth.WithPrincipal(t.Context(), auth.Principal{Sub: uuid.NewString(), Role: auth.RoleAdmin})}
}

func ptr[T any](v T) *T { return &v }

// TestDecisionOverviewAndGateUpdate: seed hiển thị đúng; sửa cổng có kiểm phiên bản + khoảng; audit_log; ảnh chụp bộ nhớ làm mới ngay.
func TestDecisionOverviewAndGateUpdate(t *testing.T) {
	e := newEnv(t)
	o, err := e.svc.Overview(e.admin)
	require.NoError(t, err)
	require.NotNil(t, o.Provider)
	require.Equal(t, decision.StatusFake, o.Provider.Status)
	require.Equal(t, "decision-fake", o.Provider.Model)
	require.False(t, o.RealAllowed)
	require.Len(t, o.Gates, 3)
	require.Equal(t, 1, o.Gates[0].CoursesTotal)

	g, err := e.svc.PutGate(e.admin, "THREADS_PRECHECK", decision.GateIn{Enabled: ptr(true), Threshold: ptr(decimal.RequireFromString("0.100")), TimeoutMS: ptr(900)}, 1)
	require.NoError(t, err)
	require.Equal(t, 2, g.Version)
	require.True(t, g.Enabled)
	gc, ok := e.snap.Gate(llm.GateThreads)
	require.True(t, ok)
	require.True(t, gc.Enabled, "ảnh chụp bộ nhớ làm mới ngay sau khi ghi")
	require.InDelta(t, 0.1, gc.Threshold, 1e-9)

	var n int
	require.NoError(t, e.pool.QueryRow(t.Context(), `select count(*) from audit_log where entity='decision_gate' and entity_id='THREADS_PRECHECK' and before->>'enabled'='false' and after->>'enabled'='true'`).Scan(&n))
	require.Equal(t, 1, n)

	var vc *llmconfig.ErrVersionConflict
	_, err = e.svc.PutGate(e.admin, "THREADS_PRECHECK", decision.GateIn{Enabled: ptr(false)}, 1)
	require.True(t, errors.As(err, &vc))
	require.Equal(t, 2, vc.Current)

	for field, in := range map[string]decision.GateIn{
		"threshold":  {Threshold: ptr(decimal.RequireFromString("0.21"))},
		"timeout_ms": {TimeoutMS: ptr(5001)},
	} {
		_, err = e.svc.PutGate(e.admin, "THREADS_PRECHECK", in, 2)
		var inv *llmconfig.ErrInvalid
		require.True(t, errors.As(err, &inv), field)
		require.Equal(t, field, inv.Field)
	}
	// khoảng riêng từng cổng
	_, err = e.svc.PutGate(e.admin, "CHAT_ROUTING", decision.GateIn{Threshold: ptr(decimal.RequireFromString("0.60"))}, 1)
	require.Error(t, err)
	_, err = e.svc.PutGate(e.admin, "INJECTION_FLAG", decision.GateIn{Threshold: ptr(decimal.RequireFromString("0.90"))}, 1)
	require.NoError(t, err)
	_, err = e.svc.PutGate(e.admin, "NOPE", decision.GateIn{}, 1)
	require.ErrorIs(t, err, llmconfig.ErrNotFound)
}

// TestDecisionCourseGates: hàng chưa có = phiên bản 0; hiệu lực = cổng hệ thống ∧ lớp ∧ provider; danh sách lớp có tìm + cursor.
func TestDecisionCourseGates(t *testing.T) {
	e := newEnv(t)
	cg, err := e.svc.PutCourseGate(e.admin, e.course, "CHAT_ROUTING", true, 0)
	require.NoError(t, err)
	require.True(t, cg.Enabled)
	require.False(t, cg.Effective, "cổng hệ thống còn tắt")
	_, err = e.svc.PutCourseGate(e.admin, e.course, "CHAT_ROUTING", false, 0)
	var vc *llmconfig.ErrVersionConflict
	require.True(t, errors.As(err, &vc))
	require.Equal(t, 1, vc.Current)

	_, err = e.svc.PutGate(e.admin, "CHAT_ROUTING", decision.GateIn{Enabled: ptr(true)}, 1)
	require.NoError(t, err)
	gs, err := e.svc.CourseGates(e.admin, e.course)
	require.NoError(t, err)
	require.Len(t, gs, 3)
	for _, g := range gs {
		require.Equal(t, g.Gate == "CHAT_ROUTING", g.Effective, g.Gate)
	}
	_, ok := e.snap.Effective(llm.GateRouting, &e.course)
	require.True(t, ok, "ảnh chụp bộ nhớ thấy ngay")

	rows, err := e.svc.Courses(e.admin, "CHAT_ROUTING", "mật", nil, nil, 50)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.True(t, rows[0].Enabled)
	require.Equal(t, 1, rows[0].Version)
	rows, err = e.svc.Courses(e.admin, "CHAT_ROUTING", "100%", nil, nil, 50) // ký tự đặc biệt không phải mẫu
	require.NoError(t, err)
	require.Empty(t, rows)
	_, err = e.svc.PutCourseGate(e.admin, uuid.New(), "CHAT_ROUTING", true, 0)
	require.ErrorIs(t, err, llmconfig.ErrNotFound)
}

// TestDecisionAdminOnly: TEACHER / TA / STUDENT không đọc hay ghi được (AC9 US-DM-02).
func TestDecisionAdminOnly(t *testing.T) {
	e := newEnv(t)
	for _, role := range []auth.Role{auth.RoleTeacher, auth.RoleTA, auth.RoleStudent} {
		ctx := auth.WithPrincipal(t.Context(), auth.Principal{Sub: uuid.NewString(), Role: role})
		_, err := e.svc.Overview(ctx)
		require.ErrorIs(t, err, llmconfig.ErrForbidden, role)
		_, err = e.svc.PutGate(ctx, "THREADS_PRECHECK", decision.GateIn{Enabled: ptr(true)}, 1)
		require.ErrorIs(t, err, llmconfig.ErrForbidden, role)
		_, err = e.svc.Observability(ctx, "24h")
		require.ErrorIs(t, err, llmconfig.ErrForbidden, role)
	}
}

// TestDecisionObservability: số liệu từ llm_audit (task DECIDE): tỉ lệ đường đi, lỗi, độ trễ; cổng không có lời gọi = 0; không nội dung.
func TestDecisionObservability(t *testing.T) {
	e := newEnv(t)
	ins := func(path, status string, latency int) {
		_, err := e.pool.Exec(t.Context(), `insert into llm_audit (task, lane, provider, model, status, trace_id, latency_ms, cost_est, cascade_gate, cascade_path, created_at)
			values ('DECIDE', 'INTERACTIVE', 'Quyết định', 'decision-fake', $1, 'x', $2, 1.5, 'THREADS_PRECHECK', $3, now())`, status, latency, path)
		require.NoError(t, err)
	}
	for i := range 6 {
		ins("FAST", "ok", 100+i*100)
	}
	ins("FALLBACK", "ok", 700)
	ins("FAIL_CLOSED", "error", 800)
	ins("FALLBACK", "overloaded", 5)
	got, err := e.svc.Observability(e.admin, "24h")
	require.NoError(t, err)
	var th, rt decision.ObsGate
	for _, g := range got {
		switch g.Gate {
		case "THREADS_PRECHECK":
			th = g
		case "CHAT_ROUTING":
			rt = g
		}
	}
	require.Equal(t, 9, th.Calls)
	require.InDelta(t, 600.0/9, th.FastPct, 1e-6)
	require.InDelta(t, 200.0/9, th.FallbackPct, 1e-6)
	require.InDelta(t, 100.0/9, th.FailClosedPct, 1e-6)
	require.InDelta(t, 200.0/9, th.ErrorPct, 1e-6)
	require.Equal(t, "14", th.CostVND) // 9 × 1,5 = 13,5 → làm tròn
	require.Positive(t, th.P95MS)
	require.Equal(t, "off", th.State)
	require.Zero(t, rt.Calls)
	_, err = e.svc.Observability(e.admin, "1y")
	require.Error(t, err)
}

// TestDecisionFailClosedAlert: ≥ 20 lời gọi + FAIL_CLOSED > 5 % → MỘT thông báo cho mỗi ADMIN, khử trùng; dưới ngưỡng thì không.
func TestDecisionFailClosedAlert(t *testing.T) {
	e := newEnv(t)
	var adminID uuid.UUID
	require.NoError(t, e.pool.QueryRow(t.Context(), `insert into users (email, full_name, role, status, password_hash, email_verified_at) values ('ad@x.vn', 'Ad', 'ADMIN', 'ACTIVE', 'x', now()) returning id`).Scan(&adminID))
	ins := func(gate, path string, n int) {
		for range n {
			_, err := e.pool.Exec(t.Context(), `insert into llm_audit (task, lane, provider, model, status, trace_id, latency_ms, cascade_gate, cascade_path)
				values ('DECIDE', 'INTERACTIVE', 'Quyết định', 'decision-fake', 'ok', 'x', 10, $1, $2)`, gate, path)
			require.NoError(t, err)
		}
	}
	ins("THREADS_PRECHECK", "FAST", 27)
	ins("THREADS_PRECHECK", "FAIL_CLOSED", 3) // 10 % của 30
	ins("CHAT_ROUTING", "FAST", 18)           // 19 lời gọi: chưa đủ để tính
	ins("CHAT_ROUTING", "FAIL_CLOSED", 1)
	ins("INJECTION_FLAG", "FAST", 39) // đúng 5 % không vượt ngưỡng
	ins("INJECTION_FLAG", "FAIL_CLOSED", 2)
	a := &decision.Alerter{Pool: e.pool, Notify: notify.New(e.pool, nil, nil)}
	n, err := a.Tick(t.Context())
	require.NoError(t, err)
	require.Equal(t, 1, n)
	var cnt int
	require.NoError(t, e.pool.QueryRow(t.Context(), `select count(*) from notifications where user_id = $1 and type = 'DECISION_FAIL_CLOSED' and link = '/observability'`, adminID).Scan(&cnt))
	require.Equal(t, 1, cnt)
	n, err = a.Tick(t.Context())
	require.NoError(t, err)
	require.Zero(t, n, "khử trùng 6 giờ")
}
