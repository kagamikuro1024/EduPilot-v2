package activity_test

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	appredis "github.com/edupilot/backend-go/internal/platform/redis"
)

// US-P5-03 AC6 / AC10: heartbeat.rollup — ghi TUYỆT ĐỐI (GREATEST), idempotent, chịu crash / mất Redis.
// Redis và `ep:study:dirty` dùng chung giữa các test: mỗi test chỉ khẳng định về sinh viên của chính nó.

func TestRollupAbsoluteGreatest(t *testing.T) {
	r := newRig(t)
	w := r.world(true)
	s := w.student(true)
	for range 3 {
		require.NoError(t, r.beat(s.u.ID, w.course, "CHAT"))
		r.next()
	}
	r.rollup()
	require.Equal(t, 90, r.dbSecs(w.course, s.u.ID))
	for range 1 {
		require.NoError(t, r.beat(s.u.ID, w.course, "CHAT"))
		r.next()
	}
	r.rollup()
	require.Equal(t, 120, r.dbSecs(w.course, s.u.ID), "ghi giá trị tích luỹ MỚI (120), không cộng dồn (210)")
	// DB đã có giá trị lớn hơn V: GREATEST giữ nguyên
	r.exec(`update study_time_daily set active_seconds = 500 where course_id = $1 and student_id = $2`, w.course, s.u.ID)
	require.NoError(t, r.beat(s.u.ID, w.course, "CHAT"))
	r.rollup()
	require.Equal(t, 500, r.dbSecs(w.course, s.u.ID))
}

func TestRollupIdempotentRedelivery(t *testing.T) {
	r := newRig(t)
	w := r.world(true)
	s := w.student(true)
	for range 4 {
		require.NoError(t, r.beat(s.u.ID, w.course, "THREADS"))
		r.next()
	}
	var wg sync.WaitGroup
	for range 4 { // giao đôi: nhiều bản cùng cuộn
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.rollup()
		}()
	}
	wg.Wait()
	r.rollup()
	require.Equal(t, 120, r.dbSecs(w.course, s.u.ID))
	require.Equal(t, 1, r.count(`select count(*) from study_time_daily where course_id = $1 and student_id = $2`, w.course, s.u.ID))
}

// TestRollupCrashSafe — chết sau khi ghi DB và trước khi cập nhật `flushed`: chạy lại cho cùng kết quả.
func TestRollupCrashSafe(t *testing.T) {
	r := newRig(t)
	w := r.world(true)
	s := w.student(true)
	for range 2 {
		require.NoError(t, r.beat(s.u.ID, w.course, "LIBRARY"))
		r.next()
	}
	r.rollup()
	require.NoError(t, r.rdb.HDel(t.Context(), appredis.Key("study", "flushed"), r.key(w.course, s.u.ID, "LIBRARY")).Err()) // "crash" trước khi đánh dấu
	r.rollup()
	require.Equal(t, 60, r.dbSecs(w.course, s.u.ID))
	require.Equal(t, 1, r.count(`select count(*) from study_time_daily where course_id = $1 and student_id = $2`, w.course, s.u.ID))
}

func TestRollupRedisLossNoDecrease(t *testing.T) {
	r := newRig(t)
	w := r.world(true)
	s := w.student(true)
	for range 3 {
		require.NoError(t, r.beat(s.u.ID, w.course, "CHAT"))
		r.next()
	}
	r.rollup()
	require.Equal(t, 90, r.dbSecs(w.course, s.u.ID))
	require.NoError(t, r.rdb.Del(t.Context(), r.key(w.course, s.u.ID, "CHAT")).Err()) // Redis khởi động lại không lưu
	r.rollup()
	require.Equal(t, 90, r.dbSecs(w.course, s.u.ID), "mất khoá: DB giữ giá trị cũ")
	require.NoError(t, r.beat(s.u.ID, w.course, "CHAT")) // đếm lại từ 30
	r.rollup()
	require.Equal(t, 90, r.dbSecs(w.course, s.u.ID), "không âm, không giảm (GREATEST)")
}

// TestRollupMidnightSplit — nhịp trước / sau 00:00 Asia/Ho_Chi_Minh tách đúng hai hàng (hai ngày).
func TestRollupMidnightSplit(t *testing.T) {
	r := newRig(t)
	w := r.world(true)
	s := w.student(true)
	ict := time.FixedZone("ICT", 7*3600)
	r.clk.Set(time.Date(2026, 10, 12, 23, 59, 30, 0, ict))
	require.NoError(t, r.beat(s.u.ID, w.course, "CHAT"))
	r.next() // 00:00:00 ngày 13
	require.NoError(t, r.beat(s.u.ID, w.course, "CHAT"))
	r.rollup()
	var d1, d2 int
	require.NoError(t, r.pool.QueryRow(t.Context(), `select active_seconds from study_time_daily where course_id = $1 and student_id = $2 and date = '2026-10-12'`, w.course, s.u.ID).Scan(&d1))
	require.NoError(t, r.pool.QueryRow(t.Context(), `select active_seconds from study_time_daily where course_id = $1 and student_id = $2 and date = '2026-10-13'`, w.course, s.u.ID).Scan(&d2))
	require.Equal(t, []int{30, 30}, []int{d1, d2})
}

// TestRollupPrunesFlushed — khoá của ngày đã qua > 1 ngày: ghi lần cuối rồi SREM khỏi dirty và HDEL ở flushed (hash không phình mãi).
func TestRollupPrunesFlushed(t *testing.T) {
	r := newRig(t)
	w := r.world(true)
	s := w.student(true)
	ict := time.FixedZone("ICT", 7*3600)
	r.clk.Set(time.Date(2026, 9, 1, 10, 0, 0, 0, ict))
	require.NoError(t, r.beat(s.u.ID, w.course, "CHAT"))
	k := r.key(w.course, s.u.ID, "CHAT")
	r.rollup()
	require.Equal(t, 30, r.dbSecs(w.course, s.u.ID), "ghi lần cuối cho ngày cũ")
	require.True(t, r.rdb.SIsMember(t.Context(), appredis.Key("study", "dirty"), k).Val(), "ngày hôm nay (chưa quá 1 ngày): còn trong dirty")
	r.clk.Set(time.Date(2026, 9, 3, 10, 0, 0, 0, ict)) // hai ngày sau
	r.rollup()
	require.False(t, r.rdb.SIsMember(t.Context(), appredis.Key("study", "dirty"), k).Val(), "đã SREM")
	require.False(t, r.rdb.HExists(t.Context(), appredis.Key("study", "flushed"), k).Val(), "đã HDEL")
	require.Equal(t, 30, r.dbSecs(w.course, s.u.ID), "dữ liệu giữ nguyên")
}
