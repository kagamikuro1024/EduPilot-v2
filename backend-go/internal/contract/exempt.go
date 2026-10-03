package contract

// Exemption là một cặp (thao tác, status) được khai báo trong spec nhưng KHÔNG thể sinh ra bằng test ở PG.
// Mỗi mục phải có lý do rõ ràng; QC đọc tệp này như artefact hợp đồng (US-PG-06 AC5).
type Exemption struct {
	Operation string // "METHOD /đường-dẫn" của spec
	Status    int
	Reason    string
}

// Exemptions: hiện KHÔNG có mục nào — mọi status khai báo đều được gọi thật bởi `scenarios_test.go`.
// Khi một phase sau khai báo status không thể sinh ra trong test (ví dụ 503 khi kéo sập hạ tầng thật), thêm vào đây kèm lý do.
func Exemptions() []Exemption { return nil }

// IsExempt cho biết (thao tác, status) có được miễn không.
func IsExempt(operation string, status int) bool {
	for _, e := range Exemptions() {
		if e.Operation == operation && e.Status == status && e.Reason != "" {
			return true
		}
	}
	return false
}
