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

// Loại thông báo / sự kiện SSE của kết quả (US-PE-08).
const (
	TypeExamPublished     = "EXAM_PUBLISHED"
	TypeExamRegraded      = "EXAM_REGRADED"
	TypeExamAppealNew     = "EXAM_APPEAL_NEW"
	TypeExamAppealReplied = "EXAM_APPEAL_REPLIED"

	EventAttempt   = "exam.attempt"
	EventPublished = "exam.published"
)

// ResultNotifier xử lý topic outbox của chấm / công bố / phúc khảo ở worker. Idempotent: khử trùng thông báo theo khoá của SRS 4.8; SSE phát lại chỉ làm máy khách nhận thêm một sự kiện cùng nội dung.
type ResultNotifier struct {
	Pool *pgxpool.Pool
	SSE  sse.Publisher
	Log  *slog.Logger
}

type resultEvent struct {
	ExamID    uuid.UUID `json:"exam_id"`
	CourseID  uuid.UUID `json:"course_id"`
	AttemptID uuid.UUID `json:"attempt_id"`
	AppealID  uuid.UUID `json:"appeal_id"`
	UserID    uuid.UUID `json:"user_id"`
	Version   int       `json:"version"`
}

func (n *ResultNotifier) load(ctx context.Context, m outbox.Message) (resultEvent, *store.Exam, error) {
	var p resultEvent
	if err := json.Unmarshal(m.Payload, &p); err != nil || p.ExamID == uuid.Nil || p.CourseID == uuid.Nil {
		return p, nil, fmt.Errorf("payload %s không hợp lệ", m.Topic)
	}
	e, err := store.New(n.Pool).ExamGet(ctx, store.ExamGetParams{CourseID: p.CourseID, ID: p.ExamID})
	if errors.Is(err, pgx.ErrNoRows) {
		return p, nil, nil
	}
	if err != nil {
		return p, nil, fmt.Errorf("đọc bài thi: %w", err)
	}
	return p, &e, nil
}

func (n *ResultNotifier) notify(ctx context.Context, p resultEvent, typ, title, link, dedupe string, users []uuid.UUID) error {
	if len(users) == 0 {
		return nil
	}
	ids := make([]string, len(users))
	for i, u := range users {
		ids[i] = u.String()
	}
	if _, err := store.New(n.Pool).ExamNotifyUsers(ctx, store.ExamNotifyUsersParams{CourseID: &p.CourseID, Type: typ, Title: title, Link: &link, DedupePrefix: dedupe, UserIds: ids}); err != nil {
		return fmt.Errorf("ghi thông báo %s: %w", typ, err)
	}
	return nil
}

func (n *ResultNotifier) push(ctx context.Context, user uuid.UUID, typ string, data map[string]any) error {
	if n.SSE == nil {
		return nil
	}
	if _, err := n.SSE.Publish(ctx, user.String(), typ, data); err != nil {
		return fmt.Errorf("phát sự kiện %s: %w", typ, err)
	}
	return nil
}

// HandleGraded (`exam.attempt_graded`): báo chủ lượt rằng đã chấm xong (SSE `exam.attempt`; chưa có điểm — điểm chỉ lộ sau công bố).
func (n *ResultNotifier) HandleGraded(ctx context.Context, m outbox.Message) error {
	p, e, err := n.load(ctx, m)
	if err != nil || e == nil {
		return err
	}
	return n.push(ctx, p.UserID, EventAttempt, map[string]any{"attempt_id": p.AttemptID, "exam_id": p.ExamID, "status": string(store.AttemptStatusGRADED)})
}

// HandlePublished (`exam.published`): EXAM_PUBLISHED cho sinh viên CÓ lượt (khử trùng `exam.published:<exam>:<user>`) + SSE `exam.published`.
func (n *ResultNotifier) HandlePublished(ctx context.Context, m outbox.Message) error {
	p, e, err := n.load(ctx, m)
	if err != nil || e == nil {
		return err
	}
	users, err := store.New(n.Pool).ExamAttemptStudents(ctx, store.ExamAttemptStudentsParams{CourseID: p.CourseID, ExamID: p.ExamID})
	if err != nil {
		return fmt.Errorf("sinh viên có lượt: %w", err)
	}
	link := fmt.Sprintf("/exams/%s/take", e.ID)
	if err := n.notify(ctx, p, TypeExamPublished, fmt.Sprintf("Điểm bài thi %s đã công bố", e.Title), link, "exam.published:"+e.ID.String(), users); err != nil {
		return err
	}
	for _, u := range users {
		if err := n.push(ctx, u, EventPublished, map[string]any{"exam_id": e.ID}); err != nil {
			return err
		}
	}
	return nil
}

// HandleRegraded (`exam.regraded`, chỉ sau PUBLISHED): EXAM_REGRADED cho chủ lượt; khoá `exam.regraded:<attempt_id>:v<version>` (SRS 4.9) nên một lần đổi điểm báo đúng một lần.
func (n *ResultNotifier) HandleRegraded(ctx context.Context, m outbox.Message) error {
	p, e, err := n.load(ctx, m)
	if err != nil || e == nil || p.AttemptID == uuid.Nil || p.UserID == uuid.Nil {
		return err
	}
	link := fmt.Sprintf("/exams/%s/take", e.ID)
	if err := n.notify(ctx, p, TypeExamRegraded, fmt.Sprintf("Điểm bài thi %s đã được giảng viên điều chỉnh", e.Title), link, fmt.Sprintf("exam.regraded:%s:v%d", p.AttemptID, p.Version), []uuid.UUID{p.UserID}); err != nil {
		return err
	}
	return n.push(ctx, p.UserID, EventPublished, map[string]any{"exam_id": e.ID})
}

// HandleAppealCreated (`exam.appeal_created`): EXAM_APPEAL_NEW cho Giảng viên + TA.
func (n *ResultNotifier) HandleAppealCreated(ctx context.Context, m outbox.Message) error {
	p, e, err := n.load(ctx, m)
	if err != nil || e == nil {
		return err
	}
	staff, err := store.New(n.Pool).ExamCourseStaff(ctx, p.CourseID)
	if err != nil {
		return fmt.Errorf("giảng viên + TA: %w", err)
	}
	return n.notify(ctx, p, TypeExamAppealNew, fmt.Sprintf("Có yêu cầu xem lại điểm bài thi %s", e.Title), fmt.Sprintf("/exams/%s/results?tab=appeals", e.ID), "exam.appeal_created:"+p.AppealID.String(), staff)
}

// HandleAppealAnswered (`exam.appeal_answered`): EXAM_APPEAL_REPLIED cho sinh viên + SSE.
func (n *ResultNotifier) HandleAppealAnswered(ctx context.Context, m outbox.Message) error {
	p, e, err := n.load(ctx, m)
	if err != nil || e == nil || p.UserID == uuid.Nil {
		return err
	}
	if err := n.notify(ctx, p, TypeExamAppealReplied, fmt.Sprintf("Giảng viên đã trả lời yêu cầu xem lại bài thi %s", e.Title), fmt.Sprintf("/exams/%s/take", e.ID), "exam.appeal_answered:"+p.AppealID.String(), []uuid.UUID{p.UserID}); err != nil {
		return err
	}
	return n.push(ctx, p.UserID, EventPublished, map[string]any{"exam_id": e.ID})
}
