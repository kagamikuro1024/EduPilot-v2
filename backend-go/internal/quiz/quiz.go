// Package quiz là Quiz Engine của bài thi (SRS FEAT-weekly-exam 4.1.3): chấm trắc nghiệm bằng CODE THUẦN. Không import `internal/llm`,
// không `float64` cho điểm (AGENTS luật 5) — mọi phép tính là `shopspring/decimal`.
package quiz

import (
	"errors"
	"fmt"

	"github.com/shopspring/decimal"
)

// Type là loại câu trắc nghiệm mà Grade chấm (câu CODE do `internal/judge` chấm).
type Type string

const (
	MCQSingle Type = "MCQ_SINGLE"
	MCQMulti  Type = "MCQ_MULTI"
	TrueFalse Type = "TRUE_FALSE"
)

// Mode là cách chấm câu nhiều đáp án của bài thi.
type Mode string

const (
	AllOrNothing Mode = "ALL_OR_NOTHING"
	Partial      Mode = "PARTIAL"
)

// Key là đáp án đúng (`question_bank.answer_key`): `OptionIDs` cho MCQ, `Value` cho TRUE_FALSE.
type Key struct {
	OptionIDs []string
	Value     *bool
}

// Answer là câu trả lời của sinh viên; con trỏ nil = bỏ trống.
type Answer struct {
	OptionIDs []string
	Value     *bool
}

// ErrInvalidOption: câu trả lời chọn id không thuộc câu (API lưu → 422 INVALID_OPTION_ID; khi chấm lại: coi như chưa chọn).
var ErrInvalidOption = errors.New("quiz: id đáp án không thuộc câu")

// ErrBadKey: đáp án đúng không hợp lệ với loại câu (thiếu / thừa — lỗi dữ liệu, không phải lỗi của sinh viên).
var ErrBadKey = errors.New("quiz: đáp án đúng không hợp lệ")

// Grade chấm một câu trắc nghiệm, trả điểm trong [0, points]. `options` là tập id đáp án của câu (để kiểm id lạ); nil = không kiểm.
func Grade(t Type, points decimal.Decimal, key Key, a *Answer, mode Mode, options ...string) (decimal.Decimal, error) {
	if points.Sign() < 0 {
		return decimal.Zero, fmt.Errorf("quiz: points âm")
	}
	switch t {
	case TrueFalse:
		if key.Value == nil {
			return decimal.Zero, ErrBadKey
		}
		if a == nil || a.Value == nil {
			return decimal.Zero, nil
		}
		if *a.Value == *key.Value {
			return points, nil
		}
		return decimal.Zero, nil
	case MCQSingle, MCQMulti:
	default:
		return decimal.Zero, fmt.Errorf("quiz: loại câu %q không do Quiz Engine chấm", t)
	}

	correct := dedupe(key.OptionIDs)
	if len(correct) == 0 || (t == MCQSingle && len(correct) != 1) {
		return decimal.Zero, ErrBadKey
	}
	if a == nil {
		return decimal.Zero, nil
	}
	chosen := dedupe(a.OptionIDs)
	if len(options) > 0 {
		valid := make(map[string]struct{}, len(options))
		for _, o := range options {
			valid[o] = struct{}{}
		}
		for _, c := range chosen {
			if _, ok := valid[c]; !ok {
				return decimal.Zero, ErrInvalidOption
			}
		}
	}
	if len(chosen) == 0 {
		return decimal.Zero, nil
	}

	right := make(map[string]struct{}, len(correct))
	for _, c := range correct {
		right[c] = struct{}{}
	}
	tp := 0
	for _, c := range chosen {
		if _, ok := right[c]; ok {
			tp++
		}
	}
	fp := len(chosen) - tp

	if t == MCQSingle {
		if len(chosen) == 1 && tp == 1 {
			return points, nil
		}
		return decimal.Zero, nil
	}
	switch mode {
	case AllOrNothing:
		if tp == len(correct) && fp == 0 {
			return points, nil
		}
		return decimal.Zero, nil
	case Partial:
		net := tp - fp
		if net <= 0 {
			return decimal.Zero, nil
		}
		return points.Mul(decimal.NewFromInt(int64(net))).DivRound(decimal.NewFromInt(int64(len(correct))), 16), nil
	default:
		return decimal.Zero, fmt.Errorf("quiz: chế độ chấm %q không hợp lệ", mode)
	}
}

// dedupe khử trùng id nhưng giữ thứ tự xuất hiện (kết quả không phụ thuộc thứ tự).
func dedupe(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
