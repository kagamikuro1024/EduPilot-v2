package apierr_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
)

// DEF-R2-02 (SRS 3.4): chỗ ghi lỗi chung — 500 INTERNAL do ctx của request hết hạn phải là 504 DEADLINE_EXCEEDED; client đã đi thì không ghi 500.
func TestWriteInternalDuringDeadline(t *testing.T) {
	expired, cancelExpired := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancelExpired()
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	live, cancelLive := context.WithTimeout(context.Background(), time.Minute)
	defer cancelLive()

	for _, tc := range []struct {
		name   string
		ctx    context.Context
		in     *apierr.Error
		status int
		code   string
	}{
		{"hết hạn → 504", expired, apierr.New(http.StatusInternalServerError, apierr.Internal), http.StatusGatewayTimeout, apierr.DeadlineExceeded},
		{"client đã đi → 499 không thân", canceled, apierr.New(http.StatusInternalServerError, apierr.Internal), apierr.StatusClientClosed, ""},
		{"ctx còn hạn → 500 giữ nguyên", live, apierr.New(http.StatusInternalServerError, apierr.Internal), http.StatusInternalServerError, apierr.Internal},
		{"lỗi khác 500 không bị đổi dù ctx hết hạn", expired, apierr.New(http.StatusConflict, apierr.Conflict), http.StatusConflict, apierr.Conflict},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			apierr.Write(rec, httptest.NewRequest(http.MethodGet, "/x", nil).WithContext(tc.ctx), tc.in)
			if rec.Code != tc.status {
				t.Fatalf("status %d, cần %d (%s)", rec.Code, tc.status, rec.Body.String())
			}
			if tc.code == "" {
				if rec.Body.Len() != 0 {
					t.Fatalf("không được có thân: %s", rec.Body.String())
				}
				return
			}
			var b struct {
				Code string `json:"code"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &b); err != nil || b.Code != tc.code {
				t.Fatalf("code %q (%v), cần %q", b.Code, err, tc.code)
			}
		})
	}
}
