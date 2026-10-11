package ingest_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/ingest"
	"github.com/edupilot/backend-go/internal/jobs"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/platform/clock"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/rag"
	"github.com/edupilot/backend-go/internal/testutil"
)

// fakeDocling cài các route của docling-serve; hành vi chỉnh qua trường của nó.
type fakeDocling struct {
	mu           sync.Mutex
	srv          *httptest.Server
	tasks        map[string]string // task_id → markdown trả về
	submits      []url2fields
	failMode     string // "", "down", "status_failure", "pagelimit", "task_failure", "lost"
	pass1, pass2 string
	submitDelay  time.Duration
	submitGate   chan struct{} // nếu có: submit chờ tới khi đóng (test tất định, không dựa vào thời gian)
	downFor      atomic.Int32  // số lần submit đầu trả 503
}

type url2fields map[string]string

func newDocling(t *testing.T) *fakeDocling {
	t.Helper()
	f := &fakeDocling{tasks: map[string]string{}}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/convert/file/async", func(w http.ResponseWriter, r *http.Request) {
		if f.downFor.Load() > 0 {
			f.downFor.Add(-1)
			http.Error(w, "down", http.StatusServiceUnavailable)
			return
		}
		if f.failMode == "down" {
			http.Error(w, "down", http.StatusServiceUnavailable)
			return
		}
		if err := r.ParseMultipartForm(64 << 20); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		fields := url2fields{}
		for k, v := range r.MultipartForm.Value {
			fields[k] = v[0]
		}
		file, _, _ := r.FormFile("files")
		if file != nil {
			_, _ = io.Copy(io.Discard, file)
		}
		time.Sleep(f.submitDelay)
		if f.submitGate != nil {
			<-f.submitGate
		}
		f.mu.Lock()
		id := fmt.Sprintf("task-%d", len(f.submits)+1)
		f.submits = append(f.submits, fields)
		md := f.pass1
		if fields["do_ocr"] == "true" {
			md = f.pass2
		}
		f.tasks[id] = md
		f.mu.Unlock()
		_ = json.NewEncoder(w).Encode(map[string]any{"task_id": id, "task_status": "pending"})
	})
	mux.HandleFunc("GET /v1/status/poll/{id}", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		_, ok := f.tasks[r.PathValue("id")]
		mode := f.failMode
		f.mu.Unlock()
		if !ok || mode == "lost" {
			http.NotFound(w, r)
			return
		}
		st := "success"
		if mode == "task_failure" {
			st = "failure"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"task_id": r.PathValue("id"), "task_status": st})
	})
	mux.HandleFunc("GET /v1/result/{id}", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		md := f.tasks[r.PathValue("id")]
		mode := f.failMode
		f.mu.Unlock()
		res := map[string]any{"document": map[string]any{"md_content": md}, "status": "success"}
		switch mode {
		case "status_failure":
			res["status"] = "failure"
			res["errors"] = []map[string]string{{"error_message": "Nguồn bị chặn"}}
		case "pagelimit":
			res["status"] = "failure"
			res["errors"] = []map[string]string{{"error_message": "Input document exceeds the maximum number of pages (400)"}}
		}
		_ = json.NewEncoder(w).Encode(res)
	})
	f.srv = httptest.NewServer(mux)
	t.Cleanup(f.srv.Close)
	return f
}

type memBlob struct {
	mu   sync.Mutex
	objs map[string][]byte
}

func (b *memBlob) Get(_ context.Context, key string) (io.ReadCloser, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	v, ok := b.objs[key]
	if !ok {
		return nil, errors.New("không có đối tượng")
	}
	return io.NopCloser(bytes.NewReader(v)), nil
}

// embedStub ghi lại từng lô và trả vectơ xác định; dims đổi được để thử ErrDimsMismatch.
type embedStub struct {
	mu      sync.Mutex
	batches []int
	lanes   []*llm.Lane
	idents  []llm.Identity
	dims    int
	fail    atomic.Int32 // số lần đầu trả lỗi tạm thời
}

func (e *embedStub) Chat(context.Context, llm.Request) (llm.Response, error) {
	return llm.Response{}, nil
}
func (e *embedStub) Stream(context.Context, llm.Request) (<-chan llm.Chunk, error) {
	return nil, errors.New("không dùng")
}
func (e *embedStub) Structured(context.Context, llm.Request, json.RawMessage) (json.RawMessage, error) {
	return nil, errors.New("không dùng")
}
func (e *embedStub) Embed(ctx context.Context, r llm.EmbedRequest) ([][]float32, error) {
	if e.fail.Load() > 0 {
		e.fail.Add(-1)
		return nil, &llm.ErrUnavailable{Reason: llm.ReasonAllFailed}
	}
	e.mu.Lock()
	e.batches = append(e.batches, len(r.Inputs))
	e.lanes = append(e.lanes, r.Lane)
	e.idents = append(e.idents, llm.IdentityFrom(ctx))
	dims := e.dims
	e.mu.Unlock()
	if dims == 0 {
		dims = llm.EmbedDims
	}
	out := make([][]float32, len(r.Inputs))
	for i, s := range r.Inputs {
		v := make([]float32, dims)
		v[len(s)%dims] = 1
		out[i] = v
	}
	return out, nil
}

type recPub struct {
	mu     sync.Mutex
	events []map[string]any
}

func (p *recPub) Publish(_ context.Context, _, typ string, data any) (string, error) {
	raw, _ := json.Marshal(data)
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	m["_type"] = typ
	p.mu.Lock()
	p.events = append(p.events, m)
	p.mu.Unlock()
	return "1-0", nil
}

type fx struct {
	pool    *pgxpool.Pool
	rdb     *appredis.Client
	dl      *fakeDocling
	blob    *memBlob
	emb     *embedStub
	pub     *recPub
	runner  *jobs.Runner
	svc     *jobs.Service
	proc    *ingest.Processor
	teacher uuid.UUID
	course  uuid.UUID
	logs    *bytes.Buffer
}

func newFx(t *testing.T) *fx {
	t.Helper()
	testutil.RequireContainers(t)
	ctx := t.Context()
	pool, err := pgxpool.New(ctx, testutil.MigratedPostgresURL(t))
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	rdb, err := appredis.New(ctx, testutil.RedisURL(t))
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })
	f := &fx{pool: pool, rdb: rdb, dl: newDocling(t), blob: &memBlob{objs: map[string][]byte{}}, emb: &embedStub{}, pub: &recPub{}, logs: &bytes.Buffer{}}
	log := slog.New(slog.NewJSONHandler(f.logs, nil))
	f.runner = jobs.NewRunner(pool, f.pub, clock.Real{}, log)
	f.svc = jobs.NewService(pool)
	f.dl.pass1 = strings.Repeat("Điều kiện bị cảnh báo học vụ được quy định tại Điều 5. ", 40)
	f.proc = &ingest.Processor{Pool: pool, Blob: f.blob, Docling: &ingest.Docling{BaseURL: f.dl.srv.URL, Poll: 5 * time.Millisecond}, LLM: f.emb, Jobs: f.runner, Log: log,
		Set:   ingest.Settings{MaxBytes: 50 << 20, MaxPages: 400, ChunkChars: 800, ExtractTimeout: time.Minute, Workers: 1, ClaimIdle: 5 * time.Minute, LeaseRenew: time.Minute, Reclaim: 50 * time.Millisecond, Retries: 3},
		Sleep: func(context.Context, time.Duration) {}}
	require.NoError(t, pool.QueryRow(ctx, `insert into users (email, full_name, role) values ($1, 'GV', 'TEACHER') returning id`, uuid.NewString()+"@example.test").Scan(&f.teacher))
	f.course = f.newCourse(t)
	return f
}

func (f *fx) newCourse(t *testing.T) uuid.UUID {
	t.Helper()
	id := uuid.New()
	const alpha = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	jc := make([]byte, 7)
	for i := range jc {
		jc[i] = alpha[int(id[i])%len(alpha)]
	}
	var c uuid.UUID
	require.NoError(t, f.pool.QueryRow(t.Context(), `insert into courses (subject_code, class_code, name, semester, join_code, created_by) values ('INT1006', $1, 'Lớp', '2026-2027-HK1', $2, $3) returning id`,
		"IG"+id.String()[:8], string(jc), f.teacher).Scan(&c))
	return c
}

type docSpec struct {
	course   uuid.UUID
	filename string
	typ      string
	visible  bool
	useRAG   bool
	body     []byte
	badHash  bool
}

// upload mô phỏng `complete`: object ở kho, dòng documents QUEUED + việc ingest cùng giao dịch.
func (f *fx) upload(t *testing.T, s docSpec) (doc uuid.UUID, job uuid.UUID) {
	t.Helper()
	if s.course == uuid.Nil {
		s.course = f.course
	}
	if s.filename == "" {
		s.filename = "bai-giang.pdf"
	}
	if s.typ == "" {
		s.typ = "LECTURE"
	}
	if s.body == nil {
		s.body = []byte("%PDF-1.4 " + uuid.NewString())
	}
	sum := sha256.Sum256(s.body)
	hash := hex.EncodeToString(sum[:])
	if s.badHash {
		hash = strings.Repeat("0", 64)
	}
	key := "courses/" + s.course.String() + "/documents/" + uuid.NewString() + "/" + s.filename
	f.blob.mu.Lock()
	f.blob.objs[key] = s.body
	f.blob.mu.Unlock()
	ctx := t.Context()
	tx, err := f.pool.Begin(ctx)
	require.NoError(t, err)
	require.NoError(t, tx.QueryRow(ctx, `insert into documents (course_id, title, type, filename, mime_type, size_bytes, sha256, blob_key, status, visible_to_students, use_for_rag, uploaded_by)
		values ($1, 'Tài liệu', $2::document_type, $3, 'application/pdf', $4, $5, $6, 'QUEUED', $7, $8, $9) returning id`,
		s.course, s.typ, s.filename, len(s.body), hash, key, s.visible, s.useRAG, f.teacher).Scan(&doc))
	j, err := f.svc.EnqueueTx(ctx, tx, f.teacher, ingest.KindIngest, ingest.KindPayload{DocumentID: doc})
	require.NoError(t, err)
	require.NoError(t, tx.Commit(ctx))
	return doc, j.ID
}

func (f *fx) status(t *testing.T, doc uuid.UUID) (status string, errMsg *string, pages *int32) {
	t.Helper()
	require.NoError(t, f.pool.QueryRow(t.Context(), `select status::text, error, page_count from documents where id=$1`, doc).Scan(&status, &errMsg, &pages))
	return
}

func (f *fx) jobState(t *testing.T, job uuid.UUID) (status string, progress int16, errJSON string) {
	t.Helper()
	var e []byte
	require.NoError(t, f.pool.QueryRow(t.Context(), `select status::text, progress, coalesce(error::text, '') from jobs where id=$1`, job).Scan(&status, &progress, &errJSON))
	_ = e
	return
}

func (f *fx) chunkCount(t *testing.T, doc uuid.UUID) (n, embedded int) {
	t.Helper()
	require.NoError(t, f.pool.QueryRow(t.Context(), `select count(*), count(embedding) from content_chunks where document_id=$1`, doc).Scan(&n, &embedded))
	return
}

func (f *fx) run(t *testing.T, doc, job uuid.UUID) {
	t.Helper()
	out, err := f.proc.Ingest(t.Context(), job, doc)
	require.NoError(t, err)
	_ = out
}

// TestIngestHappyPath — AC5: QUEUED → READY, đoạn có ord liên tục, page_count, job SUCCEEDED 100 % kèm document_id, outbox document.changed.
func TestIngestHappyPath(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	f.dl.pass1 = "# Quy chế\n\n" + strings.Repeat("Điều kiện bị cảnh báo học vụ được quy định tại Điều 5. ", 40) + ingest.PageMarker + "\n\n## Điều 6\n\n" + strings.Repeat("Sinh viên được phúc khảo trong 7 ngày. ", 30)
	doc, job := f.upload(t, docSpec{visible: true, useRAG: true})
	f.run(t, doc, job)

	st, errMsg, pages := f.status(t, doc)
	require.Equal(t, "READY", st)
	require.Nil(t, errMsg)
	require.NotNil(t, pages)
	require.Equal(t, int32(2), *pages)
	n, emb := f.chunkCount(t, doc)
	require.Positive(t, n)
	require.Equal(t, n, emb)
	var minOrd, maxOrd int
	require.NoError(t, f.pool.QueryRow(t.Context(), `select min(ord), max(ord) from content_chunks where document_id=$1`, doc).Scan(&minOrd, &maxOrd))
	require.Equal(t, 0, minOrd)
	require.Equal(t, n-1, maxOrd, "ord liên tục")
	var aud, ids string
	require.NoError(t, f.pool.QueryRow(t.Context(), `select string_agg(distinct audience::text, ','), string_agg(distinct course_ids::text, ',') from content_chunks where document_id=$1`, doc).Scan(&aud, &ids))
	require.Equal(t, "ALL", aud)
	require.Contains(t, ids, f.course.String())

	js, prog, _ := f.jobState(t, job)
	require.Equal(t, "SUCCEEDED", js)
	require.EqualValues(t, 100, prog)
	var outboxN int
	require.NoError(t, f.pool.QueryRow(t.Context(), `select count(*) from outbox where topic='document.changed' and payload->>'document_id'=$1`, doc.String()).Scan(&outboxN))
	require.Equal(t, 1, outboxN)

	// docling nhận đúng cấu hình PoC (lượt 1)
	require.Len(t, f.dl.submits, 1)
	s := f.dl.submits[0]
	require.Equal(t, "pypdfium2", s["pdf_backend"])
	require.Equal(t, "false", s["do_ocr"])
	require.Equal(t, "<!-- page -->", s["md_page_break_placeholder"])
	require.Equal(t, "md", s["to_formats"])
}

// TestIngestProgressMonotonic + TestIngestSSEProgress — AC5: tiến độ chỉ tăng 0→100, phát SSE job.progress, kết thúc có result.document_id.
func TestIngestProgressMonotonic(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	f.dl.pass1 = uniqueText(400) // nhiều đoạn → nhiều lô
	doc, job := f.upload(t, docSpec{visible: true, useRAG: true})
	f.run(t, doc, job)
	f.pub.mu.Lock()
	defer f.pub.mu.Unlock()
	last := -1
	var finalResult string
	for _, e := range f.pub.events {
		require.Equal(t, "job.progress", e["_type"])
		p := int(e["progress"].(float64))
		require.GreaterOrEqual(t, p, last, "tiến độ chỉ tăng")
		last = p
		if e["status"] == "SUCCEEDED" {
			finalResult = fmt.Sprint(e["result"])
		}
	}
	require.Equal(t, 100, last)
	require.Contains(t, finalResult, doc.String())
	require.Greater(t, len(f.pub.events), 3, "có nhiều mốc giai đoạn")
}

func TestIngestSSEProgress(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	doc, job := f.upload(t, docSpec{visible: true, useRAG: true})
	f.run(t, doc, job)
	f.pub.mu.Lock()
	defer f.pub.mu.Unlock()
	require.NotEmpty(t, f.pub.events)
	for _, e := range f.pub.events {
		require.Equal(t, job.String(), e["job_id"])
	}
}

// TestEmbedBatchesAndLane + TestEmbedInputIsHeadingPlusText(chunk_test) — AC7.
func TestEmbedBatchesAndLane(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	f.dl.pass1 = uniqueText(700)
	doc, job := f.upload(t, docSpec{visible: true, useRAG: true})
	f.run(t, doc, job)
	n, _ := f.chunkCount(t, doc)
	require.Greater(t, n, llm.EmbedBatch, "cần > 100 đoạn để thấy chia lô")
	total := 0
	for i, b := range f.emb.batches {
		require.LessOrEqual(t, b, llm.EmbedBatch)
		total += b
		require.Nil(t, f.emb.lanes[i], "không đặt làn = BATCH")
		require.NotNil(t, f.emb.idents[i].CourseID, "gắn lớp để hook che chạy")
		require.Equal(t, f.course, *f.emb.idents[i].CourseID)
	}
	require.Equal(t, n, total)
}

func TestEmbedSkippedWhenNotForRAG(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	doc, job := f.upload(t, docSpec{visible: true, useRAG: false})
	f.run(t, doc, job)
	st, _, _ := f.status(t, doc)
	require.Equal(t, "READY", st)
	n, emb := f.chunkCount(t, doc)
	require.Positive(t, n)
	require.Zero(t, emb)
	require.Empty(t, f.emb.batches)
}

func TestEmbedDimsMismatchFails(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	f.emb.dims = 768
	doc, job := f.upload(t, docSpec{visible: true, useRAG: true})
	f.run(t, doc, job)
	st, msg, _ := f.status(t, doc)
	require.Equal(t, "FAILED", st)
	require.Contains(t, *msg, "Chưa lập chỉ mục")
	_, _, e := f.jobState(t, job)
	require.Contains(t, e, ingest.CodeEmbedFailed)
	n, _ := f.chunkCount(t, doc)
	require.Zero(t, n)
}

// TestAudienceRule (phần ingest) — AC8: audience theo loại và cờ lúc ghi.
func TestIngestAudience(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	for _, c := range []struct {
		typ     string
		visible bool
		want    string
	}{{"LECTURE", true, "ALL"}, {"LECTURE", false, "STAFF"}, {"ANSWER_KEY", false, "GRADING"}} {
		doc, job := f.upload(t, docSpec{typ: c.typ, visible: c.visible, useRAG: true})
		f.run(t, doc, job)
		var aud string
		require.NoError(t, f.pool.QueryRow(t.Context(), `select string_agg(distinct audience::text, ',') from content_chunks where document_id=$1`, doc).Scan(&aud))
		require.Equal(t, c.want, aud, c.typ)
	}
}

// TestIngestRedeliveryNoDuplicateChunks + TestIngestNoopWhenReady — AC9.
func TestIngestRedeliveryNoDuplicateChunks(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	doc, job := f.upload(t, docSpec{visible: true, useRAG: true})
	f.run(t, doc, job)
	n1, _ := f.chunkCount(t, doc)
	f.run(t, doc, job) // giao lại sau READY
	n2, _ := f.chunkCount(t, doc)
	require.Equal(t, n1, n2)
	require.Len(t, f.dl.submits, 1, "no-op: không gọi docling lần hai")
	js, _, _ := f.jobState(t, job)
	require.Equal(t, "SUCCEEDED", js)

	// buộc đọc lại (QUEUED): xoá cũ + chèn mới, không nhân đôi
	_, err := f.pool.Exec(t.Context(), `update documents set status='QUEUED' where id=$1`, doc)
	require.NoError(t, err)
	f.run(t, doc, job)
	n3, _ := f.chunkCount(t, doc)
	require.Equal(t, n1, n3)
}

// TestJobNotSucceededWhileProcessing + TestOneProcessingPerCourse + TestIngestKillWorkerMidRun — AC9.
func TestOneProcessingPerCourse(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	a, jobA := f.upload(t, docSpec{visible: true, useRAG: true})
	b, jobB := f.upload(t, docSpec{visible: true, useRAG: true})
	// giả lập A đang PROCESSING (thuê còn hạn): B không nhận được và job B KHÔNG bị đóng
	_, err := f.pool.Exec(t.Context(), `update documents set status='PROCESSING', updated_at=now() where id=$1`, a)
	require.NoError(t, err)
	out, err := f.proc.Ingest(t.Context(), jobB, b)
	require.NoError(t, err)
	require.NotEqual(t, 0, int(out), "bận → giữ tin")
	st, _, _ := f.status(t, b)
	require.Equal(t, "QUEUED", st)
	js, _, _ := f.jobState(t, jobB)
	require.NotEqual(t, "SUCCEEDED", js, "không có job SUCCEEDED khi tài liệu chưa READY")
	// lớp khác không bị chặn
	other := f.newCourse(t)
	c, jobC := f.upload(t, docSpec{course: other, visible: true, useRAG: true})
	f.run(t, c, jobC)
	st, _, _ = f.status(t, c)
	require.Equal(t, "READY", st)
	_ = jobA
}

func TestJobNotSucceededWhileProcessing(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	doc, job := f.upload(t, docSpec{visible: true, useRAG: true})
	_, err := f.pool.Exec(t.Context(), `update documents set status='PROCESSING', updated_at=now() where id=$1`, doc)
	require.NoError(t, err)
	out, err := f.proc.Ingest(t.Context(), job, doc) // một lượt khác đang giữ tài liệu
	require.NoError(t, err)
	require.NotEqual(t, 0, int(out))
	js, _, _ := f.jobState(t, job)
	require.NotEqual(t, "SUCCEEDED", js)
	require.NotEqual(t, "FAILED", js)
}

func TestIngestKillWorkerMidRun(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	doc, job := f.upload(t, docSpec{visible: true, useRAG: true})
	// worker chết sau khi nhận: PROCESSING với nhịp cuối cách đây 6 phút → quá hạn thuê 5 phút → lượt sau nhận lại và hoàn tất
	backdate(t, f, doc, "6 minutes")
	f.run(t, doc, job)
	st, _, _ := f.status(t, doc)
	require.Equal(t, "READY", st)
}

// backdate lùi nhịp cuối của tài liệu: trigger set_updated_at ghi đè updated_at ở mọi UPDATE nên phải tắt nó trong giao dịch.
func backdate(t *testing.T, f *fx, doc uuid.UUID, ago string) {
	t.Helper()
	tx, err := f.pool.Begin(t.Context())
	require.NoError(t, err)
	_, err = tx.Exec(t.Context(), `alter table documents disable trigger documents_set_updated_at`)
	require.NoError(t, err)
	_, err = tx.Exec(t.Context(), fmt.Sprintf(`update documents set status='PROCESSING', updated_at = now() - interval '%s' where id=$1`, ago), doc)
	require.NoError(t, err)
	_, err = tx.Exec(t.Context(), `alter table documents enable trigger documents_set_updated_at`)
	require.NoError(t, err)
	require.NoError(t, tx.Commit(t.Context()))
}

func TestIngestLeaseRenewed(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	f.proc.Set.LeaseRenew = 30 * time.Millisecond
	f.dl.submitGate = make(chan struct{}) // docling "đọc" tới khi test nhả: không phụ thuộc tốc độ máy
	doc, job := f.upload(t, docSpec{visible: true, useRAG: true})
	done := make(chan struct{})
	go func() { defer close(done); _, _ = f.proc.Ingest(context.Background(), job, doc) }()
	updated := func() (st string, at time.Time) {
		require.NoError(t, f.pool.QueryRow(t.Context(), `select status::text, updated_at from documents where id=$1`, doc).Scan(&st, &at))
		return
	}
	require.Eventually(t, func() bool { st, _ := updated(); return st == "PROCESSING" }, 10*time.Second, 5*time.Millisecond)
	_, first := updated()
	require.Eventually(t, func() bool { _, at := updated(); return at.After(first) }, 10*time.Second, 5*time.Millisecond, "nhịp gia hạn đẩy updated_at lên khi đang đọc")
	close(f.dl.submitGate)
	<-done
}

// TestIngestDoclingDownKeepsQueued — SRS 3.3 / AC10 / AC17: docling tắt → trong lúc chờ tài liệu ở QUEUED (không PROCESSING, không FAILED), chờ đúng 5 s, 30 s, 2 phút; hết lượt mới FAILED.
func TestIngestDoclingDownKeepsQueued(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	f.dl.failMode = "down"
	doc, job := f.upload(t, docSpec{visible: true, useRAG: true})
	var waits []time.Duration
	var during []string
	f.proc.Sleep = func(_ context.Context, d time.Duration) {
		waits = append(waits, d)
		st, _, _ := f.status(t, doc) //nolint:contextcheck // trạng thái ghi lại lúc chờ; ctx của test, không phải của lần thử
		during = append(during, st)
	}
	f.run(t, doc, job)
	require.Equal(t, []time.Duration{5 * time.Second, 30 * time.Second, 2 * time.Minute}, waits)
	require.Equal(t, []string{"QUEUED", "QUEUED", "QUEUED"}, during)
	st, _, _ := f.status(t, doc)
	require.Equal(t, "FAILED", st)
	require.Len(t, f.dl.submits, 0)
}

// TestIngestDoclingRecoversWhileQueued: docling về lại ở lần thử thứ ba → READY, không FAILED.
func TestIngestDoclingRecoversWhileQueued(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	f.dl.downFor.Store(2)
	doc, job := f.upload(t, docSpec{visible: true, useRAG: true})
	f.run(t, doc, job)
	st, _, _ := f.status(t, doc)
	require.Equal(t, "READY", st)
	js, _, _ := f.jobState(t, job)
	require.Equal(t, "SUCCEEDED", js)
}

// TestNotRetrievableBeforeReady — AC9: đoạn của tài liệu chưa READY không bao giờ ra ở truy xuất.
func TestNotRetrievableBeforeReady(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	doc, job := f.upload(t, docSpec{visible: true, useRAG: true})
	svc := &rag.Service{DB: f.pool}
	q := rag.Query{CourseID: f.course, Vec: vec(0), Text: "cảnh báo học vụ"}
	// chèn tay một đoạn có vectơ vào tài liệu còn QUEUED
	_, err := f.pool.Exec(t.Context(), `insert into content_chunks (document_id, course_ids, ord, text, embedding) values ($1, array[$2]::uuid[], 0, 'cảnh báo học vụ', $3::vector)`, doc, f.course, vecLit(0))
	require.NoError(t, err)
	hs, err := svc.SearchStudent(t.Context(), q)
	require.NoError(t, err)
	require.Empty(t, hs)
	_ = job
}

// uniqueText: n đoạn văn khác nhau (đoạn trùng bị bỏ khi chia).
func uniqueText(n int) string {
	var b strings.Builder
	for i := range n {
		fmt.Fprintf(&b, "Đoạn số %d nói về chủ đề %d với nhiều chữ nhiều câu. Câu kế tiếp rất dài và đầy đủ ý nghĩa cho đoạn %d. %s\n\n", i, i*7+1, i, strings.Repeat(fmt.Sprintf("từ%d ", i), 40))
	}
	return b.String()
}

func vec(i int) []float32 { v := make([]float32, rag.Dims); v[i] = 1; return v }

func vecLit(i int) string {
	parts := make([]string, rag.Dims)
	for k := range parts {
		parts[k] = "0"
	}
	parts[i] = "1"
	return "[" + strings.Join(parts, ",") + "]"
}

// TestIngestDoclingDownRetriesThenFails — AC10: lỗi tạm thời thử lại 3 lần rồi FAILED EXTRACT_UNAVAILABLE, câu đọc được, không lộ URL.
func TestIngestDoclingDownRetriesThenFails(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	f.dl.failMode = "down"
	doc, job := f.upload(t, docSpec{visible: true, useRAG: true})
	f.run(t, doc, job)
	st, msg, _ := f.status(t, doc)
	require.Equal(t, "FAILED", st)
	require.Equal(t, "Chưa đọc được tài liệu lúc này. Thử lại sau.", *msg)
	_, _, e := f.jobState(t, job)
	require.Contains(t, e, ingest.CodeExtractUnavailable)
	require.NotContains(t, *msg+e, f.dl.srv.URL)
	require.NotContains(t, *msg+e, "127.0.0.1")
	n, _ := f.chunkCount(t, doc)
	require.Zero(t, n)
}

func TestIngestRetrySucceedsAfterTransientFailure(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	f.dl.downFor.Store(2) // hai lần đầu 503, lần ba được
	doc, job := f.upload(t, docSpec{visible: true, useRAG: true})
	f.run(t, doc, job)
	st, _, _ := f.status(t, doc)
	require.Equal(t, "READY", st)
}

func TestIngestServerLostTaskIsTransient(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	f.dl.failMode = "lost" // poll trả 404: server khởi động lại giữa việc
	doc, job := f.upload(t, docSpec{visible: true, useRAG: true})
	f.run(t, doc, job)
	st, msg, _ := f.status(t, doc)
	require.Equal(t, "FAILED", st)
	require.Contains(t, *msg, "Thử lại sau")
	require.Len(t, f.dl.submits, 4, "một lần đầu + thử lại 3 lần (5 s, 30 s, 2 phút)")
}

// TestIngestStatusFailureInBody — AC5/AC10: task_status=success nhưng status trong thân là failure → FAILED.
func TestIngestStatusFailureInBody(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	f.dl.failMode = "status_failure"
	doc, job := f.upload(t, docSpec{visible: true, useRAG: true})
	f.run(t, doc, job)
	st, _, _ := f.status(t, doc)
	require.Equal(t, "FAILED", st)
	require.Len(t, f.dl.submits, 1, "lỗi cứng: không thử lại")
}

func TestIngestNoText(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	f.dl.pass1, f.dl.pass2 = "<!-- image -->", "abc"
	doc, job := f.upload(t, docSpec{visible: true, useRAG: true})
	f.run(t, doc, job)
	st, msg, _ := f.status(t, doc)
	require.Equal(t, "FAILED", st)
	require.Equal(t, "Không đọc được chữ trong tệp (có thể là bản quét). Hãy tải bản có chữ.", *msg)
	require.Len(t, f.dl.submits, 2, "bản scan: lượt 1 không OCR, lượt 2 có OCR")
	require.Equal(t, "true", f.dl.submits[1]["do_ocr"])
	require.Equal(t, "tesseract", f.dl.submits[1]["ocr_preset"])
	require.Equal(t, "vie", f.dl.submits[1]["ocr_lang"])
}

func TestIngestScanUsesOCRPass(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	f.dl.pass1 = "<!-- image -->\n\nvài chữ"
	f.dl.pass2 = strings.Repeat("Căn cứ Quy chế đào tạo, học vụ. ", 60)
	doc, job := f.upload(t, docSpec{visible: true, useRAG: true})
	f.run(t, doc, job)
	st, _, _ := f.status(t, doc)
	require.Equal(t, "READY", st)
	var hasHocVu int
	require.NoError(t, f.pool.QueryRow(t.Context(), `select count(*) from content_chunks where document_id=$1 and text ilike '%học vụ%'`, doc).Scan(&hasHocVu))
	require.Positive(t, hasHocVu)
}

func TestIngestTooManyPages(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	f.proc.Set.MaxPages = 3
	f.dl.pass1 = strings.Repeat(strings.Repeat("Nội dung một trang khá dài. ", 20)+ingest.PageMarker+"\n", 6)
	doc, job := f.upload(t, docSpec{visible: true, useRAG: true})
	f.run(t, doc, job)
	st, msg, _ := f.status(t, doc)
	require.Equal(t, "FAILED", st)
	require.Contains(t, *msg, "Tệp dài hơn 3 trang. Hãy tách nhỏ.") // SRS 3.3: số trang = DOC_MAX_PAGES (test đặt 3)
	_, _, e := f.jobState(t, job)
	require.Contains(t, e, ingest.CodeTooManyPages)
}

func TestIngestDoclingPageLimitMapsToTooManyPages(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	f.dl.failMode = "pagelimit"
	doc, job := f.upload(t, docSpec{visible: true, useRAG: true})
	f.run(t, doc, job)
	_, _, e := f.jobState(t, job)
	require.Contains(t, e, ingest.CodeTooManyPages)
}

func TestIngestHashMismatch(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	doc, job := f.upload(t, docSpec{visible: true, useRAG: true, badHash: true})
	f.run(t, doc, job)
	st, msg, _ := f.status(t, doc)
	require.Equal(t, "FAILED", st)
	require.Equal(t, "Tệp bị lỗi khi tải lên. Tải lại.", *msg)
	require.Len(t, f.dl.submits, 1, "không thử lại")
	n, _ := f.chunkCount(t, doc)
	require.Zero(t, n)
}

func TestIngestEmbedRetriesTransient(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	f.emb.fail.Store(2)
	doc, job := f.upload(t, docSpec{visible: true, useRAG: true})
	f.run(t, doc, job)
	st, _, _ := f.status(t, doc)
	require.Equal(t, "READY", st)
}

// TestIngestErrorMessagesReadable — AC10: documents.error ≤ 1000 ký tự, tiếng Việt, không URL / đường dẫn / dấu vết.
func TestIngestErrorMessagesReadable(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"down", "status_failure", "pagelimit", "task_failure", "lost"} {
		f := newFx(t)
		f.dl.failMode = mode
		doc, job := f.upload(t, docSpec{visible: true, useRAG: true})
		f.run(t, doc, job)
		st, msg, _ := f.status(t, doc)
		require.Equal(t, "FAILED", st, mode)
		require.NotNil(t, msg, mode)
		require.LessOrEqual(t, len([]rune(*msg)), 1000)
		for _, bad := range []string{"http", "127.0.0.1", "goroutine", ".go:", "/v1/", "Traceback"} {
			require.NotContains(t, *msg, bad, mode)
		}
	}
}

// TestReindexFromStoredText / TestReindexIdempotent — AC15: nhúng lại từ chữ đã lưu, không gọi docling.
func TestReindexFromStoredText(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	doc, job := f.upload(t, docSpec{visible: true, useRAG: false}) // đọc nhưng chưa nhúng
	f.run(t, doc, job)
	n, emb := f.chunkCount(t, doc)
	require.Positive(t, n)
	require.Zero(t, emb)
	submits := len(f.dl.submits)

	rj, err := f.svc.Enqueue(t.Context(), f.teacher, ingest.KindReindex, ingest.KindPayload{DocumentID: doc})
	require.NoError(t, err)
	require.NoError(t, f.proc.Reindex(t.Context(), rj.ID, doc))
	_, emb = f.chunkCount(t, doc)
	require.Equal(t, n, emb)
	require.Len(t, f.dl.submits, submits, "không gọi docling")
	js, _, _ := f.jobState(t, rj.ID)
	require.Equal(t, "SUCCEEDED", js)
	st, _, _ := f.status(t, doc)
	require.Equal(t, "READY", st, "reindex không đổi status")

	// lập chỉ mục lại lần hai: cùng kết quả, không nhân đôi
	require.NoError(t, f.proc.Reindex(t.Context(), rj.ID, doc))
	n2, emb2 := f.chunkCount(t, doc)
	require.Equal(t, n, n2)
	require.Equal(t, n, emb2)
}

func TestReindexIdempotent(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	doc, job := f.upload(t, docSpec{visible: true, useRAG: true})
	f.run(t, doc, job)
	var before string
	require.NoError(t, f.pool.QueryRow(t.Context(), `select string_agg(id::text, ',' order by ord) from content_chunks where document_id=$1`, doc).Scan(&before))
	rj, _ := f.svc.Enqueue(t.Context(), f.teacher, ingest.KindReindex, ingest.KindPayload{DocumentID: doc})
	require.NoError(t, f.proc.Reindex(t.Context(), rj.ID, doc))
	var after string
	require.NoError(t, f.pool.QueryRow(t.Context(), `select string_agg(id::text, ',' order by ord) from content_chunks where document_id=$1`, doc).Scan(&after))
	require.Equal(t, before, after, "đoạn cũ giữ nguyên id (chỉ cập nhật vectơ)")
}

func TestReindexNotReadyFails(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	doc, _ := f.upload(t, docSpec{visible: true, useRAG: true})
	rj, _ := f.svc.Enqueue(t.Context(), f.teacher, ingest.KindReindex, ingest.KindPayload{DocumentID: doc})
	require.NoError(t, f.proc.Reindex(t.Context(), rj.ID, doc))
	js, _, e := f.jobState(t, rj.ID)
	require.Equal(t, "FAILED", js)
	require.Contains(t, e, ingest.CodeNotReady)
}

func TestReindexAll(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	var docs []uuid.UUID
	for range 2 {
		doc, job := f.upload(t, docSpec{visible: true, useRAG: true})
		f.run(t, doc, job)
		docs = append(docs, doc)
	}
	before := len(f.emb.batches)
	rj, _ := f.svc.Enqueue(t.Context(), f.teacher, ingest.KindReindexAll, ingest.KindPayload{CourseID: f.course})
	require.NoError(t, f.proc.ReindexAll(t.Context(), rj.ID, f.course, f.teacher))
	require.Greater(t, len(f.emb.batches), before)
	js, _, _ := f.jobState(t, rj.ID)
	require.Equal(t, "SUCCEEDED", js)
	_ = docs
}

// TestQueueEndToEnd: job.enqueue → handler chỉ XADD (ErrDeferred) → consumer ep:ingest chạy → job SUCCEEDED; job không SUCCEEDED trước khi consumer xong.
func TestQueueEndToEnd(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	ingest.RegisterKinds(f.runner, f.rdb)
	doc, job := f.upload(t, docSpec{visible: true, useRAG: true})
	payload, _ := json.Marshal(map[string]any{"job_id": job, "kind": ingest.KindIngest, "payload": ingest.KindPayload{DocumentID: doc}})
	require.NoError(t, f.runner.HandleMessage(t.Context(), outboxMsg(payload)))
	js, _, _ := f.jobState(t, job)
	require.Equal(t, "RUNNING", js, "handler outbox chỉ chuyển việc, chưa đóng")

	q := &ingest.Queue{P: f.proc, Redis: f.rdb, Consumer: "t-" + uuid.NewString()[:6], Log: slog.New(slog.NewJSONHandler(io.Discard, nil))}
	// stream dùng chung giữa các test: chỉ xử lý việc của mình — các tin khác (của test khác) có tài liệu không tồn tại ở DB này sẽ bị bỏ qua / Abort
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go func() { _ = q.Run(ctx) }()
	require.Eventually(t, func() bool {
		s, _, _ := f.jobState(t, job)
		return s == "SUCCEEDED"
	}, 20*time.Second, 50*time.Millisecond)
	st, _, _ := f.status(t, doc)
	require.Equal(t, "READY", st)
}

// TestDocumentChangedNotBlockedByIngest — AC14: kích hoạt ingest qua outbox chỉ XADD (không chờ docling), nên việc ngắn khác không bị trễ.
func TestDocumentChangedNotBlockedByIngest(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	ingest.RegisterKinds(f.runner, f.rdb)
	f.dl.submitDelay = 3 * time.Second // docling chậm: nếu handler outbox gọi nó thì mất ≥ 3 s
	doc, job := f.upload(t, docSpec{visible: true, useRAG: true})
	payload, _ := json.Marshal(map[string]any{"job_id": job, "kind": ingest.KindIngest, "payload": ingest.KindPayload{DocumentID: doc}})
	start := time.Now()
	require.NoError(t, f.runner.HandleMessage(t.Context(), outboxMsg(payload)))
	require.Less(t, time.Since(start), time.Second)
}
