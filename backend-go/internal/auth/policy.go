package auth

import "unicode/utf8"

// MinPasswordRunes là độ dài tối thiểu (ký tự Unicode) của mật khẩu (SRS FEAT-account-security 4.2.1).
const MinPasswordRunes = 10

// Mã lỗi chính sách mật khẩu (`details[].code` của 422 VALIDATION_FAILED).
const (
	PasswordTooShort = "PASSWORD_TOO_SHORT"
	PasswordTooLong  = "PASSWORD_TOO_LONG"
	// PasswordSameAsOld chỉ có ở đổi mật khẩu (ChangePassword); không nằm trong ValidatePasswordPolicy.
	PasswordSameAsOld = "PASSWORD_SAME_AS_OLD"
)

// PasswordMessage là lời nhắn tiếng Việt cho mã lỗi chính sách.
func PasswordMessage(code string) string {
	switch code {
	case PasswordTooShort:
		return "Mật khẩu cần ít nhất 10 ký tự và không quá 72 byte."
	case PasswordTooLong:
		return "Mật khẩu dài tối đa 72 byte."
	case PasswordSameAsOld:
		return "Mật khẩu mới phải khác mật khẩu hiện tại."
	}
	return "Mật khẩu chưa đủ an toàn."
}

// ValidatePasswordPolicy là MỘT hàm cho register / reset-password / accept-invite / đổi mật khẩu. Trả mã lỗi đầu tiên hoặc "".
// ponytail: US-P2-03 mới có độ dài; US-P2-05 thêm danh sách mật khẩu phổ biến, tên email, chuỗi lặp — vẫn trong hàm này.
func ValidatePasswordPolicy(password, _ string) string {
	switch {
	case utf8.RuneCountInString(password) < MinPasswordRunes:
		return PasswordTooShort
	case len(password) > MaxPasswordBytes:
		return PasswordTooLong
	}
	return ""
}
