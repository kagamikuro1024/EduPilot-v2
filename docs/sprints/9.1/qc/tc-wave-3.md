# TC đợt 3 — Hôm nay (US-UX91-10, 11)

Phiên bản 1 · 2026-11-08. Quy ước và kiểm chung: `tc-00-chung.md`.
Dữ liệu: QC chặn `GET /me/today` và `GET /courses/{id}/today` (`page.route`) với phản hồi theo hình đã có ở `frontend/e2e/support/screens.ts` (`mockStaffToday`: `{count, actions[{id, kind, title, reason, urgency, href, course, steps}], attention[], upcoming[]}`; sinh viên: `{no_course, email_verified, continue[], recommended, timeline[]}`). Các ca không cần dữ liệu cố định chạy trên lớp 761987 thật của stack `ep-qc`.

Fixture giảng viên **F-GV** (thứ tự máy chủ giữ nguyên):
| id | kind | urgency | title |
| --- | --- | --- | --- |
| a1 | `AI_PENDING` (câu hỏi chờ giảng viên) | `overdue` | «Câu hỏi chờ giảng viên» |
| a2 | `REVIEW_FLAGGED` (bài AI cần xem kỹ) | `high` | «4 bài cần xem kỹ» |
| a3 | `JOIN_REQUESTS` | `high` | «1 yêu cầu có email chưa khớp MSSV» |
| a4 | `REPORTED_POSTS` | `normal` | «2 bài bị báo cáo chờ xem» |
| a5 | `EXAM_SIMILARITY` | `normal` | «1 cặp bài code nghi giống nhau» |
| a6 | `UNMATCHED_SUBMISSION` | `normal` | «3 bài nộp chưa khớp sinh viên» |
| a7 | `QUESTION_REVIEW` | `normal` | «5 câu hỏi chờ duyệt» |
(`kind` lấy đúng tên của stack; nếu khác tên, QC đổi theo phản hồi thật quan sát được và ghi vào báo cáo; mục tiêu: có ≥ 1 `overdue`, ≥ 2 `high`, ≥ 2 `normal` thường, `EXAM_SIMILARITY`, `UNMATCHED_SUBMISSION`.)

---
## US-UX91-10 — Hôm nay giảng viên: nhóm việc

| TC | AC | Vai / cỡ | Bước | Kết quả mong đợi | Chạy |
| --- | --- | --- | --- | --- | --- |
| TC-UX10-1 | AC1 | giảng viên, 1440 | F-GV; mở `/`; đọc `h3` và các hàng dưới mỗi `h3` | Ba nhóm theo thứ tự **Gấp · 3** (a1, a2, a3) → **Hôm nay · 2** (a4, a7) → **Khi rảnh · 2** (a5, a6); thứ tự hàng trong nhóm giữ nguyên thứ tự máy chủ | E2E |
| TC-UX10-2 | AC1 | giảng viên | F-GV bỏ a5, a6 | Không có nhóm «Khi rảnh» (nhóm rỗng không hiện) | E2E |
| TC-UX10-3 | AC1 | giảng viên | F-GV chỉ có `normal` thường (a4, a7) | Chỉ nhóm «Hôm nay · 2» | E2E |
| TC-UX10-4 | AC1 | giảng viên | `EXAM_SIMILARITY` với `urgency=high` | Vào «Gấp» (không vào «Khi rảnh»: chỉ `normal` mới vào Khi rảnh) | E2E |
| TC-UX10-5 | AC1 | giảng viên | Quan sát request | Không thêm request / API mới; không ngưỡng giờ mới ở client (đọc `urgency` từ phản hồi) | E2E |
| TC-UX10-6 | AC2 | giảng viên, 1440 | F-GV; đếm nút đỏ trong khung 1440×900; đọc nhãn nút của a1 | Đúng **1** nút đỏ, thuộc a1 (việc đầu của nhóm đầu); nhãn động từ theo `CTA[kind]` | E2E |
| TC-UX10-7 | AC2 | giảng viên | Các hàng còn lại | Nút `ghost` + biểu tượng mũi tên (`iconEnd`); không chữ đỏ ở mọi hàng; không biến thể `link` | E2E |
| TC-UX10-8 | AC2 | giảng viên | F-GV bỏ a1, a2, a3 (không còn Gấp) | Nút đỏ chuyển sang việc đầu của «Hôm nay» (nhóm đầu không rỗng) | E2E |
| TC-UX10-9 | AC2 | giảng viên | `bash scripts/ui-antipatterns.sh` | Sạch (phép «hơn một nút đỏ» không bắt) | GREP |
| TC-UX10-10 | AC3 | giảng viên, 1440 | Chọn một lớp (761987); `/` | Không còn «lớp 761987» ở từng hàng và dòng phụ; không khối con «Việc cần xử lý» trùng tiêu đề; `h1` «N việc cần xử lý hôm nay» (N = `count`) | E2E |
| TC-UX10-11 | AC3 | giảng viên | Chọn «Tất cả lớp của tôi» | Mã lớp hiện ở hàng (`classLabel`) và ở phạm vi tiêu đề phụ | E2E |
| TC-UX10-12 | AC4 | giảng viên | Đọc màu chấm: `getComputedStyle` mỗi hàng | `overdue` đỏ; `high` hổ phách; `normal` xám | E2E |
| TC-UX10-13 | AC4 | giảng viên | Cây truy cập (`page.accessibility.snapshot()` / `getByRole('heading')`) | Mỗi nhóm có tên truy cập «Gấp, 3 việc» (vùng `aria-labelledby` hoặc `h3` kèm số) đọc **trước** hàng đầu; mức khẩn không chỉ bằng màu | E2E |
| TC-UX10-14 | AC5 | giảng viên, 1440 | F-GV + `attention` 2 mục + `upcoming` 1 mục | «Lớp cần chú ý» và «Sắp tới» ở cột phải, rộng ≤ 320 px, cạnh danh sách (cùng hàng ngang) | E2E |
| TC-UX10-15 | AC5 | giảng viên, 1099 | như trên | Hai khối nằm **dưới** danh sách | E2E |
| TC-UX10-16 | AC5 | giảng viên | `attention_total` > số hiện | Nút «Xem tất cả (N)» còn và dẫn đúng | E2E |
| TC-UX10-17 | AC6 | giảng viên | Bấm một hàng việc, đo thời gian Red Thread | < 300 ms; `reduce` → không chạy; không thêm chuyển động khác | E2E |
| TC-UX10-18 | AC7 | giảng viên | Huỷ API lần đầu; mở `/` | Thông báo lỗi + `Thử lại`; bấm khi bỏ chặn → tải được | E2E |
| TC-UX10-19 | AC7 | giảng viên | Tải thành công rồi huỷ lần làm mới (refetch) | Dải «Chưa làm mới được…»; giữ nguyên nhóm / nhãn đã hiện | E2E |
| TC-UX10-20 | AC8 | giảng viên | `actions: []` | `EmptyState` «Bạn đã xử lý hết việc.»; không nhóm trống | E2E |
| TC-UX10-21 | AC9 | giảng viên | `count: 40`, `actions` 15 mục | Dòng «Đang hiện 15 việc đầu tiên.»; nhóm vẫn đúng | E2E |
| TC-UX10-22 | AC10 | sinh viên | `/` | Hôm nay sinh viên; không `h3` Gấp / Hôm nay / Khi rảnh | E2E |
| TC-UX10-23 | AC10 | admin | `/` với việc hệ thống; rồi không có việc kiểu Khi rảnh | Hiện việc hệ thống; không nhóm «Khi rảnh» nếu rỗng | E2E |
| TC-UX10-24 | AC10 | TA | F-GV có việc `COURSE_SETUP` | TA thấy cùng nhóm như giảng viên; **không** nút bỏ qua thiết lập (`canDismiss` sai) | E2E |
| TC-UX10-25 | AC11 | giảng viên, 375 | `/` | Nhãn nhóm + hàng xếp dọc; nút hành động của hàng xuống dòng riêng; vùng chạm ≥ 44 px; K4 | E2E |
| TC-UX10-26 | AC12 | — | `today.spec.ts` diff + chạy; fixture có đủ `overdue`, `high`, `normal`, `EXAM_SIMILARITY` | Spec kiểm nhóm; không nới; thêm ảnh mốc `hom-nay-gv-1440`, `hom-nay-gv-390`; `home-*` (sinh viên) không đổi | E2E |
| TC-UX10-27 | AC13 | giảng viên | So `attention` / `upcoming` trước / sau (cùng dữ liệu) | Chỉ vị trí đổi (cột phải ≥ 1440), không thêm dữ liệu mới | E2E |
| TC-UX10-28 | (K7) | — | Chụp `hom-nay-gv` trước (`audit/shots/home__teacher__1440.png`) và sau | Đủ ảnh 1440 / 375 | MAC |
| TC-UX10-PQ | (kiểm chéo) | SV lớp khác / TA | TA mở `/` với lớp mình; GV không thấy việc của lớp người khác | Chỉ việc của lớp mình (`/me/today` theo JWT) | E2E |

---
## US-UX91-11 — Hôm nay sinh viên

Fixture **F-SV**: `recommended` (EXAM_OPEN hoặc việc kế tiếp), `timeline` 2 mục (một `NOW`, một `UPCOMING`), `continue` 5 mục: 3 cùng tiêu đề «Sâu Morris là gì…» cùng lớp (thời gian khác nhau), 1 «Ransomware hoạt động…» , 1 «Mỗi lần phát biểu…».

| TC | AC | Vai / cỡ | Bước | Kết quả mong đợi | Chạy |
| --- | --- | --- | --- | --- | --- |
| TC-UX11-1 | AC1 | sinh viên, 1440×900 | F-SV; `/` | Thứ tự trên DOM / trực quan: lời chào + ngày → «Việc nên làm tiếp» (một việc, một nút chính) → khối «Hôm nay» (timeline) → «Tiếp tục học» | E2E |
| TC-UX11-2 | AC1 | sinh viên | Đo `getBoundingClientRect()` khối «Hôm nay» | `top` < 900 (nằm trong khung nhìn đầu); tiêu đề «Hôm nay» hiện đầy đủ | E2E |
| TC-UX11-3 | AC2 | sinh viên | F-SV; đếm hàng «Tiếp tục học» | 3 hàng tối đa: «Sâu Morris…» **một** hàng (phiên gần nhất), «Ransomware…», «Mỗi lần phát biểu…» | E2E |
| TC-UX11-4 | AC2 | sinh viên | Hai phiên cùng tiêu đề nhưng **khác lớp** | Cả hai hiện (khoá trùng là cặp tiêu đề + lớp) | E2E |
| TC-UX11-5 | AC3 | sinh viên | Thời gian hàng «Tiếp tục học» = `2026-11-02T00:13+07` và hôm qua (đồng hồ cố định 2026-11-08 14:00) | «Thứ Hai, 00:13» / «Hôm qua, 00:13» theo US-02; **không** «Thứ 2, 2/11» | E2E |
| TC-UX11-6 | AC4 | sinh viên | Đếm nút đỏ trong khung 1440×900 | Đúng 1 (`Mở bài tập` hoặc tương đương) | E2E |
| TC-UX11-7 | AC5 | `sv.moi` (chưa vào lớp) | `/` | «Nhập mã tham gia lớp» như hiện tại; **không** khối «Hôm nay» trống | E2E |
| TC-UX11-8 | AC6 | giảng viên, TA | `/` | Không khối «Việc nên làm tiếp» / «Tiếp tục học» của sinh viên | E2E |
| TC-UX11-9 | AC7 | sinh viên, 375 | `/` | Cùng thứ tự; K4; thanh dưới «Hôm nay» active (`aria-current`) | E2E |
| TC-UX11-10 | AC8 | — | Chụp `home` 1440 / 390 trước (`audit/shots/home__student__*`) và sau | Đủ ảnh; `visual.spec` chỉ `home-*` đổi | MAC |
| TC-UX11-PQ | (kiểm chéo) | SV A | Danh sách «Tiếp tục học» | Chỉ phiên chat của chính SV (JWT); SV B không thấy | E2E |

---
## Bảng phủ AC đợt 3
| Story | AC | TC |
| --- | --- | --- |
| US-10 | AC1 | 10-1…5 |
| | AC2 | 10-6, 7, 8, 9 |
| | AC3 | 10-10, 11 |
| | AC4 | 10-12, 13 |
| | AC5 | 10-14, 15, 16 |
| | AC6 | 10-17 |
| | AC7 | 10-18, 19 |
| | AC8 | 10-20 |
| | AC9 | 10-21 |
| | AC10 | 10-22, 23, 24 |
| | AC11 | 10-25 |
| | AC12 | 10-26, 28 |
| | AC13 | 10-27 |
| US-11 | AC1 | 11-1, 2 |
| | AC2 | 11-3, 4 |
| | AC3 | 11-5 |
| | AC4 | 11-6 |
| | AC5 | 11-7 |
| | AC6 | 11-8 |
| | AC7 | 11-9 |
| | AC8 | 11-10 |
