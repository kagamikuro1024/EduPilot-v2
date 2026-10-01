// Threads công khai của lớp 1 (US-PROTO-01, /threads và /threads/[id]).
// 12 thread, 2 ghim; "CBC khác ECB ở điểm nào?" có câu trả lời AI đang Chờ xác nhận;
// "Vì sao cần muối (salt) khi băm mật khẩu?" đã được giảng viên xác nhận.

import type { Citation } from "./chat";
import { COURSE_1, STAFF, STUDENTS, STUDENT_B } from "./core";

export type PostAuthor = "student" | "ai" | "teacher";

export type ThreadPost = {
  id: string;
  author: string;
  role: PostAuthor;
  body: string;
  minsAgo: number;
  /** câu trả lời AI: đã được giảng viên xác nhận / còn chờ xác nhận */
  state?: "verified" | "pending";
  verifiedBy?: string;
  citations?: Citation[];
  /** bài của chính người đang xem (được sửa / xoá) */
  mine?: boolean;
};

export type Thread = {
  id: string;
  courseId: string;
  title: string;
  topic: string;
  week: number;
  minsAgo: number;
  replies: number;
  pinned?: boolean;
  answer: "verified" | "pending" | "none";
  posts: ThreadPost[];
};

const sv = (i: number) => STUDENTS[i].name;

export const THREADS: Thread[] = [
  {
    id: "t-pin-rubric",
    courseId: COURSE_1,
    title: "Đọc trước khi hỏi: rubric Bài tập 03 và cách trích nguồn",
    topic: "Thông báo",
    week: 8,
    minsAgo: 60 * 26,
    replies: 4,
    pinned: true,
    answer: "verified",
    posts: [
      {
        id: "p1",
        author: STAFF.teacher.name,
        role: "teacher",
        body: "Bài tập 03 chấm theo 4 tiêu chí, mỗi tiêu chí 2,5 điểm: tác nhân và bề mặt tấn công, phân tích tấn công, đánh giá tác động, biện pháp phòng thủ. Mọi số liệu trích từ báo cáo công bố phải ghi rõ nguồn và ngày truy cập.",
        minsAgo: 60 * 26,
      },
    ],
  },
  {
    id: "t-pin-lab",
    courseId: COURSE_1,
    title: "Hướng dẫn cài phòng lab ảo cho bài thực hành bắt gói tin",
    topic: "Thực hành",
    week: 9,
    minsAgo: 60 * 20,
    replies: 7,
    pinned: true,
    answer: "verified",
    posts: [
      {
        id: "p1",
        author: STAFF.ta.name,
        role: "teacher",
        body: "Dùng máy ảo Ubuntu 24.04, cài Wireshark và tcpdump. Bắt gói trên card mạng nội bộ của máy ảo, không bắt trên mạng trường.",
        minsAgo: 60 * 20,
      },
    ],
  },
  {
    id: "t-cbc",
    courseId: COURSE_1,
    title: "CBC khác ECB ở điểm nào?",
    topic: "Mật mã đối xứng",
    week: 10,
    minsAgo: 95,
    replies: 2,
    answer: "pending",
    posts: [
      {
        id: "p1",
        author: sv(6),
        role: "student",
        body: "Em đọc slide chương 3 thấy AES có nhiều chế độ. Em vẫn chưa hiểu vì sao ảnh mã hoá bằng ECB lại còn nhìn ra hình, còn CBC thì không. Hai chế độ khác nhau ở bước nào ạ?",
        minsAgo: 95,
      },
      {
        id: "p2",
        author: "Trợ lý AI của lớp",
        role: "ai",
        body: "ECB mã hoá từng khối 16 byte độc lập, nên hai khối bản rõ giống nhau luôn cho ra hai khối bản mã giống nhau — vì vậy ảnh mã hoá vẫn lộ đường nét. CBC lấy khối bản mã trước XOR với khối bản rõ hiện tại rồi mới mã hoá, và khối đầu tiên XOR với IV ngẫu nhiên, nên cùng một bản rõ mã hoá hai lần cho ra hai bản mã khác nhau. Đổi lại, CBC phải mã hoá tuần tự và cần IV không lặp lại.",
        minsAgo: 93,
        state: "pending",
        citations: [
          { title: "Chương 3 — Mật mã đối xứng và chế độ vận hành", locator: "trang 14–17", href: "/library" },
          { title: "Modern Network Security Threats", locator: "mục 2.4", href: "/library" },
        ],
      },
    ],
  },
  {
    id: "t-salt",
    courseId: COURSE_1,
    title: "Vì sao cần muối (salt) khi băm mật khẩu?",
    topic: "Hàm băm và chữ ký số",
    week: 9,
    minsAgo: 60 * 30,
    replies: 5,
    answer: "verified",
    posts: [
      {
        id: "p1",
        author: STUDENT_B.name,
        role: "student",
        body: "Nếu SHA-256 đã khó đảo ngược thì vì sao còn phải thêm muối cho mỗi mật khẩu ạ?",
        minsAgo: 60 * 31,
        mine: true,
      },
      {
        id: "p2",
        author: "Trợ lý AI của lớp",
        role: "ai",
        body: "Muối là chuỗi ngẫu nhiên riêng cho từng người dùng, ghép vào mật khẩu trước khi băm. Nó làm hai người đặt trùng mật khẩu vẫn có giá trị băm khác nhau và khiến bảng tra sẵn (rainbow table) vô dụng vì kẻ tấn công phải dựng bảng riêng cho từng muối. Với mật khẩu nên dùng hàm băm chậm có muối như bcrypt hoặc Argon2 thay cho SHA-256 trần.",
        minsAgo: 60 * 30,
        state: "verified",
        verifiedBy: STAFF.teacher.name,
        citations: [{ title: "Chương 4 — Hàm băm và chữ ký số", locator: "trang 8", href: "/library" }],
      },
    ],
  },
  {
    id: "t-rsa-key",
    courseId: COURSE_1,
    title: "Khoá RSA 2048 bit còn đủ an toàn tới khi nào?",
    topic: "Mật mã khoá công khai",
    week: 10,
    minsAgo: 60 * 5,
    replies: 3,
    answer: "pending",
    posts: [{ id: "p1", author: sv(10), role: "student", body: "Thầy có nói 2048 bit vẫn dùng được. Vậy khi nào thì phải chuyển sang 3072 bit hoặc đường cong elliptic ạ?", minsAgo: 60 * 5 }],
  },
  {
    id: "t-sqli",
    courseId: COURSE_1,
    title: "Câu lệnh tham số hoá có chặn được mọi dạng SQL injection không?",
    topic: "An toàn ứng dụng web",
    week: 8,
    minsAgo: 60 * 50,
    replies: 6,
    answer: "verified",
    posts: [{ id: "p1", author: sv(14), role: "student", body: "Em thấy tài liệu nói dùng prepared statement là xong, nhưng có bài viết nói vẫn bị injection ở phần ORDER BY. Thực tế thế nào ạ?", minsAgo: 60 * 50 }],
  },
  {
    id: "t-xss",
    courseId: COURSE_1,
    title: "Phân biệt XSS lưu trữ và XSS phản chiếu trong bài tập 02",
    topic: "An toàn ứng dụng web",
    week: 7,
    minsAgo: 60 * 72,
    replies: 4,
    answer: "verified",
    posts: [{ id: "p1", author: sv(18), role: "student", body: "Trong ví dụ ô bình luận của bài tập 02 thì tính là XSS lưu trữ hay phản chiếu ạ?", minsAgo: 60 * 72 }],
  },
  {
    id: "t-firewall",
    courseId: COURSE_1,
    title: "Tường lửa trạng thái xử lý gói UDP thế nào?",
    topic: "Tường lửa và phân đoạn mạng",
    week: 6,
    minsAgo: 60 * 96,
    replies: 2,
    answer: "none",
    posts: [{ id: "p1", author: sv(21), role: "student", body: "UDP không có bắt tay ba bước thì tường lửa trạng thái dựa vào đâu để biết gói trả về là hợp lệ ạ?", minsAgo: 60 * 96 }],
  },
  {
    id: "t-vpn",
    courseId: COURSE_1,
    title: "IPsec và TLS VPN: chọn cái nào cho nhà thầu truy cập từ xa?",
    topic: "VPN",
    week: 6,
    minsAgo: 60 * 110,
    replies: 3,
    answer: "verified",
    posts: [{ id: "p1", author: sv(24), role: "student", body: "Tình huống trong slide là nhà thầu bên ngoài cần vào một máy chủ nội bộ. Nên dùng IPsec site-to-site hay TLS VPN ạ?", minsAgo: 60 * 110 }],
  },
  {
    id: "t-pki",
    courseId: COURSE_1,
    title: "Chuỗi chứng thư số bị lỗi 'unable to get local issuer certificate'",
    topic: "PKI",
    week: 5,
    minsAgo: 60 * 130,
    replies: 5,
    answer: "verified",
    posts: [{ id: "p1", author: sv(27), role: "student", body: "Em dựng máy chủ thử nghiệm, trình duyệt báo lỗi chuỗi chứng thư. Thiếu chứng thư trung gian thì sửa ở đâu ạ?", minsAgo: 60 * 130 }],
  },
  {
    id: "t-wifi",
    courseId: COURSE_1,
    title: "WPA3 khắc phục điểm yếu nào của WPA2?",
    topic: "An toàn mạng không dây",
    week: 4,
    minsAgo: 60 * 170,
    replies: 2,
    answer: "none",
    posts: [{ id: "p1", author: sv(29), role: "student", body: "Tấn công KRACK có còn hiệu lực với WPA3 không ạ?", minsAgo: 60 * 170 }],
  },
  {
    id: "t-phishing",
    courseId: COURSE_1,
    title: "Dấu hiệu nhận biết thư lừa đảo nhắm mục tiêu",
    topic: "Kỹ thuật lừa đảo",
    week: 3,
    minsAgo: 60 * 220,
    replies: 8,
    answer: "verified",
    posts: [{ id: "p1", author: sv(31), role: "student", body: "Thư giả danh phòng đào tạo gửi link đổi mật khẩu thì nên kiểm tra những gì trước khi bấm ạ?", minsAgo: 60 * 220 }],
  },
];

export function threadById(id: string) {
  return THREADS.find((t) => t.id === id);
}

export const THREAD_TOPICS = [...new Set(THREADS.map((t) => t.topic))];

/** Lý do báo cáo một bài viết (SV chọn một lý do rồi gửi). */
export const REPORT_REASONS = ["Sai kiến thức", "Lộ thông tin cá nhân", "Nội dung không phù hợp", "Spam hoặc lặp lại"];

/** Bài mới do người dùng đăng trong phiên demo (lát trạng thái "threads.posts"). */
export type NewThread = { id: string; title: string; body: string; topic: string; answered: boolean };
export const NEW_THREADS_KEY = "threads.posts";
