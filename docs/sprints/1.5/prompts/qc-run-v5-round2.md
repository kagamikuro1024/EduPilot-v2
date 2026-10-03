# Prompt cho `qc` — Hàng việc (làm lần lượt, không chờ PM giữa các việc)

## Việc 1 — Sprint 1.5 vòng sửa 1: chạy lại (ưu tiên: demo với thầy cuối tuần)
Repo chính `/Users/kuro/Documents/TA_Agent`, nhánh `sprint/1.5-mock-ui`. Dev sửa 49/50 BUG-v5 (handoff `docs/sprints/1.5/handoff/dev-US-PROTO-0{0..4}.md`, bảng BUG → đã sửa → cách tự kiểm). BUG-v5-00-4 đóng theo góp ý #26 (spec v5.2, luật ngày lịch).
- Build + start (`next start` hoặc standalone) ở cổng 3400; Chrome headless theo cách PM dặn nếu treo.
- Chạy lại **mọi TC đã FAIL** ở report v5 + toàn bộ hồi quy tự động (`audit.mjs`, `regress-v24.mjs`, `threads-timeline.mjs`, `pii-matrix.mjs`, `proto-curl.sh`) + đi tay DEMO 15 phút.
- Report `docs/sprints/1.5/qc/report-v5-round2.md`: bảng BUG → PASS/FAIL, lỗi mới (nếu có), kết luận từng story + DEMO. Commit, tắt server.

## Việc 2 — Sprint 2: triage + cổng PG
Worktree `/Users/kuro/Documents/TA_Agent_v2-s2`. PM đã quyết #4–#10 (dòng #11 ở `docs/sprints/2/proposals.md`): sửa script theo đúng đề xuất, trích số góp ý. Tiếp `docs/sprints/2/prompts/qc-run.md` từ `run-status-interim.md`: triage FAIL story 01–07 (lỗi môi trường / script / sản phẩm), chạy `gate-pg.sh` + các bước tay, viết report từng story + `report-GATE-PG.md`. Chạy từng script ở foreground.

## Việc 3 — Sprint 3: viết test case (song song dev)
Worktree `/Users/kuro/Documents/TA_Agent_v2-s3`. Spec APPROVED `docs/specs/FEAT-llm-gateway/`, `docs/specs/FEAT-ui-foundation/`. Viết `docs/sprints/3/qc/tc-US-P1-0{1..5}.md`, `tc-US-PU-0{1..5}.md`, `tc-GATE-PU.md`, `tc-GATE-P1.md` (hộp đen từ AC, lệnh cụ thể). **Lập `docs/sprints/3/qc/audit-baseline.md` trước** (FEAT-ui-foundation Q10) — dev cần nó trước story PU đầu tiên.

Mỗi việc xong: commit đúng vùng `qc/**` của sprint đó, trả lời PM ≤ 8 dòng, rồi **tự sang việc kế**. Không mở subagent.
