package httpx

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
)

// etagBodyLen là số ký tự base64url của băm thân giữ lại trong ETag danh sách (SRS 6.5).
const etagBodyLen = 16

// ETagVersion là ETag của tài nguyên có cột `version`: W/"v<n>".
func ETagVersion(version int) string { return `W/"v` + strconv.Itoa(version) + `"` }

// ETagBody là ETag của một danh sách: W/"<16 ký tự đầu base64url của sha256 toàn bộ byte thân>".
func ETagBody(body []byte) string {
	sum := sha256.Sum256(body)
	return `W/"` + base64.RawURLEncoding.EncodeToString(sum[:])[:etagBodyLen] + `"`
}

// MatchIfNoneMatch cho biết client đã có đúng phiên bản này chưa (`*`, một giá trị, hoặc danh sách).
// So sánh yếu: bỏ tiền tố `W/` hai bên (SRS 6.7).
func MatchIfNoneMatch(r *http.Request, etag string) bool {
	h := strings.TrimSpace(r.Header.Get("If-None-Match"))
	if h == "" {
		return false
	}
	if h == "*" {
		return true
	}
	for _, want := range strings.Split(h, ",") {
		if strings.TrimPrefix(strings.TrimSpace(want), "W/") == strings.TrimPrefix(etag, "W/") {
			return true
		}
	}
	return false
}

// WriteJSONETag ghi phản hồi của một GET kèm ETag (SRS 6.5, 6.7): `Cache-Control: private, no-cache`,
// `Vary: Authorization`, và 304 thân rỗng khi `If-None-Match` khớp.
// etag rỗng → băm chính byte thân sắp ghi (dùng cho danh sách).
func WriteJSONETag(w http.ResponseWriter, r *http.Request, v any, etag string) {
	body, err := json.Marshal(v)
	if err != nil {
		apierr.Write(w, r, apierr.New(http.StatusInternalServerError, apierr.Internal))
		return
	}
	if etag == "" {
		etag = ETagBody(body)
	}
	h := w.Header()
	h.Set("ETag", etag)
	h.Set("Cache-Control", "private, no-cache")
	h.Add("Vary", "Authorization")
	if MatchIfNoneMatch(r, etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	h.Set("Content-Type", mediaTypeJSON+"; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

// WantedVersion gộp `version` trong thân và header `If-Match: W/"v<n>"` (SRS 6.3 mục 4, 03-AC14):
// thiếu cả hai, sai định dạng, hoặc hai nguồn khác nhau → 422 VALIDATION_FAILED field "version".
func WantedVersion(r *http.Request, bodyVersion *int) (int, *apierr.Error) {
	if raw := strings.TrimSpace(r.Header.Get("If-Match")); raw != "" {
		header, ok := parseVersionETag(raw)
		if !ok {
			return 0, versionError("Header If-Match phải có dạng W/\"v<số>\".")
		}
		if bodyVersion != nil && header != *bodyVersion {
			return 0, versionError("If-Match và version trong thân không khớp nhau.")
		}
		return header, nil
	}
	if bodyVersion != nil {
		return *bodyVersion, nil
	}
	return 0, versionError("Cần version hiện tại của bản ghi (trường version hoặc header If-Match).")
}

// parseVersionETag đọc `W/"v<n>"` (hoặc `"v<n>"`) thành số.
func parseVersionETag(raw string) (int, bool) {
	v := strings.TrimPrefix(strings.Trim(strings.TrimPrefix(raw, "W/"), `"`), "v")
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		return 0, false
	}
	return n, true
}

func versionError(msg string) *apierr.Error {
	return apierr.Validation(apierr.FieldError{Field: "version", Code: "version", Message: msg})
}

// WriteVersionConflict trả 409 VERSION_CONFLICT khi UPDATE … WHERE id AND version không khớp dòng nào
// nhưng bản ghi vẫn tồn tại: kèm `current_version`, `current` và header ETag hiện tại (FR-33).
func WriteVersionConflict(w http.ResponseWriter, r *http.Request, currentVersion int, current any) {
	w.Header().Set("ETag", ETagVersion(currentVersion))
	apierr.Write(w, r, apierr.New(http.StatusConflict, apierr.VersionConflict).
		WithDetails(map[string]any{"current_version": currentVersion, "current": current}))
}
