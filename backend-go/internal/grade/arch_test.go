package grade_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func nonTestFiles(t *testing.T) []string {
	t.Helper()
	all, err := filepath.Glob("*.go")
	require.NoError(t, err)
	var out []string
	for _, f := range all {
		if !strings.HasSuffix(f, "_test.go") {
			out = append(out, f)
		}
	}
	require.NotEmpty(t, out)
	return out
}

// TestNoFloatInGrade — AC4: không float32/float64, math.Round/Floor/Ceil, ParseFloat, RoundBank, StringFixedBank, StringFixed ở mã không-test (quét AST).
func TestNoFloatInGrade(t *testing.T) {
	for _, f := range nonTestFiles(t) {
		file, err := parser.ParseFile(token.NewFileSet(), f, nil, 0)
		require.NoError(t, err)
		ast.Inspect(file, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.Ident:
				require.NotContains(t, []string{"float32", "float64"}, x.Name, "%s: %s", f, x.Name)
			case *ast.SelectorExpr:
				if id, ok := x.X.(*ast.Ident); ok && id.Name == "math" {
					require.NotContains(t, []string{"Round", "Floor", "Ceil", "Trunc", "RoundToEven"}, x.Sel.Name, "%s: math.%s", f, x.Sel.Name)
				}
				require.NotContains(t, []string{"ParseFloat", "RoundBank", "StringFixedBank", "StringFixed", "Float64", "InexactFloat64", "NewFromFloat"}, x.Sel.Name, "%s: .%s", f, x.Sel.Name)
			case *ast.BasicLit:
				require.NotEqual(t, token.FLOAT, x.Kind, "%s: hằng float %s", f, x.Value)
			}
			return true
		})
	}
}

// TestForbiddenImports — AC5: go list -deps không có activity, crm, llm, grading, privacy, agent, openai-go (D17: thời gian học không thành điểm).
func TestForbiddenImports(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", ".").CombinedOutput()
	require.NoError(t, err, string(out))
	for _, bad := range []string{"internal/activity", "internal/crm", "internal/llm", "internal/grading", "internal/privacy", "internal/agent", "openai-go"} {
		require.NotContains(t, string(out), bad)
	}
}

// TestEngineIsPure — AC5: Compute / WhatIf / MinFinalFor không nhận context; engine.go không import context, database, mạng, đồng hồ, store.
func TestEngineIsPure(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "engine.go", nil, 0)
	require.NoError(t, err)
	for _, im := range file.Imports {
		for _, bad := range []string{"context", "net/", "database", "pgx", "internal/store", "time", "os", "math/rand"} {
			require.False(t, strings.Contains(im.Path.Value, bad), "engine.go import %s", im.Path.Value)
		}
	}
	for _, d := range file.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok && fn.Recv == nil {
			for _, p := range fn.Type.Params.List {
				require.NotContains(t, exprString(p.Type), "context", fn.Name.Name)
			}
		}
	}
}

func exprString(e ast.Expr) string {
	switch x := e.(type) {
	case *ast.SelectorExpr:
		return exprString(x.X) + "." + x.Sel.Name
	case *ast.Ident:
		return x.Name
	case *ast.StarExpr:
		return "*" + exprString(x.X)
	}
	return ""
}

// TestNoPolicyConstants — AC6: mã tính điểm không có hằng mang nghĩa quy chế (0.4, 0.6, 0.25, 0.5, 3.0, 7); chỉ 10 (thang) và 100 (tổng trọng số) ở types.go.
// (Tập bước làm tròn cho phép ở rules.go là miền hợp lệ của tham số, không phải chính sách.)
func TestNoPolicyConstants(t *testing.T) {
	policy := map[string]bool{"0.4": true, "0.6": true, "0.25": true, "0.5": true, "3.0": true, "7": true, "40": true, "60": true, "20": true}
	for _, f := range []string{"engine.go", "inputs.go"} {
		file, err := parser.ParseFile(token.NewFileSet(), f, nil, 0)
		require.NoError(t, err)
		ast.Inspect(file, func(n ast.Node) bool {
			if b, ok := n.(*ast.BasicLit); ok {
				v := strings.Trim(b.Value, "\"")
				require.False(t, policy[v], "%s: hằng %s", f, b.Value)
			}
			return true
		})
	}
	src, err := os.ReadFile("engine.go")
	require.NoError(t, err)
	require.NotContains(t, string(src), "RoundBank")
}
