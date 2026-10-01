# Prompt cho `qc` — chạy TC US-P0-03, kiểm US-P0-01

1. **US-P0-03**: chạy `docs/sprints/1/qc/tc-US-P0-03.md`; handoff `docs/sprints/1/handoff/dev-US-P0-03.md`; spec `docs/specs/FEAT-ci/` (v2). AC1 chấm trên run của HEAD `origin/sprint/1-p0-prep` tại lúc chạy (dev vừa ghim `ubuntu-24.04`, proposals #11 — chờ run đó xong rồi chấm). Khi đã chấm xong AC3/AC4, ghi rõ trong report để PM cho dev xoá `ci/red-check`. Report `docs/sprints/1/qc/report-US-P0-03.md`.
2. **US-P0-01** (story tài liệu, không có handoff dev): viết TC từ `docs/specs/FEAT-demo-script/US.md` (AC1–AC9 đều có lệnh kiểm) → `docs/sprints/1/qc/tc-US-P0-01.md`, chạy, report `docs/sprints/1/qc/report-US-P0-01.md`. Đối chiếu thêm FLOWS/DESIGN §14 khi AC yêu cầu. Câu hỏi mở Q1–Q9 của FEAT-demo-script không chặn US này.

Mỗi shell `source ~/.zprofile`. Commit chỉ `docs/sprints/1/qc/**`. Trả lời ≤ 10 dòng rồi dừng.
