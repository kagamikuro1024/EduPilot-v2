package document

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pgvector/pgvector-go"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
	"github.com/edupilot/backend-go/internal/ingest"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/store"
)

// MaxChunkChars là giới hạn sửa một đoạn (SRS FEAT-docs-calendar 4.5).
const MaxChunkChars = 4000

func escapeLike(q string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(q)
}

// Row là một dòng của danh sách `/documents`. `SharedFrom` (mã lớp gốc) khác rỗng ⇒ tài liệu chia sẻ vào lớp, chỉ đọc.
type Row struct {
	Doc
	SharedFrom string `json:"shared_from"`
}

// Page là một trang danh sách.
type Page struct {
	Items      []Row   `json:"items"`
	NextCursor *string `json:"next_cursor"`
}

// ListFilter là bộ lọc của danh sách.
type ListFilter struct{ Type, Status, Q string }

// List: tài liệu của lớp và được chia sẻ vào lớp, mới cập nhật trước, con trỏ (updated_at, id).
func (s *Service) List(ctx context.Context, course uuid.UUID, f ListFilter, p httpx.PageParams) (Page, error) {
	arg := store.DocListParams{CourseID: course, PageLimit: int32(p.Fetch())} //nolint:gosec // ≤ 101
	if f.Type != "" {
		arg.Type = &f.Type
	}
	if f.Status != "" {
		arg.Status = &f.Status
	}
	if q := strings.TrimSpace(f.Q); q != "" {
		e := escapeLike(q)
		arg.Q = &e
	}
	if p.Cursor != nil {
		arg.CursorAt, arg.CursorID = &p.Cursor.CreatedAt, &p.Cursor.ID
	}
	rows, err := store.New(s.Pool).DocList(ctx, arg)
	if err != nil {
		return Page{}, fmt.Errorf("document: danh sách: %w", err)
	}
	out := Page{Items: make([]Row, 0, len(rows))}
	if len(rows) > p.Limit {
		rows = rows[:p.Limit]
		c := httpx.EncodeCursor(rows[len(rows)-1].UpdatedAt, rows[len(rows)-1].ID.String())
		out.NextCursor = &c
	}
	for _, r := range rows {
		out.Items = append(out.Items, Row{Doc: Doc{ID: r.ID, CourseID: r.CourseID, Title: r.Title, Type: string(r.Type), Filename: r.Filename, MimeType: r.MimeType, SizeBytes: r.SizeBytes,
			Status: string(r.Status), Error: r.Error, PageCount: r.PageCount, VisibleToStudents: r.VisibleToStudents, UseForRAG: r.UseForRag, Category: r.Category, WeekNo: r.WeekNo,
			Version: r.Version, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}, SharedFrom: r.SharedFrom})
	}
	return out, nil
}

// Get: một tài liệu của lớp (hoặc chia sẻ vào lớp).
func (s *Service) Get(ctx context.Context, course, id uuid.UUID) (Row, error) {
	d, err := store.New(s.Pool).DocGetInCourse(ctx, store.DocGetInCourseParams{ID: id, CourseID: course})
	if errors.Is(err, pgx.ErrNoRows) {
		return Row{}, apierr.New(http.StatusNotFound, apierr.NotFound)
	}
	if err != nil {
		return Row{}, fmt.Errorf("document: đọc tài liệu: %w", err)
	}
	r := Row{Doc: docOf(d)}
	if d.CourseID != course {
		c, err := store.New(s.Pool).GetCourse(ctx, d.CourseID)
		if err != nil {
			return Row{}, fmt.Errorf("document: đọc lớp gốc: %w", err)
		}
		r.SharedFrom = c.ClassCode
	}
	return r, nil
}

// PatchIn: các trường đổi được; con trỏ nil = giữ nguyên. `Version` bắt buộc (khoá lạc quan).
type PatchIn struct {
	Title             *string `json:"title"`
	Type              *string `json:"type"`
	Category          *string `json:"category"`
	WeekNo            *int16  `json:"week_no"`
	UseForRAG         *bool   `json:"use_for_rag"`
	VisibleToStudents *bool   `json:"visible_to_students"`
	Version           int32   `json:"version"`
}

// audienceOf: GRADING nếu ANSWER_KEY; STAFF nếu ẩn với sinh viên; còn lại ALL (SRS 4.1).
func audienceOf(t store.DocumentType, visible bool) store.ChunkAudience {
	switch {
	case t == store.DocumentTypeANSWERKEY:
		return store.ChunkAudienceGRADING
	case !visible:
		return store.ChunkAudienceSTAFF
	}
	return store.ChunkAudienceALL
}

// Patch đổi tên / loại / chủ đề / tuần / cờ trong MỘT giao dịch: khoá hàng, kiểm version, `audience` mọi đoạn đổi theo, audit, document.changed, bật `use_for_rag` → xếp reindex.
func (s *Service) Patch(ctx context.Context, actor, course, id uuid.UUID, trace string, in PatchIn) (Doc, error) {
	if err := s.requireActive(ctx, course); err != nil {
		return Doc{}, err
	}
	if in.Version < 1 {
		return Doc{}, field("version", "required", "Thiếu phiên bản tài liệu.")
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return Doc{}, fmt.Errorf("document: mở giao dịch: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	cur, err := q.DocGetForUpdate(ctx, store.DocGetForUpdateParams{ID: id, CourseID: course})
	if errors.Is(err, pgx.ErrNoRows) {
		if _, e := q.DocGetInCourse(ctx, store.DocGetInCourseParams{ID: id, CourseID: course}); e == nil {
			return Doc{}, apierr.New(http.StatusConflict, apierr.DocumentSharedReadonly) // chia sẻ vào lớp: chỉ lớp gốc sửa
		}
		return Doc{}, apierr.New(http.StatusNotFound, apierr.NotFound)
	}
	if err != nil {
		return Doc{}, fmt.Errorf("document: khoá tài liệu: %w", err)
	}
	if cur.Version != in.Version {
		return Doc{}, apierr.New(http.StatusConflict, apierr.VersionConflict).WithDetails(map[string]any{"current": docOf(cur)})
	}
	next := cur
	if in.Title != nil {
		t := strings.TrimSpace(*in.Title)
		if n := utf8.RuneCountInString(t); n < 1 || n > 200 {
			return Doc{}, field("title", "length", "Tên tài liệu dài 1–200 ký tự.")
		}
		next.Title = t
	}
	if in.Type != nil {
		t := store.DocumentType(*in.Type)
		switch t {
		case store.DocumentTypeLECTURE, store.DocumentTypeCOURSEPOLICY, store.DocumentTypeEXAMPAPER, store.DocumentTypeANSWERKEY, store.DocumentTypeOTHER:
		default:
			return Doc{}, field("type", "enum", "Loại tài liệu không đúng.")
		}
		next.Type = t
	}
	if in.Category != nil {
		c := strings.TrimSpace(*in.Category)
		if utf8.RuneCountInString(c) > 40 {
			return Doc{}, field("category", "length", "Chủ đề tối đa 40 ký tự.")
		}
		next.Category = &c
		if c == "" {
			next.Category = nil
		}
	}
	if in.WeekNo != nil {
		if *in.WeekNo < 1 || *in.WeekNo > 20 {
			return Doc{}, field("week_no", "range", "Tuần từ 1 đến 20.")
		}
		next.WeekNo = in.WeekNo
	}
	if in.UseForRAG != nil {
		next.UseForRag = *in.UseForRAG
	}
	if in.VisibleToStudents != nil {
		if *in.VisibleToStudents && next.Type == store.DocumentTypeANSWERKEY {
			return Doc{}, apierr.New(http.StatusUnprocessableEntity, apierr.AnswerKeyNotVisible)
		}
		next.VisibleToStudents = *in.VisibleToStudents
	}
	if next.Type == store.DocumentTypeANSWERKEY {
		next.VisibleToStudents = false // đổi sang ANSWER_KEY mà không nói gì về cờ: ẩn (DB cũng chặn bằng documents_answer_key_chk)
	}
	row, err := q.DocUpdate(ctx, store.DocUpdateParams{ID: id, CourseID: course, Title: next.Title, Type: next.Type, Category: next.Category, WeekNo: next.WeekNo,
		UseForRag: next.UseForRag, VisibleToStudents: next.VisibleToStudents})
	if err != nil {
		return Doc{}, fmt.Errorf("document: cập nhật: %w", err)
	}
	if _, err := q.DocSetChunkAudience(ctx, store.DocSetChunkAudienceParams{CourseID: course, DocID: id, NewAudience: audienceOf(next.Type, next.VisibleToStudents)}); err != nil {
		return Doc{}, fmt.Errorf("document: đổi audience đoạn: %w", err)
	}
	before, _ := json.Marshal(flagsOf(cur))
	after, _ := json.Marshal(flagsOf(row))
	if _, err := q.InsertAuditLog(ctx, store.InsertAuditLogParams{CourseID: &course, ActorID: &actor, Entity: "document", EntityID: id.String(), Action: "document.patch", Before: before, After: after, TraceID: ptrIf(trace)}); err != nil {
		return Doc{}, fmt.Errorf("document: audit: %w", err)
	}
	if _, err := outbox.Write(ctx, tx, ingest.TopicDocumentChanged, map[string]string{"course_id": course.String(), "document_id": id.String()}); err != nil {
		return Doc{}, fmt.Errorf("document: outbox: %w", err)
	}
	if !cur.UseForRag && row.UseForRag && row.Status == store.DocumentStatusREADY { // bật Dùng cho AI: các đoạn chưa nhúng được nhúng ở việc nền
		if _, err := s.Jobs.EnqueueTx(ctx, tx, actor, ingest.KindReindex, ingest.KindPayload{DocumentID: id}); err != nil {
			return Doc{}, fmt.Errorf("document: xếp reindex: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return Doc{}, fmt.Errorf("document: commit: %w", err)
	}
	return docOf(row), nil
}

func flagsOf(d store.Document) map[string]any {
	return map[string]any{"title": d.Title, "type": d.Type, "category": d.Category, "week_no": d.WeekNo, "use_for_rag": d.UseForRag, "visible_to_students": d.VisibleToStudents, "version": d.Version}
}

func ptrIf(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// Impact là số liệu cho câu xác nhận xoá.
type Impact struct {
	Chunks  int64 `json:"chunks"`
	Courses int64 `json:"courses"`
}

// Delete (Giảng viên): xoá dòng `documents` (đoạn, document_courses theo CASCADE; chat_sessions.document_id → NULL), audit, document.changed; xoá object sau khi commit.
func (s *Service) Delete(ctx context.Context, actor, course, id uuid.UUID, trace string) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("document: mở giao dịch: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	cur, err := q.DocGetForUpdate(ctx, store.DocGetForUpdateParams{ID: id, CourseID: course})
	if errors.Is(err, pgx.ErrNoRows) {
		if _, e := q.DocGetInCourse(ctx, store.DocGetInCourseParams{ID: id, CourseID: course}); e == nil {
			return apierr.New(http.StatusConflict, apierr.DocumentSharedReadonly)
		}
		return apierr.New(http.StatusNotFound, apierr.NotFound)
	}
	if err != nil {
		return fmt.Errorf("document: khoá tài liệu: %w", err)
	}
	imp, err := q.DocDeleteImpact(ctx, store.DocDeleteImpactParams{ID: id, CourseID: course})
	if err != nil {
		return fmt.Errorf("document: tính ảnh hưởng: %w", err)
	}
	if _, err := q.DocDelete(ctx, store.DocDeleteParams{ID: id, CourseID: course}); err != nil {
		return fmt.Errorf("document: xoá: %w", err)
	}
	before, _ := json.Marshal(flagsOf(cur))
	after, _ := json.Marshal(Impact{Chunks: imp.Chunks, Courses: imp.Courses})
	if _, err := q.InsertAuditLog(ctx, store.InsertAuditLogParams{CourseID: &course, ActorID: &actor, Entity: "document", EntityID: id.String(), Action: "document.delete", Before: before, After: after, TraceID: ptrIf(trace)}); err != nil {
		return fmt.Errorf("document: audit: %w", err)
	}
	if _, err := outbox.Write(ctx, tx, ingest.TopicDocumentChanged, map[string]string{"course_id": course.String(), "document_id": id.String()}); err != nil {
		return fmt.Errorf("document: outbox: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("document: commit: %w", err)
	}
	if cur.BlobKey != nil {
		s.dropObject(ctx, *cur.BlobKey) // ponytail: xoá trong request sau commit (lỗi chỉ log, object mồ côi); việc nền dọn khi đo thấy cần
	}
	return nil
}

// Impact đếm đoạn và lớp chia sẻ của một tài liệu của lớp (cho hộp xác nhận xoá).
func (s *Service) Impact(ctx context.Context, course, id uuid.UUID) (Impact, error) {
	if _, err := s.Get(ctx, course, id); err != nil {
		return Impact{}, err
	}
	r, err := store.New(s.Pool).DocDeleteImpact(ctx, store.DocDeleteImpactParams{ID: id, CourseID: course})
	if err != nil {
		return Impact{}, fmt.Errorf("document: ảnh hưởng: %w", err)
	}
	return Impact{Chunks: r.Chunks, Courses: r.Courses}, nil
}

// ChunkOut là một đoạn của tài liệu (dành cho Staff).
type ChunkOut struct {
	ID       uuid.UUID `json:"id"`
	Ord      int32     `json:"ord"`
	PageNo   *int32    `json:"page_no"`
	Heading  *string   `json:"heading"`
	Text     string    `json:"text"`
	Audience string    `json:"audience"`
}

// ChunkPage là một trang đoạn; `next_cursor` là `ord` của đoạn cuối (chuỗi số, mờ với máy khách).
type ChunkPage struct {
	Items      []ChunkOut `json:"items"`
	NextCursor *string    `json:"next_cursor"`
}

// Chunks liệt kê đoạn theo thứ tự `ord`, `after` = ord của đoạn cuối trang trước (-1 = đầu).
func (s *Service) Chunks(ctx context.Context, course, id uuid.UUID, after int32, limit int) (ChunkPage, error) {
	if _, err := s.Get(ctx, course, id); err != nil {
		return ChunkPage{}, err
	}
	rows, err := store.New(s.Pool).DocChunks(ctx, store.DocChunksParams{CourseID: course, DocID: id, AfterOrd: after, PageLimit: int32(limit + 1)}) //nolint:gosec // ≤ 101
	if err != nil {
		return ChunkPage{}, fmt.Errorf("document: đoạn: %w", err)
	}
	out := ChunkPage{Items: make([]ChunkOut, 0, len(rows))}
	if len(rows) > limit {
		rows = rows[:limit]
		c := fmt.Sprint(rows[len(rows)-1].Ord)
		out.NextCursor = &c
	}
	for _, r := range rows {
		out.Items = append(out.Items, ChunkOut{ID: r.ID, Ord: r.Ord, PageNo: r.PageNo, Heading: r.Heading, Text: r.Text, Audience: string(r.Audience)})
	}
	return out, nil
}

// EditChunk sửa văn bản MỘT đoạn: nhúng lại một lần (`Embed`) rồi lưu + audit trước / sau trong một giao dịch. Nhúng lỗi → 503, giữ nguyên đoạn cũ.
// Tài liệu không dùng cho AI (`use_for_rag=false`) không nhúng (embedding NULL), vẫn sửa được văn bản để tìm trong thư viện.
func (s *Service) EditChunk(ctx context.Context, actor, course, id, chunkID uuid.UUID, trace, text string) (ChunkOut, error) {
	if err := s.requireActive(ctx, course); err != nil {
		return ChunkOut{}, err
	}
	text = strings.TrimSpace(text)
	if n := utf8.RuneCountInString(text); n < 1 || n > MaxChunkChars {
		return ChunkOut{}, field("text", "length", "Đoạn dài 1–4.000 ký tự.")
	}
	d, err := store.New(s.Pool).DocGetInCourse(ctx, store.DocGetInCourseParams{ID: id, CourseID: course})
	if errors.Is(err, pgx.ErrNoRows) {
		return ChunkOut{}, apierr.New(http.StatusNotFound, apierr.NotFound)
	}
	if err != nil {
		return ChunkOut{}, fmt.Errorf("document: đọc tài liệu: %w", err)
	}
	if d.CourseID != course {
		return ChunkOut{}, apierr.New(http.StatusConflict, apierr.DocumentSharedReadonly)
	}
	var vec *pgvector.Vector
	if d.UseForRag {
		if s.Embed == nil {
			return ChunkOut{}, apierr.New(http.StatusServiceUnavailable, apierr.ChatUnavailable)
		}
		v, err := s.Embed(ctx, text)
		if err != nil {
			if s.Log != nil {
				s.Log.WarnContext(ctx, "document: nhúng đoạn lỗi", "document_id", id.String(), "error", err.Error())
			}
			return ChunkOut{}, apierr.New(http.StatusServiceUnavailable, apierr.ChatUnavailable)
		}
		pv := pgvector.NewVector(v)
		vec = &pv
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return ChunkOut{}, fmt.Errorf("document: mở giao dịch: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	old, err := q.DocChunkLock(ctx, store.DocChunkLockParams{ID: chunkID, DocID: id, CourseID: course})
	if errors.Is(err, pgx.ErrNoRows) {
		return ChunkOut{}, apierr.New(http.StatusNotFound, apierr.NotFound)
	}
	if err != nil {
		return ChunkOut{}, fmt.Errorf("document: khoá đoạn: %w", err)
	}
	row, err := q.DocChunkSetText(ctx, store.DocChunkSetTextParams{ID: chunkID, DocID: id, CourseID: course, NewText: text, NewEmbedding: vec})
	if err != nil {
		return ChunkOut{}, fmt.Errorf("document: lưu đoạn: %w", err)
	}
	before, _ := json.Marshal(map[string]string{"text": old.Text})
	after, _ := json.Marshal(map[string]string{"text": text})
	if _, err := q.InsertAuditLog(ctx, store.InsertAuditLogParams{CourseID: &course, ActorID: &actor, Entity: "document_chunk", EntityID: chunkID.String(), Action: "document.chunk.edit", Before: before, After: after, TraceID: ptrIf(trace)}); err != nil {
		return ChunkOut{}, fmt.Errorf("document: audit: %w", err)
	}
	if err := q.DocTouch(ctx, id); err != nil { // `ETag` của /library đổi theo nội dung; version tăng để máy khách khác thấy
		return ChunkOut{}, fmt.Errorf("document: chạm tài liệu: %w", err)
	}
	if _, err := outbox.Write(ctx, tx, ingest.TopicDocumentChanged, map[string]string{"course_id": course.String(), "document_id": id.String()}); err != nil {
		return ChunkOut{}, fmt.Errorf("document: outbox: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return ChunkOut{}, fmt.Errorf("document: commit: %w", err)
	}
	return ChunkOut{ID: row.ID, Ord: row.Ord, PageNo: row.PageNo, Heading: row.Heading, Text: row.Text, Audience: string(row.Audience)}, nil
}

// Stats là thống kê tài liệu (một truy vấn tổng hợp + hai nhóm đếm).
type Stats struct {
	Total           int64            `json:"total"`
	ByType          map[string]int64 `json:"by_type"`
	ByStatus        map[string]int64 `json:"by_status"`
	Chunks          int64            `json:"chunks"`
	EmbeddedChunks  int64            `json:"embedded_chunks"`
	Pages           int64            `json:"pages"`
	Bytes           int64            `json:"bytes"`
	HasCoursePolicy bool             `json:"has_course_policy"`
	LastUploadAt    *time.Time       `json:"last_upload_at"`
}

// Stats thống kê tài liệu của lớp (kể cả chia sẻ vào).
func (s *Service) Stats(ctx context.Context, course uuid.UUID) (Stats, error) {
	q := store.New(s.Pool)
	r, err := q.DocStats(ctx, course)
	if err != nil {
		return Stats{}, fmt.Errorf("document: thống kê: %w", err)
	}
	out := Stats{Total: r.Total, Chunks: r.Chunks, EmbeddedChunks: r.EmbeddedChunks, Pages: r.Pages, Bytes: r.Bytes, HasCoursePolicy: r.PolicyReady > 0, ByType: map[string]int64{}, ByStatus: map[string]int64{}}
	if r.LastUploadAt.Unix() > 0 { // không có tài liệu: SQL trả epoch
		t := r.LastUploadAt
		out.LastUploadAt = &t
	}
	bt, err := q.DocStatsByType(ctx, course)
	if err != nil {
		return Stats{}, fmt.Errorf("document: thống kê theo loại: %w", err)
	}
	for _, x := range bt {
		out.ByType[x.K] = x.N
	}
	bs, err := q.DocStatsByStatus(ctx, course)
	if err != nil {
		return Stats{}, fmt.Errorf("document: thống kê theo trạng thái: %w", err)
	}
	for _, x := range bs {
		out.ByStatus[x.K] = x.N
	}
	return out, nil
}
