# Báo cáo QC — US-UI-05 (Giảng viên / TA → `Panel`)
**Kết luận: PASS có điều kiện** (chấm lại ở `c920b6c`; lần đầu FAIL TC-UI5-03 ở `1d6caea`). B1 đã sửa: stack thật ở `c920b6c`: GV vào lớp 761988 (rỗng) `/exams` → **1** panel chứa `EmptyState` "Chưa có bài thi nào" + nút; `sv.gioi` `/exams` rỗng trong panel; TA lớp 761987 1 panel. Điều kiện: người dùng **chưa vào lớp nào** (`sv.moi`) mở `/exams` vẫn thấy `EmptyState` "Bạn chưa vào lớp nào" **ngoài** panel (0 panel) — trạng thái khác (không có lớp), ngoài chữ AC3, báo dev xem; L1 (Dialog trong Panel) còn.

## Lỗi / lệch
- **B1 (đã sửa ở `c920b6c`) (AC3 — `/exams` rỗng không nằm trong `Panel`).** Repro: đăng nhập `teacher@edupilot.local` (lớp mặc định 761988, chưa có bài thi) → mở `/exams` → `document.querySelectorAll('[data-ep-panel]').length` = **0**; `EmptyState` "Chưa có bài thi nào" + nút `Tạo bài thi` nằm trơn trên canvas (ảnh `shots/ui05-real/exams-real:teacher-1440.png`). AC3 / UI-04 AC3: "rỗng = **một `Panel`** + `EmptyState` có nút". Bộ test của dev xanh vì mock luôn trả có dữ liệu. TA vào lớp 761987 (có bài thi): `/exams` 1 panel — đúng. Nên kiểm cả trạng thái rỗng ở `/exams` Sinh viên (UI-04 AC3, cùng yêu cầu) — QC sẽ chấm lại UI-04 phần này khi dev sửa.
- **L1 (TITLE).** `/exams/[id]/results` (GV và TA): selector `[data-ep-panel] h2.ep-section-title` = **1** — tiêu đề "Công bố điểm?" của `Dialog` đang đóng, nằm trong DOM của Panel (không hiển thị). Cùng gốc với L2 của UI-04 (`/settings`): đề nghị đưa `Dialog` ra portal hoặc định nghĩa lại TITLE bỏ `dialog` / phần tử ẩn (Q-QC-UI04-2). Không tính FAIL.
- **L2 (AC1).** Hôm nay Staff có đúng 2 vùng ("Việc cần xử lý", "Sắp tới"); "Lớp cần chú ý" chưa có (PM #2: chỉ khi có dữ liệu) — không lệch.
- **L3 (TOUCH, 375 px).** Ngoài phạm vi bắt buộc của AC7 (chỉ `/attendance`, `/inbox`: đạt, chỉ còn liên kết ẩn "Bỏ qua điều hướng" 159 × 40): `/exams/[id]` có 3 `input` 20 × 20 (ô tích trong form Thông tin); `/documents` có 20 nút `Có` 53 × 32 (màn mock). Ghi nhận.
- **L4 (ảnh trước).** Không có ảnh "trước" bằng dữ liệu thật cho màn Staff (stack cũ phải dựng lại); ảnh trước / sau ở phiên giả chỉ là khung tải. Bằng chứng thị giác: ảnh thật sau (`shots/ui05-real`) + ảnh mốc `visual.spec` mới (xem báo cáo UI-07).

## Kết quả
| TC | KQ | Bằng chứng |
| --- | --- | --- |
| TC-UI5-01 (AC1) | PASS (L2) | GV và TA `/`: **2** panel (1440 / 1024 / 375), NEST 0, TITLE 0, STRONG 0, WALL 0, `ox` 0, lề 12 px ở 375; `today.spec.ts` 24 / 4 / 0 |
| TC-UI5-02 (AC2) | PASS | `/class/members` GV và TA: 1 panel (Toolbar + bảng trong panel), `Tabs` ngoài; `/class/settings`: 3 panel (mỗi mục một); NEST 0, STRONG 0; dev `members` pass; `class-join.spec.ts` 61 / 21 / 0 |
| TC-UI5-03 (AC3) | PASS (chấm lại `c920b6c`) | `/exams` có dữ liệu (TA 761987): 1 panel; **rỗng (GV 761988): 1 panel chứa `EmptyState`** (lần đầu 0); `sv.gioi` rỗng: trong panel; chưa vào lớp (`sv.moi`): 0 panel (ghi chú) |
| TC-UI5-04 (AC4) | PASS | `/exams/[id]` (1 panel / tab, NEST 0, TITLE 0); dev `exam editor` pass (≥ 4 `PanelSection`, Drawer không có panel trong `[role=dialog]`) |
| TC-UI5-05 (AC5) | PASS | `/questions` GV và TA: 1 panel; `Drawer` là lớp nổi; dev `questions`, `route access` pass |
| TC-UI5-06 (AC6) | PASS (L1) | `/exams/[id]/results` 1 panel; `/exams/[id]/similarity` 1 panel (GV); TA không có route similarity / tab (dev `exam results\|similarity` pass); cuộn 1.000 dòng ≤ 25 ms: chỉ số của dev (QC chưa tự đo — cần 1.000 dòng) |
| TC-UI5-07 (AC7) | PASS (L3) | GV: `/inbox`, `/students`, `/attendance`, `/gradebook`, `/gradebook/scheme`, `/grading`, `/documents` 1 panel; `/insights` 2, `/analytics` 5, `/observability` 3, `/settings/integrations` 3; TA: `/inbox`, `/students`, `/gradebook`, `/grading` 1 — NEST 0, WALL 0, STRONG 0, `ox` 0 ở 1440 / 1024 / 375; lề panel 12 px ở 375; `/attendance`, `/inbox` TOUCH chỉ còn liên kết ẩn |
| TC-UI5-08 (AC8) | PASS | `git diff 36824f4..1d6caea`: `nav.ts` không đổi, `ACCESS` / `EXAM_DYNAMIC` không đổi; số pass / skip / fail trước → sau (QC chạy cả hai commit): `exam` 77 / 5 / 0 → 77 / 5 / 0; `today` 24 / 4 / 0 → 24 / 4 / 0; `account` 50 / 14 / 0 → 50 / 14 / 0; `class-join` 61 / 21 / 0 → 61 / 21 / 0; `shell` 21 / 23 / 0 → 21 / 23 / 0; `package.json` / lock không đổi; `ui-allow:` 9; `ui-antipatterns.sh` **22 `✓`, 0 `✗`**, `--selftest` **22 / 22** |

## Việc sau
- **Dev:** `/exams` khi chưa vào lớp nào (`sv.moi`) nằm trong `Panel` nếu muốn thống nhất; L1 (Dialog trong Panel).
- **QC:** ảnh mốc 14 / 14 hai lần: đã chạy ở UI-07.
