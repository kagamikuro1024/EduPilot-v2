package practice

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/exam"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/jobs"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/quiz"
	"github.com/edupilot/backend-go/internal/store"
)

// Giới hạn của sinh câu (SRS FR-6).
const (
	MaxGenerate     = 10
	genSourceChars  = 12000 // trần chữ nguồn đưa vào MỘT lời gọi
	genMaxChunkIDs  = 40
	genMaxDocuments = 5
)

// GenerateIn là thân `POST …/questions/generate`.
type GenerateIn struct {
	DocumentIDs []uuid.UUID `json:"document_ids"`
	ChunkIDs    []uuid.UUID `json:"chunk_ids"`
	Topic       string      `json:"topic" validate:"required"`
	Difficulty  string      `json:"difficulty" validate:"required"`
	Count       int         `json:"count" validate:"required"`
	Types       []string    `json:"types"`
}

// GeneratePayload là thân việc `question.generate`.
type GeneratePayload struct {
	CourseID uuid.UUID `json:"course_id"`
	GenerateIn
}

var genTypes = map[string]bool{exam.TypeMCQSingle: true, exam.TypeMCQMulti: true, exam.TypeTrueFalse: true} //nolint:gochecknoglobals // tập loại cho phép, bất biến

// EnqueueGenerate xếp việc sinh câu từ đoạn tài liệu (US-P9-03): nguồn là `document_ids` HOẶC `chunk_ids`, không phải ANSWER_KEY (422 WRONG_DOCUMENT_TYPE).
func (s *Service) EnqueueGenerate(ctx context.Context, owner, courseID uuid.UUID, in GenerateIn) (uuid.UUID, error) {
	var errs []apierr.FieldError
	bad := func(f, c, m string) { errs = append(errs, apierr.FieldError{Field: f, Code: c, Message: m}) }
	in.Topic = strings.TrimSpace(in.Topic)
	if in.Topic == "" || len([]rune(in.Topic)) > 80 {
		bad("topic", "topic", "Chủ đề từ 1 đến 80 ký tự.")
	}
	if in.Count < 1 || in.Count > MaxGenerate {
		bad("count", "count", fmt.Sprintf("Số câu từ 1 đến %d.", MaxGenerate))
	}
	if in.Difficulty != "EASY" && in.Difficulty != "MEDIUM" && in.Difficulty != "HARD" {
		bad("difficulty", "difficulty", "Độ khó là EASY, MEDIUM hoặc HARD.")
	}
	if (len(in.DocumentIDs) == 0) == (len(in.ChunkIDs) == 0) {
		bad("document_ids", "source", "Chọn tài liệu hoặc đoạn làm nguồn (một trong hai).")
	}
	if len(in.DocumentIDs) > genMaxDocuments || len(in.ChunkIDs) > genMaxChunkIDs {
		bad("document_ids", "max", "Quá nhiều nguồn được chọn.")
	}
	if in.Types == nil { // không gửi → mặc định; gửi `[]` là sai (L9)
		in.Types = []string{exam.TypeMCQSingle}
	} else if len(in.Types) == 0 {
		bad("types", "type", "Chọn ít nhất một loại câu.")
	}
	for _, t := range in.Types {
		if !genTypes[t] {
			bad("types", "type", "Chỉ sinh MCQ_SINGLE, MCQ_MULTI, TRUE_FALSE.")
			break
		}
	}
	if len(errs) > 0 {
		return uuid.Nil, apierr.Validation(errs...)
	}
	q := store.New(s.Pool)
	if err := writable(ctx, q, courseID); err != nil {
		return uuid.Nil, err
	}
	for _, d := range in.DocumentIDs {
		b, err := q.PracticeDocBrief(ctx, store.PracticeDocBriefParams{CourseID: courseID, ID: d})
		if err != nil {
			return uuid.Nil, notFound()
		}
		if b.Type == "ANSWER_KEY" {
			return uuid.Nil, apierr.New(http.StatusUnprocessableEntity, apierr.WrongDocumentType)
		}
		if b.Status != "READY" {
			return uuid.Nil, apierr.New(http.StatusConflict, apierr.DocumentNotReady)
		}
	}
	if len(in.ChunkIDs) > 0 {
		ids := make([]string, len(in.ChunkIDs))
		for i, c := range in.ChunkIDs {
			ids[i] = c.String()
		}
		rows, err := q.PracticeChunksByIDs(ctx, store.PracticeChunksByIDsParams{CourseID: courseID, Ids: ids})
		if err != nil {
			return uuid.Nil, fmt.Errorf("practice: đọc đoạn nguồn: %w", err)
		}
		if len(rows) != len(in.ChunkIDs) {
			return uuid.Nil, notFound() // đoạn của lớp khác / không tồn tại: không lộ
		}
		for _, r := range rows {
			if r.DocType == "ANSWER_KEY" {
				return uuid.Nil, apierr.New(http.StatusUnprocessableEntity, apierr.WrongDocumentType)
			}
		}
	}
	if err := s.jobLimit(ctx, "generate", courseID); err != nil {
		return uuid.Nil, err
	}
	j, err := s.Jobs.Enqueue(ctx, owner, KindGenerate, GeneratePayload{CourseID: courseID, GenerateIn: in})
	if err != nil {
		return uuid.Nil, fmt.Errorf("practice: xếp việc sinh: %w", err)
	}
	return j.ID, nil
}

const generateSchema = `{"type":"object","additionalProperties":false,"required":["questions"],"properties":{"questions":{"type":"array","items":{"type":"object","additionalProperties":false,
"required":["type","stem","options","value","explanation","sources"],"properties":{"type":{"type":"string","enum":["MCQ_SINGLE","MCQ_MULTI","TRUE_FALSE"]},"stem":{"type":"string"},
"options":{"type":"array","items":{"type":"object","additionalProperties":false,"required":["body","correct"],"properties":{"body":{"type":"string"},"correct":{"type":"boolean"}}}},
"value":{"type":["boolean","null"]},"explanation":{"type":"string"},"sources":{"type":"array","items":{"type":"string"}}}}}}}`

type genSource struct {
	label string
	chunk store.PracticeChunksByIDsRow
}

func (w *Worker) generate(ctx context.Context, j jobs.JobCtx) (any, error) {
	var p GeneratePayload
	if err := json.Unmarshal(j.Payload, &p); err != nil {
		return nil, fmt.Errorf("practice: payload sinh: %w", err)
	}
	if w.LLM == nil {
		return nil, &jobs.UserError{Code: "LLM_NOT_CONFIGURED", Message: "Chưa cấu hình AI. Nhờ quản trị viên thêm nhà cung cấp."}
	}
	owner, course := j.OwnerID, p.CourseID
	ctx = llm.WithIdentity(ctx, llm.Identity{UserID: &owner, CourseID: &course})
	q := store.New(w.Pool)
	if len(p.Types) == 0 {
		p.Types = []string{exam.TypeMCQSingle}
	}

	var rows []store.PracticeChunksByIDsRow
	if len(p.ChunkIDs) > 0 {
		ids := make([]string, len(p.ChunkIDs))
		for i, c := range p.ChunkIDs {
			ids[i] = c.String()
		}
		r, err := q.PracticeChunksByIDs(ctx, store.PracticeChunksByIDsParams{CourseID: p.CourseID, Ids: ids})
		if err != nil {
			return nil, fmt.Errorf("practice: đọc đoạn nguồn: %w", err)
		}
		rows = r
	}
	for _, d := range p.DocumentIDs {
		cs, err := q.PracticeDocChunks(ctx, store.PracticeDocChunksParams{DocumentID: d, CourseID: p.CourseID})
		if err != nil {
			return nil, fmt.Errorf("practice: đọc tài liệu nguồn: %w", err)
		}
		for _, c := range cs {
			rows = append(rows, store.PracticeChunksByIDsRow{ID: c.ID, DocumentID: d, PageNo: c.PageNo, Text: c.Text})
		}
	}
	var src []genSource
	var b strings.Builder
	for _, r := range rows {
		if b.Len()+len(r.Text) > genSourceChars && len(src) > 0 {
			break
		}
		label := fmt.Sprintf("C%d", len(src)+1)
		fmt.Fprintf(&b, "[%s] (trang %d)\n%s\n\n", label, ptrOr(r.PageNo, 0), r.Text)
		src = append(src, genSource{label: label, chunk: r})
	}
	if len(src) == 0 {
		return nil, &jobs.UserError{Code: "EMPTY_SOURCE", Message: "Nguồn không có văn bản để sinh câu."}
	}
	user := fmt.Sprintf("Chủ đề: %s\nĐộ khó: %s\nSố câu: %d\nLoại: %s\n\nĐOẠN TÀI LIỆU:\n%s", p.Topic, p.Difficulty, p.Count, strings.Join(p.Types, ", "), b.String())
	j.Progress(20)
	raw, err := w.call(ctx, generatePrompt, user, generateSchema)
	if err != nil {
		return nil, err
	}
	var parsed struct {
		Questions []struct {
			Type    string `json:"type"`
			Stem    string `json:"stem"`
			Options []struct {
				Body    string `json:"body"`
				Correct bool   `json:"correct"`
			} `json:"options"`
			Value       *bool    `json:"value"`
			Explanation string   `json:"explanation"`
			Sources     []string `json:"sources"`
		} `json:"questions"`
	}
	if json.Unmarshal(raw, &parsed) != nil {
		return nil, &jobs.UserError{Code: "LLM_BAD_RESULT", Message: "AI trả về kết quả không đọc được, thử lại."}
	}
	existing, err := q.PracticeStemsOfCourse(ctx, p.CourseID)
	if err != nil {
		return nil, fmt.Errorf("practice: đọc câu hiện có: %w", err)
	}
	seen := map[string]bool{}
	for _, s := range existing {
		seen[quiz.NormalizeShort(s)] = true
	}
	allowed := map[string]bool{}
	for _, t := range p.Types {
		allowed[t] = true
	}
	byLabel := map[string]genSource{}
	for _, s := range src {
		byLabel[s.label] = s
	}
	type ready struct {
		c     exam.Checked
		cites []map[string]any
	}
	res := ExtractResult{Created: []uuid.UUID{}, Skipped: []Skipped{}}
	var ok []ready
	for i, a := range parsed.Questions {
		skip := func(reason string) { res.Skipped = append(res.Skipped, Skipped{Number: i + 1, Reason: reason}) }
		if i >= p.Count {
			skip("EXTRA")
			continue
		}
		if !allowed[a.Type] {
			skip("TYPE_NOT_ALLOWED")
			continue
		}
		var cites []map[string]any
		for _, l := range a.Sources {
			if s, found := byLabel[strings.Trim(l, "[] ")]; found {
				cites = append(cites, map[string]any{"document_id": s.chunk.DocumentID, "chunk_id": s.chunk.ID, "page": ptrOr(s.chunk.PageNo, 0)})
			}
		}
		if len(cites) == 0 {
			skip("CITATION_REQUIRED") // câu không dẫn nguồn thật trong các đoạn được cấp: không lưu
			continue
		}
		in := exam.QuestionIn{Type: a.Type, Title: titleOf(a.Stem), Topic: p.Topic, Difficulty: p.Difficulty, Stem: strings.TrimSpace(a.Stem), Value: a.Value}
		if e := strings.TrimSpace(a.Explanation); e != "" {
			in.Explanation = &e
		}
		if a.Type != exam.TypeTrueFalse {
			for k, o := range a.Options {
				in.Options = append(in.Options, exam.OptionIn{Body: strings.TrimSpace(o.Body)})
				if o.Correct {
					in.Correct = append(in.Correct, k)
				}
			}
		}
		c, errs, _ := exam.Validate(in)
		if len(errs) > 0 {
			skip("INVALID")
			continue
		}
		norm := quiz.NormalizeShort(c.Stem)
		if seen[norm] {
			skip("DUPLICATE")
			continue
		}
		seen[norm] = true
		ok = append(ok, ready{c, cites})
	}
	j.Progress(70)
	if len(ok) > 0 {
		tx, err := w.Pool.Begin(ctx)
		if err != nil {
			return nil, fmt.Errorf("practice: mở transaction: %w", err)
		}
		defer func() { _ = tx.Rollback(ctx) }()
		qt := q.WithTx(tx)
		svc := &exam.Service{}
		for _, r := range ok {
			job := j.ID
			row, err := svc.InsertDraft(ctx, qt, p.CourseID, owner, r.c, store.QuestionOriginGENERATED, &job)
			if err != nil {
				return nil, err
			}
			cj, _ := json.Marshal(r.cites)
			if err := qt.PracticeQuestionSetCitations(ctx, store.PracticeQuestionSetCitationsParams{CourseID: p.CourseID, ID: row.ID, Citations: cj}); err != nil {
				return nil, fmt.Errorf("practice: ghi trích dẫn: %w", err)
			}
			res.Created = append(res.Created, row.ID)
		}
		if _, err := outbox.Write(ctx, tx, exam.TopicQuestionReviewed, map[string]any{"course_id": p.CourseID}); err != nil {
			return nil, fmt.Errorf("practice: outbox: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("practice: chốt lô câu: %w", err)
		}
	}
	return res, nil
}
