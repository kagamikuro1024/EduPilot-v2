package attendance

// Mã cờ (SRS 4.4); cột `flags text[]` có CHECK giới hạn đúng tập này. Cờ chỉ gắn, không tự phạt; sinh viên không bao giờ thấy.
const (
	FlagDeviceShared    = "DEVICE_SHARED"
	FlagDeviceNew       = "DEVICE_NEW"
	FlagNoDeviceKey     = "NO_DEVICE_KEY"
	FlagDeviceUntrusted = "DEVICE_UNTRUSTED"
	FlagTokenLate       = "TOKEN_LATE"
	FlagGeoOut          = "GEO_OUT"
	FlagGeoWeak         = "GEO_WEAK"
	FlagGeoDenied       = "GEO_DENIED"
	FlagGeoMock         = "GEO_MOCK"
	FlagOffCampus       = "OFF_CAMPUS"
	FlagNotApp          = "NOT_APP"
	FlagRecheckMiss     = "RECHECK_MISS"
)

// flagLabels: nhãn tiếng Việt cho Staff (SRS 4.4.1). Không chứa từ kỹ thuật (token, HMAC, key, device, attestation).
var flagLabels = map[string]string{ //nolint:gochecknoglobals // bảng tra cứu bất biến
	FlagDeviceShared:    "Cùng máy với bạn khác",
	FlagDeviceNew:       "Máy mới",
	FlagNoDeviceKey:     "Không xác minh được máy",
	FlagDeviceUntrusted: "Máy không đáng tin",
	FlagTokenLate:       "Quét mã trễ",
	FlagGeoOut:          "Ngoài khu vực lớp",
	FlagGeoWeak:         "Vị trí không chắc",
	FlagGeoDenied:       "Không cho dùng vị trí",
	FlagGeoMock:         "Vị trí có dấu hiệu giả",
	FlagOffCampus:       "Ngoài mạng trường",
	FlagNotApp:          "Không dùng app",
	FlagRecheckMiss:     "Vắng ở lượt kiểm lại",
}

// FlagLabel trả nhãn của một mã cờ; mã lạ trả "".
func FlagLabel(code string) string { return flagLabels[code] }

// AllFlags là tập mã hợp lệ (khớp CHECK của `attendance_checkins.flags`).
func AllFlags() []string {
	return []string{FlagDeviceShared, FlagDeviceNew, FlagNoDeviceKey, FlagDeviceUntrusted, FlagTokenLate, FlagGeoOut, FlagGeoWeak, FlagGeoDenied, FlagGeoMock, FlagOffCampus, FlagNotApp, FlagRecheckMiss}
}

// FlagInfo là cờ kèm nhãn cho JSON của Staff.
type FlagInfo struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}

// FlagInfos đổi mã → {code, label}; mã lạ bị bỏ (không xảy ra: CHECK của DB).
func FlagInfos(codes []string) []FlagInfo {
	out := make([]FlagInfo, 0, len(codes))
	for _, c := range codes {
		if l := flagLabels[c]; l != "" {
			out = append(out, FlagInfo{Code: c, Label: l})
		}
	}
	return out
}
