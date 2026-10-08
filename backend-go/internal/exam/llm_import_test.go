package exam_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestOnlySuggestImportsLLM — US-PE-01 AC8: trong gói `internal/exam`, chỉ `suggest.go` được import `internal/llm` (gợi ý nháp, US-PE-03);
// không prompt nào tính hay chấm điểm bài thi. Kiểm TỪNG tệp (`go list` chỉ thấy Imports của cả gói — góp ý #10). `quiz` không import llm ở bất kỳ tệp nào.
func TestOnlySuggestImportsLLM(t *testing.T) {
	t.Parallel()
	check := func(dir string, allowed map[string]bool) int {
		files, err := filepath.Glob(filepath.Join(dir, "*.go"))
		require.NoError(t, err)
		n := 0
		for _, f := range files {
			if strings.HasSuffix(f, "_test.go") {
				continue
			}
			af, err := parser.ParseFile(token.NewFileSet(), f, nil, parser.ImportsOnly)
			require.NoError(t, err, f)
			for _, imp := range af.Imports {
				if strings.Contains(imp.Path.Value, "internal/llm") {
					require.True(t, allowed[filepath.Base(f)], "%s import %s", f, imp.Path.Value)
					n++
				}
			}
		}
		return n
	}
	check(".", map[string]bool{"suggest.go": true})
	require.Zero(t, check("../quiz", nil))
	if _, err := os.Stat("../judge"); err == nil {
		require.Zero(t, check("../judge", nil))
	}
}
