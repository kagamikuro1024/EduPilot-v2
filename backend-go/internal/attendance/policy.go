package attendance

import (
	"encoding/json"
	"net/netip"
	"strconv"
)

// Require: mức yêu cầu của lớp. «APP» chỉ để cờ NOT_APP (Q2: lượt từ web vẫn ghi).
const (
	RequireWeb = "WEB"
	RequireApp = "APP"
)

// Geofence là vị trí lớp (lat, lng, bán kính). Chỉ ở courses.settings; KHÔNG vào audit_log (chỉ «bật, bán kính»).
type Geofence struct {
	Lat     float64 `json:"lat"`
	Lng     float64 `json:"lng"`
	RadiusM int     `json:"radius_m"`
}

// Policy là `courses.settings.attendance` (SRS FR-1).
type Policy struct {
	SelfCheckin   bool      `json:"self_checkin"`
	RotateSeconds int       `json:"rotate_seconds"`
	LateAfterMin  int       `json:"late_after_min"`
	Require       string    `json:"require"`
	Geofence      *Geofence `json:"geofence,omitempty"`
	CampusCIDRs   []string  `json:"campus_cidrs,omitempty"`
}

// PolicyIn là thân `attendance` của `PUT …/settings`: thay TOÀN BỘ chính sách (trường vắng = mặc định, P-108-3).
type PolicyIn struct {
	SelfCheckin   *bool     `json:"self_checkin"`
	RotateSeconds *int      `json:"rotate_seconds"`
	LateAfterMin  *int      `json:"late_after_min"`
	Require       *string   `json:"require"`
	Geofence      *Geofence `json:"geofence"`
	CampusCIDRs   []string  `json:"campus_cidrs"`
}

// PolicyError: trường sai của chính sách → 422 theo trường `attendance.<tên>`.
type PolicyError struct{ Field, Message string }

func (e *PolicyError) Error() string { return "attendance: " + e.Field + ": " + e.Message }

// DefaultPolicy là chính sách khi lớp chưa khai (self_checkin tắt).
func DefaultPolicy(c Config) Policy {
	return Policy{RotateSeconds: c.RotateDefault, LateAfterMin: c.LateAfterDefaultMin, Require: RequireWeb}
}

// PolicyOf đọc `courses.settings.attendance`; thiếu / hỏng → mặc định. Trường vắng trong JSON → mặc định của trường đó.
func PolicyOf(c Config, settings []byte) Policy {
	p := DefaultPolicy(c)
	if len(settings) == 0 {
		return p
	}
	var env struct {
		Attendance *Policy `json:"attendance"`
	}
	if json.Unmarshal(settings, &env) != nil || env.Attendance == nil {
		return p
	}
	got := *env.Attendance
	if got.RotateSeconds == 0 {
		got.RotateSeconds = p.RotateSeconds
	}
	if got.Require == "" {
		got.Require = RequireWeb
	}
	return got
}

// Merge kiểm `in` rồi dựng chính sách MỚI (thay toàn bộ); trả (chính sách mới, jsonb `settings` đã trộn, trước, sau cho audit không toạ độ).
func Merge(c Config, settings []byte, in PolicyIn) (Policy, []byte, map[string]any, map[string]any, error) {
	p := DefaultPolicy(c)
	if in.SelfCheckin != nil {
		p.SelfCheckin = *in.SelfCheckin
	}
	if in.RotateSeconds != nil {
		if *in.RotateSeconds < c.RotateMin || *in.RotateSeconds > c.RotateMax {
			return Policy{}, nil, nil, nil, &PolicyError{"rotate_seconds", "Chu kỳ đổi mã từ " + strconv.Itoa(c.RotateMin) + " đến " + strconv.Itoa(c.RotateMax) + " giây."}
		}
		p.RotateSeconds = *in.RotateSeconds
	}
	if in.LateAfterMin != nil {
		if *in.LateAfterMin < 0 || *in.LateAfterMin > 120 {
			return Policy{}, nil, nil, nil, &PolicyError{"late_after_min", "Tính muộn sau 0 đến 120 phút."}
		}
		p.LateAfterMin = *in.LateAfterMin
	}
	if in.Require != nil {
		if *in.Require != RequireWeb && *in.Require != RequireApp {
			return Policy{}, nil, nil, nil, &PolicyError{"require", "Yêu cầu là WEB hoặc APP."}
		}
		p.Require = *in.Require
	}
	if g := in.Geofence; g != nil {
		if g.Lat < -90 || g.Lat > 90 || g.Lng < -180 || g.Lng > 180 {
			return Policy{}, nil, nil, nil, &PolicyError{"geofence", "Vị trí không hợp lệ."}
		}
		if g.RadiusM < c.GeoRadiusMin || g.RadiusM > c.GeoRadiusMax {
			return Policy{}, nil, nil, nil, &PolicyError{"geofence.radius_m", "Bán kính từ " + strconv.Itoa(c.GeoRadiusMin) + " đến " + strconv.Itoa(c.GeoRadiusMax) + " mét."}
		}
		cp := *g
		p.Geofence = &cp
	}
	if len(in.CampusCIDRs) > c.CampusCIDRMax {
		return Policy{}, nil, nil, nil, &PolicyError{"campus_cidrs", "Tối đa " + strconv.Itoa(c.CampusCIDRMax) + " dải mạng."}
	}
	for _, s := range in.CampusCIDRs {
		pf, err := netip.ParsePrefix(s)
		if err != nil || !pf.IsValid() || pf.Bits() == 0 || (pf.Addr().Is4() && pf.Bits() < 8) || (pf.Addr().Is6() && !pf.Addr().Is4In6() && pf.Bits() < 32) {
			return Policy{}, nil, nil, nil, &PolicyError{"campus_cidrs", "Dải mạng không hợp lệ: " + s}
		}
		p.CampusCIDRs = append(p.CampusCIDRs, pf.Masked().String())
	}
	old := PolicyOf(c, settings)
	merged, err := mergeSettings(settings, p)
	if err != nil {
		return Policy{}, nil, nil, nil, err
	}
	return p, merged, old.audit(), p.audit(), nil
}

// audit: dạng ghi audit_log — «vị trí bật, bán kính», KHÔNG lat / lng (SRS FR-1, nguyên tắc dữ liệu).
func (p Policy) audit() map[string]any {
	m := map[string]any{"self_checkin": p.SelfCheckin, "rotate_seconds": p.RotateSeconds, "late_after_min": p.LateAfterMin, "require": p.Require, "campus_cidrs": len(p.CampusCIDRs)}
	if p.Geofence != nil {
		m["geofence"] = map[string]any{"enabled": true, "radius_m": p.Geofence.RadiusM}
	} else {
		m["geofence"] = map[string]any{"enabled": false}
	}
	return m
}

func mergeSettings(settings []byte, p Policy) ([]byte, error) {
	obj := map[string]json.RawMessage{}
	if len(settings) > 0 {
		if err := json.Unmarshal(settings, &obj); err != nil {
			obj = map[string]json.RawMessage{}
		}
	}
	b, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	obj["attendance"] = b
	return json.Marshal(obj)
}

// ForStudent là phần chính sách sinh viên được thấy: `self_checkin`, `require`, `geo_check` (P-108-3); không toạ độ, không CIDR, không chu kỳ.
type ForStudent struct {
	SelfCheckin bool   `json:"self_checkin"`
	Require     string `json:"require"`
	GeoCheck    bool   `json:"geo_check"`
}

// Student rút gọn chính sách cho sinh viên.
func (p Policy) Student() ForStudent {
	return ForStudent{SelfCheckin: p.SelfCheckin, Require: p.Require, GeoCheck: p.Geofence != nil}
}
