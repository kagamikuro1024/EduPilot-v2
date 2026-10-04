package mail

import (
	"context"
	"errors"
	"fmt"

	gomail "github.com/wneessen/go-mail"

	"github.com/edupilot/backend-go/internal/platform/config"
)

// Mail là một thư đã dựng, sẵn sàng gửi.
type Mail struct{ To, Subject, Text, HTML string }

// Sender gửi một thư. Lỗi trả về PHẢI là *SendFailure (đã phân loại, không chứa địa chỉ/nội dung).
type Sender interface {
	Send(ctx context.Context, m Mail) error
}

// SendFailure là lỗi gửi đã khử dữ liệu cá nhân: Class là nhãn ngắn ghi vào mail_outbox.last_error và log.
type SendFailure struct {
	Permanent bool
	Class     string
}

func (e *SendFailure) Error() string { return e.Class }

// SMTP gửi qua go-mail theo cấu hình SMTP_* của worker.
type SMTP struct{ Cfg config.Config }

// Send mở một kết nối, gửi, đóng. Phân loại: mã 5xx = vĩnh viễn (không thử lại), còn lại (4xx, mạng, quá hạn) = tạm thời.
func (s SMTP) Send(ctx context.Context, m Mail) error {
	msg := gomail.NewMsg()
	if err := msg.From(s.Cfg.MailFrom); err != nil {
		return &SendFailure{Permanent: true, Class: "bad_from"}
	}
	if err := msg.To(m.To); err != nil {
		return &SendFailure{Permanent: true, Class: "bad_recipient"}
	}
	msg.Subject(m.Subject)
	msg.SetBodyString(gomail.TypeTextPlain, m.Text)
	msg.AddAlternativeString(gomail.TypeTextHTML, m.HTML)

	opts := []gomail.Option{gomail.WithPort(s.Cfg.SMTPPort), gomail.WithTimeout(s.Cfg.MailSendTimeout)}
	switch s.Cfg.SMTPTLS {
	case "tls":
		opts = append(opts, gomail.WithSSL())
	case "starttls":
		opts = append(opts, gomail.WithTLSPortPolicy(gomail.TLSMandatory))
	default:
		opts = append(opts, gomail.WithTLSPortPolicy(gomail.NoTLS))
	}
	if s.Cfg.SMTPUser != "" {
		opts = append(opts, gomail.WithSMTPAuth(gomail.SMTPAuthAutoDiscover), gomail.WithUsername(s.Cfg.SMTPUser), gomail.WithPassword(s.Cfg.SMTPPass))
	}
	c, err := gomail.NewClient(s.Cfg.SMTPHost, opts...)
	if err != nil {
		return &SendFailure{Permanent: true, Class: "bad_smtp_config"}
	}
	if err := c.DialAndSendWithContext(ctx, msg); err != nil {
		return classify(err)
	}
	return nil
}

func classify(err error) *SendFailure {
	var se *gomail.SendError
	if errors.As(err, &se) {
		if code := se.ErrorCode(); code >= 500 && code <= 599 {
			return &SendFailure{Permanent: true, Class: fmt.Sprintf("smtp_permanent_%d", code)}
		} else if code != 0 {
			return &SendFailure{Class: fmt.Sprintf("smtp_temporary_%d", code)}
		}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return &SendFailure{Class: "smtp_timeout"}
	}
	return &SendFailure{Class: "smtp_unavailable"}
}
