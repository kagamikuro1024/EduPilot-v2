package grading_test

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/grading"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/llm/llmrt"
	"github.com/edupilot/backend-go/internal/platform/config"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/testutil"
)

// gatewayRuntime dựng cổng LLM THẬT (che danh tính theo danh sách cả lớp; provider giả không có bản ghi → lỗi tạm) trên CSDL thật của test.
func gatewayRuntime(t *testing.T) (*llmrt.Runtime, *pgxpool.Pool) {
	t.Helper()
	testutil.RequireContainers(t)
	url := testutil.MigratedPostgresURL(t)
	pool := testutil.RuntimePoolAt(t, url)
	env := map[string]string{
		"DATABASE_URL": url, "REDIS_URL": testutil.RedisURL(t), "JWT_SECRET_KEY": "grading-test-secret-0123456789abcdef-dev",
		"BLOB_ENDPOINT": "127.0.0.1:9", "BLOB_BUCKET": "x", "BLOB_ACCESS_KEY": "x", "BLOB_SECRET_KEY": "x",
		"APP_ENCRYPTION_KEY": "ZWR1cGlsb3QtZGV2LWVuY3J5cHRpb24ta2V5LTMyYnk=", "APP_ENV": "test", "LLM_PROVIDER": "fake",
		"LLM_REPLAY_DIR": t.TempDir(), "LLM_DEFAULT_RPM": "6000", "PRIVACY_MASK_TIMEOUT_MS": "1000", "BCRYPT_COST": "4",
	}
	cfg, err := config.Load(func(k string) string { return env[k] }, config.Gateway)
	require.NoError(t, err)
	rdb, err := appredis.New(t.Context(), cfg.RedisURL)
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })
	rt, err := llmrt.New(t.Context(), cfg, pool, rdb.Client, slog.New(slog.NewTextHandler(io.Discard, nil)))
	require.NoError(t, err)
	return rt, pool
}

// TestClassmateNameMaskedByGateway — tên bạn cùng lớp trong bài (bài nhóm), MSSV và email không rời hệ thống: việc che là của gateway theo danh sách cả lớp (TLR-6);
// nhãn "Họ tên: / MSSV:" của chính người nộp đã bị bỏ trước (chấm mù); llm_audit không có cột nội dung và mỗi lời gọi là GRADING / BATCH.
func TestClassmateNameMaskedByGateway(t *testing.T) {
	rt, pool := gatewayRuntime(t)
	e := newEnvPool(t, nil, pool)
	g := &gradeEnv{env: e, blob: &memBlob{m: map[string][]byte{}}, ext: &fakeExtract{}}
	g.rubric = rubric(t)
	g.initAssignment(t)
	g.grader = &grading.Grader{Pool: pool, LLM: rt.Gateway, Blob: g.blob, Extract: g.ext, Cfg: grading.DefaultEssayConfig()}
	mate := e.user("STUDENT", "Trần Thị Bạnhọc")
	e.exec(`insert into enrollments (course_id, user_id, role_in_course, status, joined_via, student_code_snapshot) values ($1, $2, 'STUDENT', 'ACTIVE', 'ADMIN', '20228888')`, e.course, mate)
	body := "Họ tên: Lê Văn Nộpbài\nMSSV: 20227777\nChúng em làm nhóm cùng Trần Thị Bạnhọc (MSSV 20228888, email banhoc@example.test). " + strings.Repeat("Phân tích chuỗi cung ứng phần mềm. ", 5)
	res := g.submit("Lê Văn Nộpbài", "bai.md", body, nil)
	_, _ = g.grader.Run(t.Context(), e.course, res) // provider giả không có bản ghi → không có điểm; điều cần kiểm là payload đã gửi
	payloads := rt.Registry.Fake().Payloads()
	require.NotEmpty(t, payloads)
	for _, p := range payloads {
		all := strings.Join(p.Messages, "\n")
		require.NotContains(t, all, "Trần Thị Bạnhọc", "tên bạn cùng lớp không rời hệ thống")
		require.NotContains(t, all, "20228888")
		require.NotContains(t, all, "20227777")
		require.NotContains(t, all, "Lê Văn Nộpbài")
		require.NotContains(t, all, "banhoc@example.test")
		require.Contains(t, all, "Phân tích chuỗi cung ứng", "nội dung học thuật còn nguyên")
	}
	rt.Close(context.WithoutCancel(t.Context()))
	cols, err := pool.Query(t.Context(), `select column_name from information_schema.columns where table_name = 'llm_audit'`)
	require.NoError(t, err)
	defer cols.Close()
	for cols.Next() {
		var c string
		require.NoError(t, cols.Scan(&c))
		require.NotContains(t, []string{"content", "prompt", "messages", "body", "text"}, c, "llm_audit không có cột nội dung")
	}
	rs, err := pool.Query(t.Context(), `select task, lane from llm_audit where course_id = $1`, e.course)
	require.NoError(t, err)
	defer rs.Close()
	n := 0
	for rs.Next() {
		var task, lane string
		require.NoError(t, rs.Scan(&task, &lane))
		require.Equal(t, "GRADING", task)
		require.Equal(t, "BATCH", lane)
		n++
	}
	require.Equal(t, len(payloads), n, "mỗi lời gọi tới provider có đúng một dòng llm_audit")
}

// TestMaskFailedNoLLMCall — thiếu danh tính (không CourseID) thì gateway từ chối và KHÔNG gọi provider.
func TestMaskFailedNoLLMCall(t *testing.T) {
	rt, _ := gatewayRuntime(t)
	lane := llm.LaneBatch
	_, err := rt.Gateway.Structured(t.Context(), llm.Request{Task: llm.TaskGrading, Lane: &lane, Messages: []llm.Message{{Role: "user", Content: "Bài làm có tên Nguyễn Văn A"}}}, grading.EssaySchema(rubric(t)))
	require.ErrorIs(t, err, llm.ErrMaskFailed)
	require.Empty(t, rt.Registry.Fake().Payloads(), "không lời gọi nào tới provider")
}
