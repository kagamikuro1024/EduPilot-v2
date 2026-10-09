package exam

import (
	"archive/zip"
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/store"
)

// DefaultZipUncompressed là trần giải nén mặc định (EXAM_TESTZIP_MAX_UNCOMPRESSED).
const DefaultZipUncompressed = 50 << 20

// ZipReport là kết quả nhập zip (thật hoặc thử).
type ZipReport struct {
	WouldCreate int      `json:"would_create"`
	Created     int      `json:"created"`
	Errors      []ZipErr `json:"errors"`
}

// ZipErr là một lỗi theo từng tệp trong zip.
type ZipErr struct {
	File    string `json:"file"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func zipErrs(errs []apierr.FieldError) []ZipErr {
	out := make([]ZipErr, len(errs))
	for i, e := range errs {
		out[i] = ZipErr{File: e.Field, Code: e.Code, Message: e.Message}
	}
	return out
}

type zipTest struct {
	name     string
	in, out  string
	hasIn    bool
	hasOut   bool
	outFile  string
	weight   int
	isSample bool
}

func zerr(file, code, msg string) apierr.FieldError {
	return apierr.FieldError{Field: file, Code: code, Message: msg}
}

// naturalCmp so sánh tên theo số tự nhiên (`t2` < `t10`).
func naturalCmp(a, b string) int {
	for a != "" && b != "" {
		da, db := unicode.IsDigit(rune(a[0])), unicode.IsDigit(rune(b[0]))
		if da && db {
			i, j := 0, 0
			for i < len(a) && unicode.IsDigit(rune(a[i])) {
				i++
			}
			for j < len(b) && unicode.IsDigit(rune(b[j])) {
				j++
			}
			na, nb := strings.TrimLeft(a[:i], "0"), strings.TrimLeft(b[:j], "0")
			if c := cmp.Or(cmp.Compare(len(na), len(nb)), strings.Compare(na, nb)); c != 0 {
				return c
			}
			a, b = a[i:], b[j:]
			continue
		}
		if a[0] != b[0] {
			return cmp.Compare(a[0], b[0])
		}
		a, b = a[1:], b[1:]
	}
	return cmp.Compare(len(a), len(b))
}

func badPath(n string) bool {
	if n == "" || strings.HasPrefix(n, "/") || strings.Contains(n, "\\") {
		return true
	}
	for _, seg := range strings.Split(n, "/") {
		if seg == ".." {
			return true
		}
	}
	return strings.IndexFunc(n, unicode.IsControl) >= 0
}

type manifest struct {
	Tests []struct {
		Name     string `json:"name"`
		Weight   *int   `json:"weight"`
		IsSample *bool  `json:"is_sample"`
	} `json:"tests"`
}

// parseZip kiểm toàn bộ zip theo SRS 4.1.5 và trả danh sách test đã sắp theo tên tự nhiên cùng MỌI lỗi theo từng tệp.
// Giới hạn giải nén được ĐẾM khi đọc (không tin header); vượt → lỗi cả tệp `TEST_ZIP_INVALID` và dừng đọc.
func parseZip(data []byte, maxUncompressed int) ([]zipTest, []apierr.FieldError) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil && !errors.Is(err, zip.ErrInsecurePath) { // tên không an toàn: Go vẫn trả reader dùng được; ta báo lỗi theo từng tệp ở dưới

		return nil, []apierr.FieldError{zerr("file", "TEST_ZIP_INVALID", "Không đọc được tệp zip.")}
	}
	var errs []apierr.FieldError
	tests := map[string]*zipTest{}
	var man *manifest
	seen := map[string]bool{}
	total := 0
	for _, f := range zr.File {
		name := f.Name
		switch {
		case badPath(name):
			errs = append(errs, zerr(name, "ZIP_PATH", "Đường dẫn trong zip không hợp lệ (có .., tuyệt đối hoặc ký tự điều khiển)."))
			continue
		case f.Mode()&os.ModeSymlink != 0:
			errs = append(errs, zerr(name, "ZIP_SYMLINK", "Không nhận liên kết tượng trưng."))
			continue
		case f.FileInfo().IsDir() || strings.Contains(name, "/"):
			errs = append(errs, zerr(name, "ZIP_NESTED", "Các tệp test phải nằm ở gốc zip, không dùng thư mục lồng."))
			continue
		case seen[name]:
			errs = append(errs, zerr(name, "DUPLICATE_NAME", "Tên tệp bị trùng."))
			continue
		}
		seen[name] = true
		ext := path.Ext(name)
		base := strings.TrimSuffix(name, ext)
		if name != "manifest.json" && ext != ".in" && ext != ".out" && ext != ".ans" {
			errs = append(errs, zerr(name, "ZIP_UNEXPECTED_FILE", "Chỉ nhận cặp <tên>.in / <tên>.out (hoặc .ans) và manifest.json."))
			continue
		}
		rc, err := f.Open()
		if err != nil {
			errs = append(errs, zerr(name, "TEST_ZIP_INVALID", "Không đọc được tệp."))
			continue
		}
		body, err := io.ReadAll(io.LimitReader(rc, MaxTestBytes+1))
		_ = rc.Close()
		total += len(body)
		if total > maxUncompressed {
			return nil, append(errs, zerr("file", "TEST_ZIP_INVALID", "Dung lượng sau khi giải nén vượt giới hạn."))
		}
		switch {
		case err != nil:
			errs = append(errs, zerr(name, "TEST_ZIP_INVALID", "Tệp zip bị hỏng."))
			continue
		case len(body) > MaxTestBytes:
			errs = append(errs, zerr(name, "TEST_TOO_LARGE", "Mỗi tệp tối đa 1 MiB."))
			continue
		case !utf8.Valid(body):
			errs = append(errs, zerr(name, "TEST_NOT_UTF8", "Nội dung phải là văn bản UTF-8."))
			continue
		}
		if name == "manifest.json" {
			var m manifest
			dec := json.NewDecoder(bytes.NewReader(body))
			dec.DisallowUnknownFields()
			if err := dec.Decode(&m); err != nil {
				errs = append(errs, zerr(name, "MANIFEST_INVALID", "manifest.json sai cú pháp hoặc có khoá lạ."))
			} else {
				man = &m
			}
			continue
		}
		if !testNameRE.MatchString(base) {
			errs = append(errs, zerr(name, "TEST_NAME", "Tên test dài 1 đến 60 ký tự gồm chữ, số, gạch ngang hoặc gạch dưới."))
			continue
		}
		t := tests[base]
		if t == nil {
			t = &zipTest{name: base, weight: 1, isSample: strings.HasPrefix(base, "sample")}
			tests[base] = t
		}
		if ext == ".in" {
			t.in, t.hasIn = string(body), true
			continue
		}
		if t.hasOut {
			errs = append(errs, zerr(name, "DUPLICATE_NAME", "Test có cả .out và .ans."))
			continue
		}
		t.out, t.hasOut, t.outFile = string(body), true, name
	}
	out := make([]*zipTest, 0, len(tests))
	for _, t := range tests {
		switch {
		case !t.hasIn:
			errs = append(errs, zerr(t.outFile, "ZIP_MISSING_PAIR", "Thiếu tệp .in tương ứng."))
		case !t.hasOut:
			errs = append(errs, zerr(t.name+".in", "ZIP_MISSING_PAIR", "Thiếu tệp .out (hoặc .ans) tương ứng."))
		default:
			out = append(out, t)
		}
	}
	slices.SortFunc(out, func(a, b *zipTest) int { return naturalCmp(a.name, b.name) })
	if man != nil {
		byName := map[string]*zipTest{}
		for _, t := range out {
			byName[t.name] = t
		}
		for _, m := range man.Tests {
			t := byName[m.Name]
			switch {
			case t == nil:
				errs = append(errs, zerr("manifest.json", "MANIFEST_UNKNOWN_TEST", "manifest.json nhắc tới test không có trong zip: "+m.Name))
			case m.Weight != nil && !inRange(*m.Weight, 0, 1000):
				errs = append(errs, zerr("manifest.json", "LIMIT_OUT_OF_RANGE", "Trọng số của "+m.Name+" phải từ 0 đến 1.000."))
			default:
				if m.Weight != nil {
					t.weight = *m.Weight
				}
				if m.IsSample != nil {
					t.isSample = *m.IsSample
				}
			}
		}
	}
	res := make([]zipTest, len(out))
	for i, t := range out {
		res[i] = *t
	}
	return res, errs
}

// ImportZip nhập test từ zip (all-or-nothing). `dryRun` chỉ kiểm và báo; `replace` xoá hết test cũ trước khi thêm (mặc định thêm vào cuối).
// Thành công tăng `tests_version` MỘT lần. Lỗi theo từng tệp → 422 (không ghi gì); dry-run luôn trả báo cáo 200.
func (s *Service) ImportZip(ctx context.Context, actor, courseID, id uuid.UUID, data []byte, dryRun, replace, isTeacher bool) (ZipReport, error) {
	maxU := s.ZipMaxUncompressed
	if maxU <= 0 {
		maxU = DefaultZipUncompressed
	}
	tests, errs := parseZip(data, maxU)
	rep := ZipReport{Errors: []ZipErr{}}
	existing := 0
	if !replace || dryRun {
		st, err := store.New(s.Pool).TestcaseStats(ctx, store.TestcaseStatsParams{CourseID: courseID, ProblemID: id})
		if err != nil {
			return rep, fmt.Errorf("exam: đếm test: %w", err)
		}
		existing = int(st.TotalAll)
	}
	if len(tests) == 0 && len(errs) == 0 {
		errs = append(errs, zerr("file", "TEST_ZIP_INVALID", "Tệp zip không có test nào."))
	}
	if !replace {
		if existing+len(tests) > MaxTests {
			errs = append(errs, zerr("file", "TOO_MANY_TESTS", "Mỗi bài tối đa 100 test (kể cả test đã có)."))
		}
	} else if len(tests) > MaxTests {
		errs = append(errs, zerr("file", "TOO_MANY_TESTS", "Mỗi bài tối đa 100 test."))
	}
	rep.Errors = zipErrs(errs)
	if len(errs) > 0 {
		if dryRun {
			return rep, nil
		}
		return rep, apierr.Validation(errs...)
	}
	rep.WouldCreate = len(tests)
	if dryRun {
		return rep, nil
	}
	var made, drop []string
	err := s.mutateTests(ctx, actor, courseID, id, isTeacher, func(q *store.Queries, _ store.CodeProblem) (int, error) {
		start := 0
		if replace {
			old, err := q.TestcasesDeleteAll(ctx, store.TestcasesDeleteAllParams{CourseID: courseID, ProblemID: id})
			if err != nil {
				return 0, fmt.Errorf("exam: xoá test cũ: %w", err)
			}
			for _, k := range old {
				for _, p := range []*string{k.InputBlobKey, k.ExpectedBlobKey} {
					if p != nil {
						drop = append(drop, *p)
					}
				}
			}
		} else {
			st, err := q.TestcaseStats(ctx, store.TestcaseStatsParams{CourseID: courseID, ProblemID: id})
			if err != nil {
				return 0, fmt.Errorf("exam: đếm test: %w", err)
			}
			start = int(st.MaxPosition)
			if int(st.TotalAll)+len(tests) > MaxTests {
				return 0, apierr.Validation(zerr("file", "TOO_MANY_TESTS", "Mỗi bài tối đa 100 test (kể cả test đã có)."))
			}
		}
		for i, t := range tests {
			inp, err := s.putText(ctx, courseID, id, t.in, "in", &made)
			if err != nil {
				return 0, err
			}
			exp, err := s.putText(ctx, courseID, id, t.out, "out", &made)
			if err != nil {
				return 0, err
			}
			if _, err := q.TestcaseInsert(ctx, store.TestcaseInsertParams{CourseID: courseID, ProblemID: id, Position: int32(start + i + 1), Name: t.name, IsSample: t.isSample, Weight: int16(t.weight), //nolint:gosec // ≤ 100 / ≤ 1000
				Input: inp.inline, InputBlobKey: inp.key, Expected: exp.inline, ExpectedBlobKey: exp.key, InputBytes: int32(inp.n), ExpectedBytes: int32(exp.n), Source: "IMPORT", Approved: true}); err != nil { //nolint:gosec // ≤ 1 MiB
				return 0, fmt.Errorf("exam: ghi test: %w", err)
			}
		}
		rep.Created = len(tests)
		return len(tests), nil
	})
	if err != nil {
		s.dropBlobs(ctx, made)
		rep.Created = 0
		return rep, err
	}
	s.dropBlobs(ctx, drop)
	return rep, nil
}

// ZipTooLarge dựng lỗi 413 cho handler (tệp tải lên vượt EXAM_TESTZIP_MAX_BYTES).
func ZipTooLarge(limit int) *apierr.Error {
	return apierr.New(http.StatusRequestEntityTooLarge, apierr.PayloadTooLarge).WithDetails(map[string]any{"max_bytes": limit})
}
