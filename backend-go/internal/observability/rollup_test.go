package observability_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/observability"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/testutil"
)

// TC-01-13 / P-11-44: tổng hợp qua bảng cộng gộp theo giờ KHỚP đọc thô — số đếm / tổng chính xác, p50 / p95 trong sai số histogram.
func TestRollupSummaryMatchesRaw(t *testing.T) {
	testutil.RequireContainers(t)
	pool, err := pgxpool.New(t.Context(), testutil.MigratedPostgresURL(t))
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	now := time.Date(2026, 11, 16, 10, 20, 0, 0, time.UTC)
	clk := clock.NewFake(now)
	svc := &observability.Service{Pool: pool, Clock: clk}

	// 9 ngày × 24 giờ × 11 hàng: độ trễ trải 40 ms … 12 s, vài hàng lỗi / dự phòng, hai tác vụ.
	_, err = pool.Exec(t.Context(), `insert into llm_audit (task, lane, provider, model, tokens_in, tokens_out, latency_ms, fallback_index, cost_est, status, pii_masked_count, trace_id, created_at)
		select case when g % 3 = 0 then 'GRADING' else 'CHAT' end, 'INTERACTIVE', 'fake', 'm', 100 + g % 7, 10 + g % 5, 40 + ((g::bigint * 7919) % 12000)::int, case when g % 13 = 0 then 1 else 0 end,
		       (g % 9)::numeric / 100, case when g % 10 = 0 then 'error' when g % 17 = 0 then 'degraded' else 'ok' end, g % 3, 't' || g,
		       $1::timestamptz - (g * interval '5 minutes') - interval '3 seconds'
		  from generate_series(0, 9 * 24 * 11) g`, now)
	require.NoError(t, err)

	rawWin := func(w time.Duration) (n, errs, fb int64, cost string, pii int64, p50, p95 float64) {
		require.NoError(t, pool.QueryRow(t.Context(), `select count(*), count(*) filter (where status not in ('ok','degraded','cancelled')), count(*) filter (where fallback_index > 0),
			coalesce(sum(cost_est),0)::text, coalesce(sum(pii_masked_count),0), percentile_cont(0.5) within group (order by latency_ms), percentile_cont(0.95) within group (order by latency_ms)
			from llm_audit where created_at >= $1 and created_at < $2`, now.Add(-w), now.Add(time.Second)).Scan(&n, &errs, &fb, &cost, &pii, &p50, &p95))
		return
	}
	check := func(label string) {
		for _, win := range []struct {
			name string
			d    time.Duration
		}{{"24h", 24 * time.Hour}, {"7d", 7 * 24 * time.Hour}} {
			got, err := svc.Summary(t.Context(), win.name)
			require.NoError(t, err)
			n, errs, fb, cost, pii, p50, p95 := rawWin(win.d)
			tt := got.Totals
			require.Equal(t, n, tt.Requests, "%s %s: yêu cầu", label, win.name)
			require.Equal(t, errs, tt.Errors, "%s %s: lỗi", label, win.name)
			require.Equal(t, fb, tt.Fallbacks, "%s %s: dự phòng", label, win.name)
			require.Equal(t, pii, tt.PIIMasked, "%s %s: PII", label, win.name)
			require.InDelta(t, mustFloat(t, cost), mustFloat(t, tt.Cost), 1e-6, "%s %s: chi phí", label, win.name)
			require.InEpsilon(t, p50, tt.P50Ms, 0.06, "%s %s: p50", label, win.name)
			require.InEpsilon(t, p95, tt.P95Ms, 0.06, "%s %s: p95", label, win.name)
		}
		got, err := svc.Summary(t.Context(), "7d")
		require.NoError(t, err)
		var rawDays int
		require.NoError(t, pool.QueryRow(t.Context(), `select count(distinct ((created_at at time zone 'Asia/Ho_Chi_Minh')::date)) from llm_audit where created_at >= $1`, now.Add(-7*24*time.Hour-24*time.Hour)).Scan(&rawDays))
		var rawTasks int64
		require.NoError(t, pool.QueryRow(t.Context(), `select count(distinct task) from llm_audit where created_at >= $1`, now.Add(-7*24*time.Hour)).Scan(&rawTasks))
		require.EqualValues(t, rawTasks, len(got.Tasks), label)
		var sumDays int64
		for _, d := range got.Days {
			sumDays += d.Requests
		}
		var raw7 int64
		from7 := time.Date(2026, 11, 10, 0, 0, 0, 0, time.FixedZone("ICT", 7*3600)) // 00:00 VN của 6 ngày trước
		require.NoError(t, pool.QueryRow(t.Context(), `select count(*) from llm_audit where created_at >= $1 and created_at < $2`, from7, now.Add(time.Second)).Scan(&raw7))
		require.Equal(t, raw7, sumDays, "%s: tổng bảng 7 ngày", label)
	}

	check("chưa rollup (đọc thô)")
	n, err := svc.Rollup(t.Context())
	require.NoError(t, err)
	require.Positive(t, n)
	var hours int
	require.NoError(t, pool.QueryRow(t.Context(), `select count(distinct hour) from llm_audit_hourly`).Scan(&hours))
	require.Positive(t, hours, "giờ đủ đã vào bảng cộng gộp")
	check("sau rollup (giờ đủ + phần lẻ thô)")

	// Idempotent: chạy lại không đổi gì; không đóng thêm giờ.
	var before string
	require.NoError(t, pool.QueryRow(t.Context(), `select coalesce(sum(requests),0)::text || '/' || count(*) from llm_audit_hourly`).Scan(&before))
	n2, err := svc.Rollup(t.Context())
	require.NoError(t, err)
	require.Zero(t, n2)
	var after string
	require.NoError(t, pool.QueryRow(t.Context(), `select coalesce(sum(requests),0)::text || '/' || count(*) from llm_audit_hourly`).Scan(&after))
	require.Equal(t, before, after)

	// Thời gian trôi 3 giờ + hàng mới ở giờ hiện tại: giờ cũ đóng thêm, hàng mới vẫn vào tổng nhờ phần lẻ thô.
	clk.Advance(3 * time.Hour)
	now = now.Add(3 * time.Hour)
	_, err = pool.Exec(t.Context(), `insert into llm_audit (task, lane, provider, model, latency_ms, status, trace_id, created_at) select 'CHAT', 'INTERACTIVE', 'fake', 'm', 500, 'ok', 'new' || g, $1::timestamptz - (g * interval '1 minute') from generate_series(1, 40) g`, now)
	require.NoError(t, err)
	n3, err := svc.Rollup(t.Context())
	require.NoError(t, err)
	require.EqualValues(t, 3, n3)
	check("sau 3 giờ")

	// Giữ lượng nhỏ: bảng cộng gộp ≤ giờ × tác vụ, không phải số hàng thô.
	var rolled int
	require.NoError(t, pool.QueryRow(t.Context(), `select count(*) from llm_audit_hourly`).Scan(&rolled))
	require.Less(t, rolled, 9*24*3)
	fmt.Println("rollup rows:", rolled)
}

func mustFloat(t *testing.T, s string) float64 {
	t.Helper()
	var f float64
	_, err := fmt.Sscan(s, &f)
	require.NoError(t, err)
	return f
}
