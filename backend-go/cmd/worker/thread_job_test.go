package main

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/exam"
	"github.com/edupilot/backend-go/internal/ingest"
	"github.com/edupilot/backend-go/internal/jobs"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/platform/clock"
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
	once    *sync.Once
}

func (b blockingLLM) Chat(ctx context.Context, _ llm.Request) (llm.Response, error) {
	b.once.Do(func() { close(b.started) })
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
	bl := blockingLLM{started: make(chan struct{}), release: make(chan struct{}), once: new(sync.Once)}
	ts := &thread.Service{Pool: pool, Redis: d.Redis, Rag: oneHit{}, LLM: bl, Embed: func(context.Context, string) ([]float32, error) { return make([]float32, llm.EmbedDims), nil }, Log: d.Log}
	group := "t-" + uuid.NewString() // nhóm riêng, đọc từ cuối: không nuốt tin tồn của gói khác trên Redis dùng chung
	if err := d.Redis.XGroupCreateMkStream(ctx, ingest.StreamName, group, "$").Err(); err != nil {
		t.Fatal(err)
	}
	q := &ingest.Queue{P: &ingest.Processor{Pool: pool, Set: ingest.Settings{Workers: 1, Reclaim: time.Second}, Log: d.Log}, Redis: d.Redis, Consumer: "t", Log: d.Log, Group: group,
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

type noLock struct{}

func (noLock) IsLocked(context.Context, uuid.UUID) (exam.Lock, bool, error) {
	return exam.Lock{}, false, nil
}
func (noLock) RecordChatBlocked(context.Context, uuid.UUID, uuid.UUID) error { return nil }

// TestThreadPostToAIAnswer — QC BUG-1 của US-P3-06: đi từ POST thread (thread.Service.Create) tới bài AI qua ĐÚNG bảng handler của worker:
// mọi topic outbox mà đăng thread sinh ra đều có handler (không dead-letter), handler chỉ xếp hàng, consumer ep:ingest tạo đúng một bài AI.
func TestThreadPostToAIAnswer(t *testing.T) {
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
	var teacher, course, author uuid.UUID
	scan := func(dst *uuid.UUID, q string, args ...any) {
		t.Helper()
		if err := pool.QueryRow(ctx, q, args...).Scan(dst); err != nil {
			t.Fatal(err)
		}
	}
	scan(&teacher, `insert into users (email, full_name, role) values ($1, 'GV', 'TEACHER') returning id`, uuid.NewString()+"@example.test")
	scan(&author, `insert into users (email, full_name, role) values ($1, 'SV', 'STUDENT') returning id`, uuid.NewString()+"@example.test")
	id := uuid.New()
	scan(&course, `insert into courses (subject_code, class_code, name, semester, join_code, created_by) values ('INT1006', $1, 'Lớp', '2026-2027-HK1', $2, $3) returning id`, "WP"+id.String()[:8], "ABCDEFG", teacher)

	det := &privacy.Detector{}
	ts := &thread.Service{Pool: pool, Redis: d.Redis, FW: &thread.Firewall{Detector: det, Classifier: &privacy.Classifier{Detector: det}}, Lock: noLock{}, Jobs: jobs.NewService(pool),
		Rag: oneHit{}, LLM: fakeAnswerLLM{}, Embed: func(context.Context, string) ([]float32, error) { return make([]float32, llm.EmbedDims), nil }, Clock: clock.Real{}, Log: d.Log}
	v, err := ts.Create(ctx, thread.Actor{UserID: author, Role: auth.RoleStudent}, course, thread.CreateIn{Title: "Điều 5", Body: "Quy chế thi lại ở Điều 5 nói gì về số lần thi?"})
	if err != nil {
		t.Fatal(err)
	}

	group := "t-" + uuid.NewString()
	if err := d.Redis.XGroupCreateMkStream(ctx, ingest.StreamName, group, "$").Err(); err != nil {
		t.Fatal(err)
	}
	q := &ingest.Queue{P: &ingest.Processor{Pool: pool, Set: ingest.Settings{Workers: 1, Reclaim: time.Second}, Log: d.Log}, Redis: d.Redis, Consumer: "t", Log: d.Log, Group: group,
		Extra: map[string]func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) error{thread.KindAnswer: ts.AnswerHandler}}
	qctx, stop := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() { defer close(done); _ = q.Run(qctx) }()
	t.Cleanup(func() { stop(); <-done })

	// giao mọi tin outbox của đăng thread qua bảng handler THẬT của worker
	reg := newRegistry(d)
	rows, err := pool.Query(ctx, `select id, topic, payload from outbox where payload::text like '%'||$1||'%' or topic=$2 order by created_at`, v.Thread.ID.String(), jobs.TopicEnqueue)
	if err != nil {
		t.Fatal(err)
	}
	var msgs []outbox.Message
	for rows.Next() {
		var m outbox.Message
		if err := rows.Scan(&m.ID, &m.Topic, &m.Payload); err != nil {
			t.Fatal(err)
		}
		msgs = append(msgs, m)
	}
	rows.Close()
	topics := map[string]bool{}
	for _, m := range msgs {
		topics[m.Topic] = true
		h, ok := reg.Lookup(m.Topic)
		if !ok {
			t.Fatalf("topic %q chưa có handler ở worker (sẽ dead-letter sau 4 lần)", m.Topic)
		}
		if err := h(ctx, m); err != nil {
			t.Fatalf("handler %s: %v", m.Topic, err)
		}
	}
	if !topics[thread.TopicCreated] || !topics[jobs.TopicEnqueue] {
		t.Fatalf("đăng thread phải sinh %s và %s, có %v", thread.TopicCreated, jobs.TopicEnqueue, topics)
	}
	deadline := time.Now().Add(20 * time.Second)
	for {
		var n int
		_ = pool.QueryRow(ctx, `select count(*) from forum_posts where thread_id=$1 and kind='AI' and verification_state='PENDING'`, v.Thread.ID).Scan(&n)
		if n == 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("không có bài AI sau 20 s")
		}
		time.Sleep(200 * time.Millisecond)
	}
	var state string
	_ = pool.QueryRow(ctx, `select ai_state::text from forum_threads where id=$1`, v.Thread.ID).Scan(&state)
	if state != "ANSWERED" {
		t.Fatalf("ai_state = %s", state)
	}
}

type fakeAnswerLLM struct{ llm.Client }

func (fakeAnswerLLM) Chat(context.Context, llm.Request) (llm.Response, error) {
	return llm.Response{Text: "Theo quy chế [1], sinh viên được thi lại một lần."}, nil
}
