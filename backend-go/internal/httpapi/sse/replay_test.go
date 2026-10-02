package sse

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"testing"
	"time"

	"github.com/edupilot/backend-go/internal/platform/config"
)

// payloadN bóc số thứ tự khỏi data `{"n":<số>}`.
func payloadN(t *testing.T, f frame) int {
	t.Helper()
	var d struct {
		N int `json:"n"`
	}
	if err := json.Unmarshal([]byte(f.data), &d); err != nil {
		t.Fatalf("data %q không phải JSON: %v", f.data, err)
	}
	return d.N
}

// 05-AC7 — nối lại bằng Last-Event-ID: nhận đủ sự kiện phát lúc ngắt, không mất, không trùng.
func TestSSE_LastEventID_NoLossNoDup(t *testing.T) {
	r := newRig(t, nil)
	uid := newUID()
	tok := r.token(uid, time.Hour)

	s1 := r.connect(tok, nil)
	for i := 1; i <= 10; i++ {
		r.publish(uid, "test.replay", map[string]int{"n": i})
	}
	first := s1.collect(5, grace) // đọc e1…e5 rồi ngắt
	last := first[4].id
	if n := payloadN(t, first[4]); n != 5 {
		t.Fatalf("sự kiện thứ 5 có n = %d", n)
	}
	s1.close()
	waitSlots(t, r, uid, 0)

	for i := 11; i <= 13; i++ { // phát trong lúc client đang ngắt
		r.publish(uid, "test.replay", map[string]int{"n": i})
	}

	s2 := r.connect(tok, map[string]string{"Last-Event-ID": last})
	got := s2.collect(8, grace) // e6…e13
	for i, f := range got {
		if want := i + 6; payloadN(t, f) != want {
			t.Fatalf("khung %d: n = %d, muốn %d", i, payloadN(t, f), want)
		}
		if f.id == last || !olderThan(last, f.id) {
			t.Fatalf("nhận lại id đã đọc: %q", f.id)
		}
		if f.typ == evResync {
			t.Fatal("không được resync khi id còn trong bộ đệm")
		}
	}

	r.publish(uid, "test.replay", map[string]int{"n": 14})
	if n := payloadN(t, s2.want(grace)); n != 14 {
		t.Fatalf("sau đọc bù không nhận được sự kiện mới (n = %d)", n)
	}
	s2.close()
}

// 05-AC8 — đua khi nối lại: 1.000 sự kiện + 20 lần ngắt/nối → đúng 1.000 id duy nhất, theo thứ tự.
func TestSSE_ReconnectRace(t *testing.T) {
	const total = 1000
	r := newRig(t, func(c *config.Config) {
		c.SSEBufferMaxLen = 5000 // không cắt bộ đệm giữa chừng (bài test này không kiểm resync)
		c.SSEMaxPerUser = 8      // chỗ cũ có thể chưa kịp trả khi nối lại ngay
		c.SSEHeartbeat = time.Second
	})
	uid := newUID()
	tok := r.token(uid, time.Hour)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 1; i <= total; i++ {
			if _, err := r.pub.Publish(t.Context(), uid, "test.race", map[string]int{"n": i}); err != nil {
				return
			}
			time.Sleep(time.Millisecond)
		}
	}()

	var ids []string
	seen := map[int]bool{}
	last := ""
	deadline := time.Now().Add(60 * time.Second)
	for session := 0; len(seen) < total && time.Now().Before(deadline); session++ {
		hdr := map[string]string{}
		if last != "" {
			hdr["Last-Event-ID"] = last
		}
		s := r.connect(tok, hdr)
		cut := time.Now().Add(time.Duration(20+rand.IntN(60)) * time.Millisecond)
		if session >= 20 { // sau 20 lần ngắt thì ở lại cho tới khi nhận đủ
			cut = deadline
		}
		for len(seen) < total && time.Now().Before(cut) {
			f, ok := s.next(10 * time.Second)
			if !ok {
				break
			}
			if f.hb {
				continue
			}
			if f.typ == evResync {
				t.Fatalf("phải resync ở phiên %d: %s", session, f.data)
			}
			n := payloadN(t, f)
			if seen[n] {
				t.Fatalf("sự kiện n = %d nhận hai lần (id %s)", n, f.id)
			}
			seen[n] = true
			ids = append(ids, f.id)
			last = f.id
		}
		s.close()
	}
	<-done

	if len(seen) != total {
		t.Fatalf("nhận %d sự kiện khác nhau, muốn %d", len(seen), total)
	}
	for i := 1; i < len(ids); i++ {
		if !olderThan(ids[i-1], ids[i]) {
			t.Fatalf("id không tăng theo thứ tự nhận: %q rồi %q", ids[i-1], ids[i])
		}
	}
}

// 05-AC9 — Last-Event-ID hợp lệ nhưng cũ hơn đầu bộ đệm → resync buffer_exceeded, rồi chuyển sang sự kiện mới.
func TestSSE_ResyncWhenTooOld(t *testing.T) {
	r := newRig(t, nil)
	uid := newUID()
	tok := r.token(uid, time.Hour)
	for i := 1; i <= 5; i++ {
		r.publish(uid, "test.old", map[string]int{"n": i})
	}

	s := r.connect(tok, map[string]string{"Last-Event-ID": "1-0"})
	f := s.want(grace)
	if f.typ != evResync {
		t.Fatalf("sự kiện đầu sau ready = %+v, muốn resync", f)
	}
	if got := f.reason(t); got != reasonBufferTooOld {
		t.Fatalf("reason = %q, muốn %q", got, reasonBufferTooOld)
	}
	if f.id != "" {
		t.Fatalf("resync không được mang id, có %q", f.id)
	}

	r.publish(uid, "test.old", map[string]int{"n": 99})
	if n := payloadN(t, s.want(grace)); n != 99 {
		t.Fatalf("sau resync phải nhận sự kiện mới, n = %d", n)
	}
	s.close()
}

// 05-AC9 — Last-Event-ID sai định dạng → resync invalid_last_event_id; rỗng / khoảng trắng = không có.
func TestSSE_ResyncOnMalformedLastEventID(t *testing.T) {
	r := newRig(t, nil)

	for _, bad := range []string{"abc", "1-", "-1", "1-2-3", "0x1-0", "1_0"} {
		t.Run("sai_"+bad, func(t *testing.T) {
			uid := newUID()
			tok := r.token(uid, time.Hour)
			r.publish(uid, "test.bad", map[string]int{"n": 1})

			s := r.connect(tok, map[string]string{"Last-Event-ID": bad})
			defer s.close()
			f := s.want(grace)
			if f.typ != evResync || f.reason(t) != reasonBadLastID {
				t.Fatalf("khung = %+v, muốn resync invalid_last_event_id", f)
			}
			r.publish(uid, "test.bad", map[string]int{"n": 2})
			if n := payloadN(t, s.want(grace)); n != 2 {
				t.Fatalf("sau resync phải nhận sự kiện mới, n = %d", n)
			}
		})
	}

	for i, empty := range []string{"", "   "} {
		t.Run(fmt.Sprintf("rong_%d", i), func(t *testing.T) {
			uid := newUID()
			tok := r.token(uid, time.Hour)
			r.publish(uid, "test.empty", map[string]int{"n": 1})

			// http.Header.Set("", …) không gửi được header rỗng qua net/http, nên đặt thẳng vào map.
			resp, cancel := r.get(tok, "", map[string]string{"Last-Event-ID": empty})
			defer cancel()
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("mã HTTP = %d, muốn 200", resp.StatusCode)
			}
			frames := make(chan frame, 64)
			go scanFrames(resp.Body, frames)
			s := &stream{t: t, resp: resp, cancel: cancel, frames: frames}
			defer s.close()
			s.ready()

			r.publish(uid, "test.empty", map[string]int{"n": 2})
			f := s.want(grace)
			if f.typ == evResync {
				t.Fatalf("Last-Event-ID rỗng không được sinh resync: %s", f.data)
			}
			if n := payloadN(t, f); n != 2 {
				t.Fatalf("n = %d, muốn 2 (không đọc bù sự kiện cũ)", n)
			}
		})
	}
}
