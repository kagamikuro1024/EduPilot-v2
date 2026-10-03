package auth

import (
	_ "embed" // common_passwords.txt
	"strings"
	"unicode"
	"unicode/utf8"
)

// MinPasswordRunes là độ dài tối thiểu (ký tự Unicode) của mật khẩu (SRS FEAT-account-security 4.2.1).
const MinPasswordRunes = 10

// Mã lỗi chính sách mật khẩu (`details[].code` của 422 VALIDATION_FAILED).
const (
	PasswordTooShort       = "PASSWORD_TOO_SHORT"
	PasswordTooLong        = "PASSWORD_TOO_LONG"
	PasswordCommon         = "PASSWORD_COMMON"
	PasswordContainsEmail  = "PASSWORD_CONTAINS_EMAIL"
	PasswordLowEntropy     = "PASSWORD_LOW_ENTROPY"
	PasswordSameAsOld      = "PASSWORD_SAME_AS_OLD" // chỉ ở đổi mật khẩu (ChangePassword); không nằm trong ValidatePasswordPolicy
	minEmailLocalForPolicy = 4
)

// PasswordMessage là lời nhắn tiếng Việt cho mã lỗi chính sách. Không có từ kỹ thuật; không nhắc lại mật khẩu.
func PasswordMessage(code string) string {
	switch code {
	case PasswordTooShort:
		return "Mật khẩu cần ít nhất 10 ký tự và không quá 72 byte."
	case PasswordTooLong:
		return "Mật khẩu dài tối đa 72 byte."
	case PasswordCommon:
		return "Mật khẩu này quá phổ biến. Hãy chọn mật khẩu khác."
	case PasswordContainsEmail:
		return "Mật khẩu không nên chứa phần đầu của email."
	case PasswordLowEntropy:
		return "Mật khẩu quá dễ đoán (ký tự lặp hoặc dãy liên tiếp)."
	case PasswordSameAsOld:
		return "Mật khẩu mới phải khác mật khẩu hiện tại."
	}
	return "Mật khẩu chưa đủ an toàn."
}

// commonPasswordsRaw: mỗi dòng MỘT mật khẩu phổ biến đã `Fold` (chữ thường, không dấu), dài ≥ 10 ký tự, không trùng; kiểm bởi
// TestCommonPasswordsFileWellFormed. Tra bằng strings.Contains trên chính chuỗi nhúng nên không cần biến trạng thái.
//
//go:embed common_passwords.txt
var commonPasswordsRaw string

func isCommon(folded string) bool {
	if folded == "" || strings.ContainsRune(folded, '\n') {
		return false
	}
	return strings.HasPrefix(commonPasswordsRaw, folded+"\n") || strings.Contains(commonPasswordsRaw, "\n"+folded+"\n")
}

// Chữ cái tiếng Việt có dấu (chữ thường) theo chữ gốc.
const (
	viA = "àáạảãâầấậẩẫăằắặẳẵ"
	viE = "èéẹẻẽêềếệểễ"
	viI = "ìíịỉĩ"
	viO = "òóọỏõôồốộổỗơờớợởỡ"
	viU = "ùúụủũưừứựửữ"
	viY = "ỳýỵỷỹ"
)

// stripVietnamese đưa chữ cái tiếng Việt thường về chữ gốc; `đ` → `d`.
func stripVietnamese(r rune) rune {
	switch {
	case r == 'đ':
		return 'd'
	case r < 0x00C0:
		return r // ASCII và Latin-1 thấp: không có chữ Việt cần đổi
	case strings.ContainsRune(viA, r):
		return 'a'
	case strings.ContainsRune(viE, r):
		return 'e'
	case strings.ContainsRune(viI, r):
		return 'i'
	case strings.ContainsRune(viO, r):
		return 'o'
	case strings.ContainsRune(viU, r):
		return 'u'
	case strings.ContainsRune(viY, r):
		return 'y'
	}
	return r
}

// Fold: chữ thường, bỏ dấu tiếng Việt (cả dạng dựng sẵn lẫn dấu kết hợp U+0300–U+036F), `đ`/`Đ` → `d`. GIỮ NGUYÊN khoảng trắng
// và mọi ký tự khác; KHÔNG ánh xạ leetspeak (SRS 4.2.1).
func Fold(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r >= 0x0300 && r <= 0x036F {
			continue // dấu kết hợp (chuỗi gõ ở dạng NFD)
		}
		r = unicode.ToLower(r)
		b.WriteRune(stripVietnamese(r))
	}
	return b.String()
}

// ValidatePasswordPolicy là MỘT hàm cho register / reset-password / accept-invite / đổi mật khẩu. Trả mã lỗi đầu tiên hoặc "".
// Thứ tự: quá ngắn, quá dài, phổ biến, chứa phần đầu email, dễ đoán. Không bắt buộc chữ hoa / số / ký hiệu.
func ValidatePasswordPolicy(password, email string) string {
	switch {
	case utf8.RuneCountInString(password) < MinPasswordRunes:
		return PasswordTooShort
	case len(password) > MaxPasswordBytes:
		return PasswordTooLong
	}
	f := Fold(password)
	if isCommon(f) {
		return PasswordCommon
	}
	if local, _, ok := strings.Cut(strings.ToLower(strings.TrimSpace(email)), "@"); ok && utf8.RuneCountInString(local) >= minEmailLocalForPolicy {
		if strings.Contains(f, Fold(local)) {
			return PasswordContainsEmail
		}
	}
	if lowEntropy(f) {
		return PasswordLowEntropy
	}
	return ""
}

// lowEntropy: chỉ một ký tự lặp lại, hoặc dãy tăng liên tiếp theo mã ký tự (abcdefghij, 1234567890 — '0' tiếp sau '9').
func lowEntropy(s string) bool {
	rs := []rune(s)
	same, asc := true, true
	for i := 1; i < len(rs); i++ {
		same = same && rs[i] == rs[0]
		asc = asc && (rs[i] == rs[i-1]+1 || (rs[i-1] == '9' && rs[i] == '0'))
	}
	return same || asc
}
