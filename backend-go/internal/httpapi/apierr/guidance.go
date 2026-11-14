package apierr

// Mã lỗi của harness học từ lần giảng viên sửa (SRS FEAT-grading-guidance mục 6). Lời văn tiếng Việt; sinh viên không bao giờ thấy các mã này.
const (
	GuidanceDisabled     = "GUIDANCE_DISABLED"
	ItemNotDraft         = "ITEM_NOT_DRAFT"
	NothingToApply       = "NOTHING_TO_APPLY"
	GuidanceTextRejected = "GUIDANCE_TEXT_REJECTED"
	MixedGuidance        = "MIXED_GUIDANCE"
)

var guidanceMessages = map[string]string{ //nolint:gochecknoglobals // bảng lời văn bất biến
	GuidanceDisabled:     "Tính năng đang tắt.",
	ItemNotDraft:         "Lưu ý này đã được quyết định.",
	NothingToApply:       "Chưa có gì để áp dụng.",
	GuidanceTextRejected: "Chữ lưu ý không hợp lệ.",
	MixedGuidance:        "Có bài chấm theo bản cũ.",
}

func guidanceMessage(code string) (string, bool) {
	m, ok := guidanceMessages[code]
	return m, ok
}
