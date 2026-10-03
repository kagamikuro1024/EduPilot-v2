// Hệ thống: quan sát AI, cấu hình LLM, tích hợp, việc của quản trị viên (FLOWS F15–F16).
// Dữ liệu MÔ PHỎNG, sinh tất định để hai lần render giống hệt nhau. Người giữ: US-PROTO-04.
import { COURSE_1, COURSE_2, at, fmtTime } from "./core";
import { ASSIGNED_AT, agoLabel } from "./derive";

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
    masked: ["Trích điều kiện dự thi và cách làm tròn điểm từ tài liệu “Quy chế đào tạo của trường”.", "Trích thang điểm quá trình và trọng số từ tài liệu “Quy chế đào tạo của trường”."],
    tools: [["doc_tai_lieu(ten=Quy chế đào tạo của trường)", "trich_muc(muc=dieu-kien-du-thi)"]],
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
    masked: ["Tóm tắt chương 3 tài liệu “Modern Network Security Threats” thành 5 ý chính cho lớp."],
    tools: [["doc_tai_lieu(ten=Modern Network Security Threats)", "tom_tat(so_y=5)"]],
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

/** Việc của quản trị viên ở "Hôm nay"; mốc phân công lớp 761988 lấy từ `ASSIGNED_AT` (SRS 4.8 N6). */
export function adminTasks(nowMs: number): AdminTask[] {
  return [
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
    context: `Vừa phân công TS. Lê Thu Hà ${agoLabel(ASSIGNED_AT, nowMs)}, chờ xác nhận lần đăng nhập đầu. Kiểm tra lại trong danh sách lớp.`,
    meta: `Phân công · ${agoLabel(ASSIGNED_AT, nowMs)}`,
    href: "/admin/courses",
  },
  ];
}

// ---- quản trị lớp và người dùng --------------------------------------------------------------

/** Giảng viên có thể phân công khi mở lớp mới. */
export const TEACHER_OPTIONS = ["TS. Lê Thu Hà", "ThS. Nguyễn Minh Khôi", "TS. Trần Quốc Việt"];

/** Ngày mở lớp cố định; "hoạt động gần nhất" của 761988 chính là lúc phân công (`ASSIGNED_AT`). */
export const COURSE_ADMIN_META: Record<string, { opened: string }> = {
  [COURSE_1]: { opened: "20/08/2026" },
  [COURSE_2]: { opened: "09/10/2026" },
};

export function courseLastActive(courseId: string, nowMs: number): string {
  return courseId === COURSE_2 ? agoLabel(ASSIGNED_AT, nowMs) : `hôm nay ${fmtTime(new Date(nowMs))}`;
}
