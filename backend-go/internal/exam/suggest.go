package exam

import (
	"context"
	_ "embed" // lời nhắc cố định nằm cạnh mã (SRS 4.9)
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/edupilot/backend-go/internal/jobs"
	"github.com/edupilot/backend-go/internal/judge"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/store"
)

// Structurer là phần của `llm.Client` mà gợi ý AI dùng (một lời gọi `Structured` mỗi việc). Khai báo ở đây vì đây là tệp DUY NHẤT của gói được import internal/llm.
type Structurer interface {
	Structured(ctx context.Context, r llm.Request, schema json.RawMessage) (json.RawMessage, error)
}

//go:embed prompts/suggest_mcq.txt
var mcqPrompt string

//go:embed prompts/suggest_tests.txt
var testsPrompt string

const mcqSchema = `{"type":"object","additionalProperties":false,"required":["questions"],"properties":{"questions":{"type":"array","items":{"type":"object","additionalProperties":false,
"required":["type","title","stem","options","value","explanation","difficulty"],"properties":{
"type":{"type":"string","enum":["MCQ_SINGLE","MCQ_MULTI","TRUE_FALSE"]},"title":{"type":"string"},"stem":{"type":"string"},
"options":{"type":"array","items":{"type":"object","additionalProperties":false,"required":["body","correct"],"properties":{"body":{"type":"string"},"correct":{"type":"boolean"}}}},
"value":{"type":["boolean","null"]},"explanation":{"type":"string"},"difficulty":{"type":"string","enum":["EASY","MEDIUM","HARD"]}}}}}}`

const testsSchema = `{"type":"object","additionalProperties":false,"required":["inputs"],"properties":{"inputs":{"type":"array","items":{"type":"object","additionalProperties":false,
"required":["name","input","note"],"properties":{"name":{"type":"string"},"input":{"type":"string"},"note":{"type":"string"}}}}}}`

// Dropped là một mục AI đề xuất bị bỏ, kèm lý do.
type Dropped struct {
	Index   int    `json:"index"`
	Name    string `json:"name,omitempty"`
	Reason  string `json:"reason"`
	Verdict string `json:"verdict,omitempty"`
}

// SuggestResult là `jobs.result` của `question.suggest`.
type SuggestResult struct {
	Kind      string      `json:"kind"`
	Requested int         `json:"requested"`
	Created   []uuid.UUID `json:"created"`
	Dropped   []Dropped   `json:"dropped"`
}

// llmCall gọi `Structured` MỘT lần (làn mặc định của QUESTION_GEN = BATCH — không chạm làn INTERACTIVE của chat); chỉ thử lại khi hệ thống quá tải.
func (w *Worker) llmCall(ctx context.Context, system, user string) (json.RawMessage, error) {
	temp := 0.4
	schemaOf := mcqSchema
	if system == testsPrompt {
		schemaOf = testsSchema
	}
	req := llm.Request{Task: llm.TaskQuestionGen, Messages: []llm.Message{{Role: "system", Content: system}, {Role: "user", Content: user}}, Params: llm.Params{Temperature: &temp}}
	backoff := w.RetryWait
	if backoff == nil {
		backoff = []time.Duration{2 * time.Second, 5 * time.Second}
	}
	for attempt := 0; ; attempt++ {
		raw, err := w.LLM.Structured(ctx, req, json.RawMessage(schemaOf))
		var over *llm.ErrOverloaded
		switch {
		case err == nil:
			return raw, nil
		case errors.Is(err, llm.ErrNotConfigured):
			return nil, &jobs.UserError{Code: "LLM_NOT_CONFIGURED", Message: "Chưa cấu hình AI. Nhờ quản trị viên thêm nhà cung cấp."}
		case errors.As(err, &over) && attempt < len(backoff):
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("exam: gợi ý bị huỷ: %w", ctx.Err())
			case <-time.After(backoff[attempt]):
			}
		case errors.As(err, &over):
			return nil, &jobs.UserError{Code: "OVERLOADED", Message: "Hệ thống AI đang bận, thử lại sau ít phút."}
		default:
			return nil, &jobs.UserError{Code: "LLM_UNAVAILABLE", Message: "AI chưa phản hồi được, thử lại sau."}
		}
	}
}

func (w *Worker) suggest(ctx context.Context, j jobs.JobCtx) (any, error) {
	var p SuggestPayload
	if err := json.Unmarshal(j.Payload, &p); err != nil {
		return nil, fmt.Errorf("exam: payload suggest: %w", err)
	}
	if w.LLM == nil {
		return nil, &jobs.UserError{Code: "LLM_NOT_CONFIGURED", Message: "Chưa cấu hình AI. Nhờ quản trị viên thêm nhà cung cấp."}
	}
	owner, course := j.OwnerID, p.CourseID
	ctx = llm.WithIdentity(ctx, llm.Identity{UserID: &owner, CourseID: &course})
	j.Progress(10)
	if p.Kind == SuggestCodeTests {
		return w.suggestTests(ctx, j, p)
	}
	return w.suggestMCQ(ctx, j, p)
}

type aiQuestion struct {
	Type    string `json:"type"`
	Title   string `json:"title"`
	Stem    string `json:"stem"`
	Options []struct {
		Body    string `json:"body"`
		Correct bool   `json:"correct"`
	} `json:"options"`
	Value       *bool  `json:"value"`
	Explanation string `json:"explanation"`
	Difficulty  string `json:"difficulty"`
}

// mcqPromptInput là dữ liệu vào prompt: CHỈ chủ đề, độ khó, số câu, văn bản nguồn do giảng viên nhập (không có dữ liệu sinh viên).
func mcqPromptInput(p SuggestPayload) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Chủ đề: %s\nĐộ khó: %s\nSố câu cần soạn: %d\n", p.Topic, p.Difficulty, p.Count)
	if p.SourceText != "" {
		fmt.Fprintf(&b, "Tài liệu nguồn:\n%s\n", p.SourceText)
	}
	return b.String()
}

func (w *Worker) suggestMCQ(ctx context.Context, j jobs.JobCtx, p SuggestPayload) (any, error) {
	raw, err := w.llmCall(ctx, mcqPrompt, mcqPromptInput(p))
	if err != nil {
		return nil, err
	}
	var parsed struct {
		Questions []aiQuestion `json:"questions"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, &jobs.UserError{Code: "LLM_BAD_RESULT", Message: "AI trả về kết quả không đọc được, thử lại."}
	}
	j.Progress(60)
	res := SuggestResult{Kind: SuggestMCQ, Requested: p.Count, Created: []uuid.UUID{}, Dropped: []Dropped{}}
	var ok []Checked
	var idx []int
	for i, a := range parsed.Questions {
		if i >= p.Count {
			res.Dropped = append(res.Dropped, Dropped{Index: i, Reason: "EXTRA"})
			continue
		}
		in := QuestionIn{Type: a.Type, Title: a.Title, Topic: p.Topic, Difficulty: a.Difficulty, Stem: a.Stem, Value: a.Value}
		if in.Difficulty == "" {
			in.Difficulty = p.Difficulty
		}
		if a.Explanation != "" {
			e := a.Explanation
			in.Explanation = &e
		}
		for k, o := range a.Options {
			in.Options = append(in.Options, OptionIn{Body: o.Body})
			if o.Correct {
				in.Correct = append(in.Correct, k)
			}
		}
		if a.Type == TypeTrueFalse {
			in.Options, in.Correct = nil, nil
		}
		c, errs, _ := Validate(in)
		if len(errs) > 0 {
			res.Dropped = append(res.Dropped, Dropped{Index: i, Reason: errs[0].Code})
			continue
		}
		if c.Type == TypeCode {
			res.Dropped = append(res.Dropped, Dropped{Index: i, Reason: "TYPE_NOT_SUPPORTED"})
			continue
		}
		ok, idx = append(ok, c), append(idx, i)
	}
	err = (&Service{Pool: w.Pool}).tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		for _, c := range ok {
			job := j.ID
			row, err := (&Service{}).insertQuestion(ctx, q, p.CourseID, j.OwnerID, c, store.QuestionOriginAIDRAFT, store.QuestionReviewStatusPENDING, &job)
			if err != nil {
				return err
			}
			res.Created = append(res.Created, row.ID)
		}
		if len(ok) > 0 {
			if _, err := outbox.Write(ctx, tx, TopicQuestionReviewed, map[string]any{"course_id": p.CourseID}); err != nil {
				return fmt.Errorf("exam: outbox: %w", err)
			}
		}
		return nil
	})
	_ = idx
	if err != nil {
		return nil, err
	}
	return res, nil
}

// normalizeOut chuẩn hoá stdout của lời giải mẫu: bỏ khoảng trắng cuối dòng, bỏ dòng trống cuối, kết thúc bằng một dấu xuống dòng.
func normalizeOut(s string) string {
	lines := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " \t")
	}
	out := strings.TrimRight(strings.Join(lines, "\n"), "\n")
	if out == "" {
		return ""
	}
	return out + "\n"
}

func (w *Worker) suggestTests(ctx context.Context, j jobs.JobCtx, p SuggestPayload) (any, error) {
	if w.Sandbox == nil {
		return nil, errNoJudge()
	}
	q := store.New(w.Pool)
	qb, err := q.QuestionGet(ctx, store.QuestionGetParams{CourseID: p.CourseID, ID: *p.QuestionID})
	if err != nil {
		return nil, fmt.Errorf("exam: đọc câu hỏi: %w", err)
	}
	cp, err := q.CodeProblemGet(ctx, store.CodeProblemGetParams{CourseID: p.CourseID, QuestionID: qb.ID})
	if err != nil {
		return nil, fmt.Errorf("exam: đọc bài code: %w", err)
	}
	if cp.ReferenceLanguage == nil || cp.ReferenceSource == nil {
		return nil, &jobs.UserError{Code: "REFERENCE_REQUIRED", Message: "Cần nhập lời giải mẫu để sinh đầu ra mong đợi."}
	}
	tests, err := w.approvedTests(ctx, p.CourseID, qb.ID)
	if err != nil {
		return nil, err
	}
	var in strings.Builder
	fmt.Fprintf(&in, "Đề bài:\n%s\n\nGiới hạn: %d ms, %d MiB.\nSố đầu vào cần đề xuất: %d\n", qb.Stem, cp.TimeLimitMs, cp.MemoryLimitMb, p.Count)
	shown := 0
	for _, t := range tests {
		if t.IsSample && shown < 2 {
			shown++
			fmt.Fprintf(&in, "\nVí dụ %d — input:\n%s\noutput:\n%s\n", shown, t.Input, t.Expected)
		}
	}
	raw, err := w.llmCall(ctx, testsPrompt, in.String())
	if err != nil {
		return nil, err
	}
	var parsed struct {
		Inputs []struct{ Name, Input, Note string } `json:"inputs"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, &jobs.UserError{Code: "LLM_BAD_RESULT", Message: "AI trả về kết quả không đọc được, thử lại."}
	}
	j.Progress(50)
	lang := judge.Language(*cp.ReferenceLanguage)
	comp, err := w.Sandbox.Compile(ctx, lang, *cp.ReferenceSource)
	if err != nil {
		if judge.IsSystemError(err) {
			return nil, errNoJudge()
		}
		return nil, fmt.Errorf("exam: biên dịch lời giải mẫu: %w", err)
	}
	if comp.FileID == "" {
		return nil, &jobs.UserError{Code: "REFERENCE_NOT_COMPILING", Message: "Lời giải mẫu không biên dịch được."}
	}
	defer func() { _ = w.Sandbox.DeleteFile(context.WithoutCancel(ctx), comp.FileID) }()

	st, err := q.TestcaseStats(ctx, store.TestcaseStatsParams{CourseID: p.CourseID, ProblemID: qb.ID})
	if err != nil {
		return nil, fmt.Errorf("exam: đếm test: %w", err)
	}
	res := SuggestResult{Kind: SuggestCodeTests, Requested: p.Count, Created: []uuid.UUID{}, Dropped: []Dropped{}}
	type prepared struct {
		name, in, out string
	}
	var keep []prepared
	taken := map[string]bool{}
	names, err := q.TestcaseNames(ctx, store.TestcaseNamesParams{CourseID: p.CourseID, ProblemID: qb.ID})
	if err != nil {
		return nil, fmt.Errorf("exam: đọc tên test: %w", err)
	}
	for _, n := range names {
		taken[n] = true
	}
	lim := judge.Limits{TimeMS: int(cp.TimeLimitMs), MemoryMB: int(cp.MemoryLimitMb), OutputLimitKB: int(cp.OutputLimitKb)}
	room := MaxTests - int(st.TotalAll)
	for i, c := range parsed.Inputs {
		d := Dropped{Index: i, Name: c.Name}
		switch {
		case i >= p.Count:
			d.Reason = "EXTRA"
		case len(keep) >= room:
			d.Reason = "TOO_MANY_TESTS"
		case len(c.Input) > MaxInlineBytes || !utf8.ValidString(c.Input):
			d.Reason = "INPUT_INVALID"
		}
		if d.Reason != "" {
			res.Dropped = append(res.Dropped, d)
			continue
		}
		r, err := w.Sandbox.Run(ctx, comp.FileID, c.Input, lim)
		if err != nil {
			if judge.IsSystemError(err) {
				return nil, errNoJudge()
			}
			return nil, fmt.Errorf("exam: chạy lời giải mẫu: %w", err)
		}
		if v := judge.MapRunStatus(r.Status); v != judge.AC {
			res.Dropped = append(res.Dropped, Dropped{Index: i, Name: c.Name, Reason: "REFERENCE_" + string(v), Verdict: string(v)})
			continue
		}
		out := normalizeOut(r.Stdout)
		if len(out) > MaxTestBytes {
			res.Dropped = append(res.Dropped, Dropped{Index: i, Name: c.Name, Reason: "OUTPUT_TOO_LARGE"})
			continue
		}
		name := c.Name
		if !testNameRE.MatchString(name) || taken[name] {
			name = "ai_" + strconv.Itoa(int(st.TotalAll)+len(keep)+1)
		}
		taken[name] = true
		keep = append(keep, prepared{name, c.Input, out})
	}
	j.Progress(85)
	svc := &Service{Pool: w.Pool, Blob: w.Svc.Blob}
	var made []string
	err = svc.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
		for k, t := range keep {
			inp, err := svc.putText(ctx, p.CourseID, qb.ID, t.in, "in", &made)
			if err != nil {
				return err
			}
			exp, err := svc.putText(ctx, p.CourseID, qb.ID, t.out, "out", &made)
			if err != nil {
				return err
			}
			row, err := q.TestcaseInsert(ctx, store.TestcaseInsertParams{CourseID: p.CourseID, ProblemID: qb.ID, Position: st.MaxPosition + int32(k) + 1, Name: t.name, Weight: 1, //nolint:gosec // ≤ 100
				Input: inp.inline, InputBlobKey: inp.key, Expected: exp.inline, ExpectedBlobKey: exp.key, InputBytes: int32(inp.n), ExpectedBytes: int32(exp.n), Source: "AI_DRAFT", Approved: false}) //nolint:gosec // ≤ 1 MiB
			if err != nil {
				return fmt.Errorf("exam: ghi test gợi ý: %w", err)
			}
			res.Created = append(res.Created, row.ID)
		}
		return nil
	})
	if err != nil {
		svc.dropBlobs(ctx, made)
		return nil, err
	}
	return res, nil
}
