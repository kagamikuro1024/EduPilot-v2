package auth

import (
	"errors"
	"regexp"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

// Mật khẩu thử của gói này; KHÔNG ghi mật khẩu rõ hay hash ra log test (04-AC9).
const testPassword = "mat-khau-thu-nghiem-9"

func TestBcrypt_Format(t *testing.T) {
	re := regexp.MustCompile(`^\$2[aby]\$12\$.{53}$`)
	h1, err := HashPassword(testPassword, 0)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if !re.MatchString(h1) {
		t.Fatalf("hash không khớp ^\\$2[aby]\\$12\\$.{53}$ (dài %d)", len(h1))
	}
	h2, err := HashPassword(testPassword, 0)
	if err != nil {
		t.Fatalf("HashPassword lần 2: %v", err)
	}
	if h1 == h2 {
		t.Fatal("hai lần băm cùng mật khẩu cho cùng chuỗi (thiếu salt)")
	}
	if err := CheckPassword(h2, testPassword); err != nil {
		t.Fatalf("hash lần 2 không kiểm được: %v", err)
	}
}

func TestBcrypt_DefaultCost(t *testing.T) {
	tests := []struct {
		name string
		cost int
		want int
	}{
		{"mặc định (0)", 0, DefaultBcryptCost},
		{"âm → mặc định", -1, DefaultBcryptCost},
		{"cost 4 (dùng trong test)", 4, 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, err := HashPassword(testPassword, tt.cost)
			if err != nil {
				t.Fatalf("HashPassword: %v", err)
			}
			got, err := bcrypt.Cost([]byte(h))
			if err != nil {
				t.Fatalf("bcrypt.Cost: %v", err)
			}
			if got != tt.want {
				t.Fatalf("cost=%d, muốn %d", got, tt.want)
			}
		})
	}
	if DefaultBcryptCost < 10 {
		t.Fatalf("cost mặc định %d < 10 (production không được dưới 10)", DefaultBcryptCost)
	}
}

func TestBcrypt_TooLong(t *testing.T) {
	ok72 := strings.Repeat("a", MaxPasswordBytes)
	h, err := HashPassword(ok72, 4)
	if err != nil {
		t.Fatalf("72 byte phải băm được: %v", err)
	}

	long := strings.Repeat("a", MaxPasswordBytes+1)
	if _, err := HashPassword(long, 4); !errors.Is(err, ErrPasswordTooLong) {
		t.Fatalf("HashPassword(73 byte) lỗi %v, muốn ErrPasswordTooLong", err)
	}
	// Không cắt im lặng: mật khẩu 73 byte KHÔNG được coi là khớp hash của 72 byte đầu.
	if err := CheckPassword(h, long); !errors.Is(err, ErrPasswordTooLong) {
		t.Fatalf("CheckPassword(73 byte) lỗi %v, muốn ErrPasswordTooLong", err)
	}
	// Tiếng Việt nhiều byte: đếm theo BYTE, không theo rune.
	multi := strings.Repeat("ê", 25) // 50 byte
	if _, err := HashPassword(multi, 4); err != nil {
		t.Fatalf("50 byte UTF-8 phải băm được: %v", err)
	}
	if _, err := HashPassword(strings.Repeat("ê", 37), 4); !errors.Is(err, ErrPasswordTooLong) {
		t.Fatalf("74 byte UTF-8 phải bị từ chối, lỗi=%v", err)
	}
}

func TestBcrypt_Check(t *testing.T) {
	h, err := HashPassword(testPassword, 4)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	empty, err := HashPassword("", 4)
	if err != nil {
		t.Fatalf("HashPassword(rỗng): %v", err)
	}

	tests := []struct {
		name, hash, password string
		want                 error
	}{
		{"đúng", h, testPassword, nil},
		{"sai", h, testPassword + "x", bcrypt.ErrMismatchedHashAndPassword},
		{"rỗng với hash thật", h, "", bcrypt.ErrMismatchedHashAndPassword},
		{"rỗng với hash của rỗng", empty, "", nil},
		{"đúng mật khẩu nhưng hash khác", empty, testPassword, bcrypt.ErrMismatchedHashAndPassword},
		{"hash hỏng", "khong-phai-hash", testPassword, bcrypt.ErrHashTooShort},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// So sánh luôn đi qua bcrypt.CompareHashAndPassword (hằng thời gian), không so chuỗi:
			// mật khẩu sai trả đúng ErrMismatchedHashAndPassword của bcrypt.
			err := CheckPassword(tt.hash, tt.password)
			if tt.want == nil && err != nil {
				t.Fatalf("lỗi %v, muốn nil", err)
			}
			if tt.want != nil && !errors.Is(err, tt.want) {
				t.Fatalf("lỗi %v, muốn %v", err, tt.want)
			}
		})
	}
}
