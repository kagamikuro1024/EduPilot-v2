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
