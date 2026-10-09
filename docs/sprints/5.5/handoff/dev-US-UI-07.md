# DEV handoff — US-UI-07 (cổng UI cuối sprint 5.5)
`BASE55` = `dfe05f0` (cha của `dd3a003`, commit đầu tiên chạm `frontend/` của sprint 5.5). HEAD tại thời điểm chụp: xem `git log` nhánh `sprint/5.5-ui-panels`.

## AC1 — 78 ảnh trước / sau
`docs/sprints/5.5/handoff/ui-07/{before,after}/<màn>-<1440|1024|375>.png`: **39 + 39**. Chụp bằng `SHOTS=ui07 SHOT_SIDE=before|after pnpm exec playwright test shots.spec.ts` (Playwright, không `playwright-cli`): cùng đồng hồ đóng băng 29/10/2026 09:20 giờ VN, cùng dữ liệu giả (`e2e/support/screens.ts`, `staffMock.ts`, `takeMock.ts`), cùng bản `build:gate`; "trước" dựng từ worktree `BASE55` (thư mục `e2e/support` và `shots.spec.ts` chép từ HEAD vào — chỉ hạ tầng test, mã `src/` là của `BASE55`), "sau" từ HEAD. Viewport 1440 × 900, 1024 × 900, 375 × 812 (không cuộn trang).

| Màn | Route · vai | Thay đổi (cả ba bề rộng) |
| --- | --- | --- |
| `home-student` | `/` · SV | Nền trang thành canvas ấm; "Việc nên làm tiếp" và "Hôm nay" mỗi vùng một Panel, tiêu đề ngoài panel; ở 375 lề 12 px, một cột |
| `exams-student` | `/exams` · SV | Mỗi nhóm (Đang mở / Sắp tới / Đã có điểm) một Panel, hàng ngăn đường kẻ; tiêu đề nhóm ngoài |
| `take-pre` | `/exams/[id]/take` trước giờ · SV | Mô tả, thời lượng, lưu ý và nút `Bắt đầu làm bài` nằm trong một Panel |
| `take-run` | làm bài trắc nghiệm · SV | Thanh trên (`Câu i/n`, đồng hồ, `Nộp bài`) giữ nền surface + kẻ, không phải Panel; danh sách câu và câu hiện tại mỗi bên một Panel (không lồng) |
| `home-teacher` | `/` · GV | "Việc cần xử lý" và "Sắp tới" mỗi vùng một Panel; không ô số liệu |
| `members` | `/class/members` · GV | `Tabs` ngoài; Toolbar tìm kiếm + bảng thành viên trong một Panel |
| `questions` | `/questions` · GV | Toolbar lọc + bảng trong một Panel |
| `exam-editor` | `/exams/[id]` nháp · GV | `Tabs` ngoài; tab Thông tin = một Panel chia `PanelSection` (Thời gian, Xáo trộn, Chấm điểm, Sau khi công bố) |
| `exam-results` | `/exams/[id]/results` · GV | `Tabs` ngoài; thanh tiến độ + bảng điểm trong một Panel |
| `admin-users` | `/admin/users` · Admin | Toolbar + bảng trong một Panel |
| `admin-courses` | `/admin/courses` · Admin | Bảng lớp trong một Panel |
| `settings-llm` | `/settings/llm` · Admin | Năm vùng mỗi vùng tiêu đề ngoài + một Panel (nhà cung cấp mỗi nhà một hàng) |
| `login` | `/login` | Canvas ấm; `h1` ngoài, một Panel (≤ 440 px) chứa biểu mẫu |

## AC2 — ảnh mốc
`visual.spec.ts` trong `mcr.microsoft.com/playwright:v1.63.0-noble`, **không `--update-snapshots`**: lần 1 **14 passed**, lần 2 **14 passed**. `git diff --name-only dfe05f0 -- frontend/e2e/visual.spec.ts-snapshots` = **14** tệp (`chat`, `dev-ui`, `gradebook`, `home`, `inbox`, `settings-llm`, `threads` × 1440 / 390) = hợp danh sách ở `dev-US-UI-02.md` (14 ảnh), `-03` (6), `-04` (8), `-05` (8) — mỗi ảnh có mặt ở ít nhất một handoff, không thừa / thiếu. Mọi ảnh sinh trong image trên (`--update-snapshots=all`) do dev. Ghi chú: ảnh `inbox-*`, `gradebook-*` trước `UI-05` là trang chặn quyền (lỗi ảnh nền từ sprint 3–5, PM đã ghi nhận ở proposal #4).

## AC3 — axe, tương phản
`a11y.spec.ts` (9 ca, ~2 phút): **138 lượt quét** (route × vai × bề rộng, gồm các ca "có dữ liệu" của Sinh viên / Giảng viên / Admin thêm ở UI-06 / UI-07): **critical 0, serious 0, moderate 0, minor 0**. Đã siết: `color-contrast` hiện **0 vi phạm** (mọi mức nghiêm trọng) — trước chỉ chặn critical / serious. `axe-allow.json`: không đổi (≤ 3). Mục `incomplete` của `color-contrast`: **88 lượt** có mục cần xem tay (`test-results/axe-incomplete-contrast.json`); gốc phần lớn: phím tắt `kbd ⌘` ở ô tìm kiếm của thanh trên (64 lượt), số đếm trong `Tabs` (8), chữ SVG của biểu đồ (`text`, `lastValue`), `select`, id động của trường nhập. QC xem tay. `contrast.spec.ts` (3 ca): ma trận chữ × nền ≥ 4,5 : 1, sidebar, `status text` — pass.

## AC4 — NEST / TITLE / WALL / STRONG
`panels.spec.ts`: **58 pass** (+58 skip ở dự án `mobile`), **44 s** với 2 worker (≤ 180 s). Vòng bảng 7.2: Sinh viên 16 route (+ `/settings`, `/join/ABC123`) × 1440 / 1024; Staff `teacher` và `ta` mỗi vai ~24 route × 1440 / 1024; Admin 7 route × 1440 / 1024; mọi route có `main` ≥ 1 Panel (trạng thái có dữ liệu), NEST = TITLE (chỉ tiêu đề đang hiển thị) = WALL = 0, STRONG ≤ 3. `ui-antipatterns.sh` 22 ✓ / 0 ✗, `--selftest` 22 / 22.

## AC5 — Lighthouse
LHCI_PLACEHOLDER

## AC6 — không hồi quy
Toàn bộ Playwright (`--grep-invert "@real|visual"`, 2 worker): **BASE55: 443 pass / 103 skip / 0 fail** (worktree `BASE55`, bộ e2e của chính nó) → **HEAD: FULL_PLACEHOLDER**. Số pass chỉ tăng (ca mới của `panels.spec`, `contrast.spec`, `a11y.spec`, `shell.spec` không đổi số ca). `nav.ts`: `git diff dfe05f0 -- frontend/src/shared/shell/nav.ts` = 0 dòng. `git diff dfe05f0 -- frontend/package.json pnpm-lock.yaml`: không dòng `+` (không thêm thư viện). `pnpm lint` rc 0; `build` và `build:gate` không cảnh báo; `ui-allow:` = 9 (không tăng).

## AC7 — phân quyền
`shell.spec.ts -g 'nav per role|route access'` pass (8 / 13 / 16 / 6 mục; chặn quyền như cũ). Không endpoint mới.

## AC8 — hạng mục cấm
- Tường thẻ KPI: WALL = 0 ở mọi route đo.
- **Diện tích đỏ** (quy tắc TLR-9: ΔE2000 ≤ 10 so với `--ep-red`, không tính `--ep-red-soft`, mẫu số toàn ảnh 1440 × 900; `panels.spec.ts -g 'red area'`, đo bằng canvas trong trang, không thêm thư viện):

| Màn | Đỏ | Màn | Đỏ |
| --- | --- | --- | --- |
| home-student | 0,42 % | exam-editor | 0,26 % |
| exams-student | 0,02 % | exam-results | 0,03 % |
| take-pre | 0,37 % | admin-users | 0,41 % |
| take-run | 0,03 % | admin-courses | 0,29 % |
| home-teacher | 0,05 % | settings-llm | 0,11 % |
| members | 0,02 % | login | 0,03 % |
| questions | 0,38 % | | |
Tất cả < 8 %.
- Một hành động chính mỗi vùng: `panels.spec` (`exams staff`, `questions`, `admin users`, `admin courses`) đếm `[data-variant=primary]` hiển thị = 1.
- Không `backdrop-filter`, không gradient chữ, không kính mờ (`ui-antipatterns` ✓).

## AC9
Handoff này; cập nhật `docs/PROGRESS.md` và dòng "Chủ dự án đã xem ảnh trước / sau" là việc PM (cổng nghiệm thu 1).

## Chưa làm / nợ
- Màn mock chưa dựng thật; chế độ tối; vùng "Lớp cần chú ý" / "Tiếp tục học" (proposal #2).
- `@real` chưa chạy.
