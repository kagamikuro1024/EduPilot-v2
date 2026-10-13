//go:build integration

package llmconfig_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/llm/llmrt"
)

// US-P1-04 AC9: sửa ở tiến trình A (API thật) → tiến trình B (Registry + Watch riêng, cùng Postgres + Redis) dùng cấu hình mới ≤ 1 s.
// "Hai tiến trình" ở đây là hai llmrt.Runtime độc lập trong một binary test; bản hai tiến trình OS thật là TestReloadTwoProcesses (internal/llm).
func TestReloadAcrossProcesses(t *testing.T) {
	a := getAPI(t)
	b, err := llmrt.New(t.Context(), a.cfg, a.pool, a.rdb, slog.New(slog.NewTextHandler(io.Discard, nil)))
	require.NoError(t, err)
	defer b.Close(t.Context())
	waitSubscribed(t, a.rdb, llm.ReloadChannel) // chờ B đăng ký pub/sub (không đoán bằng thời gian)

	adm := a.token(t, auth.RoleAdmin)
	m1 := a.mk(t, adm, "A1", map[string]any{"model": "fake-chat", "kind": "chat", "price_in": "0", "price_out": "0"})
	m2 := a.mk(t, adm, "A2", map[string]any{"model": "fake-chat-2", "kind": "chat", "price_in": "0", "price_out": "0"})
	id := func(p map[string]any) string { return p["models"].([]any)[0].(map[string]any)["id"].(string) }
	cur := func(r *llmrt.Runtime) string {
		rt, err := r.Registry.Route(llm.TaskChat)
		require.NoError(t, err)
		return rt.Targets[0].Model
	}

	require.Equal(t, 200, a.do(t, adm, "PUT", "/admin/llm/routes", map[string]any{"task": "CHAT", "chain": []string{id(m1)}, "version": 0}).status)
	require.Eventually(t, func() bool { return cur(b) == "fake-chat" }, 30*time.Second, 10*time.Millisecond) // lần đồng bộ đầu: không phải mốc ≤ 1 s của AC9

	start := time.Now()
	require.Equal(t, 200, a.do(t, adm, "PUT", "/admin/llm/routes", map[string]any{"task": "CHAT", "chain": []string{id(m2)}, "version": 1}).status)
	require.Eventually(t, func() bool { return cur(b) == "fake-chat-2" }, time.Second, 5*time.Millisecond)
	require.Less(t, time.Since(start), time.Second, "B phải dùng cấu hình mới ≤ 1 s sau khi A sửa")
	require.Equal(t, "fake-chat-2", cur(a.rt))
}

// waitSubscribed chờ tới khi kênh pub/sub có người nghe (thay cho Sleep đoán thời gian đăng ký).
func waitSubscribed(t testing.TB, rdb *goredis.Client, channel string) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if m, err := rdb.PubSubNumSub(context.Background(), channel).Result(); err == nil && m[channel] > 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("kênh %s chưa có người nghe", channel)
}
