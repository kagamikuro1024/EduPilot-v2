package observability

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	appredis "github.com/edupilot/backend-go/internal/platform/redis"

	"github.com/edupilot/backend-go/internal/notify"
	"github.com/edupilot/backend-go/internal/store"
)

// Ngưỡng cố định (US-P10-03): ≥ 20 yêu cầu trong 1 giờ và lỗi > 5 %.
const (
	alertWindow      = time.Hour
	alertMinRequests = 20
	alertErrorRate   = 0.05
	alertLink        = "/observability"
)

// Alerter tính cảnh báo cho ADMIN (chạy ở worker): idempotent — chạy lại cùng cửa sổ không thêm thông báo.
type Alerter struct {
	Svc    *Service
	Notify *notify.Service
	// Dedupe: không báo lại cùng loại + phạm vi trong khoảng này (mặc định 6 giờ).
	Dedupe time.Duration
}

// Tick tính cảnh báo một lần; trả số thông báo (loại × phạm vi) mới tạo.
func (a *Alerter) Tick(ctx context.Context) (int, error) {
	now := a.Svc.now()
	q := store.New(a.Svc.Pool)
	sent := 0
	w, err := q.ObsAlertErrorWindow(ctx, store.ObsAlertErrorWindowParams{FromTs: now.Add(-alertWindow), ToTs: now.Add(time.Second)})
	if err != nil {
		return sent, fmt.Errorf("observability: cửa sổ lỗi: %w", err)
	}
	if w.Requests >= alertMinRequests && rate(w.Errors, w.Requests) > alertErrorRate {
		body := fmt.Sprintf("%.0f%% yêu cầu lỗi trong 1 giờ qua (%d / %d).", 100*rate(w.Errors, w.Requests), w.Errors, w.Requests)
		n, err := a.fire(ctx, q, notify.TypeLLMErrorRate, "system", "Tỷ lệ lỗi AI đang cao", body, alertLink+"?filter=problem")
		if err != nil {
			return sent, err
		}
		sent += n
	}
	start := dayStart(now)
	bs, err := q.ObsAlertBudgetUsage(ctx, store.ObsAlertBudgetUsageParams{DayStart: start, DayEnd: start.AddDate(0, 0, 1)})
	if err != nil {
		return sent, fmt.Errorf("observability: chi phí ngày: %w", err)
	}
	for _, b := range bs {
		if !b.DailyLimit.Valid || b.Spent.Cmp(b.DailyLimit.Decimal) < 0 {
			continue
		}
		scope, label := "system", "toàn hệ thống"
		if b.Scope == "course" && b.CourseID != nil {
			scope, label = b.CourseID.String(), "lớp "+b.CourseCode
		}
		n, err := a.fire(ctx, q, notify.TypeLLMBudgetReached, scope, "Chi phí AI hôm nay chạm hạn mức", "Phạm vi: "+label+".", alertLink)
		if err != nil {
			return sent, err
		}
		sent += n
	}
	return sent, nil
}

func (a *Alerter) fire(ctx context.Context, q *store.Queries, typ, scope, title, body, link string) (int, error) {
	dedupe := a.Dedupe
	if dedupe <= 0 {
		dedupe = 6 * time.Hour
	}
	now := a.Svc.now()
	prefix := "obs:" + typ + ":" + scope + ":"
	recent, err := q.ObsAlertRecent(ctx, store.ObsAlertRecentParams{Type: typ, Prefix: prefix, Since: now.Add(-dedupe)})
	if err != nil {
		return 0, fmt.Errorf("observability: kiểm trùng cảnh báo: %w", err)
	}
	if recent {
		return 0, nil
	}
	admins, err := q.ObsActiveAdmins(ctx)
	if err != nil {
		return 0, fmt.Errorf("observability: danh sách ADMIN: %w", err)
	}
	n, err := a.Notify.SendMany(ctx, admins, notify.Spec{Type: typ, Title: title, Body: &body, Link: &link, DedupeKey: fmt.Sprintf("%s%d", prefix, now.Unix())})
	if err != nil {
		return 0, fmt.Errorf("observability: gửi cảnh báo: %w", err)
	}
	if n > 0 {
		return 1, nil
	}
	return 0, nil
}

// PurgePayloads xoá bản prompt cũ hơn `days` ngày (llm_audit giữ nguyên); trả số dòng đã xoá.
func (s *Service) PurgePayloads(ctx context.Context, days int) (int64, error) {
	n, err := store.New(s.Pool).ObsPayloadsPurge(ctx, s.now().AddDate(0, 0, -days))
	if err != nil {
		return 0, fmt.Errorf("observability: dọn payload: %w", err)
	}
	return n, nil
}

// Job chạy fn mỗi `Every`; một bản nhờ khoá leader Redis (như submission.OrphanJob).
type Job struct {
	Name   string // tên tác vụ (log + khoá)
	Holder string // tên tiến trình giữ khoá leader
	Redis  *appredis.Client
	Every  time.Duration
	Fn     func(ctx context.Context) (int64, error)
	Log    *slog.Logger
}

// Run chạy tới khi ctx huỷ.
func (j *Job) Run(ctx context.Context) error {
	tk := time.NewTicker(j.Every)
	defer tk.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-tk.C:
			if j.Redis != nil && !j.Redis.Leader(ctx, "ep:leader:"+j.Name, j.Holder, 2*j.Every) {
				continue
			}
			n, err := j.Fn(ctx)
			switch {
			case err != nil && j.Log != nil:
				j.Log.ErrorContext(ctx, j.Name+" lỗi", "error", err.Error())
			case n > 0 && j.Log != nil:
				j.Log.InfoContext(ctx, j.Name, "n", n)
			}
		}
	}
}
