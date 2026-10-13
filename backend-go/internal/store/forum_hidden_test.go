package store_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestForumQueriesFilterHidden — US-P4-06 AC16 (TLR-22): MỌI truy vấn trong store/queries/*.sql đọc `forum_threads` / `forum_posts` (from / join) phải có vị từ
// `hidden_at is null` (kể cả dạng `hidden_at is null or <Staff / chủ bài>`) hoặc nằm trong allowlist Staff / nội bộ ghi bằng dòng `-- hidden-ok: <lý do>`.
// Quét SQL thật, không dựa danh sách viết tay: truy vấn mới đọc bài mà quên lọc sẽ làm test đỏ.
func TestForumQueriesFilterHidden(t *testing.T) {
	files, err := filepath.Glob("queries/*.sql")
	if err != nil || len(files) == 0 {
		t.Fatalf("không thấy queries/*.sql: %v", err)
	}
	from := regexp.MustCompile(`(?i)\b(from|join)\s+forum_(threads|posts)\b`)
	scanned := 0
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, block := range strings.Split(string(raw), "-- name:")[1:] {
			if !from.MatchString(block) {
				continue
			}
			scanned++
			low := strings.ToLower(block)
			if strings.Contains(low, "hidden_at is null") || strings.Contains(low, "-- hidden-ok:") {
				continue
			}
			name := strings.TrimSpace(strings.SplitN(block, "\n", 2)[0])
			t.Errorf("%s: truy vấn %q đọc forum_threads / forum_posts mà không có `hidden_at is null` cũng không có `-- hidden-ok: <lý do>`", f, name)
		}
	}
	if scanned < 10 {
		t.Fatalf("chỉ quét được %d truy vấn forum_* — bộ quét hỏng?", scanned)
	}
}
