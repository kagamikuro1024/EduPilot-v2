# Sprint 8: điểm dừng (2026-10-31, chủ dự án restart máy)

- Spec `FEAT-gradebook` **APPROVED** (`a3280c95`); chủ dự án chốt Q1–Q6, TLR-1, 2, 7, 17.
- **dev:** US-S8-UI xong (CI xanh, #1); US-P6-01 xong (`ddbc8948`); US-P6-02 dở: `rules.go` WIP. Kế tiếp `engine.go` + test 30/30 → `docs/sprints/8/handoff/dev-US-P6-02-WIP.md`, rồi US-P6-03 … SEED.
- **qc:** tc-US-P6-01…06 + script xong (`a15313ee`); dở TC-07, 08, 09, SEED, UI, `gate-P6.md`, chấm US-S8-UI → `docs/sprints/8/qc/WIP-tc-p6.md`.
- **ba, research, writer:** rảnh.
- Dev đã tắt colima: bật lại bằng `colima start` (dùng chung với container llm-eval của chủ dự án).

Khởi động lại: `herdr agent start <n> --kind omp --pane <pane> -- --model claude-sonnet-5-5` (research: `claude-opus-5-5`), gửi prompt vai, rồi giao dev / qc đọc tệp WIP trên để làm tiếp. Văn phòng: `cd ~/Documents/EduPilot-office && ./start.sh`.
