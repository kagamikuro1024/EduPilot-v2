// Package mail: lõi gửi thư (D36). Handler nghiệp vụ gọi Enqueue TRONG transaction của mình (mail_outbox + outbox cùng commit);
// consumer `mail.send` ở worker dựng thư, phát token một lần lúc gửi (bản rõ chỉ nằm trong nội dung thư và bộ nhớ consumer),
// gửi SMTP, đánh dấu SENT. Không có đường HTTP đọc mail_outbox.
package mail

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/store"
)

// Topic là topic outbox mà consumer thư đăng ký.
const Topic = "mail.send"

// ErrSecretInPayload: payload thư chứa khoá có vẻ là bí mật (token, mật khẩu, liên kết) — bí mật không được nằm ở hàng đợi (AC9).
var ErrSecretInPayload = errors.New("mail: payload không được chứa token, mật khẩu hay liên kết")

var secretKeyRE = regexp.MustCompile(`(?i)token|password|passwd|secret|link|url`)

// Message là một thư cần gửi. Payload chỉ chứa định danh và dữ liệu hiển thị (user_id, full_name, at, …).
type Message struct {
	To        string
	Template  string
	Payload   map[string]any
	DedupeKey string // rỗng = không khử trùng; trùng khoá = bỏ qua, không xếp thư thứ hai
}

// Enqueue ghi mail_outbox và outbox(mail.send) bằng transaction tx của người gọi. Trả (id, queued):
// queued=false khi DedupeKey đã tồn tại (không ghi gì thêm).
func Enqueue(ctx context.Context, tx pgx.Tx, m Message) (uuid.UUID, bool, error) {
	for k := range m.Payload {
		if secretKeyRE.MatchString(k) {
			return uuid.Nil, false, fmt.Errorf("%w (khoá %q)", ErrSecretInPayload, k)
		}
	}
	raw, err := json.Marshal(m.Payload)
	if err != nil {
		return uuid.Nil, false, fmt.Errorf("mail: marshal payload: %w", err)
	}
	if m.Payload == nil {
		raw = []byte("{}")
	}
	var dedupe *string
	if m.DedupeKey != "" {
		dedupe = &m.DedupeKey
	}
	row, err := store.New(tx).InsertMailOutbox(ctx, store.InsertMailOutboxParams{
		ToAddr: strings.ToLower(strings.TrimSpace(m.To)), Template: m.Template, Payload: raw, DedupeKey: dedupe,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, false, nil // trùng dedupe_key
	}
	if err != nil {
		return uuid.Nil, false, fmt.Errorf("mail: ghi mail_outbox: %w", err)
	}
	if _, err := outbox.Write(ctx, tx, Topic, map[string]string{"mail_id": row.ID.String()}); err != nil {
		return uuid.Nil, false, err
	}
	return row.ID, true, nil
}
