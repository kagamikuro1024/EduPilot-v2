# QC sprint 10.7 (UI-R) — kết luận

Nhánh `sprint/10.7-ui-r`, HEAD kiểm `114a4461` (dev đợt 4). Stack `ep-qc`: chỉ rebuild frontend (env QC: `NEXT_PUBLIC_API_URL=http://localhost:580`), DB giữ nguyên. Chỉ đọc, không sửa mã dev.

**Kết luận: gần đạt, còn 10 lượt ở 3 mã đo (lượt đầu 191 lượt, 15 mã), cả ba đều ở `student/chat` và `documents`; chưa đạt «0 lệch».** `ui-antipatterns.sh` sạch (31/31 phép, `--selftest` 31/31); e2e dev liên quan 249 pass, 0 fail. Đạt 0 lệch: toàn bộ nhóm khung (W, H, P, E1, E3–E5, E7, L, X, D1–D3) trừ ba mã dưới đây. Trang giới thiệu và P-107-14, `/settings` sinh viên đều đạt.

## 1. Verdict

| Hạng mục | Verdict | Căn cứ |
| --- | --- | --- |
| Hợp đồng khung §23 — K1 (kiểu trang, độ rộng, panel) | **PASS** | W1–W3, P1–P3 = 0. P2 = 0 (`observability` +17 đã sửa; `/insights` ở 375 +2,1 px chấp nhận, xem 2) |
| K2 (đầu trang) | **PASS** | W4, H1–H7 = 0 |
| K3 (tiêu đề mục) | **PASS** | P4 = 0 |
| K4 / K5 (trạng thái rỗng, nút trùng) | **FAIL (Thấp)** | E1, E3–E5, E7 = 0. E2 (`student/chat` rỗng lệch tâm −4 px) và E6 (`student/chat` panel cao 654 px) còn lệch; xem 2 |
| K6 (375 px) | **PASS** | W5 = 0, 0 tràn ngang, H8 = 0 |
| Chuẩn đo mới (D1–D3, P5, T2) | **FAIL (Thấp)** | D1–D3, P5 = 0. T2 còn 4 lượt: `documents` lặp «Đáp án không hiển thị cho sinh viên» ×4 (chữ tooltip của công tắc, xem 2 và 6) |
| Đang tải / lỗi (L1–L3, X1–X3) | **PASS** | 0 lệch (dock `/attendance` 375 giữ khi tải / lỗi; `/settings` GV 2 alert đúng cột; tab `student/practice` giữ) |
| Lô 1 đo lại (`/insights`, `/questions`, `/inbox`) | **PASS có lệch nhỏ** | `/inbox`, `/questions` 0 lệch. Còn `/insights` ở 375 (P2 +2,1 px). `/insights` ở lớp rỗng: «Chưa có báo cáo», không số liệu lớp khác |
| Lỗi dữ liệu mô phỏng ở lớp rỗng | **PASS** | `/analytics` ở QCEMPTY: «Chưa có số liệu» (trước đó hiện số liệu lớp 761987) |
| `/settings` sinh viên khi lỗi API (TL 14.7) | **PASS** | 3 mục: «Mật khẩu» dùng được (ô nhập, nút hiện mật khẩu), «Thiết bị đang đăng nhập» 1 alert, «Thư» 1 alert; «Nhắc 24 giờ» và «Thời gian học» không vẽ gì, không nút/ô nào dùng được trên dữ liệu chưa tải; tổng 2 alert ≤ 4 GET hỏng khác nhau |
| **P-107-14** thanh dính đáy 375 | **PASS** | xem 3; dock còn khi đang tải / lỗi |
| **US-UX91-08** trang giới thiệu v2, AC1–24 | **PASS** (đo được) | xem 4; AC18, AC20, AC21 do `landing.spec.ts` của dev (chạy xanh ở HEAD, mục 5) |
| Bộ màu ĐHBK (P-107-8/10) | **PASS** | C3 = 0 trên mọi lượt; `contrast.spec.ts` của dev chạy xanh ở lượt e2e dưới (QC chạy lại ở HEAD) |
| `ui-antipatterns.sh` ở `114a4461` | **PASS** | exit 0, 31 phép ✓, `--selftest` 31/31 (`shots/ui-antipatterns-log.txt`) |

## 2. Lệch còn lại (10 lượt trong `audit.md`)

| Mã | Lệch | Route | Loại | Ghi chú |
| --- | --- | --- | --- | --- |
| E2 căn giữa khối rỗng | 4 | `student/chat`: rỗng −4 (1440), −4,1 (375); dữ liệu @375 −4,1; dữ liệu @1440 186 | **Thật** (−4 px, Thấp); **Không áp dụng** (186) | Khối «Hỏi bất cứ điều gì về lớp này» đã nằm trong cột trắng của chat (trước đó −828 px). Lượt dữ liệu @1440 = 186: khối nằm ở ngăn phải của `Split`, QC đo so với cả vật chứa |
| E6 vật chứa rỗng ≤ 320 | 2 | `student/chat` 654 px (1440; dữ liệu và rỗng) | **Quyết** | Panel chat `Split fill` cao hết khung nhìn, ô nhập ở đáy (DESIGN §10.12 ô nhập là bề mặt làm việc riêng); §23.3 chỉ cho `fill` khi danh sách có hàng. QC đề xuất miễn cho màn chat |
| T2 câu lặp | 4 | `documents` (GV, TA; 1440 và 375): «Đáp án không hiển thị cho sinh viên» ×4 | **Quyết** | Là nội dung `role=tooltip` của công tắc «Hiện cho sinh viên» ở hàng đáp án (2 hàng × 2 nơi vẽ), chỉ hiện khi rê. QC đề xuất T2 bỏ `[role=tooltip]` |

**P2 `/insights` ở 375 (PM hỏi dev 2,0 / QC 2,1):** đo lại bằng `getBoundingClientRect` ở DPR 1: panel 1 = 2,0 (chuẩn: 2 px viền), panel 2 = 2,1 (cao 467,3 px). Chênh 0,1 px nằm ở số lẻ của dòng chữ, không có khoảng dư thật. Theo PM, dung sai đo dưới pixel ≤ 0,5 px được chấp nhận (ngưỡng P2 thành 2,5 px; chỉ áp cho P2, không nới tiêu chí khác); cách đo ghi trong `scripts/audit_report.py`.

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
| 13 | **PASS có ghi chú** | Lighthouse devtools 3 lượt, 4G (RTT 150 ms, 1,6 Mbps, CPU 4×): mobile 375 LCP trung vị **1,94 s** (≤ 2,5), CLS 0,096 (≤ 0,1), TBT ≤ 32 ms, JS 192,5 KB; desktop 1440 LCP 80–286 ms, CLS 0, TBT 0, JS 194,9 KB. LCP là chữ `h1` ở 1440 nhưng ở 375 là **đoạn dẫn** (`p.lead`): AC13 ghi `h1` cho cả hai độ rộng, ghi nhận lệch nhỏ (cùng số đo của TL ở mẫu) |
| 14 | PASS | `reduce`: 0 `m-on`, 0 animation chạy, 0 `sticky`, số 146 hiện ngay |
| 15 | PASS | không script: không khối chữ nào bị ẩn (trừ trạng thái cuối của đoạn sản phẩm: 7,5 → 8,5, Nháp → Duyệt) |
| 16 | PASS | axe 0 serious/critical ở 1440 và 390; liên kết chân trang ≥ 44 px; có 9 thẻ `og:*` |
| 17 | PASS | không `@edupilot.local` / `Edupilot#`; có dòng dữ liệu mô phỏng, tên tác giả, liên kết GitHub |
| 18, 20, 21 | PASS (e2e dev) | `landing.spec.ts` xanh ở `114a4461`; QC không đo riêng. Ảnh mốc do dev sinh trên CI |
| 19 | PASS | 0 lỗi console |
| 22 | PASS | màn đầu có đúng một dữ kiện cứng, nguyên văn nguồn (số 146); không «1.000 sinh viên» |
| 23 | PASS | `copy-lint` 0 FAIL. `consistency.mjs` 11 FAIL (1440: 8; 390: 3) đều nằm trong danh sách miễn trừ của AC23: `scroll-reveal` ×6, `scroll-linked` ×1, `section-air` ×2, `empty-area` ×1, `copy-headings` ×1. `perf.mjs` 2 FAIL = «8 tệp phông» (miễn trừ). Chạy lại ở HEAD `9a011cd1`: `copy-lint` 0 FAIL, `consistency` 11 FAIL cùng danh sách miễn trừ |
| 24 | PASS | (1) mục số đo có ảnh Sổ điểm ở 1440 và 390, khoảng trống liền ở 390 là 347 px (≤ 844); (2) ảnh vai dưới danh sách ở 390; (3) số lớn 36/24 px; (4) 0 `container-edges`; (5) `srcset` chọn 640 ở 390 |

## 5. Phương pháp và phạm vi

- Audit `sau`: 4 vai (admin, giảng viên, trợ giảng, sinh viên) × mọi route × 1440 / 375 × 4 trạng thái (có dữ liệu lớp 761987; rỗng lớp `QCEMPTY`; đang tải bằng chặn API 5 s; lỗi bằng 500). 310 lượt chụp, 0 lượt lỗi chạy. Script: `frontend/e2e/qc-ui-audit-real.spec.ts`, `scripts/audit_report.py`.
- Landing: `qc-landing-real.spec.ts` (đo DOM ẩn danh), `copy-lint`, `perf --runs 3`, `consistency` 1440 + 390 (HEAD `1935435a` cho `perf.mjs`; HEAD `9a011cd1` cho `copy-lint` và `consistency`). Lighthouse 4G: `lhci collect` throttling `devtools`, 3 lượt mỗi kiểu (`landing/lighthouse.json`). Dock: `qc-dock-real.spec.ts`. `/settings` sinh viên: `qc-settings-sv-real.spec.ts`.
- E2e dev ở `114a4461` (`pnpm build:gate` + `playwright test` desktop và mobile, `--retries=0`: `documents`, `private-chat`, `frame`, `panels`, `attendance`, `contrast`, `landing`, `class-join`, `settings-error`, `insights-course`, `me`, `decision-observability`): **249 pass, 181 skip (theo dự án), 0 fail** (`shots/dev-e2e-log.txt`).
- Chưa kiểm: tương phản chữ–nền ngoài `contrast.spec.ts`; `IntersectionObserver` (AC8 sợi chỉ, AC15 độ trễ 1,5 s); ảnh mốc `visual.spec`; `e2e` còn lại của dev ngoài danh sách trên.
- Dọn: stack `ep-qc` đang chạy (frontend `114a4461`, env QC).

## 6. Hai điểm PM hỏi thêm

**`documents` — bỏ dòng ghi chú đáp án, lý do ở công tắc: PASS.** `shots/misc-log.txt` (`qc-docs-real.spec.ts`, 1440 và 375): dòng «Không hiển thị cho sinh viên» ở hàng đáp án không còn (`keyNote = 0`); 0 icon info / circle-help; hàng đáp án có công tắc «Hiện cho sinh viên» bị khoá (`disabled`), có `role=tooltip` «Đáp án không hiển thị cho sinh viên» nối bằng `aria-describedby`, hiện khi rê (đo opacity 1). Còn hai hạn chế (Thấp, không tính lệch): công tắc khoá không nhận focus bàn phím và màn chạm không có rê, nên chỉ người dùng trình đọc màn hình (đọc mô tả) và người rê chuột thấy lý do. Hàng tài liệu chia sẻ từ lớp khác (`shared_from`) cũng khoá công tắc nhưng không có lý do nào.

**`observability` — mục «Lớp … 7 ngày» bị ẩn khi không có số: không đúng §23.4, đề nghị là `EmptyState` của mục.** Căn cứ DESIGN §23.4: «The container keeps its frame: same panel, same width, content height» và §23.5: «Loading, empty and error never change the page kind, the header, or the tabs»; §23.3: trang từ hai vùng trở lên mỗi vùng có `Section` + `h2`. Hiện ở lớp `QCEMPTY` mục biến mất cả tiêu đề (`teacher-empty-_observability-1440.jpg`: chỉ còn «Hôm nay» và «0 yêu cầu gần nhất»), nên giảng viên không phân biệt được «lớp không có số» với «màn không có mục này». Đề xuất: giữ `Section` «Lớp … trong 7 ngày» với `EmptyState` «Chưa có số liệu» (không nút). Đây là màn mô phỏng (nợ sprint 11); loại: **Quyết** (Thấp), không có trong số 10 lượt vì mục ẩn không phải một lượt đo.
