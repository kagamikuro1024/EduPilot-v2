// Lời của màn /settings/llm (FEAT-llm-gateway SRS 7.2–7.3). Từ kỹ thuật của AI không xuất hiện: dùng bảng dịch DESIGN.md §13.

export const TASKS: Array<{ task: string; label: string; lane: "INTERACTIVE" | "NEAR_REALTIME" | "BATCH" }> = [
  { task: "CHAT", label: "Trả lời chat riêng", lane: "INTERACTIVE" },
  { task: "CLASSIFY", label: "Phân loại câu hỏi", lane: "NEAR_REALTIME" },
  { task: "UTILITY", label: "Việc nhỏ (tóm tắt, đặt tên)", lane: "NEAR_REALTIME" },
  { task: "GRADING", label: "Chấm bài", lane: "BATCH" },
  { task: "QUESTION_GEN", label: "Sinh câu hỏi", lane: "BATCH" },
  { task: "INSIGHT", label: "Tóm tắt lớp học", lane: "BATCH" },
];
export const taskLabel = (task: string) => TASKS.find((t) => t.task === task)?.label ?? task;

export const LANE_LABEL: Record<string, string> = { INTERACTIVE: "Trả lời ngay", NEAR_REALTIME: "Gần thời gian thực", BATCH: "Chạy nền" };

export const TYPE_LABEL: Record<string, string> = {
  openai: "OpenAI",
  anthropic: "Anthropic",
  gemini: "Gemini",
  openai_compatible: "Máy chủ riêng (tương thích OpenAI)",
  fake: "Giả lập để thử",
};
export const TYPES = Object.keys(TYPE_LABEL);

/** Gợi ý mô hình khi thêm nhà cung cấp (người dùng sửa được). */
export const DEFAULT_MODELS: Record<string, Array<{ model: string; kind: "chat" | "embedding" }>> = {
  openai: [{ model: "gpt-4o-mini", kind: "chat" }, { model: "gpt-4o", kind: "chat" }, { model: "text-embedding-3-small", kind: "embedding" }],
  anthropic: [{ model: "claude-haiku-4-5", kind: "chat" }],
  gemini: [{ model: "gemini-2.0-flash", kind: "chat" }],
  openai_compatible: [{ model: "", kind: "chat" }],
  fake: [{ model: "fake-chat", kind: "chat" }, { model: "fake-embed", kind: "embedding" }],
};

export type Tone = "green" | "amber" | "red" | "neutral";
type StatusInput = { enabled: boolean; key_status: string; circuit: string; last_test: { ok: boolean | null; at: string | null; error_kind: string | null } };

const hhmm = (iso: string) => new Date(iso).toLocaleTimeString("vi-VN", { hour: "2-digit", minute: "2-digit", hour12: false });

/** Trạng thái nhà cung cấp → chữ + màu (SRS 7.2); thứ tự ưu tiên như bảng. */
export function providerStatus(p: StatusInput): { text: string; tone: Tone } {
  if (!p.enabled) return { text: "Đã tắt", tone: "neutral" };
  if (p.key_status === "unreadable") return { text: "Lỗi khoá — nhập lại khoá", tone: "red" };
  if (p.circuit === "open") return { text: "Tạm dừng do lỗi liên tiếp", tone: "amber" };
  if (p.last_test.ok === false) return { text: p.last_test.error_kind === "AUTH" ? "Lỗi xác thực" : "Kết nối lỗi", tone: "red" };
  if (p.last_test.ok === true) return { text: `Đã kết nối${p.last_test.at ? ` · kiểm tra lúc ${hhmm(p.last_test.at)}` : ""}`, tone: "green" };
  return { text: "Chưa kiểm tra", tone: "neutral" };
}

export const TEST_MESSAGE: Record<string, string> = {
  AUTH: "Khoá API không được nhà cung cấp chấp nhận. Kiểm tra lại khoá.",
  RATE_LIMIT: "Nhà cung cấp đang giới hạn tốc độ. Chờ một lúc rồi thử lại.",
  UNREACHABLE: "Không với tới máy chủ của nhà cung cấp. Kiểm tra địa chỉ và mạng rồi thử lại.",
  TIMEOUT: "Nhà cung cấp trả lời quá chậm. Thử lại sau.",
  MODEL: "Nhà cung cấp không có mô hình này. Kiểm tra lại tên mô hình.",
};
export const testMessage = (kind?: string | null, server?: string) => server || (kind && TEST_MESSAGE[kind]) || "Test kết nối chưa thành công. Thử lại.";

const vnd = new Intl.NumberFormat("vi-VN", { maximumFractionDigits: 0 });
/** "1240000.0000" → "1.240.000 đ" (chỉ để hiển thị; không tính toán tiền ở giao diện). */
export const fmtVnd = (s: string | null | undefined) => (s == null ? "—" : `${vnd.format(Math.round(Number(s)))} đ`);
export const fmtInt = (n: number) => vnd.format(n);
