package rag

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/platform/outbox"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
)

// VersionKey là `ep:rag:ver:{course_id}`: phiên bản tri thức của lớp, nằm trong khoá cache câu trả lời (`ep:ans:{course}:{ver}:{sha}`).
func VersionKey(courseID uuid.UUID) string { return appredis.Key("rag", "ver", courseID.String()) }

// Version trả phiên bản hiện tại (0 nếu chưa có).
func Version(ctx context.Context, rdb *appredis.Client, courseID uuid.UUID) (int64, error) {
	n, err := rdb.Get(ctx, VersionKey(courseID)).Int64()
	if err != nil && err.Error() != "redis: nil" {
		return 0, fmt.Errorf("rag: đọc phiên bản: %w", err)
	}
	return n, nil
}

// BumpVersion là outbox.Handler cho `document.changed` / `document.deleted`: tăng phiên bản của lớp trong payload nên cache câu trả lời cũ không còn trúng
// (không chờ TTL). INCR không có TTL theo thiết kế; idempotent theo nghĩa "tăng thêm không hại".
func BumpVersion(rdb *appredis.Client) outbox.Handler {
	return func(ctx context.Context, m outbox.Message) error {
		var p struct {
			CourseID string `json:"course_id"`
		}
		if err := json.Unmarshal(m.Payload, &p); err != nil {
			return fmt.Errorf("rag: payload %s: %w", m.Topic, err)
		}
		id, err := uuid.Parse(p.CourseID)
		if err != nil {
			return nil
		}
		if err := rdb.Incr(ctx, VersionKey(id)).Err(); err != nil {
			return fmt.Errorf("rag: tăng phiên bản: %w", err)
		}
		return nil
	}
}
