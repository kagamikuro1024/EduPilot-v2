package grading

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	"github.com/edupilot/backend-go/internal/jobs"
)

// Hàng đợi việc rút lưu ý (SRS FEAT-grading-guidance §5): handler outbox chỉ XADD rồi ErrDeferred (không gọi LLM trong consumer outbox tuần tự);
// consumer riêng đọc nhóm `reflect`, thử tối đa ReflectAttempts lần, hết lượt thì job FAILED + dead-letter.
const (
	StreamReflect     = "ep:grading.reflect"
	StreamReflectDead = "ep:grading.reflect.dead"
	GroupReflect      = "reflect"
	// ReflectAttempts là số lần thử tối đa của một việc (SRS §5).
	ReflectAttempts = 3
	reflectReclaim  = 5 * time.Minute
)

// RegisterReflect đăng ký `grading.reflect` vào Runner.
func RegisterReflect(r *jobs.Runner, rdb *goredis.Client) {
	r.Register(KindReflect, func(ctx context.Context, j jobs.JobCtx) (any, error) {
		var p ReflectPayload
		if err := json.Unmarshal(j.Payload, &p); err != nil || p.AssignmentID == uuid.Nil {
			return nil, fmt.Errorf("grading: payload %s hỏng", KindReflect)
		}
		err := rdb.XAdd(ctx, &goredis.XAddArgs{Stream: StreamReflect, MaxLen: 10000, Approx: true,
			Values: map[string]any{"job_id": j.ID.String(), "owner": j.OwnerID.String(), "payload": string(j.Payload)}}).Err()
		if err != nil {
			return nil, fmt.Errorf("grading: XADD %s: %w", StreamReflect, err)
		}
		return nil, jobs.ErrDeferred
	})
}

// ReflectQueue là consumer của ep:grading.reflect.
type ReflectQueue struct {
	R        *Reflector
	Runner   *jobs.Runner
	Redis    *goredis.Client
	Consumer string
	Log      *slog.Logger
	// Group: rỗng = GroupReflect (test dùng nhóm riêng).
	Group string
	// Retry: chờ giữa hai lần thử (mặc định 2 s).
	Retry time.Duration
}

func (q *ReflectQueue) group() string {
	if q.Group != "" {
		return q.Group
	}
	return GroupReflect
}

// Run chạy tới khi ctx huỷ.
func (q *ReflectQueue) Run(ctx context.Context) error {
	if err := q.Redis.XGroupCreateMkStream(ctx, StreamReflect, q.group(), "0").Err(); err != nil && !strings.HasPrefix(err.Error(), "BUSYGROUP") {
		return fmt.Errorf("grading: tạo nhóm %s: %w", StreamReflect, err)
	}
	for ctx.Err() == nil {
		m, ok := q.next(ctx)
		if !ok {
			continue
		}
		q.handle(context.WithoutCancel(ctx), m)
	}
	return ctx.Err()
}

func (q *ReflectQueue) next(ctx context.Context) (goredis.XMessage, bool) {
	msgs, _, err := q.Redis.XAutoClaim(ctx, &goredis.XAutoClaimArgs{Stream: StreamReflect, Group: q.group(), Consumer: q.Consumer, MinIdle: reflectReclaim, Start: "0-0", Count: 1}).Result()
	if err == nil && len(msgs) > 0 {
		return msgs[0], true
	}
	res, err := q.Redis.XReadGroup(ctx, &goredis.XReadGroupArgs{Group: q.group(), Consumer: q.Consumer, Streams: []string{StreamReflect, ">"}, Count: 1, Block: 2 * time.Second}).Result()
	if err != nil || len(res) == 0 || len(res[0].Messages) == 0 {
		if err != nil && !errors.Is(err, goredis.Nil) && ctx.Err() == nil {
			q.Log.WarnContext(ctx, "grading.reflect: không đọc được hàng", "error", err.Error())
			select {
			case <-ctx.Done():
			case <-time.After(time.Second):
			}
		}
		return goredis.XMessage{}, false
	}
	return res[0].Messages[0], true
}

func (q *ReflectQueue) ack(ctx context.Context, id string) {
	if err := q.Redis.XAck(ctx, StreamReflect, q.group(), id).Err(); err != nil {
		q.Log.WarnContext(ctx, "grading.reflect: không ACK được", "msg", id, "error", err.Error())
	}
}

func (q *ReflectQueue) handle(ctx context.Context, m goredis.XMessage) {
	field := func(k string) string { s, _ := m.Values[k].(string); return s }
	jobID, err1 := uuid.Parse(field("job_id"))
	owner, _ := uuid.Parse(field("owner"))
	var p ReflectPayload
	if err2 := json.Unmarshal([]byte(field("payload")), &p); err1 != nil || err2 != nil {
		q.Log.WarnContext(ctx, "grading.reflect: tin hỏng, bỏ", "msg", m.ID)
		q.ack(ctx, m.ID)
		return
	}
	wait := q.Retry
	if wait <= 0 {
		wait = 2 * time.Second
	}
	var last error
	for attempt := 1; attempt <= ReflectAttempts; attempt++ {
		res, err := q.R.Run(ctx, jobID, owner, p)
		var ab *ErrReflectAbort
		switch {
		case err == nil:
			if cerr := q.Runner.Complete(ctx, jobID, res); cerr != nil {
				q.Log.ErrorContext(ctx, "grading.reflect: không ghi kết quả việc, giao lại", "job_id", jobID.String(), "error", cerr.Error())
				return // không ACK: giao lại; Run idempotent ở mức việc (nháp trùng bị bước chống trùng loại)
			}
			q.ack(ctx, m.ID)
			return
		case errors.As(err, &ab):
			_ = q.Runner.Abort(ctx, jobID, ab.Code, ab.Code)
			q.ack(ctx, m.ID)
			return
		}
		last = err
		q.Log.WarnContext(ctx, "grading.reflect lỗi", "job_id", jobID.String(), "attempt", attempt, "error", err.Error())
		if attempt < ReflectAttempts {
			time.Sleep(wait)
		}
	}
	q.Redis.XAdd(ctx, &goredis.XAddArgs{Stream: StreamReflectDead, MaxLen: 1000, Approx: true, Values: map[string]any{"job_id": jobID.String(), "assignment_id": p.AssignmentID.String(), "error": errClass(last)}})
	_ = q.Runner.Abort(ctx, jobID, "REFLECT_FAILED", "Chưa rút được lưu ý. Thử lại.")
	q.ack(ctx, m.ID)
}

// errClass: chỉ loại lỗi, không chữ có thể chứa dữ liệu.
func errClass(err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf("%T", errors.Unwrap(err))
}
