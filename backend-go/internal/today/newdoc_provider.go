package today

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/edupilot/backend-go/internal/platform/clock"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/store"
)

// NewDocumentWindow: tài liệu vào lớp trong chừng này thì còn là «mới» (US-P2-11 AC15); cũng là TTL của khoá «đã mở».
const NewDocumentWindow = 72 * time.Hour

// seenTimeout: hạn đọc / ghi khoá «đã mở»; Redis chậm hoặc chết thì coi như chưa mở (mục hiện, không lỗi).
const seenTimeout = 100 * time.Millisecond

// SeenKey là khoá Redis `ep:today:docseen:{user_id}:{course_id}` = lần cuối sinh viên mở thư viện lớp (RFC 3339).
func SeenKey(userID, courseID uuid.UUID) string {
	return appredis.Key("today", "docseen", userID.String(), courseID.String())
}

// MarkLibrarySeen ghi «đã mở thư viện lớp» (GET /courses/{id}/library thành công, 200 hoặc 304) rồi xoá cache `today` của người đó. Lỗi Redis bỏ qua (khoá mất ⇒ mục hiện lại).
func MarkLibrarySeen(ctx context.Context, rdb *appredis.Client, clk clock.Clock, userID, courseID uuid.UUID) {
	if rdb == nil {
		return
	}
	now := time.Now().UTC()
	if clk != nil {
		now = clk.Now().UTC()
	}
	cctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), seenTimeout)
	defer cancel()
	if rdb.Set(cctx, SeenKey(userID, courseID), now.Format(time.RFC3339Nano), NewDocumentWindow).Err() != nil {
		return
	}
	_ = rdb.Del(cctx, CacheKey(userID, "all"), CacheKey(userID, courseID.String())).Err()
}

// NewDocumentProvider: NEW_DOCUMENT (bậc 90, chỉ STUDENT) — mỗi lớp ACTIVE có tài liệu mới sau lần mở thư viện gần nhất tối đa một mục `{N} tài liệu mới · lớp {mã}`.
// MỘT truy vấn gộp mọi lớp (không đổi theo số lớp); «đã mở» đọc bằng một lệnh MGET, không tính vào ngân sách SQL. Chỉ lớp ACTIVE của chính người xem.
type NewDocumentProvider struct {
	Pool  *pgxpool.Pool
	Redis *appredis.Client // nil = không lọc «đã mở»
}

// Name implements Provider.
func (NewDocumentProvider) Name() string { return "new_document" }

// Items implements Provider.
func (p NewDocumentProvider) Items(ctx context.Context, v Viewer, s Scope) ([]Item, error) {
	courses := v.Scoped(s, "STUDENT")
	if v.Role != RoleStudent || len(courses) == 0 {
		return nil, nil
	}
	rows, err := store.New(p.Pool).TodayNewDocuments(ctx, store.TodayNewDocumentsParams{CourseIds: ids(courses), Since: v.Now.Add(-NewDocumentWindow)})
	if err != nil {
		return nil, fmt.Errorf("today: tài liệu mới: %w", err)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	seen := p.seen(ctx, v.UserID, rows)
	byID := map[uuid.UUID]CourseRef{}
	for _, c := range courses {
		byID[c.ID] = c
	}
	var out []Item
	for _, r := range rows {
		c, ok := byID[r.CourseID]
		if !ok {
			continue
		}
		n := len(r.Ats)
		if at, ok := seen[r.CourseID]; ok {
			n = 0
			for _, t := range r.Ats {
				if t.After(at) {
					n++
				}
			}
		}
		if n == 0 {
			continue // r.Ats mới nhất trước: n = số tài liệu sau lần mở, mới nhất ở đầu
		}
		reason := r.Titles[0]
		if n > 1 {
			reason = fmt.Sprintf("%s và %d tài liệu khác", r.Titles[0], n-1)
		}
		out = append(out, mk(Item{
			ID: "NEW_DOCUMENT:" + c.ID.String(), Kind: KindNewDocument, Tier: TierNewDocument, Course: courseRef(c), Href: "/library?course=" + c.ID.String(),
			Title: fmt.Sprintf("%d tài liệu mới · lớp %s", n, c.ClassCode), Reason: reason, AgeMinutes: int(v.Now.Sub(r.Ats[0]) / time.Minute),
		}))
	}
	return out, nil
}

// seen đọc «lần mở gần nhất» của các lớp bằng MỘT MGET. Lỗi / khoá mất / giá trị hỏng ⇒ lớp đó coi như chưa mở.
func (p NewDocumentProvider) seen(ctx context.Context, user uuid.UUID, rows []store.TodayNewDocumentsRow) map[uuid.UUID]time.Time {
	out := map[uuid.UUID]time.Time{}
	if p.Redis == nil {
		return out
	}
	keys := make([]string, len(rows))
	for i, r := range rows {
		keys[i] = SeenKey(user, r.CourseID)
	}
	cctx, cancel := context.WithTimeout(ctx, seenTimeout)
	defer cancel()
	vals, err := p.Redis.MGet(cctx, keys...).Result()
	if err != nil {
		return out
	}
	for i, v := range vals {
		str, ok := v.(string)
		if !ok {
			continue
		}
		if t, err := time.Parse(time.RFC3339Nano, str); err == nil {
			out[rows[i].CourseID] = t
		}
	}
	return out
}
