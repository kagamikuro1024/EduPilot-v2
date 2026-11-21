package exam

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/store"
)

// Thống kê điểm thi theo lớp / tuần (US-P10-06). CHỈ bài PUBLISHED, lượt GRADED; điểm chính thức = adjusted_score nếu có, không thì auto_score.
// Tính bằng code thuần (decimal), không LLM, không cache phía máy chủ. Ngưỡng đạt = 50 % điểm tối đa của bài (ponytail: hằng số, bài thi chưa có cấu hình ngưỡng).

// CompareMinSubmissions: lớp có dưới số bài nộp này thì so sánh hiện «Chưa đủ» (AC6).
const CompareMinSubmissions = 5

var (
	passRatio  = decimal.RequireFromString("0.5") //nolint:gochecknoglobals // hằng thập phân
	weekISO    = regexp.MustCompile(`^(\d{4})-W(\d{2})$`)
	vnZone     = time.FixedZone("ICT", 7*3600) //nolint:gochecknoglobals // múi giờ cố định của lớp
	errBadPage = apierr.New(http.StatusUnprocessableEntity, apierr.InvalidCursor)
)

// AnalyticsExam là một bài thi trong phạm vi thống kê.
type AnalyticsExam struct {
	ID       uuid.UUID `json:"id"`
	Title    string    `json:"title"`
	OpensAt  time.Time `json:"opens_at"`
	MaxScore string    `json:"max_score"`
}

// Overview là phản hồi `…/analytics/exams/overview`. ADMIN chỉ có Submissions + PassRate (Mean / Median / Distribution = null).
type Overview struct {
	Scope        string          `json:"scope"` // "exam" | "week"
	Exams        []AnalyticsExam `json:"exams"`
	Submissions  int             `json:"submissions"`
	PassRate     *string         `json:"pass_rate"`
	Mean         *string         `json:"mean"`
	Median       *string         `json:"median"`
	Distribution []StatsBucket   `json:"distribution"`
}

// TrendPoint là một bài thi trên đường xu hướng (cũ → mới).
type TrendPoint struct {
	ExamID      uuid.UUID `json:"exam_id"`
	Title       string    `json:"title"`
	OpensAt     time.Time `json:"opens_at"`
	Submissions int       `json:"submissions"`
	PassRate    *string   `json:"pass_rate"`
	Mean        *string   `json:"mean"`
	Median      *string   `json:"median"`
}

// HardItem là một câu sai nhiều (trắc nghiệm hoặc bài code).
type HardItem struct {
	ItemID    uuid.UUID `json:"item_id"`
	ExamID    uuid.UUID `json:"exam_id"`
	Position  int       `json:"position"`
	Type      string    `json:"type"` // "MCQ" | "CODE"
	Title     string    `json:"title"`
	Topic     string    `json:"topic"`
	Attempts  int       `json:"attempts"`
	WrongRate string    `json:"wrong_rate"`
	CERate    *string   `json:"ce_rate"` // chỉ bài code
}

// DecliningRow là một sinh viên có điểm giảm ở 3 bài thi liên tiếp.
type DecliningRow struct {
	StudentID   uuid.UUID      `json:"student_id"`
	Name        string         `json:"name"`
	StudentCode *string        `json:"student_code"`
	Scores      []DecliningCut `json:"scores"`
}

// DecliningCut là điểm một bài trong bộ ba.
type DecliningCut struct {
	ExamID uuid.UUID `json:"exam_id"`
	Title  string    `json:"title"`
	Score  string    `json:"score"`
}

// CompareRow là một lớp trong bảng so sánh của giảng viên.
type CompareRow struct {
	CourseID    uuid.UUID `json:"course_id"`
	ClassCode   string    `json:"class_code"`
	Name        string    `json:"name"`
	Exams       int       `json:"exams"`
	Submissions int       `json:"submissions"`
	Enough      bool      `json:"enough"`
	Mean        *string   `json:"mean"`
	Median      *string   `json:"median"`
	PassRate    *string   `json:"pass_rate"`
}

func fix(d decimal.Decimal) *string { s := d.StringFixed(2); return &s }

type scored struct {
	exam  uuid.UUID
	score decimal.Decimal
}

type summary struct {
	n                    int
	mean, median, passed *string
	dist                 []StatsBucket
}

// summarise tính trung bình / trung vị / tỉ lệ đạt / phân bố 10 khoảng đều trên [0, top] (top = điểm tối đa lớn nhất trong phạm vi).
func summarise(sc []scored, maxOf map[uuid.UUID]decimal.Decimal) summary {
	out := summary{n: len(sc)}
	if len(sc) == 0 {
		return out
	}
	top := decimal.Zero
	for _, m := range maxOf {
		top = decimal.Max(top, m)
	}
	w := top.Div(decimal.NewFromInt(10))
	out.dist = make([]StatsBucket, 10)
	for i := range out.dist {
		out.dist[i] = StatsBucket{From: w.Mul(decimal.NewFromInt(int64(i))).StringFixed(2), To: w.Mul(decimal.NewFromInt(int64(i + 1))).StringFixed(2)}
	}
	vals := make([]decimal.Decimal, len(sc))
	sum, pass := decimal.Zero, 0
	for i, x := range sc {
		vals[i] = x.score
		sum = sum.Add(x.score)
		if x.score.GreaterThanOrEqual(maxOf[x.exam].Mul(passRatio)) {
			pass++
		}
		idx := 9
		if w.IsPositive() {
			idx = min(int(x.score.Div(w).Floor().IntPart()), 9)
		}
		out.dist[max(idx, 0)].Count++
	}
	n := decimal.NewFromInt(int64(len(sc)))
	out.mean = fix(sum.DivRound(n, 16))
	slices.SortFunc(vals, func(a, b decimal.Decimal) int { return a.Cmp(b) })
	med := vals[len(vals)/2]
	if len(vals)%2 == 0 {
		med = vals[len(vals)/2-1].Add(vals[len(vals)/2]).DivRound(decimal.NewFromInt(2), 16)
	}
	out.median = fix(med)
	out.passed = fix(decimal.NewFromInt(int64(pass)).DivRound(n, 16))
	return out
}

func (s *Service) publishedExams(ctx context.Context, q *store.Queries, courseID uuid.UUID) ([]store.ExamAnalyticsExamsRow, error) {
	exs, err := q.ExamAnalyticsExams(ctx, courseID)
	if err != nil {
		return nil, fmt.Errorf("exam: bài thi đã công bố: %w", err)
	}
	return exs, nil
}

func exView(e store.ExamAnalyticsExamsRow) AnalyticsExam {
	return AnalyticsExam{ID: e.ID, Title: e.Title, OpensAt: e.OpensAt, MaxScore: e.MaxScore.StringFixed(2)}
}

func weekBounds(raw string) (time.Time, time.Time, bool) {
	var start time.Time
	if m := weekISO.FindStringSubmatch(raw); m != nil {
		var y, w int
		_, _ = fmt.Sscanf(m[1]+" "+m[2], "%d %d", &y, &w)
		if w < 1 || w > 53 {
			return start, start, false
		}
		jan4 := time.Date(y, 1, 4, 0, 0, 0, 0, vnZone)
		mon := jan4.AddDate(0, 0, -((int(jan4.Weekday())+6)%7)+(w-1)*7)
		return mon, mon.AddDate(0, 0, 7), true
	}
	d, err := time.ParseInLocation("2006-01-02", raw, vnZone)
	if err != nil {
		return start, start, false
	}
	mon := d.AddDate(0, 0, -((int(d.Weekday()) + 6) % 7))
	return mon, mon.AddDate(0, 0, 7), true
}

// AnalyticsOverview: `examID` xor `week` (cả hai → 422; không có → bài PUBLISHED mới nhất). Không có bài → cấu trúc rỗng (không 500). `identity=false` (ADMIN): chỉ tổng.
func (s *Service) AnalyticsOverview(ctx context.Context, courseID uuid.UUID, examID *uuid.UUID, week string, identity bool) (Overview, error) {
	if examID != nil && week != "" {
		return Overview{}, apierr.Validation(apierr.FieldError{Field: "week", Code: "conflict", Message: "Chỉ chọn exam_id hoặc week."})
	}
	q := store.New(s.Pool)
	exs, err := s.publishedExams(ctx, q, courseID)
	if err != nil {
		return Overview{}, err
	}
	out := Overview{Scope: "exam", Exams: []AnalyticsExam{}}
	var scope []store.ExamAnalyticsExamsRow
	switch {
	case examID != nil:
		for _, e := range exs {
			if e.ID == *examID {
				scope = append(scope, e)
			}
		}
		if len(scope) == 0 {
			return Overview{}, apierr.New(http.StatusNotFound, apierr.NotFound)
		}
	case week != "":
		from, to, ok := weekBounds(week)
		if !ok {
			return Overview{}, apierr.Validation(apierr.FieldError{Field: "week", Code: "invalid", Message: "week là YYYY-Www hoặc một ngày YYYY-MM-DD."})
		}
		out.Scope = "week"
		for _, e := range exs {
			if !e.OpensAt.Before(from) && e.OpensAt.Before(to) {
				scope = append(scope, e)
			}
		}
	case len(exs) > 0:
		scope = exs[len(exs)-1:]
	}
	if len(scope) == 0 {
		return out, nil
	}
	ids := make([]string, len(scope))
	maxOf := map[uuid.UUID]decimal.Decimal{}
	for i, e := range scope {
		ids[i] = e.ID.String()
		maxOf[e.ID] = e.MaxScore
		out.Exams = append(out.Exams, exView(e))
	}
	csv := strings.Join(ids, ",")
	rows, err := q.ExamAnalyticsScores(ctx, store.ExamAnalyticsScoresParams{CourseID: courseID, ExamIds: &csv})
	if err != nil {
		return Overview{}, fmt.Errorf("exam: điểm: %w", err)
	}
	sc := make([]scored, len(rows))
	for i, r := range rows {
		sc[i] = scored{r.ExamID, r.Score}
	}
	sm := summarise(sc, maxOf)
	out.Submissions, out.PassRate = sm.n, sm.passed
	if identity {
		out.Mean, out.Median, out.Distribution = sm.mean, sm.median, sm.dist
	}
	return out, nil
}

// AnalyticsTrend: một điểm cho mỗi bài PUBLISHED, cũ → mới. ADMIN: chỉ số bài nộp + tỉ lệ đạt.
func (s *Service) AnalyticsTrend(ctx context.Context, courseID uuid.UUID, identity bool) ([]TrendPoint, error) {
	q := store.New(s.Pool)
	exs, err := s.publishedExams(ctx, q, courseID)
	if err != nil {
		return nil, err
	}
	rows, err := q.ExamAnalyticsScores(ctx, store.ExamAnalyticsScoresParams{CourseID: courseID})
	if err != nil {
		return nil, fmt.Errorf("exam: điểm: %w", err)
	}
	by := map[uuid.UUID][]scored{}
	for _, r := range rows {
		by[r.ExamID] = append(by[r.ExamID], scored{r.ExamID, r.Score})
	}
	out := make([]TrendPoint, 0, len(exs))
	for _, e := range exs {
		sm := summarise(by[e.ID], map[uuid.UUID]decimal.Decimal{e.ID: e.MaxScore})
		p := TrendPoint{ExamID: e.ID, Title: e.Title, OpensAt: e.OpensAt, Submissions: sm.n, PassRate: sm.passed}
		if identity {
			p.Mean, p.Median = sm.mean, sm.median
		}
		out = append(out, p)
	}
	return out, nil
}

type rateCursor struct {
	R string    `json:"r"`
	I uuid.UUID `json:"i"`
}

// AnalyticsHardItems: câu sai nhiều nhất của một bài PUBLISHED (mặc định bài mới nhất), sai nhiều → ít, hoà theo item_id; con trỏ khoá (wrong_rate, item_id).
func (s *Service) AnalyticsHardItems(ctx context.Context, courseID uuid.UUID, examID *uuid.UUID, cursor string, limit int) ([]HardItem, *string, error) {
	q := store.New(s.Pool)
	exs, err := s.publishedExams(ctx, q, courseID)
	if err != nil {
		return nil, nil, err
	}
	var target *store.ExamAnalyticsExamsRow
	for i := range exs {
		if (examID == nil && i == len(exs)-1) || (examID != nil && exs[i].ID == *examID) {
			target = &exs[i]
		}
	}
	if target == nil {
		if examID != nil {
			return nil, nil, apierr.New(http.StatusNotFound, apierr.NotFound)
		}
		return []HardItem{}, nil, nil
	}
	items, err := q.ExamAnalyticsItems(ctx, store.ExamAnalyticsItemsParams{CourseID: courseID, ExamID: target.ID})
	if err != nil {
		return nil, nil, fmt.Errorf("exam: câu của bài: %w", err)
	}
	bds, err := q.ExamAnalyticsBreakdowns(ctx, store.ExamAnalyticsBreakdownsParams{CourseID: courseID, ExamID: target.ID})
	if err != nil {
		return nil, nil, fmt.Errorf("exam: breakdown: %w", err)
	}
	type tally struct {
		n, wrong, ce int
		lost         decimal.Decimal
	}
	t := map[uuid.UUID]*tally{}
	for _, raw := range bds {
		var rows []breakdownRow
		if json.Unmarshal(raw, &rows) != nil {
			continue
		}
		for _, r := range rows {
			if r.Void {
				continue
			}
			mx, _ := decimal.NewFromString(r.Max)
			er, _ := decimal.NewFromString(r.Earned)
			if !mx.IsPositive() {
				continue
			}
			x := t[r.ItemID]
			if x == nil {
				x = &tally{}
				t[r.ItemID] = x
			}
			x.n++
			x.lost = x.lost.Add(decimal.NewFromInt(1).Sub(er.DivRound(mx, 16)))
			if er.LessThan(mx) {
				x.wrong++
			}
			if r.Code != nil && !r.Code.CompileOK {
				x.ce++
			}
		}
	}
	var all []HardItem
	for _, it := range items {
		x := t[it.ItemID]
		if x == nil || x.n == 0 {
			continue
		}
		n := decimal.NewFromInt(int64(x.n))
		h := HardItem{ItemID: it.ItemID, ExamID: target.ID, Position: int(it.Position), Type: "MCQ", Title: it.Title, Topic: it.Topic, Attempts: x.n}
		if it.Type == store.QuestionTypeCODE {
			h.Type = "CODE"
			h.WrongRate = x.lost.DivRound(n, 4).StringFixed(4)
			ce := decimal.NewFromInt(int64(x.ce)).DivRound(n, 4).StringFixed(4)
			h.CERate = &ce
		} else {
			h.WrongRate = decimal.NewFromInt(int64(x.wrong)).DivRound(n, 4).StringFixed(4)
		}
		all = append(all, h)
	}
	slices.SortFunc(all, func(a, b HardItem) int {
		if c := strings.Compare(b.WrongRate, a.WrongRate); c != 0 { // cùng độ dài thập phân → so chuỗi = so số
			return c
		}
		return strings.Compare(a.ItemID.String(), b.ItemID.String())
	})
	start := 0
	if cursor != "" {
		raw, err := base64.RawURLEncoding.DecodeString(cursor)
		var c rateCursor
		if err != nil || json.Unmarshal(raw, &c) != nil {
			return nil, nil, errBadPage
		}
		start = len(all)
		for i, h := range all {
			if h.WrongRate < c.R || (h.WrongRate == c.R && h.ItemID.String() > c.I.String()) {
				start = i
				break
			}
		}
	}
	page := all[start:]
	var next *string
	if len(page) > limit {
		page = page[:limit]
		raw, _ := json.Marshal(rateCursor{R: page[limit-1].WrongRate, I: page[limit-1].ItemID}) //nolint:errchkjson // kiểu cố định
		c := base64.RawURLEncoding.EncodeToString(raw)
		next = &c
	}
	if page == nil {
		page = []HardItem{}
	}
	return page, next, nil
}

type nameCursor struct {
	N string    `json:"n"`
	I uuid.UUID `json:"i"`
}

// AnalyticsDeclining: sinh viên có điểm giảm nghiêm ngặt ở 3 bài PUBLISHED liên tiếp; chỉ staff lớp. KHÔNG đổi student_risk (chỉ gợi ý).
func (s *Service) AnalyticsDeclining(ctx context.Context, courseID uuid.UUID, cursor string, limit int) ([]DecliningRow, *string, error) {
	arg := store.ExamAnalyticsDecliningParams{CourseID: courseID, MaxRows: int32(limit + 1)} //nolint:gosec // ≤ 101
	if cursor != "" {
		raw, err := base64.RawURLEncoding.DecodeString(cursor)
		var c nameCursor
		if err != nil || json.Unmarshal(raw, &c) != nil {
			return nil, nil, errBadPage
		}
		arg.CursorName, arg.CursorID = &c.N, &c.I
	}
	rows, err := store.New(s.Pool).ExamAnalyticsDeclining(ctx, arg)
	if err != nil {
		return nil, nil, fmt.Errorf("exam: điểm đang giảm: %w", err)
	}
	var next *string
	if len(rows) > limit {
		rows = rows[:limit]
		raw, _ := json.Marshal(nameCursor{N: rows[limit-1].FullName, I: rows[limit-1].StudentID}) //nolint:errchkjson // kiểu cố định
		c := base64.RawURLEncoding.EncodeToString(raw)
		next = &c
	}
	out := make([]DecliningRow, len(rows))
	for i, r := range rows {
		out[i] = DecliningRow{StudentID: r.StudentID, Name: r.FullName, StudentCode: r.StudentCode, Scores: []DecliningCut{
			{r.Exam1ID, r.Exam1Title, r.Score1.StringFixed(2)}, {r.Exam2ID, r.Exam2Title, r.Score2.StringFixed(2)}, {r.Exam3ID, r.Exam3Title, r.Score3.StringFixed(2)}}}
	}
	return out, next, nil
}

// AnalyticsCompare: các lớp giảng viên đang dạy (ACTIVE, vai TEACHER); lớp dưới CompareMinSubmissions bài nộp: Enough=false, không số.
func (s *Service) AnalyticsCompare(ctx context.Context, teacher uuid.UUID) ([]CompareRow, error) {
	q := store.New(s.Pool)
	cs, err := q.ExamAnalyticsTeacherCourses(ctx, teacher)
	if err != nil {
		return nil, fmt.Errorf("exam: lớp của giảng viên: %w", err)
	}
	out := make([]CompareRow, 0, len(cs))
	for _, c := range cs {
		exs, err := s.publishedExams(ctx, q, c.ID)
		if err != nil {
			return nil, err
		}
		maxOf := map[uuid.UUID]decimal.Decimal{}
		for _, e := range exs {
			maxOf[e.ID] = e.MaxScore
		}
		rows, err := q.ExamAnalyticsScores(ctx, store.ExamAnalyticsScoresParams{CourseID: c.ID})
		if err != nil {
			return nil, fmt.Errorf("exam: điểm: %w", err)
		}
		sc := make([]scored, len(rows))
		for i, r := range rows {
			sc[i] = scored{r.ExamID, r.Score}
		}
		sm := summarise(sc, maxOf)
		row := CompareRow{CourseID: c.ID, ClassCode: c.ClassCode, Name: c.Name, Exams: len(exs), Submissions: sm.n, Enough: sm.n >= CompareMinSubmissions}
		if row.Enough {
			row.Mean, row.Median, row.PassRate = sm.mean, sm.median, sm.passed
		}
		out = append(out, row)
	}
	return out, nil
}
