//go:build testroutes

package testroutes

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
	"github.com/edupilot/backend-go/internal/jobs"
)

// Hai kind job thử; chỉ có trong bản dựng `-tags testroutes` của worker (SRS 6.3).
const (
	KindProgress = "test.progress"
	KindFail     = "test.fail"
)

// defaultSteps là số bước mặc định khi thân không có `steps` (#Q-QC-03-4).
const defaultSteps = 4

// stepDelay là thời gian mỗi bước của `test.progress` — đủ để client thấy tiến độ trung gian.
const stepDelay = 150 * time.Millisecond

// createJob: `POST /_test/jobs` → 202 `{job_id}` + `Location: /api/v1/jobs/<id>`;
// dòng `jobs` và dòng `outbox` ghi cùng một transaction (03-AC16).
func createJob(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		owner, e := ownerID(r)
		if e != nil {
			apierr.Write(w, r, e)
			return
		}
		var in struct {
			Steps *int   `json:"steps" validate:"omitnil,min=1,max=100"`
			Kind  string `json:"kind" validate:"omitempty,oneof=test.progress test.fail"`
		}
		if !httpx.DecodeJSON(w, r, &in) {
			return
		}
		steps := defaultSteps
		if in.Steps != nil {
			steps = *in.Steps
		}
		kind := in.Kind
		if kind == "" {
			kind = KindProgress
		}

		j, err := d.Jobs.Enqueue(r.Context(), owner, kind, map[string]int{"steps": steps})
		if err != nil {
			writeDBError(w, r, err)
			return
		}
		w.Header().Set("Location", "/api/v1/jobs/"+j.ID.String())
		httpx.WriteJSON(w, http.StatusAccepted, map[string]string{"job_id": j.ID.String()})
	}
}

// RegisterJobKinds cắm hai kind thử vào runner của worker (chỉ bản dựng có build tag gọi được).
func RegisterJobKinds(r *jobs.Runner) {
	r.Register(KindProgress, runProgress)
	r.Register(KindFail, runFail)
}

// runProgress chạy n bước, báo tiến độ tăng dần tới 100 (03-AC16).
func runProgress(ctx context.Context, j jobs.JobCtx) (any, error) {
	var p struct {
		Steps int `json:"steps"`
	}
	if err := json.Unmarshal(j.Payload, &p); err != nil || p.Steps < 1 || p.Steps > 100 {
		p.Steps = defaultSteps
	}
	for i := 1; i <= p.Steps; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(stepDelay):
		}
		j.Progress(i * 100 / p.Steps)
	}
	return map[string]int{"steps": p.Steps}, nil
}

// runFail luôn lỗi: job phải thành FAILED với `error.code`, `progress` giữ nguyên (03-AC17).
func runFail(context.Context, jobs.JobCtx) (any, error) {
	return nil, errors.New("job thử test.fail luôn thất bại")
}
