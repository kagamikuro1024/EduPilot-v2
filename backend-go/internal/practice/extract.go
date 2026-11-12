package practice

import (
	"context"
	_ "embed" // lời nhắc cố định nằm cạnh mã
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/edupilot/backend-go/internal/exam"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/jobs"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/quiz"
	"github.com/edupilot/backend-go/internal/store"
)

// Loại việc nền của luyện đề (đăng ký ở worker, `jobs.Runner`; làn BATCH của task QUESTION_GEN: không chạm làn INTERACTIVE của chat).
const (
	KindExtract  = "question.extract"
	KindGenerate = "question.generate"
)

//go:embed prompts/extract.txt
var extractPrompt string

//go:embed prompts/generate.txt
var generatePrompt string

// ExtractIn là thân `POST …/questions/extract`.
type ExtractIn struct {
	DocumentID          uuid.UUID  `json:"document_id" validate:"required"`
	AnswerKeyDocumentID *uuid.UUID `json:"answer_key_document_id"`
}

// ExtractPayload là thân việc `question.extract`.
type ExtractPayload struct {
	CourseID uuid.UUID `json:"course_id"`
	ExtractIn
}

// Skipped là một câu không lưu, kèm lý do (SRS 4.2: danh sách "bỏ qua N câu").
type Skipped struct {
	Number int    `json:"number"`
	Page   int    `json:"page"`
	Reason string `json:"reason"` // NO_ANSWER | NO_ANSWER_MATCH | DUPLICATE | INVALID | CITATION_REQUIRED | TYPE_NOT_ALLOWED
}

// ExtractResult là `jobs.result` của `question.extract` / `question.generate`.
type ExtractResult struct {
	Created []uuid.UUID `json:"created"`
	Skipped []Skipped   `json:"skipped"`
}

// checkDoc: tài liệu thuộc lớp (404 nếu không), loại đúng (422 WRONG_DOCUMENT_TYPE), đã READY (409 DOCUMENT_NOT_READY).
func checkDoc(ctx context.Context, q *store.Queries, courseID, id uuid.UUID, want string) error {
	d, err := q.PracticeDocBrief(ctx, store.PracticeDocBriefParams{CourseID: courseID, ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return notFound()
	}
	if err != nil {
		return fmt.Errorf("practice: đọc tài liệu: %w", err)
	}
	if d.Type != want {
		return apierr.New(http.StatusUnprocessableEntity, apierr.WrongDocumentType)
	}
	if d.Status != "READY" {
		return apierr.New(http.StatusConflict, apierr.DocumentNotReady)
	}
	return nil
}

// EnqueueExtract xếp việc trích câu từ đề cũ (US-P9-02 AC1, AC6): 202 + job id.
func (s *Service) EnqueueExtract(ctx context.Context, owner, courseID uuid.UUID, in ExtractIn) (uuid.UUID, error) {
	q := store.New(s.Pool)
	if err := writable(ctx, q, courseID); err != nil {
		return uuid.Nil, err
	}
	if err := checkDoc(ctx, q, courseID, in.DocumentID, "EXAM_PAPER"); err != nil {
		return uuid.Nil, err
	}
	if in.AnswerKeyDocumentID != nil {
		if err := checkDoc(ctx, q, courseID, *in.AnswerKeyDocumentID, "ANSWER_KEY"); err != nil {
			return uuid.Nil, err
		}
	}
	j, err := s.Jobs.Enqueue(ctx, owner, KindExtract, ExtractPayload{CourseID: courseID, ExtractIn: in})
	if err != nil {
		return uuid.Nil, fmt.Errorf("practice: xếp việc trích: %w", err)
	}
	return j.ID, nil
}

// Worker chạy hai việc nền gọi LLM. `LLM` chỉ cần `Structured`.
type Worker struct {
	Pool *pgxpool.Pool
	Exam *exam.Service
	LLM  interface {
		Structured(ctx context.Context, r llm.Request, schema json.RawMessage) (json.RawMessage, error)
	}
	RetryWait []time.Duration // nil = 2 s, 5 s
}

// Register đăng ký hai loại việc vào runner.
func (w *Worker) Register(r *jobs.Runner) {
	r.Register(KindExtract, w.extract)
	r.Register(KindGenerate, w.generate)
}

func (w *Worker) call(ctx context.Context, system, user string, schema string) (json.RawMessage, error) {
	temp := 0.2
	req := llm.Request{Task: llm.TaskQuestionGen, Messages: []llm.Message{{Role: "system", Content: system}, {Role: "user", Content: user}}, Params: llm.Params{Temperature: &temp, MaxTokens: 4096}}
	backoff := w.RetryWait
	if backoff == nil {
		backoff = []time.Duration{2 * time.Second, 5 * time.Second}
	}
	for attempt := 0; ; attempt++ {
		raw, err := w.LLM.Structured(ctx, req, json.RawMessage(schema))
		var over *llm.ErrOverloaded
		switch {
		case err == nil:
			return raw, nil
		case errors.Is(err, llm.ErrNotConfigured):
			return nil, &jobs.UserError{Code: "LLM_NOT_CONFIGURED", Message: "Chưa cấu hình AI. Nhờ quản trị viên thêm nhà cung cấp."}
		case errors.As(err, &over) && attempt < len(backoff):
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("practice: việc bị huỷ: %w", ctx.Err())
			case <-time.After(backoff[attempt]):
			}
		case errors.As(err, &over):
			return nil, &jobs.UserError{Code: "OVERLOADED", Message: "Chưa trích được. Thử lại."}
		default:
			return nil, &jobs.UserError{Code: "LLM_UNAVAILABLE", Message: "AI chưa phản hồi được, thử lại sau."}
		}
	}
}

const extractSchema = `{"type":"object","additionalProperties":false,"required":["questions"],"properties":{"questions":{"type":"array","items":{"type":"object","additionalProperties":false,
"required":["number","stem","options","correct_label","page"],"properties":{"number":{"type":"integer"},"stem":{"type":"string"},"page":{"type":"integer"},"correct_label":{"type":["string","null"]},
"options":{"type":"array","items":{"type":"object","additionalProperties":false,"required":["body"],"properties":{"body":{"type":"string"}}}}}}}}}`

type extracted struct {
	Number       int    `json:"number"`
	Stem         string `json:"stem"`
	Page         int    `json:"page"`
	CorrectLabel string `json:"correct_label"`
	Options      []struct {
		Body string `json:"body"`
	} `json:"options"`
}

var answerLineRE = regexp.MustCompile(`(?im)(?:câu\s*)?\b(\d{1,3})\s*[.:)\-–=]\s*([A-H])\b`)

// ParseAnswerKey đọc bảng đáp án dạng "1. B", "Câu 2: C", "3) D" bằng luật (KHÔNG qua LLM: nội dung ANSWER_KEY không bao giờ vào prompt nào, US-P9-02 AC3).
func ParseAnswerKey(text string) map[int]int {
	out := map[int]int{}
	for _, m := range answerLineRE.FindAllStringSubmatch(text, -1) {
		n, _ := strconv.Atoi(m[1])
		if _, dup := out[n]; !dup {
			out[n] = int(strings.ToUpper(m[2])[0] - 'A')
		}
	}
	return out
}

// pages gộp các đoạn thành cửa sổ ≤ maxChars chữ, mỗi đoạn đánh dấu "[Trang N]".
func windows(chunks []store.PracticeDocChunksRow, maxChars int) []string {
	var out []string
	var b strings.Builder
	for _, c := range chunks {
		part := fmt.Sprintf("[Trang %d]\n%s\n", ptrOr(c.PageNo, 0), c.Text)
		if b.Len() > 0 && b.Len()+len(part) > maxChars {
			out = append(out, b.String())
			b.Reset()
		}
		b.WriteString(part)
	}
	if b.Len() > 0 {
		out = append(out, b.String())
	}
	return out
}

func ptrOr[T any](p *T, d T) T {
	if p != nil {
		return *p
	}
	return d
}

func titleOf(stem string) string {
	r := []rune(strings.Join(strings.Fields(stem), " "))
	if len(r) > 100 {
		r = r[:100]
	}
	return string(r)
}

func (w *Worker) extract(ctx context.Context, j jobs.JobCtx) (any, error) {
	var p ExtractPayload
	if err := json.Unmarshal(j.Payload, &p); err != nil {
		return nil, fmt.Errorf("practice: payload trích: %w", err)
	}
	if w.LLM == nil {
		return nil, &jobs.UserError{Code: "LLM_NOT_CONFIGURED", Message: "Chưa cấu hình AI. Nhờ quản trị viên thêm nhà cung cấp."}
	}
	owner, course := j.OwnerID, p.CourseID
	ctx = llm.WithIdentity(ctx, llm.Identity{UserID: &owner, CourseID: &course})
	q := store.New(w.Pool)
	chunks, err := q.PracticeDocChunks(ctx, store.PracticeDocChunksParams{DocumentID: p.DocumentID, CourseID: p.CourseID})
	if err != nil {
		return nil, fmt.Errorf("practice: đọc đề: %w", err)
	}
	if len(chunks) == 0 {
		return nil, &jobs.UserError{Code: "EMPTY_DOCUMENT", Message: "Không đọc được văn bản."}
	}
	var key map[int]int
	if p.AnswerKeyDocumentID != nil {
		ak, err := q.PracticeDocChunks(ctx, store.PracticeDocChunksParams{DocumentID: *p.AnswerKeyDocumentID, CourseID: p.CourseID})
		if err != nil {
			return nil, fmt.Errorf("practice: đọc đáp án: %w", err)
		}
		var sb strings.Builder
		for _, c := range ak {
			sb.WriteString(c.Text)
			sb.WriteByte('\n')
		}
		key = ParseAnswerKey(sb.String())
	}
	existing, err := q.PracticeStemsOfCourse(ctx, p.CourseID)
	if err != nil {
		return nil, fmt.Errorf("practice: đọc câu hiện có: %w", err)
	}
	seen := map[string]bool{}
	for _, s := range existing {
		seen[quiz.NormalizeShort(s)] = true
	}
	wins := windows(chunks, 6000)
	res := ExtractResult{Created: []uuid.UUID{}, Skipped: []Skipped{}}
	byNumber := map[int]bool{}
	for i, win := range wins {
		raw, err := w.call(ctx, extractPrompt, win, extractSchema)
		if err != nil {
			return nil, err
		}
		var parsed struct {
			Questions []extracted `json:"questions"`
		}
		if json.Unmarshal(raw, &parsed) != nil {
			res.Skipped = append(res.Skipped, Skipped{Reason: "INVALID"}) // cửa sổ này sai schema: đếm vào "bỏ qua", tiếp cửa sổ sau
			continue
		}
		if err := w.save(ctx, q, p, parsed.Questions, key, seen, byNumber, owner, j.ID, &res); err != nil {
			return nil, err
		}
		j.Progress(10 + 85*(i+1)/len(wins))
	}
	return res, nil
}

func (w *Worker) save(ctx context.Context, q *store.Queries, p ExtractPayload, qs []extracted, key map[int]int, seen map[string]bool, byNumber map[int]bool, owner, jobID uuid.UUID, res *ExtractResult) error {
	type ready struct {
		c    exam.Checked
		page int
	}
	var ok []ready
	for _, e := range qs {
		skip := func(reason string) {
			res.Skipped = append(res.Skipped, Skipped{Number: e.Number, Page: e.Page, Reason: reason})
		}
		if e.Number > 0 && byNumber[e.Number] { // câu nằm ở ranh giới hai cửa sổ bị trích hai lần
			continue
		}
		correct := -1
		switch {
		case key != nil:
			k, found := key[e.Number]
			if !found || e.Number == 0 {
				skip("NO_ANSWER_MATCH")
				continue
			}
			correct = k
		case e.CorrectLabel != "":
			correct = int(strings.ToUpper(e.CorrectLabel)[0] - 'A')
		default:
			skip("NO_ANSWER")
			continue
		}
		in := exam.QuestionIn{Type: exam.TypeMCQSingle, Title: titleOf(e.Stem), Topic: "Chưa phân loại", Difficulty: "MEDIUM", Stem: strings.TrimSpace(e.Stem), Correct: []int{correct}}
		for _, o := range e.Options {
			in.Options = append(in.Options, exam.OptionIn{Body: strings.TrimSpace(o.Body)})
		}
		c, errs, _ := exam.Validate(in)
		if len(errs) > 0 || correct < 0 || correct >= len(in.Options) {
			skip("INVALID")
			continue
		}
		norm := quiz.NormalizeShort(c.Stem)
		if seen[norm] {
			skip("DUPLICATE")
			continue
		}
		seen[norm] = true
		if e.Number > 0 {
			byNumber[e.Number] = true
		}
		ok = append(ok, ready{c, e.Page})
	}
	if len(ok) == 0 {
		return nil
	}
	svc := &exam.Service{}
	tx, err := w.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("practice: mở transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qt := q.WithTx(tx)
	for _, r := range ok {
		job := jobID
		row, err := svc.InsertDraft(ctx, qt, p.CourseID, owner, r.c, store.QuestionOriginEXTRACTED, &job)
		if err != nil {
			return err
		}
		cites, _ := json.Marshal([]map[string]any{{"document_id": p.DocumentID, "page": r.page}})
		if err := qt.PracticeQuestionSetCitations(ctx, store.PracticeQuestionSetCitationsParams{CourseID: p.CourseID, ID: row.ID, Citations: cites}); err != nil {
			return fmt.Errorf("practice: ghi trích dẫn: %w", err)
		}
		res.Created = append(res.Created, row.ID)
	}
	if _, err := outbox.Write(ctx, tx, exam.TopicQuestionReviewed, map[string]any{"course_id": p.CourseID}); err != nil {
		return fmt.Errorf("practice: outbox: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("practice: chốt lô câu: %w", err)
	}
	return nil
}
