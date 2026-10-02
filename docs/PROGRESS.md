# Tiến độ

Claude Code: ĐỌC file này đầu mỗi phiên, CẬP NHẬT cuối mỗi phiên. Giữ ngắn; chi tiết nằm trong git log.

## Đang ở đâu
- Repo: `origin` = `github.com/kagamikuro1024/TA_Agent_v2` (private). Sprint 1 = `main` của repo này. Repo cũ `TA_Agent` = remote `old-origin`, không push thêm.
- Lộ trình: **10 sprint + sprint 1.5 chen giữa** (`docs/sprints/ROADMAP.md`). Chạy cuốn chiếu, nhánh xếp chồng: `sprint/2-pg` (từ `main`), `sprint/3-pu-p1` (từ 2 + 1.5), `sprint/4-p2` (từ 3).
- Sprint gần nhất: **1.5 — prototype giao diện** — xong, PASS, `docs/sprints/1.5/report.md`, PR chờ chủ dự án chốt. Sprint 2 (PG) đang QC cổng. Sprint 3, 4: kế hoạch + spec có sẵn, tạm dừng theo lệnh chủ dự án.
- Workflow giữ HF Space cũ thức đã dời vào `legacy/.github/` ở repo mới (repo cũ vẫn tự chạy bản của nó).

## Bảng phase

| Phase | Trạng thái | Bắt đầu | Xong | Cổng nghiệm thu | Ghi chú |
| --- | --- | --- | --- | --- | --- |
| P0 Chuẩn bị | Xong | 2026-10-01 | 2026-10-01 | PASS (`sprints/1/qc/gate-P0.md`) | Viết mới toàn bộ (D45), chỉ Go (D46) |
| PG Nền Go | Chưa | | | | Viết lại theo D45/D46 |
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
- Đang dở: sprint 2 (QC triage + cổng PG); sprint 3/4 dừng sau khi có kế hoạch + spec.
- Bước kế tiếp cụ thể: chủ dự án chốt PR 1.5 và PR 2 → PM merge `--no-ff` → gộp `main` vào `sprint/3-pu-p1`, gõ `tiếp` để thi công sprint 3.

## Nợ (việc thấy cần nhưng ngoài phạm vi phase)
- Image object storage lâu dài (đang `pgsty/minio` fork) → lát blob của PG chốt + ghi D mới (proposals #7).
- `typescript` ghim `^5` (typescript-eslint chưa hỗ trợ TS 7).
- GitHub Actions ghim theo major, chưa theo SHA.
- Lịch WORKFLOW §6 chưa điều chỉnh theo D45/D46 — chủ dự án quyết.
- 9 câu hỏi sản phẩm của kịch bản demo (`specs/FEAT-demo-script/QUESTIONS.md`) — chặn P1/P2/P4/P7.
- `thesis-notes/legacy-perf.md` thân bài còn tiếng Anh.

## Ánh xạ migration
| Số goose | Tên | Phase |
| --- | --- | --- |

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
