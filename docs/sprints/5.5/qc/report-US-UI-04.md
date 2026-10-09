# Báo cáo QC — US-UI-04 (màn Sinh viên dùng `Panel`)
**Kết luận: PASS có điều kiện** — AC2…AC8 đạt theo số đo của QC; AC1 lệch một chi tiết (thiếu khối "Tiếp tục học", dev đã nêu, chờ BA). Điều kiện: `lhci` (CLS / LCP / TBT) không chạy được ở máy QC, gom ở UI-07 AC5; ca code `/take` đã QC chụp tay (dev chưa đo). Bản chấm `36824f4` (mã `c4276bc` + handoff), so với trước `8ba5ec9` (UI-03). QC đo bằng `scripts/qc-measure.spec.ts` ở **hai nguồn**: (1) 14 route Sinh viên với phiên giả `asDemo` ở 1440 / 1024 / 390 / 375 px (trước + sau, ảnh `shots/ui04-before|ui04-after`, số `measure/ui04-*.json`); (2) **stack thật** (`pnpm dev` + seed, đăng nhập thật `sv.gioi`, `sv.nguyco`, `sv.moi`, `sv.kha`) cho `/`, `/exams`, `/exams/[id]/take` (đã chấm / đang làm), `/join`, `/settings` ở 1440 / 1024 / 375 (ảnh `shots/ui04-real`, `measure/ui04-real*.json`) — vì ở phiên giả các màn dữ liệu chỉ hiện khung tải. Chạy thêm bộ e2e của dev (7 file): **279 pass, 0 fail**.

## Lỗi / lệch
- **L1 (AC1).** Màn Hôm nay có đúng hai `Panel` ("Việc nên làm tiếp", "Hôm nay") nhưng **chưa có vùng "Tiếp tục học"** mà AC1 liệt kê ("vùng 'Tiếp tục học' ở dưới"): `continue` trong API là `unknown[]` chưa có hợp đồng, `today.spec.ts` pin "ẩn khi rỗng" (dev ghi trong handoff). Chờ BA (Q-QC-UI04-1); không tính FAIL.
- **L2 (AC2, TITLE).** `/settings` với dữ liệu thật: `document.querySelectorAll('[data-ep-panel] h2.ep-section-title')` = **1** — là tiêu đề của `Dialog` "Đăng xuất mọi thiết bị khác?" nằm **trong DOM của Panel** (đang đóng, không hiển thị). Bộ test của dev bỏ phần tử ẩn nên xanh; phép đo chữ của spec (không loại ẩn) ra 1. Đề nghị đưa `Dialog` ra portal hoặc định nghĩa TITLE loại `dialog` / phần tử ẩn (Q-QC-UI04-2).
- **L3 (AC6).** `grep 'border-radius\|box-shadow'` ở CSS của màn mock còn các dòng: `ChatScreen.module.css` (bong bóng tin nhắn, chỉ báo chọn 2 px — nội dung, không phải khung vùng), `Library.module.css:5 .preview` (khung xem trước tài liệu, bo `--ep-radius-md`, viền 1 px, **nằm trong panel**) — mức "hộp trong panel" nhẹ; dev ghi 1 dòng ngoại lệ (`.dots i`) nhưng không kể `.preview`. Ghi nhận.
- **L4 (TOUCH).** Ở 375 px phần tử tương tác < 44 px duy nhất là liên kết ẩn "Bỏ qua điều hướng" (159 × 40) — như UI-01 / 03.
- **L5 (đo).** `/chat` ở 375 px có một panel (lịch sử phiên) rộng 0 × 0 (thu gọn), không phải khung hiển thị; panel còn lại cách mép 12 / 12 px.

## Kết quả
| TC | KQ | Bằng chứng |
| --- | --- | --- |
| TC-UI04-01 (AC1) | PASS (L1) | thật `sv.gioi`: `/` = **2** panel ở 1440 / 1024 / 375, NEST 0, TITLE 0, STRONG 0, WALL 0, `ox` 0, lề 12 px ở 375; `sv.nguyco` (không làm bài) 2 panel; `sv.moi` (chưa vào lớp) **1** panel (`EmptyState` + `Vào lớp bằng mã`); `today.spec.ts`, `shell.spec.ts` pass |
| TC-UI04-02 (AC2) | PASS (L2) | 14 route phiên giả × 4 bề rộng: mọi route có ≥ 1 panel (`/join` 1, `/settings` 2, `/chat` 2, `/threads` 1, `/threads/t1` 1, `/practice` 3, `/practice/history` 2, `/library` 1, `/calendar` 1, `/me` 4, `/assignments/a1` 1), NEST 0, STRONG 0, WALL 0, `ox` 0; thật: `/exams`, `/take`, `/join`, `/settings` đạt (L2 ở `/settings`) |
| TC-UI04-03 (AC3) | PASS | `/exams` thật (`sv.gioi`, cả hai bài đã công bố): 1 panel = một nhóm "Đã có điểm"; không panel trong `li`; h2 ngoài panel; dev `exams student` + `exam.spec.ts -g 'student list'` pass |
| TC-UI04-04 (AC4) | PASS | thật: trước giờ / đã chấm 1 panel (240 → 1200 px); **đang làm trắc nghiệm** 2 panel (danh sách câu 260 px + câu hỏi), thanh trên (`Câu i/n`, đồng hồ, `Nộp bài`) **không** nằm trong panel; **đang làm code (1440)**: 3 panel — danh sách câu 260 px | đề + test mẫu 368 px | soạn mã 516 px, NEST 0, TITLE 0, không tràn ngang; 375 px: panel cách mép 12 / 12; ảnh `take-intro-1440`, `take-running-mcq-1440`, `take-running-code-1440`, `take-running-375` |
| TC-UI04-05 (AC5) | PASS | `/join`, `/join/ABC123`: 1 panel, một nút chính; `/settings`: 2 panel (Mật khẩu, Phiên đăng nhập), `Dialog` bảo vệ giữ nguyên (L2); `account.spec.ts` 50 pass, `class-join.spec.ts` 61 pass |
| TC-UI04-06 (AC6) | PASS (L3) | màn mock dùng chung `Page` / `Section` / `Panel`: `/chat` 2 panel (lịch sử + hội thoại, `Composer` trong panel), `/threads` 1 panel, `/practice` 3, `/me` 4 (STRONG 0), `/calendar` 1, `/library` 1; `ui-antipatterns.sh` **22 `✓`**, `--selftest` **22 / 22**, "Từ kỹ thuật trong màn sinh viên" `✓` |
| TC-UI04-07 (AC7) | PASS | mọi route ở 375 và 390: `ox` 0; mọi panel hiển thị cách mép **12 / 12 px** (đo `getBoundingClientRect`, trừ panel rộng 0); ở 1024 lề 24 px; TOUCH: chỉ L4; `shell.spec.ts` ca `LEFT page-title 240 / 12` pass |
| TC-UI04-08 (AC8) | PASS | số pass / skip / fail trước → sau (QC chạy cả hai commit): `exam.spec.ts` 77 / 5 / 0 → 77 / 5 / 0; `today` 24 / 4 / 0 → 24 / 4 / 0; `account` 50 / 14 / 0 → 50 / 14 / 0; `class-join` 61 / 21 / 0 → 61 / 21 / 0; `shell` 21 / 23 / 0 → 21 / 23 / 0 (đúng); `package.json` / `pnpm-lock.yaml` không đổi; `ui-allow:` 9 → 9; chữ đổi duy nhất: "Việc nên làm bây giờ" → "Việc nên làm tiếp" |

JS truyền (CDP, trước → sau, KB): `/`, `/exams`, `/chat`, `/threads` +0,3; `/settings` +0,2; mọi route ≤ +0,3 (≤ 2 KB). `visual.spec.ts`: dev sinh lại 8 ảnh (chat, threads ×2, bốn ảnh 390 do lề 12 px) — QC chưa chạy lại trong image (làm ở UI-07 AC2).

## Việc sau
- BA: Q-QC-UI04-1 ("Tiếp tục học"), Q-QC-UI04-2 (TITLE với `Dialog` trong panel).
- QC: `lhci`, ảnh mốc 14 / 14 hai lần ở UI-07.
