//go:build testroutes

package testroutes

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
	"github.com/edupilot/backend-go/internal/httpapi/llmhttp"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/llm/fake"
	"github.com/edupilot/backend-go/internal/llm/provider"
)

// registerLLM đăng ký 3 route thử của cổng LLM (SRS FEAT-llm-gateway 6.4). Chỉ ADMIN; chỉ số đếm, không nội dung prompt.
func registerLLM(r chi.Router, d Deps) {
	r.Group(func(r chi.Router) {
		r.Use(auth.RequireRole(auth.RoleAdmin))
		r.Post("/llm/chat", llmChat(d))
		r.Get("/llm/stats", llmStats(d))
		r.Post("/llm/fake", llmFake(d))
	})
}

func writeLLMErr(w http.ResponseWriter, r *http.Request, err error) {
	if e, ok := llmhttp.FromLLM(err); ok {
		apierr.Write(w, r, e)
		return
	}
	apierr.Write(w, r, apierr.New(http.StatusInternalServerError, apierr.Internal))
}

func parseLane(s string) (*llm.Lane, bool) {
	var l llm.Lane
	switch s {
	case "":
		return nil, true
	case "INTERACTIVE":
		l = llm.LaneInteractive
	case "NEAR_REALTIME":
		l = llm.LaneNearRealtime
	case "BATCH":
		l = llm.LaneBatch
	default:
		return nil, false
	}
	return &l, true
}

func llmChat(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Task     string `json:"task" validate:"required,oneof=CHAT CLASSIFY UTILITY GRADING QUESTION_GEN INSIGHT"`
			Prompt   string `json:"prompt" validate:"required"`
			Lane     string `json:"lane"`
			Stream   bool   `json:"stream"`
			Passages []struct {
				Text   string  `json:"text"`
				Source string  `json:"source"`
				Page   int     `json:"page"`
				Score  float64 `json:"score"`
			} `json:"passages"`
		}
		if !httpx.DecodeJSON(w, r, &in) {
			return
		}
		lane, ok := parseLane(in.Lane)
		if !ok {
			apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "lane", Code: "invalid", Message: "lane không hợp lệ."}))
			return
		}
		req := llm.Request{Task: llm.Task(in.Task), Lane: lane, Messages: []llm.Message{{Role: "user", Content: in.Prompt}}}
		for _, p := range in.Passages {
			req.Passages = append(req.Passages, llm.Passage{Text: p.Text, Source: p.Source, Page: p.Page, Score: p.Score})
		}
		if in.Stream {
			streamChat(w, r, d, req)
			return
		}
		resp, err := d.LLM.Gateway.Chat(r.Context(), req)
		if err != nil {
			writeLLMErr(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"text": resp.Text, "degraded": resp.Degraded, "provider": resp.Provider, "model": resp.Model,
			"fallback_index": resp.FallbackIndex, "queue_wait_ms": resp.QueueWait.Milliseconds(),
		})
	}
}

// streamChat phát SSE: `event: token`, rồi `event: done` hoặc `event: error {code}` (SRS FEAT-llm-gateway 3.4, US-P1-03 AC12).
func streamChat(w http.ResponseWriter, r *http.Request, d Deps, req llm.Request) {
	ch, err := d.LLM.Gateway.Stream(r.Context(), req)
	if err != nil {
		writeLLMErr(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	rc := http.NewResponseController(w)
	send := func(event string, v any) {
		b, _ := json.Marshal(v)
		_, _ = w.Write([]byte("event: " + event + "\ndata: " + string(b) + "\n\n"))
		_ = rc.Flush()
	}
	for c := range ch {
		switch {
		case c.Err != nil:
			code := "LLM_UNAVAILABLE"
			if e, ok := llmhttp.FromLLM(c.Err); ok {
				code = e.Code
			}
			send("error", map[string]string{"code": code})
		case c.Done:
			send("done", map[string]any{"degraded": c.Response.Degraded, "provider": c.Response.Provider, "model": c.Response.Model,
				"fallback_index": c.Response.FallbackIndex, "queue_wait_ms": c.Response.QueueWait.Milliseconds()})
		default:
			send("token", map[string]string{"text": c.Text})
		}
	}
}

// llmStats chỉ trả số đếm.
func llmStats(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, d.LLM.Stats(r.Context(), r.URL.Query().Get("lanes") == "1"))
	}
}

type fakeIn struct {
	LatencyMinMS *int     `json:"latency_min_ms"`
	LatencyMaxMS *int     `json:"latency_max_ms"`
	ErrorRate    *float64 `json:"error_rate"`
	ErrorKind    *string  `json:"error_kind"`
	ValidKey     *string  `json:"valid_key"`
	StreamDelay  *int     `json:"stream_delay_ms"`
	RetryAfterMS *int     `json:"retry_after_ms"`
}

func llmFake(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in fakeIn
		if !httpx.DecodeJSON(w, r, &in) {
			return
		}
		s := d.LLM.Registry.Fake().Get()
		ms := func(v *int, dst *time.Duration) {
			if v != nil {
				*dst = time.Duration(*v) * time.Millisecond
			}
		}
		ms(in.LatencyMinMS, &s.LatencyMin)
		ms(in.LatencyMaxMS, &s.LatencyMax)
		ms(in.StreamDelay, &s.StreamDelay)
		var retryAfter time.Duration
		ms(in.RetryAfterMS, &retryAfter)
		if in.RetryAfterMS != nil {
			s.RetryAfter = retryAfter
		}
		if in.ErrorRate != nil {
			if *in.ErrorRate < 0 || *in.ErrorRate > 1 {
				apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "error_rate", Code: "range", Message: "error_rate phải trong [0, 1]."}))
				return
			}
			s.ErrorRate = *in.ErrorRate
		}
		if in.ErrorKind != nil {
			s.ErrorKind = provider.Kind(*in.ErrorKind)
		}
		if in.ValidKey != nil {
			s.ValidKey = *in.ValidKey
		}
		if s.LatencyMax < s.LatencyMin {
			s.LatencyMax = s.LatencyMin
		}
		d.LLM.Registry.Fake().Set(s)
		httpx.WriteJSON(w, http.StatusOK, fakeOut(d.LLM.Registry.Fake().Get()))
	}
}

func fakeOut(s fake.Settings) map[string]any {
	return map[string]any{
		"latency_min_ms": s.LatencyMin.Milliseconds(), "latency_max_ms": s.LatencyMax.Milliseconds(), "error_rate": s.ErrorRate,
		"error_kind": string(s.ErrorKind), "valid_key_set": s.ValidKey != "", "stream_delay_ms": s.StreamDelay.Milliseconds(),
	}
}
