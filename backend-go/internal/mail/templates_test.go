package mail_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/mail"
)

// sample trả dữ liệu mẫu cố định cho từng mẫu thư (đủ mọi biến, kể cả Link).
func sample() map[string]string {
	return map[string]string{
		"FullName": "Nguyễn Văn An", "Link": "https://localhost/x?token=AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		"LoginURL": "https://localhost/login", "ForgotURL": "https://localhost/forgot-password",
		"At": "09:30 03/10/2026", "Until": "09:45 03/10/2026",
		"InviterName": "Quản trị viên Lê Hoa", "RoleVN": "giảng viên",
		"TeacherName": "Trần Minh", "CourseName": "Lập trình hướng đối tượng", "ClassCode": "OOP-01",
	}
}

var allTemplates = []string{"verify_email", "email_exists", "reset_password", "password_changed", "invite_staff", "invite_student", "account_locked"}

// requiredVars: biến người gọi PHẢI cung cấp theo SRS 6.5 (Link cho mẫu có liên kết).
func requiredVars(name string) []string {
	switch name {
	case "verify_email", "reset_password":
		return []string{"FullName", "Link"}
	case "email_exists":
		return []string{"FullName", "LoginURL", "ForgotURL"}
	case "password_changed":
		return []string{"FullName", "At", "ForgotURL"}
	case "invite_staff":
		return []string{"FullName", "InviterName", "RoleVN", "Link"}
	case "invite_student":
		return []string{"FullName", "TeacherName", "CourseName", "ClassCode", "Link"}
	default: // account_locked
		return []string{"FullName", "Until", "ForgotURL"}
	}
}

func TestTemplatesRender(t *testing.T) {
	t.Parallel()
	for _, name := range allTemplates {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			r, err := mail.Render(name, sample())
			require.NoError(t, err)
			for kind, body := range map[string]string{"subject": r.Subject, "text": r.Text, "html": r.HTML} {
				for _, bad := range []string{"{{", "}}", "<no value>", "%!"} {
					require.NotContains(t, body, bad, "%s/%s còn %q", name, kind, bad)
				}
			}
			require.NotContains(t, r.Subject, "Nguyễn", "tiêu đề không chứa tên người")
			require.Contains(t, r.Text, "Nguyễn Văn An")
			require.Contains(t, r.HTML, "<html")
			require.Contains(t, r.Text, "— EduPilot · Thư tự động từ hệ thống, vui lòng không trả lời.")
			if strings.Contains(strings.Join(requiredVars(name), ","), "Link") {
				require.Contains(t, r.HTML, `href="https://localhost/x?token=`, "bản HTML có nút liên kết")
			}

			// Golden: lời văn khớp SRS 6.5. Đặt UPDATE_GOLDEN=1 chỉ khi viết mẫu MỚI.
			got := "Subject: " + r.Subject + "\n\n" + r.Text
			path := filepath.Join("testdata", "golden", name+".txt")
			if os.Getenv("UPDATE_GOLDEN") == "1" {
				require.NoError(t, os.WriteFile(path, []byte(got), 0o600))
			}
			want, err := os.ReadFile(path)
			require.NoError(t, err)
			require.Equal(t, string(want), got)
		})
	}
}

func TestTemplatesEscape(t *testing.T) {
	t.Parallel()
	evil := `<script>alert(1)</script> "Bob" & co`
	for _, name := range allTemplates {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			d := sample()
			for _, k := range []string{"FullName", "InviterName", "TeacherName", "CourseName"} {
				d[k] = evil
			}
			r, err := mail.Render(name, d)
			require.NoError(t, err)
			require.NotContains(t, r.HTML, "<script>", "HTML phải thoát")
			require.Contains(t, r.HTML, "&lt;script&gt;")
			require.Contains(t, r.Text, evil, "chữ thuần giữ nguyên văn")
			require.NotContains(t, r.Subject, "script")
		})
	}
}

func TestTemplateMissingVar(t *testing.T) {
	t.Parallel()
	for _, name := range allTemplates {
		for _, v := range requiredVars(name) {
			t.Run(name+"/"+v, func(t *testing.T) {
				t.Parallel()
				d := sample()
				delete(d, v)
				_, err := mail.Render(name, d)
				require.ErrorIs(t, err, mail.ErrMissingVar)
				d[v] = "  "
				_, err = mail.Render(name, d)
				require.ErrorIs(t, err, mail.ErrMissingVar, "rỗng cũng là thiếu")
			})
		}
	}
	_, err := mail.Render("khong_co", sample())
	require.True(t, errors.Is(err, mail.ErrUnknownTemplate))
}
