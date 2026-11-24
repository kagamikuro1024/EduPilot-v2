# Sự cố vận hành đội agent (PM ghi, sprint 11)

PM ghi lại từ các tin báo cáo của agent và từ những gì PM tự kiểm khi sự cố xảy ra. Các sự cố này không có commit sản phẩm. Ghi ở đây để có nguồn trích dẫn. Giờ theo +07.

## SC-1. Lệnh nằm trong tin nhắn bị shell của người gửi chạy (đêm 15→16/11/2026)
- **Diễn biến:**
  - qc2 gửi PM một tin bằng lệnh nhắn tin của bộ chia terminal. Nội dung tin đặt trong nháy kép và có `pnpm dev` / `dev:down` nằm trong cặp backtick.
  - Shell của qc2 hiểu cặp backtick đó là lệnh con và chạy `pnpm dev` tại checkout chính `~/Documents/TA_Agent` (nhánh `sprint/1.5-mock-ui`).
  - Lệnh này dựng lại image gateway và frontend của compose project `edupilot` (stack demo) từ mã của checkout đó, rồi tạo lại các container phụ, chạy khoảng 2 phút.
  - `dev:down` không chạy được vì "command not found".
- **Phát hiện và xử lý:**
  - qc2 tự báo PM. qc2 `docker stop` 6 container demo, không `rm`, không `down -v`.
  - PM tự kiểm:
    - volume `edupilot_postgres_data` (tạo 12/11) và `edupilot_minio_data` còn nguyên;
    - migration cuối của DB demo là số 19 (14/11), tức sự cố không áp migration mới nào;
    - DB demo có 60 người dùng và 2 lớp, không bị seed lại.
- **Hậu quả:** image demo đã bị build đè. Muốn bật lại demo thì phải build từ `main` và `docker compose up -d` trong thư mục demo; không dùng `docker start` trơn.
- **Luật rút ra, áp cho cả đội:**
  - Không đặt lệnh trong backtick hay `$( )` bên trong tin nhắn đặt trong nháy kép. Dùng nháy đơn, hoặc heredoc có nháy.
  - Không chạy `pnpm dev` hay compose trên project `edupilot` hoặc trên checkout chính.
  - Luật nằm trong tệp khởi động đội ở công cụ điều phối mới (`prompts/MIGRATE.md`, `STATE.md`).

## SC-2. Script kiểm thử để tệp env thiếu biến bí mật trong 6 phút (16/11/2026, 01:14–01:20)
- **Diễn biến:** khi chạy ca cổng `gate:22`, script của qc2 viết lại tệp `.env.local` và tệp mới thiếu dòng `JWT_SECRET_KEY`.
- **Phát hiện và xử lý:**
  - qc2 tự phát hiện và khôi phục dòng đó sau khoảng 6 phút, rồi báo PM.
  - PM chưa xác nhận đó là tệp `.env.local` của worktree nào.
- **Luật rút ra:**
  - Script kiểm thử không được viết đè tệp env dùng chung. Muốn biến thể thì chép ra tệp tạm của riêng mình.
  - Không đưa khoá vào env, log hay commit.

## SC-3. Tin nhắn kẹt trong ô nhập sau khi đổi công cụ điều phối (16/11/2026, khoảng 21:00)
- **Diễn biến:**
  - Sau khi chuyển đội sang công cụ điều phối mới, lệnh gửi tin dán nội dung rồi bấm Enter ngay.
  - Với tin dài, Enter đến trước khi chương trình agent nhận xong khối dán và bị nuốt, nên tin nằm lại trong ô nhập.
  - Chủ dự án phát hiện qua ảnh chụp màn hình. Một báo cáo của QC gửi PM cũng bị kẹt theo cách này.
- **Sửa:**
  - Lệnh gửi tin chờ 0,6 s cộng thêm 0,1 s cho mỗi 500 ký tự, rồi bấm Enter, sau đó bấm thêm một Enter dự phòng.
  - Đã thử với tin 1.424 ký tự, 3 dòng.
  - Mã sửa nằm trong `bin/tsend` của bộ chuyển đội. Bộ này ở ngoài repo.
