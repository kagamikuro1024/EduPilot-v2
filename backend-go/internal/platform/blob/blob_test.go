package blob_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/platform/blob"
	"github.com/edupilot/backend-go/internal/testutil"
)

// --- tiện ích ----------------------------------------------------------------

// newStore dựng Store trỏ tới container MinIO dùng chung, bucket riêng cho mỗi test.
func newStore(t *testing.T, mut func(*blob.Config)) *blob.Store {
	t.Helper()
	cfg := blob.Config{
		Endpoint:     testutil.MinIOEndpoint(t),
		Bucket:       strings.ToLower(testutil.TestPrefix(t)),
		AccessKey:    testutil.MinIOAccessKey,
		SecretKey:    testutil.MinIOSecretKey,
		EnsureBucket: true,
	}
	if mut != nil {
		mut(&cfg)
	}
	s, err := blob.New(t.Context(), cfg)
	require.NoError(t, err)
	return s
}

// patternReader sinh n byte theo mẫu lặp mà không giữ hết trong RAM (dùng cho 20 MiB).
type patternReader struct {
	left int64
	off  byte
}

func (p *patternReader) Read(b []byte) (int, error) {
	if p.left <= 0 {
		return 0, io.EOF
	}
	n := int64(len(b))
	if n > p.left {
		n = p.left
	}
	for i := range b[:n] {
		b[i] = p.off
		p.off++
	}
	p.left -= n
	return int(n), nil
}

func patternSHA(t *testing.T, n int64) string {
	t.Helper()
	h := sha256.New()
	_, err := io.Copy(h, &patternReader{left: n})
	require.NoError(t, err)
	return hex.EncodeToString(h.Sum(nil))
}

func sha256Of(t *testing.T, r io.Reader) string {
	t.Helper()
	h := sha256.New()
	_, err := io.Copy(h, r)
	require.NoError(t, err)
	return hex.EncodeToString(h.Sum(nil))
}

// --- 02-AC11 -----------------------------------------------------------------

func TestBlob_RoundTrip(t *testing.T) {
	t.Parallel()
	s := newStore(t, nil)
	ctx := t.Context()

	for _, size := range []int64{0, 1, 5 << 20, 20 << 20} {
		key := "ac11/roundtrip/" + strings.ReplaceAll(time.Now().Format("150405.000000000"), ".", "") + ".bin"
		want := patternSHA(t, size)

		require.NoError(t, s.Put(ctx, key, &patternReader{left: size}, size, "application/octet-stream"))

		info, err := s.Stat(ctx, key)
		require.NoError(t, err)
		require.Equal(t, size, info.Size, "size %d", size)
		require.Equal(t, "application/octet-stream", info.ContentType)

		rc, err := s.Get(ctx, key)
		require.NoError(t, err)
		got := sha256Of(t, rc) // stream, không đọc hết vào RAM
		require.NoError(t, rc.Close())
		require.Equal(t, want, got, "nội dung %d byte", size)

		require.NoError(t, s.Delete(ctx, key))

		_, err = s.Get(ctx, key)
		require.ErrorIs(t, err, blob.ErrNotFound)
		_, err = s.Stat(ctx, key)
		require.ErrorIs(t, err, blob.ErrNotFound)
	}
}

func TestBlob_Presign(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	s := newStore(t, nil)
	body := []byte("xin chào presign")
	key := "ac11/presign/get.txt"
	require.NoError(t, s.Put(ctx, key, bytes.NewReader(body), int64(len(body)), "text/plain"))

	// PresignGet: tải được bằng http.Get thường, đúng byte, có Content-Disposition.
	u, err := s.PresignGet(ctx, key, "bài làm.txt")
	require.NoError(t, err)
	resp, err := http.Get(u) //nolint:gosec,noctx // URL do test sinh ra
	require.NoError(t, err)
	got, err := io.ReadAll(resp.Body)
	require.NoError(t, resp.Body.Close())
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, body, got)
	require.Contains(t, resp.Header.Get("Content-Disposition"), "attachment")

	// Hết hạn đúng TTL: TTL 2 s → sau 3 s trả 403.
	short := newStore(t, func(c *blob.Config) { c.Bucket = s.Bucket(); c.GetTTL = 2 * time.Second })
	su, err := short.PresignGet(ctx, key, "")
	require.NoError(t, err)
	r1, err := http.Get(su) //nolint:gosec,noctx
	require.NoError(t, err)
	_, _ = io.Copy(io.Discard, r1.Body)
	require.NoError(t, r1.Body.Close())
	require.Equal(t, http.StatusOK, r1.StatusCode)
	time.Sleep(3 * time.Second)
	r2, err := http.Get(su) //nolint:gosec,noctx
	require.NoError(t, err)
	_, _ = io.Copy(io.Discard, r2.Body)
	require.NoError(t, r2.Body.Close())
	require.Equal(t, http.StatusForbidden, r2.StatusCode)

	// PresignPut: tải lên được rồi Stat thấy đúng kích thước.
	up := []byte("nội dung tải lên qua URL ký sẵn")
	putKey := "ac11/presign/put.txt"
	pu, err := s.PresignPut(ctx, putKey, "text/plain")
	require.NoError(t, err)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, pu, bytes.NewReader(up))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "text/plain")
	pr, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	_, _ = io.Copy(io.Discard, pr.Body)
	require.NoError(t, pr.Body.Close())
	require.Equal(t, http.StatusOK, pr.StatusCode)

	info, err := s.Stat(ctx, putKey)
	require.NoError(t, err)
	require.Equal(t, int64(len(up)), info.Size)
}

func TestBlob_InvalidKey(t *testing.T) {
	t.Parallel()
	// Endpoint là cổng đóng: mọi lời gọi mạng sẽ hỏng ⇒ test chứng minh khoá sai bị chặn TRƯỚC khi gọi mạng.
	s, err := blob.New(t.Context(), blob.Config{
		Endpoint: "127.0.0.1:1", Bucket: "qc-bucket", AccessKey: "ak", SecretKey: "sk",
	})
	require.NoError(t, err)

	bad := map[string]string{
		"rỗng":             "",
		"mở đầu bằng /":    "/a/b.txt",
		"chấm chấm":        "a/../../etc/passwd",
		"chấm đơn":         "a/./b.txt",
		"hai gạch chéo":    "a//b.txt",
		"kết thúc gạch":    "a/b/",
		"ký tự điều khiển": "a/b\n.txt",
		"NUL":              "a/b\x00.txt",
		"backslash":        "a\\b.txt",
		"quá dài":          strings.Repeat("k", blob.MaxKeyLen+1),
	}
	for name, key := range bad {
		t.Run(name, func(t *testing.T) {
			ctx := t.Context()
			done := make(chan struct{})
			go func() {
				defer close(done)
				require.ErrorIs(t, s.Put(ctx, key, strings.NewReader("x"), 1, ""), blob.ErrInvalidKey)
				_, err := s.Get(ctx, key)
				require.ErrorIs(t, err, blob.ErrInvalidKey)
				_, err = s.Stat(ctx, key)
				require.ErrorIs(t, err, blob.ErrInvalidKey)
				require.ErrorIs(t, s.Delete(ctx, key), blob.ErrInvalidKey)
				_, err = s.PresignGet(ctx, key, "")
				require.ErrorIs(t, err, blob.ErrInvalidKey)
				_, err = s.PresignPut(ctx, key, "")
				require.ErrorIs(t, err, blob.ErrInvalidKey)
			}()
			select {
			case <-done:
			case <-time.After(2 * time.Second):
				t.Fatal("khoá sai phải bị chặn tại chỗ, không được gọi mạng")
			}
		})
	}

	// Khoá đúng độ dài tối đa thì hợp lệ (không phải ErrInvalidKey — chỉ hỏng vì mạng).
	err = s.Delete(t.Context(), strings.Repeat("k", blob.MaxKeyLen))
	require.Error(t, err)
	require.NotErrorIs(t, err, blob.ErrInvalidKey)
}

// blockingReader trả dữ liệu tới khi đủ ngưỡng rồi huỷ ctx và treo cho tới khi ctx xong.
type blockingReader struct {
	ctx    context.Context
	cancel context.CancelFunc
	left   int64
	before int64
}

func (b *blockingReader) Read(p []byte) (int, error) {
	if b.before <= 0 {
		b.cancel()
		<-b.ctx.Done()
		return 0, b.ctx.Err()
	}
	n := int64(len(p))
	if n > b.before {
		n = b.before
	}
	if n > b.left {
		n = b.left
	}
	if n <= 0 {
		return 0, io.EOF
	}
	b.before -= n
	b.left -= n
	return int(n), nil
}

func TestBlob_Cancel(t *testing.T) {
	t.Parallel()
	s := newStore(t, nil)
	key := "ac11/cancel/big.bin"

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	const size = 20 << 20
	err := s.Put(ctx, key, &blockingReader{ctx: ctx, cancel: cancel, left: size, before: 1 << 20}, size, "")
	require.Error(t, err)
	require.ErrorIs(t, err, context.Canceled)

	// Không để lại object nửa vời.
	_, err = s.Stat(t.Context(), key)
	require.ErrorIs(t, err, blob.ErrNotFound)
}

// --- 02-AC12 -----------------------------------------------------------------

func TestPresign_PublicHost(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	public := testutil.MinIOEndpoint(t)
	seed := newStore(t, nil)
	body := []byte("URL ký sẵn phải dùng host công khai")
	key := "ac12/public.txt"
	require.NoError(t, seed.Put(ctx, key, bytes.NewReader(body), int64(len(body)), "text/plain"))

	// Gateway chỉ thấy MinIO ở host nội bộ (ở đây là host không nối được), nhưng ký bằng host công khai.
	const internal = "minio-internal-not-reachable:9000"
	s, err := blob.New(ctx, blob.Config{
		Endpoint: internal, PublicEndpoint: public, Bucket: seed.Bucket(),
		AccessKey: testutil.MinIOAccessKey, SecretKey: testutil.MinIOSecretKey,
	})
	require.NoError(t, err)

	gu, err := s.PresignGet(ctx, key, "")
	require.NoError(t, err)
	pu, err := s.PresignPut(ctx, "ac12/public-put.txt", "")
	require.NoError(t, err)
	for _, raw := range []string{gu, pu} {
		u, err := url.Parse(raw)
		require.NoError(t, err)
		require.Equal(t, public, u.Host, "URL phải mang host công khai")
		require.NotContains(t, raw, internal, "host nội bộ không bao giờ lộ ra client")
	}

	// Chữ ký hợp lệ với host công khai: tải xuống và tải lên bằng net/http thường.
	resp, err := http.Get(gu) //nolint:gosec,noctx
	require.NoError(t, err)
	got, err := io.ReadAll(resp.Body)
	require.NoError(t, resp.Body.Close())
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, body, got)

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, pu, bytes.NewReader(body))
	require.NoError(t, err)
	pr, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	_, _ = io.Copy(io.Discard, pr.Body)
	require.NoError(t, pr.Body.Close())
	require.Equal(t, http.StatusOK, pr.StatusCode)

	info, err := seed.Stat(ctx, "ac12/public-put.txt")
	require.NoError(t, err)
	require.Equal(t, int64(len(body)), info.Size)
}

func TestPresign_NoNetwork(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// (a) Endpoint là máy chủ đếm request: ký xong mà số request vẫn bằng 0.
	var hits atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	host := strings.TrimPrefix(srv.URL, "http://")

	s, err := blob.New(ctx, blob.Config{
		Endpoint: host, PublicEndpoint: host, Bucket: "qc-bucket",
		AccessKey: "ak", SecretKey: "sk", Region: "ap-southeast-1",
	})
	require.NoError(t, err)
	gu, err := s.PresignGet(ctx, "a/b.txt", "")
	require.NoError(t, err)
	pu, err := s.PresignPut(ctx, "a/b.txt", "application/pdf")
	require.NoError(t, err)
	require.Equal(t, int64(0), hits.Load(), "ký URL không được gọi mạng")
	require.Contains(t, gu, "X-Amz-Signature=")
	require.Contains(t, gu, "ap-southeast-1", "region lấy từ cấu hình, không hỏi máy chủ")
	require.Contains(t, pu, "X-Amz-Signature=")

	// (b) Cả hai endpoint đều là cổng đóng: vẫn ký được, và nhanh.
	closed, err := blob.New(ctx, blob.Config{
		Endpoint: "127.0.0.1:1", PublicEndpoint: "127.0.0.1:2", Bucket: "qc-bucket",
		AccessKey: "ak", SecretKey: "sk",
	})
	require.NoError(t, err)
	done := make(chan error, 1)
	go func() {
		_, e1 := closed.PresignGet(ctx, "a/b.txt", "báo cáo.pdf")
		_, e2 := closed.PresignPut(ctx, "a/b.txt", "")
		done <- errors.Join(e1, e2)
	}()
	select {
	case e := <-done:
		require.NoError(t, e)
	case <-time.After(2 * time.Second):
		t.Fatal("ký URL phải xong tức thì (không gọi mạng)")
	}
}
