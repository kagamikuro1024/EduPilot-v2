# Báo cáo cổng nghiệm thu sprint 5.5 (UI panel nổi, FEAT-ui-panels v1.2)
**Kết luận: CHƯA ĐẠT — 1 điều kiện chặn (B1: axe `critical` ở `/exams/[id]/results`).** Còn lại ĐẠT. Bản chấm `c920b6c` (HEAD `sprint/5.5-ui-panels`, CI `success`); QC chạy lại mọi lệnh ở máy QC trên bản build, hai commit (`dfe05f0` trước, `c920b6c` sau).

## Chặn
1. **B1 — US-UI-07 AC3:** axe `aria-allowed-attr` + `aria-valid-attr-value` (critical) trên vùng cuộn bảng điểm ảo hoá (`aria-activedescendant` trên `role=region`, trỏ id không tồn tại) ở `/exams/[id]/results` với dữ liệu thật; có từ trước 5.5 (cùng lỗi ở `dfe05f0`), `a11y.spec.ts` không bắt vì dữ liệu giả. Chi tiết ở `report-US-UI-07.md`.

## Điều kiện vào cổng
| Điều kiện | KQ | Bằng chứng |
| --- | --- | --- |
| 7 story có báo cáo | PASS | UI-01 PASS; UI-02, 03, 04, 06 PASS có điều kiện; UI-05 PASS có điều kiện (chấm lại B1 `c920b6c`); UI-07 FAIL (B1) |
| CI GitHub xanh ở HEAD | PASS | run `c920b6c` `success` (`Go`, `Frontend`, `Judge attacks`) |
| D59 ghi, kiểm tổ tiên | PASS | `0c3110b` trước các commit sửa `DESIGN.md` / `AGENTS.md` / `ui-antipatterns.sh` |
| US-UI-01 AC6 có quyết định | PASS | D59 phương án (a) — PM chọn thay chủ dự án (`proposals.md`) |

## A. Kiểm theo US-UI-07 (G1…G9 của `gate-UI.md`)
| # | KQ | Bằng chứng |
| --- | --- | --- |
| G1 | PASS | 39 + 39 ảnh của dev; QC 72 ảnh thật trước / sau (`shots/ui07-*`) |
| G2 | PASS | `visual.spec.ts` 14 / 14 hai lần (image `v1.63.0-noble`); 14 tệp ảnh đổi = hợp handoff UI-02…05 |
| G3 | **FAIL** | B1; `color-contrast` 0 vi phạm ở 36 lượt; `axe-allow.json` 0 |
| G4 | PASS | NEST / WALL / STRONG 0 ở mọi màn đo; `ui-antipatterns` 22 `✓`, selftest 22 / 22 |
| G5 | PASS | `lhci` QC trước → sau: LCP devtools ≤ 1.945 ms, TBT ≤ 26 ms, CLS 0, JS Δ ≤ +0,5 KB (`/dev/ui` +1,3); CI xanh |
| G6 | PASS | Playwright 513 pass / 173 skip / 0 fail (≥ 443); lint, build rc 0; không thêm thư viện |
| G7 | PASS | nav / `ACCESS` không đổi; `audit-login.mjs` chưa chạy |
| G8 | PASS | đỏ ≤ 0,41 % (< 8 %); WALL 0; không `backdrop-filter` |
| G9 | CHỜ PM | `PROGRESS.md` + xác nhận của chủ dự án |

## B. Nhóm kiểm độc lập
| # | KQ | Bằng chứng |
| --- | --- | --- |
| B1 NEST / TITLE / STRONG / WALL | PASS | 36 lượt thật + 14 route Sinh viên + 19 Staff + 7 Admin ở các story: NEST 0, WALL 0, STRONG 0; TITLE thô 1–2 do `Dialog` đóng trong Panel (Q-QC-UI04-2, không FAIL) |
| B2 tương phản | PASS | QC tự tính WCAG từ màu đã giải: nhỏ nhất **4,57** (`ink-3` / canvas) ≥ 4,5; ô nhấn / `surface-subtle` / `red-soft` ≥ 4,65 |
| B3 lề mobile 12 px (TLR-4) | PASS | 375 px: panel cách mép đúng 12 / 12 px mọi màn; không tràn ngang |
| B4 bóng / bán kính đúng một chỗ (TLR-1) | PASS | chỉ `shared/ui/Panel.module.css`; `filter` / `backdrop-filter` 0; phép 20 / 21 `✓` |
| B5 đỏ (TLR-9) | PASS | ≤ 0,41 % |
| B6 ảnh trước / sau | PASS | 72 ảnh QC + 78 ảnh dev; nhận xét: canvas ấm ≠ panel trắng, tiêu đề ngoài panel, không card lồng, không tường KPI; sidebar / thanh trên trắng + kẻ 1 px |
| B7 hiệu năng (TLR-5) | PASS | xem G5; số QC từ `resource-summary:script` của `lhci` |
| B8 không hồi quy | PASS | Go không đổi (CI xanh), Playwright 513 / 0 fail, `visual` 14 / 14 ×2 |
| B9 phân quyền | PASS (một phần) | `nav.ts` / `ACCESS` không đổi; `audit-login.mjs` bốn vai chưa chạy |
| B10 khả năng dùng | PASS | `ox` 0; TOUCH 375 px: chỉ liên kết ẩn "Bỏ qua điều hướng" (159 × 40); trạng thái rỗng `/exams` trong Panel (UI-05 B1 đã sửa) |

## Điều kiện còn lại (không chặn)
- `audit-login.mjs` (bốn vai) chưa chạy; trạng thái "chưa vào lớp" (`sv.moi`) ở `/exams` nằm ngoài Panel (ghi ở UI-05); TITLE thô với `Dialog` đóng (Q-QC-UI04-2).
- Chữ nút chính Admin (`Mời giảng viên` / `Mở lớp`) khác AC — Q-QC-UI06-1 chờ BA.
- Duyệt thị giác của chủ dự án (C): PM / chủ dự án xem 13 màn × 3 bề rộng trước / sau.

## Câu hỏi
- **Q-QC-UI-1** (dữ liệu giả cố định): QC dùng cả dữ liệu thật (stack seed) — ảnh của dev dùng dữ liệu giả; hai nguồn nhất quán → *đề nghị đóng*.
- **Q-QC-UI-2** (chuẩn đo JS): QC đo bằng `lhci` ở cả hai commit trên cùng máy (Δ so sánh) → *đề nghị đóng*.
