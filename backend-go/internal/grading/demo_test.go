package grading_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/grade"
	"github.com/edupilot/backend-go/internal/grading"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/llm/fake"
	"github.com/edupilot/backend-go/internal/llm/llmrt"
	"github.com/edupilot/backend-go/internal/llm/provider"
	"github.com/edupilot/backend-go/internal/platform/config"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/testutil"
)

// demoLLM: provider giả với tệp bản ghi DEMO_MODE thật (seed/demo/fake_llm.json), đúng đường Chat có schema của gateway.
func demoLLM(t *testing.T) *fakeLLM {
	t.Helper()
	p := fake.NewNamed(fake.NewController(fake.Settings{DemoFile: "../../../seed/demo/fake_llm.json"}), "demo", "")
	return &fakeLLM{fn: func(int, context.Context) (json.RawMessage, error) { return nil, nil }, via: func(r llm.Request, schema json.RawMessage) (json.RawMessage, error) {
		msgs := make([]provider.Message, len(r.Messages))
		for i, m := range r.Messages {
			msgs[i] = provider.Message{Role: m.Role, Content: m.Content}
		}
		res, err := p.Chat(context.Background(), provider.ChatOpts{Messages: msgs, Schema: schema})
		return json.RawMessage(res.Text), err
	}}
}

// TestDemoModeRecordingsCoverPolicies — AC4: bản ghi demo có cho CẢ HAI quy chế; lớp 1 không mục chưa rõ, lớp 2 đúng một (ROUNDING_MISSING). Quy chế lạ không khớp bản ghi nào (không đoán bừa).
func TestDemoModeRecordingsCoverPolicies(t *testing.T) {
	var f struct {
		Recordings []struct {
			Name     string
			Response json.RawMessage
		}
	}
	raw, err := os.ReadFile("../../../seed/demo/fake_llm.json")
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, &f))
	require.Len(t, f.Recordings, 2)
	for _, r := range f.Recordings { // gateway kiểm đầu ra theo schema của `llm.Structured`: bản ghi sai schema = LLM_UNAVAILABLE khi seed chạy thật
		require.NoError(t, llm.ValidateJSON(grading.ExtractSchema, r.Response), r.Name)
	}
	e := newEnv(t, demoLLM(t))
	res, err := e.run(e.policyDoc("quy-che-lop1.md"), false, 0)
	require.NoError(t, err)
	require.Empty(t, res.(grading.Result).UnclearCode)
	var ws []string
	rows, err := e.pool.Query(e.ctx, `select weight::text from grade_components where course_id = $1 order by position`, e.course)
	require.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		var w string
		require.NoError(t, rows.Scan(&w))
		ws = append(ws, w)
	}
	require.Equal(t, []string{"20.00", "20.00", "60.00"}, ws)

	e2 := newEnv(t, demoLLM(t))
	res, err = e2.run(e2.policyDoc("quy-che-lop2.md"), true, 0)
	require.NoError(t, err)
	require.Equal(t, grade.CodeRoundingMissing, res.(grading.Result).UnclearCode)
}

// TestDemoModeThroughGateway — đúng đường seed chạy thật: cổng LLM thật (che danh tính, kiểm schema, provider giả có LLM_DEMO_FILE) trích được quy chế lớp 1 và lớp 2.
func TestDemoModeThroughGateway(t *testing.T) {
	testutil.RequireContainers(t)
	url := testutil.MigratedPostgresURL(t)
	pool := testutil.RuntimePoolAt(t, url)
	abs, err := filepath.Abs("../../../seed/demo/fake_llm.json")
	require.NoError(t, err)
	env := map[string]string{
		"DATABASE_URL": url, "REDIS_URL": testutil.RedisURL(t), "JWT_SECRET_KEY": "grading-test-secret-0123456789abcdef-dev",
		"BLOB_ENDPOINT": "127.0.0.1:9", "BLOB_BUCKET": "x", "BLOB_ACCESS_KEY": "x", "BLOB_SECRET_KEY": "x",
		"APP_ENCRYPTION_KEY": "ZWR1cGlsb3QtZGV2LWVuY3J5cHRpb24ta2V5LTMyYnk=", "APP_ENV": "test", "LLM_PROVIDER": "fake",
		"LLM_DEMO_FILE": abs, "LLM_DEFAULT_RPM": "6000", "PRIVACY_MASK_TIMEOUT_MS": "1000", "BCRYPT_COST": "4",
	}
	cfg, err := config.Load(func(k string) string { return env[k] }, config.Gateway)
	require.NoError(t, err)
	rdb, err := appredis.New(t.Context(), cfg.RedisURL)
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })
	rt, err := llmrt.New(t.Context(), cfg, pool, rdb.Client, slog.New(slog.NewTextHandler(io.Discard, nil)))
	require.NoError(t, err)
	t.Cleanup(func() { rt.Close(context.WithoutCancel(t.Context())) })
	for md, code := range map[string]string{"quy-che-lop1.md": "", "quy-che-lop2.md": grade.CodeRoundingMissing} {
		e := newEnvPool(t, nil, pool)
		e.svc.LLM = rt.Gateway
		res, err := e.run(e.policyDoc(md), false, 0)
		require.NoError(t, err, md)
		require.Equal(t, code, res.(grading.Result).UnclearCode, md)
	}
}

func TestDemoFileProdForbidden(t *testing.T) {
	env := map[string]string{
		"DATABASE_URL": "postgres://u:p@h/db", "REDIS_URL": "redis://h", "JWT_SECRET_KEY": "grading-test-secret-0123456789abcdef-dev",
		"BLOB_ENDPOINT": "127.0.0.1:9", "BLOB_BUCKET": "x", "BLOB_ACCESS_KEY": "x", "BLOB_SECRET_KEY": "x",
		"APP_ENCRYPTION_KEY": "ZWR1cGlsb3QtZGV2LWVuY3J5cHRpb24ta2V5LTMyYnk=", "APP_ENV": "production", "LLM_DEMO_FILE": "/x.json",
	}
	_, err := config.Load(func(k string) string { return env[k] }, config.Gateway)
	require.ErrorContains(t, err, "LLM_DEMO_FILE")
}
