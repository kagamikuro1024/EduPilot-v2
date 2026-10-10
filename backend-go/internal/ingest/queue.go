package ingest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	"github.com/edupilot/backend-go/internal/jobs"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
)

// Ba loại việc của hàng ep:ingest. `document.ingest` và `document.reindex*` đều đi qua jobs.Runner chỉ để XADD (jobs.ErrDeferred).
const (
	KindIngest     = "document.ingest"
	KindReindex    = "document.reindex"
	KindReindexAll = "document.reindex_all"

	// StreamName là hàng việc dài dùng chung (ingest, lập chỉ mục lại, AI trả lời Threads). Tên Stream không mang tiền tố "ep:" theo quy ước của gói redis
	// (SYSTEM_DESIGN §3.3) — hằng này là tên đã chốt trong spec: `ep:ingest`.
	StreamName = "ep:ingest"
	GroupName  = "ingest"
	maxLen     = 10000
)

// KindPayload là payload của việc: tài liệu (ingest / reindex) hoặc lớp (reindex_all).
type KindPayload struct {
	DocumentID uuid.UUID `json:"document_id,omitempty"`
	CourseID   uuid.UUID `json:"course_id,omitempty"`
}

// RegisterKinds đăng ký ba loại việc vào jobs.Runner: handler chỉ XADD rồi trả ErrDeferred — không gọi docling trong consumer outbox
// (consumer đó tuần tự một goroutine; một PDF 143 trang ≈ 100 s sẽ chặn xoá cache roster, cache "Hôm nay", mail).
func RegisterKinds(r *jobs.Runner, rdb *appredis.Client) {
	for _, kind := range []string{KindIngest, KindReindex, KindReindexAll} {
		r.Register(kind, func(ctx context.Context, j jobs.JobCtx) (any, error) {
			var p KindPayload
			if err := json.Unmarshal(j.Payload, &p); err != nil {
				return nil, fmt.Errorf("ingest: payload %s: %w", kind, err)
			}
			id := p.DocumentID
			if kind == KindReindexAll {
				id = p.CourseID
			}
			if id == uuid.Nil {
				return nil, &jobs.UserError{Code: CodeNotReady, Message: message(CodeNotReady, 0)}
			}
			if err := Enqueue(ctx, rdb, j.ID, j.OwnerID, kind, id); err != nil {
				return nil, err
			}
			return nil, jobs.ErrDeferred
		})
	}
}

// Enqueue đẩy việc vào hàng.
func Enqueue(ctx context.Context, rdb *appredis.Client, jobID, owner uuid.UUID, kind string, id uuid.UUID) error {
	err := rdb.XAdd(ctx, &goredis.XAddArgs{Stream: StreamName, MaxLen: maxLen, Approx: true,
		Values: map[string]any{"job_id": jobID.String(), "owner": owner.String(), "kind": kind, "id": id.String()}}).Err()
	if err != nil {
		return fmt.Errorf("ingest: XADD: %w", err)
	}
	return nil
}

// Queue là consumer của hàng ep:ingest: Workers goroutine; tin chưa nhận được tài liệu (lớp đang bận) KHÔNG được ACK và được giao lại sau Reclaim.
type Queue struct {
	P        *Processor
	Redis    *appredis.Client
	Consumer string
	Log      *slog.Logger
	// Extra: các loại việc do gói khác định nghĩa (vd. `thread.answer` — AI trả lời Threads). Trả error = lỗi hạ tầng, giữ tin để giao lại.
	Extra map[string]func(ctx context.Context, jobID, owner, id uuid.UUID) error
	// Group: tên nhóm tiêu thụ; rỗng = GroupName. Test dùng nhóm riêng để không nuốt tin tồn của gói khác trên Redis dùng chung.
	Group string

	mu       sync.Mutex
	inflight map[string]bool
}

func (q *Queue) group() string {
	if q.Group != "" {
		return q.Group
	}
	return GroupName
}

func (q *Queue) tryLock(id string) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.inflight == nil {
		q.inflight = map[string]bool{}
	}
	if q.inflight[id] {
		return false
	}
	q.inflight[id] = true
	return true
}

func (q *Queue) unlock(id string) {
	q.mu.Lock()
	delete(q.inflight, id)
	q.mu.Unlock()
}

// Run chạy tới khi ctx huỷ.
func (q *Queue) Run(ctx context.Context) error {
	if err := q.Redis.XGroupCreateMkStream(ctx, StreamName, q.group(), "0").Err(); err != nil && !strings.HasPrefix(err.Error(), "BUSYGROUP") {
		return fmt.Errorf("ingest: tạo nhóm %s: %w", StreamName, err)
	}
	n := max(1, q.P.Set.Workers)
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			q.loop(ctx, fmt.Sprintf("%s-%d", q.Consumer, i))
		}()
	}
	wg.Wait()
	return ctx.Err()
}

func (q *Queue) loop(ctx context.Context, consumer string) {
	for ctx.Err() == nil {
		msg, ok := q.next(ctx, consumer)
		if !ok {
			continue
		}
		if !q.tryLock(msg.ID) {
			continue
		}
		q.handle(context.WithoutCancel(ctx), msg)
		q.unlock(msg.ID)
	}
}

// next: ưu tiên tin treo quá Reclaim (bận / consumer chết), rồi tin mới (chặn tối đa 2 s).
func (q *Queue) next(ctx context.Context, consumer string) (goredis.XMessage, bool) {
	msgs, _, err := q.Redis.XAutoClaim(ctx, &goredis.XAutoClaimArgs{Stream: StreamName, Group: q.group(), Consumer: consumer, MinIdle: q.P.Set.Reclaim, Start: "0-0", Count: 1}).Result()
	if err == nil && len(msgs) > 0 {
		return msgs[0], true
	}
	res, err := q.Redis.XReadGroup(ctx, &goredis.XReadGroupArgs{Group: q.group(), Consumer: consumer, Streams: []string{StreamName, ">"}, Count: 1, Block: 2 * time.Second}).Result()
	if err != nil || len(res) == 0 || len(res[0].Messages) == 0 {
		if err != nil && !errors.Is(err, goredis.Nil) && ctx.Err() == nil {
			q.Log.WarnContext(ctx, "ingest: không đọc được hàng", "error", err.Error())
			select {
			case <-ctx.Done():
			case <-time.After(time.Second):
			}
		}
		return goredis.XMessage{}, false
	}
	return res[0].Messages[0], true
}

func (q *Queue) handle(ctx context.Context, m goredis.XMessage) {
	field := func(k string) string { s, _ := m.Values[k].(string); return s }
	jobID, err1 := uuid.Parse(field("job_id"))
	id, err2 := uuid.Parse(field("id"))
	owner, _ := uuid.Parse(field("owner"))
	if err1 != nil || err2 != nil {
		q.Log.WarnContext(ctx, "ingest: tin hỏng, bỏ", "msg", m.ID)
		q.ack(ctx, m.ID)
		return
	}
	var perr error
	busy := false
	switch field("kind") {
	case KindIngest:
		var out outcome
		out, perr = q.P.Ingest(ctx, jobID, id)
		busy = out == outBusy && perr == nil
	case KindReindex:
		perr = q.P.Reindex(ctx, jobID, id)
	case KindReindexAll:
		perr = q.P.ReindexAll(ctx, jobID, id, owner)
	default:
		if h := q.Extra[field("kind")]; h != nil {
			perr = h(ctx, jobID, owner, id)
		} else {
			q.Log.WarnContext(ctx, "ingest: loại việc lạ, bỏ", "msg", m.ID, "kind", field("kind"))
		}
	}
	switch {
	case perr != nil:
		// lỗi hạ tầng (DB, Redis): giữ tin để giao lại; việc idempotent
		q.Log.ErrorContext(ctx, "ingest: lỗi hạ tầng, sẽ giao lại", "job_id", jobID.String(), "error", perr.Error())
	case busy:
		// lớp đang có tài liệu khác PROCESSING: không ACK, không đóng job
	default:
		q.ack(ctx, m.ID)
	}
}

func (q *Queue) ack(ctx context.Context, id string) {
	if err := q.Redis.XAck(ctx, StreamName, q.group(), id).Err(); err != nil {
		q.Log.WarnContext(ctx, "ingest: không ACK được", "msg", id, "error", err.Error())
	}
}
