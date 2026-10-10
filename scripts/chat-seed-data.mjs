// Dữ liệu mẫu của US-P3-08 / US-P8-03 (AC17): câu hỏi chat riêng, thread lớp, PDF tối giản, sự kiện lịch.
// KHÔNG chứa tên người, MSSV, email, số điện thoại: canary quét seed (`docs/sprints/6/qc/scripts/scan-pii-leak.sh`) phải sạch.

/** Chủ đề A (an ninh mạng — `Mordern_Network_Security_Threats.pdf`). */
export const TOPIC_A = [
  "Phishing là gì và làm sao nhận ra một email lừa đảo?",
  "Ransomware hoạt động như thế nào?",
  "Tấn công DDoS khác tấn công DoS ở điểm nào?",
  "Man-in-the-middle xảy ra trong tình huống nào?",
  "SQL injection được ngăn chặn bằng cách nào?",
  "Zero-day exploit nghĩa là gì?",
  "Botnet được dùng để làm gì?",
  "Social engineering gồm những kỹ thuật phổ biến nào?",
  "Mã độc tống tiền khác virus thường ở điểm nào?",
  "Firewall bảo vệ mạng nội bộ ra sao?",
  "Hệ thống phát hiện xâm nhập IDS hoạt động thế nào?",
  "Vì sao cần cập nhật bản vá bảo mật thường xuyên?",
  "Mối đe dọa từ bên trong tổ chức gồm những gì?",
  "Spyware thu thập dữ liệu người dùng bằng cách nào?",
  "Cross-site scripting là gì?",
  "Brute force mật khẩu được giảm thiểu ra sao?",
  "Tấn công chuỗi cung ứng phần mềm là gì?",
  "VPN giúp giảm rủi ro nào khi dùng Wi-Fi công cộng?",
  "Honeypot dùng để làm gì trong an ninh mạng?",
  "Đánh giá rủi ro an ninh gồm những bước nào?",
];

/** Chủ đề B (quy chế học vụ — `Quyche.pdf`). */
export const TOPIC_B = [
  "Điều kiện để được dự thi cuối kỳ là gì?",
  "Sinh viên bị cảnh báo học vụ khi nào?",
  "Quy định về vắng mặt tối đa trong một học phần?",
  "Điểm chuyên cần được tính như thế nào?",
  "Thời hạn phúc khảo điểm thi là bao lâu?",
  "Khi nào sinh viên bị buộc thôi học?",
  "Quy chế thi quy định gì về mang tài liệu vào phòng thi?",
  "Học lại và cải thiện điểm được quy định ra sao?",
  "Thang điểm chữ quy đổi sang thang 4 như thế nào?",
  "Sinh viên được bảo lưu kết quả học tập trong trường hợp nào?",
  "Hình thức xử lý khi vi phạm quy chế thi?",
  "Điều kiện xét tốt nghiệp gồm những gì?",
  "Đăng ký học phần trễ hạn thì xử lý ra sao?",
  "Quy định về nghỉ học có phép?",
  "Điểm trung bình tích luỹ được tính thế nào?",
];

/** Các câu có trong tài liệu nhưng không thuộc A, B. */
export const TOPIC_C = [
  "Dự báo bằng trung bình trượt là gì?",
  "Hồi quy tuyến tính đơn dùng để làm gì?",
  "Sai số dự báo MAD được tính thế nào?",
  "Làm trơn mũ khác trung bình trượt ra sao?",
  "Thành phần xu hướng và mùa vụ của chuỗi thời gian là gì?",
  "Khi nào nên dùng dự báo định tính?",
  "Hệ số xác định R bình phương cho biết điều gì?",
  "Độ chệch của dự báo được đo bằng chỉ số nào?",
  "Chuỗi thời gian có tính mùa vụ xử lý ra sao?",
  "Chọn hằng số làm trơn alpha như thế nào?",
];

/** 10 câu ngoài tài liệu — trợ lý phải trả lời "chưa tìm thấy trong tài liệu của lớp". */
export const OUTSIDE = [
  "Hôm nay thời tiết ở Hà Nội thế nào?",
  "Công thức nấu phở bò gồm những gì?",
  "Đội tuyển bóng đá quốc gia đá trận nào tới?",
  "Giá vàng hôm nay là bao nhiêu?",
  "Gợi ý bộ phim hay để xem cuối tuần?",
  "Cách trồng cây xương rồng trong chậu nhỏ?",
  "Nên mua điện thoại nào trong tầm giá mười triệu?",
  "Lịch thi đấu giải bóng rổ chuyên nghiệp?",
  "Cách pha cà phê muối ngon tại nhà?",
  "Địa điểm du lịch đẹp ở miền Trung?",
];

const PREFIX = ["", "Cho em hỏi: ", "Giải thích giúp em: "];

/**
 * 150 câu hỏi cố định (theo `rand` có hạt giống): lệch về A và B (≈ 53 % + 33 % trong các câu trong tài liệu), 10 câu ngoài tài liệu.
 * Mỗi câu một khoá riêng (`#n`) để chạy lại không nhân đôi.
 */
export function chatQuestions(rand) {
  const out = [];
  const add = (list, n) => {
    for (let i = 0; i < n; i++) out.push(`${PREFIX[Math.floor(i / list.length) % PREFIX.length]}${list[i % list.length]}`);
  };
  add(TOPIC_A, 70);
  add(TOPIC_B, 45);
  add(TOPIC_C, 25);
  add(OUTSIDE, 10);
  for (let i = out.length - 1; i > 0; i--) {
    const j = Math.floor(rand() * (i + 1));
    [out[i], out[j]] = [out[j], out[i]];
  }
  return out;
}

/**
 * Thread lớp 1: `want` là quyết định của Staff mong muốn trên bài AI (pending = để chờ; skip = câu ngoài tài liệu, AI không trả lời được).
 * Vì sao thread "có tài liệu" lấy chữ từ đoạn tài liệu (`doc`): ở `LLM_PROVIDER=fake` vectơ nhúng là băm của chuỗi (không có ngữ nghĩa), nên câu hỏi
 * tự nhiên LUÔN cho cosine < RAG_SIM_FLOOR (0,25) → AI bỏ qua. Thread có `title\nbody` đúng bằng chữ của một đoạn thì cosine = 1 → AI trả lời (xác định, tái lập được).
 * Seed chọn lần lượt các đoạn chưa dùng (theo thứ tự) có ≥ 2 dòng và qua tường lửa PII; tiêu đề = dòng đầu, nội dung = phần còn lại.
 */
export const THREADS_1 = [
  { k: "sv.gioi", doc: "Network Security Threats", week: 3, want: "pending" },
  { k: "sv.kha", doc: "Network Security Threats", week: 3, want: "pending" },
  { k: "sv04", doc: "Network Security Threats", week: 3, want: "pending" },
  { k: "sv05", doc: "Network Security Threats", week: 4, want: "pending" },
  { k: "sv06", doc: "Quy chế học vụ", week: 5, want: "verify" },
  { k: "sv07", doc: "Quy chế học vụ", week: 5, want: "verify" },
  { k: "sv08", doc: "Quy chế học vụ", week: 5, want: "verify" },
  { k: "sv09", doc: "Forecasting (QMB ch. 6b)", week: 6, want: "correct" },
  { k: "sv10", doc: "Forecasting (QMB ch. 6b)", week: 6, want: "correct" },
  { k: "sv11", doc: "Forecasting (QMB ch. 6b)", week: 6, want: "reject" },
  { k: "sv12", title: "Cách pha cà phê muối ngon nhất?", body: "Câu này không liên quan môn học, em hỏi cho vui.", week: null, want: "skip" },
  { k: "sv13", title: "Lịch thi đấu giải bóng đá cuối tuần này", body: "Ai biết lịch thi đấu thì chia sẻ giúp em.", week: null, want: "skip" },
];

/** Thread lớp 2 (ba thread, không ép trạng thái; tài liệu dùng chung từ lớp 1). */
export const THREADS_2 = [
  { k: "sv31", doc: "Network Security Threats", week: 3 },
  { k: "sv32", doc: "Network Security Threats", week: 3 },
  { k: "sv33", doc: "Forecasting (QMB ch. 6b)", week: 6 },
];

/** Hai sự kiện của lớp 1 (tương lai) và một của lớp 2 (SRS FEAT-docs-calendar 4.10 / US-P8-03 AC17); `days` = số ngày kể từ hôm nay, giờ Việt Nam. */
export const EVENTS = {
  "761987": [
    { type: "EXAM", title: "Thi giữa kỳ", days: 14, at: "07:00", location: "Phòng 301" },
    { type: "EXAM", title: "Thi cuối kỳ", days: 56, at: "07:00", location: "Phòng 301" },
  ],
  "761988": [{ type: "OTHER", title: "Nộp báo cáo nhóm", days: 10, at: "23:00", location: null }],
};

/** PDF một trang có lớp chữ (Helvetica, không phông nhúng) — đủ cho docling; không thêm thư viện. `lines` là ASCII (không dấu) để khỏi cần mã hoá phông. */
export function miniPdf(lines) {
  const esc = (s) => s.replace(/[\\()]/g, "\\$&");
  const text = lines.map((l, i) => `${i === 0 ? "" : "0 -18 Td "}(${esc(l)}) Tj`).join("\n");
  const stream = `BT /F1 12 Tf 72 740 Td\n${text}\nET`;
  const objs = [
    "<< /Type /Catalog /Pages 2 0 R >>",
    "<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
    "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>",
    `<< /Length ${stream.length} >>\nstream\n${stream}\nendstream`,
    "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
  ];
  let out = "%PDF-1.4\n";
  const offs = [];
  objs.forEach((o, i) => {
    offs.push(out.length);
    out += `${i + 1} 0 obj\n${o}\nendobj\n`;
  });
  const xref = out.length;
  out += `xref\n0 ${objs.length + 1}\n0000000000 65535 f \n${offs.map((o) => `${String(o).padStart(10, "0")} 00000 n \n`).join("")}`;
  out += `trailer\n<< /Size ${objs.length + 1} /Root 1 0 R >>\nstartxref\n${xref}\n%%EOF\n`;
  return Buffer.from(out, "latin1");
}

export const CANARY = "CANARY-7Q2X";
export const EXAM_PAPER_LINES = ["De tham khao tuan 5 - An ninh mang", "Cau 1: Neu ba loai tan cong mang pho bien.", "Cau 2: Phan biet phishing va spear phishing.", "Cau 3: Trinh bay cach phong chong ransomware."];
export const ANSWER_KEY_LINES = [`Dap an de tham khao tuan 5 ${CANARY}`, "Cau 1: phishing, ransomware, DDoS.", "Cau 2: spear phishing nham vao mot muc tieu cu the.", "Cau 3: sao luu ngoai tuyen va cap nhat ban va."];
