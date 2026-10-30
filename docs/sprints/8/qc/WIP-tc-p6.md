# QC P6 — trạng thái dở dang (2026-10-31, dừng theo lệnh PM trước khi restart máy)
Nhánh: commit trên `HEAD` detached rồi push `HEAD:sprint/8-p6`.

## Đã xong (viết, chưa chạy vì chưa có code của dev)
- `tc-US-P6-01.md` … `tc-US-P6-06.md` (TC-P6-01…06; mỗi file có nhánh lỗi, phân quyền, câu hỏi cho BA/PM).
- `scripts/`: `p6-hand-calc.py` (tự tính tay độc lập: **đã chạy** → expected_final_grades.csv 30/30, cases-tong-hop.csv 18/18 khớp), `p6-citation-check.py` (đã chạy trên fixture: 10/10 và 6/6), `p6-make-xlsx.py` (đã sinh 12 tệp thử vào `/tmp/qc-p6-xlsx`), `p6-env.sh`, `p6-rbac.sh`, `p6-go-tests.sh` (chỉ `bash -n`, chưa chạy với stack).

## Chưa làm
- `tc-US-P6-07.md`, `tc-US-P6-08.md`, `tc-US-P6-09.md`, `tc-US-S8-SEED.md`, `tc-US-S8-UI.md`, `gate-P6.md`.
- Chấm US-S8-UI theo `handoff/dev-US-S8-UI.md` → `report-US-S8-UI.md` (chưa chạy lệnh nào).
- Script pha 2 đã hứa trong TC: `p6-preview-vs-csv.mjs`, `p6-load1000.sql`, `p6-put-race.sh`, `p6-grid-ui.mjs`.
- Chưa push commit nào trước lần này; chưa chạy docker / playwright (không có tiến trình cần dừng).

## Việc kế tiếp
Viết TC-07, 08, 09, SEED, UI, gate-P6 → commit `sprint 8: QC test case P6` → báo PM; rồi chấm US-S8-UI.
