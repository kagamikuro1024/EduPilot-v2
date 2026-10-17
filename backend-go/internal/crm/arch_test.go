package crm_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var sensitiveTables = []string{"student_risk", "student_observations", "v_student_360"} //nolint:gochecknoglobals // danh sách tra cứu bất biến

// sensitiveQueries liệt kê hàm sqlc (tên Go) có câu SQL chạm bảng / view dữ liệu rủi ro hoặc ghi chú quan sát.
func sensitiveQueries(t *testing.T) map[string]bool {
	t.Helper()
	files, err := filepath.Glob("../store/*.sql.go")
	require.NoError(t, err)
	re := regexp.MustCompile("(?s)-- name: (\\w+) :\\w+\\n(.*?)`")
	out := map[string]bool{}
	for _, f := range files {
		b, err := os.ReadFile(f)
		require.NoError(t, err)
		for _, m := range re.FindAllStringSubmatch(string(b), -1) {
			for _, tb := range sensitiveTables {
				if strings.Contains(m[2], tb) {
					out[m[1]] = true
				}
			}
		}
	}
	return out
}

// sensitiveCallers trả "tệp: hàm" cho mọi lời gọi hàm nhạy cảm ở mã không-test của các gói `roots` (so với thư mục internal/).
func sensitiveCallers(t *testing.T, names map[string]bool, roots ...string) []string {
	t.Helper()
	var bad []string
	for _, root := range roots {
		require.NoError(t, filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") || strings.HasSuffix(p, ".sql.go") {
				return err
			}
			rel := filepath.ToSlash(p)
			if strings.Contains(rel, "internal/crm/") || strings.Contains(rel, "internal/store/") || strings.HasPrefix(rel, "../crm/") || strings.HasPrefix(rel, "../store/") {
				return nil
			}
			f, perr := parser.ParseFile(token.NewFileSet(), p, nil, 0)
			require.NoError(t, perr)
			ast.Inspect(f, func(n ast.Node) bool {
				if s, ok := n.(*ast.SelectorExpr); ok && names[s.Sel.Name] {
					bad = append(bad, rel+": "+s.Sel.Name)
				}
				return true
			})
			return nil
		}))
	}
	return bad
}

// TestSensitiveQueriesOnlyFromCRM: hàm sqlc của `student_risk` / `student_observations` / `v_student_360` chỉ được gọi từ internal/crm (US-P5-04 AC9) — sinh viên không bao giờ thấy nhãn rủi ro.
func TestSensitiveQueriesOnlyFromCRM(t *testing.T) {
	names := sensitiveQueries(t)
	require.NotEmpty(t, names)
	require.True(t, names["CrmRiskUpsert"] && names["CrmStudentsList"] && names["CrmAttention"], "bộ quét phải thấy các hàm của risk.sql: %v", names)
	require.Empty(t, sensitiveCallers(t, names, "..", "../../cmd"))
}
