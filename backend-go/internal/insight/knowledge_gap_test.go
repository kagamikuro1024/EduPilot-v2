package insight_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/insight"
	"github.com/edupilot/backend-go/internal/jobs"
	"github.com/edupilot/backend-go/internal/llm"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/testutil"
)

// stubLLM: nhúng theo chủ đề trong câu (TCP→trục 0, SQL→1, Docker→2, còn lại→3), đặt tên theo từ khoá; failNames / failEmbed mô phỏng LLM hỏng.
type stubLLM struct {
	llm.Client
	failNames, failEmbed bool
	structured           int
}

func (s *stubLLM) Embed(_ context.Context, r llm.EmbedRequest) ([][]float32, error) {
	if s.failEmbed {
		return nil, errors.New("embed down")
	}
	out := make([][]float32, len(r.Inputs))
	for i, t := range r.Inputs {
		v := make([]float32, llm.EmbedDims)
		axis := 3
		switch {
		case strings.Contains(t, "TCP"):
			axis = 0
		case strings.Contains(t, "SQL"):
			axis = 1
		case strings.Contains(t, "Docker"):
			axis = 2
		}
		v[axis] = 1
		v[100+len(t)%7] = 0.05
		out[i] = v
	}
	return out, nil
}

func (s *stubLLM) Structured(_ context.Context, r llm.Request, _ json.RawMessage) (json.RawMessage, error) {
	s.structured++
	if s.failNames {
		return nil, errors.New("llm down")
	}
	// đặt tên theo từ khoá trong nhóm: đọc "Nhóm N" + câu mẫu đầu tiên
	var topics []map[string]any
	for _, blk := range strings.Split(r.Messages[1].Content, "Nhóm ")[1:] {
		id := 0
		_, _ = fmt.Sscanf(blk, "%d", &id)
		name := "Chủ đề khác"
		for _, k := range []string{"TCP", "SQL", "Docker"} {
			if strings.Contains(blk, k) {
				name = "Chủ đề " + k
			}
		}
		topics = append(topics, map[string]any{"id": id, "name": name, "recommendation": "Ôn lại " + name})
	}
	return json.Marshal(map[string]any{"topics": topics})
}

type rig struct {
	t       *testing.T
	pool    *pgxpool.Pool
	svc     *insight.Service
	llm     *stubLLM
	course  uuid.UUID
	teacher uuid.UUID
	studs   []uuid.UUID
	sess    map[uuid.UUID]uuid.UUID
}

func newRig(t *testing.T, students int) *rig {
	t.Helper()
	pool, err := pgxpool.New(t.Context(), testutil.MigratedPostgresURL(t))
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	rdb, err := appredis.New(t.Context(), testutil.RedisURL(t))
	require.NoError(t, err)
	sfx := strings.ToLower(uuid.NewString()[:8])
	r := &rig{t: t, pool: pool, llm: &stubLLM{}, sess: map[uuid.UUID]uuid.UUID{}}
	r.svc = &insight.Service{Pool: pool, Redis: rdb, Jobs: jobs.NewService(pool), LLM: r.llm}
	require.NoError(t, pool.QueryRow(t.Context(), `insert into users (email, full_name, role) values ($1, 'Giảng Viên Thử', 'TEACHER') returning id`, "gv."+sfx+"@example.test").Scan(&r.teacher))
	require.NoError(t, pool.QueryRow(t.Context(), `insert into courses (subject_code, class_code, name, semester, join_code, created_by) values ('INT1006', $1, 'Lớp', '2026-2027-HK1', $2, $3) returning id`, "IN-"+strings.ToUpper(sfx), joinCode(), r.teacher).Scan(&r.course))
	for i := range students {
		var id uuid.UUID
		require.NoError(t, pool.QueryRow(t.Context(), `insert into users (email, full_name, role) values ($1, $2, 'STUDENT') returning id`, fmt.Sprintf("sv%d.%s@example.test", i, sfx), fmt.Sprintf("Sinh Viên Số%d", i)).Scan(&id))
		r.exec(`insert into enrollments (course_id, user_id, role_in_course, status, joined_via) values ($1, $2, 'STUDENT', 'ACTIVE', 'ADMIN')`, r.course, id)
		r.studs = append(r.studs, id)
	}
	return r
}

func (r *rig) exec(sql string, args ...any) {
	r.t.Helper()
	_, err := r.pool.Exec(r.t.Context(), sql, args...)
	require.NoError(r.t, err, sql)
}

// ask: sinh viên thứ i hỏi `text`; AI trả lời với cờ tương ứng.
func (r *rig) ask(i int, text string, noCtx, low bool) {
	r.t.Helper()
	s := r.studs[i]
	sid, ok := r.sess[s]
	if !ok {
		require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `insert into chat_sessions (course_id, user_id) values ($1, $2) returning id`, r.course, s).Scan(&sid))
		r.sess[s] = sid
	}
	var q uuid.UUID
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `insert into chat_messages (course_id, session_id, user_id, role, content) values ($1,$2,$3,'USER',$4) returning id`, r.course, sid, s, text).Scan(&q))
	r.exec(`insert into chat_messages (course_id, session_id, user_id, role, content, reply_to, no_context, low_confidence) values ($1,$2,$3,'ASSISTANT','Trả lời',$4,$5,$6)`, r.course, sid, s, q, noCtx, low)
}

func (r *rig) seed() {
	for k := range 30 { // A: 6 sinh viên, tự tin thấp nhiều
		r.ask(k%6, fmt.Sprintf("TCP bắt tay ba bước là gì, câu %d?", k), false, k%2 == 0)
	}
	for k := range 20 { // B: 5 sinh viên
		r.ask(6+k%5, fmt.Sprintf("Viết SQL join hai bảng thế nào, câu %d?", k), false, false)
	}
	for k := range 2 { // C: đúng 2 sinh viên
		r.ask(11+k, fmt.Sprintf("Docker compose dùng ra sao, câu %d?", k), false, false)
	}
	for k := range 10 { // ngoài tài liệu
		r.ask(k%4, "Cách cấu hình quantum router hãy giải thích", true, false)
	}
}

func (r *rig) report() insight.Payload {
	r.t.Helper()
	out, err := r.svc.Latest(r.t.Context(), r.course)
	require.NoError(r.t, err)
	return out.Report.Payload
}

// TestKnowledgeGapUncoveredKAnonymity — P-11-31 F-1: «Tài liệu chưa đề cập» chỉ liệt kê câu do ≥ 3 sinh viên KHÁC NHAU hỏi; một người hỏi nhiều lần không hiện.
func TestKnowledgeGapUncoveredKAnonymity(t *testing.T) {
	r := newRig(t, 13)
	for range 5 {
		r.ask(0, "Em ở KTX khu B phòng 302 xin hoãn bài được không", true, false) // 1 sinh viên × 5 lần
	}
	for k := range 2 {
		r.ask(1+k, "Cách cấu hình quantum router hãy giải thích", true, false) // 2 sinh viên
	}
	for k := range 3 {
		r.ask(3+k, "Hệ thống chấm bài dùng ngôn ngữ nào", true, false) // 3 sinh viên
	}
	for k := range 12 { // đủ dữ liệu để chạy báo cáo
		r.ask(k%6, fmt.Sprintf("TCP bắt tay ba bước là gì, câu %d?", k), false, false)
	}
	_, err := r.svc.RunNow(t.Context(), r.teacher, r.course, 30)
	require.NoError(t, err)
	p := r.report()
	require.Len(t, p.Uncovered, 1, "chỉ câu do ≥ 3 sinh viên hỏi")
	require.Equal(t, "Hệ thống chấm bài dùng ngôn ngữ nào", p.Uncovered[0].Text)
}

// TestKnowledgeGap — US-P10-04: A, B đứng đầu; chủ đề < 3 sinh viên không có câu mẫu; ngoài tài liệu có mục riêng; không danh tính trong payload.
func TestKnowledgeGap(t *testing.T) {
	r := newRig(t, 13)
	ctx := t.Context()
	r.seed()
	_, err := r.svc.RunNow(ctx, r.teacher, r.course, 30)
	require.NoError(t, err)
	p := r.report()

	require.Equal(t, 30, p.PeriodDays)
	require.Equal(t, 62, p.Counts.Questions)
	require.GreaterOrEqual(t, len(p.Topics), 3)
	require.Equal(t, "Chủ đề TCP", p.Topics[0].Name)
	require.Equal(t, "Chủ đề SQL", p.Topics[1].Name)
	require.Equal(t, "llm", p.Topics[0].NamedBy)
	require.Equal(t, 1, p.Topics[0].Rank)
	require.Equal(t, 6, p.Topics[0].Signals.Students)
	require.InDelta(t, 0.5, p.Topics[0].Signals.LowRate, 0.01)
	require.Nil(t, p.Topics[0].Signals.PracticeWrongRate, "lớp chưa luyện đề: tín hiệu P vắng, trọng số chia đều")

	// k = 3: A, B có 3–5 câu mẫu; C (2 sinh viên) không có câu mẫu + nhãn
	require.GreaterOrEqual(t, len(p.Topics[0].Examples), 3)
	require.LessOrEqual(t, len(p.Topics[0].Examples), insight.MaxExamples)
	var c *insight.Topic
	for i := range p.Topics {
		if p.Topics[i].Name == "Chủ đề Docker" {
			c = &p.Topics[i]
		}
	}
	require.NotNil(t, c)
	require.True(t, c.FewAsked)
	require.Empty(t, c.Examples)

	// ngoài tài liệu: 10 câu trùng gộp thành 1 mục
	require.Len(t, p.Uncovered, 1)
	require.Equal(t, 10, p.Uncovered[0].Count)

	// payload không mang danh tính
	raw, _ := json.Marshal(p)
	for _, s := range r.studs {
		require.NotContains(t, string(raw), s.String())
	}
	require.NotContains(t, string(raw), "Sinh Viên")
	require.NotContains(t, string(raw), "@example.test")

	// Markdown xuất được
	rep, err := r.svc.Latest(ctx, r.course)
	require.NoError(t, err)
	md, err := r.svc.Export(ctx, r.course, rep.Report.ID)
	require.NoError(t, err)
	require.Contains(t, md, "## 1. Chủ đề TCP")
}

func TestKnowledgeGapGenerateGates(t *testing.T) {
	r := newRig(t, 13)
	ctx := t.Context()

	// < 20 câu → 409 NOT_ENOUGH_DATA, không xếp việc
	for k := range 5 {
		r.ask(k, "TCP là gì", false, false)
	}
	_, err := r.svc.Generate(ctx, r.teacher, r.course, 30)
	var ae *apierr.Error
	require.ErrorAs(t, err, &ae)
	require.Equal(t, http.StatusConflict, ae.Status)
	require.Equal(t, apierr.NotEnoughData, ae.Code)

	_, err = r.svc.Latest(ctx, r.course)
	require.ErrorAs(t, err, &ae)
	require.Equal(t, http.StatusNotFound, ae.Status)

	r.seed()
	out, err := r.svc.Generate(ctx, r.teacher, r.course, 30)
	require.NoError(t, err)
	require.NotNil(t, out.JobID)
	require.False(t, out.Reused)

	// kỳ ngoài {7, 14, 30, 120} → 422 (Q-E-7); 0 = mặc định 14
	for _, bad := range []int{-1, 5, 999} {
		_, err = r.svc.Generate(ctx, r.teacher, r.course, bad)
		require.ErrorAs(t, err, &ae, bad)
		require.Equal(t, http.StatusUnprocessableEntity, ae.Status, bad)
	}

	// chạy xong rồi gọi lại không có dữ liệu mới → reused, 0 lời gọi LLM, không xếp việc
	_, err = r.svc.RunNow(ctx, r.teacher, r.course, 30)
	require.NoError(t, err)
	calls := r.llm.structured
	out, err = r.svc.Generate(ctx, r.teacher, r.course, 30)
	require.NoError(t, err)
	require.True(t, out.Reused)
	require.NotNil(t, out.ReportID)
	require.Equal(t, calls, r.llm.structured)
	latest, err := r.svc.Latest(ctx, r.course)
	require.NoError(t, err)
	require.False(t, latest.HasNewData)

	// có câu hỏi mới → xếp việc mới + has_new_data
	time.Sleep(10 * time.Millisecond)
	r.ask(0, "TCP bổ sung câu mới", false, false)
	latest, err = r.svc.Latest(ctx, r.course)
	require.NoError(t, err)
	require.True(t, latest.HasNewData)
	out, err = r.svc.Generate(ctx, r.teacher, r.course, 30)
	require.NoError(t, err)
	require.NotNil(t, out.JobID)

	// trần 3 việc / giờ / lớp: đã có 2 việc (30 ngày); việc thứ 3 qua, thứ 4 → 429 + Retry-After
	_, err = r.svc.Generate(ctx, r.teacher, r.course, 7)
	require.NoError(t, err)
	_, err = r.svc.Generate(ctx, r.teacher, r.course, 14)
	require.ErrorAs(t, err, &ae)
	require.Equal(t, http.StatusTooManyRequests, ae.Status)
	require.Positive(t, ae.RetryAfter)
}

func TestKnowledgeGapLLMFailures(t *testing.T) {
	r := newRig(t, 13)
	ctx := t.Context()
	r.seed()

	// lỗi ở bước NAME → vẫn có báo cáo, tên tạm từ từ khoá, named_by = fallback
	r.llm.failNames = true
	_, err := r.svc.RunNow(ctx, r.teacher, r.course, 30)
	require.NoError(t, err)
	p := r.report()
	require.NotEmpty(t, p.Topics)
	for _, tp := range p.Topics {
		require.Equal(t, "fallback", tp.NamedBy)
		require.NotEmpty(t, tp.Name)
	}
	require.Equal(t, 1, p.Topics[0].Rank) // hạng do code, không phụ thuộc LLM

	// lỗi trước NAME (nhúng) → việc lỗi, không thêm báo cáo
	r.llm.failEmbed = true
	_, err = r.svc.RunNow(ctx, r.teacher, r.course, 30)
	require.Error(t, err)
	var n int
	require.NoError(t, r.pool.QueryRow(ctx, `select count(*) from insight_reports where course_id = $1`, r.course).Scan(&n))
	require.Equal(t, 1, n)
}

// Dữ liệu lớp khác không lọt vào báo cáo.
func TestKnowledgeGapCourseIsolation(t *testing.T) {
	a := newRig(t, 13)
	a.seed()
	b := newRig(t, 13)
	for k := range 25 {
		b.ask(k%6, fmt.Sprintf("Docker volume là gì, câu %d?", k), false, false)
	}
	_, err := b.svc.RunNow(t.Context(), b.teacher, b.course, 30)
	require.NoError(t, err)
	p := b.report()
	require.Equal(t, 25, p.Counts.Questions)
	require.Equal(t, "Chủ đề Docker", p.Topics[0].Name)
}

func joinCode() string {
	const alpha = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	id := uuid.New()
	b := make([]byte, 7)
	for i := range b {
		b[i] = alpha[int(id[i])%len(alpha)]
	}
	return string(b)
}

// Hai việc đồng thời cùng lớp + kỳ + dữ liệu không tạo hai báo cáo trùng watermark (TC-04-31).
func TestKnowledgeGapConcurrentSingleReport(t *testing.T) {
	r := newRig(t, 13)
	r.seed()
	errs := make(chan error, 2)
	for range 2 {
		go func() { _, err := r.svc.RunNow(context.Background(), r.teacher, r.course, 30); errs <- err }()
	}
	require.NoError(t, <-errs)
	require.NoError(t, <-errs)
	var n int
	require.NoError(t, r.pool.QueryRow(t.Context(), `select count(*) from insight_reports where course_id = $1`, r.course).Scan(&n))
	require.Equal(t, 1, n)
}
