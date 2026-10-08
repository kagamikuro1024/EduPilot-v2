package httpx_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/edupilot/backend-go/internal/httpapi/httpx"
)

type body struct {
	Title string   `json:"title" validate:"required"`
	Items []string `json:"items"`
	Nest  struct {
		Note string `json:"note"`
	} `json:"nest"`
}

func decode(raw string) (int, string) {
	r := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(raw))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	var b body
	if httpx.DecodeJSON(w, r, &b) {
		return http.StatusOK, ""
	}
	return w.Code, w.Body.String()
}

// QC US-PE-03 B1: NUL (U+0000) làm Postgres lỗi 22021 → 500; chặn chung ở bộ đọc JSON thành 422 VALIDATION_FAILED, nêu đường dẫn trường.
func TestDecodeJSONRejectsNUL(t *testing.T) {
	cases := []struct {
		name, raw, field string
		want             int
	}{
		{"chuỗi cấp cao", `{"title":"a\u0000b"}`, `"field":"title"`, 422},
		{"trong mảng", `{"title":"t","items":["ok","x\u0000"]}`, `"field":"items[1]"`, 422},
		{"trong đối tượng lồng", `{"title":"t","nest":{"note":"\u0000"}}`, `"field":"nest.note"`, 422},
		{"gạch chéo ngược thật rồi u0000 không phải NUL", `{"title":"a\\u0000b"}`, "", 200},
		{"văn bản thường và ký tự điều khiển khác", `{"title":"a\u0001\n\tb"}`, "", 200},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			code, out := decode(c.raw)
			if code != c.want {
				t.Fatalf("status %d, muốn %d: %s", code, c.want, out)
			}
			if c.want == 422 && (!strings.Contains(out, "VALIDATION_FAILED") || !strings.Contains(out, c.field) || !strings.Contains(out, "invalid_text")) {
				t.Fatalf("thiếu mã/trường: %s", out)
			}
		})
	}
}
