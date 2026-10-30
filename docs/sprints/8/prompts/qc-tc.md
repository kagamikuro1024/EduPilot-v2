# Prompt cho `qc`: test case sprint 8 (P6, hộp đen)

Worktree QC `/Users/kuro/Documents/TA_Agent_qc5`: `git fetch && git switch --detach origin/sprint/8-p6`.

## Đọc
`docs/team/QC.md`, `docs/team/CONTEXT.md`, `docs/sprints/8/plan.md`; spec **APPROVED** `docs/specs/FEAT-gradebook/` (US, SRS, QUESTIONS cột Trả lời, `TL-REVIEW.md` "Quyết định PM", `fixtures/`), `seed/expected_final_grades.csv`; `docs/sprints/8/proposals.md`; `docs/phases/P6.md` (cổng, "Bạn tự kiểm"); `docs/FLOWS.md` F10; `docs/DEMO_SCRIPT.md` bước 6. **Không đọc code của dev.**

## Việc
Viết `docs/sprints/8/qc/tc-US-P6-01…09.md`, `tc-US-S8-SEED.md`, `tc-US-S8-UI.md`, `gate-P6.md` (cột `TC-id → AC → tiền điều kiện → bước/lệnh → kết quả mong đợi`). Mỗi AC ≥ 1 TC; mỗi story có TC nhánh lỗi và phân quyền.

Phải có:
- **Đúng tuyệt đối:** 30/30 khớp `expected_final_grades.csv`; QC **tự tính tay** ít nhất 5 dòng (A, B, C + 2 ca biên làm tròn x,25 / x,75) từ `fixtures/quy-che-lop1.md` + `inputs-lop1.csv`, ghi phép tính trong TC; ca chia không hết (TLR-1), bài thi chưa công bố → `INCOMPLETE` (TLR-2), điểm 3 chữ số thập phân bị từ chối (TLR-7), điểm trừ ngoài quy chế không trừ (TLR-17), trọng số 0 (TLR-9).
- **Không float / không LLM tính điểm:** lệnh `grep` cổng P6; giải trình từng dòng tính lại được bằng tay.
- **AI chỉ trích nháp:** mục "chưa rõ" chặn `Xác nhận công thức`; quy chế lớp 2 thiếu làm tròn → mục "chưa rõ"; trích dẫn có số trang đúng; chỉ TEACHER xác nhận / chốt / xuất (TA, ADMIN, STUDENT → 403).
- **Chốt / mở khoá:** snapshot bất biến (UPDATE / DELETE / TRUNCATE bị chặn), mở khoá bắt buộc lý do, phiên bản mới, sinh viên bị đổi điểm được báo, chốt khi thiếu điểm → 409 kèm danh sách; đua `PUT` với `finalize` (TLR-6).
- **XLSX:** ánh xạ cột, xem trước, dòng lỗi (MSSV không thuộc lớp — khớp theo MSSV của danh sách lớp, KHÔNG theo MSSV tự khai, TLR-3), điểm ngoài thang, nhập lại cùng tệp không đôi; tệp zip bomb / > 5.000 dòng bị chặn (TLR-14); xuất có dòng ghi chú điểm chính thức.
- **Sinh viên:** không thấy điểm nháp / cột chưa công bố / điểm học phần trước khi chốt (kể cả qua API và qua AI); what-if không lưu; hỏi điểm người khác bị từ chối.
- **Nhắc:** lớp 2 chưa có quy chế → banner, `Chốt điểm` khoá kèm lý do, việc trên "Hôm nay", AI nói "chưa có công thức chính thức".
- **`/gradebook` 1.000 dòng:** Enter / Tab, cột tên dính, ảo hoá mượt; 409 hiện giá trị người kia.
- **US-S8-UI:** 3 màn cài đặt không còn câu mô tả dưới tiêu đề.

Nghiệm thu tay bằng `playwright-cli` (global). Xong: commit `sprint 8: QC test case P6`, push, báo PM ≤ 5 dòng.
