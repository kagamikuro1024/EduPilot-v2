package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/agent"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/privacy"
	"github.com/edupilot/backend-go/internal/rag"
	"github.com/edupilot/backend-go/internal/store"
)

var (
	errCancelled = errors.New("chat: người dùng bấm Dừng")
	errDrain     = errors.New("chat: gateway đang tắt")
)

// Chữ cố định của đường suy giảm (AC8): KHÔNG dùng câu của gói llm (hứa "giảng viên sẽ xem").
const (
	DegradedNotice   = "Trả lời tạm thời, trích nguyên văn từ tài liệu của lớp."
	DegradedNoContxt = "AI đang gián đoạn. Thử lại sau."
)

type run struct {
	sess store.ChatSession
	user store.ChatMessage
	asst store.ChatMessage
	text string
	busy string // giá trị khoá CHAT_BUSY của lượt này
	free sync.Once

	// điền khi start
	cancel context.CancelCauseFunc
	trace  string
}

// start đăng ký kênh huỷ và bộ đệm TRƯỚC khi phát `status{received}` (nhờ vậy PUBLISH huỷ = 0 người nghe nghĩa là không có G), rồi chạy G.
// G dùng ctx tách khỏi request (rớt kết nối không huỷ sinh chữ) nhưng giữ danh tính, trace và hạn StreamMax.
func (s *Service) start(reqCtx context.Context, a Actor, r *run) error {
	base := context.WithoutCancel(reqCtx)
	base = llm.WithIdentity(base, llm.Identity{UserID: &a.UserID, CourseID: &r.sess.CourseID})
	base = privacy.WithSession(base, privacy.NewSession("chat:"+r.sess.ID.String()))
	tctx, tcancel := context.WithTimeout(base, s.c().StreamMax)
	ctx, cancel := context.WithCancelCause(tctx)
	r.cancel = func(err error) { cancel(err); tcancel() }
	r.trace = a.TraceID

	var ps *goredis.PubSub
	if s.Redis != nil {
		sub := s.Redis.Subscribe(ctx, cancelChan(r.asst.ID))
		rctx, rc := context.WithTimeout(ctx, time.Second)
		_, err := sub.Receive(rctx)
		rc()
		if err != nil {
			_ = sub.Close()
		} else {
			ps = sub
		}
	}
	s.emit(ctx, r.asst.ID, r.asst.Attempt, EvStatus, map[string]any{"message_id": r.asst.ID, "stage": "received"})

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer r.cancel(nil)
		if ps != nil {
			defer func() { _ = ps.Close() }()
			go func() {
				select {
				case <-ps.Channel():
					cancel(errCancelled)
				case <-ctx.Done():
				}
			}()
		}
		go func() {
			select {
			case <-s.Drain:
				cancel(errDrain)
			case <-ctx.Done():
			}
		}()
		s.generate(ctx, a, r)
	}()
	return nil
}

type result struct {
	text      string
	runes     int
	resp      *llm.Response
	degraded  bool
	streamErr error
}

func (s *Service) generate(ctx context.Context, a Actor, r *run) {
	mid, att := r.asst.ID, r.asst.Attempt
	tc := agent.TrustedContext{UserID: a.UserID, CourseID: r.sess.CourseID, Role: string(a.Role), SessionID: r.sess.ID, TraceID: a.TraceID}
	defer s.release(ctx, a, r)

	in := agent.Input{Text: r.text, History: s.history(ctx, r), DocumentID: r.sess.DocumentID,
		OnStage: func(st string) { s.emit(ctx, mid, att, EvStatus, map[string]any{"message_id": mid, "stage": st}) }}
	out, err := s.Agent.Respond(ctx, tc, in)
	if err != nil {
		s.fail(ctx, a, r, "", result{}, err)
		return
	}
	intent := string(out.Plan)
	for _, b := range out.Blocks {
		s.emit(ctx, mid, att, EvBlock, b)
	}
	res := result{}
	switch {
	case out.Canned != "": // câu mẫu / cache: cùng khuôn SSE — một token đủ câu
		res.text, res.runes = out.Canned, utf8.RuneCountInString(out.Canned)
		s.emit(ctx, mid, att, EvToken, map[string]any{"off": 0, "t": out.Canned})
	case out.Stream != nil:
		s.emit(ctx, mid, att, EvStatus, map[string]any{"message_id": mid, "stage": "generating"})
		res = s.pump(ctx, r, out.Stream)
		if res.streamErr != nil || ctx.Err() != nil {
			cause := res.streamErr
			if c := context.Cause(ctx); c != nil && (cause == nil || errors.Is(c, errCancelled) || errors.Is(c, errDrain)) {
				cause = c
			}
			s.fail(ctx, a, r, intent, res, cause)
			return
		}
	}
	masked := 0
	if res.resp != nil {
		masked = res.resp.MaskedCurrent
		if res.resp.Degraded || res.degraded {
			res.degraded = true
		}
	}
	text, hits := res.text, out.Hits
	var cites []Citation
	if res.degraded {
		text, cites = s.extractive(hits)
		s.emit(ctx, mid, att, EvNotice, map[string]any{"degraded": true})
		s.emit(ctx, mid, att, EvToken, map[string]any{"off": 0, "t": text})
	} else {
		text, cites = citationsFrom(text, hits)
	}
	if cites == nil {
		cites = []Citation{}
	}
	if masked > 0 {
		s.emit(ctx, mid, att, EvNotice, map[string]any{"masked": masked})
	}

	fctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	meta := out.Metadata(text, s.threshold(fctx, r.sess.CourseID), res.degraded, masked)
	ok, err := s.finish(fctx, a, r, intent, text, cites, out.Blocks, meta)
	if err != nil {
		s.fail(fctx, a, r, intent, res, err)
		return
	}
	if !ok { // bị Dừng / reaper đóng trước: im lặng
		return
	}
	if !res.degraded && !meta.LowConfidence && out.CacheKey != "" { // câu "chưa chắc" không được cache để người sau khỏi nhận lại
		if st, ok := s.Agent.(interface {
			StoreAnswer(ctx context.Context, key, text string)
		}); ok {
			st.StoreAnswer(fctx, out.CacheKey, text)
		}
	}
	d := map[string]any{"message_id": mid, "citations": cites, "low_confidence": meta.LowConfidence, "degraded": res.degraded}
	if text != res.text {
		d["content"] = text // đã bỏ [n] không có trong danh sách / thay bằng câu trích
	}
	s.release(fctx, a, r) // nhả TRƯỚC khi báo `done`: client gửi tin kế tiếp ngay khi thấy done không được dính CHAT_BUSY
	s.emit(fctx, mid, att, EvDone, d)
}

// release nhả khoá CHAT_BUSY đúng một lần cho lượt này.
func (s *Service) release(ctx context.Context, a Actor, r *run) {
	r.free.Do(func() { s.releaseBusy(context.WithoutCancel(ctx), a.UserID, r.busy) })
}

// pump đọc luồng sinh chữ: phát `token` (có `off` theo rune), ghi partial_content mỗi FlushEvery khi có thay đổi.
func (s *Service) pump(ctx context.Context, r *run, ch <-chan llm.Chunk) result {
	var b strings.Builder
	res := result{}
	tick := time.NewTicker(s.c().FlushEvery)
	defer tick.Stop()
	dirty := false
	flush := func() bool {
		if !dirty {
			return true
		}
		dirty = false
		n, err := store.New(s.Pool).SetChatPartial(context.WithoutCancel(ctx), store.SetChatPartialParams{ID: r.asst.ID, Attempt: r.asst.Attempt, PartialContent: ptr(b.String())})
		return err != nil || n > 0 // 0 hàng = đã bị đóng: dừng
	}
	for {
		select {
		case c, ok := <-ch:
			if !ok {
				res.text = b.String()
				return res
			}
			if c.Err != nil {
				res.streamErr = c.Err
				res.text = b.String()
				return res
			}
			if c.Degraded {
				res.degraded = true
				continue
			}
			if c.Text != "" {
				s.emit(ctx, r.asst.ID, r.asst.Attempt, EvToken, map[string]any{"off": res.runes, "t": c.Text})
				b.WriteString(c.Text)
				res.runes += utf8.RuneCountInString(c.Text)
				dirty = true
			}
			if c.Done {
				res.resp = c.Response
				res.text = b.String()
				return res
			}
		case <-tick.C:
			if !flush() {
				r.cancel(errCancelled)
			}
		case <-ctx.Done():
			res.text = b.String()
			return res
		}
	}
}

// threshold là courses.escalation_threshold của lớp (đổi có hiệu lực ở tin kế tiếp: đọc mỗi lần). Lỗi đọc → 0,80 (mặc định SRS) thay vì bỏ cả câu trả lời.
func (s *Service) threshold(ctx context.Context, course uuid.UUID) decimal.Decimal {
	c, err := store.New(s.Pool).GetCourse(ctx, course)
	if err != nil {
		return decimal.RequireFromString("0.80")
	}
	return c.EscalationThreshold
}

// finish: một giao dịch — ghi cuối (có điều kiện STREAMING + đúng lượt) + pii_events. 0 hàng → false.
func (s *Service) finish(ctx context.Context, a Actor, r *run, intent, text string, cites []Citation, blocks []agent.Block, meta agent.ResponseMetadata) (bool, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	cj, _ := json.Marshal(cites)
	bj, _ := json.Marshal(blocks)
	if blocks == nil {
		bj = []byte("[]")
	}
	n, err := q.FinishChatMessage(ctx, store.FinishChatMessageParams{
		ID: r.asst.ID, Attempt: r.asst.Attempt, Content: text, Citations: cj, Blocks: bj, Confidence: meta.Confidence, LowConfidence: meta.LowConfidence,
		NoContext: meta.NoContext, Degraded: meta.Degraded, MaskedCount: int32(meta.MaskedCount), Intent: nonEmpty(intent), TraceID: nonEmpty(r.trace), //nolint:gosec // đếm nhỏ
	})
	if err != nil {
		return false, err
	}
	if n == 0 {
		return false, nil
	}
	if err := q.TouchChatSession(ctx, r.sess.ID); err != nil {
		return false, err
	}
	if meta.MaskedCount > 0 {
		s.piiEvents(ctx, q, a, r)
	}
	return true, tx.Commit(ctx)
}

// piiEvents ghi MỘT dòng MASKED cho mỗi loại có trong tin của người dùng (chỉ loại + số đếm, không chữ). Lỗi chỉ log: không làm hỏng câu trả lời.
func (s *Service) piiEvents(ctx context.Context, q *store.Queries, a Actor, r *run) {
	if s.PII == nil {
		return
	}
	fs, err := s.PII.Detect(ctx, r.sess.CourseID, r.text)
	if err != nil {
		return
	}
	cnt := map[privacy.Kind]int{}
	for _, f := range fs {
		cnt[f.Kind]++
	}
	for k, n := range cnt {
		if err := q.InsertPIIEvent(ctx, store.InsertPIIEventParams{
			CourseID: r.sess.CourseID, SessionID: &r.sess.ID, UserID: a.UserID, Channel: store.ChatChannelPRIVATE,
			PiiType: store.PiiKind(k), Count: int32(n), Action: store.PiiActionMASKED, //nolint:gosec // đếm nhỏ
		}); err != nil && s.Log != nil {
			s.Log.WarnContext(ctx, "chat: ghi pii_events lỗi", "error", err)
		}
	}
}

func nonEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// fail đóng lượt ở FAILED (hoặc CANCELLED khi người dùng bấm Dừng) và phát sự kiện cuối. Mọi lệnh ghi có điều kiện STREAMING + đúng lượt.
func (s *Service) fail(ctx context.Context, a Actor, r *run, intent string, res result, cause error) {
	fctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	q := store.New(s.Pool)
	var partial *string
	if res.text != "" {
		partial = &res.text
	}
	mid, att := r.asst.ID, r.asst.Attempt
	if errors.Is(cause, errCancelled) {
		n, err := q.CancelChatMessage(fctx, store.CancelChatMessageParams{ID: mid, Attempt: att, PartialContent: partial})
		if err == nil && n > 0 {
			s.release(fctx, a, r)
			s.emit(fctx, mid, att, EvError, map[string]any{"code": "CANCELLED", "message": "Đã dừng."})
		}
		return
	}
	code, retry := classify(cause)
	if errors.Is(cause, errDrain) {
		code = "INTERRUPTED"
	}
	n, err := q.FailChatMessage(fctx, store.FailChatMessageParams{ID: mid, Attempt: att, ErrorCode: &code, PartialContent: partial, Intent: nonEmpty(intent), TraceID: nonEmpty(r.trace)})
	if err != nil {
		if s.Log != nil {
			s.Log.ErrorContext(fctx, "chat: ghi FAILED lỗi", "error", err, "code", code)
		}
		return
	}
	if n == 0 {
		return
	}
	if s.Log != nil && code != "OVERLOADED" {
		s.Log.WarnContext(fctx, "chat: lượt sinh thất bại", "code", code, "error", cause)
	}
	s.release(fctx, a, r)
	d := map[string]any{"code": code, "message": errorText(code)}
	if retry > 0 {
		d["retry_after"] = retry
	}
	s.emit(fctx, mid, att, EvError, d)
}

// classify ánh xạ lỗi sang error_code (SRS 4.7.1 bước 11) và thời gian chờ ước tính.
func classify(err error) (string, int) {
	var ov *llm.ErrOverloaded
	switch {
	case errors.As(err, &ov):
		return "OVERLOADED", max(1, int(ov.RetryAfter.Seconds()+0.5))
	case errors.Is(err, llm.ErrNotConfigured):
		return "NOT_CONFIGURED", 0
	case errors.Is(err, privacy.ErrMaskFailed):
		return "MASK_FAILED", 0
	case errors.Is(err, errDrain):
		return "INTERRUPTED", 0
	default:
		return "PROVIDER_ERROR", 0
	}
}

func (s *Service) history(ctx context.Context, r *run) []llm.Message {
	rows, err := store.New(s.Pool).ChatHistory(ctx, store.ChatHistoryParams{SessionID: r.sess.ID, BeforeAt: r.user.CreatedAt, BeforeID: r.user.ID, PageLimit: int32(s.c().History)}) //nolint:gosec // nhỏ
	if err != nil {
		return nil
	}
	out := make([]llm.Message, 0, len(rows))
	for i := len(rows) - 1; i >= 0; i-- {
		role := "user"
		if rows[i].Role == store.ChatRoleASSISTANT {
			role = "assistant"
		}
		out = append(out, llm.Message{Role: role, Content: rows[i].Content})
	}
	return out
}

// Citation là một nguồn được trích [n] trong câu trả lời (định nghĩa chung ở agent: Threads dùng lại).
type Citation = agent.Citation

func citationsFrom(text string, hits []rag.Hit) (string, []Citation) {
	return agent.ExtractCitations(text, hits)
}

func cut(s string, n int) string { return agent.Cut(s, n) }

// extractive dựng câu trả lời suy giảm: ≤ ExtractHits đoạn đầu, mỗi đoạn ≤ ExtractChars ký tự cắt ở ranh giới câu, kèm [n]; không có ngữ cảnh → câu xin lỗi (không bịa).
func (s *Service) extractive(hits []rag.Hit) (string, []Citation) {
	if len(hits) == 0 {
		return DegradedNoContxt, []Citation{}
	}
	k := min(len(hits), s.c().ExtractHits)
	var b strings.Builder
	b.WriteString(DegradedNotice)
	cites := make([]Citation, 0, k)
	for i := range k {
		h := hits[i]
		fmt.Fprintf(&b, "\n\n«%s» [%d]", cut(h.Text, s.c().ExtractChars), i+1)
		cites = append(cites, Citation{N: i + 1, DocumentID: h.DocumentID, Title: h.Title, PageNo: h.PageNo, Snippet: cut(h.Text, 200)})
	}
	return b.String(), cites
}
