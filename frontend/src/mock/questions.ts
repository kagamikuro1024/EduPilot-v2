// Ngân hàng câu hỏi An ninh mạng (/questions — US-PROTO-03): 80 đã duyệt + 20 chờ duyệt.
// 20 câu chờ duyệt là các câu AI sinh từ bài giảng — đúng chủ đề "AI chỉ nháp, người quyết".

export type QTopic =
  | "Mật mã đối xứng"
  | "Hàm băm và chữ ký số"
  | "Khoá công khai và PKI"
  | "Tường lửa, VPN và phân đoạn mạng"
  | "An ninh ứng dụng web"
  | "Xác thực và kiểm soát truy cập"
  | "Mối đe doạ và ứng phó sự cố";

export type QDifficulty = "easy" | "medium" | "hard";
export type QKind = "mcq" | "truefalse" | "short" | "essay";
export type QSource = "bank" | "ai" | "teacher" | "exam";
export type QStatus = "approved" | "pending" | "rejected";

export const TOPICS: QTopic[] = [
  "Mật mã đối xứng",
  "Hàm băm và chữ ký số",
  "Khoá công khai và PKI",
  "Tường lửa, VPN và phân đoạn mạng",
  "An ninh ứng dụng web",
  "Xác thực và kiểm soát truy cập",
  "Mối đe doạ và ứng phó sự cố",
];

export const DIFFICULTY_LABEL: Record<QDifficulty, string> = { easy: "Dễ", medium: "Trung bình", hard: "Khó" };
export const KIND_LABEL: Record<QKind, string> = { mcq: "Trắc nghiệm", truefalse: "Đúng/Sai", short: "Trả lời ngắn", essay: "Tự luận" };
export const SOURCE_LABEL: Record<QSource, string> = {
  bank: "Ngân hàng của trường",
  ai: "AI sinh từ bài giảng",
  teacher: "Giảng viên soạn",
  exam: "Đề cũ của học phần",
};
export const STATUS_LABEL: Record<QStatus, string> = { approved: "Đã duyệt", pending: "Chờ duyệt", rejected: "Đã loại" };

export type Question = {
  id: string;
  text: string;
  topic: QTopic;
  difficulty: QDifficulty;
  kind: QKind;
  source: QSource;
  status: QStatus;
  options?: string[];
  /** đáp án đúng: nội dung phương án với mcq / đúng–sai, gợi ý chấm với câu tự luận */
  answer: string;
};

type Raw = [text: string, topic: QTopic, difficulty: QDifficulty, kind: QKind, source: QSource, answer: string, options?: string[]];

const RAW: Raw[] = [
  // ---- Mật mã đối xứng ----
  ["AES-128 dùng khoá dài bao nhiêu bit?", "Mật mã đối xứng", "easy", "mcq", "bank", "128 bit", ["128 bit", "192 bit", "256 bit", "512 bit"]],
  ["Kích thước khối của AES là bao nhiêu bit?", "Mật mã đối xứng", "easy", "mcq", "bank", "128 bit", ["64 bit", "128 bit", "192 bit", "256 bit"]],
  ["AES-256 thực hiện bao nhiêu vòng biến đổi?", "Mật mã đối xứng", "medium", "mcq", "bank", "14 vòng", ["10 vòng", "12 vòng", "14 vòng", "16 vòng"]],
  ["Nhược điểm lớn nhất của chế độ ECB so với CBC là gì?", "Mật mã đối xứng", "medium", "mcq", "bank", "Để lộ mẫu lặp của bản rõ", ["Để lộ mẫu lặp của bản rõ", "Chậm hơn nhiều lần", "Bắt buộc dùng khoá 256 bit", "Không mã hoá được tệp nhị phân"]],
  ["Vector khởi tạo (IV) của chế độ CBC cần tính chất nào?", "Mật mã đối xứng", "medium", "mcq", "bank", "Ngẫu nhiên và không lặp lại", ["Ngẫu nhiên và không lặp lại", "Bí mật tuyệt đối", "Là số nguyên tố", "Dài gấp đôi khoá"]],
  ["Chế độ GCM cung cấp thêm điều gì so với CBC?", "Mật mã đối xứng", "medium", "mcq", "bank", "Toàn vẹn và xác thực dữ liệu", ["Toàn vẹn và xác thực dữ liệu", "Nén dữ liệu trước khi mã", "Khoá ngắn hơn", "Không cần IV"]],
  ["Chế độ CTR biến mật mã khối thành dạng nào?", "Mật mã đối xứng", "medium", "mcq", "ai", "Mật mã dòng", ["Mật mã dòng", "Hàm băm", "Chữ ký số", "Mã sửa lỗi"]],
  ["Thuật toán nào sau đây là mật mã đối xứng?", "Mật mã đối xứng", "easy", "mcq", "bank", "AES", ["AES", "RSA", "ECDSA", "Diffie-Hellman"]],
  ["DES bị coi là không an toàn chủ yếu vì lý do nào?", "Mật mã đối xứng", "easy", "mcq", "bank", "Khoá 56 bit quá ngắn", ["Khoá 56 bit quá ngắn", "Thuật toán bị giữ bí mật", "Không có bước hoán vị", "Dùng khoá công khai"]],
  ["AES là mật mã khối có kích thước khối 128 bit.", "Mật mã đối xứng", "easy", "truefalse", "bank", "Đúng", ["Đúng", "Sai"]],
  ["Tăng khoá AES từ 128 lên 256 bit làm thời gian mã hoá tăng gấp đôi.", "Mật mã đối xứng", "medium", "truefalse", "ai", "Sai — chỉ tăng số vòng từ 10 lên 14, chi phí tăng khoảng 40%.", ["Đúng", "Sai"]],
  ["Dùng lại cùng một cặp khoá–nonce ở chế độ CTR vẫn an toàn.", "Mật mã đối xứng", "hard", "truefalse", "bank", "Sai — hai bản rõ bị XOR với nhau, lộ nội dung.", ["Đúng", "Sai"]],
  ["Nonce khác vector khởi tạo (IV) ở điểm nào?", "Mật mã đối xứng", "medium", "short", "teacher", "Nonce chỉ cần không lặp lại với cùng khoá; IV của CBC còn phải khó đoán."],
  ["Khoá phiên (session key) là gì và vì sao nên đổi theo phiên?", "Mật mã đối xứng", "easy", "short", "teacher", "Khoá đối xứng dùng cho một phiên; đổi theo phiên để hạn chế thiệt hại khi lộ khoá."],
  ["Giải thích tấn công padding oracle và cách phòng tránh.", "Mật mã đối xứng", "hard", "essay", "exam", "Nêu được: kẻ tấn công dò padding hợp lệ ở CBC; phòng bằng mã hoá có xác thực (AES-GCM) hoặc encrypt-then-MAC."],
  ["So sánh mật mã dòng và mật mã khối; mỗi loại phù hợp tình huống nào?", "Mật mã đối xứng", "medium", "essay", "exam", "Nêu đặc điểm, độ trễ, yêu cầu đệm; dòng hợp luồng thời gian thực, khối hợp tệp/bản ghi."],
  ["Vì sao phải trao đổi khoá an toàn trước khi dùng mật mã đối xứng?", "Mật mã đối xứng", "medium", "essay", "ai", "Nêu bài toán phân phối khoá và giải pháp: Diffie-Hellman hoặc bọc khoá bằng khoá công khai."],
  ["Mã hoá đĩa toàn phần bảo vệ dữ liệu trong tình huống nào?", "Mật mã đối xứng", "easy", "short", "bank", "Khi thiết bị bị mất hoặc đánh cắp lúc đang tắt máy."],

  // ---- Hàm băm và chữ ký số ----
  ["SHA-256 cho giá trị băm dài bao nhiêu bit?", "Hàm băm và chữ ký số", "easy", "mcq", "bank", "256 bit", ["128 bit", "160 bit", "256 bit", "512 bit"]],
  ["Hàm băm nào sau đây được khuyến nghị dùng hiện nay?", "Hàm băm và chữ ký số", "easy", "mcq", "bank", "SHA-256", ["MD5", "SHA-1", "SHA-256", "CRC32"]],
  ["Vì sao MD5 không còn được dùng cho chữ ký số?", "Hàm băm và chữ ký số", "medium", "mcq", "bank", "Đã tạo được va chạm trong thực tế", ["Đã tạo được va chạm trong thực tế", "Tính toán quá chậm", "Đầu ra quá dài", "Cần khoá bí mật"]],
  ["Tính kháng tiền ảnh của hàm băm nghĩa là gì?", "Hàm băm và chữ ký số", "medium", "mcq", "bank", "Biết giá trị băm, khó tìm được dữ liệu gốc", ["Biết giá trị băm, khó tìm được dữ liệu gốc", "Khó tìm hai dữ liệu cùng băm", "Băm luôn khác nhau", "Băm đảo ngược được bằng khoá"]],
  ["HMAC dùng để làm gì?", "Hàm băm và chữ ký số", "medium", "mcq", "bank", "Xác thực toàn vẹn thông điệp bằng khoá chia sẻ", ["Xác thực toàn vẹn thông điệp bằng khoá chia sẻ", "Mã hoá thông điệp", "Sinh khoá công khai", "Nén thông điệp"]],
  ["Khi ký số, người ký dùng khoá nào?", "Hàm băm và chữ ký số", "easy", "mcq", "bank", "Khoá riêng của người ký", ["Khoá riêng của người ký", "Khoá công khai của người ký", "Khoá công khai của người nhận", "Khoá phiên"]],
  ["Chữ ký số đảm bảo tính chất nào sau đây?", "Hàm băm và chữ ký số", "medium", "mcq", "ai", "Chống chối bỏ", ["Chống chối bỏ", "Tính bí mật", "Tính sẵn sàng", "Tính ẩn danh"]],
  ["Hàm băm là một phép mã hoá hai chiều.", "Hàm băm và chữ ký số", "easy", "truefalse", "bank", "Sai — hàm băm một chiều, không khôi phục được dữ liệu gốc.", ["Đúng", "Sai"]],
  ["HMAC-SHA256 an toàn hơn cách ghép khoá rồi băm trực tiếp.", "Hàm băm và chữ ký số", "hard", "truefalse", "bank", "Đúng — cách ghép trực tiếp dính tấn công nối dài thông điệp.", ["Đúng", "Sai"]],
  ["Vai trò của salt khi lưu mật khẩu là gì?", "Hàm băm và chữ ký số", "easy", "short", "bank", "Làm hai mật khẩu giống nhau cho băm khác nhau, vô hiệu bảng tra sẵn."],
  ["Va chạm (collision) của hàm băm là gì?", "Hàm băm và chữ ký số", "easy", "short", "bank", "Hai dữ liệu khác nhau cho cùng một giá trị băm."],
  ["Tấn công ngày sinh ảnh hưởng thế nào tới độ dài băm cần chọn?", "Hàm băm và chữ ký số", "hard", "short", "exam", "Độ an toàn chống va chạm chỉ còn một nửa số bit, nên cần băm ≥ 256 bit."],
  ["Vì sao nên dùng Argon2 hoặc bcrypt thay cho SHA-256 khi lưu mật khẩu?", "Hàm băm và chữ ký số", "hard", "essay", "teacher", "Nêu: hàm chậm, có tham số chi phí và bộ nhớ, chống dò bằng GPU."],
  ["Mô tả quy trình ký và kiểm chữ ký số của một tệp.", "Hàm băm và chữ ký số", "medium", "essay", "exam", "Băm tệp, ký giá trị băm bằng khoá riêng; bên nhận băm lại và kiểm bằng khoá công khai."],
  ["Dấu thời gian (timestamp) trong chữ ký số dùng để làm gì?", "Hàm băm và chữ ký số", "medium", "short", "bank", "Chứng minh chữ ký có trước thời điểm chứng thư hết hạn hoặc bị thu hồi."],
  ["Cây Merkle được dùng để làm gì trong thực tế?", "Hàm băm và chữ ký số", "hard", "short", "ai", "Kiểm toàn vẹn nhanh một phần dữ liệu lớn: Git, blockchain, nhật ký chứng thư."],

  // ---- Khoá công khai và PKI ----
  ["Độ an toàn của RSA dựa trên bài toán khó nào?", "Khoá công khai và PKI", "easy", "mcq", "bank", "Phân tích số lớn ra thừa số nguyên tố", ["Phân tích số lớn ra thừa số nguyên tố", "Logarit rời rạc trên đường cong", "Bài toán ba lô", "Tìm va chạm hàm băm"]],
  ["Để gửi dữ liệu bí mật cho một người, ta mã hoá bằng khoá nào?", "Khoá công khai và PKI", "easy", "mcq", "bank", "Khoá công khai của người nhận", ["Khoá công khai của người nhận", "Khoá riêng của người nhận", "Khoá riêng của người gửi", "Khoá phiên của người gửi"]],
  ["Thành phần nào trong PKI phát hành chứng thư số?", "Khoá công khai và PKI", "easy", "mcq", "bank", "CA — tổ chức chứng thực", ["CA — tổ chức chứng thực", "RA — tổ chức đăng ký", "CRL", "OCSP"]],
  ["ECC có lợi thế gì so với RSA ở cùng mức an toàn?", "Khoá công khai và PKI", "medium", "mcq", "bank", "Khoá ngắn hơn, tính toán nhẹ hơn", ["Khoá ngắn hơn, tính toán nhẹ hơn", "Không cần số ngẫu nhiên", "Không cần chứng thư", "Mã hoá được dữ liệu lớn"]],
  ["Diffie-Hellman giải quyết vấn đề gì?", "Khoá công khai và PKI", "medium", "mcq", "ai", "Thoả thuận khoá chung trên kênh công khai", ["Thoả thuận khoá chung trên kênh công khai", "Ký số thông điệp", "Băm mật khẩu", "Nén dữ liệu"]],
  ["Chứng thư số có chứa khoá riêng của chủ thể.", "Khoá công khai và PKI", "easy", "truefalse", "bank", "Sai — chứng thư chỉ chứa khoá công khai và thông tin định danh đã được CA ký.", ["Đúng", "Sai"]],
  ["Khoá RSA 1024 bit vẫn đủ an toàn cho hệ thống mới.", "Khoá công khai và PKI", "medium", "truefalse", "bank", "Sai — khuyến nghị tối thiểu 2048 bit, hoặc dùng ECC 256 bit.", ["Đúng", "Sai"]],
  ["Phân biệt CRL và OCSP.", "Khoá công khai và PKI", "medium", "short", "exam", "CRL là danh sách thu hồi tải định kỳ; OCSP hỏi trạng thái từng chứng thư theo thời gian thực."],
  ["Tấn công người đứng giữa với Diffie-Hellman được ngăn bằng cách nào?", "Khoá công khai và PKI", "hard", "short", "exam", "Xác thực các bên bằng chữ ký số hoặc chứng thư trong quá trình thoả thuận khoá."],
  ["Chứng thư tự ký nên dùng trong trường hợp nào?", "Khoá công khai và PKI", "medium", "short", "teacher", "Môi trường nội bộ, thử nghiệm; không dùng cho dịch vụ công khai."],
  ["Ghim chứng thư (certificate pinning) là gì?", "Khoá công khai và PKI", "hard", "short", "ai", "Ứng dụng chỉ chấp nhận một chứng thư hoặc khoá công khai định sẵn, chặn CA giả mạo."],
  ["Giải thích tính bí mật chuyển tiếp hoàn hảo (PFS) và vì sao TLS cần nó.", "Khoá công khai và PKI", "hard", "essay", "exam", "Khoá phiên sinh tạm theo phiên; lộ khoá riêng máy chủ không giải mã được lưu lượng đã ghi."],
  ["Mô tả các bước thoả thuận khoá phiên trong bắt tay TLS 1.3.", "Khoá công khai và PKI", "hard", "essay", "teacher", "ClientHello kèm key share, máy chủ chọn nhóm, dẫn xuất khoá qua HKDF, xác thực bằng chứng thư."],

  // ---- Tường lửa, VPN và phân đoạn mạng ----
  ["Tường lửa trạng thái khác lọc gói đơn giản ở điểm nào?", "Tường lửa, VPN và phân đoạn mạng", "medium", "mcq", "bank", "Theo dõi trạng thái kết nối để quyết định", ["Theo dõi trạng thái kết nối để quyết định", "Chỉ lọc theo địa chỉ IP", "Hoạt động ở tầng vật lý", "Không cần luật"]],
  ["IPsec hoạt động chủ yếu ở tầng nào của mô hình TCP/IP?", "Tường lửa, VPN và phân đoạn mạng", "medium", "mcq", "bank", "Tầng mạng", ["Tầng ứng dụng", "Tầng giao vận", "Tầng mạng", "Tầng liên kết"]],
  ["WPA2 dùng thuật toán mã hoá nào?", "Tường lửa, VPN và phân đoạn mạng", "medium", "mcq", "bank", "AES-CCMP", ["RC4", "AES-CCMP", "DES", "3DES"]],
  ["Chính sách mặc định của tường lửa nên là gì?", "Tường lửa, VPN và phân đoạn mạng", "easy", "mcq", "bank", "Từ chối tất cả, chỉ mở cổng cần thiết", ["Từ chối tất cả, chỉ mở cổng cần thiết", "Cho phép tất cả rồi chặn dần", "Chỉ chặn cổng 80", "Tuỳ người dùng chọn"]],
  ["Vùng DMZ dùng để làm gì?", "Tường lửa, VPN và phân đoạn mạng", "easy", "mcq", "ai", "Đặt dịch vụ công khai tách khỏi mạng nội bộ", ["Đặt dịch vụ công khai tách khỏi mạng nội bộ", "Lưu bản sao dự phòng", "Chạy máy ảo thử nghiệm", "Kết nối chi nhánh"]],
  ["IDS khác IPS ở điểm nào?", "Tường lửa, VPN và phân đoạn mạng", "medium", "mcq", "bank", "IDS chỉ cảnh báo, IPS chặn được lưu lượng", ["IDS chỉ cảnh báo, IPS chặn được lưu lượng", "IDS nhanh hơn IPS", "IPS chỉ dùng cho mạng không dây", "IDS cần khoá công khai"]],
  ["NAT là một cơ chế bảo mật đầy đủ cho mạng nội bộ.", "Tường lửa, VPN và phân đoạn mạng", "easy", "truefalse", "bank", "Sai — NAT chỉ che địa chỉ, không thay thế tường lửa và kiểm soát truy cập.", ["Đúng", "Sai"]],
  ["Nên tắt các giao thức quản trị không mã hoá như Telnet trên thiết bị mạng.", "Tường lửa, VPN và phân đoạn mạng", "easy", "truefalse", "bank", "Đúng — mật khẩu đi dạng rõ, dễ bị nghe lén; dùng SSH thay thế.", ["Đúng", "Sai"]],
  ["Phân đoạn mạng giúp giảm rủi ro nào rõ nhất?", "Tường lửa, VPN và phân đoạn mạng", "medium", "short", "teacher", "Hạn chế di chuyển ngang của kẻ tấn công sau khi chiếm được một máy."],
  ["Phân biệt VPN site-to-site và VPN client-to-site.", "Tường lửa, VPN và phân đoạn mạng", "medium", "short", "bank", "Site-to-site nối hai mạng qua gateway; client-to-site cho một máy người dùng vào mạng nội bộ."],
  ["802.1X dùng để làm gì trong mạng doanh nghiệp?", "Tường lửa, VPN và phân đoạn mạng", "hard", "short", "ai", "Xác thực thiết bị/người dùng ngay tại cổng mạng trước khi cấp quyền truy cập."],
  ["VLAN hopping là gì?", "Tường lửa, VPN và phân đoạn mạng", "hard", "short", "exam", "Kỹ thuật vượt ranh giới VLAN bằng gắn thẻ kép hoặc lạm dụng cổng trunk."],
  ["Mô tả cách tấn công DDoS khuếch đại qua DNS và biện pháp giảm nhẹ.", "Tường lửa, VPN và phân đoạn mạng", "hard", "essay", "exam", "Giả mạo IP nguồn, truy vấn nhỏ – trả lời lớn; giảm nhẹ bằng chặn giả mạo nguồn, giới hạn tốc độ, dịch vụ chống DDoS."],
  ["So sánh mô hình an ninh vành đai với mô hình không tin mặc định (zero trust).", "Tường lửa, VPN và phân đoạn mạng", "hard", "essay", "teacher", "Vành đai tin mạng trong; zero trust xác thực và cấp quyền theo từng yêu cầu, giả định mạng đã bị xâm nhập."],
  ["Nêu ba dấu hiệu nhận biết hoạt động quét cổng trong nhật ký mạng.", "Tường lửa, VPN và phân đoạn mạng", "medium", "essay", "ai", "Nhiều kết nối SYN tới nhiều cổng từ một nguồn, tỉ lệ RST cao, mẫu cổng tuần tự trong thời gian ngắn."],

  // ---- An ninh ứng dụng web ----
  ["Cách phòng SQL injection hiệu quả nhất là gì?", "An ninh ứng dụng web", "easy", "mcq", "bank", "Dùng truy vấn tham số hoá", ["Dùng truy vấn tham số hoá", "Lọc dấu nháy đơn", "Mã hoá cơ sở dữ liệu", "Đổi cổng cơ sở dữ liệu"]],
  ["Thuộc tính cookie HttpOnly ngăn điều gì?", "An ninh ứng dụng web", "medium", "mcq", "bank", "JavaScript đọc cookie", ["JavaScript đọc cookie", "Cookie gửi qua HTTP", "Cookie bị hết hạn", "Trình duyệt lưu cookie"]],
  ["Thẻ tiêu đề nào ép trình duyệt luôn dùng HTTPS?", "An ninh ứng dụng web", "medium", "mcq", "bank", "Strict-Transport-Security", ["Strict-Transport-Security", "X-Frame-Options", "Referrer-Policy", "Accept-Encoding"]],
  ["Mã hoá HTTPS ngăn được tấn công SQL injection.", "An ninh ứng dụng web", "easy", "truefalse", "bank", "Sai — HTTPS chỉ bảo vệ đường truyền, lỗ hổng nằm ở xử lý dữ liệu phía máy chủ.", ["Đúng", "Sai"]],
  ["Giới hạn tốc độ đăng nhập chủ yếu chống tấn công nào?", "An ninh ứng dụng web", "easy", "mcq", "ai", "Dò mật khẩu bằng vét cạn", ["Dò mật khẩu bằng vét cạn", "XSS phản chiếu", "Tràn bộ đệm", "Nghe lén gói tin"]],
  ["Mã hoá đầu ra (output encoding) chống lỗ hổng nào?", "An ninh ứng dụng web", "medium", "mcq", "bank", "XSS", ["XSS", "SQL injection", "CSRF", "SSRF"]],
  ["Phân biệt XSS lưu trữ và XSS phản chiếu.", "An ninh ứng dụng web", "medium", "short", "bank", "Lưu trữ: mã độc nằm trong dữ liệu máy chủ, tấn công mọi người xem; phản chiếu: mã nằm trong URL, cần dụ nạn nhân bấm."],
  ["CSRF được phòng bằng những biện pháp nào?", "An ninh ứng dụng web", "medium", "short", "bank", "Token chống CSRF theo phiên, cookie SameSite, kiểm tra Origin/Referer."],
  ["Lỗ hổng IDOR là gì? Cho một ví dụ.", "An ninh ứng dụng web", "medium", "short", "exam", "Truy cập đối tượng của người khác bằng cách đổi định danh trên URL, ví dụ /hoa-don/1042."],
  ["SSRF là gì và vì sao nguy hiểm trong môi trường đám mây?", "An ninh ứng dụng web", "hard", "short", "ai", "Máy chủ bị ép gọi tới địa chỉ nội bộ; trên đám mây có thể lấy thông tin xác thực từ dịch vụ metadata."],
  ["Chính sách CSP giúp giảm rủi ro gì?", "An ninh ứng dụng web", "hard", "short", "ai", "Giới hạn nguồn script/style được chạy, giảm tác hại của XSS."],
  ["Tải tệp lên không kiểm tra có thể dẫn tới rủi ro nào?", "An ninh ứng dụng web", "medium", "short", "teacher", "Thực thi mã trên máy chủ, lưu mã độc phát tán cho người dùng khác, chiếm dung lượng."],
  ["Lỗi gán hàng loạt (mass assignment) xảy ra khi nào?", "An ninh ứng dụng web", "hard", "short", "exam", "Khi khung ứng dụng gán thẳng dữ liệu người dùng vào thuộc tính nhạy cảm như vai trò."],
  ["Khi người dùng đăng xuất, hệ thống cần làm gì với phiên?", "An ninh ứng dụng web", "easy", "short", "bank", "Huỷ phiên phía máy chủ và xoá cookie, không chỉ xoá ở trình duyệt."],
  ["Giải thích mục Broken Access Control trong OWASP Top 10 và cách kiểm thử.", "An ninh ứng dụng web", "hard", "essay", "exam", "Thiếu kiểm tra quyền ở phía máy chủ; kiểm bằng cách đổi định danh, đổi vai trò, gọi thẳng API."],
  ["Vì sao không bao giờ lưu mật khẩu người dùng ở dạng rõ?", "An ninh ứng dụng web", "easy", "essay", "teacher", "Rò rỉ cơ sở dữ liệu là lộ toàn bộ tài khoản, kể cả ở dịch vụ khác do dùng lại mật khẩu."],

  // ---- Xác thực và kiểm soát truy cập ----
  ["Ba yếu tố xác thực cơ bản là gì?", "Xác thực và kiểm soát truy cập", "easy", "mcq", "bank", "Điều bạn biết, điều bạn có, điều bạn là", ["Điều bạn biết, điều bạn có, điều bạn là", "Mật khẩu, PIN, mã OTP", "Người dùng, nhóm, vai trò", "Thiết bị, mạng, vị trí"]],
  ["OTP qua SMS an toàn ngang với khoá phần cứng FIDO2.", "Xác thực và kiểm soát truy cập", "medium", "truefalse", "bank", "Sai — SMS bị tráo SIM và lừa đảo chuyển tiếp; FIDO2 gắn với tên miền nên chống lừa đảo.", ["Đúng", "Sai"]],
  ["RBAC khác ABAC ở điểm nào?", "Xác thực và kiểm soát truy cập", "medium", "mcq", "bank", "RBAC cấp quyền theo vai trò, ABAC theo thuộc tính ngữ cảnh", ["RBAC cấp quyền theo vai trò, ABAC theo thuộc tính ngữ cảnh", "RBAC chỉ dùng cho tệp", "ABAC không cần xác thực", "Hai mô hình giống nhau"]],
  ["Nguyên tắc đặc quyền tối thiểu nghĩa là gì?", "Xác thực và kiểm soát truy cập", "easy", "short", "bank", "Mỗi tài khoản chỉ có đúng quyền cần cho công việc, trong đúng khoảng thời gian cần."],
  ["JWT đặt alg=none gây rủi ro gì?", "Xác thực và kiểm soát truy cập", "hard", "short", "ai", "Máy chủ chấp nhận token không chữ ký, kẻ tấn công tự tạo token tuỳ ý."],
  ["Tấn công cố định phiên (session fixation) hoạt động thế nào?", "Xác thực và kiểm soát truy cập", "hard", "short", "exam", "Ép nạn nhân dùng mã phiên do kẻ tấn công biết; phòng bằng cấp mã phiên mới sau khi đăng nhập."],
  ["Golden SAML là kỹ thuật gì?", "Xác thực và kiểm soát truy cập", "hard", "short", "ai", "Đánh cắp khoá ký của nhà cung cấp danh tính để tự phát hành thẻ truy cập hợp lệ."],
  ["Khoá API bị lộ trong mã nguồn nên xử lý thế nào?", "Xác thực và kiểm soát truy cập", "medium", "short", "teacher", "Thu hồi và cấp lại khoá, rà nhật ký sử dụng, chuyển khoá sang kho bí mật và quét mã định kỳ."],
  ["Chính sách mật khẩu nào được khuyến nghị hiện nay?", "Xác thực và kiểm soát truy cập", "medium", "mcq", "bank", "Ưu tiên độ dài và kiểm tra danh sách mật khẩu đã rò rỉ", ["Ưu tiên độ dài và kiểm tra danh sách mật khẩu đã rò rỉ", "Bắt đổi mật khẩu mỗi 30 ngày", "Bắt buộc ký tự đặc biệt ở đầu", "Giới hạn tối đa 8 ký tự"]],
  ["Trình bày cách đăng nhập một lần (SSO) với SAML hoặc OIDC và rủi ro đi kèm.", "Xác thực và kiểm soát truy cập", "hard", "essay", "exam", "Nêu luồng chuyển hướng, bên tin cậy và nhà cung cấp danh tính; rủi ro tập trung một điểm hỏng."],

  // ---- Mối đe doạ và ứng phó sự cố ----
  ["Tấn công chuỗi cung ứng phần mềm là gì?", "Mối đe doạ và ứng phó sự cố", "medium", "mcq", "bank", "Xâm nhập nhà cung cấp để phát tán mã độc qua bản cập nhật hợp lệ", ["Xâm nhập nhà cung cấp để phát tán mã độc qua bản cập nhật hợp lệ", "Tấn công vào kho hàng vật lý", "Gửi thư rác hàng loạt", "Dò mật khẩu quản trị"]],
  ["Biện pháp quan trọng nhất để phục hồi sau ransomware là gì?", "Mối đe doạ và ứng phó sự cố", "easy", "mcq", "bank", "Bản sao lưu ngoại tuyến đã kiểm thử phục hồi", ["Bản sao lưu ngoại tuyến đã kiểm thử phục hồi", "Trả tiền chuộc", "Cài lại phần mềm diệt virus", "Đổi tên tệp bị mã hoá"]],
  ["Chữ ký số hợp lệ bảo đảm bản cập nhật là an toàn.", "Mối đe doạ và ứng phó sự cố", "medium", "truefalse", "bank", "Sai — nếu hệ thống dựng của nhà cung cấp bị xâm nhập, mã độc vẫn được ký hợp lệ (SolarWinds 2020).", ["Đúng", "Sai"]],
  ["Nêu bốn giai đoạn chính của quy trình ứng phó sự cố.", "Mối đe doạ và ứng phó sự cố", "medium", "short", "bank", "Chuẩn bị; phát hiện và phân tích; ngăn chặn – loại bỏ – phục hồi; rút kinh nghiệm."],
  ["IOC (chỉ dấu xâm nhập) là gì? Cho hai ví dụ.", "Mối đe doạ và ứng phó sự cố", "medium", "short", "ai", "Dấu hiệu kỹ thuật của một cuộc tấn công: băm tệp độc hại, tên miền điều khiển, IP, khoá registry."],
  ["Di chuyển ngang (lateral movement) nghĩa là gì?", "Mối đe doạ và ứng phó sự cố", "medium", "short", "ai", "Từ một máy đã chiếm được, kẻ tấn công mở rộng sang máy khác trong cùng mạng."],
  ["Nhật ký cần giữ những trường nào để phục vụ điều tra?", "Mối đe doạ và ứng phó sự cố", "hard", "short", "teacher", "Thời gian có múi giờ, định danh người dùng/dịch vụ, IP nguồn, hành động, kết quả và mã tương quan."],
  ["SBOM giúp gì khi xảy ra tấn công chuỗi cung ứng?", "Mối đe doạ và ứng phó sự cố", "hard", "short", "ai", "Biết nhanh hệ thống nào chứa thành phần bị ảnh hưởng để vá và cô lập."],
  ["Nêu năm dấu hiệu nhận biết thư lừa đảo nhắm mục tiêu.", "Mối đe doạ và ứng phó sự cố", "easy", "essay", "teacher", "Tên miền gần giống, thúc ép thời gian, liên kết lệch nội dung hiển thị, tệp đính kèm lạ, yêu cầu thông tin xác thực."],
  ["Trình bày quy trình vá lỗi khẩn cấp cho một lỗ hổng đang bị khai thác.", "Mối đe doạ và ứng phó sự cố", "hard", "essay", "exam", "Đánh giá phạm vi, giảm nhẹ tạm thời, kiểm thử bản vá, triển khai theo đợt, xác minh và thông báo."],
  ["Phân tích vì sao vụ SolarWinds được xem là bước ngoặt về an ninh chuỗi cung ứng.", "Mối đe doạ và ứng phó sự cố", "hard", "essay", "ai", "Niềm tin vào chữ ký số và nhà cung cấp bị lợi dụng; dẫn tới yêu cầu build tái lập, SBOM, giám sát hành vi."],
  ["Thế nào là mối đe doạ nội bộ (insider threat) và cách giảm thiểu?", "Mối đe doạ và ứng phó sự cố", "medium", "essay", "ai", "Người có quyền hợp lệ gây hại; giảm bằng đặc quyền tối thiểu, tách nhiệm vụ, giám sát truy cập dữ liệu nhạy cảm."],
];

export const QUESTIONS: Question[] = RAW.map(([text, topic, difficulty, kind, source, answer, options], i) => ({
  id: `q-${String(i + 1).padStart(3, "0")}`,
  text,
  topic,
  difficulty,
  kind,
  source,
  status: source === "ai" ? "pending" : "approved",
  options,
  answer,
}));

/** 5 câu do luồng `Tạo câu hỏi` sinh ra (thêm vào danh sách ở trạng thái chờ duyệt). */
export const GENERATED_QUESTIONS: Question[] = [
  {
    id: "q-gen-1",
    text: "Vì sao chế độ CBC cần vector khởi tạo ngẫu nhiên cho mỗi bản tin?",
    topic: "Mật mã đối xứng",
    difficulty: "medium",
    kind: "short",
    source: "ai",
    status: "pending",
    answer: "Để hai bản rõ giống nhau cho ra bản mã khác nhau, không lộ việc lặp nội dung.",
  },
  {
    id: "q-gen-2",
    text: "Chế độ nào sau đây vừa mã hoá vừa xác thực dữ liệu?",
    topic: "Mật mã đối xứng",
    difficulty: "medium",
    kind: "mcq",
    source: "ai",
    status: "pending",
    options: ["AES-GCM", "AES-ECB", "AES-CBC", "AES-CTR"],
    answer: "AES-GCM",
  },
  {
    id: "q-gen-3",
    text: "Khoá AES 256 bit an toàn hơn 128 bit trước máy tính lượng tử ở mức nào?",
    topic: "Mật mã đối xứng",
    difficulty: "hard",
    kind: "short",
    source: "ai",
    status: "pending",
    answer: "Thuật toán Grover giảm độ an toàn còn một nửa số bit, nên 256 bit vẫn còn 128 bit hiệu dụng.",
  },
  {
    id: "q-gen-4",
    text: "Mã hoá đối xứng và mã hoá bất đối xứng thường được kết hợp thế nào trong thực tế?",
    topic: "Mật mã đối xứng",
    difficulty: "medium",
    kind: "essay",
    source: "ai",
    status: "pending",
    answer: "Bất đối xứng để thoả thuận khoá phiên, đối xứng để mã hoá dữ liệu vì nhanh hơn nhiều.",
  },
  {
    id: "q-gen-5",
    text: "Chế độ ECB không nên dùng cho ảnh bitmap vì lý do nào?",
    topic: "Mật mã đối xứng",
    difficulty: "easy",
    kind: "truefalse",
    source: "ai",
    status: "pending",
    options: ["Đúng", "Sai"],
    answer: "Đúng — các khối giống nhau cho bản mã giống nhau nên vẫn nhìn ra hình.",
  },
];
