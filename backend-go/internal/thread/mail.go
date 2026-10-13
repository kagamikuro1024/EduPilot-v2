package thread

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/edupilot/backend-go/internal/mail"
	"github.com/edupilot/backend-go/internal/store"
)

// MailThreadReply là tên mẫu thư báo người đăng thread (SRS escalation 4.5.2). Sự kiện: REPLY (bình luận mới), VERIFIED, CORRECTED.
const MailThreadReply = "thread_reply"

var eventText = map[string]string{ //nolint:gochecknoglobals // bảng tra cứu bất biến
	"REPLY": "Có bình luận mới", "VERIFIED": "Giảng viên đã xác nhận câu trả lời", "CORRECTED": "Giảng viên đã sửa câu trả lời",
}

// MailResolver nạp biến hiển thị của `thread_reply` lúc gửi (payload chỉ có `{thread_id, event, post_id?}`).
type MailResolver struct{ PublicURL string }

// Register gắn resolver vào bảng của mail.Handler.
func (m MailResolver) Register(into map[string]mail.Resolver) { into[MailThreadReply] = m }

// Resolve implements mail.Resolver.
func (m MailResolver) Resolve(ctx context.Context, tx pgx.Tx, row store.MailOutbox) (map[string]string, error) {
	var p struct {
		ThreadID uuid.UUID `json:"thread_id"`
		Event    string    `json:"event"`
	}
	if err := json.Unmarshal(row.Payload, &p); err != nil || p.ThreadID == uuid.Nil || eventText[p.Event] == "" {
		return nil, mail.Permanent("payload_invalid")
	}
	t, err := store.New(tx).ThreadMailInfo(ctx, p.ThreadID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, mail.Permanent("thread_missing")
	}
	if err != nil {
		return nil, fmt.Errorf("thread: đọc thread cho thư: %w", err)
	}
	return map[string]string{"Event": eventText[p.Event], "ThreadTitle": mail.Clip(t.Title, 200, "…"), "ThreadURL": m.PublicURL + "/threads/" + t.ID.String()}, nil
}

// enqueueMail xếp thư `thread_reply` cho người đăng thread TRONG giao dịch của người gọi. Chỉ tới email đã xác minh và khi `notify_answer_by_mail` bật.
// dedupe: `thread_reply:<post_id|thread_id>:<event>:<user_id>`.
func enqueueMail(ctx context.Context, tx pgx.Tx, to, threadID uuid.UUID, postID *uuid.UUID, event string) error {
	tgt, err := store.New(tx).AnswerMailTarget(ctx, to)
	if err != nil {
		return fmt.Errorf("thread: người nhận thư: %w", err)
	}
	if !tgt.Verified || !tgt.WantsMail || tgt.Email == "" {
		return nil
	}
	ref := threadID.String()
	payload := map[string]any{"thread_id": threadID, "event": event}
	if postID != nil {
		ref = postID.String()
		payload["post_id"] = *postID
	}
	if _, _, err := mail.Enqueue(ctx, tx, mail.Message{To: tgt.Email, Template: MailThreadReply, Payload: payload, DedupeKey: fmt.Sprintf("thread_reply:%s:%s:%s", ref, event, to)}); err != nil {
		return fmt.Errorf("thread: xếp thư: %w", err)
	}
	return nil
}
