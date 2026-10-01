# Prompt cho `ba` — Spec v5.1 (góp ý #19: 37 lỗi QC thăm dò)

**Ưu tiên cao hơn spec sprint 2**: prototype phải trình bày cuối tuần. Dừng sprint 2 ở điểm an toàn (commit dở dang vào worktree `TA_Agent_v2-s2` nếu cần), làm việc này trong repo chính (`/Users/kuro/Documents/TA_Agent`, nhánh `sprint/1.5-mock-ui`), xong quay lại sprint 2.

## Đọc
Góp ý #19 trong `docs/sprints/1.5/proposals.md`, `docs/sprints/1.5/qc/explore-v4.md` + ảnh `shots/qc-explore-v4/`, spec v5 hiện tại.

## Việc
1. Với **từng** lỗi E1–E37: lỗi vi phạm AC đã duyệt thì ghi AC đó; nếu AC chưa nói đủ để kiểm thì viết AC mới đo được (route, vai, bề rộng, số liệu đúng, chữ hiển thị đúng). Thêm vào `SRS.md` một bảng truy vết `E# → AC → mức`.
2. Số liệu phải nhất quán giữa các màn (E6, E10, E12, E19, E25, E26, E28, E29, E30): chốt **một nguồn mock** cho mỗi con số trong SRS (ví dụ số phiếu chờ > 24 h, tỉ lệ AI tự trả lời, thứ tự sắp sinh viên, tên/tuần tài liệu) để dev lấy chung, không viết cứng từng màn.
3. E2 (SV chưa vào lớp) và E5 (PII ở Threads): viết rõ màn SV D thấy gì ở từng route, và trình tự hộp chặn PII khi tạo thread / phản hồi; khớp 4.3.1 v5.
4. Nâng spec lên **v5.1**, ghi lịch sử trích #19. Không đổi số liệu #12/#13.
5. Câu hỏi sản phẩm: ghi `QUESTIONS.md` kèm mặc định an toàn, không chặn.

Chỉ sửa `docs/sprints/1.5/spec/**`. Commit `sprint 1.5: spec v5.1 (#19)`. Trả lời tóm tắt (số AC mới, số E map vào AC cũ) rồi quay lại sprint 2 (`/Users/kuro/Documents/TA_Agent_v2-s2/docs/sprints/2/prompts/ba-spec.md`). Làm thật kỹ.
