package escalation

import (
	"slices"
	"time"
)

// ICT: Việt Nam không có giờ mùa hè nên múi cố định +7 đủ dùng, không phụ thuộc tzdata của image.
var ICT = time.FixedZone("ICT", 7*60*60) //nolint:gochecknoglobals // hằng số múi giờ, bất biến

// WorkHours là khung "giờ làm việc" để tính SLA (SRS 4.6.1): [Start, End) tính từ 00:00 giờ Việt Nam của mỗi ngày thuộc Days. Không có lịch nghỉ lễ (nợ).
type WorkHours struct {
	Start, End time.Duration
	Days       []time.Weekday
}

// DefaultWorkHours là khung mặc định (chủ dự án chốt Q4): cả ngày thứ Hai–thứ Sáu.
func DefaultWorkHours() WorkHours {
	return WorkHours{Start: 0, End: 24 * time.Hour, Days: []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday}}
}

const maxSearchDays = 400 // chặn vòng lặp khi khung rỗng / Days rỗng

// window trả khoảng làm việc của ngày chứa `day` (giờ VN), hoặc ok=false nếu ngày đó không tính.
func (w WorkHours) window(day time.Time) (from, to time.Time, ok bool) {
	d := day.In(ICT)
	if !slices.Contains(w.Days, d.Weekday()) {
		return time.Time{}, time.Time{}, false
	}
	mid := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, ICT)
	return mid.Add(w.Start), mid.Add(w.End), true
}

// Elapsed = Σ độ dài giao của [from, to) với khung làm việc của mọi ngày. from ≥ to → 0.
func (w WorkHours) Elapsed(from, to time.Time) time.Duration {
	if !from.Before(to) {
		return 0
	}
	var sum time.Duration
	f := from.In(ICT)
	day := time.Date(f.Year(), f.Month(), f.Day(), 0, 0, 0, 0, ICT)
	for ; day.Before(to); day = day.AddDate(0, 0, 1) {
		ws, we, ok := w.window(day)
		if !ok {
			continue
		}
		lo, hi := maxTime(ws, from), minTime(we, to)
		if hi.After(lo) {
			sum += hi.Sub(lo)
		}
	}
	return sum
}

// At là thời điểm đầu tiên t mà Elapsed(from, t) ≥ after. after ≤ 0 → from. Không tìm được trong ~400 ngày (khung rỗng) → zero time.
func (w WorkHours) At(from time.Time, after time.Duration) time.Time {
	if after <= 0 {
		return from
	}
	f := from.In(ICT)
	day := time.Date(f.Year(), f.Month(), f.Day(), 0, 0, 0, 0, ICT)
	left := after
	for i := 0; i < maxSearchDays; i++ {
		ws, we, ok := w.window(day)
		day = day.AddDate(0, 0, 1)
		if !ok {
			continue
		}
		lo := maxTime(ws, from)
		if !we.After(lo) {
			continue
		}
		if l := we.Sub(lo); l >= left {
			return lo.Add(left)
		} else {
			left -= l
		}
	}
	return time.Time{}
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

// SLA gắn khung với hai mốc (SRS 4.6): ReminderAt = nhắc cả nhóm, EscalateAt = lên đầu "Hôm nay" + báo Admin.
type SLA struct {
	Hours                      WorkHours
	RemindAfter, EscalateAfter time.Duration
}

// ReminderAt là thời điểm nhắc cả nhóm của ticket chờ từ `waitingSince`.
func (s SLA) ReminderAt(waitingSince time.Time) time.Time {
	return s.Hours.At(waitingSince, s.RemindAfter)
}

// EscalateAt là thời điểm quá hạn (lên đầu "Hôm nay").
func (s SLA) EscalateAt(waitingSince time.Time) time.Time {
	return s.Hours.At(waitingSince, s.EscalateAfter)
}
