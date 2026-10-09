package exam

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/edupilot/backend-go/internal/platform/clock"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/store"
)

// Lock là một khoá chat đang có hiệu lực. `ExamID` chỉ có khi tra từ DB (khoá Redis chỉ giữ id lượt); `Until` = hạn + grace.
type Lock struct {
	AttemptID uuid.UUID
	ExamID    uuid.UUID
	Until     time.Time
}

// Locker là hợp đồng "chat AI bị khoá trong giờ làm bài" cho P3 (US-PE-07 AC1–AC2, AC11, D56).
// Nguồn sự thật là DB (`exam_attempts`); khoá Redis `ep:exam_lock:<user_id>` = `<attempt_id>` (TTL = hạn − now + grace) chỉ là bộ nhớ đệm cho đường nóng.
type Locker struct {
	Pool  *pgxpool.Pool
	Redis *appredis.Client // nil ⇒ chỉ DB
	Clock clock.Clock
	Grace time.Duration
}

func (l *Locker) now() time.Time {
	if l.Clock == nil {
		return time.Now().UTC()
	}
	return l.Clock.Now()
}

func (l *Locker) grace() time.Duration {
	if l.Grace <= 0 {
		return 10 * time.Second
	}
	return l.Grace
}

// LockKey là khoá Redis của một người dùng.
func LockKey(userID uuid.UUID) string { return appredis.Key("exam_lock", userID.String()) }

func (l *Locker) latest(ctx context.Context, userID uuid.UUID) (Lock, bool, error) {
	row, err := store.New(l.Pool).LockLatestAttempt(ctx, store.LockLatestAttemptParams{StudentID: userID, GraceSeconds: int32(l.grace() / time.Second), Now: l.now()}) //nolint:gosec // ≤ 300
	if errors.Is(err, pgx.ErrNoRows) {
		return Lock{}, false, nil
	}
	if err != nil {
		return Lock{}, false, fmt.Errorf("exam: tra khoá chat: %w", err)
	}
	return Lock{AttemptID: row.ID, ExamID: row.ExamID, Until: row.DeadlineAt.Add(l.grace())}, true, nil
}

// Refresh đồng bộ khoá Redis của người này với DB: còn lượt IN_PROGRESS → `SET` (lượt có hạn muộn nhất, TTL = hạn − now + grace, không bao giờ −1); không còn → `DEL`.
// Gọi sau khi bắt đầu / làm tiếp / nộp / gia hạn (SAU commit, vì trong giao dịch DB còn thấy trạng thái cũ).
func (l *Locker) Refresh(ctx context.Context, userID uuid.UUID) error {
	if l.Redis == nil {
		return nil
	}
	lk, ok, err := l.latest(ctx, userID)
	if err != nil {
		return err
	}
	if !ok {
		return l.Redis.Del(ctx, LockKey(userID)).Err()
	}
	return l.set(ctx, userID, lk)
}

func (l *Locker) set(ctx context.Context, userID uuid.UUID, lk Lock) error {
	ttl := lk.Until.Sub(l.now())
	if ttl < time.Second {
		ttl = time.Second
	}
	return l.Redis.Set(ctx, LockKey(userID), lk.AttemptID.String(), ttl).Err()
}

// IsLocked: trúng Redis → true (không đụng DB; `ExamID` để trống); trượt → tra DB rồi nạp lại Redis; Redis lỗi → vẫn trả đúng nhờ DB;
// DB cũng lỗi → trả lỗi: người gọi phải TỪ CHỐI chat (an toàn khi nghi ngờ).
func (l *Locker) IsLocked(ctx context.Context, userID uuid.UUID) (Lock, bool, error) {
	if l.Redis != nil {
		if v, err := l.Redis.Get(ctx, LockKey(userID)).Result(); err == nil {
			if id, perr := uuid.Parse(v); perr == nil {
				if ttl, err := l.Redis.PTTL(ctx, LockKey(userID)).Result(); err == nil && ttl > 0 {
					return Lock{AttemptID: id, Until: l.now().Add(ttl)}, true, nil
				}
			}
		}
	}
	lk, ok, err := l.latest(ctx, userID)
	if err != nil {
		return Lock{}, false, err
	}
	if ok && l.Redis != nil {
		_ = l.set(ctx, userID, lk) // nạp lại; lỗi Redis không ảnh hưởng kết quả
	}
	return lk, ok, nil
}

// Sweep gỡ khoá của lượt đã kết thúc mà `DEL` lúc nộp không tới được Redis (Redis sập đúng lúc nộp): quét tối đa `limit` khoá, đối chiếu DB. Gọi ở nhịp tick.
func (l *Locker) Sweep(ctx context.Context, limit int) (int, error) {
	if l.Redis == nil {
		return 0, nil
	}
	var keys []string
	var cursor uint64
	for len(keys) < limit {
		ks, next, err := l.Redis.Scan(ctx, cursor, appredis.Key("exam_lock", "*"), 100).Result()
		if err != nil {
			return 0, fmt.Errorf("exam: quét khoá chat: %w", err)
		}
		keys = append(keys, ks...)
		if cursor = next; cursor == 0 {
			break
		}
	}
	if len(keys) == 0 {
		return 0, nil
	}
	ids := make([]string, 0, len(keys))
	vals, err := l.Redis.MGet(ctx, keys...).Result()
	if err != nil {
		return 0, fmt.Errorf("exam: đọc khoá chat: %w", err)
	}
	for _, v := range vals {
		if s, ok := v.(string); ok {
			ids = append(ids, s)
		}
	}
	live, err := store.New(l.Pool).LockStillRunning(ctx, ids)
	if err != nil {
		return 0, fmt.Errorf("exam: đối chiếu khoá chat: %w", err)
	}
	alive := make(map[string]bool, len(live))
	for _, id := range live {
		alive[id.String()] = true
	}
	n := 0
	for i, k := range keys {
		if s, ok := vals[i].(string); ok && !alive[s] {
			if l.Redis.Del(ctx, k).Err() == nil {
				n++
			}
		}
	}
	return n, nil
}

// locker dựng Locker từ cấu hình của Service.
func (s *Service) locker() *Locker {
	return &Locker{Pool: s.Pool, Redis: s.Redis, Clock: s.Clock, Grace: s.Attempt.grace()}
}

// refreshLock: đồng bộ khoá chat sau một thay đổi lượt. Lỗi KHÔNG làm hỏng thao tác của sinh viên: DB là nguồn sự thật, `IsLocked` tự nạp lại, tick quét khoá mồ côi.
func (s *Service) refreshLock(ctx context.Context, userID uuid.UUID) {
	_ = s.locker().Refresh(context.WithoutCancel(ctx), userID)
}
