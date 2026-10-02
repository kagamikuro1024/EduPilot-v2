package contract

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"
)

// Routes liệt kê mọi (method, mẫu đường dẫn) đã đăng ký trong router chi, đã chuẩn hoá `{x}` → `{}`, dạng "METHOD /path".
func Routes(h http.Handler) ([]string, error) {
	rt, ok := h.(chi.Routes)
	if !ok {
		return nil, fmt.Errorf("router không phải chi.Routes (%T)", h)
	}
	set := map[string]struct{}{}
	err := chi.Walk(rt, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		route = strings.TrimSuffix(route, "/")
		if route == "" {
			route = "/"
		}
		set[method+" "+NormalizePath(route)] = struct{}{}
		return nil
	})
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out, nil
}

// SpecRoutes liệt kê "METHOD /path" (đã chuẩn hoá) của một spec, lọc theo đường dẫn thuộc nhóm thử hay không.
func SpecRoutes(s *Spec, testGroup bool) []string {
	var out []string
	for _, o := range s.Operations() {
		if strings.HasPrefix(o.Path, TestPathPrefix) != testGroup {
			continue
		}
		out = append(out, o.Method+" "+NormalizePath(o.Path))
	}
	sort.Strings(out)
	return out
}

// CodeRoutes lọc route trong mã theo nhóm (thử / sản xuất).
func CodeRoutes(all []string, testGroup bool) []string {
	var out []string
	for _, r := range all {
		p := r[strings.Index(r, " ")+1:]
		if strings.HasPrefix(p, TestPathPrefix) != testGroup {
			continue
		}
		out = append(out, r)
	}
	return out
}

// Diff trả (có trong a nhưng không có trong b).
func Diff(a, b []string) []string {
	m := map[string]struct{}{}
	for _, x := range b {
		m[x] = struct{}{}
	}
	var out []string
	for _, x := range a {
		if _, ok := m[x]; !ok {
			out = append(out, x)
		}
	}
	return out
}
