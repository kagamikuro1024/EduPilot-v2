package llmhttp

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/llm/provider"
	"github.com/edupilot/backend-go/internal/llmconfig"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
)

const (
	probeTimeout = 10 * time.Second // hạn tuyệt đối của một lần kiểm tra kết nối (SRS 8.2)
	testPerMin   = 10               // giới hạn Test / phút / người (`ep:llm:test:{user}`, SRS 5.5)
	testWindow   = time.Minute
)

// Loại lỗi của Test (SRS 6.3).
const (
	kAuth     = "AUTH"
	kNetwork  = "NETWORK"
	kTimeout  = "TIMEOUT"
	kRate     = "RATE_LIMIT"
	kModel    = "MODEL_NOT_FOUND"
	kBadResp  = "BAD_RESPONSE"
	kDimsMism = "DIMS_MISMATCH"
)

// testMessage là câu tiếng Việt theo loại lỗi. Thân phản hồi của nhà cung cấp KHÔNG bao giờ ra ngoài — chỉ câu này (US-P1-04 AC13).
func testMessage(kind string) string {
	switch kind {
	case kAuth:
		return "Khoá API không được nhà cung cấp chấp nhận. Kiểm tra lại khoá."
	case kTimeout:
		return "Nhà cung cấp phản hồi quá chậm."
	case kRate:
		return "Nhà cung cấp đang giới hạn tốc độ. Thử lại sau."
	case kModel:
		return "Nhà cung cấp không có mô hình này."
	case kBadResp:
		return "Nhà cung cấp trả về dữ liệu không đúng định dạng."
	case kDimsMism:
		return "Mô hình nhúng không trả về 1536 chiều. Chọn mô hình khác."
	}
	return "Không kết nối được tới nhà cung cấp. Kiểm tra địa chỉ và mạng."
}

// candidate là mô hình dùng để kiểm tra kết nối.
type candidate struct{ model, kind string }

// pickInput chọn mô hình bật đầu tiên, ưu tiên chat (một lời gọi max_tokens=1 rẻ hơn), rồi tới embedding.
func pickInput(ms []llmconfig.ModelInput) *candidate {
	var emb *candidate
	for _, m := range ms {
		if m.Enabled != nil && !*m.Enabled {
			continue
		}
		if m.Kind == "chat" {
			return &candidate{m.Model, m.Kind}
		}
		if emb == nil {
			emb = &candidate{m.Model, m.Kind}
		}
	}
	return emb
}

func pickStored(ms []llmconfig.Model) *candidate {
	in := make([]llmconfig.ModelInput, 0, len(ms))
	for _, m := range ms {
		e := m.Enabled
		in = append(in, llmconfig.ModelInput{Model: m.Model, Kind: m.Kind, Enabled: &e})
	}
	return pickInput(in)
}

type outcome struct {
	OK      bool
	Kind    string
	Latency time.Duration
}

// kindOf chuẩn hoá lỗi của nhà cung cấp về 7 loại của Test. `gemini` trả 400 nhắc "API key" khi khoá sai → AUTH (CHỈ ở Test; SRS 4.2).
func kindOf(err error, typ string) string {
	var pe *provider.Error
	if !errors.As(err, &pe) {
		if errors.Is(err, context.DeadlineExceeded) {
			return kTimeout
		}
		return kNetwork
	}
	switch pe.Kind {
	case provider.KindAuth:
		return kAuth
	case provider.KindModelNotFound:
		return kModel
	case provider.KindRateLimit:
		return kRate
	case provider.KindTimeout:
		return kTimeout
	case provider.KindBadResponse:
		return kBadResp
	case provider.KindDimsMismatch:
		return kDimsMism
	case provider.KindBadRequest:
		d := strings.ToLower(pe.Detail)
		if typ == "gemini" && (strings.Contains(d, "api key") || strings.Contains(d, "api_key")) {
			return kAuth
		}
		return kBadResp
	}
	return kNetwork
}

// probe gọi MỘT lời gọi nhỏ trực tiếp tới nhà cung cấp (không qua Scheduler: không tranh chỗ của chat): Chat max_tokens=1 hoặc Embed "ping".
func (a *Admin) probe(ctx context.Context, typ string, baseURL *string, key string, c candidate) outcome {
	ctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	start := a.Clock.Now()
	done := func(kind string) outcome {
		return outcome{OK: kind == "", Kind: kind, Latency: a.Clock.Now().Sub(start)}
	}
	spec := llm.ProviderSpec{ID: "probe", Name: "probe", Type: typ, Key: key}
	if baseURL != nil {
		spec.BaseURL = strings.TrimSpace(*baseURL)
	}
	p, err := a.RT.Registry.Build(spec)
	if err != nil {
		return done(kNetwork)
	}
	if c.kind == "embedding" {
		vecs, _, err := p.Embed(ctx, provider.EmbedOpts{Model: c.model, Inputs: []string{"ping"}, Dims: llmconfig.EmbedDims})
		if err != nil {
			return done(kindOf(err, typ))
		}
		if len(vecs) != 1 || len(vecs[0]) != llmconfig.EmbedDims {
			return done(kDimsMism)
		}
		return done("")
	}
	if _, err := p.Chat(ctx, provider.ChatOpts{Model: c.model, Messages: []provider.Message{{Role: "user", Content: "ping"}}, MaxTokens: 1}); err != nil {
		return done(kindOf(err, typ))
	}
	return done("")
}

// verifyBeforeSave kiểm tra kết nối trước khi lưu (SRS 3.3). Sai → ghi 422 VALIDATION_FAILED và trả false: KHÔNG ghi gì.
func (a *Admin) verifyBeforeSave(w http.ResponseWriter, r *http.Request, typ string, baseURL *string, key string, c *candidate) bool {
	if c == nil {
		apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "models", Code: "REQUIRED", Message: "Cần ít nhất một mô hình đang bật để kiểm tra kết nối."}))
		return false
	}
	out := a.probe(r.Context(), typ, baseURL, key, *c)
	if out.OK {
		return true
	}
	f := apierr.FieldError{Field: "base_url", Code: "PROVIDER_UNREACHABLE", Message: testMessage(out.Kind)}
	switch out.Kind {
	case kAuth:
		f.Field, f.Code = "api_key", "PROVIDER_AUTH_FAILED"
	case kModel:
		f.Field, f.Code = "models", "MODEL_NOT_FOUND"
	case kDimsMism:
		f.Field, f.Code = "models", "DIMS_MISMATCH"
	}
	apierr.Write(w, r, apierr.Validation(f))
	return false
}

// allowTest đếm lần Test của người gọi (cửa sổ 60 s). Quá 10 → 429 RATE_LIMITED + Retry-After và trả false. Redis lỗi → cho qua (log).
func (a *Admin) allowTest(w http.ResponseWriter, r *http.Request) bool {
	p, _ := auth.FromContext(r.Context())
	ctx := r.Context()
	key := appredis.Key("llm", "test", p.Sub)
	pipe := a.Redis.TxPipeline()
	n := pipe.Incr(ctx, key)
	pipe.ExpireNX(ctx, key, testWindow)
	ttl := pipe.TTL(ctx, key)
	if _, err := pipe.Exec(ctx); err != nil {
		a.Log.WarnContext(ctx, "không đếm được lượt Test LLM (Redis); cho qua", "error", err.Error())
		return true
	}
	if n.Val() <= testPerMin {
		return true
	}
	wait := int((ttl.Val() + time.Second - 1) / time.Second)
	apierr.Write(w, r, apierr.New(http.StatusTooManyRequests, apierr.RateLimited).WithRetryAfter(max(1, wait)))
	return false
}

// decodeOptional đọc thân JSON nếu có; thân rỗng là hợp lệ (Test bản ghi đã lưu không cần ghi đè).
func decodeOptional(w http.ResponseWriter, r *http.Request, dst any) bool {
	b, err := io.ReadAll(r.Body)
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			apierr.Write(w, r, apierr.New(http.StatusRequestEntityTooLarge, apierr.PayloadTooLarge))
		} else {
			apierr.Write(w, r, apierr.New(http.StatusBadRequest, apierr.BadRequest))
		}
		return false
	}
	if len(bytes.TrimSpace(b)) == 0 {
		return true
	}
	r.Body = io.NopCloser(bytes.NewReader(b))
	return httpx.DecodeJSON(w, r, dst)
}

func baseHostOf(typ string, baseURL *string) string {
	raw := ""
	if baseURL != nil {
		raw = strings.TrimSpace(*baseURL)
	}
	u := provider.BaseURLFor(typ, raw)
	return llmconfig.BaseHost(&u)
}

// finishTest ghi audit_log (host, không khoá / đường dẫn) rồi trả kết quả. providerID nil = khoá ứng viên chưa lưu.
func (a *Admin) finishTest(w http.ResponseWriter, r *http.Request, providerID *uuid.UUID, typ string, baseURL *string, c candidate, out outcome) {
	ctx := r.Context()
	if err := a.RT.Config.AuditTest(ctx, providerID, typ, baseHostOf(typ, baseURL), c.model, out.OK, out.Kind); err != nil {
		a.Log.ErrorContext(ctx, "không ghi được audit_log của Test LLM", "error", err.Error())
	}
	res := testResultView{OK: out.OK, LatencyMs: out.Latency.Milliseconds()}
	if !out.OK {
		res.ErrorKind, res.Message = out.Kind, testMessage(out.Kind)
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

func errNoCandidate() *apierr.Error {
	return apierr.Validation(apierr.FieldError{Field: "model", Code: "REQUIRED", Message: "Chọn một mô hình để kiểm tra kết nối."})
}

// testSaved: POST /admin/llm/providers/{id}/test — thử bản ghi đã lưu; thân tuỳ chọn {api_key?, base_url?, model?} ghi đè, KHÔNG lưu.
func (a *Admin) testSaved(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		notFound(w, r)
		return
	}
	if !a.allowTest(w, r) {
		return
	}
	var in struct {
		APIKey  *llmconfig.Secret `json:"api_key"`
		BaseURL *string           `json:"base_url"`
		Model   string            `json:"model"`
	}
	if !decodeOptional(w, r, &in) {
		return
	}
	ctx := r.Context()
	p, err := a.RT.Config.GetProvider(ctx, id)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	base := p.BaseURL
	if in.BaseURL != nil {
		if err := llmconfig.ValidateBaseURL(*in.BaseURL); err != nil {
			a.fail(w, r, err)
			return
		}
		base = in.BaseURL
	}
	key := ""
	if in.APIKey != nil {
		key = in.APIKey.Reveal()
	} else if k, kerr := a.RT.Resolver.DecryptKey(ctx, id); kerr == nil {
		key = k // thiếu / không giải mã được → thử với khoá rỗng: nhà cung cấp tự báo AUTH
	}
	c := pickStored(p.Models)
	if in.Model != "" {
		c = &candidate{model: in.Model, kind: "chat"}
		for _, m := range p.Models {
			if m.Model == in.Model {
				c.kind = m.Kind
			}
		}
	}
	if c == nil {
		apierr.Write(w, r, errNoCandidate())
		return
	}
	out := a.probe(ctx, p.Type, base, key, *c)
	if in.APIKey == nil && in.BaseURL == nil && in.Model == "" { // chỉ kết quả của cấu hình ĐÃ LƯU mới vào last_test
		if err := a.RT.Config.RecordTest(ctx, id, out.OK, out.Kind); err != nil {
			a.Log.WarnContext(ctx, "không lưu được kết quả Test", "error", err.Error())
		}
	}
	a.finishTest(w, r, &id, p.Type, base, *c, out)
}

// testCandidate: POST /admin/llm/providers/test — thử khoá ứng viên chưa lưu (SRS 6.2 #6). Không ghi bản ghi nào.
func (a *Admin) testCandidate(w http.ResponseWriter, r *http.Request) {
	if !a.allowTest(w, r) {
		return
	}
	var in struct {
		Type    string            `json:"type" validate:"required,oneof=openai anthropic gemini openai_compatible fake"`
		BaseURL *string           `json:"base_url"`
		APIKey  *llmconfig.Secret `json:"api_key"`
		Model   string            `json:"model" validate:"required"`
		Kind    string            `json:"kind" validate:"omitempty,oneof=chat embedding"`
	}
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	if in.BaseURL != nil && strings.TrimSpace(*in.BaseURL) != "" {
		if err := llmconfig.ValidateBaseURL(*in.BaseURL); err != nil {
			a.fail(w, r, err)
			return
		}
	} else if in.Type == "openai_compatible" {
		apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "base_url", Code: "REQUIRED", Message: "Nhà cung cấp tương thích OpenAI cần địa chỉ gốc."}))
		return
	}
	key := ""
	if in.APIKey != nil {
		key = in.APIKey.Reveal()
	}
	c := candidate{model: in.Model, kind: "chat"}
	if in.Kind != "" {
		c.kind = in.Kind
	}
	out := a.probe(r.Context(), in.Type, in.BaseURL, key, c)
	a.finishTest(w, r, nil, in.Type, in.BaseURL, c, out)
}
