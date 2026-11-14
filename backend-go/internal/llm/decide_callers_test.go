package llm_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// decideCallers là DANH SÁCH TRẮNG các hàm được gọi Decide / Cascade ngoài gói llm (D60, US-DM-04 AC9, US-DM-09): chỉ ba cổng đã định nghĩa.
// Khủng hoảng, tính điểm, mức rubric, tổng, làm tròn, escalate, `internal/grade` không bao giờ nằm đây; thêm một dòng ở đây = đổi ranh giới → cần proposal.
var decideCallers = map[string]bool{
	"thread.CheckPublish":           true, // THREADS_PRECHECK
	"grading.flagInjectionDecision": true, // INJECTION_FLAG
	"agent.routeDecision":           true, // CHAT_ROUTING (L4)
}

// unrelated: hàm trùng tên `Decide` nhưng KHÔNG phải decision model (quyết định của giảng viên về câu trả lời AI ở threadhttp). Phân biệt theo tên vì quét cú pháp không có thông tin kiểu.
var unrelated = map[string]bool{"threadhttp.decide": true} //nolint:gochecknoglobals // bảng bất biến cho test

// TestDecideCallersAllowlist quét mã nguồn không-test: mọi lời gọi `.Decide(` / `.Cascade(` ngoài internal/llm phải nằm trong decideCallers, và mỗi mục trong danh sách phải còn được dùng.
func TestDecideCallersAllowlist(t *testing.T) {
	t.Parallel()
	root := filepath.Join("..", "..") // backend-go
	seen := map[string]bool{}
	var bad []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		slash := filepath.ToSlash(path)
		if d.IsDir() && (strings.HasSuffix(slash, "/internal/llm") || strings.Contains(slash, "/internal/llm/") || d.Name() == "vendor" || d.Name() == "legacy") {
			return filepath.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, perr := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if perr != nil {
			return perr
		}
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || (sel.Sel.Name != "Decide" && sel.Sel.Name != "Cascade") {
					return true
				}
				name := f.Name.Name + "." + fn.Name.Name
				if unrelated[name] {
					return true
				}
				if decideCallers[name] {
					seen[name] = true
				} else {
					bad = append(bad, name+" ("+filepath.ToSlash(path)+")")
				}
				return true
			})
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(bad)
	if len(bad) > 0 {
		t.Fatalf("gọi Decide / Cascade ngoài danh sách trắng: %v", bad)
	}
	for name := range decideCallers {
		if !seen[name] && name != "agent.routeDecision" { // agent.routeDecision: thêm ở L4
			t.Errorf("%s nằm trong danh sách trắng nhưng không còn lời gọi", name)
		}
	}
}
