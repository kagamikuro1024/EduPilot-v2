package exam

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/store"
)

// RecordChatBlocked ghi một `exam_events` CHAT_BLOCKED (không nội dung, meta `{}`) cho lượt IN_PROGRESS của đúng sinh viên (D56, US-P3-05 AC9).
// Khử trùng ≤ 1 dòng / lượt / phút bằng `SET ep:chat_blocked:{attempt} NX EX 60`; không tính vào hạn mức sự kiện của máy khách.
// Redis lỗi → vẫn ghi (thà dư một dòng còn hơn mất dấu vết).
func (l *Locker) RecordChatBlocked(ctx context.Context, userID, attemptID uuid.UUID) error {
	if l.Redis != nil {
		ok, err := l.Redis.SetNX(ctx, appredis.Key("chat_blocked", attemptID.String()), "1", time.Minute).Result()
		if err == nil && !ok {
			return nil
		}
	}
	if err := store.New(l.Pool).ExamChatBlockedInsert(ctx, store.ExamChatBlockedInsertParams{At: l.now(), AttemptID: attemptID, StudentID: userID}); err != nil {
		return fmt.Errorf("exam: ghi CHAT_BLOCKED: %w", err)
	}
	return nil
}
