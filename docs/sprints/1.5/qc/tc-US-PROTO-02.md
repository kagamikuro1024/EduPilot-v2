# QC test case — US-PROTO-02 (Giảng viên / trợ giảng: vận hành lớp)
Nguồn: `docs/sprints/1.5/spec/US.md` + `SRS.md` 4.1, 4.4. Hộp đen. Công cụ: **C** = `proto-curl.sh`, **B** = trình duyệt. Đặt lại dữ liệu demo trước chuỗi TC.

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
| TC-02-16 | AC5 | GV trên desktop (1440 px) | **B** Mở `/inbox` | Bố cục split view 2 panel độc lập (FR-X11, Proposal #16): Cột danh sách ticket (bên trái, max 380px) và Panel chi tiết ticket (bên phải), đều có viền `1px solid var(--ep-rule)`, nền `var(--ep-surface)`, bo góc (`var(--radius-sm)` hoặc `var(--radius-md)`), padding rõ ràng (`var(--space-4)` hoặc `var(--space-5)`), cuộn độc lập giữa 2 cột |
| TC-02-17 | AC5 | 375 × 812 | **B** `/inbox` trên mobile | Chuyển thành danh sách → chi tiết (hai bước, nút quay lại); chữ không tràn; vùng chạm ≥ 44 px |
| TC-02-18 | AC5 | 375 × 812 | **B** `/attendance` trên mobile | Hàng có kẻ, **không card**, vùng chạm ≥ 44 px, thao tác bằng chạm; không cuộn ngang |
| TC-02-19 | AC6 (nhánh lỗi) | `/attendance` | **B** Bật "Giả lập mất mạng", đổi 2 ô | "Đang chờ mạng · 2 thay đổi"; tắt công tắc → "Đã lưu 09:2x", không mất thay đổi, không trùng |
| TC-02-20 | AC6 | – | **B** `/students?state=error` | Lỗi có câu khắc phục + `Thử lại` |
| TC-02-21 | AC7 (phân quyền) | – | **C** `tc_02_07` | SV, Admin bị chặn mọi route của US; TA không có `Tạo lại mã`, không có `Mời ra khỏi lớp` |
| TC-02-22 | AC7 | TA | **B** `/class/members` lớp 2 | TA xem được danh sách, `Duyệt` được; thiếu `Tạo lại mã` và mời ra |
| TC-02-23 | SRS 4.4 `/students` | GV | **B** Tìm "Huy"/"20229003"/"zzzz"; chip `Cần chú ý`, `Vắng nhiều`, `Điểm giảm`, `Ít hoạt động` | 30 SV lớp 1; chip `Cần chú ý` = C + 2; tìm theo tên/MSSV; không khớp → rỗng có hướng; bấm hàng → `/students/[id]`, quay lại giữ vị trí cuộn |
| TC-02-24 | SRS 4.4 `/students/[id]` | GV | **B** `/students/sv-3` (C) | Câu rủi ro "Vắng 5/9 buổi, đã bị trừ 1,5 điểm; thiếu BT02"; tab Tổng quan / Chuyên cần / Điểm / Hoạt động học / Ghi chú; ghi chú "Chỉ giảng viên/TA thấy" |
| TC-02-25 | SRS 4.4 | GV | **B** `Thêm ghi chú`; `Nhắn riêng` | Ghi chú thêm tại chỗ + Hoàn tác (không modal); `Nhắn riêng` → dòng xác nhận mô phỏng |
| TC-02-26 | **Quyền chat riêng** (CLAUDE.md) | Sau D1–D3 của B | **B** GV ở mọi route (`/inbox`, `/students/sv-2`, `/`, `/analytics`) | Không thấy nội dung D1/D2 (chat riêng chưa escalate); chỉ thấy ticket D3 (đã escalate) với câu hỏi D3 |
| TC-02-27 | SRS 4.4 trạng thái | – | **B** `?state=` trên 6 route; Lọc Open rỗng ở `/inbox` | Rỗng "Không còn câu hỏi đang chờ"; loading skeleton đúng hình |
| TC-02-28 | AC8 | GV / TA ở `/threads/[id]` có câu trả lời AI `Chờ xác nhận` | **B** Bấm `Xác nhận` | Trạng thái câu trả lời chuyển thành `Đã được giảng viên xác nhận` (verified) |
| TC-02-29 | AC8 | GV / TA ở `/threads/[id]` có câu trả lời AI `Chờ xác nhận` | **B** Bấm `Chỉnh sửa` | Mở ô sửa inline chứa sẵn nội dung câu trả lời AI |
| TC-02-30 | AC8 | Sau TC-02-29 | **B** Sửa nội dung và bấm `Lưu và xác nhận` | Trạng thái chuyển thành `Đã được giảng viên sửa & xác nhận` (CORRECTED), hiển thị nội dung đã sửa kèm nút/chi tiết "Xem câu trả lời AI gốc" để đối chiếu (Proposal #15) |
| TC-02-31 | AC8 | GV / TA ở `/threads/[id]` có câu trả lời AI | **B** Bấm `Loại khỏi tri thức` | Câu trả lời AI bị ẩn/loại bỏ khỏi thread và hiển thị dòng Hoàn tác |
| TC-02-32 | AC9 (#17c) | GV, 1440 × 900, `/inbox` | **A** "02-AC9 chip", AUDIT `/inbox` 1440; **B** | Hàng tab / chip lọc cuộn ngang (`data-scroll-x`), đọc đủ "Tất cả 6" (và Đang chờ, Đã nhận, Đã trả lời); `cut: []`; không chữ nào bị cắt ngang |
| TC-02-33 | AC9 | GV, 1440 | **A** đo cột danh sách; **B** ảnh | Cột danh sách ≤ 380 px, không phần tử vượt cột; mỗi ticket: dòng 1 tên + thời gian, dòng 2 câu hỏi tối đa 2 dòng (ticket dài của Lý Gia Thảo bị cắt đúng 2 dòng, không tràn), dòng 3 meta xuống dòng được (không cắt giữa chữ như v4 "…bấm Nhờ giảng viên hỗ trợ") |
| TC-02-34 | AC9 | GV, 375 và 390 px | **A** `/inbox` | Chỉ thấy danh sách; bấm ticket (Lý Gia Thảo / D3) → chi tiết toàn bề rộng, URL có `?ticket=<id>`, có `← Hộp thư` cao và rộng ≥ 44 px, danh sách ẩn; AUDIT `cut: []` ở 375 |
| TC-02-35 | AC9 | GV, 375 và 390 px | **A** cuộn danh sách 250 px → mở ticket → `← Hộp thư`; lặp lại với Back của trình duyệt | Về danh sách **đúng vị trí cuộn** (±2 px) cả hai cách; URL bỏ `?ticket=` |
| TC-02-36 | AC9 (biên) | GV | **A** 1100 → 1099 px; **B** mở thẳng `/inbox?ticket=<id>` ở 1440 và 375, tải lại ở 375 khi đang xem chi tiết | 1100: hai panel; 1099: chỉ danh sách rồi chi tiết (ngưỡng SRS 4.7 c2). 1440 + `?ticket=`: ticket đó được chọn sẵn; 375: tải lại vẫn ở chi tiết đúng ticket |
| TC-02-37 | AC9 (nhánh lỗi) | GV | **B** `/inbox?ticket=khong-co`; `/inbox?state=error` ở 375 | `?ticket=` sai → quay về danh sách hoặc thông báo rỗng có hướng, không trắng trang, không lỗi console; `?state=error` có `Thử lại`, không vỡ bố cục |
| TC-02-38 | AC9 | GV, 375 | **B** Chip lọc: Đang chờ → Đã nhận → Đã trả lời | Hàng chip cuộn ngang, chip đang chọn tự cuộn vào khung; số trên chip đủ chữ; không cắt |
| TC-02-39 | AC10 (#17d) | GV, 1440 | **A** "02-AC10" cho 5 ticket: Nguyễn Minh Trung, Đặng Gia An, Lê Quang Huy, Đỗ Thanh Long, Lý Gia Thảo | `[data-part=inbox-reply]` cách phần tử liền trước ≤ 24 px (v4: 100 px); `Gửi trả lời` cách ô soạn ≤ 24 px; ảnh ticket Lý Gia Thảo |
| TC-02-40 | AC10 | GV, 1440 | **B** Ticket ngắn (D3) so với ticket có bản nháp AI dài | Panel chi tiết cao theo nội dung, không kéo giãn khối thông tin để lấp chiều cao; không khoảng trống lớn trước hoặc sau ô trả lời |
| TC-02-41 | AC10 | GV | **A** sau `Nhận`; sau `Gửi trả lời` (Answered) | Khoảng cách ≤ 24 px được giữ ở trạng thái Claimed; ở Answered dòng "Đã gửi thư thông báo… (mô phỏng)" nằm sát phía dưới, không khoảng trống |
| TC-02-42 | AC10 (biên) | GV | **B** Ticket "đã có người nhận" (Phạm Quốc Bảo, 09:12) | Thay nút `Nhận` bằng dòng "đã nhận lúc 09:12"; ô "Trả lời của bạn" vẫn cách khối trên ≤ 24 px |
| TC-02-43 | AC11 (#18) | GV, sau `Đặt lại dữ liệu`; B vừa tạo thread khớp mẫu | **T** `T1j`, `T1k`, `T1l` (hoặc **B**: B tạo "Dùng lại IV trong CTR có sao không?" → đổi vai GV) | Hôm nay: "7 việc cần xử lý hôm nay"; có việc "Câu hỏi mới: «Dùng lại IV trong CTR có sao không?» · Mật mã đối xứng · vừa xong" nhãn `Chờ xác nhận`; chuông: "Câu hỏi mới trong Threads: «…»" |
| TC-02-44 | AC11 | Sau TC-02-43 | **T** `T1m`, `T1n` | Bấm việc → `/threads/<id>`; `Xác nhận` → việc rời Hôm nay, tiêu đề "6 việc cần xử lý hôm nay" |
| TC-02-45 | AC11 (nhánh không khớp) | B vừa tạo "WPA3 chặn được KRACK không?" | **T** `T4e`, `T4f` | Hôm nay "7 việc", nhãn `Cần giảng viên trả lời`; GV gửi phản hồi trong thread → việc rời, "6 việc" |
| TC-02-46 | AC11 / H (TA) | TA, cùng trạng thái TC-02-43 và TC-02-45 | **T** `T4g`; **B** | Trợ giảng thấy giống GV: việc, nhãn, đếm, chuông; thao tác của TA làm việc rời |
| TC-02-47 | AC11 / H | GV, sau TC-02-43 | **B** `Chỉnh sửa` → `Lưu và xác nhận`; làm lại ở thread khác với `Loại khỏi tri thức` | Mỗi thao tác (Xác nhận / Chỉnh sửa / Loại / gửi phản hồi) đều làm việc rời Hôm nay và đếm giảm 1 |
| TC-02-48 | AC11 (biên) | GV, sau `Đặt lại dữ liệu demo` | **B** B tạo 2 thread khớp → đếm; Xác nhận một thread mới; Xác nhận `t-cbc` của seed; `Đặt lại dữ liệu demo` | 8 → 7 → 6 (việc seed "1 câu trả lời của AI chờ bạn xác nhận" rời độc lập với việc mới); sau Đặt lại: 6, thread mới biến mất khỏi `/threads` |
| TC-02-49 | AC11 / phân quyền | SV B, D | **T** `T1o`; **B** | SV không thấy việc "Câu hỏi mới" của GV ở `/`; GV không thấy việc của SV; việc dành cho GV/TA không lộ nội dung chat riêng |


## Công cụ bổ sung (spec v5)
**A** = `scripts/audit.mjs` (phép đo "02-AC9", "02-AC10"); **T** = `scripts/threads-timeline.mjs` (các bước `T1j`…`T1o`, `T4e`…`T4g`).

## Nhánh lỗi
TC-02-07 (409 giả lập), TC-02-18 (mất mạng), TC-02-19, TC-02-26 (`?state=`), TC-02-11 (hoàn tác); spec v5: TC-02-36, 37 (`?ticket=` sai / tải lại), 45 (nhánh không khớp), 48 (đếm khi nhiều thread).

## Phân quyền
TC-02-20, 21 (SV/Admin/TA); TC-02-25 (GV không đọc chat riêng chưa escalate — luật CLAUDE.md); TC-02-23 (ghi chú chỉ GV/TA).

## Kiểm chéo
- Bàn phím và focus: TC-02-09 chỉ dùng bàn phím; kiểm Tab/Shift+Tab, focus thấy rõ trên `/attendance` và Dialog.
- Phản mẫu §21: `/` không tường KPI (TC-02-03); `/attendance` không card (TC-02-17); ≤ 1 nút chính đỏ mỗi vùng.
- 375 px: `/inbox`, `/attendance` bắt buộc; các route khác 390 px không vỡ.

## Điểm khó kiểm
- "Toàn bộ ≤ 60 s" phụ thuộc người bấm: ghi thời gian thực tế trong report (bấm giờ bằng script `demo-run.mjs` khi có).
- Phụ thuộc chéo: TC-02-05 cần D3 (US-PROTO-01); TC-02-13/14 cần yêu cầu của D; TC-02-43…49 cần thread do B tạo (US-PROTO-01 TC-01-39 / TC-01-79).
- AC9 / AC10 đo ở trình duyệt thật (Console); `data-part` `inbox-list`, `inbox-detail`, `inbox-reply` do dev thêm — thiếu móc = FAIL.
- Spec chưa nói thứ tự của việc "Câu hỏi mới" trong danh sách 7 việc (02-AC11) nên TC chỉ kiểm có mặt, nhãn và số đếm.

## Lịch sử sửa TC (chỉ khi SPEC đổi: ngày, TC nào, lý do)
- 2026-10-01 · TC-02-16..18: Cập nhật kiểm tra cấu trúc 2 panel độc lập (viền, nền surface, bo góc, cuộn độc lập) của `/inbox` (02-AC5); thêm TC-02-28..31 kiểm duyệt câu trả lời AI ở Threads (Xác nhận, Chỉnh sửa inline lưu CORRECTED, Loại bỏ có Hoàn tác) theo spec v4 (02-AC8, proposals #15, #16).
- 2026-10-02 · Thêm TC-02-32…49 (không sửa TC cũ): spec v5 — 02-AC9 (#17c `/inbox` danh sách + mobile `?ticket=`), 02-AC10 (#17d khoảng trống chi tiết), 02-AC11 (#18 việc "Câu hỏi mới" ở Hôm nay, 6 ↔ 7).
