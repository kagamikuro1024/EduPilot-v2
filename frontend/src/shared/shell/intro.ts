/**
 * Khung trang vẽ TỪ MÁY CHỦ trước khi có phiên (US-PU-06): tiêu đề + câu phụ của route, để chữ lớn nhất đã có trên màn ngay từ
 * lần vẽ đầu (LCP) thay vì chờ JS → `/auth/refresh` → `/me/courses`. Chỉ route có chữ ổn định theo vai / lớp mới có mục ở đây;
 * route khác giữ khung xương. Câu phụ phụ thuộc lớp dùng chỗ giữ chỗ cùng độ dài ("761987 · An ninh mạng") nên chữ thật thay vào
 * không lớn hơn chữ đã vẽ. QUY TẮC LCP (AC8): ứng viên LCP đầu tiên phải là khối chữ LỚN NHẤT mà màn sẽ có sau phiên — chữ nào xuất hiện sau
 * phiên mà lớn hơn sẽ thành ứng viên mới và kéo LCP về lúc có phiên. Kiểm bằng e2e `lcp before refresh`.
 */
export type RouteIntro = { title: string; description: string; width?: "reading" | "wide" | "full"; body?: string };

const CLASS_LABEL = "An ninh mạng – 761987"; // cùng dạng `course.label` của lớp mẫu

export const INBOX_INTRO = {
  title: "Hộp thư hỗ trợ",
  description: "Câu hỏi được chuyển cho người thật khi AI không đủ chắc chắn hoặc sinh viên yêu cầu.",
} as const;

export const LLM_INTRO = {
  title: "Cấu hình LLM",
  description: "Tác vụ nào chạy bằng mô hình nào, hỏng thì chuyển sang đâu, và tiêu bao nhiêu tiền.",
} as const;

const INTROS: Record<string, RouteIntro> = {
  "/inbox": INBOX_INTRO,
  "/settings/llm": LLM_INTRO,
  "/threads": {
    title: "Threads",
    description: `Câu hỏi công khai của ${CLASS_LABEL}. Câu trả lời có nhãn xác nhận là đã được giảng viên duyệt.`,
  },
  "/gradebook": { title: "Sổ điểm", description: `${CLASS_LABEL} · Thứ Năm 09:00–11:30 · P.302 – G2`, width: "wide" },
  "/chat": {
    title: "Chat riêng",
    description: `Hỏi về điểm, chuyên cần và bài học của chính bạn — ${CLASS_LABEL}. Giảng viên không đọc được phiên chat này.`,
    width: "full",
    // lời giới thiệu của phiên chat trống (cùng chữ với ChatScreen): là khối chữ lớn nhất của màn đầu nên phải có sẵn từ máy chủ
    body: "Phiên chat này chỉ bạn đọc được. Bạn có thể hỏi về số buổi đã vắng, điểm cộng phát biểu, hạn nộp bài hoặc nội dung bài giảng. Khi AI chưa đủ chắc chắn, câu hỏi sẽ được chuyển cho giảng viên.",
  },
  // h1 thật là "Chào {tên}" (sinh viên) hoặc "N việc cần xử lý hôm nay" (GV / TA / Admin): chữ máy chủ phải không nhỏ hơn cả hai
  "/": { title: "Việc cần xử lý hôm nay", description: "Việc của bạn và các buổi học sắp tới hiện ở đây." },
};

export const introFor = (pathname: string): RouteIntro | null => INTROS[pathname] ?? null;
