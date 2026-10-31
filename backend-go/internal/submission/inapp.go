// Package submission: nguồn nộp bài (SubmissionSource) và các bước chung — hiện có InAppSource (US-P7-03). Mọi nguồn cấp `version` dưới cùng một khoá (SRS FEAT-assignments 4.3–4.4).
package submission

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"path"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/edupilot/backend-go/internal/grading"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/platform/blob"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/store"
)

// Hằng số mặc định (SRS 5): ASSIGN_MAX_BYTES, SUBMIT_PRESIGN_PER_MIN, SUBMIT_UPLOAD_TTL, ORPHAN_UPLOAD_TTL.
const (
	DefaultMaxBytes      = 20 << 20
	DefaultPresignPerMin = 10
	UploadTTL            = 15 * time.Minute
	presignExpiry        = 10 * time.Minute
	OrphanTTL            = 24 * time.Hour
	maxFilename          = 255

	// TopicCreated là sự kiện outbox khi có bài nộp mới (mọi nguồn).
	TopicCreated = "submission.created"
)

var reHash = regexp.MustCompile(`^[0-9a-fA-F]{64}$`) //nolint:gochecknoglobals // biểu thức chính quy bất biến

// BlobAPI là phần của platform/blob mà gói này dùng.
type BlobAPI interface {
	Reader
	Stat(ctx context.Context, key string) (blob.Info, error)
	Delete(ctx context.Context, key string) error
	PresignPutChecked(ctx context.Context, key, contentType string, size int64, sha256Hex string) (string, error)
	List(ctx context.Context, prefix string) ([]blob.Info, error)
}

// Service là InAppSource: presign + complete.
type Service struct {
	Pool          *pgxpool.Pool
	Redis         *appredis.Client
	Blob          BlobAPI
	Clock         clock.Clock
	Log           *slog.Logger
	MaxBytes      int64
	PresignPerMin int
}

func (s *Service) maxBytes() int64 {
	if s.MaxBytes > 0 {
		return s.MaxBytes
	}
	return DefaultMaxBytes
}

func field(f, code, msg string) *apierr.Error {
	return apierr.Validation(apierr.FieldError{Field: f, Code: code, Message: msg})
}

func asgNotFound() *apierr.Error { return apierr.New(http.StatusNotFound, apierr.AssignmentNotFound) }

// PresignIn là thân POST …/submit/presign (KHÔNG có mime_type và không có trường danh tính).
type PresignIn struct {
	Filename  string `json:"filename"`
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

// SafeName làm sạch tên tệp: bỏ thư mục, thay `/ \` và ký tự điều khiển bằng `_`, bỏ `..`, cắt ≤ 255 ký tự giữ đuôi. Rỗng → "".
func SafeName(name string) string {
	name = strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r < 0x20 || r == 0x7f {
			return '_'
		}
		return r
	}, name)
	for strings.Contains(name, "..") {
		name = strings.ReplaceAll(name, "..", ".")
	}
	name = strings.TrimSpace(name)
	if utf8.RuneCountInString(name) > maxFilename {
		ext := path.Ext(name)
		rs := []rune(strings.TrimSuffix(name, ext))
		name = string(rs[:maxFilename-utf8.RuneCountInString(ext)]) + ext
	}
	return name
}

// Gate là cửa nộp tại thời điểm t (SRS 4.3.3): trả số ngày muộn (0 = đúng hạn); đóng → 409 SUBMISSION_CLOSED.
func Gate(a store.Assignment, t time.Time) (int, error) {
	if !t.After(a.DueAt) {
		return 0, nil
	}
	if !a.AllowLate || a.LateMaxDays < 1 || t.After(a.DueAt.Add(time.Duration(a.LateMaxDays)*24*time.Hour)) {
		return 0, apierr.New(http.StatusConflict, apierr.SubmissionClosed)
	}
	return grading.LateDays(&t, a.DueAt), nil
}

func (s *Service) now() time.Time {
	if s.Clock == nil {
		return time.Now().UTC()
	}
	return s.Clock.Now()
}

// open nạp bài và kiểm điều kiện chung của sinh viên: lớp còn hoạt động, bài đã công bố (chưa → 404, không lộ), kênh IN_APP bật, cửa nộp còn mở.
func (s *Service) open(ctx context.Context, q *store.Queries, course, aid uuid.UUID) (store.Assignment, int, error) {
	st, err := q.CalCourseStatus(ctx, course)
	if errors.Is(err, pgx.ErrNoRows) {
		return store.Assignment{}, 0, apierr.New(http.StatusNotFound, apierr.NotFound)
	}
	if err != nil {
		return store.Assignment{}, 0, fmt.Errorf("submission: đọc lớp: %w", err)
	}
	if st == store.CourseStatusARCHIVED {
		return store.Assignment{}, 0, apierr.New(http.StatusConflict, apierr.CourseArchived)
	}
	a, err := q.AssignmentGet(ctx, store.AssignmentGetParams{CourseID: course, ID: aid})
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && a.PublishedAt == nil) {
		return store.Assignment{}, 0, asgNotFound()
	}
	if err != nil {
		return store.Assignment{}, 0, fmt.Errorf("submission: đọc bài: %w", err)
	}
	if !slices.Contains(a.Channels, "IN_APP") {
		return store.Assignment{}, 0, apierr.New(http.StatusConflict, apierr.ChannelNotAllowed)
	}
	late, err := Gate(a, s.now())
	return a, late, err
}

// rate: PresignPerMin lượt / phút / người (ep:rl:submit:{uid}:{phút}).
func (s *Service) rate(ctx context.Context, actor uuid.UUID) error {
	limit := s.PresignPerMin
	if limit <= 0 {
		limit = DefaultPresignPerMin
	}
	key := appredis.Key("rl", "submit", actor.String(), fmt.Sprint(s.now().Unix()/60))
	pipe := s.Redis.TxPipeline()
	n := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, appredis.TTLRateLimit)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("submission: giới hạn tải lên: %w", err)
	}
	if int(n.Val()) > limit {
		return apierr.New(http.StatusTooManyRequests, apierr.RateLimited).WithRetryAfter(60 - int(s.now().Unix()%60))
	}
	return nil
}

// Presign kiểm điều kiện + tệp rồi cấp URL PUT ký sẵn (10 phút; ký kèm Content-Length, Content-Type suy từ đuôi và x-amz-checksum-sha256) và ghi vé `ep:submit:{id}` (15 phút).
func (s *Service) Presign(ctx context.Context, actor, course, aid uuid.UUID, in PresignIn) (PresignOut, error) {
	if s.Blob == nil || s.Redis == nil {
		return PresignOut{}, apierr.New(http.StatusServiceUnavailable, apierr.ServiceUnavailable)
	}
	if err := s.rate(ctx, actor); err != nil {
		return PresignOut{}, err
	}
	if _, _, err := s.open(ctx, store.New(s.Pool), course, aid); err != nil {
		return PresignOut{}, err
	}
	name := SafeName(in.Filename)
	if name == "" {
		return PresignOut{}, field("filename", "invalid", "Tên tệp không hợp lệ.")
	}
	kind, ok := KindOf(strings.ToLower(path.Ext(name)))
	if !ok {
		return PresignOut{}, apierr.New(http.StatusUnprocessableEntity, apierr.FileTypeNotAllowed)
	}
	switch {
	case in.SizeBytes <= 0:
		return PresignOut{}, apierr.New(http.StatusUnprocessableEntity, apierr.FileEmpty)
	case in.SizeBytes > s.maxBytes():
		return PresignOut{}, apierr.New(http.StatusUnprocessableEntity, apierr.FileTooLarge)
	case !reHash.MatchString(in.SHA256):
		return PresignOut{}, field("sha256", "format", "Mã băm tệp không hợp lệ.")
	}
	hash := strings.ToLower(in.SHA256)
	id := uuid.New()
	key := fmt.Sprintf("courses/%s/submissions/%s/%s", course, id, name)
	url, err := s.Blob.PresignPutChecked(ctx, key, kind.Mime, in.SizeBytes, hash)
	if err != nil {
		return PresignOut{}, fmt.Errorf("submission: ký URL tải lên: %w", err)
	}
	sum, _ := blob.ChecksumHeader(hash)
	rk := appredis.Key("submit", id.String())
	pipe := s.Redis.TxPipeline()
	pipe.HSet(ctx, rk, map[string]any{"user_id": actor.String(), "course_id": course.String(), "assignment_id": aid.String(), "filename": name, "size": in.SizeBytes, "sha256": hash, "blob_key": key})
	pipe.Expire(ctx, rk, UploadTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return PresignOut{}, fmt.Errorf("submission: ghi vé tải lên: %w", err)
	}
	return PresignOut{UploadID: id, Method: http.MethodPut, URL: url, BlobKey: key, ExpiresIn: int(presignExpiry.Seconds()),
		Headers: map[string]string{"Content-Type": kind.Mime, "Content-Length": fmt.Sprint(in.SizeBytes), "x-amz-checksum-sha256": sum}}, nil
}

// Submitted là bản ghi bài nộp trả về cho sinh viên (`late_days` tính khi đọc, chỉ để hiển thị).
type Submitted struct {
	ID         uuid.UUID `json:"id"`
	Version    int       `json:"version"`
	ReceivedAt time.Time `json:"received_at"`
	LateDays   int       `json:"late_days"`
	FileName   string    `json:"file_name"`
}

// Complete (Idempotency-Key ở middleware): kiểm vé, kích thước thật, nội dung, cửa nộp, khoá nộp lại; một giao dịch cấp `version` + ghi `submissions` + audit + outbox.
func (s *Service) Complete(ctx context.Context, actor, course, aid, uploadID uuid.UUID, trace string) (Submitted, error) {
	if s.Blob == nil || s.Redis == nil {
		return Submitted{}, apierr.New(http.StatusServiceUnavailable, apierr.ServiceUnavailable)
	}
	rk := appredis.Key("submit", uploadID.String())
	tk, err := s.Redis.HGetAll(ctx, rk).Result()
	if err != nil {
		return Submitted{}, fmt.Errorf("submission: đọc vé tải lên: %w", err)
	}
	if len(tk) == 0 || tk["user_id"] != actor.String() || tk["course_id"] != course.String() || tk["assignment_id"] != aid.String() {
		return Submitted{}, apierr.New(http.StatusNotFound, apierr.UploadNotFound) // của người khác / bài khác / hết hạn: một câu trả lời
	}
	if _, _, err := s.open(ctx, store.New(s.Pool), course, aid); err != nil {
		return Submitted{}, err
	}
	key, name, hash := tk["blob_key"], tk["filename"], tk["sha256"]
	var size int64
	_, _ = fmt.Sscan(tk["size"], &size)
	kind, _ := KindOf(strings.ToLower(path.Ext(name)))
	info, err := s.Blob.Stat(ctx, key)
	switch {
	case errors.Is(err, blob.ErrNotFound):
		return Submitted{}, apierr.New(http.StatusUnprocessableEntity, apierr.UploadIncomplete)
	case err != nil:
		return Submitted{}, fmt.Errorf("submission: kiểm tệp: %w", err)
	case info.Size != size || info.Size > s.maxBytes():
		s.drop(ctx, key)
		return Submitted{}, apierr.New(http.StatusUnprocessableEntity, apierr.UploadIncomplete)
	}
	if !kind.Sniff(ctx, s.Blob, key, size) {
		s.drop(ctx, key)
		return Submitted{}, apierr.New(http.StatusUnprocessableEntity, apierr.FileContentMismatch)
	}

	var out Submitted
	err = pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		a, _, err := s.open(ctx, q, course, aid) // kiểm lại trong giao dịch (hạn có thể vừa qua)
		if err != nil {
			return err
		}
		if err := q.SubmissionLockStudent(ctx, store.SubmissionLockStudentParams{AssignmentID: aid.String(), StudentID: actor.String()}); err != nil {
			return fmt.Errorf("submission: khoá nộp: %w", err)
		}
		if cur, err := q.SubmissionLatestMatched(ctx, store.SubmissionLatestMatchedParams{CourseID: course, AssignmentID: aid, StudentID: actor}); err == nil {
			if cur.ResultStatus == "REVIEWED" || cur.ResultStatus == "PUBLISHED" {
				return apierr.New(http.StatusConflict, apierr.SubmissionLocked) // câu chung: không lộ trạng thái trước công bố (TLR-17)
			}
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("submission: bản mới nhất: %w", err)
		}
		at := s.now()
		row, err := q.SubmissionInsertInApp(ctx, store.SubmissionInsertInAppParams{CourseID: course, AssignmentID: &aid, StudentID: &actor, BlobKey: key, FileName: name,
			FileSize: size, FileMime: kind.Mime, Sha256: hash, At: at})
		if err != nil {
			return fmt.Errorf("submission: ghi bài nộp: %w", err)
		}
		after, _ := json.Marshal(map[string]any{"assignment_id": aid, "version": row.Version, "source": "IN_APP"})
		var tr *string
		if trace != "" {
			tr = &trace
		}
		if _, err := q.InsertAuditLog(ctx, store.InsertAuditLogParams{CourseID: &course, ActorID: &actor, Entity: "submissions", EntityID: row.ID.String(), Action: "create", After: after, TraceID: tr}); err != nil {
			return fmt.Errorf("submission: audit: %w", err)
		}
		if _, err := outbox.Write(ctx, tx, TopicCreated, map[string]any{"course_id": course.String(), "assignment_id": aid.String(), "submission_id": row.ID.String(), "student_id": actor.String(), "user_id": actor.String(), "version": row.Version, "source": "IN_APP"}); err != nil {
			return fmt.Errorf("submission: outbox: %w", err)
		}
		out = Submitted{ID: row.ID, Version: int(*row.Version), ReceivedAt: row.ReceivedAt, LateDays: grading.LateDays(row.SubmittedAt, a.DueAt), FileName: name}
		return nil
	})
	if err != nil {
		return Submitted{}, err
	}
	_ = s.Redis.Del(ctx, rk).Err()
	return out, nil
}

func (s *Service) drop(ctx context.Context, key string) {
	if err := s.Blob.Delete(context.WithoutCancel(ctx), key); err != nil && s.Log != nil {
		s.Log.WarnContext(ctx, "submission: không xoá được tệp bị từ chối", "error", err.Error())
	}
}

// OrphanCleanup xoá object `courses/*/submissions/{upload_id}/…` không có bài nộp nào trỏ tới và quá OrphanTTL (job `submission.orphan_cleanup`, mỗi giờ). Trả số object đã xoá.
// ponytail: liệt kê cả tiền tố `courses/` rồi lọc `/submissions/`; đổi sang tiền tố riêng nếu số đối tượng lớn.
func (s *Service) OrphanCleanup(ctx context.Context) (int, error) {
	infos, err := s.Blob.List(ctx, "courses/")
	if err != nil {
		return 0, fmt.Errorf("submission: liệt kê: %w", err)
	}
	cut := s.now().Add(-OrphanTTL)
	var keys []string
	for _, o := range infos {
		if strings.Contains(o.Key, "/submissions/") && o.LastModified.Before(cut) {
			keys = append(keys, o.Key)
		}
	}
	if len(keys) == 0 {
		return 0, nil
	}
	used, err := store.New(s.Pool).SubmissionBlobKeysUsed(ctx, keys)
	if err != nil {
		return 0, fmt.Errorf("submission: khoá đang dùng: %w", err)
	}
	inUse := make(map[string]bool, len(used))
	for _, k := range used {
		inUse[k] = true
	}
	n := 0
	for _, k := range keys {
		if inUse[k] {
			continue
		}
		if err := s.Blob.Delete(ctx, k); err != nil {
			return n, fmt.Errorf("submission: xoá %s: %w", k, err)
		}
		n++
	}
	return n, nil
}
