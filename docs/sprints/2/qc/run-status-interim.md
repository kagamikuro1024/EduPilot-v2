# QC sprint 2 — trạng thái chạy dở (chuyển máy, 2026-10-03)

Worktree `sprint/2-pg` @ `bcc80c1`. **Chưa có kết luận cổng PG; chưa viết `report-*.md`.** Nhật ký thô: `run-logs/0{1..7}.log` (story 01 là lượt chạy lại sau khi sửa môi trường).

## Môi trường đã dựng
- `make -C backend-go test` rc=0 (21 gói OK) sau `test-clean`.
- `go test` trần cần `~/.testcontainers.properties` (`docker.host=unix://$HOME/.colima/default/docker.sock`, `ryuk.disabled=true`) — theo README; lượt đầu của pg01 đỏ 10 TC vì thiếu tệp này (lỗi môi trường).
- Script cần stack **đầy đủ** (caddy, frontend) đã lên: `docker compose --env-file .env.local -f docker-compose.local.yml -p edupilot up -d --build --scale gateway=2 --wait`. `run-all.sh` chết sau story 01 (nghi bị cắt khi chạy nền); chạy từng script `pgNN.sh` ở foreground.
- Đã `down` stack và `test-clean`; không còn container.

## Kết quả từng script (đếm của script, chưa triage)
| Story | PASS | FAIL | MANUAL | TC FAIL |
| --- | --- | --- | --- | --- |
| 01 | 79 | 6 | 0 | 09, 33, 41, 76, 78, 79 |
| 02 | 63 | 7 | 0 | 37, 39, 47, 48, 50, 64, 65 |
| 03 | 103 | 5 | 0 | 07, 74, 76, 87, 104 |
| 04 | 55 | 3 | 0 | 02, 41, 42 |
| 05 | 59 | 13 | 2 | 02, 05, 18, 20, 21, 23, 33, 35, 39, 41, 59, 61, 68; MANUAL 25, 27 |
| 06 | 34 | 3 | 1 | 05, 18, 30; MANUAL 38 |
| 07 | 61 | 6 | 2 | 15, 28, 54, 56, 57, 58; MANUAL 45, 63 |
| GATE | — | — | — | chưa chạy |

## Triage đã làm (story 01)
- TC-41: 2 test `--- PASS`, rc=0 → nghi lỗi script (cờ "no tests to run" của gói `cmd/gateway`).
- TC-78: `git grep` trúng `legacy/.env.example:43` (tệp hệ cũ) — đúng như proposals #9(2); TC đúng chữ → FAIL, cần PM chốt pathspec `':!legacy'`.
- TC-79: `.env.example` = `.env.local` (dev sao chép) — proposals #9(1), cần PM chốt.
- TC-09 (ms 3038 > 1000, không có dòng error): proposals #4 chưa có quyết định; chưa đo tay với khoá 31 byte.
- TC-33: TTL `ep:idem` 83433 < 86300 — có thể do khoá cũ (tuổi khoá); chưa đo lại.
- TC-76: 1 dòng warn thay vì ≥ 2 — chưa điều tra.
- Proposals #4–#10 (dev) chưa có trạng thái của PM; nhiều FAIL story 07 khớp #5–#9 (script, không phải mã).

## Chưa làm
Triage FAIL story 02–07; `gate-pg.sh`; bước tay (rút mạng SSE, tắt gateway giữa stream, `--scale gateway=2`, Idempotency-Key đôi, `down -v` rồi `up`); `report-US-PG-0{1..7}.md`, `report-GATE-PG.md`; kết luận PASS/FAIL cổng.
