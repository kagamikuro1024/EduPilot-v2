package grading

import (
	"context"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	"github.com/edupilot/backend-go/internal/store"
)

// Bốn cờ theo lớp của harness học từ lần giảng viên sửa (SRS FEAT-grading-guidance 5.4). Mặc định tắt.
const (
	FlagGradingGuidance   = "GRADING_GUIDANCE"
	FlagRegressionCapture = "REGRESSION_CAPTURE"
	FlagThreadSuggest     = "THREAD_SUGGEST"
	FlagQuestionHints     = "QUESTION_HINTS"
)

// GuidanceFlagList là bốn cờ theo thứ tự hiển thị.
var GuidanceFlagList = []string{FlagGradingGuidance, FlagThreadSuggest, FlagQuestionHints, FlagRegressionCapture} //nolint:gochecknoglobals // bảng bất biến

// FlagsChannel là kênh pub/sub báo cờ đã đổi (mẫu D60 `ep:decision:cfg`).
const FlagsChannel = "ep:guidance:flags"

// FlagsPoll là chu kỳ làm mới dự phòng khi mất thông báo (SRS §8: TTL 60 s).
const FlagsPoll = 60 * time.Second

type flagKey struct {
	course  uuid.UUID
	feature string
}

// GuidanceFlags giữ ảnh chụp «lớp nào bật tính năng nào» trong bộ nhớ: đường nóng (chấm bài, soạn bài, sinh câu hỏi) KHÔNG chạm DB. Chưa nạp được → mọi cờ tắt.
type GuidanceFlags struct {
	// Poll: chu kỳ làm mới dự phòng khi mất tin PUBLISH (luật 15: TTL chỉ là lưới an toàn); 0 = FlagsPoll (60 s). Test đặt nhỏ.
	Poll time.Duration
	pool store.DBTX
	log  *slog.Logger
	cur  atomic.Pointer[map[flagKey]struct{}]
}

// NewGuidanceFlags dựng ảnh chụp rỗng (mọi cờ tắt); gọi Load để nạp.
func NewGuidanceFlags(pool store.DBTX, log *slog.Logger) *GuidanceFlags {
	if log == nil {
		log = slog.Default()
	}
	return &GuidanceFlags{pool: pool, log: log}
}

// Load nạp lại từ DB; lỗi → giữ ảnh cũ.
func (f *GuidanceFlags) Load(ctx context.Context) error {
	rows, err := store.New(f.pool).GuidanceFlagsAll(ctx)
	if err != nil {
		return err
	}
	m := make(map[flagKey]struct{}, len(rows))
	for _, r := range rows {
		m[flagKey{r.CourseID, r.Feature}] = struct{}{}
	}
	f.cur.Store(&m)
	return nil
}

// Set thay ảnh chụp (test).
func (f *GuidanceFlags) Set(on map[uuid.UUID][]string) {
	m := map[flagKey]struct{}{}
	for c, fs := range on {
		for _, ft := range fs {
			m[flagKey{c, ft}] = struct{}{}
		}
	}
	f.cur.Store(&m)
}

// On báo cờ `feature` của lớp có bật không. nil → tắt.
func (f *GuidanceFlags) On(course uuid.UUID, feature string) bool {
	if f == nil {
		return false
	}
	m := f.cur.Load()
	if m == nil {
		return false
	}
	_, ok := (*m)[flagKey{course, feature}]
	return ok
}

// Watch làm mới theo sự kiện Redis và mỗi 60 s; chạy tới khi ctx huỷ. rdb nil → chỉ thăm dò định kỳ.
func (f *GuidanceFlags) Watch(ctx context.Context, rdb *goredis.Client) {
	var msgs <-chan *goredis.Message
	if rdb != nil {
		ps := rdb.Subscribe(ctx, FlagsChannel)
		defer func() { _ = ps.Close() }()
		msgs = ps.Channel()
	}
	poll := f.Poll
	if poll <= 0 {
		poll = FlagsPoll
	}
	t := time.NewTicker(poll)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		case _, ok := <-msgs:
			if !ok {
				msgs = nil
				continue
			}
		}
		if err := f.Load(ctx); err != nil {
			f.log.WarnContext(ctx, "không nạp được cờ guidance, giữ bản cũ", "error", err.Error())
		}
	}
}
