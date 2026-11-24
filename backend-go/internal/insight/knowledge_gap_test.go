package insight_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/insight"
	"github.com/edupilot/backend-go/internal/jobs"
	"github.com/edupilot/backend-go/internal/llm"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/privacy"
	"github.com/edupilot/backend-go/internal/testutil"
)

// stubLLM: nhúng theo chủ đề trong câu (TCP→trục 0, SQL→1, Docker→2, còn lại→3), đặt tên theo từ khoá; failNames / failEmbed mô phỏng LLM hỏng.
type stubLLM struct {
	llm.Client
	mu                   sync.Mutex
	failNames, failEmbed bool
	leakName             string // khác rỗng: mọi chủ đề được «đặt tên» bằng chuỗi này (mô phỏng LLM chép câu dưới ngưỡng k)
	structured           int
	seen                 []string // nội dung user của mọi lời gọi đặt tên
	embedded             []string // mọi chuỗi gửi đi nhúng
	batches              []int    // cỡ từng lời gọi nhúng
	piiSeen              []int    // PIIMaskedCount của mọi lời gọi đặt tên
}

func (s *stubLLM) Embed(_ context.Context, r llm.EmbedRequest) ([][]float32, error) {
	if s.failEmbed {
		return nil, errors.New("embed down")
	}
	s.mu.Lock()
	s.embedded = append(s.embedded, r.Inputs...)
	s.batches = append(s.batches, len(r.Inputs))
	s.mu.Unlock()
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
	s.mu.Lock()
	s.structured++
	s.seen = append(s.seen, r.Messages[1].Content)
	s.piiSeen = append(s.piiSeen, r.PIIMaskedCount)
	s.mu.Unlock()
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
			if strings.Contains(strings.ToLower(blk), strings.ToLower(k)) {
				name = "Chủ đề " + k
			}
		}
		if s.leakName != "" {
			name = s.leakName
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
		r.ask(k%6, fmt.Sprintf("TCP bắt tay ba bước là gì, câu %d?", k%5), false, k%2 == 0) // 5 câu, mỗi câu do nhiều sinh viên hỏi
	}
	for k := range 20 { // B: 5 sinh viên
		r.ask(6+k%5, fmt.Sprintf("Viết SQL join hai bảng thế nào, câu %d?", k%4), false, false)
	}
	for k := range 2 { // C: đúng 2 sinh viên
		r.ask(11+k, fmt.Sprintf("Docker compose dùng ra sao, câu %d?", k), false, false)
	}
	for k := range 3 { // ngoài tài liệu N1: 3 sinh viên
		r.ask(k, "Cách cấu hình quantum router hãy giải thích", true, false)
	}
	for k := range 4 { // N2: 4 sinh viên
		r.ask(k, "Giá vàng hôm nay là bao nhiêu", true, false)
	}
	for range 3 { // N3: MỘT sinh viên hỏi 3 lần — bị bỏ (P-11-31)
		r.ask(5, "Em ở KTX khu B phòng 302 xin hoãn bài được không", true, false)
	}
}

func (r *rig) report() insight.Payload {
	r.t.Helper()
	out, err := r.svc.Latest(r.t.Context(), r.course)
	require.NoError(r.t, err)
	return out.Report.Payload
}

// TestKnowledgeGapUncoveredMinStudents — P-11-31 F-1 (US-P10-04 AC14): «Tài liệu chưa đề cập» gom theo câu chuẩn hoá và chỉ giữ nhóm có ≥ 3 sinh viên KHÁC NHAU hỏi;
// một người hỏi nhiều lần vẫn là một; nhóm dưới ngưỡng bỏ hẳn khỏi payload (không câu, không số lần).
func TestKnowledgeGapUncoveredMinStudents(t *testing.T) {
	cases := []struct {
		name  string
		asks  []int // chỉ số sinh viên hỏi cùng một câu (cách viết lệch hoa thường / khoảng trắng)
		count int   // 0 = không hiện
	}{
		{"1 sinh viên hỏi 5 lần", []int{0, 0, 0, 0, 0}, 0},
		{"2 sinh viên", []int{0, 1, 1, 0}, 0},
		{"đúng 3 sinh viên", []int{0, 1, 2}, 3},
		{"3 sinh viên, một người hỏi thêm", []int{0, 1, 2, 2, 2}, 5},
	}
	variants := []string{"Em ở KTX khu B xin hoãn bài được không", "  em ở ktx khu b   xin hoãn bài được không ", "Em ở KTX khu B xin hoãn bài được không"}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := newRig(t, 13)
			for k, i := range c.asks {
				r.ask(i, variants[k%len(variants)], true, false)
			}
			for k := range 40 { // đủ dữ liệu để chạy báo cáo
				r.ask(5+k%6, fmt.Sprintf("TCP bắt tay ba bước là gì, câu %d?", k), false, false)
			}
			_, err := r.svc.RunNow(t.Context(), r.teacher, r.course, 30)
			require.NoError(t, err)
			p := r.report()
			if c.count == 0 {
				require.Empty(t, p.Uncovered)
				return
			}
			require.Len(t, p.Uncovered, 1)
			require.Equal(t, c.count, p.Uncovered[0].Count)
		})
	}
}

// TestKnowledgeGapExamplesPerQuestion — P-11-32: câu mẫu của chủ đề chỉ gồm câu (chuẩn hoá) do ≥ 3 sinh viên KHÁC NHAU hỏi; cụm 5 sinh viên mà mỗi câu chỉ 1 người hỏi → 0 câu mẫu
// (chủ đề vẫn xếp hạng); câu do 3 sinh viên hỏi (lệch khoảng trắng) gộp một dòng.
func TestKnowledgeGapExamplesPerQuestion(t *testing.T) {
	r := newRig(t, 13)
	for k := range 5 { // cụm SQL: 5 sinh viên, mỗi câu một người hỏi một lần
		r.ask(k, fmt.Sprintf("Viết SQL join hai bảng thế nào, riêng em %d bị lỗi mã sinh viên", k), false, false)
	}
	for k := range 25 { // cụm TCP: 5 câu, mỗi câu 5 sinh viên hỏi — chỉ cần đủ dữ liệu; câu thứ 0 có lần viết thêm khoảng trắng
		text := fmt.Sprintf("TCP bắt tay ba bước là gì, câu %d?", k%5)
		if k%5 == 0 && k%2 == 0 {
			text = "       " + text // thêm 7 khoảng trắng (cùng cụm nhúng của stub): vẫn là một câu chuẩn hoá
		}
		r.ask(5+k/5, text, false, false)
	}
	_, err := r.svc.RunNow(t.Context(), r.teacher, r.course, 30)
	require.NoError(t, err)
	p := r.report()
	var sql, tcp *insight.Topic
	for i := range p.Topics {
		switch p.Topics[i].Signals.Questions {
		case 5:
			sql = &p.Topics[i]
		case 25:
			tcp = &p.Topics[i]
		}
	}
	require.NotNil(t, sql)
	require.Equal(t, 5, sql.Signals.Students)
	require.False(t, sql.FewAsked)
	require.Empty(t, sql.Examples, "5 sinh viên, mỗi câu một người → không câu mẫu")
	require.NotNil(t, tcp)
	require.NotEmpty(t, tcp.Examples)
	for _, e := range tcp.Examples {
		require.NotContains(t, e, "riêng em")
	}
	rep, err := r.svc.Latest(t.Context(), r.course)
	require.NoError(t, err)
	md, err := r.svc.Export(t.Context(), r.course, rep.Report.ID)
	require.NoError(t, err)
	require.NotContains(t, md, "riêng em")
}

func seedLeakRig(t *testing.T) *rig {
	r := newRig(t, 13)
	for k := range 5 { // cụm SQL: 5 sinh viên, mỗi câu một người → câu dưới ngưỡng k
		r.ask(k, fmt.Sprintf("Viết SQL join hai bảng thế nào, riêng em %d bị lỗi mã sinh viên", k), false, false)
	}
	for k := range 25 { // cụm TCP: mỗi câu 5 sinh viên → câu đạt ngưỡng
		r.ask(5+k/5, fmt.Sprintf("TCP bắt tay ba bước là gì, câu %d?", k%5), false, false)
	}
	return r
}

// TestKnowledgeGapPromptOnlyQualified — P-11-32 / FR-29: lời nhắc đặt tên chỉ có câu đạt ngưỡng k + từ khoá; câu dưới ngưỡng không vào.
func TestKnowledgeGapPromptOnlyQualified(t *testing.T) {
	r := seedLeakRig(t)
	_, err := r.svc.RunNow(t.Context(), r.teacher, r.course, 30)
	require.NoError(t, err)
	require.NotEmpty(t, r.llm.seen)
	all := strings.Join(r.llm.seen, "\n")
	require.NotContains(t, all, "riêng em")
	require.Contains(t, all, "TCP bắt tay ba bước là gì")
}

// TestKnowledgeGapNameGuard — FR-29: name chép ≥ 6 từ liên tiếp của câu dưới ngưỡng → «Chủ đề N», fallback, why / recommendation rỗng (khoá vẫn có); 5 từ → giữ; từ trùng chỉ ở câu đạt ngưỡng → giữ.
func TestKnowledgeGapNameGuard(t *testing.T) {
	cases := []struct {
		name, leak string
		replaced   bool
	}{
		{"đúng 6 từ liên tiếp", "Ôn: viết SQL join hai bảng thế nào, riêng", true},
		{"5 từ liên tiếp", "Ôn: viết SQL join hai bảng, riêng", false},
		{"từ trùng ở câu đạt ngưỡng", "Ôn: TCP bắt tay ba bước là gì", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := seedLeakRig(t)
			r.llm.leakName = c.leak
			_, err := r.svc.RunNow(t.Context(), r.teacher, r.course, 30)
			require.NoError(t, err)
			p := r.report()
			require.NotEmpty(t, p.Topics)
			for _, tp := range p.Topics {
				if c.replaced {
					require.Equal(t, fmt.Sprintf("Chủ đề %d", tp.Rank), tp.Name)
					require.Equal(t, "fallback", tp.NamedBy)
					require.Empty(t, tp.Why)
					require.Empty(t, tp.Recommendation)
				} else {
					require.Equal(t, c.leak, tp.Name)
					require.Equal(t, "llm", tp.NamedBy)
					require.NotEmpty(t, tp.Recommendation)
				}
			}
			rep, err := r.svc.Latest(t.Context(), r.course)
			require.NoError(t, err)
			md, err := r.svc.Export(t.Context(), r.course, rep.Report.ID)
			require.NoError(t, err)
			if c.replaced {
				require.NotContains(t, md, "riêng")
			}
		})
	}
}

// TestKnowledgeGapCrisisExcluded — P-11-34 F1: tin khủng hoảng (chat riêng) không bao giờ vào báo cáo: không nhúng ra ngoài, không từ khoá / tên chủ đề, không câu mẫu / «chưa đề cập» dù ≥ 3 sinh viên gõ cùng câu.
func TestKnowledgeGapCrisisExcluded(t *testing.T) {
	r := newRig(t, 13)
	const canary = "Em không muốn sống nữa, bỏ học luôn canary-khung-hoang"
	for k := range 4 { // 4 sinh viên cùng một câu khủng hoảng (qua ngưỡng k)
		r.ask(k, canary, true, false)
	}
	// câu trung tính nhưng trợ lý đã gắn intent CRISIS (phòng thủ ở SQL)
	const marked = "Dạo này em mệt lắm marker-intent-crisis"
	for k := range 4 {
		r.ask(4+k, marked, true, false)
	}
	r.exec(`update chat_messages set intent = 'CRISIS' where role = 'ASSISTANT' and reply_to in (select id from chat_messages where content = $1)`, marked)
	for k := range 40 {
		r.ask(k%6, fmt.Sprintf("TCP bắt tay ba bước là gì, câu %d?", k%5), false, false)
	}
	_, err := r.svc.RunNow(t.Context(), r.teacher, r.course, 30)
	require.NoError(t, err)
	var payload, all string
	require.NoError(t, r.pool.QueryRow(t.Context(), `select payload::text from insight_reports where course_id = $1`, r.course).Scan(&payload))
	all = strings.Join(append(append([]string{payload}, r.llm.embedded...), r.llm.seen...), "\n")
	for _, canaryText := range []string{"canary-khung-hoang", "sống nữa", "marker-intent-crisis", "canary", "marker"} {
		require.NotContains(t, strings.ToLower(all), strings.ToLower(canaryText), canaryText)
	}
	rep, err := r.svc.Latest(t.Context(), r.course)
	require.NoError(t, err)
	md, err := r.svc.Export(t.Context(), r.course, rep.Report.ID)
	require.NoError(t, err)
	require.NotContains(t, md, "canary")
	require.Equal(t, 40, rep.Report.Payload.Counts.Questions, "chỉ 40 câu thường được đếm")
}

// TestKnowledgeGapOldPayloadIgnored — P-11-34 F2: báo cáo cũ (payload không có schema = v1, sinh trước P-11-31/32/34) → 404 ở latest và export, không dùng lại ở Generate; chạy báo cáo mới thay được.
func TestKnowledgeGapOldPayloadIgnored(t *testing.T) {
	r := newRig(t, 13)
	for k := range 40 {
		r.ask(k%6, fmt.Sprintf("TCP bắt tay ba bước là gì, câu %d?", k%5), false, false)
	}
	var oldID uuid.UUID
	require.NoError(t, r.pool.QueryRow(t.Context(), `insert into insight_reports (course_id, kind, period_from, period_to, payload, data_watermark, generated_by)
		values ($1, 'KNOWLEDGE_GAP', current_date - 30, current_date, '{"topics":[{"rank":1,"name":"cũ","why":"x","examples":["câu nguyên văn của một sinh viên"],"keywords":[],"signals":{}}],"uncovered":[{"text":"câu lẻ","count":1}],"counts":{"questions":40,"students":6,"clusters":1},"period_days":30}'::jsonb,
		now() + interval '1 hour', $2) returning id`, r.course, r.teacher).Scan(&oldID))

	_, err := r.svc.Latest(t.Context(), r.course)
	var ae *apierr.Error
	require.ErrorAs(t, err, &ae)
	require.Equal(t, http.StatusNotFound, ae.Status)
	_, err = r.svc.Export(t.Context(), r.course, oldID)
	require.ErrorAs(t, err, &ae)
	require.Equal(t, http.StatusNotFound, ae.Status)
	out, err := r.svc.Generate(t.Context(), r.teacher, r.course, 30)
	require.NoError(t, err)
	require.False(t, out.Reused, "báo cáo cũ không được dùng lại dù không có dữ liệu mới")
	require.NotNil(t, out.JobID)

	_, err = r.svc.RunNow(t.Context(), r.teacher, r.course, 30)
	require.NoError(t, err)
	rep, err := r.svc.Latest(t.Context(), r.course)
	require.NoError(t, err)
	require.Equal(t, insight.PayloadSchema, rep.Report.Payload.Schema)
	require.NotEqual(t, oldID, rep.Report.ID)
	require.Nil(t, rep.Previous, "so sánh không lấy tên chủ đề từ báo cáo cũ")
	require.NotContains(t, rep.Report.Payload.Topics[0].Name, "cũ")
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

	// k = 3 theo từng câu (P-11-32): A, B có câu mẫu (mỗi câu ≥ 3 sinh viên hỏi); C (2 sinh viên) không có câu mẫu + nhãn
	require.GreaterOrEqual(t, len(p.Topics[0].Examples), 1)
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

	// ngoài tài liệu: 10 tin = 3 nhóm (3 SV / 4 SV / 1 SV × 3) → hiện 2 nhóm (4 lần trước, 3 lần sau), nhóm 1 sinh viên không có
	require.Len(t, p.Uncovered, 2)
	require.Equal(t, 4, p.Uncovered[0].Count)
	require.Equal(t, 3, p.Uncovered[1].Count)
	raw0, _ := json.Marshal(p.Uncovered)
	require.NotContains(t, string(raw0), "KTX")

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

	// kỳ ngoài {7, 14, 30, 120} → 422 (Q-E-7), kể cả 0 (P-11-35); «vắng = 14» do handler (contract)
	for _, bad := range []int{-1, 0, 5, 999} {
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

// TestKnowledgeGapPIICountOnNameCall — TC-04-15 / AC13: số thông tin cá nhân đã che ở bước thu thập đi vào lời gọi INSIGHT (llm_audit.pii_masked_count), không còn 0.
func TestKnowledgeGapPIICountOnNameCall(t *testing.T) {
	r := newRig(t, 14)
	r.svc.Det = &privacy.Detector{Roster: &privacy.Roster{Src: privacy.StoreRoster{Pool: r.pool}}}
	r.seed()
	for i := range 3 { // ba câu nêu họ tên bạn trong lớp: mỗi câu che 1 tên
		r.ask(i, fmt.Sprintf("TCP bắt tay ba bước: bạn Sinh Viên Số%d giải thích giúp mình với", 10+i), false, false)
	}
	_, err := r.svc.RunNow(t.Context(), r.teacher, r.course, 30)
	require.NoError(t, err)
	require.Len(t, r.llm.piiSeen, 1)
	require.GreaterOrEqual(t, r.llm.piiSeen[0], 3, "mỗi tên trong lớp đã che phải được đếm")
	for _, s := range r.llm.seen {
		require.NotContains(t, s, "Sinh Viên Số11")
	}
}

// P-11-48 / TC-04-11: nhúng chia lô ≤ 50 chuỗi và giãn nhịp giữa các lô (kỳ 120 ngày nhiều câu không dồn một lượt làm provider trả 429).
func TestKnowledgeGapEmbedBatches(t *testing.T) {
	r := newRig(t, 14)
	r.seed()
	const pause = 40 * time.Millisecond
	r.svc.EmbedPause = pause
	t0 := time.Now()
	_, err := r.svc.RunNow(t.Context(), r.teacher, r.course, 30)
	require.NoError(t, err)
	took := time.Since(t0)
	require.Greater(t, len(r.llm.batches), 1, "đủ nhiều câu để phải chia lô")
	for _, n := range r.llm.batches {
		require.LessOrEqual(t, n, insight.EmbedBatch)
	}
	require.GreaterOrEqual(t, took, time.Duration(len(r.llm.batches)-1)*pause, "giữa hai lô phải nghỉ")
}

// TestKnowledgeGapChapterNullCategory — DEF-GP10-05: tài liệu có `category` NULL (mặc định của mọi tài liệu thường) vẫn là ứng viên gán chương / tuần;
// bộ lọc «category <> 'THREAD_KNOWLEDGE'» loại cả chúng (NULL) và chỉ còn tài liệu có category → mọi chủ đề ra cùng một tên.
func TestKnowledgeGapChapterNullCategory(t *testing.T) {
	r := newRig(t, 13)
	ctx := t.Context()
	r.seed()
	var doc uuid.UUID
	require.NoError(t, r.pool.QueryRow(ctx, `insert into documents (course_id, title, type, status, week_no, sha256) values ($1, 'Giáo trình TCP', 'LECTURE', 'READY', 3, $2) returning id`, r.course, strings.Repeat("a", 64)).Scan(&doc))
	vec := make([]float32, llm.EmbedDims) // trục 0 = chủ đề TCP của stubLLM
	vec[0] = 1
	r.exec(`insert into content_chunks (document_id, course_ids, ord, text, embedding) values ($1, $2, 0, 'TCP', $3)`, doc, []uuid.UUID{r.course}, pgvector.NewVector(vec))
	_, err := r.svc.RunNow(ctx, r.teacher, r.course, 30)
	require.NoError(t, err)
	tcp := r.report().Topics[0]
	require.Equal(t, "Chủ đề TCP", tcp.Name)
	require.NotNil(t, tcp.Chapter)
	require.Equal(t, "Giáo trình TCP", *tcp.Chapter)
	require.NotNil(t, tcp.Week)
	require.Equal(t, 3, *tcp.Week)
}

// TestKnowledgeGapChapterFloor — DEF-GP10-05: chương / tuần chỉ gán khi tâm cụm đủ gần một đoạn tài liệu; cụm lạc đề (vuông góc mọi đoạn) để trống, không ép về tài liệu «gần nhất».
func TestKnowledgeGapChapterFloor(t *testing.T) {
	r := newRig(t, 13)
	ctx := t.Context()
	r.seed()
	var doc uuid.UUID
	require.NoError(t, r.pool.QueryRow(ctx, `insert into documents (course_id, title, type, status, week_no, sha256) values ($1, 'Giáo trình TCP', 'LECTURE', 'READY', 3, $2) returning id`, r.course, strings.Repeat("a", 64)).Scan(&doc))
	vec := make([]float32, llm.EmbedDims) // trục 0 = chủ đề TCP của stubLLM
	vec[0] = 1
	r.exec(`insert into content_chunks (document_id, course_ids, ord, text, embedding) values ($1, $2, 0, 'TCP', $3)`, doc, []uuid.UUID{r.course}, pgvector.NewVector(vec))
	_, err := r.svc.RunNow(ctx, r.teacher, r.course, 30)
	require.NoError(t, err)
	byName := map[string]insight.Topic{}
	for _, tp := range r.report().Topics {
		byName[tp.Name] = tp
	}
	tcp, sql := byName["Chủ đề TCP"], byName["Chủ đề SQL"]
	require.NotNil(t, tcp.Chapter)
	require.Equal(t, "Giáo trình TCP", *tcp.Chapter)
	require.NotNil(t, tcp.Week)
	require.Equal(t, 3, *tcp.Week)
	require.Nil(t, sql.Chapter, "cụm SQL vuông góc với đoạn duy nhất: không gán chương")
	require.Nil(t, sql.Week)
}
