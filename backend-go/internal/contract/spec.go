// Package contract: hợp đồng API khoá hai chiều (US-PG-06). `api/openapi.yaml` là nguồn sự thật của route sản xuất,
// `api/openapi.test.yaml` của route thử `/api/v1/_test/*`; các test của gói dựng gateway thật và đối chiếu response với spec.
// Đây là nơi DUY NHẤT (cùng các _test.go) được import kin-openapi (D52; lint depguard chặn chỗ khác).
package contract

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/legacy"
)

// TestPathPrefix là tiền tố của route thử (chỉ có trong binary dựng bằng build tag `testroutes`).
const TestPathPrefix = "/api/v1/_test/"

// Operation là một thao tác của spec.
type Operation struct {
	Method string
	Path   string // mẫu gốc của spec, ví dụ /api/v1/jobs/{id}
	ID     string
	Op     *openapi3.Operation
}

// Key là "METHOD /đường-dẫn" — dạng dùng trong thông báo lỗi.
func (o Operation) Key() string { return o.Method + " " + o.Path }

// Spec là một tệp OpenAPI đã nạp và kiểm hợp lệ.
type Spec struct {
	File   string
	Doc    *openapi3.T
	router routers.Router
	ops    []Operation
}

// APIDir là thư mục `backend-go/api` (suy từ vị trí mã nguồn của gói này).
func APIDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "api")
}

// ProdSpecPath là đường dẫn spec sản xuất; biến `OPENAPI_PATH` cho phép QC thử bản spec sửa tạm (không sửa repo).
func ProdSpecPath() string {
	if p := strings.TrimSpace(os.Getenv("OPENAPI_PATH")); p != "" {
		return p
	}
	return filepath.Join(APIDir(), "openapi.yaml")
}

// TestSpecPath là đường dẫn spec route thử.
func TestSpecPath() string { return filepath.Join(APIDir(), "openapi.test.yaml") }

// Load nạp và kiểm hợp lệ một tệp spec.
func Load(ctx context.Context, path string) (*Spec, error) {
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = false
	doc, err := loader.LoadFromFile(path)
	if err != nil {
		return nil, fmt.Errorf("nạp %s: %w", path, err)
	}
	if err := doc.Validate(ctx); err != nil {
		return nil, fmt.Errorf("spec %s không hợp lệ: %w", path, err)
	}
	// Router của kin-openapi so cả host với `servers`; spec dùng `servers: [{url: /}]` (tương đối) nên bỏ để khớp theo đường dẫn.
	doc.Servers = nil
	rt, err := legacy.NewRouter(doc)
	if err != nil {
		return nil, fmt.Errorf("dựng router cho %s: %w", path, err)
	}
	s := &Spec{File: path, Doc: doc, router: rt}
	for p, item := range doc.Paths.Map() {
		for m, op := range item.Operations() {
			s.ops = append(s.ops, Operation{Method: strings.ToUpper(m), Path: p, ID: op.OperationID, Op: op})
		}
	}
	sort.Slice(s.ops, func(i, j int) bool { return s.ops[i].Key() < s.ops[j].Key() })
	return s, nil
}

// Operations trả mọi thao tác, sắp theo "METHOD đường-dẫn".
func (s *Spec) Operations() []Operation { return s.ops }

// Find tìm thao tác khớp một request; ok=false nếu spec không khai báo đường dẫn/method đó.
func (s *Spec) Find(req *http.Request) (Operation, bool) {
	route, _, err := s.router.FindRoute(req)
	if err != nil {
		return Operation{}, false
	}
	return Operation{Method: strings.ToUpper(route.Method), Path: route.Path, ID: route.Operation.OperationID, Op: route.Operation}, true
}

// Declared cho biết thao tác có khai báo status (hoặc `default`).
func (o Operation) Declared(status int) bool {
	rs := o.Op.Responses
	if rs == nil {
		return false
	}
	if _, ok := rs.Map()[strconv.Itoa(status)]; ok {
		return true
	}
	return rs.Default() != nil
}

// Statuses trả các status đã khai báo (không tính `default`), tăng dần.
func (o Operation) Statuses() []int {
	var out []int
	if o.Op.Responses == nil {
		return nil
	}
	for k := range o.Op.Responses.Map() {
		if n, err := strconv.Atoi(k); err == nil {
			out = append(out, n)
		}
	}
	sort.Ints(out)
	return out
}

// RequiresAuth: thao tác cần bearerAuth (security riêng của thao tác, hoặc mặc định của tài liệu).
func (s *Spec) RequiresAuth(o Operation) bool {
	sec := o.Op.Security
	if sec == nil {
		sec = &s.Doc.Security
	}
	for _, req := range *sec {
		if _, ok := req["bearerAuth"]; ok {
			return true
		}
	}
	return false
}

// ValidateOpts tinh chỉnh ValidateResponse.
type ValidateOpts struct {
	// SkipBody bỏ kiểm thân (luồng SSE).
	SkipBody bool
}

// ValidateResponse kiểm một response thật với spec: thao tác có trong spec, status đã khai báo, header bắt buộc, thân khớp schema.
func (s *Spec) ValidateResponse(ctx context.Context, req *http.Request, status int, header http.Header, body []byte, opts ValidateOpts) error {
	route, pathParams, err := s.router.FindRoute(req)
	if err != nil {
		return fmt.Errorf("%s %s không có trong spec: %w", req.Method, req.URL.Path, err)
	}
	in := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: &openapi3filter.RequestValidationInput{Request: req, PathParams: pathParams, Route: route},
		Status:                 status,
		Header:                 header,
		Body:                   io.NopCloser(bytes.NewReader(body)),
		Options:                &openapi3filter.Options{IncludeResponseStatus: true, ExcludeResponseBody: opts.SkipBody},
	}
	if err := openapi3filter.ValidateResponse(ctx, in); err != nil {
		return fmt.Errorf("%s %s → %d: %w", route.Method, route.Path, status, err)
	}
	return nil
}

var paramRE = regexp.MustCompile(`\{[^}]+\}`)

// NormalizePath đổi mọi `{tên}` thành `{}` để so route trong mã với mẫu của spec.
func NormalizePath(p string) string { return paramRE.ReplaceAllString(p, "{}") }
