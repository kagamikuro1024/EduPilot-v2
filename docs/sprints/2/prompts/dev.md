# Prompt cho `dev` — Thi công sprint 2 (PG Nền Go)

**Làm trong worktree `/Users/kuro/Documents/TA_Agent_v2-s2` (nhánh `sprint/2-pg`)**, không phải repo chính (repo chính là nhánh 1.5). Mọi lệnh chạy với `cwd` đó; git dùng `git -C /Users/kuro/Documents/TA_Agent_v2-s2`.

## Đọc
`docs/team/CONTEXT.md`, `docs/team/DEV.md`, `AGENTS.md` (luật 1–15, quy ước Go), `docs/sprints/2/plan.md`, `docs/sprints/2/proposals.md` (#1 build tag `testroutes`, #2), spec **v1.2 APPROVED** `docs/specs/FEAT-pg-foundation/{US.md,SRS.md,QUESTIONS.md}`, `docs/phases/PG.md`, `docs/ARCHITECTURE.md` §2–§5, §8, `docs/SYSTEM_DESIGN.md` §2–§5. Test case của QC ở `docs/sprints/2/qc/` chỉ để tham khảo cách kiểm, không sửa.

## Thứ tự
US-PG-01 → 02 → 03 → 04 → 05 → 06 → 07, mỗi story:
1. Code + test Go table-driven (testcontainers-go cho Postgres/Redis/MinIO; colima: đặt `DOCKER_HOST`, `TESTCONTAINERS_RYUK_DISABLED=true` trong Makefile nếu cần).
2. `cd backend-go && go vet ./... && golangci-lint run && go test -race ./...` xanh; từ US-PG-02: `sqlc generate && sqlc diff` sạch.
3. Commit `US-PG-0N: <việc>` đúng đường dẫn (`git commit -- <paths>`), push.
4. Handoff `docs/sprints/2/handoff/dev-US-PG-0N.md`: đã làm gì, AC nào, lệnh tự kiểm + kết quả thật đã chạy, nợ.

## Luật
- Thư viện chỉ theo bảng `ARCHITECTURE.md` + `kin-openapi` (D52, chỉ test). Cần thêm gì khác → ghi góp ý vào `docs/sprints/2/proposals.md`, dùng phương án trong bảng.
- Không endpoint nghiệp vụ. Không ghi đĩa cục bộ, không biến toàn cục giữ trạng thái người dùng.
- Không đụng `frontend/` (trừ khi build gãy), `legacy/`, `scripts/team-up.sh`.
- Spec sai/thiếu: ghi góp ý (cột Quyết định PM để trống), làm phần khác; không tự đổi spec, không hỏi QC/BA trực tiếp.
- Cuối: chạy cổng PG trong `PG.md` (gồm `docker compose … up -d --scale gateway=2`, Caddy, k6 smoke), dán kết quả vào `docs/sprints/2/handoff/dev-GATE-PG.md`; `docker compose down` khi xong.

Trả lời tóm tắt và dừng. Làm thật kỹ.
