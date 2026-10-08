package today

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/edupilot/backend-go/internal/platform/outbox"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/store"
)

// Topic xoá cache "Hôm nay" (SRS 4.7, bảng 4.9). Chuỗi trùng hằng ở gói course / auth; khai lại ở đây để gói này không import chúng.
const (
	TopicJoinRequested = "course.join_requested"
	TopicJoinDecided   = "course.join_decided"
	TopicMemberChanged = "course.member_changed"
	TopicAssigned      = "course.assigned"
	TopicChanged       = "course.changed"
	TopicUserVerified  = "user.verified"
	TopicRosterImport  = "roster.imported"
	// Bài thi (US-PE-04): mốc lịch đổi việc của CẢ lớp (sinh viên và Staff).
	TopicExamScheduled   = "exam.scheduled"
	TopicExamUnscheduled = "exam.unscheduled"
	TopicExamOpened      = "exam.opened"
	TopicExamClosed      = "exam.closed"
)

// Topics là mọi topic mà Invalidator phải được đăng ký.
func Topics() []string {
	return []string{TopicJoinRequested, TopicJoinDecided, TopicMemberChanged, TopicAssigned, TopicChanged, TopicUserVerified, TopicRosterImport,
		TopicExamScheduled, TopicExamUnscheduled, TopicExamOpened, TopicExamClosed}
}

// Invalidator xoá `ep:today:{uid}:{scope}` của những người bị một sự kiện outbox ảnh hưởng. Idempotent (DEL).
type Invalidator struct {
	Pool  *pgxpool.Pool
	Redis *appredis.Client
	Log   *slog.Logger
}

type payload struct {
	CourseID string   `json:"course_id"`
	UserID   string   `json:"user_id"`
	UserIDs  []string `json:"user_ids"`
}

// Handle là outbox.Handler cho mọi topic trong Topics().
func (i Invalidator) Handle(ctx context.Context, m outbox.Message) error {
	if i.Redis == nil {
		return nil
	}
	var p payload
	if err := json.Unmarshal(m.Payload, &p); err != nil {
		return fmt.Errorf("today: payload %s: %w", m.Topic, err)
	}
	q := store.New(i.Pool)
	users := map[uuid.UUID]bool{}
	add := func(s string) {
		if id, err := uuid.Parse(s); err == nil {
			users[id] = true
		}
	}
	add(p.UserID)
	for _, u := range p.UserIDs {
		add(u)
	}
	var course uuid.UUID
	if c, err := uuid.Parse(p.CourseID); err == nil {
		course = c
		// Giảng viên + TA của lớp luôn bị ảnh hưởng (hàng chờ, thiết lập). course.changed ảnh hưởng cả người học.
		var uids []uuid.UUID
		var err error
		if m.Topic == TopicChanged || isExamTopic(m.Topic) { // sự kiện của bài thi ảnh hưởng cả người học
			uids, err = q.TodayCourseMembers(ctx, c)
		} else {
			uids, err = q.TodayCourseStaff(ctx, c)
		}
		if err != nil {
			return fmt.Errorf("today: người bị ảnh hưởng: %w", err)
		}
		for _, u := range uids {
			users[u] = true
		}
		if m.Topic == TopicChanged || m.Topic == TopicAssigned {
			admins, err := q.TodayAdminIDs(ctx)
			if err != nil {
				return fmt.Errorf("today: admin: %w", err)
			}
			for _, u := range admins {
				users[u] = true
			}
		}
	}
	var keys []string
	for u := range users {
		keys = append(keys, CacheKey(u, "all"))
		if course != uuid.Nil {
			keys = append(keys, CacheKey(u, course.String()))
		}
	}
	if m.Topic == TopicUserVerified {
		for u := range users { // xác minh email đổi `Viewer` ở MỌI lớp của người đó
			cs, err := q.TodayUserCourses(ctx, u)
			if err != nil {
				return fmt.Errorf("today: lớp của người dùng: %w", err)
			}
			for _, c := range cs {
				keys = append(keys, CacheKey(u, c.String()))
			}
		}
	}
	for len(keys) > 0 {
		n := min(len(keys), 500)
		if err := i.Redis.Del(ctx, keys[:n]...).Err(); err != nil {
			return fmt.Errorf("today: xoá cache: %w", err)
		}
		keys = keys[n:]
	}
	return nil
}

func isExamTopic(t string) bool {
	return t == TopicExamScheduled || t == TopicExamUnscheduled || t == TopicExamOpened || t == TopicExamClosed
}
