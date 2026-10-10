package contract

import (
	"slices"
	"testing"
)

// TestChatThreadsContract — US-P3-08 AC8: 20 thao tác của SRS FEAT-private-chat-pii 6 (tag `chat` 10 + `threads` 10) đều có trong openapi.yaml; mọi status đã khai báo được
// gọi thật (hoặc miễn trừ có lý do), mọi phản hồi khớp schema (kể cả 4xx) và phản hồi của sinh viên không có khoá cấm (confidence, retrieval_score, groundedness, ai_body, hidden_reason).
func TestChatThreadsContract(t *testing.T) {
	prod, _ := loadBoth(t)
	var keys []string
	for _, o := range prod.Operations() {
		if slices.ContainsFunc(o.Op.Tags, func(tag string) bool { return tag == "chat" || tag == "threads" }) {
			keys = append(keys, o.Key())
		}
	}
	if len(keys) != 20 {
		t.Fatalf("chat + threads: %d thao tác (cần 20): %v", len(keys), keys)
	}
	seen := map[string]bool{}
	for _, o := range observations(t) {
		if !slices.Contains(keys, o.Op) {
			continue
		}
		seen[o.Op+"|"+itoa(o.Status)] = true
		if o.Err != nil {
			t.Errorf("%s (status %d): %v", o.Requested, o.Status, o.Err)
		}
		if o.Leak != "" {
			t.Errorf("%s: phản hồi của sinh viên lộ khoá cấm %q", o.Requested, o.Leak)
		}
	}
	for _, o := range prod.Operations() {
		if !slices.Contains(keys, o.Key()) {
			continue
		}
		for _, st := range o.Statuses() {
			if !seen[o.Key()+"|"+itoa(st)] && !IsExempt(o.Key(), st) {
				t.Errorf("%s khai báo status %d nhưng không test nào sinh ra", o.Key(), st)
			}
		}
	}
}
