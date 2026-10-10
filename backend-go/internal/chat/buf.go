package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	"github.com/edupilot/backend-go/internal/store"
)

// Tên sự kiện SSE (SRS 4.7.2).
const (
	EvStatus   = "status"
	EvSnapshot = "snapshot"
	EvToken    = "token"
	EvBlock    = "block"
	EvNotice   = "notice"
	EvDone     = "done"
	EvError    = "error"
)

// FrameWriter ghi một khung SSE; id đã gồm `attempt:`.
type FrameWriter interface {
	Frame(id, event string, data []byte) error
}

func bufKey(mid uuid.UUID, attempt int16) string {
	return "ep:chat:buf:" + mid.String() + ":" + strconv.Itoa(int(attempt))
}

func cancelChan(mid uuid.UUID) string { return "ep:chat:cancel:" + mid.String() }

func busyKey(uid uuid.UUID) string { return "ep:chat:active:" + uid.String() }

func isTerminal(ev string) bool { return ev == EvDone || ev == EvError }

// emit đẩy một sự kiện vào bộ đệm Redis Stream của lượt. Mọi người xem (kết nối đầu và nối lại) đọc CÙNG luồng này nên không có khe.
// Redis lỗi không làm hỏng lượt sinh: bản bền nằm ở DB (partial_content / content), người xem rơi về snapshot.
func (s *Service) emit(ctx context.Context, mid uuid.UUID, attempt int16, ev string, data any) {
	if s.Redis == nil {
		return
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return
	}
	k := bufKey(mid, attempt)
	pipe := s.Redis.Pipeline()
	pipe.XAdd(ctx, &goredis.XAddArgs{Stream: k, MaxLen: 5000, Approx: true, Values: map[string]any{"e": ev, "d": string(raw)}})
	pipe.Expire(ctx, k, s.c().BufTTL)
	if _, err := pipe.Exec(ctx); err != nil && s.Log != nil {
		s.Log.WarnContext(ctx, "chat: ghi bộ đệm SSE lỗi", "error", err)
	}
}

// Tail phát cho w mọi sự kiện của lượt hiện tại của tin `mid`: từ sau `lastEventID` (dạng `attempt:streamID`) hoặc từ đầu.
// Trả khi gặp sự kiện cuối (done / error), khi ctx kết thúc hoặc khi w lỗi. Bộ đệm mất / hết hạn → snapshot từ DB rồi thăm dò DB.
func (s *Service) Tail(ctx context.Context, w FrameWriter, m store.ChatMessage, lastEventID string) error {
	after := "0"
	if a, id, ok := strings.Cut(lastEventID, ":"); ok && a == strconv.Itoa(int(m.Attempt)) && id != "" {
		after = id
	}
	if s.Redis != nil {
		if n, err := s.Redis.Exists(ctx, bufKey(m.ID, m.Attempt)).Result(); err == nil && n > 0 {
			done, err := s.tailStream(ctx, w, m, after)
			if done || err != nil {
				return err
			}
		}
	}
	return s.tailDB(ctx, w, m)
}

func (s *Service) tailStream(ctx context.Context, w FrameWriter, m store.ChatMessage, after string) (bool, error) {
	k := bufKey(m.ID, m.Attempt)
	idle := 0
	for {
		res, err := s.Redis.XRead(ctx, &goredis.XReadArgs{Streams: []string{k, after}, Count: 100, Block: time.Second}).Result()
		switch {
		case errors.Is(err, goredis.Nil):
			idle++
			if idle%2 == 0 { // G có thể đã chết (reaper / tắt máy): DB đã đóng mà bộ đệm không có sự kiện cuối
				cur, derr := s.reload(ctx, m.ID)
				if derr != nil {
					return false, derr
				}
				if cur.Attempt != m.Attempt {
					return true, w.Frame("", EvError, []byte(`{"code":"SUPERSEDED"}`))
				}
				if cur.StreamStatus != store.ChatStreamStatusSTREAMING {
					return s.drainThenFinal(ctx, w, cur, k, &after)
				}
			}
			if ctx.Err() != nil {
				return true, ctx.Err()
			}
			continue
		case err != nil:
			if ctx.Err() != nil {
				return true, ctx.Err()
			}
			return false, nil // Redis lỗi: rơi về DB
		}
		idle = 0
		for _, st := range res {
			for _, e := range st.Messages {
				ev, _ := e.Values["e"].(string)
				d, _ := e.Values["d"].(string)
				after = e.ID
				if err := w.Frame(strconv.Itoa(int(m.Attempt))+":"+e.ID, ev, []byte(d)); err != nil {
					return true, err
				}
				if isTerminal(ev) {
					return true, nil
				}
			}
		}
	}
}

// drainThenFinal: DB đã đóng — đọc nốt phần còn lại của bộ đệm (G ghi DB trước rồi mới đẩy `done`), thiếu sự kiện cuối thì dựng từ DB.
func (s *Service) drainThenFinal(ctx context.Context, w FrameWriter, cur store.ChatMessage, k string, after *string) (bool, error) {
	res, err := s.Redis.XRead(ctx, &goredis.XReadArgs{Streams: []string{k, *after}, Count: 5000, Block: -1}).Result()
	if err == nil {
		for _, st := range res {
			for _, e := range st.Messages {
				ev, _ := e.Values["e"].(string)
				d, _ := e.Values["d"].(string)
				if err := w.Frame(strconv.Itoa(int(cur.Attempt))+":"+e.ID, ev, []byte(d)); err != nil {
					return true, err
				}
				if isTerminal(ev) {
					return true, nil
				}
			}
		}
	}
	return true, writeFinal(w, cur)
}

func (s *Service) reload(ctx context.Context, mid uuid.UUID) (store.ChatMessage, error) {
	m, err := store.New(s.Pool).GetChatMessageByID(ctx, mid)
	if err != nil {
		return store.ChatMessage{}, fmt.Errorf("chat: nạp lại tin: %w", err)
	}
	return m, nil
}

// tailDB: không có bộ đệm — snapshot từ DB, rồi (nếu còn STREAMING) thăm dò mỗi 500 ms và phát phần mới dạng token.
func (s *Service) tailDB(ctx context.Context, w FrameWriter, m store.ChatMessage) error {
	cur, err := s.reload(ctx, m.ID)
	if err != nil {
		return err
	}
	sent := 0
	text := textOf(cur)
	snap, _ := json.Marshal(map[string]any{"off": 0, "t": text})
	if err := w.Frame("", EvSnapshot, snap); err != nil {
		return err
	}
	sent = utf8.RuneCountInString(text)
	for cur.StreamStatus == store.ChatStreamStatusSTREAMING {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
		if cur, err = s.reload(ctx, m.ID); err != nil {
			return err
		}
		if cur.Attempt != m.Attempt {
			return w.Frame("", EvError, []byte(`{"code":"SUPERSEDED"}`))
		}
		r := []rune(textOf(cur))
		if len(r) > sent {
			d, _ := json.Marshal(map[string]any{"off": sent, "t": string(r[sent:])})
			if err := w.Frame("", EvToken, d); err != nil {
				return err
			}
			sent = len(r)
		}
	}
	// Đã đóng: phần còn thiếu giữa lần thăm dò cuối và bản cuối
	r := []rune(textOf(cur))
	if len(r) > sent {
		d, _ := json.Marshal(map[string]any{"off": sent, "t": string(r[sent:])})
		if err := w.Frame("", EvToken, d); err != nil {
			return err
		}
	}
	return writeFinal(w, cur)
}

func textOf(m store.ChatMessage) string {
	if m.StreamStatus != store.ChatStreamStatusDONE && m.PartialContent != nil {
		return *m.PartialContent
	}
	return m.Content
}

func writeFinal(w FrameWriter, m store.ChatMessage) error {
	ev, d := finalEvent(m)
	return w.Frame("", ev, d)
}

// finalEvent dựng sự kiện cuối từ hàng DB.
func finalEvent(m store.ChatMessage) (string, []byte) {
	switch m.StreamStatus {
	case store.ChatStreamStatusDONE:
		cites := json.RawMessage(m.Citations)
		if len(cites) == 0 {
			cites = json.RawMessage("[]")
		}
		d, _ := json.Marshal(map[string]any{"message_id": m.ID, "citations": cites, "low_confidence": m.LowConfidence, "degraded": m.Degraded})
		return EvDone, d
	case store.ChatStreamStatusCANCELLED:
		return EvError, []byte(`{"code":"CANCELLED","message":"Đã dừng."}`)
	default:
		code := "INTERRUPTED"
		if m.ErrorCode != nil {
			code = *m.ErrorCode
		}
		d, _ := json.Marshal(map[string]any{"code": code, "message": errorText(code)})
		return EvError, d
	}
}

func errorText(code string) string {
	switch code {
	case "INTERRUPTED":
		return "Câu trả lời bị gián đoạn."
	case "OVERLOADED":
		return "AI đang bận."
	default:
		return "AI đang gián đoạn. Thử lại sau."
	}
}
