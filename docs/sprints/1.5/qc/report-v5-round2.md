# QC report v5 vòng 2 — **DỞ DANG** (PM ngắt để ưu tiên sprint 2)

Bản dựng: worktree riêng `TA_Agent_qcr2` @ `b15a5ba` (spec v5.2), `pnpm -C frontend build` OK, `next start -p 3400`, Chrome headless mặc định. Đo ngày 2026-10-03. Server đã tắt.

## Đã chạy (kết quả thật)
| Cổng | Kết quả |
| --- | --- |
| `pnpm -C frontend lint` | rc=0 |
| `bash scripts/ui-antipatterns.sh` | rc=0, 0 dòng ✗ |
| `proto-curl.sh all` (`F=http://localhost:3400`) | **497 PASS / 0 FAIL** (v5 vòng 1: 497) |
| `regress-v24.mjs` | **30/30 PASS** (TC-00-69, TC-01-153, TC-02-90, TC-02-91 đều PASS mọi bề rộng; v5: FAIL) |
| `threads-timeline.mjs` | 62 dòng, **0 FAIL** |
| `pii-matrix.mjs` | **48/48 PASS** |
| `audit.mjs` vai student | 165 dòng, **0 FAIL** (kể cả `wide`; `/threads` 390/375 hết nở khung) |
| `audit.mjs` vai teacher | 170 dòng, **0 FAIL** (`/students`, `/attendance`, `/grading`, `/documents` 720/1024 đạt) |

## Chưa chạy
`audit.mjs` vai ta, admin; chạy tay lại từng TC FAIL của v5 (bảng BUG → PASS/FAIL); đi tay DEMO 15 phút (`demo-run.mjs`); kết luận từng story. **Chưa có bảng BUG → PASS/FAIL, chưa có kết luận story/DEMO.**

## Lỗi sản phẩm mới
Chưa phát hiện (chưa đi tay).

## Suy luận từ máy (chưa xác nhận tay)
Các lỗi cao v5 liên quan đến bố cục (BUG-v5-01-1 `/threads` 390, 02-3 tab inbox, 02-4 ô Buổi, 00-5 nút hồ sơ, 01-2 chip) đã hết ở máy đo; các lỗi logic (02-1 badge, 02-2 link thẻ, DEMO-1 mất trạng thái chốt, 03-1 ô làm tròn, 04-2 /analytics) cần đi tay mới kết luận.
