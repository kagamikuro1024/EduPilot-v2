# Sprint 1.5 — ghi cho luận văn

- **Prototype trước backend (D51)**: dựng toàn bộ 27 nhóm route bằng dữ liệu mock trong app Next.js thật, trạng thái giả lập ở `localStorage`, để chốt luồng và giao diện với người dùng trước khi viết backend. Primitive + shell ở `frontend/src/shared/` được giữ lại làm nền cho PU.
- **Một nguồn số liệu** (`mock/derive.ts`, FR-X13): lỗi phổ biến nhất QC tìm thấy là cùng một con số khác nhau giữa các màn (số phiếu chờ, tỉ lệ AI tự trả lời, thứ tự sinh viên). Gom mọi số về một hàm dẫn xuất xoá cả nhóm lỗi — bài học áp dụng cho truy vấn thật sau này.
- **Kiểm bố cục bằng máy** (`audit.mjs`): đo tràn ngang, chữ bị cắt, dấu `…` không có tooltip, vùng chạm < 44 px trên 4 vai × 3 bề rộng (495 phép đo). Từ 15 FAIL ở v5 về 0 ở vòng 2.
- **QC thăm dò ngoài spec** tìm được 37 lỗi (8 mức cao, gồm 2 lỗi quyền / dữ liệu cá nhân) mà test case viết từ AC không bắt được → thêm bước thăm dò vào quy trình trước khi chốt spec.
- **Threads "như thật"**: câu trả lời AI chọn theo từ khoá + chủ đề, có nhánh "chưa đủ chắc chắn" chuyển giảng viên (D49), trình tự đang soạn → stream → nguồn; mô phỏng đúng hành vi hệ thống thật sẽ có ở P3/P4 để giảng viên đánh giá được luồng.
- **Chi phí đội agent**: subagent đọc lại toàn bộ bối cảnh mỗi lần → cấm khi chưa được PM cho phép; giới hạn tốc độ API (429) làm dev/QC dừng ~3 giờ, xử lý bằng gửi lại việc khi hết hạn.
