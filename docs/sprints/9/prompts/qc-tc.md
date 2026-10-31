# Prompt cho `qc`: test case sprint 9 (P7, hộp đen)

Viết TC trên `origin/sprint/9-p7` (worktree QC: switch --detach), commit `docs/sprints/9/qc/`, push `sprint/9-p7`. Việc chấm sprint 8 vẫn ưu tiên trước khi có handoff mới.

## Đọc
`docs/sprints/9/plan.md`; spec **APPROVED** `docs/specs/FEAT-assignments/` (US, SRS, QUESTIONS, `TL-REVIEW.md` "Quyết định PM", `fixtures/`); `docs/sprints/9/proposals.md`; `docs/phases/P7.md` (cổng 7 lệnh); `docs/FLOWS.md` F9, F11; `docs/DEMO_SCRIPT.md` bước 5 → 6. **Không đọc code của dev.**

## Việc
`docs/sprints/9/qc/tc-US-P7-01…11.md`, `tc-US-S9-SEED.md`, `gate-P7.md`. Mỗi AC ≥ 1 TC; mỗi story có TC nhánh lỗi và phân quyền. Phải có:
- **Sinh viên không bao giờ thấy nháp:** API, SSE, chuông, thư, AI chat, lịch, Hôm nay, `/me/grades`, mã lỗi khi nộp lại (TLR-17).
- **Chấm:** khớp `fixtures/expected_grading.json` (provider giả); điểm tiêu chí ≤ max, tổng = Σ, trích dẫn có thật; hai lượt lệch → `NEEDS_REVIEW`; bài injection → cờ, câu thật không bị xoá (TLR-4); bài trống, lạc đề; chấm mù (không tên / MSSV trong payload gửi LLM, kiểm qua `llm_audit` của ADMIN).
- **Hàng đợi:** chấm hai lần không chấm đôi; giết worker giữa chừng; dead-letter + chạy lại; chat vẫn trả lời trong lúc "Chấm tất cả".
- **Nộp:** sai loại / quá cỡ bị chặn trước khi tải; URL ký sẵn hết hạn; nộp lại → phiên bản; muộn → trừ theo Q2 + TLR-8 (gia hạn `due_at` thì hết trừ); 375 px.
- **Khớp:** email đã xác minh; MSSV chỉ theo danh sách lớp ROSTER / ADMIN (TLR-1); thư giả `From` không có `dmarc=pass` → không gán (TLR-2); ZIP; Chưa khớp + gán tay; bài `UNMATCHED` → `INCOMPLETE`, không 0 (TLR-9).
- **Công bố → sổ điểm P6:** điểm vào sổ đọc trực tiếp (TLR-14); sổ đã chốt; `expected_final_grades.csv` không đổi.
- **Phúc khảo:** trong hạn / quá hạn / lần 2 / sổ đã chốt; chỉ TEACHER quyết; không vào `/inbox` (TLR-3 A).
- **Teams giả lập:** 7 kịch bản, đồng bộ hai lần không trùng, nhãn "Môi trường giả lập", `TEAMS_CONSENT_REQUIRED` đọc hiểu được.
- **Cổng P7:** đủ 7 lệnh của `phases/P7.md`.

Xong: commit `sprint 9: QC test case P7`, push, báo PM ≤ 5 dòng.
