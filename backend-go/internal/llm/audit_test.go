package llm_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/llm/provider"
)

func TestAuditRow(t *testing.T) {
	t.Parallel()
	uid := uuid.New()
	cid := uuid.New()
	ctx := auth.WithCourseAccess(auth.WithPrincipal(t.Context(), auth.Principal{Sub: uid.String(), Role: auth.RoleStudent}), auth.CourseAccess{CourseID: cid.String()})
	bad := &stub{name: "A", chat: func(int, provider.ChatOpts) (provider.Result, error) {
		return provider.Result{}, perr(provider.KindAuth, 401)
	}}
	g, cap := newGateway(t, bad, &stub{name: "B"})
	if _, err := g.Chat(ctx, llm.Request{Task: llm.TaskChat, PIIMaskedCount: 3, Messages: userMsg("nội dung bí mật của sinh viên")}); err != nil {
		t.Fatal(err)
	}
	// lỗi cũng đúng MỘT dòng
	lane := llm.LaneBatch
	bad2 := &stub{name: "X", chat: func(int, provider.ChatOpts) (provider.Result, error) {
		return provider.Result{}, perr(provider.KindRateLimit, 429)
	}}
	g2, cap2 := newGateway(t, bad2)
	_, _ = g2.Chat(ctx, llm.Request{Task: llm.TaskInsight, Lane: &lane, Messages: userMsg("x")})

	rows := cap.all(t, g)
	if len(rows) != 1 {
		t.Fatalf("số dòng = %d, muốn 1", len(rows))
	}
	r := rows[0]
	if r.Task != "CHAT" || r.Lane != "INTERACTIVE" || r.Provider != "B" || r.Model != "m-B" || r.FallbackIndex != 1 || r.Status != "ok" ||
		r.PIIMaskedCount != 3 || r.TokensIn != 10 || r.TokensOut != 5 || r.Attempts != 2 || r.UserID == nil || *r.UserID != uid ||
		r.CourseID == nil || *r.CourseID != cid || r.TraceID == "" || r.CostEst.IsZero() {
		t.Errorf("audit = %+v", r)
	}
	rows2 := cap2.all(t, g2)
	if len(rows2) != 1 || rows2[0].Status != "rate_limited" || rows2[0].ErrorKind != "RATE_LIMIT" || rows2[0].Attempts != 3 {
		t.Errorf("audit lỗi = %+v", rows2)
	}
}

func TestAuditAsync(t *testing.T) {
	t.Parallel()
	block := make(chan struct{})
	var mu sync.Mutex
	var got [][]llm.AuditRow
	a := llm.NewAuditor(context.Background(), func(_ context.Context, rows []llm.AuditRow) error {
		<-block
		mu.Lock()
		got = append(got, rows)
		mu.Unlock()
		return errors.New("hỏng ghi")
	}, newDiscardLog())
	defer func() { close(block); a.Close(context.Background()) }()

	start := time.Now()
	for range 250 {
		a.Record(llm.AuditRow{Task: "CHAT", Lane: "INTERACTIVE", Status: "ok", TraceID: "t"})
	}
	if d := time.Since(start); d > 100*time.Millisecond {
		t.Errorf("Record chặn %v dù writer bị treo — phải bất đồng bộ", d)
	}
	// lỗi ghi không làm hỏng người gọi: Gateway vẫn trả kết quả khi writer lỗi
	p := &stub{name: "A"}
	g, cap := newGateway(t, p)
	cap.fail.Store(true)
	if _, err := g.Chat(t.Context(), llm.Request{Task: llm.TaskChat, Messages: userMsg("x")}); err != nil {
		t.Fatalf("lỗi ghi audit làm hỏng lời gọi: %v", err)
	}
}

func TestAuditBatches(t *testing.T) {
	t.Parallel()
	cap := &capture{}
	a := llm.NewAuditor(context.Background(), cap.write, newDiscardLog())
	for range 100 { // đủ 100 dòng → đẩy ngay, không đợi 1 s
		a.Record(llm.AuditRow{Task: "CHAT", Lane: "BATCH", Status: "ok", TraceID: "t"})
	}
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		cap.mu.Lock()
		n := len(cap.rows)
		cap.mu.Unlock()
		if n == 100 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	cap.mu.Lock()
	defer cap.mu.Unlock()
	if len(cap.rows) != 100 || len(cap.batch) != 1 {
		t.Fatalf("rows=%d batches=%v, muốn 100 dòng trong 1 lô", len(cap.rows), cap.batch)
	}
	if a.Flushed() != 100 || a.Pending() != 0 || a.Dropped() != 0 {
		t.Errorf("flushed=%d pending=%d dropped=%d, muốn 100/0/0 (nguồn của stats.audit)", a.Flushed(), a.Pending(), a.Dropped())
	}
	a.Close(context.Background())
}

func TestAuditDropOldest(t *testing.T) {
	t.Parallel()
	block := make(chan struct{})
	var mu sync.Mutex
	var written []llm.AuditRow
	entered := make(chan struct{}, 1)
	a := llm.NewAuditor(context.Background(), func(_ context.Context, rows []llm.AuditRow) error {
		select {
		case entered <- struct{}{}:
		default:
		}
		<-block
		mu.Lock()
		written = append(written, rows...)
		mu.Unlock()
		return nil
	}, newDiscardLog())
	// 100 dòng đầu được lấy đi và writer bị treo; 1.400 dòng sau đổ vào đệm 1.000 → bỏ đúng 400 dòng CŨ nhất
	for i := range 100 {
		a.Record(llm.AuditRow{Task: "CHAT", Lane: "BATCH", Status: "ok", TraceID: "t", LatencyMS: i})
	}
	<-entered
	for i := 100; i < 1500; i++ {
		a.Record(llm.AuditRow{Task: "CHAT", Lane: "BATCH", Status: "ok", TraceID: "t", LatencyMS: i})
	}
	if a.Dropped() != 400 {
		t.Fatalf("dropped = %d, muốn 400", a.Dropped())
	}
	close(block)
	a.Close(context.Background())
	mu.Lock()
	defer mu.Unlock()
	if len(written) == 0 || written[len(written)-1].LatencyMS != 1499 {
		t.Fatalf("dòng mới nhất phải còn: %d dòng, cuối=%v", len(written), written[len(written)-1:])
	}
	if int64(len(written))+a.Dropped() != 1500 {
		t.Fatalf("ghi %d + bỏ %d ≠ 1500 (mất dòng không đếm)", len(written), a.Dropped())
	}
	seen := map[int]bool{}
	for _, r := range written {
		seen[r.LatencyMS] = true
	}
	for i := 100; i < 100+int(a.Dropped()); i++ { // sau lô đầu (100 dòng đã lấy ra), các dòng bị bỏ là dòng CŨ NHẤT còn trong đệm
		if seen[i] {
			t.Fatalf("dòng cũ %d lẽ ra đã bị bỏ", i)
		}
	}
}

func TestAuditFlushOnShutdown(t *testing.T) {
	t.Parallel()
	cap := &capture{}
	a := llm.NewAuditor(context.Background(), cap.write, newDiscardLog())
	for range 7 {
		a.Record(llm.AuditRow{Task: "CHAT", Lane: "BATCH", Status: "ok", TraceID: "t"})
	}
	a.Close(context.Background()) // trước khi tick 1 s
	cap.mu.Lock()
	defer cap.mu.Unlock()
	if len(cap.rows) != 7 {
		t.Fatalf("đẩy %d dòng khi tắt, muốn 7", len(cap.rows))
	}
	a.Close(context.Background()) // gọi lần hai an toàn
}
