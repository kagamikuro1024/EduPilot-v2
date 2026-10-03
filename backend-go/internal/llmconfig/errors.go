package llmconfig

import (
	"errors"
	"fmt"
	"strings"
)

// Lỗi nghiệp vụ; handler (US-P1-04) ánh xạ sang mã HTTP của PG/SRS 6.1.
var (
	ErrForbidden     = errors.New("llmconfig: không đủ quyền")
	ErrNotFound      = errors.New("llmconfig: không tìm thấy")
	ErrLimit         = errors.New("llmconfig: vượt hạn mức số lượng")
	ErrDuplicateName = errors.New("llmconfig: trùng tên nhà cung cấp")
	ErrKeyMissing    = errors.New("llmconfig: nhà cung cấp chưa có khoá")
	ErrKeyUnreadable = errors.New("llmconfig: không đọc được khoá của nhà cung cấp")
)

// ErrVersionConflict: sửa với version cũ; Current là version hiện hành (409 VERSION_CONFLICT).
type ErrVersionConflict struct{ Current int }

func (e *ErrVersionConflict) Error() string {
	return fmt.Sprintf("llmconfig: version đã đổi (hiện là %d)", e.Current)
}

// ErrProviderInUse: xoá / bỏ mô hình đang được tuyến dùng (409 PROVIDER_IN_USE).
type ErrProviderInUse struct{ Tasks []string }

func (e *ErrProviderInUse) Error() string {
	return "llmconfig: đang được dùng bởi tác vụ " + strings.Join(e.Tasks, ", ")
}

// ErrInvalid: dữ liệu sai ở một trường (422 VALIDATION_FAILED, details[].field/code).
type ErrInvalid struct{ Field, Code, Message string }

func (e *ErrInvalid) Error() string { return fmt.Sprintf("llmconfig: %s: %s", e.Field, e.Message) }

// Rule của ErrRouteInvalid (SRS 6.1).
const (
	RuleChainEmpty       = "chain_empty"
	RuleChainTooLong     = "chain_too_long"
	RuleKindMismatch     = "kind_mismatch"
	RuleEmbeddingSingle  = "embedding_single"
	RuleProviderDisabled = "provider_disabled"
	RuleDuplicateModel   = "duplicate_model"
	RuleParamsOutOfRange = "params_out_of_range"
)

// ErrRouteInvalid: vi phạm quy tắc tuyến (422 ROUTE_INVALID).
type ErrRouteInvalid struct{ Rule, Field string }

func (e *ErrRouteInvalid) Error() string {
	return "llmconfig: tuyến không hợp lệ: " + e.Rule + " (" + e.Field + ")"
}

// ErrDimsMismatch: mô hình nhúng khác 1536 chiều (422 MODEL_DIMS_MISMATCH).
type ErrDimsMismatch struct{ Expected, Actual int }

func (e *ErrDimsMismatch) Error() string {
	return fmt.Sprintf("llmconfig: mô hình nhúng %d chiều, cần %d", e.Actual, e.Expected)
}
