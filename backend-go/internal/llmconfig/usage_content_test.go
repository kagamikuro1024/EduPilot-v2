package llmconfig_test

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/privacy"
)

// TestUsageNoContent — US-P3-03 AC9: sau một lượt chat có canary, `GET /admin/llm/usage` (mọi nhóm) chỉ có số đếm / trạng thái;
// `llm_audit` không có cột chứa chữ nội dung và không cột nào chứa canary; TA không đọc được.
func TestUsageNoContent(t *testing.T) {
	r := getAPI(t)
	ctx := t.Context()
	canary := "CANARY-" + strings.ToUpper(uuid.NewString()[:8])
	var teacher, course uuid.UUID
	require.NoError(t, r.pool.QueryRow(ctx, `insert into users (email, full_name, role) values ($1, 'GV', 'TEACHER') returning id`, uuid.NewString()+"@example.test").Scan(&teacher))
	id := uuid.New()
	const alpha = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	jc := make([]byte, 7)
	for i := range jc {
		jc[i] = alpha[int(id[i])%len(alpha)]
	}
	require.NoError(t, r.pool.QueryRow(ctx, `insert into courses (subject_code, class_code, name, semester, join_code, created_by) values ('INT1006', $1, 'Lớp', '2026-2027-HK1', $2, $3) returning id`, "UC"+id.String()[:8], string(jc), teacher).Scan(&course))

	gctx := privacy.WithSession(llm.WithIdentity(ctx, llm.Identity{UserID: &teacher, CourseID: &course}), privacy.NewSession(uuid.NewString()))
	_, err := r.rt.Gateway.Chat(gctx, llm.Request{Task: llm.TaskChat, Messages: []llm.Message{{Role: "user", Content: "Câu hỏi có " + canary + " và mail a@b.vn"}}})
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		var n int
		_ = r.pool.QueryRow(ctx, `select count(*) from llm_audit`).Scan(&n)
		return n > 0
	}, 10*time.Second, 20*time.Millisecond, "llm_audit được ghi bất đồng bộ")

	// lược đồ: không cột nào kiểu chữ dài có thể chứa nội dung (chỉ mã / nhãn ngắn)
	rows, err := r.pool.Query(ctx, `select column_name from information_schema.columns where table_name='llm_audit' and data_type in ('text','jsonb','json','character varying')`)
	require.NoError(t, err)
	var cols []string
	for rows.Next() {
		var c string
		require.NoError(t, rows.Scan(&c))
		cols = append(cols, c)
	}
	rows.Close()
	for _, c := range cols {
		require.NotContains(t, []string{"prompt", "content", "messages", "response", "output", "input"}, c)
		var n int
		require.NoError(t, r.pool.QueryRow(ctx, `select count(*) from llm_audit where `+c+`::text like $1 or `+c+`::text like '%a@b.vn%'`, "%"+canary+"%").Scan(&n))
		require.Zero(t, n, "llm_audit.%s chứa nội dung", c)
	}

	adm := r.token(t, auth.RoleAdmin)
	allowed := map[string]bool{"key": true, "calls": true, "tokens_in": true, "tokens_out": true, "cost_est": true, "latency_p50_ms": true, "latency_p95_ms": true, "errors": true, "degraded": true, "pii_masked": true, "pii_masked_count": true}
	for _, g := range []string{"task", "day", "provider", "model"} {
		x := r.do(t, adm, "GET", "/admin/llm/usage?group="+g, nil)
		if x.status == 422 { // nhóm không hỗ trợ ở bản này
			continue
		}
		require.Equal(t, 200, x.status, string(x.body))
		require.NotContains(t, string(x.body), canary)
		require.NotContains(t, string(x.body), "a@b.vn")
		for _, it := range x.obj(t)["items"].([]any) {
			for k := range it.(map[string]any) {
				require.True(t, allowed[k], "usage.%s: trường lạ — có thể chứa nội dung", k)
			}
		}
	}
	require.Equal(t, 403, r.do(t, r.token(t, auth.RoleTA), "GET", "/admin/llm/usage", nil).status)
}
