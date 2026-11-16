# Trạng thái qc2 (dừng theo lệnh chủ dự án, 2026-11-16 tối)
ep-r đang chạy (13 container, gateway chạy lại với BLOB_PUBLIC_ENDPOINT=localhost:1143 + BLOB_PUBLIC_USE_SSL=true trong /tmp/ep-r.seed.override.yml — cần giữ khi dựng lại ep-r, nếu không UI tải tệp lên bị CSP chặn). Không xoá volume nào. Nhánh evidence: `qc2/regression`.

## Còn N theo sprint (số thật trong smoke-<N>.md)
| Sprint | P | F | LT | K | N |
|---|---|---|---|---|---|
| 1 / 1.5 / 2 | xem smoke-1/1.5/2 | 3 F + 3 F (DEF-R2-04 đã sửa) | | | ca X (pnpm dev) làm cuối trên project compose riêng |
| 3 | 443 | 1 (P105-44) | 5 | 16 | 58 (SmkO: GATEP1/GATEPU/PU01/PU05, P103-11/12/13/49/50, P104-48, P105-50..52) |
| 4 | 289 | 0 | 1 | 4 | 244 (chưa chạy ca nào trong đợt này) |
| 5 | 419 | 9 | 16 | 0 | 32 (k6, seed sạch PE09, cổng gate-pe/F19, PE02-25, PU06) |
| 5.5 | 50 | 1 | 7 | 0 | 0 |
| 6 | 298 | 0 | 6 | 2 | 230 |
| 7 | 321 | 1 | 2 | 9 | 249 |
| 8 | 405 | 4 | 0 | 0 | 103 (05-xx API/Today/nhắc nhở/chat LLM, 09-xx chat LLM, SEED 37, S8-UI 23, 04-05/29/30/31/34, 06-31 long task khi máy rảnh, 06-63 phần playwright gradebook rbac, 09-12 canary-scan, 09-46 trạng thái rỗng, 03-39..) |
| 9 | 195 | 0 | 0 | 152 | 152 |
| 9.1 | 199 | 67 | 51 | 0 | 97 (chạy lại nhóm F môi trường w4/w5 với timeout dài) |
| 9.5 | 90 | 24 | 18 | 0 | 29 |
| 10 / 10.5 / 10.6 | 137 / 80 / 142 | 0 | | | 108 / 103 / 118 |
| 10.7 | 86 | 0 | | | 0 (audit sau P-11-58 = 0 lệch) |
| 10.8 | 270 | 0 | | 2 K(a) | 0 |

## F chờ chạy lại sau sửa của dev
- DEF-R8-A1 (dev @6c75bd54): TC-P6-03-36, TC-P6-03-19 (kiểm lại nhánh stub của 03-19: weight abc phải ra PARAM_INVALID từng mục, kịch bản /evidence-8/scripts/s8r-03b.py).
- DEF-R8-A2 (dev2): TC-P6-04-03, TC-P6-07-47 (focus về <body> sau đóng Drawer/Dialog).
- Sprint 5: R5-03 (PE03-49), R5-04 (PE03-51), R5-06 (PE06-27), R5-07 (PE07-18), R5-08 (PE08-31), R5-09 (PE08-38), R5-10 (PE03-33), R5-11 (PE08-04), R5-12 (lint spec QC — PM: sửa kiểu/biến thừa trong frontend/e2e/qc2-*, 1 commit «qc2: lint spec QC», không đổi assertion).
- Sprint 3: R3-01 (P105-44, chữ EMBEDDING thô).
- 9.5: DEF-02, DEF-19 (một phần), DEF-R9.5-19 (SEED_DEFAULT_PASSWORD), DEF-R9.5-02 (canary CN0); 7: DEF-R7-01 đã đóng.

## Việc kế
1) Merge origin/sprint/11-p10-eval, chạy /tmp/qc2-rebuild.sh (nhớ giữ override BLOB), rerun các F trên. 2) Sprint 8 còn N (script s8r-*.py, spec frontend/e2e/qc2-r8/real-04|07|08). 3) Sprint 4, 6, 7, 9, 10, 10.5, 10.6, 9.5 còn N. 4) Ca X sprint 1/2 cuối cùng (project compose riêng, KHÔNG edupilot).
