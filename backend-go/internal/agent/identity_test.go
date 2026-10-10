package agent

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestTrustedContextOnlyIdentitySource — AC15: gói agent không có hàm / phương thức nào nhận user_id / mã sinh viên làm tham số (trừ qua TrustedContext),
// và không nhập net/http (không có đường nhận danh tính từ yêu cầu).
func TestTrustedContextOnlyIdentitySource(t *testing.T) {
	t.Parallel()
	banned := map[string]bool{"userid": true, "user_id": true, "studentid": true, "studentcode": true, "student_code": true, "mssv": true, "uid": true, "email": true, "fullname": true}
	files, err := filepath.Glob("*.go")
	require.NoError(t, err)
	fset := token.NewFileSet()
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		require.NoError(t, err)
		file, err := parser.ParseFile(fset, f, src, 0)
		require.NoError(t, err)
		for _, im := range file.Imports {
			require.NotEqual(t, `"net/http"`, im.Path.Value, "%s nhập net/http", f)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			var ft *ast.FuncType
			switch x := n.(type) {
			case *ast.FuncDecl:
				ft = x.Type
			case *ast.FuncLit:
				ft = x.Type
			}
			if ft == nil {
				return true
			}
			for _, p := range ft.Params.List {
				for _, name := range p.Names {
					require.False(t, banned[strings.ToLower(name.Name)], "%s: tham số danh tính %q", f, name.Name)
				}
			}
			return true
		})
	}
}
