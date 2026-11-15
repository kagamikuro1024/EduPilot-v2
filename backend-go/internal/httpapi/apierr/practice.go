package apierr

// Mã lỗi của P9 — luyện đề, thi thử, nhập Form (SRS FEAT-practice mục 3, 6). Lời văn tiếng Việt, sinh viên không thấy từ kỹ thuật.
const (
	NotEnoughQuestions   = "NOT_ENOUGH_QUESTIONS"
	TabTakenOver         = "TAB_TAKEN_OVER"
	AnswerLocked         = "ANSWER_LOCKED"
	InvalidOptionID      = "INVALID_OPTION_ID"
	InvalidQuestion      = "INVALID_QUESTION"
	NotAnswered          = "NOT_ANSWERED"
	NotSubmitted         = "NOT_SUBMITTED"
	WrongDocumentType    = "WRONG_DOCUMENT_TYPE"
	CitationRequired     = "CITATION_REQUIRED"
	PreviewRequired      = "PREVIEW_REQUIRED"
	TooLong              = "TOO_LONG"
	TypeNotSupportedExam = "TYPE_NOT_SUPPORTED_IN_EXAM"
)

var practiceMessages = map[string]string{ //nolint:gochecknoglobals // bảng lời văn bất biến
	NotEnoughQuestions:   "Chưa đủ câu để luyện chủ đề này.",
	TabTakenOver:         "Bài này đang được làm ở tab khác.",
	AnswerLocked:         "Câu này đã trả lời, không đổi được.",
	InvalidOptionID:      "Đáp án chọn không thuộc câu hỏi.",
	InvalidQuestion:      "Câu hỏi không hợp lệ.",
	NotAnswered:          "Hãy trả lời câu này trước.",
	NotSubmitted:         "Nộp bài xong mới xem được giải thích.",
	WrongDocumentType:    "Tài liệu này không đúng loại cần dùng.",
	CitationRequired:     "Câu cần có nguồn trích dẫn trước khi duyệt.",
	PreviewRequired:      "Cần xem trước kết quả trước khi xác nhận.",
	TooLong:              "Nội dung quá dài.",
	TypeNotSupportedExam: "Loại câu này chỉ dùng để luyện đề.",
}

func practiceMessage(code string) (string, bool) {
	m, ok := practiceMessages[code]
	return m, ok
}
