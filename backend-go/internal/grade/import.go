package grade

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"github.com/xuri/excelize/v2"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/store"
)

// Hằng số nhập XLSX (SRS FEAT-gradebook 4.1, 4.6.4). Mọi giá trị có tên.
const (
	ImportMaxBytes   = 2 << 20 // GRADE_IMPORT_MAX_BYTES (cùng giá trị với document.GradeImportMaxBytes)
	ImportMaxRows    = 5000    // GRADE_IMPORT_MAX_ROWS: số dòng DỮ LIỆU tối đa
	importUnzipLimit = 20 << 20
	importSampleMax  = 20
	purposeImport    = "grade_import"

	// Mã lỗi của từng dòng (khác mã lỗi HTTP).
	RowNotInCourse = "STUDENT_NOT_IN_COURSE"
	RowUnverified  = "STUDENT_CODE_UNVERIFIED"
	RowDuplicate   = "DUPLICATE_IN_FILE"
	RowAmbiguous   = "AMBIGUOUS_STUDENT_CODE"
	RowOutOfRange  = "SCORE_OUT_OF_RANGE"
	RowPrecision   = "SCORE_PRECISION"
	RowNotNumber   = "SCORE_NOT_NUMBER"
)

// ImportMapping: cột là chữ cái ("A", "C"); HeaderRow ≥ 1 (dòng tiêu đề, dữ liệu bắt đầu ở dòng kế).
type ImportMapping struct {
	HeaderRow      int    `json:"header_row"`
	StudentCodeCol string `json:"student_code_col"`
	ScoreCol       string `json:"score_col"`
}

// ImportInput là thân `POST …/grade-entries/import`.
type ImportInput struct {
	UploadID   uuid.UUID     `json:"upload_id"`
	ItemID     uuid.UUID     `json:"item_id"`
	Mode       string        `json:"mode"`
	Mapping    ImportMapping `json:"mapping"`
	SkipErrors bool          `json:"skip_errors"`
}

type (
	ImportError struct {
		Row         int    `json:"row"`
		StudentCode string `json:"student_code"`
		Code        string `json:"code"`
		Message     string `json:"message"`
	}
	ImportStudent struct {
		ID          uuid.UUID `json:"id"`
		FullName    string    `json:"full_name"`
		StudentCode string    `json:"student_code"`
	}
	ImportSample struct {
		Student ImportStudent `json:"student"`
		Before  *string       `json:"before"`
		After   string        `json:"after"`
	}
	// ImportOut: kết quả xem trước / ghi. `Headers` + `Suggested` là dòng tiêu đề của tệp và ánh xạ gợi ý theo tên cột.
	ImportOut struct {
		RowsTotal  int            `json:"rows_total"`
		RowsOK     int            `json:"rows_ok"`
		Overwrites int            `json:"overwrites"`
		Unchanged  int            `json:"unchanged"`
		Errors     []ImportError  `json:"errors"`
		Sample     []ImportSample `json:"sample"`
		Headers    []string       `json:"headers"`
		Suggested  ImportMapping  `json:"suggested_mapping"`
		Written    int            `json:"written"`
		Skipped    int            `json:"skipped_errors"`
	}
)

var rowMessage = map[string]string{ //nolint:gochecknoglobals // bảng chữ bất biến
	RowNotInCourse: "MSSV không có trong lớp",
	RowUnverified:  "MSSV chưa có trong danh sách lớp",
	RowDuplicate:   "MSSV bị lặp trong tệp",
	RowAmbiguous:   "Hai sinh viên cùng MSSV trong lớp",
	RowNotNumber:   "Điểm không phải số",
	RowPrecision:   "Điểm có quá 2 chữ số thập phân",
}

// BlobReader là phần của object storage mà nhập điểm dùng.
type BlobReader interface {
	Get(ctx context.Context, key string) (io.ReadCloser, error)
}

var (
	reRealCode = regexp.MustCompile(`^(\d+)\.0+$`)                                                    //nolint:gochecknoglobals // MSSV số thực "20229002.0"
	reCodeHdr  = []string{"mssv", "msv", "mã sinh viên", "ma sinh vien", "mã sv", "ma sv", "student"} //nolint:gochecknoglobals
	reScoreHdr = []string{"điểm", "diem", "score", "grade", "mark"}                                   //nolint:gochecknoglobals
)

// normCode chuẩn hoá MSSV so SANH CHUỖI: TrimSpace + chữ hoa; ô số thực "20229002.0" → "20229002"; giữ số 0 đầu (không bao giờ đi qua số).
func normCode(raw string) string {
	t := strings.ToUpper(strings.TrimSpace(raw))
	if m := reRealCode.FindStringSubmatch(t); m != nil {
		return m[1]
	}
	return t
}

// SuggestMapping gợi ý cột theo tên tiêu đề ("MSSV", "Điểm"…); không thấy thì để trống.
func SuggestMapping(headerRow int, headers []string) ImportMapping {
	m := ImportMapping{HeaderRow: max(headerRow, 1)}
	for i, h := range headers {
		l := strings.ToLower(strings.TrimSpace(h))
		col, _ := excelize.ColumnNumberToName(i + 1)
		if m.StudentCodeCol == "" && containsAny(l, reCodeHdr) {
			m.StudentCodeCol = col
		} else if m.ScoreCol == "" && containsAny(l, reScoreHdr) {
			m.ScoreCol = col
		}
	}
	return m
}

func containsAny(s string, subs []string) bool {
	for _, x := range subs {
		if strings.Contains(s, x) {
			return true
		}
	}
	return false
}

// sheetData: dòng dữ liệu đã đọc (giá trị ĐÃ ĐỊNH DẠNG của ô; công thức không bao giờ được thực thi — chỉ đọc giá trị lưu sẵn).
type sheetData struct {
	headers []string
	rows    []sheetRow
}
type sheetRow struct {
	n    int
	cols []string
}

// readUpload kiểm vé, đọc lại tệp từ object storage (xem trước = ghi), so sha256 đã khai ở presign rồi đọc trang tính đầu.
func (s *Service) readUpload(ctx context.Context, actor, course, upload uuid.UUID, headerRow int) (data sheetData, err error) {
	if s.Redis == nil {
		return data, errors.New("grade: nhập XLSX cần Redis")
	}
	tk, err := s.Redis.HGetAll(ctx, appredis.Key("upload", upload.String())).Result()
	if err != nil {
		return data, fmt.Errorf("grade: đọc vé tải lên: %w", err)
	}
	if len(tk) == 0 || tk["user_id"] != actor.String() || tk["course_id"] != course.String() || tk["purpose"] != purposeImport {
		return data, apierr.New(http.StatusNotFound, apierr.UploadNotFound)
	}
	if s.Blob == nil {
		return data, errors.New("grade: nhập XLSX cần object storage")
	}
	obj, err := s.Blob.Get(ctx, tk["blob_key"])
	if err != nil {
		return data, apierr.New(http.StatusUnprocessableEntity, apierr.FileInvalid)
	}
	defer func() { _ = obj.Close() }()
	raw, err := io.ReadAll(io.LimitReader(obj, ImportMaxBytes+1))
	if err != nil {
		return data, fmt.Errorf("grade: đọc tệp nhập: %w", err)
	}
	if len(raw) > ImportMaxBytes {
		return data, apierr.New(http.StatusUnprocessableEntity, apierr.FileTooLarge)
	}
	sum := sha256.Sum256(raw)
	if hex.EncodeToString(sum[:]) != tk["sha256"] {
		return data, apierr.New(http.StatusUnprocessableEntity, apierr.FileInvalid)
	}
	defer func() { // tệp lạ: thư viện không được làm sập tiến trình
		if r := recover(); r != nil {
			data, err = sheetData{}, apierr.New(http.StatusUnprocessableEntity, apierr.FileInvalid)
		}
	}()
	f, err := excelize.OpenReader(bytes.NewReader(raw), excelize.Options{UnzipSizeLimit: importUnzipLimit, UnzipXMLSizeLimit: importUnzipLimit})
	if err != nil {
		return data, apierr.New(http.StatusUnprocessableEntity, apierr.FileInvalid)
	}
	defer func() { _ = f.Close() }()
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return data, apierr.New(http.StatusUnprocessableEntity, apierr.FileInvalid)
	}
	it, err := f.Rows(sheets[0])
	if err != nil {
		return data, apierr.New(http.StatusUnprocessableEntity, apierr.FileInvalid)
	}
	defer func() { _ = it.Close() }()
	n := 0
	for it.Next() {
		n++
		cols, err := it.Columns()
		if err != nil {
			return data, apierr.New(http.StatusUnprocessableEntity, apierr.FileInvalid)
		}
		if n == headerRow {
			data.headers = cols
		}
		if n <= headerRow {
			continue
		}
		if n-headerRow > ImportMaxRows {
			return data, apierr.New(http.StatusUnprocessableEntity, apierr.TooManyRows)
		}
		data.rows = append(data.rows, sheetRow{n: n, cols: cols})
	}
	if err := it.Error(); err != nil {
		return data, apierr.New(http.StatusUnprocessableEntity, apierr.FileInvalid)
	}
	return data, nil
}

func cellAt(cols []string, col int) string {
	if col < 1 || col > len(cols) {
		return ""
	}
	return cols[col-1]
}

type importRow struct {
	row     int
	student uuid.UUID
	score   decimal.Decimal
}

type importPlan struct {
	out   ImportOut
	apply []importRow // dòng hợp lệ có điểm (kể cả dòng không đổi — câu lệnh theo lô tự bỏ qua dòng không đổi)
}

// analyze khớp MSSV với `enrollments.student_code_snapshot` của dòng ACTIVE `joined_via IN ('ROSTER','ADMIN')` (KHÔNG theo MSSV tự khai — #20), kiểm điểm,
// so với điểm hiện có. Thuần theo dữ liệu đã nạp: số lượt truy vấn không phụ thuộc số dòng.
func analyze(data sheetData, m ImportMapping, max decimal.Decimal, roster []store.GradeImportRosterRow, existing map[uuid.UUID]decimal.Decimal) importPlan {
	trusted, selfDeclared := map[string][]store.GradeImportRosterRow{}, map[string]bool{}
	for _, r := range roster {
		if r.StudentCode == nil {
			continue
		}
		c := normCode(*r.StudentCode)
		if r.JoinedVia == store.EnrollmentJoinedViaCODE {
			selfDeclared[c] = true
		} else {
			trusted[c] = append(trusted[c], r)
		}
	}
	codeCol, _ := excelize.ColumnNameToNumber(strings.ToUpper(m.StudentCodeCol))
	scoreCol, _ := excelize.ColumnNameToNumber(strings.ToUpper(m.ScoreCol))
	seen := map[string]bool{}
	p := importPlan{out: ImportOut{Errors: []ImportError{}, Sample: []ImportSample{}, Headers: data.headers}}
	fail := func(row int, code, codeStr, msg string) {
		if msg == "" {
			msg = rowMessage[code]
		}
		p.out.Errors = append(p.out.Errors, ImportError{Row: row, StudentCode: codeStr, Code: code, Message: msg})
	}
	for _, r := range data.rows {
		rawCode, rawScore := cellAt(r.cols, codeCol), strings.TrimSpace(cellAt(r.cols, scoreCol))
		code := normCode(rawCode)
		if code == "" && rawScore == "" {
			continue // dòng trống: bỏ qua, không lỗi, không tính
		}
		p.out.RowsTotal++
		if code != "" && seen[code] {
			fail(r.n, RowDuplicate, code, "")
			continue
		}
		seen[code] = true
		var st store.GradeImportRosterRow
		switch hits := trusted[code]; {
		case code == "" || (len(hits) == 0 && !selfDeclared[code]):
			fail(r.n, RowNotInCourse, code, "")
			continue
		case len(hits) == 0:
			fail(r.n, RowUnverified, code, "")
			continue
		case len(hits) > 1:
			fail(r.n, RowAmbiguous, code, "")
			continue
		default:
			st = hits[0]
		}
		if rawScore == "" {
			p.out.RowsOK++
			p.out.Unchanged++
			continue
		}
		d, err := ParseScore(rawScore)
		if err != nil {
			if rawScore != "" && scoreManyRE.MatchString(rawScore) {
				fail(r.n, RowPrecision, code, "")
			} else {
				fail(r.n, RowNotNumber, code, "")
			}
			continue
		}
		if d.GreaterThan(max) {
			fail(r.n, RowOutOfRange, code, fmt.Sprintf("Điểm ngoài thang 0–%s", strings.Replace(max.String(), ".", ",", 1)))
			continue
		}
		p.apply = append(p.apply, importRow{row: r.n, student: st.StudentID, score: d})
		p.out.RowsOK++
		old, has := existing[st.StudentID]
		if has && old.Equal(d) {
			p.out.Unchanged++
			continue
		}
		if has {
			p.out.Overwrites++
		}
		if len(p.out.Sample) < importSampleMax {
			s := ImportSample{Student: ImportStudent{ID: st.StudentID, FullName: st.FullName, StudentCode: code}, After: d.Round(2).String()}
			if has {
				b := old.Round(2).String()
				s.Before = &b
			}
			p.out.Sample = append(p.out.Sample, s)
		}
	}
	return p
}

// Import xem trước (`preview`, không ghi) hoặc ghi (`commit`) điểm từ XLSX vào MỘT item nhập tay (SRS 4.6.4). Ghi: một giao dịch, khoá sổ `FOR SHARE`
// (LOCKED → 409), MỘT câu lệnh theo lô (CTE `old … FOR UPDATE` → `INSERT … ON CONFLICT` → audit từng điểm đổi) + một dòng audit tổng của lô.
func (s *Service) Import(ctx context.Context, actor, courseID uuid.UUID, in ImportInput) (ImportOut, error) {
	if in.Mode != "preview" && in.Mode != "commit" {
		return ImportOut{}, apierr.Validation(apierr.FieldError{Field: "mode", Code: "enum", Message: "Chế độ là preview hoặc commit."})
	}
	if in.Mapping.HeaderRow < 1 || in.Mapping.HeaderRow > 100 {
		in.Mapping.HeaderRow = 1
	}
	q := store.New(s.Pool)
	item, err := q.GradeItemForEdit(ctx, store.GradeItemForEditParams{CourseID: courseID, ItemID: in.ItemID})
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && item.SchemeStatus != store.GradeSchemeStatusCONFIRMED) {
		return ImportOut{}, apierr.New(http.StatusNotFound, apierr.NotFound)
	}
	if err != nil {
		return ImportOut{}, fmt.Errorf("grade: đọc cột điểm: %w", err)
	}
	if item.Source != store.GradeItemSourceMANUAL {
		return ImportOut{}, apierr.New(http.StatusConflict, apierr.ItemNotEditable)
	}
	data, err := s.readUpload(ctx, actor, courseID, in.UploadID, in.Mapping.HeaderRow)
	if err != nil {
		return ImportOut{}, err
	}
	suggested := SuggestMapping(in.Mapping.HeaderRow, data.headers)
	if in.Mapping.StudentCodeCol == "" || in.Mapping.ScoreCol == "" { // chưa chọn cột: chỉ trả tiêu đề + gợi ý
		return ImportOut{Errors: []ImportError{}, Sample: []ImportSample{}, Headers: data.headers, Suggested: suggested}, nil
	}
	if in.Mode == "preview" {
		p, err := s.plan(ctx, q, courseID, item, in, data)
		if err != nil {
			return ImportOut{}, err
		}
		p.out.Suggested = suggested
		return p.out, nil
	}

	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return ImportOut{}, fmt.Errorf("grade: mở giao dịch: %w", err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	tq := store.New(tx)
	gb, err := tq.GradebookStatusShare(ctx, courseID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ImportOut{}, apierr.New(http.StatusConflict, apierr.SchemeNotConfirmed)
	}
	if err != nil {
		return ImportOut{}, fmt.Errorf("grade: trạng thái sổ điểm: %w", err)
	}
	if gb == store.GradebookStatusLOCKED {
		return ImportOut{}, apierr.New(http.StatusConflict, apierr.GradebookLocked)
	}
	p, err := s.plan(ctx, tq, courseID, item, in, data)
	if err != nil {
		return ImportOut{}, err
	}
	if len(p.out.Errors) > 0 && !in.SkipErrors {
		return ImportOut{}, apierr.New(http.StatusConflict, apierr.ImportHasErrors).WithDetails(map[string]any{"errors": len(p.out.Errors), "rows_ok": p.out.RowsOK})
	}
	if len(p.apply) > 0 {
		ids, scores := make([]string, len(p.apply)), make([]string, len(p.apply))
		for i, r := range p.apply {
			ids[i], scores[i] = r.student.String(), r.score.Round(2).String()
		}
		res, err := tq.GradeImportApply(ctx, store.GradeImportApplyParams{CourseID: courseID, ItemID: in.ItemID, ItemKey: item.Key, ActorID: actor, StudentIds: ids, Scores: scores})
		if err != nil {
			return ImportOut{}, fmt.Errorf("grade: ghi điểm nhập: %w", err)
		}
		p.out.Written = int(res.Inserted + res.Updated)
	}
	p.out.Skipped = len(p.out.Errors)
	if p.out.Written == 0 { // không đổi gì (nhập lại cùng tệp): không audit thừa
		if err := tx.Commit(ctx); err != nil {
			return ImportOut{}, fmt.Errorf("grade: kết thúc giao dịch nhập: %w", err)
		}
		p.out.Suggested = suggested
		return p.out, nil
	}
	after, _ := json.Marshal(map[string]any{"item_id": in.ItemID, "item_key": item.Key, "rows_ok": p.out.RowsOK, "written": p.out.Written, "unchanged": p.out.Unchanged, "errors_skipped": p.out.Skipped, "skip_errors": in.SkipErrors})
	if _, err := tq.InsertAuditLog(ctx, store.InsertAuditLogParams{CourseID: &courseID, ActorID: &actor, Entity: "grade_imports", EntityID: in.ItemID.String(), Action: "import", After: after}); err != nil {
		return ImportOut{}, fmt.Errorf("grade: audit lô nhập: %w", err) // fail-closed
	}
	if err := tx.Commit(ctx); err != nil {
		return ImportOut{}, fmt.Errorf("grade: lưu điểm nhập: %w", err)
	}
	p.out.Suggested = suggested
	return p.out, nil
}

func (s *Service) plan(ctx context.Context, q *store.Queries, courseID uuid.UUID, item store.GradeItemForEditRow, in ImportInput, data sheetData) (importPlan, error) {
	roster, err := q.GradeImportRoster(ctx, courseID)
	if err != nil {
		return importPlan{}, fmt.Errorf("grade: danh sách lớp: %w", err)
	}
	cur, err := q.GradeImportExisting(ctx, store.GradeImportExistingParams{CourseID: courseID, ItemID: in.ItemID})
	if err != nil {
		return importPlan{}, fmt.Errorf("grade: điểm hiện có: %w", err)
	}
	existing := make(map[uuid.UUID]decimal.Decimal, len(cur))
	for _, e := range cur {
		existing[e.StudentID] = e.Score
	}
	return analyze(data, in.Mapping, item.MaxScore, roster, existing), nil
}
