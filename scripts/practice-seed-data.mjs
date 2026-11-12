// Dữ liệu mẫu của Luyện đề (SRS FEAT-practice 4.9 / US-P9-12): 80 câu APPROVED (8 chủ đề × 10) + 2 tự luận, hợp với 20 câu của PE ở lớp 761987 = đủ ngân hàng,
// + 20 câu PENDING (2 chủ đề riêng) cho hàng chờ duyệt. Câu sinh từ bảng (thuật ngữ → định nghĩa): đáp án sai lấy từ định nghĩa của các thuật ngữ KHÁC
// cùng chủ đề (chắc chắn sai). Không có số ngẫu nhiên: `title` là khoá tự nhiên (idempotent) trong lớp.

const T = (topic, pairs) => ({ topic, pairs });

export const APPROVED_TOPICS = [
  T("Mật mã khối", [
    ["AES", "Mật mã khối đối xứng dùng một khoá cho cả mã hoá và giải mã, khối 128 bit"],
    ["DES", "Mật mã khối cũ có khoá 56 bit, nay không còn được xem là an toàn"],
    ["3DES", "Áp dụng DES ba lần liên tiếp để tăng độ dài khoá hiệu dụng"],
    ["ECB", "Chế độ mã hoá từng khối độc lập nên để lộ mẫu lặp của dữ liệu"],
    ["CBC", "Chế độ XOR mỗi khối bản rõ với khối bản mã trước đó, cần vector khởi tạo"],
    ["CTR", "Chế độ biến mật mã khối thành mật mã dòng bằng một bộ đếm"],
    ["GCM", "Chế độ mã hoá có xác thực, bảo vệ cả bí mật và toàn vẹn của dữ liệu"],
    ["IV", "Giá trị khởi tạo giúp cùng một bản rõ cho ra bản mã khác nhau"],
    ["Khoá phiên", "Khoá đối xứng sinh tạm thời dùng trong một phiên liên lạc"],
    ["Padding", "Phần đệm thêm vào để dữ liệu vừa đủ số khối của mật mã khối"],
  ]),
  T("Mật mã công khai", [
    ["RSA", "Hệ mật dựa trên độ khó của việc phân tích số nguyên lớn ra thừa số nguyên tố"],
    ["ECC", "Mật mã đường cong elliptic, cùng mức an toàn với khoá ngắn hơn RSA"],
    ["Diffie-Hellman", "Giao thức thoả thuận khoá chung qua kênh công khai"],
    ["ElGamal", "Hệ mật khoá công khai dựa trên bài toán logarit rời rạc"],
    ["Khoá công khai", "Khoá có thể phát cho mọi người, dùng để mã hoá hoặc kiểm chữ ký"],
    ["Khoá bí mật", "Khoá chỉ chủ sở hữu giữ, dùng để giải mã hoặc ký"],
    ["Chữ ký số", "Giá trị tính bằng khoá bí mật để chứng minh nguồn gốc và toàn vẹn thông điệp"],
    ["Chứng thư số", "Văn bản do CA ký xác nhận một khoá công khai thuộc về một chủ thể"],
    ["PKI", "Hạ tầng gồm CA, chứng thư và quy trình quản lý khoá công khai"],
    ["Forward secrecy", "Tính chất lộ khoá dài hạn không làm lộ các phiên liên lạc đã qua"],
  ]),
  T("Hàm băm và chữ ký", [
    ["SHA-256", "Hàm băm cho đầu ra 256 bit, thuộc họ SHA-2"],
    ["MD5", "Hàm băm 128 bit đã có va chạm thực tế, không dùng cho chữ ký"],
    ["Kháng tiền ảnh", "Tính chất khó tìm đầu vào khi chỉ biết giá trị băm"],
    ["Kháng va chạm", "Tính chất khó tìm hai đầu vào khác nhau có cùng giá trị băm"],
    ["Hiệu ứng thác đổ", "Đổi một bit đầu vào làm thay đổi gần nửa số bit của giá trị băm"],
    ["HMAC", "Mã xác thực thông điệp xây dựng từ hàm băm và một khoá bí mật"],
    ["Salt", "Giá trị ngẫu nhiên trộn vào mật khẩu trước khi băm để chống bảng tra"],
    ["bcrypt", "Hàm băm mật khẩu cố ý chậm, có hệ số chi phí điều chỉnh được"],
    ["Cây Merkle", "Cấu trúc băm dạng cây cho phép kiểm một phần tử mà không cần toàn bộ dữ liệu"],
    ["Checksum", "Giá trị kiểm tra phát hiện lỗi truyền, không nhằm chống kẻ tấn công"],
  ]),
  T("Xác thực và phân quyền", [
    ["MFA", "Xác thực kết hợp từ hai loại yếu tố khác nhau trở lên"],
    ["TOTP", "Mã dùng một lần sinh theo thời gian từ khoá chia sẻ và đồng hồ"],
    ["OAuth 2.0", "Khung uỷ quyền cho ứng dụng truy cập tài nguyên mà không chia sẻ mật khẩu"],
    ["OpenID Connect", "Lớp danh tính xây trên OAuth 2.0, trả về thông tin người dùng"],
    ["JWT", "Token tự chứa ba phần, có chữ ký để kiểm toàn vẹn"],
    ["RBAC", "Phân quyền theo vai trò thay vì gán quyền trực tiếp cho từng người"],
    ["Nguyên tắc đặc quyền tối thiểu", "Chỉ cấp đúng quyền cần thiết cho công việc"],
    ["SSO", "Đăng nhập một lần để dùng nhiều ứng dụng"],
    ["Session fixation", "Tấn công ép nạn nhân dùng mã phiên do kẻ tấn công chọn"],
    ["Brute force", "Thử lần lượt mọi khả năng mật khẩu cho tới khi đúng"],
  ]),
  T("An toàn ứng dụng web", [
    ["SQL injection", "Chèn câu lệnh SQL qua dữ liệu nhập để thay đổi truy vấn của ứng dụng"],
    ["XSS", "Chèn mã script vào trang để chạy trong trình duyệt của người khác"],
    ["CSRF", "Lừa trình duyệt đã đăng nhập gửi yêu cầu ngoài ý muốn tới một website"],
    ["Prepared statement", "Truy vấn tham số hoá tách mã SQL khỏi dữ liệu người dùng"],
    ["CSP", "Chính sách nội dung giới hạn nguồn script và tài nguyên trang được tải"],
    ["HSTS", "Header buộc trình duyệt chỉ truy cập website qua HTTPS"],
    ["CORS", "Cơ chế cho phép hoặc chặn trang gọi tài nguyên từ nguồn gốc khác"],
    ["Cookie HttpOnly", "Cookie mà script phía trình duyệt không đọc được"],
    ["Path traversal", "Dùng chuỗi ../ để truy cập tệp ngoài thư mục cho phép"],
    ["SSRF", "Khiến máy chủ gửi yêu cầu tới địa chỉ nội bộ do kẻ tấn công chọn"],
  ]),
  T("Mạng và tường lửa", [
    ["Tường lửa lọc gói", "Kiểm từng gói theo địa chỉ, cổng và giao thức"],
    ["Tường lửa trạng thái", "Theo dõi trạng thái kết nối để cho phép gói thuộc phiên hợp lệ"],
    ["DMZ", "Vùng mạng trung gian đặt các dịch vụ công khai tách khỏi mạng nội bộ"],
    ["VLAN", "Mạng cục bộ ảo tách miền quảng bá bằng thẻ trên switch"],
    ["IDS", "Hệ thống phát hiện xâm nhập chỉ cảnh báo, không chặn"],
    ["IPS", "Hệ thống ngăn chặn xâm nhập có thể chặn lưu lượng độc hại"],
    ["NAT", "Dịch địa chỉ riêng sang địa chỉ công khai khi ra Internet"],
    ["VPN", "Đường hầm mã hoá nối hai mạng hoặc người dùng với mạng qua Internet"],
    ["Zero Trust", "Mô hình không mặc định tin cậy bất kỳ ai, luôn kiểm tra từng yêu cầu"],
    ["Port scanning", "Dò các cổng mở trên máy đích để tìm dịch vụ đang chạy"],
  ]),
  T("Tấn công và phòng thủ", [
    ["Phishing", "Giả mạo nguồn tin cậy để lừa người dùng đưa thông tin nhạy cảm"],
    ["Ransomware", "Mã độc mã hoá dữ liệu rồi đòi tiền chuộc để trả khoá"],
    ["DDoS", "Dùng nhiều máy cùng gửi lưu lượng khiến dịch vụ quá tải"],
    ["Man-in-the-middle", "Kẻ tấn công chen giữa hai bên để nghe hoặc sửa thông điệp"],
    ["Zero-day", "Lỗ hổng chưa được nhà cung cấp biết hoặc chưa có bản vá"],
    ["Social engineering", "Khai thác tâm lý con người thay vì lỗi kỹ thuật"],
    ["Honeypot", "Hệ thống mồi nhử để quan sát hành vi của kẻ tấn công"],
    ["Patch management", "Quy trình theo dõi, thử và cài bản vá bảo mật"],
    ["Penetration test", "Thử tấn công có phép để tìm điểm yếu trước kẻ xấu"],
    ["Defense in depth", "Nhiều lớp bảo vệ độc lập để một lớp thất bại không làm mất tất cả"],
  ]),
  T("Quản trị rủi ro", [
    ["CIA", "Ba mục tiêu bảo mật: bí mật, toàn vẹn, sẵn sàng"],
    ["Rủi ro", "Khả năng một mối đe doạ khai thác điểm yếu gây thiệt hại"],
    ["Tài sản", "Thứ có giá trị với tổ chức cần được bảo vệ"],
    ["Mối đe doạ", "Nguyên nhân tiềm tàng có thể gây hại cho tài sản"],
    ["Điểm yếu", "Khiếm khuyết cho phép mối đe doạ gây tác hại"],
    ["Biện pháp kiểm soát", "Cách giảm khả năng hoặc mức thiệt hại của rủi ro"],
    ["Chấp nhận rủi ro", "Quyết định giữ nguyên rủi ro vì chi phí xử lý cao hơn thiệt hại"],
    ["BCP", "Kế hoạch duy trì hoạt động thiết yếu khi sự cố xảy ra"],
    ["RTO", "Thời gian tối đa chấp nhận được để khôi phục dịch vụ"],
    ["Audit log", "Bản ghi các hành động để truy vết và điều tra sau này"],
  ]),
];

/** Hai chủ đề dành cho hàng chờ duyệt (PENDING). */
export const PENDING_TOPICS = [
  T("Mạng không dây", [
    ["WPA3", "Chuẩn bảo mật Wi-Fi dùng SAE thay cho khoá chia sẻ trước dễ bị dò"],
    ["WPA2", "Chuẩn bảo mật Wi-Fi dùng AES-CCMP phổ biến trước WPA3"],
    ["WEP", "Chuẩn bảo mật Wi-Fi cũ dùng RC4, đã bị phá hoàn toàn"],
    ["SSID", "Tên mạng không dây hiển thị cho người dùng"],
    ["Evil twin", "Điểm truy cập giả mạo tên mạng thật để thu thông tin"],
    ["Deauthentication", "Tấn công gửi khung ngắt kết nối khiến thiết bị rớt mạng"],
    ["WPS", "Tính năng kết nối nhanh bằng mã PIN từng gây lỗ hổng dò PIN"],
    ["802.1X", "Chuẩn xác thực theo cổng, thường đi cùng RADIUS"],
    ["RADIUS", "Dịch vụ xác thực tập trung cho truy cập mạng"],
    ["MAC filtering", "Lọc theo địa chỉ MAC, dễ bị giả mạo nên chỉ là lớp phụ"],
  ]),
  T("Quản lý khoá", [
    ["KMS", "Dịch vụ tập trung sinh, lưu và kiểm soát việc dùng khoá mã hoá"],
    ["HSM", "Thiết bị phần cứng chống can thiệp để giữ và dùng khoá"],
    ["Key rotation", "Định kỳ thay khoá để giới hạn dữ liệu bị lộ nếu một khoá lộ"],
    ["Key escrow", "Gửi bản sao khoá cho bên thứ ba tin cậy để khôi phục khi cần"],
    ["Envelope encryption", "Dùng khoá dữ liệu mã hoá nội dung rồi mã hoá khoá đó bằng khoá chính"],
    ["Key derivation", "Sinh khoá từ mật khẩu hoặc khoá gốc bằng hàm chuyên dụng"],
    ["Entropy", "Độ ngẫu nhiên không đoán trước của nguồn sinh khoá"],
    ["Split knowledge", "Chia hiểu biết về khoá cho nhiều người để không ai tự dùng được"],
    ["Khoá hết hạn", "Khoá không còn dùng để tạo dữ liệu mới nhưng có thể còn dùng để đọc dữ liệu cũ"],
    ["Thu hồi khoá", "Vô hiệu một khoá trước hạn khi nghi bị lộ"],
  ]),
];

const rot = (arr, i, k) => arr[(i + k) % arr.length];

/** Từ một chủ đề dựng 10 câu: 6 chọn một, 2 đúng–sai, 1 trả lời ngắn, 1 chọn nhiều. */
export function buildQuestions(group) {
  const { topic, pairs } = group;
  const defs = pairs.map((p) => p[1]);
  const out = [];
  pairs.forEach(([term, def], i) => {
    const title = `${topic} ${String(i + 1).padStart(2, "0")}`;
    const difficulty = ["EASY", "EASY", "MEDIUM", "MEDIUM", "MEDIUM", "HARD"][i % 6];
    const wrong = [rot(defs, i, 3), rot(defs, i, 5), rot(defs, i, 7)];
    if (i === 3 || i === 7) {
      const truth = i === 3;
      out.push({ topic, title, difficulty, type: "TRUE_FALSE", stem: `${term}: ${truth ? def : rot(defs, i, 4)}.`, value: truth, explanation: truth ? `Đúng: ${term} — ${def}.` : `Sai. ${term} — ${def}.` });
    } else if (i === 9) {
      out.push({ topic, title, difficulty, type: "SHORT", stem: `Thuật ngữ nào chỉ: ${def.charAt(0).toLowerCase() + def.slice(1)}?`, accepted: [term], explanation: `Đó là ${term}.` });
    } else if (i === 5) {
      const right = [def, rot(defs, i, 1)];
      const opts = [...right, wrong[0], wrong[1]];
      out.push({ topic, title, difficulty: "HARD", type: "MCQ_MULTI", stem: `Chọn các mô tả đúng về ${term} hoặc ${rot(pairs, i, 1)[0]}.`, options: opts, correct: [0, 1], explanation: `${term}: ${def}. ${rot(pairs, i, 1)[0]}: ${rot(defs, i, 1)}.` });
    } else {
      const opts = [def, ...wrong];
      const shift = i % 4; // đáp án đúng không luôn ở vị trí đầu
      const rotated = opts.map((_, k) => opts[(k + 4 - shift) % 4]);
      out.push({ topic, title, difficulty, type: "MCQ_SINGLE", stem: `Mô tả nào đúng về ${term}?`, options: rotated, correct: [rotated.indexOf(def)], explanation: `${term}: ${def}.` });
    }
  });
  return out;
}

export const APPROVED = APPROVED_TOPICS.flatMap(buildQuestions); // 80 câu (8 chủ đề × 10)
export const PENDING = PENDING_TOPICS.flatMap(buildQuestions); // 20 câu

/** 2 câu tự luận có rubric (APPROVED) cho chủ đề luyện tự luận. */
export const ESSAYS = [
  { topic: "An toàn ứng dụng web", title: "Tự luận — SQL injection", difficulty: "MEDIUM", type: "ESSAY", stem: "Trình bày cách một cuộc tấn công SQL injection diễn ra và hai biện pháp phòng chống.", modelAnswer: "Kẻ tấn công chèn mã SQL vào dữ liệu nhập; nếu ứng dụng ghép chuỗi vào truy vấn, mã đó được thực thi. Phòng chống: truy vấn tham số hoá (prepared statement) và kiểm tra / giới hạn dữ liệu nhập; thêm quyền tối thiểu cho tài khoản CSDL.", rubric: "Mô tả cơ chế chèn mã (4đ); nêu prepared statement (3đ); nêu biện pháp thứ hai hợp lý (3đ)." },
  { topic: "Mật mã khối", title: "Tự luận — Chế độ ECB và CBC", difficulty: "HARD", type: "ESSAY", stem: "So sánh chế độ ECB và CBC của mật mã khối và cho biết vì sao ECB không nên dùng cho dữ liệu có mẫu lặp.", modelAnswer: "ECB mã hoá từng khối độc lập nên các khối bản rõ giống nhau cho bản mã giống nhau, lộ mẫu lặp. CBC XOR mỗi khối với bản mã trước và dùng IV nên cùng bản rõ cho bản mã khác nhau.", rubric: "Mô tả ECB (3đ); mô tả CBC và IV (3đ); giải thích lộ mẫu (4đ)." },
];
