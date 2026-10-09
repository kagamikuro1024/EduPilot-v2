// Package httpx: helper HTTP dùng chung của mọi API — giải mã + kiểm dữ liệu JSON, phân trang con trỏ,
// ETag / If-None-Match và khoá lạc quan `version` (SRS 6.1, 6.4, 6.5, 6.7).
//
// Gói LÁ (chỉ phụ thuộc apierr): `internal/httpapi`, `internal/jobs` và `internal/testroutes` đều dùng được.
// Không đặt trong `internal/httpapi` vì bản dựng `-tags testroutes` có chiều import httpapi → testroutes.
package httpx

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/go-playground/validator/v10"
)

// mediaTypeJSON là kiểu nội dung duy nhất API nhận (SRS 6.1: khác → 415).
const mediaTypeJSON = "application/json"

// WriteJSON ghi một phản hồi JSON (`application/json; charset=utf-8`).
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", mediaTypeJSON+"; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// DecodeJSON đọc thân JSON vào dst rồi kiểm theo tag `validate` (FR-29). Hỏng thì tự ghi lỗi và trả false:
// Content-Type ≠ application/json → 415; JSON hỏng → 400; thân quá lớn → 413;
// sai kiểu / sai tag / trường lạ → 422 VALIDATION_FAILED với details liệt kê MỌI lỗi theo thứ tự trường.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	if e := decodeJSON(r, dst); e != nil {
		apierr.Write(w, r, e)
		return false
	}
	return true
}

func decodeJSON(r *http.Request, dst any) *apierr.Error {
	mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mt != mediaTypeJSON {
		return apierr.New(http.StatusUnsupportedMediaType, apierr.UnsupportedMediaType)
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return apierr.New(http.StatusRequestEntityTooLarge, apierr.PayloadTooLarge).
				WithDetails(map[string]any{"max_bytes": tooLarge.Limit})
		}
		return apierr.New(http.StatusBadRequest, apierr.BadRequest)
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return apierr.New(http.StatusBadRequest, apierr.BadRequest)
	}
	if err := json.Unmarshal(body, dst); err != nil {
		var typeErr *json.UnmarshalTypeError
		if errors.As(err, &typeErr) {
			return apierr.Validation(apierr.FieldError{
				Field: typeErrField(typeErr), Code: "type", Message: "Giá trị không đúng kiểu dữ liệu.",
			})
		}
		return apierr.New(http.StatusBadRequest, apierr.BadRequest)
	}
	fields := append(validateStruct(dst), unknownFields(body, dst)...)
	fields = append(fields, nulFields(body)...)
	if len(fields) > 0 {
		return apierr.Validation(fields...)
	}
	return nil
}

func typeErrField(e *json.UnmarshalTypeError) string {
	if e.Field == "" {
		return "body"
	}
	return e.Field
}

// validateStruct chạy go-playground/validator và đổi lỗi sang `{field, code, message}` theo tên JSON.
func validateStruct(dst any) []apierr.FieldError {
	// ponytail: dựng validator mỗi lần giải mã (vài µs, không có biến toàn cục).
	// Nếu hồ sơ đo cho thấy tốn, đưa một instance vào Deps rồi truyền xuống.
	v := validator.New(validator.WithRequiredStructEnabled())
	v.RegisterTagNameFunc(jsonTagName)
	var invalid validator.ValidationErrors
	if err := v.Struct(dst); err != nil && errors.As(err, &invalid) {
		out := make([]apierr.FieldError, 0, len(invalid))
		for _, fe := range invalid {
			out = append(out, apierr.FieldError{Field: fe.Field(), Code: fe.Tag(), Message: ruleMessage(fe)})
		}
		return out
	}
	return nil
}

func jsonTagName(f reflect.StructField) string {
	name := strings.Split(f.Tag.Get("json"), ",")[0]
	if name == "" || name == "-" {
		return f.Name
	}
	return name
}

func ruleMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "Trường này là bắt buộc."
	case "min", "gte":
		return "Giá trị nhỏ hơn mức cho phép."
	case "max", "lte":
		return "Giá trị lớn hơn mức cho phép."
	case "oneof":
		return "Giá trị không nằm trong danh sách cho phép."
	default:
		return "Giá trị không hợp lệ."
	}
}

// unknownFields liệt kê khoá lạ ở cấp cao nhất của thân, theo đúng thứ tự xuất hiện (03-AC5).
func unknownFields(body []byte, dst any) []apierr.FieldError {
	known := knownFields(reflect.TypeOf(dst))
	if known == nil {
		return nil
	}
	var out []apierr.FieldError
	for _, k := range topLevelKeys(body) {
		if _, ok := known[k]; !ok {
			out = append(out, apierr.FieldError{Field: k, Code: "unknown", Message: "Trường không được hỗ trợ."})
		}
	}
	return out
}

func knownFields(t reflect.Type) map[string]struct{} {
	for t != nil && t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t == nil || t.Kind() != reflect.Struct {
		return nil
	}
	names := make(map[string]struct{}, t.NumField())
	for i := range t.NumField() {
		names[jsonTagName(t.Field(i))] = struct{}{}
	}
	return names
}

func topLevelKeys(body []byte) []string {
	dec := json.NewDecoder(bytes.NewReader(body))
	if t, err := dec.Token(); err != nil || t != json.Delim('{') {
		return nil
	}
	var keys []string
	for dec.More() {
		t, err := dec.Token()
		if err != nil {
			return keys
		}
		k, _ := t.(string)
		keys = append(keys, k)
		var skip json.RawMessage
		if err := dec.Decode(&skip); err != nil {
			return keys
		}
	}
	return keys
}

// nulFields báo mọi chuỗi (giá trị hoặc khoá) chứa NUL: Postgres không lưu được U+0000 trong text / jsonb (SQLSTATE 22021 → 500),
// nên bị chặn ở đây cho MỌI endpoint thay vì ở từng handler (QC US-PE-03 B1). Đường nhanh: thân không có chuỗi `\u0000` thì khỏi duyệt.
// Dấu `\\u0000` (gạch chéo ngược thật + "u0000") khớp đường nhanh nhưng không phải NUL: bước duyệt cây loại ca đó.
func nulFields(body []byte) []apierr.FieldError {
	if !bytes.Contains(body, []byte(`\u0000`)) { // cách DUY NHẤT JSON hợp lệ biểu diễn U+0000
		return nil
	}
	var root any
	if json.Unmarshal(body, &root) != nil {
		return nil
	}
	var out []apierr.FieldError
	var walk func(path string, v any)
	walk = func(path string, v any) {
		switch x := v.(type) {
		case string:
			if strings.ContainsRune(x, 0) {
				out = append(out, apierr.FieldError{Field: path, Code: "invalid_text", Message: "Văn bản không được chứa ký tự NUL (U+0000)."})
			}
		case []any:
			for i, e := range x {
				walk(path+"["+strconv.Itoa(i)+"]", e)
			}
		case map[string]any:
			keys := make([]string, 0, len(x))
			for k := range x {
				keys = append(keys, k)
			}
			slices.Sort(keys)
			for _, k := range keys {
				if strings.ContainsRune(k, 0) {
					out = append(out, apierr.FieldError{Field: path, Code: "invalid_text", Message: "Tên trường không được chứa ký tự NUL (U+0000)."})
					continue
				}
				child := k
				if path != "" {
					child = path + "." + k
				}
				walk(child, x[k])
			}
		}
	}
	walk("", root)
	return out
}
