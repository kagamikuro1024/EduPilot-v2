package thread

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/notify"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/store"
)

// Notifier là các handler outbox của kiểm duyệt Threads (SRS 4.11): chuông cho chủ bài (POST_HIDDEN / POST_UNHIDDEN) và đếm lại hai dòng gộp của Staff
// (`POST_REPORTED`, `AI_PENDING`). Mọi mắt xích idempotent: `dedupe_key` theo id thông điệp outbox, đếm theo trạng thái HIỆN TẠI (không theo sự kiện).
type Notifier struct {
	Q      *store.Queries // trên pool (đọc người nhận)
	Notify *notify.Service
}

type modEvent struct {
	CourseID uuid.UUID  `json:"course_id"`
	ThreadID uuid.UUID  `json:"thread_id"`
	TargetID uuid.UUID  `json:"target_id"`
	AuthorID *uuid.UUID `json:"author_id"`
	Auto     bool       `json:"auto"`
	Reason   string     `json:"reason"`
}

func decodeMod(m outbox.Message) (modEvent, error) {
	var e modEvent
	if err := json.Unmarshal(m.Payload, &e); err != nil || e.CourseID == uuid.Nil {
		return e, fmt.Errorf("payload %s không hợp lệ", m.Topic)
	}
	return e, nil
}

// RecountReported cập nhật dòng gộp POST_REPORTED theo số bài bị báo cáo / tự ẩn chờ duyệt HIỆN TẠI của lớp.
func (n *Notifier) RecountReported(ctx context.Context, courseID uuid.UUID) error {
	staff, err := n.Q.ModerationStaffIDs(ctx, courseID)
	if err != nil {
		return fmt.Errorf("thread: người nhận: %w", err)
	}
	return n.Notify.Aggregate(ctx, notify.Agg{
		CourseID: courseID, Type: notify.TypePostReported, Recipients: staff, Link: "/threads?view=moderation&course=" + courseID.String(),
		Title: func(k int) string { return fmt.Sprintf("%d bài bị báo cáo chờ xem", k) },
		Count: func(ctx context.Context, q *store.Queries) (int, error) {
			k, err := q.ModerationPendingCount(ctx, courseID)
			return int(k), err
		},
	})
}

// RecountAIPending cập nhật dòng gộp AI_PENDING theo số bài AI PENDING chưa ẩn HIỆN TẠI của lớp (về 0 → tự đánh dấu đã đọc).
func (n *Notifier) RecountAIPending(ctx context.Context, courseID uuid.UUID) error {
	staff, err := n.Q.ModerationStaffIDs(ctx, courseID)
	if err != nil {
		return fmt.Errorf("thread: người nhận: %w", err)
	}
	return n.Notify.Aggregate(ctx, notify.Agg{
		CourseID: courseID, Type: notify.TypeAIPending, Recipients: staff, Link: "/threads?state=pending&course=" + courseID.String(),
		Title: func(k int) string { return fmt.Sprintf("%d câu trả lời AI chờ duyệt", k) },
		Count: func(ctx context.Context, q *store.Queries) (int, error) {
			k, err := q.ModerationAIPendingCount(ctx, courseID)
			return int(k), err
		},
	})
}

// HandleReported (`thread.reported`): đếm lại POST_REPORTED.
func (n *Notifier) HandleReported(ctx context.Context, m outbox.Message) error {
	e, err := decodeMod(m)
	if err != nil {
		return err
	}
	return n.RecountReported(ctx, e.CourseID)
}

// HandleAIPending (`thread.ai_answered`, `thread.post_decided`, `thread.edited`): đếm lại AI_PENDING.
func (n *Notifier) HandleAIPending(ctx context.Context, m outbox.Message) error {
	e, err := decodeMod(m)
	if err != nil {
		return err
	}
	return n.RecountAIPending(ctx, e.CourseID)
}

// HandleHidden (`thread.hidden`): báo chủ bài (tự ẩn: "đang được giảng viên xem xét"; ẩn thủ công: kèm lý do — không lộ người báo cáo), rồi đếm lại.
func (n *Notifier) HandleHidden(ctx context.Context, m outbox.Message) error {
	e, err := decodeMod(m)
	if err != nil {
		return err
	}
	if e.AuthorID != nil {
		title := "Bài của bạn đã bị ẩn: " + e.Reason
		if e.Auto {
			title = "Bài của bạn đang được giảng viên xem xét"
		}
		if r := []rune(title); len(r) > 200 {
			title = string(r[:200])
		}
		if err := n.sendAuthor(ctx, e, m, notify.TypePostHidden, title, "hidden"); err != nil {
			return err
		}
	}
	if err := n.RecountReported(ctx, e.CourseID); err != nil {
		return err
	}
	return n.RecountAIPending(ctx, e.CourseID)
}

// HandleUnhidden (`thread.unhidden`): "Bài của bạn đã hiện lại" cho chủ bài, rồi đếm lại.
func (n *Notifier) HandleUnhidden(ctx context.Context, m outbox.Message) error {
	e, err := decodeMod(m)
	if err != nil {
		return err
	}
	if e.AuthorID != nil {
		if err := n.sendAuthor(ctx, e, m, notify.TypePostUnhidden, "Bài của bạn đã hiện lại", "unhidden"); err != nil {
			return err
		}
	}
	if err := n.RecountReported(ctx, e.CourseID); err != nil {
		return err
	}
	return n.RecountAIPending(ctx, e.CourseID)
}

// HandleDeleted (`thread.deleted`): đếm lại cả hai dòng gộp (bài đã xoá không còn "chờ xem" / "chờ duyệt").
func (n *Notifier) HandleDeleted(ctx context.Context, m outbox.Message) error {
	e, err := decodeMod(m)
	if err != nil {
		return err
	}
	if err := n.RecountReported(ctx, e.CourseID); err != nil {
		return err
	}
	return n.RecountAIPending(ctx, e.CourseID)
}

func (n *Notifier) sendAuthor(ctx context.Context, e modEvent, m outbox.Message, typ, title, key string) error {
	link := "/threads/" + e.ThreadID.String()
	if _, err := n.Notify.Send(ctx, notify.Spec{UserID: *e.AuthorID, CourseID: &e.CourseID, Type: typ, Title: title, Link: &link, DedupeKey: key + ":" + e.TargetID.String() + ":" + m.ID.String()}); err != nil {
		return fmt.Errorf("thread: thông báo chủ bài: %w", err)
	}
	return nil
}
