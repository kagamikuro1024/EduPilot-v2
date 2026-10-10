// Package rag: truy xuất tất định cho hỏi–đáp (SRS FEAT-docs-calendar 4.3, D47 mục 1 và 7).
// Một câu SQL kết hợp vectơ + từ khoá (RRF k = 60); mọi điều kiện quyền nằm TRONG câu SQL, trước ORDER BY … LIMIT.
// Gói này KHÔNG có đường tìm đoạn `audience = 'GRADING'` (đáp án): chỉ hai hàm công khai bên dưới, không tham số audience.
package rag

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"

	"github.com/edupilot/backend-go/internal/store"
)

// DefaultTopK là RAG_TOP_K.
const DefaultTopK = 8

// Dims là số chiều vectơ nhúng của hệ thống (khớp content_chunks.embedding).
const Dims = 1536

// ErrBadQuery: truy vấn thiếu lớp hoặc vectơ sai chiều.
var ErrBadQuery = errors.New("rag: truy vấn không hợp lệ")

// Query là một lần truy xuất.
type Query struct {
	CourseID    uuid.UUID
	Vec         []float32   // đã nhúng một lần ở tầng gọi (1536 chiều)
	Text        string      // cho nhánh từ khoá
	DocumentIDs []uuid.UUID // rỗng = cả lớp; có = chỉ các tài liệu này
	K           int         // 0 = DefaultTopK
}

// Hit là một đoạn trả về. Cosine luôn có giá trị (cả hai nhánh đều đòi embedding IS NOT NULL).
type Hit struct {
	ChunkID, DocumentID uuid.UUID
	Title               string
	PageNo              *int
	Heading             *string
	Text                string
	Cosine, Score       float64
}

// Service truy xuất đoạn tài liệu.
type Service struct{ DB store.DBTX }

// SearchStudent: audience {ALL} và tài liệu `visible_to_students`.
func (s *Service) SearchStudent(ctx context.Context, q Query) ([]Hit, error) {
	return s.search(ctx, q, true, []string{string(store.ChunkAudienceALL)})
}

// SearchStaff: audience {ALL, STAFF}.
func (s *Service) SearchStaff(ctx context.Context, q Query) ([]Hit, error) {
	return s.search(ctx, q, false, []string{string(store.ChunkAudienceALL), string(store.ChunkAudienceSTAFF)})
}

func (s *Service) search(ctx context.Context, q Query, studentOnly bool, audiences []string) ([]Hit, error) {
	if q.CourseID == uuid.Nil || len(q.Vec) != Dims {
		return nil, ErrBadQuery
	}
	k := q.K
	if k <= 0 {
		k = DefaultTopK
	}
	// text[] rồi ép ::uuid[] ở SQL: pool chạy QueryExecModeExec (PgBouncer) không mã hoá được []uuid.UUID (OID 0) — QC BUG-1 của US-P3-05
	docs := make([]string, len(q.DocumentIDs))
	for i, d := range q.DocumentIDs {
		docs[i] = d.String()
	}
	rows, err := store.New(s.DB).RagSearch(ctx, store.RagSearchParams{
		CourseID: q.CourseID, Vec: pgvector.NewVector(q.Vec), QueryText: q.Text, StudentOnly: studentOnly,
		Audiences: audiences, DocumentIds: docs, K: int32(k), //nolint:gosec // k ≤ vài chục
	})
	if err != nil {
		return nil, fmt.Errorf("rag: truy xuất: %w", err)
	}
	hits := make([]Hit, len(rows))
	for i, r := range rows {
		hits[i] = Hit{ChunkID: r.ChunkID, DocumentID: r.DocumentID, Title: r.Title, Heading: r.Heading, Text: r.Text, Cosine: r.Cosine, Score: r.Score}
		if r.PageNo != nil {
			p := int(*r.PageNo)
			hits[i].PageNo = &p
		}
	}
	return hits, nil
}
