package rag_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/knowledge"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/rag"
	"github.com/edupilot/backend-go/internal/testutil"
)

// upsert ghi một mục tri thức trong một giao dịch rồi commit; trả chunk_id.
func (f *fx) upsert(t *testing.T, course, source uuid.UUID, q, a string) uuid.UUID {
	t.Helper()
	tx, err := f.pool.Begin(t.Context())
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(t.Context()) }()
	id, err := knowledge.Upsert(t.Context(), tx, knowledge.Source{CourseID: course, Type: knowledge.SourceTicket, ID: source}, q, a)
	require.NoError(t, err)
	require.NoError(t, tx.Commit(t.Context()))
	return id
}

// sync chạy handler `knowledge.sync` với embedder cố định (một lần Embed mỗi lần gọi).
func (f *fx) sync(t *testing.T, course, chunk uuid.UUID, vec []float32, embedded *[]string) error {
	t.Helper()
	s := &knowledge.Syncer{Pool: f.pool, Embed: func(_ context.Context, text string) ([]float32, error) {
		if embedded != nil {
			*embedded = append(*embedded, text)
		}
		return vec, nil
	}}
	var doc uuid.UUID
	if err := f.pool.QueryRow(t.Context(), `select document_id from content_chunks where id = $1`, chunk).Scan(&doc); err != nil {
		doc = uuid.New() // đoạn đã gỡ: handler phải bỏ qua, không lỗi
	}
	p, _ := json.Marshal(map[string]any{"chunk_id": chunk, "course_id": course, "document_id": doc})
	return s.Handle(t.Context(), outbox.Message{Topic: knowledge.TopicSync, Payload: p})
}

func (f *fx) outboxCount(t *testing.T, topic string, course uuid.UUID) int {
	t.Helper()
	var n int
	require.NoError(t, f.pool.QueryRow(t.Context(), `select count(*) from outbox where topic = $1 and payload->>'course_id' = $2`, topic, course.String()).Scan(&n))
	return n
}

// TestKnowledgeRetrievable — AC11: sau knowledge.sync (một Embed) sinh viên hỏi câu tương tự thấy đoạn tri thức, kèm nhãn nguồn; nhãn không phải liên kết tài liệu.
func TestKnowledgeRetrievable(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	src := uuid.New()
	chunk := f.upsert(t, f.c1, src, "Hạn nộp bài 3 là khi nào?", "Hết thứ Sáu tuần 5.")
	var embedded []string
	require.NoError(t, f.sync(t, f.c1, chunk, axis(3, nil), &embedded))
	require.Len(t, embedded, 1, "đúng một lần Embed")
	require.Contains(t, embedded[0], "Hết thứ Sáu tuần 5.")

	hs, err := f.svc.SearchStudent(t.Context(), rag.Query{CourseID: f.c1, Vec: axis(3, nil), Text: "hạn nộp bài 3"})
	require.NoError(t, err)
	require.Len(t, hs, 1)
	require.Equal(t, chunk, hs[0].ChunkID)
	require.True(t, hs[0].Knowledge)
	require.Equal(t, knowledge.DocTitle, hs[0].Title)
	other, err := f.svc.SearchStudent(t.Context(), rag.Query{CourseID: f.c2, Vec: axis(3, nil), Text: "hạn nộp bài 3"})
	require.NoError(t, err)
	require.Empty(t, other, "lớp khác không thấy")
}

// TestKnowledgeUnembeddedNotRetrieved — AC11: đoạn có embedding NULL (chưa nạp) không lọt ra kết quả tìm.
func TestKnowledgeUnembeddedNotRetrieved(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	f.upsert(t, f.c1, uuid.New(), "Hạn nộp bài 3 là khi nào?", "Hết thứ Sáu tuần 5.")
	hs, err := f.svc.SearchStudent(t.Context(), rag.Query{CourseID: f.c1, Vec: axis(3, nil), Text: "hạn nộp bài 3"})
	require.NoError(t, err)
	require.Empty(t, hs)
	var nulls int
	require.NoError(t, f.pool.QueryRow(t.Context(), `select count(*) from content_chunks where $1 = any(course_ids) and embedding is null`, f.c1).Scan(&nulls))
	require.Equal(t, 1, nulls)
}

// TestKnowledgeRemovedNotRetrieved — AC11: gỡ → hết trong kết quả tìm, document.changed được xếp để vô hiệu cache.
func TestKnowledgeRemovedNotRetrieved(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	src := uuid.New()
	chunk := f.upsert(t, f.c1, src, "Hạn nộp bài 3 là khi nào?", "Hết thứ Sáu tuần 5.")
	require.NoError(t, f.sync(t, f.c1, chunk, axis(3, nil), nil))
	before := f.outboxCount(t, knowledge.TopicDocChanged, f.c1)

	tx, err := f.pool.Begin(t.Context())
	require.NoError(t, err)
	ok, err := knowledge.Remove(t.Context(), tx, f.c1, knowledge.SourceTicket, src)
	require.NoError(t, err)
	require.True(t, ok)
	require.NoError(t, tx.Commit(t.Context()))

	hs, err := f.svc.SearchStudent(t.Context(), rag.Query{CourseID: f.c1, Vec: axis(3, nil), Text: "hạn nộp bài 3"})
	require.NoError(t, err)
	require.Empty(t, hs)
	require.Equal(t, before+1, f.outboxCount(t, knowledge.TopicDocChanged, f.c1))
	require.NoError(t, f.sync(t, f.c1, chunk, axis(3, nil), nil), "đoạn đã gỡ: handler bỏ qua, không lỗi")
}

// TestKnowledgeRetryRewritesOutbox — AC11: lưu lại (= `Thử lại`) thay đoạn cũ, không đôi, và ghi lại outbox knowledge.sync; bản sửa mới đưa embedding về NULL tới khi nạp xong.
func TestKnowledgeRetryRewritesOutbox(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	src := uuid.New()
	c1 := f.upsert(t, f.c1, src, "Hạn nộp bài 3?", "Thứ Sáu.")
	require.NoError(t, f.sync(t, f.c1, c1, axis(3, nil), nil))
	c2 := f.upsert(t, f.c1, src, "Hạn nộp bài 3?", "Thứ Sáu tuần 6 (đã sửa).")
	require.Equal(t, c1, c2, "cùng đoạn, không thêm dòng")
	require.Equal(t, 2, f.outboxCount(t, knowledge.TopicSync, f.c1))
	var n, nulls int
	require.NoError(t, f.pool.QueryRow(t.Context(), `select count(*), count(*) filter (where embedding is null) from content_chunks where $1 = any(course_ids)`, f.c1).Scan(&n, &nulls))
	require.Equal(t, 1, n)
	require.Equal(t, 1, nulls, "bản sửa chờ nạp lại")
	require.NoError(t, f.sync(t, f.c1, c1, axis(3, nil), nil))
	hs, err := f.svc.SearchStudent(t.Context(), rag.Query{CourseID: f.c1, Vec: axis(3, nil), Text: "hạn nộp bài 3"})
	require.NoError(t, err)
	require.Len(t, hs, 1)
	require.Contains(t, hs[0].Text, "đã sửa")
}

// TestKnowledgeBumpsRagVersion — AC11: nạp xong phát document.changed; handler của nó tăng ep:rag:ver của lớp.
func TestKnowledgeBumpsRagVersion(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	rdb, err := appredis.New(t.Context(), testutil.RedisURL(t))
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })
	chunk := f.upsert(t, f.c1, uuid.New(), "Hạn nộp bài 3?", "Thứ Sáu.")
	require.NoError(t, f.sync(t, f.c1, chunk, axis(3, nil), nil))
	var payload []byte
	require.NoError(t, f.pool.QueryRow(t.Context(), `select payload from outbox where topic = $1 and payload->>'course_id' = $2`, knowledge.TopicDocChanged, f.c1.String()).Scan(&payload))
	v0, err := rag.Version(t.Context(), rdb, f.c1)
	require.NoError(t, err)
	require.NoError(t, rag.BumpVersion(rdb)(t.Context(), outbox.Message{Topic: knowledge.TopicDocChanged, Payload: payload}))
	v1, err := rag.Version(t.Context(), rdb, f.c1)
	require.NoError(t, err)
	require.Equal(t, v0+1, v1)
}

// TestKnowledgeDocExcludedEverywhere — AC11: MỌI `from/join documents` trong store/queries/*.sql (ngoài rag.sql, knowledge.sql) loại tài liệu tri thức.
func TestKnowledgeDocExcludedEverywhere(t *testing.T) {
	t.Parallel()
	files, err := filepath.Glob("../store/queries/*.sql")
	require.NoError(t, err)
	require.NotEmpty(t, files)
	re := regexp.MustCompile(`(?i)\b(from|join)\s+documents\b`)
	scanned := 0
	for _, f := range files {
		if base := filepath.Base(f); base == "rag.sql" || base == "knowledge.sql" {
			continue
		}
		raw, err := os.ReadFile(f)
		require.NoError(t, err)
		for _, stmt := range strings.Split(string(raw), ";") {
			if !re.MatchString(stmt) {
				continue
			}
			scanned++
			require.Contains(t, stmt, "THREAD_KNOWLEDGE", "%s: truy vấn đọc documents mà không loại tài liệu tri thức:\n%s", filepath.Base(f), strings.TrimSpace(stmt))
		}
	}
	require.Greater(t, scanned, 15, "phải quét được các truy vấn thật")
}
