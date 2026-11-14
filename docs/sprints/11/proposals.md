# Sprint 11: góp ý

| # | Ai | Vấn đề | Đề xuất | Lý do + bằng chứng | Ảnh hưởng | Quyết định PM |
| --- | --- | --- | --- | --- | --- | --- |
| P-11-1 | ba (Q-E-1, Q-E-4, Q-E-21 FEAT-p10-eval) | PII ở M13 | (a) Bảng mới `llm_request_payloads`: chỉ lưu prompt ĐÃ CHE, giữ 30 ngày, `llm_audit` không có cột nội dung; (b) mở chi tiết bằng `POST …/requests/{id}/open` (bắt buộc lý do, ghi `audit_log`) thay `GET`; (c) ADMIN không đọc câu mẫu insight; (d) 5 endpoint thống kê điểm thi + tab «Điểm thi» ở `/analytics` | Khớp luật «nội dung prompt chỉ ADMIN xem, có `audit_log`»; không lưu dạng rõ | Migration mới + ARCHITECTURE §4/§5 + openapi | **CHẤP NHẬN cả 4** (PM 2026-11-14). Dev vá `ARCHITECTURE.md` (bảng, endpoint) + `openapi.yaml` trong cùng lát, ghi P-11-1. | PM |
