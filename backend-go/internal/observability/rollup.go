package observability

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/store"
)

// Bảng cộng gộp theo giờ của llm_audit (TC-01-13, P-11-44): tổng hợp 7 ngày không còn quét hàng triệu hàng thô.
// Số đếm / tổng (yêu cầu, lỗi, dự phòng, chi phí, token, PII) CHÍNH XÁC; p50 / p95 lấy từ histogram độ trễ nhóm hình học tỷ lệ 1,1 (sai số ≤ ~5 %).
const (
	latBuckets    = 140
	latRatio      = 1.1
	rollupGuard   = 2 * time.Minute // giờ h chỉ đóng khi now ≥ h+1h+guard: hàng audit đến trễ vài giây vẫn kịp vào
	rollupBackfil = 8 * 24 * time.Hour
	rollupKeep    = 35 * 24 * time.Hour
	rollupChunk   = 48 * time.Hour
)

// Rollup đóng các giờ đã qua vào llm_audit_hourly(_lat) và đẩy watermark. Idempotent: mỗi chunk xoá rồi ghi lại cả khoảng trong MỘT giao dịch
// (khoá advisory nên hai tiến trình không chen nhau); chạy lại hay bị ngắt giữa chừng đều ra cùng kết quả. Trả số giờ vừa đóng.
// Lần đầu: nạp 8 ngày gần nhất (đủ cho tổng hợp 7 ngày). Hàng audit đến trễ hơn rollupGuard sau khi giờ của nó đã đóng không vào bảng (chỉ lệch số liệu giờ đó).
func (s *Service) Rollup(ctx context.Context) (int64, error) {
	target := s.now().Add(-rollupGuard).Truncate(time.Hour)
	var closed int64
	for {
		n, done, err := s.rollupChunk(ctx, target)
		closed += n
		if err != nil {
			return closed, err
		}
		if done {
			break
		}
	}
	if closed > 0 {
		cut := s.now().Add(-rollupKeep)
		q := store.New(s.Pool)
		if err := q.ObsRollupPurge(ctx, cut); err != nil {
			return closed, fmt.Errorf("observability: dọn rollup: %w", err)
		}
		if err := q.ObsRollupPurgeLat(ctx, cut); err != nil {
			return closed, fmt.Errorf("observability: dọn rollup độ trễ: %w", err)
		}
	}
	return closed, nil
}

func (s *Service) rollupChunk(ctx context.Context, target time.Time) (hours int64, done bool, err error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return 0, false, fmt.Errorf("observability: mở giao dịch rollup: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	if err := q.ObsRollupLockState(ctx); err != nil {
		return 0, false, fmt.Errorf("observability: khoá rollup: %w", err)
	}
	wm, err := q.ObsRollupWatermark(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		wm = s.now().Add(-rollupBackfil).Truncate(time.Hour)
	} else if err != nil {
		return 0, false, fmt.Errorf("observability: đọc watermark: %w", err)
	}
	if !wm.Before(target) {
		return 0, true, nil
	}
	end := target
	if end.Sub(wm) > rollupChunk {
		end = wm.Add(rollupChunk)
	}
	if err := q.ObsRollupDeleteHours(ctx, store.ObsRollupDeleteHoursParams{FromTs: wm, ToTs: end}); err != nil {
		return 0, false, fmt.Errorf("observability: xoá giờ: %w", err)
	}
	if err := q.ObsRollupDeleteLat(ctx, store.ObsRollupDeleteLatParams{FromTs: wm, ToTs: end}); err != nil {
		return 0, false, fmt.Errorf("observability: xoá độ trễ: %w", err)
	}
	if err := q.ObsRollupInsertHours(ctx, store.ObsRollupInsertHoursParams{FromTs: wm, ToTs: end}); err != nil {
		return 0, false, fmt.Errorf("observability: ghi giờ: %w", err)
	}
	if err := q.ObsRollupInsertLat(ctx, store.ObsRollupInsertLatParams{FromTs: wm, ToTs: end}); err != nil {
		return 0, false, fmt.Errorf("observability: ghi độ trễ: %w", err)
	}
	if err := q.ObsRollupSetWatermark(ctx, end); err != nil {
		return 0, false, fmt.Errorf("observability: đặt watermark: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, false, fmt.Errorf("observability: commit rollup: %w", err)
	}
	return int64(end.Sub(wm) / time.Hour), !end.Before(target), nil
}

// bucketStats cộng dồn một nhóm hàng (cả cửa sổ, hoặc một ngày).
type bucketStats struct {
	requests, errors, fallbacks, tokensIn, tokensOut, pii int64
	cost                                                  decimal.Decimal
	lat                                                   [latBuckets]int64
}

func (b *bucketStats) add(r store.ObsRollupPartsRow) {
	b.requests += r.Requests
	b.errors += r.Errors
	b.fallbacks += r.Fallbacks
	b.tokensIn += r.TokensIn
	b.tokensOut += r.TokensOut
	b.pii += r.PiiMasked
	b.cost = b.cost.Add(r.Cost)
}

// quantile: phân vị q (0…1) của histogram, nội suy hình học trong nhóm; không có mẫu → 0.
func quantile(h *[latBuckets]int64, q float64) float64 {
	var total int64
	for _, n := range h {
		total += n
	}
	if total == 0 {
		return 0
	}
	rank := q * float64(total)
	var cum int64
	for b, n := range h {
		if n == 0 {
			continue
		}
		if float64(cum+n) >= rank {
			f := (rank - float64(cum)) / float64(n)
			return math.Pow(latRatio, float64(b)+math.Min(math.Max(f, 0), 1))
		}
		cum += n
	}
	return math.Pow(latRatio, latBuckets)
}

// rollupSummary đọc cửa sổ [from, to) từ bảng cộng gộp + phần lẻ thô: tổng cả cửa sổ, theo ngày (múi giờ VN), theo tác vụ.
type rollupSummary struct {
	all   bucketStats
	days  map[string]*bucketStats
	tasks map[string]*bucketStats
}

func (s *Service) rollupSummary(ctx context.Context, q *store.Queries, from, to time.Time, withBreakdown bool) (rollupSummary, error) {
	out := rollupSummary{days: map[string]*bucketStats{}, tasks: map[string]*bucketStats{}}
	// giờ nguyên đầu tiên ≥ from; hi = min(watermark, giờ nguyên ≤ to); chưa có watermark → không giờ đủ, đọc thô cả cửa sổ
	lo := from.Truncate(time.Hour)
	if lo.Before(from) {
		lo = lo.Add(time.Hour)
	}
	hi := lo
	if wm, err := q.ObsRollupWatermark(ctx); err == nil {
		if hi = minTime(wm, to.Truncate(time.Hour)); hi.Before(lo) {
			hi = lo
		}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return out, fmt.Errorf("observability: đọc watermark: %w", err)
	}
	headTo := minTime(lo, to)
	parts, err := q.ObsRollupParts(ctx, store.ObsRollupPartsParams{Lo: lo, Hi: hi, FromTs: from, HeadTo: headTo, ToTs: to})
	if err != nil {
		return out, fmt.Errorf("observability: đọc tổng hợp: %w", err)
	}
	lat, err := q.ObsRollupLatency(ctx, store.ObsRollupLatencyParams{Lo: lo, Hi: hi, FromTs: from, HeadTo: headTo, ToTs: to})
	if err != nil {
		return out, fmt.Errorf("observability: đọc histogram: %w", err)
	}
	loc := vnLocation()
	day := func(t time.Time) string { return t.In(loc).Format(time.DateOnly) }
	for _, r := range parts {
		out.all.add(r)
		if !withBreakdown {
			continue
		}
		d := out.days[day(r.Hour)]
		if d == nil {
			d = &bucketStats{}
			out.days[day(r.Hour)] = d
		}
		d.add(r)
		t := out.tasks[r.Task]
		if t == nil {
			t = &bucketStats{}
			out.tasks[r.Task] = t
		}
		t.add(r)
	}
	for _, r := range lat {
		if r.Bucket < 0 || int(r.Bucket) >= latBuckets {
			continue
		}
		out.all.lat[r.Bucket] += r.N
		if withBreakdown {
			d := out.days[day(r.Hour)]
			if d == nil {
				d = &bucketStats{}
				out.days[day(r.Hour)] = d
			}
			d.lat[r.Bucket] += r.N
		}
	}
	return out, nil
}

func sortedDaysDesc(m map[string]*bucketStats) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(ks)))
	return ks
}

func vnLocation() *time.Location {
	loc, err := time.LoadLocation(vnZone)
	if err != nil {
		return time.FixedZone("ICT", 7*3600)
	}
	return loc
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}
