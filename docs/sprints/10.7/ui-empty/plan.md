# Lát UI-EMPTY: trạng thái rỗng và dòng phụ dưới tiêu đề (chủ dự án 2026-11-14)

Chủ dự án chụp `/inbox` lớp KC106 (rỗng). Lỗi:
1. **Câu giải thích trong trạng thái rỗng** («Khi AI không đủ chắc chắn…») — vi phạm luật cứng chữ UI (CLAUDE.md 09/10/2026). Luật này thắng câu «empty = câu dạy bước kế» cũ trong DESIGN.md / PageState. 79/96 chỗ `<EmptyState>` có thân chữ.
2. **Dòng meta dưới tiêu đề lặp mã lớp** («KC106») đã có ở thanh trên.
3. **Bố cục rỗng**: khối rỗng dính góc trái trên trong panel trắng cao hết màn; nút «Xem tất cả» kiểu ghost thụt lề trông như chữ thường, và trùng tab «Tất cả».
4. Vì sao sót: khảo sát 9.1 đi màn với dữ liệu seed, ít gặp trạng thái rỗng; `ui-antipatterns.sh` không kiểm `EmptyState`.

## Việc
- **QC (ngay, chỉ đọc):** quét MỌI route × vai × 1440/375 ở trạng thái rỗng (`?state=empty` theo FR-X5 + lớp rỗng thật), chụp ảnh, liệt kê `docs/sprints/ui-empty/qc/sweep.md`: route, lỗi (1)/(2)/(3)/khác.
- **Dev (sau hàng sửa 10.6, nhánh `fix/ui-empty` từ main):**
  - `EmptyState`: bỏ thân chữ (chỉ tiêu đề ngắn + tối đa một hành động); xoá câu mặc định của `PageState`; xoá thân chữ ở mọi chỗ gọi. Chỗ nào câu đó là cú pháp / lý do khoá thật sự cần → đưa vào nhãn nút hoặc tooltip trên chính nút (luật PO).
  - `PageHeader meta`: bỏ meta lặp mã lớp / tên lớp đã có ở thanh trên.
  - Bố cục rỗng: khối rỗng căn giữa vùng danh sách; panel không kéo cao khi rỗng; hành động rỗng dùng nút chuẩn, không thụt; bỏ hành động trùng tab.
  - `scripts/ui-antipatterns.sh`: thêm luật chặn `<EmptyState …>chữ</EmptyState>` và câu mặc định cũ.
  - Mỗi nhóm 1 commit; Playwright + `ui-antipatterns` sạch; ảnh trước/sau các màn trong sweep.
- **QC:** kiểm lại theo sweep.md, PASS khi 0 mục còn lại.
- Đổi / xoá message có trong SRS → BA vá SRS cùng đợt.
