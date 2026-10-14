package crm

import (
	"encoding/json"
	"fmt"

	"github.com/shopspring/decimal"
)

// TopicSettingsChanged: cài đặt lớp đổi (ngưỡng at-risk) → handler `crm.risk` chạy lại riêng lớp đó (SRS FEAT-crm-360 4.11).
const TopicSettingsChanged = "course.settings_changed"

// TopicRiskUpdated: một lượt `crm.risk` của lớp xong → "Hôm nay" của nhân viên làm mới (≤ 60 s).
const TopicRiskUpdated = "crm.risk_updated"

// RiskIn là thân `risk` của `PUT /courses/{id}/settings`: trường nil = giữ giá trị hiện có. Tỷ lệ là chuỗi decimal ("0.70"), không float.
type RiskIn struct {
	AttendanceOKMin    *string `json:"attendance_ok_min"`
	ActivityMinPerWeek *int    `json:"activity_min_per_week"`
	StudyMinMinutes    *int    `json:"study_min_minutes"`
	SignalsRequired    *int    `json:"signals_required"`
	AbsentAlert        *int    `json:"absent_alert"`
}

// RiskValues là ngưỡng hiệu lực (đã trộn mặc định) trả cho giao diện.
type RiskValues struct {
	AttendanceOKMin    string `json:"attendance_ok_min"`
	ActivityMinPerWeek int    `json:"activity_min_per_week"`
	StudyMinMinutes    int    `json:"study_min_minutes"`
	SignalsRequired    int    `json:"signals_required"`
	AbsentAlert        int    `json:"absent_alert"`
}

// Values đổi Thresholds sang dạng trả ra.
func (t Thresholds) Values() RiskValues {
	return RiskValues{AttendanceOKMin: t.AttendanceOKMin.StringFixed(2), ActivityMinPerWeek: t.ActivityMinPerWeek, StudyMinMinutes: t.StudyMinMinutes, SignalsRequired: t.SignalsRequired, AbsentAlert: t.AbsentAlert}
}

// stored là `courses.settings.risk`: chỉ chứa khoá đã đặt rõ.
type stored struct {
	Risk RiskIn `json:"risk"`
}

// ThresholdsFrom trộn `courses.settings.risk` lên RISK_DEFAULTS. Giá trị hỏng (không đáng xảy ra: ghi qua MergeRisk) bị bỏ qua, lấy mặc định.
func ThresholdsFrom(settings json.RawMessage) Thresholds {
	th := DefaultThresholds()
	var s stored
	if json.Unmarshal(settings, &s) != nil {
		return th
	}
	apply(&th, s.Risk)
	return th
}

func apply(th *Thresholds, in RiskIn) {
	if in.AttendanceOKMin != nil {
		if v, err := decimal.NewFromString(*in.AttendanceOKMin); err == nil {
			th.AttendanceOKMin = v
		}
	}
	if in.ActivityMinPerWeek != nil {
		th.ActivityMinPerWeek = *in.ActivityMinPerWeek
	}
	if in.StudyMinMinutes != nil {
		th.StudyMinMinutes = *in.StudyMinMinutes
	}
	if in.SignalsRequired != nil {
		th.SignalsRequired = *in.SignalsRequired
	}
	if in.AbsentAlert != nil {
		th.AbsentAlert = *in.AbsentAlert
	}
}

var (
	okMin  = decimal.New(50, -2) //nolint:gochecknoglobals // hằng tra cứu bất biến
	okMax  = decimal.New(1, 0)   //nolint:gochecknoglobals
	okStep = decimal.New(5, -2)  //nolint:gochecknoglobals
)

// RiskError nêu trường sai (→ 422 VALIDATION_FAILED, field = `risk.<Field>`).
type RiskError struct{ Field, Message string }

func (e *RiskError) Error() string {
	return fmt.Sprintf("crm: ngưỡng %s sai: %s", e.Field, e.Message)
}

// MergeRisk kiểm khoảng của từng trường trong `in` (SRS 4.7.3) rồi ghi đè lên khoá tương ứng của `courses.settings`; khoá khác của settings giữ nguyên.
// Trả settings mới và ngưỡng hiệu lực trước / sau để audit.
func MergeRisk(settings json.RawMessage, in RiskIn) (merged json.RawMessage, before, after Thresholds, err error) {
	before = ThresholdsFrom(settings)
	if in.AttendanceOKMin != nil {
		v, perr := decimal.NewFromString(*in.AttendanceOKMin)
		if perr != nil || v.LessThan(okMin) || v.GreaterThan(okMax) || !v.Mod(okStep).IsZero() {
			return nil, before, before, &RiskError{"attendance_ok_min", "Từ 0,50 đến 1,00, mỗi bước 0,05."}
		}
	}
	for _, c := range []struct {
		v      *int
		field  string
		lo, hi int
		msg    string
	}{
		{in.ActivityMinPerWeek, "activity_min_per_week", 0, 20, "Từ 0 đến 20."},
		{in.StudyMinMinutes, "study_min_minutes", 0, 600, "Từ 0 đến 600."},
		{in.SignalsRequired, "signals_required", 1, 3, "Từ 1 đến 3."},
		{in.AbsentAlert, "absent_alert", 1, 20, "Từ 1 đến 20."},
	} {
		if c.v != nil && (*c.v < c.lo || *c.v > c.hi) {
			return nil, before, before, &RiskError{c.field, c.msg}
		}
	}
	var all map[string]json.RawMessage
	if len(settings) > 0 {
		if err := json.Unmarshal(settings, &all); err != nil {
			return nil, before, before, fmt.Errorf("crm: settings hỏng: %w", err)
		}
	}
	if all == nil {
		all = map[string]json.RawMessage{}
	}
	var cur RiskIn
	if raw, ok := all["risk"]; ok {
		if err := json.Unmarshal(raw, &cur); err != nil {
			return nil, before, before, fmt.Errorf("crm: settings.risk hỏng: %w", err)
		}
	}
	if in.AttendanceOKMin != nil {
		v := decimal.RequireFromString(*in.AttendanceOKMin).StringFixed(2)
		cur.AttendanceOKMin = &v
	}
	for _, p := range []struct{ dst, src **int }{{&cur.ActivityMinPerWeek, &in.ActivityMinPerWeek}, {&cur.StudyMinMinutes, &in.StudyMinMinutes}, {&cur.SignalsRequired, &in.SignalsRequired}, {&cur.AbsentAlert, &in.AbsentAlert}} {
		if *p.src != nil {
			*p.dst = *p.src
		}
	}
	raw, err := json.Marshal(cur)
	if err != nil {
		return nil, before, before, fmt.Errorf("crm: mã hoá settings.risk: %w", err)
	}
	all["risk"] = raw
	merged, err = json.Marshal(all)
	if err != nil {
		return nil, before, before, fmt.Errorf("crm: mã hoá settings: %w", err)
	}
	return merged, before, ThresholdsFrom(merged), nil
}
