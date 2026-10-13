package mail

import (
	"context"
	"errors"
	"fmt"
	netmail "net/mail"
	"strings"

	"github.com/google/uuid"
	gomail "github.com/wneessen/go-mail"

	"github.com/edupilot/backend-go/internal/platform/config"
)

// Mail là một thư đã dựng, sẵn sàng gửi.
// ID là mail_outbox.id: đặt Message-ID tất định `<id@domain MAIL_FROM>` để người nhận (Mailpit `MP_IGNORE_DUPLICATE_IDS`) gộp thư giao lại sau khi worker chết giữa lúc gửi và ghi SENT (SRS 4.5.4).
type Mail struct {
	ID                      uuid.UUID
	To, Subject, Text, HTML string
}

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
	if m.ID != uuid.Nil {
		msg.SetMessageIDWithValue(MessageID(m.ID, s.Cfg.MailFrom))
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

// MessageID là giá trị Message-ID (không có dấu ngoặc nhọn): `<mail_id>@<domain của MAIL_FROM>`.
func MessageID(id uuid.UUID, from string) string {
	domain := "edupilot.local"
	if a, err := netmail.ParseAddress(from); err == nil {
		if i := strings.LastIndex(a.Address, "@"); i >= 0 {
			domain = a.Address[i+1:]
		}
	}
	return id.String() + "@" + domain
}
