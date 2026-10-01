// Hệ thống: quan sát AI, cấu hình LLM, tích hợp, việc của quản trị viên (FLOWS F15–F16).
// Dữ liệu MÔ PHỎNG, sinh tất định để hai lần render giống hệt nhau. Người giữ: US-PROTO-04.
import { COURSE_1, COURSE_2, at } from "./core";

// ---- dải trạng thái + số tổng hợp theo lớp ---------------------------------------------------

/** Dải trạng thái hôm nay (DESIGN §14.22: dải gọn, không thẻ số liệu). */
export const STATUS_STRIP = [
  { label: "Yêu cầu hôm nay", value: "1.284", hint: "từ 00:00 tới 09:20" },
  { label: "Độ trễ p95", value: "3,1 s", hint: "ngưỡng 5 s" },
  { label: "Tỷ lệ lỗi", value: "0,6%", hint: "8 yêu cầu" },
  { label: "Model dự phòng", value: "1,2%", hint: "15 yêu cầu chuyển sang model khác" },
  { label: "Chi phí hôm nay", value: "42.000 đ", hint: "trần ngày 80.000 đ" },
];

/** Giảng viên chỉ thấy số tổng hợp của lớp mình (FLOWS F16). */
export const COURSE_SUMMARY: Record<string, Array<{ label: string; value: string; hint?: string }>> = {
  [COURSE_1]: [
    { label: "Yêu cầu 7 ngày", value: "1.962" },
    { label: "Độ trễ p95", value: "2,9 s" },
    { label: "Tỷ lệ lỗi", value: "0,4%", hint: "7 yêu cầu" },
    { label: "Câu chuyển giảng viên", value: "6", hint: "AI không đủ chắc chắn" },
    { label: "Lần che thông tin cá nhân", value: "9", hint: "không lần nào lọt ra ngoài" },
  ],
  [COURSE_2]: [
    { label: "Yêu cầu 7 ngày", value: "412" },
    { label: "Độ trễ p95", value: "2,6 s" },
    { label: "Tỷ lệ lỗi", value: "0,2%", hint: "1 yêu cầu" },
    { label: "Câu chuyển giảng viên", value: "2", hint: "AI không đủ chắc chắn" },
    { label: "Lần che thông tin cá nhân", value: "3", hint: "không lần nào lọt ra ngoài" },
  ],
};

// ---- bảng yêu cầu ----------------------------------------------------------------------------

export type RequestStatus = "ok" | "fallback" | "error";

export type LlmRequest = {
  id: string;
  at: Date;
  /** tác vụ nghiệp vụ, không phải tên hàm kỹ thuật */
  task: string;
  model: string;
  latencyMs: number;
  /** 0–1; chỉ TA/GV/Admin được thấy (INTEGRATION mục 2 #2) */
  confidence: number;
  /** sự kiện bảo vệ thông tin cá nhân, null nếu không có */
  privacy: string | null;
  status: RequestStatus;
  courseId: string;
  /** nội dung đã che: chỉ nhãn [[SV_n]], không tên thật */
  masked: string;
  tools: string[];
  tokensIn: number;
  tokensOut: number;
};

const TASKS = ["Trả lời chat", "Chấm bài tập", "Trích quy chế", "Sinh câu hỏi", "Tóm tắt tài liệu", "Đánh chỉ mục tài liệu"];
const MODELS: Record<string, string> = {
  "Trả lời chat": "gpt-4o-mini",
  "Chấm bài tập": "gpt-4o",
  "Trích quy chế": "gemini-1.5-pro",
  "Sinh câu hỏi": "gpt-4o-mini",
  "Tóm tắt tài liệu": "qwen2.5-14b-instruct",
  "Đánh chỉ mục tài liệu": "text-embedding-3-small",
};
/** Nội dung đã che và công cụ đã gọi, theo đúng loại tác vụ. */
const BY_TASK: Record<string, { masked: string[]; tools: string[][]; privacy: boolean }> = {
  "Trả lời chat": {
    masked: [
      "[[SV_1]] hỏi: Vì sao CBC cần IV ngẫu nhiên còn ECB thì không cần gì cả ạ?",
      "[[SV_2]] hỏi: Ký số là mã hoá bằng khoá riêng đúng không ạ?",
      "[[SV_3]] hỏi: Em lọc hết dấu nháy đơn rồi mà vẫn bị khai thác, là do đâu ạ?",
      "[[SV_1]] hỏi: Padding PKCS#7 thêm bao nhiêu byte khi dữ liệu vừa đúng 16 byte?",
      "[[SV_4]] hỏi: Luật tường lửa khớp từ trên xuống hay cái cụ thể nhất thắng ạ?",
      "[[SV_2]] hỏi: Thi cuối kỳ có được mang một tờ A4 ghi chú viết tay vào phòng thi không ạ?",
      "[[SV_3]] hỏi: Mã số sinh viên của em là [[MSSV]], em nộp lại bài được không ạ?",
      "[[SV_4]] hỏi: Gói tin đi từ tầng ứng dụng xuống thì mỗi tầng thêm gì vào ạ?",
    ],
    tools: [
      ["tim_tai_lieu(lop=761987)", "lay_doan_lien_quan(k=6)"],
      ["tim_tai_lieu(lop=761988)", "lay_doan_lien_quan(k=4)"],
      ["tim_tai_lieu(lop=761987)", "lay_bai_tap(ma=BT03)"],
    ],
    privacy: true,
  },
  "Chấm bài tập": {
    masked: [
      "Chấm bài nộp của [[SV_2]] theo rubric 4 tiêu chí, mỗi tiêu chí tối đa 2,5 điểm.",
      "Chấm lại tiêu chí 2 của bài nộp [[SV_2]] và nêu lý do cho điểm.",
    ],
    tools: [["lay_rubric(ma=BT03)", "cham_theo_tieu_chi()"]],
    privacy: true,
  },
  "Trích quy chế": {
    masked: ["Trích điều kiện dự thi và cách làm tròn điểm từ tài liệu Quyche.pdf.", "Trích thang điểm quá trình và trọng số từ tài liệu Quyche.pdf."],
    tools: [["doc_tai_lieu(ten=Quyche.pdf)", "trich_muc(muc=dieu-kien-du-thi)"]],
    privacy: false,
  },
  "Sinh câu hỏi": {
    masked: [
      "Sinh 10 câu trắc nghiệm về chế độ mã hoá AES-CBC từ tài liệu của lớp, kèm đáp án và giải thích.",
      "Sinh 8 câu hỏi về hàm băm và chữ ký số, mức độ trung bình.",
    ],
    tools: [["tim_tai_lieu(lop=761987)", "sinh_cau_hoi(so_luong=10)"]],
    privacy: false,
  },
  "Tóm tắt tài liệu": {
    masked: ["Tóm tắt chương 3 tài liệu Mordern_Network_Security_Threats.pdf thành 5 ý chính cho lớp."],
    tools: [["doc_tai_lieu(ten=Mordern_Network_Security_Threats.pdf)", "tom_tat(so_y=5)"]],
    privacy: false,
  },
  "Đánh chỉ mục tài liệu": {
    masked: ["Tính vector cho 24 đoạn của tài liệu QMB12ch6b.pdf."],
    tools: [["cat_doan(kich_thuoc=800)", "tinh_vector()"]],
    privacy: false,
  },
};
const PRIVACY = [null, null, null, null, null, "Đã che 1 mã số sinh viên", "Đã che 1 họ tên", "Đã che 1 địa chỉ email"];

function lcg(seed: number) {
  let s = seed >>> 0;
  return () => {
    s = (s * 1664525 + 1013904223) >>> 0;
    return s / 4294967296;
  };
}

function makeRequests(): LlmRequest[] {
  const rnd = lcg(761987);
  const rows: LlmRequest[] = [];
  for (let i = 0; i < 50; i++) {
    const task = TASKS[Math.floor(rnd() * TASKS.length)];
    const r = rnd();
    const status: RequestStatus = r > 0.97 ? "error" : r > 0.93 ? "fallback" : "ok";
    const embedding = task === "Đánh chỉ mục tài liệu";
    const kind = BY_TASK[task];
    rows.push({
      id: `rq-${(1284 - i).toString().padStart(4, "0")}`,
      at: at(-2 - i * 3 - Math.floor(rnd() * 3)),
      task,
      model: status === "fallback" ? "gpt-4o-mini (dự phòng)" : MODELS[task],
      latencyMs: embedding ? 180 + Math.floor(rnd() * 320) : 900 + Math.floor(rnd() * 3600),
      confidence: embedding ? 1 : Math.round((0.38 + rnd() * 0.6) * 100) / 100,
      privacy: kind.privacy ? PRIVACY[Math.floor(rnd() * PRIVACY.length)] : null,
      status,
      courseId: rnd() > 0.72 ? COURSE_2 : COURSE_1,
      masked: kind.masked[Math.floor(rnd() * kind.masked.length)],
      tools: kind.tools[Math.floor(rnd() * kind.tools.length)],
      tokensIn: 420 + Math.floor(rnd() * 2400),
      tokensOut: 90 + Math.floor(rnd() * 700),
    });
  }
  return rows;
}

/** 50 yêu cầu gần nhất, mới → cũ. */
export const REQUESTS: LlmRequest[] = makeRequests();

export const STATUS_LABEL: Record<RequestStatus, string> = { ok: "Xong", fallback: "Dùng model dự phòng", error: "Lỗi" };

// ---- cấu hình LLM ----------------------------------------------------------------------------

export type Provider = {
  id: string;
  name: string;
  kind: string;
  endpoint: string;
  /** 4 ký tự cuối của khoá; khoá chỉ ghi, không đọc lại được */
  keyTail: string;
  lastCheck: string;
  /** kết quả `Test kết nối` giả lập */
  test: { ok: true; latencyMs: number } | { ok: false; problem: string; recovery: string };
};

export const PROVIDERS: Provider[] = [
  {
    id: "openai",
    name: "OpenAI",
    kind: "API công cộng",
    endpoint: "https://api.openai.com/v1",
    keyTail: "3f9a",
    lastCheck: "08:40 hôm nay",
    test: { ok: true, latencyMs: 412 },
  },
  {
    id: "gemini",
    name: "Gemini",
    kind: "API công cộng",
    endpoint: "https://generativelanguage.googleapis.com/v1",
    keyTail: "a77c",
    lastCheck: "09:05 hôm nay · 3 lần lỗi trong 15 phút",
    test: {
      ok: false,
      problem: "Gemini từ chối khoá API (HTTP 401). Ba yêu cầu gần nhất đều hỏng, các tác vụ đang chạy bằng OpenAI theo chuỗi dự phòng.",
      recovery: "Tạo khoá mới trong Google AI Studio, dán vào ô Khoá API ở hàng này rồi bấm Test kết nối lại. Nếu khoá mới vẫn bị từ chối, kiểm tra hạn mức thanh toán của dự án.",
    },
  },
  {
    id: "local",
    name: "Máy chủ trong trường",
    kind: "Tương thích OpenAI, đặt tại phòng máy chủ của trường",
    endpoint: "http://llm.ptit.local:8000/v1",
    keyTail: "1d04",
    lastCheck: "08:52 hôm nay",
    test: { ok: true, latencyMs: 180 },
  },
];

export type TaskRoute = { id: string; task: string; note: string; model: string; options: string[] };

export const TASK_ROUTES: TaskRoute[] = [
  { id: "chat", task: "Trả lời chat", note: "Câu hỏi của sinh viên, cần nhanh và rẻ", model: "gpt-4o-mini", options: ["gpt-4o-mini", "gpt-4o", "gemini-1.5-flash", "qwen2.5-14b-instruct"] },
  { id: "grade", task: "Chấm bài tập", note: "Chỉ ra bản nháp cho giảng viên duyệt", model: "gpt-4o", options: ["gpt-4o", "gpt-4o-mini", "gemini-1.5-pro"] },
  { id: "extract", task: "Trích quy chế", note: "Đọc tài liệu dài, cần cửa sổ ngữ cảnh lớn", model: "gemini-1.5-pro", options: ["gemini-1.5-pro", "gpt-4o"] },
  { id: "quiz", task: "Sinh câu hỏi", note: "Câu hỏi luyện tập, giảng viên duyệt trước khi dùng", model: "gpt-4o-mini", options: ["gpt-4o-mini", "gpt-4o", "qwen2.5-14b-instruct"] },
  { id: "summary", task: "Tóm tắt tài liệu", note: "Chạy nền, không cần thời gian thực", model: "qwen2.5-14b-instruct", options: ["qwen2.5-14b-instruct", "gpt-4o-mini"] },
];

export const FALLBACK_CHAIN = ["OpenAI", "Gemini", "Máy chủ trong trường"];

export const EMBEDDING = {
  model: "text-embedding-3-small",
  options: ["text-embedding-3-small", "text-embedding-3-large", "bge-m3 (máy chủ trong trường)"],
  dims: 1536,
  chunks: "12.480 đoạn",
  indexedAt: "27/10, 23:10",
};

export const BUDGET = {
  dayUsed: "42.000 đ",
  dayCap: "80.000 đ",
  dayPercent: 53,
  monthUsed: "1.240.000 đ",
  monthCap: "2.000.000 đ",
  monthPercent: 62,
  note: "Chạm 80% thì quản trị viên nhận cảnh báo; chạm 100% thì việc nền xếp hàng chờ, chat chuyển sang model rẻ nhất chứ không tắt.",
};

export const ADVANCED = [
  { term: "Nhiệt độ (chat)", value: "0,2" },
  { term: "Số token tối đa mỗi câu trả lời", value: "1.024" },
  { term: "Hết giờ chờ", value: "30 giây" },
  { term: "Số lần thử lại", value: "2" },
  { term: "Ngưỡng độ tin cậy chuyển giảng viên", value: "0,80" },
];

// ---- tích hợp --------------------------------------------------------------------------------

export type Integration = {
  id: string;
  name: string;
  purpose: string;
  state: "connected" | "unset" | "waiting";
  stateText: string;
  fields: Array<{ term: string; value: string }>;
  actionLabel: string;
  /** kết quả giả lập khi bấm hành động */
  result: { ok: boolean; text: string };
};

export const INTEGRATIONS: Integration[] = [
  {
    id: "mail",
    name: "Thư đi (SMTP)",
    purpose: "Gửi thư mời giảng viên, thông báo điểm và nhắc hạn nộp.",
    state: "connected",
    stateText: "Đã kết nối · kiểm gần nhất 08:55 hôm nay",
    fields: [
      { term: "Máy chủ", value: "smtp.ptit.edu.vn:587 (STARTTLS)" },
      { term: "Địa chỉ gửi", value: "no-reply@edupilot.test" },
      { term: "Thư đã gửi 7 ngày", value: "214 · hỏng 0" },
    ],
    actionLabel: "Gửi thư thử",
    result: { ok: true, text: "Đã gửi thư thử tới nam.dh@edupilot.test lúc 09:20. Nếu 5 phút nữa chưa thấy, kiểm tra hộp thư rác." },
  },
  {
    id: "imap",
    name: "Thư đến (IMAP)",
    purpose: "Nhận bài nộp sinh viên gửi qua email và tự khớp vào đúng bài tập.",
    state: "unset",
    stateText: "Chưa cấu hình",
    fields: [
      { term: "Máy chủ", value: "— chưa điền" },
      { term: "Hộp thư nhận bài", value: "— chưa điền" },
      { term: "Việc đang chờ", value: "Không có" },
    ],
    actionLabel: "Kiểm tra",
    result: {
      ok: false,
      text: "Chưa có máy chủ IMAP để kiểm tra. Điền máy chủ, cổng và tài khoản hộp thư nhận bài, lưu lại rồi kiểm tra lần nữa. Trong lúc chưa cấu hình, sinh viên vẫn nộp bài trực tiếp trên EduPilot được.",
    },
  },
  {
    id: "teams",
    name: "Microsoft Teams",
    purpose: "Đồng bộ lịch buổi học và gửi thông báo vào kênh lớp.",
    state: "waiting",
    stateText: "Chờ quản trị viên của trường đồng ý",
    fields: [
      { term: "Tổ chức", value: "ptit.edu.vn" },
      { term: "Yêu cầu đã gửi", value: "27/10, 14:20" },
      { term: "Người duyệt", value: "Quản trị viên Microsoft 365 của trường" },
    ],
    actionLabel: "Kiểm tra",
    result: {
      ok: false,
      text: "Teams vẫn chờ quản trị viên Microsoft 365 của trường bấm đồng ý cho EduPilot đọc lịch và gửi thông báo. Đây là việc của trường, không làm được từ màn này. Gửi lại yêu cầu nếu quá 5 ngày chưa có hồi âm.",
    },
  },
];

// ---- việc của quản trị viên ("Hôm nay") ------------------------------------------------------

export type AdminTask = { id: string; tone: "red" | "amber" | "blue" | "neutral"; title: string; context: string; meta: string; href: string };

export const ADMIN_TASKS: AdminTask[] = [
  {
    id: "gemini",
    tone: "red",
    title: "Gemini lỗi 3 lần trong 15 phút",
    context: "Khoá API bị từ chối. Tác vụ trích quy chế đang chạy bằng OpenAI theo chuỗi dự phòng nên lớp không bị dừng.",
    meta: "Lỗi gần nhất 09:05",
    href: "/settings/llm",
  },
  {
    id: "budget",
    tone: "amber",
    title: "Ngân sách tháng sắp chạm ngưỡng cảnh báo 80%",
    context: "Đã dùng 1.240.000 đ trên 2.000.000 đ (62%) khi mới qua hai phần ba tháng; theo đà này sẽ chạm 80% vào 04/11.",
    meta: "Cập nhật 09:15",
    href: "/settings/llm",
  },
  {
    id: "dead-letter",
    tone: "amber",
    title: "2 việc nằm trong hàng chờ xử lý lỗi",
    context: "Trích quy chế lớp 761988 hỏng 2 lần liên tiếp lúc Gemini lỗi. Chạy lại được, không mất dữ liệu.",
    meta: "Việc cũ nhất chờ 48 phút",
    href: "/observability",
  },
  {
    id: "no-teacher",
    tone: "neutral",
    title: "Lớp 761988 chưa có giảng viên hoạt động",
    context: "Vừa phân công TS. Lê Thu Hà lúc 08:30, chờ xác nhận lần đăng nhập đầu. Kiểm tra lại trong danh sách lớp.",
    meta: "Phân công 08:30",
    href: "/admin/courses",
  },
];

// ---- quản trị lớp và người dùng --------------------------------------------------------------

/** Giảng viên có thể phân công khi mở lớp mới. */
export const TEACHER_OPTIONS = ["TS. Lê Thu Hà", "ThS. Nguyễn Minh Khôi", "TS. Trần Quốc Việt"];

export const COURSE_ADMIN_META: Record<string, { opened: string; lastActive: string }> = {
  [COURSE_1]: { opened: "20/08/2026", lastActive: "Hôm nay 09:20" },
  [COURSE_2]: { opened: "09/10/2026", lastActive: "Hôm qua 16:40" },
};
