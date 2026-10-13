package escalation_test

import (
	"github.com/edupilot/backend-go/internal/workhours"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Tuần mẫu: thứ Hai 12/10/2026 (giờ Việt Nam). d(0) = T2, d(4) = T6, d(5) = T7, d(6) = CN, d(7) = T2 tuần sau.
func at(day, h, m int) time.Time {
	return time.Date(2026, 10, 12+day, h, m, 0, 0, workhours.ICT)
}

func sla(h workhours.Hours) workhours.SLA {
	return workhours.SLA{Hours: h, RemindAfter: 24 * time.Hour, EscalateAfter: 72 * time.Hour}
}

func TestWorkingElapsedTable(t *testing.T) {
	require.Equal(t, time.Monday, at(0, 0, 0).Weekday())
	w := workhours.Default()
	hrs := func(n float64) time.Duration { return time.Duration(n * float64(time.Hour)) }
	for _, c := range []struct {
		name     string
		from, to time.Time
		want     time.Duration
	}{
		{"T2 09:00 → T2 17:00", at(0, 9, 0), at(0, 17, 0), hrs(8)},
		{"T6 16:00 → T2 09:00", at(4, 16, 0), at(7, 9, 0), hrs(17)},
		{"T7 10:00 → CN 10:00", at(5, 10, 0), at(6, 10, 0), 0},
		{"T6 23:00 → T2 01:00", at(4, 23, 0), at(7, 1, 0), hrs(2)},
		{"T2 07:00 → T2 08:00", at(0, 7, 0), at(0, 8, 0), hrs(1)},
		{"T2 08:00 → T3 08:00", at(0, 8, 0), at(1, 8, 0), hrs(24)},
		{"T2 17:00 → T3 08:00", at(0, 17, 0), at(1, 8, 0), hrs(15)},
		{"cùng một thời điểm", at(2, 12, 0), at(2, 12, 0), 0},
		{"T5 08:00 → T3 tuần sau 17:00", at(3, 8, 0), at(8, 17, 0), hrs(81)},
		{"from > to", at(2, 12, 0), at(1, 12, 0), 0},
		{"cả cuối tuần", at(5, 0, 0), at(7, 0, 0), 0},
		{"qua nhiều tuần", at(0, 0, 0), at(14, 0, 0), hrs(240)},
		{"múi giờ khác (UTC) cùng thời điểm", at(0, 9, 0).UTC(), at(0, 17, 0).UTC(), hrs(8)},
	} {
		require.Equal(t, c.want, w.Elapsed(c.from, c.to), c.name)
	}
}

func TestReminderAtTable(t *testing.T) {
	s := sla(workhours.Default())
	require.True(t, at(1, 8, 0).Equal(s.ReminderAt(at(0, 8, 0))), "T2 08:00 → T3 08:00")
	require.True(t, at(7, 12, 0).Equal(s.ReminderAt(at(4, 12, 0))), "T6 12:00 → T2 12:00 (không nhắc T7 / CN)")
	require.True(t, at(1, 10, 0).Equal(s.ReminderAt(at(0, 10, 0))), "hỏi tiếp T2 10:00 → T3 10:00 (đồng hồ tính từ waiting_since)")
	require.True(t, at(8, 0, 0).Equal(s.ReminderAt(at(5, 10, 0))), "mở T7: đồng hồ bắt đầu T2 00:00 → hết T2 = T3 00:00 tuần sau")
}

func TestEscalateAtTable(t *testing.T) {
	s := sla(workhours.Default())
	require.True(t, at(3, 8, 0).Equal(s.EscalateAt(at(0, 8, 0))), "T2 08:00 → T5 08:00")
	require.True(t, at(9, 12, 0).Equal(s.EscalateAt(at(4, 12, 0))), "T6 12:00 → T4 12:00 tuần sau")
}

func TestWorkingHoursParametrised(t *testing.T) {
	w := workhours.Hours{Start: 8 * time.Hour, End: 17 * time.Hour, Days: workhours.Default().Days}
	s := sla(w)
	require.Equal(t, 8*time.Hour, w.Elapsed(at(0, 9, 0), at(0, 17, 0)))
	require.Equal(t, 2*time.Hour, w.Elapsed(at(4, 16, 0), at(7, 9, 0)), "T6 1h + T2 1h")
	require.Equal(t, 24*time.Hour, w.Elapsed(at(0, 8, 0), at(2, 14, 0)), "T2 9h + T3 9h + T4 6h")
	require.True(t, at(2, 14, 0).Equal(s.ReminderAt(at(0, 8, 0))), "ReminderAt(T2 08:00) = T4 14:00")
	require.True(t, at(9, 17, 0).Equal(s.EscalateAt(at(0, 8, 0))), "EscalateAt(T2 08:00) = T4 tuần sau 17:00")
	// khung kín: không bao giờ tới hạn
	require.True(t, workhours.Hours{Start: 0, End: time.Hour}.At(at(0, 0, 0), time.Hour).IsZero(), "Days rỗng → không có mốc")
}

// Thuộc tính: Elapsed không âm, đơn điệu theo `to`, cộng được (Elapsed(a,c) = Elapsed(a,b) + Elapsed(b,c)); At ngược của Elapsed.
func TestWorkingHoursProperty(t *testing.T) {
	rng := rand.New(rand.NewPCG(7, 11))
	for _, w := range []workhours.Hours{workhours.Default(), {Start: 8 * time.Hour, End: 17 * time.Hour, Days: workhours.Default().Days}, {Start: 9 * time.Hour, End: 12 * time.Hour, Days: []time.Weekday{time.Saturday, time.Sunday}}} {
		for range 300 {
			a := at(0, 0, 0).Add(time.Duration(rng.IntN(20*24*60)) * time.Minute)
			b := a.Add(time.Duration(rng.IntN(10*24*60)) * time.Minute)
			c := b.Add(time.Duration(rng.IntN(10*24*60)) * time.Minute)
			require.GreaterOrEqual(t, w.Elapsed(a, b), time.Duration(0))
			require.LessOrEqual(t, w.Elapsed(a, b), w.Elapsed(a, c), "đơn điệu")
			require.Equal(t, w.Elapsed(a, c), w.Elapsed(a, b)+w.Elapsed(b, c), "cộng được")
			after := time.Duration(1+rng.IntN(100)) * time.Hour
			m := w.At(a, after)
			require.False(t, m.IsZero())
			require.GreaterOrEqual(t, w.Elapsed(a, m), after, "tới mốc thì đủ giờ")
			require.Less(t, w.Elapsed(a, m.Add(-time.Minute)), after, "trước mốc 1 phút thì chưa đủ")
		}
	}
}
