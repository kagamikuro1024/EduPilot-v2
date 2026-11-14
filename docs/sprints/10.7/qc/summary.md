# QC sprint 10.7 (UI-R) — kết luận

Nhánh `sprint/10.7-ui-r`, HEAD kiểm `a99a84e9` (dev xong sau TL mục 14, kể cả `/settings` sinh viên). Stack `ep-qc`: chỉ rebuild frontend (env QC: `NEXT_PUBLIC_API_URL=http://localhost:580`), DB giữ nguyên. Chỉ đọc, không sửa mã dev.

**Kết luận: CHƯA đạt «0 lệch», còn 40 lượt ở 8 mã đo (trước đó 191 lượt, 15 mã).** Đã đạt 0 lệch: H3, H6, H7, E5, E7 (3 route lệch thật đã sửa), D1, T2 và toàn bộ nhóm cứng W, H1–H5, H8, P1, P3, P4, E1–E4, L1, L3, X2, T1, C3. Trang giới thiệu: mọi AC đo được đều đạt. P-107-14 đạt.

## 1. Verdict

| Hạng mục | Verdict | Căn cứ |
| --- | --- | --- |
| Hợp đồng khung §23 — K1 (kiểu trang, độ rộng, panel) | **FAIL (Thấp)** | W1–W3, P1, P3 = 0. P2 còn 2 lượt: `/insights` ở 375 dư +2,1 px (GV, TA) |
| K2 (đầu trang) | **PASS** | W4, H1–H7 = 0 |
| K3 (tiêu đề mục) | **PASS** | P4 = 0 |
| K4 / K5 (trạng thái rỗng, nút trùng) | **FAIL (Thấp)** | E1, E3–E5 = 0, E7 = 0. E2 (`student/chat` rỗng lệch tâm −397 px / −187 px) và E6 (3 route) còn lệch |
| K6 (375 px) | **PASS** | W5 = 0, 0 tràn ngang, H8 = 0 |
| Chuẩn đo mới (D1–D3, P5) | **FAIL (Thấp)** | D1 = 0. D3 còn 12 lượt (6 route), D2 11 lượt (4 route), P5 2 lượt (`admin/observability`) |
| Đang tải / lỗi (L1–L3, X1–X3) | **FAIL (Thấp)** | L1, L3, X2 = 0. L2 = 0, X1 4 lượt, X3 4 lượt |
| Lô 1 đo lại (`/insights`, `/questions`, `/inbox`) | **PASS có lệch nhỏ** | `/inbox`, `/questions` 0 lệch. Còn `/insights` ở 375 (P2 +2,1 px). `/insights` ở lớp rỗng: «Chưa có báo cáo», không số liệu lớp khác |
| Lỗi dữ liệu mô phỏng ở lớp rỗng | **PASS** | `/analytics` ở QCEMPTY: «Chưa có số liệu» (trước đó hiện số liệu lớp 761987) |
| `/settings` sinh viên khi lỗi API (TL 14.7) | **PASS** | 3 mục: «Mật khẩu» dùng được (ô nhập, nút hiện mật khẩu), «Thiết bị đang đăng nhập» 1 alert, «Thư» 1 alert; «Nhắc 24 giờ» và «Thời gian học» không vẽ gì, không nút/ô nào dùng được trên dữ liệu chưa tải; tổng 2 alert ≤ 4 GET hỏng khác nhau |
| **P-107-14** thanh dính đáy 375 | **PASS** | xem 3. Riêng `/attendance` đang tải / lỗi ở 375: dock mất (X3, xem 2) |
| **US-UX91-08** trang giới thiệu v2, AC1–24 | **PASS** (đo được) | xem 4; AC18, AC20, AC21 thuộc e2e dev, QC không chạy lại |
| Bộ màu ĐHBK (P-107-8/10) | **Chưa kiểm hết** | C3 = 0 trên mọi lượt; `contrast.spec.ts` QC không chạy lại (dev báo xanh ở CI) |
| `ui-antipatterns.sh` | **PASS** | exit 0 (HEAD trước), `--selftest` 31/31, `scripts/ui-frame-debt.txt` chỉ còn dòng chú thích |

## 2. Lệch còn lại (40 lượt trong `audit.md`)

| Mã | Lệch | Route | Loại | Ghi chú |
| --- | --- | --- | --- | --- |
| D3 hàng bảng 48\|44 (±1) | 12 | `grading` 53; `documents` 57; `gradebook` 65 (GV, TA); ở 375: `students` 70/92, `gradebook` 65/83/84, `admin/observability` 62/80/81 | **Thật** | Đã bỏ hàng đệm của bảng ảo (≥ 200 px) và nới ±1 theo TL 1.2; ở 1440 `students`, `attendance` nay đạt (48/49) |
| D2 cao điều khiển | 11 | `student/chat` ô nhập 32; `teacher/gradebook` 32; `class/settings` 16 (GV, TA) | **Thật** | |
| P2 | 2 | `insights` @375 (GV, TA) +2,1 px | **Thật** | |
| P5 | 2 | `admin/observability`: mục→mục 60,8 (1440) và 127,9 (375) so với 48 | **Thật** | |
| E2 | 2 | `student/chat` rỗng: khối rỗng lệch tâm −397,5 / −187,5 | **Thật** | Khối rỗng đã có `data-ep-empty` nhưng nằm trong cột trái của `Split` |
| E6 | 3 | `admin/observability` 377; `student/chat` rỗng 654; `teacher/observability` rỗng 341 | **Thật** | |
| X1 khối lỗi | 4 | `student/chat`: rộng 355 (1440) / 0 (375) so với cột; `teacher/settings`: 2 alert hợp lệ (TL 14.7) nhưng rộng 750 / 317 so với cột 800 / 351 | **Thật** | `settings`: mỗi alert nằm trong đệm của panel nên hẹp hơn cột 50 / 34 px; TL 14.7: alert «thay chỗ panel, rộng = cột» → lệch thật, đã giao dev (PM 15/11) |
| X3 | 4 | `student/practice` đang tải mất tab (1440, 375); `teacher/attendance` đang tải và lỗi ở 375 mất thanh dính đáy | **Thật** | TL 14.6: dock 375 vẫn hiện, disabled |

**Không áp dụng (PM 15/11, TL 12.2):** L2 và X1 ở `analytics`, `observability` (màn mô phỏng, dev kiểm bằng `?state=` ở `frame.spec`); QC bỏ khỏi cột lệch (`/insights` đã vậy từ trước).

**Chưa đo (TL 14.3):** E7 ở `students` (lớp QCEMPTY vẫn có 1 sinh viên) cho GV, TA; `analytics` (màn mô phỏng, nợ sprint 11).

`audit.html` (lưới ảnh) và `audit-truoc.*` (trước) so được từng màn; ảnh sau ở `audit/sau/`. Cách đo đã chỉnh theo TL (ghi trong script): H7 gom theo (độ rộng, cao h1, dòng meta, hàng nút, hàng quay lại) (12.1, 14.5); H6 bỏ nút trong dock, trong form, trong `data-ep-action-list` (P-107-14, 14.1); X3 giữ h1, nhãn tab, kiểu trang và nút (số, kiểu, thứ tự), đo cả lúc đang tải (14.2, 14.6); X1 ≤ 1 alert mỗi mục, trang không mục đúng 1, tổng ≤ số GET hỏng khác nhau (14.7); E7 3 thật / 4 chưa đo / 11 không áp dụng (14.3); T2 chỉ chuỗi tĩnh (14.4); L/X bỏ qua màn không phát yêu cầu `/api/v1`. Cách chặn API khi đo lỗi / đang tải: chỉ giữ `/api/v1/auth/*`, `/api/v1/me`, `/api/v1/me/courses` (bản đầu giữ cả `/me/*` nên `/settings` không bao giờ vào lỗi: QC đã sửa, tái hiện ở `repro-settings-x1.md`).

## 3. P-107-14 thanh dính đáy (`shots/dock-log.txt`, `shots/misc-log.txt`)

| Kiểm | `/attendance` 375 | `/grading/[id]` 375 |
| --- | --- | --- |
| Có `data-ep-header-dock`; nút chính nằm trong dock; đầu trang không có primary | PASS (nút «Lưu điểm danh · 30 có mặt · 0 vắng», 343×44) | PASS (nút «Duyệt bài», 125×44) |
| Dock cố định, nằm ngay trên thanh điều hướng dưới (cách đáy 56 px ≈ thanh dưới 57 px), không đổi sau cuộn | PASS | PASS |
| Nút ≥ 44 px | PASS | PASS |
| Chữ cuối trang không bị dock che | PASS | – |
| 1440: một nút chính ở đầu trang, không dock trùng việc | PASS | – |

Ảnh: `shots/attendance-375-bottom.png`, `shots/review-375.png`.

## 4. US-UX91-08 trang giới thiệu, AC1–24 (`landing/ac-log.txt`, `landing/landing-skill.log`, ảnh `shots/landing-*`)

| AC | Verdict | Đo được |
| --- | --- | --- |
| 1 | PASS | `h1` «Vận hành lớp học có trợ lý AI» (8 từ); dòng dẫn nguyên văn; mục cao 1 màn (4 mục) và khung ghim đúng 2,00 màn; 4 `h2` |
| 2 | PASS | đúng hai nút ở màn đầu: `Đăng nhập`, `Xem video`; skip link «Tới nội dung» |
| 3 | PASS | `role=img` + `aria-label` mô tả; 0 `img`/`video`/nền ảnh trong màn đầu |
| 4 | PASS | nút Tạm dừng 32×32, `aria-pressed` đổi, nhớ sau tải lại |
| 5 | PASS | 0 yêu cầu video trước khi bấm; mở `<dialog>`, focus vào hộp, `Esc` đóng, focus về nút, `preload=none`, bỏ nguồn khi đóng, `#video-loi` có |
| 6 | PASS | khung `sticky`; ba tên màn; ảnh rộng 1278–1304 px; ảnh đổi theo cuộn |
| 7 | PASS | 1024 px và 390 px: 0 `sticky`, ảnh đều hiện |
| 8 | PASS | ba tên bước hiện sau cuộn. Chưa đo sợi chỉ 1 s, nhịp 500 ms |
| 9 | PASS | ba số đo nguyên văn nguồn (so không phân biệt hoa thường; «Bộ phát hiện…» viết hoa đầu câu); số 146 đếm (giữa chừng 91, cuối 146); số lớn 36 px (≤ 40) |
| 10 | PASS | hai tab, mũi tên đổi tab, ảnh vai đổi; 390: ảnh vai nằm dưới danh sách |
| 11 | PASS | `h1` 72/38 px, `h2` 48/30 px; chữ thân ≥ 16 px (chữ 13–15 px chỉ nằm trong ảnh minh hoạ giao diện) |
| 12 | PASS | rê: nhích −1 px; bấm: `scale(0.985)` + lún 1 px |
| 13 | PASS có ghi chú | LCP là chữ của `h1` ở 1440 (28–88 ms); ở 390 `perf.mjs` báo đoạn dẫn (100 ms). JS 194,9 KB gz (≤ 256). Chưa đo Lighthouse 4G |
| 14 | PASS | `reduce`: 0 `m-on`, 0 animation chạy, 0 `sticky`, số 146 hiện ngay |
| 15 | PASS | không script: không khối chữ nào bị ẩn (trừ trạng thái cuối của đoạn sản phẩm: 7,5 → 8,5, Nháp → Duyệt) |
| 16 | PASS | axe 0 serious/critical ở 1440 và 390; liên kết chân trang ≥ 44 px; có 9 thẻ `og:*` |
| 17 | PASS | không `@edupilot.local` / `Edupilot#`; có dòng dữ liệu mô phỏng, tên tác giả, liên kết GitHub |
| 18, 20, 21 | Không chạy | thuộc e2e dev (`landing.spec.ts`, ảnh mốc) |
| 19 | PASS | 0 lỗi console |
| 22 | PASS | màn đầu có đúng một dữ kiện cứng, nguyên văn nguồn (số 146); không «1.000 sinh viên» |
| 23 | PASS | `copy-lint` 0 FAIL. `consistency.mjs` 11 FAIL (1440: 8; 390: 3) đều nằm trong danh sách miễn trừ của AC23: `scroll-reveal` ×6, `scroll-linked` ×1, `section-air` ×2, `empty-area` ×1, `copy-headings` ×1. `perf.mjs` 2 FAIL = «8 tệp phông» (miễn trừ). Số liệu từ lượt chạy ở HEAD `1935435a`, chưa chạy lại ở HEAD mới |
| 24 | PASS | (1) mục số đo có ảnh Sổ điểm ở 1440 và 390, khoảng trống liền ở 390 là 347 px (≤ 844); (2) ảnh vai dưới danh sách ở 390; (3) số lớn 36/24 px; (4) 0 `container-edges`; (5) `srcset` chọn 640 ở 390 |

## 5. Phương pháp và phạm vi

- Audit `sau`: 4 vai (admin, giảng viên, trợ giảng, sinh viên) × mọi route × 1440 / 375 × 4 trạng thái (có dữ liệu lớp 761987; rỗng lớp `QCEMPTY`; đang tải bằng chặn API 5 s; lỗi bằng 500). 310 lượt chụp, 0 lượt lỗi chạy. Script: `frontend/e2e/qc-ui-audit-real.spec.ts`, `scripts/audit_report.py`.
- Landing: `qc-landing-real.spec.ts` (đo DOM ẩn danh), `copy-lint`, `perf --runs 3`, `consistency` 1440 + 390 (HEAD `1935435a`). Dock: `qc-dock-real.spec.ts`. `/settings` sinh viên: `qc-settings-sv-real.spec.ts`.
- Chưa kiểm: `contrast.spec.ts` / `panels.spec.ts` / `frame.spec.ts` / `landing.spec.ts` của dev (CI của dev); tương phản chữ–nền theo bộ màu; Lighthouse 4G thật; `IntersectionObserver` (AC8 sợi chỉ, AC15 độ trễ 1,5 s); ảnh mốc `visual.spec` (dev sinh trên CI); `ui-antipatterns.sh` chưa chạy lại ở `a99a84e9`.
- Dọn: stack `ep-qc` đang chạy (frontend `a99a84e9`, env QC).
