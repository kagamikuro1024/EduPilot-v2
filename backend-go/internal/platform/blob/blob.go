// Package blob: lưu trữ đối tượng trên MinIO/S3 (SRS 4.2 FR-23 — 02-AC11, 02-AC12, 07-AC11).
// Mọi thao tác đi thẳng qua stream, KHÔNG bao giờ ghi đĩa cục bộ. URL ký sẵn dùng BLOB_PUBLIC_ENDPOINT
// + BLOB_REGION nên ký được mà không gọi mạng (client đã biết region ⇒ minio-go bỏ qua bước hỏi vị trí bucket).
package blob

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Lỗi mốc của gói. Người gọi so bằng errors.Is.
var (
	// ErrNotFound: khoá không tồn tại (hoặc bucket không tồn tại).
	ErrNotFound = errors.New("blob: không tìm thấy đối tượng")
	// ErrInvalidKey: khoá không hợp lệ — phát hiện tại chỗ, không gọi mạng.
	ErrInvalidKey = errors.New("blob: khoá đối tượng không hợp lệ")
)

// Mặc định theo SRS 4.2 FR-23 và 8.1.
const (
	DefaultGetTTL = 5 * time.Minute  // TTL URL tải xuống
	DefaultPutTTL = 10 * time.Minute // TTL URL tải lên
	DefaultRegion = "us-east-1"      // BLOB_REGION
	MaxKeyLen     = 512              // độ dài tối đa của khoá, tính theo byte
)

// Config là cấu hình của Store. Người gọi ánh xạ từ các biến BLOB_* (SRS 8.1); gói này
// cố ý KHÔNG phụ thuộc platform/config để dùng lại được ở test và ở worker.
type Config struct {
	Endpoint       string // BLOB_ENDPOINT, dạng host:port (không scheme)
	PublicEndpoint string // BLOB_PUBLIC_ENDPOINT; trống ⇒ = Endpoint
	Bucket         string // BLOB_BUCKET
	AccessKey      string // BLOB_ACCESS_KEY
	SecretKey      string // BLOB_SECRET_KEY
	Region         string // BLOB_REGION; trống ⇒ DefaultRegion
	UseSSL         bool   // BLOB_USE_SSL
	EnsureBucket   bool   // tạo bucket nếu chưa có (APP_ENV ≠ production)
	GetTTL         time.Duration
	PutTTL         time.Duration
}

// Info là siêu dữ liệu của một đối tượng (bản rút gọn của minio.ObjectInfo).
type Info struct {
	Key          string
	Size         int64
	ContentType  string
	ETag         string
	LastModified time.Time
}

// Store là kho đối tượng. Dùng hai client: `cli` nói chuyện với endpoint nội bộ,
// `signer` chỉ để ký URL cho trình duyệt (endpoint công khai) và không bao giờ gọi mạng.
type Store struct {
	cli    *minio.Client
	signer *minio.Client
	bucket string
	getTTL time.Duration
	putTTL time.Duration
}

// New dựng Store. Chỉ chạm mạng khi cfg.EnsureBucket = true.
func New(ctx context.Context, cfg Config) (*Store, error) {
	if cfg.Endpoint == "" || cfg.Bucket == "" || cfg.AccessKey == "" || cfg.SecretKey == "" {
		return nil, errors.New("blob: thiếu BLOB_ENDPOINT / BLOB_BUCKET / BLOB_ACCESS_KEY / BLOB_SECRET_KEY")
	}
	region := cfg.Region
	if region == "" {
		region = DefaultRegion
	}
	opts := func() *minio.Options {
		return &minio.Options{
			Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
			Secure: cfg.UseSSL,
			Region: region, // region cố định ⇒ presign không cần gọi GetBucketLocation
		}
	}
	cli, err := minio.New(cfg.Endpoint, opts())
	if err != nil {
		return nil, fmt.Errorf("blob: endpoint %q không hợp lệ: %w", cfg.Endpoint, err)
	}
	signer := cli
	if cfg.PublicEndpoint != "" && cfg.PublicEndpoint != cfg.Endpoint {
		signer, err = minio.New(cfg.PublicEndpoint, opts())
		if err != nil {
			return nil, fmt.Errorf("blob: public endpoint %q không hợp lệ: %w", cfg.PublicEndpoint, err)
		}
	}
	s := &Store{cli: cli, signer: signer, bucket: cfg.Bucket, getTTL: cfg.GetTTL, putTTL: cfg.PutTTL}
	if s.getTTL <= 0 {
		s.getTTL = DefaultGetTTL
	}
	if s.putTTL <= 0 {
		s.putTTL = DefaultPutTTL
	}
	if cfg.EnsureBucket {
		if err := s.ensureBucket(ctx, region); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (s *Store) ensureBucket(ctx context.Context, region string) error {
	ok, err := s.cli.BucketExists(ctx, s.bucket)
	if err != nil {
		return fmt.Errorf("blob: kiểm tra bucket %q: %w", s.bucket, err)
	}
	if ok {
		return nil
	}
	if err := s.cli.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{Region: region}); err != nil {
		// Nhiều tiến trình cùng khởi động có thể đua nhau tạo bucket.
		switch minio.ToErrorResponse(err).Code {
		case "BucketAlreadyOwnedByYou", "BucketAlreadyExists":
			return nil
		}
		return fmt.Errorf("blob: tạo bucket %q: %w", s.bucket, err)
	}
	return nil
}

// Bucket trả tên bucket đang dùng.
func (s *Store) Bucket() string { return s.bucket }

// Put ghi một đối tượng từ stream. size < 0 nghĩa là chưa biết độ dài.
func (s *Store) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	if err := validateKey(key); err != nil {
		return err
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	_, err := s.cli.PutObject(ctx, s.bucket, key, r, size, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return s.mapErr(ctx, "put", key, err)
	}
	return nil
}

// Get mở stream đọc đối tượng. Người gọi phải Close. Khoá không tồn tại ⇒ ErrNotFound.
func (s *Store) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	if err := validateKey(key); err != nil {
		return nil, err
	}
	o, err := s.cli.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, s.mapErr(ctx, "get", key, err)
	}
	// minio.GetObject là lazy: phải Stat để biết đối tượng có thật trước khi trả stream.
	if _, err := o.Stat(); err != nil {
		_ = o.Close()
		return nil, s.mapErr(ctx, "get", key, err)
	}
	return o, nil
}

// Stat trả siêu dữ liệu của đối tượng.
func (s *Store) Stat(ctx context.Context, key string) (Info, error) {
	if err := validateKey(key); err != nil {
		return Info{}, err
	}
	oi, err := s.cli.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		return Info{}, s.mapErr(ctx, "stat", key, err)
	}
	return Info{
		Key:          key,
		Size:         oi.Size,
		ContentType:  oi.ContentType,
		ETag:         oi.ETag,
		LastModified: oi.LastModified,
	}, nil
}

// Delete xoá đối tượng. Xoá khoá không tồn tại không phải lỗi (S3 vốn idempotent).
func (s *Store) Delete(ctx context.Context, key string) error {
	if err := validateKey(key); err != nil {
		return err
	}
	if err := s.cli.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return s.mapErr(ctx, "delete", key, err)
	}
	return nil
}

// PresignGet trả URL tải xuống ký sẵn (TTL mặc định 5 phút) trên host công khai, không gọi mạng.
// filename khác rỗng ⇒ thêm Content-Disposition: attachment.
func (s *Store) PresignGet(ctx context.Context, key, filename string) (string, error) {
	if err := validateKey(key); err != nil {
		return "", err
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	var params url.Values
	if filename != "" {
		params = url.Values{}
		params.Set("response-content-disposition", `attachment; filename="`+safeFilename(filename)+`"`)
	}
	u, err := s.signer.PresignedGetObject(ctx, s.bucket, key, s.getTTL, params)
	if err != nil {
		return "", fmt.Errorf("blob: ký URL tải xuống: %w", err)
	}
	return u.String(), nil
}

// PresignPut trả URL tải lên ký sẵn (TTL mặc định 10 phút) trên host công khai, không gọi mạng.
// contentType khác rỗng ⇒ URL chỉ dùng được khi client gửi đúng header Content-Type đó.
func (s *Store) PresignPut(ctx context.Context, key, contentType string) (string, error) {
	if err := validateKey(key); err != nil {
		return "", err
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	var (
		u   *url.URL
		err error
	)
	if contentType == "" {
		u, err = s.signer.PresignedPutObject(ctx, s.bucket, key, s.putTTL)
	} else {
		h := http.Header{}
		h.Set("Content-Type", contentType)
		u, err = s.signer.PresignHeader(ctx, http.MethodPut, s.bucket, key, s.putTTL, nil, h)
	}
	if err != nil {
		return "", fmt.Errorf("blob: ký URL tải lên: %w", err)
	}
	return u.String(), nil
}

// mapErr đổi lỗi minio thành lỗi mốc của gói; lỗi do ctx bị huỷ giữ nguyên để errors.Is thấy.
func (s *Store) mapErr(ctx context.Context, op, key string, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return fmt.Errorf("blob: %s %q: %w", op, key, ctxErr)
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("blob: %s %q: %w", op, key, err)
	}
	resp := minio.ToErrorResponse(err)
	switch {
	case resp.Code == "NoSuchKey" || resp.Code == "NoSuchBucket" || resp.StatusCode == http.StatusNotFound:
		return fmt.Errorf("blob: %s %q: %w", op, key, ErrNotFound)
	}
	return fmt.Errorf("blob: %s %q: %w", op, key, err)
}

// validateKey kiểm khoá tại chỗ (không gọi mạng): không rỗng, ≤ MaxKeyLen byte, UTF-8 hợp lệ,
// không mở đầu bằng '/', không có đoạn rỗng / '.' / '..', không ký tự điều khiển hay '\'.
func validateKey(key string) error {
	switch {
	case key == "":
		return fmt.Errorf("%w: rỗng", ErrInvalidKey)
	case len(key) > MaxKeyLen:
		return fmt.Errorf("%w: dài %d byte (tối đa %d)", ErrInvalidKey, len(key), MaxKeyLen)
	case !utf8.ValidString(key):
		return fmt.Errorf("%w: không phải UTF-8", ErrInvalidKey)
	case key[0] == '/':
		return fmt.Errorf("%w: mở đầu bằng '/'", ErrInvalidKey)
	}
	for _, r := range key {
		if r < 0x20 || r == 0x7f || r == '\\' {
			return fmt.Errorf("%w: chứa ký tự không cho phép", ErrInvalidKey)
		}
	}
	for _, seg := range strings.Split(key, "/") {
		if seg == "" || seg == "." || seg == ".." {
			return fmt.Errorf("%w: đoạn đường dẫn %q không hợp lệ", ErrInvalidKey, seg)
		}
	}
	return nil
}

// safeFilename bỏ ký tự có thể phá header Content-Disposition.
func safeFilename(name string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f || r == '"' || r == '\\' {
			return -1
		}
		return r
	}, name)
}
