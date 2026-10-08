/**
 * Khung trang vẽ TỪ MÁY CHỦ trước khi có phiên (US-PU-06): tiêu đề + câu phụ của route, để chữ lớn nhất đã có trên màn ngay từ
 * lần vẽ đầu (LCP) thay vì chờ JS → `/auth/refresh` → `/me/courses`. Chỉ route có chữ ổn định theo vai / lớp mới có mục ở đây;
 * route khác giữ khung xương. Câu phụ phụ thuộc lớp dùng chỗ giữ chỗ cùng độ dài ("761987 · An ninh mạng") nên chữ thật thay vào
 * không lớn hơn chữ đã vẽ.
 */
export type RouteIntro = { title: string; description: string; width?: "reading" | "wide" | "full" };

const CLASS_LABEL = "761987 · An ninh mạng";

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
  },
  "/": { title: "Hôm nay", description: "Việc cần xử lý của bạn" },
};

export const introFor = (pathname: string): RouteIntro | null => INTROS[pathname] ?? null;
