# Báo cáo QC — US-UI-01 (ba phương án độ nổi của panel, D59)
**Kết luận: PASS** — AC1…AC8 đạt. Bản chấm: `ace622a` (trang `/dev/panels` + 18 ảnh + dừng chờ chọn) cho AC1–AC5; `origin/sprint/5.5-ui-panels` (HEAD `8e2313b`, D59 ở `0c3110b`) cho AC6–AC8. Phương pháp: build `gbuild` / `pbuild` tại chỗ, `next start -p 3300`; QC **tự đo** bằng `scripts/qc-measure.spec.ts` (NEST / TITLE / STRONG / WALL, `getBoundingClientRect`, token đọc qua canvas → sRGB 8 bit) và `scripts/contrast.py` (WCAG 2.x + ΔL OKLab tự cài), không dùng mã đo của dev; chạy thêm `panels-variants.spec.ts` của dev (18 pass). Ảnh QC: `docs/sprints/5.5/qc/shots/ui01/` (18 ảnh, 1440 / 1024 / 375, thời gian đóng băng 2026-10-29T09:20+07:00).

## Lỗi / lệch
- **L1 (AC3, TOUCH).** Ở 375 px có 1 phần tử tương tác < 44 px trên cả hai URL Sinh viên: `a[Bỏ qua điều hướng] 159 × 40` (liên kết bỏ qua điều hướng, ẩn ngoài màn hình tới khi focus). Không phải phần tử người dùng chạm thường; ghi nhận, không tính FAIL.
- **L2 (AC1/AC3).** Thanh nav ở `/dev/panels?role=teacher` theo vai **của phiên** (QC dùng `asDemo(teacher)` nhưng thanh nav hiện mục Sinh viên: "Chat riêng", "Luyện đề", "Kết quả của tôi") — dev đã ghi "khung theo vai của phiên đăng nhập"; nội dung mẫu đúng vai. Không ảnh hưởng nghiệm thu của mẫu.
- Không có bug.

## Kết quả
| TC | KQ | Bằng chứng |
| --- | --- | --- |
| TC-UI01-01 (AC1) | PASS | `gbuild`: 6 URL `a/b/c × teacher/student` → `200 200 200 200 200 200`; `surface` lạ → `200` (mặc định a / teacher); **`pbuild`** → 6 URL **`404`**, `/dev/ui` `404`; `grep -rn 'data-surface' frontend/src` ngoài `panel-variants.css` + `app/dev/` = **0** |
| TC-UI01-02 (AC2) | PASS | QC tự tính từ màu tính ra: ΔL canvas ↔ panel **5,03 (a) / 7,10 (b) / 3,06 (c)** (≥ 5 / 7 / 3); tương phản chữ nhỏ nhất **4,57 (a: `ink-3` / canvas)**, **4,60 (b: `red` / canvas)**, **4,65 (c: `ink-3` / `red-soft`)** — mọi cặp {ink, ink-2, ink-3, red, green, blue} × {canvas, panel, ô nhấn, surface-subtle, red-soft} ≥ 4,5; trùng số của dev (4,57 / 4,60). Bóng: (a) `rgba(71,32,37,.05) 0 1px …`, (b) / (c) `none`; bán kính 14 px ở cả ba |
| TC-UI01-03 (AC3) | PASS | 6 URL × 1440 / 1024 / 375 (18 lượt): NEST 0, TITLE 0, STRONG 0, WALL 0, `ox` 0; mẫu Giảng viên 3 panel, Sinh viên 2 panel; lề panel ở 375 px = **12 px**; `panels-variants.spec.ts` `sample rules` (6 × 2 dự án) PASS; TOUCH: L1 |
| TC-UI01-04 (AC4) | PASS | `ls handoff/ui-01/*.png \| wc -l` = 18; bảng số đo trong `dev-US-UI-01.md` ≥ 4,5 mọi ô (đối chiếu QC ở TC-02); có khuyến nghị dev |
| TC-UI01-05 (AC5) | PASS | từ `bc09dd2` tới `ace622a`: **không** commit nào đổi `DESIGN.md`, `AGENTS.md`, `ui-antipatterns.sh`, `Panel.tsx`, `tokens.css`; CSS đổi chỉ `app/dev/panels/*` và `panel-variants.css`; dev dừng đúng điểm chờ |
| TC-UI01-06 (AC6) | PASS | D59 ở `docs/DECISIONS.md` (`0c3110b`: phương án (a), PM chọn thay chủ dự án — `proposals.md`); `git log -S'D59'` → `0c3110b`, các commit sửa `DESIGN.md` / `AGENTS.md` / `ui-antipatterns.sh` (`9a5b97d`, `d5a2d9c`) **sau** D59 (quan hệ tổ tiên, TLR-10) |
| TC-UI01-07 (AC7) | PASS | bảng 4.1 của SRS đối chiếu **nguyên dòng** ở `DESIGN.md`, `AGENTS.md`: mọi dòng cũ `grep -cx` = 0; dòng mới có đúng 1 lần (kể cả `Disciplined panels: …` với `(1 px --ep-panel-border border and static --ep-elevation-1)`, `--ep-canvas` ở §4, dòng `AGENTS.md` có "riêng `Panel` dùng `--ep-radius-panel`"); `grep -rn ep-paper docs/design frontend/src frontend/e2e` = **0** (gồm `DESIGN_TOKENS.css`, TLR-7) |
| TC-UI01-08 (AC8) | PASS | HEAD: `ui-antipatterns.sh` **22 `✓`**, bốn dòng `✓` (Bóng ngoài Popover…, Bóng elevation ngoài Panel, Bo góc panel ngoài Panel, Panel lồng Panel); `--selftest` **`22 / 22 phép bắt được`**; `panel-variants.css` và `app/dev/panels` đã xoá; `grep -rn data-surface frontend/src` = 0 |

## Việc sau
- Ảnh trước / sau các màn thật (13 màn × 3 bề rộng) làm ở US-UI-07; ảnh QC của UI-02/03 xem `report-US-UI-02.md`, `report-US-UI-03.md`.
- Công cụ: `docs/sprints/5.5/qc/scripts/qc-measure.spec.ts` (sao chép vào `frontend/e2e` khi chạy, không commit ở `frontend`), `contrast.py`.
