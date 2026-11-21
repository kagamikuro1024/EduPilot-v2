package practice

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/edupilot/backend-go/internal/store"
)

// CloseJob chạy `practice.close` (FR-16): mỗi `Every` đóng các lượt MOCK quá hạn + dung sai. Không cần khoá leader: `PracticeDue` dùng FOR UPDATE SKIP LOCKED
// nên nhiều worker không đóng trùng, và `CloseDue` idempotent.
type CloseJob struct {
	Svc   *Service
	Every time.Duration
	Log   *slog.Logger
}

// Run chạy tới khi ctx bị huỷ.
func (j *CloseJob) Run(ctx context.Context) error {
	every := j.Every
	if every <= 0 {
		every = time.Minute
	}
	tk := time.NewTicker(every)
	defer tk.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-tk.C:
			n, err := j.Svc.CloseDue(ctx)
			if err != nil && j.Log != nil {
				j.Log.ErrorContext(ctx, "practice.close lỗi", "error", err.Error())
			} else if n > 0 && j.Log != nil {
				j.Log.InfoContext(ctx, "practice.close", "closed", n)
			}
		}
	}
}

// FormsKeepDays: tệp Forms gốc giữ tối đa 90 ngày (US-P9-11 AC8, Q-P9-7).
const FormsKeepDays = 90

// FormsBlob là phần của object storage mà job xoá tệp dùng.
type FormsBlob interface {
	Delete(ctx context.Context, key string) error
}

// FormsPurgeJob chạy `practice.forms_purge`: mỗi `Every` xoá tệp gốc của lô Forms quá `FormsKeepDays`, ghi `blob_purged_at` + `audit_log`.
// Idempotent và an toàn khi nhiều worker chạy cùng lúc (xoá khoá thiếu không lỗi; đánh dấu chỉ một lần).
type FormsPurgeJob struct {
	Svc   *Service
	Blob  FormsBlob
	Every time.Duration
	Log   *slog.Logger
}

// Once xoá một lượt tối đa 100 lô quá hạn; trả số lô đã xoá.
func (j *FormsPurgeJob) Once(ctx context.Context) (int, error) {
	q := store.New(j.Svc.Pool)
	rows, err := q.FormsBatchesExpired(ctx, store.FormsBatchesExpiredParams{KeepDays: FormsKeepDays, MaxRows: 100})
	if err != nil {
		return 0, fmt.Errorf("practice: lô Forms quá hạn: %w", err)
	}
	n := 0
	for _, r := range rows {
		if err := j.Blob.Delete(ctx, r.BlobKey); err != nil {
			return n, fmt.Errorf("practice: xoá tệp Forms: %w", err)
		}
		done, err := q.FormsBatchMarkPurged(ctx, store.FormsBatchMarkPurgedParams{CourseID: r.CourseID, ID: r.ID})
		if err != nil {
			return n, fmt.Errorf("practice: đánh dấu đã xoá tệp Forms: %w", err)
		}
		if done == 0 {
			continue // worker khác đã đánh dấu
		}
		if _, err := q.InsertAuditLog(ctx, store.InsertAuditLogParams{CourseID: &r.CourseID, Entity: "forms_imports", EntityID: r.ID.String(), Action: "purge_blob"}); err != nil {
			return n, fmt.Errorf("practice: audit xoá tệp Forms: %w", err)
		}
		n++
	}
	return n, nil
}

// Run chạy tới khi ctx bị huỷ.
func (j *FormsPurgeJob) Run(ctx context.Context) error {
	every := j.Every
	if every <= 0 {
		every = time.Hour
	}
	tk := time.NewTicker(every)
	defer tk.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-tk.C:
			n, err := j.Once(ctx)
			if err != nil && j.Log != nil {
				j.Log.ErrorContext(ctx, "practice.forms_purge lỗi", "error", err.Error())
			} else if n > 0 && j.Log != nil {
				j.Log.InfoContext(ctx, "practice.forms_purge", "purged", n)
			}
		}
	}
}
