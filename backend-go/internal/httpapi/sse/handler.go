package sse

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/config"
	"github.com/edupilot/backend-go/internal/platform/redis"
)

// HandlerDeps là phụ thuộc của handler `GET /api/v1/events`. Handler tự xác thực bằng Verifier (route nằm NGOÀI nhóm timeout
// của router vì stream sống lâu hơn REQUEST_TIMEOUT).
type HandlerDeps struct {
	Cfg      config.Config
	Log      *slog.Logger
	Redis    *redis.Client
	Verifier *auth.Verifier
	Clock    clock.Clock
	// Drain đóng lại khi gateway bắt đầu tắt (httpapi.State.DrainC()): handler gửi `event: shutdown` rồi đóng.
	Drain <-chan struct{}
}

// Khung byte của giao thức (SRS 6.8).
const (
	retryPrelude   = "retry: 3000\n\n"
	heartbeatFrame = ": hb\n\n"
)

// Tên sự kiện điều khiển (không mang `id:`) và `reason` của chúng (SRS 6.8).
const (
	evReady     = "ready"
	evResync    = "resync"
	evReconnect = "reconnect"
	evShutdown  = "shutdown"

	reasonMaxDuration  = "max_duration"
	reasonTokenExpired = "token_expired"
	reasonUpstream     = "upstream_unavailable"
	reasonShutdown     = "server_shutdown"
	reasonBufferTooOld = "buffer_exceeded"
	reasonBadLastID    = "invalid_last_event_id"
)

const (
	// limitRetryAfter là `Retry-After` / `retry_after` của 429 SSE_LIMIT_REACHED (SRS 6.1).
	limitRetryAfter = 5
	// writeDeadline là hạn cho MỖI lần ghi; gia hạn trước từng khung vì http.Server.WriteTimeout (30 s) ngắn hơn stream.
	writeDeadline = 10 * time.Second
	// releaseTimeout là hạn trả chỗ kết nối sau khi ctx của request đã bị huỷ.
	releaseTimeout = 2 * time.Second
	// catchupBatch là số sự kiện mỗi XRANGE khi đọc bù.
	catchupBatch = 200
	// defaultHeartbeat chỉ là lưới an toàn khi Config chưa nạp (time.NewTicker panic với 0).
	defaultHeartbeat = 25 * time.Second
)

// eventIDRE là định dạng id Redis Stream; `Last-Event-ID` không khớp → `resync` invalid_last_event_id.
var eventIDRE = regexp.MustCompile(`^\d+-\d+$`)

// acquireLua cấp một chỗ kết nối nguyên tử (SRS 5.6): xoá member hết hạn → đếm → thêm.
// KEYS[1] = ep:sse:conn:{uid}; ARGV = now_ms, max, expire_at_ms, connection_id, ttl_ms.
const acquireLua = `
redis.call('ZREMRANGEBYSCORE', KEYS[1], '-inf', ARGV[1])
if redis.call('ZCARD', KEYS[1]) >= tonumber(ARGV[2]) then return 0 end
redis.call('ZADD', KEYS[1], ARGV[3], ARGV[4])
redis.call('PEXPIRE', KEYS[1], ARGV[5])
return 1
`

type handler struct {
	cfg     config.Config
	log     *slog.Logger
	rdb     *redis.Client
	clk     clock.Clock
	drain   <-chan struct{}
	acquire *goredis.Script
}

// NewHandler dựng handler SSE: xác thực Bearer (chỉ header), cấp chỗ kết nối, rồi stream (SRS 6.8).
func NewHandler(d HandlerDeps) http.Handler {
	h := &handler{cfg: d.Cfg, log: d.Log, rdb: d.Redis, clk: d.Clock, drain: d.Drain, acquire: goredis.NewScript(acquireLua)}
	if h.log == nil {
		h.log = slog.Default()
	}
	if h.clk == nil {
		h.clk = clock.Real{}
	}
	return auth.Middleware(d.Verifier)(http.HandlerFunc(h.stream))
}

func (h *handler) stream(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	p := auth.MustFromContext(ctx)
	connID := uuid.NewString()

	ok, err := h.takeSlot(ctx, p.Sub, connID)
	switch {
	case err != nil:
		h.log.WarnContext(ctx, "sse: không cấp được chỗ kết nối", "error", err)
		apierr.Write(w, r, apierr.New(http.StatusServiceUnavailable, apierr.ServiceUnavailable))
		return
	case !ok:
		apierr.Write(w, r, apierr.New(http.StatusTooManyRequests, apierr.SSELimitReached).WithRetryAfter(limitRetryAfter))
		return
	}
	defer h.freeSlot(ctx, p.Sub, connID)

	// Mốc bắt đầu đọc TRƯỚC khi subscribe, subscribe TRƯỚC khi gửi `ready` và đọc bù: sự kiện phát xen giữa
	// nằm trong khoảng XRANGE của bước đọc bù nên không bao giờ sót (SRS 6.8, 05-AC7, 05-AC8).
	last, resync, err := h.startFrom(ctx, p.Sub, r.Header.Get("Last-Event-ID"))
	if err != nil {
		h.log.WarnContext(ctx, "sse: không đọc được bộ đệm", "error", err)
		apierr.Write(w, r, apierr.New(http.StatusServiceUnavailable, apierr.ServiceUnavailable))
		return
	}

	ps := h.rdb.Subscribe(ctx, ChanKey(p.Sub))
	defer func() { _ = ps.Close() }()
	if _, err := ps.Receive(ctx); err != nil {
		h.log.WarnContext(ctx, "sse: không đăng ký được Pub/Sub", "error", err)
		apierr.Write(w, r, apierr.New(http.StatusServiceUnavailable, apierr.ServiceUnavailable))
		return
	}

	setStreamHeaders(w, r)
	w.WriteHeader(http.StatusOK)
	out := &writer{w: w, rc: http.NewResponseController(w)}
	if out.write(retryPrelude) != nil || out.write(h.readyFrame(connID)) != nil {
		return
	}
	if resync != "" && out.write(controlFrame(evResync, resync)) != nil {
		return
	}
	if err := h.deliver(ctx, out, p.Sub, &last); err != nil {
		h.afterDeliverErr(ctx, out, err)
		return
	}

	h.pump(ctx, out, ps, p, connID, &last)
}

// pump chạy vòng sống của stream: Pub/Sub → đọc bù, heartbeat + làm mới chỗ, hạn stream, tắt gateway.
func (h *handler) pump(ctx context.Context, out *writer, ps *goredis.PubSub, p auth.Principal, connID string, last *string) {
	every := h.cfg.SSEHeartbeat
	if every <= 0 {
		every = defaultHeartbeat
	}
	hb := time.NewTicker(every)
	defer hb.Stop()

	life, lifeReason := h.lifetime(p)
	limit := time.NewTimer(life)
	defer limit.Stop()

	wake := make(chan struct{}, 1)
	fail := make(chan error, 1)
	go receive(ctx, ps, wake, fail)

	for {
		select {
		case <-ctx.Done(): // client ngắt: chỉ dọn (defer), không log lỗi (05-AC4)
			return
		case <-h.drain:
			_ = out.write(controlFrame(evShutdown, reasonShutdown))
			return
		case <-limit.C:
			_ = out.write(controlFrame(evReconnect, lifeReason))
			return
		case err := <-fail:
			h.upstreamGone(ctx, out, err)
			return
		case <-hb.C:
			if out.write(heartbeatFrame) != nil {
				return
			}
			if err := h.touchSlot(ctx, p.Sub, connID); err != nil {
				h.upstreamGone(ctx, out, err)
				return
			}
		case <-wake:
			if err := h.deliver(ctx, out, p.Sub, last); err != nil {
				h.afterDeliverErr(ctx, out, err)
				return
			}
		}
	}
}

// afterDeliverErr phân biệt lỗi ghi (client đã đi — im lặng) với lỗi Redis (báo reconnect).
func (h *handler) afterDeliverErr(ctx context.Context, out *writer, err error) {
	if out.broken {
		return
	}
	h.upstreamGone(ctx, out, err)
}

// upstreamGone báo `reconnect` upstream_unavailable rồi đóng (05-AC15).
func (h *handler) upstreamGone(ctx context.Context, out *writer, err error) {
	if ctx.Err() == nil {
		h.log.WarnContext(ctx, "sse: nguồn sự kiện không dùng được", "error", err)
	}
	_ = out.write(controlFrame(evReconnect, reasonUpstream))
}

// lifetime là min(SSE_MAX_DURATION, thời gian còn lại của token) kèm lý do tương ứng (05-AC6).
func (h *handler) lifetime(p auth.Principal) (time.Duration, string) {
	d, reason := h.cfg.SSEMaxDuration, reasonMaxDuration
	if !p.ExpiresAt.IsZero() {
		if rest := p.ExpiresAt.Sub(h.clk.Now()); rest < d {
			d, reason = rest, reasonTokenExpired
		}
	}
	return max(d, 0), reason
}

func (h *handler) readyFrame(connID string) string {
	data, err := json.Marshal(struct {
		ConnectionID string `json:"connection_id"`
		ServerTime   string `json:"server_time"`
	}{connID, h.clk.Now().UTC().Format(time.RFC3339Nano)})
	if err != nil { // không thể xảy ra với struct trên
		return "event: " + evReady + "\ndata: {}\n\n"
	}
	return "event: " + evReady + "\ndata: " + string(data) + "\n\n"
}

// startFrom quyết định điểm bắt đầu từ `Last-Event-ID` (SRS 6.8): rỗng/khoảng trắng = không có (chỉ sự kiện mới),
// sai định dạng → resync invalid_last_event_id, cũ hơn đầu bộ đệm → resync buffer_exceeded.
func (h *handler) startFrom(ctx context.Context, uid, raw string) (last, resync string, err error) {
	raw = strings.TrimSpace(raw)
	switch {
	case raw == "":
	case !eventIDRE.MatchString(raw):
		resync = reasonBadLastID
	default:
		first, ferr := h.edgeID(ctx, uid, false)
		if ferr != nil {
			return "", "", ferr
		}
		if first == "" || !olderThan(raw, first) {
			return raw, "", nil
		}
		resync = reasonBufferTooOld
	}
	last, err = h.edgeID(ctx, uid, true)
	return last, resync, err
}

// edgeID trả id đầu (newest=false) hoặc cuối (newest=true) của bộ đệm; "0-0" khi bộ đệm trống hoặc chưa có.
func (h *handler) edgeID(ctx context.Context, uid string, newest bool) (string, error) {
	key := BufKey(uid)
	cmd := h.rdb.XRangeN(ctx, key, "-", "+", 1)
	if newest {
		cmd = h.rdb.XRevRangeN(ctx, key, "+", "-", 1)
	}
	msgs, err := cmd.Result()
	switch {
	case err != nil:
		return "", err
	case len(msgs) == 0 && newest:
		return "0-0", nil
	case len(msgs) == 0:
		return "", nil
	}
	return msgs[0].ID, nil
}

// deliver gửi mọi sự kiện có id > *last rồi cập nhật *last (loại trùng theo id — 05-AC7, 05-AC8).
func (h *handler) deliver(ctx context.Context, out *writer, uid string, last *string) error {
	for {
		msgs, err := h.rdb.XRangeN(ctx, BufKey(uid), "("+*last, "+", catchupBatch).Result()
		if err != nil {
			return err
		}
		for i := range msgs {
			id := msgs[i].ID
			if !olderThan(*last, id) {
				continue
			}
			typ, _ := msgs[i].Values["type"].(string)
			data, _ := msgs[i].Values["data"].(string)
			if typ != "" {
				if err := out.write(eventFrame(id, typ, data)); err != nil {
					return err
				}
			}
			*last = id
		}
		if len(msgs) < catchupBatch {
			return nil
		}
	}
}

func (h *handler) takeSlot(ctx context.Context, uid, connID string) (bool, error) {
	now := h.clk.Now()
	ttl := h.cfg.SSEConnTTL
	n, err := h.acquire.Run(ctx, h.rdb, []string{ConnKey(uid)},
		now.UnixMilli(), h.cfg.SSEMaxPerUser, now.Add(ttl).UnixMilli(), connID, ttl.Milliseconds()).Int64()
	if err != nil {
		return false, err
	}
	return n == 1, nil
}

// touchSlot gia hạn chỗ mỗi heartbeat; chỗ của gateway chết tự hết sau SSE_CONN_TTL (05-AC5).
func (h *handler) touchSlot(ctx context.Context, uid, connID string) error {
	key := ConnKey(uid)
	exp := h.clk.Now().Add(h.cfg.SSEConnTTL)
	if err := h.rdb.ZAdd(ctx, key, goredis.Z{Score: float64(exp.UnixMilli()), Member: connID}).Err(); err != nil {
		return err
	}
	return h.rdb.PExpire(ctx, key, h.cfg.SSEConnTTL).Err()
}

// freeSlot trả chỗ ngay khi kết nối đóng; ctx của request đã huỷ nên dùng bản không huỷ có hạn ngắn.
func (h *handler) freeSlot(ctx context.Context, uid, connID string) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), releaseTimeout)
	defer cancel()
	if err := h.rdb.ZRem(ctx, ConnKey(uid), connID).Err(); err != nil {
		h.log.DebugContext(ctx, "sse: không trả được chỗ kết nối", "error", err)
	}
}

// receive biến mỗi tin Pub/Sub thành một tín hiệu đánh thức (gộp lô được); lỗi = Redis chết.
func receive(ctx context.Context, ps *goredis.PubSub, wake chan<- struct{}, fail chan<- error) {
	for {
		if _, err := ps.Receive(ctx); err != nil {
			select {
			case fail <- err:
			default:
			}
			return
		}
		select {
		case wake <- struct{}{}:
		default:
		}
	}
}

func setStreamHeaders(w http.ResponseWriter, r *http.Request) {
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache, no-transform")
	h.Set("X-Accel-Buffering", "no")
	if r.ProtoMajor == 1 && r.ProtoMinor == 1 {
		h.Set("Connection", "keep-alive")
	}
}

func eventFrame(id, typ, data string) string {
	var b strings.Builder
	b.Grow(len(id) + len(typ) + len(data) + 20)
	b.WriteString("id: ")
	b.WriteString(id)
	b.WriteString("\nevent: ")
	b.WriteString(typ)
	b.WriteString("\ndata: ")
	b.WriteString(data)
	b.WriteString("\n\n")
	return b.String()
}

func controlFrame(event, reason string) string {
	return "event: " + event + `
data: {"reason":"` + reason + `"}` + "\n\n"
}

// olderThan so sánh hai id Redis Stream theo (ms, seq).
func olderThan(a, b string) bool {
	am, as := idParts(a)
	bm, bs := idParts(b)
	return am < bm || (am == bm && as < bs)
}

func idParts(id string) (uint64, uint64) {
	ms, seq, _ := strings.Cut(id, "-")
	return parseUint(ms), parseUint(seq)
}

func parseUint(s string) uint64 {
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil { // id dài hơn uint64: coi như lớn nhất (không bao giờ "cũ hơn")
		return math.MaxUint64
	}
	return v
}

// writer ghi từng khung SSE: gia hạn write deadline rồi flush ngay (không đệm — 05-AC14).
type writer struct {
	w      http.ResponseWriter
	rc     *http.ResponseController
	broken bool // client đã đóng: thôi ghi, thôi log
}

func (x *writer) write(s string) error {
	if x.broken {
		return io.ErrClosedPipe
	}
	if err := x.rc.SetWriteDeadline(time.Now().Add(writeDeadline)); err != nil && !errors.Is(err, http.ErrNotSupported) {
		x.broken = true
		return err
	}
	if _, err := io.WriteString(x.w, s); err != nil {
		x.broken = true
		return err
	}
	if err := x.rc.Flush(); err != nil && !errors.Is(err, http.ErrNotSupported) {
		x.broken = true
		return err
	}
	return nil
}
