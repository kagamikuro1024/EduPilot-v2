package llm_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/llm"
)

// TestAuditPGWriter — US-P1-02 AC8: lô được COPY vào llm_audit đúng cột; không có cột nội dung nào.
func TestAuditPGWriter(t *testing.T) {
	t.Parallel()
	r := newCfgRig(t)
	uid, cid := uuid.New(), uuid.New()
	rows := []llm.AuditRow{
		{Task: "CHAT", Lane: "INTERACTIVE", Provider: "OA", Model: "gpt-x", TokensIn: 10, TokensOut: 5, LatencyMS: 120, QueueWaitMS: 3, Attempts: 2,
			FallbackIndex: 1, CostEst: decimal.RequireFromString("0.0123"), Status: "ok", PIIMaskedCount: 2, UserID: &uid, CourseID: &cid, TraceID: "abc", At: time.Now()},
		{Task: "INSIGHT", Lane: "BATCH", Status: "rate_limited", ErrorKind: "RATE_LIMIT", Degraded: true, TraceID: "def", At: time.Now()},
	}
	require.NoError(t, llm.PGWriter(r.pool)(context.Background(), rows))

	var n int
	var cost decimal.Decimal
	var provider, errKind *string
	require.NoError(t, r.pool.QueryRow(t.Context(), `select count(*) from llm_audit`).Scan(&n))
	require.Equal(t, 2, n)
	require.NoError(t, r.pool.QueryRow(t.Context(), `select cost_est, provider from llm_audit where task='CHAT'`).Scan(&cost, &provider))
	require.True(t, cost.Equal(decimal.RequireFromString("0.0123")))
	require.Equal(t, "OA", *provider)
	require.NoError(t, r.pool.QueryRow(t.Context(), `select error_kind from llm_audit where task='INSIGHT'`).Scan(&errKind))
	require.Equal(t, "RATE_LIMIT", *errKind)

	var content int
	require.NoError(t, r.pool.QueryRow(t.Context(),
		`select count(*) from information_schema.columns where table_name='llm_audit' and column_name ~* 'prompt|content|message|text|response'`).Scan(&content))
	require.Zero(t, content, "llm_audit không được có cột nội dung")
}

// TestAuditPIIMaskedCountPG — US-P3-03 AC5: gateway có Masker ghi pii_masked_count thật vào llm_audit (PG); không cột nội dung.
func TestAuditPIIMaskedCountPG(t *testing.T) {
	t.Parallel()
	r := newCfgRig(t)
	aud := llm.NewAuditor(context.Background(), llm.PGWriter(r.pool), newDiscardLog())
	defer aud.Close(context.Background())
	p := &stub{name: "p"}
	reg := llm.NewStaticRegistry(map[llm.Task]llm.Route{})
	reg.SetRoute(llm.TaskChat, llm.Route{Targets: []llm.Target{{ProviderID: "p", ProviderName: "p", Type: "fake", Model: "m", RPM: 60, TPM: 100000, P: p}}})
	g := llm.New(llm.Options{Registry: reg, Auditor: aud, Log: newDiscardLog(), Masker: realMasker(newDiscardLog())})
	_, err := g.Chat(withCourse(t.Context()), llm.Request{Task: llm.TaskChat, Messages: userMsg("Em " + leakName + " mssv " + leakMSSV + " mail " + leakEmail)})
	require.NoError(t, err)
	g.FlushAudit(context.Background())
	var n int
	var course *uuid.UUID
	require.NoError(t, r.pool.QueryRow(t.Context(), `select pii_masked_count, course_id from llm_audit order by created_at desc limit 1`).Scan(&n, &course))
	require.Equal(t, 3, n)
	require.NotNil(t, course)
	require.Equal(t, maskCourse, *course)
}
