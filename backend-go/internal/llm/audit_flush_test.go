package llm

import (
	"context"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"
)

// FlushAudit (flush từ ngoài) phải chờ lô mà vòng nền ĐÃ lấy và đang ghi: không chờ thì trả về khi dòng chưa vào DB (flake TestCascadeThreadsHTTP path=FAST).
func TestAuditFlushWaitsForInFlightBatch(t *testing.T) {
	t.Parallel()
	var written atomic.Int64
	a := NewAuditor(context.Background(), func(_ context.Context, rows []AuditRow) error {
		time.Sleep(200 * time.Millisecond)
		written.Add(int64(len(rows)))
		return nil
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	defer a.Close(context.Background())
	for range auditBatch { // đủ lô → vòng nền lấy ngay
		a.Record(AuditRow{Task: "DECIDE", Status: "ok", TraceID: "t"})
	}
	for a.Pending() != 0 { // vòng nền đã lấy lô (đang ghi 200 ms)
		time.Sleep(time.Millisecond)
	}
	a.flush(context.Background())
	if got := written.Load(); got != int64(auditBatch) {
		t.Fatalf("flush trả về khi mới ghi %d / %d dòng", got, auditBatch)
	}
}
