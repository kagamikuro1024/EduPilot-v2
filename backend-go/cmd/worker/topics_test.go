package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// TestEveryEmittedTopicHasHandler — QC BUG-1 / BUG-1b của US-P3-06: MỌI topic mà mã sản xuất ghi vào outbox (outbox.Write) phải có handler ở bảng của worker;
// topic không có handler rơi vào dead-letter sau 4 lần. Quét AST toàn `internal/`: lấy đối số thứ ba của `outbox.Write` (literal hoặc hằng `Topic…` / `TopicEnqueue`).
func TestEveryEmittedTopicHasHandler(t *testing.T) {
	d, _ := workerDeps(t)
	reg := newRegistry(d)

	consts := map[string][]string{} // tên hằng → các giá trị (cùng tên có thể ở nhiều gói)
	type use struct{ file, name, lit string }
	var uses []use
	fset := token.NewFileSet()
	err := filepath.WalkDir("../../internal", func(p string, de fs.DirEntry, err error) error {
		if err != nil || de.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return err
		}
		f, err := parser.ParseFile(fset, p, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.ValueSpec:
				for i, id := range x.Names {
					if i < len(x.Values) {
						if bl, ok := x.Values[i].(*ast.BasicLit); ok && bl.Kind == token.STRING && strings.HasPrefix(id.Name, "Topic") {
							v, _ := strconv.Unquote(bl.Value)
							consts[id.Name] = append(consts[id.Name], v)
						}
					}
				}
			case *ast.CallExpr:
				sel, ok := x.Fun.(*ast.SelectorExpr)
				if !ok || sel.Sel.Name != "Write" || len(x.Args) < 3 {
					return true
				}
				if pk, ok := sel.X.(*ast.Ident); !ok || pk.Name != "outbox" {
					return true
				}
				switch a := x.Args[2].(type) {
				case *ast.BasicLit:
					v, _ := strconv.Unquote(a.Value)
					uses = append(uses, use{file: p, lit: v})
				case *ast.Ident:
					uses = append(uses, use{file: p, name: a.Name})
				case *ast.SelectorExpr:
					uses = append(uses, use{file: p, name: a.Sel.Name})
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(uses) < 20 {
		t.Fatalf("chỉ thấy %d lời gọi outbox.Write — quét sai", len(uses))
	}
	topics := map[string]string{}
	for _, u := range uses {
		switch {
		case u.lit != "":
			topics[u.lit] = u.file
		case consts[u.name] != nil:
			for _, v := range consts[u.name] {
				topics[v] = u.file
			}
		case u.name == "Topic": // mail.Topic
			topics["mail.send"] = u.file
		case strings.HasSuffix(u.file, "exam/tick.go"): // biến `topic`: exam.opened / exam.closed (có hằng riêng, được quét ở nhánh trên)
			topics["exam.opened"], topics["exam.closed"] = u.file, u.file
		default:
			t.Errorf("%s: không xác định được topic của outbox.Write (%q) — dùng hằng `Topic…` hoặc literal", u.file, u.name)
		}
	}
	var names []string
	for k := range topics {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, topic := range names {
		if topic == "judge.enqueue" { // chỉ đăng ký khi worker có cổng chấm (d.Judge != nil); môi trường test không có
			continue
		}
		if _, ok := reg.Lookup(topic); !ok {
			t.Errorf("topic %q (phát ở %s) chưa có handler ở worker", topic, topics[topic])
		}
	}
}
