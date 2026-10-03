package llm_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/llm/fake"
	"github.com/edupilot/backend-go/internal/llm/provider"
)

func fakeGateway(t *testing.T, s fake.Settings) (*llm.Gateway, *fake.Controller, *capture) {
	t.Helper()
	reg := llm.NewRegistry(nil, llm.EnvConfig{Provider: "fake", Fake: s}, nil, newDiscardLog())
	if err := reg.Load(t.Context()); err != nil {
		t.Fatal(err)
	}
	cap := &capture{}
	aud := llm.NewAuditor(context.Background(), cap.write, newDiscardLog())
	t.Cleanup(func() { aud.Close(context.Background()) })
	return llm.New(llm.Options{Registry: reg, Auditor: aud, Log: newDiscardLog()}), reg.Fake(), cap
}

func drain(t *testing.T, ch <-chan llm.Chunk) (string, *llm.Response, error) {
	t.Helper()
	var sb strings.Builder
	var resp *llm.Response
	var err error
	timeout := time.After(5 * time.Second)
	for {
		select {
		case c, ok := <-ch:
			if !ok {
				return sb.String(), resp, err
			}
			sb.WriteString(c.Text)
			if c.Response != nil {
				resp = c.Response
			}
			if c.Err != nil {
				err = c.Err
			}
		case <-timeout:
			t.Fatal("kênh stream không đóng")
		}
	}
}

func TestStreamOneGeneration(t *testing.T) {
	t.Parallel()
	g, ctl, cap := fakeGateway(t, fake.Settings{StreamDelay: time.Millisecond})
	ch, err := g.Stream(t.Context(), llm.Request{Task: llm.TaskChat, Messages: userMsg("Hạn nộp bài tập là khi nào?")})
	if err != nil {
		t.Fatal(err)
	}
	text, resp, err := drain(t, ch)
	if err != nil || resp == nil {
		t.Fatalf("resp=%v err=%v", resp, err)
	}
	if !strings.HasPrefix(text, "Đây là câu trả lời giả lập cho: Hạn nộp") || resp.Text != text {
		t.Errorf("text = %q", text)
	}
	// D47: đúng MỘT lời gọi sinh văn bản cho một câu trả lời
	if got := ctl.Calls(); got != 1 {
		t.Errorf("fake.Calls = %d, muốn 1", got)
	}
	rows := cap.all(t, g)
	if len(rows) != 1 || rows[0].Status != "ok" || rows[0].TokensOut == 0 {
		t.Errorf("audit = %+v", rows)
	}
	// Structured đi kèm (phân loại) là MỘT lời gọi riêng cho mục đích khác → tổng 2, không gọi đôi cho cùng mục đích
	if _, err := g.Structured(t.Context(), llm.Request{Task: llm.TaskClassify, Messages: userMsg("phân loại")},
		[]byte(`{"type":"object","properties":{"label":{"type":"string"}},"required":["label"],"additionalProperties":false}`)); err != nil {
		t.Fatal(err)
	}
	if got := ctl.Calls(); got != 2 {
		t.Errorf("fake.Calls = %d, muốn 2", got)
	}
}

func TestStreamCancel(t *testing.T) {
	t.Parallel()
	g, ctl, cap := fakeGateway(t, fake.Settings{StreamDelay: 50 * time.Millisecond})
	ctx, cancel := context.WithCancel(t.Context())
	ch, err := g.Stream(ctx, llm.Request{Task: llm.TaskChat, Messages: userMsg("một câu hỏi khá dài để có nhiều từ phát ra lần lượt")})
	if err != nil {
		t.Fatal(err)
	}
	<-ch // nhận mẩu đầu
	start := time.Now()
	cancel()
	deadline := time.After(time.Second)
	for open := true; open; {
		select {
		case _, open = <-ch:
		case <-deadline:
			t.Fatal("kênh không đóng sau khi huỷ")
		}
	}
	if d := time.Since(start); d > 200*time.Millisecond {
		t.Errorf("đóng sau %v, muốn ≤ 200 ms", d)
	}
	for i := 0; i < 50 && ctl.Active() != 0; i++ {
		time.Sleep(10 * time.Millisecond)
	}
	if ctl.Active() != 0 {
		t.Errorf("lời gọi nhà cung cấp còn chạy: %d", ctl.Active())
	}
	rows := cap.all(t, g)
	if len(rows) != 1 || rows[0].Status != "cancelled" {
		t.Errorf("audit = %+v, muốn 1 dòng cancelled", rows)
	}
}

func TestStreamFallsBackBeforeFirstToken(t *testing.T) {
	t.Parallel()
	a := &stub{name: "A", stream: func(context.Context, int, provider.ChatOpts) (<-chan provider.Delta, error) {
		ch := make(chan provider.Delta, 1)
		ch <- provider.Delta{Err: perr(provider.KindServer, 503)}
		close(ch)
		return ch, nil
	}}
	b := &stub{name: "B"}
	g, _ := newGateway(t, a, b)
	ch, err := g.Stream(t.Context(), llm.Request{Task: llm.TaskClassify, Messages: userMsg("x")})
	if err != nil {
		t.Fatal(err)
	}
	text, resp, err := drain(t, ch)
	if err != nil || text != "xin chào" || resp.FallbackIndex != 1 {
		t.Fatalf("text=%q resp=%+v err=%v", text, resp, err)
	}
}

func TestStreamMidFailure(t *testing.T) {
	t.Parallel()
	a := &stub{name: "A", stream: func(context.Context, int, provider.ChatOpts) (<-chan provider.Delta, error) {
		ch := make(chan provider.Delta, 3)
		ch <- provider.Delta{Text: "một nửa "}
		ch <- provider.Delta{Err: perr(provider.KindServer, 503)}
		close(ch)
		return ch, nil
	}}
	b := &stub{name: "B"}
	g, cap := newGateway(t, a, b)
	ch, err := g.Stream(t.Context(), llm.Request{Task: llm.TaskClassify, Messages: userMsg("x")})
	if err != nil {
		t.Fatal(err)
	}
	text, resp, err := drain(t, ch)
	if !errors.Is(err, llm.ErrStream) || resp != nil || text != "một nửa " {
		t.Fatalf("text=%q resp=%v err=%v: đã phát token thì KHÔNG chuyển dự phòng, kết thúc có lỗi", text, resp, err)
	}
	if b.calls.Load() != 0 {
		t.Error("không được gọi nhà dự phòng sau khi đã phát token")
	}
	if rows := cap.all(t, g); len(rows) != 1 || rows[0].Status != "error" || rows[0].ErrorKind != "SERVER" {
		t.Errorf("audit = %+v", rows)
	}
}
