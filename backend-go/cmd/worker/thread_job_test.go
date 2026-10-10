package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/edupilot/backend-go/internal/ingest"
	"github.com/edupilot/backend-go/internal/jobs"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/privacy"
	"github.com/edupilot/backend-go/internal/rag"
	"github.com/edupilot/backend-go/internal/testutil"
	"github.com/edupilot/backend-go/internal/thread"
)

type blockingLLM struct {
	llm.Client
	started chan struct{}
	release chan struct{}
}

func (b blockingLLM) Chat(ctx context.Context, _ llm.Request) (llm.Response, error) {
	close(b.started)
	select {
	case <-b.release:
	case <-ctx.Done():
	}
	return llm.Response{Text: "Trả lời [1]."}, nil
}

type oneHit struct{}

func (oneHit) SearchStudent(context.Context, rag.Query) ([]rag.Hit, error) {
	return []rag.Hit{{ChunkID: uuid.New(), DocumentID: uuid.New(), Title: "Quy chế", Text: "Nội dung.", Cosine: 0.9}}, nil
}

// TestRosterInvalidateNotBlockedByLongJob — US-P3-02 AC3 / SRS 4.9.5 (TLR-9): việc AI trả lời Threads (LLM treo) chạy ở consumer ep:ingest, KHÔNG ở consumer outbox tuần tự;
// handler `job.enqueue` của `thread.answer` chỉ XADD rồi trả ngay, và `course.member_changed` xoá ep:roster:{course} ≤ 5 s trong lúc lời gọi LLM còn treo.
func TestRosterInvalidateNotBlockedByLongJob(t *testing.T) {
	testutil.RequireContainers(t)
	d, _ := workerDeps(t)
	d.Cfg.DatabaseURL = testutil.MigratedPostgresURL(t)
	pool, err := pgxpool.New(t.Context(), d.Cfg.DatabaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	d.DB = pool
	ctx := t.Context()
	var teacher, course, author, threadID uuid.UUID
	mustExec := func(q string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, q, args...); err != nil {
			t.Fatal(err)
		}
	}
	scan := func(dst *uuid.UUID, q string, args ...any) {
		t.Helper()
		if err := pool.QueryRow(ctx, q, args...).Scan(dst); err != nil {
			t.Fatal(err)
		}
	}
	scan(&teacher, `insert into users (email, full_name, role) values ($1, 'GV', 'TEACHER') returning id`, uuid.NewString()+"@example.test")
	scan(&author, `insert into users (email, full_name, role) values ($1, 'SV', 'STUDENT') returning id`, uuid.NewString()+"@example.test")
	id := uuid.New()
	scan(&course, `insert into courses (subject_code, class_code, name, semester, join_code, created_by) values ('INT1006', $1, 'Lớp', '2026-2027-HK1', $2, $3) returning id`, "WK"+id.String()[:8], "ABCDEFG", teacher)
	scan(&threadID, `insert into forum_threads (course_id, author_id, title, body) values ($1, $2, 'Hỏi', 'Điều 5 nói gì?') returning id`, course, author)
	_ = mustExec

	reg := newRegistry(d)
	svc := jobs.NewService(pool)
	job, err := svc.Enqueue(ctx, author, thread.KindAnswer, map[string]any{"thread_id": threadID, "course_id": course})
	if err != nil {
		t.Fatal(err)
	}
	var payload []byte
	if err := pool.QueryRow(ctx, `select payload from outbox where topic=$1 order by created_at desc limit 1`, jobs.TopicEnqueue).Scan(&payload); err != nil {
		t.Fatal(err)
	}

	// consumer ep:ingest với LLM treo
	bl := blockingLLM{started: make(chan struct{}), release: make(chan struct{})}
	ts := &thread.Service{Pool: pool, Redis: d.Redis, Rag: oneHit{}, LLM: bl, Embed: func(context.Context, string) ([]float32, error) { return make([]float32, llm.EmbedDims), nil }, Log: d.Log}
	q := &ingest.Queue{P: &ingest.Processor{Set: ingest.Settings{Workers: 1, Reclaim: time.Second}, Log: d.Log}, Redis: d.Redis, Consumer: "t", Log: d.Log,
		Extra: map[string]func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) error{thread.KindAnswer: ts.AnswerHandler}}
	qctx, stop := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() { defer close(done); _ = q.Run(qctx) }()
	t.Cleanup(func() { close(bl.release); stop(); <-done })

	// handler outbox của việc: chỉ XADD, trả ngay (không chờ LLM)
	h, ok := reg.Lookup(jobs.TopicEnqueue)
	if !ok {
		t.Fatal("topic job.enqueue chưa đăng ký")
	}
	start := time.Now()
	if err := h(ctx, outbox.Message{ID: uuid.New(), Topic: jobs.TopicEnqueue, Payload: payload}); err != nil {
		t.Fatalf("job.enqueue: %v", err)
	}
	if time.Since(start) > 2*time.Second {
		t.Fatalf("handler job.enqueue chờ quá %v: phải chỉ xếp hàng", time.Since(start))
	}
	_ = job
	select {
	case <-bl.started: // LLM đang treo trong consumer ep:ingest
	case <-time.After(15 * time.Second):
		t.Fatal("consumer ep:ingest không nhận việc thread.answer")
	}

	// trong lúc LLM treo: xoá roster vẫn ≤ 5 s
	key := privacy.RosterKey(course)
	if err := d.Redis.Set(ctx, key, `[{"n":"Nguyễn Văn An"}]`, time.Hour).Err(); err != nil {
		t.Fatal(err)
	}
	inv, _ := reg.Lookup("course.member_changed")
	body, _ := json.Marshal(map[string]string{"course_id": course.String()})
	start = time.Now()
	if err := inv(ctx, outbox.Message{Topic: "course.member_changed", Payload: body}); err != nil {
		t.Fatal(err)
	}
	if n, _ := d.Redis.Exists(ctx, key).Result(); n != 0 {
		t.Fatal("khoá roster còn sau sự kiện")
	}
	if time.Since(start) > 5*time.Second {
		t.Fatalf("vô hiệu roster mất %v > 5 s khi hàng AI đang bận", time.Since(start))
	}
}
