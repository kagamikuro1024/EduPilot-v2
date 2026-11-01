<!-- ĐÁP ÁN CHUẨN Bài tập 03 (`assignments.reference_answer`). Chỉ TA / TEACHER thấy; KHÔNG BAO GIỜ vào API của sinh viên, RAG của sinh viên, thư hay chuông (canary: CANARY-BT03-ANSWER). BA sở hữu. -->

CANARY-BT03-ANSWER

**Tác nhân và bề mặt tấn công (tiêu chí 1).** Tác nhân được quy cho nhóm APT29 (Cozy Bear), có tài trợ nhà nước, ưu tiên gián điệp. Bề mặt bị khai thác: hệ thống dựng phần mềm (build system) của SolarWinds, nơi mã được biên dịch và ký số trước khi phát hành; khách hàng nhận bản cập nhật Orion hợp lệ, ký đúng chứng thư.

**Chuỗi tấn công (tiêu chí 2).** (1) xâm nhập môi trường dựng phần mềm; (2) cấy backdoor SUNBURST vào thư viện SolarWinds.Orion.Core.BusinessLayer.dll của bản phát hành chính thức; (3) bản cập nhật có chữ ký hợp lệ được phát cho khoảng 18.000 tổ chức; (4) mã độc nằm im 12–14 ngày rồi gọi về máy chủ điều khiển qua tên miền avsvmcloud.com, giả dạng lưu lượng của Orion; (5) với khoảng 100 mục tiêu được chọn: nạp TEARDROP / Cobalt Strike, di chuyển ngang, đánh cắp khoá ký SAML (Golden SAML) và leo thang lên dịch vụ thư điện tử đám mây.

**Tác động (tiêu chí 3).** Nhiều cơ quan liên bang Mỹ và doanh nghiệp phải coi hệ thống đã bị xâm nhập, dựng lại máy chủ, xoay vòng khoá và chứng thư; thiệt hại chính là niềm tin vào chuỗi cung ứng phần mềm; SolarWinds công bố chi phí ứng phó năm 2021 khoảng 40 triệu đô la; chưa có số liệu công khai đáng tin về thời gian gián đoạn từng tổ chức (chấp nhận nêu rõ là định tính).

**Phòng thủ (tiêu chí 4).** Phân đoạn mạng cho máy chủ giám sát (không ra Internet trực tiếp); xác thực đa yếu tố, giới hạn quyền tài khoản dịch vụ, xoay vòng khoá ký; kiểm tra toàn vẹn bản dựng (build tái lập được, SBOM); giám sát hành vi bất thường của tiến trình hợp lệ (ví dụ DNS lạ); không tin chữ ký số là đủ.
