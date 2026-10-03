# Prompt cho `qc` — Chạy nghiệm thu sprint 2 (PG Nền Go)

Worktree `/Users/kuro/Documents/TA_Agent_v2-s2`, nhánh `sprint/2-pg`. Dev đã bàn giao: handoff `docs/sprints/2/handoff/dev-US-PG-0{1..7}.md`, `dev-GATE-PG.md` (commit `235f174`). Spec v1.3 + góp ý #1–#3 ở `docs/sprints/2/proposals.md`.

## Việc
1. `make -C backend-go test-clean` (research đã kéo image mới), kiểm `DOCKER_HOST` theo README backend.
2. Chạy toàn bộ `docs/sprints/2/qc/tc-US-PG-0{1..7}.md` và `tc-GATE-PG.md` bằng `scripts/run-all.sh` + các bước tay (rút mạng SSE, tắt một gateway giữa stream, `--scale gateway=2`, gửi đôi Idempotency-Key, `down -v` rồi `up`). Không tin số trong handoff: tự đo.
3. Report `docs/sprints/2/qc/report-US-PG-0{1..7}.md` + `report-GATE-PG.md`: PASS/FAIL từng TC, lệnh + kết quả thật, lỗi (bước tái hiện, mong đợi, AC). Kết luận cổng PG: PASS / FAIL.
4. Dọn: `docker compose … down` (giữ volume trừ khi TC cần `-v`), không để stack chạy.

Không mở subagent. Commit chỉ `docs/sprints/2/qc/**`. Trả lời bảng PASS/FAIL theo story + lỗi mức cao, rồi dừng.
