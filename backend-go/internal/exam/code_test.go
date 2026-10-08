package exam_test

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/exam"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
)

// fieldCodes lấy details[].code của lỗi 422.
func fieldCodes(t *testing.T, err error) []string {
	t.Helper()
	var ae *apierr.Error
	require.True(t, errors.As(err, &ae), "cần *apierr.Error, có %v", err)
	var out []string
	for _, f := range ae.Details.([]apierr.FieldError) {
		out = append(out, f.Code)
	}
	return out
}

func apiStatus(t *testing.T, err error) (int, string) {
	t.Helper()
	var ae *apierr.Error
	require.True(t, errors.As(err, &ae), "cần *apierr.Error, có %v", err)
	return ae.Status, ae.Code
}

type problem struct {
	TestsVersion int
	Verified     *int
	Limits       [3]int
	Checker      string
	Languages    []string
}

func (r *rig) problem(qid uuid.UUID) problem {
	r.t.Helper()
	var p problem
	var v *int32
	var tv, tl, ml, ol int32
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `select tests_version, reference_verified_version, time_limit_ms, memory_limit_mb, output_limit_kb, checker::text, languages from code_problems where question_id=$1`, qid).
		Scan(&tv, &v, &tl, &ml, &ol, &p.Checker, &p.Languages))
	p.TestsVersion, p.Limits = int(tv), [3]int{int(tl), int(ml), int(ol)}
	if v != nil {
		x := int(*v)
		p.Verified = &x
	}
	return p
}

func (r *rig) verify(qid uuid.UUID) {
	r.exec(`update code_problems set reference_verified_version = tests_version, reference_verified_at = now() where question_id=$1`, qid)
}

func (r *rig) version(qid uuid.UUID) int {
	r.t.Helper()
	d, err := r.svc.Get(r.t.Context(), r.course, qid)
	require.NoError(r.t, err)
	return d.Version
}

// TestCodeProblemDefaults — AC3: câu CODE mới có cấu hình mặc định 1000 ms / 256 MiB / 1024 KiB / EXACT, `cpp17`, tests_version 1, chưa có test.
func TestCodeProblemDefaults(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	d, err := r.svc.Create(t.Context(), r.teacher, r.course, exam.QuestionIn{Type: "CODE", Title: "c", Topic: "t", Stem: "s"})
	require.NoError(t, err)
	require.NotNil(t, d.Code)
	require.Equal(t, []string{"cpp17"}, d.Code.Languages)
	require.Equal(t, [3]int{1000, 256, 1024}, [3]int{d.Code.TimeLimitMS, d.Code.MemoryLimitMB, d.Code.OutputLimitKB})
	require.Equal(t, "EXACT", d.Code.Checker)
	require.Equal(t, 1, d.Code.TestsVersion)
	require.Nil(t, d.Code.ReferenceVerifiedVersion)
	require.Equal(t, exam.TestStats{}, d.Code.Tests)
	require.Nil(t, d.AnswerKey)
	// PUT chỉ có ngôn ngữ → mặc định điền đủ.
	d, err = r.svc.PutCode(t.Context(), r.course, d.ID, exam.CodeIn{Languages: []string{"c11", "cpp17"}}, d.Version)
	require.NoError(t, err)
	require.Equal(t, [3]int{1000, 256, 1024}, [3]int{d.Code.TimeLimitMS, d.Code.MemoryLimitMB, d.Code.OutputLimitKB})
}

// TestCodeProblemValidation — AC3: các giới hạn 100…10.000 ms, 16…512 MiB, 1…16.384 KiB, FLOAT_EPS, ngôn ngữ, starter ≤ 16 KiB, lời giải mẫu ≤ 64 KiB.
func TestCodeProblemValidation(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	q := r.code("v")
	eps := func(s string) *decimal.Decimal { d := decimal.RequireFromString(s); return &d }
	base := func(f func(*exam.CodeIn)) exam.CodeIn {
		in := exam.CodeIn{Languages: []string{"cpp17"}}
		f(&in)
		return in
	}
	bad := []struct {
		name string
		in   exam.CodeIn
		want string
	}{
		{"không ngôn ngữ", base(func(c *exam.CodeIn) { c.Languages = nil }), "LANGUAGE_NOT_ALLOWED"},
		{"ngôn ngữ lạ", base(func(c *exam.CodeIn) { c.Languages = []string{"java"} }), "LANGUAGE_NOT_ALLOWED"},
		{"thời gian 99", base(func(c *exam.CodeIn) { c.TimeLimitMS = new(99) }), "LIMIT_OUT_OF_RANGE"},
		{"thời gian 10001", base(func(c *exam.CodeIn) { c.TimeLimitMS = new(10001) }), "LIMIT_OUT_OF_RANGE"},
		{"bộ nhớ 15", base(func(c *exam.CodeIn) { c.MemoryLimitMB = new(15) }), "LIMIT_OUT_OF_RANGE"},
		{"bộ nhớ 513 (trần 512)", base(func(c *exam.CodeIn) { c.MemoryLimitMB = new(513) }), "LIMIT_OUT_OF_RANGE"},
		{"đầu ra 0", base(func(c *exam.CodeIn) { c.OutputLimitKB = new(0) }), "LIMIT_OUT_OF_RANGE"},
		{"đầu ra 16385", base(func(c *exam.CodeIn) { c.OutputLimitKB = new(16385) }), "LIMIT_OUT_OF_RANGE"},
		{"FLOAT_EPS thiếu eps", base(func(c *exam.CodeIn) { c.Checker = "FLOAT_EPS" }), "FLOAT_EPS_REQUIRED"},
		{"FLOAT_EPS eps 0", base(func(c *exam.CodeIn) { c.Checker = "FLOAT_EPS"; c.FloatEps = eps("0") }), "LIMIT_OUT_OF_RANGE"},
		{"FLOAT_EPS eps 0.2", base(func(c *exam.CodeIn) { c.Checker = "FLOAT_EPS"; c.FloatEps = eps("0.2") }), "LIMIT_OUT_OF_RANGE"},
		{"checker lạ", base(func(c *exam.CodeIn) { c.Checker = "REGEX" }), "INVALID_CHECKER"},
		{"starter cho ngôn ngữ ngoài danh sách", base(func(c *exam.CodeIn) { c.StarterCode = map[string]string{"c11": "x"} }), "LANGUAGE_NOT_ALLOWED"},
		{"starter 16 KiB + 1", base(func(c *exam.CodeIn) { c.StarterCode = map[string]string{"cpp17": strings.Repeat("a", 16<<10+1)} }), "SOURCE_TOO_LARGE"},
		{"lời giải mẫu 64 KiB + 1", base(func(c *exam.CodeIn) {
			c.Reference = &exam.Reference{Language: "cpp17", Source: strings.Repeat("a", 64<<10+1)}
		}), "SOURCE_TOO_LARGE"},
		{"lời giải mẫu rỗng", base(func(c *exam.CodeIn) { c.Reference = &exam.Reference{Language: "cpp17", Source: " "} }), "SOURCE_EMPTY"},
		{"lời giải mẫu ngôn ngữ ngoài danh sách", base(func(c *exam.CodeIn) { c.Reference = &exam.Reference{Language: "c11", Source: "x"} }), "LANGUAGE_NOT_ALLOWED"},
	}
	for _, c := range bad {
		_, err := r.svc.PutCode(t.Context(), r.course, q.ID, c.in, q.Version)
		require.Contains(t, fieldCodes(t, err), c.want, c.name)
	}
	ok := base(func(c *exam.CodeIn) {
		c.TimeLimitMS, c.MemoryLimitMB, c.OutputLimitKB = new(100), new(512), new(16384)
		c.Checker, c.FloatEps = "FLOAT_EPS", eps("0.1")
		c.StarterCode = map[string]string{"cpp17": strings.Repeat("a", 16<<10)}
	})
	d, err := r.svc.PutCode(t.Context(), r.course, q.ID, ok, q.Version)
	require.NoError(t, err)
	require.Equal(t, "0.1", *d.Code.FloatEps)
}

// TestCodeProblemBumpsTestsVersion — AC3: đổi giới hạn / checker / ngôn ngữ tăng tests_version MỘT lần và xoá cờ lời giải mẫu đã kiểm; lưu lại y nguyên thì không đổi gì.
func TestCodeProblemBumpsTestsVersion(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	q := r.code("b")
	r.test(q.ID, "t1", "1", "1", true, 1)
	r.verify(q.ID)
	p0 := r.problem(q.ID)
	require.NotNil(t, p0.Verified)
	cur := func() exam.QuestionDetail {
		d, err := r.svc.Get(t.Context(), r.course, q.ID)
		require.NoError(t, err)
		return d
	}
	d := cur()
	same := exam.CodeIn{Languages: []string{"cpp17"}, Reference: &exam.Reference{Language: "cpp17", Source: "int main(){}"}}
	d, err := r.svc.PutCode(t.Context(), r.course, q.ID, same, d.Version)
	require.NoError(t, err)
	require.Equal(t, p0.TestsVersion, r.problem(q.ID).TestsVersion, "không đổi gì → không bump")
	steps := []struct {
		name string
		mut  func(*exam.CodeIn)
	}{
		{"đổi thời gian", func(c *exam.CodeIn) { c.TimeLimitMS = new(2000) }},
		{"đổi bộ nhớ", func(c *exam.CodeIn) { c.TimeLimitMS, c.MemoryLimitMB = new(2000), new(128) }},
		{"đổi ngôn ngữ", func(c *exam.CodeIn) {
			c.TimeLimitMS, c.MemoryLimitMB = new(2000), new(128)
			c.Languages = []string{"c11", "cpp17"}
		}},
		{"đổi checker", func(c *exam.CodeIn) {
			c.TimeLimitMS, c.MemoryLimitMB = new(2000), new(128)
			c.Languages, c.Checker = []string{"c11", "cpp17"}, "TOKENS"
		}},
	}
	want := p0.TestsVersion
	for _, s := range steps {
		r.verify(q.ID)
		in := same
		s.mut(&in)
		d, err = r.svc.PutCode(t.Context(), r.course, q.ID, in, d.Version)
		require.NoError(t, err, s.name)
		want++
		p := r.problem(q.ID)
		require.Equal(t, want, p.TestsVersion, s.name)
		require.Nil(t, p.Verified, s.name)
	}
	// đổi chỉ lời giải mẫu: không bump nhưng gỡ cờ.
	r.verify(q.ID)
	in := exam.CodeIn{Languages: []string{"c11", "cpp17"}, TimeLimitMS: new(2000), MemoryLimitMB: new(128), Checker: "TOKENS", Reference: &exam.Reference{Language: "cpp17", Source: "int main(){return 0;}"}}
	_, err = r.svc.PutCode(t.Context(), r.course, q.ID, in, d.Version)
	require.NoError(t, err)
	p := r.problem(q.ID)
	require.Equal(t, want, p.TestsVersion)
	require.Nil(t, p.Verified)
}

// TestStarterCodeChangeKeepsTestsVersion — AC3 / góp ý #2 (a): chỉ đổi mã khởi tạo thì tests_version và cờ lời giải mẫu giữ nguyên.
func TestStarterCodeChangeKeepsTestsVersion(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	q := r.code("s")
	r.test(q.ID, "t1", "1", "1", true, 1)
	r.verify(q.ID)
	p0 := r.problem(q.ID)
	d, err := r.svc.PutCode(t.Context(), r.course, q.ID, exam.CodeIn{Languages: []string{"cpp17"}, Reference: &exam.Reference{Language: "cpp17", Source: "int main(){}"}, StarterCode: map[string]string{"cpp17": "#include <cstdio>\n"}}, r.version(q.ID))
	require.NoError(t, err)
	p := r.problem(q.ID)
	require.Equal(t, p0.TestsVersion, p.TestsVersion)
	require.Equal(t, p0.Verified, p.Verified)
	require.JSONEq(t, `{"cpp17":"#include <cstdio>\n"}`, string(d.Code.StarterCode))
}

// TestTestcaseCRUD — AC4: thêm / sửa / đổi vị trí / xoá, `position` tự gán cuối rồi đánh số lại liên tục, GET trả cả test ẩn và test chưa duyệt, phân trang theo position.
func TestTestcaseCRUD(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	q := r.code("crud")
	a := r.test(q.ID, "a", "1 1", "2", true, 1)
	b := r.test(q.ID, "b", "2 2", "4", false, 2)
	c := r.test(q.ID, "c", "3 3", "6", false, 0)
	require.Equal(t, []int{1, 2, 3}, []int{a.Position, b.Position, c.Position})
	require.True(t, a.Approved)
	require.Equal(t, "MANUAL", a.Source)
	// chèn c lên đầu
	got, err := r.svc.UpdateTest(t.Context(), r.teacher, r.course, q.ID, c.ID, exam.TestcaseIn{Position: new(1), Weight: new(7)}, true)
	require.NoError(t, err)
	require.Equal(t, 1, got.Position)
	require.Equal(t, 7, got.Weight)
	r.exec(`update code_testcases set approved=false where id=$1`, b.ID) // test AI chưa duyệt vẫn hiện cho Staff
	list, err := r.svc.ListTests(t.Context(), r.course, q.ID, nil, 100)
	require.NoError(t, err)
	require.Equal(t, []string{"c", "a", "b"}, []string{list[0].Name, list[1].Name, list[2].Name})
	require.False(t, list[2].Approved)
	// phân trang theo vị trí
	page, err := r.svc.ListTests(t.Context(), r.course, q.ID, new(1), 2)
	require.NoError(t, err)
	require.Equal(t, []string{"a", "b"}, []string{page[0].Name, page[1].Name})
	require.NoError(t, r.svc.DeleteTest(t.Context(), r.teacher, r.course, q.ID, a.ID, true))
	list, err = r.svc.ListTests(t.Context(), r.course, q.ID, nil, 100)
	require.NoError(t, err)
	require.Len(t, list, 2)
	err = r.svc.DeleteTest(t.Context(), r.teacher, r.course, q.ID, uuid.New(), true)
	st, code := apiStatus(t, err)
	require.Equal(t, 404, st)
	require.Equal(t, "NOT_FOUND", code)
	// sinh viên / câu lớp khác: câu không CODE → 404
	m := r.mcq("mcq")
	_, err = r.svc.AddTest(t.Context(), r.teacher, r.course, m.ID, exam.TestcaseIn{Name: new("x"), Input: new("1"), Expected: new("1")}, true)
	st, _ = apiStatus(t, err)
	require.Equal(t, 404, st)
}

// TestTestcaseLimits — AC4: tên, trọng số 0…1.000, UTF-8, ≤ 1 MiB, tối đa 100 test.
func TestTestcaseLimits(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	q := r.code("lim")
	add := func(in exam.TestcaseIn) error {
		_, err := r.svc.AddTest(t.Context(), r.teacher, r.course, q.ID, in, true)
		return err
	}
	ok := func() exam.TestcaseIn { return exam.TestcaseIn{Name: new("ok"), Input: new("1"), Expected: new("1")} }
	for name, mut := range map[string]func(*exam.TestcaseIn){
		"tên có dấu cách":        func(i *exam.TestcaseIn) { i.Name = new("a b") },
		"tên 61 ký tự":           func(i *exam.TestcaseIn) { i.Name = new(strings.Repeat("a", 61)) },
		"trọng số 1001":          func(i *exam.TestcaseIn) { i.Weight = new(1001) },
		"trọng số âm":            func(i *exam.TestcaseIn) { i.Weight = new(-1) },
		"input không phải UTF-8": func(i *exam.TestcaseIn) { i.Input = new("\xff\xfe") },
		"expected > 1 MiB":       func(i *exam.TestcaseIn) { i.Expected = new(strings.Repeat("a", 1<<20+1)) },
	} {
		in := ok()
		mut(&in)
		require.Error(t, add(in), name)
	}
	w := ok()
	w.Weight = new(1000)
	require.NoError(t, add(w))
	z := ok()
	z.Weight = new(0)
	require.NoError(t, add(z))
	for i := 2; i < exam.MaxTests; i++ {
		require.NoError(t, add(ok()))
	}
	err := add(ok())
	require.Contains(t, fieldCodes(t, err), "TOO_MANY_TESTS")
}

// memBlob là kho đối tượng giả trong bộ nhớ.
type memBlob struct {
	mu sync.Mutex
	m  map[string]string
}

func (b *memBlob) Put(_ context.Context, key string, r io.Reader, _ int64, _ string) error {
	d, _ := io.ReadAll(r)
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.m == nil {
		b.m = map[string]string{}
	}
	b.m[key] = string(d)
	return nil
}

func (b *memBlob) Get(_ context.Context, key string) (io.ReadCloser, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	v, ok := b.m[key]
	if !ok {
		return nil, os.ErrNotExist
	}
	return io.NopCloser(strings.NewReader(v)), nil
}

func (b *memBlob) Delete(_ context.Context, key string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.m, key)
	return nil
}

func (b *memBlob) n() int { b.mu.Lock(); defer b.mu.Unlock(); return len(b.m) }

// TestTestcaseBlobSpill — AC4: ≤ 64 KiB lưu DB, lớn hơn lưu kho đối tượng (`*_blob_key`), đọc lại đủ nội dung; xoá test dọn blob; không có kho → 503.
func TestTestcaseBlobSpill(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	q := r.code("blob")
	small, big := strings.Repeat("s", 64<<10), strings.Repeat("b", 64<<10+1)
	_, err := r.svc.AddTest(t.Context(), r.teacher, r.course, q.ID, exam.TestcaseIn{Name: new("big"), Input: new(big), Expected: new("1")}, true)
	st, code := apiStatus(t, err)
	require.Equal(t, 503, st, "chưa có kho đối tượng")
	require.Equal(t, "SERVICE_UNAVAILABLE", code)
	blob := &memBlob{}
	r.svc.Blob = blob
	s1, err := r.svc.AddTest(t.Context(), r.teacher, r.course, q.ID, exam.TestcaseIn{Name: new("small"), Input: new(small), Expected: new("1")}, true)
	require.NoError(t, err)
	b1, err := r.svc.AddTest(t.Context(), r.teacher, r.course, q.ID, exam.TestcaseIn{Name: new("big"), Input: new(big), Expected: new(big)}, true)
	require.NoError(t, err)
	require.Equal(t, 2, blob.n())
	var inline *string
	var key *string
	var bytesN int
	require.NoError(t, r.pool.QueryRow(t.Context(), `select input, input_blob_key, input_bytes from code_testcases where id=$1`, s1.ID).Scan(&inline, &key, &bytesN))
	require.NotNil(t, inline)
	require.Nil(t, key)
	require.Equal(t, 64<<10, bytesN)
	require.NoError(t, r.pool.QueryRow(t.Context(), `select input, input_blob_key from code_testcases where id=$1`, b1.ID).Scan(&inline, &key))
	require.Nil(t, inline)
	require.NotNil(t, key)
	got, err := r.svc.ReadText(t.Context(), inline, key)
	require.NoError(t, err)
	require.Equal(t, big, got)
	listed, err := r.svc.ListTests(t.Context(), r.course, q.ID, nil, 100)
	require.NoError(t, err)
	require.True(t, listed[1].InputTruncated, "test lớn chỉ báo cắt, không trả 64 KiB+ trong danh sách")
	require.NoError(t, r.svc.DeleteTest(t.Context(), r.teacher, r.course, q.ID, b1.ID, true))
	require.Equal(t, 0, blob.n(), "xoá test dọn blob")
}

// TestTestcaseBumpsVersion — AC4: MỌI thay đổi tập test tăng tests_version đúng 1 và gỡ cờ lời giải mẫu đã kiểm; câu về DRAFT.
func TestTestcaseBumpsVersion(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	q := r.code("ver")
	tv := r.problem(q.ID).TestsVersion
	step := func(name string, f func()) {
		r.verify(q.ID)
		f()
		tv++
		p := r.problem(q.ID)
		require.Equal(t, tv, p.TestsVersion, name)
		require.Nil(t, p.Verified, name)
	}
	var a exam.Testcase
	step("thêm", func() { a = r.test(q.ID, "a", "1", "1", true, 1) })
	step("sửa", func() {
		_, err := r.svc.UpdateTest(t.Context(), r.teacher, r.course, q.ID, a.ID, exam.TestcaseIn{Weight: new(5)}, true)
		require.NoError(t, err)
	})
	step("duyệt", func() {
		_, err := r.svc.ApproveTests(t.Context(), r.teacher, r.course, q.ID, []uuid.UUID{a.ID}, true)
		require.NoError(t, err)
	})
	step("nhập zip", func() {
		_, err := r.svc.ImportZip(t.Context(), r.teacher, r.course, q.ID, mkZip(t, map[string]string{"x.in": "1", "x.out": "1"}), false, false, true)
		require.NoError(t, err)
	})
	step("xoá", func() { require.NoError(t, r.svc.DeleteTest(t.Context(), r.teacher, r.course, q.ID, a.ID, true)) })
}

// ---- zip ----

func mkZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var b bytes.Buffer
	w := zip.NewWriter(&b)
	for n, c := range files {
		f, err := w.Create(n)
		require.NoError(t, err)
		_, _ = f.Write([]byte(c))
	}
	require.NoError(t, w.Close())
	return b.Bytes()
}

func (r *rig) tests(qid uuid.UUID) []exam.Testcase {
	r.t.Helper()
	l, err := r.svc.ListTests(r.t.Context(), r.course, qid, nil, 100)
	require.NoError(r.t, err)
	return l
}

func zipFiles(t *testing.T, err error) []string {
	t.Helper()
	var ae *apierr.Error
	require.True(t, errors.As(err, &ae))
	var out []string
	for _, f := range ae.Details.([]apierr.FieldError) {
		out = append(out, f.Field+":"+f.Code)
	}
	return out
}

// TestTestZipImport — AC5: cặp .in/.out (và .ans), `sample*` → mẫu, sắp theo tên TỰ NHIÊN (t2 trước t10), dry_run không ghi, thêm cuối / thay thế, tests_version +1 một lần.
func TestTestZipImport(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	q := r.code("zip")
	z := mkZip(t, map[string]string{"t10.in": "10", "t10.out": "10", "t2.in": "2", "t2.ans": "2", "sample1.in": "1", "sample1.out": "1"})
	tv := r.problem(q.ID).TestsVersion
	rep, err := r.svc.ImportZip(t.Context(), r.teacher, r.course, q.ID, z, true, false, true)
	require.NoError(t, err)
	require.Equal(t, 3, rep.WouldCreate)
	require.Empty(t, rep.Errors)
	require.Empty(t, r.tests(q.ID), "dry_run không ghi gì")
	require.Equal(t, tv, r.problem(q.ID).TestsVersion)
	rep, err = r.svc.ImportZip(t.Context(), r.teacher, r.course, q.ID, z, false, false, true)
	require.NoError(t, err)
	require.Equal(t, 3, rep.Created)
	got := r.tests(q.ID)
	require.Equal(t, []string{"sample1", "t2", "t10"}, []string{got[0].Name, got[1].Name, got[2].Name})
	require.Equal(t, []bool{true, false, false}, []bool{got[0].IsSample, got[1].IsSample, got[2].IsSample})
	require.Equal(t, "IMPORT", got[0].Source)
	require.Equal(t, tv+1, r.problem(q.ID).TestsVersion, "tăng MỘT lần cho cả lô")
	// thêm vào cuối
	_, err = r.svc.ImportZip(t.Context(), r.teacher, r.course, q.ID, mkZip(t, map[string]string{"z.in": "z", "z.out": "z"}), false, false, true)
	require.NoError(t, err)
	require.Len(t, r.tests(q.ID), 4)
	require.Equal(t, 4, r.tests(q.ID)[3].Position)
	// thay thế
	_, err = r.svc.ImportZip(t.Context(), r.teacher, r.course, q.ID, mkZip(t, map[string]string{"only.in": "o", "only.out": "o"}), false, true, true)
	require.NoError(t, err)
	got = r.tests(q.ID)
	require.Len(t, got, 1)
	require.Equal(t, "only", got[0].Name)
	require.Equal(t, 1, got[0].Position)
}

// TestTestZipSlip — AC5: `..`, đường dẫn tuyệt đối, ký tự điều khiển, thư mục lồng, liên kết tượng trưng, tệp lạ → lỗi theo từng tệp, không ghi gì.
func TestTestZipSlip(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	q := r.code("slip")
	var sym bytes.Buffer
	zw := zip.NewWriter(&sym)
	h := &zip.FileHeader{Name: "link.in"}
	h.SetMode(os.ModeSymlink | 0o777)
	f, err := zw.CreateHeader(h)
	require.NoError(t, err)
	_, _ = f.Write([]byte("/etc/passwd"))
	require.NoError(t, zw.Close())
	cases := map[string]struct {
		zip  []byte
		want string
	}{
		"..":             {mkZip(t, map[string]string{"../evil.in": "x", "../evil.out": "x"}), "ZIP_PATH"},
		"tuyệt đối":      {mkZip(t, map[string]string{"/abs.in": "x", "/abs.out": "x"}), "ZIP_PATH"},
		"gạch ngược":     {mkZip(t, map[string]string{`a\b.in`: "x"}), "ZIP_PATH"},
		"điều khiển":     {mkZip(t, map[string]string{"a\x07b.in": "x"}), "ZIP_PATH"},
		"thư mục lồng":   {mkZip(t, map[string]string{"d/t.in": "x", "d/t.out": "x"}), "ZIP_NESTED"},
		"liên kết":       {sym.Bytes(), "ZIP_SYMLINK"},
		"tệp lạ":         {mkZip(t, map[string]string{"t.in": "1", "t.out": "1", "run.sh": "rm -rf /"}), "ZIP_UNEXPECTED_FILE"},
		"không phải zip": {[]byte("PK nhưng không phải"), "TEST_ZIP_INVALID"},
	}
	for name, c := range cases {
		_, err := r.svc.ImportZip(t.Context(), r.teacher, r.course, q.ID, c.zip, false, false, true)
		var found bool
		for _, s := range zipFiles(t, err) {
			found = found || strings.HasSuffix(s, ":"+c.want)
		}
		require.True(t, found, "%s: %v", name, zipFiles(t, err))
		require.Empty(t, r.tests(q.ID), name)
	}
}

// TestTestZipBomb — AC5: giới hạn giải nén ĐẾM KHI ĐỌC (zip sinh lúc chạy, không commit tệp lớn); tệp > 1 MiB bị từ chối riêng.
func TestTestZipBomb(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	q := r.code("bomb")
	r.svc.ZipMaxUncompressed = 3 << 20
	files := map[string]string{}
	for i := 0; i < 6; i++ { // 6 × 1 MiB số 0 nén còn vài KB nhưng giải nén 6 MiB > 3 MiB
		z := strings.Repeat("0", 1<<20)
		files[fmt.Sprintf("t%d.in", i)] = z
		files[fmt.Sprintf("t%d.out", i)] = "0"
	}
	data := mkZip(t, files)
	require.Less(t, len(data), 100<<10, "zip nén nhỏ (đúng kiểu bom)")
	_, err := r.svc.ImportZip(t.Context(), r.teacher, r.course, q.ID, data, false, false, true)
	require.Contains(t, fieldCodes(t, err), "TEST_ZIP_INVALID")
	require.Empty(t, r.tests(q.ID))
	r.svc.ZipMaxUncompressed = 0
	_, err = r.svc.ImportZip(t.Context(), r.teacher, r.course, q.ID, mkZip(t, map[string]string{"big.in": strings.Repeat("a", 1<<20+1), "big.out": "1"}), false, false, true)
	require.Contains(t, fieldCodes(t, err), "TEST_TOO_LARGE")
}

// TestTestZipAtomic — AC5: một tệp hỏng (không UTF-8, thiếu nửa cặp, tên sai) thì KHÔNG ghi gì; lỗi liệt kê đủ từng tệp; quá 100 test bị từ chối.
func TestTestZipAtomic(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	q := r.code("atomic")
	tv := r.problem(q.ID).TestsVersion
	data := mkZip(t, map[string]string{"a.in": "1", "a.out": "1", "b.in": "\xff\xfe", "b.out": "2", "c.in": "3", "bad name.in": "1", "bad name.out": "1"})
	rep, err := r.svc.ImportZip(t.Context(), r.teacher, r.course, q.ID, data, false, false, true)
	files := zipFiles(t, err)
	require.Contains(t, files, "b.in:TEST_NOT_UTF8")
	require.Contains(t, files, "c.in:ZIP_MISSING_PAIR")
	require.Contains(t, files, "bad name.in:TEST_NAME")
	require.Zero(t, rep.Created)
	require.Empty(t, r.tests(q.ID))
	require.Equal(t, tv, r.problem(q.ID).TestsVersion)
	// dry_run trả cùng báo cáo với 200 (không lỗi HTTP)
	rep, err = r.svc.ImportZip(t.Context(), r.teacher, r.course, q.ID, data, true, false, true)
	require.NoError(t, err)
	require.NotEmpty(t, rep.Errors)
	// 101 test
	many := map[string]string{}
	for i := 0; i <= exam.MaxTests; i++ {
		many[fmt.Sprintf("t%d.in", i)], many[fmt.Sprintf("t%d.out", i)] = "1", "1"
	}
	_, err = r.svc.ImportZip(t.Context(), r.teacher, r.course, q.ID, mkZip(t, many), false, false, true)
	require.Contains(t, zipFiles(t, err), "file:TOO_MANY_TESTS")
}

// TestTestZipManifest — AC5: manifest.json ghi đè trọng số / cờ mẫu; khoá lạ, tên không có trong zip, trọng số ngoài khoảng bị từ chối.
func TestTestZipManifest(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	q := r.code("man")
	pair := map[string]string{"a.in": "1", "a.out": "1", "sample2.in": "2", "sample2.out": "2"}
	with := func(m string) map[string]string {
		out := map[string]string{"manifest.json": m}
		for k, v := range pair {
			out[k] = v
		}
		return out
	}
	_, err := r.svc.ImportZip(t.Context(), r.teacher, r.course, q.ID, mkZip(t, with(`{"tests":[{"name":"a","weight":9,"is_sample":true},{"name":"sample2","is_sample":false}]}`)), false, false, true)
	require.NoError(t, err)
	got := map[string]exam.Testcase{}
	for _, tc := range r.tests(q.ID) {
		got[tc.Name] = tc
	}
	require.Equal(t, 9, got["a"].Weight)
	require.True(t, got["a"].IsSample)
	require.False(t, got["sample2"].IsSample)
	for name, m := range map[string]string{
		"khoá lạ":         `{"tests":[],"extra":1}`,
		"tên không có":    `{"tests":[{"name":"zzz"}]}`,
		"trọng số 1001":   `{"tests":[{"name":"a","weight":1001}]}`,
		"không phải JSON": `not json`,
	} {
		_, err := r.svc.ImportZip(t.Context(), r.teacher, r.course, q.ID, mkZip(t, with(m)), false, true, true)
		require.Error(t, err, name)
		require.Len(t, r.tests(q.ID), 2, name+": replace không được xoá khi lỗi")
	}
}

// ---- khoá sửa (AC8) ----

// TestQuestionInUseLocked — AC8: câu đang được bài thi SCHEDULED dùng: sửa đề / đáp án / code / test → 409 QUESTION_IN_USE kèm tên bài; chủ đề / độ khó / giải thích vẫn sửa được.
func TestQuestionInUseLocked(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	q := r.mcq("khoá")
	r.useIn(q.ID, "SCHEDULED")
	d, err := r.svc.Get(t.Context(), r.course, q.ID)
	require.NoError(t, err)
	in := exam.QuestionIn{Type: "MCQ_SINGLE", Title: d.Title, Topic: "Chủ đề mới", Difficulty: "HARD", Stem: d.Stem, Explanation: new("giải thích mới"), Options: opts("3", "4"), Correct: []int{1}}
	upd, err := r.svc.Update(t.Context(), r.course, q.ID, in, d.Version)
	require.NoError(t, err, "chỉ đổi chủ đề / độ khó / giải thích")
	require.Equal(t, "HARD", upd.Difficulty)
	in.Stem = "Đề khác"
	_, err = r.svc.Update(t.Context(), r.course, q.ID, in, upd.Version)
	st, code := apiStatus(t, err)
	require.Equal(t, 409, st)
	require.Equal(t, "QUESTION_IN_USE", code)
	var ae *apierr.Error
	require.True(t, errors.As(err, &ae))
	exams := ae.Details.(map[string]any)["exams"].([]map[string]any)
	require.Len(t, exams, 1)
	require.Equal(t, "Bài thi SCHEDULED", exams[0]["title"])
	// đổi đáp án cũng bị khoá
	in.Stem = d.Stem
	in.Options = opts("3", "4", "5")
	_, err = r.svc.Update(t.Context(), r.course, q.ID, in, upd.Version)
	_, code = apiStatus(t, err)
	require.Equal(t, "QUESTION_IN_USE", code)
	// bài code đang dùng: cấu hình và test bị khoá
	c := r.code("code khoá")
	r.useIn(c.ID, "OPEN")
	_, err = r.svc.PutCode(t.Context(), r.course, c.ID, exam.CodeIn{Languages: []string{"c11"}}, r.version(c.ID))
	_, code = apiStatus(t, err)
	require.Equal(t, "QUESTION_IN_USE", code)
	_, err = r.svc.AddTest(t.Context(), r.teacher, r.course, c.ID, exam.TestcaseIn{Name: new("x"), Input: new("1"), Expected: new("1")}, true)
	_, code = apiStatus(t, err)
	require.Equal(t, "QUESTION_IN_USE", code)
	// bài thi còn DRAFT không khoá
	free := r.mcq("tự do")
	r.useIn(free.ID, "DRAFT")
	d2, _ := r.svc.Get(t.Context(), r.course, free.ID)
	in.Stem, in.Options = "Đề tự do", opts("3", "4")
	_, err = r.svc.Update(t.Context(), r.course, free.ID, in, d2.Version)
	require.NoError(t, err)
}

// TestTestsEditAfterCloseNeedsRegrade — AC8: khi MỌI bài thi dùng bài code đã CLOSED / PUBLISHED, Giảng viên sửa TEST được (tests_version tăng, bài cũ chấm lại ở US-PE-08); còn bài SCHEDULED / OPEN thì 409.
func TestTestsEditAfterCloseNeedsRegrade(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	q := r.code("sau đóng")
	tc := r.test(q.ID, "t1", "1", "1", true, 1)
	r.useIn(q.ID, "CLOSED")
	r.useIn(q.ID, "PUBLISHED")
	tv := r.problem(q.ID).TestsVersion
	_, err := r.svc.UpdateTest(t.Context(), r.teacher, r.course, q.ID, tc.ID, exam.TestcaseIn{Weight: new(4)}, true)
	require.NoError(t, err)
	require.Equal(t, tv+1, r.problem(q.ID).TestsVersion)
	// cấu hình (giới hạn) vẫn khoá kể cả với Giảng viên
	_, err = r.svc.PutCode(t.Context(), r.course, q.ID, exam.CodeIn{Languages: []string{"cpp17"}, TimeLimitMS: new(5000)}, r.version(q.ID))
	_, code := apiStatus(t, err)
	require.Equal(t, "QUESTION_IN_USE", code)
	// thêm một bài còn mở → sửa test bị khoá
	r.useIn(q.ID, "OPEN")
	_, err = r.svc.UpdateTest(t.Context(), r.teacher, r.course, q.ID, tc.ID, exam.TestcaseIn{Weight: new(5)}, true)
	_, code = apiStatus(t, err)
	require.Equal(t, "QUESTION_IN_USE", code)
}

// TestTAcannotEditUsedTests — AC8: TA không sửa test của bài đã dùng (đã đóng) → 403 reason=role; câu chưa dùng thì TA sửa được.
func TestTAcannotEditUsedTests(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	q := r.code("ta")
	tc := r.test(q.ID, "t1", "1", "1", true, 1)
	_, err := r.svc.UpdateTest(t.Context(), r.ta, r.course, q.ID, tc.ID, exam.TestcaseIn{Weight: new(2)}, false)
	require.NoError(t, err, "chưa dùng: TA sửa được")
	r.useIn(q.ID, "CLOSED")
	_, err = r.svc.UpdateTest(t.Context(), r.ta, r.course, q.ID, tc.ID, exam.TestcaseIn{Weight: new(3)}, false)
	st, code := apiStatus(t, err)
	require.Equal(t, 403, st)
	require.Equal(t, "FORBIDDEN", code)
	require.Error(t, r.svc.DeleteTest(t.Context(), r.ta, r.course, q.ID, tc.ID, false))
	_, err = r.svc.ImportZip(t.Context(), r.ta, r.course, q.ID, mkZip(t, map[string]string{"x.in": "1", "x.out": "1"}), false, false, false)
	st, _ = apiStatus(t, err)
	require.Equal(t, 403, st)
}

// TestArchiveKeepsHistory — AC8: câu đang dùng vẫn lưu trữ được; bài thi vẫn tham chiếu câu đó (lịch sử giữ nguyên); danh sách mặc định ẩn câu đã lưu trữ.
func TestArchiveKeepsHistory(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	q := r.mcq("lưu trữ")
	eid := r.useIn(q.ID, "PUBLISHED")
	d, err := r.svc.Archive(t.Context(), r.teacher, r.course, q.ID)
	require.NoError(t, err)
	require.NotNil(t, d.ArchivedAt)
	var n int
	require.NoError(t, r.pool.QueryRow(t.Context(), `select count(*) from exam_items where exam_id=$1 and question_id=$2`, eid, q.ID).Scan(&n))
	require.Equal(t, 1, n)
	rows, err := r.svc.List(t.Context(), r.course, exam.ListFilter{}, nil, 50)
	require.NoError(t, err)
	require.Empty(t, rows)
	rows, err = r.svc.List(t.Context(), r.course, exam.ListFilter{Archived: true}, nil, 50)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, 1, rows[0].UsedInExams)
}
