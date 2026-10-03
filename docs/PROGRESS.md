# Tiến độ

Claude Code: ĐỌC file này đầu mỗi phiên, CẬP NHẬT cuối mỗi phiên. Giữ ngắn; chi tiết nằm trong git log.

## Đang ở đâu
- Repo: `origin` = `github.com/kagamikuro1024/TA_Agent_v2` (private). `main` = sprint 1 + 1.5 + 2 (merge 2026-10-03). Repo cũ `TA_Agent` = remote `old-origin`, không push thêm.
- Lộ trình: **10 sprint + sprint 1.5 chen giữa** (`docs/sprints/ROADMAP.md`). Chạy cuốn chiếu; `sprint/3-pu-p1` và `sprint/4-p2` xếp chồng, gộp `main` trước khi mở PR.
- Sprint gần nhất: **2 — PG Nền Go** — xong, cổng PG PASS, `docs/sprints/2/report.md`. 1.5 — prototype — xong (`docs/sprints/1.5/report.md`). Đang làm: sprint 3 (PU + P1), sprint 4 (P2).
- Workflow giữ HF Space cũ thức đã dời vào `legacy/.github/` ở repo mới (repo cũ vẫn tự chạy bản của nó).

## Bảng phase

| Phase | Trạng thái | Bắt đầu | Xong | Cổng nghiệm thu | Ghi chú |
| --- | --- | --- | --- | --- | --- |
| P0 Chuẩn bị | Xong | 2026-10-01 | 2026-10-01 | PASS (`sprints/1/qc/gate-P0.md`) | Viết mới toàn bộ (D45), chỉ Go (D46) |
| PG Nền Go | Xong | 2026-10-01 | 2026-10-03 | PASS (`sprints/2/qc/report-GATE-PG.md`) | Gateway không trạng thái, `--scale gateway=2`, image 9,4 MB |
| PU Nền giao diện | Chưa | | | | |
| P1 LLM Gateway | Chưa | | | | |
| P2 Lớp học | Chưa | | | | |
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
- Đã làm: sprint 1.5 — prototype 5 story + DEMO PASS sau 2 vòng QC (spec v5.2, góp ý #15–#26). Thêm vai Research, luật cấm subagent, quy trình cuốn chiếu.
- Đang dở: sprint 3 (dev US-P1-01 dở dang), sprint 4 (spec).
- Bước kế tiếp cụ thể: thi công sprint 3 theo `docs/sprints/3/plan.md`, BA spec sprint 4.

## Nợ (việc thấy cần nhưng ngoài phạm vi phase)
- Image object storage lâu dài (đang `pgsty/minio` fork) → lát blob của PG chốt + ghi D mới (proposals #7).
- `typescript` ghim `^5` (typescript-eslint chưa hỗ trợ TS 7).
- GitHub Actions ghim theo major, chưa theo SHA.
- Lịch WORKFLOW §6 chưa điều chỉnh theo D45/D46 — chủ dự án quyết.
- 9 câu hỏi sản phẩm của kịch bản demo (`specs/FEAT-demo-script/QUESTIONS.md`) — chặn P1/P2/P4/P7.
- `thesis-notes/legacy-perf.md` thân bài còn tiếng Anh.
- Caddy: upstream tĩnh + `health_uri` khi số bản gateway cố định (P10/PR, sprint 2 #3).
- **LCP Lighthouse > 2,5 s (góp ý #26, sprint 3):** sau khi bỏ phông 500 (còn 400 / 600 / 700, subset latin + vietnamese) trung vị vẫn 2,71–3,39 s trên 7 route (CLS 0, TBT ≤ 83 ms, JS ≤ 230 KB đạt). `lighthouserc.json` đang `warn` **tạm** cho LCP, ngưỡng giữ 2500. **Cổng P2 (sprint 4): đưa mock ra khỏi bundle layout rồi đổi LCP về `error` ≤ 2,5 s.**

## Ánh xạ migration
| Số goose | Tên | Phase |
| --- | --- | --- |
| 00001 | pg_platform | PG |

## Việc chỉ chủ dự án làm được
- [ ] API key ≥ 2 provider LLM (trước P1)
- [ ] Hỏi thầy hướng dẫn về D4 (trước tuần 4)
- [ ] Hộp thư thử + app password cho IMAP/SMTP (trước P7)
- [ ] Quy chế môn học thật (trước P6, nếu có)
- [ ] Chấm tay ≥ 60 bài theo rubric (bắt đầu ở P7, xong trước P10)
- [ ] Duyệt 100 câu hỏi AI sinh, ghi tỷ lệ (P9)
- [ ] VPS hoặc máy demo (trước P10)
- [ ] Gặp trường về pháp lý dữ liệu cá nhân, hạ tầng, mail, LLM được phép (trước PR; bắt đầu hỏi từ sớm)
- [ ] 2 người ngoài dự án dùng thử (PR)
- [ ] Quyết lịch sau D45 (lùi vạch bảo vệ hay cắt) — trước sprint 2
