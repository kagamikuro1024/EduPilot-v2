package agent

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/platform/outbox"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/rag"
	"github.com/edupilot/backend-go/internal/testutil"
)

func cacheRig(t *testing.T) (*rig, *appredis.Client) {
	t.Helper()
	testutil.RequireContainers(t)
	rdb, err := appredis.New(t.Context(), testutil.RedisURL(t))
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })
	r := newRig(t, true, 3)
	r.tc.CourseID = uuid.New() // Redis dùng chung giữa các test: mỗi test một lớp
	r.a.Cache = &AnswerCache{Redis: NewRedisKV(rdb), Version: func(ctx context.Context, c uuid.UUID) (int64, error) { return rag.Version(ctx, rdb, c) }}
	return r, rdb
}

func ansKeys(t *testing.T, rdb *appredis.Client, r *rig) int {
	t.Helper()
	keys, err := rdb.Keys(t.Context(), "ep:ans:"+r.tc.CourseID.String()+":*").Result()
	require.NoError(t, err)
	return len(keys)
}

// putIfKeyed mô phỏng chat: ghi cache sau khi sinh xong khi Outcome.CacheKey khác rỗng.
func putIfKeyed(t *testing.T, r *rig, out Outcome) {
	t.Helper()
	if out.CacheKey != "" {
		require.NoError(t, r.a.Cache.Put(t.Context(), out.CacheKey, CachedAnswer{Text: "đã trả lời"}))
	}
}

// TestPersonalNeverCached — AC12: intent cá nhân không đọc và không ghi cache.
func TestPersonalNeverCached(t *testing.T) {
	t.Parallel()
	r, rdb := cacheRig(t)
	for _, s := range []string{"Em đã vắng mấy buổi rồi", "Điểm giữa kỳ của em là bao nhiêu", "Lịch thi khi nào vậy ạ", "Tuần này có gì sắp tới không", "Cho em xem điểm của Lê Thị Bình", "Em không muốn sống nữa", "Cách tính điểm môn này ra sao"} {
		for range 2 {
			out := r.respond(t, s)
			require.Empty(t, out.CacheKey, s)
			require.False(t, out.Cached, s)
			putIfKeyed(t, r, out)
		}
	}
	require.Zero(t, ansKeys(t, rdb, r))
}

// TestPIIMessageNeverCached — AC12: tin nhắn có PII bị che không vào cache dù là COURSE_QA.
func TestPIIMessageNeverCached(t *testing.T) {
	t.Parallel()
	r, rdb := cacheRig(t)
	for _, s := range []string{"Em là Nguyễn Văn An, giải thích giao thức TCP giúp em", "mail của em là an@sv.edu.vn, TCP hoạt động ra sao nhỉ", "Quy chế học vụ 20201234 quy định gì"} {
		out := r.respond(t, s)
		require.Empty(t, out.CacheKey, s)
		putIfKeyed(t, r, out)
	}
	require.Zero(t, ansKeys(t, rdb, r))
}

// TestCacheInvalidatedOnDocumentChange — AC12: COURSE_QA không PII có cache; document.changed tăng phiên bản nên khoá cũ không trúng (không chờ TTL).
func TestCacheInvalidatedOnDocumentChange(t *testing.T) {
	t.Parallel()
	r, rdb := cacheRig(t)
	const q = "Quy chế cảnh báo học vụ quy định ra sao"
	first := r.respond(t, q)
	require.NotEmpty(t, first.CacheKey)
	putIfKeyed(t, r, first)
	require.Equal(t, 1, ansKeys(t, rdb, r))
	hit := r.respond(t, q)
	require.True(t, hit.Cached)
	require.Equal(t, "đã trả lời", hit.Canned)
	require.EqualValues(t, 1, r.g.calls.Load(), "lần hai trúng cache: không sinh lại")
	// hỏi lại khác hoa-thường / dấu / khoảng trắng vẫn trúng
	require.True(t, r.respond(t, "  quy che canh bao hoc vu quy dinh ra sao ").Cached)

	payload, _ := json.Marshal(map[string]string{"course_id": r.tc.CourseID.String()})
	require.NoError(t, rag.BumpVersion(rdb)(t.Context(), outbox.Message{Topic: "document.changed", Payload: payload}))
	after := r.respond(t, q)
	require.False(t, after.Cached, "phiên bản tri thức đã đổi")
	require.NotEmpty(t, after.CacheKey)
	require.NotEqual(t, first.CacheKey, after.CacheKey)
}
