package apierr

// Mã lỗi của P7 — bài tập, nộp bài, chấm, phúc khảo, tích hợp (SRS FEAT-assignments mục 6). Lời văn tiếng Việt thân thiện, không lộ tên bảng / SQL.
const (
	TypeNotSupported        = "TYPE_NOT_SUPPORTED"
	RubricInvalid           = "RUBRIC_INVALID"
	RubricLocked            = "RUBRIC_LOCKED"
	RubricRequired          = "RUBRIC_REQUIRED"
	ReferenceAnswerRequired = "REFERENCE_ANSWER_REQUIRED"
	RubricSuggestInvalid    = "RUBRIC_SUGGEST_INVALID"
	HasSubmissions          = "HAS_SUBMISSIONS"
	LinkedToGradebook       = "LINKED_TO_GRADEBOOK"
	AssignmentNotFound      = "ASSIGNMENT_NOT_FOUND"
	ChannelNotAllowed       = "CHANNEL_NOT_ALLOWED"
	SubmissionClosed        = "SUBMISSION_CLOSED"
	AlreadyReviewed         = "ALREADY_REVIEWED"
	SubmissionLocked        = "SUBMISSION_LOCKED"
	LateRuleLocked          = "LATE_RULE_LOCKED"
	LateRecomputed          = "LATE_RECOMPUTED"
	FileEmpty               = "FILE_EMPTY"
	FileContentMismatch     = "FILE_CONTENT_MISMATCH"
	Superseded              = "SUPERSEDED"
	ScoreStep               = "SCORE_STEP"
	ScoresRequired          = "SCORES_REQUIRED"
	ResultPublished         = "RESULT_PUBLISHED"
	NotReviewed             = "NOT_REVIEWED"
	AlreadyPublished        = "ALREADY_PUBLISHED"
	AssignmentScaleMismatch = "ASSIGNMENT_SCALE_MISMATCH"
	AssignmentAlreadyLinked = "ASSIGNMENT_ALREADY_LINKED"
	AppealAlreadyFiled      = "APPEAL_ALREADY_FILED"
	AppealAlreadyDecided    = "APPEAL_ALREADY_DECIDED"
	AppealDisabled          = "APPEAL_DISABLED"
	CriterionUnknown        = "CRITERION_UNKNOWN"
	ReasonLength            = "REASON_LENGTH"
	ResponseRequired        = "RESPONSE_REQUIRED"
	NoChange                = "NO_CHANGE"
	TeamsAuthFailed         = "TEAMS_AUTH_FAILED"
	TeamsConsentRequired    = "TEAMS_CONSENT_REQUIRED"
	TeamsResponseInvalid    = "TEAMS_RESPONSE_INVALID"
	FileDownloadFailed      = "FILE_DOWNLOAD_FAILED"
)

var assignmentMessages = map[string]string{ //nolint:gochecknoglobals // bảng lời văn bất biến
	TypeNotSupported:        "Loại bài tập này chưa được hỗ trợ.",
	RubricInvalid:           "Tổng điểm các tiêu chí phải bằng điểm tối đa của bài.",
	RubricLocked:            "Bài đã có kết quả chấm, không đổi tiêu chí nữa.",
	RubricRequired:          "Cần có tiêu chí chấm trước khi công bố bài.",
	ReferenceAnswerRequired: "Cần nhập đáp án chuẩn (ít nhất 50 ký tự) để gợi ý tiêu chí.",
	RubricSuggestInvalid:    "Không gợi ý được tiêu chí. Hãy nhập tay.",
	HasSubmissions:          "Bài tập đã có bài nộp, không xoá được.",
	LinkedToGradebook:       "Bài tập đang được tính vào công thức điểm, không xoá được.",
	AssignmentNotFound:      "Không tìm thấy bài tập.",
	ChannelNotAllowed:       "Bài này không nhận bài qua kênh đó.",
	SubmissionClosed:        "Đã hết hạn nộp bài.",
	AlreadyReviewed:         "Bài này đã được duyệt.",
	SubmissionLocked:        "Bài của bạn đang được xử lý, chưa nộp lại được.",
	LateRuleLocked:          "Đã có điểm công bố, không đổi hạn hoặc quy tắc nộp muộn.",
	LateRecomputed:          "Khoản trừ nộp muộn đã được tính lại theo hạn mới.",
	FileEmpty:               "Tệp trống.",
	FileContentMismatch:     "Nội dung tệp không đúng định dạng.",
	Superseded:              "Bài đã có phiên bản nộp mới hơn.",
	ScoreStep:               "Điểm phải là bội của 0,25.",
	ScoresRequired:          "Cần nhập điểm cho mọi tiêu chí.",
	ResultPublished:         "Kết quả đã công bố, không sửa được.",
	NotReviewed:             "Bài chưa được duyệt.",
	AlreadyPublished:        "Đã công bố rồi.",
	AssignmentScaleMismatch: "Điểm tối đa của cột phải bằng điểm tối đa của bài tập.",
	AssignmentAlreadyLinked: "Bài tập này đã được gắn vào một cột khác.",
	AppealAlreadyFiled:      "Bạn đã gửi phúc khảo cho bài này.",
	AppealAlreadyDecided:    "Phúc khảo này đã có quyết định.",
	AppealDisabled:          "Bài này không nhận phúc khảo.",
	CriterionUnknown:        "Tiêu chí không có trong bài.",
	ReasonLength:            "Lý do cần từ 10 đến 1.000 ký tự.",
	ResponseRequired:        "Cần nhập phản hồi cho sinh viên.",
	NoChange:                "Không có thay đổi nào.",
	TeamsAuthFailed:         "Không đăng nhập được vào Teams.",
	TeamsConsentRequired:    "Cần quản trị viên của trường cấp quyền cho ứng dụng.",
	TeamsResponseInvalid:    "Teams trả dữ liệu không đúng.",
	FileDownloadFailed:      "Không tải được tệp.",
}

// assignmentMessage trả lời văn của mã P7 (ok = false nếu không phải mã P7).
func assignmentMessage(code string) (string, bool) {
	m, ok := assignmentMessages[code]
	return m, ok
}
