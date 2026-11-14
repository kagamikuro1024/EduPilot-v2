package fake

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/rand/v2"
	"sort"
	"time"

	"github.com/edupilot/backend-go/internal/llm/provider"
)

// DecisionRecord là một dòng bản phát lại: câu `noul` (Noul) hoặc câu `choice` (Choice + Probabilities).
type DecisionRecord struct {
	Noul          *float64           `json:"noul,omitempty"`
	Choice        string             `json:"choice,omitempty"`
	Probabilities map[string]float64 `json:"probabilities,omitempty"`
}

// DecisionKey là khoá bản phát lại: sha256 của đầu vào ĐÃ CHE + ":" + id câu hỏi (SRS 4.6).
func DecisionKey(maskedInput, questionID string) string {
	h := sha256.Sum256([]byte(maskedInput))
	return hex.EncodeToString(h[:]) + ":" + questionID
}

func parseReplay(raw []byte) (map[string]DecisionRecord, error) {
	m := map[string]DecisionRecord{}
	if len(raw) == 0 {
		return m, nil
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		return map[string]DecisionRecord{}, fmt.Errorf("bản phát lại decision: %w", err)
	}
	return m, nil
}

// Decisions là số lời gọi Decide đã tới provider giả (tách khỏi Calls(): test D47 "đúng một lời gọi sinh chữ" vẫn đếm đúng).
func (c *Controller) Decisions() int64 { return c.decisions.Load() }

// AddDecision thêm một dòng bản phát lại cho đầu vào đã che + id câu hỏi (test / seed dựng ca "decision nói có" mà không gọi Google).
func (c *Controller) AddDecision(maskedInput, questionID string, r DecisionRecord) {
	c.mu.Lock()
	c.replay[DecisionKey(maskedInput, questionID)] = r
	c.mu.Unlock()
}

// logNormal lấy mẫu độ trễ log-chuẩn có trung vị p50 và phân vị 95 là p95 (σ = (ln p95 − ln p50) / 1,645).
func logNormal(p50, p95 time.Duration) time.Duration {
	if p50 <= 0 {
		return 0
	}
	if p95 < p50 {
		p95 = p50
	}
	sigma := (math.Log(float64(p95)) - math.Log(float64(p50))) / 1.645
	return time.Duration(float64(p50) * math.Exp(sigma*rand.NormFloat64())) //nolint:gosec // độ trễ giả, không cần CSPRNG
}

// UnknownNoul là P(có) của đầu vào không có trong bản phát lại: 0,25 (P-105-1 Q3) — trung tính rõ ràng, dưới ngưỡng "có" 0,5 và trên ε của mọi cổng
// (khoảng ε 0,01–0,20), nên cascade luôn đi fallback; ở `choice` các nhãn chia đều (confidence = 1/n, dưới mọi τ).
const UnknownNoul = 0.25

// Decide trả xác suất theo bản phát lại; thiếu khoá → noul UnknownNoul / các nhãn đều nhau (ép đi fallback). Không mạng, không khoá.
func (p *Provider) Decide(ctx context.Context, o provider.DecideOpts) (provider.DecideResult, error) {
	p.c.decisions.Add(1)
	p.c.record(Payload{Task: "DECIDE", Inputs: []string{o.Input}})
	s := p.c.Get()
	if err := sleep(ctx, logNormal(s.DecisionP50, s.DecisionP95)); err != nil {
		return provider.DecideResult{}, ctxKind(err)
	}
	switch s.DecisionMode {
	case "error":
		return provider.DecideResult{}, &provider.Error{Kind: provider.KindServer, Status: 503}
	case "timeout":
		<-ctx.Done()
		return provider.DecideResult{}, ctxKind(ctx.Err())
	}
	if s.DecisionErrorRate > 0 && rand.Float64() < s.DecisionErrorRate { //nolint:gosec // tỉ lệ lỗi giả
		return provider.DecideResult{}, &provider.Error{Kind: provider.KindServer, Status: 503}
	}
	p.c.mu.RLock()
	defer p.c.mu.RUnlock()
	if p.c.replayErr != nil {
		return provider.DecideResult{}, &provider.Error{Kind: provider.KindBadResponse, Detail: "bản phát lại decision không đọc được"}
	}
	out := provider.DecideResult{Answers: make(map[string]provider.DecideAnswer, len(o.Questions)), TokensIn: tokens(o.Input) + 64*len(o.Questions)}
	for id, q := range o.Questions {
		rec, ok := p.c.replay[DecisionKey(o.Input, id)]
		switch q.Type {
		case provider.QuestionNoul:
			v := UnknownNoul
			switch {
			case s.DecisionMode == "always_yes":
				v = 0.99
			case s.DecisionMode == "always_no":
				v = 0.01
			case ok && rec.Noul != nil:
				v = min(max(*rec.Noul, 0.01), 0.99)
			case s.DecisionProb != nil: // chỉ cho câu KHÔNG có trong bản phát lại (SRS 4.6)
				v = min(max(*s.DecisionProb, 0.01), 0.99)
			}
			out.Answers[id] = provider.DecideAnswer{Type: provider.QuestionNoul, Noul: v, Confidence: max(v, 1-v)}
		case provider.QuestionChoice:
			crit, _ := q.Criteria.(map[string]string)
			keys := make([]string, 0, len(crit))
			for k := range crit {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			if len(keys) == 0 {
				return provider.DecideResult{}, &provider.Error{Kind: provider.KindBadRequest}
			}
			ans := provider.DecideAnswer{Type: provider.QuestionChoice, Choice: keys[0], Confidence: 1 / float64(len(keys)), Probabilities: make(map[string]float64, len(keys))}
			for _, k := range keys {
				ans.Probabilities[k] = 1 / float64(len(keys))
			}
			if ok && rec.Choice != "" {
				if _, known := crit[rec.Choice]; known {
					ans.Choice, ans.Probabilities = rec.Choice, rec.Probabilities
					ans.Confidence = rec.Probabilities[rec.Choice]
				}
			}
			if s.DecisionProb != nil && len(keys) > 1 && !(ok && rec.Choice != "") {
				top := min(max(*s.DecisionProb, 0), 1)
				ans.Confidence = top
				for _, k := range keys {
					ans.Probabilities[k] = (1 - top) / float64(len(keys)-1)
				}
				ans.Choice = keys[0]
				ans.Probabilities[ans.Choice] = top
			}
			out.Answers[id] = ans
		default:
			return provider.DecideResult{}, &provider.Error{Kind: provider.KindBadRequest}
		}
	}
	return out, nil
}

var _ provider.Decider = (*Provider)(nil)
