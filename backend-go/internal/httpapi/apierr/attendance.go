package apierr

// Mã lỗi của điểm danh tự phục vụ (SRS FEAT-attendance-selfcheck mục 6). Lời văn tiếng Việt cho người dùng cuối. `TOKEN_INVALID` / `TOKEN_EXPIRED`
// dùng lại mã có sẵn (status 422, lời văn đặt tại chỗ bằng WithMessage) — xem proposals.md P-108-11.
const (
	SelfCheckinDisabled    = "SELF_CHECKIN_DISABLED"
	WindowAlreadyOpen      = "WINDOW_ALREADY_OPEN"
	WindowClosed           = "WINDOW_CLOSED"
	WindowOpen             = "WINDOW_OPEN"
	NoWindow               = "NO_WINDOW"
	RoundLimit             = "ROUND_LIMIT"
	NotInCourse            = "NOT_IN_COURSE"
	CheckinUnresolved      = "CHECKIN_UNRESOLVED"
	DeviceChallengeInvalid = "DEVICE_CHALLENGE_INVALID"
	AttestationFailed      = "ATTESTATION_FAILED"
	AttestationUnavailable = "ATTESTATION_UNAVAILABLE"
)

var attendanceMessages = map[string]string{ //nolint:gochecknoglobals // bảng lời văn bất biến
	SelfCheckinDisabled:    "Lớp này không điểm danh bằng mã.",
	WindowAlreadyOpen:      "Đang có cửa sổ mở.",
	WindowClosed:           "Đã đóng điểm danh.",
	WindowOpen:             "Còn cửa sổ điểm danh đang mở.",
	NoWindow:               "Buổi này chưa mở cửa sổ điểm danh.",
	RoundLimit:             "Đã mở đủ số vòng điểm danh.",
	NotInCourse:            "Bạn không thuộc lớp này.",
	CheckinUnresolved:      "Còn sinh viên chưa điểm danh.",
	DeviceChallengeInvalid: "Mã thử thách hết hạn. Thử lại.",
	AttestationFailed:      "Không xác minh được thiết bị.",
	AttestationUnavailable: "Chưa kiểm tra được thiết bị. Thử lại sau.",
}

func attendanceMessage(code string) (string, bool) {
	m, ok := attendanceMessages[code]
	return m, ok
}
