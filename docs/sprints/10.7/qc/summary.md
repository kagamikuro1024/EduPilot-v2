# QC sprint 10.7 (UI-R) — kết luận

Nhánh `sprint/10.7-ui-r`, HEAD kiểm `1935435a` (dev xong lô 3–9). Stack `ep-qc`: chỉ rebuild frontend (env QC: `NEXT_PUBLIC_API_URL=http://localhost:580`), DB giữ nguyên. Chỉ đọc, không sửa mã dev.

**Kết luận chung: KHÔNG ĐẠT mục tiêu «0 lệch».** Phần đạt: các hợp đồng cứng về khung (W1–W3, W5, H1–H2, H4–H5, P1, P3–P4, E1–E4, L3, X2, T1, C3 = 0 lệch trên mọi route × vai × 1440/375 × 4 trạng thái), thanh dính đáy P-107-14, trang giới thiệu (trừ AC16 `og:`), `ui-antipatterns.sh`. Phần chưa đạt: 15 mã đo còn lệch (bảng 2), trong đó `K1/K2/K4` còn P2, H3, H6, E5–E7; nhịp hàng bảng D3 lệch ở 18 route.

## 1. Verdict

| Hạng mục | Verdict | Căn cứ |
| --- | --- | --- |
| Hợp đồng khung §23 — K1 (kiểu trang, độ rộng, panel) | **FAIL (Thấp)** | W1–W3, P1, P3 = 0. P2 (panel cao hơn nội dung) lệch 6 route |
| K2 (đầu trang) | **FAIL (Thấp)** | H1, H2, H4, H5, W4 = 0. H3 3 route (meta lặp lớp), H6 4 route |
| K3 (tiêu đề mục) | **PASS** | P4 = 0 |
| K4 / K5 (trạng thái rỗng, nút trùng) | **FAIL (Thấp)** | E1–E4 = 0, E5 1 route, E6 3 route, E7 1 route lệch thật (xem 2) |
| K6 (375 px) | **PASS** | W5 = 0, 0 tràn ngang, H8 = 0 |
| Chuẩn đo mới (H7, D1–D3, T2, P5) | **FAIL (Thấp)** | D3 18 route, D2 4, D1 2, H7 3, P5 1, T2 1 nhãn tĩnh (xem 2) |
| Đang tải / lỗi (L1–L3, X1–X3) | **FAIL (Thấp)** | L1 = 0, L3 = 0, X2 = 0. L2 3 route, X1 4 route, X3 7 route |
| Lô 1 đo lại sau `ff19b6cb` (`/insights`, `/questions`, `/inbox`) | **PASS có lệch nhỏ** | `/inbox` 0 lệch. 34 lượt `audit-lo1` → còn `/insights` P2 +2,1 px ở 375; `/questions` D3 (hàng 48/59/62), X3 |
| Lỗi dữ liệu `/insights` hiện báo cáo lớp khác | **PASS** | Lớp QCEMPTY: «Chưa có báo cáo», không số liệu 761987 (`audit/sau/teacher-empty-_insights-1440.jpg`) |
| **P-107-14** thanh dính đáy 375 (`/attendance`, `/grading/[id]`) | **PASS** | xem 3 |
| **US-UX91-08** trang giới thiệu v2, AC1–24 | **FAIL (Thấp)** | 1 AC lệch (AC16 thiếu `og:`), xem 4 |
| Bộ màu ĐHBK (P-107-8/10) | **Chưa kiểm hết** | C3 (vàng ≤ 3 nền, 0 chữ) = 0 trên mọi lượt; `contrast.spec.ts` QC không chạy lại (dev báo xanh ở CI) |
| `ui-antipatterns.sh` | **PASS** | exit 0; `--selftest` 31/31; `scripts/ui-frame-debt.txt` chỉ còn dòng chú thích |

## 2. Lệch còn lại (191 lượt trong `audit.md`; bảng gom theo mã và route)

Phân loại: **Thật** = dev sửa. **Quyết** = cần TL / PM quyết (QC đề xuất). **Không áp dụng** = đo trên dữ liệu hoặc màn mô phỏng, đã loại khỏi kết luận.

| Mã | Lệch | Route (vai × 1440/375) | Loại | Ghi chú |
| --- | --- | --- | --- | --- |
| D3 hàng bảng 48|44 | 29 | `students`, `gradebook`, `grading`, `questions`, `documents`, `class/members`, `attendance`, `observability` (GV, TA), `admin/courses`, `admin/users` | **Thật** | Hàng thực đo 49–62 px (375: 81–106). `DataTable` không được chỉnh ở lô nào; tiêu chí D3 có trong `techlead.md` 1.2 |
| D2 cao điều khiển | 11 | `student/chat` ô nhập 32; `teacher/gradebook` 32; `class/settings` 16 (GV, TA) | **Thật** | |
| D1 cỡ chữ | 8 | `class/settings` 52 px (1440), 32,2 px (375) | **Thật** | |
| H7 chiều cao đầu trang | 12 | `student/practice/history` (72/80 so 36), `gradebook/scheme` (102–110 so 63) | **Thật** | Sau khi đổi khoá gom theo TL 12.1 |
| P2 panel cao hơn nội dung | 13 | `student/me` +17, `analytics` +17 (GV, TA), `insights` +2,1 ở 375, `admin/observability` +2,1 | **Thật** | |
| H3 meta lặp lớp | 12 | `analytics` (GV, TA), `observability` (GV) | **Thật** | Meta chứa «QC lớp rỗng · QCEMPTY» |
| H6 nút primary ngoài đầu trang | 14 | `student/practice/history` rỗng «Luyện đề»; `/` GV, TA nút «Mở» ở hàng việc; `student/chat` «Gửi» | **Thật** (`practice`); **Quyết** (`Mở`, `Gửi`) | QC đề xuất miễn: nút gửi của ô nhập và nút hành động của `ActionList` |
| E5 nút rỗng secondary | 2 | `student/practice/history` | **Thật** | Cùng nguyên nhân H6 |
| E6 vật chứa rỗng ≤ 320 | 3 | `documents` rỗng 454 px (GV, TA); `admin/observability` 377 | **Thật** | |
| E7 rỗng tự dựng | 36 | 18 route | **Thật** ×1 (`student/` «Hôm nay bạn không có việc gấp.» là dòng chữ tự dựng); **Không áp dụng** ×17 | Lớp QCEMPTY vẫn có 3 thành viên nên `students`, `class/members` có dữ liệu; `settings`, `class/settings`, `settings/*`, `analytics`, `observability` không có khái niệm rỗng |
| P5 nhịp dọc | 2 | `admin/observability` (mục→mục 60,8 và 127,9 so 48) | **Thật** | |
| L2 khung xương | 6 | `settings`, `analytics`, `observability` (GV) | **Thật** | Các route này có gọi `/api/v1` nhưng không vẽ `aria-busy` |
| X1 khối lỗi | 8 | `settings`, `analytics`, `observability` (GV), `student/chat` | **Thật** | `alert×0` khi API trả 500 |
| X3 khung khi lỗi | 13 | `students`, `attendance`, `gradebook`, `gradebook/scheme`, `questions` (đầu trang thấp hơn do mất `meta`), `class/members` (tab mất số đếm), `student/practice` (mất tab) | **Thật** ×1 (`practice`), **Quyết** ×6 | QC đề xuất: không tính chênh vì `meta` / số đếm lấy từ dữ liệu |
| T2 câu lặp ≥ 3 lần | 22 | `documents` «Không dùng cho AI của sinh viên» ×4 | **Thật** ×1 | Còn lại là dữ liệu thật (IP phiên, tên bài tập ×6 ở hàng chờ chấm): **Không áp dụng** — đề xuất T2 chỉ đếm chuỗi tĩnh |

**Lỗi dữ liệu ghi thêm (Trung):** `/analytics` ở lớp `QCEMPTY` vẫn hiện số liệu của lớp 761987 («392 câu hỏi trong 7 ngày…», biểu đồ). Cùng loại lỗi đã sửa ở `/insights` (màn mô phỏng, dev đã ghi nợ sprint 11). `/observability` cần kiểm cùng cách.

`audit.html` (lưới ảnh) và `audit-truoc.*` (trước) so được từng màn; ảnh sau ở `audit/sau/`.

## 3. P-107-14 thanh dính đáy (`shots/dock-log.txt`)

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
| 6 | PASS | khung `sticky`; ba tên màn; ảnh rộng 1278–1304 px; ảnh đổi theo cuộn (ảnh 0 → 1 → 2) |
| 7 | PASS | 1024 px và 390 px: 0 `sticky`, ảnh đều hiện |
| 8 | PASS | ba tên bước hiện sau cuộn (opacity 1). Chưa đo sợi chỉ 1 s, nhịp 500 ms |
| 9 | PASS | ba số đo nguyên văn nguồn (câu 1 có «recall 0,960 trên 200 mẫu tự soạn» viết hoa chữ đầu vì đầu câu); số 146 đếm (giữa chừng 91, cuối 146); số lớn 36 px (≤ 40) |
| 10 | PASS | hai tab, mũi tên đổi tab, ảnh vai đổi; 390: ảnh vai nằm dưới danh sách |
| 11 | PASS | `h1` 72/38 px, `h2` 48/30 px; chữ ngoài khung minh hoạ ≥ 16 px. Chữ < 16 px chỉ nằm trong ảnh minh hoạ giao diện (13–15 px) |
| 12 | PASS | rê: nhích −1 px; bấm: `scale(0.985)` + lún 1 px |
| 13 | **PASS có ghi chú** | LCP là chữ (`span` trong `h1`, 88 ms ở 1440; ở 390 là đoạn dẫn, 100 ms). AC ghi LCP là `h1` ở cả hai độ rộng; ở 390 LCP là đoạn dẫn như số đo của TL ở mẫu. JS 194,9 KB gz (≤ 256) |
| 14 | PASS | `reduce`: 0 `m-on`, 0 animation chạy, 0 `sticky`, số 146 hiện ngay |
| 15 | PASS | không script: 4 phần tử opacity 0 đều là trạng thái cuối của đoạn sản phẩm (7,5 thay 8,5; Nháp thay Duyệt); không khối chữ nào bị ẩn |
| 16 | **FAIL (Thấp)** | axe 0 serious/critical ở 1440 và 390; liên kết chân trang ≥ 44 px; **`/` không có thẻ `og:*`** (chỉ có `description`) |
| 17 | PASS | không `@edupilot.local` / `Edupilot#`; có dòng dữ liệu mô phỏng, tên tác giả, liên kết GitHub |
| 18, 20 | Không chạy | dev có e2e (`landing.spec.ts`); QC không chạy lại |
| 19 | PASS | 0 lỗi console |
| 21 | Không chạy | thuộc dev (`landing.spec.ts`, ảnh mốc) |
| 22 | PASS | màn đầu có đúng một dữ kiện cứng, nguyên văn nguồn (số 146); không «1.000 sinh viên» |
| 23 | **PASS** | `copy-lint` 0 FAIL. `consistency.mjs` 11 FAIL (1440: 8; 390: 3) đều nằm trong danh sách miễn trừ của AC23: `scroll-reveal` ×6, `scroll-linked` ×1, `section-air` ×2, `empty-area` ×1, `copy-headings` ×1. `perf.mjs` 2 FAIL = «8 tệp phông» (miễn trừ) |
| 24 | PASS | (1) mục số đo có ảnh Sổ điểm ở 1440 và 390, khoảng trống liền ở 390 là 347 px (≤ 844); (2) ảnh vai dưới danh sách ở 390; (3) số lớn 36/24 px; (4) 0 `container-edges`; (5) `srcset` chọn 640 ở 390. `perf.mjs` còn 1 WARN «ảnh dư cỡ» ở 390 (`story-gradebook-1280` cho 340 px, đo ở DPR của công cụ) |

## 5. Phương pháp và phạm vi

- Audit `sau`: 4 vai (admin, giảng viên, trợ giảng, sinh viên) × mọi route × 1440 / 375 × 4 trạng thái (có dữ liệu lớp 761987; rỗng lớp `QCEMPTY`; đang tải bằng chặn API 5 s; lỗi bằng 500). 310 lượt chụp, 0 lượt lỗi chạy. Script: `frontend/e2e/qc-ui-audit-real.spec.ts`, `scripts/audit_report.py`.
- Chỉnh cách đo theo TL / PM (đã ghi trong script): H7 gom theo (độ rộng, cao `h1`, dòng meta, có hàng nút) (TL 12.1); nút trong `data-ep-header-dock` không tính H6 (P-107-14); màn không phát yêu cầu `/api/v1` khi bị chặn (mô phỏng) bỏ qua L/X (TL 12.2).
- `landing`: `qc-landing-real.spec.ts` (đo DOM ẩn danh), `copy-lint`, `perf --runs 3`, `consistency` 1440 + 390 trên `http://localhost:580/`.
- Chưa kiểm: `contrast.spec.ts` / `panels.spec.ts` / `frame.spec.ts` / `landing.spec.ts` của dev (chạy ở CI dev); tương phản chữ–nền theo bộ màu; Lighthouse 4G thật cho LCP; số liệu hành vi cuộn `IntersectionObserver` (AC8 sợi chỉ, AC15 độ trễ 1,5 s); ảnh mốc `visual.spec` (dev sinh trên CI).
- Dọn: stack `ep-qc` đang chạy (frontend `1935435a`, env QC).
