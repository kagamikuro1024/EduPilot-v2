// Package knowledge là kho tri thức của lớp (SRS FEAT-escalation-notify 4.7): câu trả lời đã được giảng viên xác nhận được nạp lại cho AI qua MỘT dòng `documents` ảo mỗi lớp
// (category `THREAD_KNOWLEDGE`), dùng lại toàn bộ đường RAG. `Upsert` / `Remove` là đường DUY NHẤT ghi `content_chunks` của tài liệu này; văn bản do người gọi đưa vào
// PHẢI đã che thông tin cá nhân (người gọi chịu trách nhiệm qua `privacy`).
package knowledge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"

	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/store"
)

// Hằng của kho tri thức.
const (
	Category     = "THREAD_KNOWLEDGE"
	DocTitle     = "Hỏi đáp đã được giảng viên xác nhận"
	SourcePost   = "POST"
	SourceTicket = "TICKET"
	// TopicSync: nhúng một đoạn. TopicDocChanged: cùng chuỗi với ingest.TopicDocumentChanged (rag.BumpVersion đăng ký).
	TopicSync       = "knowledge.sync"
	TopicDocChanged = "document.changed"
	headingMax      = 80
)

// Source định danh nguồn của một mục tri thức.
type Source struct {
	CourseID uuid.UUID
	Type     string // SourcePost | SourceTicket
	ID       uuid.UUID
	By       *uuid.UUID
}

// EnsureDocument lấy (tạo lười) tài liệu ảo của lớp.
func EnsureDocument(ctx context.Context, q *store.Queries, course uuid.UUID) (uuid.UUID, error) {
	id, err := q.KnowledgeEnsureDoc(ctx, store.KnowledgeEnsureDocParams{CourseID: course, Title: DocTitle})
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, fmt.Errorf("knowledge: tạo tài liệu: %w", err)
	}
	id, err = q.KnowledgeDocID(ctx, course)
	if err != nil {
		return uuid.Nil, fmt.Errorf("knowledge: đọc tài liệu: %w", err)
	}
	return id, nil
}

func chunkText(question, answer string) (heading, text string) {
	q := strings.TrimSpace(question)
	heading = q
	if utf8.RuneCountInString(heading) > headingMax {
		heading = string([]rune(heading)[:headingMax])
	}
	return strings.Join(strings.Fields(heading), " "), "Câu hỏi: " + q + "\nTrả lời: " + strings.TrimSpace(answer)
}

// Upsert ghi (hoặc thay) mục tri thức của `src` TRONG giao dịch của người gọi: content_chunks (embedding = NULL), knowledge_items, outbox `knowledge.sync`.
// Idempotent theo (source_type, source_id): lưu lại = thay đoạn cũ, không thêm dòng. Trả chunk_id.
func Upsert(ctx context.Context, tx pgx.Tx, src Source, question, answer string) (uuid.UUID, error) {
	q := store.New(tx)
	heading, text := chunkText(question, answer)
	var chunk uuid.UUID
	item, err := q.KnowledgeItemBySource(ctx, store.KnowledgeItemBySourceParams{SourceType: src.Type, SourceID: src.ID})
	switch {
	case err == nil:
		chunk = item.ChunkID
		if err := q.KnowledgeChunkRewrite(ctx, store.KnowledgeChunkRewriteParams{ID: chunk, CourseID: src.CourseID, Heading: &heading, Text: text}); err != nil {
			return uuid.Nil, fmt.Errorf("knowledge: ghi lại đoạn: %w", err)
		}
		if err := q.KnowledgeItemTouch(ctx, store.KnowledgeItemTouchParams{ID: item.ID, CreatedBy: src.By}); err != nil {
			return uuid.Nil, fmt.Errorf("knowledge: cập nhật mục: %w", err)
		}
	case errors.Is(err, pgx.ErrNoRows):
		doc, err := EnsureDocument(ctx, q, src.CourseID)
		if err != nil {
			return uuid.Nil, err
		}
		ord, err := q.KnowledgeNextOrd(ctx, store.KnowledgeNextOrdParams{DocumentID: doc, CourseID: src.CourseID})
		if err != nil {
			return uuid.Nil, fmt.Errorf("knowledge: thứ tự đoạn: %w", err)
		}
		chunk, err = q.KnowledgeChunkInsert(ctx, store.KnowledgeChunkInsertParams{DocumentID: doc, CourseID: src.CourseID, Ord: ord, Heading: &heading, Text: text})
		if err != nil {
			return uuid.Nil, fmt.Errorf("knowledge: thêm đoạn: %w", err)
		}
		if _, err := q.KnowledgeItemInsert(ctx, store.KnowledgeItemInsertParams{CourseID: src.CourseID, SourceType: src.Type, SourceID: src.ID, ChunkID: chunk, CreatedBy: src.By}); err != nil {
			return uuid.Nil, fmt.Errorf("knowledge: thêm mục: %w", err)
		}
	default:
		return uuid.Nil, fmt.Errorf("knowledge: tìm mục: %w", err)
	}
	if _, err := outbox.Write(ctx, tx, TopicSync, map[string]any{"chunk_id": chunk, "course_id": src.CourseID}); err != nil {
		return uuid.Nil, fmt.Errorf("knowledge: outbox sync: %w", err)
	}
	return chunk, nil
}

// Remove gỡ mục tri thức của `src` (nếu có) trong giao dịch của người gọi và phát `document.changed` (tăng `ep:rag:ver`, vô hiệu cache câu trả lời của lớp).
func Remove(ctx context.Context, tx pgx.Tx, courseID uuid.UUID, typ string, id uuid.UUID) (bool, error) {
	q := store.New(tx)
	item, err := q.KnowledgeItemBySource(ctx, store.KnowledgeItemBySourceParams{SourceType: typ, SourceID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("knowledge: tìm mục: %w", err)
	}
	doc, err := q.KnowledgeChunkDelete(ctx, item.ChunkID)
	if err != nil {
		return false, fmt.Errorf("knowledge: gỡ đoạn: %w", err)
	}
	if _, err := outbox.Write(ctx, tx, TopicDocChanged, map[string]any{"course_id": courseID, "document_id": doc}); err != nil {
		return false, fmt.Errorf("knowledge: outbox: %w", err)
	}
	return true, nil
}

// Syncer là handler outbox `knowledge.sync`: MỘT lần Embed (ngoài mọi outbox.Chain) rồi ghi `embedding` chỉ khi chữ chưa đổi, sau đó phát `document.changed`.
type Syncer struct {
	Pool    *pgxpool.Pool
	Embed   func(ctx context.Context, text string) ([]float32, error)
	Timeout time.Duration // KNOWLEDGE_EMBED_TIMEOUT (mặc định 10 giây)
}

// Handle implements outbox.Handler. Đoạn đã bị gỡ → bỏ qua; lỗi nhúng → trả lỗi để outbox thử lại rồi DEAD.
func (s *Syncer) Handle(ctx context.Context, m outbox.Message) error {
	var p struct {
		ChunkID  uuid.UUID `json:"chunk_id"`
		CourseID uuid.UUID `json:"course_id"`
	}
	if err := json.Unmarshal(m.Payload, &p); err != nil || p.ChunkID == uuid.Nil {
		return errors.New("payload knowledge.sync không hợp lệ")
	}
	q := store.New(s.Pool)
	c, err := q.KnowledgeChunkForSync(ctx, store.KnowledgeChunkForSyncParams{ID: p.ChunkID, CourseID: p.CourseID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil // đã gỡ
	}
	if err != nil {
		return fmt.Errorf("knowledge: đọc đoạn: %w", err)
	}
	to := s.Timeout
	if to <= 0 {
		to = 10 * time.Second
	}
	ectx, cancel := context.WithTimeout(ctx, to)
	defer cancel()
	vec, err := s.Embed(ectx, c.Text)
	if err != nil {
		return fmt.Errorf("knowledge: nhúng: %w", err)
	}
	v := pgvector.NewVector(vec)
	n, err := q.KnowledgeSetEmbedding(ctx, store.KnowledgeSetEmbeddingParams{ID: p.ChunkID, CourseID: p.CourseID, Embedding: &v, Digest: c.Digest})
	if err != nil {
		return fmt.Errorf("knowledge: ghi embedding: %w", err)
	}
	if n == 0 {
		return nil // đã có bản sửa mới hơn: handler của bản đó sẽ nhúng
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("knowledge: mở giao dịch: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := outbox.Write(ctx, tx, TopicDocChanged, map[string]any{"course_id": p.CourseID, "document_id": c.DocumentID}); err != nil {
		return fmt.Errorf("knowledge: outbox: %w", err)
	}
	return tx.Commit(ctx)
}
