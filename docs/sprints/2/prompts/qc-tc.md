# Prompt cho `qc` — Test case sprint 2 (PG Nền Go)

**Làm trong worktree `/Users/kuro/Documents/TA_Agent_v2-s2` (nhánh `sprint/2-pg`)**; git chạy `git -C /Users/kuro/Documents/TA_Agent_v2-s2 …`.

Spec `docs/specs/FEAT-pg-foundation/{US.md, SRS.md, QUESTIONS.md}` đã APPROVED (104 AC, US-PG-01…07), kèm góp ý #1 ở `docs/sprints/2/proposals.md` (route thử `/api/v1/_test/*` khoá bằng build tag `testroutes`, image mặc định trả 404).

## Đọc
`docs/team/CONTEXT.md`, `docs/team/QC.md`, `docs/team/TEMPLATES.md`, `docs/sprints/2/plan.md`, `docs/phases/PG.md` (cổng nghiệm thu), spec trên.

## Việc
1. Viết `docs/sprints/2/qc/tc-US-PG-0{1..7}.md`: mỗi AC ít nhất một TC hộp đen có lệnh chạy cụ thể (curl / go test -run / psql / redis-cli / docker compose / k6) và kết quả mong đợi đo được. Nhánh lỗi trong spec mỗi nhánh một TC.
2. Viết `docs/sprints/2/qc/tc-GATE-PG.md`: đúng khối "Cổng nghiệm thu" và "Bạn tự kiểm" của `PG.md` thành từng bước có kết quả mong đợi (gồm `--scale gateway=2`, tắt một gateway giữa stream, rút mạng SSE 10 s, gửi đôi Idempotency-Key).
3. Script tự động trong `docs/sprints/2/qc/scripts/` (bash + curl + jq, hoặc k6 đã cài) cho các TC chạy được bằng lệnh; không thêm thư viện.
4. Chưa có code dev: không chạy, không đọc code dev.

Commit chỉ `docs/sprints/2/qc/**` trên nhánh `sprint/2-pg`. Trả lời số TC theo story rồi dừng. Làm thật kỹ.
