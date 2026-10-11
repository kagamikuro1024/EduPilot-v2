package chat

import (
	"context"
	"fmt"
	"time"

	"github.com/edupilot/backend-go/internal/store"
)

// Reap đánh dấu FAILED INTERRUPTED mọi tin STREAMING không cập nhật quá ReapAfter (gateway chết giữa lúc sinh), giữ partial_content,
// nhả CHAT_BUSY và báo người đang xem. Lệnh UPDATE có điều kiện STREAMING nên chạy lại không đụng tin còn sống đã hoàn tất.
func (s *Service) Reap(ctx context.Context) (int, error) {
	q := store.New(s.Pool)
	rows, err := q.ReapStaleChatMessages(ctx, s.now().Add(-s.c().ReapAfter))
	if err != nil {
		return 0, fmt.Errorf("chat: reaper: %w", err)
	}
	for _, r := range rows {
		m, err := s.reload(ctx, r.ID)
		if err != nil {
			continue
		}
		s.releaseBusyOf(ctx, r.UserID, m)
		s.emit(ctx, r.ID, r.Attempt, EvError, map[string]any{"code": "INTERRUPTED", "message": errorText("INTERRUPTED")})
	}
	return len(rows), nil
}

// ReapTask chạy Reap mỗi `Every` (mặc định 30 s) tới khi ctx huỷ. Không cần khoá leader: UPDATE có điều kiện nên nhiều bản chạy chung vô hại.
type ReapTask struct {
	Svc   *Service
	Every time.Duration
}

// Name là tên việc nền.
func (ReapTask) Name() string { return "chat.reaper" }

// Run lặp tới khi ctx huỷ.
func (t ReapTask) Run(ctx context.Context) error {
	every := t.Every
	if every <= 0 {
		every = 30 * time.Second
	}
	tick := time.NewTicker(every)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-tick.C:
			if n, err := t.Svc.Reap(ctx); err != nil && t.Svc.Log != nil {
				t.Svc.Log.ErrorContext(ctx, "chat: reaper lỗi", "error", err)
			} else if n > 0 && t.Svc.Log != nil {
				t.Svc.Log.WarnContext(ctx, "chat: đánh dấu tin bị gián đoạn", "count", n)
			}
		}
	}
}
