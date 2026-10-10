package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/privacy"
	"github.com/edupilot/backend-go/internal/rag"
)

// RagSimFloor là RAG_SIM_FLOOR: cos_top1 dưới sàn = không có ngữ cảnh (bản tạm, hiệu chỉnh ở E2).
const RagSimFloor = 0.25

// Retriever là rag.Service (SearchStudent): tìm trong tài liệu sinh viên được thấy, audience ALL.
type Retriever interface {
	SearchStudent(ctx context.Context, q rag.Query) ([]rag.Hit, error)
}

// Generator là cổng sinh chữ (llm.Client.Stream). Agent gọi nó TỐI ĐA một lần mỗi tin nhắn.
type Generator interface {
	Stream(ctx context.Context, r llm.Request) (<-chan llm.Chunk, error)
}

// Agent định tuyến tất định: phân loại một lần → Route → (câu mẫu | tool | truy xuất) → đúng MỘT lần sinh chữ.
type Agent struct {
	An      *Analyzer
	Private *PrivateRegistry
	Schemes GradeSchemeSource // nil = chưa có công thức (P6)
	Rag     Retriever
	Gen     Generator
	Cache   *AnswerCache // nil = không cache
	Events  EventSink    // nil = không ghi sự kiện
	Log     *slog.Logger
	// SupportResources là SUPPORT_RESOURCES_VI; rỗng → DefaultSupport.
	SupportResources string
	TopK             int
}

// Input là một tin nhắn cần trả lời. History là các lượt gần nhất (còn nguyên chữ — hook che ở internal/llm sẽ che lại).
type Input struct {
	Text       string
	History    []llm.Message
	DocumentID *uuid.UUID // phiên "Hỏi AI về tài liệu này": giới hạn truy xuất
	// OnStage (tuỳ chọn) được gọi khi bắt đầu một giai đoạn người dùng thấy được: "searching" trước truy xuất.
	OnStage func(stage string)
}

// Outcome là kết quả định tuyến. Đúng một trong: Canned (câu mẫu, 0 lời gọi LLM) hoặc Stream (một lần sinh chữ).
type Outcome struct {
	Class     Classification
	Plan      Intent
	Canned    string
	Cached    bool
	Stream    <-chan llm.Chunk
	Blocks    []Block
	Hits      []rag.Hit
	Sources   []Source
	NoContext bool
	// CacheKey khác rỗng: chat ghi câu trả lời vào cache sau khi sinh xong (COURSE_QA / LIBRARY_SEARCH không PII).
	CacheKey string
	Request  llm.Request
}

// Respond xử lý một tin nhắn. Danh tính chỉ lấy từ tc.
func (a *Agent) Respond(ctx context.Context, tc TrustedContext, in Input) (Outcome, error) {
	cl, err := a.An.Analyze(ctx, tc, in.Text)
	if err != nil {
		return Outcome{}, err
	}
	out := Outcome{Class: cl, Plan: cl.Intent}
	switch cl.Intent {
	case IntentCrisis:
		out.Canned = a.crisisReply()
		return out, nil // không log nội dung, không ghi pii_events, không báo ai
	case IntentOtherPerson:
		a.recordBlocked(ctx, tc)
		out.Canned = ReplyOtherPerson
		return out, nil
	case IntentGradeFormula:
		return a.formula(ctx, tc, in, out)
	case IntentAttendance:
		return a.tool(ctx, tc, in, out, "get_my_attendance", nil, "điểm danh")
	case IntentParticipation:
		return a.tool(ctx, tc, in, out, "get_my_participation", nil, "điểm cộng")
	case IntentGrade:
		return a.tool(ctx, tc, in, out, "get_my_grade_summary", nil, "điểm")
	case IntentWhatIf:
		return a.tool(ctx, tc, in, out, "what_if_final_grade", whatIfArgs(in.Text), "điểm")
	case IntentExamSchedule:
		return a.tool(ctx, tc, in, out, "get_exam_schedule", nil, "lịch")
	case IntentUpcoming:
		return a.tool(ctx, tc, in, out, "get_upcoming_events", json.RawMessage(`{"days":7}`), "lịch")
	case IntentLibrary:
		args, _ := json.Marshal(LibraryArgs{Query: in.Text})
		return a.tool(ctx, tc, in, out, "search_library", args, "tài liệu")
	case IntentSmalltalk:
		return a.generate(ctx, in, out, "")
	}
	return a.courseQA(ctx, tc, in, out)
}

func (a *Agent) crisisReply() string {
	sup := strings.TrimSpace(a.SupportResources)
	if sup == "" {
		sup = DefaultSupport
	}
	return ReplyCrisis + " " + sup
}

// recordBlocked ghi pii_events BLOCKED / OTHER_PERSON (không nội dung) và một dòng log không kèm tên.
func (a *Agent) recordBlocked(ctx context.Context, tc TrustedContext) {
	if a.Log != nil {
		a.Log.InfoContext(ctx, "chặn hỏi hộ người khác", "trace_id", tc.TraceID)
	}
	if a.Events != nil {
		if err := a.Events.Record(ctx, tc, "OTHER_PERSON", "BLOCKED", privacy.ChannelPrivate, 1); err != nil && a.Log != nil {
			a.Log.WarnContext(ctx, "không ghi được pii_events", "error", err.Error())
		}
	}
}

func (a *Agent) formula(ctx context.Context, tc TrustedContext, in Input, out Outcome) (Outcome, error) {
	if a.Schemes == nil {
		out.Canned = ReplyNoFormula
		return out, nil
	}
	f, ok, err := a.Schemes.Scheme(ctx, tc)
	if err != nil {
		return out, fmt.Errorf("agent: công thức điểm: %w", err)
	}
	if !ok {
		out.Canned = ReplyNoFormula
		return out, nil
	}
	out.Blocks = []Block{{Kind: "grade_scheme", Data: f}}
	return a.generate(ctx, in, out, ContextFromFacts(f))
}

func (a *Agent) tool(ctx context.Context, tc TrustedContext, in Input, out Outcome, name string, args json.RawMessage, what string) (Outcome, error) {
	res, err := a.Private.Run(ctx, name, tc, args)
	if err != nil {
		return out, fmt.Errorf("agent: tool %s: %w", name, err)
	}
	if res.NoData {
		out.Canned = ReplyNoData(what)
		return out, nil
	}
	if res.Block != nil {
		out.Blocks = append(out.Blocks, *res.Block)
	}
	out.Sources = res.Sources
	// search_library không PII có thể cache (khoá theo lớp + phiên bản tri thức); mọi tool còn lại là dữ liệu cá nhân hoặc theo người.
	if cl := out.Class; !cl.Intent.Personal() && !cl.HasPII && a.Cache != nil {
		key := a.Cache.Key(ctx, tc.CourseID, in)
		if c, ok := a.Cache.Get(ctx, key); ok {
			out.Canned, out.Cached = c.Text, true
			return out, nil
		}
		out.CacheKey = key
	}
	return a.generate(ctx, in, out, ContextFromFacts(res.Facts))
}

func (a *Agent) courseQA(ctx context.Context, tc TrustedContext, in Input, out Outcome) (Outcome, error) {
	cacheable := a.Cache != nil && !out.Class.HasPII && !out.Class.Intent.Personal()
	if cacheable {
		out.CacheKey = a.Cache.Key(ctx, tc.CourseID, in)
		if c, ok := a.Cache.Get(ctx, out.CacheKey); ok {
			out.Canned, out.Cached, out.CacheKey = c.Text, true, ""
			return out, nil
		}
	}
	if in.OnStage != nil {
		in.OnStage("searching")
	}
	q := rag.Query{CourseID: tc.CourseID, Vec: out.Class.Vec, Text: in.Text, K: a.TopK}
	if in.DocumentID != nil {
		q.DocumentIDs = []uuid.UUID{*in.DocumentID}
	}
	if q.Vec == nil && a.An.Embed != nil { // luật đã quyết (không nhúng khi phân loại) nhưng truy xuất cần vectơ: nhúng MỘT lần ở đây
		v, err := a.An.Embed(ctx, in.Text)
		if err != nil {
			return out, fmt.Errorf("agent: nhúng câu hỏi: %w", err)
		}
		q.Vec = v
	}
	hits, err := a.Rag.SearchStudent(ctx, q)
	if err != nil {
		return out, fmt.Errorf("agent: truy xuất: %w", err)
	}
	out.Hits = hits
	best := 0.0
	for _, h := range hits {
		best = max(best, h.Cosine)
	}
	if len(hits) == 0 || best < RagSimFloor {
		out.NoContext, out.Canned, out.CacheKey = true, ReplyNoContext, ""
		return out, nil
	}
	return a.generate(ctx, in, out, ContextFromHits(hits))
}

// generate là LẦN SINH CHỮ DUY NHẤT của tin nhắn.
func (a *Agent) generate(ctx context.Context, in Input, out Outcome, ctxBlock string) (Outcome, error) {
	req := llm.Request{Task: llm.TaskChat, Messages: BuildMessages(in.History, ctxBlock, in.Text)}
	for _, h := range out.Hits { // chỉ để đường suy giảm trích nguyên văn; không phải payload gửi provider
		req.Passages = append(req.Passages, llm.Passage{Text: h.Text, Source: h.Title, Score: h.Cosine})
	}
	ch, err := a.Gen.Stream(ctx, req)
	if err != nil {
		return out, err
	}
	out.Request, out.Stream = req, ch
	return out, nil
}

// whatIfArgs: Go phân tích giả định "nếu … được 8 …" (không LLM). Chưa gắn được thành phần → để rỗng, tool trả NoData.
func whatIfArgs(text string) json.RawMessage {
	f := auth.Fold(text)
	comp := map[string]string{"cuoi ky": "final", "giua ky": "midterm", "qua trinh": "process", "bai tap": "assignment"}
	assumed := map[string]string{}
	for key, id := range comp {
		if i := strings.Index(f, key); i >= 0 {
			if n := firstNumber(f[i+len(key):]); n != "" {
				assumed[id] = n
			}
		}
	}
	b, _ := json.Marshal(map[string]any{"assumed": assumed})
	return b
}

func firstNumber(s string) string {
	var b strings.Builder
	started := false
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			started = true
		case (r == ',' || r == '.') && started:
			b.WriteRune('.')
		case started:
			return strings.TrimRight(b.String(), ".")
		}
	}
	return strings.TrimRight(b.String(), ".")
}

// AnswerCache là cache câu trả lời theo (lớp, phiên bản tri thức, băm câu hỏi chuẩn hoá) — chỉ cho COURSE_QA / LIBRARY_SEARCH không PII (SRS 5.7).
type AnswerCache struct {
	Redis RedisKV
	// Version trả phiên bản tri thức của lớp (`ep:rag:ver:{course}`); đổi khi tài liệu đổi nên khoá cũ không trúng.
	Version func(ctx context.Context, course uuid.UUID) (int64, error)
	TTL     int // giây; 0 = 3600
}

// CachedAnswer là giá trị cache.
type CachedAnswer struct {
	Text      string   `json:"text"`
	Citations []string `json:"citations,omitempty"`
}

// RedisKV là phần tối thiểu của Redis mà cache dùng.
type RedisKV interface {
	GetString(ctx context.Context, key string) (string, bool, error)
	SetString(ctx context.Context, key, val string, ttlSec int) error
}

var errNoCache = errors.New("agent: không cache")

// Key dựng khoá cache; phiên có document_id có khoá riêng.
func (c *AnswerCache) Key(ctx context.Context, course uuid.UUID, in Input) string {
	ver := int64(0)
	if c.Version != nil {
		if v, err := c.Version(ctx, course); err == nil {
			ver = v
		}
	}
	norm := strings.Join(strings.Fields(auth.Fold(in.Text)), " ")
	scope := ""
	if in.DocumentID != nil {
		scope = in.DocumentID.String()
	}
	sum := sha256.Sum256([]byte(scope + "|" + norm))
	return fmt.Sprintf("ep:ans:%s:%d:%s", course, ver, hex.EncodeToString(sum[:]))
}

// Get đọc cache; lỗi Redis coi như trượt.
func (c *AnswerCache) Get(ctx context.Context, key string) (CachedAnswer, bool) {
	raw, ok, err := c.Redis.GetString(ctx, key)
	if err != nil || !ok {
		return CachedAnswer{}, false
	}
	var v CachedAnswer
	if json.Unmarshal([]byte(raw), &v) != nil || v.Text == "" {
		return CachedAnswer{}, false
	}
	return v, true
}

// Put ghi cache (chat gọi sau khi sinh xong và chỉ khi Outcome.CacheKey khác rỗng).
func (c *AnswerCache) Put(ctx context.Context, key string, v CachedAnswer) error {
	if key == "" {
		return errNoCache
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("agent: mã hoá cache: %w", err)
	}
	ttl := c.TTL
	if ttl <= 0 {
		ttl = 3600
	}
	return c.Redis.SetString(ctx, key, string(raw), ttl)
}

// StoreAnswer ghi câu trả lời đã sinh xong vào cache (key = Outcome.CacheKey). Không cache → bỏ qua.
func (a *Agent) StoreAnswer(ctx context.Context, key, text string) {
	if a.Cache == nil || key == "" {
		return
	}
	if err := a.Cache.Put(ctx, key, CachedAnswer{Text: text}); err != nil && a.Log != nil {
		a.Log.WarnContext(ctx, "agent: ghi cache câu trả lời lỗi", "error", err)
	}
}
