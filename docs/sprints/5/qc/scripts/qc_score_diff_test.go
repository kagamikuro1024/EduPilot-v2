//go:build ignore

// QC TC-PE01-24/25: kiểm vi sai exam.Score với số hữu tỉ chính xác. Chạy: chép tệp này vào backend-go/internal/exam/zz_qc_score_test.go (bỏ 2 dòng đầu `//go:build ignore`), `go test ./internal/exam -run TestQCScoreDiff -v`, rồi xoá tệp tạm.
package exam_test

import (
	"fmt"
	"math/big"
	"math/rand"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/exam"
)

func qcExact(raw, total, max, step *big.Rat) *big.Rat {
	v := new(big.Rat).Mul(new(big.Rat).Quo(raw, total), max)
	q := new(big.Rat).Quo(v, step)
	q.Add(q, big.NewRat(1, 2))
	fl := new(big.Int).Quo(q.Num(), q.Denom())
	r := new(big.Rat).Mul(new(big.Rat).SetInt(fl), step)
	if r.Cmp(max) > 0 {
		return max
	}
	return r
}

func TestQCScoreDiff(t *testing.T) {
	rng := rand.New(rand.NewSource(20261009))
	maxes := []string{"10", "100", "20", "4.5", "7.5", "1.5", "99.99"}
	steps := []string{"0.01", "0.1", "0.25", "0.5", "1"}
	bad, total := 0, 0
	byMax := map[string]int{}
	for n := 0; n < 20000; n++ {
		k := 1 + rng.Intn(8)
		var items []exam.ScoreItem
		raw, tot := new(big.Rat), new(big.Rat)
		for i := 0; i < k; i++ {
			p := int64(1 + rng.Intn(5))
			e := rng.Int63n(p*4 + 1)
			items = append(items, exam.ScoreItem{Points: decimal.NewFromInt(p), Earned: decimal.NewFromInt(e).Div(decimal.NewFromInt(4))})
			tot.Add(tot, new(big.Rat).SetInt64(p))
			raw.Add(raw, big.NewRat(e, 4))
		}
		mx, st := maxes[rng.Intn(len(maxes))], steps[rng.Intn(len(steps))]
		m, _ := new(big.Rat).SetString(mx)
		s, _ := new(big.Rat).SetString(st)
		got, err := exam.Score(items, decimal.RequireFromString(mx), decimal.RequireFromString(st))
		if err != nil {
			t.Fatal(err)
		}
		want := qcExact(raw, tot, m, s)
		gr, _ := new(big.Rat).SetString(got.String())
		total++
		if gr.Cmp(want) != 0 {
			bad++
			byMax[mx]++
			if bad <= 5 {
				t.Logf("LỆCH max=%s step=%s raw=%s total=%s got=%s want=%s", mx, st, raw.RatString(), tot.RatString(), got, want.FloatString(4))
			}
		}
	}
	fmt.Printf("QC: %d ca, lệch %d, theo max %v\n", total, bad, byMax)
}
