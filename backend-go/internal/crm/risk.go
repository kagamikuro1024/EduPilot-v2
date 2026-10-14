package crm

import (
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// Mã mức và mã tín hiệu của at-risk (SRS FEAT-crm-360 4.7). Chuỗi khớp enum `risk_level` và mã lý do ghi ở `student_risk.reasons`.
const (
	LevelNone   = "NONE"
	LevelWatch  = "WATCH"
	LevelAtRisk = "AT_RISK"

	SignalAttendance = "ATTENDANCE"
	SignalActivity   = "ACTIVITY"
	SignalStudy      = "STUDY"

	ReasonAttendanceLow = "ATTENDANCE_LOW"
	ReasonStudyLow      = "STUDY_LOW"
	ReasonActivityLow   = "ACTIVITY_LOW"
)

// Thresholds là ngưỡng theo lớp (`courses.settings.risk`, SRS 4.7.3). Tỷ lệ chuyên cần là decimal (luật 5: không float64 cho số liệu chấm / ngưỡng).
type Thresholds struct {
	AttendanceOKMin    decimal.Decimal
	ActivityMinPerWeek int
	StudyMinMinutes    int
	SignalsRequired    int
	AbsentAlert        int
}

// DefaultThresholds = RISK_DEFAULTS: 0,70 / 1 / 60 / 2 / 3.
func DefaultThresholds() Thresholds {
	return Thresholds{AttendanceOKMin: decimal.New(70, -2), ActivityMinPerWeek: 1, StudyMinMinutes: 60, SignalsRequired: 2, AbsentAlert: 3}
}

// Attendance của một tuần: số buổi đã ghi, số buổi tính là có mặt (PRESENT+LATE+EXCUSED) và số buổi ABSENT không phép.
type Attendance struct{ Recorded, OK, Absent int }

// Week là ba tín hiệu của một tuần trọn. Attendance == nil: tuần không có buổi nào được ghi cho sinh viên; StudyMinutes == nil: không đo (chưa xác nhận thông báo / đã tắt đo).
type Week struct {
	Start        time.Time
	Attendance   *Attendance
	Activity     int
	StudyMinutes *int
}

// WeekResult là một tuần sau khi chấm: Low liệt kê tín hiệu thấp theo thứ tự ATTENDANCE, STUDY, ACTIVITY; Weak theo SRS 4.7.2.
type WeekResult struct {
	Week
	Weak bool
	Low  []string
}

// Reason là một lý do có dữ liệu (`student_risk.reasons[]`): giao diện / câu nhận định chỉ dựng từ đây.
type Reason struct {
	Code   string         `json:"code"`
	Values map[string]int `json:"values"`
}

// Result của Evaluate. Weeks cùng thứ tự đầu vào (mới nhất trước).
type Result struct {
	Level   string
	Reasons []Reason
	Weeks   []WeekResult
}

// Evaluate (hàm thuần, SRS 4.7.2): tuần yếu = có ít nhất `SignalsRequired` tín hiệu CÓ dữ liệu và thấp, VÀ tín hiệu neo (chuyên cần) thấp. weeks[0] = W1 (tuần trọn gần nhất),
// weeks[1] = W2. AT_RISK khi cả hai yếu; WATCH khi chỉ W1 yếu; còn lại NONE. Tuần thiếu (len < 2) không yếu.
func Evaluate(weeks []Week, th Thresholds) Result {
	res := Result{Level: LevelNone, Reasons: []Reason{}, Weeks: make([]WeekResult, len(weeks))}
	for i, w := range weeks {
		res.Weeks[i] = score(w, th)
	}
	weak := func(i int) bool { return i < len(res.Weeks) && res.Weeks[i].Weak }
	var used []WeekResult
	switch {
	case weak(0) && weak(1):
		res.Level, used = LevelAtRisk, res.Weeks[:2]
	case weak(0):
		res.Level, used = LevelWatch, res.Weeks[:1]
	default:
		return res
	}
	res.Reasons = reasons(used)
	return res
}

func score(w Week, th Thresholds) WeekResult {
	r := WeekResult{Week: w, Low: []string{}}
	data := 0
	var anchor bool
	if a := w.Attendance; a != nil && a.Recorded > 0 {
		data++
		// ok/recorded < min ⇔ ok < min × recorded (không chia, không làm tròn).
		if decimal.NewFromInt(int64(a.OK)).LessThan(th.AttendanceOKMin.Mul(decimal.NewFromInt(int64(a.Recorded)))) {
			r.Low, anchor = append(r.Low, SignalAttendance), true
		}
	}
	if w.StudyMinutes != nil {
		data++
		if *w.StudyMinutes < th.StudyMinMinutes {
			r.Low = append(r.Low, SignalStudy)
		}
	}
	data++ // hoạt động luôn có dữ liệu
	if w.Activity < th.ActivityMinPerWeek {
		r.Low = append(r.Low, SignalActivity)
	}
	r.Weak = anchor && len(r.Low) >= th.SignalsRequired && data >= th.SignalsRequired
	return r
}

func reasons(used []WeekResult) []Reason {
	out := []Reason{}
	has := func(w WeekResult, s string) bool {
		for _, l := range w.Low {
			if l == s {
				return true
			}
		}
		return false
	}
	var abs, rec, attK, studySum, studyK, actK int
	for _, w := range used {
		if has(w, SignalAttendance) {
			attK++
			abs += w.Attendance.Absent
			rec += w.Attendance.Recorded
		}
		if has(w, SignalStudy) {
			studyK++
			studySum += *w.StudyMinutes
		}
		if has(w, SignalActivity) {
			actK++
		}
	}
	if attK > 0 {
		out = append(out, Reason{Code: ReasonAttendanceLow, Values: map[string]int{"absent": abs, "recorded": rec, "weeks": len(used)}})
	}
	if studyK > 0 {
		out = append(out, Reason{Code: ReasonStudyLow, Values: map[string]int{"minutes": studySum / studyK}})
	}
	if actK > 0 {
		out = append(out, Reason{Code: ReasonActivityLow, Values: map[string]int{"weeks": actK}})
	}
	return out
}

// Sentence dựng câu nhận định tiếng Việt từ MỘT dữ liệu (reasons đã lưu), SRS 4.7.5: không tên người, không điểm, không nội dung chat. NONE → "".
func Sentence(level string, rs []Reason) string {
	var prefix string
	switch level {
	case LevelAtRisk:
		prefix = "Cần chú ý: "
	case LevelWatch:
		prefix = "Đang theo dõi: "
	default:
		return ""
	}
	order := []string{ReasonAttendanceLow, ReasonStudyLow, ReasonActivityLow}
	var parts []string
	for _, code := range order {
		for _, r := range rs {
			if r.Code != code {
				continue
			}
			switch code {
			case ReasonAttendanceLow:
				parts = append(parts, fmt.Sprintf("vắng %d/%d buổi trong %d tuần gần nhất", r.Values["absent"], r.Values["recorded"], r.Values["weeks"]))
			case ReasonStudyLow:
				parts = append(parts, fmt.Sprintf("học trên EduPilot khoảng %d phút mỗi tuần", r.Values["minutes"]))
			case ReasonActivityLow:
				parts = append(parts, fmt.Sprintf("không hỏi AI hay đăng bài ở Threads trong %d tuần", r.Values["weeks"]))
			}
		}
	}
	if len(parts) == 0 {
		return prefix + "chưa đủ dữ liệu."
	}
	return prefix + strings.Join(parts, "; ") + "."
}
