package llm_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/llm/provider"
	"github.com/edupilot/backend-go/internal/privacy"
)

var maskCourse = uuid.MustParse("00000000-0000-0000-0000-0000000000c1")

type rosterSrc struct{}

func (rosterSrc) Members(context.Context, uuid.UUID) ([]privacy.Member, error) {
	return []privacy.Member{{Name: "Nguyễn Văn An", Code: "ZZ99887766"}, {Name: "Lê Thị Bình"}}, nil
}

func realMasker(log *slog.Logger) *privacy.Masker {
	return &privacy.Masker{Detector: &privacy.Detector{Roster: &privacy.Roster{Src: rosterSrc{}}}, Log: log}
}

// maskGateway dựng Gateway có Masker thật (roster tĩnh, ánh xạ trong bộ nhớ) trên các provider stub.
func maskGateway(t *testing.T, log *slog.Logger, ps ...*stub) (*llm.Gateway, *capture) {
	t.Helper()
	cap := &capture{}
	aud := llm.NewAuditor(context.Background(), cap.write, log)
	t.Cleanup(func() { aud.Close(context.Background()) })
	var targets []llm.Target
	for _, p := range ps {
		targets = append(targets, llm.Target{ProviderID: p.name, ProviderName: p.name, Type: "fake", Model: "m-" + p.name,
			PriceIn: decimal.RequireFromString("1000"), PriceOut: decimal.RequireFromString("2000"), RPM: 60, TPM: 100000, P: p})
	}
	reg := llm.NewStaticRegistry(map[llm.Task]llm.Route{})
	for _, task := range append(llm.ChatTasks(), llm.TaskEmbedding) {
		reg.SetRoute(task, llm.Route{Targets: targets})
	}
	g := llm.New(llm.Options{Registry: reg, Auditor: aud, Log: log, Masker: realMasker(log),
		Sleep: func(ctx context.Context, d time.Duration) error { return ctx.Err() }, Rand: func() float64 { return 1 }})
	return g, cap
}

func withCourse(ctx context.Context) context.Context {
	c := maskCourse
	return llm.WithIdentity(ctx, llm.Identity{CourseID: &c})
}

const (
	leakName  = "Nguyễn Văn An"
	leakMSSV  = "20201234"
	leakEmail = "an@sv.edu.vn"
)

func leaks(s string) bool {
	return strings.Contains(s, leakName) || strings.Contains(s, "Nguyễn Văn An") || strings.Contains(strings.ToLower(s), "nguyen van an") ||
		strings.Contains(s, leakMSSV) || strings.Contains(s, leakEmail) || strings.Contains(s, "Lê Thị Bình")
}

// TestMaskCoversAllPayloadParts — AC2: system, lịch sử, tin hiện tại, kết quả tool, đầu vào nhúng đều đã che trước khi tới provider; 5 thành phần × 4 hàm.
func TestMaskCoversAllPayloadParts(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var embedded [][]string
	p := &stub{name: "p", embed: func(_ int, o provider.EmbedOpts) ([][]float32, int, error) {
		mu.Lock()
		embedded = append(embedded, append([]string(nil), o.Inputs...))
		mu.Unlock()
		v := make([]float32, llm.EmbedDims)
		v[0] = 1
		out := make([][]float32, len(o.Inputs))
		for i := range out {
			out[i] = v
		}
		return out, len(o.Inputs), nil
	}}
	p.chat = func(_ int, o provider.ChatOpts) (provider.Result, error) {
		if len(o.Schema) > 0 {
			return provider.Result{Text: `{"answer":"ok"}`, TokensIn: 1, TokensOut: 1}, nil
		}
		return provider.Result{Text: "ok", TokensIn: 1, TokensOut: 1}, nil
	}
	g, _ := maskGateway(t, newDiscardLog(), p)
	ctx := withCourse(t.Context())
	msgs := []llm.Message{
		{Role: "system", Content: "Bạn là trợ giảng. Người dùng là " + leakName + "."},
		{Role: "user", Content: "Em là " + leakName + " mssv " + leakMSSV},
		{Role: "assistant", Content: "Chào " + leakName + ", email " + leakEmail},
		{Role: "user", Content: "Kết quả tool: điểm của Lê Thị Bình là 9"},
	}
	schema := json.RawMessage(`{"type":"object","properties":{"answer":{"type":"string"}},"required":["answer"]}`)
	_, err := g.Chat(ctx, llm.Request{Task: llm.TaskChat, Messages: msgs})
	require.NoError(t, err)
	ch, err := g.Stream(ctx, llm.Request{Task: llm.TaskChat, Messages: msgs})
	require.NoError(t, err)
	_, _, _ = drain(t, ch)
	_, err = g.Structured(ctx, llm.Request{Task: llm.TaskUtility, Messages: msgs}, schema)
	require.NoError(t, err)
	_, err = g.Embed(ctx, llm.EmbedRequest{Inputs: []string{"câu hỏi của " + leakName, "mail " + leakEmail, "mssv " + leakMSSV}})
	require.NoError(t, err)

	p.mu.Lock()
	defer p.mu.Unlock()
	require.NotEmpty(t, p.seen)
	placeholders := 0
	for _, o := range p.seen {
		for _, m := range o.Messages {
			require.False(t, leaks(m.Content), "rò trong tin %s: %q", m.Role, m.Content)
			placeholders += strings.Count(m.Content, "[[")
		}
	}
	mu.Lock()
	defer mu.Unlock()
	require.Len(t, embedded, 1)
	for _, in := range embedded[0] {
		require.False(t, leaks(in), in)
		placeholders += strings.Count(in, "[[")
	}
	require.Greater(t, placeholders, 10, "payload có placeholder: kiểm không rỗng nghĩa")
}

// TestMaskNoCourseFailsClosed — AC2: không gắn lớp → MASK_FAILED, 0 payload tới provider; llm_audit một dòng status error / MASK_FAILED.
func TestMaskNoCourseFailsClosed(t *testing.T) {
	t.Parallel()
	p := &stub{name: "p"}
	g, cap := maskGateway(t, newDiscardLog(), p)
	ctx := t.Context() // không có CourseID
	for name, call := range map[string]func() error{
		"chat": func() error {
			_, err := g.Chat(ctx, llm.Request{Task: llm.TaskChat, Messages: userMsg("xin chào")})
			return err
		},
		"stream": func() error {
			_, err := g.Stream(ctx, llm.Request{Task: llm.TaskChat, Messages: userMsg("xin chào")})
			return err
		},
		"structured": func() error {
			_, err := g.Structured(ctx, llm.Request{Task: llm.TaskUtility, Messages: userMsg("x")}, json.RawMessage(`{"type":"object"}`))
			return err
		},
		"embed": func() error { _, err := g.Embed(ctx, llm.EmbedRequest{Inputs: []string{"x"}}); return err },
	} {
		require.ErrorIs(t, call(), llm.ErrMaskFailed, name)
	}
	require.Zero(t, p.calls.Load(), "0 payload tới provider")
	rows := cap.all(t, g)
	require.Len(t, rows, 4)
	for _, r := range rows {
		require.Equal(t, "error", r.Status)
		require.Equal(t, "MASK_FAILED", r.ErrorKind)
	}
}

// TestMaskOnceBeforeFallback — AC3: provider 1 lỗi, provider 2 nhận lời gọi lại; cả hai nhận payload đã che GIỐNG HỆT nhau; che đúng một lần.
func TestMaskOnceBeforeFallback(t *testing.T) {
	t.Parallel()
	p1 := &stub{name: "p1", chat: func(int, provider.ChatOpts) (provider.Result, error) {
		return provider.Result{}, perr(provider.KindAuth, 401)
	}}
	p2 := &stub{name: "p2"}
	g, cap := maskGateway(t, newDiscardLog(), p1, p2)
	resp, err := g.Chat(withCourse(t.Context()), llm.Request{Task: llm.TaskChat, Messages: userMsg("Em là " + leakName + " (" + leakMSSV + ")")})
	require.NoError(t, err)
	require.Equal(t, 1, resp.FallbackIndex)
	require.Equal(t, p1.seen[0].Messages, p2.seen[0].Messages)
	require.False(t, leaks(p1.seen[0].Messages[0].Content))
	rows := cap.all(t, g)
	require.Len(t, rows, 1)
	require.Equal(t, 2, rows[0].PIIMaskedCount, "một lần che: 2 thực thể, không nhân đôi theo số nhà cung cấp")
}

// TestUnmaskBeforeCaller / TestStructuredUnmaskAfterSchema — AC4.
func TestUnmaskBeforeCaller(t *testing.T) {
	t.Parallel()
	echo := func(_ int, o provider.ChatOpts) (provider.Result, error) {
		return provider.Result{Text: "Chào " + o.Messages[len(o.Messages)-1].Content + "!", TokensIn: 1, TokensOut: 1}, nil
	}
	p := &stub{name: "p", chat: echo, stream: func(_ context.Context, _ int, o provider.ChatOpts) (<-chan provider.Delta, error) {
		ch := make(chan provider.Delta, 16)
		text := "Chào " + o.Messages[len(o.Messages)-1].Content + "!"
		for _, r := range text { // từng rune: placeholder bị cắt ở mọi vị trí
			ch <- provider.Delta{Text: string(r)}
		}
		ch <- provider.Delta{Usage: &provider.Result{TokensIn: 1, TokensOut: 1}}
		close(ch)
		return ch, nil
	}}
	g, _ := maskGateway(t, newDiscardLog(), p)
	ctx := withCourse(t.Context())
	req := llm.Request{Task: llm.TaskChat, Messages: userMsg("nguyen van an")}
	resp, err := g.Chat(ctx, req)
	require.NoError(t, err)
	require.Equal(t, "Chào nguyen van an!", resp.Text, "Chat trả Response.Text đã khôi phục")
	ch, err := g.Stream(ctx, req)
	require.NoError(t, err)
	var got strings.Builder
	var done *llm.Response
	for c := range ch {
		require.NotContains(t, c.Text, "[[", "không bao giờ nhận placeholder")
		got.WriteString(c.Text)
		if c.Done {
			done = c.Response
		}
	}
	require.Equal(t, "Chào nguyen van an!", got.String())
	require.NotNil(t, done)
	require.Equal(t, "Chào nguyen van an!", done.Text)
}

func TestStructuredUnmaskAfterSchema(t *testing.T) {
	t.Parallel()
	p := &stub{name: "p", chat: func(_ int, o provider.ChatOpts) (provider.Result, error) {
		// mô hình trả JSON có placeholder trong giá trị chuỗi, cả lồng sâu; khoá giữ nguyên
		return provider.Result{Text: `{"answer":"Chào [[SV_1]]","items":[{"who":"[[SV_1]]"}]}`, TokensIn: 1, TokensOut: 1}, nil
	}}
	g, _ := maskGateway(t, newDiscardLog(), p)
	schema := json.RawMessage(`{"type":"object","required":["answer"],"properties":{"answer":{"type":"string"},"items":{"type":"array","items":{"type":"object","properties":{"who":{"type":"string"}}}}}}`)
	raw, err := g.Structured(withCourse(t.Context()), llm.Request{Task: llm.TaskUtility, Messages: userMsg("Nguyễn Văn An hỏi")}, schema)
	require.NoError(t, err)
	var out struct {
		Answer string                 `json:"answer"`
		Items  []struct{ Who string } `json:"items"`
	}
	require.NoError(t, json.Unmarshal(raw, &out))
	require.Equal(t, "Chào Nguyễn Văn An", out.Answer)
	require.Equal(t, "Nguyễn Văn An", out.Items[0].Who)
	// đầu ra sai schema vẫn bị từ chối trước khi khôi phục
	bad := &stub{name: "bad", chat: func(int, provider.ChatOpts) (provider.Result, error) {
		return provider.Result{Text: `{"khong_co_answer":"[[SV_1]]"}`}, nil
	}}
	g2, _ := maskGateway(t, newDiscardLog(), bad)
	_, err = g2.Structured(withCourse(t.Context()), llm.Request{Task: llm.TaskUtility, Messages: userMsg("Nguyễn Văn An hỏi")}, schema)
	require.Error(t, err)
}

// TestAuditPIIMaskedCount — AC5: pii_masked_count = tổng lần thay trong mọi tin nhắn của lời gọi; llm_audit không có nội dung.
func TestAuditPIIMaskedCount(t *testing.T) {
	t.Parallel()
	g, cap := maskGateway(t, newDiscardLog(), &stub{name: "p"})
	_, err := g.Chat(withCourse(t.Context()), llm.Request{Task: llm.TaskChat, Messages: []llm.Message{
		{Role: "system", Content: "Gọi người dùng là bạn."}, {Role: "user", Content: "Em " + leakName + " mssv " + leakMSSV}, {Role: "assistant", Content: "Chào " + leakName}}})
	require.NoError(t, err)
	_, err = g.Chat(withCourse(t.Context()), llm.Request{Task: llm.TaskChat, Messages: userMsg("không có gì cá nhân")})
	require.NoError(t, err)
	rows := cap.all(t, g)
	require.Len(t, rows, 2)
	require.Equal(t, 3, rows[0].PIIMaskedCount)
	require.Zero(t, rows[1].PIIMaskedCount)
	raw, _ := json.Marshal(rows)
	require.False(t, leaks(string(raw)), "llm_audit không có nội dung")
}

// TestProviderErrorsNotLogged — AC8: lỗi provider có thân phản hồi / hết hạn → log (cả debug) không chứa họ tên, MSSV, nội dung, ánh xạ.
func TestProviderErrorsNotLogged(t *testing.T) {
	t.Parallel()
	buf := &bytes.Buffer{}
	log := slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	for _, kind := range []provider.Kind{provider.KindServer, provider.KindBadRequest, provider.KindTimeout} {
		p := &stub{name: "p", chat: func(_ int, o provider.ChatOpts) (provider.Result, error) {
			// provider thật hay trả lại nội dung đã nhận trong thân lỗi; ở đây nội dung đã được che nên không có gì để lộ
			return provider.Result{}, &provider.Error{Kind: kind, Status: 500, Detail: o.Messages[0].Content}
		}}
		g, _ := maskGateway(t, log, p)
		_, _ = g.Chat(withCourse(t.Context()), llm.Request{Task: llm.TaskChat, Messages: userMsg("Em " + leakName + " mssv " + leakMSSV + " mail " + leakEmail)})
	}
	require.NotEmpty(t, buf.String(), "phải có log để kiểm có nghĩa")
	require.False(t, leaks(buf.String()), buf.String())
}

// TestValidateRequiresMasker — AC7 (nửa gói llm): thiếu Masker mà không NoMask → lỗi cấu hình; mọi lời gọi đều bị từ chối.
func TestValidateRequiresMasker(t *testing.T) {
	t.Parallel()
	require.Error(t, llm.Options{}.Validate())
	require.NoError(t, llm.Options{NoMask: true}.Validate())
	require.NoError(t, llm.Options{Masker: realMasker(newDiscardLog())}.Validate())
	p := &stub{name: "p"}
	reg := llm.NewStaticRegistry(map[llm.Task]llm.Route{})
	reg.SetRoute(llm.TaskChat, llm.Route{Targets: []llm.Target{{ProviderID: "p", ProviderName: "p", Type: "fake", Model: "m", RPM: 60, TPM: 1000, P: p}}})
	g := llm.New(llm.Options{Registry: reg, Log: newDiscardLog()}) // không Masker, không NoMask
	_, err := g.Chat(withCourse(t.Context()), llm.Request{Task: llm.TaskChat, Messages: userMsg("x")})
	require.Error(t, err)
	require.Zero(t, p.calls.Load())
}

// TestMaskOnlyInLLMGateway — AC1: ký hiệu Mask / Unmask / NewStreamUnmasker của privacy chỉ được GỌI ở internal/llm (và dây nối cmd/*, llmrt).
func TestMaskOnlyInLLMGateway(t *testing.T) {
	t.Parallel()
	root := filepath.Join("..", "..")
	banned := map[string]bool{"Mask": true, "Unmask": true, "NewStreamUnmasker": true, "UnmaskStream": true}
	allowedDir := func(rel string) bool {
		rel = filepath.ToSlash(rel)
		return strings.HasPrefix(rel, "internal/llm/") || strings.HasPrefix(rel, "internal/privacy/") || strings.HasPrefix(rel, "cmd/")
	}
	fset := token.NewFileSet()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		if allowedDir(rel) || strings.HasPrefix(filepath.ToSlash(rel), "legacy/") {
			return nil
		}
		src, rerr := os.ReadFile(path)
		require.NoError(t, rerr)
		f, perr := parser.ParseFile(fset, path, src, 0)
		require.NoError(t, perr)
		importsPrivacy := false
		for _, im := range f.Imports {
			if strings.HasSuffix(strings.Trim(im.Path.Value, `"`), "/internal/privacy") {
				importsPrivacy = true
			}
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if se, ok := n.(*ast.SelectorExpr); ok && banned[se.Sel.Name] && importsPrivacy {
				t.Errorf("%s gọi %s: chỉ internal/llm được che / khôi phục", rel, se.Sel.Name)
			}
			return true
		})
		return nil
	})
	require.NoError(t, err)
}

var _ = errors.New
