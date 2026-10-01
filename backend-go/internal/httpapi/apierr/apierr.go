// Package apierr: định dạng lỗi HTTP thống nhất {code, message, details?, retry_after?, trace_id} (SRS 6.1).
// Mọi middleware/handler (httpapi, auth, sse, jobs, testroutes) trả lỗi qua gói này — không tự viết JSON lỗi.
package apierr

import (
	"encoding/json"
	"net/http"
	"strconv"

	"go.opentelemetry.io/otel/trace"
)

// Mã lỗi (cột `code` ở SRS 6.1). Chuỗi này là khoá để frontend dịch lời văn.
const (
	BadRequest           = "BAD_REQUEST"
	Unauthenticated      = "UNAUTHENTICATED"
	TokenExpired         = "TOKEN_EXPIRED"
	TokenInvalid         = "TOKEN_INVALID"
	Forbidden            = "FORBIDDEN"
	NotFound             = "NOT_FOUND"
	MethodNotAllowed     = "METHOD_NOT_ALLOWED"
	Conflict             = "CONFLICT"
	VersionConflict      = "VERSION_CONFLICT"
	IdempotencyInProg    = "IDEMPOTENCY_IN_PROGRESS"
	PayloadTooLarge      = "PAYLOAD_TOO_LARGE"
	UnsupportedMediaType = "UNSUPPORTED_MEDIA_TYPE"
	ValidationFailed     = "VALIDATION_FAILED"
	InvalidCursor        = "INVALID_CURSOR"
	IdempotencyReused    = "IDEMPOTENCY_KEY_REUSED"
	IdempotencyRequired  = "IDEMPOTENCY_KEY_REQUIRED"
	RateLimited          = "RATE_LIMITED"
	SSELimitReached      = "SSE_LIMIT_REACHED"
	Internal             = "INTERNAL"
	ServiceUnavailable   = "SERVICE_UNAVAILABLE"
	NotReady             = "NOT_READY"
	DeadlineExceeded     = "DEADLINE_EXCEEDED"
)

// Error là một lỗi API. Status + Code + Message bắt buộc; Details/RetryAfter tuỳ chọn.
type Error struct {
	Status     int
	Code       string
	Message    string
	Details    any
	RetryAfter int // giây; > 0 thì đặt cả header Retry-After và trường retry_after
}

// Error cho phép dùng *Error như error thường.
func (e *Error) Error() string { return e.Code + ": " + e.Message }

// FieldError là một lỗi validation (`details[]` của 422 VALIDATION_FAILED).
type FieldError struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// New dựng lỗi với message mặc định (tiếng Việt) của code.
func New(status int, code string) *Error {
	return &Error{Status: status, Code: code, Message: DefaultMessage(code)}
}

// Validation dựng 422 VALIDATION_FAILED với danh sách lỗi trường.
func Validation(fields ...FieldError) *Error {
	e := New(http.StatusUnprocessableEntity, ValidationFailed)
	e.Details = fields
	return e
}

// WithDetails trả bản sao có details.
func (e *Error) WithDetails(d any) *Error { c := *e; c.Details = d; return &c }

// WithRetryAfter trả bản sao có retry_after (giây).
func (e *Error) WithRetryAfter(sec int) *Error { c := *e; c.RetryAfter = sec; return &c }

// WithMessage trả bản sao có message khác (vẫn phải tiếng Việt, không lộ nội bộ).
func (e *Error) WithMessage(m string) *Error { c := *e; c.Message = m; return &c }

type body struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	TraceID    string `json:"trace_id"`
	Details    any    `json:"details,omitempty"`
	RetryAfter int    `json:"retry_after,omitempty"`
}

// Write ghi lỗi dạng JSON. trace_id lấy từ span trong ctx của request (otelhttp tạo ở middleware đầu tiên).
func Write(w http.ResponseWriter, r *http.Request, e *Error) {
	h := w.Header()
	h.Set("Content-Type", "application/json; charset=utf-8")
	if e.RetryAfter > 0 {
		h.Set("Retry-After", strconv.Itoa(e.RetryAfter))
	}
	if e.Status == http.StatusMethodNotAllowed && h.Get("Allow") == "" {
		h.Set("Allow", http.MethodGet)
	}
	w.WriteHeader(e.Status)
	_ = json.NewEncoder(w).Encode(body{
		Code: e.Code, Message: e.Message, TraceID: trace.SpanContextFromContext(r.Context()).TraceID().String(),
		Details: e.Details, RetryAfter: e.RetryAfter,
	})
}

// DefaultMessage là lời văn tiếng Việt mặc định của từng mã; không lộ chi tiết nội bộ.
func DefaultMessage(code string) string {
	switch code {
	case BadRequest:
		return "Yêu cầu không đọc được."
	case Unauthenticated:
		return "Bạn cần đăng nhập để tiếp tục."
	case TokenExpired:
		return "Phiên đăng nhập đã hết hạn. Vui lòng đăng nhập lại."
	case TokenInvalid:
		return "Phiên đăng nhập không hợp lệ."
	case Forbidden:
		return "Bạn không có quyền thực hiện thao tác này."
	case NotFound:
		return "Không tìm thấy."
	case MethodNotAllowed:
		return "Phương thức không được hỗ trợ."
	case Conflict:
		return "Xung đột trạng thái."
	case VersionConflict:
		return "Dữ liệu đã được người khác thay đổi. Hãy tải lại rồi thử lại."
	case IdempotencyInProg:
		return "Yêu cầu giống hệt đang được xử lý. Hãy thử lại sau giây lát."
	case PayloadTooLarge:
		return "Nội dung gửi lên quá lớn."
	case UnsupportedMediaType:
		return "Định dạng nội dung không được hỗ trợ."
	case ValidationFailed:
		return "Dữ liệu chưa hợp lệ."
	case InvalidCursor:
		return "Con trỏ phân trang không hợp lệ."
	case IdempotencyReused:
		return "Khoá Idempotency-Key đã dùng cho yêu cầu khác."
	case IdempotencyRequired:
		return "Thiếu tiêu đề Idempotency-Key."
	case RateLimited:
		return "Bạn thao tác quá nhanh. Hãy thử lại sau giây lát."
	case SSELimitReached:
		return "Đã đạt số kết nối thời gian thực tối đa."
	case ServiceUnavailable:
		return "Dịch vụ tạm thời không khả dụng. Hãy thử lại sau."
	case NotReady:
		return "Hệ thống chưa sẵn sàng."
	case DeadlineExceeded:
		return "Xử lý quá thời hạn."
	default:
		return "Đã xảy ra lỗi. Hãy thử lại sau."
	}
}

// ByStatus trả lỗi mặc định của một status (dùng cho 404/405 của router và `GET /_test/error/{status}`).
// Status ngoài bảng → nil.
func ByStatus(status int) *Error {
	codes := map[int]string{
		400: BadRequest, 401: Unauthenticated, 403: Forbidden, 404: NotFound, 405: MethodNotAllowed, 409: Conflict,
		413: PayloadTooLarge, 415: UnsupportedMediaType, 422: ValidationFailed, 429: RateLimited, 500: Internal,
		503: ServiceUnavailable, 504: DeadlineExceeded,
	}
	c, ok := codes[status]
	if !ok {
		return nil
	}
	return New(status, c)
}
