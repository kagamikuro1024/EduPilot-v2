# QC test case — US-PROTO-04 (Hiểu lớp + hệ thống)
Nguồn: `docs/sprints/1.5/spec/US.md` + `SRS.md` 4.1, 4.6. Hộp đen. Công cụ: **C** = `proto-curl.sh`, **B** = trình duyệt. Đặt lại dữ liệu demo trước chuỗi TC.

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-04-01 | AC1 | – | **C** `tc_04_01` (GV: 5 route; Admin: 6 route) | Đều `MO` |
| TC-04-02 | AC1 | – | **B** Sweep GV 5 route + Admin 6 route 1440 px | Mỗi route một `h1`; khung nhìn đầu khớp DESIGN §14.21–14.25 / INTEGRATION mục 2; ảnh `shots/` |
| TC-04-03 | AC2 | GV, lớp 1 `/insights` | **B** Mở `/insights` | Báo cáo 22/10; chủ đề A "Mật mã đối xứng (AES, CBC)", B "Hàm băm và chữ ký số" đứng đầu, mỗi chủ đề có lý do, 5 tín hiệu, 3–5 câu mẫu **đã ẩn danh**; có một chủ đề "dưới 3 sinh viên hỏi — không hiện câu mẫu"; mục "Tài liệu chưa đề cập" |
| TC-04-04 | AC2 | – | **C** `tc_04_02`; **B** quét `innerText` sau mọi tương tác | Không `2022xxxx`, không "Trần Thu Uyên", "Lê Quang Huy", "Nguyễn Minh Trung", "Phạm Ngọc Linh" ở câu mẫu |
| TC-04-05 | AC2 | – | **B** `Tạo báo cáo mới` | Tiến độ ~3 s (không spinner giữa trang) → báo cáo 29/10 |
| TC-04-06 | AC2 | Sau TC-04-05 | **B** `Tạo thread ghim` | Thread ghim mới xuất hiện ở `/threads` lớp 1 (kiểm bằng vai GV và vai B); `Tạo buổi ôn tập` → sự kiện trong `/calendar` |
| TC-04-07 | AC2 | – | **B** Đổi lớp 2 | Chủ đề C "Tường lửa và phân đoạn mạng" đứng đầu |
| TC-04-08 | AC3 | Admin `/observability` | **B** Mở một yêu cầu | Drawer **bắt nhập lý do** trước khi hiện nội dung (không lý do → không hiện); nội dung đã che chỉ có `[[SV_1]]`, không tên thật; tool, độ tin cậy, link "Chi tiết yêu cầu" (giả); dòng "Đã ghi nhật ký kiểm toán" |
| TC-04-09 | AC3 | – | **B** Admin: lý do rỗng / toàn khoảng trắng | Không mở; báo lỗi tại ô |
| TC-04-10 | AC3 | GV | **C** `tc_04_03`; **B** GV ở `/observability` | Chỉ dải trạng thái + số tổng hợp lớp mình; hàng không mở được (không con trỏ link, không Drawer); không `[[SV_` |
| TC-04-11 | AC3 / SRS 4.6 | Admin | **B** Dải trạng thái + bảng | 1.284 yêu cầu hôm nay, p95 3,1 s, lỗi 0,6%, model dự phòng 1,2%, chi phí 42.000 đ; bảng 50 yêu cầu |
| TC-04-12 | AC4 | Admin `/settings/llm` | **B** `Test kết nối` theo hàng; đổi model tác vụ CHAT | "Kết nối được · 412 ms" tại chỗ; đổi model → dòng Hoàn tác (không modal); khoá API hiện "••••3f9a · đã kết nối" (không bao giờ hiện khoá đầy đủ, kiểm `innerText` + DOM `value`) |
| TC-04-13 | AC4 | – | **B** Đổi embedding | Cảnh báo "phải đánh chỉ mục lại" |
| TC-04-14 | AC4 | GV | **C** `tc_04_04`; **B** `/settings/llm` | Cùng màn nhưng không `Test kết nối`, không nút sửa/đổi model |
| TC-04-15 | AC5 (nhánh lỗi) | Admin | **B** Test provider lỗi mẫu | Lỗi nói rõ vấn đề + cách khắc phục (DESIGN §15); không mã lỗi trần |
| TC-04-16 | AC5 | – | **B** `/insights?state=empty` | Rỗng + `Tạo báo cáo mới` |
| TC-04-17 | AC6 (phân quyền) | – | **C** `tc_04_06` | SV, TA bị chặn 5 route hệ thống; GV bị chặn `/admin/courses`; TA ở `/analytics` không có "chi phí" |
| TC-04-18 | AC6 / SRS 2 | – | **C** `tc_00_matrix` hàng `/observability`, `/settings/*`, `/admin/*` | GV R ở `/observability`, `/settings/*`; Admin mở `/admin/*`; GV bị chặn `/admin/*` |
| TC-04-19 | SRS 4.6 `/analytics` | GV; TA | **B** Đổi 7 / 30 ngày | GV thấy mục chi phí; TA không; biểu đồ SVG tự vẽ (không thư viện chart); đổi khoảng đổi số liệu |
| TC-04-20 | SRS 4.6 `/` Admin | Admin | **B** Mở `/` | 4 việc (Gemini lỗi 3 lần/15 phút; sắp chạm 80% ngân sách; 2 việc dead-letter; lớp 761988 chưa có giảng viên hoạt động); bấm việc → đúng màn; rỗng "Hệ thống đang vận hành bình thường" |
| TC-04-21 | SRS 4.6 `/admin/courses` | Admin | **B** `Mở lớp` (form tại chỗ, không modal) → chọn giảng viên; `Lưu trữ` | Lớp mới + "Đã gửi thông báo phân công"; `Lưu trữ` qua xác nhận (Dialog danh sách trắng) |
| TC-04-22 | SRS 4.6 `/admin/users` | Admin | **B** Lọc vai; `Mời giảng viên`; khoá/mở khoá | 57 SV + GV, TA, Admin; "Đã gửi link mời, hạn 72 giờ"; khoá/mở khoá có Hoàn tác; **không có "tạo sinh viên"** |
| TC-04-23 | SRS 4.6 `/settings/integrations` | Admin; GV | **B** `Gửi thư thử` / `Kiểm tra` | Mail đã kết nối (kiểm 08:55), IMAP chưa cấu hình, Teams "cần quản trị viên trường đồng ý" bằng lời; kết quả tại chỗ; GV chỉ xem |
| TC-04-24 | Trạng thái | – | **B** `?state=loading|empty|error` 8 route | Skeleton đúng hình, rỗng có một hành động, lỗi có `Thử lại` |
| TC-04-25 | **Chéo: nội dung prompt chỉ ADMIN, có audit** (CLAUDE.md) | – | **B** GV/TA tìm nội dung yêu cầu ở mọi màn (`/observability`, `/analytics`, `/inbox`) | Không thấy nội dung prompt/chat riêng; chỉ Admin sau khi nhập lý do |
| TC-04-26 | AC7 (#17g) | Admin, `/settings/llm`, 1440 | **A** "04-AC7 cột thẳng hàng"; **B** Console của US 04-AC7 | `['provider-status','provider-action'].map(… Set(left).size)` → `[1, 1]` với ≥ 3 hàng (OpenAI, Gemini, Máy chủ trong trường); v4 lệch x = 693 / 716 / 764; ảnh |
| TC-04-27 | AC7 | Như TC-04-26 | **A** "04-AC7 lưới cố định" | Cột trạng thái rộng 200 px, cột nút 140 px (±1) ở mọi hàng (lưới `minmax(0, 1fr) 200px 140px`); dòng "Kiểm gần nhất: 09:05 hôm nay · 3 lần lỗi trong 15 phút" của Gemini (dài nhất) xuống dòng trong cột, không tràn, không làm lệch hàng |
| TC-04-28 | AC7 | Admin, 390 và 375 px | **A** "04-AC7 hàng xếp dọc" | Mỗi hàng: thông tin → trạng thái → nút theo thứ tự trên xuống; nút rộng 100 % hàng, cao ≥ 44 px; dòng hạn mức "Tháng 10 · 1.240.000 đ / 2.000.000 đ" xuống dòng, không `…`; AUDIT `ell: []`, `cut: []`; ảnh |
| TC-04-29 | AC7 (biên) | Admin | **A** AUDIT `/settings/llm` ở 1100, 1024, 720, 719, 600 px | Không `…` không `title`, không cắt, không tràn ngang ở mọi bề rộng; 1440 → 390 không có bề rộng nào làm hai cột chồng lên nhau |
| TC-04-30 | AC7 / AC4 | GV (chỉ xem), 1440 và 390 | **A** `provider-status` thẳng hàng; **B** | Cột trạng thái vẫn thẳng hàng khi không có nút sửa/test; không có khoảng trống cột nút kéo dài lạ; không `Test kết nối` (US 04-AC4) |
| TC-04-31 | AC7 / AC4 | Admin | **B** `Test kết nối` ở từng hàng (kể cả Gemini lỗi); đổi model tác vụ CHAT; đổi khoá | Kết quả hiện tại chỗ ("Kết nối được · 412 ms" / lỗi có cách khắc phục) và dòng Hoàn tác **không làm lệch** các cột (đo lại `[1, 1]`); ở 390 kết quả dài xuống dòng, không tràn |
| TC-04-32 | AC7 | Admin, 390 | **B** Khoá API ở hàng: "••••3f9a · đã kết nối" | Chữ khoá che vẫn đủ, không `…`; nhãn "Kiểm gần nhất" không bị cắt |
| TC-04-33 | 00-AC10 h1 (US-04) | GV, Admin; 390 và 375 | **A** `/observability` (GV, Admin) | `ox` = 0 (v4: 34 px); bảng 50 yêu cầu của Admin đọc được (cuộn ngang trong vùng `data-scroll-x` hoặc dạng danh sách), `cut: []`; GV chỉ thấy dải trạng thái + số tổng hợp |
| TC-04-34 | 00-AC10 h3 (US-04) | Admin, 390 và 719 | **A** `/admin/courses`, `/admin/users` | Dạng danh sách (không `<table>`), chip trạng thái và vai đủ chữ ("Đang học", "Mới mở"), `ox` = 0 (v4 `/admin/users` 101 px); hàng có `…` mở menu `Lưu trữ` / khoá dùng được bằng chạm ≥ 44 px |


## Công cụ bổ sung (spec v5)
**A** = `scripts/audit.mjs` (phép đo "04-AC7"), `proto-curl.sh` `tc_04_07` (có `data-part` `provider-status` ≥ 3, GV không có `Test kết nối`).

## Nhánh lỗi
TC-04-09, 15, 16, 24; spec v5: TC-04-29 (mọi bề rộng), 31 (kết quả test lỗi không làm vỡ lưới).

## Phân quyền
TC-04-10, 14, 17, 18, 25 (GV R, TA bị chặn, SV bị chặn; Admin có lý do + audit).

## Kiểm chéo
- Không PII: TC-04-04, 08 (placeholder `[[SV_n]]`, không tên thật). Không thẻ KPI wall: `/observability` có dải trạng thái một dòng, không ≥ 3 thẻ cùng cỡ (DESIGN §21) — xem ảnh.
- Đỏ chỉ là tín hiệu: màu đỏ trong `/observability`, `/` Admin chỉ ở lỗi / cần làm.
- 390 px: các route này chỉ cần không vỡ.

## Điểm khó kiểm
- Biểu đồ SVG tự vẽ: kiểm bằng ảnh, không tính chính xác số.
- "Khoá API chỉ ghi": phải xem DOM (không chỉ chữ hiển thị).
- AC7: lưới `200px / 140px` ghi trong SRS 4.7 g1 (không có trong US); TC-04-27 chấm theo SRS 4.7. Hàng 390 px "nút rộng 100 %" đo bằng so sánh với bề rộng hàng, cho sai số 32 px lề trong.
- Sinh viên / TA không mở `/settings/llm` nên TC-04-26…32 chỉ cần Admin và GV.

## Lịch sử sửa TC (chỉ khi SPEC đổi: ngày, TC nào, lý do)
- 2026-10-02 · Thêm TC-04-26…34 (không sửa TC cũ): spec v5 — 04-AC7 (#17g `/settings/llm` lưới cột 1440 / xếp dọc 390), cộng 00-AC10 h1 / h3 cho `/observability`, `/admin/courses`, `/admin/users` (SRS 4.7 g1, g2, h1, h3).
