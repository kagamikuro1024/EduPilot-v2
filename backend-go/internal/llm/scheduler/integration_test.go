//go:build integration

package scheduler_test

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"
	shopDec "github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/llm/budget"
	"github.com/edupilot/backend-go/internal/llm/fake"
	"github.com/edupilot/backend-go/internal/llm/scheduler"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/testutil"
)

func redisClient(t testing.TB, url string) *goredis.Client {
	t.Helper()
	opt, err := goredis.ParseURL(url)
	require.NoError(t, err)
	c := goredis.NewClient(opt)
	t.Cleanup(func() { _ = c.Close() })
	return c
}

// ---- tiến trình con ----

// TestSchedWorker là tiến trình con (SCHED_WORKER=1): hai tiến trình thật dùng chung Redis và provider id.
// Chế độ (SCHED_MODE): "pump" bơm N lời gọi qua Gateway + fake; "rpm" xin chỗ liên tục trong cửa sổ; "cmd" đọc lệnh từ stdin (mạch ngắt).
func TestSchedWorker(t *testing.T) {
	if os.Getenv("SCHED_WORKER") != "1" {
		t.Skip("chỉ chạy như tiến trình con")
	}
	ctx := context.Background()
	rdb := redisClient(t, os.Getenv("REDIS_URL"))
	prov := os.Getenv("SCHED_PROVIDER")
	cfg := scheduler.Config{MaxConcurrency: envInt("SCHED_MAX", 10), BatchShare: 0.5, QueueMax: 1000, QueueWaitMax: time.Minute,
		BreakerFails: 5, BreakerOpen: time.Duration(envInt("SCHED_OPEN_MS", 30000)) * time.Millisecond}
	s := scheduler.New(cfg, rdb, discard())

	switch os.Getenv("SCHED_MODE") {
	case "pump":
		ctl := fake.NewController(fake.Settings{LatencyMin: 200 * time.Millisecond, LatencyMax: 400 * time.Millisecond})
		reg := llm.NewStaticRegistry(map[llm.Task]llm.Route{})
		reg.SetRoute(llm.TaskUtility, llm.Route{Targets: []llm.Target{{ProviderID: prov, ProviderName: "F", Model: "m", RPM: 1e6, TPM: 1e9, P: fake.New(ctl, "")}}})
		g := llm.New(llm.Options{NoMask: true, Registry: reg, Gate: s, Log: discard(), RequestTimeout: 2 * time.Minute})
		var done atomic.Int64
		var wg sync.WaitGroup
		for i := range envInt("SCHED_N", 200) {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if _, err := g.Chat(ctx, llm.Request{Task: llm.TaskUtility, Messages: msg(fmt.Sprint("câu ", i))}); err == nil {
					done.Add(1)
				}
			}()
		}
		fmt.Println("READY")
		wg.Wait()
		fmt.Printf("DONE %d peak_local=%d\n", done.Load(), ctl.Peak())
	case "rpm":
		window := time.Duration(envInt("SCHED_WINDOW_MS", 10000)) * time.Millisecond
		cctx, cancel := context.WithTimeout(ctx, window)
		defer cancel()
		var grants, tokens atomic.Int64
		var wg sync.WaitGroup
		est := envInt("SCHED_EST", 1)
		for range 20 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for cctx.Err() == nil {
					p, err := s.Admit(cctx, llm.Work{Lane: llm.LaneNearRealtime, Task: llm.TaskUtility, ProviderID: prov,
						RPM: envInt("SCHED_RPM", 60), TPM: envInt("SCHED_TPM", 100000), EstTokens: est})
					if err != nil {
						return
					}
					grants.Add(1)
					tokens.Add(int64(est))
					p.Done(est)
				}
			}()
		}
		fmt.Println("READY")
		wg.Wait()
		fmt.Printf("GRANTS %d TOKENS %d\n", grants.Load(), tokens.Load())
	case "cmd":
		fmt.Println("READY")
		sc := bufio.NewScanner(os.Stdin)
		for sc.Scan() {
			switch sc.Text() {
			case "allow":
				fmt.Println("allow=", s.BreakerAllow(ctx, prov))
			case "fail":
				s.BreakerReport(ctx, prov, "SERVER")
				fmt.Println("ok")
			case "ok":
				s.BreakerReport(ctx, prov, "")
				fmt.Println("ok")
			case "state":
				fmt.Println("state=" + s.CircuitState(ctx, prov))
			}
		}
	}
}

func envInt(k string, def int) int {
	var n int
	if _, err := fmt.Sscan(os.Getenv(k), &n); err != nil || n == 0 {
		return def
	}
	return n
}

type worker struct {
	cmd   *exec.Cmd
	stdin io.WriteCloser
	lines chan string
}

func startWorker(t *testing.T, env map[string]string) *worker {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestSchedWorker$", "-test.timeout=240s") //nolint:gosec // chạy chính binary test
	cmd.Env = append(os.Environ(), "SCHED_WORKER=1")
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)
	stdin, err := cmd.StdinPipe()
	require.NoError(t, err)
	require.NoError(t, cmd.Start())
	w := &worker{cmd: cmd, stdin: stdin, lines: make(chan string, 256)}
	go func() {
		sc := bufio.NewScanner(stdout)
		for sc.Scan() {
			w.lines <- strings.TrimSpace(sc.Text())
		}
		close(w.lines)
	}()
	t.Cleanup(func() { _ = stdin.Close(); _ = cmd.Process.Kill(); _ = cmd.Wait() })
	w.expectPrefix(t, "READY", 30*time.Second)
	return w
}

func (w *worker) expectPrefix(t *testing.T, prefix string, within time.Duration) string {
	t.Helper()
	deadline := time.After(within)
	for {
		select {
		case l, ok := <-w.lines:
			if !ok {
				t.Fatalf("tiến trình con thoát trước khi in %q", prefix)
			}
			if strings.HasPrefix(l, prefix) {
				return l
			}
		case <-deadline:
			t.Fatalf("không thấy %q trong %v", prefix, within)
		}
	}
}

func (w *worker) ask(t *testing.T, cmd, prefix string) string {
	t.Helper()
	_, err := fmt.Fprintln(w.stdin, cmd)
	require.NoError(t, err)
	return w.expectPrefix(t, prefix, 10*time.Second)
}

// ---- Redis và proxy ----

func baseEnv(t *testing.T, mode, prov string) map[string]string {
	return map[string]string{"REDIS_URL": testutil.RedisURL(t), "SCHED_PROVIDER": prov, "SCHED_MODE": mode}
}

// ---- kiểm thử ----

func TestTwoProcessesGlobalConcurrency(t *testing.T) {
	prov := "it-" + uuid.NewString()
	rdb := redisClient(t, testutil.RedisURL(t))
	env := baseEnv(t, "pump", prov)
	env["SCHED_N"], env["SCHED_MAX"] = "200", "10"
	var peak atomic.Int64
	stop := make(chan struct{})
	go func() {
		for {
			select {
			case <-stop:
				return
			default:
				if n, err := rdb.ZCard(context.Background(), "ep:llm:inflight:"+prov).Result(); err == nil && n > peak.Load() {
					peak.Store(n)
				}
				time.Sleep(5 * time.Millisecond)
			}
		}
	}()
	w1, w2 := startWorker(t, env), startWorker(t, env)
	var completed int
	for _, w := range []*worker{w1, w2} {
		var n int
		line := w.expectPrefix(t, "DONE", 3*time.Minute)
		_, err := fmt.Sscanf(line, "DONE %d", &n)
		require.NoError(t, err)
		completed += n
	}
	close(stop)
	t.Logf("peak_inflight=%d completed=%d", peak.Load(), completed)
	require.LessOrEqual(t, peak.Load(), int64(10), "đỉnh inflight cộng cả hai tiến trình")
	require.GreaterOrEqual(t, peak.Load(), int64(8), "phải thực sự dùng song song")
	require.Equal(t, 400, completed)
	n, err := rdb.ZCard(context.Background(), "ep:llm:inflight:"+prov).Result()
	require.NoError(t, err)
	require.Zero(t, n, "không rò chỗ")
}

func TestTwoProcessesRPM(t *testing.T) {
	prov := "it-" + uuid.NewString()
	env := baseEnv(t, "rpm", prov)
	env["SCHED_RPM"], env["SCHED_TPM"], env["SCHED_MAX"], env["SCHED_WINDOW_MS"], env["SCHED_EST"] = "60", "100000000", "100", "10000", "1"
	w1, w2 := startWorker(t, env), startWorker(t, env)
	total := 0
	for _, w := range []*worker{w1, w2} {
		var g, tok int
		_, err := fmt.Sscanf(w.expectPrefix(t, "GRANTS", time.Minute), "GRANTS %d TOKENS %d", &g, &tok)
		require.NoError(t, err)
		total += g
	}
	ceiling := 60*10/60 + 6 // rpm × 10/60 + burst (10 % hạn mức)
	t.Logf("rpm: %d lời gọi được phép trong 10 s, trần tính được %d", total, ceiling)
	require.LessOrEqual(t, total, ceiling+1)
	require.GreaterOrEqual(t, total, ceiling/2)

	// TPM: 6.000 token/phút, mỗi lời gọi ước tính 100 → trần 6000×10/60 + 600 = 1.600 token (16 lời gọi)
	prov2 := "it-" + uuid.NewString()
	env = baseEnv(t, "rpm", prov2)
	env["SCHED_RPM"], env["SCHED_TPM"], env["SCHED_MAX"], env["SCHED_WINDOW_MS"], env["SCHED_EST"] = "100000", "6000", "100", "10000", "100"
	w1, w2 = startWorker(t, env), startWorker(t, env)
	tokens := 0
	for _, w := range []*worker{w1, w2} {
		var g, tok int
		_, err := fmt.Sscanf(w.expectPrefix(t, "GRANTS", time.Minute), "GRANTS %d TOKENS %d", &g, &tok)
		require.NoError(t, err)
		tokens += tok
	}
	t.Logf("tpm: %d token ước tính trong 10 s, trần 1600", tokens)
	require.LessOrEqual(t, tokens, 1600+100)
	require.GreaterOrEqual(t, tokens, 800)
}

func TestTPMReconcile(t *testing.T) {
	prov := "it-" + uuid.NewString()
	rdb := redisClient(t, testutil.RedisURL(t))
	s := scheduler.New(baseCfg(), rdb, discard())
	w := llm.Work{Lane: llm.LaneNearRealtime, Task: llm.TaskUtility, ProviderID: prov, RPM: 100000, TPM: 6000, EstTokens: 500} // dung lượng TPM 600

	p, err := s.Admit(t.Context(), w)
	require.NoError(t, err)
	p.Done(50) // thật chỉ 50 → hoàn 450
	start := time.Now()
	p2, err := s.Admit(t.Context(), w)
	require.NoError(t, err)
	require.Less(t, time.Since(start), 200*time.Millisecond, "đã hoàn phần dư nên lời gọi kế tiếp không phải chờ")
	p2.Done(5000) // thật nhiều hơn ước tính rất nhiều → bị trừ thêm
	ctx, cancel := context.WithTimeout(t.Context(), 300*time.Millisecond)
	defer cancel()
	_, err = s.Admit(ctx, w)
	require.ErrorIs(t, err, llm.ErrDeadline, "token thật vượt ước tính phải làm lời gọi kế tiếp chờ")
}

func TestBatchDoesNotStarveInteractive(t *testing.T) {
	prov := "it-" + uuid.NewString()
	rdb := redisClient(t, testutil.RedisURL(t))
	ctl := fake.NewController(fake.Settings{LatencyMin: 800 * time.Millisecond, LatencyMax: 1200 * time.Millisecond, StreamDelay: time.Millisecond})
	reg := llm.NewStaticRegistry(map[llm.Task]llm.Route{})
	rt := llm.Route{Targets: []llm.Target{{ProviderID: prov, ProviderName: "F", Model: "m", RPM: 1e6, TPM: 1e9, P: fake.New(ctl, "")}}}
	reg.SetRoute(llm.TaskChat, rt)
	reg.SetRoute(llm.TaskInsight, rt)
	cfg := scheduler.Config{MaxConcurrency: 10, BatchShare: 0.5, QueueMax: 1000, QueueWaitMax: 10 * time.Second}
	s := scheduler.New(cfg, rdb, discard())
	g := llm.New(llm.Options{NoMask: true, Registry: reg, Gate: s, Log: discard(), RequestTimeout: 3 * time.Minute, BatchTimeout: 3 * time.Minute})

	ttft := func(q string) (time.Duration, time.Duration) {
		start := time.Now()
		ch, err := g.Stream(t.Context(), llm.Request{Task: llm.TaskChat, Messages: msg(q)})
		require.NoError(t, err)
		first := time.Duration(0)
		var wait time.Duration
		for c := range ch {
			if first == 0 && c.Text != "" {
				first = time.Since(start)
			}
			if c.Response != nil {
				wait = c.Response.QueueWait
			}
		}
		return first, wait
	}
	runInteractive := func(n int) (mean time.Duration, waits []time.Duration) {
		sem := make(chan struct{}, 5) // tới ≤ 50 % công suất: tối đa 5 yêu cầu cùng lúc
		var mu sync.Mutex
		var wg sync.WaitGroup
		var total time.Duration
		for i := range n {
			sem <- struct{}{}
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer func() { <-sem }()
				f, w := ttft(fmt.Sprint("hỏi ", i))
				mu.Lock()
				total += f
				waits = append(waits, w)
				mu.Unlock()
			}()
			time.Sleep(60 * time.Millisecond)
		}
		wg.Wait()
		return total / time.Duration(n), waits
	}

	// đường cơ sở: không có BATCH
	baseMean, baseWaits := runInteractive(15)

	var wg sync.WaitGroup
	for i := range 100 { // 100 việc BATCH đang chờ
		wg.Add(1)
		go func() {
			defer wg.Done()
			lane := llm.LaneBatch
			_, _ = g.Chat(t.Context(), llm.Request{Task: llm.TaskInsight, Lane: &lane, Messages: msg(fmt.Sprint("batch ", i))})
		}()
	}
	var batchPeak atomic.Int64
	stop := make(chan struct{})
	go func() {
		for {
			select {
			case <-stop:
				return
			default:
				if st := s.Stats(context.Background()); st.InflightBatch[prov] > int(batchPeak.Load()) {
					batchPeak.Store(int64(st.InflightBatch[prov]))
				}
				time.Sleep(10 * time.Millisecond)
			}
		}
	}()
	time.Sleep(300 * time.Millisecond)
	mixedMean, waits := runInteractive(25)
	close(stop)

	slices.Sort(waits)
	p95 := waits[int(float64(len(waits))*0.95)-1]
	slices.Sort(baseWaits)
	baseP95 := baseWaits[int(float64(len(baseWaits))*0.95)-1]
	ratio := float64(mixedMean) / float64(baseMean)
	t.Logf("interactive_wait_p95_ms=%d (cơ sở %d) ttft_ratio=%.2f batch_peak=%d", p95.Milliseconds(), baseP95.Milliseconds(), ratio, batchPeak.Load())
	require.LessOrEqual(t, p95, 500*time.Millisecond)
	require.LessOrEqual(t, batchPeak.Load(), int64(5))
	require.LessOrEqual(t, ratio, 1.2)
	t.Cleanup(func() { wg.Wait() })
}

func TestBreakerSharedAcrossProcesses(t *testing.T) {
	prov := "it-" + uuid.NewString()
	env := baseEnv(t, "cmd", prov)
	env["SCHED_OPEN_MS"] = "400"
	a, b := startWorker(t, env), startWorker(t, env)
	for range 5 {
		a.ask(t, "fail", "ok")
	}
	require.Equal(t, "allow= false", b.ask(t, "allow", "allow="), "mạch mở ở tiến trình A phải chặn cả B")
	require.Equal(t, "state=open", b.ask(t, "state", "state="))
	time.Sleep(450 * time.Millisecond)
	require.Equal(t, "allow= true", b.ask(t, "allow", "allow="), "hết thời gian mở: B lấy lượt thử")
	require.Equal(t, "allow= false", a.ask(t, "allow", "allow="), "chỉ MỘT lời gọi thử trên toàn hệ thống")
	b.ask(t, "ok", "ok")
	require.Equal(t, "state=closed", a.ask(t, "state", "state="), "thử thành công ở B đóng mạch cho A")
}

// ---- Redis mất / về ----

type tcpProxy struct {
	target string
	mu     sync.Mutex
	ln     net.Listener
	conns  []net.Conn
	addr   string
}

func newProxy(t *testing.T, target string) *tcpProxy {
	p := &tcpProxy{target: target}
	p.up(t)
	t.Cleanup(p.down)
	return p
}

func (p *tcpProxy) up(t testing.TB) {
	var err error
	addr := p.addr
	if addr == "" {
		addr = "127.0.0.1:0"
	}
	p.ln, err = net.Listen("tcp", addr)
	require.NoError(t, err)
	p.addr = p.ln.Addr().String()
	ln := p.ln
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			up, err := net.Dial("tcp", p.target)
			if err != nil {
				_ = c.Close()
				continue
			}
			p.mu.Lock()
			p.conns = append(p.conns, c, up)
			p.mu.Unlock()
			go func() { _, _ = io.Copy(up, c); _ = up.Close(); _ = c.Close() }()
			go func() { _, _ = io.Copy(c, up); _ = up.Close(); _ = c.Close() }()
		}
	}()
}

func (p *tcpProxy) down() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.ln != nil {
		_ = p.ln.Close()
	}
	for _, c := range p.conns {
		_ = c.Close()
	}
	p.conns = nil
}

func hostPort(t *testing.T, redisURL string) string {
	opt, err := goredis.ParseURL(redisURL)
	require.NoError(t, err)
	return opt.Addr
}

func logBuf() (*bytes.Buffer, *slog.Logger) {
	b := &bytes.Buffer{}
	return b, slog.New(slog.NewTextHandler(&syncWriter{w: b}, nil))
}

type syncWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (s *syncWriter) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.w.Write(p)
}

func TestRedisDownFailsOpenLocal(t *testing.T) {
	proxy := newProxy(t, hostPort(t, testutil.RedisURL(t)))
	rdb := redisClient(t, "redis://"+proxy.addr+"/0")
	buf, log := logBuf()
	prov := "it-" + uuid.NewString()
	s := scheduler.New(scheduler.Config{MaxConcurrency: 2, QueueMax: 100}, rdb, log)
	w := llm.Work{Lane: llm.LaneNearRealtime, Task: llm.TaskUtility, ProviderID: prov, RPM: 1e6, TPM: 1e9, EstTokens: 10}

	p, err := s.Admit(t.Context(), w)
	require.NoError(t, err)
	p.Done(-1)
	require.False(t, s.RedisDown())

	proxy.down() // Redis mất
	start := time.Now()
	var held []llm.Permit
	for range 2 { // vẫn cấp chỗ (giới hạn cục bộ MAX=2), không treo
		p, err := s.Admit(t.Context(), w)
		require.NoError(t, err)
		held = append(held, p)
	}
	require.Less(t, time.Since(start), 5*time.Second, "không được treo khi Redis mất")
	require.True(t, s.RedisDown())
	ctx, cancel := context.WithTimeout(t.Context(), 250*time.Millisecond)
	defer cancel()
	_, err = s.Admit(ctx, w)
	require.ErrorIs(t, err, llm.ErrDeadline, "giới hạn cục bộ vẫn áp: chỗ thứ 3 phải chờ")
	for _, h := range held {
		h.Done(-1)
	}
	for range 5 { // nhiều lần gọi nữa vẫn chạy
		p, err := s.Admit(t.Context(), w)
		require.NoError(t, err)
		p.Done(-1)
	}
	require.Equal(t, 1, strings.Count(buf.String(), "Redis không dùng được"), "log error một lần mỗi 30 s:\n%s", buf.String())
	require.True(t, s.BreakerAllow(t.Context(), prov), "mạch cục bộ vẫn dùng được")

	proxy.up(t) // Redis về
	time.Sleep(1100 * time.Millisecond)
	p, err = s.Admit(t.Context(), w)
	require.NoError(t, err)
	p.Done(-1)
	require.False(t, s.RedisDown(), "Redis về thì quay lại chế độ toàn cục, không cần khởi động lại")
}

func TestRedisBackReconciles(t *testing.T) {
	proxy := newProxy(t, hostPort(t, testutil.RedisURL(t)))
	rdb := redisClient(t, "redis://"+proxy.addr+"/0")
	pool, err := pgxpool.New(t.Context(), testutil.MigratedPostgresURL(t))
	require.NoError(t, err)
	defer pool.Close()
	clk := clock.NewFake(time.Now())
	pfx := uuid.NewString() + ":"
	m := budget.New(rdb, limits{"system": {Daily: dec("1000")}}, pool, clk, discard()).WithPrefix(pfx)
	prov := "it-" + uuid.NewString()
	s := scheduler.New(scheduler.Config{MaxConcurrency: 2}, rdb, discard(), scheduler.WithBudget(m))
	recovered := make(chan struct{}, 1)
	s.OnRedisRecover(func(ctx context.Context) {
		require.NoError(t, m.Reconcile(ctx))
		recovered <- struct{}{}
	})
	w := llm.Work{Lane: llm.LaneNearRealtime, Task: llm.TaskUtility, ProviderID: prov, RPM: 1e6, TPM: 1e9, EstTokens: 10}

	proxy.down()
	p, err := s.Admit(t.Context(), w) // chuyển sang cục bộ
	require.NoError(t, err)
	p.Done(-1)
	s.BudgetCharge(t.Context(), nil, shopDec.NewFromInt(2000)) // Redis mất: ngừng cộng
	// llm_audit vẫn ghi đúng: chi phí thật 2.000 đ đã nằm trong bảng
	_, err = pool.Exec(t.Context(), `insert into llm_audit (task, lane, status, trace_id, cost_est) values ('CHAT','INTERACTIVE','ok','t',2000)`)
	require.NoError(t, err)
	require.Equal(t, llm.BudgetOK, s.BudgetState(t.Context(), nil), "Redis mất: không chặn chat vì lỗi hạ tầng")

	proxy.up(t)
	time.Sleep(1100 * time.Millisecond)
	p, err = s.Admit(t.Context(), w) // lần gọi đầu sau khi Redis về kích hoạt đối soát
	require.NoError(t, err)
	p.Done(-1)
	select {
	case <-recovered:
	case <-time.After(5 * time.Second):
		t.Fatal("không đối soát khi Redis về")
	}
	require.Equal(t, llm.BudgetExhausted, s.BudgetState(t.Context(), nil), "bộ đếm được dựng lại từ llm_audit")
}

func BenchmarkSchedulerAcquire(b *testing.B) {
	rdb := redisClient(b, testutil.RedisURL(b))
	prov := "bench-" + uuid.NewString()
	s := scheduler.New(scheduler.Config{MaxConcurrency: 1000, QueueMax: 100000}, rdb, discard())
	w := llm.Work{Lane: llm.LaneNearRealtime, Task: llm.TaskUtility, ProviderID: prov, RPM: 100_000_000, TPM: 1_000_000_000, EstTokens: 10}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			p, err := s.Admit(context.Background(), w)
			if err != nil {
				b.Error(err)
				return
			}
			p.Done(-1)
		}
	})
	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "acquires/s")
}
