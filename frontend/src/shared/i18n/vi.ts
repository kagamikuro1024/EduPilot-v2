// Lời tiếng Việt cho lỗi của gateway (SRS FEAT-ui-foundation 6.2). Không có từ kỹ thuật của AI: sinh viên có thể đọc được mọi câu.
// Bảng dịch thuật ngữ AI → lời thường (DESIGN.md §13) cũng sống ở đây.

/** Câu chung cho mã lạ và cho INTERNAL. */
export const GENERIC_ERROR = "Có lỗi xảy ra. Dữ liệu của bạn vẫn an toàn. Thử lại sau ít phút.";

/** Mã lỗi → lời. `{n}` được thay bằng số giây `retry_after` (không có → "ít giây"). */
export const ERROR_MESSAGES: Record<string, string> = {
  BAD_REQUEST: "Yêu cầu chưa đúng. Hãy tải lại trang và thử lại.",
  UNAUTHENTICATED: "Bạn cần đăng nhập để tiếp tục.",
  TOKEN_EXPIRED: "Phiên đăng nhập đã hết hạn. Vui lòng đăng nhập lại.",
  TOKEN_INVALID: "Phiên đăng nhập không hợp lệ.",
  FORBIDDEN: "Bạn không có quyền thực hiện việc này.",
  NOT_FOUND: "Không tìm thấy nội dung này.",
  METHOD_NOT_ALLOWED: "Thao tác này không được hỗ trợ.",
  CONFLICT: "Dữ liệu vừa thay đổi. Hãy tải lại và thử lại.",
  VERSION_CONFLICT: "Có người vừa sửa mục này. Chọn giữ bản nào.",
  IDEMPOTENCY_IN_PROGRESS: "Thao tác trước đang xử lý. Chờ vài giây.",
  PAYLOAD_TOO_LARGE: "Nội dung quá lớn. Rút gọn rồi gửi lại.",
  UNSUPPORTED_MEDIA_TYPE: "Định dạng gửi lên chưa được hỗ trợ.",
  VALIDATION_FAILED: "Một số thông tin chưa hợp lệ. Kiểm tra các ô được đánh dấu.",
  INVALID_CURSOR: "Danh sách đã đổi. Đang tải lại từ đầu.",
  IDEMPOTENCY_KEY_REUSED: "Thao tác này đã gửi với nội dung khác. Hãy làm lại từ đầu.",
  IDEMPOTENCY_KEY_REQUIRED: GENERIC_ERROR,
  RATE_LIMITED: "Bạn thao tác hơi nhanh. Thử lại sau {n}.",
  SSE_LIMIT_REACHED: "Bạn đang mở nhiều cửa sổ; cập nhật tự động tạm dừng.",
  INTERNAL: GENERIC_ERROR,
  SERVICE_UNAVAILABLE: "Hệ thống đang bận. Thử lại sau ít phút.",
  NOT_READY: "Hệ thống đang khởi động lại. Thử lại sau ít giây.",
  DEADLINE_EXCEEDED: "Mất quá lâu để phản hồi. Thử lại sau.",
  OVERLOADED: "Hệ thống đang rất đông. Thử lại sau {n}.",
  LLM_NOT_CONFIGURED: "Trợ lý chưa được cấu hình. Báo quản trị viên.",
  LLM_UNAVAILABLE: "Trợ lý tạm thời không khả dụng.",
  PROVIDER_IN_USE: "Nhà cung cấp này đang được dùng. Chuyển các tác vụ sang nhà khác trước.",
  MODEL_DIMS_MISMATCH: "Mô hình này không dùng được cho tìm kiếm tài liệu (sai số chiều).",
  ROUTE_INVALID: "Cấu hình tác vụ chưa hợp lệ.",
  INVALID_CREDENTIALS: "Email hoặc mật khẩu không đúng.",
  LOGIN_THROTTLED: "Bạn đã thử quá nhiều lần. Thử lại sau {n}.",
  ACCOUNT_DISABLED: "Tài khoản đã bị khoá. Hãy liên hệ quản trị viên.",
  SESSION_REVOKED: "Bạn đã bị đăng xuất. Hãy đăng nhập lại.",
  LINK_INVALID: "Liên kết đã hết hạn hoặc đã được dùng.",
  EMAIL_NOT_VERIFIED: "Hãy xác minh email trước khi vào lớp.",
  JOIN_CODE_INVALID: "Mã không hợp lệ hoặc đã hết hạn. Kiểm tra lại với giảng viên.",
  COURSE_FULL: "Lớp đã đủ sĩ số.",
  COURSE_ARCHIVED: "Lớp này đã được lưu trữ.",
  // mã phía client
  BAD_GATEWAY: "Máy chủ chưa phản hồi đúng. Dữ liệu của bạn vẫn an toàn.",
  NETWORK: "Không kết nối được tới máy chủ. Chữ bạn đã nhập vẫn được giữ.",
  ABORTED: "",
  PARSE_ERROR: "Phản hồi từ máy chủ không đọc được. Thử lại sau.",
  BAD_TARGET: "",
};

/** Lời cho một mã; mã lạ → câu chung. `retryAfter` (giây) điền vào "{n}". */
export function userMessageFor(code: string, retryAfter?: number): string {
  const raw = code in ERROR_MESSAGES ? ERROR_MESSAGES[code] : GENERIC_ERROR;
  return raw.replace("{n}", retryAfter && retryAfter > 0 ? `${retryAfter} giây` : "ít giây");
}
