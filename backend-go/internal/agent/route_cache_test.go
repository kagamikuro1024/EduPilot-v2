package agent

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/llm"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/testutil"
)

// signedEmbedder là llm.Client chỉ có Embed + chữ ký tuyến; sig đổi = admin đổi tuyến EMBEDDING.
type signedEmbedder struct {
	llm.Client
	sig   atomic.Value
	calls atomic.Int32
}

func (s *signedEmbedder) RouteSignature(...llm.Task) string { return s.sig.Load().(string) }
func (s *signedEmbedder) Embed(context.Context, llm.EmbedRequest) ([][]float32, error) {
	s.calls.Add(1)
	return [][]float32{make([]float32, llm.EmbedDims)}, nil
}

// TestEmbedCacheFollowsRoute — luật 15: đổi tuyến EMBEDDING thì vectơ câu hỏi cache bằng mô hình cũ không được dùng lại (khác không gian vectơ).
func TestEmbedCacheFollowsRoute(t *testing.T) {
	t.Parallel()
	rdb, err := appredis.New(t.Context(), testutil.RedisURL(t))
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })
	gw := &signedEmbedder{}
	gw.sig.Store("fake/embed-a")
	embed := NewEmbedder(gw, rdb)
	q := "câu hỏi " + uuid.NewString()
	_, err = embed(t.Context(), q)
	require.NoError(t, err)
	_, err = embed(t.Context(), q)
	require.NoError(t, err)
	require.EqualValues(t, 1, gw.calls.Load(), "cùng tuyến: trúng cache")
	gw.sig.Store("opencode/qwen3")
	_, err = embed(t.Context(), q)
	require.NoError(t, err)
	require.EqualValues(t, 2, gw.calls.Load(), "đổi tuyến: nhúng lại")
}

// TestAnswerCacheInvalidatedOnRouteChange — luật 15: đổi tuyến CHAT / EMBEDDING thì câu trả lời cache bằng tuyến cũ không trúng.
func TestAnswerCacheInvalidatedOnRouteChange(t *testing.T) {
	t.Parallel()
	r, _ := cacheRig(t)
	var sig atomic.Value
	sig.Store("fake")
	r.a.Cache.Route = func() string { return sig.Load().(string) }
	const q = "Quy chế cảnh báo học vụ quy định ra sao"
	first := r.respond(t, q)
	putIfKeyed(t, r, first)
	require.True(t, r.respond(t, q).Cached)
	sig.Store("opencode")
	after := r.respond(t, q)
	require.False(t, after.Cached)
	require.NotEqual(t, first.CacheKey, after.CacheKey)
}
