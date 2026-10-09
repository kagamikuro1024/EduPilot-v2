# Báo cáo QC — US-UI-05 (Giảng viên / TA → `Panel`)
**Kết luận: FAIL — 1 TC (TC-UI5-03, AC3, B1).** Còn lại PASS / PASS có điều kiện. Bản chấm `1d6caea` (handoff UI-05; mã `ef9056a…26adacc`), so với trước `36824f4` (UI-04). QC đo bằng `scripts/qc-measure.spec.ts` ở hai nguồn: (1) phiên giả `asDemo` `teacher` + `ta` × 19 route × 1440 / 1024 / 375 (trước + sau; ở nguồn này các màn dữ liệu chỉ hiện khung tải nên chỉ có giá trị cho màn mock); (2) **stack thật** `pnpm dev` + seed, đăng nhập thật `teacher@`, `ta@` — 19 route Giảng viên + 9 route TA × 1440 / 1024 / 375, ảnh `shots/ui05-real/` (84 lượt, 84 pass), số `measure/ui05-real.json`. Bộ e2e của dev: **286 pass, 0 fail** (`panels.spec.ts` + 5 file giữ nguyên số).

## Lỗi / lệch
- **B1 (AC3 — `/exams` rỗng không nằm trong `Panel`).** Repro: đăng nhập `teacher@edupilot.local` (lớp mặc định 761988, chưa có bài thi) → mở `/exams` → `document.querySelectorAll('[data-ep-panel]').length` = **0**; `EmptyState` "Chưa có bài thi nào" + nút `Tạo bài thi` nằm trơn trên canvas (ảnh `shots/ui05-real/exams-real:teacher-1440.png`). AC3 / UI-04 AC3: "rỗng = **một `Panel`** + `EmptyState` có nút". Bộ test của dev xanh vì mock luôn trả có dữ liệu. TA vào lớp 761987 (có bài thi): `/exams` 1 panel — đúng. Nên kiểm cả trạng thái rỗng ở `/exams` Sinh viên (UI-04 AC3, cùng yêu cầu) — QC sẽ chấm lại UI-04 phần này khi dev sửa.
- **L1 (TITLE).** `/exams/[id]/results` (GV và TA): selector `[data-ep-panel] h2.ep-section-title` = **1** — tiêu đề "Công bố điểm?" của `Dialog` đang đóng, nằm trong DOM của Panel (không hiển thị). Cùng gốc với L2 của UI-04 (`/settings`): đề nghị đưa `Dialog` ra portal hoặc định nghĩa lại TITLE bỏ `dialog` / phần tử ẩn (Q-QC-UI04-2). Không tính FAIL.
- **L2 (AC1).** Hôm nay Staff có đúng 2 vùng ("Việc cần xử lý", "Sắp tới"); "Lớp cần chú ý" chưa có (PM #2: chỉ khi có dữ liệu) — không lệch.
- **L3 (TOUCH, 375 px).** Ngoài phạm vi bắt buộc của AC7 (chỉ `/attendance`, `/inbox`: đạt, chỉ còn liên kết ẩn "Bỏ qua điều hướng" 159 × 40): `/exams/[id]` có 3 `input` 20 × 20 (ô tích trong form Thông tin); `/documents` có 20 nút `Có` 53 × 32 (màn mock). Ghi nhận.
- **L4 (ảnh trước).** Không có ảnh "trước" bằng dữ liệu thật cho màn Staff (stack cũ phải dựng lại); ảnh trước / sau ở phiên giả chỉ là khung tải. Bằng chứng thị giác: ảnh thật sau (`shots/ui05-real`) + ảnh mốc `visual.spec` mới (xem báo cáo UI-07).

## Kết quả
| TC | KQ | Bằng chứng |
| --- | --- | --- |
| TC-UI5-01 (AC1) | PASS (L2) | GV và TA `/`: **2** panel (1440 / 1024 / 375), NEST 0, TITLE 0, STRONG 0, WALL 0, `ox` 0, lề 12 px ở 375; `today.spec.ts` 24 / 4 / 0 |
| TC-UI5-02 (AC2) | PASS | `/class/members` GV và TA: 1 panel (Toolbar + bảng trong panel), `Tabs` ngoài; `/class/settings`: 3 panel (mỗi mục một); NEST 0, STRONG 0; dev `members` pass; `class-join.spec.ts` 61 / 21 / 0 |
| TC-UI5-03 (AC3) | **FAIL (B1)** | `/exams` có dữ liệu (TA, lớp 761987): 1 panel ở 1440 / 375; **rỗng (GV lớp 761988): 0 panel** |
| TC-UI5-04 (AC4) | PASS | `/exams/[id]` (1 panel / tab, NEST 0, TITLE 0); dev `exam editor` pass (≥ 4 `PanelSection`, Drawer không có panel trong `[role=dialog]`) |
| TC-UI5-05 (AC5) | PASS | `/questions` GV và TA: 1 panel; `Drawer` là lớp nổi; dev `questions`, `route access` pass |
| TC-UI5-06 (AC6) | PASS (L1) | `/exams/[id]/results` 1 panel; `/exams/[id]/similarity` 1 panel (GV); TA không có route similarity / tab (dev `exam results\|similarity` pass); cuộn 1.000 dòng ≤ 25 ms: chỉ số của dev (QC chưa tự đo — cần 1.000 dòng) |
| TC-UI5-07 (AC7) | PASS (L3) | GV: `/inbox`, `/students`, `/attendance`, `/gradebook`, `/gradebook/scheme`, `/grading`, `/documents` 1 panel; `/insights` 2, `/analytics` 5, `/observability` 3, `/settings/integrations` 3; TA: `/inbox`, `/students`, `/gradebook`, `/grading` 1 — NEST 0, WALL 0, STRONG 0, `ox` 0 ở 1440 / 1024 / 375; lề panel 12 px ở 375; `/attendance`, `/inbox` TOUCH chỉ còn liên kết ẩn |
| TC-UI5-08 (AC8) | PASS | `git diff 36824f4..1d6caea`: `nav.ts` không đổi, `ACCESS` / `EXAM_DYNAMIC` không đổi; số pass / skip / fail trước → sau (QC chạy cả hai commit): `exam` 77 / 5 / 0 → 77 / 5 / 0; `today` 24 / 4 / 0 → 24 / 4 / 0; `account` 50 / 14 / 0 → 50 / 14 / 0; `class-join` 61 / 21 / 0 → 61 / 21 / 0; `shell` 21 / 23 / 0 → 21 / 23 / 0; `package.json` / lock không đổi; `ui-allow:` 9; `ui-antipatterns.sh` **22 `✓`, 0 `✗`**, `--selftest` **22 / 22** |

## Việc sau
- **Dev:** B1 (đặt `EmptyState` của `/exams` rỗng vào một `Panel`, cho cả Staff và Sinh viên; thêm ca e2e mock rỗng); L1 (Dialog trong Panel).
- **QC:** chấm lại TC-UI5-03 (và UI-04 TC-UI4-03 rỗng) sau khi dev sửa; ảnh mốc 14 / 14 hai lần ở UI-07.
