package llm

import (
	"context"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/edupilot/backend-go/internal/llm/provider"
)

// Giới hạn của một lời gọi Decide (ta tự giới hạn; Google cho nhiều hơn).
const (
	maxQuestions = 10
	maxOptions   = 20
	maxTextRunes = 600 // chỉ thị / mô tả nhãn: chữ tĩnh trong mã
)

var idRe = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)
var optionKeyRe = regexp.MustCompile(`^[A-Za-z0-9_ -]{1,128}$`)

// CapCounter đếm lời gọi decision THẬT theo ngày (khoá `ep:decision:real:<yyyymmdd UTC+7>`), dùng chung giữa các gateway qua Redis.
type CapCounter interface {
	Incr(ctx context.Context, key string, ttl time.Duration) (int64, error)
}

// memCounter là bộ đếm trong tiến trình (test, dev không Redis).
type memCounter struct {
	mu sync.Mutex
	n  map[string]int64
}

func (m *memCounter) Incr(_ context.Context, key string, _ time.Duration) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.n == nil {
		m.n = map[string]int64{}
	}
	m.n[key]++
	return m.n[key], nil
}

// inputCap là trần ký tự đầu vào theo cổng (SRS 4.3). INJECTION_FLAG giữ 8.000 ký tự đầu + 8.000 ký tự cuối.
func inputCap(g DecideGate) int {
	switch g {
	case GateThreads:
		return 4000
	case GateInjection:
		return 16000
	case GateRouting:
		return 2000
	}
	return 2000
}

// clip cắt s theo trần ký tự (rune); `headTail` giữ nửa đầu + nửa cuối.
func clip(s string, limit int, headTail bool) string {
	if utf8.RuneCountInString(s) <= limit {
		return s
	}
	r := []rune(s)
	if headTail {
		return string(r[:limit/2]) + "\n" + string(r[len(r)-limit/2:])
	}
	return string(r[:limit])
}

func validDecide(r DecideRequest) error {
	if !r.Gate.Valid() || strings.TrimSpace(r.Input) == "" || len(r.Questions) == 0 || len(r.Questions) > maxQuestions {
		return ErrBadRequest
	}
	seen := map[string]bool{}
	for _, q := range r.Questions {
		id := q.QuestionID()
		if !idRe.MatchString(id) || seen[id] {
			return ErrBadRequest
		}
		seen[id] = true
		switch v := q.(type) {
		case NoulQ:
			if strings.TrimSpace(v.Text) == "" || utf8.RuneCountInString(v.Text) > maxTextRunes {
				return ErrBadRequest
			}
		case ChoiceQ:
			if strings.TrimSpace(v.Text) == "" || utf8.RuneCountInString(v.Text) > maxTextRunes || len(v.Options) < 2 || len(v.Options) > maxOptions {
				return ErrBadRequest
			}
			keys := map[string]bool{}
			for _, o := range v.Options {
				if !optionKeyRe.MatchString(o.Key) || keys[o.Key] || utf8.RuneCountInString(o.Text) > maxTextRunes {
					return ErrBadRequest
				}
				keys[o.Key] = true
			}
		default: // ScoreQ: khai báo cho đủ hợp đồng, 10.5 không dùng (D60 (5))
			return ErrBadRequest
		}
	}
	return nil
}

func toOpts(model, input string, qs []Question) provider.DecideOpts {
	m := make(map[string]provider.DecideQuestion, len(qs))
	for _, q := range qs {
		switch v := q.(type) {
		case NoulQ:
			m[v.ID] = provider.DecideQuestion{Type: provider.QuestionNoul, Instructions: v.Text}
		case ChoiceQ:
			crit := make(map[string]string, len(v.Options))
			for _, o := range v.Options {
				crit[o.Key] = o.Text
			}
			m[v.ID] = provider.DecideQuestion{Type: provider.QuestionChoice, Instructions: v.Text, Criteria: crit}
		}
	}
	return provider.DecideOpts{Model: model, Input: input, Questions: m}
}

// fromResult trả câu trả lời THEO THỨ TỰ câu hỏi, khớp bằng id (nhà cung cấp sắp khoá theo bảng chữ cái).
func fromResult(qs []Question, res provider.DecideResult) ([]Answer, error) {
	out := make([]Answer, 0, len(qs))
	for _, q := range qs {
		a, ok := res.Answers[q.QuestionID()]
		if !ok {
			return nil, &provider.Error{Kind: provider.KindBadResponse}
		}
		switch q.(type) {
		case NoulQ:
			out = append(out, NoulA{ID: q.QuestionID(), Noul: a.Noul, Confidence: a.Confidence})
		default:
			out = append(out, ChoiceA{ID: q.QuestionID(), Choice: a.Choice, Probabilities: a.Probabilities, Confidence: a.Confidence})
		}
	}
	return out, nil
}

// vnDay là ngày hiện tại theo UTC+7 (yyyymmdd): khoá của trần lời gọi thật.
func vnDay(now time.Time) string {
	return now.UTC().Add(7 * time.Hour).Format("20060102")
}

// pending là lời gọi decision đã xong nhưng CHƯA ghi llm_audit: Close gắn `cascade_path` rồi ghi MỘT dòng. Trễ của dòng là trễ của lời gọi decision.
type pending struct {
	g *Gateway
	c *call
}

// Close ghi dòng audit với đường đi ("" = không thuộc cascade). Gọi lần hai là no-op.
func (p *pending) Close(path Path) {
	if p == nil || p.c == nil {
		return
	}
	if path != "" && path != PathNotUsed {
		p.c.row.CascadePath = string(path)
	}
	p.g.o.Auditor.Record(p.c.row)
	p.c = nil
}

// Decide hỏi decision model: che PII → giấy phép Scheduler (task DECIDE, bucket riêng provider `decision`) → provider → llm_audit → cầu dao.
// Không thử lại. Lỗi nào cũng trả ngay để cascade chọn fallback. Gọi thẳng (không qua Cascade) thì dòng audit không có `cascade_path`.
func (g *Gateway) Decide(ctx context.Context, r DecideRequest) (DecideResponse, error) {
	resp, p, err := g.decide(ctx, r, 0)
	p.Close("")
	return resp, err
}

// decide là lõi của Decide; trả `pending` (luôn khác nil) để người gọi gắn đường đi. timeout = 0 → thời hạn mặc định (LLM_DECISION_TIMEOUT).
func (g *Gateway) decide(ctx context.Context, r DecideRequest, timeout time.Duration) (DecideResponse, *pending, error) {
	lane := r.Lane
	if lane != LaneBatch {
		lane = LaneInteractive
	}
	if timeout <= 0 {
		timeout = g.o.DecisionTimeout
	}
	now := g.o.Now()
	id := IdentityFrom(ctx)
	tid := TraceID(ctx)
	if tid == "" {
		tid = "0"
	}
	c := &call{g: g, task: TaskDecide, start: now, lane: lane, maxRet: 0, noWait: lane == LaneInteractive}
	c.row = AuditRow{Task: string(TaskDecide), Lane: lane.String(), UserID: id.UserID, CourseID: id.CourseID, TraceID: tid, At: now, CascadeGate: string(r.Gate)}
	p := &pending{g: g, c: c}
	fail := func(err error, kind string) (DecideResponse, *pending, error) {
		c.ctx = ctx
		c.row.ErrorKind = kind
		c.seal(nil, err)
		return DecideResponse{}, p, err
	}
	if err := validDecide(r); err != nil {
		return fail(err, string(provider.KindBadRequest))
	}
	target, ok := g.o.Registry.DecideTarget()
	if !ok {
		return fail(ErrNotConfigured, "")
	}
	c.route = Route{Targets: []Target{target}}
	if g.gate.BudgetState(ctx, id.CourseID) == BudgetExhausted { // hết ngân sách: không đổi sang mô hình chat (Decide không sinh chữ) → fallback
		return fail(&ErrUnavailable{Reason: ReasonBudgetExhausted}, "BUDGET")
	}
	if target.Model == RealDecisionModel { // trần lời gọi thật mỗi ngày UTC+7 (US-DM-01 AC7)
		n, err := g.o.RealCounter.Incr(ctx, "ep:decision:real:"+vnDay(now), 36*time.Hour)
		if err == nil && n > int64(g.o.RealDailyCap) {
			return fail(&ErrUnavailable{Reason: ReasonBudgetExhausted}, "REAL_DAILY_CAP")
		}
	}
	var cancel context.CancelFunc
	c.ctx, cancel = context.WithTimeout(ctx, timeout) // không vượt phần hạn còn lại của request gốc: WithTimeout lấy hạn sớm hơn
	c.cancel = cancel
	in, _, err := g.maskTexts(c, []string{r.Input}) // CHE như Embed; Text của câu hỏi là chữ tĩnh, không che
	if err != nil {
		c.seal(nil, err)
		return DecideResponse{}, p, err
	}
	input := clip(in[0], inputCap(r.Gate), r.Gate == GateInjection)
	type decVal struct {
		res provider.DecideResult
	}
	ch, err := runChain(c,
		func(Target) int { return (len(input)+3)/4 + 64*len(r.Questions) },
		func(ctx context.Context, t Target) (decVal, int, error) {
			d, ok := t.P.(provider.Decider)
			if !ok {
				return decVal{}, 0, &provider.Error{Kind: provider.KindBadRequest}
			}
			res, err := d.Decide(ctx, toOpts(t.Model, input, r.Questions))
			return decVal{res}, res.TokensIn, err
		})
	if err != nil {
		c.seal(nil, err)
		return DecideResponse{}, p, err
	}
	answers, aerr := fromResult(r.Questions, ch.val.res)
	if aerr != nil { // thiếu câu trả lời: coi như lỗi của nhà cung cấp
		c.row.ErrorKind = string(provider.KindBadResponse)
		ch.permit.Done(-1)
		c.seal(nil, aerr)
		return DecideResponse{}, p, aerr
	}
	base := c.response(ch.target, ch.idx, "", ch.val.res.TokensIn, 0, ch.permit) // chi phí chỉ tính token vào; price_out của decision là 0
	c.seal(&base, nil)
	return DecideResponse{Answers: answers, Usage: Usage{TokensIn: ch.val.res.TokensIn}, Provider: base.Provider, Model: base.Model, CostEst: base.CostEst}, p, nil
}
