package exam

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/edupilot/backend-go/internal/httpapi/sse"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/store"
)

// Loại sự kiện SSE gửi cho sinh viên khi một lần chạy thử / bản nộp xong (SRS 4.4.2–4.4.3). Dữ liệu CHỈ gồm id + trạng thái; kết quả đọc bằng `GET`.
const (
	EventRun        = "exam.run"
	EventSubmission = "exam.submission"
)

// CodeNotifier chuyển `exam.submission_done` (outbox, máy chấm phát) thành sự kiện SSE tới ĐÚNG chủ bản nộp. Payload outbox không mang danh tính: chủ được tra ở DB.
type CodeNotifier struct {
	Pool *pgxpool.Pool
	SSE  sse.Publisher
	Log  *slog.Logger
	// Svc: hoàn tất chấm lượt (US-PE-08) khi một bản SUBMIT xong; nil ⇒ chỉ phát SSE.
	Svc *Service
}

// HandleDone idempotent: phát lại chỉ làm máy khách nhận thêm một sự kiện cùng id (máy khách khử trùng theo id).
func (n *CodeNotifier) HandleDone(ctx context.Context, m outbox.Message) error {
	var p struct {
		SubmissionID uuid.UUID `json:"submission_id"`
	}
	if err := json.Unmarshal(m.Payload, &p); err != nil || p.SubmissionID == uuid.Nil {
		return nil // tin không có bản nộp (không còn gì để báo): bỏ, không thử lại vô ích
	}
	row, err := store.New(n.Pool).CodeSubmissionStudent(ctx, p.SubmissionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("exam: tra chủ bản nộp: %w", err)
	}
	if row.Kind == store.SubmissionKindSUBMIT && n.Svc != nil { // chấm xong lượt TRƯỚC khi báo: kết quả đã nhất quán khi máy khách đọc
		if err := n.Svc.OnSubmissionDone(ctx, row.CourseID, row.AttemptID); err != nil {
			return fmt.Errorf("exam: hoàn tất chấm: %w", err)
		}
	}
	typ, key := EventSubmission, "submission_id"
	if row.Kind == store.SubmissionKindRUN {
		typ, key = EventRun, "run_id"
	}
	if _, err := n.SSE.Publish(ctx, row.StudentID.String(), typ, map[string]any{key: p.SubmissionID, "status": string(row.Status), "attempt_id": row.AttemptID, "item_id": row.ItemID}); err != nil {
		return fmt.Errorf("exam: phát sự kiện chấm: %w", err)
	}
	return nil
}
