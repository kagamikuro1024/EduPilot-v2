package sse

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/testutil"
)

// 05-AC10 — sự kiện phát từ bản B tới được stream đang mở ở bản A trong ≤ 1 s.
func TestSSE_CrossInstance(t *testing.T) {
	a := newRig(t, nil)
	b := rigAt(t, testutil.RedisURL(t), nil)
	uid := newUID()

	s := a.connect(a.token(uid, time.Hour), nil)
	defer s.close()

	start := time.Now()
	b.publish(uid, "test.cross", map[string]int{"n": 1})
	f := s.want(2 * time.Second)
	if f.typ != "test.cross" || payloadN(t, f) != 1 {
		t.Fatalf("khung = %+v", f)
	}
	if took := time.Since(start); took > time.Second {
		t.Fatalf("độ trễ chéo bản = %s, muốn ≤ 1 s", took)
	}
}

// 05-AC10 — hai kết nối của CÙNG người dùng ở hai bản đều nhận sự kiện.
func TestSSE_TwoConnectionsTwoInstances(t *testing.T) {
	a := newRig(t, nil)
	b := rigAt(t, testutil.RedisURL(t), nil)
	uid := newUID()
	tok := a.token(uid, time.Hour)

	sa := a.connect(tok, nil)
	defer sa.close()
	sb := b.connect(tok, nil)
	defer sb.close()

	a.publish(uid, "test.both", map[string]int{"n": 7})
	for name, s := range map[string]*stream{"bản A": sa, "bản B": sb} {
		f := s.want(2 * time.Second)
		if f.typ != "test.both" || payloadN(t, f) != 7 {
			t.Fatalf("%s: khung = %+v", name, f)
		}
	}
}

// 05-AC11 — sự kiện của U1 không lọt sang U2.
func TestSSE_IsolationBetweenUsers(t *testing.T) {
	r := newRig(t, nil)
	u1, u2 := newUID(), newUID()

	s1 := r.connect(r.token(u1, time.Hour), nil)
	defer s1.close()
	s2 := r.connect(r.token(u2, time.Hour), nil)
	defer s2.close()

	r.publish(u1, "test.iso", map[string]int{"n": 1})
	if f := s1.want(grace); f.typ != "test.iso" {
		t.Fatalf("U1 không nhận được sự kiện của mình: %+v", f)
	}
	s2.quiet(500 * time.Millisecond)
}

// 05-AC11 — `?user_id=` bị bỏ qua: không có cách chọn stream người khác.
func TestSSE_UserIDQueryIgnored(t *testing.T) {
	r := newRig(t, nil)
	u1, u2 := newUID(), newUID()

	s := r.connectQ(r.token(u2, time.Hour), "?user_id="+u1, nil)
	defer s.close()

	r.publish(u1, "test.q", map[string]int{"n": 1})
	s.quiet(500 * time.Millisecond)

	r.publish(u2, "test.q", map[string]int{"n": 2})
	f := s.want(grace)
	if payloadN(t, f) != 2 {
		t.Fatalf("chỉ được nhận sự kiện của chính mình, nhận %+v", f)
	}
}

// 05-AC11 / SRS 6.1 — không token / token hỏng / token qua query string → 401 JSON trước khi mở stream.
func TestSSE_Unauthenticated(t *testing.T) {
	r := newRig(t, nil)
	good := r.token(newUID(), time.Hour)
	expired := r.token(newUID(), -time.Hour)
	forged := good[:strings.LastIndex(good, ".")] + ".AAAAinvalidsignatureAAAA"

	cases := []struct {
		name, tok, query, code string
	}{
		{"không token", "", "", apierr.Unauthenticated},
		{"token qua ?token=", "", "?token=" + good, apierr.Unauthenticated},
		{"token qua ?access_token=", "", "?access_token=" + good, apierr.Unauthenticated},
		{"token hết hạn", expired, "", apierr.TokenExpired},
		{"sai chữ ký", forged, "", apierr.TokenInvalid},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			resp, cancel := r.get(c.tok, c.query, nil)
			defer cancel()
			if resp.StatusCode != http.StatusUnauthorized {
				t.Fatalf("mã HTTP = %d, muốn 401", resp.StatusCode)
			}
			if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
				t.Errorf("Content-Type = %q, muốn JSON", ct)
			}
			if got := resp.Header.Get("WWW-Authenticate"); !strings.Contains(got, `Bearer realm="edupilot"`) {
				t.Errorf("WWW-Authenticate = %q", got)
			}
			if got := resp.Header.Get("X-Accel-Buffering"); got != "" {
				t.Errorf("401 không được mang header của stream")
			}
			if body := errBody(t, resp); body["code"] != c.code {
				t.Errorf("code = %v, muốn %s", body["code"], c.code)
			}
		})
	}
}

// 05-AC13 — tắt bản đang giữ stream: `event: shutdown` rồi đóng; nối lại bản còn lại bằng cùng token
// và Last-Event-ID thì không mất, không trùng.
func TestSSE_GatewayShutdownFailover(t *testing.T) {
	a := newRig(t, nil)
	b := rigAt(t, testutil.RedisURL(t), nil)
	uid := newUID()
	tok := a.token(uid, time.Hour)

	s1 := a.connect(tok, nil)
	a.publish(uid, "test.fo", map[string]int{"n": 1})
	a.publish(uid, "test.fo", map[string]int{"n": 2})
	got := s1.collect(2, grace)
	last := got[1].id

	start := time.Now()
	close(a.drain)
	f := s1.wantType(2*time.Second, evShutdown)
	if got := f.reason(t); got != reasonShutdown {
		t.Fatalf("reason = %q, muốn %q", got, reasonShutdown)
	}
	if f.id != "" {
		t.Fatalf("shutdown không được mang id, có %q", f.id)
	}
	s1.eof(2 * time.Second)
	if took := time.Since(start); took > 2*time.Second {
		t.Fatalf("đóng sau %s", took)
	}

	// Phát trong lúc chuyển bản.
	b.publish(uid, "test.fo", map[string]int{"n": 3})
	b.publish(uid, "test.fo", map[string]int{"n": 4})

	s2 := b.connect(tok, map[string]string{"Last-Event-ID": last})
	defer s2.close()
	after := s2.collect(2, grace)
	for i, f := range after {
		if want := i + 3; payloadN(t, f) != want {
			t.Fatalf("khung %d: n = %d, muốn %d", i, payloadN(t, f), want)
		}
		if !olderThan(last, f.id) {
			t.Fatalf("nhận lại sự kiện cũ: %q", f.id)
		}
	}
	b.publish(uid, "test.fo", map[string]int{"n": 5})
	if n := payloadN(t, s2.want(grace)); n != 5 {
		t.Fatalf("phiên 2 không nhận được sự kiện mới (n = %d)", n)
	}
}

// 05-AC15 — Redis chết giữa stream → `event: reconnect` upstream_unavailable rồi đóng, không treo.
func TestSSE_RedisDownMidStream(t *testing.T) {
	up := testutil.RedisURL(t)
	p := newProxy(t, up)
	r := rigAt(t, p.url(t, up), nil)
	uid := newUID()

	s := r.connect(r.token(uid, time.Hour), nil)
	defer s.close()
	r.publish(uid, "test.alive", map[string]int{"n": 1})
	s.want(grace)

	start := time.Now()
	p.stop()
	f := s.wantType(5*time.Second, evReconnect)
	if got := f.reason(t); got != reasonUpstream {
		t.Fatalf("reason = %q, muốn %q", got, reasonUpstream)
	}
	s.eof(3 * time.Second)
	if took := time.Since(start); took > 5*time.Second {
		t.Fatalf("stream treo %s sau khi Redis chết", took)
	}
}

// 05-AC15 — mở kết nối mới khi Redis còn chết → 503 SERVICE_UNAVAILABLE (JSON, không mở stream).
func TestSSE_RedisDownOnConnect(t *testing.T) {
	up := testutil.RedisURL(t)
	p := newProxy(t, up)
	r := rigAt(t, p.url(t, up), nil)
	p.stop()

	resp, cancel := r.get(r.token(newUID(), time.Hour), "", nil)
	defer cancel()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("mã HTTP = %d, muốn 503", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type = %q, muốn JSON", ct)
	}
	if body := errBody(t, resp); body["code"] != apierr.ServiceUnavailable {
		t.Errorf("code = %v, muốn %s", body["code"], apierr.ServiceUnavailable)
	}
}
