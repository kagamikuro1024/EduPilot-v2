package exam_test

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/exam"
)

func d(s string) decimal.Decimal { return decimal.RequireFromString(s) }

// oracle: điểm chuẩn tính bằng số hữu tỉ chính xác (big.Rat) rồi làm tròn nửa lên theo bước — độc lập với DivRound / Round của decimal.
func oracle(raw, total, max, step *big.Rat) *big.Rat {
	v := new(big.Rat).Mul(new(big.Rat).Quo(raw, total), max)
	q := new(big.Rat).Quo(v, step)
	q.Add(q, big.NewRat(1, 2))
	fl := new(big.Int).Quo(q.Num(), q.Denom()) // q ≥ 0 nên Quo (cắt về 0) = floor
	r := new(big.Rat).Mul(new(big.Rat).SetInt(fl), step)
	if r.Cmp(max) > 0 {
		return max
	}
	return r
}

func rat(s string) *big.Rat { r, _ := new(big.Rat).SetString(s); return r }

// TestScoringDecimal — US-PE-01 AC7: ví dụ bắt buộc 7.75 + 39 ca so với số hữu tỉ chính xác; kẹp, void, bước làm tròn.
func TestScoringDecimal(t *testing.T) {
	t.Parallel()
	matched, total := 0, 0
	check := func(name string, items []exam.ScoreItem, max, step, want string) {
		total++
		got, err := exam.Score(items, d(max), d(step))
		require.NoError(t, err, name)
		require.Truef(t, got.Equal(d(want)), "%s: got %s want %s", name, got, want)
		matched++
	}

	// Ví dụ bắt buộc: max=10; Q1 MCQ_SINGLE 1 đ đúng; Q2 MCQ_MULTI 2 đ (K=3) chọn 2 đúng, PARTIAL; Q3 code 3 đ, test [1,1,2] đạt 1 và 3.
	q2 := d("2").Mul(d("2")).DivRound(d("3"), 16)
	q3 := exam.CodeEarned(d("3"), 3, 4)
	require.True(t, q3.Equal(d("2.25")))
	check("ví dụ bắt buộc", []exam.ScoreItem{{Points: d("1"), Earned: d("1")}, {Points: d("2"), Earned: q2}, {Points: d("3"), Earned: q3}}, "10", "0.25", "7.75")

	// 38 ca sinh: tổ hợp bước × thang điểm × tỉ lệ đạt, kiểm với oracle hữu tỉ.
	steps := []string{"0.01", "0.1", "0.25", "0.5", "1"}
	maxes := []string{"10", "100", "7.5"}
	earned := [][2]string{{"0", "3"}, {"1", "3"}, {"2", "3"}, {"3", "3"}, {"1", "7"}, {"5", "7"}, {"13", "17"}, {"1", "6"}, {"2.5", "5"}, {"0.125", "1"}, {"0.875", "1"}, {"4.5", "9"}, {"1.5", "4"}}
	n := 0
	for _, e := range earned {
		for si, st := range steps {
			if (n+si)%2 == 1 && n >= 38 { // đủ 38
				continue
			}
			mx := maxes[(n+si)%len(maxes)]
			if n >= 38 {
				break
			}
			raw, tot := rat(e[0]), rat(e[1])
			want := oracle(raw, tot, rat(mx), rat(st))
			items := []exam.ScoreItem{{Points: d(e[1]), Earned: d(e[0])}}
			check(fmt.Sprintf("sinh %d (%s/%s ×%s bước %s)", n, e[0], e[1], mx, st), items, mx, st, want.FloatString(2))
			n++
		}
	}
	require.GreaterOrEqual(t, n, 38, "đủ ca sinh")

	// ca đặc biệt: void, kẹp, bỏ trống.
	check("void tính đủ điểm kể cả bỏ trống", []exam.ScoreItem{{Points: d("1"), Earned: d("0")}, {Points: d("1"), Earned: d("0"), Void: true}}, "10", "0.01", "5")
	check("không câu nào đạt", []exam.ScoreItem{{Points: d("2"), Earned: d("0")}}, "10", "0.25", "0")
	check("kẹp trên max_score", []exam.ScoreItem{{Points: d("1"), Earned: d("1.5")}}, "10", "0.01", "10")
	check("kẹp dưới 0", []exam.ScoreItem{{Points: d("1"), Earned: d("-3")}}, "10", "0.01", "0")
	check("nửa lên theo bước 0.5", []exam.ScoreItem{{Points: d("4"), Earned: d("1")}}, "10", "0.5", "2.5")
	check("nửa lên đúng tại điểm giữa", []exam.ScoreItem{{Points: d("8"), Earned: d("1")}}, "10", "1", "1")
	t.Logf("matched %d/%d", matched, total)
	require.Equal(t, total, matched)
	require.GreaterOrEqual(t, total, 40)
}

func TestScoringErrors(t *testing.T) {
	t.Parallel()
	_, err := exam.Score(nil, d("10"), d("0.01"))
	require.ErrorIs(t, err, exam.ErrNoPoints)
	require.True(t, exam.CodeEarned(d("3"), 0, 4).IsZero())
	require.True(t, exam.CodeEarned(d("3"), 4, 0).IsZero())
	require.True(t, exam.CodeEarned(d("3"), 9, 4).Equal(d("3"))) // đạt không vượt tổng
}
