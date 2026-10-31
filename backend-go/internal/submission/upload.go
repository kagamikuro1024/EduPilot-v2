package submission

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/jobs"
	"github.com/edupilot/backend-go/internal/platform/blob"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/store"
)

// Giới hạn ZIP (SRS 4.4.3): ZIP_MAX_BYTES (200 MiB, ở document.Presign), ZIP_MAX_ENTRIES, ZIP_MAX_UNZIPPED (đếm khi ĐỌC, không tin header), ZIP_SYNC_ENTRIES.
const (
	ZipMaxEntries  = 200
	ZipMaxUnzipped = 200 << 20
	ZipSyncEntries = 5

	// KindZip là loại việc nền xử lý ZIP nhiều mục.
	KindZip = "submission.zip"
	// uploadEpoch: `external_ts` hằng của UploadSource (cùng tệp / tên mục ⇒ cùng khoá khử trùng).
	uploadSourceName = "UPLOAD"
)

var uploadEpoch = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC) //nolint:gochecknoglobals // hằng thời gian

// Rejection là một mục bị từ chối (không làm hỏng cả lô).
type Rejection struct {
	Entry string `json:"entry"`
	Code  string `json:"code"`
}

// UploadOut là kết quả nhập tệp lẻ / ZIP.
type UploadOut struct {
	Created   int         `json:"created"`
	Matched   int         `json:"matched"`
	Unmatched int         `json:"unmatched"`
	Deduped   []string    `json:"deduped"`
	Rejected  []Rejection `json:"rejected"`
}

// ZipPayload là payload việc `submission.zip`.
type ZipPayload struct {
	CourseID     uuid.UUID `json:"course_id"`
	AssignmentID uuid.UUID `json:"assignment_id"`
	Actor        uuid.UUID `json:"actor"`
	BlobKey      string    `json:"blob_key"`
	FileName     string    `json:"file_name"`
}

// memReader cho phép dùng lại Kind.Sniff trên nội dung đã nằm trong bộ nhớ.
type memReader struct{ b []byte }

func (m memReader) GetRange(_ context.Context, _ string, off, n int64) ([]byte, error) {
	if off >= int64(len(m.b)) {
		return nil, nil
	}
	return m.b[off:min(int64(len(m.b)), off+n)], nil
}

// entryKind phân loại tên mục ZIP: skip (bỏ lặng lẽ), reject (kèm mã), hoặc ok.
func entryKind(name string) (skip bool, code string, kind Kind) {
	if strings.HasSuffix(name, "/") || strings.HasPrefix(name, "__MACOSX/") || strings.Contains(name, "/__MACOSX/") {
		return true, "", Kind{}
	}
	base := path.Base(name)
	if strings.HasPrefix(base, ".") {
		return true, "", Kind{}
	}
	if strings.HasPrefix(name, "/") || strings.ContainsAny(name, "\\") || slices.Contains(strings.Split(name, "/"), "..") ||
		strings.ContainsFunc(name, func(r rune) bool { return r < 0x20 || r == 0x7f }) || strings.EqualFold(path.Ext(base), ".zip") {
		return false, apierr.EntryUnsafe, Kind{}
	}
	k, ok := KindOf(strings.ToLower(path.Ext(base)))
	if !ok {
		return false, apierr.EntryTypeNotAllowed, Kind{}
	}
	return false, "", k
}

// UploadComplete: `POST …/assignments/{aid}/submissions/upload {upload_id}` (TA, TEACHER; Idempotency-Key ở middleware). Tệp lẻ → 200 đồng bộ; ZIP ≤ 5 mục hợp lệ → 200;
// nhiều hơn → việc nền `submission.zip` (jobID khác nil → 202).
func (s *Service) UploadComplete(ctx context.Context, actor, course, aid, uploadID uuid.UUID) (UploadOut, *uuid.UUID, error) {
	if s.Blob == nil || s.Redis == nil {
		return UploadOut{}, nil, apierr.New(http.StatusServiceUnavailable, apierr.ServiceUnavailable)
	}
	rk := appredis.Key("upload", uploadID.String())
	tk, err := s.Redis.HGetAll(ctx, rk).Result()
	if err != nil {
		return UploadOut{}, nil, fmt.Errorf("submission: đọc vé tải lên: %w", err)
	}
	if len(tk) == 0 || tk["user_id"] != actor.String() || tk["course_id"] != course.String() || tk["purpose"] != "submission_upload" {
		return UploadOut{}, nil, apierr.New(http.StatusNotFound, apierr.UploadNotFound)
	}
	q := store.New(s.Pool)
	if err := s.uploadGate(ctx, q, course, aid); err != nil {
		return UploadOut{}, nil, err
	}
	key, name := tk["blob_key"], tk["filename"]
	var size int64
	_, _ = fmt.Sscan(tk["size"], &size)
	info, err := s.Blob.Stat(ctx, key)
	switch {
	case errors.Is(err, blob.ErrNotFound):
		return UploadOut{}, nil, apierr.New(http.StatusUnprocessableEntity, apierr.UploadIncomplete)
	case err != nil:
		return UploadOut{}, nil, fmt.Errorf("submission: kiểm tệp: %w", err)
	case info.Size != size:
		s.drop(ctx, key)
		return UploadOut{}, nil, apierr.New(http.StatusUnprocessableEntity, apierr.UploadIncomplete)
	}
	if !strings.EqualFold(path.Ext(name), ".zip") {
		out, err := s.ingestSingle(ctx, actor, course, aid, key, name, size)
		if err == nil {
			_ = s.Redis.Del(ctx, rk).Err()
		}
		return out, nil, err
	}
	// ZIP: đếm mục hợp lệ để chọn đồng bộ / nền
	zr, err := zip.NewReader(rangeAt{ctx, s.Blob, key, size}, size)
	if err != nil {
		s.drop(ctx, key)
		return UploadOut{}, nil, apierr.New(http.StatusUnprocessableEntity, apierr.FileInvalid)
	}
	n := 0
	for _, f := range zr.File {
		if skip, _, _ := entryKind(f.Name); !skip {
			n++
		}
	}
	if n <= ZipSyncEntries {
		out, err := s.ProcessZip(ctx, ZipPayload{CourseID: course, AssignmentID: aid, Actor: actor, BlobKey: key, FileName: name})
		if err == nil {
			_ = s.Redis.Del(ctx, rk).Err()
		}
		return out, nil, err
	}
	if s.Jobs == nil {
		return UploadOut{}, nil, apierr.New(http.StatusServiceUnavailable, apierr.ServiceUnavailable)
	}
	j, err := s.Jobs.Enqueue(ctx, actor, KindZip, ZipPayload{CourseID: course, AssignmentID: aid, Actor: actor, BlobKey: key, FileName: name})
	if err != nil {
		return UploadOut{}, nil, fmt.Errorf("submission: xếp việc ZIP: %w", err)
	}
	_ = s.Redis.Del(ctx, rk).Err()
	return UploadOut{}, &j.ID, nil
}

// uploadGate: lớp còn hoạt động, bài đã công bố (409 ASSIGNMENT_NOT_PUBLISHED), kênh UPLOAD bật (409 CHANNEL_NOT_ALLOWED).
func (s *Service) uploadGate(ctx context.Context, q *store.Queries, course, aid uuid.UUID) error {
	st, err := q.CalCourseStatus(ctx, course)
	if errors.Is(err, pgx.ErrNoRows) {
		return apierr.New(http.StatusNotFound, apierr.NotFound)
	}
	if err != nil {
		return fmt.Errorf("submission: đọc lớp: %w", err)
	}
	if st == store.CourseStatusARCHIVED {
		return apierr.New(http.StatusConflict, apierr.CourseArchived)
	}
	a, err := q.AssignmentGet(ctx, store.AssignmentGetParams{CourseID: course, ID: aid})
	if errors.Is(err, pgx.ErrNoRows) {
		return apierr.New(http.StatusNotFound, apierr.AssignmentNotFound)
	}
	if err != nil {
		return fmt.Errorf("submission: đọc bài: %w", err)
	}
	switch {
	case a.PublishedAt == nil:
		return apierr.New(http.StatusConflict, apierr.AssignmentNotPublished)
	case !slices.Contains(a.Channels, "UPLOAD"):
		return apierr.New(http.StatusConflict, apierr.ChannelNotAllowed)
	}
	return nil
}

func (s *Service) env(ctx context.Context, course uuid.UUID) (Env, error) {
	return LoadEnv(ctx, store.New(s.Pool), course, "UPLOAD")
}

// ingestTx chạy Ingest trong giao dịch riêng của một mục.
func (s *Service) ingestTx(ctx context.Context, env Env, raw Raw, actor uuid.UUID) (Outcome, error) {
	var out Outcome
	err := pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		var err error
		out, err = Ingest(ctx, tx, env, raw, &actor)
		return err
	})
	return out, err
}

func tally(out *UploadOut, o Outcome, label string) {
	switch {
	case o.Deduped:
		out.Deduped = append(out.Deduped, label)
	case o.Matched:
		out.Created++
		out.Matched++
	default:
		out.Created++
		out.Unmatched++
	}
}

// ingestSingle: tệp lẻ — kiểm nội dung (FILE_INVALID), băm khi đọc (khử trùng theo sha256 + tên), khớp theo 4.4.2 với bài của route.
func (s *Service) ingestSingle(ctx context.Context, actor, course, aid uuid.UUID, key, name string, size int64) (UploadOut, error) {
	out := UploadOut{Deduped: []string{}, Rejected: []Rejection{}}
	kind, ok := KindOf(strings.ToLower(path.Ext(name)))
	if !ok {
		s.drop(ctx, key)
		return out, apierr.New(http.StatusUnprocessableEntity, apierr.FileTypeNotAllowed)
	}
	if size <= 0 {
		s.drop(ctx, key)
		return out, apierr.New(http.StatusUnprocessableEntity, apierr.FileEmpty)
	}
	if size > s.maxBytes() {
		s.drop(ctx, key)
		return out, apierr.New(http.StatusUnprocessableEntity, apierr.FileTooLarge)
	}
	if !kind.Sniff(ctx, s.Blob, key, size) {
		s.drop(ctx, key)
		return out, apierr.New(http.StatusUnprocessableEntity, apierr.FileInvalid)
	}
	rc, err := s.Blob.Get(ctx, key)
	if err != nil {
		return out, fmt.Errorf("submission: mở tệp: %w", err)
	}
	h := sha256.New()
	_, err = io.Copy(h, rc)
	_ = rc.Close()
	if err != nil {
		return out, fmt.Errorf("submission: băm tệp: %w", err)
	}
	sum := hex.EncodeToString(h.Sum(nil))
	env, err := s.env(ctx, course)
	if err != nil {
		return out, err
	}
	o, err := s.ingestTx(ctx, env, Raw{Source: uploadSourceName, ExternalRef: sum + ":" + name, ExternalTS: uploadEpoch, SenderVerified: true, FileName: name, BlobKey: key,
		FileMime: kind.Mime, SHA256: sum, FileSize: size, AssignmentID: &aid}, actor)
	if err != nil {
		return out, err
	}
	if o.Deduped {
		s.drop(ctx, key) // cùng tệp đã nhập: không giữ bản thứ hai
	}
	tally(&out, o, name)
	return out, nil
}

// ProcessZip đọc ZIP THẲNG từ object storage (Range GET; không giải nén ra đĩa), mỗi mục hợp lệ → một lần Ingest (giao dịch riêng: mục hỏng không làm hỏng cả lô).
func (s *Service) ProcessZip(ctx context.Context, p ZipPayload) (UploadOut, error) {
	out := UploadOut{Deduped: []string{}, Rejected: []Rejection{}}
	info, err := s.Blob.Stat(ctx, p.BlobKey)
	if err != nil {
		return out, fmt.Errorf("submission: kiểm ZIP: %w", err)
	}
	zr, err := zip.NewReader(rangeAt{ctx, s.Blob, p.BlobKey, info.Size}, info.Size)
	if err != nil {
		return out, apierr.New(http.StatusUnprocessableEntity, apierr.FileInvalid)
	}
	env, err := s.env(ctx, p.CourseID)
	if err != nil {
		return out, err
	}
	var total int64
	count := 0
	for _, f := range zr.File {
		skip, code, kind := entryKind(f.Name)
		if skip {
			continue
		}
		if count++; count > ZipMaxEntries {
			out.Rejected = append(out.Rejected, Rejection{Entry: "*", Code: apierr.ZipLimit})
			break
		}
		if code != "" {
			out.Rejected = append(out.Rejected, Rejection{Entry: f.Name, Code: code})
			continue
		}
		data, over, err := readEntry(f, s.maxBytes(), ZipMaxUnzipped-total)
		if err != nil {
			out.Rejected = append(out.Rejected, Rejection{Entry: f.Name, Code: apierr.FileInvalid})
			continue
		}
		total += int64(len(data))
		switch {
		case over == overTotal:
			out.Rejected = append(out.Rejected, Rejection{Entry: "*", Code: apierr.ZipLimit})
			return out, nil
		case over == overEntry:
			out.Rejected = append(out.Rejected, Rejection{Entry: f.Name, Code: apierr.FileTooLarge})
			continue
		case len(data) == 0:
			out.Rejected = append(out.Rejected, Rejection{Entry: f.Name, Code: apierr.FileEmpty})
			continue
		case !kind.Sniff(ctx, memReader{data}, "", int64(len(data))):
			out.Rejected = append(out.Rejected, Rejection{Entry: f.Name, Code: apierr.FileInvalid})
			continue
		}
		sum := sha256.Sum256(data)
		hexSum := hex.EncodeToString(sum[:])
		safe := SafeName(path.Base(f.Name))
		key := fmt.Sprintf("courses/%s/submissions/%s/%s", p.CourseID, uuid.New(), safe)
		if err := s.Blob.Put(ctx, key, bytes.NewReader(data), int64(len(data)), kind.Mime); err != nil {
			return out, fmt.Errorf("submission: lưu mục ZIP: %w", err)
		}
		aid := p.AssignmentID
		o, err := s.ingestTx(ctx, env, Raw{Source: uploadSourceName, ExternalRef: hexSum + ":" + f.Name, ExternalTS: uploadEpoch, SenderVerified: true, FileName: safe, MatchName: f.Name,
			BlobKey: key, FileMime: kind.Mime, SHA256: hexSum, FileSize: int64(len(data)), AssignmentID: &aid}, p.Actor)
		if err != nil {
			s.drop(ctx, key)
			return out, err
		}
		if o.Deduped {
			s.drop(ctx, key)
		}
		tally(&out, o, f.Name)
	}
	return out, nil
}

const (
	overNone = iota
	overEntry
	overTotal
)

// readEntry đọc một mục, ĐẾM byte khi đọc (không tin header): quá `entryMax` → overEntry; quá `remaining` của cả ZIP → overTotal.
func readEntry(f *zip.File, entryMax, remaining int64) ([]byte, int, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, overNone, fmt.Errorf("mở mục: %w", err)
	}
	defer func() { _ = rc.Close() }()
	limit := min(entryMax, remaining)
	data, err := io.ReadAll(io.LimitReader(rc, limit+1))
	if err != nil {
		return nil, overNone, fmt.Errorf("đọc mục: %w", err)
	}
	if int64(len(data)) > limit {
		if remaining <= entryMax {
			return nil, overTotal, nil
		}
		return nil, overEntry, nil
	}
	return data, overNone, nil
}

// RegisterZip gắn loại việc `submission.zip` vào jobs.Runner.
func (s *Service) RegisterZip(r *jobs.Runner) {
	r.Register(KindZip, func(ctx context.Context, j jobs.JobCtx) (any, error) {
		var p ZipPayload
		if err := json.Unmarshal(j.Payload, &p); err != nil {
			return nil, &jobs.UserError{Code: apierr.FileInvalid, Message: "Yêu cầu nhập ZIP không đọc được."}
		}
		j.Progress(5)
		out, err := s.ProcessZip(ctx, p)
		var ae *apierr.Error
		if errors.As(err, &ae) {
			return nil, &jobs.UserError{Code: ae.Code, Message: ae.Message}
		}
		return out, err
	})
}
