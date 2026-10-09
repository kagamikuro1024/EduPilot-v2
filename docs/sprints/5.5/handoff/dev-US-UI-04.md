# DEV handoff — US-UI-04 (màn Sinh viên → Panel)
Nhánh `sprint/5.5-ui-panels`; D59 (a). CI xanh tại `c4276bc` (Frontend, Go, Judge).

## Làm gì
- `shared/ui/Layout.tsx`: `Section` có prop `panel?: boolean | "lg" | "none"` — bọc **children** trong một `Panel`, tiêu đề vùng (`h2`) ở NGOÀI. Mọi màn dưới đây dùng prop này thay vì tự dựng khung. `Layout.module.css`: ở < 720 px `.page` padding-inline = `--ep-space-3` (12 px; trước 16) — quy tắc lề duy nhất (TLR-4), `Panel` không `margin`.
- Hôm nay (`StudentToday`): "Việc nên làm tiếp" (đổi tên từ "Việc nên làm bây giờ" cho khớp AC1), "Vào lớp bằng mã" / "Nhập mã tham gia lớp", "Hôm nay" = mỗi vùng một `Panel`.
- `/exams`: mỗi nhóm một `Panel`, rỗng / đang tải = một `Panel`. `/exams/[id]/take`: Intro, Done, loading, lỗi = `Panel`; đang làm: thanh trên **không** phải Panel (giữ `--ep-surface` + kẻ), câu hỏi = `Panel`, danh sách câu bên cạnh = `Panel` riêng trong `<aside>`; code: hai `Panel` (đề + test mẫu | soạn mã); kết quả: `PanelSection tone="strong"` cho điểm (1 ô nhấn).
- `/join`, `/join/[code]`, `/settings` (Mật khẩu, Phiên đăng nhập): `Section panel`; bỏ `border-top` của `.list` trong `security.module.css` (đường kẻ do `Panel`/`ActionList`).
- Màn mock: `/chat` — hai `Panel` (lịch sử phiên cạnh; cuộc trò chuyện với `Composer` bên trong), CSS `.history` chỉ định vị (bỏ viền / bo góc riêng); `/threads` — bộ lọc ngoài, danh sách một `Panel`, form đặt câu hỏi `Section panel` (bỏ nền / viền `.createPanel`); `/threads/[id]` — một `Panel` (câu hỏi + AI chính, ngăn bằng `PanelSection`) + một `Panel` thảo luận; hàng phản hồi ngăn bằng kẻ 1 px (bỏ khung từng bài, vạch trái 2 px màu cho bài AI giữ); `/practice`, `/practice/[id]`, `/practice/history`, `/me`, `/assignments/[id]`: `Section panel`; `/library`, `/calendar`: danh sách / lịch = một `Panel`; rỗng = `Panel` + `EmptyState`.
- `PreShell` bọc đoạn `body` (chỉ `/chat`) trong `Panel` — nếu không, chữ thật trong Panel xuống dòng nhiều hơn chữ PreShell ⇒ ứng viên LCP mới (`lcp.spec.ts` đỏ ở `/chat`). **Đính chính** `dev-US-UI-03.md` ("PreShell không import Panel").
- Test: `panels.spec.ts` + `student home`, `student routes`, `student mock routes`, `exams student`, `take states` (4 trạng thái × 1440 / 1024 / 375), `join|settings`, `mobile gutter` (375 và 390); `e2e/support/panels.ts` (phép đo dùng chung, bỏ panel ẩn), `e2e/support/takeMock.ts` (gateway giả đọc-chỉ cho bốn trạng thái làm bài).

## AC
| AC | Kết quả |
| --- | --- |
| 1 | `-g 'student home'` pass ở 1440 / 1024 / 375: đúng 2 `Panel` (h2 "Việc nên làm tiếp", "Hôm nay"), NEST = TITLE = WALL = 0, STRONG ≤ 1, 375 px một cột. `today.spec.ts`, `shell.spec.ts` pass. **Chưa có** khối "Tiếp tục học": `continue` trong API là `unknown[]` (chưa có hợp đồng dữ liệu) và `today.spec.ts` đang pin "ẩn khi rỗng" — không bịa cấu trúc |
| 2 | `-g 'student routes'` pass: 16 route (+ `/settings`, `/join/ABC123`) × 1440 / 1024 — ≥ 1 Panel, không `h1/h2` trong panel, không tràn ngang, NEST / WALL = 0, STRONG ≤ 3. `/exams/[id]/take` đo ở `take states` |
| 3 | `-g 'exams student'` pass: 3 nhóm có dữ liệu = 3 Panel, không Panel trong `li`, h2 ngoài panel, rỗng = 1 Panel. `exam.spec.ts -g 'student list'` pass |
| 4 | `-g 'take states'` 12 ca pass (intro / running / submitted / published × 3 bề rộng): NEST / TITLE / WALL = 0, thanh trên có đồng hồ không nằm trong Panel, câu hỏi trong Panel, `AUDIT_SRC` sạch. Ca code: `exam.spec.ts` (`code viewport gate`, `submission history`) pass; bố cục hai Panel của code **chưa** có phép đo tự động trong `panels.spec` và tôi chưa chụp / soi tay (xem nợ) |
| 5 | `-g 'join\|settings'` pass; `account.spec.ts`, `class-join.spec.ts` (không `@real`) pass |
| 6 | `-g 'student mock routes'` pass (chat: thread + Composer trong Panel, không lồng; threads: list trong Panel, `searchbox` ngoài Panel; thread: câu hỏi trong Panel; practice, me (STRONG ≤ 3), calendar). `ui-antipatterns` 22 ✓ / 0 ✗. Lệnh `grep -rnE 'border-radius\|box-shadow'` của AC cho **1** dòng: `threads/Threads.module.css:90 .dots i { border-radius: 50% }` = chấm "đang gõ", không phải khung vùng (có từ trước) |
| 7 | `-g 'mobile gutter'` pass ở 375 và 390: mọi `[data-ep-panel]` hiện của main cách mép đúng 12 px hai phía; `AUDIT_SRC` ox = 0 / cut = 0 ở 375, 390, 1024; `TOUCH_SRC` rỗng ở 375, cho mọi route Sinh viên của `STUDENT_ROUTES` |
| 8 | Chuỗi hiển thị chỉ đổi: "Việc nên làm bây giờ" → "Việc nên làm tiếp" (AC1); không thêm / bớt chữ khác. `ui-antipatterns` "Từ kỹ thuật trong màn sinh viên" ✓. `e2e/*.spec.ts` của màn Sinh viên: số ca không đổi (xem dưới) |

## Số ca (non-`@real`, non-`visual`)
| Bộ | Sau story | Trước story |
| --- | --- | --- |
| `exam today account class-join shell` | 232 pass / 49 skip | giống (không thêm / bớt ca; chỉ đổi tên + một hằng ở `shell.spec`) |
| Toàn bộ Playwright | 490 pass / 150 skip / 0 fail (lần chạy cuối trước PreShell: 489 pass + `lcp /chat` đỏ; sau sửa `lcp.spec` + `shell.spec` 27 pass) | 466 pass / 127 skip (UI-03) → +23 ca mới (23 desktop; mobile skip +23) |

Sửa selector (lý do): `shell.spec.ts` `regression-1.5: LEFT page-title 240 / 16` → `… / 12`, giá trị mong đợi 16 → 12 ở 390 px: lề ngang mobile đổi theo AC7 / TLR-4. Tên ca đổi theo; QC tìm theo tên cũ sẽ không thấy.

## Ảnh mốc (14 ảnh, image `mcr.microsoft.com/playwright:v1.63.0-noble`, `--update-snapshots=all`; sau đó 2 lần không `--update` = 14 passed)
Đổi 8 ảnh:
| Ảnh | Lý do |
| --- | --- |
| `chat-1440.png`, `chat-390.png` | Chat thành hai Panel, Composer trong panel |
| `threads-1440.png`, `threads-390.png` | Danh sách thread trong Panel, bộ lọc ngoài |
| `home-390.png`, `gradebook-390.png`, `inbox-390.png`, `settings-llm-390.png` | Chỉ do lề ngang 12 px (16 → 12) — các màn Staff / Admin **chưa** đổi cấu trúc, UI-05 / UI-06 sẽ đổi lại |
Giữ nguyên 6 ảnh (`dev-ui-*`, `home-1440`, `gradebook-1440`, `inbox-1440`, `settings-llm-1440`).

## Gate
`pnpm lint` sạch; `tsc --noEmit` sạch; `ui-antipatterns.sh` 22 ✓ / 0 ✗, `--selftest` 22/22; `lint-selftest.sh` 7/7 + 22/22; `ui-allow:` vẫn 9. Không thêm thư viện.

## Chưa chạy / nợ
- Lighthouse (`lhci`) chưa chạy (CLS / LCP / TBT) — gom ở UI-07 AC5.
- Ca code của `/exams/[id]/take`: chưa có phép đo `panels.spec` riêng, chưa soi bằng mắt (hai Panel đề | soạn mã đã dựng, `exam.spec` code pass).
- `FEAT-ui-foundation` 7.2 chưa có mục `Panel` — cần proposal PM (file `proposals.md` của sprint 5.5 do PM giữ).
- `@real` e2e chưa chạy.
- CI của UI-02 (`3f83d84`) đỏ duy nhất ở `bell real` (đã sửa ở UI-03); CI của UI-03 (`8ba5ec9`) bị huỷ do push kế tiếp (concurrency); HEAD `c4276bc` xanh gồm cả hai.
