# Smoke hồi quy sprint 5.5 — kết quả
Mã đã chạy: be0c0d96 (sprint/11-p10-eval) · Stack ep-r · Người chạy: qc2 (SmkC) · Ngày: 2026-11-15
Tổng TC cũ: 58 · Đạt: 0 · Trượt: 0 · K (a/b/c): 0 · LT: 2 · Chưa chạy: 56 — bản dựng cổng ĐÃ build (`pnpm build:gate`), spec mock đang chạy nền (`/tmp/qc2-c5/pw55b.json`); chạy `python3 evidence-5/scripts/finalize_55.py` khi xong để nối bảng spec vào tệp này.

**Chưa chạy gần hết — hết ngân sách lượt sau khi ưu tiên sprint 5.** Các ca 5.5 đều là Playwright chạy trên bản dựng cổng + gateway giả (`panels.spec.ts`, `shell.spec.ts`, `a11y.spec.ts`, `exam.spec.ts`…, cần `pnpm build:gate`), không chạy được trực tiếp trên ep-r; phần thăm dò UI thật dùng `evidence-5/ui/ui_pe.cjs` (axe/tràn ngang trên khung AppShell hiện tại) cho sprint 5.

## Bằng chứng lỗi thời (khung §23 của 10.7 thay panel nổi)
- `frontend/e2e/panels-variants.spec.ts` và route `/dev/panels` không còn (chỉ còn `src/app/dev/ui`) → TC-UI1-02, TC-UI1-03: LT.
- Token `ep-paper` không còn trong `frontend/src`, `frontend/e2e`, `docs/design`; `--ep-radius-panel` chỉ còn dùng ở `shared/ui/Panel.module.css`; `Page` chỉ còn độ rộng `work` (P-107-4), `PageState` đổi câu mặc định (P-107-3) → các ca đếm `ep-paper`/độ rộng reading-wide-full (TC-UI2-01/02, TC-UI3-03) sẽ LT khi chạy.
- Quan sát thật trên khung mới: không tràn ngang ở 1440 và 390 (`/questions`, `/exams`, danh sách SV, `/take`), mỗi màn đúng 1 nút primary; axe `label-content-name-mismatch` ở AppShell (DEF-R5-01, ảnh hưởng mọi màn thuộc 5.5 TC-UI7-03).

## Tổng hợp theo US
| US | TC tổng | P | F | LT | S |
| --- | --- | --- | --- | --- | --- |
| US-UI-01 | 8 | 0 | 0 | 2 | 6 |
| US-UI-02 | 11 | 0 | 0 | 0 | 11 |
| US-UI-03 | 8 | 0 | 0 | 0 | 8 |
| US-UI-04 | 8 | 0 | 0 | 0 | 8 |
| US-UI-05 | 8 | 0 | 0 | 0 | 8 |
| US-UI-06 | 6 | 0 | 0 | 0 | 6 |
| US-UI-07 | 9 | 0 | 0 | 0 | 9 |

## Kết quả từng ca
TC-UI1-02, TC-UI1-03 | LT | (xem trên). Mọi TC còn lại trong `docs/sprints/5.5/qc/tc-US-UI-0{1..7}.md` | S | chưa chạy (mẫu): cần bản dựng cổng + gateway giả; PM quyết có chạy nốt (khuyến nghị: chạy `pnpm build:gate` rồi các spec `--grep-invert @real` trên worktree qc2).

## Trượt: bước tái hiện
Không có ca trượt trong phần đã chạy; DEF-R5-01 (axe AppShell) tham chiếu `defects-5.md`.

## Ca K và lý do
K (a/b/c) = 0.
