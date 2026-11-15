# QC test case — US-DM-06 (`/settings/llm` khối «Quyết định nhanh»), US-DM-07 (`/observability` bảng «Quyết định nhanh»)
Nguồn: US.md + SRS §6, §7, §8; Q-D-4, 11, 12; DESIGN §13, §14.22, §14.23; UX.md mục 6; CLAUDE.md luật chữ UI. Hộp đen; quy ước: `tc-00-chung.md` (**provider giả**). Chụp ảnh `qc/shots/` 1440 + 375.

## US-DM-06 Màn cấu hình
| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-D06-01 | AC1 | ADMIN, 1440 px | Mở `/settings/llm` | Đúng **một** khối «Quyết định nhanh» **ngay sau** «Mô hình theo tác vụ», **trước** «Mô hình tìm kiếm tài liệu»; một cột; ngăn bằng khoảng trắng + đường kẻ 1 px; không khung lồng khung, không thẻ số liệu | E2E+TAY |
| TC-D06-02 | AC2 | Seed (`decision-fake`) | Xem hàng provider | Nhãn «Quyết định» · mô hình `decision-fake` · trạng thái 1–2 từ («Giả») · nút `Test kết nối` | E2E |
| TC-D06-03 | AC2 | – | `Test kết nối` | Gọi `POST …/providers/{id}/test` (cục bộ, fake); báo kết quả tại chỗ, không gọi mạng ngoài (kiểm TC-D09-11) | E2E+API |
| TC-D06-04 | AC2 | `ALLOW_REAL=false` | Đổi mô hình sang `gemini-decision-1` | Báo tại chỗ «Chưa được phép dùng bản thật»; API `409 DECISION_REAL_DISABLED`; mô hình **không** đổi | E2E+API |
| TC-D06-05 | AC2 | – | Nhập khoá API (giả) rồi lưu; tải lại trang | Khoá **chỉ ghi**: sau lưu chỉ hiện trạng thái («Đã kết nối» / «Chưa có khoá»); không `<input>` / phản hồi API nào chứa khoá (`grep` DOM + mạng) | E2E+API |
| TC-D06-06 | AC2 | Provider chưa có khoá / tạm dừng | Quan sát trạng thái | Đúng 4 giá trị: «Giả», «Đã kết nối», «Chưa có khoá», «Tạm dừng» | E2E |
| TC-D06-07 | AC3 | – | Xem ba hàng cổng | Mỗi hàng: tên («Kiểm duyệt Threads», «Cờ chèn lệnh khi chấm», «Định tuyến câu hỏi chat») · công tắc toàn hệ thống (mặc định **tắt**) · trạng thái («Tắt» / «Bật ở N lớp») · `Cài đặt nâng cao` | E2E |
| TC-D06-08 | AC3 | – | Mở `Cài đặt nâng cao` mỗi cổng; nhập ngưỡng / thời hạn ngoài khoảng (0,001; 5.001) | Đơn vị và khoảng ở `placeholder`; tooltip chỉ nêu cú pháp / khoảng; báo `422` tại ô (không toast); trong khoảng → lưu | E2E |
| TC-D06-09 | AC4 | Cổng tắt toàn hệ thống | Bấm `Theo lớp` | Mở tại chỗ `DataTable` (tên + mã, công tắc lớp); **công tắc lớp khoá**; tooltip trên công tắc «Đang tắt toàn hệ thống» (lý do khoá) | E2E |
| TC-D06-10 | AC4 | Cổng bật toàn hệ thống; ≥ 120 lớp (tạo thêm bằng SQL, ghi rõ) | `Theo lớp`: tìm theo tên / mã; cuộn tới trang kế | Tìm đúng; cursor duyệt hết không trùng; `limit` ≤ 100; bật 1 lớp → trạng thái hàng «Bật ở 1 lớp» đổi | E2E |
| TC-D06-11 | AC5 | `decision-fake` | Bật công tắc cổng | **Không** có tooltip «Gửi nội dung đã che…» (bản giả) | E2E |
| TC-D06-12 | AC5 | Provider bản thật (dựng bằng ghi DB trên stack thử + `ALLOW_REAL=true`, **không** gọi mạng; ghi rõ) | Bật công tắc | Tooltip tại chính công tắc: «Gửi nội dung đã che tới nhà cung cấp»; không câu mô tả dưới tiêu đề / dưới hàng; không icon Info (`grep` DOM) | E2E |
| TC-D06-13 | AC6 | – | Đổi công tắc / ngưỡng | Cập nhật lạc quan + dòng «Hoàn tác» tĩnh lặng; không hộp xác nhận, không toast «Thành công»; «Hoàn tác» khôi phục (`version` + `audit_log` hai dòng) | E2E+DB |
| TC-D06-14 | AC6 | Hai tab cùng ADMIN đổi một cổng | Tab 2 lưu với `version` cũ | `409` → tải lại hàng + báo tại chỗ; không ghi đè | E2E |
| TC-D06-15 | AC6 | – | Đổi cổng; `GET /admin/audit` | Dòng `audit_log` (`entity='decision_gate'`, `before`/`after`, actor) | API |
| TC-D06-16 | AC7 | API `GET $DEC` chậm 3 s; trả `500`; chưa có provider decision (xoá bản ghi trên DB thử) | Mở khối | Tải: `<PageState>`/skeleton (không spinner riêng); lỗi: `Thử lại` hoạt động; chưa cấu hình: hàng «Quyết định» «Chưa cấu hình» + nút `Thêm` (một hành động) | E2E |
| TC-D06-17 | AC8 | TEACHER, TA, STUDENT | Mở `/settings/llm`; gọi `GET $DEC`, `PUT $DEC/gates/…`, `PUT /courses/{id}/decision-gates/…` | Không thấy khối / route; API `403` | E2E+API |
| TC-D06-18 | AC9 | 375 px | Mở khối; cuộn; bảng `Theo lớp` | Ba hàng cổng xếp dọc; công tắc vùng chạm ≥ 44 px (đo bounding box); bảng cuộn trong khung; không tràn ngang trang | E2E |
| TC-D06-19 | AC10 | – | Quét chữ UI khối | Không «RAG / PII / fallback / trace / cascade / decision» (chữ hiển thị); tên cổng, trạng thái đúng bảng; `bash scripts/ui-antipatterns.sh` mã 0 | TAY |
| TC-D06-20 | AC10 | Repo | `docs/specs/FEAT-llm-gateway/SRS.md` 7.3 | Có nhãn tác vụ `DECIDE` («Quyết định nhanh», làn «Trả lời ngay» / «Chạy nền») — vá cùng đợt | DIFF |
| TC-D06-21 | AC1 | `/settings/llm` ADMIN | Chạy spec cũ của màn (`settings` / `llm`) | Xanh; không sửa assertion cũ (`git diff` chỉ thêm) | E2E+DIFF |
| TC-D06-22 | AC1 | Ảnh mốc | Đối chiếu DESIGN §14.23 | Khớp hợp đồng route; ≥ 8/10 điều kiện §22 (ghi từng điều chưa đạt) | TAY |
| TC-D06-23 | RBAC | Giảng viên / SV | Mọi phản hồi `/settings`, `/threads`, `/chat` | Không từ «Quyết định nhanh», tên cổng (SRS §2: không bao giờ) | API |

## US-DM-07 Bảng số liệu
| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-D07-01 | AC1 | Đã tạo lưu lượng ở cả ba cổng (≥ 50 lời gọi mỗi cổng, có FAST / FALLBACK / FAIL_CLOSED, lỗi) | `GET /admin/observability/decision?window=24h`; so với SQL `llm_audit` (`task='DECIDE'`, `cascade_gate`, `cascade_path`) | Từng số khớp SQL: `calls`, `fast_pct`, `fallback_pct`, `fail_closed_pct` (tổng ≈ 100 %), p50 / p95, `error_pct`, `cost_vnd` (tổng `cost_est`); `window=7d` cũng khớp | API+DB |
| TC-D07-02 | AC1 | – | Mở `/observability` (ADMIN) | Một bảng «Quyết định nhanh» **dưới** bảng 7 ngày: hàng = cổng, cột như AC1; chọn 24 giờ / 7 ngày đổi số; **không** nội dung ở bất kỳ ô nào | E2E |
| TC-D07-03 | AC2 | – | Dải «24 giờ qua» | **Không** có chỉ số decision trong dải; mỗi số đúng một chỗ | E2E |
| TC-D07-04 | AC3 | Cổng tắt / bật ở N/M lớp | Xem cột trạng thái | «Tắt» + `—` thay số khi tắt; «Bật ở N/M lớp» khớp `courses_enabled / courses_total` (Q-S105-7: M là gì) | E2E+API |
| TC-D07-05 | AC4 | Cổng có 30 lời gọi trong giờ, 3 `FAIL_CLOSED` (10 %) (ghi `llm_audit`, ghi rõ); chạy worker (cách ép: Q-S11-3 / handoff) | Chạy | 1 thông báo `DECISION_FAIL_CLOSED` cho mọi ADMIN (chuông), liên kết `/observability`; chạy lại → vẫn 1 (khử trùng 6 giờ) | DB+E2E |
| TC-D07-06 | AC4 | 19 lời gọi 100 % `FAIL_CLOSED`; 20 lời gọi 5,0 % (1/20) và 2/20 (10 %) | Chạy worker | 19: không; 20 với 5,0 %: **không** (> 5 % strict); 10 %: có | DB |
| TC-D07-07 | AC4 | – | Sau +6 giờ (dịch `created_at`) điều kiện còn | Thông báo mới | DB |
| TC-D07-08 | AC5 | Cầu dao `decision` mở / `budget_blocked` ≥ 1 trong giờ | Mở `/settings/llm` | Hàng provider «Tạm dừng»; **không** thông báo riêng mới (không dội) | E2E+DB |
| TC-D07-09 | AC6 | Teacher, TA, STUDENT | `GET /admin/observability/decision`; Teacher mở `/observability` (tổng hợp lớp) | `403`; bản tổng hợp lớp của Teacher **không** có số decision | API+E2E |
| TC-D07-10 | AC6 | – | `GET /notifications` của Teacher / TA / SV | Không có `DECISION_FAIL_CLOSED` | API |
| TC-D07-11 | AC7 | `llm_audit` ≥ 1 triệu hàng (nạp SQL `generate_series`, ghi rõ) | 30 lần `GET …/decision?window=7d`; `EXPLAIN` | p95 ≤ 500 ms; dùng chỉ mục `(created_at, cascade_gate) WHERE task='DECIDE'`; không `OFFSET`, không Seq Scan | API+DB |
| TC-D07-12 | AC8 | Chưa có lời gọi `DECIDE`; chặn API `500` | Mở bảng | `EmptyState` một dòng; lỗi: `Thử lại` | E2E |
| TC-D07-13 | AC9 | 375 px | Mở bảng | Bảng cuộn trong khung; không tràn trang | E2E |
| TC-D07-14 | – | Header | `GET …/decision` | `Cache-Control: no-store` | API |
| TC-D07-15 | – | `go test ./internal/... -run 'TestDecisionRBAC\|TestDecisionGateVersionConflict\|TestDecisionAudit' -v` | – | PASS | GO |
| TC-D07-16 | AC1 | Chạy cổng chỉ ở lớp 1 rồi xem số | Quan sát | Số chỉ của lớp có cổng; không lẫn dữ liệu lớp khác ở bảng tổng (hệ thống) — ghi nhận quan sát | API |

## Nhánh lỗi
TC-D06-04, 08, 14, 16; TC-D07-12.

## Phân quyền
TC-D06-17, 23; TC-D07-09, 10.

## Script chạy được
Pha 2: `frontend/e2e/qc-dm-settings.spec.ts`, `qc-dm-observability.spec.ts` (1440 + 375), `docs/sprints/10.5/qc/scripts/dm_ui_api.py`.

## Lịch sử sửa TC
(chưa có)
