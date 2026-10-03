// Package llmhttp: ánh xạ lỗi của cổng LLM và dịch vụ cấu hình sang lỗi HTTP chuẩn (SRS FEAT-llm-gateway 6.1).
package llmhttp

import (
	"errors"
	"net/http"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/llmconfig"
)

// FromLLM đổi lỗi `internal/llm` thành lỗi API; ok=false nếu không nhận ra (người gọi trả 500).
func FromLLM(err error) (*apierr.Error, bool) {
	var ov *llm.ErrOverloaded
	var un *llm.ErrUnavailable
	var dm *llm.ErrDimsMismatch
	switch {
	case errors.As(err, &ov):
		return apierr.New(http.StatusServiceUnavailable, apierr.Overloaded).WithRetryAfter(max(1, int(ov.RetryAfter.Seconds()))), true
	case errors.Is(err, llm.ErrNotConfigured):
		return apierr.New(http.StatusServiceUnavailable, apierr.LLMNotConfigured), true
	case errors.As(err, &un):
		return apierr.New(http.StatusServiceUnavailable, apierr.LLMUnavailable).WithDetails(map[string]string{"reason": un.Reason}), true
	case errors.Is(err, llm.ErrAllProvidersFailed):
		return apierr.New(http.StatusServiceUnavailable, apierr.LLMUnavailable).WithDetails(map[string]string{"reason": llm.ReasonAllFailed}), true
	case errors.Is(err, llm.ErrDeadline):
		return apierr.New(http.StatusGatewayTimeout, apierr.DeadlineExceeded), true
	case errors.As(err, &dm):
		return apierr.New(http.StatusUnprocessableEntity, apierr.ModelDimsMismatch).
			WithDetails(map[string]int{"expected": dm.Expected, "actual": dm.Actual}), true
	case errors.Is(err, llm.ErrBadLane):
		return apierr.Validation(apierr.FieldError{Field: "lane", Code: "invalid", Message: "Làn không hợp lệ cho tác vụ này."}), true
	case errors.Is(err, llm.ErrBadRequest):
		return apierr.Validation(apierr.FieldError{Field: "messages", Code: "invalid", Message: "Nội dung yêu cầu không hợp lệ."}), true
	}
	return nil, false
}

// FromConfig đổi lỗi `internal/llmconfig` thành lỗi API.
func FromConfig(err error) (*apierr.Error, bool) {
	var vc *llmconfig.ErrVersionConflict
	var inUse *llmconfig.ErrProviderInUse
	var ri *llmconfig.ErrRouteInvalid
	var dm *llmconfig.ErrDimsMismatch
	var inv *llmconfig.ErrInvalid
	switch {
	case errors.Is(err, llmconfig.ErrForbidden):
		return apierr.New(http.StatusForbidden, apierr.Forbidden).WithDetails(map[string]string{"reason": "role"}), true
	case errors.Is(err, llmconfig.ErrNotFound):
		return apierr.New(http.StatusNotFound, apierr.NotFound), true
	case errors.Is(err, llmconfig.ErrDuplicateName):
		return apierr.New(http.StatusConflict, apierr.Conflict).WithMessage("Đã có nhà cung cấp trùng tên."), true
	case errors.Is(err, llmconfig.ErrLimit):
		return apierr.Validation(apierr.FieldError{Field: "models", Code: "max", Message: "Vượt hạn mức số lượng cho phép."}), true
	case errors.As(err, &vc):
		return apierr.New(http.StatusConflict, apierr.VersionConflict).WithDetails(map[string]int{"current_version": vc.Current}), true
	case errors.As(err, &inUse):
		return apierr.New(http.StatusConflict, apierr.ProviderInUse).WithDetails(map[string][]string{"tasks": inUse.Tasks}), true
	case errors.As(err, &ri):
		return apierr.New(http.StatusUnprocessableEntity, apierr.RouteInvalid).WithDetails(map[string]string{"rule": ri.Rule, "field": ri.Field}), true
	case errors.As(err, &dm):
		return apierr.New(http.StatusUnprocessableEntity, apierr.ModelDimsMismatch).
			WithDetails(map[string]int{"expected": dm.Expected, "actual": dm.Actual}), true
	case errors.As(err, &inv):
		return apierr.Validation(apierr.FieldError{Field: inv.Field, Code: inv.Code, Message: inv.Message}), true
	}
	return nil, false
}
