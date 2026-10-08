package contract

// Exemption là một cặp (thao tác, status) được khai báo trong spec nhưng KHÔNG thể sinh ra bằng test ở PG.
// Mỗi mục phải có lý do rõ ràng; QC đọc tệp này như artefact hợp đồng (US-PG-06 AC5).
type Exemption struct {
	Operation string // "METHOD /đường-dẫn" của spec
	Status    int
	Reason    string
}

// Exemptions: mọi status khai báo đều được gọi thật bởi `scenarios_test.go`, trừ các mục dưới đây.
// Khi một phase sau khai báo status không thể sinh ra trong test (ví dụ 503 khi kéo sập hạ tầng thật), thêm vào đây kèm lý do.
func Exemptions() []Exemption {
	return []Exemption{
		{"GET /api/v1/me/today", 504, "Quá hạn yêu cầu chỉ sinh ra khi DB chậm thật; hành vi được `today.TestTodayDeadline504` kiểm bằng ngữ cảnh đã hết hạn."},
		{"GET /api/v1/courses/{id}/today", 504, "Như trên: `today.TestTodayDeadline504`."},
		{"GET /api/v1/me/today", 304, "304 không có thân; ETag / If-None-Match được `today.TestTodayETag` kiểm."},
		{"GET /api/v1/courses/{id}/today", 304, "Như trên: `today.TestTodayETag`."},
		{"GET /api/v1/courses/{id}", 503, "Guard trả 503 khi tra `enrollments` lỗi; không kéo sập Postgres dùng chung của bộ contract. Hành vi được `auth.TestGuardResolverError503` kiểm bằng resolver lỗi."},
		{"PUT /api/v1/courses/{id}/exams/{eid}/attempts/{aid}/answers", 429, "Giới hạn 240 lần lưu / phút / lượt: không gọi 240 lần trong kịch bản hợp đồng; hành vi được `exam.TestSaveRateLimit` kiểm (cấu hình nhỏ)."},
		{"PUT /api/v1/courses/{id}/exams/{eid}/attempts/{aid}/code/{itemId}/draft", 429, "Bản nháp dùng chung hạn mức 240 lần / phút / lượt với câu trả lời: không gọi 240 lần trong kịch bản hợp đồng; hành vi được `exam.TestSaveRateLimit` kiểm."},
		{"GET /api/v1/me/exam-lock", 503, "503 chỉ khi Redis và Postgres cùng lỗi (người gọi coi như bị khoá); hành vi được `exam.TestLockerBothDownDeniesChat` kiểm."},
		{"PUT /api/v1/courses/{id}/exams/{eid}/items/{itemId}/override", 202, "202 chỉ khi bài có > 200 lượt GRADED (tính lại ở việc nền); đường 200 được kịch bản gọi, đường 202 do `exam.TestOverrideLargeEnqueues` kiểm."},
		{"GET /api/v1/courses/{id}/exams/{eid}/results.csv", 422, "EXPORT_TOO_LARGE chỉ khi lớp > 5.000 sinh viên (thực tế sĩ số ≤ 1.000); `exam.TestResultsCSVTooLarge` kiểm bằng hạ trần."},
	}
}

// IsExempt cho biết (thao tác, status) có được miễn không.
func IsExempt(operation string, status int) bool {
	for _, e := range Exemptions() {
		if e.Operation == operation && e.Status == status && e.Reason != "" {
			return true
		}
	}
	return false
}
