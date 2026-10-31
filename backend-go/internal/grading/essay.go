package grading

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"path"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	norm_ "golang.org/x/text/unicode/norm"

	"github.com/edupilot/backend-go/internal/ingest"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/store"
)

// Mã lỗi của kết quả `FAILED` (SRS 4.6.10).
const (
	ErrExtractFailed      = "EXTRACT_FAILED"
	ErrTextTooLong        = "TEXT_TOO_LONG"
	ErrLLMUnavailable     = "LLM_UNAVAILABLE"
	ErrLLMBudgetExhausted = "LLM_BUDGET_EXHAUSTED"
	ErrInternal           = "INTERNAL"
)

// EmptyFeedback là nhận xét cố định của bài trống (không do AI).
const EmptyFeedback = "Bài nộp không có nội dung để chấm."

// EssayConfig là các hằng của Grading Engine (SRS 4.1).
type EssayConfig struct {
	MinChars, MaxChars, EchoWords int
	LowConfidence, Disagree       decimal.Decimal
	ExtractTimeout                time.Duration
	AttemptTimeout                time.Duration
}

// DefaultEssayConfig là giá trị mặc định của SRS 4.1.
func DefaultEssayConfig() EssayConfig {
	return EssayConfig{MinChars: 50, MaxChars: 60000, EchoWords: 12, LowConfidence: decimal.RequireFromString("0.6"), Disagree: decimal.NewFromInt(1),
		ExtractTimeout: 90 * time.Second, AttemptTimeout: 120 * time.Second}
}

// BlobGetter đọc tệp bài nộp (stream, không ghi đĩa).
type BlobGetter interface {
	Get(ctx context.Context, key string) (io.ReadCloser, error)
}

// TextExtractor trích văn bản từ PDF / DOCX. Lỗi tạm bọc `ingest.ErrExtractUnavailable`; tệp hỏng bọc `ingest.ErrExtractFailed`.
type TextExtractor interface {
	Extract(ctx context.Context, filename string, body io.Reader) (string, error)
}

// DoclingExtractor gọi container `docling-serve` (D46): lượt 1 không OCR, văn bản quá ngắn thì lượt 2 có OCR (cùng quy tắc `internal/ingest`).
type DoclingExtractor struct {
	D       *ingest.Docling
	MinText int
}

func (e DoclingExtractor) Extract(ctx context.Context, filename string, body io.Reader) (string, error) {
	raw, err := io.ReadAll(body)
	if err != nil {
		return "", err
	}
	res, err := e.D.Convert(ctx, filename, bytes.NewReader(raw), ingest.ConvertOptions{}, nil)
	if err != nil {
		return "", err
	}
	if utf8.RuneCountInString(strings.TrimSpace(res.Markdown)) >= max(e.MinText, 1) {
		return res.Markdown, nil
	}
	res, err = e.D.Convert(ctx, filename, bytes.NewReader(raw), ingest.ConvertOptions{OCR: true}, nil)
	if err != nil {
		return "", err
	}
	return res.Markdown, nil
}

// Grader chấm nháp MỘT bài nộp (SRS 4.6). Hàng đợi (US-P7-07) gọi Run; mọi nhánh lỗi vĩnh viễn được GHI vào `grading_results` (FAILED, tiêu chí NULL — không bao giờ 0 điểm),
// lỗi tạm trả `*TransientError` để hàng đợi thử lại. Không tự công bố: chỉ ghi DRAFT.
type Grader struct {
	Pool    *pgxpool.Pool
	LLM     Structurer
	Blob    BlobGetter
	Extract TextExtractor
	Cfg     EssayConfig
	Log     *slog.Logger
}

// Outcome là kết quả của một lần chạy: DONE (DRAFT) hoặc FAILED (đã ghi, mã lỗi vĩnh viễn).
type Outcome struct {
	State string // "DONE" | "FAILED"
	Code  string // mã lỗi khi FAILED
	Flags []string
}

// TransientError: lỗi tạm (LLM chết sau dự phòng của gateway, quá hạn, docling 5xx) — hàng đợi quyết thử lại hay hết lượt.
type TransientError struct {
	Code string
	Err  error
}

func (e *TransientError) Error() string {
	return "grading: lỗi tạm " + e.Code + ": " + e.Err.Error()
}
func (e *TransientError) Unwrap() error { return e.Err }

func (g *Grader) log() *slog.Logger {
	if g.Log != nil {
		return g.Log
	}
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// Run chạy việc chấm của `resultID`. Danh tính LLM luôn có `CourseID` (thiếu → gateway trả ErrMaskFailed, KHÔNG gọi LLM).
func (g *Grader) Run(ctx context.Context, courseID, resultID uuid.UUID) (Outcome, error) {
	q := store.New(g.Pool)
	c, err := q.EssayGradeContext(ctx, store.EssayGradeContextParams{CourseID: courseID, ResultID: resultID})
	if err != nil {
		return Outcome{}, fmt.Errorf("grading: đọc việc chấm: %w", err)
	}
	if c.Status != "DRAFT" {
		return Outcome{}, fmt.Errorf("grading: kết quả đã %s, không chấm lại", c.Status)
	}
	var rubric []RubricItem
	if err := json.Unmarshal(c.Rubric, &rubric); err != nil || len(rubric) == 0 {
		return g.fail(ctx, q, c, ErrInternal, nil)
	}
	ctx = llm.WithIdentity(ctx, llm.Identity{CourseID: &courseID})

	text, outcome, err := g.text(ctx, q, c)
	if err != nil || outcome != nil {
		if outcome != nil {
			return *outcome, nil
		}
		return Outcome{}, err
	}
	cleaned := StripIdentityLines(text)
	hits := ScanInjection(text) // quét trên văn bản gốc để vị trí tô khớp màn duyệt; chỉ gắn cờ, KHÔNG sửa đầu vào
	var flags []string
	if len(InjectionCodes(hits)) > 0 {
		flags = append(flags, FlagInjectionSuspected)
	}
	late := LatePolicy{AllowLate: c.AllowLate, PerDay: c.LatePenaltyPerDay, MaxDays: int(c.LateMaxDays)}
	lateDays := LateDays(c.SubmittedAt, c.DueAt)
	if !c.AllowLate && lateDays > 0 && (c.Source == "IMAP" || c.Source == "TEAMS") {
		flags = append(flags, FlagLateAfterCutoff)
	}

	if utf8.RuneCountInString(cleaned) < g.Cfg.MinChars { // bài trống: không gọi LLM
		return g.saveEmpty(ctx, q, c, rubric, hits, append(flags, FlagEmptySubmission), lateDays)
	}

	p1, p2, perr := g.passes(ctx, c, rubric, cleaned, text)
	if perr != nil {
		var te *TransientError
		if errors.As(perr, &te) {
			return Outcome{}, perr
		}
		return g.fail(ctx, q, c, perr.Error(), flags)
	}
	m := Merge(p1, p2, rubric, g.Cfg.Disagree)
	flags = append(flags, m.Flags...)
	var comments []string
	for _, r := range rubric {
		comments = append(comments, p1.Comments[r.ID])
	}
	comments = append(comments, p1.Feedback)
	if c.ReferenceAnswer != nil && AnswerEcho(comments, *c.ReferenceAnswer, g.Cfg.EchoWords) {
		flags = append(flags, FlagAnswerEcho)
	}
	flags = uniqSorted(flags)
	penalty := LatePenalty(lateDays, late, p1.Total)
	if c.LatePenaltyManual {
		penalty = c.ManualLatePenalty
	}
	return g.saveDraft(ctx, q, c, rubric, m, p2, flags, hits, lateDays, penalty)
}

// text trả văn bản bài (đã NFC) hoặc ghi FAILED / trả lỗi tạm.
func (g *Grader) text(ctx context.Context, q *store.Queries, c store.EssayGradeContextRow) (string, *Outcome, error) {
	if c.ExtractedText != nil && c.ExtractStatus != nil && *c.ExtractStatus == "OK" {
		return *c.ExtractedText, nil, nil
	}
	ectx, cancel := context.WithTimeout(ctx, g.Cfg.ExtractTimeout)
	defer cancel()
	rc, err := g.Blob.Get(ectx, c.BlobKey)
	if err != nil {
		return "", nil, &TransientError{Code: ErrExtractFailed, Err: err}
	}
	defer func() { _ = rc.Close() }()
	var text string
	switch strings.ToLower(path.Ext(c.FileName)) {
	case ".txt", ".md":
		b, err := io.ReadAll(io.LimitReader(rc, int64(g.Cfg.MaxChars)*4+4))
		if err != nil {
			return "", nil, &TransientError{Code: ErrExtractFailed, Err: err}
		}
		text = string(b)
	default:
		text, err = g.Extract.Extract(ectx, c.FileName, rc)
		if err != nil {
			if errors.Is(err, ingest.ErrExtractFailed) {
				_ = q.EssaySetExtract(ctx, store.EssaySetExtractParams{CourseID: c.CourseID, ID: c.SubmissionID, Status: ptr("FAILED")})
				o, ferr := g.fail(ctx, q, c, ErrExtractFailed, []string{FlagExtractFailed})
				return "", &o, ferr
			}
			return "", nil, &TransientError{Code: ErrExtractFailed, Err: err}
		}
	}
	text = strings.ToValidUTF8(norm(text), "")
	switch {
	case utf8.RuneCountInString(text) > g.Cfg.MaxChars:
		_ = q.EssaySetExtract(ctx, store.EssaySetExtractParams{CourseID: c.CourseID, ID: c.SubmissionID, Status: ptr("FAILED")})
		o, ferr := g.fail(ctx, q, c, ErrTextTooLong, nil)
		return "", &o, ferr
	case strings.TrimSpace(text) == "" && strings.ToLower(path.Ext(c.FileName)) != ".txt" && strings.ToLower(path.Ext(c.FileName)) != ".md":
		_ = q.EssaySetExtract(ctx, store.EssaySetExtractParams{CourseID: c.CourseID, ID: c.SubmissionID, Status: ptr("FAILED")})
		o, ferr := g.fail(ctx, q, c, ErrExtractFailed, []string{FlagExtractFailed})
		return "", &o, ferr
	}
	if err := q.EssaySetExtract(ctx, store.EssaySetExtractParams{CourseID: c.CourseID, ID: c.SubmissionID, Text: &text, Status: ptr("OK")}); err != nil {
		return "", nil, fmt.Errorf("grading: lưu văn bản: %w", err)
	}
	return text, nil, nil
}

// passes chạy hai lượt SONG SONG (lượt 2 đảo thứ tự tiêu chí), mỗi lượt đúng một llm.Structured (task GRADING, làn BATCH, temperature 0).
func (g *Grader) passes(ctx context.Context, c store.EssayGradeContextRow, rubric []RubricItem, cleaned, text string) (PassResult, PassResult, error) {
	pctx, cancel := context.WithTimeout(ctx, g.Cfg.AttemptTimeout)
	defer cancel()
	schema := EssaySchema(rubric)
	var wg sync.WaitGroup
	var res [2]PassResult
	var errs [2]error
	for i := range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res[i], errs[i] = g.onePass(pctx, c, rubric, cleaned, text, schema, i == 1)
		}()
	}
	wg.Wait()
	if errs[0] != nil {
		return PassResult{}, PassResult{}, errs[0]
	}
	if errs[1] != nil { // lượt 2 hỏng khi lượt 1 tốt: giữ lượt 1, Merge gắn NEEDS_REVIEW
		var te *TransientError
		if errors.As(errs[1], &te) && errors.Is(pctx.Err(), context.Canceled) && ctx.Err() != nil {
			return PassResult{}, PassResult{}, errs[1]
		}
		g.log().WarnContext(ctx, "lượt chấm 2 lỗi", "result_id", c.ResultID, "error", errs[1].Error())
		res[1] = PassResult{}
	}
	return res[0], res[1], nil
}

func (g *Grader) onePass(ctx context.Context, c store.EssayGradeContextRow, rubric []RubricItem, cleaned, text string, schema json.RawMessage, second bool) (PassResult, error) {
	ref := ""
	if c.ReferenceAnswer != nil {
		ref = *c.ReferenceAnswer
	}
	system, user := BuildEssayPrompt(PromptInput{Title: c.Title, Description: c.Description, Reference: ref, Rubric: rubric, Text: cleaned, Second: second}, NewBoundary(cleaned))
	zero := 0.0
	lane := llm.LaneBatch
	req := llm.Request{Task: llm.TaskGrading, Lane: &lane, Params: llm.Params{Temperature: &zero, MaxTokens: 4096},
		Messages: []llm.Message{{Role: "system", Content: system}, {Role: "user", Content: user}}}
	raw, err := g.LLM.Structured(ctx, req, schema)
	if err != nil {
		return PassResult{}, classify(err)
	}
	var out RawOutput
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&out); err != nil {
		return PassResult{}, errors.New(ErrOutputInvalid)
	}
	res := Check(out, text, rubric, g.Cfg.LowConfidence)
	if !res.Valid {
		return PassResult{}, errors.New(ErrOutputInvalid)
	}
	return res, nil
}

// classify: ngân sách hết / đầu ra sai schema là lỗi VĨNH VIỄN (trả error có mã, ghi FAILED); còn lại là lỗi tạm.
func classify(err error) error {
	var un *llm.ErrUnavailable
	switch {
	case errors.As(err, &un) && un.Reason == llm.ReasonBudgetExhausted:
		return errors.New(ErrLLMBudgetExhausted)
	case errors.Is(err, llm.ErrBadRequest):
		return errors.New(ErrOutputInvalid)
	case errors.Is(err, llm.ErrMaskFailed):
		return errors.New(ErrInternal)
	}
	return &TransientError{Code: ErrLLMUnavailable, Err: err}
}

type critJSON struct {
	ID       string     `json:"id"`
	Score    *string    `json:"score"`
	Comment  string     `json:"comment"`
	Edited   bool       `json:"edited"`
	Evidence []Evidence `json:"evidence"`
}

func (g *Grader) fail(ctx context.Context, q *store.Queries, c store.EssayGradeContextRow, code string, flags []string) (Outcome, error) {
	var rubric []RubricItem
	_ = json.Unmarshal(c.Rubric, &rubric)
	crit := make([]critJSON, len(rubric))
	for i, r := range rubric {
		crit[i] = critJSON{ID: r.ID, Evidence: []Evidence{}}
	}
	raw, _ := json.Marshal(crit)
	flags = uniqSorted(flags)
	if _, err := q.EssaySaveFailed(ctx, store.EssaySaveFailedParams{CourseID: c.CourseID, ID: c.ResultID, ErrorCode: &code, Criteria: raw, Flags: flags}); err != nil {
		return Outcome{}, fmt.Errorf("grading: ghi lỗi: %w", err)
	}
	return Outcome{State: "FAILED", Code: code, Flags: flags}, nil
}

func (g *Grader) saveEmpty(ctx context.Context, q *store.Queries, c store.EssayGradeContextRow, rubric []RubricItem, hits []InjectionHit, flags []string, lateDays int) (Outcome, error) {
	crit := make([]critJSON, len(rubric))
	ai := map[string]string{}
	for i, r := range rubric {
		z := "0.00"
		crit[i] = critJSON{ID: r.ID, Score: &z, Evidence: []Evidence{}}
		ai[r.ID] = z
	}
	return g.write(ctx, q, c, crit, ai, EmptyFeedback, decimal.Zero, decimal.Zero, lateDays, nil, uniqSorted(flags), nil, passDetail(nil, nil, rubric, hits, nil))
}

func (g *Grader) saveDraft(ctx context.Context, q *store.Queries, c store.EssayGradeContextRow, rubric []RubricItem, m Merged, p2 PassResult, flags []string, hits []InjectionHit, lateDays int, penalty decimal.Decimal) (Outcome, error) {
	p1 := m.Pass1
	crit := make([]critJSON, len(rubric))
	ai := map[string]string{}
	for i, r := range rubric {
		s := p1.Scores[r.ID].StringFixed(2)
		crit[i] = critJSON{ID: r.ID, Score: &s, Comment: p1.Comments[r.ID], Evidence: p1.Evidence[r.ID]}
		ai[r.ID] = s
	}
	return g.write(ctx, q, c, crit, ai, p1.Feedback, p1.Total, penalty, lateDays, p1.Reported, flags, &p1, passDetail(&p1, &p2, rubric, hits, p1.Dropped))
}

func passDetail(p1, p2 *PassResult, rubric []RubricItem, hits []InjectionHit, dropped []Dropped) json.RawMessage {
	d := map[string]any{}
	if p1 != nil {
		for _, r := range rubric {
			e := map[string]any{"pass1": p1.Scores[r.ID].StringFixed(2), "pass2": nil}
			if p2 != nil && p2.Valid {
				e["pass2"] = p2.Scores[r.ID].StringFixed(2)
			}
			d[r.ID] = e
		}
		t := map[string]any{"pass1": p1.Total.StringFixed(2), "pass2": nil}
		if p2 != nil && p2.Valid {
			t["pass2"] = p2.Total.StringFixed(2)
		}
		d["total"] = t
	}
	d["_injection"] = hits
	if hits == nil {
		d["_injection"] = []InjectionHit{}
	}
	if dropped == nil {
		dropped = []Dropped{}
	}
	d["_dropped"] = dropped
	raw, _ := json.Marshal(d)
	return raw
}

func (g *Grader) write(ctx context.Context, q *store.Queries, c store.EssayGradeContextRow, crit []critJSON, ai map[string]string, feedback string, sum, penalty decimal.Decimal, lateDays int, reported *decimal.Decimal, flags []string, p1 *PassResult, detail json.RawMessage) (Outcome, error) {
	critRaw, _ := json.Marshal(crit)
	aiRaw, _ := json.Marshal(ai)
	total := TotalAfterLate(sum, penalty)
	conf := decimal.NullDecimal{}
	if p1 != nil {
		conf = nd(&p1.Confidence)
	}
	n, err := q.EssaySaveDraft(ctx, store.EssaySaveDraftParams{
		CourseID: c.CourseID, ID: c.ResultID, Criteria: critRaw, OverallFeedback: &feedback, Total: decimal.NullDecimal{Decimal: total, Valid: true}, AiScores: aiRaw, AiTotalBeforeLate: decimal.NullDecimal{Decimal: sum, Valid: true},
		LateDays: int16(lateDays), LatePenalty: penalty, LlmReportedTotal: nd(reported), Flags: flags, Confidence: conf, PassDetail: detail,
	})
	if err != nil {
		return Outcome{}, fmt.Errorf("grading: ghi bản nháp: %w", err)
	}
	if n == 0 {
		return Outcome{}, errors.New("grading: kết quả không còn ở trạng thái DRAFT")
	}
	return Outcome{State: "DONE", Flags: flags}, nil
}

func ptr[T any](v T) *T { return &v }

func nd(d *decimal.Decimal) decimal.NullDecimal {
	if d == nil {
		return decimal.NullDecimal{}
	}
	return decimal.NullDecimal{Decimal: *d, Valid: true}
}

func norm(s string) string { return norm_.NFC.String(s) }
