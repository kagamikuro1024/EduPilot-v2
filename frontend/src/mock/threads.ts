// Threads công khai của lớp (US-PROTO-01, /threads và /threads/[id]) — "như thật" theo SRS 4.3.1 (#18).
// Dữ liệu seed, bộ mẫu trả lời AI theo chủ đề, quy tắc chọn mẫu, và bộ chọn (selector) dựng ThreadView từ seed + trạng thái phiên.
// Mọi con số hiển thị (số phản hồi, số người tham gia, hoạt động gần nhất) TÍNH từ dữ liệu, không lưu số riêng.

import type { Citation } from "./chat";
import { COURSE_1, NOW, STAFF, STUDENTS, STUDENT_B } from "./core";
import { agoLabel } from "./derive";

export type PostRole = "student" | "ta" | "teacher" | "ai";
export type AiState = "pending" | "waiting" | "verified" | "corrected";

export type AiMeta = {
  /** answer: trả lời câu hỏi gốc · hint: "Gợi ý thêm" · fallback: không khớp mẫu nào (D49) */
  kind: "answer" | "hint" | "fallback";
  /** câu trả lời AI chính của thread (hiện ở khối riêng, không tính vào "Thảo luận (n)") */
  main: boolean;
  state: AiState;
  verifiedBy?: string;
  /** bản AI gốc khi giảng viên đã sửa */
  originalBody?: string;
  /** "↳ trả lời <tên>" */
  replyToName?: string;
  citations: Citation[];
  template?: string;
  /** chỉ bài tạo trong phiên: mốc bắt đầu soạn, để tính tiến trình "đang soạn → chảy chữ → nguồn" */
  startedMs?: number;
  /** số ký tự đã hiện khi bấm Dừng */
  stoppedAt?: number;
};

export type ThreadPost = {
  id: string;
  threadId: string;
  authorId: string;
  author: string;
  role: PostRole;
  body: string;
  /** bài seed: số phút trước "bây giờ" 29/10 09:20 */
  minsAgo?: number;
  /** bài tạo trong phiên: mốc thời gian thật (Date.now) */
  ms?: number;
  /** id bài được trích (cùng thread) */
  quoteOf?: string;
  ai?: AiMeta;
  edited?: boolean;
};

export type SeedThread = {
  id: string;
  courseId: string;
  title: string;
  topic: string;
  week: number;
  pinned?: boolean;
  posts: ThreadPost[];
};

/** Chủ đề môn học dùng cho bộ lọc và ô chọn khi đặt câu hỏi. */
export const THREAD_TOPICS = [
  "Mật mã đối xứng",
  "Hàm băm và chữ ký số",
  "Mật mã khoá công khai",
  "PKI",
  "An toàn ứng dụng web",
  "Tường lửa và phân đoạn mạng",
  "VPN",
  "An toàn mạng không dây",
  "Kỹ thuật lừa đảo",
  "Thực hành",
  "Thông báo",
] as const;

/** Lý do báo cáo một bài viết (SV chọn một lý do rồi gửi). */
export const REPORT_REASONS = ["Sai kiến thức", "Lộ thông tin cá nhân", "Nội dung không phù hợp", "Spam hoặc lặp lại"];

export const AI_NAME = "Trợ lý AI của lớp";
export const ROLE_NAME: Record<PostRole, string> = { student: "Sinh viên", ta: "Trợ giảng", teacher: "Giảng viên", ai: "Trợ lý AI" };

// ---- mẫu trả lời AI theo chủ đề (SRS 4.3.1 C) -------------------------------------------------------------------------

export type AiTemplate = {
  id: string;
  topic: string;
  /** từ khoá viết không dấu, so khớp nguyên từ / cụm từ */
  keywords: string[];
  /** gợi ý + câu hỏi ngược (phần in đậm đánh dấu bằng **…**) */
  hint: string;
  /** "Thử thêm: …" — dùng khi bấm Hỏi trợ lý AI với ô soạn trống */
  more: string;
  citations: Citation[];
  /** phản hồi trễ của trợ giảng (SRS 4.3.1 G) */
  taReply: string;
};

export const AI_TEMPLATES: AiTemplate[] = [
  {
    id: "S1",
    topic: "Mật mã đối xứng",
    keywords: ["ecb", "cbc", "che do", "khoi ban ma", "khoi ban ro", "anh ma hoa", "padding"],
    hint: "Hãy so sánh cách mỗi chế độ xử lý hai khối bản rõ giống hệt nhau: ECB mã hoá từng khối độc lập, CBC trộn khối bản mã trước vào khối hiện tại rồi mới mã hoá. **Với ECB, hai khối giống nhau cho ra hai khối bản mã thế nào — và điều đó làm lộ gì trong ảnh mã hoá?**",
    more: "Thử thêm: nếu một khối bản mã CBC bị hỏng khi truyền, khi giải mã sẽ có bao nhiêu khối bản rõ bị ảnh hưởng?",
    citations: [{ title: "Chương 3 — Mật mã đối xứng và chế độ vận hành", locator: "trang 14–17", href: "/library" }, { title: "Modern Network Security Threats", locator: "mục 2.4", href: "/library" }],
    taReply: "Em thử mã hoá ảnh mẫu trong lab 3 bằng cả hai chế độ rồi so hai ảnh là thấy ngay.",
  },
  {
    id: "S2",
    topic: "Mật mã đối xứng",
    keywords: ["iv", "vec-to khoi tao", "ctr", "nonce", "bo dem", "gcm", "dung lai"],
    hint: "IV không cần giữ bí mật, nhưng có một điều kiện quan trọng hơn bí mật. Ở CTR, bộ đếm sinh ra dòng khoá rồi XOR với bản rõ. **Nếu dùng lại cùng nonce với cùng khoá cho hai thông điệp, XOR hai bản mã với nhau sẽ cho ra gì?**",
    more: "Thử thêm: GCM thêm gì so với CTR để phát hiện bản mã bị sửa?",
    citations: [{ title: "Chương 3 — Mật mã đối xứng và chế độ vận hành", locator: "trang 18–20", href: "/library" }],
    taReply: "Đúng hướng rồi. Điều kiện đó là không bao giờ dùng lại nonce với cùng khoá — slide 19 có ví dụ.",
  },
  {
    id: "H1",
    topic: "Hàm băm và chữ ký số",
    keywords: ["bam", "hash", "sha", "muoi", "salt", "mat khau", "rainbow", "bcrypt"],
    hint: "Kẻ tấn công không cần đảo ngược hàm băm — họ chỉ cần đoán rồi băm thử. **Nếu hai người đặt cùng mật khẩu mà không có muối, giá trị băm của họ có gì đặc biệt, và bảng tra sẵn tận dụng điều đó ra sao?**",
    more: "Thử thêm: vì sao hàm băm mật khẩu nên chậm có chủ đích, khác với SHA-256 dùng cho tính toàn vẹn?",
    citations: [{ title: "Chương 4 — Hàm băm và chữ ký số", locator: "trang 6–9", href: "/library" }],
    taReply: "Em xem bảng so sánh SHA-256 với bcrypt ở trang 9 rồi trả lời câu AI hỏi nhé.",
  },
  {
    id: "H2",
    topic: "Hàm băm và chữ ký số",
    keywords: ["chu ky", "ky so", "toan ven", "xac thuc nguon"],
    hint: "Chữ ký số ký lên giá trị băm của thông điệp bằng khoá bí mật của người gửi. **Người nhận dùng khoá nào để kiểm tra, và nếu thông điệp bị sửa một bit thì bước kiểm tra nào thất bại?**",
    more: "Thử thêm: chữ ký số chứng minh được điều gì mà HMAC không chứng minh được?",
    citations: [{ title: "Chương 4 — Hàm băm và chữ ký số", locator: "trang 12–15", href: "/library" }],
    taReply: "Em vẽ lại sơ đồ ký và kiểm tra ở trang 13, đánh dấu chỗ dùng khoá công khai.",
  },
  {
    id: "K1",
    topic: "Mật mã khoá công khai",
    keywords: ["rsa", "khoa cong khai", "2048", "3072", "ecc", "duong cong", "thua so"],
    hint: "Độ an toàn của RSA dựa trên độ khó phân tích một số rất lớn ra thừa số nguyên tố, và khuyến nghị độ dài khoá thay đổi theo năng lực tính toán. **Nếu dữ liệu cần giữ bí mật 15 năm, bạn chọn độ dài khoá theo hôm nay hay theo dự báo của thời điểm đó?**",
    more: "Thử thêm: vì sao khoá ECC 256 bit được xem là tương đương RSA 3072 bit?",
    citations: [{ title: "Chương 5 — Quản lý khoá và PKI", locator: "trang 3–5", href: "/library" }],
    taReply: "Bảng độ dài khoá theo năm ở trang 5 trả lời được câu này.",
  },
  {
    id: "P1",
    topic: "PKI",
    keywords: ["chung thu", "certificate", "issuer", "chuoi chung thu", "ocsp", "crl", "tls", "trung gian"],
    hint: "Trình duyệt chỉ tin một chứng thư khi lần ngược được tới một CA gốc nó đã tin sẵn. **Máy chủ của bạn đang gửi kèm những chứng thư nào — có thiếu chứng thư trung gian không?**",
    more: "Thử thêm: khi một chứng thư bị thu hồi, trình duyệt biết bằng cách nào?",
    citations: [{ title: "Chương 5 — Quản lý khoá và PKI", locator: "trang 9–12", href: "/library" }],
    taReply: "Em chạy `openssl s_client -showcerts` để xem máy chủ gửi những gì.",
  },
  {
    id: "W1",
    topic: "An toàn ứng dụng web",
    keywords: ["sql", "injection", "tham so", "prepared", "cau lenh", "order by"],
    hint: "Hãy phân biệt chỗ dữ liệu người dùng được coi là dữ liệu và chỗ nó bị hiểu thành mã; câu lệnh tham số hoá giữ giá trị ở vai dữ liệu. **Còn tên cột hay chiều sắp xếp lấy từ người dùng thì có truyền dưới dạng tham số được không — nếu không, bạn chặn bằng cách nào?**",
    more: "Thử thêm: tài khoản cơ sở dữ liệu của ứng dụng nên có những quyền gì để giảm thiệt hại nếu vẫn bị chèn?",
    citations: [{ title: "Chương 2 — Tấn công mạng phổ biến", locator: "trang 18–22", href: "/library" }],
    taReply: "Em thử với bảng `products` trong lab 2, đổi `ORDER BY` thành tham số rồi xem lỗi gì.",
  },
  {
    id: "W2",
    topic: "An toàn ứng dụng web",
    keywords: ["xss", "script", "phan chieu", "luu tru", "html", "cookie"],
    hint: "Hãy lần theo đoạn mã độc đi đâu trước khi tới trình duyệt nạn nhân. **Trong ô bình luận của bài tập 02, script được lưu rồi hiện cho mọi người, hay chỉ phản chiếu lại trong một đường link?**",
    more: "Thử thêm: mã hoá đầu ra theo ngữ cảnh HTML khác gì với lọc đầu vào?",
    citations: [{ title: "Chương 2 — Tấn công mạng phổ biến", locator: "trang 23–25", href: "/library" }],
    taReply: "Em mở tab Network xem script đi từ request nào là phân biệt được.",
  },
  {
    id: "V1",
    topic: "VPN",
    keywords: ["vpn", "ipsec", "tls vpn", "nha thau", "truy cap tu xa", "duong ham"],
    hint: "IPsec nối cả mạng ở tầng mạng; TLS VPN thường cấp truy cập theo ứng dụng. **Nhà thầu cần vào cả mạng nội bộ hay chỉ vài ứng dụng — và cấp rộng hơn mức cần thì rủi ro gì?**",
    more: "Thử thêm: thiết bị của nhà thầu không do trường quản lý thì cần kiểm tra gì trước khi cho kết nối?",
    citations: [{ title: "Chương 1 — Tổng quan an ninh mạng và mô hình đe doạ", locator: "trang 20–22", href: "/library" }],
    taReply: "Em đối chiếu với nguyên tắc quyền tối thiểu ở chương 1.",
  },
  {
    id: "L1",
    topic: "Kỹ thuật lừa đảo",
    keywords: ["lua dao", "phishing", "thu gia", "gia danh", "ten mien", "duong link"],
    hint: "Thư lừa đảo nhắm mục tiêu thường đúng ngữ cảnh và tạo áp lực thời gian. **Trước khi bấm link, bạn kiểm tra được gì từ địa chỉ người gửi và tên miền thật của đường link?**",
    more: "Thử thêm: nếu lỡ nhập mật khẩu vào trang giả, ba việc đầu tiên cần làm là gì?",
    citations: [{ title: "Chương 2 — Tấn công mạng phổ biến", locator: "trang 26–28", href: "/library" }, { title: "Modern Network Security Threats", locator: "mục 1.3", href: "/library" }],
    taReply: "Em thử rê chuột lên link trong thư mẫu ở lab 1 xem tên miền thật.",
  },
];

export const AI_FALLBACK_TEXT =
  "Mình chưa đủ chắc chắn để gợi ý câu này từ tài liệu của lớp. Mình đã báo giảng viên và trợ giảng; câu trả lời sẽ hiện ngay trong thread này.";

/** Bỏ dấu, `đ` → `d`, chữ thường (SRS 4.3.1 D.1). */
export function normalizeVi(s: string): string {
  return s.toLowerCase().normalize("NFD").replace(/[\u0300-\u036f]/g, "").replace(/đ/g, "d");
}

function escapeRe(s: string) {
  return s.replace(/[.*+?^${}()|[\]\\-]/g, "\\$&");
}

/** Số từ khoá của mẫu khớp trong văn bản đã chuẩn hoá (khớp nguyên từ / cụm từ, có ranh giới hai bên). */
export function keywordScore(t: AiTemplate, normalized: string): number {
  return t.keywords.filter((k) => new RegExp(`(?:^|[^a-z0-9])${escapeRe(k)}(?![a-z0-9])`).test(normalized)).length;
}

/** Quy tắc chọn mẫu (D.2–D.3): điểm cao nhất; hoà → mẫu cùng chủ đề của thread; vẫn hoà → mẫu đứng trước. Không mẫu nào ≥ 1 điểm → null. */
export function pickTemplate(text: string, topic?: string): AiTemplate | null {
  const n = normalizeVi(text);
  let best: AiTemplate | null = null;
  let bestScore = 0;
  for (const t of AI_TEMPLATES) {
    const sc = keywordScore(t, n);
    if (sc === 0) continue;
    if (sc > bestScore || (sc === bestScore && best && t.topic === topic && best.topic !== topic)) {
      best = t;
      bestScore = sc;
    }
  }
  return best;
}

export type AiPlan = { kind: "answer" | "hint" | "fallback"; body: string; citations: Citation[]; template?: string; state: "pending" | "waiting" };

/** D: AI trả lời một văn bản hỏi (thread mới hoặc phản hồi có chữ). */
export function planAnswer(text: string, topic?: string): AiPlan {
  const t = pickTemplate(text, topic);
  if (!t) return { kind: "fallback", body: AI_FALLBACK_TEXT, citations: [], state: "waiting" };
  return { kind: "answer", body: t.hint, citations: t.citations, template: t.id, state: "pending" };
}

/** F: `Hỏi trợ lý AI` khi ô soạn trống và thread đã có câu AI → "Gợi ý thêm" của mẫu khớp câu hỏi gốc. */
export function planHint(question: string, topic?: string): AiPlan {
  const t = pickTemplate(question, topic);
  if (!t) return { kind: "fallback", body: AI_FALLBACK_TEXT, citations: [], state: "waiting" };
  return { kind: "hint", body: t.more, citations: t.citations, template: t.id, state: "pending" };
}

/** G: nội dung phản hồi trễ của trợ giảng. Chủ đề không có mẫu → câu chung có số tuần. */
export function taReplyFor(question: string, topic: string, week: number): string {
  const t = pickTemplate(question, topic) ?? AI_TEMPLATES.find((x) => x.topic === topic);
  if (t) return t.taReply;
  return `Cảm ơn em, anh đã ghi nhận. Thầy cô sẽ trả lời chi tiết trong buổi học tới; em xem trước tài liệu tuần ${week} nhé.`;
}

// ---- dòng thời gian một câu trả lời AI mới (SRS 4.3.1 E) ---------------------------------------------------------------

export const AI_TYPING_MS = 1200;
export const AI_SOURCES_MS = 300;
/** 2,5–3,5 s tuỳ độ dài, tất định theo nội dung. */
export function aiStreamMs(body: string) {
  return 2500 + (body.length % 11) * 100;
}

// ---- seed ------------------------------------------------------------------------------------------------------------------

const sv = (n: number) => ({ authorId: `sv-${n}`, author: STUDENTS[n - 1].name, role: "student" as const });
const TA = { authorId: STAFF.ta.id, author: STAFF.ta.name, role: "ta" as const };
const GV = { authorId: STAFF.teacher.id, author: `TS. ${STAFF.teacher.name}`, role: "teacher" as const };
const B = { authorId: "sv-2", author: STUDENT_B.name, role: "student" as const };
const AI = { authorId: "ai", author: AI_NAME, role: "ai" as const };
const H = (h: number) => h * 60;

const tpl = (id: string) => AI_TEMPLATES.find((t) => t.id === id)!;

function aiMain(threadId: string, id: string, minsAgo: number, t: AiTemplate | null, state: AiState, extra?: { body?: string; citations?: Citation[]; verifiedBy?: string }): ThreadPost {
  return {
    id,
    threadId,
    ...AI,
    body: extra?.body ?? t!.hint,
    minsAgo,
    ai: {
      kind: "answer",
      main: true,
      state,
      verifiedBy: state === "verified" ? (extra?.verifiedBy ?? STAFF.teacher.name) : undefined,
      citations: extra?.citations ?? t!.citations,
      template: t?.id,
    },
  };
}

export const SEED_THREADS: SeedThread[] = [
  {
    id: "t-pin-rubric",
    courseId: COURSE_1,
    title: "Đọc trước khi hỏi: rubric Bài tập 03 và cách trích nguồn",
    topic: "Thông báo",
    week: 8,
    pinned: true,
    posts: [
      {
        id: "p1",
        threadId: "t-pin-rubric",
        ...GV,
        body: "Bài tập 03 chấm theo 4 tiêu chí, mỗi tiêu chí 2,5 điểm: tác nhân và bề mặt tấn công, phân tích tấn công, đánh giá tác động, biện pháp phòng thủ. Mọi số liệu trích từ báo cáo công bố phải ghi rõ nguồn và ngày truy cập.",
        minsAgo: H(26),
      },
      { id: "p2", threadId: "t-pin-rubric", ...sv(12), body: "Trích nguồn từ slide thì ghi số trang slide được không ạ?", minsAgo: H(25) },
      { id: "p3", threadId: "t-pin-rubric", ...TA, body: "Được, ghi 'Chương 3, slide 14'; nguồn web thì thêm ngày truy cập.", minsAgo: H(24), quoteOf: "p2" },
      { id: "p4", threadId: "t-pin-rubric", ...sv(20), body: "Nộp muộn có bị trừ trong rubric không ạ?", minsAgo: H(20) },
      {
        id: "p5",
        threadId: "t-pin-rubric",
        ...GV,
        body: "Trừ 0,5 điểm mỗi ngày, tối đa 2 ngày, tính sau khi chấm rubric — xem quy chế môn học trang 2.",
        minsAgo: H(19),
        quoteOf: "p4",
      },
    ],
  },
  {
    id: "t-pin-lab",
    courseId: COURSE_1,
    title: "Hướng dẫn cài phòng lab ảo cho bài thực hành bắt gói tin",
    topic: "Thực hành",
    week: 9,
    pinned: true,
    posts: [
      {
        id: "p1",
        threadId: "t-pin-lab",
        ...TA,
        body: "Dùng máy ảo Ubuntu 24.04, cài Wireshark và tcpdump. Bắt gói trên card mạng nội bộ của máy ảo, không bắt trên mạng trường.",
        minsAgo: H(20),
      },
      { id: "p2", threadId: "t-pin-lab", ...sv(9), body: "Máy em là Windows ARM, VirtualBox không cài được ạ.", minsAgo: H(19) },
      {
        id: "p3",
        threadId: "t-pin-lab",
        ...TA,
        body: "Dùng UTM bản ARM; ảnh máy ảo ARM ở mục Thực hành trong Thư viện. Vẫn lỗi thì em mang máy lên P.302 giờ thực hành.",
        minsAgo: H(17),
        quoteOf: "p2",
      },
      { id: "p4", threadId: "t-pin-lab", ...sv(14), body: "Wireshark trong máy ảo không thấy gói của máy thật ạ?", minsAgo: H(15) },
    ],
  },
  {
    id: "t-cbc",
    courseId: COURSE_1,
    title: "CBC khác ECB ở điểm nào?",
    topic: "Mật mã đối xứng",
    week: 10,
    posts: [
      {
        id: "p1",
        threadId: "t-cbc",
        ...sv(7),
        body: "Em đọc slide chương 3 thấy AES có nhiều chế độ. Em vẫn chưa hiểu vì sao ảnh mã hoá bằng ECB lại còn nhìn ra hình, còn CBC thì không. Hai chế độ khác nhau ở bước nào ạ?",
        minsAgo: 95,
      },
      aiMain("t-cbc", "p2", 93, tpl("S1"), "pending"),
      { id: "p3", threadId: "t-cbc", ...sv(10), body: "Vậy IV có cần giữ bí mật không ạ? Trong bài lab em thấy IV được gửi kèm bản mã.", minsAgo: 80 },
      {
        id: "p4",
        threadId: "t-cbc",
        ...TA,
        body: "IV không cần bí mật nhưng phải không đoán trước được và không dùng lại với cùng khoá. Em xem ví dụ trang 16: nếu IV cố định thì CBC lộ ra điều gì giống ECB?",
        minsAgo: 70,
        quoteOf: "p3",
      },
      {
        id: "p5",
        threadId: "t-cbc",
        ...sv(7),
        body: "Em hiểu rồi: IV cố định thì hai thông điệp có phần đầu giống nhau sẽ cho khối bản mã đầu giống nhau. Em cảm ơn anh Bảo.",
        minsAgo: 52,
      },
      { id: "p6", threadId: "t-cbc", ...sv(19), body: "Còn CTR thì sao ạ, có cần IV ngẫu nhiên giống CBC không ạ?", minsAgo: 31 },
    ],
  },
  {
    id: "t-salt",
    courseId: COURSE_1,
    title: "Vì sao cần muối (salt) khi băm mật khẩu?",
    topic: "Hàm băm và chữ ký số",
    week: 9,
    posts: [
      { id: "p1", threadId: "t-salt", ...B, body: "Nếu SHA-256 đã khó đảo ngược thì vì sao còn phải thêm muối cho mỗi mật khẩu ạ?", minsAgo: H(31) },
      aiMain("t-salt", "p2", H(30), null, "verified", {
        body:
          "Muối là chuỗi ngẫu nhiên riêng cho từng người dùng, ghép vào mật khẩu trước khi băm. Nó làm hai người đặt trùng mật khẩu vẫn có giá trị băm khác nhau và khiến bảng tra sẵn (rainbow table) vô dụng vì kẻ tấn công phải dựng bảng riêng cho từng muối. Với mật khẩu nên dùng hàm băm chậm có muối như bcrypt hoặc Argon2 thay cho SHA-256 trần.",
        citations: [{ title: "Chương 4 — Hàm băm và chữ ký số", locator: "trang 8", href: "/library" }],
      }),
      { id: "p3", threadId: "t-salt", ...sv(11), body: "Muối lưu ở đâu ạ? Nếu lưu cùng cơ sở dữ liệu thì kẻ tấn công lấy được luôn?", minsAgo: H(29) },
      {
        id: "p4",
        threadId: "t-salt",
        ...GV,
        body: "Muối lưu ngay cạnh giá trị băm, không cần bí mật. Mục đích là buộc kẻ tấn công phải tính lại cho từng người. Câu hỏi cho cả lớp: vậy vì sao bcrypt còn cố tình chậm?",
        minsAgo: H(27),
        quoteOf: "p3",
      },
      { id: "p5", threadId: "t-salt", ...B, body: "Em nghĩ là để mỗi lần đoán thử tốn thời gian hơn, dò mật khẩu hàng loạt sẽ rất lâu ạ.", minsAgo: H(26) },
    ],
  },
  {
    id: "t-rsa-key",
    courseId: COURSE_1,
    title: "Khoá RSA 2048 bit còn đủ an toàn tới khi nào?",
    topic: "Mật mã khoá công khai",
    week: 10,
    posts: [
      {
        id: "p1",
        threadId: "t-rsa-key",
        ...sv(11),
        body: "Thầy có nói 2048 bit vẫn dùng được. Vậy khi nào thì phải chuyển sang 3072 bit hoặc đường cong elliptic ạ?",
        minsAgo: H(5),
      },
      aiMain("t-rsa-key", "p2", H(5) - 5, tpl("K1"), "verified"),
      { id: "p3", threadId: "t-rsa-key", ...sv(23), body: "Có bảng khuyến nghị theo năm không ạ?", minsAgo: H(3) },
      {
        id: "p4",
        threadId: "t-rsa-key",
        ...TA,
        body: "Có, bảng độ dài khoá ở Chương 5 trang 5. Em đọc rồi tự trả lời câu AI hỏi nhé.",
        minsAgo: H(2),
        quoteOf: "p3",
      },
    ],
  },
  {
    id: "t-sqli",
    courseId: COURSE_1,
    title: "Câu lệnh tham số hoá có chặn được mọi dạng SQL injection không?",
    topic: "An toàn ứng dụng web",
    week: 8,
    posts: [
      {
        id: "p1",
        threadId: "t-sqli",
        ...sv(15),
        body: "Em thấy tài liệu nói dùng prepared statement là xong, nhưng có bài viết nói vẫn bị injection ở phần ORDER BY. Thực tế thế nào ạ?",
        minsAgo: H(50),
      },
      aiMain("t-sqli", "p2", 2 * 1440, tpl("W1"), "verified"),
      { id: "p3", threadId: "t-sqli", ...sv(15), body: "Vậy ORDER BY theo cột người dùng chọn thì em phải làm sao ạ?", minsAgo: 2 * 1440 - 60 },
      {
        id: "p4",
        threadId: "t-sqli",
        ...TA,
        body: "Không tham số hoá được tên cột. Em so với một danh sách cột cho phép, không khớp thì dùng cột mặc định.",
        minsAgo: H(40),
        quoteOf: "p3",
      },
      { id: "p5", threadId: "t-sqli", ...sv(26), body: "Bọn em làm danh sách cho phép trong bài tập 02 rồi, chạy ổn ạ.", minsAgo: H(30) },
    ],
  },
  {
    id: "t-xss",
    courseId: COURSE_1,
    title: "Phân biệt XSS lưu trữ và XSS phản chiếu trong bài tập 02",
    topic: "An toàn ứng dụng web",
    week: 7,
    posts: [
      { id: "p1", threadId: "t-xss", ...sv(19), body: "Trong ví dụ ô bình luận của bài tập 02 thì tính là XSS lưu trữ hay phản chiếu ạ?", minsAgo: H(72) },
      aiMain("t-xss", "p2", H(72) - 20, tpl("W2"), "verified"),
      {
        id: "p3",
        threadId: "t-xss",
        ...sv(5),
        body: "Ô bình luận lưu vào cơ sở dữ liệu rồi hiện cho mọi người, vậy là XSS lưu trữ ạ?",
        minsAgo: H(60),
      },
      { id: "p4", threadId: "t-xss", ...TA, body: "Đúng. Còn ô tìm kiếm in lại từ khoá trên URL là loại kia.", minsAgo: H(58), quoteOf: "p3" },
    ],
  },
  {
    id: "t-firewall",
    courseId: COURSE_1,
    title: "Tường lửa trạng thái xử lý gói UDP thế nào?",
    topic: "Tường lửa và phân đoạn mạng",
    week: 6,
    posts: [
      {
        id: "p1",
        threadId: "t-firewall",
        ...sv(22),
        body: "UDP không có bắt tay ba bước thì tường lửa trạng thái dựa vào đâu để biết gói trả về là hợp lệ ạ?",
        minsAgo: H(96),
      },
      { id: "p2", threadId: "t-firewall", ...sv(26), body: "Em nghĩ nó nhớ bộ địa chỉ, cổng, giao thức và một thời gian chờ.", minsAgo: H(90) },
      { id: "p3", threadId: "t-firewall", ...sv(21), body: "Hết thời gian chờ thì gói trả lời bị chặn ạ?", minsAgo: H(84) },
    ],
  },
  {
    id: "t-vpn",
    courseId: COURSE_1,
    title: "IPsec và TLS VPN: chọn cái nào cho nhà thầu truy cập từ xa?",
    topic: "VPN",
    week: 6,
    posts: [
      {
        id: "p1",
        threadId: "t-vpn",
        ...sv(25),
        body: "Tình huống trong slide là nhà thầu bên ngoài cần vào một máy chủ nội bộ. Nên dùng IPsec site-to-site hay TLS VPN ạ?",
        minsAgo: H(110),
      },
      aiMain("t-vpn", "p2", H(110) - 20, tpl("V1"), "verified"),
      { id: "p3", threadId: "t-vpn", ...sv(25), body: "Nhà thầu chỉ cần vào một ứng dụng web nội bộ thôi ạ.", minsAgo: H(100) },
      { id: "p4", threadId: "t-vpn", ...GV, body: "Vậy TLS VPN theo ứng dụng là hợp lý hơn, cấp quyền hẹp.", minsAgo: H(98), quoteOf: "p3" },
    ],
  },
  {
    id: "t-pki",
    courseId: COURSE_1,
    title: "Chuỗi chứng thư số bị lỗi 'unable to get local issuer certificate'",
    topic: "PKI",
    week: 5,
    posts: [
      {
        id: "p1",
        threadId: "t-pki",
        ...sv(28),
        body: "Em dựng máy chủ thử nghiệm, trình duyệt báo lỗi chuỗi chứng thư. Thiếu chứng thư trung gian thì sửa ở đâu ạ?",
        minsAgo: H(130),
      },
      aiMain("t-pki", "p2", H(130) - 20, tpl("P1"), "verified"),
      { id: "p3", threadId: "t-pki", ...sv(28), body: "Em thêm chứng thư trung gian vào file fullchain là hết lỗi ạ.", minsAgo: H(120) },
      { id: "p4", threadId: "t-pki", ...TA, body: "Tốt. Ghi lại thứ tự file trong báo cáo lab cho cả nhóm nhé.", minsAgo: H(118) },
    ],
  },
  {
    id: "t-wifi",
    courseId: COURSE_1,
    title: "WPA3 khắc phục điểm yếu nào của WPA2?",
    topic: "An toàn mạng không dây",
    week: 4,
    posts: [
      { id: "p1", threadId: "t-wifi", ...sv(30), body: "Tấn công KRACK có còn hiệu lực với WPA3 không ạ?", minsAgo: H(170) },
      {
        id: "p2",
        threadId: "t-wifi",
        ...sv(17),
        body: "WPA3 dùng SAE thay cho bắt tay 4 bước, nhưng em chưa chắc có chặn được KRACK không.",
        minsAgo: H(150),
      },
      { id: "p3", threadId: "t-wifi", ...sv(25), body: "Mình cũng thắc mắc, chờ thầy cô ạ.", minsAgo: H(140) },
    ],
  },
  {
    id: "t-phishing",
    courseId: COURSE_1,
    title: "Dấu hiệu nhận biết thư lừa đảo nhắm mục tiêu",
    topic: "Kỹ thuật lừa đảo",
    week: 3,
    posts: [
      {
        id: "p1",
        threadId: "t-phishing",
        ...sv(32),
        body: "Thư giả danh phòng đào tạo gửi link đổi mật khẩu thì nên kiểm tra những gì trước khi bấm ạ?",
        minsAgo: H(220),
      },
      aiMain("t-phishing", "p2", H(220) - 20, tpl("L1"), "verified"),
      { id: "p3", threadId: "t-phishing", ...sv(13), body: "Thư gửi từ đúng tên miền trường thì sao ạ?", minsAgo: H(200) },
      {
        id: "p4",
        threadId: "t-phishing",
        ...GV,
        body: "Tên hiển thị giả được dễ; hãy xem tên miền thật của đường link. Phòng đào tạo không bao giờ xin mật khẩu qua thư.",
        minsAgo: H(198),
        quoteOf: "p3",
      },
    ],
  },
];

// ---- trạng thái phiên (lưu trong `ep_demo_state`, khoá "threads.live") ------------------------------------------------

export type LiveThread = {
  id: string;
  courseId: string;
  title: string;
  topic: string;
  week: number;
  body: string;
  ms: number;
  askAi: boolean;
  authorId: string;
  author: string;
  role: PostRole;
};

export type Moderation = { state: "verified" | "corrected" | "removed"; by: string; edited?: string };
/** Phản hồi trễ của trợ giảng, hẹn theo mốc thời gian thật để sống qua chuyển trang và tải lại (G). */
export type DueReply = { id: string; threadId: string; quoteOf: string; sentMs: number; studentId: string; fired?: boolean };
export type ThreadDraft = { text: string; quoteOf?: string };
export type NewThreadForm = { title: string; topic: string; content: string; askAi: boolean };

export type ThreadsLive = {
  threads: LiveThread[];
  /** bài thêm trong phiên, mọi thread (kể cả thread seed) */
  posts: ThreadPost[];
  /** `${threadId}:${postId}` → kiểm duyệt câu trả lời AI */
  mods: Record<string, Moderation>;
  /** `${threadId}:${postId}` → nội dung sau khi người viết tự sửa */
  edits: Record<string, string>;
  /** `${threadId}:${postId}` đã xoá */
  removed: string[];
  due: DueReply[];
  drafts: Record<string, ThreadDraft>;
  form?: NewThreadForm;
  /** dòng báo một lần ở màn thread kế tiếp, ví dụ "Đã ẩn 2 thông tin cá nhân" (không chứa giá trị gốc) */
  flash?: string;
};

export const THREADS_LIVE_KEY = "threads.live";
export const THREADS_LIVE_SEED: ThreadsLive = { threads: [], posts: [], mods: {}, edits: {}, removed: [], due: [], drafts: {} };

/** Phản hồi trễ: dòng "đang trả lời…" sau 2 s, phản hồi xuất hiện sau 6 s. */
export const TA_TYPING_AFTER_MS = 2000;
export const TA_REPLY_AFTER_MS = 6000;

// ---- dựng ThreadView -------------------------------------------------------------------------------------------------------

export type ListLabel = "verified" | "corrected" | "pending" | "waiting" | "none" | "staff";

export type ThreadView = {
  id: string;
  courseId: string;
  title: string;
  topic: string;
  week: number;
  pinned: boolean;
  origin: "seed" | "live" | "insight";
  question: ThreadPost;
  /** câu trả lời AI chính (đã áp kiểm duyệt); null nếu chưa có hoặc đã bị loại */
  mainAi: ThreadPost | null;
  /** thảo luận theo thời gian tăng dần: phản hồi của người + câu AI sinh từ `Hỏi trợ lý AI` */
  discussion: ThreadPost[];
  participants: number;
  /** mốc giả lập (ms) của bài mới nhất */
  lastMs: number;
  label: ListLabel;
};

const key = (threadId: string, postId: string) => `${threadId}:${postId}`;

export type InsightThreadLike = { id: string; courseId: string; title: string; topic: string; body: string };


/** Số phút trước của một bài tại `nowMs` (giờ giả lập). Bài seed có mốc = 09:20 29/10 trừ `minsAgo`; bài tạo trong phiên mang `ms` giả lập. */
export function postAgeMin(p: ThreadPost, nowMs: number): number {
  const at = p.ms ?? NOW.getTime() - (p.minsAgo ?? 0) * 60000;
  return Math.max(0, (nowMs - at) / 60000);
}

/** Mốc giả lập (ms) của một bài. */
export const postMs = (p: ThreadPost) => p.ms ?? NOW.getTime() - (p.minsAgo ?? 0) * 60000;

/** Nhãn thời gian của một bài: cùng quy tắc N6 với phần còn lại của app. */
export const postAgo = (p: ThreadPost, nowMs: number) => agoLabel(postMs(p), nowMs);

function applyMod(p: ThreadPost, live: ThreadsLive): ThreadPost | null {
  if (live.removed.includes(key(p.threadId, p.id))) return null;
  const edited = live.edits[key(p.threadId, p.id)];
  let out = edited !== undefined ? { ...p, body: edited, edited: true } : p;
  if (out.ai) {
    const m = live.mods[key(p.threadId, p.id)];
    if (m?.state === "removed") return null;
    if (m) {
      out = {
        ...out,
        body: m.state === "corrected" ? (m.edited ?? out.body) : out.body,
        ai: { ...out.ai, state: m.state, verifiedBy: m.by, originalBody: m.state === "corrected" ? out.body : out.ai.originalBody },
      };
    }
  }
  return out;
}

function labelOf(ai: ThreadPost | null | undefined, question: ThreadPost): ListLabel {
  if (ai?.ai) return ai.ai.state;
  return question.role === "teacher" || question.role === "ta" ? "staff" : "none";
}

export function buildThread(
  base: { id: string; courseId: string; title: string; topic: string; week: number; pinned?: boolean; origin: ThreadView["origin"]; posts: ThreadPost[] },
  live: ThreadsLive,
  nowMs: number,
): ThreadView {
  const extra = live.posts.filter((p) => p.threadId === base.id);
  const all = [...base.posts, ...extra].map((p) => applyMod(p, live)).filter((p): p is ThreadPost => p !== null);
  const question = all[0];
  const rest = all.slice(1);
  const mainAi = rest.find((p) => p.ai?.main) ?? null;
  const discussion = rest.filter((p) => p !== mainAi).sort((a, b) => postAgeMin(b, nowMs) - postAgeMin(a, nowMs));
  const humans = new Set([question, ...discussion].filter((p) => p.role !== "ai").map((p) => p.authorId));
  const primary = mainAi ?? discussion.find((p) => p.ai && p.ai.kind !== "hint") ?? null;
  return {
    id: base.id,
    courseId: base.courseId,
    title: base.title,
    topic: base.topic,
    week: base.week,
    pinned: Boolean(base.pinned),
    origin: base.origin,
    question,
    mainAi,
    discussion,
    participants: humans.size,
    lastMs: Math.max(...all.map(postMs)),
    label: labelOf(primary, question),
  };
}

export function liveThreadBase(t: LiveThread) {
  const question: ThreadPost = { id: "p1", threadId: t.id, authorId: t.authorId, author: t.author, role: t.role, body: t.body, ms: t.ms };
  return { id: t.id, courseId: t.courseId, title: t.title, topic: t.topic, week: t.week, origin: "live" as const, posts: [question] };
}

export function insightThreadBase(t: InsightThreadLike) {
  const question: ThreadPost = { id: "p1", threadId: t.id, authorId: STAFF.teacher.id, author: `TS. ${STAFF.teacher.name}`, role: "teacher", body: t.body, minsAgo: 0 };
  return { id: t.id, courseId: t.courseId, title: t.title, topic: t.topic, week: 10, pinned: true, origin: "insight" as const, posts: [question] };
}

/** Mọi thread của một lớp (seed + tạo trong phiên + ghim từ Insights). */
export function threadsOf(courseId: string, live: ThreadsLive, insight: InsightThreadLike[], nowMs: number): ThreadView[] {
  return [
    ...insight.filter((t) => t.courseId === courseId).map((t) => buildThread(insightThreadBase(t), live, nowMs)),
    ...live.threads.filter((t) => t.courseId === courseId).map((t) => buildThread(liveThreadBase(t), live, nowMs)),
    ...SEED_THREADS.filter((t) => t.courseId === courseId).map((t) => buildThread({ ...t, origin: "seed" }, live, nowMs)),
  ];
}

export function findThread(id: string, live: ThreadsLive, insight: InsightThreadLike[], nowMs: number): ThreadView | null {
  const lt = live.threads.find((t) => t.id === id);
  if (lt) return buildThread(liveThreadBase(lt), live, nowMs);
  const it = insight.find((t) => t.id === id);
  if (it) return buildThread(insightThreadBase(it), live, nowMs);
  const st = SEED_THREADS.find((t) => t.id === id);
  return st ? buildThread({ ...st, origin: "seed" }, live, nowMs) : null;
}

/** Số phản hồi ở danh sách = số bài trong "Thảo luận (n)" (SRS 4.3.1 A). */
export const replyCount = (t: ThreadView) => t.discussion.length;

// ---- việc của giảng viên / trợ giảng (SRS 4.3.1 H, J3) ---------------------------------------------------------------------

export type ThreadTask = { threadId: string; no: number; title: string; topic: string; courseId: string; label: "Chờ xác nhận" | "Cần giảng viên trả lời"; ms: number };

const isStaffRole = (r: PostRole) => r === "teacher" || r === "ta";
/** `t-new-3` → 3 (số thứ tự thread tạo trong phiên). */
export const threadNo = (id: string) => Number(id.replace("t-new-", ""));

/**
 * Việc "Câu hỏi mới": mỗi thread SV tạo trong phiên là MỘT việc (mới nhất trước), không gộp.
 * Rời khi GV / TA `Xác nhận` / `Lưu và xác nhận` / `Loại` câu AI của thread, hoặc gửi phản hồi trong thread.
 * Nhánh không khớp (AI chờ giảng viên): chỉ phản hồi mới làm rời.
 */
export function threadTasks(live: ThreadsLive, courseIds: string[], nowMs: number): ThreadTask[] {
  const out: ThreadTask[] = [];
  for (const t of live.threads) {
    if (!courseIds.includes(t.courseId)) continue;
    const v = buildThread(liveThreadBase(t), live, nowMs);
    if (v.discussion.some((p) => isStaffRole(p.role) && p.ms !== undefined && p.ms >= t.ms)) continue;
    const main = live.posts.find((p) => p.threadId === t.id && p.ai?.main);
    let label: ThreadTask["label"] | null;
    if (!main || main.ai?.state === "waiting") label = "Cần giảng viên trả lời";
    else label = live.mods[key(t.id, main.id)] ? null : "Chờ xác nhận";
    if (label) out.push({ threadId: t.id, no: threadNo(t.id), title: t.title, topic: t.topic, courseId: t.courseId, label, ms: t.ms });
  }
  return out.sort((a, b) => b.ms - a.ms);
}

export type PendingAi = { threadId: string; postId: string };

/** N10: các bài AI `Chờ xác nhận` chưa bị ẩn trong thread của lớp, trừ thread đã có việc "Câu hỏi mới". */
export function pendingAi(live: ThreadsLive, insight: InsightThreadLike[], courseIds: string[], nowMs: number): PendingAi[] {
  const taken = threadTasks(live, courseIds, nowMs).map((t) => t.threadId);
  return courseIds
    .flatMap((c) => threadsOf(c, live, insight, nowMs))
    .filter((v) => !taken.includes(v.id))
    .flatMap((v) => [v.mainAi, ...v.discussion].filter((p): p is ThreadPost => p?.ai?.state === "pending").map((p) => ({ threadId: v.id, postId: p.id })));
}

/** Mọi thread của lớp có ít nhất một câu AI `Chờ xác nhận` (chip lọc ở /threads, đếm cả thread mới). */
export const hasPendingAi = (v: ThreadView) => [v.mainAi, ...v.discussion].some((p) => p?.ai?.state === "pending");
