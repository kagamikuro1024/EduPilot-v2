package apierr

// Mã lỗi quan sát AI (SRS FEAT-p10-eval mục 6). Màn dành cho ADMIN nên được nhắc kỹ thuật vừa phải, vẫn tiếng Việt.
const (
	ReasonRequired = "REASON_REQUIRED"
	PayloadExpired = "PAYLOAD_EXPIRED"
)

var observabilityMessages = map[string]string{ //nolint:gochecknoglobals // bảng lời văn bất biến
	ReasonRequired: "Cần nhập lý do (10–200 ký tự) trước khi mở nội dung.",
	PayloadExpired: "Nội dung đã hết hạn lưu.",
}

func observabilityMessage(code string) (string, bool) {
	m, ok := observabilityMessages[code]
	return m, ok
}
