// Luyện đề của sinh viên (US-PROTO-01: /practice, /practice/[attemptId], /practice/history).
// Câu hỏi theo nội dung An ninh mạng; giải thích luôn kèm nguồn trong thư viện.

import type { Citation } from "./chat";
import { NOW } from "./core";
import { docCite } from "./docs";
import { normalizeVi } from "./threads";
import { ASSIGNMENTS } from "./student";

export type Question = {
  id: string;
  /** trắc nghiệm hoặc trả lời ngắn */
  kind: "choice" | "short";
  text: string;
  options?: string[];
  /** chỉ số đáp án đúng (choice) hoặc từ khoá chấm (short) */
  correct?: number;
  keywords?: string[];
  /** đáp án mẫu của câu trả lời ngắn (hiện ở màn kết quả) */
  answer?: string;
  explain: string;
  source: Citation;
};

// Trích dẫn lấy từ bảng tài liệu duy nhất (SRS 4.8 N5).
const CH3: Citation = docCite("d-ch3", "trang 11–19");
const CH4: Citation = docCite("d-ch4", "trang 5–12");
const CH5: Citation = docCite("d-ch5", "trang 3–9");

export const SYMMETRIC_QUESTIONS: Question[] = [
  {
    id: "q1",
    kind: "choice",
    text: "Vì sao ảnh mã hoá bằng chế độ ECB vẫn nhìn thấy được đường nét của ảnh gốc?",
    options: [
      "Vì AES chỉ mã hoá 128 bit đầu tiên của tệp",
      "Vì các khối bản rõ giống nhau luôn cho ra khối bản mã giống nhau",
      "Vì ECB nén dữ liệu trước khi mã hoá",
      "Vì ECB dùng khoá ngắn hơn các chế độ khác",
    ],
    correct: 1,
    explain: "ECB mã hoá từng khối 16 byte độc lập với cùng một khoá, nên khối bản rõ trùng nhau cho ra bản mã trùng nhau và cấu trúc ảnh lộ ra.",
    source: CH3,
  },
  {
    id: "q2",
    kind: "choice",
    text: "Trong chế độ CBC, khối bản rõ đầu tiên được XOR với cái gì trước khi mã hoá?",
    options: ["Khoá phiên", "Véc-tơ khởi tạo (IV)", "Giá trị băm của tệp", "Khối bản mã cuối cùng"],
    correct: 1,
    explain: "Khối đầu XOR với IV; các khối sau XOR với khối bản mã liền trước. IV phải không lặp lại nhưng không cần bí mật.",
    source: CH3,
  },
  {
    id: "q3",
    kind: "choice",
    text: "Hậu quả trực tiếp của việc dùng lại một IV với cùng khoá trong CBC là gì?",
    options: [
      "Bản mã dài gấp đôi",
      "Hai bản rõ giống nhau cho ra phần đầu bản mã giống nhau, làm lộ thông tin",
      "Quá trình giải mã sai hoàn toàn",
      "Khoá bị lộ ngay lập tức",
    ],
    correct: 1,
    explain: "IV lặp lại khiến hai thông điệp có phần đầu giống nhau sinh ra bản mã giống nhau ở phần đầu — kẻ tấn công suy ra được nội dung lặp.",
    source: CH3,
  },
  {
    id: "q4",
    kind: "choice",
    text: "AES-128 có kích thước khối là bao nhiêu bit?",
    options: ["64 bit", "128 bit", "192 bit", "256 bit"],
    correct: 1,
    explain: "AES luôn có khối 128 bit; con số 128/192/256 chỉ độ dài khoá, không phải độ dài khối.",
    source: CH3,
  },
  {
    id: "q5",
    kind: "choice",
    text: "Chế độ nào biến mã khối thành mã dòng và cho phép mã hoá song song?",
    options: ["CBC", "ECB", "CTR", "CFB"],
    correct: 2,
    explain: "CTR mã hoá bộ đếm rồi XOR với bản rõ, nên mỗi khối tính độc lập và chạy song song được; CBC thì phải tuần tự.",
    source: CH3,
  },
  {
    id: "q6",
    kind: "choice",
    text: "Vì sao chỉ mã hoá bằng CBC là chưa đủ cho dữ liệu truyền trên mạng?",
    options: [
      "Vì CBC chậm hơn ECB",
      "Vì CBC không bảo vệ tính toàn vẹn, kẻ tấn công có thể sửa bản mã mà không bị phát hiện",
      "Vì CBC không dùng được với AES",
      "Vì CBC cần khoá dài hơn 256 bit",
    ],
    correct: 1,
    explain: "Cần thêm mã xác thực thông điệp (HMAC) hoặc dùng chế độ có xác thực sẵn như GCM; mã hoá chỉ che nội dung, không chống sửa đổi.",
    source: CH3,
  },
  {
    id: "q7",
    kind: "choice",
    text: "Đệm (padding) trong CBC dùng để làm gì?",
    options: [
      "Làm bản rõ đủ bội số của kích thước khối",
      "Tăng độ dài khoá",
      "Sinh IV ngẫu nhiên",
      "Nén dữ liệu trước khi mã hoá",
    ],
    correct: 0,
    explain: "CBC xử lý theo khối 16 byte nên bản rõ phải được đệm (PKCS#7) cho đủ bội số; xử lý sai lỗi đệm dẫn tới tấn công padding oracle.",
    source: CH3,
  },
  {
    id: "q8",
    kind: "choice",
    text: "Mã hoá đối xứng và mã hoá khoá công khai thường được dùng cùng nhau theo cách nào?",
    options: [
      "Mã hoá khoá công khai mã hoá toàn bộ tệp, AES chỉ ký",
      "Khoá công khai dùng để trao đổi khoá phiên, AES mã hoá dữ liệu",
      "Cả hai mã hoá song song cùng một dữ liệu",
      "AES sinh ra cặp khoá RSA",
    ],
    correct: 1,
    explain: "Khoá công khai chậm nên chỉ dùng để trao đổi khoá phiên; dữ liệu lớn do AES mã hoá — đây là mô hình lai trong TLS.",
    source: CH5,
  },
  {
    id: "q9",
    kind: "choice",
    text: "Khoá AES nên được sinh ra bằng cách nào?",
    options: [
      "Băm mật khẩu người dùng bằng MD5",
      "Dùng bộ sinh số ngẫu nhiên an toàn mật mã của hệ điều hành",
      "Dùng thời gian hệ thống làm hạt giống",
      "Lấy 16 ký tự đầu của tên tệp",
    ],
    correct: 1,
    explain: "Khoá phải không đoán được: dùng nguồn ngẫu nhiên của hệ điều hành, hoặc dẫn xuất từ mật khẩu bằng hàm chậm có muối (PBKDF2, Argon2).",
    source: CH5,
  },
  {
    id: "q10",
    kind: "short",
    text: "Nêu ngắn gọn một lý do không dùng ECB để mã hoá tệp cấu hình của hệ thống.",
    keywords: ["lặp", "giống nhau", "lộ", "mẫu", "cấu trúc"],
    answer: "ECB làm lộ cấu trúc tệp: những dòng cấu hình giống nhau cho ra các khối bản mã giống nhau.",
    explain: "Đáp án cần nêu được: ECB làm lộ cấu trúc dữ liệu vì các đoạn giống nhau cho ra bản mã giống nhau — tệp cấu hình có rất nhiều dòng lặp lại.",
    source: CH3,
  },
];

export const HASH_QUESTIONS: Question[] = [
  {
    id: "h1",
    kind: "choice",
    text: "Tính chất nào khiến hàm băm dùng được để kiểm tra toàn vẹn tệp?",
    options: ["Khả nghịch", "Kháng va chạm", "Nén dữ liệu", "Mã hoá nội dung"],
    correct: 1,
    explain: "Kháng va chạm nghĩa là rất khó tìm hai nội dung khác nhau cho cùng giá trị băm, nên băm khác đi tức là tệp đã đổi.",
    source: CH4,
  },
  {
    id: "h2",
    kind: "choice",
    text: "Chữ ký số của một văn bản thực chất là gì?",
    options: [
      "Giá trị băm của văn bản được mã hoá bằng khoá riêng của người ký",
      "Ảnh chữ ký tay được chèn vào tệp",
      "Toàn bộ văn bản mã hoá bằng khoá công khai",
      "Một mã OTP gắn vào tệp",
    ],
    correct: 0,
    explain: "Người ký băm văn bản rồi mã hoá giá trị băm bằng khoá riêng; người nhận dùng khoá công khai để kiểm tra.",
    source: CH4,
  },
  {
    id: "h3",
    kind: "choice",
    text: "Vì sao không nên dùng SHA-256 trần để lưu mật khẩu?",
    options: ["Vì SHA-256 đã bị phá", "Vì nó quá nhanh, dễ dò hàng tỷ mật khẩu mỗi giây", "Vì nó cho ra chuỗi quá dài", "Vì nó cần khoá bí mật"],
    correct: 1,
    explain: "Lưu mật khẩu cần hàm chậm có muối: bcrypt, scrypt hoặc Argon2.",
    source: CH4,
  },
  {
    id: "h4",
    kind: "choice",
    text: "HMAC khác hàm băm thường ở điểm nào?",
    options: ["Có dùng khoá bí mật", "Cho ra chuỗi ngắn hơn", "Có thể giải ngược", "Không cần dữ liệu đầu vào"],
    correct: 0,
    explain: "HMAC trộn khoá bí mật vào quá trình băm, nên chỉ ai có khoá mới tạo được mã hợp lệ.",
    source: CH4,
  },
  {
    id: "h5",
    kind: "choice",
    text: "Va chạm của hàm băm nghĩa là gì?",
    options: ["Hai đầu vào khác nhau cho cùng giá trị băm", "Băm bị lỗi tràn bộ nhớ", "Hai khoá trùng nhau", "Giá trị băm trùng với bản rõ"],
    correct: 0,
    explain: "MD5 và SHA-1 đã có va chạm thực tế nên không còn dùng cho chữ ký số.",
    source: CH4,
  },
  {
    id: "h6",
    kind: "choice",
    text: "Chứng thư số liên kết điều gì với nhau?",
    options: ["Khoá công khai với danh tính chủ thể", "Mật khẩu với tên đăng nhập", "Khoá riêng với địa chỉ IP", "Tệp với thời gian tải về"],
    correct: 0,
    explain: "Tổ chức chứng thực (CA) ký chứng thư để khẳng định khoá công khai này thuộc về chủ thể kia.",
    source: CH5,
  },
];

/**
 * Đề tính điểm QUIZ01 có bộ câu RIÊNG (SRS 4.3.4, E11): 8 câu Chương 3, không câu nào trùng chữ đề
 * (sau chuẩn hoá) với `SYMMETRIC_QUESTIONS`, `HASH_QUESTIONS`, ngân hàng `/questions` hay lượt luyện nào.
 * Chỉ `at-quiz01` dùng bộ này; tra câu theo id của luyện đề (`questionById`) không bao giờ trả câu QUIZ01.
 */
export const QUIZ01_ITEMS: Question[] = [
  {
    id: "qz1",
    kind: "choice",
    text: "Khi mã hoá hai tệp khác nhau bằng cùng một khoá AES-CTR, giá trị nào bắt buộc phải khác nhau?",
    options: ["Bộ đếm khởi tạo (nonce)", "Độ dài khoá", "Kiểu đệm", "Số vòng biến đổi"],
    correct: 0,
    explain: "Dòng khoá của CTR sinh từ (khoá, nonce); lặp lại cặp này làm hai bản rõ bị XOR với cùng dòng khoá.",
    source: CH3,
  },
  {
    id: "qz2",
    kind: "choice",
    text: "Chế độ vận hành nào bắt buộc mã hoá tuần tự vì khối sau cần bản mã của khối trước?",
    options: ["CTR", "CBC", "ECB", "OFB"],
    correct: 1,
    explain: "CBC XOR khối bản rõ với bản mã liền trước nên không chạy song song khi mã hoá.",
    source: CH3,
  },
  {
    id: "qz3",
    kind: "choice",
    text: "Bước nào của AES trộn bốn byte trong cùng một cột của trạng thái?",
    options: ["SubBytes", "ShiftRows", "MixColumns", "AddRoundKey"],
    correct: 2,
    explain: "MixColumns tạo khuếch tán theo cột; vòng cuối của AES bỏ bước này.",
    source: CH3,
  },
  {
    id: "qz4",
    kind: "choice",
    text: "Tệp 1000 byte mã hoá bằng AES-CBC có đệm PKCS#7 cho ra bản mã dài bao nhiêu byte (chưa kể IV)?",
    options: ["1000", "1008", "1016", "1024"],
    correct: 1,
    explain: "1000 byte = 62 khối 16 byte dư 8 byte; đệm thêm 8 byte thành 63 khối = 1008 byte.",
    source: CH3,
  },
  {
    id: "qz5",
    kind: "choice",
    text: "Nếu một bit của IV bị sửa trước khi giải mã CBC thì hậu quả là gì?",
    options: [
      "Hỏng đúng một bit tương ứng ở khối bản rõ đầu tiên",
      "Hỏng toàn bộ các khối bản rõ",
      "Khoá giải mã bị sai",
      "Không ảnh hưởng gì",
    ],
    correct: 0,
    explain: "Khối đầu được XOR với IV sau khi giải mã khối, nên lỗi IV lan đúng vị trí bit đó.",
    source: CH3,
  },
  {
    id: "qz6",
    kind: "choice",
    text: "Vì sao 3DES không còn được khuyến nghị cho hệ thống mới?",
    options: [
      "Vì khối chỉ 64 bit nên không an toàn khi mã hoá khối lượng dữ liệu lớn",
      "Vì không có thư viện nào hỗ trợ",
      "Vì khoá của nó dài hơn AES-256",
      "Vì nó không dùng được với chế độ CBC",
    ],
    correct: 0,
    explain: "Khối 64 bit dẫn tới va chạm sinh nhật sau vài chục GB dữ liệu (tấn công Sweet32).",
    source: CH3,
  },
  {
    id: "qz7",
    kind: "choice",
    text: "Muốn vừa che nội dung vừa phát hiện bản mã bị sửa mà không ghép thêm HMAC, nên chọn chế độ nào?",
    options: ["AES-GCM", "AES-ECB", "AES-CBC", "AES-OFB"],
    correct: 0,
    explain: "GCM là chế độ mã hoá có xác thực: sinh kèm thẻ xác thực, giải mã sai thẻ thì từ chối dữ liệu.",
    source: CH3,
  },
  {
    id: "qz8",
    kind: "choice",
    text: "Mật mã dòng tạo bản mã bằng phép toán nào giữa dòng khoá và bản rõ?",
    options: ["XOR từng bit", "Cộng modulo 10 từng chữ số", "Hoán vị bảng chữ cái", "Nhân ma trận"],
    correct: 0,
    explain: "Mật mã dòng XOR bản rõ với dòng khoá giả ngẫu nhiên; giải mã là XOR lại đúng dòng khoá đó.",
    source: CH3,
  },
];

export type Attempt = {
  id: string;
  mode: "topic" | "quiz" | "review";
  title: string;
  topic: string;
  questions: Question[];
  minutes?: number;
  /** lượt đã làm xong: đáp án đã chọn và điểm */
  answers?: number[];
  score?: number;
  minsAgo?: number;
};

export const ATTEMPTS: Attempt[] = [
  { id: "at-symmetric", mode: "topic", title: "Luyện chủ đề: Mật mã đối xứng", topic: "Mật mã đối xứng", questions: SYMMETRIC_QUESTIONS },
  { id: "at-quiz01", mode: "quiz", title: `${ASSIGNMENTS[3].code} — ${ASSIGNMENTS[3].title}`, topic: "Mật mã đối xứng", questions: QUIZ01_ITEMS, minutes: 20 },
  { id: "at-h1", mode: "review", title: "Mật mã đối xứng · 7 câu", topic: "Mật mã đối xứng", questions: SYMMETRIC_QUESTIONS.slice(0, 7), answers: [1, 0, 1, 1, 0, 1, 2], score: 3, minsAgo: 60 * 26 },
  { id: "at-h2", mode: "review", title: "Hàm băm và chữ ký số · 6 câu", topic: "Hàm băm và chữ ký số", questions: HASH_QUESTIONS, answers: [1, 0, 1, 0, 0, 0], score: 6, minsAgo: 60 * 50 },
  { id: "at-h3", mode: "review", title: "Mật mã đối xứng · 5 câu", topic: "Mật mã đối xứng", questions: SYMMETRIC_QUESTIONS.slice(0, 5), answers: [1, 1, 0, 1, 2], score: 4, minsAgo: 60 * 74 },
  { id: "at-h4", mode: "review", title: "An toàn ứng dụng web · 4 câu", topic: "An toàn ứng dụng web", questions: HASH_QUESTIONS.slice(0, 4), answers: [1, 0, 1, 3], score: 3, minsAgo: 60 * 120 },
  { id: "at-h5", mode: "review", title: "Hàm băm và chữ ký số · 6 câu", topic: "Hàm băm và chữ ký số", questions: HASH_QUESTIONS, answers: [1, 2, 1, 0, 0, 2], score: 4, minsAgo: 60 * 168 },
  { id: "at-h6", mode: "review", title: "Mật mã đối xứng · 6 câu", topic: "Mật mã đối xứng", questions: SYMMETRIC_QUESTIONS.slice(0, 6), answers: [0, 1, 1, 1, 2, 0], score: 4, minsAgo: 60 * 200 },
];

export function attemptById(id: string) {
  return ATTEMPTS.find((a) => a.id === id);
}

export const HISTORY = ATTEMPTS.filter((a) => a.mode === "review");

/** Chủ đề yếu: tỷ lệ sai cộng dồn của các lượt đã làm. */
export const WEAK_TOPICS = [
  { topic: "Mật mã đối xứng", wrong: 11, total: 18 },
  { topic: "An toàn ứng dụng web", wrong: 1, total: 4 },
  { topic: "Hàm băm và chữ ký số", wrong: 2, total: 12 },
];

// ---- Lượt luyện của phiên (SRS 4.3.4) ---------------------------------------------------------

/** Đáp án một câu: chỉ số lựa chọn, chữ đã gõ, hoặc chưa trả lời. */
export type Answer = number | string | null;

/** Lượt luyện theo chủ đề đang mở; có `finishedMs` nghĩa là đã bấm `Xem kết quả`. */
export type Run = { attemptId: string; title: string; topic: string; qids: string[]; answers: Answer[]; startedMs: number; finishedMs?: number };
/** Lượt đã xong, ghi vào lịch sử ngay lúc bấm `Xem kết quả`. */
export type RunRecord = { id: string; topic: string; qids: string[]; answers: Answer[]; score: number; atMs: number };

// Lát trạng thái tách theo sinh viên: khoá thật là `practice.run.<studentId>` / `practice.history.<studentId>`.
export const RUN_KEY = "practice.run";
export const HISTORY_KEY = "practice.history";
export const RUN_SEED: Run | null = null;
export const HISTORY_SEED: RunRecord[] = [];

/** Dữ liệu luyện đề seed là của sinh viên B; sinh viên khác bắt đầu từ con số 0 (FR-X18). */
export const SEED_STUDENT = "sv-2";

/** Lượt dở sẵn có của kịch bản: dừng ở câu 6/10, bắt đầu cách mốc 09:20 đúng 95 phút; `Tiếp tục` ghi lượt này. */
export const SEED_RUN: Run = {
  attemptId: "at-symmetric",
  title: "Luyện chủ đề: Mật mã đối xứng",
  topic: "Mật mã đối xứng",
  qids: SYMMETRIC_QUESTIONS.map((q) => q.id),
  answers: SYMMETRIC_QUESTIONS.map((q, i) => (i < 5 ? (q.correct ?? null) : null)),
  startedMs: NOW.getTime() - 95 * 60000,
};

const PRACTICE_BANK = [...SYMMETRIC_QUESTIONS, ...HASH_QUESTIONS];

/** Tra câu theo id CHỈ trong ngân hàng luyện: không lượt luyện nào chạm được vào câu QUIZ01 (E11). */
export const questionsOf = (qids: string[]) => qids.map((id) => PRACTICE_BANK.find((q) => q.id === id)).filter((q): q is Question => Boolean(q));

/** Lượt mới từ câu 1; `qids` hẹp hơn khi ôn lại các câu sai. */
export function newRun(attempt: Attempt, startedMs: number, qids = attempt.questions.map((q) => q.id), title = attempt.title): Run {
  return { attemptId: attempt.id, title, topic: attempt.topic, qids, answers: qids.map(() => null), startedMs };
}

/**
 * Chấm một câu (SRS 4.3.4): trắc nghiệm đúng khi chỉ số chọn = `correct`; trả lời ngắn đúng khi chữ đã
 * chuẩn hoá chứa ≥ 1 từ khoá, khớp nguyên từ / cụm từ ("lộ" không khớp bên trong "logic").
 */
export function isCorrect(q: Question, given: Answer): boolean {
  if (q.kind === "choice") return typeof given === "number" && given === q.correct;
  if (typeof given !== "string") return false;
  const text = normalizeVi(given);
  return (q.keywords ?? []).some((k) => {
    const kw = normalizeVi(k).replace(/[.*+?^${}()|[\]\\-]/g, "\\$&");
    return new RegExp(`(?:^|[^a-z0-9])${kw}(?![a-z0-9])`).test(text);
  });
}

/** Số câu đúng của một lượt. */
export const scoreOf = (qids: string[], answers: Answer[]) => questionsOf(qids).filter((q, i) => isCorrect(q, answers[i])).length;

/** Chủ đề yếu: nền kịch bản (chỉ sinh viên B) cộng dồn các lượt đã làm trong phiên. */
export function weakTopics(done: RunRecord[], seeded: boolean) {
  const out = seeded ? WEAK_TOPICS.map((w) => ({ ...w })) : ([] as { topic: string; wrong: number; total: number }[]);
  for (const r of done) {
    let row = out.find((w) => w.topic === r.topic);
    if (!row) {
      row = { topic: r.topic, wrong: 0, total: 0 };
      out.push(row);
    }
    row.total += r.qids.length;
    row.wrong += r.qids.length - r.score;
  }
  return out;
}

export type HistoryRow = { id: string; topic: string; total: number; score: number; atMs: number };

/** Lịch sử hiển thị: lượt của phiên (mới nhất trước) rồi các lượt nền của sinh viên B. */
export function historyRows(done: RunRecord[], seeded: boolean): HistoryRow[] {
  return [
    ...[...done].sort((a, b) => b.atMs - a.atMs).map((r) => ({ id: r.id, topic: r.topic, total: r.qids.length, score: r.score, atMs: r.atMs })),
    ...(seeded ? HISTORY.map((h) => ({ id: h.id, topic: h.topic, total: h.questions.length, score: h.score ?? 0, atMs: NOW.getTime() - (h.minsAgo ?? 0) * 60000 })) : []),
  ];
}

/** Lát trạng thái bài QUIZ01, tách theo sinh viên (FR-X18): chat chỉ trả lời thủ tục khi người đó đang làm. */
export const quizKey = (studentId?: string) => `practice.quiz01.${studentId ?? "khach"}`;
export type QuizState = { status: "idle" | "doing" | "submitted"; answers: Record<string, number>; score?: number };
export const QUIZ_SEED: QuizState = { status: "idle", answers: {} };
