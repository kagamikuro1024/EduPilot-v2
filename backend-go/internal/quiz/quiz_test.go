package quiz_test

import (
	"math/big"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/quiz"
)

func d(s string) decimal.Decimal { return decimal.RequireFromString(s) }
func bp(b bool) *bool            { return &b }

var opts = []string{"a", "b", "c", "d", "e", "f", "g", "h"}

// TestQuizGrade — US-PE-01 AC6: mọi loại × đúng / sai / trống / thừa / thiếu / trùng (≥ 40 ca).
func TestQuizGrade(t *testing.T) {
	t.Parallel()
	pts := d("2.00")
	cases := []struct {
		name string
		typ  quiz.Type
		key  quiz.Key
		ans  *quiz.Answer
		mode quiz.Mode
		want string
	}{
		{"single đúng", quiz.MCQSingle, quiz.Key{OptionIDs: []string{"a"}}, &quiz.Answer{OptionIDs: []string{"a"}}, quiz.Partial, "2"},
		{"single sai", quiz.MCQSingle, quiz.Key{OptionIDs: []string{"a"}}, &quiz.Answer{OptionIDs: []string{"b"}}, quiz.Partial, "0"},
		{"single trống nil", quiz.MCQSingle, quiz.Key{OptionIDs: []string{"a"}}, nil, quiz.Partial, "0"},
		{"single trống rỗng", quiz.MCQSingle, quiz.Key{OptionIDs: []string{"a"}}, &quiz.Answer{}, quiz.Partial, "0"},
		{"single chọn nhiều gồm đúng", quiz.MCQSingle, quiz.Key{OptionIDs: []string{"a"}}, &quiz.Answer{OptionIDs: []string{"a", "b"}}, quiz.Partial, "0"},
		{"single trùng id đúng", quiz.MCQSingle, quiz.Key{OptionIDs: []string{"a"}}, &quiz.Answer{OptionIDs: []string{"a", "a"}}, quiz.Partial, "2"},
		{"single không phụ thuộc chế độ", quiz.MCQSingle, quiz.Key{OptionIDs: []string{"c"}}, &quiz.Answer{OptionIDs: []string{"c"}}, quiz.AllOrNothing, "2"},
		{"tf đúng true", quiz.TrueFalse, quiz.Key{Value: bp(true)}, &quiz.Answer{Value: bp(true)}, quiz.Partial, "2"},
		{"tf đúng false", quiz.TrueFalse, quiz.Key{Value: bp(false)}, &quiz.Answer{Value: bp(false)}, quiz.Partial, "2"},
		{"tf sai", quiz.TrueFalse, quiz.Key{Value: bp(true)}, &quiz.Answer{Value: bp(false)}, quiz.Partial, "0"},
		{"tf sai ngược", quiz.TrueFalse, quiz.Key{Value: bp(false)}, &quiz.Answer{Value: bp(true)}, quiz.Partial, "0"},
		{"tf trống nil", quiz.TrueFalse, quiz.Key{Value: bp(true)}, nil, quiz.Partial, "0"},
		{"tf trống Value nil", quiz.TrueFalse, quiz.Key{Value: bp(true)}, &quiz.Answer{}, quiz.Partial, "0"},
		{"tf bỏ qua option_ids", quiz.TrueFalse, quiz.Key{Value: bp(true)}, &quiz.Answer{Value: bp(true), OptionIDs: []string{"zzz"}}, quiz.Partial, "2"},
		{"multi partial đủ", quiz.MCQMulti, quiz.Key{OptionIDs: []string{"a", "b", "c"}}, &quiz.Answer{OptionIDs: []string{"c", "b", "a"}}, quiz.Partial, "2"},
		{"multi partial thiếu 2/3", quiz.MCQMulti, quiz.Key{OptionIDs: []string{"a", "b", "c"}}, &quiz.Answer{OptionIDs: []string{"a", "b"}}, quiz.Partial, "1.3333333333333333"},
		{"multi partial thiếu 1/3", quiz.MCQMulti, quiz.Key{OptionIDs: []string{"a", "b", "c"}}, &quiz.Answer{OptionIDs: []string{"a"}}, quiz.Partial, "0.6666666666666667"},
		{"multi partial thừa 1", quiz.MCQMulti, quiz.Key{OptionIDs: []string{"a", "b"}}, &quiz.Answer{OptionIDs: []string{"a", "b", "c"}}, quiz.Partial, "1"},
		{"multi partial thừa nhiều không âm", quiz.MCQMulti, quiz.Key{OptionIDs: []string{"a", "b"}}, &quiz.Answer{OptionIDs: []string{"a", "c", "d", "e"}}, quiz.Partial, "0"},
		{"multi partial toàn sai", quiz.MCQMulti, quiz.Key{OptionIDs: []string{"a", "b"}}, &quiz.Answer{OptionIDs: []string{"c", "d"}}, quiz.Partial, "0"},
		{"multi partial trống", quiz.MCQMulti, quiz.Key{OptionIDs: []string{"a", "b"}}, nil, quiz.Partial, "0"},
		{"multi partial trùng gộp", quiz.MCQMulti, quiz.Key{OptionIDs: []string{"a", "b"}}, &quiz.Answer{OptionIDs: []string{"a", "a", "a"}}, quiz.Partial, "1"},
		{"multi partial K=1", quiz.MCQMulti, quiz.Key{OptionIDs: []string{"a"}}, &quiz.Answer{OptionIDs: []string{"a"}}, quiz.Partial, "2"},
		{"multi all-or-nothing đủ", quiz.MCQMulti, quiz.Key{OptionIDs: []string{"a", "b"}}, &quiz.Answer{OptionIDs: []string{"b", "a"}}, quiz.AllOrNothing, "2"},
		{"multi all-or-nothing thiếu", quiz.MCQMulti, quiz.Key{OptionIDs: []string{"a", "b"}}, &quiz.Answer{OptionIDs: []string{"a"}}, quiz.AllOrNothing, "0"},
		{"multi all-or-nothing thừa", quiz.MCQMulti, quiz.Key{OptionIDs: []string{"a", "b"}}, &quiz.Answer{OptionIDs: []string{"a", "b", "c"}}, quiz.AllOrNothing, "0"},
		{"multi all-or-nothing trống", quiz.MCQMulti, quiz.Key{OptionIDs: []string{"a", "b"}}, nil, quiz.AllOrNothing, "0"},
		{"multi all-or-nothing trùng + đủ", quiz.MCQMulti, quiz.Key{OptionIDs: []string{"a", "b"}}, &quiz.Answer{OptionIDs: []string{"a", "b", "b"}}, quiz.AllOrNothing, "2"},
		{"multi key trùng id vẫn đúng", quiz.MCQMulti, quiz.Key{OptionIDs: []string{"a", "a", "b"}}, &quiz.Answer{OptionIDs: []string{"a", "b"}}, quiz.AllOrNothing, "2"},
	}
	for _, c := range cases {
		got, err := quiz.Grade(c.typ, pts, c.key, c.ans, c.mode, opts...)
		require.NoError(t, err, c.name)
		require.Truef(t, got.Equal(d(c.want)), "%s: got %s want %s", c.name, got, c.want)
	}
}

// TestQuizPartialFormula — công thức `points × max(0, (TP − FP) ÷ K)` với K = 1…5 so với số hữu tỉ chính xác (big.Rat) làm chuẩn.
func TestQuizPartialFormula(t *testing.T) {
	t.Parallel()
	pts := d("3.00")
	n := 0
	for k := 1; k <= 5; k++ {
		key := quiz.Key{OptionIDs: opts[:k]}
		for tp := 0; tp <= k; tp++ {
			for fp := 0; fp <= 3; fp++ {
				if tp+fp == 0 {
					continue
				}
				chosen := append(append([]string{}, opts[:tp]...), opts[k:k+fp]...)
				got, err := quiz.Grade(quiz.MCQMulti, pts, key, &quiz.Answer{OptionIDs: chosen}, quiz.Partial, opts...)
				require.NoError(t, err)
				want := new(big.Rat)
				if net := tp - fp; net > 0 {
					want = big.NewRat(int64(3*net), int64(k))
				}
				// 16 chữ số thập phân là độ chính xác đã chốt: so sau khi làm tròn chuẩn cùng độ chính xác.
				require.Truef(t, got.Equal(d(want.FloatString(16)).Round(16)), "K=%d TP=%d FP=%d: got %s want %s", k, tp, fp, got, want.FloatString(16))
				require.False(t, got.Sign() < 0)
				require.False(t, got.GreaterThan(pts))
				n++
			}
		}
	}
	require.Greater(t, n, 40)
}

// TestQuizInvalidOption — id không thuộc câu → ErrInvalidOption; khoá hỏng → ErrBadKey; loại lạ → lỗi.
func TestQuizInvalidOption(t *testing.T) {
	t.Parallel()
	pts := d("1")
	_, err := quiz.Grade(quiz.MCQSingle, pts, quiz.Key{OptionIDs: []string{"a"}}, &quiz.Answer{OptionIDs: []string{"zzz"}}, quiz.Partial, opts...)
	require.ErrorIs(t, err, quiz.ErrInvalidOption)
	_, err = quiz.Grade(quiz.MCQMulti, pts, quiz.Key{OptionIDs: []string{"a", "b"}}, &quiz.Answer{OptionIDs: []string{"a", "zzz"}}, quiz.Partial, opts...)
	require.ErrorIs(t, err, quiz.ErrInvalidOption)
	_, err = quiz.Grade(quiz.MCQSingle, pts, quiz.Key{OptionIDs: []string{"a", "b"}}, nil, quiz.Partial)
	require.ErrorIs(t, err, quiz.ErrBadKey)
	_, err = quiz.Grade(quiz.MCQMulti, pts, quiz.Key{}, nil, quiz.Partial)
	require.ErrorIs(t, err, quiz.ErrBadKey)
	_, err = quiz.Grade(quiz.TrueFalse, pts, quiz.Key{}, &quiz.Answer{Value: bp(true)}, quiz.Partial)
	require.ErrorIs(t, err, quiz.ErrBadKey)
	_, err = quiz.Grade("CODE", pts, quiz.Key{}, nil, quiz.Partial)
	require.Error(t, err)
	_, err = quiz.Grade(quiz.MCQMulti, pts, quiz.Key{OptionIDs: []string{"a", "b"}}, &quiz.Answer{OptionIDs: []string{"a"}}, "BOGUS")
	require.Error(t, err)
	// không có danh sách đáp án để đối chiếu → không kiểm id lạ (khi chấm lại id đã xoá coi như không đúng)
	got, err := quiz.Grade(quiz.MCQSingle, pts, quiz.Key{OptionIDs: []string{"a"}}, &quiz.Answer{OptionIDs: []string{"zzz"}}, quiz.Partial)
	require.NoError(t, err)
	require.True(t, got.IsZero())
}
