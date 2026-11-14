# Sprint 10.5: góp ý

| # | Ai | Vấn đề | Đề xuất | Lý do + bằng chứng | Ảnh hưởng | Quyết định PM |
| --- | --- | --- | --- | --- | --- | --- |
| P-105-1 | qc (`qc/questions.md` Q-S105-1…13) | Câu hỏi TC 10.5 | Phương án QC | — | — | **PM 2026-11-14:** **Q3** bản giả trả **0,25** (trung tính rõ ràng, dưới ngưỡng) cho đầu vào lạ để ép fallback; giữ AC «có khi P ≥ 0,5» — BA sửa mô tả bản giả trong SRS. **Q1** base URL provider decision cấu hình được (env + bảng provider) — dev làm, QC dựng máy chủ giả. **Q2** dev nêu bảng ca replay (đầu vào → P/nhãn) trong handoff + hỗ trợ `FAKE_DECISION_PROB` ghi đè. **Q6** fixture có 6 bài đối chứng là chuẩn; BA sửa «7» → «6». Các câu còn lại: theo đề xuất QC. Lượt thật US-DM-08 AC6 «chờ phép PM», không tính FAIL. |
