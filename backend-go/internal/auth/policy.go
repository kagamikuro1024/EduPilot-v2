package auth

import "unicode/utf8"

// MinPasswordRunes là độ dài tối thiểu (ký tự Unicode) của mật khẩu (SRS FEAT-account-security 4.2.1).
const MinPasswordRunes = 10

// Mã lỗi chính sách mật khẩu (`details[].code` của 422 VALIDATION_FAILED).
const (
	PasswordTooShort = "PASSWORD_TOO_SHORT"
	PasswordTooLong  = "PASSWORD_TOO_LONG"
)

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
