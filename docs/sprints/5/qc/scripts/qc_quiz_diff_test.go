//go:build ignore

// QC TC-PE01-21: kiểm vi sai quiz.Grade (MCQ_MULTI PARTIAL / ALL_OR_NOTHING, MCQ_SINGLE, TRUE_FALSE) với số hữu tỉ. Chép vào backend-go/internal/quiz/zz_qc_quiz_test.go (bỏ 2 dòng đầu), `go test ./internal/quiz -run TestQCQuizDiff -v`, xoá tệp tạm.
package quiz_test

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/quiz"
)

func TestQCQuizDiff(t *testing.T) {
	bad, n := 0, 0
	pts := decimal.RequireFromString("3")
	for K := 1; K <= 5; K++ {
		ids := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
		key := quiz.Key{OptionIDs: ids[:K]}
		for TP := 0; TP <= K; TP++ {
			for FP := 0; FP <= 8-K; FP++ {
				sel := append(append([]string{}, ids[:TP]...), ids[K:K+FP]...)
				a := &quiz.Answer{OptionIDs: sel}
				for _, mode := range []quiz.Mode{quiz.Partial, quiz.AllOrNothing} {
					got, err := quiz.Grade(quiz.MCQMulti, pts, key, a, mode, ids...)
					if err != nil {
						t.Fatalf("err %v K=%d TP=%d FP=%d", err, K, TP, FP)
					}
					want := new(big.Rat)
					if mode == quiz.Partial {
						d := TP - FP
						if d < 0 {
							d = 0
						}
						want = big.NewRat(int64(3*d), int64(K))
					} else if TP == K && FP == 0 {
						want = big.NewRat(3, 1)
					}
					gr, _ := new(big.Rat).SetString(got.String())
					if got.String() != "0" && got.Exponent() < -15 { // làm tròn 16 chữ số
						gr = new(big.Rat).SetFrac(got.Mul(decimal.New(1, 16)).BigInt(), new(big.Int).Exp(big.NewInt(10), big.NewInt(16), nil))
					}
					diff := new(big.Rat).Sub(gr, want)
					lim := big.NewRat(1, 10000000000000000)
					n++
					if new(big.Rat).Abs(diff).Cmp(lim) > 0 || got.Sign() < 0 {
						bad++
						if bad < 6 {
							t.Logf("LỆCH %v K=%d TP=%d FP=%d got=%s want=%s", mode, K, TP, FP, got, want.FloatString(10))
						}
					}
				}
			}
		}
	}
	// trống / id lạ / trùng id / SINGLE / TRUE_FALSE
	z, e := quiz.Grade(quiz.MCQMulti, pts, quiz.Key{OptionIDs: []string{"a", "b"}}, nil, quiz.Partial)
	fmt.Println("trống:", z, e)
	_, e = quiz.Grade(quiz.MCQMulti, pts, quiz.Key{OptionIDs: []string{"a", "b"}}, &quiz.Answer{OptionIDs: []string{"a", "zzz"}}, quiz.Partial, "a", "b", "c")
	fmt.Println("id lạ:", e)
	d1, _ := quiz.Grade(quiz.MCQMulti, pts, quiz.Key{OptionIDs: []string{"a", "b"}}, &quiz.Answer{OptionIDs: []string{"a", "a", "b"}}, quiz.Partial, "a", "b", "c")
	fmt.Println("trùng id (a,a,b) PARTIAL:", d1)
	s1, _ := quiz.Grade(quiz.MCQSingle, pts, quiz.Key{OptionIDs: []string{"a"}}, &quiz.Answer{OptionIDs: []string{"a"}}, quiz.AllOrNothing, "a", "b")
	s2, _ := quiz.Grade(quiz.MCQSingle, pts, quiz.Key{OptionIDs: []string{"a"}}, &quiz.Answer{OptionIDs: []string{"b"}}, quiz.AllOrNothing, "a", "b")
	tr, fa := true, false
	f1, _ := quiz.Grade(quiz.TrueFalse, pts, quiz.Key{Value: &tr}, &quiz.Answer{Value: &tr}, quiz.AllOrNothing)
	f2, _ := quiz.Grade(quiz.TrueFalse, pts, quiz.Key{Value: &tr}, &quiz.Answer{Value: &fa}, quiz.AllOrNothing)
	fmt.Println("SINGLE đúng/sai:", s1, s2, " TF đúng/sai:", f1, f2)
	fmt.Printf("QC: %d ca MULTI, lệch %d\n", n, bad)
}
