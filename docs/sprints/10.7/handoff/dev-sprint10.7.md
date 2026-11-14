# Dev — Sprint 10.7 (UI-R)

## Lô 1 — `/insights` hiện báo cáo lớp khác (điều tra → sửa)
- Nguyên nhân: `/insights` là màn mô phỏng (không có API). `mockCourseFor(class_code)` rơi về `COURSES[0]` (761987) cho mã lớp thật không có trong mock (KC106…), `reportFor` lại rơi về `TOPICS[COURSE_1]`; `courses` dedupe theo id nên lớp lạ gộp vào 761987. Không rò dữ liệu (hằng số trong bundle, không truy vấn theo lớp).
- Sửa: `mockCourseFor(real)` giữ id / mã / tên THẬT cho lớp ngoài mock; `reportFor` trả `null` → «Chưa có báo cáo» (không thân chữ), header dùng nhãn lớp thật; trạng thái rỗng của màn dùng cùng mẫu. Test: `frontend/e2e/insights-course.spec.ts`.
- **Nợ (trái D51 «cuối sprint 10 không còn mock»):** `/insights` và các màn mô phỏng còn lại (analytics, observability…) thay bằng API thật ở sprint 11 (M13-A/B). Các mock khác vẫn còn `?? X[COURSE_1]` fallback theo lớp — sẽ hết khi nối API.
- `e2e/visual.spec.ts` «insights» đỏ sẵn trên `main` (lệch chữ dưới 1 px so với ảnh gốc, cùng máy) — không do lô này.

## Nợ chuyển từ 10.6 (PM 2026-11-14)
- (d) Bộ lọc «Bài tập» của hàng chờ chấm mất khi rời route (10.6 chỉ giữ Trạng thái / Cần xem kỹ qua `sessionStorage`; chọn bài tập bị `relaxed`/danh sách bài tập tải sau ghi đè). Làm cùng hợp đồng khung §23.
- (k) Dòng báo xong sau «Chấm lại» (`Đã chấm xong N bài`) — nếu QC thấy còn thiếu ở một nhánh (lô không tới SUCCEEDED, tải lại trang giữa chừng), bổ sung.

## Nợ cho hợp đồng khung §23 (techlead-ui)
- **Toolbar 390 px: ba nút xuống 2 dòng.** `/grading` tab Hàng chờ (giảng viên), `Toolbar` `end` = «Tải bài lên» + «Chấm tất cả» + «Công bố» (nút Công bố luôn hiện từ 10.6 (l)). `shared/ui/Layout.module.css`: `.toolbarEnd` không `flex-wrap`, nút co lại nên mỗi nhãn ngắt dòng. Ảnh chuẩn `visual.spec.ts-snapshots/grading-queue-390.png` (P-106-16) ghi nhận hiện trạng này. Gợi ý: `.toolbarEnd { flex-wrap: wrap }` + nút `flex: none` ở ≤ 719 px; hợp đồng §23 nên định nghĩa hàng hành động trên mobile (≤ 1 nút chính + menu `⋯`, theo luật «tối đa 2 hành động phụ»). Khi sửa phải cập nhật lại ảnh chuẩn trên Linux/CI.

## Lô 0 + lô 1 (dev, 2026-11-14) — theo `techlead.md` TL-1
**Đã làm (nhánh `sprint/10.7-ui-r`, mỗi bước một commit):**
1. `lô 0.1` bộ màu ĐHBK (P-107-8/10): token `tokens.css` + `DESIGN_TOKENS.css`, 3 logo, `contrast.spec` thêm cặp vàng, `panels.spec` nền trắng.
2. `lô 0.2` hợp đồng khung (§23): `Page kind="work"|"read"` (bắt buộc; bỏ `width`, `className`), `PageHeader` (`meta[]`, `primary`, `secondary ≤ 2`, `more`; dưới 720 px nút phụ gom vào «Thêm»), `HeaderAction` (dữ liệu; thêm `expanded`, `danger`, `desktopOnly`, `menu` là mảng hoặc hàm), `Section` luôn bọc `Panel` (bỏ `description`, `panel`, `className`), `Panel` bỏ `padding`, `Split` (`main-aside|half|list-detail`, `fill`, `detailOpen`), `EmptyState` chỉ `title`+`action`, `PageState` (lỗi một dòng, `unsaved`). Sửa máy móc ~280 chỗ gọi bằng codemod TypeScript (`tsc` dẫn đường), 34 `PageHeader` có `actions` chuyển tay. Móc DOM `data-ep-*`.
3. `ui-antipatterns.sh` 24–31 (khung độ rộng, chiều cao theo màn, với tay vào primitive, meta lặp lớp, tiêu đề tự dựng, nút rỗng trùng, tiêu đề rỗng là câu, vàng làm chữ) + hạt giống: `--selftest` 31 / 31. **Nợ khung** nằm ở `scripts/ui-frame-debt.txt` (regex đường dẫn các route chưa chuyển); mỗi lô sửa route xoá dòng của mình, lô 7 xong thì tệp rỗng.
4. `frontend/e2e/frame.spec.ts` + `support/frame.ts`: chế độ báo cáo (in / đính kèm độ lệch W1–W5, H1–H6, P1, P3, P4, E1–E5, T1 theo móc `data-ep-*`); `FRAME_STRICT=1` làm mỗi độ lệch thành lỗi (bật sau lô 7).
5. `lô 1` `/insights` (read, không meta lặp lớp, nút chính ở đầu trang), `/questions` (rỗng chỉ tiêu đề; lọc rỗng có «Xoá bộ lọc»), `/inbox` (`Split list-detail fill`; xoá CSS chiều cao; rỗng = panel thấp). Ảnh trước / sau: `docs/sprints/10.7/dev/lo1/index.html`.

**Test đã cập nhật theo hợp đồng mới (không nới ngưỡng):** chữ thân rỗng bị bỏ (documents, exam, escalation, panels, class-join); `LEFT` 240 (work) / 428 (read) (`shell.spec`); nền trang trắng (`shell.spec` frame layers); `Panel` chỉ một đệm (`panels.spec`); lớp lạ giữ mã thật (`class-join`); «Tải tài liệu lên» chỉ một nút. `description` mang dữ liệu (số sinh viên, mã SV, ngày, `quy đổi x/10`, buổi học…) được chuyển sang `meta`; câu giải thích thì xoá.

**Việc còn lại / nợ cho lô sau:**
- **Ảnh mốc `visual.spec`**: đổi gần hết (nền trắng + khung). Phải sinh trên Linux / CI sau khi lô 1 được duyệt (cách làm ở P-106-16: lấy `*-actual.png` từ artifact `e2e-reports-*`).
- `PageHeader` của `error.tsx`, `not-found.tsx`, forbidden: dùng `primary` thay nút cũ. Màn forbidden vẫn giữ câu «Trang này dành cho …» ở `meta` (một dòng): nếu §23 muốn bỏ hẳn thì đổi thành tooltip.
- Hàng trống chiếm cao: `EmptyState` đệm dọc `--ep-space-16`; `Split fill` dùng `calc(100dvh − topbar − 12rem)` — chỉnh theo ảnh QC.
- Landing (8/9) chờ mẫu mới (PM).
- Chập chờn có sẵn: `landing.spec` «responsive» (axe color-contrast), không do lô này.

## Lô 2–4 (dev, 2026-11-14)
- **Sửa lệch QC lô 1 ở primitive** (`audit-lo1.md`): điều khiển nhập/chọn cao 36 (nút chữ 30; 44 ở màn nhỏ), `h2` của Dialog có `data-ep-section-title`, `/insights` mục→mục 48, `/questions` mỗi hàng một dòng (cột «Chủ đề» riêng) + lỗi thay cả vùng bằng một `role=alert` rộng đúng cột, `Nhận` ở `/inbox` không còn primary, `PreShell` có móc `data-ep-page` / `data-ep-page-header`.
  - **Không sửa được ở khung, cần TL quyết:** H7 ở 375 (đầu trang có hàng nút full-width cao hơn đầu trang không có → hai giá trị theo «có nút hay không», không theo «có meta»); L2/X1 của `/insights` (màn mô phỏng, không gọi API: không có trạng thái tải/lỗi thật — hết khi nối API ở sprint 11).
- **Lô 2** `/chat` `Split list-detail fill`, `/threads` work, `/threads/[id]` read.
- **Lô 3** `/grading` work: «Chấm tất cả» (chính) + «Tải bài lên» + «Công bố» (phụ) lên `PageHeader` (≤ 2 nút phụ → 390 px một hàng nút); «Đồng bộ từ Teams» ở thanh lọc (mục tiêu của kênh nguồn, giữ nút + khoá + «Chờ N giây»). Bộ lọc hàng chờ (`assignment`, `status`, `needs`) sống trong URL, bỏ `sessionStorage` — gỡ nợ (d); rỗng nói đúng bộ lọc («Không có bài chưa duyệt», «Không có bài cần xem kỹ», «Chọn một bài tập»). `/grading/[id]` = `Split half fill` (hai nửa bằng nhau, mỗi nửa cuộn riêng; thay 55/45 + vh), `Split only` ẩn một ngăn dưới 720 px theo tab «Bài làm / Chấm». `/gradebook/scheme` work + `Split main-aside` (công thức | trích dẫn, hai Panel).
- **Lô 4** `/exams` read, `/exams/[id]` read, `take` + thi thử `Page work` + `Split main-aside` (danh sách câu thành ngăn phụ **bên phải**, trước là bên trái; dưới 1100 px vẫn nút «Danh sách câu» + Drawer), bỏ `max-width` riêng của khung làm bài; `/me` tiêu đề rỗng không dấu chấm.

## ĐANG DỞ / BƯỚC TIẾP (dev, dừng theo lệnh PM 2026-11-15)
- **Đã commit nhưng CHƯA kiểm đủ:** lô 3–7 viết xong mã (tsc + eslint 0 lỗi, `ui-antipatterns.sh` sạch với `ui-frame-debt.txt` RỖNG, `--selftest` 31/31). e2e đã chạy xanh theo cụm ở lô 3, 4, 5 (assignment-lifecycle, gradebook, grading-guidance, exam, me, attendance, students, class-join, panels, frame); lô 6 (`/` GV/TA = `Split main-aside`, `/library` read, `/calendar`) và lô 7 (PreShell/AuthShell dùng `PageHeader`, NoBackend, admin, empty không dấu chấm) CHƯA chạy e2e.
- **Bước tiếp:** `pnpm build:gate` rồi chạy FULL `pnpm exec playwright test --project=desktop --project=mobile --retries=0` (đã bị huỷ giữa chừng). `frame.spec` nay MẶC ĐỊNH chặn (đổi `FRAME_STRICT=1` → `FRAME_REPORT=1` để chỉ báo cáo). Sửa test cũ theo hợp đồng mới (chữ rỗng không dấu chấm; nút rỗng trùng đầu trang bị bỏ; `Split` thay lưới tự dựng; bộ lọc hàng chờ chấm sống trong URL, vào lại bằng liên kết «Hàng chờ chấm» từ màn duyệt).
- Sau đó: ảnh trước/sau `docs/sprints/10.7/dev/lo3…lo7/`, sinh lại ảnh mốc `visual.spec` trên CI (artifact `e2e-reports-*`), lô 8–9 landing theo `mau/landing.html` (P-107-12, có chuyển động) + AC24 + axe landing (reducedMotion).
- **Cần TL quyết:** H7 ở 375 (header có/không hàng nút full-width); `/insights` là màn mô phỏng nên L2/X1 chưa đo được; nút lưu điểm danh ở 375 giờ có hai nút cùng việc (primary ở đầu trang + thanh dưới dạng secondary) để H6 sạch.
- Ảnh trước/sau lô 3–7 CHƯA chụp.
