package decision

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/edupilot/backend-go/internal/notify"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/store"
)

// Ngưỡng cố định (US-DM-07 AC4): ≥ 20 lời gọi của một cổng trong 1 giờ và FAIL_CLOSED > 5 %.
const (
	alertWindow      = time.Hour
	alertMinCalls    = 20
	alertFailClosed  = 0.05
	alertLink        = "/observability"
	defaultAlertEach = 5 * time.Minute
	defaultDedupe    = 6 * time.Hour
)

// Alerter tính cảnh báo `DECISION_FAIL_CLOSED` cho mọi ADMIN (chạy ở worker, một bản nhờ khoá leader Redis). Idempotent: chạy lại trong khoảng khử trùng không thêm thông báo.
// Hàng provider «Tạm dừng» (AC5) KHÔNG đi qua đây: nó tính lúc đọc (Service.provider), không thông báo riêng.
type Alerter struct {
	Pool   *pgxpool.Pool
	Notify *notify.Service
	Redis  *appredis.Client
	Name   string        // tên tiến trình giữ khoá leader
	Every  time.Duration // chu kỳ (mặc định 5 phút)
	Dedupe time.Duration // khử trùng theo cổng (mặc định 6 giờ)
	Log    *slog.Logger
	Now    func() time.Time
}

func (a *Alerter) now() time.Time {
	if a.Now != nil {
		return a.Now()
	}
	return time.Now()
}

// Tick tính một lần; trả số cổng vừa báo.
func (a *Alerter) Tick(ctx context.Context) (int, error) {
	now := a.now()
	q := store.New(a.Pool)
	rows, err := q.DecisionObservability(ctx, now.Add(-alertWindow))
	if err != nil {
		return 0, fmt.Errorf("decision: cửa sổ cảnh báo: %w", err)
	}
	dedupe := a.Dedupe
	if dedupe <= 0 {
		dedupe = defaultDedupe
	}
	sent := 0
	for _, r := range rows {
		if r.Calls < alertMinCalls || float64(r.FailClosed) <= alertFailClosed*float64(r.Calls) {
			continue
		}
		prefix := "dec:" + notify.TypeDecisionFailClosed + ":" + r.Gate
		recent, err := q.DecisionAlertRecent(ctx, store.DecisionAlertRecentParams{Type: notify.TypeDecisionFailClosed, Prefix: prefix, CreatedAt: now.Add(-dedupe)})
		if err != nil {
			return sent, fmt.Errorf("decision: kiểm trùng cảnh báo: %w", err)
		}
		if recent {
			continue
		}
		admins, err := q.DecisionActiveAdmins(ctx)
		if err != nil {
			return sent, fmt.Errorf("decision: danh sách ADMIN: %w", err)
		}
		body := fmt.Sprintf("%.0f%% lời gọi bị chặn an toàn trong 1 giờ qua (%d / %d).", 100*float64(r.FailClosed)/float64(r.Calls), r.FailClosed, r.Calls)
		link := alertLink
		n, err := a.Notify.SendMany(ctx, admins, notify.Spec{Type: notify.TypeDecisionFailClosed, Title: "Quyết định nhanh: " + GateLabel(r.Gate) + " đang chặn nhiều", Body: &body, Link: &link,
			DedupeKey: fmt.Sprintf("%s:%d", prefix, now.Unix())})
		if err != nil {
			return sent, fmt.Errorf("decision: gửi cảnh báo: %w", err)
		}
		if n > 0 {
			sent++
		}
	}
	return sent, nil
}

// GateLabel là tên cổng bằng chữ cho người dùng (SRS FEAT-decision-model 7.1).
func GateLabel(gate string) string {
	switch gate {
	case "THREADS_PRECHECK":
		return "Kiểm duyệt Threads"
	case "INJECTION_FLAG":
		return "Cờ chèn lệnh khi chấm"
	case "CHAT_ROUTING":
		return "Định tuyến câu hỏi chat"
	}
	return gate
}

// Run chạy tới khi ctx huỷ.
func (a *Alerter) Run(ctx context.Context) error {
	every := a.Every
	if every <= 0 {
		every = defaultAlertEach
	}
	tk := time.NewTicker(every)
	defer tk.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-tk.C:
			if a.Redis != nil && !a.Redis.Leader(ctx, "ep:leader:decision.alert", a.Name, 2*every) {
				continue
			}
			n, err := a.Tick(ctx)
			switch {
			case err != nil && a.Log != nil:
				a.Log.ErrorContext(ctx, "decision.alert lỗi", "error", err.Error())
			case n > 0 && a.Log != nil:
				a.Log.InfoContext(ctx, "decision.alert", "n", n)
			}
		}
	}
}
