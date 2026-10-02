package sse

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/edupilot/backend-go/internal/platform/config"
)

var idLineRE = regexp.MustCompile(`^\d+-\d+$`)

// waitSlots đợi ZCARD ep:sse:conn:<uid> và PUBSUB NUMSUB ep:sse:ch:<uid> về đúng want (05-AC4).
func waitSlots(t *testing.T, r *rig, uid string, want int64) {
	t.Helper()
	ctx := context.Background()
	deadline := time.Now().Add(grace)
	var zc, nsub int64
	for time.Now().Before(deadline) {
		zc, _ = r.rdb.ZCard(ctx, ConnKey(uid)).Result()
		subs, err := r.rdb.PubSubNumSub(ctx, ChanKey(uid)).Result()
		if err == nil {
			nsub = subs[ChanKey(uid)]
		}
		if zc == want && nsub == want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("ZCARD = %d, NUMSUB = %d, muốn %d", zc, nsub, want)
}

// 05-AC1 — header phản hồi và thứ tự byte mở stream (SRS 6.8).
func TestSSE_Headers(t *testing.T) {
	r := newRig(t, nil)
	uid := newUID()
	resp, cancel := r.get(r.token(uid, time.Hour), "", nil)
	defer cancel()
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("mã HTTP = %d, muốn 200", resp.StatusCode)
	}
	want := map[string]string{
		"Content-Type":      "text/event-stream",
		"Cache-Control":     "no-cache, no-transform",
		"X-Accel-Buffering": "no",
		"Connection":        "keep-alive",
	}
	for k, v := range want {
		if got := resp.Header.Get(k); got != v {
			t.Errorf("%s = %q, muốn %q", k, got, v)
		}
	}
	if got := resp.Header.Get("Content-Encoding"); got != "" {
		t.Errorf("Content-Encoding = %q, muốn rỗng (không nén)", got)
	}

	head := make([]byte, len(retryPrelude))
	if _, err := io.ReadFull(resp.Body, head); err != nil {
		t.Fatalf("đọc byte đầu: %v", err)
	}
	if string(head) != retryPrelude {
		t.Fatalf("byte đầu = %q, muốn %q", head, retryPrelude)
	}
}

// 05-AC1 — sự kiện trạng thái đầu tiên đến trong ≤ 300 ms, data có connection_id duy nhất và server_time UTC.
func TestSSE_FirstEventLatency(t *testing.T) {
	r := newRig(t, nil)
	uid := newUID()
	tok := r.token(uid, time.Hour)

	seen := map[string]bool{}
	for i := range 5 {
		start := time.Now()
		resp, cancel := r.get(tok, "", nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("mã HTTP = %d", resp.StatusCode)
		}
		frames := make(chan frame, 8)
		go scanFrames(resp.Body, frames)
		s := &stream{t: t, resp: resp, cancel: cancel, frames: frames}
		<-frames // retry: 3000
		ready := <-frames
		took := time.Since(start)
		s.close()
		waitSlots(t, r, uid, 0)

		if took > 300*time.Millisecond {
			t.Fatalf("lần %d: ready sau %s, muốn ≤ 300 ms", i, took)
		}
		var d struct {
			ConnectionID string `json:"connection_id"`
			ServerTime   string `json:"server_time"`
		}
		if err := json.Unmarshal([]byte(ready.data), &d); err != nil {
			t.Fatalf("data của ready không phải JSON: %v", err)
		}
		if d.ConnectionID == "" || seen[d.ConnectionID] {
			t.Fatalf("connection_id = %q, muốn giá trị mới mỗi kết nối", d.ConnectionID)
		}
		seen[d.ConnectionID] = true
		ts, err := time.Parse(time.RFC3339Nano, d.ServerTime)
		if err != nil || !strings.HasSuffix(d.ServerTime, "Z") {
			t.Fatalf("server_time = %q, muốn ISO-8601 UTC", d.ServerTime)
		}
		if time.Since(ts) > time.Minute || time.Until(ts) > time.Minute {
			t.Fatalf("server_time = %q lệch quá xa hiện tại", d.ServerTime)
		}
	}
}

// 05-AC2 — khung `id`/`event`/`data` + dòng trống; data là JSON một dòng; type biên 1 và 64 ký tự.
func TestSSE_Framing(t *testing.T) {
	r := newRig(t, nil)
	uid := newUID()
	s := r.connect(r.token(uid, time.Hour), nil)

	types := []string{"a", strings.Repeat("a", 64), "test.frame"}
	for i, typ := range types {
		r.publish(uid, typ, map[string]any{"n": i, "s": "nhiều\ndòng"})
	}
	got := s.collect(len(types), grace)
	for i, f := range got {
		if !idLineRE.MatchString(f.id) {
			t.Errorf("khung %d: id = %q, muốn ^\\d+-\\d+$", i, f.id)
		}
		if f.typ != types[i] {
			t.Errorf("khung %d: event = %q, muốn %q", i, f.typ, types[i])
		}
		if strings.ContainsAny(f.data, "\n\r") {
			t.Errorf("khung %d: data có xuống dòng: %q", i, f.data)
		}
		var payload map[string]any
		if err := json.Unmarshal([]byte(f.data), &payload); err != nil {
			t.Errorf("khung %d: data không phải JSON: %v", i, err)
		}
	}
}

// 05-AC2 — id là id Redis Stream, tăng NGHIÊM NGẶT.
func TestSSE_IDsMonotonic(t *testing.T) {
	r := newRig(t, nil)
	uid := newUID()
	s := r.connect(r.token(uid, time.Hour), nil)

	const n = 20
	for i := range n {
		r.publish(uid, "test.mono", map[string]int{"n": i})
	}
	got := s.collect(n, grace)
	for i, f := range got {
		if i > 0 && !olderThan(got[i-1].id, f.id) {
			t.Fatalf("id không tăng nghiêm ngặt: %q rồi %q", got[i-1].id, f.id)
		}
		var d struct{ N int }
		if err := json.Unmarshal([]byte(f.data), &d); err != nil || d.N != i {
			t.Fatalf("khung %d: data = %q", i, f.data)
		}
	}
}

// 05-AC3 — heartbeat `: hb` đều đặn mỗi SSE_HEARTBEAT, không mang id.
func TestSSE_Heartbeat(t *testing.T) {
	r := newRig(t, func(c *config.Config) { c.SSEHeartbeat = 200 * time.Millisecond })
	s := r.connect(r.token(newUID(), time.Hour), nil)

	var beats []time.Time
	for len(beats) < 3 {
		f, ok := s.next(2 * time.Second)
		if !ok {
			t.Fatal("stream đóng sớm")
		}
		if !f.hb {
			t.Fatalf("khung ngoài mong đợi khi nhàn rỗi: %+v", f)
		}
		if f.id != "" {
			t.Fatalf("heartbeat không được mang id, có %q", f.id)
		}
		beats = append(beats, time.Now())
	}
	for i := 1; i < len(beats); i++ {
		gap := beats[i].Sub(beats[i-1])
		if gap < 100*time.Millisecond || gap > 400*time.Millisecond {
			t.Fatalf("khoảng cách nhịp %d = %s, muốn ≈ 200 ms (không gộp lô)", i, gap)
		}
	}
}

// 05-AC4 — client ngắt liên tục: không rò goroutine, chỗ kết nối và subscription được trả, không log ERROR.
func TestSSE_NoLeakOnClientDisconnect(t *testing.T) {
	r := newRig(t, func(c *config.Config) { c.SSEHeartbeat = 50 * time.Millisecond })
	uid := newUID()
	tok := r.token(uid, time.Hour)

	// Khởi động pool Redis + transport HTTP trước khi lấy mốc goroutine.
	warm := r.connect(tok, nil)
	r.publish(uid, "test.warm", map[string]int{"n": 0})
	warm.want(grace)
	warm.close()
	waitSlots(t, r, uid, 0)
	runtime.GC()
	base := runtime.NumGoroutine()

	const cycles = 30
	for i := range cycles {
		s := r.connect(tok, nil)
		switch i % 3 {
		case 0: // ngắt khi đang chờ
		case 1: // ngắt ngay sau một sự kiện
			r.publish(uid, "test.cut", map[string]int{"n": i})
			s.want(grace)
		case 2: // ngắt sau ít nhất một heartbeat
			time.Sleep(70 * time.Millisecond)
		}
		s.close()
		waitSlots(t, r, uid, 0)
	}

	deadline := time.Now().Add(grace)
	var now int
	for time.Now().Before(deadline) {
		runtime.GC()
		now = runtime.NumGoroutine()
		if now <= base+5 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if now > base+5 {
		t.Fatalf("goroutine = %d sau %d chu kỳ, mốc ban đầu %d (±5)", now, cycles, base)
	}
	if logs := r.logs.String(); strings.Contains(logs, `"level":"ERROR"`) {
		t.Fatalf("có log mức ERROR vì client ngắt:\n%s", logs)
	}
}
