package store_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestNoUUIDSliceParams: runtime chạy qua PgBouncer (QueryExecModeExec) nên `[]uuid.UUID` không mã hoá được thành tham số mảng (503 ở production,
// test dùng pool trực tiếp không thấy; D-AS-01, QC BUG-1 US-P3-05). Truy vấn sqlc phải nhận `sqlc.arg(x)::text[]::uuid[]` (Go: []string).
func TestNoUUIDSliceParams(t *testing.T) {
	files, err := filepath.Glob("*.sql.go")
	require.NoError(t, err)
	require.NotEmpty(t, files)
	fset := token.NewFileSet()
	for _, f := range files {
		file, err := parser.ParseFile(fset, f, nil, 0)
		require.NoError(t, err)
		for _, d := range file.Decls {
			gd, ok := d.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, sp := range gd.Specs {
				ts, ok := sp.(*ast.TypeSpec)
				if !ok || !strings.HasSuffix(ts.Name.Name, "Params") {
					continue
				}
				st, ok := ts.Type.(*ast.StructType)
				if !ok {
					continue
				}
				for _, fld := range st.Fields.List {
					at, ok := fld.Type.(*ast.ArrayType)
					if !ok || at.Len != nil {
						continue
					}
					if sel, ok := at.Elt.(*ast.SelectorExpr); ok && sel.Sel.Name == "UUID" {
						t.Errorf("%s: %s.%s là []uuid.UUID — đổi truy vấn sang ::text[]::uuid[]", f, ts.Name.Name, fld.Names[0].Name)
					}
				}
			}
		}
	}
}
