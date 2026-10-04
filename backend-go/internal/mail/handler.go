package mail

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/config"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/store"
)

// vietnam: Việt Nam không có giờ mùa hè nên múi cố định +7 đủ dùng và không phụ thuộc tzdata của image.
var vietnam = time.FixedZone("ICT", 7*60*60) //nolint:gochecknoglobals // hằng số múi giờ, bất biến

// Handler là consumer của topic `mail.send` (đăng ký ở worker). Idempotent theo mail_outbox.id: khoá dòng
// FOR UPDATE, dòng đã SENT/DEAD thì bỏ qua — hai lần giao cùng tin chỉ gửi một thư.
type Handler struct {
	Pool   *pgxpool.Pool
	Clock  clock.Clock
	Sender Sender
	Cfg    config.Config
	Log    *slog.Logger
}

// permanentErr đánh dấu lỗi không thể khắc phục bằng thử lại.
type permanentErr struct{ class string }

func (e *permanentErr) Error() string { return e.class }

// Handle xử lý một tin mail.send. Giữ khoá dòng suốt lúc gửi (≤ MAIL_SEND_TIMEOUT) — đổi lại không bao giờ gửi đôi.
// ponytail: khoá giữ qua một lần SMTP; nếu thông lượng thư lớn thì chuyển sang trạng thái SENDING + claim.
func (h *Handler) Handle(ctx context.Context, m outbox.Message) error {
	var p struct {
		MailID uuid.UUID `json:"mail_id"`
	}
	if err := json.Unmarshal(m.Payload, &p); err != nil || p.MailID == uuid.Nil {
		return errors.New("payload mail.send không hợp lệ")
	}

	tx, err := h.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("mở transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }() // sau Commit là no-op; trên lỗi thì token vừa phát bị huỷ cùng.

	q := store.New(tx)
	row, err := q.LockMailOutbox(ctx, p.MailID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("khoá mail_outbox: %w", err)
	}
	if row.Status != store.MailStatusQUEUED {
		return nil // đã gửi hoặc đã chết: giao lại là no-op
	}

	out, err := h.build(ctx, tx, row)
	if err != nil {
		return h.fail(ctx, tx, row, m, err)
	}
	sctx, cancel := context.WithTimeout(ctx, h.Cfg.MailSendTimeout)
	defer cancel()
	if err := h.Sender.Send(sctx, out); err != nil {
		return h.fail(ctx, tx, row, m, err)
	}
	if err := q.MarkMailSent(ctx, row.ID); err != nil {
		return fmt.Errorf("đánh dấu SENT: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit SENT: %w", err)
	}
	return nil
}

// fail ghi kết quả thất bại bằng transaction riêng sau khi huỷ tx gửi (token phát lúc gửi chưa tới người nhận → bỏ).
// Vĩnh viễn → DEAD ngay, trả nil (không thử lại). Tạm thời → ghi lỗi, trả lỗi cho outbox thử lại; lần cuối → DEAD + trả lỗi để dead-letter.
func (h *Handler) fail(ctx context.Context, tx pgx.Tx, row store.MailOutbox, m outbox.Message, cause error) error {
	_ = tx.Rollback(ctx)
	class, permanent := failureClass(cause)
	last := m.Attempts >= len(h.Cfg.OutboxRetryBackoff)
	q := store.New(h.Pool)
	var err error
	if permanent || last {
		err = q.MarkMailDead(ctx, store.MarkMailDeadParams{ID: row.ID, LastError: class})
		h.Log.WarnContext(ctx, "thư chuyển DEAD", "mail_id", row.ID.String(), "template", row.Template, "class", class, "permanent", permanent)
	} else {
		err = q.RecordMailFailure(ctx, store.RecordMailFailureParams{ID: row.ID, LastError: class})
		h.Log.WarnContext(ctx, "gửi thư thất bại, sẽ thử lại", "mail_id", row.ID.String(), "template", row.Template, "class", class, "attempt", m.Attempts+1)
	}
	if err != nil {
		return fmt.Errorf("ghi kết quả gửi: %w", err)
	}
	if permanent {
		return nil
	}
	return errors.New(class) // chỉ nhãn: outbox ghi nguyên văn vào last_error và log
}

func failureClass(err error) (class string, permanent bool) {
	var sf *SendFailure
	if errors.As(err, &sf) {
		return sf.Class, sf.Permanent
	}
	var pe *permanentErr
	if errors.As(err, &pe) {
		return pe.class, true
	}
	return "send_failed", false
}

// build dựng thư: nạp biến từ payload, phát token (mẫu cần liên kết) và render.
func (h *Handler) build(ctx context.Context, tx pgx.Tx, row store.MailOutbox) (Mail, error) {
	s, ok := specs()[row.Template]
	if !ok {
		return Mail{}, &permanentErr{"template_unknown"}
	}
	var payload map[string]any
	if err := json.Unmarshal(row.Payload, &payload); err != nil {
		return Mail{}, &permanentErr{"payload_invalid"}
	}
	data := map[string]string{
		"LoginURL":  h.Cfg.AppPublicURL + "/login",
		"ForgotURL": h.Cfg.AppPublicURL + "/forgot-password",
	}
	for key, v := range map[string]string{
		"full_name": "FullName", "inviter_name": "InviterName", "role_vn": "RoleVN",
		"teacher_name": "TeacherName", "course_name": "CourseName", "class_code": "ClassCode",
	} {
		if s, ok := payload[key].(string); ok {
			data[v] = s
		}
	}
	for key, v := range map[string]string{"at": "At", "until": "Until"} {
		if s, ok := payload[key].(string); ok {
			ts, err := time.Parse(time.RFC3339, s)
			if err != nil {
				return Mail{}, &permanentErr{"payload_invalid"}
			}
			data[v] = ts.In(vietnam).Format("15:04 02/01/2006")
		}
	}

	if s.kind != "" {
		uid, err := uuid.Parse(fmt.Sprint(payload["user_id"]))
		if err != nil {
			return Mail{}, &permanentErr{"payload_invalid"}
		}
		if _, err := store.New(tx).GetUser(ctx, uid); errors.Is(err, pgx.ErrNoRows) {
			return Mail{}, &permanentErr{"user_missing"}
		} else if err != nil {
			return Mail{}, fmt.Errorf("đọc người dùng: %w", err)
		}
		plain, err := auth.Tokens{Clock: h.Clock}.Issue(ctx, tx, uid, s.kind, h.ttl(s.kind), nil)
		if err != nil {
			return Mail{}, err
		}
		data[linkVar] = h.Cfg.AppPublicURL + s.route + plain
	}

	r, err := Render(row.Template, data)
	if err != nil {
		if errors.Is(err, ErrMissingVar) {
			return Mail{}, &permanentErr{"template_missing_var"}
		}
		return Mail{}, &permanentErr{"template_render"}
	}
	return Mail{To: strings.ToLower(row.ToAddr), Subject: r.Subject, Text: r.Text, HTML: r.HTML}, nil
}

func (h *Handler) ttl(k auth.TokenKind) time.Duration {
	switch k {
	case auth.TokenVerifyEmail:
		return h.Cfg.VerifyTokenTTL
	case auth.TokenResetPassword:
		return h.Cfg.ResetTokenTTL
	default:
		return h.Cfg.InviteTokenTTL
	}
}
