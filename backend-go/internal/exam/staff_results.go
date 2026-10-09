package exam

import (
	"bufio"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/store"
)

// ---- kết quả cho Staff (US-PE-08 AC9–AC12) ---------------------------------------------------------------------------------------------

// ResultsProgress: đếm theo trạng thái. `not_started` = chưa có lượt khi bài chưa đóng; `absent` = không có lượt khi bài đã đóng.
type ResultsProgress struct {
	NotStarted int `json:"not_started"`
	InProgress int `json:"in_progress"`
	Grading    int `json:"grading"`
	Graded     int `json:"graded"`
	Absent     int `json:"absent"`
}

// ResultStudent là danh tính trong bảng điểm (Staff được thấy; sinh viên thì không có DTO này).
type ResultStudent struct {
	ID          uuid.UUID `json:"id"`
	FullName    string    `json:"full_name"`
	StudentCode string    `json:"student_code"`
}

// ResultFlags chỉ có với Giảng viên (TA không có khoá `flags`).
type ResultFlags struct {
	Similarity int `json:"similarity"`
	TabHidden  int `json:"tab_hidden"`
	Paste      int `json:"paste"`
}

// ResultRow là một hàng bảng điểm.
type ResultRow struct {
	AttemptID    *uuid.UUID    `json:"attempt_id"`
	Student      ResultStudent `json:"student"`
	Status       string        `json:"status"`
	AutoScore    *string       `json:"auto_score"`
	Score        *string       `json:"score"`
	Adjusted     bool          `json:"adjusted"`
	SubmittedAt  *time.Time    `json:"submitted_at"`
	SubmitReason *string       `json:"submit_reason"`
	Flags        *ResultFlags  `json:"flags,omitempty"`
}

// ResultsPage là phản hồi `GET …/results`.
type ResultsPage struct {
	Progress ResultsProgress `json:"progress"`
	Items    []ResultRow     `json:"items"`
}

// ResultsFilter: `status` ∈ ABSENT|IN_PROGRESS|GRADING|GRADED (rỗng = tất cả); `sort` ∈ score|name (mặc định score).
type ResultsFilter struct {
	Status  string
	Sort    string
	Q       string
	Teacher bool
}

// scoresVisible: điểm chỉ hiện khi bài đã đóng (trước đó lượt trắc nghiệm đã GRADED sớm nhưng điểm chưa được lộ cho ai ngoài chính trình xem trước).
func scoresVisible(e store.Exam) bool {
	return e.Status == store.ExamStatusCLOSED || e.Status == store.ExamStatusPUBLISHED
}

func closed(e store.Exam) bool { return scoresVisible(e) }

func (s *Service) examOf(ctx context.Context, q *store.Queries, courseID, examID uuid.UUID) (store.Exam, error) {
	e, err := q.ExamGet(ctx, store.ExamGetParams{CourseID: courseID, ID: examID})
	if errors.Is(err, pgx.ErrNoRows) {
		return store.Exam{}, notFound()
	}
	if err != nil {
		return store.Exam{}, fmt.Errorf("exam: đọc bài thi: %w", err)
	}
	if e.Status == store.ExamStatusDRAFT {
		return store.Exam{}, notFound() // chưa lên lịch: chưa có kết quả nào
	}
	return e, nil
}

func rowStatus(e store.Exam, r store.ExamResultRowsRow) string {
	if r.AttemptStatus == nil {
		if closed(e) {
			return "ABSENT"
		}
		return "NOT_STARTED"
	}
	return string(*r.AttemptStatus)
}

// ResultsList: bảng điểm của lớp (MỘT truy vấn cho mọi cỡ lớp). Sắp xếp / con trỏ ở đây: con trỏ là `student_id` của hàng cuối, vị trí tìm lại trên danh sách đã sắp (≤ 1.000).
func (s *Service) ResultsList(ctx context.Context, courseID, examID uuid.UUID, f ResultsFilter, after *uuid.UUID, limit int) (ResultsPage, []ResultRow, error) {
	q := store.New(s.Pool)
	e, err := s.examOf(ctx, q, courseID, examID)
	if err != nil {
		return ResultsPage{}, nil, err
	}
	all, err := q.ExamResultRows(ctx, store.ExamResultRowsParams{CourseID: courseID, ExamID: examID, Status: f.Status, WithFlags: f.Teacher})
	if err != nil {
		return ResultsPage{}, nil, fmt.Errorf("exam: bảng điểm: %w", err)
	}
	rows := make([]ResultRow, 0, len(all))
	type key struct {
		score decimal.Decimal
		has   bool
		name  string
	}
	keys := make(map[uuid.UUID]key, len(all))
	var pr ResultsProgress
	needle := strings.ToLower(strings.TrimSpace(f.Q))
	for _, r := range all {
		if needle != "" && !strings.Contains(strings.ToLower(r.FullName), needle) && !strings.Contains(strings.ToLower(r.StudentCode), needle) { // chỉ lọc hiển thị
			continue
		}
		st := rowStatus(e, r)
		switch st {
		case "NOT_STARTED":
			pr.NotStarted++
		case "ABSENT":
			pr.Absent++
		case "IN_PROGRESS":
			pr.InProgress++
		case "GRADING":
			pr.Grading++
		case "GRADED":
			pr.Graded++
		}
		row := ResultRow{AttemptID: r.AttemptID, Student: ResultStudent{ID: r.StudentID, FullName: r.FullName, StudentCode: r.StudentCode}, Status: st, SubmittedAt: r.SubmittedAt, Adjusted: r.AdjustedScore.Valid}
		if r.SubmitReason != nil {
			row.SubmitReason = new(string(*r.SubmitReason))
		}
		k := key{name: strings.ToLower(r.FullName)}
		if scoresVisible(e) && st == "GRADED" {
			if r.AutoScore.Valid {
				row.AutoScore = new(r.AutoScore.Decimal.StringFixed(2))
			}
			sc := r.AdjustedScore
			if !sc.Valid {
				sc = r.AutoScore
			}
			if sc.Valid {
				row.Score = new(sc.Decimal.StringFixed(2))
				k.score, k.has = sc.Decimal, true
			}
		}
		if f.Teacher {
			row.Flags = &ResultFlags{Similarity: int(r.SimFlags), TabHidden: int(r.TabHidden), Paste: int(r.Paste)}
		}
		keys[r.StudentID] = k
		rows = append(rows, row)
	}
	byName := f.Sort == "name"
	slices.SortFunc(rows, func(a, b ResultRow) int {
		ka, kb := keys[a.Student.ID], keys[b.Student.ID]
		if byName {
			if c := strings.Compare(ka.name, kb.name); c != 0 {
				return c
			}
		} else {
			if ka.has != kb.has { // chưa có điểm xếp cuối
				if ka.has {
					return -1
				}
				return 1
			}
			if c := kb.score.Cmp(ka.score); c != 0 {
				return c
			}
			if c := strings.Compare(ka.name, kb.name); c != 0 {
				return c
			}
		}
		return strings.Compare(a.Student.ID.String(), b.Student.ID.String())
	})
	if f.Status != "" || f.Q != "" { // lọc ở SQL nhưng tiến độ phải tính trên cả lớp
		full, err := q.ExamResultRows(ctx, store.ExamResultRowsParams{CourseID: courseID, ExamID: examID, WithFlags: false})
		if err != nil {
			return ResultsPage{}, nil, fmt.Errorf("exam: tiến độ: %w", err)
		}
		pr = ResultsProgress{}
		for _, r := range full {
			switch rowStatus(e, r) {
			case "NOT_STARTED":
				pr.NotStarted++
			case "ABSENT":
				pr.Absent++
			case "IN_PROGRESS":
				pr.InProgress++
			case "GRADING":
				pr.Grading++
			case "GRADED":
				pr.Graded++
			}
		}
	}
	start := 0
	if after != nil {
		i := slices.IndexFunc(rows, func(r ResultRow) bool { return r.Student.ID == *after })
		if i < 0 {
			return ResultsPage{}, nil, apierr.New(422, apierr.InvalidCursor)
		}
		start = i + 1
	}
	end := min(start+limit, len(rows))
	return ResultsPage{Progress: pr}, rows[start:end], nil
}

// ---- chi tiết một lượt (Staff) -----------------------------------------------------------------------------------------------------------

// StaffTestResult là verdict MỘT test của một bản nộp (kể cả test ẩn — chỉ Staff).
type StaffTestResult struct {
	Position int    `json:"position"`
	IsSample bool   `json:"is_sample"`
	Verdict  string `json:"verdict"`
	TimeMS   int    `json:"time_ms"`
	MemoryKB int    `json:"memory_kb"`
}

// StaffSubmission là một bản nộp trong lịch sử của lượt.
type StaffSubmission struct {
	ID         uuid.UUID         `json:"id"`
	ItemID     uuid.UUID         `json:"item_id"`
	Status     string            `json:"status"`
	Verdict    *string           `json:"verdict"`
	Language   string            `json:"language"`
	Source     string            `json:"source"`
	CreatedAt  time.Time         `json:"created_at"`
	CompileOK  *bool             `json:"compile_ok"`
	CompileLog *string           `json:"compile_log"`
	Tests      []StaffTestResult `json:"tests"`
}

// StaffAdjust là điều chỉnh tay (khi có).
type StaffAdjust struct {
	Score  string    `json:"score"`
	Reason string    `json:"reason"`
	At     time.Time `json:"at"`
}

// StaffResultDetail là phản hồi `GET …/results/{aid}`.
type StaffResultDetail struct {
	AttemptID    uuid.UUID         `json:"attempt_id"`
	Student      ResultStudent     `json:"student"`
	Status       string            `json:"status"`
	Version      int               `json:"version"`
	AutoScore    *string           `json:"auto_score"`
	Score        *string           `json:"score"`
	Adjust       *StaffAdjust      `json:"adjust"`
	SubmittedAt  *time.Time        `json:"submitted_at"`
	SubmitReason *string           `json:"submit_reason"`
	Items        []ResultItemView  `json:"items"`
	Submissions  []StaffSubmission `json:"submissions"`
	Integrity    *IntegritySummary `json:"integrity,omitempty"`
	Appeal       *AppealView       `json:"appeal"`
}

// ResultDetail: `GET …/results/{aid}` (Staff). `integrity` chỉ với Giảng viên.
func (s *Service) ResultDetail(ctx context.Context, courseID, examID, attemptID uuid.UUID, teacher bool) (StaffResultDetail, error) {
	q := store.New(s.Pool)
	e, err := s.examOf(ctx, q, courseID, examID)
	if err != nil {
		return StaffResultDetail{}, err
	}
	h, err := q.AttemptStaffHead(ctx, store.AttemptStaffHeadParams{CourseID: courseID, ExamID: examID, ID: attemptID})
	if errors.Is(err, pgx.ErrNoRows) {
		return StaffResultDetail{}, notFound()
	}
	if err != nil {
		return StaffResultDetail{}, fmt.Errorf("exam: đọc lượt: %w", err)
	}
	a := store.ExamAttempt{ID: h.ID, CourseID: h.CourseID, ExamID: h.ExamID, StudentID: h.StudentID, Status: h.Status, AutoScore: h.AutoScore, AdjustedScore: h.AdjustedScore, Breakdown: h.Breakdown}
	d := StaffResultDetail{AttemptID: h.ID, Student: ResultStudent{ID: h.StudentID, FullName: h.FullName, StudentCode: h.StudentCode}, Status: string(h.Status), Version: int(h.Version), SubmittedAt: h.SubmittedAt}
	if h.SubmitReason != nil {
		d.SubmitReason = new(string(*h.SubmitReason))
	}
	if scoresVisible(e) && h.Status == store.AttemptStatusGRADED {
		if h.AutoScore.Valid {
			d.AutoScore = new(h.AutoScore.Decimal.StringFixed(2))
		}
		if sc, ok := officialScore(a); ok {
			d.Score = new(sc.StringFixed(2))
		}
		if h.AdjustedScore.Valid && h.AdjustedAt != nil {
			d.Adjust = &StaffAdjust{Score: h.AdjustedScore.Decimal.StringFixed(2), Reason: derefStr(h.AdjustedReason), At: *h.AdjustedAt}
		}
	}
	if d.Items, err = s.resultItems(ctx, q, e, a, true); err != nil {
		return StaffResultDetail{}, err
	}
	subs, err := q.AttemptSubmitHistory(ctx, store.AttemptSubmitHistoryParams{CourseID: courseID, AttemptID: attemptID})
	if err != nil {
		return StaffResultDetail{}, fmt.Errorf("exam: lịch sử bản nộp: %w", err)
	}
	d.Submissions = make([]StaffSubmission, 0, len(subs))
	for _, sb := range subs {
		v := StaffSubmission{ID: sb.ID, ItemID: sb.ItemID, Status: string(sb.Status), Language: sb.Language, Source: sb.Source, CreatedAt: sb.CreatedAt, CompileOK: sb.CompileOk, CompileLog: sb.CompileLog, Tests: []StaffTestResult{}}
		if sb.Verdict != nil {
			v.Verdict = new(string(*sb.Verdict))
		}
		var rs []storedResult
		_ = json.Unmarshal(sb.Results, &rs)
		for _, r := range rs {
			v.Tests = append(v.Tests, StaffTestResult{Position: r.Position, IsSample: r.IsSample, Verdict: r.Verdict, TimeMS: r.TimeMS, MemoryKB: r.MemoryKB})
		}
		d.Submissions = append(d.Submissions, v)
	}
	if teacher {
		sum, err := s.Summary(ctx, courseID, attemptID)
		if err != nil {
			return StaffResultDetail{}, err
		}
		d.Integrity = &sum
	}
	ap, err := q.AppealByAttempt(ctx, store.AppealByAttemptParams{CourseID: courseID, AttemptID: attemptID})
	switch {
	case err == nil:
		v := appealView(ap)
		d.Appeal = &v
	case !errors.Is(err, pgx.ErrNoRows):
		return StaffResultDetail{}, fmt.Errorf("exam: đọc phúc khảo: %w", err)
	}
	return d, nil
}

func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// ---- thống kê (AC11) -------------------------------------------------------------------------------------------------------------------

// StatsBucket là một khoảng điểm nửa mở `[from, to)` (khoảng cuối đóng).
type StatsBucket struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Count int    `json:"count"`
}

// StatsHard là câu trắc nghiệm có tỉ lệ đúng thấp.
type StatsHard struct {
	ItemID      uuid.UUID `json:"item_id"`
	Title       string    `json:"title"`
	CorrectRate string    `json:"correct_rate"`
}

// StatsCode là thống kê một bài code.
type StatsCode struct {
	ItemID    uuid.UUID `json:"item_id"`
	Title     string    `json:"title"`
	MeanRatio string    `json:"mean_ratio"`
	CERate    string    `json:"ce_rate"`
}

// Stats là phản hồi `GET …/stats`. Không định danh.
type Stats struct {
	Distribution []StatsBucket `json:"distribution"`
	Mean         string        `json:"mean"`
	Median       string        `json:"median"`
	Hardest      []StatsHard   `json:"hardest"`
	Code         []StatsCode   `json:"code"`
}

// ExamStats tính từ lượt GRADED, điểm chính thức. Giá trị decimal 2 chữ số chỉ để đọc.
func (s *Service) ExamStats(ctx context.Context, courseID, examID uuid.UUID) (Stats, error) {
	q := store.New(s.Pool)
	e, err := s.examOf(ctx, q, courseID, examID)
	if err != nil {
		return Stats{}, err
	}
	atts, err := q.ExamScoredAttempts(ctx, store.ExamScoredAttemptsParams{CourseID: courseID, ExamID: examID})
	if err != nil {
		return Stats{}, fmt.Errorf("exam: lượt đã chấm: %w", err)
	}
	metas, err := q.ExamItemsMeta(ctx, store.ExamItemsMetaParams{CourseID: courseID, ExamID: examID})
	if err != nil {
		return Stats{}, fmt.Errorf("exam: câu của bài: %w", err)
	}
	st := Stats{Distribution: make([]StatsBucket, 10), Mean: "0.00", Median: "0.00", Hardest: []StatsHard{}, Code: []StatsCode{}}
	w := e.MaxScore.Div(decimal.NewFromInt(10))
	for i := range st.Distribution {
		st.Distribution[i] = StatsBucket{From: w.Mul(decimal.NewFromInt(int64(i))).StringFixed(2), To: w.Mul(decimal.NewFromInt(int64(i + 1))).StringFixed(2)}
	}
	type tally struct{ right, n int }
	mc := map[uuid.UUID]*tally{}
	type codeT struct {
		ratio decimal.Decimal
		n, ce int
	}
	cd := map[uuid.UUID]*codeT{}
	var scores []decimal.Decimal
	sum := decimal.Zero
	for _, a := range atts {
		sc := a.AutoScore
		if a.AdjustedScore.Valid {
			sc = a.AdjustedScore
		}
		if !sc.Valid {
			continue
		}
		scores = append(scores, sc.Decimal)
		sum = sum.Add(sc.Decimal)
		idx := 9
		if w.IsPositive() {
			idx = min(int(sc.Decimal.Div(w).Floor().IntPart()), 9)
		}
		st.Distribution[max(idx, 0)].Count++
		var rows []breakdownRow
		if err := json.Unmarshal(a.Breakdown, &rows); err != nil {
			continue
		}
		for _, r := range rows {
			if r.Void {
				continue
			}
			mx, _ := decimal.NewFromString(r.Max)
			er, _ := decimal.NewFromString(r.Earned)
			if r.Code != nil {
				t := cd[r.ItemID]
				if t == nil {
					t = &codeT{}
					cd[r.ItemID] = t
				}
				t.n++
				if !r.Code.CompileOK {
					t.ce++
				}
				if mx.IsPositive() {
					t.ratio = t.ratio.Add(er.DivRound(mx, 16))
				}
				continue
			}
			t := mc[r.ItemID]
			if t == nil {
				t = &tally{}
				mc[r.ItemID] = t
			}
			t.n++
			if er.Equal(mx) {
				t.right++
			}
		}
	}
	if n := len(scores); n > 0 {
		cnt := decimal.NewFromInt(int64(n))
		st.Mean = sum.DivRound(cnt, 16).StringFixed(2)
		slices.SortFunc(scores, func(a, b decimal.Decimal) int { return a.Cmp(b) })
		med := scores[n/2]
		if n%2 == 0 {
			med = scores[n/2-1].Add(scores[n/2]).DivRound(decimal.NewFromInt(2), 16)
		}
		st.Median = med.StringFixed(2)
	}
	type hard struct {
		StatsHard
		rate decimal.Decimal
		pos  int16
	}
	var hs []hard
	for _, m := range metas {
		if m.Type == store.QuestionTypeCODE {
			if t := cd[m.ItemID]; t != nil && t.n > 0 {
				n := decimal.NewFromInt(int64(t.n))
				st.Code = append(st.Code, StatsCode{ItemID: m.ItemID, Title: m.Title, MeanRatio: t.ratio.DivRound(n, 16).StringFixed(2), CERate: decimal.NewFromInt(int64(t.ce)).DivRound(n, 16).StringFixed(2)})
			}
			continue
		}
		if t := mc[m.ItemID]; t != nil && t.n > 0 {
			rate := decimal.NewFromInt(int64(t.right)).DivRound(decimal.NewFromInt(int64(t.n)), 16)
			hs = append(hs, hard{StatsHard{ItemID: m.ItemID, Title: m.Title, CorrectRate: rate.StringFixed(2)}, rate, m.Position})
		}
	}
	slices.SortFunc(hs, func(a, b hard) int {
		if c := a.rate.Cmp(b.rate); c != 0 {
			return c
		}
		return int(a.pos) - int(b.pos)
	})
	for _, h := range hs[:min(5, len(hs))] {
		st.Hardest = append(st.Hardest, h.StatsHard)
	}
	return st, nil
}

// ---- CSV (AC12) ---------------------------------------------------------------------------------------------------------------------------

// ResultsConfig: giới hạn của kết quả. Số 0 = mặc định (CSV tối đa 5.000 dòng — vượt thì báo lỗi, KHÔNG cắt im lặng, SRS 4.8.6; tính lại đồng bộ ≤ 200 lượt, SRS 4.8.4).
type ResultsConfig struct {
	CSVMaxRows    int
	RecomputeSync int
}

func (c ResultsConfig) csvMax() int {
	if c.CSVMaxRows <= 0 {
		return 5000
	}
	return c.CSVMaxRows
}

func (c ResultsConfig) recomputeSync() int {
	if c.RecomputeSync <= 0 {
		return 200
	}
	return c.RecomputeSync
}

// csvSafe chống chèn công thức: ô bắt đầu bằng `= + - @` (và tab / CR) thêm `'` phía trước.
func csvSafe(v string) string {
	if v != "" && strings.ContainsRune("=+-@\t\r", rune(v[0])) {
		return "'" + v
	}
	return v
}

// csvNum đổi số thập phân sang dấu phẩy (mở thẳng bằng Excel tiếng Việt).
func csvNum(d decimal.Decimal) string { return strings.Replace(d.StringFixed(2), ".", ",", 1) }

// ResultsCSV ghi bảng điểm ra `w` theo luồng (từng dòng một, `Flush` định kỳ). Dữ liệu được kiểm trước khi ghi byte đầu nên lỗi `EXPORT_TOO_LARGE` còn trả được JSON.
// Không có cột liêm chính. Không ghi `audit_log` (thao tác đọc); không log nội dung.
func (s *Service) ResultsCSV(ctx context.Context, courseID, examID uuid.UUID, w io.Writer, before func()) error {
	q := store.New(s.Pool)
	e, err := s.examOf(ctx, q, courseID, examID)
	if err != nil {
		return err
	}
	rows, err := q.ExamResultRows(ctx, store.ExamResultRowsParams{CourseID: courseID, ExamID: examID})
	if err != nil {
		return fmt.Errorf("exam: bảng điểm: %w", err)
	}
	if len(rows) > s.Results.csvMax() {
		return apierr.Validation(apierr.FieldError{Field: "export", Code: "EXPORT_TOO_LARGE", Message: "Lớp quá lớn để xuất một lần."})
	}
	metas, err := q.ExamItemsMeta(ctx, store.ExamItemsMetaParams{CourseID: courseID, ExamID: examID})
	if err != nil {
		return fmt.Errorf("exam: câu của bài: %w", err)
	}
	atts, err := q.ExamScoredAttempts(ctx, store.ExamScoredAttemptsParams{CourseID: courseID, ExamID: examID})
	if err != nil {
		return fmt.Errorf("exam: lượt đã chấm: %w", err)
	}
	perItem := make(map[uuid.UUID]map[uuid.UUID]string, len(atts))
	for _, a := range atts {
		var br []breakdownRow
		if json.Unmarshal(a.Breakdown, &br) != nil {
			continue
		}
		m := make(map[uuid.UUID]string, len(br))
		for _, r := range br {
			d, _ := decimal.NewFromString(r.Earned)
			m[r.ItemID] = csvNum(d)
		}
		perItem[a.StudentID] = m
	}
	before()
	bw := bufio.NewWriter(w)
	if _, err := bw.WriteString("\ufeff"); err != nil {
		return err
	}
	cw := csv.NewWriter(bw)
	cw.Comma = ';'
	head := []string{"mssv", "ho_ten", "trang_thai", "diem_tu_dong", "diem_chinh_thuc", "nop_luc", "ly_do_nop"}
	for _, m := range metas {
		head = append(head, fmt.Sprintf("cau_%d", m.Position))
	}
	if err := cw.Write(head); err != nil {
		return err
	}
	for i, r := range rows {
		st := rowStatus(e, r)
		if st == "NOT_STARTED" {
			st = "ABSENT"
		}
		rec := []string{csvSafe(r.StudentCode), csvSafe(r.FullName), st, "", "", "", ""}
		if st == "GRADED" {
			if r.AutoScore.Valid {
				rec[3] = csvNum(r.AutoScore.Decimal)
			}
			sc := r.AdjustedScore
			if !sc.Valid {
				sc = r.AutoScore
			}
			if sc.Valid {
				rec[4] = csvNum(sc.Decimal)
			}
		}
		if r.SubmittedAt != nil {
			rec[5] = ViTime(*r.SubmittedAt)
		}
		if r.SubmitReason != nil {
			rec[6] = string(*r.SubmitReason)
		}
		for _, m := range metas {
			rec = append(rec, perItem[r.StudentID][m.ItemID])
		}
		if err := cw.Write(rec); err != nil {
			return err
		}
		if i%200 == 199 {
			cw.Flush()
			if err := bw.Flush(); err != nil {
				return err
			}
			if err := ctx.Err(); err != nil {
				return err
			}
		}
	}
	cw.Flush()
	if err := cw.Error(); err != nil {
		return err
	}
	return bw.Flush()
}
