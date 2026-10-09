# Ghi chú luận văn — sprint 5.5 (Giao diện panel nổi)

- **Đổi hướng thị giác có kiểm soát (D59):** chọn bằng ảnh thật (3 phương án × 2 vai × 3 bề rộng) thay vì tranh luận trên mô tả; tiêu chí đo được — tương phản thấp nhất ≥ 4,5, diện tích đỏ < 8 %, TBT +≤ 30 ms, JS +≤ 2 KB, CLS 0. Kết quả: TBT +≤ 6 ms, JS +≤ 1,4 KB, đỏ ≤ 0,42 %.
- **Luật thiết kế thành mã:** `ui-antipatterns.sh` thêm 3 phép (bóng ngoài elevation, bo góc panel lạ, Panel lồng Panel) kèm selftest — luật "không card lồng card" chuyển từ văn bản sang kiểm tự động.
- **Đổi giao diện toàn bộ mà không đổi hành vi:** một primitive `Panel` dùng chung, ảnh mốc sinh lại theo từng story; e2e hành vi giữ nguyên làm lưới an toàn.
- **Kiểm thử với dữ liệu thật bắt lỗi mà mock không thấy:** trang rỗng ngoài Panel (mock luôn có dữ liệu), lỗi a11y `aria-activedescendant` của bảng ảo hoá (chỉ lộ khi danh sách dài hơn cửa sổ), và ảnh mốc cũ thực chất là trang chặn quyền do token demo hết hạn dưới đồng hồ đóng băng.
