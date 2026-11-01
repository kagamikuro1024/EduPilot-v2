package grading_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/grading"
	"github.com/edupilot/backend-go/internal/jobs"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/store"
	"github.com/edupilot/backend-go/internal/testutil"
)

const fixDir = "../../../docs/specs/FEAT-gradebook/fixtures/"

// fakeLLM là Structurer giả: ghi lại mọi lời gọi, trả theo `fn` (mặc định: bản ghi của fixture).
type fakeLLM struct {
	mu    sync.Mutex
	calls int
	reqs  []llm.Request
	ctxs  []context.Context
	fn    func(n int, ctx context.Context) (json.RawMessage, error)
	via   func(r llm.Request, schema json.RawMessage) (json.RawMessage, error) // khác nil: thay fn, nhìn thấy nội dung yêu cầu
}

func (f *fakeLLM) Structured(ctx context.Context, r llm.Request, schema json.RawMessage) (json.RawMessage, error) {
	f.mu.Lock()
	f.calls++
	n := f.calls
	f.reqs = append(f.reqs, r)
	f.ctxs = append(f.ctxs, ctx)
	f.mu.Unlock()
	if f.via != nil {
		return f.via(r, schema)
	}
	return f.fn(n, ctx)
}

func (f *fakeLLM) count() int { f.mu.Lock(); defer f.mu.Unlock(); return f.calls }

// recorded: bản ghi sẵn của provider giả = fixture bỏ các mục unclear_items do CODE thêm (README fixtures).
func recorded(t *testing.T, name string, mutate func(m map[string]any)) json.RawMessage {
	t.Helper()
	b, err := os.ReadFile(fixDir + name)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(b, &m))
	var keep []any
	for _, u := range m["unclear_items"].([]any) {
		if um := u.(map[string]any); um["source"] != "CODE" {
			keep = append(keep, map[string]any{"message": um["message"], "target": um["target"]})
		}
	}
	if keep == nil {
		keep = []any{}
	}
	m["unclear_items"] = keep
	if mutate != nil {
		mutate(m)
	}
	out, err := json.Marshal(m)
	require.NoError(t, err)
	return out
}

func constLLM(out json.RawMessage) *fakeLLM {
	return &fakeLLM{fn: func(int, context.Context) (json.RawMessage, error) { return out, nil }}
}

type env struct {
	t       *testing.T
	ctx     context.Context
	pool    *pgxpool.Pool
	q       *store.Queries
	teacher uuid.UUID
	course  uuid.UUID
	svc     *grading.Service
	fake    *fakeLLM
}

func newEnv(t *testing.T, f *fakeLLM) *env {
	t.Helper()
	testutil.RequireContainers(t)
	return newEnvPool(t, f, testutil.RuntimePool(t))
}

func newEnvPool(t *testing.T, f *fakeLLM, pool *pgxpool.Pool) *env {
	t.Helper()
	e := &env{t: t, ctx: t.Context(), pool: pool, q: store.New(pool), fake: f}
	e.teacher = e.user("TEACHER", "Giảng Viên")
	e.course = e.newCourse()
	e.svc = &grading.Service{Pool: pool, Jobs: jobs.NewService(pool), Log: slog.New(slog.NewTextHandler(io.Discard, nil)), Cfg: grading.Config{RetryWait: []time.Duration{0}}}
	if f != nil {
		e.svc.LLM = f
	}
	return e
}

func (e *env) exec(sql string, args ...any) {
	e.t.Helper()
	_, err := e.pool.Exec(e.ctx, sql, args...)
	require.NoError(e.t, err, sql)
}

func (e *env) one(dst any, sql string, args ...any) {
	e.t.Helper()
	require.NoError(e.t, e.pool.QueryRow(e.ctx, sql, args...).Scan(dst), sql)
}

func (e *env) user(role, name string) uuid.UUID {
	e.t.Helper()
	var id uuid.UUID
	e.one(&id, `insert into users (email, full_name, role) values ($1, $2, $3::user_role) returning id`, strings.ToLower(role)+"."+uuid.NewString()[:8]+"@example.test", name, role)
	return id
}

func (e *env) newCourse() uuid.UUID {
	e.t.Helper()
	id := uuid.New()
	const alpha = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	b := make([]byte, 7)
	for i := range b {
		b[i] = alpha[int(id[i])%len(alpha)]
	}
	var c uuid.UUID
	e.one(&c, `insert into courses (subject_code, class_code, name, semester, join_code, created_by) values ('INT1006', $1, 'An ninh mạng', '2026-2027-HK1', $2, $3) returning id`, "G-"+id.String()[:8], string(b), e.teacher)
	e.exec(`insert into enrollments (course_id, user_id, role_in_course, status, joined_via) values ($1, $2, 'TEACHER', 'ACTIVE', 'ADMIN')`, c, e.teacher)
	return c
}

var pageRe = regexp.MustCompile(`<!-- trang (\d+) -->`)

// policyDoc nạp một COURSE_POLICY READY từ fixtures/*.md: mỗi "<!-- trang N -->" là một trang, mỗi đoạn văn một chunk (page_no, ord).
func (e *env) policyDoc(md string) uuid.UUID {
	e.t.Helper()
	raw, err := os.ReadFile(fixDir + md)
	require.NoError(e.t, err)
	var id uuid.UUID
	e.one(&id, `insert into documents (course_id, title, type, status, page_count) values ($1, $2, 'COURSE_POLICY', 'READY', 2) returning id`, e.course, md)
	parts := pageRe.Split(string(raw), -1)
	nums := pageRe.FindAllStringSubmatch(string(raw), -1)
	ord := 0
	for i, n := range nums {
		page := atoi(n[1])
		for _, para := range strings.Split(parts[i+1], "\n\n") {
			para = strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(para), "#"))
			if para == "" {
				continue
			}
			e.exec(`insert into content_chunks (document_id, course_ids, ord, page_no, text) values ($1, array[$2::uuid], $3, $4, $5)`, id, e.course, ord, page, para)
			ord++
		}
	}
	return id
}

func atoi(s string) int {
	n := 0
	for _, c := range s {
		n = n*10 + int(c-'0')
	}
	return n
}

// enqueue xếp việc thật (jobs + outbox) và trả id + thân việc, để Run chạy trực tiếp như worker.
func (e *env) enqueue(p grading.Payload) (uuid.UUID, json.RawMessage) {
	e.t.Helper()
	j, err := e.svc.Jobs.Enqueue(e.ctx, e.teacher, grading.KindExtract, p)
	require.NoError(e.t, err)
	raw, _ := json.Marshal(p)
	return j.ID, raw
}

func (e *env) run(doc uuid.UUID, replace bool, draftVersion int) (any, error) {
	e.t.Helper()
	id, raw := e.enqueue(grading.Payload{CourseID: e.course, DocumentID: doc, ReplaceDraft: replace, DraftVersion: draftVersion})
	return e.svc.Run(e.ctx, e.teacher, id, raw, func(int) {})
}

type unclearRow struct {
	Source, Code, Target string
}

// draftUnclear đọc mục chưa rõ đã lưu của bản nháp hiện tại.
func (e *env) draftUnclear() []unclearRow {
	e.t.Helper()
	d, err := e.q.GradeDraftGet(e.ctx, e.course)
	require.NoError(e.t, err)
	var x struct {
		Unclear []unclearRow `json:"unclear_items"`
	}
	require.NoError(e.t, json.Unmarshal(d.Extraction, &x))
	return x.Unclear
}
