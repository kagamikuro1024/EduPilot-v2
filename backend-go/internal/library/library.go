// Package library: thư viện tài liệu của SINH VIÊN (SRS FEAT-docs-calendar 4.4, 4.6): tìm, xem, tải, và tool `search_library`.
// Mọi truy vấn chỉ thấy tài liệu READY + visible_to_students + không ANSWER_KEY của lớp (kể cả chia sẻ vào) — không bao giờ có nhánh nào khác.
package library

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/edupilot/backend-go/internal/agent"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
	"github.com/edupilot/backend-go/internal/platform/blob"
	"github.com/edupilot/backend-go/internal/store"
)

// MinQuery: `q` ngắn hơn bị bỏ qua (coi như không có `q`, không 422 — #12).
const MinQuery = 2

// BlobAPI là phần của platform/blob mà gói này dùng.
type BlobAPI interface {
	Stat(ctx context.Context, key string) (blob.Info, error)
	PresignGet(ctx context.Context, key, filename string) (string, error)
	PresignGetInline(ctx context.Context, key string) (string, error)
}

// Service là thư viện của sinh viên.
type Service struct {
	Pool *pgxpool.Pool
	Blob BlobAPI
	Log  *slog.Logger
}

// Filter là bộ lọc danh sách.
type Filter struct {
	Q, Type, Category string
	Week              *int
}

// Item là một dòng danh sách gọn.
type Item struct {
	ID        uuid.UUID `json:"id"`
	Title     string    `json:"title"`
	Type      string    `json:"type"`
	FileKind  string    `json:"file_kind"` // PDF | DOCX | PPTX
	Category  *string   `json:"category"`
	WeekNo    *int16    `json:"week_no"`
	UpdatedAt time.Time `json:"updated_at"`
	Snippet   string    `json:"snippet"`
	CanAskAI  bool      `json:"can_ask_ai"`
}

// Page là một trang danh sách.
type Page struct {
	Items      []Item  `json:"items"`
	NextCursor *string `json:"next_cursor"`
}

func escapeLike(q string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(q)
}

func fileKind(filename *string) string {
	if filename != nil {
		switch strings.ToLower(filename2ext(*filename)) {
		case ".pdf":
			return "PDF"
		case ".docx":
			return "DOCX"
		case ".pptx":
			return "PPTX"
		}
	}
	return ""
}

func filename2ext(f string) string {
	if i := strings.LastIndexByte(f, '.'); i >= 0 {
		return f[i:]
	}
	return ""
}

// List: tìm / liệt kê. `q` ≥ 2 ký tự khớp tên / tệp / chủ đề (không dấu) hoặc từ khoá trong đoạn; mới cập nhật trước, con trỏ (updated_at, id).
func (s *Service) List(ctx context.Context, course uuid.UUID, f Filter, p httpx.PageParams) (Page, error) {
	arg := store.LibListParams{CourseID: course, PageLimit: int32(p.Fetch())} //nolint:gosec // ≤ 101
	if q := strings.TrimSpace(f.Q); utf8.RuneCountInString(q) >= MinQuery {
		e := escapeLike(q) // LIKE cần thoát; vn_bigram_query bỏ mọi ký tự không phải chữ-số nên dấu `\` vô hại
		arg.Q = &e
	}
	if f.Type != "" {
		arg.Type = &f.Type
	}
	if f.Category != "" {
		arg.Category = &f.Category
	}
	if f.Week != nil {
		w := int32(*f.Week) //nolint:gosec // 1–20 đã kiểm ở handler
		arg.WeekNo = &w
	}
	if p.Cursor != nil {
		arg.CursorAt, arg.CursorID = &p.Cursor.CreatedAt, &p.Cursor.ID
	}
	rows, err := store.New(s.Pool).LibList(ctx, arg)
	if err != nil {
		return Page{}, fmt.Errorf("library: danh sách: %w", err)
	}
	out := Page{Items: make([]Item, 0, len(rows))}
	if len(rows) > p.Limit {
		rows = rows[:p.Limit]
		c := httpx.EncodeCursor(rows[len(rows)-1].UpdatedAt, rows[len(rows)-1].ID.String())
		out.NextCursor = &c
	}
	for _, r := range rows {
		out.Items = append(out.Items, Item{ID: r.ID, Title: r.Title, Type: string(r.Type), FileKind: fileKind(r.Filename), Category: r.Category, WeekNo: r.WeekNo, UpdatedAt: r.UpdatedAt, Snippet: r.Snippet, CanAskAI: r.CanAskAi})
	}
	return out, nil
}

// Detail là chi tiết một tài liệu.
type Detail struct {
	Item
	SizeBytes  *int64  `json:"size_bytes"`
	PageCount  *int32  `json:"page_count"`
	PreviewURL *string `json:"preview_url"`
}

func (s *Service) load(ctx context.Context, course, id uuid.UUID) (store.LibGetRow, error) {
	d, err := store.New(s.Pool).LibGet(ctx, store.LibGetParams{ID: id, CourseID: course})
	if errors.Is(err, pgx.ErrNoRows) {
		return d, apierr.New(http.StatusNotFound, apierr.NotFound) // không được phép = không có: không lộ tồn tại
	}
	if err != nil {
		return d, fmt.Errorf("library: đọc tài liệu: %w", err)
	}
	return d, nil
}

// Get: metadata + `preview_url` (chỉ PDF, ký sẵn 5 phút, inline) + `can_ask_ai`. Tệp đã mất → preview_url null (vẫn đọc được thông tin).
func (s *Service) Get(ctx context.Context, course, id uuid.UUID) (Detail, error) {
	d, err := s.load(ctx, course, id)
	if err != nil {
		return Detail{}, err
	}
	out := Detail{Item: Item{ID: d.ID, Title: d.Title, Type: string(d.Type), FileKind: fileKind(d.Filename), Category: d.Category, WeekNo: d.WeekNo, UpdatedAt: d.UpdatedAt, CanAskAI: d.CanAskAi},
		SizeBytes: d.SizeBytes, PageCount: d.PageCount}
	if out.FileKind == "PDF" && d.BlobKey != nil && s.Blob != nil {
		if _, err := s.Blob.Stat(ctx, *d.BlobKey); err == nil {
			if u, err := s.Blob.PresignGetInline(ctx, *d.BlobKey); err == nil {
				out.PreviewURL = &u
			}
		}
	}
	return out, nil
}

// Download kiểm quyền, kiểm tệp còn, ký URL 5 phút (attachment) rồi tăng `download_count` bằng một câu nguyên tử.
func (s *Service) Download(ctx context.Context, course, id uuid.UUID) (string, error) {
	d, err := s.load(ctx, course, id)
	if err != nil {
		return "", err
	}
	if d.BlobKey == nil || s.Blob == nil {
		return "", apierr.New(http.StatusNotFound, apierr.FileGone)
	}
	if _, err := s.Blob.Stat(ctx, *d.BlobKey); err != nil {
		if errors.Is(err, blob.ErrNotFound) {
			return "", apierr.New(http.StatusNotFound, apierr.FileGone)
		}
		return "", fmt.Errorf("library: kiểm tệp: %w", err)
	}
	name := d.Title
	if d.Filename != nil {
		name = *d.Filename
	}
	u, err := s.Blob.PresignGet(ctx, *d.BlobKey, name)
	if err != nil {
		return "", fmt.Errorf("library: ký URL: %w", err)
	}
	if err := store.New(s.Pool).LibCountDownload(ctx, d.ID); err != nil {
		return "", fmt.Errorf("library: đếm lượt tải: %w", err)
	}
	return u, nil
}

// Search là agent.LibrarySource (tool search_library): tối đa 5 tài liệu sinh viên được thấy; không tham số danh tính — lớp lấy từ TrustedContext.
func (s *Service) Search(ctx context.Context, tc agent.TrustedContext, query string) (agent.Facts, []agent.Source, bool, error) {
	q := strings.TrimSpace(query)
	if utf8.RuneCountInString(q) < MinQuery {
		return nil, nil, false, nil
	}
	rows, err := store.New(s.Pool).LibSearchTool(ctx, store.LibSearchToolParams{CourseID: tc.CourseID, Q: escapeLike(q)})
	if err != nil {
		return nil, nil, false, fmt.Errorf("library: tìm cho tool: %w", err)
	}
	if len(rows) == 0 {
		return nil, nil, false, nil
	}
	res := make([]map[string]any, 0, len(rows))
	srcs := make([]agent.Source, 0, len(rows))
	for _, r := range rows {
		m := map[string]any{"document_id": r.ID, "title": r.Title, "type": string(r.Type), "week_no": r.WeekNo, "snippet": r.Snippet, "href": "/library/" + r.ID.String()}
		src := agent.Source{Title: r.Title}
		if r.PageNo > 0 {
			m["page_no"] = r.PageNo
			src.Page = int(r.PageNo)
		}
		res = append(res, m)
		srcs = append(srcs, src)
	}
	return agent.Facts{"results": res}, srcs, true, nil
}
