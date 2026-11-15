package insight_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/insight"
)

// SRS 4.1: trọng số gốc 0,30/0,20/0,20/0,15/0,15; nguồn thiếu → trọng số chia đều cho tín hiệu còn lại; tổng luôn 1.
func TestWeights(t *testing.T) {
	w := insight.Weights(insight.Present{Low: true, NotHelpful: true, Practice: true})
	require.Equal(t, [5]float64{0.30, 0.20, 0.20, 0.15, 0.15}, w)
	for _, p := range []insight.Present{{}, {Low: true}, {NotHelpful: true, Practice: true}} {
		sum := 0.0
		for _, x := range insight.Weights(p) {
			sum += x
		}
		require.InDelta(t, 1, sum, 1e-9)
	}
	// thiếu luyện đề: 0,15 chia đều cho 4 tín hiệu còn lại
	w = insight.Weights(insight.Present{Low: true, NotHelpful: true})
	require.InDelta(t, 0.30+0.15/4, w[0], 1e-9)
	require.Zero(t, w[4])
}

func TestScoreAndOrderStable(t *testing.T) {
	sigs := []insight.Signals{
		{Students: 5, Questions: 20, LowRate: 0.2, NotHelpfulRate: 0.1, PracticeWrongRate: 0.5},
		{Students: 9, Questions: 40, LowRate: 0.4, NotHelpfulRate: 0.3, PracticeWrongRate: 0.8}, // trội mọi tín hiệu → hạng 1
		{Students: 5, Questions: 20, LowRate: 0.2, NotHelpfulRate: 0.1, PracticeWrongRate: 0.5}, // hoà với cụm 0 → giữ thứ tự (ổn định)
	}
	p := insight.Present{Low: true, NotHelpful: true, Practice: true}
	sc := insight.Score(sigs, p)
	require.InDelta(t, 1, sc[1], 1e-9)
	require.Equal(t, []int{1, 0, 2}, insight.Order(sigs, sc))
	// hoà điểm → S rồi Q
	tie := []insight.Signals{{Students: 3, Questions: 9}, {Students: 4, Questions: 1}}
	require.Equal(t, []int{1, 0}, insight.Order(tie, []float64{0.5, 0.5}))
	// toàn 0: không chia cho 0
	require.NotPanics(t, func() { insight.Score([]insight.Signals{{}}, p) })
}

func vec(x, y float32) []float32 { return []float32{x, y} }

func TestClusterSeparatesTopics(t *testing.T) {
	vs := [][]float32{vec(1, 0), vec(0.99, 0.1), vec(0, 1), vec(0.05, 0.99), vec(1, 0.02)}
	cl := insight.Cluster(vs, 0.9)
	require.Equal(t, [][]int{{0, 1, 4}, {2, 3}}, cl)
	// tau cao hơn mọi cặp → mỗi câu một cụm; vectơ 0 không gây NaN
	require.Len(t, insight.Cluster([][]float32{vec(1, 0), vec(0, 0), vec(0, 1)}, 0.99), 3)
	c := insight.Centroid(vs, []int{2, 3})
	require.False(t, math.IsNaN(float64(c[0])))
}

func TestKeywords(t *testing.T) {
	got := insight.Keywords([]string{"TCP và UDP khác nhau thế nào?", "Phân biệt TCP với UDP", "UDP dùng khi nào"}, 2)
	require.Equal(t, []string{"udp", "tcp"}, got)
}
