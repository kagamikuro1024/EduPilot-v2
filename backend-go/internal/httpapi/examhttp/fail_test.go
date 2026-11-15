package examhttp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// DEF-R2-02 (SRS 3.4): lỗi i/o do ctx của request hết hạn qua nhánh `default` của h.fail phải ra 504, không phải 500 INTERNAL.
func TestFailDeadlineIs504(t *testing.T) {
	h := &Handler{Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	rec := httptest.NewRecorder()
	h.fail(rec, httptest.NewRequest(http.MethodGet, "/x", nil).WithContext(ctx), fmt.Errorf("đọc câu hỏi: %w", errors.Join(context.DeadlineExceeded)))
	if rec.Code != http.StatusGatewayTimeout {
		t.Fatalf("status %d, cần 504: %s", rec.Code, rec.Body.String())
	}
}
