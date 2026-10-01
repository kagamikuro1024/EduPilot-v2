package auth

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

const (
	// DefaultBcryptCost là cost mặc định (BCRYPT_COST cho phép 4–14; production không dưới 10).
	DefaultBcryptCost = 12
	// MaxPasswordBytes là giới hạn cứng của bcrypt: quá thì từ chối, KHÔNG cắt im lặng.
	MaxPasswordBytes = 72
)

// ErrPasswordTooLong báo mật khẩu vượt 72 byte (giới hạn của bcrypt).
var ErrPasswordTooLong = errors.New("auth: mật khẩu vượt quá 72 byte")

// HashPassword băm mật khẩu bằng bcrypt; cost <= 0 dùng DefaultBcryptCost.
func HashPassword(password string, cost int) (string, error) {
	if len(password) > MaxPasswordBytes {
		return "", ErrPasswordTooLong
	}
	if cost <= 0 {
		cost = DefaultBcryptCost
	}
	h, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", err
	}
	return string(h), nil
}

// CheckPassword so mật khẩu với hash bằng bcrypt.CompareHashAndPassword (hằng thời gian, không so chuỗi).
// Sai → bcrypt.ErrMismatchedHashAndPassword.
func CheckPassword(hash, password string) error {
	if len(password) > MaxPasswordBytes {
		return ErrPasswordTooLong
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
