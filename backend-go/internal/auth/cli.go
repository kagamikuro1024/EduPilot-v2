package auth

import (
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/platform/clock"
)

// DevEmail là email mặc định của token do CLI cấp (không phải người dùng thật).
const DevEmail = "dev@edupilot.local"

// RunTokenCommand chạy `gateway token --role R [--sub UUID] [--email E] [--ttl D]` (FR-44):
// in ĐÚNG một dòng JWT ra stdout, mọi thông báo lỗi ra stderr. Bị chặn khi APP_ENV=production.
// Trả mã thoát của tiến trình (0 thành công, 1 lỗi).
func RunTokenCommand(args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("token", flag.ContinueOnError)
	fs.SetOutput(stderr)
	role := fs.String("role", "", "vai trò: ADMIN|TEACHER|TA|STUDENT")
	sub := fs.String("sub", "", "uuid người dùng (bỏ trống: sinh uuid v7 ngẫu nhiên)")
	email := fs.String("email", DevEmail, "claim email")
	ttl := fs.String("ttl", DefaultTTL.String(), "hạn token, ví dụ 15m")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	fail := func(msg string) int {
		_, _ = fmt.Fprintln(stderr, "token: "+msg)
		return 1
	}

	if strings.EqualFold(strings.TrimSpace(getenv("APP_ENV")), "production") {
		return fail("bị chặn khi APP_ENV=production (chỉ là công cụ dev)")
	}
	r := Role(*role)
	if !r.Valid() {
		return fail("--role phải là một trong ADMIN, TEACHER, TA, STUDENT")
	}
	id, err := devSubject(*sub)
	if err != nil {
		return fail("--sub phải là uuid")
	}
	d, err := time.ParseDuration(*ttl)
	if err != nil {
		return fail("--ttl phải là khoảng thời gian, ví dụ 15m")
	}
	secret := strings.TrimSpace(getenv("JWT_SECRET_KEY"))
	if secret == "" {
		return fail("thiếu biến môi trường JWT_SECRET_KEY")
	}

	tok, err := NewIssuer(secret, d, clock.Real{}).Issue(id, r, *email)
	if err != nil {
		return fail("không ký được token")
	}
	_, _ = fmt.Fprintln(stdout, tok)
	return 0
}

// devSubject trả `sub`: uuid v7 ngẫu nhiên khi bỏ --sub, ngược lại là uuid do người dùng đưa.
func devSubject(sub string) (string, error) {
	if sub == "" {
		u, err := uuid.NewV7()
		if err != nil {
			return "", err
		}
		return u.String(), nil
	}
	u, err := uuid.Parse(sub)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}
