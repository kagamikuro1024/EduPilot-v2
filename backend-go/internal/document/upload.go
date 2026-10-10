// Package document: tải tài liệu lên (URL ký sẵn), hoàn tất → 202 + việc nền, thử lại, lập chỉ mục lại (SRS FEAT-docs-calendar 4.1, 4.5).
// Nội dung tệp KHÔNG đi qua gateway: trình duyệt PUT thẳng lên object storage; gateway chỉ cấp URL, kiểm tệp có thật, rồi xếp việc.
package document

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/ingest"
	"github.com/edupilot/backend-go/internal/jobs"
	"github.com/edupilot/backend-go/internal/platform/blob"
	"github.com/edupilot/backend-go/internal/platform/clock"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/store"
)

// Hằng số (SRS 4.1, 5.4).
const (
	DefaultMaxBytes    = 50 << 20
	DefaultPresignRate = 10
	uploadTicketTTL    = 15 * time.Minute
	presignExpiry      = 10 * time.Minute
	headBytes          = 8 << 10
	maxFilename        = 255
)

// BlobAPI là phần của platform/blob mà gói này dùng.
type BlobAPI interface {
	Stat(ctx context.Context, key string) (blob.Info, error)
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	PresignPutSized(ctx context.Context, key, contentType string, size int64) (string, error)
}

// Service là nghiệp vụ tải tài liệu.
type Service struct {
	Pool          *pgxpool.Pool
	Redis         *appredis.Client
	Blob          BlobAPI
	Jobs          *jobs.Service
	Clock         clock.Clock
	MaxBytes      int64
	PresignPerMin int
	// Embed nhúng một chuỗi (sửa đoạn); nil = sửa đoạn trả 503.
	Embed func(ctx context.Context, text string) ([]float32, error)
	Log   *slog.Logger
}

type fileKind struct {
	mime  string
	magic []byte
}

// kindOf: đuôi → loại + chữ ký đầu tệp (PDF `%PDF-`; DOCX / PPTX là zip `PK\x03\x04`).
func kindOf(ext string) (fileKind, bool) {
	switch ext {
	case ".pdf":
		return fileKind{"application/pdf", []byte("%PDF-")}, true
	case ".docx":
		return fileKind{"application/vnd.openxmlformats-officedocument.wordprocessingml.document", []byte("PK\x03\x04")}, true
	case ".pptx":
		return fileKind{"application/vnd.openxmlformats-officedocument.presentationml.presentation", []byte("PK\x03\x04")}, true
	}
	return fileKind{}, false
}

var reHash = regexp.MustCompile(`^[0-9a-fA-F]{64}$`)

// PresignIn là thân POST …/uploads/presign.
type PresignIn struct {
	Purpose   string `json:"purpose"`
	Filename  string `json:"filename"`
	MimeType  string `json:"mime_type"`
	SizeBytes int64  `json:"size_bytes"`
	SHA256    string `json:"sha256"`
}

// PresignOut là phản hồi presign.
type PresignOut struct {
	UploadID  uuid.UUID         `json:"upload_id"`
	Method    string            `json:"method"`
	URL       string            `json:"url"`
	Headers   map[string]string `json:"headers"`
	BlobKey   string            `json:"blob_key"`
	ExpiresIn int               `json:"expires_in"`
}

func field(f, code, msg string) *apierr.Error {
	return apierr.Validation(apierr.FieldError{Field: f, Code: code, Message: msg})
}

func (s *Service) maxBytes() int64 {
	if s.MaxBytes > 0 {
		return s.MaxBytes
	}
	return DefaultMaxBytes
}

func (s *Service) requireActive(ctx context.Context, course uuid.UUID) error {
	st, err := store.New(s.Pool).DocCourseStatus(ctx, course)
	if errors.Is(err, pgx.ErrNoRows) {
		return apierr.New(http.StatusNotFound, apierr.NotFound)
	}
	if err != nil {
		return fmt.Errorf("document: đọc lớp: %w", err)
	}
	if st == store.CourseStatusARCHIVED {
		return apierr.New(http.StatusConflict, apierr.CourseArchived)
	}
	return nil
}

// Presign kiểm đầu vào rồi cấp URL PUT ký sẵn (hạn 10 phút, ký kèm Content-Length) và ghi vé ep:upload:{id} (15 phút).
func (s *Service) Presign(ctx context.Context, actor, course uuid.UUID, in PresignIn) (PresignOut, error) {
	if err := s.rate(ctx, actor); err != nil {
		return PresignOut{}, err
	}
	if err := s.requireActive(ctx, course); err != nil {
		return PresignOut{}, err
	}
	if in.Purpose != "document" {
		return PresignOut{}, field("purpose", "enum", "Mục đích tải lên không hợp lệ.")
	}
	name := in.Filename
	if name == "" || utf8.RuneCountInString(name) > maxFilename || strings.ContainsAny(name, `/\`) || strings.Contains(name, "..") || strings.ContainsFunc(name, func(r rune) bool { return r < 0x20 || r == 0x7f }) {
		return PresignOut{}, field("filename", "invalid", "Tên tệp không hợp lệ.")
	}
	kind, ok := kindOf(strings.ToLower(path.Ext(name)))
	if !ok || in.MimeType != kind.mime {
		return PresignOut{}, apierr.New(http.StatusUnprocessableEntity, apierr.FileTypeNotAllowed)
	}
	if in.SizeBytes <= 0 || in.SizeBytes > s.maxBytes() {
		return PresignOut{}, apierr.New(http.StatusUnprocessableEntity, apierr.FileTooLarge)
	}
	if !reHash.MatchString(in.SHA256) {
		return PresignOut{}, field("sha256", "format", "Mã băm tệp không hợp lệ.")
	}
	id := uuid.New()
	key := fmt.Sprintf("courses/%s/documents/%s/%s", course, id, name)
	url, err := s.Blob.PresignPutSized(ctx, key, kind.mime, in.SizeBytes)
	if err != nil {
		return PresignOut{}, fmt.Errorf("document: ký URL tải lên: %w", err)
	}
	rk := appredis.Key("upload", id.String())
	pipe := s.Redis.TxPipeline()
	pipe.HSet(ctx, rk, map[string]any{"user_id": actor.String(), "course_id": course.String(), "filename": name, "size": in.SizeBytes, "mime": kind.mime,
		"sha256": strings.ToLower(in.SHA256), "blob_key": key})
	pipe.Expire(ctx, rk, uploadTicketTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return PresignOut{}, fmt.Errorf("document: ghi vé tải lên: %w", err)
	}
	return PresignOut{UploadID: id, Method: http.MethodPut, URL: url, Headers: map[string]string{"Content-Type": kind.mime, "Content-Length": fmt.Sprint(in.SizeBytes)},
		BlobKey: key, ExpiresIn: int(presignExpiry.Seconds())}, nil
}

// rate: PresignPerMin lượt / phút / người (ep:rl:upload:{uid}:{phút}, TTL 120 s).
func (s *Service) rate(ctx context.Context, actor uuid.UUID) error {
	limit := s.PresignPerMin
	if limit <= 0 {
		limit = DefaultPresignRate
	}
	key := appredis.Key("rl", "upload", actor.String(), fmt.Sprint(s.Clock.Now().Unix()/60))
	pipe := s.Redis.TxPipeline()
	n := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, appredis.TTLRateLimit)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("document: giới hạn tải lên: %w", err)
	}
	if int(n.Val()) > limit {
		return apierr.New(http.StatusTooManyRequests, apierr.RateLimited).WithRetryAfter(60 - int(s.Clock.Now().Unix()%60))
	}
	return nil
}

// CompleteIn là thân POST …/uploads/complete.
type CompleteIn struct {
	UploadID          uuid.UUID `json:"upload_id"`
	Title             string    `json:"title"`
	Type              string    `json:"type"`
	UseForRAG         *bool     `json:"use_for_rag"`
	VisibleToStudents *bool     `json:"visible_to_students"`
	Category          *string   `json:"category"`
	WeekNo            *int16    `json:"week_no"`
}

// Doc là dạng trả về của tài liệu (không lộ blob_key).
type Doc struct {
	ID                uuid.UUID `json:"id"`
	CourseID          uuid.UUID `json:"course_id"`
	Title             string    `json:"title"`
	Type              string    `json:"type"`
	Filename          *string   `json:"filename"`
	MimeType          *string   `json:"mime_type"`
	SizeBytes         *int64    `json:"size_bytes"`
	Status            string    `json:"status"`
	Error             *string   `json:"error"`
	PageCount         *int32    `json:"page_count"`
	VisibleToStudents bool      `json:"visible_to_students"`
	UseForRAG         bool      `json:"use_for_rag"`
	Category          *string   `json:"category"`
	WeekNo            *int16    `json:"week_no"`
	Version           int32     `json:"version"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func docOf(d store.Document) Doc {
	return Doc{ID: d.ID, CourseID: d.CourseID, Title: d.Title, Type: string(d.Type), Filename: d.Filename, MimeType: d.MimeType, SizeBytes: d.SizeBytes, Status: string(d.Status),
		Error: d.Error, PageCount: d.PageCount, VisibleToStudents: d.VisibleToStudents, UseForRAG: d.UseForRag, Category: d.Category, WeekNo: d.WeekNo, Version: d.Version,
		CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt}
}

// CompleteOut là phản hồi 202.
type CompleteOut struct {
	JobID    uuid.UUID `json:"job_id"`
	Document Doc       `json:"document"`
}

// DefaultFlags: cờ mặc định theo loại (SRS 4.1): ANSWER_KEY ẩn với sinh viên; đề tham khảo không dùng cho AI.
func DefaultFlags(t store.DocumentType) (visible, useRAG bool) {
	switch t {
	case store.DocumentTypeANSWERKEY:
		return false, true
	case store.DocumentTypeEXAMPAPER:
		return true, false
	}
	return true, true
}

// Complete kiểm tệp có thật (Stat + 8 KiB đầu), rồi TRONG MỘT giao dịch: khoá tư vấn theo (lớp, băm) → kiểm trùng → chèn documents QUEUED + việc ingest.
func (s *Service) Complete(ctx context.Context, actor, course uuid.UUID, in CompleteIn) (CompleteOut, error) {
	t := store.DocumentType(in.Type)
	switch t {
	case store.DocumentTypeLECTURE, store.DocumentTypeCOURSEPOLICY, store.DocumentTypeEXAMPAPER, store.DocumentTypeANSWERKEY, store.DocumentTypeOTHER:
	default:
		return CompleteOut{}, field("type", "enum", "Loại tài liệu không hợp lệ.")
	}
	title := strings.TrimSpace(in.Title)
	switch {
	case utf8.RuneCountInString(title) < 1 || utf8.RuneCountInString(title) > 200:
		return CompleteOut{}, field("title", "length", "Tên tài liệu dài 1–200 ký tự.")
	case in.Category != nil && utf8.RuneCountInString(*in.Category) > 40:
		return CompleteOut{}, field("category", "length", "Chủ đề tối đa 40 ký tự.")
	case in.WeekNo != nil && (*in.WeekNo < 1 || *in.WeekNo > 20):
		return CompleteOut{}, field("week_no", "range", "Tuần từ 1 đến 20.")
	}
	visible, useRAG := DefaultFlags(t)
	if in.VisibleToStudents != nil {
		visible = *in.VisibleToStudents
	}
	if in.UseForRAG != nil {
		useRAG = *in.UseForRAG
	}
	if t == store.DocumentTypeANSWERKEY && visible {
		return CompleteOut{}, field("visible_to_students", "answer_key", "Đáp án không được hiện cho sinh viên.")
	}

	tk, err := s.Redis.HGetAll(ctx, appredis.Key("upload", in.UploadID.String())).Result()
	if err != nil {
		return CompleteOut{}, fmt.Errorf("document: đọc vé tải lên: %w", err)
	}
	if len(tk) == 0 || tk["user_id"] != actor.String() || tk["course_id"] != course.String() {
		return CompleteOut{}, apierr.New(http.StatusNotFound, apierr.UploadNotFound) // của người khác / lớp khác / hết hạn: một câu trả lời
	}
	if err := s.requireActive(ctx, course); err != nil {
		return CompleteOut{}, err
	}
	key, filename, hash := tk["blob_key"], tk["filename"], tk["sha256"]
	var size int64
	_, _ = fmt.Sscan(tk["size"], &size)
	kind, _ := kindOf(strings.ToLower(path.Ext(filename)))

	info, err := s.Blob.Stat(ctx, key)
	switch {
	case errors.Is(err, blob.ErrNotFound):
		return CompleteOut{}, apierr.New(http.StatusUnprocessableEntity, apierr.UploadIncomplete)
	case err != nil:
		return CompleteOut{}, fmt.Errorf("document: kiểm tệp: %w", err)
	case info.Size != size:
		s.dropObject(ctx, key)
		return CompleteOut{}, apierr.New(http.StatusUnprocessableEntity, apierr.UploadIncomplete)
	}
	head, err := s.readHead(ctx, key)
	if err != nil {
		return CompleteOut{}, fmt.Errorf("document: đọc đầu tệp: %w", err)
	}
	if !bytes.HasPrefix(head, kind.magic) {
		s.dropObject(ctx, key)
		return CompleteOut{}, apierr.New(http.StatusUnprocessableEntity, apierr.FileTypeMismatch)
	}

	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return CompleteOut{}, fmt.Errorf("document: mở giao dịch: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	if err := q.DocLockHash(ctx, store.DocLockHashParams{CourseID: course.String(), Sha256: hash}); err != nil {
		return CompleteOut{}, fmt.Errorf("document: khoá băm: %w", err)
	}
	if id, err := q.DocFindDuplicateInCourse(ctx, store.DocFindDuplicateInCourseParams{Sha256: &hash, CourseID: course}); err == nil {
		s.dropObject(ctx, key)
		return CompleteOut{}, apierr.New(http.StatusConflict, apierr.DocumentDuplicate).WithDetails(map[string]string{"existing_id": id.String()})
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return CompleteOut{}, fmt.Errorf("document: kiểm trùng: %w", err)
	}
	if other, err := q.DocFindDuplicateElsewhere(ctx, store.DocFindDuplicateElsewhereParams{Sha256: &hash, CourseID: course, UserID: actor}); err == nil {
		s.dropObject(ctx, key)
		return CompleteOut{}, apierr.New(http.StatusConflict, apierr.DocumentDuplicateElsewhere).WithDetails(map[string]string{"class_code": other.ClassCode, "document_id": other.ID.String()})
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return CompleteOut{}, fmt.Errorf("document: kiểm trùng lớp khác: %w", err)
	}
	doc, err := q.DocInsert(ctx, store.DocInsertParams{CourseID: course, Title: title, Type: t, Filename: &filename, MimeType: &kind.mime, SizeBytes: &size, Sha256: &hash, BlobKey: &key,
		VisibleToStudents: visible, UseForRag: useRAG, Category: in.Category, WeekNo: in.WeekNo, UploadedBy: &actor})
	if err != nil {
		return CompleteOut{}, fmt.Errorf("document: chèn tài liệu: %w", err)
	}
	job, err := s.Jobs.EnqueueTx(ctx, tx, actor, ingest.KindIngest, ingest.KindPayload{DocumentID: doc.ID})
	if err != nil {
		return CompleteOut{}, fmt.Errorf("document: xếp việc: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return CompleteOut{}, fmt.Errorf("document: commit: %w", err)
	}
	_ = s.Redis.Del(ctx, appredis.Key("upload", in.UploadID.String())).Err()
	return CompleteOut{JobID: job.ID, Document: docOf(doc)}, nil
}

func (s *Service) readHead(ctx context.Context, key string) ([]byte, error) {
	rc, err := s.Blob.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("mở tệp: %w", err)
	}
	defer func() { _ = rc.Close() }()
	buf := make([]byte, headBytes)
	n, err := io.ReadFull(rc, buf)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("đọc: %w", err)
	}
	return buf[:n], nil
}

func (s *Service) dropObject(ctx context.Context, key string) {
	if err := s.Blob.Delete(context.WithoutCancel(ctx), key); err != nil && s.Log != nil {
		s.Log.WarnContext(ctx, "document: không xoá được tệp bị từ chối", "error", err.Error())
	}
}

// JobOut là phản hồi 202 của retry / reindex.
type JobOut struct {
	JobID uuid.UUID `json:"job_id"`
}

// Retry: FAILED → QUEUED, job mới + XADD (qua jobs). Chỉ tài liệu của chính lớp.
func (s *Service) Retry(ctx context.Context, actor, course, docID uuid.UUID) (JobOut, error) {
	if err := s.requireActive(ctx, course); err != nil {
		return JobOut{}, err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return JobOut{}, fmt.Errorf("document: mở giao dịch: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	if _, err := q.DocRetry(ctx, store.DocRetryParams{ID: docID, CourseID: course}); err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return JobOut{}, fmt.Errorf("document: thử lại: %w", err)
		}
		d, gerr := q.DocGetInCourse(ctx, store.DocGetInCourseParams{ID: docID, CourseID: course})
		switch {
		case errors.Is(gerr, pgx.ErrNoRows):
			return JobOut{}, apierr.New(http.StatusNotFound, apierr.NotFound)
		case gerr != nil:
			return JobOut{}, fmt.Errorf("document: đọc tài liệu: %w", gerr)
		case d.CourseID != course:
			return JobOut{}, apierr.New(http.StatusConflict, apierr.DocumentSharedReadonly)
		}
		return JobOut{}, apierr.New(http.StatusConflict, apierr.DocumentNotFailed)
	}
	job, err := s.Jobs.EnqueueTx(ctx, tx, actor, ingest.KindIngest, ingest.KindPayload{DocumentID: docID})
	if err != nil {
		return JobOut{}, fmt.Errorf("document: xếp việc: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return JobOut{}, fmt.Errorf("document: commit: %w", err)
	}
	return JobOut{JobID: job.ID}, nil
}

// Reindex nhúng lại MỘT tài liệu READY của lớp (không đổi status, không gọi docling).
func (s *Service) Reindex(ctx context.Context, actor, course, docID uuid.UUID) (JobOut, error) {
	if err := s.requireActive(ctx, course); err != nil {
		return JobOut{}, err
	}
	d, err := store.New(s.Pool).DocGetInCourse(ctx, store.DocGetInCourseParams{ID: docID, CourseID: course})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return JobOut{}, apierr.New(http.StatusNotFound, apierr.NotFound)
	case err != nil:
		return JobOut{}, fmt.Errorf("document: đọc tài liệu: %w", err)
	case d.CourseID != course:
		return JobOut{}, apierr.New(http.StatusConflict, apierr.DocumentSharedReadonly)
	case d.Status != store.DocumentStatusREADY:
		return JobOut{}, apierr.New(http.StatusConflict, apierr.DocumentNotReady)
	}
	job, err := s.Jobs.Enqueue(ctx, actor, ingest.KindReindex, ingest.KindPayload{DocumentID: docID})
	if err != nil {
		return JobOut{}, fmt.Errorf("document: xếp việc: %w", err)
	}
	return JobOut{JobID: job.ID}, nil
}

// ReindexAll nhúng lại mọi tài liệu READY + use_for_rag của lớp (Giảng viên, sau khi Admin đổi mô hình nhúng).
func (s *Service) ReindexAll(ctx context.Context, actor, course uuid.UUID) (JobOut, error) {
	if err := s.requireActive(ctx, course); err != nil {
		return JobOut{}, err
	}
	job, err := s.Jobs.Enqueue(ctx, actor, ingest.KindReindexAll, ingest.KindPayload{CourseID: course})
	if err != nil {
		return JobOut{}, fmt.Errorf("document: xếp việc: %w", err)
	}
	return JobOut{JobID: job.ID}, nil
}
