package thread

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/agent"
	"github.com/edupilot/backend-go/internal/ingest"
	"github.com/edupilot/backend-go/internal/jobs"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/rag"
	"github.com/edupilot/backend-go/internal/store"
	"github.com/pgvector/pgvector-go"
)

// Lý do AI bỏ qua (khớp ràng buộc forum_threads_skip_chk).
const (
	SkipNoContext      = "NO_CONTEXT"
	SkipLowScore       = "LOW_SCORE"
	SkipLLMUnavailable = "LLM_UNAVAILABLE"
)

// RegisterKind đăng ký việc `thread.answer` vào jobs.Runner: handler CHỈ XADD vào hàng dài dùng chung `ep:ingest` rồi trả ErrDeferred —
// không gọi LLM trong consumer outbox tuần tự (nếu không, việc xoá cache roster có thể trễ quá 5 s; SRS 4.9.5).
func RegisterKind(r *jobs.Runner, rdb *appredis.Client) {
	r.Register(KindAnswer, func(ctx context.Context, j jobs.JobCtx) (any, error) {
		var p struct {
			ThreadID uuid.UUID `json:"thread_id"`
		}
		if err := json.Unmarshal(j.Payload, &p); err != nil || p.ThreadID == uuid.Nil {
			return nil, &jobs.UserError{Code: "NOT_FOUND", Message: "Không tìm thấy câu hỏi."}
		}
		if err := ingest.Enqueue(ctx, rdb, j.ID, j.OwnerID, KindAnswer, p.ThreadID); err != nil {
			return nil, err
		}
		return nil, jobs.ErrDeferred
	})
}

// OnCreated là handler outbox `thread.created`: ghi nhận và trả ngay (idempotent). Việc AI trả lời KHÔNG chạy ở đây (consumer outbox tuần tự một goroutine, TLR-9);
// nó đi bằng việc `thread.answer` (`job.enqueue` → XADD ep:ingest). Topic phải có handler: không có thì tin rơi vào dead-letter sau 4 lần.
func OnCreated(context.Context, outbox.Message) error { return nil }

// AnswerHandler là hàm xử lý của consumer ep:ingest cho loại `thread.answer` (cắm vào ingest.Queue.Extra).
func (s *Service) AnswerHandler(ctx context.Context, jobID, owner, threadID uuid.UUID) error {
	return s.Answer(ctx, jobID, owner, threadID)
}

// Answer: AI trả lời MỘT lần mỗi thread, trong việc nền. Idempotent (ai_state != PENDING → bỏ). Không đủ tin cậy → không tự trả lời, ghi SKIPPED + lý do.
// Lỗi tạm thời (LLM / nhúng / truy xuất) trả error để hàng giao lại; tới lần thử thứ 3 tự ghi SKIPPED LLM_UNAVAILABLE (thread không kẹt PENDING).
func (s *Service) Answer(ctx context.Context, jobID, owner, threadID uuid.UUID) error {
	q := store.New(s.Pool)
	t, err := q.ThreadAnswerLoad(ctx, threadID)
	if errors.Is(err, pgx.ErrNoRows) {
		return s.finish(ctx, jobID, map[string]any{"thread_id": threadID, "gone": true})
	}
	if err != nil {
		return fmt.Errorf("thread: nạp thread: %w", err)
	}
	if t.AiState != store.ThreadAiStatePENDING {
		return s.finish(ctx, jobID, map[string]any{"thread_id": threadID, "ai_state": t.AiState})
	}
	ctx = llm.WithIdentity(ctx, llm.Identity{CourseID: &t.CourseID}) // việc nền không có Principal (TLR-8)
	text := t.Title + "\n" + t.Body

	var vec []float32
	if t.Embedding != nil {
		vec = t.Embedding.Slice()
	} else if s.Embed != nil {
		if vec, err = s.Embed(ctx, text); err != nil {
			return s.unavailable(ctx, jobID, threadID, fmt.Errorf("nhúng: %w", err))
		}
		v := pgvector.NewVector(vec)
		_ = q.ThreadSetEmbedding(ctx, store.ThreadSetEmbeddingParams{ID: threadID, Embedding: &v})
	}
	if len(vec) > 0 {
		_ = q.ThreadSetSimilarOf(ctx, store.ThreadSetSimilarOfParams{ID: threadID, MinCosine: SimilarOfMin})
	}
	hits, err := s.Rag.SearchStudent(ctx, rag.Query{CourseID: t.CourseID, Vec: vec, Text: text})
	if err != nil {
		return s.unavailable(ctx, jobID, threadID, fmt.Errorf("truy xuất: %w", err))
	}
	switch {
	case len(hits) == 0:
		return s.skip(ctx, jobID, threadID, SkipNoContext)
	case hits[0].Cosine < agent.RagSimFloor:
		return s.skip(ctx, jobID, threadID, SkipLowScore)
	}
	lane := llm.LaneNearRealtime // không chạm làn INTERACTIVE của chat riêng
	resp, err := s.LLM.Chat(ctx, llm.Request{Task: llm.TaskChat, Lane: &lane, Messages: agent.BuildMessages(nil, agent.ContextFromHits(hits), text)})
	if err != nil || resp.Degraded {
		if err == nil {
			err = errors.New("mọi nhà cung cấp lỗi")
		}
		return s.unavailable(ctx, jobID, threadID, err)
	}
	body, cites := agent.ExtractCitations(strings.TrimSpace(resp.Text), hits)
	if body == "" {
		return s.unavailable(ctx, jobID, threadID, errors.New("câu trả lời rỗng"))
	}
	if len(cites) == 0 { // luôn kèm nguồn: câu trả lời dựa trên đoạn đầu tiên
		cites = []agent.Citation{agent.CitationOf(1, hits[0])}
	}
	craw, _ := json.Marshal(cites)
	conf := agent.Score(agent.Retrieval(agent.BestCosine(hits)), agent.Groundedness(body, hits)) // cùng công thức với chat (SRS 4.8), không LLM

	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("thread: mở giao dịch: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tq := store.New(tx)
	if n, err := tq.ThreadMarkAnswered(ctx, threadID); err != nil {
		return fmt.Errorf("thread: cập nhật thread: %w", err)
	} else if n == 0 {
		return s.finish(ctx, jobID, map[string]any{"thread_id": threadID, "already": true})
	}
	if _, err := tq.InsertForumPost(ctx, store.InsertForumPostParams{CourseID: t.CourseID, ThreadID: threadID, Kind: store.PostKindAI, Body: body,
		VerificationState: store.PostVerificationPENDING, Citations: craw, Confidence: decimal.NullDecimal{Decimal: conf, Valid: true}}); err != nil {
		var pe *pgconn.PgError
		if errors.As(err, &pe) && pe.Code == "23505" { // bài AI thứ hai (giao lại): đã có
			return s.finish(ctx, jobID, map[string]any{"thread_id": threadID, "already": true})
		}
		return fmt.Errorf("thread: lưu bài AI: %w", err)
	}
	s.notify(ctx, tq, t.AuthorID, t.CourseID, threadID, "THREAD_ANSWERED", "AI đã trả lời câu hỏi của bạn", "answered:"+threadID.String())
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("thread: commit: %w", err)
	}
	return s.finish(ctx, jobID, map[string]any{"thread_id": threadID, "ai_state": "ANSWERED"})
}

// unavailable: lỗi tạm thời — thử lại (trả error) cho tới lần thứ maxAnswerTries rồi SKIPPED LLM_UNAVAILABLE.
func (s *Service) unavailable(ctx context.Context, jobID, threadID uuid.UUID, cause error) error {
	n := 1
	if s.Redis != nil {
		k := appredis.Key("thread", "ans", threadID.String())
		if v, err := s.Redis.Incr(ctx, k).Result(); err == nil {
			n = int(v)
			_ = s.Redis.Expire(ctx, k, time.Hour).Err()
		}
	}
	if n >= maxAnswerTries {
		if s.Log != nil {
			s.Log.WarnContext(ctx, "thread: AI không trả lời được, bỏ qua", "thread_id", threadID.String(), "error", cause.Error())
		}
		return s.skip(ctx, jobID, threadID, SkipLLMUnavailable)
	}
	return fmt.Errorf("thread: AI trả lời lần %d: %w", n, cause)
}

func (s *Service) skip(ctx context.Context, jobID, threadID uuid.UUID, reason string) error {
	if _, err := store.New(s.Pool).ThreadMarkSkipped(ctx, store.ThreadMarkSkippedParams{ID: threadID, Reason: &reason}); err != nil {
		return fmt.Errorf("thread: ghi SKIPPED: %w", err)
	}
	return s.finish(ctx, jobID, map[string]any{"thread_id": threadID, "ai_state": "SKIPPED", "reason": reason})
}

func (s *Service) finish(ctx context.Context, jobID uuid.UUID, result any) error {
	if s.Run == nil || jobID == uuid.Nil {
		return nil
	}
	return s.Run.Complete(ctx, jobID, result)
}
