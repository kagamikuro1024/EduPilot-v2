package provider

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/ssestream"
	"github.com/openai/openai-go/v3/shared"
)

// Địa chỉ gốc mặc định theo loại (SRS 4.2).
const (
	BaseOpenAI    = "https://api.openai.com/v1"
	BaseAnthropic = "https://api.anthropic.com/v1/"
	BaseGemini    = "https://generativelanguage.googleapis.com/v1beta/openai/"
)

// BaseURLFor trả địa chỉ gốc của loại nhà cung cấp; openai_compatible dùng `override`.
func BaseURLFor(typ, override string) string {
	if override != "" {
		return override
	}
	switch typ {
	case "openai":
		return BaseOpenAI
	case "anthropic":
		return BaseAnthropic
	case "gemini":
		return BaseGemini
	}
	return override
}

// OpenAIConfig dựng một client qua lớp tương thích OpenAI.
type OpenAIConfig struct {
	Type    string // openai | anthropic | gemini | openai_compatible
	BaseURL string // rỗng = mặc định theo Type
	APIKey  string
	HTTP    *http.Client // tuỳ chọn (test)
}

type oaClient struct {
	cl   openai.Client
	typ  string
	key  string
	base string
}

// NewOpenAI dựng provider. WithMaxRetries(0) là BẮT BUỘC: SDK mặc định tự thử lại 2 lần và có thể chờ tới 2 phút theo Retry-After,
// chồng lên thử lại của Gateway và làm mất status 429 khi hết hạn (docs/research/2026-10-03-openai-go-compat.md).
func NewOpenAI(c OpenAIConfig) Provider {
	base := BaseURLFor(c.Type, c.BaseURL)
	opts := []option.RequestOption{option.WithBaseURL(base), option.WithMaxRetries(0)}
	if c.APIKey != "" {
		opts = append(opts, option.WithAPIKey(c.APIKey))
	} else {
		opts = append(opts, option.WithAPIKey("none")) // openai_compatible có thể không cần khoá; SDK đòi chuỗi khác rỗng
	}
	hc := c.HTTP
	if hc == nil {
		// Không theo chuyển hướng: base_url do ADMIN đặt, chuyển hướng 3xx sang địa chỉ khác là đường SSRF (SRS 4.2, Q-QC-P102-1).
		hc = &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	opts = append(opts, option.WithHTTPClient(hc))
	return &oaClient{cl: openai.NewClient(opts...), typ: c.Type, key: c.APIKey, base: base}
}

func (p *oaClient) fail(err error) error { return classify(err, p.key) }

func (p *oaClient) params(o ChatOpts) openai.ChatCompletionNewParams {
	msgs := make([]openai.ChatCompletionMessageParamUnion, 0, len(o.Messages)+1)
	for _, m := range o.Messages {
		switch m.Role {
		case "system":
			msgs = append(msgs, openai.SystemMessage(m.Content))
		case "assistant":
			msgs = append(msgs, openai.AssistantMessage(m.Content))
		default:
			msgs = append(msgs, openai.UserMessage(m.Content))
		}
	}
	prm := openai.ChatCompletionNewParams{Model: shared.ChatModel(o.Model), Messages: msgs}
	if o.Temperature != nil {
		t := *o.Temperature
		if p.typ == "anthropic" || p.typ == "gemini" {
			t = math.Min(t, 1) // 0–1 ở hai nhà này (OpenAI cho tới 2)
		}
		prm.Temperature = openai.Float(t)
	}
	if o.MaxTokens > 0 {
		// OpenAI / Anthropic: max_completion_tokens; Gemini và máy chủ tương thích: max_tokens (research mục "Ảnh hưởng").
		if p.typ == "openai" || p.typ == "anthropic" {
			prm.MaxCompletionTokens = openai.Int(int64(o.MaxTokens))
		} else {
			prm.MaxTokens = openai.Int(int64(o.MaxTokens))
		}
	}
	if o.Fast && p.typ == "gemini" {
		prm.ReasoningEffort = shared.ReasoningEffort("low")
	}
	if len(o.Schema) > 0 {
		p.applySchema(&prm, o.Schema)
	}
	return prm
}

const toolName = "emit_result"

// applySchema chọn cách ép đầu ra có cấu trúc THEO LOẠI (cố định, một lời gọi — D47): openai/gemini → json_schema strict;
// anthropic bỏ qua response_format nên dùng tool bắt buộc; openai_compatible → json_object kèm schema trong lời nhắc.
func (p *oaClient) applySchema(prm *openai.ChatCompletionNewParams, schema json.RawMessage) {
	var sch map[string]any
	_ = json.Unmarshal(schema, &sch)
	switch p.typ {
	case "openai", "gemini":
		prm.ResponseFormat = openai.ChatCompletionNewParamsResponseFormatUnion{OfJSONSchema: &shared.ResponseFormatJSONSchemaParam{
			JSONSchema: shared.ResponseFormatJSONSchemaJSONSchemaParam{Name: "result", Schema: sch, Strict: openai.Bool(p.typ == "openai")},
		}}
	case "anthropic":
		prm.Tools = []openai.ChatCompletionToolUnionParam{openai.ChatCompletionFunctionTool(shared.FunctionDefinitionParam{
			Name: toolName, Description: openai.String("Trả kết quả theo đúng schema."), Parameters: shared.FunctionParameters(sch),
		})}
		prm.ToolChoice = openai.ChatCompletionToolChoiceOptionUnionParam{OfFunctionToolChoice: &openai.ChatCompletionNamedToolChoiceParam{
			Function: openai.ChatCompletionNamedToolChoiceFunctionParam{Name: toolName},
		}}
	default:
		prm.ResponseFormat = openai.ChatCompletionNewParamsResponseFormatUnion{OfJSONObject: &shared.ResponseFormatJSONObjectParam{}}
		prm.Messages = append([]openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage("Chỉ trả về MỘT đối tượng JSON hợp lệ theo đúng JSON Schema sau, không thêm chữ nào khác:\n" + string(schema)),
		}, prm.Messages...)
	}
}

func estimate(s string) int { return (len(s) + 3) / 4 }

// Chat gọi sinh văn bản một lần.
func (p *oaClient) Chat(ctx context.Context, o ChatOpts) (Result, error) {
	res, err := p.cl.Chat.Completions.New(ctx, p.params(o))
	if err != nil {
		return Result{}, p.fail(err)
	}
	if len(res.Choices) == 0 {
		return Result{}, &Error{Kind: KindBadResponse}
	}
	msg := res.Choices[0].Message
	text := msg.Content
	if len(o.Schema) > 0 && p.typ == "anthropic" {
		if len(msg.ToolCalls) == 0 {
			return Result{}, &Error{Kind: KindBadResponse}
		}
		text = msg.ToolCalls[0].Function.Arguments
	}
	out := Result{Text: text, TokensIn: int(res.Usage.PromptTokens), TokensOut: int(res.Usage.CompletionTokens)}
	if res.Usage.TotalTokens == 0 { // nhà cung cấp không trả usage
		out.TokensOut, out.Estimated = estimate(text), true
	}
	return out, nil
}

// Stream phát từng mẩu văn bản; luôn bật include_usage và bỏ qua chunk không có choices (chỉ lấy usage).
func (p *oaClient) Stream(ctx context.Context, o ChatOpts) (<-chan Delta, error) {
	prm := p.params(o)
	prm.StreamOptions = openai.ChatCompletionStreamOptionsParam{IncludeUsage: openai.Bool(true)}
	st := p.cl.Chat.Completions.NewStreaming(ctx, prm)
	out := make(chan Delta, 16)
	go func() {
		defer close(out)
		defer func() { _ = st.Close() }()
		var text strings.Builder
		var usage *Result
		for st.Next() {
			ch := st.Current()
			if len(ch.Choices) > 0 && ch.Choices[0].Delta.Content != "" {
				text.WriteString(ch.Choices[0].Delta.Content)
				if !send(ctx, out, Delta{Text: ch.Choices[0].Delta.Content}) {
					return
				}
			}
			if ch.Usage.TotalTokens > 0 {
				usage = &Result{TokensIn: int(ch.Usage.PromptTokens), TokensOut: int(ch.Usage.CompletionTokens)}
			}
		}
		if err := st.Err(); err != nil {
			send(ctx, out, Delta{Err: p.fail(err)})
			return
		}
		est := false
		if usage == nil {
			usage, est = &Result{TokensOut: estimate(text.String())}, true
		}
		send(ctx, out, Delta{Usage: usage, Estimated: est})
	}()
	return out, nil
}

func send(ctx context.Context, ch chan<- Delta, d Delta) bool {
	select {
	case ch <- d:
		return true
	case <-ctx.Done():
		return false
	}
}

// Embed nhúng một lô chuỗi; gửi `dimensions` khi Dims > 0 (Gemini: chuẩn hoá L2 vì gemini-embedding-001 không tự chuẩn hoá).
func (p *oaClient) Embed(ctx context.Context, o EmbedOpts) ([][]float32, int, error) {
	prm := openai.EmbeddingNewParams{Model: openai.EmbeddingModel(o.Model),
		Input: openai.EmbeddingNewParamsInputUnion{OfArrayOfStrings: o.Inputs}}
	if o.Dims > 0 {
		prm.Dimensions = openai.Int(int64(o.Dims))
	}
	res, err := p.cl.Embeddings.New(ctx, prm)
	if err != nil {
		return nil, 0, p.fail(err)
	}
	if len(res.Data) != len(o.Inputs) {
		return nil, 0, &Error{Kind: KindBadResponse}
	}
	out := make([][]float32, len(res.Data))
	for _, d := range res.Data {
		if d.Index < 0 || int(d.Index) >= len(out) {
			return nil, 0, &Error{Kind: KindBadResponse}
		}
		v := make([]float32, len(d.Embedding))
		for i, f := range d.Embedding {
			v[i] = float32(f)
		}
		if p.typ == "gemini" {
			Normalize(v)
		}
		out[d.Index] = v
	}
	return out, int(res.Usage.PromptTokens), nil
}

// Normalize chuẩn hoá L2 tại chỗ.
func Normalize(v []float32) {
	var s float64
	for _, x := range v {
		s += float64(x) * float64(x)
	}
	if s == 0 {
		return
	}
	n := float32(math.Sqrt(s))
	for i := range v {
		v[i] /= n
	}
}

// classify ánh xạ lỗi SDK / mạng về Kind. secrets: chuỗi cần bôi khỏi Detail.
func classify(err error, secrets ...string) error {
	var pe *Error
	if errors.As(err, &pe) {
		return pe
	}
	var ae *openai.Error
	if errors.As(err, &ae) {
		raw := ae.RawJSON()
		e := &Error{Status: ae.StatusCode}
		if ae.Response != nil {
			e.RetryAfter = retryAfter(ae.Response.Header)
			if b := bodyOf(ae.Response); b != "" { // SDK nạp lại thân; dạng mảng kiểu Google / HTML không nằm trong RawJSON
				raw = b
			}
		}
		e.Detail = Redact(raw, secrets...)
		switch s := ae.StatusCode; {
		case s == http.StatusUnauthorized || s == http.StatusForbidden:
			e.Kind = KindAuth
		case s == http.StatusNotFound:
			e.Kind = KindModelNotFound
		case s == http.StatusTooManyRequests:
			e.Kind = KindRateLimit
		case s == http.StatusRequestTimeout:
			e.Kind = KindTimeout
		case s >= 500:
			e.Kind = KindServer
		case s == http.StatusBadRequest && looksLikeAuth(raw):
			e.Kind = KindAuth // Gemini trả 400 khi khoá sai
		default:
			e.Kind = KindBadRequest
		}
		return e
	}
	var se *ssestream.StreamError
	if errors.As(err, &se) {
		return &Error{Kind: KindServer, Detail: Redact(err.Error(), secrets...)}
	}
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return &Error{Kind: KindTimeout}
	case errors.Is(err, context.Canceled):
		return &Error{Kind: KindCancelled}
	}
	var ne net.Error
	var ue *url.Error
	switch {
	case errors.As(err, &ne) && ne.Timeout(), errors.Is(err, os.ErrDeadlineExceeded):
		return &Error{Kind: KindTimeout}
	case errors.As(err, &ne), errors.As(err, &ue):
		return &Error{Kind: KindNetwork}
	}
	return &Error{Kind: KindBadResponse, Detail: Redact(err.Error(), secrets...)} // SDK không giải mã được phản hồi (kể cả 3xx không theo)
}

// bodyOf đọc tối đa 4 KiB thân đã được SDK nạp lại.
func bodyOf(res *http.Response) string {
	if res == nil || res.Body == nil {
		return ""
	}
	b, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
	return string(b)
}

func looksLikeAuth(raw string) bool {
	l := strings.ToLower(raw)
	return strings.Contains(l, "api key") || strings.Contains(l, "api_key") || strings.Contains(l, "api-key")
}

// retryAfter đọc `Retry-After-Ms` rồi `Retry-After` (giây); ngày giờ HTTP bị bỏ qua.
func retryAfter(h http.Header) time.Duration {
	if v := h.Get("Retry-After-Ms"); v != "" {
		if ms, err := strconv.ParseFloat(v, 64); err == nil && ms >= 0 {
			return time.Duration(ms * float64(time.Millisecond))
		}
	}
	if v := h.Get("Retry-After"); v != "" {
		if s, err := strconv.ParseFloat(v, 64); err == nil && s >= 0 {
			return time.Duration(s * float64(time.Second))
		}
	}
	return 0
}
