# Sprint 11: P10 Đánh giá + hoàn thiện (→ vạch bảo vệ)

Chủ dự án 2026-11-14: «chạy một mạch những cái chưa làm xong, chạy xong hết». Không chờ báo cáo giữa chừng; PM quyết proposal theo luật, chỉ dừng hỏi khi đụng tính điểm / auth / dữ liệu cá nhân mà spec không nêu.

Nguồn: `docs/phases/P10.md`, ROADMAP dòng 11, Nợ trong `docs/PROGRESS.md`. Máy: MacBook (Colima), không VPS. LLM thật qua 9router trên Mac (`127.0.0.1:20128`) hoặc khoá Gemini trong `.env.local`.

## Đợt
| # | Nội dung | Ai |
| --- | --- | --- |
| 0 | Hoàn tất 9.5: lượt chấm 1.000 bài (driver đã sửa), đo LLM thật (R1–R6) qua 9router local, báo cáo NFR; merge 9.5 | dev, techlead |
| 1 | M13-A Observability (`/observability` thật, chỉ ADMIN xem chi tiết + `audit_log`, cảnh báo lỗi/chi phí) — trả Nợ UX17-8/9 | dev |
| 2 | M13-B Báo cáo lỗ hổng kiến thức (`/insights` thật, `GenerateKnowledgeGapReport`, `data_watermark`, k ≥ 3) + thống kê điểm thi PE trên màn quan sát lớp | dev |
| 3 | LCP 7 trang ≤ 2,5 s (chủ dự án chốt sửa mã); Nợ L6 (P-10-10) | dev |
| 4 | Đánh giá `make eval` E1–E6 (E3 cần chấm tay của chủ dự án → ghi «chờ chủ dự án» nếu chưa có); E2E F1–F17 bảng luồng × spec | techlead, qc |
| 5 | Hoàn thiện: rà bảo mật, README / hướng dẫn chạy trên máy, visual QA 2 lượt, `DEMO_SCRIPT.md` chạy trọn 15 phút | dev, qc |

Thứ tự: BA spec `FEAT-p10-eval` (US + SRS + QUESTIONS) → QC TC → dev theo đợt, QC kiểm từng đợt. Đợt 0 song song ngay.
