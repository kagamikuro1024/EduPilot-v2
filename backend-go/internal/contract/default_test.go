//go:build !testroutes

package contract

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/google/uuid"
)

// AC3 (không tag): mọi đường dẫn của openapi.test.yaml phải trả 404 ở binary mặc định.
func TestDefaultBuild_TestPathsAre404(t *testing.T) {
	r := getRig(t)
	_, test := loadBoth(t)
	tok := r.token(t, uuid.NewString(), auth.RoleAdmin)
	n := 0
	for _, o := range test.Operations() {
		req, _ := http.NewRequestWithContext(context.Background(), o.Method, r.srv.URL+samplePath(o.Path), strings.NewReader(`{}`))
		req.Header.Set("Authorization", "Bearer "+tok)
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s: %v", o.Key(), err)
		}
		_ = resp.Body.Close()
		n++
		if resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("%s ở bản dựng mặc định → %d (cần 404)", o.Key(), resp.StatusCode)
		}
	}
	if n < 15 {
		t.Errorf("chỉ thử %d thao tác thử (cần 15)", n)
	}
}
