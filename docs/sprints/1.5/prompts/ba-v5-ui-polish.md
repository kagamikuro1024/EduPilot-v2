# Prompt cho `ba` — Spec v5 "UI polish + Threads như thật" (góp ý #17, #18)

Chủ dự án chưa ưng giao diện prototype ("logo hơi nhỏ, …"; "luồng Threads chưa ổn một tí nào, chưa thấy mock phản hồi thật") và giao PM toàn quyền nghiệm thu UI/UX. PM đã rà ảnh và ghi góp ý **#17 và #18 (ACCEPTED)** ở `docs/sprints/1.5/proposals.md`. Ảnh bằng chứng: `docs/sprints/1.5/shots/pm-review-v4/*.png` (mở từng ảnh mà xem).

## Việc
1. Đọc `docs/team/CONTEXT.md`, góp ý #17, `docs/design/DESIGN.md` (§ logo/brand, shell, topbar, §14 route liên quan, §21 phản mẫu, §22 định nghĩa xong), `docs/UX.md` (375 px).
2. Nâng `docs/sprints/1.5/spec/US.md` + `SRS.md` lên **v5**, ghi lịch sử phiên bản trích #17. Thêm yêu cầu chung **FR-X12 UI polish** và AC đo được (có số px, token, route, bề rộng) cho 7 lỗi PM nêu:
   - (a) Logo sidebar: kích thước mark + wordmark, chiều cao vùng brand thẳng hàng topbar; logo trên topbar 390 px và trang `/login`.
   - (b) Bộ chọn lớp topbar hiện đủ tên lớp ở 1440 px; quy tắc rút gọn có `…` + tooltip ở bề rộng hẹp.
   - (c) `/inbox` panel danh sách: không cắt ngang chữ; tab cuộn ngang hoặc xuống dòng; ở 390 px chỉ hiện danh sách, bấm vào mới mở chi tiết, có nút quay lại.
   - (d) `/inbox` panel chi tiết: ô trả lời nối ngay sau thông tin, không khoảng trống vô nghĩa.
   - (e) `/chat` desktop: danh sách phiên là một panel (FR-X11); cột hội thoại và composer cùng trục, cùng bề rộng; composer ghim đáy vùng chat.
   - (f) Menu hồ sơ hiện tên người thật của vai (GV, TA, Admin trong mock), dòng phụ là vai trò.
   - (g) `/settings/llm`: hàng nhà cung cấp dùng lưới cột cố định, cột trạng thái và nút thẳng hàng.
3. **Tự rà thêm** mọi route còn lại (chạy `pnpm -C frontend build && pnpm -C frontend exec next start -p 3300`, đổi vai ở `/login`, xem 1440 và 390 px). Thấy lỗi cùng loại (tràn, cắt chữ, lệch cột, khoảng trống vô nghĩa, phân cấp chữ yếu, trái DESIGN §21) thì thêm AC, ghi rõ route + ảnh. Nhớ dừng server khi xong.
4. **Threads như thật (#18)**: viết lại phần Threads của US/SRS thành kịch bản mock cụ thể, dev chỉ việc làm theo:
   - Bảng seed: mỗi thread trong danh sách có đúng số phản hồi như số đếm hiển thị; ghi rõ người viết (SV khác, TA, GV), nội dung, thời gian, trạng thái xác nhận. Ít nhất `t-cbc` và 2 thread khác phải có thảo luận qua lại đầy đủ để trình diễn.
   - Bộ câu trả lời AI theo chủ đề (`THREAD_TOPICS`): với mỗi chủ đề, câu trả lời Socratic (gợi ý + câu hỏi ngược), nguồn trích đúng tài liệu chủ đề. Quy tắc chọn câu trả lời theo từ khoá câu hỏi; không khớp thì AI nói không chắc và đề nghị chuyển giảng viên (khớp D49).
   - Trình tự hiển thị: trạng thái "đang soạn" → stream chữ → hiện nguồn → nhãn "Chờ xác nhận". Thời lượng từng bước.
   - "Hỏi trợ lý AI": ô soạn có chữ thì hỏi theo chữ đó, trống thì hỏi theo câu hỏi gốc; luôn có phản hồi nhìn thấy.
   - Sau khi SV gửi phản hồi: một phản hồi mock trễ vài giây (TA hoặc bạn cùng lớp, nội dung theo chủ đề) + chuông thông báo. GV/TA: thread mới hiện ở "Hôm nay" và bộ đếm.
   - Một tiêu đề vùng thảo luận duy nhất.
   - Đối chiếu `legacy/frontend/src/app/threads/[id]/page.tsx` để giữ đủ hành vi hệ cũ.
5. Ngoài #18, không đổi hành vi nghiệp vụ, số liệu (#12, #13) hay luồng đã duyệt ở v4.
6. Câu hỏi sản phẩm (nếu có) ghi `docs/sprints/1.5/spec/QUESTIONS.md`, tự chọn phương án an toàn ghi rõ giả định, không chặn.

Chỉ sửa `docs/sprints/1.5/spec/**` (và ảnh rà của bạn ở `docs/sprints/1.5/shots/ba-review-v5/`). Commit đúng các đường dẫn đó, thông điệp `sprint 1.5: spec v5 UI polish + Threads như thật (#17, #18)`. Trả lời tóm tắt danh sách AC mới và dừng chờ PM duyệt.
