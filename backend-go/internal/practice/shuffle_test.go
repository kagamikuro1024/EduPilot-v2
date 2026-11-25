package practice_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/exam"
	"github.com/edupilot/backend-go/internal/practice"
)

func opts4(correct ...int) ([]exam.OptionIn, []int) {
	return []exam.OptionIn{{Body: "Đúng"}, {Body: "Sai một"}, {Body: "Sai hai"}, {Body: "Sai ba"}}, correct
}

// E5: 72/100 đáp án đúng nằm ở A vì LLM luôn đặt đáp án đúng đầu tiên. Xáo tất định theo câu: không vị trí nào chiếm quá 40 % trên 100 câu, đáp án vẫn đúng.
func TestShuffleOptionsDistribution(t *testing.T) {
	t.Parallel()
	var pos [4]int
	for i := range 100 {
		o, c := opts4(0)
		got, gc := practice.ShuffleOptions(fmt.Sprintf("câu hỏi số %d về mạng máy tính?", i), o, c)
		require.Len(t, gc, 1)
		require.Equal(t, "Đúng", got[gc[0]].Body, "chỉ số đáp án phải theo phương án đúng sau xáo")
		pos[gc[0]]++
	}
	for p, n := range pos {
		require.LessOrEqual(t, n, 40, "vị trí %d chiếm %d/100 (phân bố %v)", p, n, pos)
	}
}

func TestShuffleOptionsKeepsKey(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		correct []int
		want    []string
	}{
		"một đáp án":   {[]int{0}, []string{"Đúng"}},
		"nhiều đáp án": {[]int{0, 2}, []string{"Đúng", "Sai hai"}},
	} {
		for i := range 30 {
			o, _ := opts4()
			got, gc := practice.ShuffleOptions(fmt.Sprintf("%s %d", name, i), o, tc.correct)
			var bodies []string
			for _, k := range gc {
				bodies = append(bodies, got[k].Body)
			}
			slices.Sort(bodies)
			want := slices.Clone(tc.want)
			slices.Sort(want)
			require.Equal(t, want, bodies, name)
			require.True(t, slices.IsSorted(gc), "chỉ số đáp án tăng dần")
			all := []string{}
			for _, x := range got {
				all = append(all, x.Body)
			}
			slices.Sort(all)
			require.Equal(t, []string{"Sai ba", "Sai hai", "Sai một", "Đúng"}, all, "mọi phương án còn đủ, không trùng")
		}
	}
}

func TestShuffleOptionsDeterministicAndSkips(t *testing.T) {
	t.Parallel()
	o1, c := opts4(0)
	a, ac := practice.ShuffleOptions("cùng một câu", o1, c)
	o2, _ := opts4()
	b, bc := practice.ShuffleOptions("cùng một câu", o2, c)
	require.Equal(t, a, b, "cùng câu → cùng thứ tự")
	require.Equal(t, ac, bc)
	// phương án ghim cuối (nếu có) ⇒ không xáo; một phương án ⇒ không xáo
	pinned := []exam.OptionIn{{Body: "A"}, {Body: "B"}, {Body: "Tất cả", PinnedLast: true}}
	got, gc := practice.ShuffleOptions("x", pinned, []int{0})
	require.Equal(t, pinned, got)
	require.Equal(t, []int{0}, gc)
	one := []exam.OptionIn{{Body: "A"}}
	got, gc = practice.ShuffleOptions("x", one, []int{0})
	require.Equal(t, one, got)
	require.Equal(t, []int{0}, gc)
}
