// Package thread là Threads (hỏi đáp công khai trong lớp) — SRS FEAT-private-chat-pii 4.9: tường lửa PII, đăng / bình luận, AI trả lời một lần trong việc nền,
// xác nhận / sửa / loại của Staff, thread tương tự. Một tường lửa duy nhất (`CheckPost`) cho mọi đường ghi bài công khai.
package thread

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/privacy"
)

// Redacted là chỗ thay cho thông tin cá nhân khi người dùng chọn "Ẩn thông tin rồi đăng".
const Redacted = "[đã ẩn]"

// Reason là một loại thông tin cá nhân và số lần xuất hiện (không có nội dung).
type Reason struct {
	Type  string `json:"type"` // MSSV | EMAIL | PHONE | CCCD | NAME | PERSONAL_QUESTION
	Count int    `json:"count"`
}

// Verdict là kết luận của tường lửa cho một bài (tiêu đề + nội dung).
type Verdict struct {
	Allowed       bool
	Reasons       []Reason
	RedactedTitle string
	RedactedBody  string
	// Personal: câu hỏi về dữ liệu riêng của người hỏi không có định danh để thay ("Em được mấy điểm?") — không có lối "Ẩn rồi đăng".
	Personal bool
	// Vec: vectơ của `title + "\n" + body` nếu phân loại đã nhúng (dùng lại khi lưu thread).
	Vec []float32
}

// Firewall là tường lửa PII của bài công khai.
type Firewall struct {
	Detector   *privacy.Detector
	Classifier *privacy.Classifier
	// Embed nhúng một chuỗi (làn INTERACTIVE, cache ep:emb); nil = chỉ luật.
	Embed func(ctx context.Context, text string) ([]float32, error)
}

// CheckPost: `Detect` trên tiêu đề và nội dung (từng phần, chỉ số rune của chính phần đó) + `Classify` câu hỏi cá nhân.
// `Allowed = không có Finding && !Personal`. `Redacted*` chỉ thay các Finding; câu hỏi cá nhân không có khoảng để thay.
// Đây là hàm DUY NHẤT được phép dùng trước khi lưu một bài công khai (đăng, bình luận; P4: sửa bài).
func (f *Firewall) CheckPost(ctx context.Context, courseID uuid.UUID, title, body string) (Verdict, error) {
	var v Verdict
	counts := map[string]int{}
	redact := func(text string) (string, error) {
		fs, err := f.Detector.Detect(ctx, courseID, text)
		if err != nil {
			return "", fmt.Errorf("thread: tường lửa: %w", err)
		}
		if len(fs) == 0 {
			return text, nil
		}
		r := []rune(text)
		var b strings.Builder
		pos := 0
		for _, x := range fs {
			counts[string(x.Kind)]++
			b.WriteString(string(r[pos:x.Start]))
			b.WriteString(Redacted)
			pos = x.End
		}
		b.WriteString(string(r[pos:]))
		return b.String(), nil
	}
	var err error
	if v.RedactedTitle, err = redact(title); err != nil {
		return v, err
	}
	if v.RedactedBody, err = redact(body); err != nil {
		return v, err
	}
	full := strings.TrimSpace(title + "\n" + body)
	var embed func(context.Context) ([]float32, error)
	if f.Embed != nil {
		embed = func(c context.Context) ([]float32, error) { return f.Embed(c, full) }
	}
	res, vec, err := f.Classifier.Classify(ctx, courseID, full, embed)
	if err != nil {
		return v, fmt.Errorf("thread: phân loại: %w", err)
	}
	for _, r := range res.Reasons {
		if r == privacy.ReasonPattern || r == privacy.ReasonSimilarty {
			v.Personal = true
		}
	}
	if v.Personal {
		counts["PERSONAL_QUESTION"] = 1
	}
	v.Vec = vec
	for k, n := range counts {
		v.Reasons = append(v.Reasons, Reason{Type: k, Count: n})
	}
	sort.Slice(v.Reasons, func(i, j int) bool { return v.Reasons[i].Type < v.Reasons[j].Type })
	v.Allowed = len(v.Reasons) == 0
	if v.Personal && len(counts) == 1 { // chỉ là câu hỏi cá nhân: không có khoảng để thay
		v.RedactedTitle, v.RedactedBody = title, body
	}
	return v, nil
}
