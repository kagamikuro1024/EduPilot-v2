// Chat riêng của sinh viên (US-PROTO-01, /chat). Câu nhập D1–D3 nguyên văn DEMO_SCRIPT mục 3.
// Bản mô phỏng chỉ có câu trả lời ghi sẵn cho đúng ba câu này.

import { STUDENTS, STUDENT_B, STUDENT_C, fmtScore } from "./core";
import { docCite } from "./docs";
import { D3_TEXT } from "./support";

export const D1_TEXT = `Em là ${STUDENT_B.name}, MSSV ${STUDENT_B.code}. Em đã nghỉ mấy buổi và được cộng bao nhiêu điểm phát biểu rồi ạ?`;
export const D2_TEXT = `Bạn ${STUDENT_C.name} nghỉ mấy buổi rồi ạ?`;
export { D3_TEXT };

export type Script = "d1" | "d2" | "d3" | "other";

/** Nhận ra câu nào trong kịch bản demo vừa được gửi (so khớp nới tay để người trình bày gõ tay vẫn trúng). */
export function matchScript(text: string): Script {
  const t = text.toLowerCase();
  if (t.includes(STUDENT_C.name.toLowerCase()) || t.includes(STUDENT_C.code)) return "d2";
  if (t.includes("a4") || t.includes("phòng thi")) return "d3";
  if (t.includes(STUDENT_B.code) || t.includes("nghỉ mấy buổi") || t.includes("phát biểu")) return "d1";
  return "other";
}

export const NEUTRAL_ANSWER =
  "Bản mô phỏng chỉ có câu trả lời ghi sẵn cho các câu trong kịch bản demo. Bạn thử hỏi về số buổi vắng, điểm cộng phát biểu, hoặc quy định phòng thi nhé.";

export const REFUSAL_ANSWER = "Mình chỉ trả lời được thông tin của chính bạn. Nếu cần trao đổi về bạn khác, hãy hỏi giảng viên.";

export const LOW_CONFIDENCE_ANSWER =
  "AI chưa đủ chắc chắn về câu này. Quy chế môn học không nói rõ về tài liệu được mang vào phòng thi, nên mình đã chuyển câu hỏi cho giảng viên.";

export const QUIZ_ANSWER =
  "Bạn đang làm QUIZ01 nên mình chỉ trả lời câu hỏi thủ tục: bài có 20 phút, bấm Nộp bài khi xong, và kết quả hiện ngay sau khi nộp. Câu hỏi về nội dung bài học bạn hỏi lại sau khi nộp nhé.";

/** Câu trả lời D1 dựng từ số liệu thật của sổ điểm (vắng, phát biểu, điểm cộng). */
export function answerD1(input: { absences: number; dates: string[]; speaks: number; bonus: number; penalty: number }) {
  return (
    `Bạn đã vắng ${input.absences} buổi (${input.dates.join(", ")}) và được cộng ${fmtScore(input.bonus, 2)} điểm cho ${input.speaks} lần phát biểu. ` +
    "Vắng thêm 1 buổi không phép sẽ bị trừ 0,5 điểm."
  );
}

export type Citation = { title: string; locator: string; href: string };

/** Nguồn tham khảo luôn là tên hiển thị của bảng N5 (SRS 4.8 N5, 01-AC24). */
export const D1_CITATIONS: Citation[] = [docCite("d-quyche-mon", "trang 2 · mục Điểm quá trình")];

/** Giải thích ngắn mở tại chỗ sau khi bấm `Tìm hiểu`. */
export const PII_EXPLAINER =
  "Tên và mã số sinh viên trong câu hỏi được thay bằng ký hiệu trước khi gửi đi. Danh tính của bạn lấy từ phiên đăng nhập, nên câu trả lời vẫn đúng người.";

const NAMES = STUDENTS.slice(0, 4).map((s) => s.name);
const CODE_RE = /\b20\d{6}\b/g;
const GRADE_PHRASE = /điểm của (em|mình|tôi|con)/gi;

/** Thông tin cá nhân trong một đoạn chữ: mã số sinh viên, tên người trong lớp, câu hỏi điểm cá nhân. */
export function scanPersonal(text: string): { count: number; reasons: string[] } {
  const reasons: string[] = [];
  const codes = text.match(CODE_RE) ?? [];
  if (codes.length) reasons.push(`mã số sinh viên ${codes.join(", ")}`);
  const names = NAMES.filter((n) => text.includes(n));
  if (names.length) reasons.push(`họ tên ${names.join(", ")}`);
  const grade = GRADE_PHRASE.test(text);
  GRADE_PHRASE.lastIndex = 0;
  if (grade) reasons.push("điểm cá nhân");
  return { count: codes.length + names.length + (grade ? 1 : 0), reasons };
}

/** Bản đã thay thông tin nhận dạng bằng "[đã ẩn]" để đăng công khai. */
export function redact(text: string) {
  let out = text.replace(CODE_RE, "[đã ẩn]");
  for (const n of NAMES) out = out.split(n).join("[đã ẩn]");
  return out;
}

// ---- lịch sử phiên chat ---------------------------------------------------------------------

export type ChatSession = {
  id: string;
  /** phiên thuộc về một sinh viên: bạn khác mở /chat thấy "Chưa có phiên nào" (E2) */
  studentId: string;
  title: string;
  at: Date;
  msgs: number;
  q: string;
  a: string;
};

export const CHAT_SESSIONS: ChatSession[] = [
  {
    id: "cs-1",
    studentId: STUDENT_B.id,
    title: "Cách chọn độ dài khoá RSA",
    at: new Date("2026-10-28T20:10:00+07:00"),
    msgs: 6,
    q: "Khoá RSA 2048 bit với 3072 bit khác nhau nhiều không ạ?",
    a: "2048 bit vẫn được khuyến nghị cho dữ liệu dùng trong vài năm tới; 3072 bit dành cho dữ liệu cần bảo vệ lâu hơn, đổi lại ký và giải mã chậm hơn khoảng 3 lần. Với bài thực hành của môn, bạn dùng 2048 bit là đủ.",
  },
  {
    id: "cs-2",
    studentId: STUDENT_B.id,
    title: "Nộp muộn Bài tập 03 bị trừ bao nhiêu?",
    at: new Date("2026-10-26T15:30:00+07:00"),
    msgs: 4,
    q: "Em nộp Bài tập 03 muộn một ngày thì bị trừ bao nhiêu điểm ạ?",
    a: "Bài tập 03 cho nộp muộn tối đa 2 ngày, mỗi ngày trừ 0,5 điểm vào tổng điểm bài. Nộp muộn 1 ngày thì bị trừ 0,5 điểm.",
  },
  {
    id: "cs-3",
    studentId: STUDENT_B.id,
    title: "Hàm băm SHA-256 dùng ở đâu trong chữ ký số",
    at: new Date("2026-10-22T09:45:00+07:00"),
    msgs: 9,
    q: "Trong chữ ký số thì băm SHA-256 nằm ở bước nào ạ?",
    a: "Người ký băm văn bản bằng SHA-256 rồi mã hoá giá trị băm đó bằng khoá riêng. Người nhận băm lại văn bản và so với giá trị giải mã từ chữ ký — trùng nghĩa là văn bản chưa bị sửa.",
  },
  {
    id: "cs-4",
    studentId: STUDENT_B.id,
    title: "Phân biệt IDS và IPS",
    at: new Date("2026-10-21T16:05:00+07:00"),
    msgs: 5,
    q: "IDS và IPS khác nhau thế nào ạ?",
    a: "IDS chỉ phát hiện và cảnh báo, đặt song song với luồng mạng. IPS nằm trực tiếp trên đường đi của gói tin nên chặn được tấn công, đổi lại một luật sai có thể cắt nhầm lưu lượng hợp lệ.",
  },
];

export function sessionsFor(studentId: string): ChatSession[] {
  return CHAT_SESSIONS.filter((c) => c.studentId === studentId);
}
