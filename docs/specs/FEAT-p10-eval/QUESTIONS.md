# Câu hỏi mở — FEAT-p10-eval (sprint 11 · P10)
Ủy quyền: chủ dự án «chạy một mạch, không chờ hỏi» (2026-11-14). Câu không đụng tính điểm / auth / dữ liệu cá nhân → «PM chốt theo đề xuất BA». Hai câu đụng dữ liệu cá nhân (**Q-E-1**, **Q-E-21**; kèm Q-E-4) viết theo phương án an toàn nhất; **PM xác nhận P-11-1** (2026-11-14).

| # | Câu hỏi | Phương án BA đề xuất | Trả lời của chủ dự án | Ngày |
| --- | --- | --- | --- | --- |
| Q-E-1 | **(PII)** `llm_audit` không có cột nội dung, nhưng P10 L1 bắt buộc «chi tiết: prompt ĐÃ CHE, tool, confidence». Lấy nội dung ở đâu? | Bảng mới `llm_request_payloads` (SRS 5): **chỉ bản đã che**, ghi trong `internal/llm` sau bước che, giữ 30 ngày rồi xoá, chỉ ADMIN đọc qua API có lý do + `audit_log`. Không suy ngược từ `chat_messages` (có thể chứa chữ gốc). Phương án thay: không lưu nội dung, chi tiết chỉ metadata + `trace_id` (an toàn hơn, mất khả năng điều tra) | PM xác nhận P-11-1 (phương án đề xuất) | 2026-11-14 |
| Q-E-2 | Ngưỡng cảnh báo cụ thể? | Lỗi: > 5% trong cửa sổ 1 giờ **và** ≥ 20 yêu cầu; chi phí: ≥ 100% `daily_limit`; tính mỗi 5 phút; cùng loại + phạm vi không lặp trong 6 giờ; chỉ chuông ADMIN (không mail) | PM chốt theo đề xuất BA | 2026-11-14 |
| Q-E-3 | P10 L1 nói «chọn lý do»; prototype 9.1 đã có ô tự do ≥ 10 ký tự. Chọn kiểu nào? | Giữ ô tự do 10–200 ký tự (đã có ở prototype, QC đã kiểm); không thêm danh sách cố định | PM chốt theo đề xuất BA | 2026-11-14 |
| Q-E-4 | `ARCHITECTURE.md` §5 dùng `GET …/requests/{id}` nhưng mỗi lần mở phải ghi `audit_log` và có lý do | Đổi thành `POST …/requests/{id}/open {reason}`: GET có tác dụng phụ dễ bị cache / prefetch / lộ lý do trong URL | PM xác nhận P-11-1 (cần vá ARCHITECTURE §5) | 2026-11-14 |
| Q-E-5 | Jaeger có trong compose không? | Liên kết chỉ hiện khi có `JAEGER_BASE_URL`; không thì hiện `trace_id` + sao chép. Không thêm service | PM chốt theo đề xuất BA | 2026-11-14 |
| Q-E-6 | Trọng số xếp hạng 5 tín hiệu và cách gom cụm? | SRS 4.1: 0,30 / 0,20 / 0,20 / 0,15 / 0,15 trên giá trị chuẩn hoá; nguồn thiếu → chia đều; gom cụm cosine; hằng số ở cấu hình. Chỉ xếp hạng giảng dạy, không đụng điểm | PM chốt theo đề xuất BA | 2026-11-14 |
| Q-E-7 | Khoảng thời gian, ngưỡng dữ liệu, trần việc? | `period_days` ∈ {7, 14, 30, toàn kỳ}, mặc định 14; ≥ 20 câu hỏi mới gom; tối đa 3 `generate` / 60 phút / lớp (`BATCH`) | PM chốt theo đề xuất BA | 2026-11-14 |
| Q-E-8 | «Xuất PDF» khi bảng thư viện không có thư viện PDF? | Markdown do máy chủ; PDF = in của trình duyệt với CSS in. Không thêm thư viện (AGENTS.md) | PM chốt theo đề xuất BA | 2026-11-14 |
| Q-E-9 | `Tạo thread ghim` / `Tạo buổi ôn tập` tự đăng không? | Không: mở nháp → giảng viên sửa → `Đăng` / `Lưu`. Thread dùng `pinned_at`; sự kiện lịch dùng type `OTHER`; nội dung không chứa câu mẫu | PM chốt theo đề xuất BA | 2026-11-14 |
| Q-E-10 | «Sinh viên tụt điểm liên tiếp» (P10 L2: gợi ý cho P5) có ghi vào `student_risk`? | Không ở P10: chỉ danh sách trên `/analytics` + liên kết hồ sơ 360 (đổi luật rủi ro cần spec CRM riêng). 3 bài thi liên tiếp giảm = tụt | PM chốt theo đề xuất BA | 2026-11-14 |
| Q-E-11 | «LCP 7 trang»: nợ ghi `/assignments/[id]` + «6 trang» nhưng chỉ nêu 5 tên; TL-2f nêu 6 route trượt | Đo cả 8 route của bộ Lighthouse (trừ `/dev/ui`) sau sửa để khỏi bỏ sót; mục tiêu ≤ 2.500 ms mọi route | PM chốt theo đề xuất BA | 2026-11-14 |
| Q-E-12 | Giữ payload prompt bao lâu? | 30 ngày; `llm_audit` giữ nguyên; hết hạn → `410` | PM chốt theo đề xuất BA | 2026-11-14 |
| Q-E-13 | TA có quyền như TEACHER ở `/insights`? | Có (TA là staff lớp, PRD M13-B «TA, TEACHER»); ADMIN không | PM chốt theo đề xuất BA | 2026-11-14 |
| Q-E-14 | Mục tiêu E1? | F1 phân loại kênh ≥ 0,90; rò rỉ payload = 0. Đây là mục tiêu để báo cáo, không phải điều kiện sửa số đo | PM chốt theo đề xuất BA | 2026-11-14 |
| Q-E-15 | ADMIN xem thống kê điểm thi thế nào? | Chỉ số tổng (số bài nộp, tỉ lệ đạt); không danh tính, không danh sách tụt điểm (P10 L2 «ADMIN chỉ số tổng») | PM chốt theo đề xuất BA | 2026-11-14 |
| Q-E-16 | Ngưỡng ẩn danh ở thống kê? | Câu mẫu insight k ≥ 3 sinh viên; so sánh lớp cần ≥ 5 bài nộp/lớp mới hiện số | PM chốt theo đề xuất BA | 2026-11-14 |
| Q-E-17 | LLM lỗi lúc đặt tên? | Vẫn tạo báo cáo với tên tạm từ từ khoá cụm (`named_by:"fallback"`); lỗi trước bước đặt tên → không lưu báo cáo | PM chốt theo đề xuất BA | 2026-11-14 |
| Q-E-18 | E3 chưa có điểm chuẩn của người? | Ghi `WAITING_HUMAN`, không dùng điểm LLM làm chuẩn (vòng tròn, P10 L3). Chủ dự án chấm ≥ 60 bài sau | PM chốt theo đề xuất BA | 2026-11-14 |
| Q-E-19 | Video dự phòng? | Chủ dự án quay; story ghi «chờ chủ dự án», không FAIL | PM chốt theo đề xuất BA | 2026-11-14 |
| Q-E-20 | Seed T1, k6, chaos của P10 L3? | Đã làm ở 9.5; không lặp. Dev chỉ trích số liệu có sẵn vào báo cáo | PM chốt theo đề xuất BA | 2026-11-14 |
| Q-E-21 | **(PII)** Admin đọc câu mẫu (đã che) của báo cáo lỗ hổng? | Không: `latest` / `export` chỉ TA, TEACHER của lớp; ADMIN `403`. Admin có chỉ số tổng ở `/observability` | PM xác nhận P-11-1 (phương án đề xuất) | 2026-11-14 |
| Q-E-22 | TA có xem `/observability`? | Không (`403`): màn là hệ thống AI, TEACHER chỉ thấy tổng hợp lớp mình (PRD M13) | PM chốt theo đề xuất BA | 2026-11-14 |
