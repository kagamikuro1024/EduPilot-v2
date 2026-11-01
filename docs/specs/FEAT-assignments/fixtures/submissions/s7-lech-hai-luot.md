Bài viết phân tích chiến dịch tấn công chuỗi cung ứng nhằm vào phần mềm giám sát hạ tầng SolarWinds Orion, bị phát hiện tháng 12 năm 2020. Đây là ví dụ điển hình cho việc kẻ tấn công không đánh thẳng vào mục tiêu cuối, mà chọn một nhà cung cấp phần mềm được hàng nghìn tổ chức tin tưởng và cài đặt sẵn trong mạng nội bộ.

Tác nhân được các hãng an ninh quy cho nhóm APT29 (còn gọi là Cozy Bear), một nhóm tấn công có tài trợ nhà nước, hoạt động kiên trì và ưu tiên gián điệp hơn phá hoại. Bề mặt tấn công bị khai thác là hệ thống dựng phần mềm của SolarWinds — nơi mã nguồn được biên dịch và ký số trước khi phát hành — chứ không phải máy chủ của khách hàng.

Kẻ tấn công cấy một thư viện độc hại tên SUNBURST vào tệp SolarWinds.Orion.Core.BusinessLayer.dll. Vì bản cập nhật vẫn được ký bằng chứng thư số hợp lệ của SolarWinds nên mọi cơ chế kiểm tra chữ ký ở phía khách hàng đều cho qua. Khoảng 18.000 tổ chức tải bản cập nhật có mã độc, nhưng kẻ tấn công chỉ chọn khoảng 100 mục tiêu để khai thác tiếp.

Chuỗi tấn công gồm bốn bước: xâm nhập môi trường dựng phần mềm, cấy mã vào bản phát hành chính thức, nằm im 12–14 ngày để vượt qua các phép kiểm tra tự động, rồi gọi về máy chủ điều khiển qua tên miền avsvmcloud.com với lưu lượng giả dạng giao thức cập nhật hợp lệ của chính Orion.

Ở giai đoạn sau, nhóm tấn công nạp TEARDROP và Cobalt Strike để di chuyển ngang trong mạng, rồi đánh cắp khoá ký SAML của máy chủ liên kết danh tính. Với khoá này họ tự phát hành thẻ truy cập cho chính mình (kỹ thuật Golden SAML) và leo thang lên quyền quản trị dịch vụ thư điện tử đám mây mà không cần biết mật khẩu hay vượt qua xác thực đa yếu tố.

Thiệt hại không nằm ở dữ liệu bị xoá mà ở niềm tin: nhiều cơ quan liên bang Mỹ phải coi toàn bộ hệ thống là đã bị xâm nhập, dựng lại máy chủ và xoay vòng toàn bộ khoá cùng chứng thư. Riêng SolarWinds công bố chi phí ứng phó năm 2021 khoảng 40 triệu đô la, chưa kể thiệt hại thương hiệu và các vụ kiện sau đó.

Em chưa tìm được số liệu đáng tin về thời gian gián đoạn dịch vụ của từng tổ chức bị ảnh hưởng, nên phần đánh giá tác động vận hành ở đây mới dừng ở mức định tính.

Về phòng thủ, em đề xuất ba nhóm biện pháp: (1) phân đoạn mạng để máy chủ giám sát không có đường đi thẳng ra Internet, (2) xác thực đa yếu tố và giới hạn quyền cho tài khoản dịch vụ, kèm xoay vòng khoá ký định kỳ, (3) kiểm tra tính toàn vẹn của bản dựng bằng quy trình build tái lập được và bản kê thành phần phần mềm (SBOM).

Ngoài ra cần giám sát hành vi bất thường của tiến trình hợp lệ, ví dụ Orion tự mở kết nối DNS tới một tên miền lạ ngoài danh sách đã biết, vì chữ ký số hợp lệ không còn là bằng chứng đủ để tin một bản cập nhật.

Bài học rút ra: trong mô hình không tin mặc định, phần mềm của bên thứ ba cũng là một bề mặt tấn công. Kiểm soát phải đặt cả ở nơi nhận bản cập nhật chứ không chỉ ở nơi phát hành.
