package contract

import (
	"os"
	"strings"
	"testing"
)

// DEF-R2-01 (TC-P0-02/21): `parameters:` để trống (= null) làm `redocly lint` báo lỗi struct; để không có tham số thì bỏ hẳn khoá.
func TestOpenAPINoEmptyParameters(t *testing.T) {
	b, err := os.ReadFile("../../api/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(b), "\n")
	indent := func(s string) int { return len(s) - len(strings.TrimLeft(s, " ")) }
	for i, l := range lines {
		if strings.TrimSpace(l) != "parameters:" {
			continue
		}
		for j := i + 1; j < len(lines); j++ {
			if strings.TrimSpace(lines[j]) == "" {
				continue
			}
			if indent(lines[j]) <= indent(l) && !strings.HasPrefix(strings.TrimSpace(lines[j]), "- ") {
				t.Errorf("openapi.yaml:%d: `parameters:` rỗng (null) — bỏ khoá hoặc dùng []", i+1)
			}
			break
		}
	}
}
