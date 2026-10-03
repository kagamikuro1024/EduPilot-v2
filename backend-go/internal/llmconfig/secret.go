package llmconfig

import (
	"encoding/json"
	"log/slog"
)

const redacted = "[REDACTED]"

// Secret giữ chuỗi nhạy cảm (khoá API đầu vào). Mọi cách in / ghi log / JSON đều ra [REDACTED]; không có hàm xuất giá trị —
// chỉ gói này đọc được (để mã hoá).
type Secret struct{ v string }

// NewSecret bọc một chuỗi nhạy cảm.
func NewSecret(s string) Secret { return Secret{v: s} }

// IsEmpty cho biết chuỗi rỗng (sau TrimSpace không tính — người gọi quyết định).
func (s Secret) IsEmpty() bool { return s.v == "" }

func (Secret) String() string               { return redacted }
func (Secret) GoString() string             { return redacted }
func (Secret) LogValue() slog.Value         { return slog.StringValue(redacted) }
func (Secret) MarshalText() ([]byte, error) { return []byte(redacted), nil }
func (Secret) MarshalJSON() ([]byte, error) { return json.Marshal(redacted) }

// UnmarshalJSON nhận chuỗi JSON đầu vào (thân PUT/POST của US-P1-04).
func (s *Secret) UnmarshalJSON(b []byte) error {
	var v string
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	s.v = v
	return nil
}
