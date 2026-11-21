package grading_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/edupilot/backend-go/internal/grading"
)

// P-11-26(c): các mẫu bổ sung bắt 5/5 biến thể không từ khoá điểm VÀ cờ nhầm = 0 trên mọi bài sạch có trong repo (nếu thêm mẫu làm một bài sạch bị cờ, test đỏ ⇒ bỏ mẫu).
func TestInjectionExtraPatterns(t *testing.T) {
	variants, _ := filepath.Glob("../../../benchmarks/real-llm/data/injection/*.md")
	if len(variants) != 5 {
		t.Fatalf("cần 5 biến thể, có %d", len(variants))
	}
	for _, f := range variants {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if len(grading.ScanInjection(string(b))) == 0 {
			t.Errorf("%s: không bị gắn cờ", filepath.Base(f))
		}
	}
	var clean []string
	for _, g := range []string{
		"../../../docs/specs/FEAT-assignments/fixtures/submissions/*.md",
		"../../../benchmarks/guidance/kc/data/bt-a/*.txt", "../../../benchmarks/guidance/kc/data/bt-b/*.txt",
		"../../../seed/privacy/*.txt",
	} {
		fs, _ := filepath.Glob(g)
		clean = append(clean, fs...)
	}
	if len(clean) < 90 {
		t.Fatalf("bộ sạch chỉ có %d tệp, muốn ≥ 90", len(clean))
	}
	for _, f := range clean {
		if filepath.Base(f) == "s6-injection.md" {
			continue // mẫu chèn lệnh có từ khoá, phải bị cờ (đã có test riêng)
		}
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if h := grading.ScanInjection(string(b)); len(h) > 0 {
			t.Errorf("%s: cờ nhầm %v", filepath.Base(f), h[0].Codes)
		}
	}
}
