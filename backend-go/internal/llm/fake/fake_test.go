package fake_test

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/edupilot/backend-go/internal/llm/fake"
	"github.com/edupilot/backend-go/internal/llm/provider"
)

func opts(q string) provider.ChatOpts {
	return provider.ChatOpts{Model: "fake-chat", Messages: []provider.Message{{Role: "user", Content: q}}, MaxTokens: 100}
}

func TestChatDeterministic(t *testing.T) {
	t.Parallel()
	p := fake.New(fake.NewController(fake.Settings{}), "")
	a, err := p.Chat(t.Context(), opts("Hạn nộp bài?"))
	b, _ := p.Chat(t.Context(), opts("Hạn nộp bài?"))
	if err != nil || a != b || !strings.Contains(a.Text, "Hạn nộp bài?") || a.TokensIn == 0 || a.TokensOut == 0 {
		t.Fatalf("a=%+v b=%+v err=%v", a, b, err)
	}
}

func TestEmbedDeterministicL2(t *testing.T) {
	t.Parallel()
	p := fake.New(fake.NewController(fake.Settings{}), "")
	v1, _, err := p.Embed(t.Context(), provider.EmbedOpts{Inputs: []string{"a", "b", "a"}})
	if err != nil || len(v1) != 3 || len(v1[0]) != 1536 {
		t.Fatalf("len=%d err=%v", len(v1), err)
	}
	var s float64
	for _, x := range v1[0] {
		s += float64(x) * float64(x)
	}
	if math.Abs(s-1) > 1e-4 {
		t.Errorf("|v|² = %v, muốn 1", s)
	}
	for i := range v1[0] {
		if v1[0][i] != v1[2][i] {
			t.Fatal("cùng chuỗi phải cùng vectơ")
		}
	}
	same := true
	for i := range v1[0] {
		same = same && v1[0][i] == v1[1][i]
	}
	if same {
		t.Error("chuỗi khác phải khác vectơ")
	}
	v2, _, _ := p.Embed(t.Context(), provider.EmbedOpts{Inputs: []string{"a"}, Dims: 8})
	if len(v2[0]) != 8 {
		t.Errorf("Dims=8 → %d", len(v2[0]))
	}
}

func TestStructuredFromSchema(t *testing.T) {
	t.Parallel()
	schema := json.RawMessage(`{"type":"object","properties":{"label":{"type":"string","enum":["x","y"]},"n":{"type":"integer","minimum":3},"ok":{"type":"boolean"},"items":{"type":"array","minItems":2,"items":{"type":"object","properties":{"k":{"type":"string"}}}}},"required":["label"]}`)
	p := fake.New(fake.NewController(fake.Settings{}), "")
	o := opts("x")
	o.Schema = schema
	r, err := p.Chat(t.Context(), o)
	if err != nil {
		t.Fatal(err)
	}
	var v struct {
		Label string           `json:"label"`
		N     int              `json:"n"`
		OK    bool             `json:"ok"`
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal([]byte(r.Text), &v); err != nil || v.Label != "x" || v.N != 3 || !v.OK || len(v.Items) != 2 {
		t.Fatalf("%s → %+v err=%v", r.Text, v, err)
	}
}

func TestStreamWordByWord(t *testing.T) {
	t.Parallel()
	p := fake.New(fake.NewController(fake.Settings{StreamDelay: 20 * time.Millisecond}), "")
	start := time.Now()
	ch, err := p.Stream(t.Context(), opts("một hai ba"))
	if err != nil {
		t.Fatal(err)
	}
	var words int
	var usage *provider.Result
	for d := range ch {
		if d.Usage != nil {
			usage = d.Usage
		} else {
			words++
		}
	}
	if words < 5 || usage == nil || time.Since(start) < 80*time.Millisecond {
		t.Errorf("words=%d usage=%v elapsed=%v: phải phát từng từ cách ~20 ms", words, usage, time.Since(start))
	}
}

func TestLatencyErrorRateAndKey(t *testing.T) {
	t.Parallel()
	c := fake.NewController(fake.Settings{LatencyMin: 40 * time.Millisecond, LatencyMax: 60 * time.Millisecond})
	p := fake.New(c, "")
	start := time.Now()
	if _, err := p.Chat(t.Context(), opts("x")); err != nil {
		t.Fatal(err)
	}
	if d := time.Since(start); d < 40*time.Millisecond || d > 200*time.Millisecond {
		t.Errorf("độ trễ %v ngoài 40–60 ms", d)
	}

	c.Set(fake.Settings{ErrorRate: 1, ErrorKind: provider.KindRateLimit, RetryAfter: 2 * time.Second})
	_, err := p.Chat(t.Context(), opts("x"))
	var pe *provider.Error
	if !errors.As(err, &pe) || pe.Kind != provider.KindRateLimit || pe.Status != 429 || pe.RetryAfter != 2*time.Second {
		t.Errorf("err = %v", err)
	}

	c.Set(fake.Settings{ValidKey: "good"})
	if _, err := fake.New(c, "bad").Chat(t.Context(), opts("x")); !errors.As(err, &pe) || pe.Kind != provider.KindAuth {
		t.Errorf("khoá sai: %v", err)
	}
	if _, err := fake.New(c, "good").Chat(t.Context(), opts("x")); err != nil {
		t.Errorf("khoá đúng: %v", err)
	}

	// tỉ lệ lỗi trung gian
	c.Set(fake.Settings{ErrorRate: 0.5})
	fails := 0
	for range 400 {
		if _, err := p.Chat(t.Context(), opts("x")); err != nil {
			fails++
		}
	}
	if fails < 140 || fails > 260 {
		t.Errorf("tỉ lệ lỗi 0,5 cho %d/400", fails)
	}
}

func TestReplayHitAndMiss(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	o := opts("hỏi thử")
	if err := fake.Record(dir, o, provider.Result{Text: "đã ghi", TokensIn: 3, TokensOut: 4}); err != nil {
		t.Fatal(err)
	}
	p := fake.New(fake.NewController(fake.Settings{ReplayDir: dir}), "")
	r, err := p.Chat(t.Context(), o)
	if err != nil || r.Text != "đã ghi" || r.TokensIn != 3 || r.TokensOut != 4 {
		t.Fatalf("hit: %+v %v", r, err)
	}
	o2 := opts("câu khác")
	_, err = p.Chat(t.Context(), o2)
	var pe *provider.Error
	if !errors.As(err, &pe) || pe.Kind != provider.KindReplayMiss {
		t.Fatalf("miss: %v", err)
	}
	if fake.ReplayKey(o) == fake.ReplayKey(o2) || len(fake.ReplayKey(o)) != 64 {
		t.Error("khoá phát lại phải là sha256 hex và khác nhau theo yêu cầu")
	}
	// schema cùng nghĩa khác thứ tự khoá → cùng khoá (chuẩn hoá)
	a, b := opts("s"), opts("s")
	a.Schema = json.RawMessage(`{"b":1,"a":2}`)
	b.Schema = json.RawMessage(`{ "a": 2, "b": 1 }`)
	if fake.ReplayKey(a) != fake.ReplayKey(b) {
		t.Error("schema phải được chuẩn hoá trước khi băm")
	}
}

func TestCtxCancelStopsCall(t *testing.T) {
	t.Parallel()
	c := fake.NewController(fake.Settings{LatencyMin: 5 * time.Second, LatencyMax: 5 * time.Second})
	p := fake.New(c, "")
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { _, err := p.Chat(ctx, opts("x")); done <- err }()
	time.Sleep(30 * time.Millisecond)
	if c.Active() != 1 {
		t.Errorf("active = %d", c.Active())
	}
	cancel()
	select {
	case err := <-done:
		var pe *provider.Error
		if !errors.As(err, &pe) || pe.Kind != provider.KindCancelled {
			t.Errorf("err = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("không dừng sau khi huỷ")
	}
	if c.Active() != 0 {
		t.Errorf("active = %d sau khi huỷ", c.Active())
	}
}

// TestFakeNoNetwork — US-P1-02 AC12: fake không gọi mạng. Thay mọi đường ra mạng bằng bẫy.
func TestFakeNoNetwork(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:1")
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:1")
	old := http.DefaultTransport
	oldDial := net.DefaultResolver.Dial
	var tripped bool
	http.DefaultTransport = roundTripFn(func(*http.Request) (*http.Response, error) { tripped = true; return nil, errors.New("cấm mạng") })
	net.DefaultResolver.Dial = func(context.Context, string, string) (net.Conn, error) {
		tripped = true
		return nil, errors.New("cấm mạng")
	}
	defer func() { http.DefaultTransport = old; net.DefaultResolver.Dial = oldDial }()

	p := fake.New(fake.NewController(fake.Settings{StreamDelay: time.Millisecond}), "")
	if _, err := p.Chat(t.Context(), opts("x")); err != nil {
		t.Fatal(err)
	}
	ch, _ := p.Stream(t.Context(), opts("a b"))
	for range ch {
	}
	if _, _, err := p.Embed(t.Context(), provider.EmbedOpts{Inputs: []string{"x"}}); err != nil {
		t.Fatal(err)
	}
	if tripped {
		t.Fatal("fake đã chạm mạng")
	}
}

type roundTripFn func(*http.Request) (*http.Response, error)

func (f roundTripFn) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// fake_calls của route thử stats: đếm theo từng provider fake, không lẫn nhau.
func TestCallsByProvider(t *testing.T) {
	t.Parallel()
	c := fake.NewController(fake.Settings{})
	a, b := fake.NewNamed(c, "A", ""), fake.NewNamed(c, "B", "")
	for range 3 {
		if _, err := a.Chat(t.Context(), opts("x")); err != nil {
			t.Fatal(err)
		}
	}
	ch, _ := b.Stream(t.Context(), opts("y"))
	for range ch {
	}
	if _, _, err := b.Embed(t.Context(), provider.EmbedOpts{Inputs: []string{"z"}}); err != nil { // nhúng không tính vào lời gọi sinh văn bản
		t.Fatal(err)
	}
	got := c.CallsByProvider()
	if got["A"] != 3 || got["B"] != 1 || c.Calls() != 4 {
		t.Errorf("CallsByProvider = %v, Calls = %d", got, c.Calls())
	}
}

// Schema kiểu nullable và danh sách đáp án {body, correct}: câu trắc nghiệm sinh ra có đúng một đáp án đúng (US-PE-09: seed 5 câu AI_DRAFT bằng provider fake).
func TestGenerateNullableAndOptions(t *testing.T) {
	raw, err := fake.Generate(json.RawMessage(`{"type":"object","properties":{"value":{"type":["boolean","null"]},"options":{"type":"array","items":{"type":"object","properties":{"body":{"type":"string"},"correct":{"type":"boolean"}}}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Value   *bool `json:"value"`
		Options []struct {
			Correct bool `json:"correct"`
		} `json:"options"`
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got.Value == nil || len(got.Options) != 2 || !got.Options[0].Correct || got.Options[1].Correct {
		t.Fatalf("sinh sai: %s", raw)
	}
}
