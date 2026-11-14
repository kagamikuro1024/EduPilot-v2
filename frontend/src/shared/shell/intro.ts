/**
 * Khung trang vẽ TỪ MÁY CHỦ trước khi có phiên (US-PU-06): tiêu đề + câu phụ của route, để chữ lớn nhất đã có trên màn ngay từ
 * lần vẽ đầu (LCP) thay vì chờ JS → `/auth/refresh` → `/me/courses`. Chỉ route có chữ ổn định theo vai / lớp mới có mục ở đây;
 * route khác giữ khung xương. Không còn câu phụ dưới tiêu đề (FEAT-uiux-91 US-01). QUY TẮC LCP (AC8): ứng viên LCP đầu tiên phải là khối chữ LỚN NHẤT mà màn sẽ có sau phiên — chữ nào xuất hiện sau
 * phiên mà lớn hơn sẽ thành ứng viên mới và kéo LCP về lúc có phiên. Kiểm bằng e2e `lcp before refresh`.
 */
export type RouteIntro = { title: string; width?: "work" | "read"; body?: string };

export const INBOX_INTRO = {
  title: "Hộp thư hỗ trợ",
} as const;

export const LLM_INTRO = {
  title: "Cấu hình LLM",
} as const;

const INTROS: Record<string, RouteIntro> = {
  "/inbox": INBOX_INTRO,
  "/settings/llm": LLM_INTRO,
  "/threads": { title: "Hỏi đáp công khai" },
  // h1 thật = "Kết quả của tôi" + mã lớp (RealMe); các khối sau phiên nhỏ hơn h1 nên h1 là ứng viên LCP duy nhất
  "/me": { title: "Kết quả của tôi" },
  "/gradebook": { title: "Sổ điểm", width: "work" },
  "/chat": {
    title: "Chat riêng",
    width: "work",
    // lời giới thiệu của phiên chat trống (cùng chữ với ChatScreen): là khối chữ lớn nhất của màn đầu nên phải có sẵn từ máy chủ
    body: "Phiên chat này chỉ bạn đọc được. Bạn có thể hỏi về số buổi đã vắng, điểm cộng phát biểu, hạn nộp bài hoặc nội dung bài giảng. Khi AI chưa đủ chắc chắn, câu hỏi sẽ được chuyển cho giảng viên.",
  },
  // h1 thật là "Chào {tên}" (sinh viên) hoặc "N việc cần xử lý hôm nay" (GV / TA / Admin): chữ máy chủ phải không nhỏ hơn cả hai
  "/": { title: "Việc cần xử lý hôm nay" },
};

export const introFor = (pathname: string): RouteIntro | null => INTROS[pathname] ?? null;
