package grading

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/grade"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/jobs"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/store"
)

// KindExtract là loại việc nền trích công thức (jobs.kind).
const KindExtract = "grade_scheme.extract"

// Mã lỗi của việc nền (jobs.error.code).
const (
	CodeLLMUnavailable   = "LLM_UNAVAILABLE"
	CodeExtractionBad    = "EXTRACTION_INVALID"
	CodeDraftExists      = "DRAFT_EXISTS"
	maxAttempts          = 3
	defaultMaxChars      = 60000
	defaultTimeout       = 90 * time.Second
	defaultAttemptTimout = 45 * time.Second
)

// Structurer là phần của llm.Gateway mà việc trích dùng: đúng MỘT lời gọi `Structured` mỗi lần thử (test thay bằng bản giả).
type Structurer interface {
	Structured(ctx context.Context, r llm.Request, schema json.RawMessage) (json.RawMessage, error)
}

// Config là hằng số có tên của việc trích (SRS 4.1).
type Config struct {
	MaxChars       int           // GRADE_EXTRACT_MAX_CHARS (rune)
	Timeout        time.Duration // GRADE_EXTRACT_TIMEOUT: cả việc, kèm mọi lần thử lại
	AttemptTimeout time.Duration // GRADE_EXTRACT_ATTEMPT_TIMEOUT: ctx con của MỖI lần thử (gateway mặc định 30 s khi ctx không có hạn — TLR-18)
	// RetryWait: chờ giữa các lần thử lại lỗi tạm (mặc định 2 s, 5 s).
	RetryWait []time.Duration
}

func (c Config) maxChars() int {
	if c.MaxChars > 0 {
		return c.MaxChars
	}
	return defaultMaxChars
}

func (c Config) timeout() time.Duration {
	if c.Timeout > 0 {
		return c.Timeout
	}
	return defaultTimeout
}

func (c Config) attempt() time.Duration {
	if c.AttemptTimeout > 0 {
		return c.AttemptTimeout
	}
	return defaultAttemptTimout
}

func (c Config) wait(i int) time.Duration {
	w := c.RetryWait
	if w == nil {
		w = []time.Duration{2 * time.Second, 5 * time.Second}
	}
	if len(w) == 0 {
		return 0
	}
	return w[min(i, len(w)-1)]
}

// Service là nghiệp vụ trích công thức. Không giữ trạng thái giữa các yêu cầu.
type Service struct {
	Pool *pgxpool.Pool
	LLM  Structurer
	Jobs *jobs.Service
	Cfg  Config
	Log  *slog.Logger
}

// Payload là thân việc nền. DraftVersion = `version` của bản DRAFT lúc gọi `extract` (0 nếu chưa có) để lưu có thể phát hiện sửa chen ngang (TLR-10).
type Payload struct {
	CourseID     uuid.UUID `json:"course_id"`
	DocumentID   uuid.UUID `json:"document_id"`
	ReplaceDraft bool      `json:"replace_draft"`
	DraftVersion int       `json:"draft_version"`
}

// Result là `jobs.result` của việc trích.
type Result struct {
	SchemeID    uuid.UUID `json:"scheme_id"`
	SchemeNo    int       `json:"scheme_no"`
	UnclearCode string    `json:"unclear_codes,omitempty"`
}

func notFound() *apierr.Error { return apierr.New(http.StatusNotFound, apierr.NotFound) }

// Request kiểm đầu vào rồi xếp việc (SRS 4.2.1): tài liệu thuộc lớp, COURSE_POLICY, READY, độ dài ≤ MaxChars; có DRAFT chỉnh tay mà không replace_draft → 409 DRAFT_EXISTS.
// Quyền (chỉ TEACHER) do guard ở handler; gọi hàm này nghĩa là người gọi đã là TEACHER của lớp.
func (s *Service) Request(ctx context.Context, actor, courseID, documentID uuid.UUID, replace bool) (uuid.UUID, error) {
	q := store.New(s.Pool)
	if err := s.checkDoc(ctx, q, courseID, documentID); err != nil {
		return uuid.Nil, err
	}
	var draftVersion int
	d, err := q.GradeDraftGet(ctx, courseID)
	switch {
	case err == nil:
		draftVersion = int(d.Version)
		if !replace {
			n, err := q.GradeDraftEdited(ctx, store.GradeDraftEditedParams{CourseID: courseID, SchemeID: d.ID})
			if err != nil {
				return uuid.Nil, fmt.Errorf("grading: đếm mục đã sửa: %w", err)
			}
			if n > 0 || d.Version > 1 {
				return uuid.Nil, apierr.New(http.StatusConflict, apierr.DraftExists).WithDetails(map[string]any{"edited_items": int(n)})
			}
		}
	case !errors.Is(err, pgx.ErrNoRows):
		return uuid.Nil, fmt.Errorf("grading: đọc bản nháp: %w", err)
	}
	j, err := s.Jobs.Enqueue(ctx, actor, KindExtract, Payload{CourseID: courseID, DocumentID: documentID, ReplaceDraft: replace, DraftVersion: draftVersion})
	if err != nil {
		return uuid.Nil, fmt.Errorf("grading: xếp việc: %w", err)
	}
	return j.ID, nil
}

func (s *Service) checkDoc(ctx context.Context, q *store.Queries, courseID, documentID uuid.UUID) error {
	doc, err := q.GradeDocForExtract(ctx, store.GradeDocForExtractParams{DocumentID: documentID, CourseID: courseID})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return notFound()
	case err != nil:
		return fmt.Errorf("grading: đọc tài liệu: %w", err)
	case doc.Type != store.DocumentTypeCOURSEPOLICY:
		return apierr.New(http.StatusUnprocessableEntity, apierr.DocumentNotPolicy)
	case doc.Status != store.DocumentStatusREADY:
		return apierr.New(http.StatusConflict, apierr.DocumentNotReady)
	case doc.Chars > int64(s.Cfg.maxChars()):
		return apierr.New(http.StatusUnprocessableEntity, apierr.PolicyTooLong)
	}
	return nil
}

// Register gắn loại việc vào jobs.Runner.
func (s *Service) Register(r *jobs.Runner) {
	r.Register(KindExtract, func(ctx context.Context, j jobs.JobCtx) (any, error) {
		return s.Run(ctx, j.OwnerID, j.ID, j.Payload, j.Progress)
	})
}

func userErr(code, msg string) error { return &jobs.UserError{Code: code, Message: msg} }

// Run chạy một lần trích (SRS 4.2.2). Idempotent: việc nhận lại (worker bị giết) không tạo bản nháp thứ hai (`grade_schemes.extract_job_id` duy nhất).
func (s *Service) Run(ctx context.Context, owner, jobID uuid.UUID, raw json.RawMessage, progress func(int)) (any, error) {
	progress(5)
	var p Payload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, userErr(CodeExtractionBad, "Yêu cầu trích công thức không đọc được.")
	}
	q := store.New(s.Pool)
	if id, err := q.GradeSchemeByJob(ctx, store.GradeSchemeByJobParams{CourseID: p.CourseID, JobID: &jobID}); err == nil {
		return s.resultOf(ctx, q, p.CourseID, id)
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("grading: kiểm việc đã lưu: %w", err)
	}
	if err := s.checkDoc(ctx, q, p.CourseID, p.DocumentID); err != nil {
		var ae *apierr.Error
		if errors.As(err, &ae) {
			return nil, userErr(ae.Code, ae.Message)
		}
		return nil, err
	}
	doc, err := q.GradeDocForExtract(ctx, store.GradeDocForExtractParams{DocumentID: p.DocumentID, CourseID: p.CourseID})
	if err != nil {
		return nil, fmt.Errorf("grading: đọc tài liệu: %w", err)
	}
	rows, err := q.GradeDocPages(ctx, p.DocumentID)
	if err != nil {
		return nil, fmt.Errorf("grading: đọc trang: %w", err)
	}
	pages := groupPages(rows)
	progress(15)

	jctx, cancel := context.WithTimeout(llm.WithIdentity(ctx, llm.Identity{UserID: &owner, CourseID: &p.CourseID}), s.Cfg.timeout())
	defer cancel()
	progress(20)
	out, err := s.callLLM(jctx, pages)
	if err != nil {
		return nil, err
	}
	progress(80)

	var ex Extraction
	dec := json.NewDecoder(bytes.NewReader(out))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&ex); err != nil || dec.More() {
		return nil, userErr(CodeExtractionBad, "Không đọc được công thức từ quy chế. Thử lại hoặc nhập tay.")
	}
	progress(90)
	d, err := buildDraft(ex)
	if err != nil {
		return nil, userErr(CodeExtractionBad, "Không đọc được công thức từ quy chế. Thử lại hoặc nhập tay.")
	}
	cited, err := s.verifyCitations(ctx, q, p.DocumentID, d)
	if err != nil {
		return nil, err
	}
	unclear, _ := grade.Check(d.draft, cited)
	unclear = append(append(unclear, d.static...), llmUnclear(ex)...)
	return s.save(ctx, owner, jobID, p, doc.Title, d, out, unclear)
}

func (s *Service) resultOf(ctx context.Context, q *store.Queries, courseID, schemeID uuid.UUID) (any, error) {
	d, err := q.GradeDraftGet(ctx, courseID)
	if err != nil || d.ID != schemeID {
		return Result{SchemeID: schemeID}, nil //nolint:nilerr // bản nháp đã được xác nhận / thay: việc coi như xong
	}
	return Result{SchemeID: schemeID, SchemeNo: int(d.SchemeNo)}, nil
}

// groupPages gộp các đoạn thành văn bản theo trang (thứ tự trang tăng dần).
func groupPages(rows []store.GradeDocPagesRow) []Page {
	var out []Page
	for _, r := range rows {
		if n := len(out); n > 0 && out[n-1].No == int(r.PageNo) {
			out[n-1].Text += "\n" + r.Text
			continue
		}
		out = append(out, Page{No: int(r.PageNo), Text: r.Text})
	}
	return out
}

// callLLM gọi đúng một llm.Structured mỗi lần thử; chỉ lỗi TẠM (ErrUnavailable do nhà cung cấp, quá tải, hết hạn của riêng lần thử) mới thử lại, tối đa maxAttempts (TLR-11).
// Gateway đã tự chuyển nhà cung cấp dự phòng khi đầu ra sai schema, nên ở đây KHÔNG gọi lại kèm "nhắc schema".
func (s *Service) callLLM(ctx context.Context, pages []Page) (json.RawMessage, error) {
	if s.LLM == nil {
		return nil, userErr(CodeLLMUnavailable, "AI đang không dùng được. Hãy thử lại sau.")
	}
	temp := 0.0
	req := llm.Request{Task: llm.TaskUtility, Params: llm.Params{Temperature: &temp, MaxTokens: 4096},
		Messages: []llm.Message{{Role: "system", Content: SystemPrompt}, {Role: "user", Content: BuildUser(pages)}}}
	for i := range maxAttempts {
		actx, cancel := context.WithTimeout(ctx, s.Cfg.attempt())
		out, err := s.LLM.Structured(actx, req, ExtractSchema)
		attemptExpired := errors.Is(actx.Err(), context.DeadlineExceeded) && ctx.Err() == nil
		cancel()
		if err == nil {
			return out, nil
		}
		if ctx.Err() != nil {
			return nil, userErr(CodeLLMUnavailable, "AI trả lời quá lâu. Hãy thử lại sau.")
		}
		var over *llm.ErrOverloaded
		var unav *llm.ErrUnavailable
		transient := attemptExpired || errors.As(err, &over) || (errors.As(err, &unav) && unav.Reason != llm.ReasonBudgetExhausted) || errors.Is(err, llm.ErrAllProvidersFailed)
		if !transient {
			if errors.Is(err, llm.ErrBadRequest) {
				return nil, userErr(CodeExtractionBad, "Không đọc được công thức từ quy chế.")
			}
			return nil, userErr(CodeLLMUnavailable, "AI đang không dùng được. Hãy thử lại sau.")
		}
		if i == maxAttempts-1 {
			break
		}
		if w := s.Cfg.wait(i); w > 0 {
			t := time.NewTimer(w)
			select {
			case <-t.C:
			case <-ctx.Done():
				t.Stop()
				return nil, userErr(CodeLLMUnavailable, "AI trả lời quá lâu. Hãy thử lại sau.")
			}
		}
	}
	return nil, userErr(CodeLLMUnavailable, "AI đang không dùng được. Hãy thử lại sau.")
}

// built là bản nháp đã chuẩn hoá để lưu: dữ liệu cho bộ kiểm + hàng DB + mục chưa rõ tĩnh (không suy ra lại được từ dữ liệu đã lưu).
type built struct {
	draft  grade.Draft
	comps  []compRow
	rules  []ruleRow
	static []grade.Unclear
}

type compRow struct {
	key, name, grp string
	weight         decimal.Decimal
	cites          []grade.Citation
}

type ruleRow struct {
	kind       string
	applicable bool
	params     json.RawMessage
	quote      string
	page       int
}

var validKinds = map[string]bool{"BONUS": true, "PENALTY": true, "ROUNDING": true, "LETTER_SCALE": true, "FAIL_CONDITION": true, "MISSING_SCORE": true} //nolint:gochecknoglobals // tập tra cứu bất biến

// buildDraft chuyển đầu ra LLM sang dữ liệu lưu được. Giá trị không lưu nổi (khoá / tên sai dạng, loại luật lạ) → lỗi EXTRACTION_INVALID;
// trọng số không phải số hợp lệ, khoá / luật trùng → lưu phần còn lại và thêm mục chưa rõ PARAM_INVALID.
func buildDraft(ex Extraction) (built, error) {
	var b built
	cit := map[string]grade.Citation{}
	for _, c := range ex.Citations {
		cit[c.ID] = grade.Citation{Page: c.Page, Quote: c.Quote}
	}
	refs := func(ids []string) []grade.Citation {
		var out []grade.Citation
		for _, id := range ids {
			out = append(out, cit[id]) // id lạ → {0, ""}: không khớp ở bước kiểm trích dẫn
		}
		return out
	}
	seen := map[string]bool{}
	for _, c := range ex.Components {
		if n := utf8.RuneCountInString(c.Name); n < 1 || n > 80 || !validKey(c.Key) || (c.Group != "QT" && c.Group != "CK") {
			return built{}, errors.New("thành phần sai dạng")
		}
		if seen[c.Key] {
			b.static = append(b.static, grade.Unclear{Source: grade.SourceCode, Code: grade.CodeParamInvalid, Target: "components." + c.Key, Message: "Thành phần bị khai hai lần."})
			continue
		}
		seen[c.Key] = true
		w, err := decimal.NewFromString(c.Weight)
		if err != nil || w.Sign() < 0 || w.GreaterThan(grade.TotalWeight) || w.Exponent() < -2 {
			b.static = append(b.static, grade.Unclear{Source: grade.SourceCode, Code: grade.CodeParamInvalid, Target: "components." + c.Key, Message: "Trọng số không phải số hợp lệ."})
			w = decimal.Zero
		}
		cs := refs(c.CitationIDs)
		b.comps = append(b.comps, compRow{key: c.Key, name: c.Name, grp: c.Group, weight: w, cites: cs})
		b.draft.Components = append(b.draft.Components, grade.DraftComponent{Key: c.Key, Name: c.Name, Group: grade.Group(c.Group), Weight: w.String(), Citations: cs})
	}
	kinds := map[string]bool{}
	for _, r := range ex.Rules {
		if !validKinds[r.Kind] || len(bytes.TrimSpace(r.Params)) == 0 {
			return built{}, errors.New("luật sai dạng")
		}
		if r.Kind != "FAIL_CONDITION" {
			if kinds[r.Kind] {
				b.static = append(b.static, grade.Unclear{Source: grade.SourceCode, Code: grade.CodeParamInvalid, Target: "rules." + r.Kind, Message: "Luật bị khai hai lần."})
				continue
			}
			kinds[r.Kind] = true
		}
		var q string
		var pg int
		if cs := refs(r.CitationIDs); len(cs) > 0 {
			q, pg = cs[0].Quote, cs[0].Page
		}
		b.rules = append(b.rules, ruleRow{kind: r.Kind, applicable: r.Applicable, params: r.Params, quote: q, page: pg})
		b.draft.Rules = append(b.draft.Rules, grade.DraftRule{Kind: grade.RuleKind(r.Kind), Applicable: r.Applicable, Params: r.Params, Quote: q, Page: pg})
	}
	return b, nil
}

func validKey(k string) bool {
	if k == "" || len(k) > 40 || k[0] < 'a' || k[0] > 'z' {
		return false
	}
	for _, c := range k {
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '_' {
			return false
		}
	}
	return true
}

// verifyCitations tra SQL một lần cho mọi trích dẫn (4.2.6): NFC → gộp khoảng trắng → vn_fold → hạ chữ thường, trên văn bản ĐÚNG trang. Không viết lại vn_fold ở Go.
// Trích dẫn đã là đầu ra SAU bước gỡ che danh tính của gateway (TLR-19).
func (s *Service) verifyCitations(ctx context.Context, q *store.Queries, docID uuid.UUID, b built) (func(grade.Citation) bool, error) {
	type key struct {
		page  int
		quote string
	}
	idx := map[key]int{}
	var list []map[string]any
	add := func(c grade.Citation) {
		k := key{c.Page, c.Quote}
		if _, ok := idx[k]; !ok && c.Page > 0 && strings.TrimSpace(c.Quote) != "" {
			idx[k] = len(list)
			list = append(list, map[string]any{"page": c.Page, "quote": c.Quote})
		}
	}
	for _, c := range b.draft.Components {
		for _, ct := range c.Citations {
			add(ct)
		}
	}
	for _, r := range b.draft.Rules {
		add(grade.Citation{Page: r.Page, Quote: r.Quote})
	}
	found := map[key]bool{}
	if len(list) > 0 {
		raw, _ := json.Marshal(list)
		rows, err := q.GradeCitationsCheck(ctx, store.GradeCitationsCheckParams{Cites: raw, DocumentID: docID})
		if err != nil {
			return nil, fmt.Errorf("grading: kiểm trích dẫn: %w", err)
		}
		for k, i := range idx {
			for _, r := range rows {
				if int(r.Idx) == i {
					found[k] = r.Found
				}
			}
		}
	}
	return func(c grade.Citation) bool { return found[key{c.Page, c.Quote}] }, nil
}

func llmUnclear(ex Extraction) []grade.Unclear {
	var out []grade.Unclear
	for _, u := range ex.Unclear {
		out = append(out, grade.Unclear{Source: grade.SourceLLM, Target: u.Target, Message: u.Message})
	}
	return out
}

// save ghi bản nháp trong MỘT giao dịch (SRS 4.2.7): kiểm lại DRAFT hiện có (TLR-10), thay bản nháp chưa chỉnh / theo replace_draft, thêm công thức mới `scheme_no = max + 1`,
// thành phần, MỘT item mặc định mỗi thành phần, luật, audit. Trùng `extract_job_id` (nhận lại việc / hai worker) → coi như đã xong. KHÔNG bao giờ đổi trạng thái sang CONFIRMED.
func (s *Service) save(ctx context.Context, owner, jobID uuid.UUID, p Payload, title string, b built, rawOut json.RawMessage, unclear []grade.Unclear) (any, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("grading: mở giao dịch: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)

	if _, err := tx.Exec(ctx, `select pg_advisory_xact_lock(hashtextextended($1, 0))`, "grade_scheme:"+p.CourseID.String()); err != nil {
		return nil, fmt.Errorf("grading: khoá lớp: %w", err)
	}
	if id, err := q.GradeSchemeByJob(ctx, store.GradeSchemeByJobParams{CourseID: p.CourseID, JobID: &jobID}); err == nil {
		return Result{SchemeID: id}, nil // đã có bản nháp của chính việc này
	}
	replaced := false
	cur, err := q.GradeDraftLock(ctx, p.CourseID)
	switch {
	case err == nil:
		edited, err := q.GradeDraftEdited(ctx, store.GradeDraftEditedParams{CourseID: p.CourseID, SchemeID: cur.ID})
		if err != nil {
			return nil, fmt.Errorf("grading: đếm mục đã sửa: %w", err)
		}
		changed := edited > 0 || cur.Version > 1
		if (p.ReplaceDraft && int(cur.Version) != p.DraftVersion) || (!p.ReplaceDraft && changed) {
			return nil, userErr(CodeDraftExists, "Bản nháp vừa được sửa; trích lại sẽ thay bản đó.")
		}
		if err := q.GradeDraftDelete(ctx, store.GradeDraftDeleteParams{CourseID: p.CourseID, ID: cur.ID}); err != nil {
			return nil, fmt.Errorf("grading: thay bản nháp: %w", err)
		}
		replaced = true
	case !errors.Is(err, pgx.ErrNoRows):
		return nil, fmt.Errorf("grading: khoá bản nháp: %w", err)
	}

	no, err := q.GradeSchemeNextNo(ctx, p.CourseID)
	if err != nil {
		return nil, fmt.Errorf("grading: số công thức: %w", err)
	}
	extraction, _ := json.Marshal(map[string]any{"llm": json.RawMessage(rawOut), "unclear_items": unclear, "trace_id": llm.TraceID(ctx)})
	doc := p.DocumentID
	id, err := q.GradeSchemeInsert(ctx, store.GradeSchemeInsertParams{CourseID: p.CourseID, SchemeNo: no, SourceDocumentID: &doc, SourceTitle: &title, Extraction: extraction, ExtractJobID: &jobID, CreatedBy: owner})
	if err != nil {
		var pg *pgconn.PgError
		if errors.As(err, &pg) && pg.Code == "23505" {
			return nil, userErr(CodeDraftExists, "Bản nháp vừa được sửa; trích lại sẽ thay bản đó.")
		}
		return nil, fmt.Errorf("grading: lưu công thức: %w", err)
	}
	for i, c := range b.comps {
		cites := make([]map[string]any, len(c.cites))
		for j, ct := range c.cites {
			cites[j] = map[string]any{"page": ct.Page, "quote": ct.Quote}
		}
		craw, _ := json.Marshal(cites)
		cid, err := q.GradeComponentInsert(ctx, store.GradeComponentInsertParams{CourseID: p.CourseID, SchemeID: id, Key: c.key, Name: c.name, Grp: store.GradeGroup(c.grp), Weight: c.weight, Position: int32(i), Citations: craw}) //nolint:gosec // i < 20
		if err != nil {
			return nil, fmt.Errorf("grading: lưu thành phần: %w", err)
		}
		if _, err := q.GradeItemInsert(ctx, store.GradeItemInsertParams{CourseID: p.CourseID, SchemeID: id, ComponentID: cid, Key: c.key, Name: c.name, MaxScore: grade.MaxScale, Source: store.GradeItemSourceMANUAL, Position: int32(i)}); err != nil { //nolint:gosec // i < 20
			return nil, fmt.Errorf("grading: lưu cột điểm: %w", err)
		}
	}
	for _, r := range b.rules {
		var quote *string
		var page *int32
		if r.quote != "" {
			quote = &r.quote
			pg := int32(r.page) //nolint:gosec // số trang nhỏ
			page = &pg
		}
		if err := q.GradeRuleInsert(ctx, store.GradeRuleInsertParams{CourseID: p.CourseID, SchemeID: id, Kind: store.GradeRuleKind(r.kind), Applicable: r.applicable, Params: r.params, SourceQuote: quote, SourcePage: page}); err != nil {
			return nil, fmt.Errorf("grading: lưu luật: %w", err)
		}
	}
	action := "extract"
	if replaced {
		action = "extract_replace_draft"
	}
	after, _ := json.Marshal(map[string]any{"scheme_no": no, "document_id": p.DocumentID, "unclear_codes": grade.CodesOf(unclear)})
	if _, err := q.InsertAuditLog(ctx, store.InsertAuditLogParams{CourseID: &p.CourseID, ActorID: &owner, Entity: "grade_schemes", EntityID: id.String(), Action: action, After: after}); err != nil {
		return nil, fmt.Errorf("grading: audit: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("grading: lưu công thức: %w", err)
	}
	return Result{SchemeID: id, SchemeNo: int(no), UnclearCode: grade.CodesOf(unclear)}, nil
}
