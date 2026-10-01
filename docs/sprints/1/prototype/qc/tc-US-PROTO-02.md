# QC test case — US-PROTO-02 (Giảng viên / trợ giảng: vận hành lớp)
Nguồn: `docs/sprints/1/prototype/spec/US.md` + `SRS.md` 4.1, 4.4. Hộp đen. Công cụ: **C** = `proto-curl.sh`, **B** = trình duyệt. Đặt lại dữ liệu demo trước chuỗi TC.

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-02-01 | AC1 | – | **C** `tc_02_01` (GV: `/ /inbox /students /students/sv-3 /attendance /class/members`) | Đều `MO` |
| TC-02-02 | AC1 / SRS 4.4 `/` | GV, lớp "Tất cả lớp của tôi" | **B** Mở `/` | "6 việc cần xử lý hôm nay" đúng thứ tự: (1) ticket chờ 3 ngày 4 giờ, (2) điểm danh buổi 10 lớp 761987 đang diễn ra, (3) BT03 có 4 bài "Cần xem kỹ", (4) 1 câu AI chờ xác nhận, (5) "Thiết lập lớp mới — 761988" (chia sẻ mã → tải quy chế → tạo lịch buổi học → tải tài liệu), (6) 3 yêu cầu vào lớp chờ duyệt; mỗi việc ghi tên lớp |
| TC-02-03 | AC1 | – | **B** `/` | Chuông có thông báo "Bạn được phân công lớp An ninh mạng – 761988" kèm `BX4P9TW`; **không thẻ số liệu, không biểu đồ** ở khung nhìn đầu (DESIGN §21: ≥ 3 KPI cùng cỡ là phản mẫu); "Lớp cần chú ý" hẹp (C + 2 SV); dải lịch sắp tới |
| TC-02-04 | AC1 | – | **B** Bấm từng việc; xử lý một việc | Đúng màn đích; việc đã xử lý biến khỏi danh sách; hết việc → "Không còn việc cần bạn quyết định" |
| TC-02-05 | AC2 | B đã gửi D3 (US-PROTO-01 TC-01-07) | **B** GV → `/inbox` → mở ticket D3 → `Nhận` | Trạng thái Claimed, tên giảng viên |
| TC-02-06 | AC2 | Sau TC-02-05 | **B** Gõ D4 (nguyên văn) → `Gửi trả lời` | Chuyển Answered; dòng "Đã gửi thư thông báo tới email của sinh viên (mô phỏng)" hiện tại chỗ (FR-X7); tuỳ chọn `Lưu thành tri thức` có |
| TC-02-07 | AC2 / SRS mục 3 | – | **B** Ticket mẫu "đã có người nhận" | Hiện "Phạm Quốc Bảo đã nhận lúc 09:12" thay nút `Nhận` (409 giả lập) |
| TC-02-08 | AC2 | – | **B** Danh sách ticket | 5 ticket mở (26 phút, 3 giờ, 1 ngày, 2 ngày, 3 ngày 4 giờ) + 1 đã có người nhận; hàng có tên SV, câu ngắn, tuổi, trạng thái, lý do ("Độ tin cậy 0,42 < 0,80" — GV được thấy) |
| TC-02-09 | AC3 | `/attendance` buổi 10 lớp 1 (29/10) | **B** Bàn phím: ↑/↓ chọn hàng, `4` đánh vắng 2 SV, `2` một SV muộn, `P` cho B → `Lưu điểm danh`; bấm giờ | Mỗi thao tác hiện "Đã đánh vắng · Hoàn tác" 5 s + "Đã lưu 09:2x"; kết thúc "Đã hoàn tất buổi 10 · 27 có mặt, 1 muộn, 2 vắng"; toàn bộ ≤ 60 s |
| TC-02-10 | AC3 | Sau TC-02-09 | **B** Đổi vai B → `/me` | QT = 8,5 |
| TC-02-11 | AC3 | – | **B** `Hoàn tác` trong 5 s; phím `3`; `P` hai lần | Hoàn tác khôi phục trạng thái; `3` = Vắng phép; `P` hai lần = +0,5 phát biểu (trần +1,0 tính ở `/me`) |
| TC-02-12 | AC3 / SRS 4.4 | – | **B** Bộ chọn buổi 1–15 | Buổi 1–9 đã điểm danh (xem lại), buổi 11+ → "Buổi này chưa diễn ra" |
| TC-02-13 | AC4 | D đã gửi yêu cầu (US-PROTO-01 TC-01-23) | **B** GV → `/class/members` lớp 2 | 24 thành viên + 3 chờ duyệt + D; mã `BX4P9TW`, bật duyệt |
| TC-02-14 | AC4 | Sau TC-02-13 | **B** `Duyệt` D | D thành thành viên; có Hoàn tác; đổi vai về D → bộ chọn lớp có 761988, "Hôm nay" bình thường |
| TC-02-15 | AC4 / SRS 4.4 | GV | **B** `Từ chối` một yêu cầu; `Tạo lại mã` | `Từ chối` có Hoàn tác; `Tạo lại mã` qua xác nhận "mã cũ vô hiệu ngay" → mã mới + `Sao chép link`; mã cũ ở `/join` của D → câu chung "Mã không hợp lệ…" |
| TC-02-16 | AC5 | – | **B** 375 × 812: `/inbox` | Danh sách → chi tiết (hai bước, nút quay lại); chữ không tràn; vùng chạm ≥ 44 px |
| TC-02-17 | AC5 | – | **B** 375 × 812: `/attendance` | Hàng có kẻ, **không card**, vùng chạm ≥ 44 px, thao tác bằng chạm; không cuộn ngang |
| TC-02-18 | AC6 (nhánh lỗi) | `/attendance` | **B** Bật "Giả lập mất mạng", đổi 2 ô | "Đang chờ mạng · 2 thay đổi"; tắt công tắc → "Đã lưu 09:2x", không mất thay đổi, không trùng |
| TC-02-19 | AC6 | – | **B** `/students?state=error` | Lỗi có câu khắc phục + `Thử lại` |
| TC-02-20 | AC7 (phân quyền) | – | **C** `tc_02_07` | SV, Admin bị chặn mọi route của US; TA không có `Tạo lại mã`, không có `Mời ra khỏi lớp` |
| TC-02-21 | AC7 | TA | **B** `/class/members` lớp 2 | TA xem được danh sách, `Duyệt` được; thiếu `Tạo lại mã` và mời ra |
| TC-02-22 | SRS 4.4 `/students` | GV | **B** Tìm "Huy"/"20229003"/"zzzz"; chip `Cần chú ý`, `Vắng nhiều`, `Điểm giảm`, `Ít hoạt động` | 30 SV lớp 1; chip `Cần chú ý` = C + 2; tìm theo tên/MSSV; không khớp → rỗng có hướng; bấm hàng → `/students/[id]`, quay lại giữ vị trí cuộn |
| TC-02-23 | SRS 4.4 `/students/[id]` | GV | **B** `/students/sv-3` (C) | Câu rủi ro "Vắng 5/9 buổi, đã bị trừ 1,5 điểm; thiếu BT02"; tab Tổng quan / Chuyên cần / Điểm / Hoạt động học / Ghi chú; ghi chú "Chỉ giảng viên/TA thấy" |
| TC-02-24 | SRS 4.4 | GV | **B** `Thêm ghi chú`; `Nhắn riêng` | Ghi chú thêm tại chỗ + Hoàn tác (không modal); `Nhắn riêng` → dòng xác nhận mô phỏng |
| TC-02-25 | **Quyền chat riêng** (CLAUDE.md) | Sau D1–D3 của B | **B** GV ở mọi route (`/inbox`, `/students/sv-2`, `/`, `/analytics`) | Không thấy nội dung D1/D2 (chat riêng chưa escalate); chỉ thấy ticket D3 (đã escalate) với câu hỏi D3 |
| TC-02-26 | SRS 4.4 trạng thái | – | **B** `?state=` trên 6 route; Lọc Open rỗng ở `/inbox` | Rỗng "Không còn câu hỏi đang chờ"; loading skeleton đúng hình |

## Nhánh lỗi
TC-02-07 (409 giả lập), TC-02-18 (mất mạng), TC-02-19, TC-02-26 (`?state=`), TC-02-11 (hoàn tác).

## Phân quyền
TC-02-20, 21 (SV/Admin/TA); TC-02-25 (GV không đọc chat riêng chưa escalate — luật CLAUDE.md); TC-02-23 (ghi chú chỉ GV/TA).

## Kiểm chéo
- Bàn phím và focus: TC-02-09 chỉ dùng bàn phím; kiểm Tab/Shift+Tab, focus thấy rõ trên `/attendance` và Dialog.
- Phản mẫu §21: `/` không tường KPI (TC-02-03); `/attendance` không card (TC-02-17); ≤ 1 nút chính đỏ mỗi vùng.
- 375 px: `/inbox`, `/attendance` bắt buộc; các route khác 390 px không vỡ.

## Điểm khó kiểm
- "Toàn bộ ≤ 60 s" phụ thuộc người bấm: ghi thời gian thực tế trong report (bấm giờ bằng script `demo-run.mjs` khi có).
- Phụ thuộc chéo: TC-02-05 cần D3 (US-PROTO-01); TC-02-13/14 cần yêu cầu của D.

## Lịch sử sửa TC (chỉ khi SPEC đổi: ngày, TC nào, lý do)
(chưa có)
