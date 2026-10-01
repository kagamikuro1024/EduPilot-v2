# Sprint 1 — Góp ý của đội

Ai trong `ba`, `dev`, `qc` thấy điều gì không hợp lý thì thêm một dòng và báo PM. PM quyết định (xem `docs/team/PM.md` §5). Trong lúc chờ: làm theo spec hiện hành, trừ khi việc đó gây hỏng hoặc vi phạm `CLAUDE.md`.

| # | Ai | Vấn đề | Đề xuất | Lý do + bằng chứng | Ảnh hưởng nếu không đổi | Quyết định PM | Ngày |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | ba | `docs/phases/P0.md` dòng 7, 20, 25 vẫn yêu cầu "Python AI", `src/` Python 3.11 + `pytest -q`, job Python trong CI — trái D46 | Sửa P0.md theo D46: mục tiêu còn hai phần mới (Go gateway, Next.js); bỏ dòng 20; bỏ `Python (pytest -q)` ở L2 | D46 bỏ service Python; spec FEAT-scaffold / FEAT-ci đã viết theo D46 (`950bc08`). `grep -nE 'Python\|pytest\|src/' docs/phases/P0.md` → dòng 7, 20, 25 | QC hoặc `/gate P0` đối chiếu P0.md sẽ đòi `src/` + pytest, báo FAIL sai, hoặc dev dựng thừa khung Python | | 2026-10-01 |
| 2 | ba | Toolchain đã cài là `node@20` (`plan.md` dòng 46), trong khi D48 và FEAT-scaffold FR-12 / FEAT-ci FR-3 dùng Node 24 | Cài `node@24` (Homebrew) trước khi dev làm US-P0-02; sửa dòng Toolchain trong `plan.md` | D48: Node 24 LTS, Node 20 hết hỗ trợ 04/2026 (`plan.md` "Quyết định PM tự chốt") | Máy dev build bằng Node 20 còn CI chạy Node 24 → lệch hành vi, lỗi chỉ hiện ở một bên | | 2026-10-01 |
