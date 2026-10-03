package mail

import (
	"bytes"
	"errors"
	"fmt"
	htmltpl "html/template"
	"strings"
	texttpl "text/template"

	"github.com/edupilot/backend-go/internal/auth"
)

// ErrMissingVar: mẫu thiếu biến bắt buộc → lỗi dựng, thư không gửi (DEAD ngay, không thử lại — US-P2-01 AC8).
var ErrMissingVar = errors.New("mail: thiếu biến của mẫu")

// ErrUnknownTemplate: tên mẫu không có trong bảng mẫu.
var ErrUnknownTemplate = errors.New("mail: mẫu không tồn tại")

const footer = "— EduPilot · Thư tự động từ hệ thống, vui lòng không trả lời."

// linkVar là biến liên kết một lần; đoạn chỉ gồm biến này hiện thành nút ở bản HTML.
const linkVar = "Link"

// spec là một mẫu thư (SRS FEAT-account-security 6.5). Thân là các đoạn; bản chữ thuần nối bằng dòng trống,
// bản HTML bọc <p>. Tiêu đề KHÔNG chứa tên người (an toàn cho dòng hiển thị ở hộp thư).
type spec struct {
	subject string
	paras   []string
	// kind != "" → consumer phát token lúc gửi và đặt biến Link (SRS 5.6); route là đường dẫn dựng liên kết.
	kind  auth.TokenKind
	route string // "/verify-email?token=" | "/reset-password?token=" | "/invite/"
	// vars: biến người gọi (payload) phải cung cấp; biến dựng sẵn (Link, LoginURL, ForgotURL) do consumer thêm.
	vars []string
}

// specs dựng bảng mẫu mỗi lần gọi (không biến toàn cục). Thêm mẫu = thêm một phần tử, không cần ALTER (P4).
func specs() map[string]spec {
	return map[string]spec{
		"verify_email": {
			subject: "Xác minh email EduPilot của bạn", kind: auth.TokenVerifyEmail, route: "/verify-email?token=",
			vars: []string{"FullName"},
			paras: []string{
				"Chào {{.FullName}},",
				"Bạn vừa đăng ký tài khoản EduPilot bằng địa chỉ email này. Để hoàn tất, hãy xác minh email bằng liên kết dưới đây (dùng một lần, có hiệu lực 24 giờ):",
				"{{.Link}}",
				"Nếu không phải bạn đăng ký, hãy bỏ qua thư này — sẽ không có tài khoản nào được kích hoạt.",
			},
		},
		"email_exists": {
			subject: "Bạn đã có tài khoản EduPilot", vars: []string{"FullName", "LoginURL", "ForgotURL"},
			paras: []string{
				"Chào {{.FullName}},",
				"Có người vừa dùng địa chỉ email này để đăng ký EduPilot, nhưng email đã có tài khoản.",
				"Nếu là bạn: đăng nhập tại {{.LoginURL}}, hoặc đặt lại mật khẩu nếu quên tại {{.ForgotURL}}.\nNếu không phải bạn: bỏ qua thư này, tài khoản của bạn không bị ảnh hưởng.",
			},
		},
		"reset_password": {
			subject: "Đặt lại mật khẩu EduPilot", kind: auth.TokenResetPassword, route: "/reset-password?token=",
			vars: []string{"FullName"},
			paras: []string{
				"Chào {{.FullName}},",
				"Chúng tôi nhận được yêu cầu đặt lại mật khẩu cho tài khoản EduPilot của bạn. Dùng liên kết dưới đây để đặt mật khẩu mới (dùng một lần, có hiệu lực 30 phút):",
				"{{.Link}}",
				"Nếu không phải bạn yêu cầu, hãy bỏ qua thư này — mật khẩu hiện tại vẫn giữ nguyên.",
			},
		},
		"password_changed": {
			subject: "Mật khẩu EduPilot của bạn đã được đổi", vars: []string{"FullName", "At", "ForgotURL"},
			paras: []string{
				"Chào {{.FullName}},",
				"Mật khẩu tài khoản EduPilot của bạn vừa được đổi lúc {{.At}} (giờ Việt Nam). Mọi thiết bị khác đã bị đăng xuất.",
				"Nếu không phải bạn: đặt lại mật khẩu ngay tại {{.ForgotURL}} và báo cho giảng viên hoặc quản trị viên.",
			},
		},
		"invite_staff": {
			subject: "Lời mời tham gia EduPilot", kind: auth.TokenInvite, route: "/invite/",
			vars: []string{"FullName", "InviterName", "RoleVN"},
			paras: []string{
				"Chào {{.FullName}},",
				"{{.InviterName}} mời bạn làm {{.RoleVN}} trên EduPilot. Hãy đặt mật khẩu để bắt đầu (liên kết dùng một lần, có hiệu lực 72 giờ):",
				"{{.Link}}",
				"Quản trị viên không biết và không bao giờ cần mật khẩu của bạn. Nếu bạn không mong đợi lời mời này, hãy bỏ qua thư.",
			},
		},
		"invite_student": {
			subject: "Bạn được thêm vào lớp trên EduPilot", kind: auth.TokenInvite, route: "/invite/",
			vars: []string{"FullName", "TeacherName", "CourseName", "ClassCode"},
			paras: []string{
				"Chào {{.FullName}},",
				"Giảng viên {{.TeacherName}} đã thêm bạn vào lớp {{.CourseName}} – {{.ClassCode}} trên EduPilot. Hãy đặt mật khẩu để vào lớp (liên kết dùng một lần, có hiệu lực 72 giờ):",
				"{{.Link}}",
				"Nếu bạn không học lớp này, hãy bỏ qua thư.",
			},
		},
		"account_locked": {
			subject: "Tài khoản EduPilot bị khoá tạm thời", vars: []string{"FullName", "Until", "ForgotURL"},
			paras: []string{
				"Chào {{.FullName}},",
				"Có nhiều lần đăng nhập sai vào tài khoản EduPilot của bạn. Để an toàn, tài khoản bị khoá tạm thời 15 phút, đến {{.Until}} (giờ Việt Nam).",
				"Nếu đó là bạn, hãy đợi hết giờ rồi đăng nhập lại, hoặc đặt lại mật khẩu tại {{.ForgotURL}}. Nếu không phải bạn, nên đặt lại mật khẩu ngay.",
			},
		},
	}
}

// Rendered là thư đã dựng: tiêu đề + chữ thuần + HTML.
type Rendered struct{ Subject, Text, HTML string }

// Render dựng thư từ mẫu `name` với `data` (khoá = tên biến). Thiếu hoặc rỗng một biến bắt buộc → ErrMissingVar.
// Chữ thuần giữ nguyên văn (text/template); HTML thoát tự động (html/template).
func Render(name string, data map[string]string) (Rendered, error) {
	s, ok := specs()[name]
	if !ok {
		return Rendered{}, fmt.Errorf("%w: %s", ErrUnknownTemplate, name)
	}
	need := s.vars
	if s.kind != "" {
		need = append(append([]string(nil), need...), linkVar)
	}
	for _, v := range need {
		if strings.TrimSpace(data[v]) == "" {
			return Rendered{}, fmt.Errorf("%w: %s.%s", ErrMissingVar, name, v)
		}
	}

	tt, err := texttpl.New(name).Option("missingkey=error").Parse(strings.Join(s.paras, "\n\n") + "\n\n" + footer)
	if err != nil {
		return Rendered{}, fmt.Errorf("mail: parse text %s: %w", name, err)
	}
	var tb bytes.Buffer
	if err := tt.Execute(&tb, data); err != nil {
		return Rendered{}, fmt.Errorf("mail: render text %s: %w", name, err)
	}

	var hb strings.Builder
	for _, p := range s.paras {
		if p == "{{."+linkVar+"}}" {
			hb.WriteString(`<p><a href="{{.Link}}" style="display:inline-block;padding:10px 18px;background:#C4161C;color:#ffffff;text-decoration:none;border-radius:8px;font-weight:600">Mở liên kết</a></p>` +
				`<p style="color:#746968;font-size:13px">Nếu nút không bấm được, sao chép liên kết này vào trình duyệt: {{.Link}}</p>`)
			continue
		}
		hb.WriteString("<p>" + strings.ReplaceAll(p, "\n", "<br>") + "</p>")
	}
	hb.WriteString(`<p style="color:#746968;font-size:13px">` + footer + `</p>`)
	ht, err := htmltpl.New(name).Option("missingkey=error").Parse(`<!doctype html><html lang="vi"><body style="font-family:sans-serif;color:#222">` + hb.String() + `</body></html>`)
	if err != nil {
		return Rendered{}, fmt.Errorf("mail: parse html %s: %w", name, err)
	}
	var hbuf bytes.Buffer
	if err := ht.Execute(&hbuf, data); err != nil {
		return Rendered{}, fmt.Errorf("mail: render html %s: %w", name, err)
	}
	return Rendered{Subject: s.subject, Text: tb.String(), HTML: hbuf.String()}, nil
}
