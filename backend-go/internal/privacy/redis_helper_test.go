package privacy

import (
	"testing"

	appredis "github.com/edupilot/backend-go/internal/platform/redis"
)

// deadRedis: client trỏ tới cổng không ai nghe — mô phỏng Redis hỏng.
func deadRedis(t testing.TB) *appredis.Client {
	t.Helper()
	c, err := appredis.New(t.Context(), "redis://127.0.0.1:1/0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}
