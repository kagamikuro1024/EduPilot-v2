package agent

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/privacy"
)

// Self là danh tính của CHÍNH người đang chat (để phân biệt nhắc chính mình với hỏi hộ người khác). Lấy theo TrustedContext, không từ thân yêu cầu.
type Self struct {
	NameKey string // họ tên không dấu, chữ thường, gộp khoảng trắng
	Code    string // student_code_snapshot, chữ thường
	Email   string // chữ thường
}

// SelfSource tra danh tính của người trong TrustedContext.
type SelfSource interface {
	Self(ctx context.Context, tc TrustedContext) (Self, error)
}

// EventSink ghi sự kiện PII (chỉ loại + số lượng, không nội dung).
type EventSink interface {
	Record(ctx context.Context, tc TrustedContext, pii, action string, channel privacy.Channel, count int) error
}

// Classification là kết quả phân loại MỘT lần cho một tin nhắn.
type Classification struct {
	Class       privacy.Result
	Intent      Intent
	Vec         []float32 // vectơ đã nhúng (nil nếu luật quyết rõ) — rag.SearchStudent dùng lại
	OtherPerson bool
	// HasPII: tin nhắn có PII (khác mẫu câu cá nhân) — bị che quanh LLM nên KHÔNG bao giờ vào cache câu trả lời.
	HasPII bool
}

// Analyzer chạy phân loại kênh (luật + embedding) và quyết intent. Chạy đúng một lần mỗi tin nhắn.
type Analyzer struct {
	Classifier *privacy.Classifier
	Detector   *privacy.Detector
	Self       SelfSource
	// Embed nhúng một câu hỏi (làn INTERACTIVE, cache ep:emb) — chỉ được gọi khi luật chưa quyết.
	Embed func(ctx context.Context, text string) ([]float32, error)
	Log   *slog.Logger
}

// Analyze phân loại tin nhắn. Không gọi LLM sinh chữ.
func (a *Analyzer) Analyze(ctx context.Context, tc TrustedContext, text string) (Classification, error) {
	var embed func(context.Context) ([]float32, error)
	if a.Embed != nil {
		embed = func(c context.Context) ([]float32, error) { return a.Embed(c, text) }
	}
	res, vec, err := a.Classifier.Classify(ctx, tc.CourseID, text, embed)
	if err != nil {
		return Classification{}, fmt.Errorf("agent: phân loại: %w", err)
	}
	out := Classification{Class: res, Vec: vec, Intent: DetectIntent(text)}
	for _, r := range res.Reasons {
		switch r {
		case privacy.ReasonPIIMSSV, privacy.ReasonPIIEmail, privacy.ReasonPIIPhone, privacy.ReasonPIICCCD, privacy.ReasonPIIName:
			out.HasPII = true
		}
	}
	if out.Intent == IntentCrisis {
		return out, nil
	}
	// Hỏi hộ: ý định cá nhân + định danh KHÁC mình (roster ≠ mình, MSSV / email lạ — kể cả ngoài roster).
	if out.HasPII && (out.Intent.Personal() || hasPersonalKeyword(text)) {
		other, err := a.otherPerson(ctx, tc, text)
		if err != nil {
			return Classification{}, err
		}
		if other {
			out.OtherPerson, out.Intent = true, IntentOtherPerson
			return out, nil
		}
	}
	// Câu hỏi cá nhân mà luật intent không nhận ra: dùng mẫu / độ tương đồng → hỏi điểm của mình.
	if out.Intent == IntentCourseQA && res.Personal && res.Similarity > 0 {
		if pi := PersonalIntent(text); pi != "" {
			out.Intent = pi
		}
	}
	return out, nil
}

func (a *Analyzer) otherPerson(ctx context.Context, tc TrustedContext, text string) (bool, error) {
	ents, err := a.Detector.Entities(ctx, tc.CourseID, text)
	if err != nil {
		return false, fmt.Errorf("agent: thực thể: %w", err)
	}
	self, err := a.Self.Self(ctx, tc)
	if err != nil {
		return false, fmt.Errorf("agent: danh tính người hỏi: %w", err)
	}
	runes := []rune(text)
	for _, e := range ents {
		if isSelf(self, e) {
			continue
		}
		switch e.Kind {
		case privacy.KindPhone, privacy.KindCCCD:
			continue // số điện thoại / CCCD không định danh người trong lớp ở bước này; vẫn bị che quanh LLM
		case privacy.KindMSSV:
			if isSelfDeclaration(e.Start, runes) { // tự khai "MSSV của em là …": bỏ qua nếu không thuộc ai, từ chối nếu thuộc roster khác
				if !e.InRoster {
					continue
				}
			}
		}
		return true, nil
	}
	return false, nil
}

func isSelf(s Self, e privacy.Entity) bool {
	switch e.Kind {
	case privacy.KindName:
		return s.NameKey != "" && e.Key == s.NameKey
	case privacy.KindMSSV:
		return s.Code != "" && e.Key == s.Code
	case privacy.KindEmail:
		return s.Email != "" && e.Key == s.Email
	}
	return false
}

// NewSelf chuẩn hoá danh tính tra được từ DB.
func NewSelf(name, code, mail string) Self {
	return Self{NameKey: strings.Join(strings.Fields(auth.Fold(name)), " "), Code: strings.ToLower(strings.TrimSpace(code)), Email: strings.ToLower(strings.TrimSpace(mail))}
}
