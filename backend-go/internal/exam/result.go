package exam

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/store"
)

// ---- DTO kết quả (sinh viên SAU công bố; Staff dùng lại phần câu hỏi) ---------------------------------------------------------------------
// Trường nhạy cảm của giai đoạn làm bài (`correct`, `answer`, `explanation`, `hidden`) CÓ ở đây vì đây là kết quả đã công bố (SRS 4.8.3); `answer` / `explanation` chỉ khi
// `reveal_answers`. Không có: test ẩn (tên / input / expected / verdict từng test), trọng số, lời giải mẫu, `override` chi tiết, bài / điểm của người khác.

// ResultView là kết quả của sinh viên SAU khi công bố.
type ResultView struct {
	Exam          ResultExamView   `json:"exam"`
	Score         *string          `json:"score"`
	ScoreAdjusted bool             `json:"score_adjusted"`
	Appeal        ResultAppealView `json:"appeal"`
	Items         []ResultItemView `json:"items"`
}

// ResultExamView là khối `exam` của kết quả.
type ResultExamView struct {
	ID              uuid.UUID  `json:"id"`
	Title           string     `json:"title"`
	MaxScore        string     `json:"max_score"`
	PublishedAt     *time.Time `json:"published_at"`
	RevealAnswers   bool       `json:"reveal_answers"`
	AppealDays      int        `json:"appeal_days"`
	AppealOpenUntil *time.Time `json:"appeal_open_until"`
}

// ResultAppealView là trạng thái phúc khảo của lượt (`status` null khi chưa gửi).
type ResultAppealView struct {
	Status   *string `json:"status"`
	Response *string `json:"response"`
}

// ResultItemView là một câu trong kết quả.
type ResultItemView struct {
	ItemID          uuid.UUID          `json:"item_id"`
	Position        int                `json:"position"`
	Type            string             `json:"type"`
	Stem            string             `json:"stem"`
	Options         []OptionView       `json:"options"`
	Earned          string             `json:"earned"`
	Max             string             `json:"max"`
	Correct         *bool              `json:"correct"`
	Mine            json.RawMessage    `json:"mine"`
	Answer          json.RawMessage    `json:"answer"`
	Explanation     *string            `json:"explanation"`
	Overridden      bool               `json:"overridden"`
	Samples         []ResultSampleView `json:"samples"`
	Hidden          *ResultHiddenView  `json:"hidden"`
	FinalSubmission *ResultFinalView   `json:"final_submission"`
	CompileLog      *string            `json:"compile_log"`
}

// ResultSampleView là một test MẪU (công khai theo định nghĩa) kèm kết quả của sinh viên.
type ResultSampleView struct {
	Name     string `json:"name"`
	Verdict  string `json:"verdict"`
	TimeMS   int    `json:"time_ms"`
	MemoryKB int    `json:"memory_kb"`
	Input    string `json:"input"`
	Expected string `json:"expected"`
}

// ResultHiddenView CHỈ có số test ẩn đạt / tổng.
type ResultHiddenView struct {
	Passed int `json:"passed"`
	Total  int `json:"total"`
}

// ResultFinalView là bản nộp tính điểm; các trường này không đổi khi chấm lại.
type ResultFinalView struct {
	ID        uuid.UUID `json:"id"`
	Language  string    `json:"language"`
	Source    string    `json:"source"`
	CreatedAt time.Time `json:"created_at"`
	CompileOK bool      `json:"compile_ok"`
}

// officialScore = điểm chính thức: `adjusted_score` nếu có, nếu không `auto_score` (SRS 4.8.3).
func officialScore(a store.ExamAttempt) (decimal.Decimal, bool) {
	if a.AdjustedScore.Valid {
		return a.AdjustedScore.Decimal, true
	}
	return a.AutoScore.Decimal, a.AutoScore.Valid
}

// appealOpenUntil: hạn gửi yêu cầu xem lại (`published_at + appeal_days`); null khi `appeal_days = 0` hoặc chưa công bố.
func appealOpenUntil(e store.Exam) *time.Time {
	if e.AppealDays <= 0 || e.PublishedAt == nil {
		return nil
	}
	t := e.PublishedAt.Add(time.Duration(e.AppealDays) * 24 * time.Hour)
	return &t
}

// resultItems dựng các câu của kết quả một lượt từ ảnh chụp `breakdown` (không từ bản nộp — nên không trống khi đang chấm lại). `reveal` quyết định `answer` / `explanation`.
func (s *Service) resultItems(ctx context.Context, q *store.Queries, e store.Exam, a store.ExamAttempt, reveal bool) ([]ResultItemView, error) {
	metas, err := q.ExamItemsMeta(ctx, store.ExamItemsMetaParams{CourseID: a.CourseID, ExamID: a.ExamID})
	if err != nil {
		return nil, fmt.Errorf("exam: câu của bài: %w", err)
	}
	opts, err := q.ExamPreviewOptions(ctx, store.ExamPreviewOptionsParams{CourseID: a.CourseID, ExamID: a.ExamID})
	if err != nil {
		return nil, fmt.Errorf("exam: lựa chọn: %w", err)
	}
	optsOf := map[uuid.UUID][]OptionView{}
	for _, o := range opts {
		optsOf[o.QuestionID] = append(optsOf[o.QuestionID], OptionView{ID: o.ID, Body: o.Body})
	}
	saved, err := q.AnswersList(ctx, store.AnswersListParams{CourseID: a.CourseID, AttemptID: a.ID})
	if err != nil {
		return nil, fmt.Errorf("exam: câu trả lời: %w", err)
	}
	mine := make(map[uuid.UUID]json.RawMessage, len(saved))
	for _, r := range saved {
		mine[r.ItemID] = r.Answer
	}
	samples, err := q.ExamResultSamples(ctx, store.ExamResultSamplesParams{CourseID: a.CourseID, ExamID: a.ExamID})
	if err != nil {
		return nil, fmt.Errorf("exam: test mẫu: %w", err)
	}
	sampleOf := make(map[uuid.UUID]store.ExamResultSamplesRow, len(samples))
	for _, t := range samples {
		sampleOf[t.ID] = t
	}
	finals, err := q.AttemptFinalSubmissions(ctx, store.AttemptFinalSubmissionsParams{CourseID: a.CourseID, AttemptID: a.ID})
	if err != nil {
		return nil, fmt.Errorf("exam: bản nộp cuối: %w", err)
	}
	finalOf := make(map[uuid.UUID]store.AttemptFinalSubmissionsRow, len(finals))
	for _, f := range finals {
		finalOf[f.ItemID] = f
	}
	var rows []breakdownRow
	if len(a.Breakdown) > 0 {
		if err := json.Unmarshal(a.Breakdown, &rows); err != nil {
			return nil, fmt.Errorf("exam: đọc breakdown: %w", err)
		}
	}
	byItem := make(map[uuid.UUID]breakdownRow, len(rows))
	for _, r := range rows {
		byItem[r.ItemID] = r
	}
	out := make([]ResultItemView, 0, len(metas))
	for _, m := range metas {
		ov := parseOverride(m.Override)
		b := byItem[m.ItemID]
		earned, max := b.Earned, m.Points.StringFixed(2)
		if earned == "" {
			earned = "0"
		}
		it := ResultItemView{ItemID: m.ItemID, Position: int(m.Position), Type: string(m.Type), Stem: m.Stem, Options: []OptionView{}, Earned: decimalFixed(earned), Max: max, Overridden: ov.Void || len(ov.AnswerKey) > 0, Samples: []ResultSampleView{}}
		if m.Type == store.QuestionTypeCODE {
			it.fillCode(b.Code, sampleOf, finalOf[m.ItemID])
			out = append(out, it)
			continue
		}
		it.Options = optsOf[m.QuestionID]
		if it.Options == nil {
			it.Options = []OptionView{}
		}
		it.Mine = cleanAnswer(mine[m.ItemID])
		if !ov.Void {
			ok := decimalFixed(earned) == max
			it.Correct = &ok
		}
		if reveal {
			key := m.AnswerKey
			if len(ov.AnswerKey) > 0 {
				key = ov.AnswerKey
			}
			it.Answer, it.Explanation = cleanAnswer(key), m.Explanation
		}
		out = append(out, it)
	}
	return out, nil
}

// decimalFixed đưa `earned` (đủ chữ số) về 2 chữ số CHỈ để hiển thị; điểm cuối không bao giờ tính ngược từ số này (SRS 4.8.3).
func decimalFixed(s string) string {
	d, err := decimal.NewFromString(s)
	if err != nil {
		return "0.00"
	}
	return d.StringFixed(2)
}

func (it *ResultItemView) fillCode(c *breakdownCode, sampleOf map[uuid.UUID]store.ExamResultSamplesRow, f store.AttemptFinalSubmissionsRow) {
	if c != nil {
		it.Hidden = &ResultHiddenView{Passed: c.Hidden.Passed, Total: c.Hidden.Total}
		for _, sm := range c.Samples {
			t := sampleOf[sm.TestID]
			v := ResultSampleView{Name: t.Name, Verdict: sm.Verdict, TimeMS: sm.TimeMS, MemoryKB: sm.MemoryKB}
			if t.Input != nil {
				v.Input = *t.Input
			}
			if t.Expected != nil {
				v.Expected = *t.Expected
			}
			it.Samples = append(it.Samples, v)
		}
	}
	if f.ID == uuid.Nil {
		return
	}
	ok := f.CompileOk != nil && *f.CompileOk
	it.FinalSubmission = &ResultFinalView{ID: f.ID, Language: f.Language, Source: f.Source, CreatedAt: f.CreatedAt, CompileOK: ok}
	if !ok && f.CompileLog != nil {
		it.CompileLog = f.CompileLog
	}
}

// AttemptResult: `GET …/result`. Chưa PUBLISHED → 409 RESULT_NOT_PUBLISHED, thân KHÔNG chứa dữ liệu; lượt của người khác → 404.
func (s *Service) AttemptResult(ctx context.Context, userID, courseID, examID, attemptID uuid.UUID) (ResultView, error) {
	q := store.New(s.Pool)
	a, err := q.AttemptOwn(ctx, store.AttemptOwnParams{CourseID: courseID, ExamID: examID, ID: attemptID, StudentID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return ResultView{}, notFound()
	}
	if err != nil {
		return ResultView{}, fmt.Errorf("exam: đọc lượt làm: %w", err)
	}
	e, err := q.ExamGet(ctx, store.ExamGetParams{CourseID: courseID, ID: examID})
	if err != nil {
		return ResultView{}, fmt.Errorf("exam: đọc bài thi: %w", err)
	}
	if e.Status != store.ExamStatusPUBLISHED {
		return ResultView{}, apierr.New(http.StatusConflict, apierr.ResultNotPublished)
	}
	v := ResultView{Exam: ResultExamView{ID: e.ID, Title: e.Title, MaxScore: e.MaxScore.StringFixed(2), PublishedAt: e.PublishedAt, RevealAnswers: e.RevealAnswers, AppealDays: int(e.AppealDays), AppealOpenUntil: appealOpenUntil(e)},
		ScoreAdjusted: a.AdjustedScore.Valid}
	if sc, ok := officialScore(a); ok {
		v.Score = new(sc.StringFixed(2))
	}
	ap, err := q.AppealByAttempt(ctx, store.AppealByAttemptParams{CourseID: courseID, AttemptID: attemptID})
	switch {
	case err == nil:
		v.Appeal.Status = new(string(ap.Status))
		v.Appeal.Response = ap.Response
	case !errors.Is(err, pgx.ErrNoRows):
		return ResultView{}, fmt.Errorf("exam: đọc phúc khảo: %w", err)
	}
	if v.Items, err = s.resultItems(ctx, q, e, a, e.RevealAnswers); err != nil {
		return ResultView{}, err
	}
	return v, nil
}

// cleanAnswer chỉ giữ hai khoá `option_ids` / `value` của một đáp án (đáp án đúng hoặc câu trả lời đã lưu): trường lạ trong JSON gốc không bao giờ ra ngoài (SRS 6.4).
func cleanAnswer(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	var a struct {
		OptionIDs []string `json:"option_ids,omitempty"`
		Value     *bool    `json:"value,omitempty"`
	}
	if json.Unmarshal(raw, &a) != nil {
		return nil
	}
	out, err := json.Marshal(a)
	if err != nil {
		return nil
	}
	return out
}
