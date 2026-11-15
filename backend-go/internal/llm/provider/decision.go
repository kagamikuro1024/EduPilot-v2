package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// Decider là phần mở rộng tuỳ chọn của Provider cho decision model (D60): trả xác suất cho câu hỏi đóng, không sinh chữ.
// Chỉ provider loại `decision` (và fake) cài; gateway hỏi bằng type-assert nên mọi provider sinh chữ giữ nguyên.
type Decider interface {
	Decide(ctx context.Context, o DecideOpts) (DecideResult, error)
}

// Loại câu hỏi trên dây.
const (
	QuestionNoul   = "noul"   // có / không: trả P(có)
	QuestionChoice = "choice" // chọn một nhãn: trả nhãn + độ tin cậy
	QuestionScore  = "score"  // khai báo cho đủ hợp đồng; 10.5 không dùng (D60 (5))
)

// DecideQuestion là MỘT câu hỏi. Criteria: `map[string]string` (nhãn → mô tả) cho choice, `[]string` cho score, nil cho noul.
type DecideQuestion struct {
	Type         string
	Instructions string
	Criteria     any
}

// DecideOpts là một lời gọi decision tới MỘT mô hình. Input phải đã được che PII.
type DecideOpts struct {
	Model     string
	Input     string
	Questions map[string]DecideQuestion
}

// DecideAnswer là câu trả lời của một câu hỏi. noul: Noul ∈ [0,01; 0,99], Confidence = max(p, 1−p);
// choice: Choice + Probabilities, Confidence = xác suất của nhãn đứng đầu.
type DecideAnswer struct {
	Type          string
	Choice        string
	Noul          float64
	Confidence    float64
	Probabilities map[string]float64
}

// DecideResult là phản hồi của một lời gọi decision (không có token ra tính tiền).
type DecideResult struct {
	Answers  map[string]DecideAnswer
	TokensIn int
}

// DecisionBaseURL là điểm gọi mặc định của `gemini-decision-1`.
const DecisionBaseURL = "https://generativelanguage.googleapis.com/v1beta"

// DecisionConfig dựng provider loại `decision`.
type DecisionConfig struct {
	BaseURL string // rỗng → DecisionBaseURL
	APIKey  string
	HTTP    *http.Client // tuỳ chọn (test)
}

type decision struct {
	base string
	key  string
	cl   *http.Client
}

// NewDecision dựng provider `decision`: `POST {base}/decisions` bằng net/http (cấm SDK, D60 (1)), khoá ở header `x-goog-api-key` (không ở URL).
// Chat / Stream / Embed trả BAD_REQUEST: loại này chỉ dùng qua Decide. Client không theo chuyển hướng (chống SSRF), không Timeout: hạn chót lấy từ ctx.
func NewDecision(c DecisionConfig) Provider {
	base := strings.TrimRight(c.BaseURL, "/")
	if base == "" {
		base = DecisionBaseURL
	}
	cl := c.HTTP
	if cl == nil {
		cl = &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	return &decision{base: base, key: c.APIKey, cl: cl}
}

func (*decision) Chat(context.Context, ChatOpts) (Result, error) {
	return Result{}, &Error{Kind: KindBadRequest, Detail: "provider decision không sinh chữ"}
}

func (*decision) Stream(context.Context, ChatOpts) (<-chan Delta, error) {
	return nil, &Error{Kind: KindBadRequest, Detail: "provider decision không sinh chữ"}
}

func (*decision) Embed(context.Context, EmbedOpts) ([][]float32, int, error) {
	return nil, 0, &Error{Kind: KindBadRequest, Detail: "provider decision không nhúng"}
}

type wireQuestion struct {
	Type         string `json:"type"`
	Instructions string `json:"instructions"`
	Criteria     any    `json:"criteria,omitempty"`
}

type wireRequest struct {
	Model     string                  `json:"model"`
	Input     string                  `json:"input"`
	Questions map[string]wireQuestion `json:"questions"`
}

type wireAnswer struct {
	Type          string             `json:"type"`
	Noul          *float64           `json:"noul"`
	Choice        string             `json:"choice"`
	Probabilities map[string]float64 `json:"probabilities"`
	Confidence    float64            `json:"confidence"`
}

type wireResponse struct {
	Answers map[string]wireAnswer `json:"answers"`
	Usage   struct {
		In int `json:"total_input_tokens"`
	} `json:"usage"`
}

// Decide gọi `POST {base}/decisions`. Đổi mã HTTP sang Kind: 401/403 AUTH · 404 MODEL_NOT_FOUND · 429 RATE_LIMIT (+Retry-After) · 5xx SERVER ·
// 400 lọc nội dung BLOCKED · 400 khác BAD_REQUEST · hết giờ / mạng TIMEOUT / NETWORK · thân sai định dạng, thiếu câu trả lời, sai kiểu BAD_RESPONSE.
func (d *decision) Decide(ctx context.Context, o DecideOpts) (DecideResult, error) {
	qs := make(map[string]wireQuestion, len(o.Questions))
	for id, q := range o.Questions {
		crit, ok := q.Criteria.(map[string]string)
		if id == "" || (q.Type != QuestionNoul && q.Type != QuestionChoice) || (q.Type == QuestionChoice && (!ok || len(crit) < 2)) {
			return DecideResult{}, &Error{Kind: KindBadRequest, Detail: "câu hỏi decision không hợp lệ"}
		}
		wq := wireQuestion{Type: q.Type, Instructions: q.Instructions}
		if q.Type == QuestionChoice {
			wq.Criteria = crit
		}
		qs[id] = wq
	}
	if len(qs) == 0 || strings.TrimSpace(o.Input) == "" {
		return DecideResult{}, &Error{Kind: KindBadRequest, Detail: "thiếu câu hỏi hoặc đầu vào"}
	}
	body, err := json.Marshal(wireRequest{Model: o.Model, Input: o.Input, Questions: qs})
	if err != nil {
		return DecideResult{}, &Error{Kind: KindBadRequest}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.base+"/decisions", bytes.NewReader(body))
	if err != nil {
		return DecideResult{}, &Error{Kind: KindBadRequest}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", d.key)
	resp, err := d.cl.Do(req)
	if err != nil {
		return DecideResult{}, d.netErr(err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return DecideResult{}, d.netErr(err)
	}
	if resp.StatusCode != http.StatusOK {
		return DecideResult{}, d.statusErr(resp, string(raw))
	}
	var w wireResponse
	if err := json.Unmarshal(raw, &w); err != nil {
		return DecideResult{}, &Error{Kind: KindBadResponse, Status: resp.StatusCode}
	}
	out := DecideResult{Answers: make(map[string]DecideAnswer, len(qs)), TokensIn: w.Usage.In}
	for id, q := range o.Questions {
		a, ok := w.Answers[id]
		bad := func(why string) error { return &Error{Kind: KindBadResponse, Status: resp.StatusCode, Detail: why} }
		if !ok || (a.Type != "" && a.Type != q.Type) {
			return DecideResult{}, bad("thiếu câu trả lời hoặc sai kiểu: " + id)
		}
		switch q.Type {
		case QuestionNoul:
			if a.Noul == nil || *a.Noul < 0 || *a.Noul > 1 {
				return DecideResult{}, bad("noul ngoài [0,1]")
			}
			out.Answers[id] = DecideAnswer{Type: QuestionNoul, Noul: *a.Noul, Confidence: max(*a.Noul, 1-*a.Noul)}
		default:
			if _, known := q.Criteria.(map[string]string)[a.Choice]; !known || a.Confidence < 0 || a.Confidence > 1 {
				return DecideResult{}, bad("nhãn lạ hoặc confidence ngoài [0,1]")
			}
			out.Answers[id] = DecideAnswer{Type: QuestionChoice, Choice: a.Choice, Probabilities: a.Probabilities, Confidence: a.Confidence}
		}
	}
	return out, nil
}

func (d *decision) statusErr(resp *http.Response, body string) error {
	e := &Error{Status: resp.StatusCode, RetryAfter: retryAfter(resp.Header), Detail: Redact(body, d.key)}
	switch s := resp.StatusCode; {
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
	case s == http.StatusBadRequest && looksBlocked(body):
		e.Kind = KindBlocked
	case s == http.StatusBadRequest && looksLikeAuth(body):
		e.Kind = KindAuth // Google trả 400 khi khoá sai
	default:
		e.Kind = KindBadRequest
	}
	return e
}

func (d *decision) netErr(err error) error {
	var ne net.Error
	var ue *url.Error
	switch {
	case errors.Is(err, context.Canceled):
		return &Error{Kind: KindCancelled}
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, os.ErrDeadlineExceeded), errors.As(err, &ne) && ne.Timeout():
		return &Error{Kind: KindTimeout}
	case errors.As(err, &ne), errors.As(err, &ue):
		return &Error{Kind: KindNetwork, Detail: Redact(err.Error(), d.key)}
	}
	return &Error{Kind: KindNetwork}
}

// blockedCodes là `error.code` của lọc nội dung (decisions-api, mục lỗi).
var blockedCodes = []string{"safety", "prohibited_content", "blocklist", "content_blocked"} //nolint:gochecknoglobals // bảng hằng

// looksBlocked nhận 400 do lọc nội dung: `error.code` thuộc blockedCodes (đọc JSON; thân không phải JSON thì tìm chuỗi).
func looksBlocked(body string) bool {
	var e struct {
		Error struct {
			Code   any    `json:"code"`
			Status string `json:"status"`
		} `json:"error"`
	}
	hay := strings.ToLower(body)
	if json.Unmarshal([]byte(body), &e) == nil {
		hay = strings.ToLower(e.Error.Status + " " + strings.TrimSpace(strings.Trim(strings.ReplaceAll(strings.ToLower(jsonString(e.Error.Code)), "\"", ""), " ")))
	}
	for _, c := range blockedCodes {
		if strings.Contains(hay, c) {
			return true
		}
	}
	return false
}

func jsonString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
