package crm_test

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/crm"
)

// wk dựng một tuần: a < 0 ⇒ không có dữ liệu chuyên cần (n/a); s < 0 ⇒ không đo thời gian học. Tỷ lệ a (0..1) trên 4 buổi ghi.
func wk(a float64, act, s int) crm.Week {
	w := crm.Week{Activity: act}
	if a >= 0 {
		ok := int(a*4 + 0.5)
		w.Attendance = &crm.Attendance{Recorded: 4, OK: ok, Absent: 4 - ok}
	}
	if s >= 0 {
		w.StudyMinutes = &s
	}
	return w
}

// TestRiskTable: bảng 16 ca của SRS 9.3 (W1 mới hơn: weeks[0]). Mặc định 0,70 / 1 / 60 / 2.
func TestRiskTable(t *testing.T) {
	t.Parallel()
	def := crm.DefaultThresholds()
	req := func(n int) crm.Thresholds { th := def; th.SignalsRequired = n; return th }
	cases := []struct {
		name   string
		w2, w1 crm.Week
		th     crm.Thresholds
		want   string
	}{
		{"1 hai tuần liền yếu", wk(.5, 0, 30), wk(.5, 0, 30), def, crm.LevelAtRisk},
		{"2 W2 không có neo thấp", wk(1, 0, 30), wk(.5, 0, 30), def, crm.LevelWatch},
		{"3 chỉ 1 tín hiệu thấp", wk(.5, 3, 200), wk(.5, 3, 200), def, crm.LevelNone},
		{"4 neo + hoạt động", wk(.5, 0, 200), wk(.5, 0, 200), def, crm.LevelAtRisk},
		{"5 không có neo", wk(-1, 0, 30), wk(-1, 0, 30), def, crm.LevelNone},
		{"6 W2 không dữ liệu chuyên cần", wk(-1, 0, 30), wk(.5, 0, 30), def, crm.LevelWatch},
		{"7 tuần nghỉ cắt chuỗi", wk(.5, 0, 30), wk(-1, 0, 30), def, crm.LevelNone},
		{"8 chưa ack: neo + hoạt động", wk(.5, 0, -1), wk(.5, 0, -1), def, crm.LevelAtRisk},
		{"9 chỉ 1 thấp", wk(.5, 2, -1), wk(.5, 2, -1), def, crm.LevelNone},
		{"10 đúng ngưỡng không phải thấp", wk(.7, 1, 60), wk(.7, 1, 60), def, crm.LevelNone},
		{"11 signals_required=1", wk(.5, 5, 200), wk(.5, 5, 200), req(1), crm.LevelAtRisk},
		{"12 signals_required=3", wk(.5, 0, -1), wk(.5, 0, -1), req(3), crm.LevelNone},
		{"13 vào lớp giữa kỳ", crm.Week{}, wk(.5, 0, 30), def, crm.LevelWatch},
		{"14 vắng có phép", wk(1, 0, 30), wk(1, 0, 30), def, crm.LevelNone},
		{"15 STUDY=null vẫn AT_RISK", wk(.5, 0, -1), wk(.5, 0, -1), def, crm.LevelAtRisk},
		{"16 STUDY=null không thành tín hiệu thứ hai", wk(.5, 5, -1), wk(.5, 5, -1), def, crm.LevelNone},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, c.want, crm.Evaluate([]crm.Week{c.w1, c.w2}, c.th).Level)
		})
	}
}

// TestRiskAnchorRequired: thời gian học + hoạt động thấp nhưng chuyên cần tốt (hoặc không có) → không bao giờ yếu, dù đủ số tín hiệu.
func TestRiskAnchorRequired(t *testing.T) {
	t.Parallel()
	for _, a := range []float64{1, -1} {
		r := crm.Evaluate([]crm.Week{wk(a, 0, 0), wk(a, 0, 0)}, crm.DefaultThresholds())
		require.Equal(t, crm.LevelNone, r.Level)
		require.False(t, r.Weeks[0].Weak)
		require.Empty(t, r.Reasons)
	}
}

// TestRiskInsufficientDataNotWeak: không đủ hai tuần (lớp mới) hoặc thiếu dữ liệu → không AT_RISK; một tuần yếu duy nhất → WATCH.
func TestRiskInsufficientDataNotWeak(t *testing.T) {
	t.Parallel()
	th := crm.DefaultThresholds()
	require.Equal(t, crm.LevelNone, crm.Evaluate(nil, th).Level)
	require.Equal(t, crm.LevelNone, crm.Evaluate([]crm.Week{{}, {}}, th).Level)
	require.Equal(t, crm.LevelWatch, crm.Evaluate([]crm.Week{wk(.5, 0, 30)}, th).Level)
	// ngưỡng chuyên cần là decimal: 0,50 ⇒ 2/4 không thấp
	th.AttendanceOKMin = decimal.New(50, -2)
	require.Equal(t, crm.LevelNone, crm.Evaluate([]crm.Week{wk(.5, 0, 30), wk(.5, 0, 30)}, th).Level)
}

// TestRiskIdempotent: cùng dữ liệu → cùng kết quả (không phụ thuộc thời điểm chạy hay thứ tự gọi).
func TestRiskIdempotent(t *testing.T) {
	t.Parallel()
	in := []crm.Week{wk(.5, 0, 30), wk(.25, 0, 20)}
	a, b := crm.Evaluate(in, crm.DefaultThresholds()), crm.Evaluate(in, crm.DefaultThresholds())
	require.Equal(t, a, b)
}

// TestRiskSentenceFromData: câu dựng từ số liệu đã lưu, theo mẫu SRS 4.7.5; thứ tự chuyên cần → thời gian học → hoạt động.
func TestRiskSentenceFromData(t *testing.T) {
	t.Parallel()
	// 3/4 buổi vắng trong 2 tuần (mỗi tuần 2 buổi ghi, 1.5 vắng không phù hợp ⇒ dùng số nguyên): W1 vắng 2/2, W2 vắng 1/2.
	w1 := crm.Week{Attendance: &crm.Attendance{Recorded: 2, OK: 0, Absent: 2}, Activity: 2, StudyMinutes: ptr(30)}
	w2 := crm.Week{Attendance: &crm.Attendance{Recorded: 2, OK: 1, Absent: 1}, Activity: 2, StudyMinutes: ptr(10)}
	r := crm.Evaluate([]crm.Week{w1, w2}, crm.DefaultThresholds())
	require.Equal(t, crm.LevelAtRisk, r.Level)
	require.Equal(t, "Cần chú ý: vắng 3/4 buổi trong 2 tuần gần nhất; học trên EduPilot khoảng 20 phút mỗi tuần.", crm.Sentence(r.Level, r.Reasons))
	// hoạt động thấp + chuyên cần thấp, không đo thời gian
	r = crm.Evaluate([]crm.Week{wk(.5, 0, -1), wk(.5, 0, -1)}, crm.DefaultThresholds())
	require.Equal(t, "Cần chú ý: vắng 4/8 buổi trong 2 tuần gần nhất; không hỏi AI hay đăng bài ở Threads trong 2 tuần.", crm.Sentence(r.Level, r.Reasons))
	require.Empty(t, crm.Sentence(crm.LevelNone, nil))
}

// TestRiskSentenceWatchPrefix: WATCH mở đầu "Đang theo dõi:" và chỉ nêu tuần gần nhất.
func TestRiskSentenceWatchPrefix(t *testing.T) {
	t.Parallel()
	r := crm.Evaluate([]crm.Week{wk(.5, 0, 30), wk(1, 3, 100)}, crm.DefaultThresholds())
	require.Equal(t, crm.LevelWatch, r.Level)
	require.Equal(t, "Đang theo dõi: vắng 2/4 buổi trong 1 tuần gần nhất; học trên EduPilot khoảng 30 phút mỗi tuần; không hỏi AI hay đăng bài ở Threads trong 1 tuần.", crm.Sentence(r.Level, r.Reasons))
}

// TestRiskSentenceConsistentWithCounts: số trong câu = số của tuần yếu (danh sách / tab dùng cùng Weeks).
func TestRiskSentenceConsistentWithCounts(t *testing.T) {
	t.Parallel()
	r := crm.Evaluate([]crm.Week{wk(.25, 0, 5), wk(.5, 0, 15)}, crm.DefaultThresholds())
	var abs, rec int
	for _, w := range r.Weeks[:2] {
		abs += w.Attendance.Absent
		rec += w.Attendance.Recorded
	}
	for _, rs := range r.Reasons {
		if rs.Code == crm.ReasonAttendanceLow {
			require.Equal(t, abs, rs.Values["absent"])
			require.Equal(t, rec, rs.Values["recorded"])
		}
		if rs.Code == crm.ReasonStudyLow {
			require.Equal(t, 10, rs.Values["minutes"]) // (5+15)/2
		}
	}
}

func ptr(n int) *int { return &n }
