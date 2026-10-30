# Sprint 8: góp ý của đội

Mỗi dòng: #, ai, vấn đề, đề xuất, lý do + bằng chứng, ảnh hưởng nếu không đổi. PM ghi cột "Quyết định PM" + ngày. Đề xuất đụng cách tính điểm: PM hỏi chủ dự án.

| # | Ai | Vấn đề | Đề xuất | Lý do + bằng chứng | Ảnh hưởng nếu không đổi | Quyết định PM |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | dev | US-S8-UI: `e2e/lcp.spec.ts` ca `/settings/llm` đỏ sau khi bỏ câu mô tả dưới tiêu đề. Ca không giả API nên sau phiên màn hiện thông báo lỗi "Cấu hình hiện có không bị ảnh hưởng." (size 8080) lớn hơn h1 PreShell (6440) → ứng viên LCP mới. Trước đây câu mô tả ở PreShell (cùng chữ) lớn hơn nên che mất. | Sửa ca: gọi `mockLlmApi(page)` (đã có ở `e2e/support/staffMock.ts`, dùng cho ảnh mốc `settings-llm`) khi `path === "/settings/llm"`, thêm 1 dòng cạnh `mockMe`. Không đổi ngưỡng, không đổi assertion. | Chạy thử bản sao tạm của spec có thêm dòng đó: `lcp before refresh /settings/llm` xanh, ứng viên duy nhất là h1 (`size 6440, fromPre true`). Thông báo lỗi là trạng thái hiếm, không phải màn thật của admin. Test cũ nên chưa đụng. | Ca `lcp before refresh /settings/llm` đỏ trong CI (US-S8-UI không xanh được). | |
