// Báo cáo lỗ hổng kiến thức (FLOWS F17) — dữ liệu MÔ PHỎNG cho /insights. Người giữ: US-PROTO-04.
// Câu mẫu đã ẩn danh ngay trong dữ liệu: không có tên, không có MSSV (F17 + SRS 4.6).
import { COURSE_1, COURSE_2, NOW } from "./core";

export type InsightTopic = {
  id: string;
  title: string;
  /** số sinh viên đã hỏi về chủ đề; < 3 thì không hiện câu mẫu (F17) */
  students: number;
  questions: number;
  /** thay đổi số câu hỏi so với báo cáo trước */
  delta: number;
  /** vì sao chủ đề này xếp cao */
  why: string;
  signals: string[];
  /** 3–5 câu hỏi đã ẩn danh; rỗng khi dưới 3 sinh viên hỏi */
  samples: string[];
  /** việc dạy nên đổi */
  action: string;
  /** nội dung thread ghim sinh ra từ chủ đề */
  threadBody: string;
};

export type UncoveredGap = { title: string; asks: number; note: string };

export type InsightReport = {
  courseId: string;
  generatedAt: Date;
  /** khoảng dữ liệu của báo cáo */
  window: string;
  questions: number;
  students: number;
  topics: InsightTopic[];
  uncovered: UncoveredGap[];
  /** chỉ có ở báo cáo vừa tạo: so với bản trước */
  comparison?: string[];
};

/** Mốc báo cáo đã có sẵn khi mở màn (trước "bây giờ"). */
const PREVIOUS_AT: Record<string, Date> = {
  [COURSE_1]: new Date("2026-10-22T16:40:00+07:00"),
  [COURSE_2]: new Date("2026-10-28T15:10:00+07:00"),
};

const TOPICS: Record<string, InsightTopic[]> = {
  [COURSE_1]: [
    {
      id: "aes-cbc",
      title: "Mật mã đối xứng (AES, CBC)",
      students: 14,
      questions: 31,
      delta: 6,
      why: "Gần một nửa lớp hỏi lại cùng một ý sau buổi 8, và hỏi đi hỏi lại trong ba ngày liền chứ không phải hỏi một lần rồi thôi.",
      signals: [
        "14/30 sinh viên hỏi về chế độ CBC trong tuần 8–9",
        "9 câu lẫn giữa ECB và CBC khi nói về khối lặp",
        "Câu AES trong đề luyện: đúng 46% — thấp nhất trong 8 chủ đề",
        "7 lượt luyện chủ đề “Mã hoá đối xứng” bỏ dở giữa chừng",
        "3 câu phải chuyển giảng viên vì AI không đủ chắc chắn",
      ],
      samples: [
        "Vì sao CBC cần IV ngẫu nhiên còn ECB thì không cần gì cả ạ?",
        "Nếu dùng lại IV cho hai bản tin khác nhau thì kẻ tấn công biết được gì?",
        "AES-128 và AES-256 khác nhau ở số vòng hay ở kích thước khối ạ?",
        "Padding PKCS#7 thêm bao nhiêu byte khi dữ liệu vừa đúng 16 byte?",
        "Giải mã CBC có chạy song song được không, hay phải tuần tự như lúc mã hoá?",
      ],
      action: "Dành 20 phút đầu buổi 11 vẽ lại sơ đồ CBC với một khối lặp, đối chiếu ảnh chim cánh cụt mã bằng ECB, rồi giao 2 câu luyện về IV.",
      threadBody:
        "Nhiều bạn đang vướng ở chỗ vì sao CBC cần IV. Tóm tắt: ECB mã mỗi khối độc lập nên khối giống nhau cho bản mã giống nhau — lộ cấu trúc dữ liệu. CBC XOR mỗi khối với bản mã khối trước, khối đầu tiên không có khối trước nên phải mượn IV; IV cần ngẫu nhiên và không lặp lại, nhưng không cần bí mật. Buổi 11 mình sẽ vẽ lại sơ đồ này, mọi người cứ hỏi tiếp dưới thread.",
    },
    {
      id: "hash-sign",
      title: "Hàm băm và chữ ký số",
      students: 11,
      questions: 24,
      delta: 4,
      why: "Câu hỏi dồn vào đúng phần sẽ kiểm tra giữa kỳ, và phần lớn là hiểu ngược vai trò của khoá riêng với khoá công khai.",
      signals: [
        "11/30 sinh viên hỏi trong 6 ngày gần nhất",
        "6 câu nói “mã hoá bằng khoá riêng” khi ý là “ký”",
        "5 câu hỏi vì sao băm rồi mới ký, không ký thẳng tài liệu",
        "Bài tập 02: 8 bài mất điểm ở phần kiểm tra toàn vẹn",
        "2 câu hỏi về SHA-1 còn dùng được không — tài liệu lớp chưa nói",
      ],
      samples: [
        "Ký số là mã hoá bằng khoá riêng đúng không ạ, hay là hai việc khác nhau?",
        "Vì sao phải băm tài liệu trước rồi mới ký, ký thẳng cả file không được sao?",
        "Va chạm của hàm băm nghĩa là hai file khác nhau ra cùng một mã băm phải không?",
        "SHA-1 bị coi là yếu rồi thì hệ thống cũ đang dùng nên đổi sang gì ạ?",
      ],
      action: "Thêm một bảng đối chiếu “ký ≠ mã hoá” vào slide buổi 11 và một câu trong đề ôn giữa kỳ bắt chọn đúng khoá cho từng việc.",
      threadBody:
        "Ghi nhớ nhanh trước giữa kỳ: mã hoá dùng khoá công khai của người nhận (để chỉ họ đọc được), ký dùng khoá riêng của người gửi (để ai cũng kiểm được là của mình). Băm trước khi ký vì chữ ký làm trên dữ liệu ngắn cố định sẽ nhanh và an toàn hơn ký cả tệp. Ai còn lẫn hai việc này cứ hỏi dưới đây.",
    },
    {
      id: "sqli",
      title: "SQL injection và truy vấn tham số hoá",
      students: 7,
      questions: 13,
      delta: 1,
      why: "Số câu hỏi ổn định sau buổi thực hành, chủ yếu ở bước phòng thủ chứ không ở bước khai thác.",
      signals: [
        "7 sinh viên hỏi, tập trung trong hai ngày sau buổi thực hành",
        "4 câu hỏi vì sao lọc dấu nháy đơn vẫn chưa đủ an toàn",
        "3 câu nhầm truy vấn tham số hoá với việc escape chuỗi",
        "Bài nộp 02: 5 bài dùng nối chuỗi trong ví dụ phòng thủ",
        "Không câu nào phải chuyển giảng viên — AI trả lời đủ chắc chắn",
      ],
      samples: [
        "Em lọc hết dấu nháy đơn rồi mà vẫn bị khai thác, là do đâu ạ?",
        "Truy vấn tham số hoá khác gì với escape chuỗi đầu vào?",
        "ORM có tự chống SQL injection không hay vẫn phải tự lo?",
      ],
      action: "Giữ nguyên nội dung, chỉ thêm một ví dụ sai–đúng cạnh nhau trong tài liệu thực hành.",
      threadBody:
        "Lọc ký tự là vá tạm; truy vấn tham số hoá mới là cách đúng vì dữ liệu không bao giờ được ghép vào câu lệnh. Mình đã thêm ví dụ sai–đúng vào tài liệu thực hành, mọi người xem và hỏi tiếp ở đây.",
    },
    {
      id: "rsa",
      title: "RSA và trao đổi khoá",
      students: 5,
      questions: 9,
      delta: 2,
      why: "Nhóm nhỏ nhưng hỏi sâu, phần lớn vướng ở quan hệ giữa độ dài khoá và chi phí tính toán.",
      signals: [
        "5 sinh viên hỏi trong tuần 9",
        "4 câu so sánh RSA với Diffie–Hellman",
        "2 câu hỏi vì sao không mã hoá cả tệp bằng RSA",
        "Đề luyện chủ đề này mới có 6 lượt làm",
        "Tài liệu lớp có mục này nhưng chưa có ví dụ số",
      ],
      samples: [
        "Vì sao không mã hoá cả tệp bằng RSA cho gọn mà phải ghép với AES?",
        "RSA và Diffie–Hellman đều là khoá công khai, khác nhau ở chỗ nào ạ?",
        "Khoá 2048 bit với 4096 bit chênh nhau bao nhiêu về tốc độ thực tế?",
      ],
      action: "Thêm một ví dụ số nhỏ (p, q nhỏ) vào tài liệu để sinh viên tự tính một lần.",
      threadBody: "Mình bổ sung ví dụ số của RSA với p, q nhỏ để mọi người tự tính một lượt rồi mới nhìn công thức tổng quát. Hỏi thêm ở dưới.",
    },
    {
      id: "tls",
      title: "Bắt tay TLS 1.3",
      students: 2,
      questions: 3,
      delta: 1,
      why: "Mới xuất hiện cuối tuần 9, chưa đủ người hỏi để kết luận là lỗ hổng chung của lớp.",
      signals: [
        "2 sinh viên hỏi, cùng trong một buổi tối",
        "Chủ đề chưa nằm trong bài tập nào đã giao",
        "Chưa có lượt luyện nào cho chủ đề này",
        "Tài liệu lớp mới nhắc tên, chưa có sơ đồ bắt tay",
        "Không có câu nào phải chuyển giảng viên",
      ],
      samples: [],
      action: "Chưa cần đổi bài giảng. Nếu sau buổi 11 vẫn còn người hỏi thì thêm sơ đồ bắt tay vào tài liệu.",
      threadBody: "Mình thêm sơ đồ bắt tay TLS 1.3 vào tài liệu tham khảo cho bạn nào muốn đọc trước.",
    },
  ],
  [COURSE_2]: [
    {
      id: "firewall",
      title: "Tường lửa và phân đoạn mạng",
      students: 12,
      questions: 22,
      delta: 5,
      why: "Lớp mới học xong buổi 2 về kiến trúc mạng và gần như cả lớp vướng ở chỗ luật tường lửa đọc theo thứ tự nào.",
      signals: [
        "12/30 sinh viên hỏi trong 5 ngày đầu",
        "7 câu hỏi về thứ tự khớp luật và luật mặc định cuối bảng",
        "5 câu lẫn giữa phân đoạn mạng và chia dải địa chỉ",
        "Chưa có bài tập nào về chủ đề này nên chưa có dữ liệu bài nộp",
        "2 câu phải chuyển giảng viên vì hỏi về thiết bị của phòng lab",
      ],
      samples: [
        "Luật tường lửa khớp từ trên xuống hay cái cụ thể nhất thắng ạ?",
        "Phân đoạn mạng khác gì với việc chia subnet cho từng phòng?",
        "Tường lửa chặn theo cổng thì ứng dụng chạy cổng 443 có thoát được không?",
        "DMZ đặt máy chủ web vào đó thì khác gì để trong mạng nội bộ?",
      ],
      action: "Buổi 4 bắt đầu bằng một bảng luật 5 dòng cho cả lớp đọc thứ tự khớp, rồi mới nói tới phân đoạn.",
      threadBody:
        "Luật tường lửa khớp từ trên xuống, dừng ở luật đầu tiên trùng, nên luật deny-all luôn nằm cuối bảng. Phân đoạn mạng là tách vùng theo mức tin cậy (lab, văn phòng, máy chủ), không phải chỉ chia dải địa chỉ. Buổi 4 mình đi lại phần này, mọi người hỏi tiếp dưới thread.",
    },
    {
      id: "osi",
      title: "Mô hình OSI và đường đi của gói tin",
      students: 8,
      questions: 15,
      delta: 3,
      why: "Câu hỏi nền tảng của lớp mới: không đọc được đường đi của gói thì không theo kịp phần tường lửa.",
      signals: [
        "8/30 sinh viên hỏi trong tuần đầu",
        "6 câu hỏi tầng nào đóng gói thêm phần đầu nào",
        "4 câu hỏi vì sao thực tế dùng TCP/IP 4 tầng mà học OSI 7 tầng",
        "Chưa có lượt luyện nào — lớp chưa mở đề luyện",
        "Không câu nào phải chuyển giảng viên",
      ],
      samples: [
        "Gói tin đi từ tầng ứng dụng xuống thì mỗi tầng thêm gì vào ạ?",
        "Vì sao thực tế dùng TCP/IP 4 tầng mà lớp vẫn học OSI 7 tầng?",
        "Switch làm việc ở tầng 2, vậy router ở tầng 3 thì khác nhau ở cái gì?",
      ],
      action: "Giao một bài đọc ngắn trước buổi 4 và mở đề luyện 10 câu cho chủ đề này.",
      threadBody: "Mình mở đề luyện 10 câu cho phần OSI và đường đi của gói tin; làm trước buổi 4 sẽ theo được phần tường lửa.",
    },
    {
      id: "vpn",
      title: "VPN site-to-site và IPsec",
      students: 4,
      questions: 7,
      delta: 2,
      why: "Nhóm nhỏ hỏi trước chương trình, chủ yếu vì liên quan tới chỗ thực tập của các bạn.",
      signals: [
        "4 sinh viên hỏi, đều ngoài giờ lên lớp",
        "3 câu hỏi khác nhau giữa site-to-site và VPN cá nhân",
        "2 câu hỏi về chế độ tunnel và transport của IPsec",
        "Chủ đề nằm ở buổi 9 theo kế hoạch lớp",
        "Tài liệu lớp chưa có phần này",
      ],
      samples: ["VPN site-to-site khác gì VPN mình cài trên máy cá nhân ạ?", "IPsec chế độ tunnel và transport dùng khi nào ạ?", "VPN có chống được người trong cùng mạng nghe lén không?"],
      action: "Chưa đổi lịch; trả lời trong thread và giữ nội dung cho buổi 9.",
      threadBody: "Phần VPN nằm ở buổi 9, nhưng mình trả lời trước ở đây cho bạn nào đang thực tập cần dùng ngay.",
    },
    {
      id: "ids",
      title: "Phát hiện xâm nhập (IDS/IPS)",
      students: 2,
      questions: 3,
      delta: 0,
      why: "Mới hai người hỏi, chưa đủ để kết luận là lỗ hổng chung của lớp.",
      signals: [
        "2 sinh viên hỏi trong cùng một ngày",
        "Chủ đề chưa được giảng",
        "Chưa có bài tập hay đề luyện liên quan",
        "Tài liệu lớp chưa có phần này",
        "Không có câu nào phải chuyển giảng viên",
      ],
      samples: [],
      action: "Chưa cần làm gì. Xem lại ở báo cáo tuần sau.",
      threadBody: "Mình ghim một bài đọc ngắn về IDS/IPS cho bạn nào muốn tìm hiểu trước.",
    },
  ],
};

const UNCOVERED: Record<string, UncoveredGap[]> = {
  [COURSE_1]: [
    {
      title: "Quy định mang tài liệu vào phòng thi cuối kỳ",
      asks: 4,
      note: "Không tài liệu nào của lớp trả lời được. Quyche.pdf chỉ nói thang điểm và điều kiện dự thi.",
    },
    {
      title: "Thiết bị tường lửa dùng trong bài thực hành 3",
      asks: 3,
      note: "Mordern_Network_Security_Threats.pdf dừng ở mức khái niệm, chưa có hướng dẫn thao tác trên thiết bị của phòng lab.",
    },
    {
      title: "Cách nộp lại bài khi gửi nhầm tệp",
      asks: 2,
      note: "Quy định nộp bài đang nằm trong lời nói ở buổi học, chưa thành văn bản trong lớp.",
    },
  ],
  [COURSE_2]: [
    { title: "Phạm vi ôn và lịch kiểm tra giữa kỳ", asks: 5, note: "Lớp chưa tải quy chế môn học nên chưa có nguồn nào để trả lời." },
    { title: "Danh sách phần mềm cần cài trước buổi thực hành", asks: 3, note: "Chưa có tài liệu hướng dẫn cài đặt trong lớp." },
  ],
};

/** Buổi ôn tập đề xuất khi giảng viên bấm `Tạo buổi ôn tập` (ngoài giờ lên lớp chính thức). */
export const REVIEW_SLOT: Record<string, { at: string; label: string; durationMin: number }> = {
  [COURSE_1]: { at: "2026-11-03T19:30:00+07:00", label: "Thứ Ba 03/11, 19:30–20:30", durationMin: 60 },
  [COURSE_2]: { at: "2026-11-05T19:30:00+07:00", label: "Thứ Năm 05/11, 19:30–20:30", durationMin: 60 },
};

/**
 * Báo cáo của một lớp. `fresh` = vừa bấm `Tạo báo cáo mới`: ngày hôm nay, số liệu mới và phần so
 * với bản trước; ngược lại là bản đã có sẵn (số liệu trừ đi phần tăng thêm).
 */
export function reportFor(courseId: string, fresh: boolean): InsightReport {
  const topics = TOPICS[courseId] ?? TOPICS[COURSE_1];
  const list = fresh ? topics : topics.map((t) => ({ ...t, questions: t.questions - t.delta }));
  const questions = list.reduce((n, t) => n + t.questions, 0);
  const students = Math.max(...list.map((t) => t.students));
  return {
    courseId,
    generatedAt: fresh ? NOW : PREVIOUS_AT[courseId] ?? PREVIOUS_AT[COURSE_1],
    window: fresh ? "Câu hỏi 7 ngày gần nhất, đã ẩn danh" : "Câu hỏi 7 ngày trước đó, đã ẩn danh",
    questions,
    students,
    topics: list,
    uncovered: UNCOVERED[courseId] ?? UNCOVERED[COURSE_1],
    comparison: fresh
      ? [
          `${list[0].title}: +${list[0].delta} câu so với báo cáo trước, vẫn đứng đầu`,
          `${list[1].title}: +${list[1].delta} câu, giữ nguyên vị trí thứ hai`,
          `${list[list.length - 1].title}: ${list[list.length - 1].students} sinh viên hỏi — chưa đủ 3 người để hiện câu mẫu`,
        ]
      : undefined,
  };
}
