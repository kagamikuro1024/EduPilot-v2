package apierr

// Mã lỗi của decision model (SRS FEAT-decision-model 6). Chỉ ADMIN thấy.
const (
	DecisionRealDisabled = "DECISION_REAL_DISABLED"
)

var decisionMessages = map[string]string{ //nolint:gochecknoglobals // bảng lời văn bất biến
	DecisionRealDisabled: "Chưa được phép dùng bản thật.",
}

func decisionMessage(code string) (string, bool) {
	m, ok := decisionMessages[code]
	return m, ok
}
