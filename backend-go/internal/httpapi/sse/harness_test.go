package sse

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/config"
	"github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/testutil"
)

// testSecret là khoá HS256 đủ 32 byte dùng cho cả Issuer và Verifier trong test.
const testSecret = "0123456789abcdef0123456789abcdef"

// Hạn chờ mặc định của test: ngắn nhưng đủ rộng cho Redis trong container.
const grace = 5 * time.Second

var testClient = &http.Client{}

// testCfg là cấu hình SSE "nhanh" cho test (nhịp 150 ms thay vì 25 s).
func testCfg() config.Config {
	return config.Config{
		SSEHeartbeat:    150 * time.Millisecond,
		SSEMaxDuration:  30 * time.Second,
		SSEMaxPerUser:   2,
		SSEConnTTL:      150 * time.Second,
		SSEBufferMaxLen: 1000,
		SSEBufferTTL:    time.Hour,
	}
}

// safeBuf là đích ghi log an toàn khi dùng song song (test kiểm "không có dòng ERROR").
type safeBuf struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *safeBuf) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *safeBuf) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

// rig là một "bản gateway": handler SSE + Publisher trên cùng một Redis.
type rig struct {
	t     *testing.T
	srv   *httptest.Server
	rdb   *redis.Client
	pub   Publisher
	cfg   config.Config
	logs  *safeBuf
	drain chan struct{}

	mu   sync.Mutex
	open []*stream
}

// newRig dựng một bản gateway trên Redis dùng chung của testutil.
func newRig(t *testing.T, tweak func(*config.Config)) *rig {
	t.Helper()
	return rigAt(t, testutil.RedisURL(t), tweak)
}

// rigAt dựng một bản gateway trên một URL Redis bất kỳ (dùng cho bản thứ hai và cho proxy "Redis chết").
func rigAt(t *testing.T, redisURL string, tweak func(*config.Config)) *rig {
	t.Helper()
	cfg := testCfg()
	if tweak != nil {
		tweak(&cfg)
	}
	rdb, err := redis.New(context.Background(), redisURL)
	if err != nil {
		t.Fatalf("redis.New: %v", err)
	}
	r := &rig{
		t: t, rdb: rdb, cfg: cfg, logs: &safeBuf{}, drain: make(chan struct{}),
		pub: NewPublisher(rdb, cfg.SSEBufferMaxLen, cfg.SSEBufferTTL),
	}
	log := slog.New(slog.NewJSONHandler(r.logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	r.srv = httptest.NewServer(NewHandler(HandlerDeps{
		Cfg: cfg, Log: log, Redis: rdb, Verifier: auth.NewVerifier(testSecret, clock.Real{}),
		Clock: clock.Real{}, Drain: r.drain,
	}))
	t.Cleanup(func() {
		r.srv.Close()
		_ = rdb.Close()
	})
	t.Cleanup(r.closeStreams) // chạy TRƯỚC srv.Close (cleanup theo thứ tự ngược)
	return r
}

func (r *rig) closeStreams() {
	r.mu.Lock()
	open := r.open
	r.open = nil
	r.mu.Unlock()
	for _, s := range open {
		s.close()
	}
}

// token ký một token STUDENT cho sub; ttl = 0 → mặc định của Issuer.
func (r *rig) token(sub string, ttl time.Duration) string {
	r.t.Helper()
	tok, err := auth.NewIssuer(testSecret, ttl, clock.Real{}).Issue(sub, auth.RoleStudent, "sse@test.local")
	if err != nil {
		r.t.Fatalf("Issue: %v", err)
	}
	return tok
}

// publish phát một sự kiện cho uid và trả id Redis Stream.
func (r *rig) publish(uid, typ string, data any) string {
	r.t.Helper()
	id, err := r.pub.Publish(r.t.Context(), uid, typ, data)
	if err != nil {
		r.t.Fatalf("Publish: %v", err)
	}
	return id
}

// get mở `GET /api/v1/events` và trả phản hồi thô (test tự đóng body) — dùng cho nhánh 401/429/503.
func (r *rig) get(tok, query string, hdr map[string]string) (*http.Response, context.CancelFunc) {
	r.t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.srv.URL+"/api/v1/events"+query, nil)
	if err != nil {
		cancel()
		r.t.Fatalf("NewRequest: %v", err)
	}
	if tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	resp, err := testClient.Do(req)
	if err != nil {
		cancel()
		r.t.Fatalf("GET /events: %v", err)
	}
	return resp, cancel
}

// connect mở stream và đọc xong `retry: 3000` + `event: ready`.
func (r *rig) connect(tok string, hdr map[string]string) *stream {
	r.t.Helper()
	return r.connectQ(tok, "", hdr)
}

func (r *rig) connectQ(tok, query string, hdr map[string]string) *stream {
	r.t.Helper()
	resp, cancel := r.get(tok, query, hdr)
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		cancel()
		r.t.Fatalf("mã HTTP = %d, muốn 200 (%s)", resp.StatusCode, body)
	}
	s := &stream{t: r.t, resp: resp, cancel: cancel, frames: make(chan frame, 4096)}
	go scanFrames(resp.Body, s.frames)
	r.mu.Lock()
	r.open = append(r.open, s)
	r.mu.Unlock()
	s.ready()
	return s
}

// frame là một khối SSE đã bóc tách.
type frame struct {
	id, typ, data, retry string
	hb                   bool
}

type stream struct {
	t      *testing.T
	resp   *http.Response
	cancel context.CancelFunc
	frames chan frame
	once   sync.Once
}

func scanFrames(body io.ReadCloser, out chan<- frame) {
	defer close(out)
	sc := bufio.NewScanner(body)
	sc.Buffer(make([]byte, 0, 4096), 1<<20)
	var f frame
	var filled bool
	for sc.Scan() {
		line := sc.Text()
		switch {
		case line == "":
			if filled {
				out <- f
			}
			f, filled = frame{}, false
		case strings.HasPrefix(line, "id: "):
			f.id, filled = strings.TrimPrefix(line, "id: "), true
		case strings.HasPrefix(line, "event: "):
			f.typ, filled = strings.TrimPrefix(line, "event: "), true
		case strings.HasPrefix(line, "data: "):
			f.data, filled = strings.TrimPrefix(line, "data: "), true
		case strings.HasPrefix(line, "retry: "):
			f.retry, filled = strings.TrimPrefix(line, "retry: "), true
		case line == ": hb":
			f.hb, filled = true, true
		}
	}
}

// close ngắt kết nối như client đột ngột biến mất.
func (s *stream) close() {
	s.once.Do(func() {
		s.cancel()
		_ = s.resp.Body.Close()
		for range s.frames { // xả cho goroutine quét thoát
		}
	})
}

// next trả khung kế tiếp; ok=false khi stream đã đóng.
func (s *stream) next(d time.Duration) (frame, bool) {
	s.t.Helper()
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case f, ok := <-s.frames:
		return f, ok
	case <-timer.C:
		s.t.Fatalf("quá %s mà không có khung nào", d)
		return frame{}, false
	}
}

// ready kiểm `retry: 3000` rồi `event: ready` (SRS 6.8).
func (s *stream) ready() {
	s.t.Helper()
	f, ok := s.next(grace)
	if !ok || f.retry != "3000" {
		s.t.Fatalf("khung đầu = %+v, muốn retry: 3000", f)
	}
	f, ok = s.next(grace)
	if !ok || f.typ != evReady {
		s.t.Fatalf("khung thứ hai = %+v, muốn event: ready", f)
	}
	if f.id != "" {
		s.t.Fatalf("event: ready không được mang id, có %q", f.id)
	}
}

// want đợi khung dữ liệu / điều khiển kế tiếp, bỏ qua heartbeat.
func (s *stream) want(d time.Duration) frame {
	s.t.Helper()
	deadline := time.Now().Add(d)
	for {
		f, ok := s.next(time.Until(deadline) + time.Millisecond)
		if !ok {
			s.t.Fatal("stream đóng khi đang chờ khung")
		}
		if !f.hb {
			return f
		}
		if time.Now().After(deadline) {
			s.t.Fatalf("quá %s mà chỉ thấy heartbeat", d)
		}
	}
}

// wantType đợi một khung đúng type.
func (s *stream) wantType(d time.Duration, typ string) frame {
	s.t.Helper()
	f := s.want(d)
	if f.typ != typ {
		s.t.Fatalf("khung = %+v, muốn event: %s", f, typ)
	}
	return f
}

// collect gom n khung dữ liệu (bỏ heartbeat).
func (s *stream) collect(n int, d time.Duration) []frame {
	s.t.Helper()
	out := make([]frame, 0, n)
	deadline := time.Now().Add(d)
	for len(out) < n {
		f, ok := s.next(time.Until(deadline) + time.Millisecond)
		if !ok {
			s.t.Fatalf("stream đóng sau %d/%d khung", len(out), n)
		}
		if !f.hb {
			out = append(out, f)
		}
	}
	return out
}

// quiet khẳng định không còn khung nào trong d (dùng cho kiểm cô lập người dùng).
func (s *stream) quiet(d time.Duration) {
	s.t.Helper()
	timer := time.NewTimer(d)
	defer timer.Stop()
	for {
		select {
		case f, ok := <-s.frames:
			if !ok {
				s.t.Fatal("stream đóng ngoài mong đợi")
			}
			if !f.hb {
				s.t.Fatalf("nhận khung ngoài mong đợi: %+v", f)
			}
		case <-timer.C:
			return
		}
	}
}

// eof đợi stream đóng hẳn.
func (s *stream) eof(d time.Duration) {
	s.t.Helper()
	deadline := time.Now().Add(d)
	for {
		select {
		case f, ok := <-s.frames:
			if !ok {
				return
			}
			if !f.hb {
				s.t.Fatalf("muốn stream đóng, còn nhận %+v", f)
			}
		case <-time.After(time.Until(deadline) + time.Millisecond):
			s.t.Fatalf("stream không đóng sau %s", d)
		}
	}
}

// reason bóc `reason` của một sự kiện điều khiển.
func (f frame) reason(t *testing.T) string {
	t.Helper()
	var d struct {
		Reason string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(f.data), &d); err != nil {
		t.Fatalf("data %q không phải JSON: %v", f.data, err)
	}
	return d.Reason
}

func newUID() string { return uuid.NewString() }

// errBody đọc thân lỗi JSON của một phản hồi không phải 200.
func errBody(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	defer func() { _ = resp.Body.Close() }()
	var m map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		t.Fatalf("thân lỗi không phải JSON: %v", err)
	}
	return m
}

// proxy là cầu TCP tới Redis để test "Redis chết" mà không đụng container dùng chung.
type proxy struct {
	ln      net.Listener
	mu      sync.Mutex
	conns   []net.Conn
	stopped bool
}

func newProxy(t *testing.T, upstream string) *proxy {
	t.Helper()
	target, err := url.Parse(upstream)
	if err != nil {
		t.Fatalf("REDIS_URL: %v", err)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	p := &proxy{ln: ln}
	go p.serve(target.Host)
	t.Cleanup(p.stop)
	return p
}

// url trả URL Redis đi qua proxy (giữ nguyên thông tin đăng nhập / số db của URL gốc).
func (p *proxy) url(t *testing.T, upstream string) string {
	t.Helper()
	u, err := url.Parse(upstream)
	if err != nil {
		t.Fatalf("REDIS_URL: %v", err)
	}
	u.Host = p.ln.Addr().String()
	return u.String()
}

func (p *proxy) serve(target string) {
	for {
		down, err := p.ln.Accept()
		if err != nil {
			return
		}
		up, err := net.Dial("tcp", target)
		if err != nil {
			_ = down.Close()
			continue
		}
		p.mu.Lock()
		if p.stopped {
			p.mu.Unlock()
			_, _ = down.Close(), up.Close()
			return
		}
		p.conns = append(p.conns, down, up)
		p.mu.Unlock()
		go func() { _, _ = io.Copy(up, down); _ = up.Close() }()
		go func() { _, _ = io.Copy(down, up); _ = down.Close() }()
	}
}

// stop mô phỏng Redis chết: không nhận kết nối mới, cắt mọi kết nối đang mở.
func (p *proxy) stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.stopped {
		return
	}
	p.stopped = true
	_ = p.ln.Close()
	for _, c := range p.conns {
		_ = c.Close()
	}
	p.conns = nil
}
