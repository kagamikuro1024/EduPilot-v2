// Số liệu lớp học cho /analytics (DESIGN §14.25). Dữ liệu MÔ PHỎNG. Người giữ: US-PROTO-04.
// Chi phí chỉ giảng viên và quản trị viên được xem (SRS 4.6, Q2) — màn tự lọc theo vai.
import { COURSE_1, COURSE_2, NOW, fmtShortDate } from "./core";

export type AnalyticsRange = "7" | "30";

type Series = { label: string; value: number };

type CourseSeed = {
  /** câu hỏi mỗi ngày, 30 ngày gần nhất, cũ → mới */
  daily: number[];
  students: number;
  active7: number;
  active30: number;
  topics7: Series[];
  topics30: Series[];
  /** hỗ trợ: [7 ngày, 30 ngày] */
  escalated: [number, number];
  medianReply: [string, string];
  overdue: [number, number];
  answeredByAi: [number, number];
  piiDetected: [number, number];
  piiChannels: Series[];
  submissions: [number, number];
  edited: [number, number];
  avgGap: [string, string];
  /** chi phí (đồng) */
  cost: [number, number];
};

const SEED: Record<string, CourseSeed> = {
  [COURSE_1]: {
    daily: [34, 41, 28, 19, 52, 61, 44, 38, 47, 33, 22, 58, 66, 51, 43, 49, 31, 24, 57, 69, 48, 55, 62, 40, 29, 71, 83, 64, 58, 47],
    students: 30,
    active7: 22,
    active30: 28,
    topics7: [
      { label: "Mật mã đối xứng (AES, CBC)", value: 31 },
      { label: "Hàm băm và chữ ký số", value: 24 },
      { label: "SQL injection", value: 13 },
      { label: "RSA và trao đổi khoá", value: 9 },
      { label: "Bắt tay TLS 1.3", value: 3 },
    ],
    topics30: [
      { label: "Mật mã đối xứng (AES, CBC)", value: 112 },
      { label: "Hàm băm và chữ ký số", value: 87 },
      { label: "SQL injection", value: 64 },
      { label: "RSA và trao đổi khoá", value: 41 },
      { label: "Quy chế và cách tính điểm", value: 33 },
    ],
    escalated: [6, 19],
    medianReply: ["3 giờ 40 phút", "4 giờ 05 phút"],
    overdue: [1, 3],
    answeredByAi: [94, 92],
    piiDetected: [9, 31],
    piiChannels: [
      { label: "Chat riêng", value: 6 },
      { label: "Bài nộp qua email", value: 2 },
      { label: "Thread công khai", value: 1 },
    ],
    submissions: [28, 84],
    edited: [5, 14],
    avgGap: ["0,4 điểm", "0,5 điểm"],
    cost: [298000, 1240000],
  },
  [COURSE_2]: {
    daily: [0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 5, 9, 7, 11, 8, 6, 14, 17, 12, 9, 19, 23, 16, 21, 18],
    students: 30,
    active7: 17,
    active30: 21,
    topics7: [
      { label: "Tường lửa và phân đoạn mạng", value: 22 },
      { label: "Mô hình OSI", value: 15 },
      { label: "VPN site-to-site và IPsec", value: 7 },
      { label: "Phát hiện xâm nhập (IDS/IPS)", value: 3 },
    ],
    topics30: [
      { label: "Tường lửa và phân đoạn mạng", value: 38 },
      { label: "Mô hình OSI", value: 29 },
      { label: "Quy chế và cách tính điểm", value: 18 },
      { label: "VPN site-to-site và IPsec", value: 11 },
    ],
    escalated: [2, 4],
    medianReply: ["5 giờ 10 phút", "5 giờ 10 phút"],
    overdue: [0, 1],
    answeredByAi: [89, 89],
    piiDetected: [3, 5],
    piiChannels: [
      { label: "Chat riêng", value: 2 },
      { label: "Yêu cầu vào lớp", value: 1 },
    ],
    submissions: [0, 0],
    edited: [0, 0],
    avgGap: ["—", "—"],
    cost: [46000, 88000],
  },
};

export type Analytics = {
  range: AnalyticsRange;
  /** câu hỏi theo ngày (7 ngày) hoặc theo cụm 3 ngày (30 ngày) */
  trend: Array<{ x: string; y: number }>;
  questions: number;
  questionsPerDay: number;
  activeStudents: number;
  students: number;
  topics: Series[];
  answeredByAi: number;
  escalated: number;
  medianReply: string;
  overdue: number;
  piiDetected: number;
  piiChannels: Series[];
  piiLeaked: number;
  submissions: number;
  edited: number;
  editRate: number;
  avgGap: string;
  cost: number;
  costPerStudent: number;
};

const DAY_LABEL = ["T2", "T3", "T4", "T5", "T6", "T7", "CN"];

/** 30 ngày gần nhất kết thúc ở hôm nay (Thứ Năm 29/10/2026). */
function trendPoints(daily: number[], range: AnalyticsRange) {
  if (range === "7") {
    const last = daily.slice(-7);
    // hôm nay là Thứ Năm → 7 ngày gần nhất bắt đầu từ Thứ Sáu tuần trước
    return last.map((y, i) => ({ x: DAY_LABEL[(4 + i) % 7], y }));
  }
  const buckets: Array<{ x: string; y: number }> = [];
  for (let i = 0; i < daily.length; i += 3) {
    const sum = daily.slice(i, i + 3).reduce((a, b) => a + b, 0);
    // nhãn là ngày đầu của cụm 3 ngày, tính lùi từ hôm nay
    buckets.push({ x: fmtShortDate(new Date(NOW.getTime() - (daily.length - 1 - i) * 86400000)), y: sum });
  }
  return buckets;
}

export function analyticsFor(courseId: string, range: AnalyticsRange): Analytics {
  const s = SEED[courseId] ?? SEED[COURSE_1];
  const i = range === "7" ? 0 : 1;
  const days = range === "7" ? 7 : 30;
  const daily = s.daily.slice(-days);
  const questions = daily.reduce((a, b) => a + b, 0);
  const edited = s.edited[i];
  const submissions = s.submissions[i];
  return {
    range,
    trend: trendPoints(s.daily, range),
    questions,
    questionsPerDay: Math.round(questions / days),
    activeStudents: range === "7" ? s.active7 : s.active30,
    students: s.students,
    topics: range === "7" ? s.topics7 : s.topics30,
    answeredByAi: s.answeredByAi[i],
    escalated: s.escalated[i],
    medianReply: s.medianReply[i],
    overdue: s.overdue[i],
    piiDetected: s.piiDetected[i],
    piiChannels: s.piiChannels,
    piiLeaked: 0,
    submissions,
    edited,
    editRate: submissions ? Math.round((edited / submissions) * 100) : 0,
    avgGap: s.avgGap[i],
    cost: s.cost[i],
    costPerStudent: Math.round(s.cost[i] / s.students / 100) * 100,
  };
}

/** "298.000 đ" — tự nhóm nghìn để server và trình duyệt cho ra cùng một chuỗi. */
export function fmtVnd(n: number) {
  return `${n.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ".")} đ`;
}
