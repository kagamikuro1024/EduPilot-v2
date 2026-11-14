package practice

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"github.com/xuri/excelize/v2"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/quiz"
	"github.com/edupilot/backend-go/internal/store"
)

// Nhập kết quả Microsoft / Google Forms vào sổ điểm (US-P9-11, SRS FEAT-practice 4.5). Chỉ TEACHER. Chấm bằng `quiz` + decimal, KHÔNG LLM.
// Ba bước tách biệt: tải lên + `preview` (đọc, không ghi) → `preview` kèm ánh xạ (xem trước điểm, không ghi) → xác nhận (ghi). Xác nhận đòi `preview` trước.
const (
	FormsMaxBytes     = 5 << 20 // cùng giá trị với document.FormsImportMaxBytes
	FormsMaxRows      = 5000
	FormsMaxQuestions = 100
	formsMaxCols      = 200
	formsPreviewRows  = 20
	formsCellMax      = 200
	formsUnzipLimit   = 50 << 20
	formsTicketTTL    = time.Hour // vé tải lên (mặc định 15 phút) được kéo dài khi đã xem trước, đủ thời gian ánh xạ cột
	purposeForms      = "forms_import"
)

// FormsQuestion ánh xạ MỘT cột của tệp: câu của ngân hàng (`question_id`) hoặc đáp án nhập tay (`accepted`, so khớp NormalizeShort).
type FormsQuestion struct {
	Col        string     `json:"col"`
	QuestionID *uuid.UUID `json:"question_id"`
	Accepted   []string   `json:"accepted"`
}

// FormsMapping: cột là chữ cái ("A", "C"); `HeaderRow` ≥ 1. Định danh: `EmailCol` và / hoặc `CodeCol`. `TimeCol` (tuỳ chọn) chọn dòng mới nhất khi trùng.
type FormsMapping struct {
	HeaderRow int             `json:"header_row"`
	EmailCol  string          `json:"email_col"`
	CodeCol   string          `json:"code_col"`
	TimeCol   string          `json:"time_col"`
	Questions []FormsQuestion `json:"questions"`
}

// FormsIn là thân của `preview` và `forms-import`. `preview` chỉ cần `UploadID`; có thêm ánh xạ + `ItemID` + `AnswerSource` thì trả bảng điểm.
type FormsIn struct {
	UploadID     uuid.UUID     `json:"upload_id"`
	ItemID       *uuid.UUID    `json:"item_id"`
	AnswerSource string        `json:"answer_source"` // BANK | MANUAL
	Mapping      *FormsMapping `json:"mapping"`
}

type (
	FormsColumn struct {
		Col    string `json:"col"`
		Header string `json:"header"`
	}
	FormsStudent struct {
		ID          uuid.UUID `json:"id"`
		FullName    string    `json:"full_name"`
		StudentCode string    `json:"student_code"`
	}
	FormsScored struct {
		Student  FormsStudent `json:"student"`
		Correct  int          `json:"correct"`
		Total    int          `json:"total"`
		Score    string       `json:"score"`
		Mismatch int          `json:"mismatch"` // ô có đáp án không khớp lựa chọn nào của câu (tính 0, không đoán); 0 khi chỉ đọc lại lô đã ghi
	}
	FormsUnmatched struct {
		Row        int    `json:"row"`
		Identifier string `json:"identifier"` // đã che
		Reason     string `json:"reason"`     // NOT_FOUND | MISMATCH
		Correct    int    `json:"correct"`
		Total      int    `json:"total"`
		Score      string `json:"score"`
	}
	// FormsResult: bảng điểm đã chấm. `BatchID` chỉ có sau khi ghi; `Replayed` = cùng tệp (cùng hash) đã nhập trước đó, không ghi gì thêm.
	FormsResult struct {
		Matched    int              `json:"matched"`
		Unmatched  int              `json:"unmatched"`
		Duplicates int              `json:"duplicates"`
		Items      []FormsScored    `json:"items"`
		Rows       []FormsUnmatched `json:"unmatched_rows"`
		BatchID    *uuid.UUID       `json:"batch_id,omitempty"`
		Written    int              `json:"written"`
		Replayed   bool             `json:"replayed"`
	}
	FormsPreviewOut struct {
		Columns   []FormsColumn `json:"columns"`
		Rows      [][]string    `json:"rows"`
		RowsTotal int           `json:"rows_total"`
		Suggested FormsMapping  `json:"suggested_mapping"`
		Result    *FormsResult  `json:"result,omitempty"`
	}
)

var (
	reFormsReal = regexp.MustCompile(`^(\d+)\.0+$`)                                                                                   //nolint:gochecknoglobals // MSSV số thực "20229002.0"
	hdrEmail    = []string{"email", "e-mail"}                                                                                         //nolint:gochecknoglobals
	hdrCode     = []string{"mssv", "msv", "mã sinh viên", "ma sinh vien", "mã sv", "student id", "student code"}                      //nolint:gochecknoglobals
	hdrTime     = []string{"completion time", "thời gian hoàn thành", "timestamp", "dấu thời gian", "submitted", "end time"}          //nolint:gochecknoglobals
	timeLayouts = []string{"2006-01-02 15:04:05", "2006-01-02T15:04:05Z07:00", "2006-01-02 15:04", "1/2/06 15:04:05", "1/2/06 15:04", //nolint:gochecknoglobals
		"1/2/2006 15:04:05", "1/2/2006 15:04", "2006/01/02 3:04:05 PM", "2006/01/02 15:04:05", "2/1/2006 15:04:05"}
)

func formsCode(raw string) string {
	t := strings.ToUpper(strings.TrimSpace(raw))
	if m := reFormsReal.FindStringSubmatch(t); m != nil {
		return m[1]
	}
	return t
}

func hasAny(s string, subs []string) bool {
	for _, x := range subs {
		if strings.Contains(s, x) {
			return true
		}
	}
	return false
}

// suggestForms gợi ý cột theo tiêu đề; không thấy thì để trống.
func suggestForms(headerRow int, headers []string) FormsMapping {
	m := FormsMapping{HeaderRow: max(headerRow, 1), Questions: []FormsQuestion{}}
	for i, h := range headers {
		l := strings.ToLower(strings.TrimSpace(h))
		col, _ := excelize.ColumnNumberToName(i + 1)
		switch {
		case m.EmailCol == "" && hasAny(l, hdrEmail):
			m.EmailCol = col
		case m.CodeCol == "" && hasAny(l, hdrCode):
			m.CodeCol = col
		case m.TimeCol == "" && hasAny(l, hdrTime):
			m.TimeCol = col
		}
	}
	return m
}

func parseFormsTime(s string) time.Time {
	s = strings.TrimSpace(s)
	for _, l := range timeLayouts {
		if t, err := time.Parse(l, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

// maskIdent che định danh để lưu / trả về: "ab***@x.edu", "20****02".
func maskIdent(s string) string {
	s = strings.TrimSpace(s)
	r := []rune(s)
	if at := strings.IndexByte(s, '@'); at > 0 {
		head := []rune(s[:at])
		return string(head[:min(2, len(head))]) + "***" + s[at:]
	}
	if len(r) <= 4 {
		return strings.Repeat("*", len(r))
	}
	return string(r[:2]) + "****" + string(r[len(r)-2:])
}

// ---- đọc tệp ------------------------------------------------------------------------------------------------------------

type formsFile struct {
	hash, blobKey string
	grid          [][]string
}

// ticket kiểm vé tải lên của TEACHER này, ở lớp này, mục đích forms_import.
func (s *Service) ticket(ctx context.Context, actor, course, upload uuid.UUID) (map[string]string, error) {
	if s.Redis == nil {
		return nil, errors.New("practice: nhập Forms cần Redis")
	}
	tk, err := s.Redis.HGetAll(ctx, appredis.Key("upload", upload.String())).Result()
	if err != nil {
		return nil, fmt.Errorf("practice: đọc vé tải lên: %w", err)
	}
	if len(tk) == 0 || tk["user_id"] != actor.String() || tk["course_id"] != course.String() || tk["purpose"] != purposeForms {
		return nil, apierr.New(http.StatusNotFound, apierr.UploadNotFound)
	}
	return tk, nil
}

// load đọc lại tệp từ object storage, so sha256 đã khai ở presign rồi tách thành lưới ô (xlsx: trang đầu; csv: tự dò dấu phân cách).
func (s *Service) load(ctx context.Context, tk map[string]string) (f formsFile, err error) {
	if s.Blob == nil {
		return f, errors.New("practice: nhập Forms cần object storage")
	}
	obj, err := s.Blob.Get(ctx, tk["blob_key"])
	if err != nil {
		return f, apierr.New(http.StatusUnprocessableEntity, apierr.FileInvalid)
	}
	defer func() { _ = obj.Close() }()
	raw, err := io.ReadAll(io.LimitReader(obj, FormsMaxBytes+1))
	if err != nil {
		return f, fmt.Errorf("practice: đọc tệp Forms: %w", err)
	}
	if len(raw) > FormsMaxBytes {
		return f, apierr.New(http.StatusUnprocessableEntity, apierr.FileTooLarge)
	}
	sum := sha256.Sum256(raw)
	f.hash, f.blobKey = hex.EncodeToString(sum[:]), tk["blob_key"]
	if f.hash != tk["sha256"] {
		return f, apierr.New(http.StatusUnprocessableEntity, apierr.FileInvalid)
	}
	defer func() { // tệp lạ: thư viện không được làm sập tiến trình
		if r := recover(); r != nil {
			f, err = formsFile{}, apierr.New(http.StatusUnprocessableEntity, apierr.FileInvalid)
		}
	}()
	if f.grid, err = parseGrid(raw, strings.ToLower(tk["filename"])); err != nil {
		return formsFile{}, err
	}
	return f, nil
}

func parseGrid(raw []byte, filename string) ([][]string, error) {
	bad := apierr.New(http.StatusUnprocessableEntity, apierr.FileInvalid)
	var grid [][]string
	if strings.HasSuffix(filename, ".csv") {
		raw = bytes.TrimPrefix(raw, []byte("\xef\xbb\xbf"))
		first, _, _ := bytes.Cut(raw, []byte("\n"))
		r := csv.NewReader(bytes.NewReader(raw))
		r.Comma, r.FieldsPerRecord, r.LazyQuotes = sniffComma(first), -1, true
		recs, err := r.ReadAll()
		if err != nil {
			return nil, bad
		}
		grid = recs
	} else {
		f, err := excelize.OpenReader(bytes.NewReader(raw), excelize.Options{UnzipSizeLimit: formsUnzipLimit, UnzipXMLSizeLimit: formsUnzipLimit})
		if err != nil {
			return nil, bad
		}
		defer func() { _ = f.Close() }()
		sheets := f.GetSheetList()
		if len(sheets) == 0 {
			return nil, bad
		}
		if grid, err = f.GetRows(sheets[0]); err != nil {
			return nil, bad
		}
	}
	if len(grid) == 0 {
		return nil, bad
	}
	if len(grid) > FormsMaxRows+100 {
		return nil, apierr.New(http.StatusUnprocessableEntity, apierr.TooManyRows)
	}
	for _, row := range grid {
		if len(row) > formsMaxCols {
			return nil, bad
		}
	}
	return grid, nil
}

// sniffComma chọn dấu phân cách có nhiều nhất ở dòng đầu (Google Forms: phẩy; Excel tiếng Việt: chấm phẩy hoặc tab).
func sniffComma(line []byte) rune {
	best, n := ',', bytes.Count(line, []byte(","))
	for _, c := range []rune{';', '\t'} {
		if k := bytes.Count(line, []byte(string(c))); k > n {
			best, n = c, k
		}
	}
	return best
}

// ---- ánh xạ + chấm (thuần, không DB) --------------------------------------------------------------------------------------

type formsQ struct {
	col  int
	sn   snapshot
	key  json.RawMessage
	byID map[string]string // NormalizeShort(nội dung lựa chọn) → id lựa chọn
}

type formsRow struct {
	row               int
	ts                time.Time
	email, code       string
	correct, mismatch int
}

// cellDoc đổi ô của tệp thành câu trả lời theo loại câu. mismatch = ô có chữ nhưng không khớp lựa chọn nào (tính sai, không đoán).
func cellDoc(q formsQ, cell string) (doc []byte, mismatch bool) {
	cell = strings.TrimSpace(cell)
	var d answerDoc
	switch q.sn.Type {
	case "MCQ_SINGLE", "MCQ_MULTI":
		parts := []string{cell}
		if q.sn.Type == "MCQ_MULTI" {
			parts = strings.Split(cell, ";") // Forms ghi nhiều lựa chọn cách nhau bằng dấu chấm phẩy
		}
		d.OptionIDs = []string{}
		for _, p := range parts {
			if strings.TrimSpace(p) == "" {
				continue
			}
			id, ok := q.byID[quiz.NormalizeShort(p)]
			if !ok {
				return nil, true
			}
			d.OptionIDs = append(d.OptionIDs, id)
		}
	case "TRUE_FALSE":
		switch quiz.NormalizeShort(cell) {
		case "":
		case "đúng", "true", "yes", "đ", "t":
			v := true
			d.Value = &v
		case "sai", "false", "no", "s", "f":
			v := false
			d.Value = &v
		default:
			return nil, true
		}
	default: // SHORT (và đáp án nhập tay)
		d.Text = &cell
	}
	b, _ := json.Marshal(d)
	return b, false
}

// scoreRows chấm từng dòng dữ liệu bằng `grade` (Quiz Engine). Dòng không có định danh và không có chữ nào bị bỏ qua.
func scoreRows(grid [][]string, m FormsMapping, qs []formsQ) []formsRow {
	emailCol, _ := excelize.ColumnNameToNumber(strings.ToUpper(m.EmailCol))
	codeCol, _ := excelize.ColumnNameToNumber(strings.ToUpper(m.CodeCol))
	timeCol, _ := excelize.ColumnNameToNumber(strings.ToUpper(m.TimeCol))
	var out []formsRow
	for i := m.HeaderRow; i < len(grid); i++ {
		cols := grid[i]
		r := formsRow{row: i + 1, email: strings.ToLower(strings.TrimSpace(cellAt(cols, emailCol))), code: formsCode(cellAt(cols, codeCol))}
		if r.email == "" && r.code == "" {
			continue
		}
		if timeCol > 0 {
			r.ts = parseFormsTime(cellAt(cols, timeCol))
		}
		for _, q := range qs {
			doc, bad := cellDoc(q, cellAt(cols, q.col))
			if bad {
				r.mismatch++
				continue
			}
			if g := grade(q.sn, q.key, doc); g != nil && *g {
				r.correct++
			}
		}
		out = append(out, r)
	}
	return out
}

func cellAt(cols []string, col int) string {
	if col < 1 || col > len(cols) {
		return ""
	}
	return cols[col-1]
}

type resolver struct {
	byEmail map[string][]store.FormsRosterRow
	byCode  map[string][]store.FormsRosterRow
}

func newResolver(roster []store.FormsRosterRow) resolver {
	r := resolver{byEmail: map[string][]store.FormsRosterRow{}, byCode: map[string][]store.FormsRosterRow{}}
	for _, e := range roster {
		if e.Email != "" {
			r.byEmail[e.Email] = append(r.byEmail[e.Email], e)
		}
		// MSSV của dòng `CODE` là tự khai: không mở dữ liệu (luật AGENTS) — chỉ ROSTER / ADMIN đáng tin.
		if e.StudentCode != nil && e.JoinedVia != store.EnrollmentJoinedViaCODE {
			c := formsCode(*e.StudentCode)
			r.byCode[c] = append(r.byCode[c], e)
		}
	}
	return r
}

const (
	reasonNotFound = "NOT_FOUND"
	reasonMismatch = "MISMATCH"
)

// resolve: email đã xác minh hoặc MSSV trong danh sách lớp (không mơ hồ). Cả hai cùng có mà trỏ tới hai người khác nhau → không khớp.
func (r resolver) resolve(row formsRow) (st store.FormsRosterRow, reason string) {
	var e, c *store.FormsRosterRow
	if h := r.byEmail[row.email]; row.email != "" && len(h) == 1 {
		e = &h[0]
	}
	if h := r.byCode[row.code]; row.code != "" && len(h) == 1 {
		c = &h[0]
	}
	switch {
	case e != nil && c != nil && e.StudentID != c.StudentID:
		return st, reasonMismatch
	case e != nil:
		return *e, ""
	case c != nil:
		return *c, ""
	}
	return st, reasonNotFound
}

func scoreOfMax(correct, total int, maxScore decimal.Decimal) decimal.Decimal {
	if total == 0 {
		return decimal.Zero
	}
	return decimal.NewFromInt(int64(correct)).Mul(maxScore).DivRound(decimal.NewFromInt(int64(total)), 2)
}

// formsLine là một dòng ĐÃ khớp (đã chọn bản mới nhất) hoặc chưa khớp, sẵn để ghi.
type formsLine struct {
	row        formsRow
	student    store.FormsRosterRow
	reason     string // rỗng = đã khớp
	score      decimal.Decimal
	total      int
	identifier string
}

// analyzeForms khớp, khử trùng (dòng MỚI NHẤT theo dấu thời gian của Forms; không có dấu thời gian → dòng sau thắng) và tính điểm thang `maxScore`.
func analyzeForms(rows []formsRow, total int, maxScore decimal.Decimal, roster []store.FormsRosterRow) (matched, unmatched []formsLine, dups int) {
	rs := newResolver(roster)
	latest := map[uuid.UUID]formsLine{}
	for _, r := range rows {
		ident := r.email
		if ident == "" {
			ident = r.code
		}
		ln := formsLine{row: r, total: total, score: scoreOfMax(r.correct, total, maxScore), identifier: maskIdent(ident)}
		st, reason := rs.resolve(r)
		if reason != "" {
			ln.reason = reason
			unmatched = append(unmatched, ln)
			continue
		}
		ln.student = st
		if old, ok := latest[st.StudentID]; ok {
			dups++
			if r.ts.Before(old.row.ts) { // chỉ khi CÓ dấu thời gian cũ hơn; hoà / không có → dòng sau
				continue
			}
		}
		latest[st.StudentID] = ln
	}
	for _, ln := range latest {
		matched = append(matched, ln)
	}
	sort.Slice(matched, func(i, j int) bool { return matched[i].row.row < matched[j].row.row })
	return matched, unmatched, dups
}

// ---- dịch vụ -----------------------------------------------------------------------------------------------------------

// checkMapping kiểm ánh xạ so với tệp; `qs` lấy câu từ ngân hàng khi `answer_source=BANK`.
func (s *Service) checkMapping(ctx context.Context, q *store.Queries, courseID uuid.UUID, in FormsIn, grid [][]string) (FormsMapping, []formsQ, error) {
	m := *in.Mapping
	if m.HeaderRow < 1 || m.HeaderRow > 100 || m.HeaderRow >= len(grid) {
		return m, nil, fieldErr("mapping.header_row", "range", "Dòng tiêu đề nằm ngoài tệp.")
	}
	// cột là chữ cái ("A", "C") HOẶC tên ở dòng tiêu đề ("Email", ví dụ ở openapi); chuẩn hoá về chữ cái để phần còn lại dùng một dạng (L12).
	header := grid[m.HeaderRow-1]
	letters := func(c string) (string, bool) {
		c = strings.TrimSpace(c)
		if n, err := excelize.ColumnNameToNumber(strings.ToUpper(c)); err == nil && n <= formsMaxCols {
			return strings.ToUpper(c), true
		}
		for i, h := range header {
			if c != "" && strings.EqualFold(strings.TrimSpace(h), c) {
				name, _ := excelize.ColumnNumberToName(i + 1)
				return name, true
			}
		}
		return "", false
	}
	for field, c := range map[string]*string{"email_col": &m.EmailCol, "code_col": &m.CodeCol, "time_col": &m.TimeCol} {
		if *c == "" {
			continue
		}
		l, ok := letters(*c)
		if !ok {
			return m, nil, fieldErr("mapping."+field, "column", "Không thấy cột này trong tệp.")
		}
		*c = l
	}
	m.Questions = append([]FormsQuestion(nil), m.Questions...)
	if m.EmailCol == "" && m.CodeCol == "" {
		return m, nil, fieldErr("mapping.email_col", "identifier_required", "Chọn cột Email hoặc MSSV.")
	}
	if len(m.Questions) == 0 || len(m.Questions) > FormsMaxQuestions {
		return m, nil, fieldErr("mapping.questions", "count", fmt.Sprintf("Chọn từ 1 đến %d câu.", FormsMaxQuestions))
	}
	if in.AnswerSource != "BANK" && in.AnswerSource != "MANUAL" {
		return m, nil, fieldErr("answer_source", "enum", "Nguồn đáp án là BANK hoặc MANUAL.")
	}
	qs := make([]formsQ, len(m.Questions))
	seen := map[int]bool{}
	var ids []string
	for i, mq := range m.Questions {
		f := fmt.Sprintf("mapping.questions[%d]", i)
		l, ok := letters(mq.Col)
		n, _ := excelize.ColumnNameToNumber(l)
		if !ok || seen[n] {
			return m, nil, fieldErr(f+".col", "column", "Không thấy cột này trong tệp, hoặc bị chọn hai lần.")
		}
		seen[n] = true
		m.Questions[i].Col = l
		qs[i].col = n
		if in.AnswerSource == "BANK" {
			if mq.QuestionID == nil {
				return m, nil, fieldErr(f+".question_id", "required", "Chọn câu hỏi của ngân hàng cho cột này.")
			}
			ids = append(ids, mq.QuestionID.String())
			continue
		}
		accepted := make([]string, 0, len(mq.Accepted))
		for _, a := range mq.Accepted {
			if quiz.NormalizeShort(a) != "" {
				accepted = append(accepted, a)
			}
		}
		if len(accepted) == 0 {
			return m, nil, fieldErr(f+".accepted", "required", "Nhập đáp án đúng cho cột này.")
		}
		qs[i].sn = snapshot{Type: "SHORT"}
		qs[i].key, _ = json.Marshal(map[string]any{"accepted": accepted})
	}
	if in.AnswerSource == "BANK" {
		if err := s.bankQuestions(ctx, q, courseID, ids, qs); err != nil {
			return m, nil, err
		}
	}
	return m, qs, nil
}

func (s *Service) bankQuestions(ctx context.Context, q *store.Queries, courseID uuid.UUID, ids []string, qs []formsQ) error {
	rows, err := q.PracticeQuestionsByIDs(ctx, store.PracticeQuestionsByIDsParams{CourseID: courseID, Ids: ids})
	if err != nil {
		return fmt.Errorf("practice: đọc câu hỏi: %w", err)
	}
	opts, err := q.PracticeOptionsByQuestions(ctx, store.PracticeOptionsByQuestionsParams{CourseID: courseID, Ids: ids})
	if err != nil {
		return fmt.Errorf("practice: đọc đáp án: %w", err)
	}
	byQ := make(map[uuid.UUID]store.PracticeQuestionsByIDsRow, len(rows))
	for _, r := range rows {
		byQ[r.ID] = r
	}
	optsOf := map[uuid.UUID][]store.PracticeOptionsByQuestionsRow{}
	for _, o := range opts {
		optsOf[o.QuestionID] = append(optsOf[o.QuestionID], o)
	}
	for i := range qs {
		id := uuid.MustParse(ids[i])
		row, ok := byQ[id]
		f := fmt.Sprintf("mapping.questions[%d].question_id", i)
		switch {
		case !ok:
			return fieldErr(f, "not_found", "Không tìm thấy câu hỏi trong ngân hàng.")
		case row.Type == store.QuestionTypeESSAY || row.Type == store.QuestionTypeCODE:
			return fieldErr(f, "type", "Loại câu này không chấm tự động được.")
		}
		sn := snapshot{Type: string(row.Type), Options: []snapOption{}}
		qs[i].byID = map[string]string{}
		for _, o := range optsOf[id] {
			sn.Options = append(sn.Options, snapOption{ID: o.ID, Body: o.Body})
			if _, dup := qs[i].byID[quiz.NormalizeShort(o.Body)]; !dup {
				qs[i].byID[quiz.NormalizeShort(o.Body)] = o.ID.String()
			}
		}
		qs[i].sn, qs[i].key = sn, row.AnswerKey
		if len(qs[i].key) == 0 {
			qs[i].key = json.RawMessage(`{}`)
		}
	}
	return nil
}

func previewKey(upload uuid.UUID) string { return appredis.Key("forms", "preview", upload.String()) }

// Preview đọc tệp và trả cột + 20 dòng đầu (chưa ánh xạ), hoặc bảng điểm chưa ghi (đã ánh xạ). Đánh dấu tệp "đã xem trước" cho bước xác nhận.
func (s *Service) Preview(ctx context.Context, actor, courseID uuid.UUID, in FormsIn) (FormsPreviewOut, error) {
	tk, err := s.ticket(ctx, actor, courseID, in.UploadID)
	if err != nil {
		return FormsPreviewOut{}, err
	}
	f, err := s.load(ctx, tk)
	if err != nil {
		return FormsPreviewOut{}, err
	}
	hr := 1
	if in.Mapping != nil && in.Mapping.HeaderRow > 0 {
		hr = in.Mapping.HeaderRow
	}
	out := FormsPreviewOut{Columns: []FormsColumn{}, Rows: [][]string{}}
	if hr <= len(f.grid) {
		for i, h := range f.grid[hr-1] {
			col, _ := excelize.ColumnNumberToName(i + 1)
			out.Columns = append(out.Columns, FormsColumn{Col: col, Header: clip(h)})
		}
		out.Suggested = suggestForms(hr, f.grid[hr-1])
		for i := hr; i < len(f.grid) && i < hr+formsPreviewRows; i++ {
			row := make([]string, len(f.grid[i]))
			for j, c := range f.grid[i] {
				row[j] = clip(c)
			}
			out.Rows = append(out.Rows, row)
		}
		out.RowsTotal = max(len(f.grid)-hr, 0)
	} else {
		out.Suggested = FormsMapping{HeaderRow: hr, Questions: []FormsQuestion{}}
	}
	if in.Mapping != nil {
		if in.ItemID == nil {
			return out, fieldErr("item_id", "required", "Chọn cột điểm của sổ điểm.")
		}
		q := store.New(s.Pool)
		item, err := s.formsItem(ctx, q, courseID, *in.ItemID)
		if err != nil {
			return out, err
		}
		m, qs, err := s.checkMapping(ctx, q, courseID, in, f.grid)
		if err != nil {
			return out, err
		}
		roster, err := q.FormsRoster(ctx, courseID)
		if err != nil {
			return out, fmt.Errorf("practice: danh sách lớp: %w", err)
		}
		matched, unmatched, dups := analyzeForms(scoreRows(f.grid, m, qs), len(qs), item.MaxScore, roster)
		res := formsResultOf(matched, unmatched, dups)
		out.Result = &res
	}
	pipe := s.Redis.TxPipeline()
	pipe.Set(ctx, previewKey(in.UploadID), actor.String(), formsTicketTTL)
	pipe.Expire(ctx, appredis.Key("upload", in.UploadID.String()), formsTicketTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return out, fmt.Errorf("practice: ghi dấu xem trước: %w", err)
	}
	return out, nil
}

func clip(s string) string {
	if r := []rune(s); len(r) > formsCellMax {
		return string(r[:formsCellMax])
	}
	return s
}

// formsItem: cột điểm phải thuộc công thức ĐÃ XÁC NHẬN và là loại nhập tay (MANUAL); khác → 404 / 409.
func (s *Service) formsItem(ctx context.Context, q *store.Queries, courseID, itemID uuid.UUID) (store.GradeItemForEditRow, error) {
	item, err := q.GradeItemForEdit(ctx, store.GradeItemForEditParams{CourseID: courseID, ItemID: itemID})
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && item.SchemeStatus != store.GradeSchemeStatusCONFIRMED) {
		return item, notFound()
	}
	if err != nil {
		return item, fmt.Errorf("practice: đọc cột điểm: %w", err)
	}
	if item.Source != store.GradeItemSourceMANUAL {
		return item, apierr.New(http.StatusConflict, apierr.ItemNotEditable)
	}
	return item, nil
}

func formsResultOf(matched, unmatched []formsLine, dups int) FormsResult {
	res := FormsResult{Matched: len(matched), Unmatched: len(unmatched), Duplicates: dups, Items: []FormsScored{}, Rows: []FormsUnmatched{}}
	for _, l := range matched {
		code := ""
		if l.student.StudentCode != nil {
			code = *l.student.StudentCode
		}
		res.Items = append(res.Items, FormsScored{Student: FormsStudent{ID: l.student.StudentID, FullName: l.student.FullName, StudentCode: code},
			Correct: l.row.correct, Total: l.total, Score: l.score.String(), Mismatch: l.row.mismatch})
	}
	for _, l := range unmatched {
		res.Rows = append(res.Rows, FormsUnmatched{Row: l.row.row, Identifier: l.identifier, Reason: l.reason, Correct: l.row.correct, Total: l.total, Score: l.score.String()})
	}
	return res
}

// Import ghi điểm (xác nhận). Đòi `preview` trước (409 PREVIEW_REQUIRED); sổ LOCKED → 409; cùng tệp (cùng hash) vào cùng cột → trả lại lô cũ, không ghi gì.
// Điểm vào `grade_entries(source='QUIZ')` của cột MANUAL đã chọn, KHÔNG đặt `published_at` (công bố chỉ qua `grade-items/{id}/publish`).
func (s *Service) Import(ctx context.Context, actor, courseID uuid.UUID, in FormsIn) (FormsResult, error) {
	if in.Mapping == nil || in.ItemID == nil {
		return FormsResult{}, fieldErr("mapping", "required", "Cần ánh xạ cột và cột điểm.")
	}
	tk, err := s.ticket(ctx, actor, courseID, in.UploadID)
	if err != nil {
		return FormsResult{}, err
	}
	if v, err := s.Redis.Get(ctx, previewKey(in.UploadID)).Result(); err != nil || v != actor.String() {
		return FormsResult{}, apierr.New(http.StatusConflict, apierr.PreviewRequired)
	}
	q := store.New(s.Pool)
	item, err := s.formsItem(ctx, q, courseID, *in.ItemID)
	if err != nil {
		return FormsResult{}, err
	}
	f, err := s.load(ctx, tk)
	if err != nil {
		return FormsResult{}, err
	}
	m, qs, err := s.checkMapping(ctx, q, courseID, in, f.grid)
	if err != nil {
		return FormsResult{}, err
	}
	roster, err := q.FormsRoster(ctx, courseID)
	if err != nil {
		return FormsResult{}, fmt.Errorf("practice: danh sách lớp: %w", err)
	}
	matched, unmatched, dups := analyzeForms(scoreRows(f.grid, m, qs), len(qs), item.MaxScore, roster)

	var res FormsResult
	err = s.tx(ctx, func(tq *store.Queries, _ pgx.Tx) error {
		gb, err := tq.GradebookStatusShare(ctx, courseID)
		if errors.Is(err, pgx.ErrNoRows) {
			return apierr.New(http.StatusConflict, apierr.SchemeNotConfirmed)
		}
		if err != nil {
			return fmt.Errorf("practice: trạng thái sổ điểm: %w", err)
		}
		if gb == store.GradebookStatusLOCKED {
			return apierr.New(http.StatusConflict, apierr.GradebookLocked)
		}
		if id, err := tq.FormsBatchByHash(ctx, store.FormsBatchByHashParams{CourseID: courseID, ItemID: *in.ItemID, FileHash: f.hash}); err == nil {
			res, err = s.replay(ctx, tq, courseID, id)
			return err
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("practice: tìm lô cũ: %w", err)
		}
		mj, _ := json.Marshal(struct {
			FormsMapping
			AnswerSource string `json:"answer_source"`
		}{m, in.AnswerSource})
		batch, err := tq.FormsBatchInsert(ctx, store.FormsBatchInsertParams{CourseID: courseID, ItemID: *in.ItemID, FileHash: f.hash, BlobKey: f.blobKey, Mapping: mj,
			Matched: int32(len(matched)), Unmatched: int32(len(unmatched)), CreatedBy: actor}) //nolint:gosec // ≤ 5000 dòng
		if err != nil {
			return fmt.Errorf("practice: ghi lô Forms: %w", err)
		}
		lines := append(append([]formsLine{}, matched...), unmatched...)
		arg := store.FormsRowsInsertParams{CourseID: courseID, BatchID: batch}
		for _, l := range lines {
			sid := ""
			if l.reason == "" {
				sid = l.student.StudentID.String()
			}
			arg.RowNos, arg.StudentIds, arg.Idents = append(arg.RowNos, int32(l.row.row)), append(arg.StudentIds, sid), append(arg.Idents, l.identifier)                //nolint:gosec // ≤ 5100
			arg.Corrects, arg.Totals, arg.Scores = append(arg.Corrects, int32(l.row.correct)), append(arg.Totals, int32(l.total)), append(arg.Scores, l.score.String()) //nolint:gosec // ≤ 100
		}
		if len(lines) > 0 {
			if err := tq.FormsRowsInsert(ctx, arg); err != nil {
				return fmt.Errorf("practice: ghi dòng Forms: %w", err)
			}
		}
		if len(matched) > 0 {
			ids, scores := make([]string, len(matched)), make([]string, len(matched))
			for i, l := range matched {
				ids[i], scores[i] = l.student.StudentID.String(), l.score.String()
			}
			w, err := tq.GradeImportApply(ctx, store.GradeImportApplyParams{CourseID: courseID, ItemID: *in.ItemID, ItemKey: item.Key, ActorID: actor, StudentIds: ids, Scores: scores, Source: store.GradeEntrySourceQUIZ})
			if err != nil {
				return fmt.Errorf("practice: ghi điểm Forms: %w", err)
			}
			res.Written = int(w.Inserted + w.Updated)
		}
		after, _ := json.Marshal(map[string]any{"batch_id": batch, "item_id": in.ItemID, "item_key": item.Key, "file_hash": f.hash, "matched": len(matched), "unmatched": len(unmatched), "duplicates": dups, "written": res.Written})
		if _, err := tq.InsertAuditLog(ctx, store.InsertAuditLogParams{CourseID: &courseID, ActorID: &actor, Entity: "forms_imports", EntityID: batch.String(), Action: "import", After: after}); err != nil {
			return fmt.Errorf("practice: audit lô Forms: %w", err) // fail-closed
		}
		w := res.Written
		res = formsResultOf(matched, unmatched, dups)
		res.BatchID, res.Written = &batch, w
		return nil
	})
	if isUnique(err) { // hai lần xác nhận cùng tệp chạy song song: thắng một, lần kia đọc lại lô đã ghi
		err = s.tx(ctx, func(tq *store.Queries, _ pgx.Tx) error {
			id, e := tq.FormsBatchByHash(ctx, store.FormsBatchByHashParams{CourseID: courseID, ItemID: *in.ItemID, FileHash: f.hash})
			if e != nil {
				return e
			}
			res, e = s.replay(ctx, tq, courseID, id)
			return e
		})
	}
	return res, err
}

// replay đọc lại lô đã ghi (cùng tệp nhập lần nữa): không đổi điểm, không audit thừa.
func (s *Service) replay(ctx context.Context, q *store.Queries, courseID, batch uuid.UUID) (FormsResult, error) {
	rows, err := q.FormsRowsByBatch(ctx, store.FormsRowsByBatchParams{CourseID: courseID, BatchID: batch})
	if err != nil {
		return FormsResult{}, fmt.Errorf("practice: đọc lô Forms: %w", err)
	}
	res := FormsResult{Items: []FormsScored{}, Rows: []FormsUnmatched{}, BatchID: &batch, Replayed: true}
	for _, r := range rows {
		if r.StudentID == nil {
			res.Unmatched++
			res.Rows = append(res.Rows, FormsUnmatched{Row: int(r.RowNo), Identifier: r.IdentifierMasked, Reason: reasonNotFound, Correct: int(r.Correct), Total: int(r.Total), Score: r.Score})
			continue
		}
		res.Matched++
		res.Items = append(res.Items, FormsScored{Student: FormsStudent{ID: *r.StudentID, FullName: r.FullName, StudentCode: r.StudentCode}, Correct: int(r.Correct), Total: int(r.Total), Score: r.Score})
	}
	return res, nil
}
