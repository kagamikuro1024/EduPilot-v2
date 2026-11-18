package llm_test

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/edupilot/backend-go/internal/llm"
)

// QC105 (TC-D01-18): Decide khi chưa có provider/khoá: not_configured trong < 1 ms (đo), không treo, đúng MỘT dòng llm_audit mỗi lần gọi.
func TestQC105NotConfiguredFast(t *testing.T) {
	g, c, ctx := decideGW(t, nil, "")
	const n = 2000
	ds := make([]time.Duration, 0, n)
	for range n {
		s := time.Now()
		_, err := g.Decide(ctx, req(llm.GateThreads, noulQ))
		ds = append(ds, time.Since(s))
		if err == nil {
			t.Fatal("muốn ErrNotConfigured")
		}
	}
	sort.Slice(ds, func(i, j int) bool { return ds[i] < ds[j] })
	t.Logf("n=%d min=%v p50=%v p99=%v max=%v", n, ds[0], ds[n/2], ds[n*99/100], ds[n-1])
	if ds[n/2] >= time.Millisecond || ds[n*99/100] >= time.Millisecond {
		t.Fatalf("p50=%v p99=%v ≥ 1ms", ds[n/2], ds[n*99/100])
	}
	rows := decRows(t, g, c)
	if len(rows) != n {
		t.Fatalf("%d dòng audit cho %d lần gọi", len(rows), n)
	}
	for _, r := range rows {
		if r.Status != "not_configured" {
			t.Fatalf("status %q", r.Status)
		}
	}
}

// QC105 (TC-D01-21): ALLOW_REAL, trần ngày 3 (UTC+7), máy chủ giả: 5 lời gọi → 3 đầu qua, 4–5 budget_blocked → cascade đi fallback; sang ngày (UTC+7) đếm lại.
func TestQC105RealDailyCap3(t *testing.T) {
	d := &dstub{}
	clock := time.Date(2026, 11, 19, 10, 0, 0, 0, time.UTC) // 17:00 UTC+7
	g, c, ctx := decideGW(t, d, llm.RealDecisionModel, func(o *llm.Options) { o.RealDailyCap = 3; o.Now = func() time.Time { return clock } })
	var st []string
	for range 5 {
		_, err := g.Decide(ctx, req(llm.GateThreads, noulQ))
		st = append(st, map[bool]string{true: "ok", false: "blocked"}[err == nil])
	}
	t.Logf("lượt 1–5: %v · provider gọi %d lần", st, d.calls.Load())
	if d.calls.Load() != 3 || st[2] != "ok" || st[3] != "blocked" || st[4] != "blocked" {
		t.Fatalf("%v calls=%d", st, d.calls.Load())
	}
	rows := decRows(t, g, c)
	if rows[3].Status != "budget_blocked" || rows[4].Status != "budget_blocked" {
		t.Fatalf("rows=%+v", rows)
	}
	// cascade: lời thứ 6 cùng ngày → budget_blocked → FALLBACK (chạy fallback, không FAIL_CLOSED)
	fb := 0
	_, path := g.Cascade(ctx, llm.GateThreads, req(llm.GateThreads, noulQ), llm.AddOnly{QID: "q"}, func(context.Context) (llm.Verdict, error) { fb++; return llm.Verdict{}, nil }, llm.Verdict{Yes: true})
	t.Logf("cascade lời 6 cùng ngày: path=%s fallback chạy %d lần", path, fb)
	if path != llm.PathFallback || fb != 1 {
		t.Fatalf("path=%s fb=%d", path, fb)
	}
	// 16:59:59 UTC (23:59:59 UTC+7) vẫn cùng ngày → còn chặn; 17:00:00 UTC (00:00 UTC+7 hôm sau) → đếm lại
	clock = time.Date(2026, 11, 19, 16, 59, 59, 0, time.UTC)
	if _, err := g.Decide(ctx, req(llm.GateThreads, noulQ)); err == nil {
		t.Fatal("23:59:59 UTC+7 phải còn bị chặn")
	}
	clock = time.Date(2026, 11, 19, 17, 0, 0, 0, time.UTC)
	for i := range 3 {
		if _, err := g.Decide(ctx, req(llm.GateThreads, noulQ)); err != nil {
			t.Fatalf("sang ngày UTC+7: lần %d err=%v", i+1, err)
		}
	}
	if _, err := g.Decide(ctx, req(llm.GateThreads, noulQ)); err == nil {
		t.Fatal("ngày mới: lời thứ 4 phải bị chặn")
	}
	t.Logf("sang 00:00 UTC+7: 3 lời qua, lời 4 chặn · provider gọi tổng %d", d.calls.Load())
}
