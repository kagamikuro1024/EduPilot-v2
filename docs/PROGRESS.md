# Tiến độ

Claude Code: ĐỌC file này đầu mỗi phiên, CẬP NHẬT cuối mỗi phiên. Giữ ngắn; chi tiết nằm trong git log.

## Đang ở đâu
- Repo: `origin` = `github.com/kagamikuro1024/TA_Agent_v2` (private). `main` = sprint 1 + 1.5 + 2 + 3 + 4 (merge 2026-10-04). Repo cũ `TA_Agent` = remote `old-origin`, không push thêm.
- Lộ trình: **11 sprint + sprint 1.5 chen giữa** (`docs/sprints/ROADMAP.md`; PE thi hằng tuần = sprint 5, D54). Chạy cuốn chiếu; `sprint/4-p2` và `sprint/5-pe` xếp chồng, gộp `main` trước khi mở PR.
- Sprint gần nhất: **4 — P2 Lớp học** — xong, cổng P2 ĐẠT CÓ ĐIỀU KIỆN (LCP + TBT `warn` — **đã trả ở US-PU-06** theo #14: TBT `error` ở `lighthouserc.json`, LCP `error` ở lượt devtools), `docs/sprints/4/report.md`. Kế tiếp: sprint 5 (PE + US-PU-06): spec APPROVED + TC xong, chờ thi công.
- Workflow giữ HF Space cũ thức đã dời vào `legacy/.github/` ở repo mới (repo cũ vẫn tự chạy bản của nó).

## Bảng phase

| Phase | Trạng thái | Bắt đầu | Xong | Cổng nghiệm thu | Ghi chú |
| --- | --- | --- | --- | --- | --- |
| P0 Chuẩn bị | Xong | 2026-10-01 | 2026-10-01 | PASS (`sprints/1/qc/gate-P0.md`) | Viết mới toàn bộ (D45), chỉ Go (D46) |
| PG Nền Go | Xong | 2026-10-01 | 2026-10-03 | PASS (`sprints/2/qc/report-GATE-PG.md`) | Gateway không trạng thái, `--scale gateway=2`, image 9,4 MB |
| PU Nền giao diện | Xong | 2026-10-02 | 2026-10-03 | PASS, LCP có điều kiện (`sprints/3/qc/report-GATE-PU.md`) | Ảnh mốc, axe, Lighthouse CI, `ci/ui-drift` |
| P1 LLM Gateway | Xong | 2026-10-02 | 2026-10-03 | PASS (`sprints/3/qc/report-GATE-P1.md`) | openai-go/v3, Scheduler ba làn, cầu dao; Anthropic chưa ghi replay |
| P2 Lớp học | Xong | 2026-10-03 | 2026-10-04 | ĐẠT CÓ ĐIỀU KIỆN (`sprints/4/qc/report-GATE-P2.md`) | Phiên cookie xoay vòng, nối chỉ bằng email, "Hôm nay", seed API thật; 0 lỗ hổng |
| P3 Hai kênh + PII | Chưa | | | | |
| P4 Escalation + Mail | Chưa | | | | |
| P5 CRM + 360 | Chưa | | | | |
| P6 Sổ điểm | Chưa | | | | |
| P7 Chấm bài | Chưa | | | | |
| P8 Tài liệu + Lịch | Chưa | | | | |
| P9 Luyện đề | Chưa | | | | |
| P10 Đánh giá | Chưa | | | | → vạch bảo vệ |
| PR Sẵn sàng thí điểm | Chưa | | | | → vạch thí điểm thật |

## Luồng end-to-end (tick khi có spec E2E xanh cho đường chính + một nhánh lỗi)

F1 ☐ · F2 ☐ · F3 ☐ · F4 ☐ · F5 ☐ · F6 ☐ · F7 ☐ · F8 ☐ · F9 ☐ · F10 ☐ · F11 ☐ · F12 ☐ · F13 ☐ · F14 ☐ · F15 ☐ · F16 ☐ · F17 ☐ · F18 ☐

## Phiên gần nhất
- Ngày: 2026-10-03
- Đã làm: sprint 4 — P2, 12 story PASS, cổng P2 đạt có điều kiện sau 1 vòng sửa (góp ý #1–#15, TL-1). Sprint 3 đã merge `main` (PR #3).
- Đang dở: không. Đội dừng tuần này (chủ dự án yêu cầu 2026-10-04).
- Bước kế tiếp cụ thể: tuần sau bật colima (giảm còn 4 GiB), tạo lại worktree `sprint/5-pe`, dev thi công sprint 5 theo `docs/sprints/5/prompts/dev.md` bắt đầu bằng US-PU-06.

## Nợ (việc thấy cần nhưng ngoài phạm vi phase)
- Image object storage lâu dài (đang `pgsty/minio` fork) → lát blob của PG chốt + ghi D mới (proposals #7).
- `typescript` ghim `^5` (typescript-eslint chưa hỗ trợ TS 7).
- GitHub Actions ghim theo major, chưa theo SHA.
- Lịch WORKFLOW §6 chưa điều chỉnh theo D45/D46 — chủ dự án quyết.
- 9 câu hỏi sản phẩm của kịch bản demo (`specs/FEAT-demo-script/QUESTIONS.md`) — chặn P1/P2/P4/P7.
- `thesis-notes/legacy-perf.md` thân bài còn tiếng Anh.
- ~~**Lighthouse (PM #15 / #14):** `total-blocking-time` của 6 route người dùng đang `warn`…~~ **ĐÃ TRẢ (US-PU-06, sprint 5, #14):** TBT 6 route `error` ở `lighthouserc.json` (mô phỏng, ngưỡng 200 giữ nguyên); LCP `error` 2.500 ms ở `lighthouserc.devtools.json` (throttle thật; mô phỏng không chấm LCP vì sàn ≈ 254 KB JS ở 1,6 Mbps). Hồ sơ: `docs/sprints/5/handoff/dev-US-PU-06.md`, `docs/sprints/5/techlead.md` TL-1 / TL-2. Còn lại: `/dev/ui` giữ `warn`; TBT hai đỉnh (≈ 90 / ≈ 175) do một tác vụ chạy mã khung (`EvaluateScript` ≈ 230 ms ở 12×) rơi trong hay ngoài cửa sổ FCP → TTI.
- Caddy: upstream tĩnh + `health_uri` khi số bản gateway cố định (P10/PR, sprint 2 #3).
- **Nguồn việc của "Hôm nay" mà phase sau phải đăng ký (US-P2-11, SRS 4.7 bảng bậc):** P4 `TICKET` (10), `AI_CONFIRM` (50), thread; P5 điểm danh, `STUDENT_ATTENTION` (95) + "Lớp cần chú ý"; P6 `GRADE_SCHEME_UNCONFIRMED` (70); P7 hạn nộp, `GRADING_REVIEW` (30), `APPEAL` (20), `UNMATCHED_SUBMISSION` (60); P9 `QUESTION_REVIEW` (90), QUIZ; P10 insight. Mỗi phase gọi `today.Aggregator.Register(provider)` ở `today.NewService` và thêm Kind vào `allowed` (bộ lọc theo vai) — Provider P2 hiện chỉ có ở `internal/today/providers.go`. `continue[]` của sinh viên rỗng tới khi P3 / P9 đăng ký.

## Ánh xạ migration
| Số goose | Tên | Phase |
| --- | --- | --- |
| 00001 | pg_platform | PG |
| 00002 | llm | P1 |
| 00003 | course_foundation | P2 |
| 00004 | auth_hardening | P2 |
| 00005 | vn_fold | P2 |
| 00006 | weekly_exam | PE (US-PE-01) |

## Việc chỉ chủ dự án làm được
- [x] API key ≥ 2 provider LLM (OpenAI + Gemini, 2026-10-03; nên xoay khoá vì đã dán trong chat, đặt trần chi phí)
- [ ] Hỏi thầy hướng dẫn về D4 (trước tuần 4)
- [ ] Hộp thư thử + app password cho IMAP/SMTP (trước P7)
- [ ] Quy chế môn học thật (trước P6, nếu có)
- [ ] Chấm tay ≥ 60 bài theo rubric (bắt đầu ở P7, xong trước P10)
- [ ] Duyệt 100 câu hỏi AI sinh, ghi tỷ lệ (P9)
- [ ] VPS hoặc máy demo (trước P10)
- [ ] Gặp trường về pháp lý dữ liệu cá nhân, hạ tầng, mail, LLM được phép (trước PR; bắt đầu hỏi từ sớm)
- [ ] 2 người ngoài dự án dùng thử (PR)
- [ ] Quyết lịch sau D45 (lùi vạch bảo vệ hay cắt) — trước sprint 2
