package exam

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/store"
)

// Giới hạn của bài code (SRS 4.5.3) và của test (4.1.5).
const (
	MaxTests          = 100
	MaxTestBytes      = 1 << 20  // 1 MiB mỗi input / expected
	MaxInlineBytes    = 64 << 10 // lớn hơn thì lưu ở kho đối tượng
	MaxStarterBytes   = 16 << 10
	MaxReferenceBytes = 64 << 10
	previewBytes      = 4 << 10
)

var testNameRE = regexp.MustCompile(`^[A-Za-z0-9_-]{1,60}$`)

// CodeIn là thân `PUT …/questions/{qid}/code` (thay toàn bộ cấu hình; trường vắng lấy mặc định).
type CodeIn struct {
	Languages     []string          `json:"languages"`
	TimeLimitMS   *int              `json:"time_limit_ms"`
	MemoryLimitMB *int              `json:"memory_limit_mb"`
	OutputLimitKB *int              `json:"output_limit_kb"`
	Checker       string            `json:"checker"`
	FloatEps      *decimal.Decimal  `json:"float_eps"`
	StarterCode   map[string]string `json:"starter_code"`
	Reference     *Reference        `json:"reference"`
	Version       *int              `json:"version"`
}

func orDefault(p *int, def int) int {
	if p == nil {
		return def
	}
	return *p
}

type codeChecked struct {
	Languages            []string
	TimeMS, MemMB, OutKB int
	Checker              store.CheckerKind
	Eps                  decimal.NullDecimal
	Starter              json.RawMessage
	RefLang, RefSrc      *string
}

func inRange(v, lo, hi int) bool { return v >= lo && v <= hi }

// validateCode áp SRS 4.5.3: mặc định 1000 ms / 256 MiB / 1024 KiB / EXACT; trần bộ nhớ 512 MiB.
func validateCode(in CodeIn) (codeChecked, []apierr.FieldError) {
	var errs []apierr.FieldError
	bad := func(field, code, msg string) {
		errs = append(errs, apierr.FieldError{Field: field, Code: code, Message: msg})
	}
	c := codeChecked{TimeMS: orDefault(in.TimeLimitMS, 1000), MemMB: orDefault(in.MemoryLimitMB, 256), OutKB: orDefault(in.OutputLimitKB, 1024)}
	if len(in.Languages) == 0 {
		bad("languages", "LANGUAGE_NOT_ALLOWED", "Chọn ít nhất một ngôn ngữ (C11 hoặc C++17).")
	}
	seen := map[string]bool{}
	for _, l := range in.Languages {
		if l != "c11" && l != "cpp17" {
			bad("languages", "LANGUAGE_NOT_ALLOWED", "Ngôn ngữ chỉ có thể là c11 hoặc cpp17.")
			continue
		}
		if !seen[l] {
			seen[l] = true
			c.Languages = append(c.Languages, l)
		}
	}
	if !inRange(c.TimeMS, 100, 10000) {
		bad("time_limit_ms", "LIMIT_OUT_OF_RANGE", "Giới hạn thời gian từ 100 đến 10.000 ms.")
	}
	if !inRange(c.MemMB, 16, 512) {
		bad("memory_limit_mb", "LIMIT_OUT_OF_RANGE", "Giới hạn bộ nhớ từ 16 đến 512 MiB.")
	}
	if !inRange(c.OutKB, 1, 16384) {
		bad("output_limit_kb", "LIMIT_OUT_OF_RANGE", "Giới hạn đầu ra từ 1 đến 16.384 KiB.")
	}
	switch in.Checker {
	case "", "EXACT":
		c.Checker = store.CheckerKindEXACT
	case "TOKENS":
		c.Checker = store.CheckerKindTOKENS
	case "FLOAT_EPS":
		c.Checker = store.CheckerKindFLOATEPS
		switch {
		case in.FloatEps == nil:
			bad("float_eps", "FLOAT_EPS_REQUIRED", "So khớp số thực cần sai số float_eps lớn hơn 0.")
		case !in.FloatEps.IsPositive() || in.FloatEps.GreaterThan(decimal.RequireFromString("0.1")):
			bad("float_eps", "LIMIT_OUT_OF_RANGE", "Sai số float_eps phải lớn hơn 0 và không quá 0,1.")
		default:
			c.Eps = decimal.NullDecimal{Decimal: *in.FloatEps, Valid: true}
		}
	default:
		bad("checker", "INVALID_CHECKER", "Cách so khớp chỉ có thể là EXACT, TOKENS hoặc FLOAT_EPS.")
	}
	starter := map[string]string{}
	for l, src := range in.StarterCode {
		switch {
		case !seen[l]:
			bad("starter_code."+l, "LANGUAGE_NOT_ALLOWED", "Mã khởi tạo chỉ cho ngôn ngữ nằm trong danh sách ngôn ngữ của bài.")
		case len(src) > MaxStarterBytes:
			bad("starter_code."+l, "SOURCE_TOO_LARGE", "Mã khởi tạo tối đa 16 KiB.")
		default:
			starter[l] = src
		}
	}
	c.Starter, _ = json.Marshal(starter)
	if r := in.Reference; r != nil {
		switch {
		case !seen[r.Language]:
			bad("reference.language", "LANGUAGE_NOT_ALLOWED", "Lời giải mẫu phải dùng ngôn ngữ nằm trong danh sách ngôn ngữ của bài.")
		case strings.TrimSpace(r.Source) == "":
			bad("reference.source", "SOURCE_EMPTY", "Lời giải mẫu không được để trống.")
		case len(r.Source) > MaxReferenceBytes:
			bad("reference.source", "SOURCE_TOO_LARGE", "Lời giải mẫu tối đa 64 KiB.")
		default:
			c.RefLang, c.RefSrc = &r.Language, &r.Source
		}
	}
	return c, errs
}

// guardEdit áp khoá sửa (SRS 4.1.4, AC8) cho thao tác lên bài code / test: bài thi SCHEDULED | OPEN dùng → 409; chỉ còn bài CLOSED | PUBLISHED →
// chỉ Giảng viên sửa TEST được (`testsOnly`); mọi trường hợp khác → 409 QUESTION_IN_USE.
func guardEdit(use []store.QuestionUsageRow, testsOnly, isTeacher bool) error {
	locked := inUse(use)
	if len(locked) == 0 {
		return nil
	}
	for _, u := range locked {
		if u.Status == store.ExamStatusSCHEDULED || u.Status == store.ExamStatusOPEN || !testsOnly {
			return inUseError(locked)
		}
	}
	if !isTeacher {
		return forbiddenRole()
	}
	return nil
}

// PutCode thay cấu hình bài code (khoá lạc quan theo `version` của câu). Đổi giới hạn / checker / ngôn ngữ tăng `tests_version`;
// đổi lời giải mẫu chỉ gỡ cờ "đã kiểm"; đổi `starter_code` không đổi gì ở phiên bản test (góp ý #2 (a)). Có đổi gì thì câu về DRAFT.
func (s *Service) PutCode(ctx context.Context, courseID, id uuid.UUID, in CodeIn, version int) (QuestionDetail, error) {
	c, errs := validateCode(in)
	var out QuestionDetail
	err := s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
		if err := writable(ctx, q, courseID); err != nil {
			return err
		}
		row, err := q.QuestionLock(ctx, store.QuestionLockParams{CourseID: courseID, ID: id})
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && row.Type != store.QuestionTypeCODE) {
			return notFound()
		}
		if err != nil {
			return fmt.Errorf("exam: khoá câu hỏi: %w", err)
		}
		if int(row.Version) != version {
			cur, derr := s.detail(ctx, q, row)
			if derr != nil {
				return derr
			}
			return &VersionConflict{Version: int(row.Version), Current: cur}
		}
		if len(errs) > 0 {
			return apierr.Validation(errs...)
		}
		cp, err := q.CodeProblemLock(ctx, store.CodeProblemLockParams{CourseID: courseID, QuestionID: id})
		if err != nil {
			return fmt.Errorf("exam: khoá bài code: %w", err)
		}
		bump := !slices.Equal(cp.Languages, c.Languages) || int(cp.TimeLimitMs) != c.TimeMS || int(cp.MemoryLimitMb) != c.MemMB || int(cp.OutputLimitKb) != c.OutKB ||
			cp.Checker != c.Checker || cp.FloatEps.Valid != c.Eps.Valid || (c.Eps.Valid && !cp.FloatEps.Decimal.Equal(c.Eps.Decimal))
		refChanged := !equalPtr(cp.ReferenceLanguage, c.RefLang) || !equalPtr(cp.ReferenceSource, c.RefSrc)
		starterChanged := !jsonEqual(cp.StarterCode, c.Starter)
		if !bump && !refChanged && !starterChanged {
			out, err = s.detail(ctx, q, row)
			return err
		}
		use, err := q.QuestionUsage(ctx, store.QuestionUsageParams{CourseID: courseID, QuestionID: id})
		if err != nil {
			return fmt.Errorf("exam: đọc bài thi đang dùng: %w", err)
		}
		if err := guardEdit(use, false, false); err != nil {
			return err
		}
		if _, err := q.CodeProblemUpdate(ctx, store.CodeProblemUpdateParams{
			CourseID: courseID, QuestionID: id, Languages: c.Languages, TimeLimitMs: int32(c.TimeMS), MemoryLimitMb: int32(c.MemMB), OutputLimitKb: int32(c.OutKB), //nolint:gosec // đã kiểm khoảng
			Checker: c.Checker, FloatEps: c.Eps, StarterCode: c.Starter, ReferenceLanguage: c.RefLang, ReferenceSource: c.RefSrc, Bump: bump, ClearVerified: refChanged,
		}); err != nil {
			return fmt.Errorf("exam: sửa bài code: %w", err)
		}
		upd, err := q.QuestionBump(ctx, store.QuestionBumpParams{CourseID: courseID, ID: id})
		if err != nil {
			return fmt.Errorf("exam: tăng version câu: %w", err)
		}
		out, err = s.detail(ctx, q, upd)
		return err
	})
	return out, err
}

func equalPtr(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func jsonEqual(a, b json.RawMessage) bool {
	var x, y map[string]string
	_ = json.Unmarshal(a, &x)
	_ = json.Unmarshal(b, &y)
	if len(x) != len(y) {
		return false
	}
	for k, v := range x {
		if y[k] != v {
			return false
		}
	}
	return true
}

// ---- test ----

// TestcaseIn là thân tạo / sửa một test (khi sửa, trường vắng = giữ nguyên).
type TestcaseIn struct {
	Name     *string `json:"name"`
	Input    *string `json:"input"`
	Expected *string `json:"expected"`
	IsSample *bool   `json:"is_sample"`
	Weight   *int    `json:"weight"`
	Position *int    `json:"position"`
}

// Testcase là một test trong phản hồi cho Staff. Nội dung lớn (lưu ở kho đối tượng) chỉ trả bản xem trước.
type Testcase struct {
	ID                uuid.UUID `json:"id"`
	Position          int       `json:"position"`
	Name              string    `json:"name"`
	IsSample          bool      `json:"is_sample"`
	Weight            int       `json:"weight"`
	Input             string    `json:"input"`
	InputTruncated    bool      `json:"input_truncated"`
	Expected          string    `json:"expected"`
	ExpectedTruncated bool      `json:"expected_truncated"`
	InputBytes        int       `json:"input_bytes"`
	ExpectedBytes     int       `json:"expected_bytes"`
	Source            string    `json:"source"`
	Approved          bool      `json:"approved"`
}

type stored struct {
	inline *string
	key    *string
	n      int
}

// putText lưu một nội dung test: ≤ 64 KiB ở DB, lớn hơn ở kho đối tượng (không có kho → 503). `made` ghi khoá mới để dọn khi lỗi.
func (s *Service) putText(ctx context.Context, courseID, qid uuid.UUID, text, ext string, made *[]string) (stored, error) {
	if !utf8.ValidString(text) {
		return stored{}, apierr.Validation(apierr.FieldError{Field: ext, Code: "TEST_NOT_UTF8", Message: "Nội dung test phải là văn bản UTF-8."})
	}
	n := len(text)
	if n > MaxTestBytes {
		return stored{}, apierr.Validation(apierr.FieldError{Field: ext, Code: "TEST_TOO_LARGE", Message: "Mỗi input / expected tối đa 1 MiB."})
	}
	if n <= MaxInlineBytes {
		return stored{inline: &text, n: n}, nil
	}
	if s.Blob == nil {
		return stored{}, apierr.New(http.StatusServiceUnavailable, apierr.ServiceUnavailable)
	}
	k := fmt.Sprintf("exam/tests/%s/%s/%s.%s", courseID, qid, uuid.NewString(), ext)
	if err := s.Blob.Put(ctx, k, strings.NewReader(text), int64(n), "text/plain; charset=utf-8"); err != nil {
		return stored{}, fmt.Errorf("exam: ghi kho đối tượng: %w", err)
	}
	*made = append(*made, k)
	return stored{key: &k, n: n}, nil
}

func (s *Service) dropBlobs(ctx context.Context, keys []string) {
	if s.Blob == nil {
		return
	}
	for _, k := range keys {
		_ = s.Blob.Delete(context.WithoutCancel(ctx), k) // dọn nỗ lực tối đa; mồ côi vô hại
	}
}

// ReadText đọc nội dung đầy đủ của một phía test (DB hoặc kho đối tượng).
func (s *Service) ReadText(ctx context.Context, inline, key *string) (string, error) {
	if key == nil {
		if inline == nil {
			return "", nil
		}
		return *inline, nil
	}
	if s.Blob == nil {
		return "", errors.New("exam: không có kho đối tượng để đọc test lớn")
	}
	rc, err := s.Blob.Get(ctx, *key)
	if err != nil {
		return "", fmt.Errorf("exam: đọc kho đối tượng: %w", err)
	}
	defer func() { _ = rc.Close() }()
	b, err := io.ReadAll(io.LimitReader(rc, MaxTestBytes+1))
	if err != nil {
		return "", fmt.Errorf("exam: đọc kho đối tượng: %w", err)
	}
	return string(b), nil
}

func preview(inline *string, n int) (string, bool) {
	if inline != nil {
		return *inline, false
	}
	return "", n > 0
}

func toTestcase(r store.CodeTestcase) Testcase {
	in, inT := preview(r.Input, int(r.InputBytes))
	ex, exT := preview(r.Expected, int(r.ExpectedBytes))
	return Testcase{ID: r.ID, Position: int(r.Position), Name: r.Name, IsSample: r.IsSample, Weight: int(r.Weight), Input: in, InputTruncated: inT, Expected: ex, ExpectedTruncated: exT,
		InputBytes: int(r.InputBytes), ExpectedBytes: int(r.ExpectedBytes), Source: r.Source, Approved: r.Approved}
}

// ListTests trả test của bài (cả test ẩn, cả chưa duyệt) theo `position`, phân trang theo vị trí: `afterPos` = vị trí cuối của trang trước.
func (s *Service) ListTests(ctx context.Context, courseID, id uuid.UUID, afterPos *int, fetch int) ([]Testcase, error) {
	q := store.New(s.Pool)
	row, err := q.QuestionGet(ctx, store.QuestionGetParams{CourseID: courseID, ID: id})
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && row.Type != store.QuestionTypeCODE) {
		return nil, notFound()
	}
	if err != nil {
		return nil, fmt.Errorf("exam: đọc câu hỏi: %w", err)
	}
	arg := store.TestcaseListParams{CourseID: courseID, ProblemID: id, MaxRows: int32(fetch)} //nolint:gosec // ≤ 101
	if afterPos != nil {
		p := int32(*afterPos) //nolint:gosec // vị trí ≤ 100
		arg.CursorPos = &p
	}
	rows, err := q.TestcaseList(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("exam: danh sách test: %w", err)
	}
	out := make([]Testcase, len(rows))
	for i, r := range rows {
		out[i] = toTestcase(store.CodeTestcase{ID: r.ID, Position: r.Position, Name: r.Name, IsSample: r.IsSample, Weight: r.Weight, Input: r.Input, InputBlobKey: r.InputBlobKey,
			Expected: r.Expected, ExpectedBlobKey: r.ExpectedBlobKey, InputBytes: r.InputBytes, ExpectedBytes: r.ExpectedBytes, Source: r.Source, Approved: r.Approved})
	}
	return out, nil
}

// mutateTests chạy một thay đổi lên tập test trong MỘT transaction: khoá sửa (guardEdit), tăng `tests_version`, gỡ cờ lời giải mẫu,
// đưa câu về DRAFT, ghi `audit_log` (`testcases.change`). `fn` trả số test bị đổi.
func (s *Service) mutateTests(ctx context.Context, actor, courseID, id uuid.UUID, isTeacher bool, fn func(q *store.Queries, cp store.CodeProblem) (int, error)) error {
	return s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
		if err := writable(ctx, q, courseID); err != nil {
			return err
		}
		row, err := q.QuestionLock(ctx, store.QuestionLockParams{CourseID: courseID, ID: id})
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && row.Type != store.QuestionTypeCODE) {
			return notFound()
		}
		if err != nil {
			return fmt.Errorf("exam: khoá câu hỏi: %w", err)
		}
		cp, err := q.CodeProblemLock(ctx, store.CodeProblemLockParams{CourseID: courseID, QuestionID: id})
		if err != nil {
			return fmt.Errorf("exam: khoá bài code: %w", err)
		}
		use, err := q.QuestionUsage(ctx, store.QuestionUsageParams{CourseID: courseID, QuestionID: id})
		if err != nil {
			return fmt.Errorf("exam: đọc bài thi đang dùng: %w", err)
		}
		if err := guardEdit(use, true, isTeacher); err != nil {
			return err
		}
		n, err := fn(q, cp)
		if err != nil {
			return err
		}
		upd, err := q.CodeProblemBumpTests(ctx, store.CodeProblemBumpTestsParams{CourseID: courseID, QuestionID: id})
		if err != nil {
			return fmt.Errorf("exam: tăng tests_version: %w", err)
		}
		if _, err := q.QuestionBump(ctx, store.QuestionBumpParams{CourseID: courseID, ID: id}); err != nil {
			return fmt.Errorf("exam: tăng version câu: %w", err)
		}
		return audit(ctx, q, courseID, actor, "question", id, "testcases.change",
			map[string]any{"tests_version": cp.TestsVersion}, map[string]any{"tests_version": upd.TestsVersion, "changed": n})
	})
}

func (in TestcaseIn) checkName() []apierr.FieldError {
	if in.Name != nil && !testNameRE.MatchString(*in.Name) {
		return []apierr.FieldError{{Field: "name", Code: "TEST_NAME", Message: "Tên test dài 1 đến 60 ký tự gồm chữ, số, dấu gạch ngang hoặc gạch dưới."}}
	}
	if in.Weight != nil && !inRange(*in.Weight, 0, 1000) {
		return []apierr.FieldError{{Field: "weight", Code: "LIMIT_OUT_OF_RANGE", Message: "Trọng số từ 0 đến 1.000."}}
	}
	return nil
}

// AddTest thêm một test tạo tay (`approved=true`, `source=MANUAL`), `position` tự gán cuối. Tối đa 100 test mỗi bài.
func (s *Service) AddTest(ctx context.Context, actor, courseID, id uuid.UUID, in TestcaseIn, isTeacher bool) (Testcase, error) {
	if errs := in.checkName(); len(errs) > 0 {
		return Testcase{}, apierr.Validation(errs...)
	}
	if in.Name == nil || in.Input == nil || in.Expected == nil {
		return Testcase{}, apierr.Validation(apierr.FieldError{Field: "name", Code: "required", Message: "Cần name, input và expected."})
	}
	var out Testcase
	var made []string
	err := s.mutateTests(ctx, actor, courseID, id, isTeacher, func(q *store.Queries, _ store.CodeProblem) (int, error) {
		st, err := q.TestcaseStats(ctx, store.TestcaseStatsParams{CourseID: courseID, ProblemID: id})
		if err != nil {
			return 0, fmt.Errorf("exam: đếm test: %w", err)
		}
		if st.TotalAll >= MaxTests {
			return 0, apierr.Validation(apierr.FieldError{Field: "testcases", Code: "TOO_MANY_TESTS", Message: "Mỗi bài tối đa 100 test."})
		}
		inp, err := s.putText(ctx, courseID, id, *in.Input, "in", &made)
		if err != nil {
			return 0, err
		}
		exp, err := s.putText(ctx, courseID, id, *in.Expected, "out", &made)
		if err != nil {
			return 0, err
		}
		w := 1
		if in.Weight != nil {
			w = *in.Weight
		}
		row, err := q.TestcaseInsert(ctx, store.TestcaseInsertParams{CourseID: courseID, ProblemID: id, Position: st.MaxPosition + 1, Name: *in.Name, IsSample: in.IsSample != nil && *in.IsSample,
			Weight: int16(w), Input: inp.inline, InputBlobKey: inp.key, Expected: exp.inline, ExpectedBlobKey: exp.key, InputBytes: int32(inp.n), ExpectedBytes: int32(exp.n), Source: "MANUAL", Approved: true}) //nolint:gosec // đã kiểm khoảng
		if err != nil {
			return 0, fmt.Errorf("exam: ghi test: %w", err)
		}
		out = toTestcase(row)
		return 1, nil
	})
	if err != nil {
		s.dropBlobs(ctx, made)
	}
	return out, err
}

// UpdateTest sửa một test (trường vắng giữ nguyên); đổi `position` chèn test vào vị trí mới và đánh số lại.
func (s *Service) UpdateTest(ctx context.Context, actor, courseID, id, tid uuid.UUID, in TestcaseIn, isTeacher bool) (Testcase, error) {
	if errs := in.checkName(); len(errs) > 0 {
		return Testcase{}, apierr.Validation(errs...)
	}
	var out Testcase
	var made, drop []string
	err := s.mutateTests(ctx, actor, courseID, id, isTeacher, func(q *store.Queries, _ store.CodeProblem) (int, error) {
		cur, err := q.TestcaseGet(ctx, store.TestcaseGetParams{CourseID: courseID, ProblemID: id, ID: tid})
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, notFound()
		}
		if err != nil {
			return 0, fmt.Errorf("exam: đọc test: %w", err)
		}
		arg := store.TestcaseUpdateParams{CourseID: courseID, ProblemID: id, ID: tid, Name: cur.Name, IsSample: cur.IsSample, Weight: cur.Weight, Position: cur.Position,
			Input: cur.Input, InputBlobKey: cur.InputBlobKey, Expected: cur.Expected, ExpectedBlobKey: cur.ExpectedBlobKey, InputBytes: cur.InputBytes, ExpectedBytes: cur.ExpectedBytes}
		if in.Name != nil {
			arg.Name = *in.Name
		}
		if in.IsSample != nil {
			arg.IsSample = *in.IsSample
		}
		if in.Weight != nil {
			arg.Weight = int16(*in.Weight) //nolint:gosec // đã kiểm khoảng
		}
		if in.Input != nil {
			st, err := s.putText(ctx, courseID, id, *in.Input, "in", &made)
			if err != nil {
				return 0, err
			}
			if cur.InputBlobKey != nil {
				drop = append(drop, *cur.InputBlobKey)
			}
			arg.Input, arg.InputBlobKey, arg.InputBytes = st.inline, st.key, int32(st.n) //nolint:gosec // ≤ 1 MiB
		}
		if in.Expected != nil {
			st, err := s.putText(ctx, courseID, id, *in.Expected, "out", &made)
			if err != nil {
				return 0, err
			}
			if cur.ExpectedBlobKey != nil {
				drop = append(drop, *cur.ExpectedBlobKey)
			}
			arg.Expected, arg.ExpectedBlobKey, arg.ExpectedBytes = st.inline, st.key, int32(st.n) //nolint:gosec // ≤ 1 MiB
		}
		row, err := q.TestcaseUpdate(ctx, arg)
		if err != nil {
			return 0, fmt.Errorf("exam: sửa test: %w", err)
		}
		if in.Position != nil && *in.Position != int(cur.Position) {
			if err := renumber(ctx, q, courseID, id, tid, *in.Position); err != nil {
				return 0, err
			}
			row, err = q.TestcaseGet(ctx, store.TestcaseGetParams{CourseID: courseID, ProblemID: id, ID: tid})
			if err != nil {
				return 0, fmt.Errorf("exam: đọc test: %w", err)
			}
		}
		out = toTestcase(row)
		return 1, nil
	})
	if err != nil {
		s.dropBlobs(ctx, made)
		return out, err
	}
	s.dropBlobs(ctx, drop)
	return out, nil
}

// renumber đặt test `tid` vào vị trí `pos` (kẹp vào 1…n) rồi đánh số lại 1…n theo thứ tự mới.
func renumber(ctx context.Context, q *store.Queries, courseID, id, tid uuid.UUID, pos int) error {
	ids, err := q.TestcaseIDsByPosition(ctx, store.TestcaseIDsByPositionParams{CourseID: courseID, ProblemID: id})
	if err != nil {
		return fmt.Errorf("exam: đọc thứ tự test: %w", err)
	}
	ids = slices.DeleteFunc(ids, func(x uuid.UUID) bool { return x == tid })
	pos = max(1, min(pos, len(ids)+1))
	ids = slices.Insert(ids, pos-1, tid)
	for i, x := range ids {
		if err := q.TestcaseSetPosition(ctx, store.TestcaseSetPositionParams{CourseID: courseID, ProblemID: id, ID: x, Position: int32(i + 1)}); err != nil { //nolint:gosec // ≤ 100
			return fmt.Errorf("exam: đánh số lại test: %w", err)
		}
	}
	return nil
}

// DeleteTest xoá một test.
func (s *Service) DeleteTest(ctx context.Context, actor, courseID, id, tid uuid.UUID, isTeacher bool) error {
	var drop []string
	err := s.mutateTests(ctx, actor, courseID, id, isTeacher, func(q *store.Queries, _ store.CodeProblem) (int, error) {
		k, err := q.TestcaseDelete(ctx, store.TestcaseDeleteParams{CourseID: courseID, ProblemID: id, ID: tid})
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, notFound()
		}
		if err != nil {
			return 0, fmt.Errorf("exam: xoá test: %w", err)
		}
		for _, p := range []*string{k.InputBlobKey, k.ExpectedBlobKey} {
			if p != nil {
				drop = append(drop, *p)
			}
		}
		return 1, nil
	})
	if err == nil {
		s.dropBlobs(ctx, drop)
	}
	return err
}

// ApproveTests duyệt các test `ids` (test AI gợi ý mặc định chưa duyệt, không được chấm). Trả số test đã duyệt.
func (s *Service) ApproveTests(ctx context.Context, actor, courseID, id uuid.UUID, ids []uuid.UUID, isTeacher bool) (int, error) {
	n := 0
	err := s.mutateTests(ctx, actor, courseID, id, isTeacher, func(q *store.Queries, _ store.CodeProblem) (int, error) {
		c, err := q.TestcasesApprove(ctx, store.TestcasesApproveParams{CourseID: courseID, ProblemID: id, Ids: idStrings(ids)})
		if err != nil {
			return 0, fmt.Errorf("exam: duyệt test: %w", err)
		}
		n = int(c)
		return n, nil
	})
	return n, err
}

// copyCode sao chép cấu hình bài code + test (và nội dung ở kho đối tượng) sang câu mới; trả khoá blob mới để dọn nếu transaction lỗi.
func (s *Service) copyCode(ctx context.Context, q *store.Queries, courseID, from, to uuid.UUID) ([]string, error) {
	cp, err := q.CodeProblemGet(ctx, store.CodeProblemGetParams{CourseID: courseID, QuestionID: from})
	if err != nil {
		return nil, fmt.Errorf("exam: đọc bài code: %w", err)
	}
	if _, err := q.CodeProblemUpdate(ctx, store.CodeProblemUpdateParams{CourseID: courseID, QuestionID: to, Languages: cp.Languages, TimeLimitMs: cp.TimeLimitMs, MemoryLimitMb: cp.MemoryLimitMb,
		OutputLimitKb: cp.OutputLimitKb, Checker: cp.Checker, FloatEps: cp.FloatEps, StarterCode: cp.StarterCode, ReferenceLanguage: cp.ReferenceLanguage, ReferenceSource: cp.ReferenceSource}); err != nil {
		return nil, fmt.Errorf("exam: ghi bài code bản sao: %w", err)
	}
	tests, err := q.TestcaseList(ctx, store.TestcaseListParams{CourseID: courseID, ProblemID: from, MaxRows: MaxTests})
	if err != nil {
		return nil, fmt.Errorf("exam: đọc test: %w", err)
	}
	var made []string
	for _, t := range tests {
		arg := store.TestcaseInsertParams{CourseID: courseID, ProblemID: to, Position: t.Position, Name: t.Name, IsSample: t.IsSample, Weight: t.Weight, Input: t.Input, Expected: t.Expected,
			InputBytes: t.InputBytes, ExpectedBytes: t.ExpectedBytes, Source: t.Source, Approved: t.Approved}
		for _, side := range []struct {
			from *string
			to   **string
			ext  string
		}{{t.InputBlobKey, &arg.InputBlobKey, "in"}, {t.ExpectedBlobKey, &arg.ExpectedBlobKey, "out"}} {
			if side.from == nil {
				continue
			}
			text, err := s.ReadText(ctx, nil, side.from)
			if err != nil {
				return made, err
			}
			st, err := s.putText(ctx, courseID, to, text, side.ext, &made)
			if err != nil {
				return made, err
			}
			*side.to = st.key
		}
		if _, err := q.TestcaseInsert(ctx, arg); err != nil {
			return made, fmt.Errorf("exam: ghi test bản sao: %w", err)
		}
	}
	return made, nil
}

// idStrings đổi danh sách uuid sang chuỗi: PgBouncer (giao thức đơn giản) không mã hoá được []uuid.UUID; truy vấn ép lại bằng `::text[]::uuid[]`.
func idStrings(ids []uuid.UUID) []string {
	out := make([]string, len(ids))
	for i, x := range ids {
		out[i] = x.String()
	}
	return out
}
