package privacy

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func axisVec(i int) []float32 { v := make([]float32, 4); v[i] = 1; return v }

func TestExemplarsAtLeast40(t *testing.T) {
	t.Parallel()
	ex := Exemplars()
	require.GreaterOrEqual(t, len(ex), 40)
	for _, s := range ex {
		require.NotEmpty(t, s)
	}
}

// TestClassify — SRS 4.4: luật quyết rõ thì không nhúng; chưa quyết + đủ dài thì nhúng một lần; ≥ ngưỡng cao → cá nhân; giữa / thấp → công khai; nhúng lỗi → chỉ luật.
func TestClassify(t *testing.T) {
	t.Parallel()
	d, _ := newDetector(nil)
	c := &Classifier{Detector: d, Protos: NewPrototypesFromVectors([][]float32{axisVec(0)})}
	calls := 0
	embedAxis := func(i int, mix float32) func(context.Context) ([]float32, error) {
		return func(context.Context) ([]float32, error) {
			calls++
			v := axisVec(i)
			v[1] += mix
			return v, nil
		}
	}
	ctx := context.Background()

	res, vec, err := c.Classify(ctx, uuid.Nil, "Điểm giữa kỳ của em là bao nhiêu", embedAxis(0, 0))
	require.NoError(t, err)
	require.True(t, res.Personal)
	require.Equal(t, ChannelPrivate, res.Channel)
	require.Contains(t, res.Reasons, ReasonPattern)
	require.Nil(t, vec)
	require.Zero(t, calls, "luật quyết rõ → không nhúng")

	res, _, _ = c.Classify(ctx, uuid.Nil, "mail a@b.vn hỏi về giao thức TCP", embedAxis(0, 0))
	require.Contains(t, res.Reasons, ReasonPIIEmail)
	require.Zero(t, calls)

	res, vec, _ = c.Classify(ctx, uuid.Nil, "Giải thích giúp mình thuật toán Dijkstra nhé", embedAxis(2, 0))
	require.False(t, res.Personal)
	require.Equal(t, ChannelPublic, res.Channel)
	require.True(t, res.UsedEmbedding)
	require.NotNil(t, vec, "vectơ trả ra cho rag dùng lại")
	require.Equal(t, 1, calls)

	res, _, _ = c.Classify(ctx, uuid.Nil, "Cho mình biết tình hình học tập của bản thân nhé", embedAxis(0, 0))
	require.True(t, res.Personal)
	require.Contains(t, res.Reasons, ReasonSimilarty)
	require.Equal(t, 2, calls)

	res, _, _ = c.Classify(ctx, uuid.Nil, "Cho mình biết tình hình học tập của bản thân nhé", embedAxis(0, 1.0)) // cos = 1/√2 ≈ 0,707: vùng giữa → không cá nhân
	require.False(t, res.Personal)

	_, _, _ = c.Classify(ctx, uuid.Nil, "ngắn thôi", embedAxis(0, 0))
	require.Equal(t, 3, calls, "văn bản < 20 ký tự không nhúng")

	res, vec, err = c.Classify(ctx, uuid.Nil, "Giải thích giúp mình thuật toán Dijkstra nhé", func(context.Context) ([]float32, error) { return nil, errors.New("hết hạn mức") })
	require.NoError(t, err)
	require.False(t, res.UsedEmbedding, "nhúng lỗi → chỉ luật")
	require.Nil(t, vec)
}
