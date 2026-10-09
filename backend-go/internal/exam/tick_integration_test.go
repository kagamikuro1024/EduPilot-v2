//go:build integration

package exam_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/exam"
	"github.com/edupilot/backend-go/internal/judge"
	"github.com/edupilot/backend-go/internal/platform/clock"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/testutil"
)

// TestTickLeaderLock — AC6: khoá `ep:exam:tick:leader` (SET NX PX, gia hạn bởi chính chủ) để MỘT bộ lập lịch chạy; hai Ticker thật trên Redis thật chỉ chuyển một lần.
func TestTickLeaderLock(t *testing.T) {
	rdb, err := appredis.New(t.Context(), testutil.RedisURL(t))
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })
	key := judge.LeaderKey() + ":" + t.Name() // tách khỏi các test khác dùng chung Redis
	ttl := 300 * time.Millisecond
	require.True(t, rdb.Leader(t.Context(), key, "a", ttl))
	require.False(t, rdb.Leader(t.Context(), key, "b", ttl), "b không giành được khi a còn giữ")
	require.True(t, rdb.Leader(t.Context(), key, "a", ttl), "a gia hạn")
	time.Sleep(ttl + 100*time.Millisecond)
	require.True(t, rdb.Leader(t.Context(), key, "b", ttl), "hết hạn thì b giành được")
	require.False(t, rdb.Leader(t.Context(), key, "a", ttl))
	require.Equal(t, "ep:exam:tick:leader", judge.LeaderKey())

	// hai Ticker chạy thật: bài đến giờ chỉ chuyển một lần, một sự kiện
	r := newRig(t)
	d := r.scheduled("hai bộ lập lịch")
	clk := clock.NewFake(time.Now().UTC().Add(2*time.Hour + time.Minute))
	ctx := t.Context()
	for _, name := range []string{"w1", "w2"} {
		tk := &exam.Ticker{Svc: r.at(clk), Redis: rdb, Name: name, Every: 20 * time.Millisecond}
		go func() { _ = tk.Run(ctx) }()
	}
	require.Eventually(t, func() bool { return r.status(d.ID) == "OPEN" }, 5*time.Second, 20*time.Millisecond)
	time.Sleep(200 * time.Millisecond)
	require.Equal(t, 1, r.count(`select count(*) from outbox where topic='exam.opened' and payload->>'exam_id'=$1`, d.ID.String()))
}
