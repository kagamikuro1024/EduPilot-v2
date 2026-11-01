// Dữ liệu chấm bài dùng chung giữa /grading (US-PROTO-03) và /assignments/bt03 (US-PROTO-01).


// ---- bài làm của SV B (BT03) ---------------------------------------------------------------

export const BT03 = {
  id: "bt03",
  title: "Phân tích một vụ tấn công thực tế",
  kind: "ESSAY" as const,
  due: new Date("2026-10-22T23:59:00+07:00"),
  latePerDay: 0.5,
  lateMaxDays: 2,
};


/** Bài nộp của B: nộp 23/10 08:10 = muộn 1 ngày. */
export const BT03_SUBMITTED_AT = new Date("2026-10-23T08:10:00+07:00");


export type Submission = {
  id: string;
  studentId: string;
  /** điểm nháp của AI, thang 10, trước khi trừ nộp muộn */
  ai: number;
  /** lý do cờ "Cần xem kỹ"; không có = bình thường */
  flag?: string;
  flagKind?: ReviewKind;
  /** đã duyệt sẵn từ trước (B không nằm trong số này — GV duyệt trong kịch bản) */
  approved: boolean;
  submittedAt: Date;
  source: "Tải lên trên EduPilot" | "Nộp qua email lớp";
  lateDays: number;
};

/** Loại cờ "Cần xem kỹ" (SRS 4.8 N8). Nhãn rút gọn dùng ở thẻ Hôm nay: "1 lệch hai lượt chấm · 1 bài ngắn bất thường…". */
export type ReviewKind = "gap" | "short" | "overlap" | "unsure" | "format";
export const REVIEW_KIND_SHORT: Record<ReviewKind, string> = {
  gap: "lệch hai lượt chấm",
  short: "bài ngắn bất thường",
  overlap: "trùng đoạn với bài khác",
  unsure: "AI không chắc ở một tiêu chí",
  format: "chưa khớp định dạng rubric",
};

const FLAG_KIND: Record<string, ReviewKind> = { "sv-2": "gap", "sv-6": "gap", "sv-9": "short", "sv-14": "overlap", "sv-21": "unsure", "sv-18": "format" };

const FLAGS: Record<string, string> = {
  "sv-2": "Hai lượt chấm lệch 1,5 điểm",
  "sv-9": "Bài ngắn bất thường so với yêu cầu (dưới 400 chữ)",
  "sv-14": "Trùng đoạn với bài của một sinh viên khác",
  "sv-21": "AI không đủ chắc chắn ở tiêu chí Đánh giá tác động",
  "sv-6": "Hai lượt chấm lệch hơn 1 điểm",
  "sv-18": "Bài nộp qua email, chưa khớp định dạng rubric",
};
/** Hai bài có cờ nhưng GV đã duyệt từ hôm qua → lọc mặc định vẫn còn đúng 4 bài. */
const PRE_APPROVED = ["sv-6", "sv-18", "sv-11", "sv-15", "sv-23", "sv-27", "sv-30", "sv-7", "sv-19"];
/** Hai sinh viên chưa nộp BT03 (30 − 2 = 28 bài). */
const NOT_SUBMITTED = ["sv-3", "sv-26"];

/** 28 bài nộp BT03 của lớp 1, tất định. Bài của B luôn đứng đầu khi lọc "Cần xem kỹ". */
export function bt03Submissions(students: Array<{ id: string }>): Submission[] {
  return students
    .filter((s) => !NOT_SUBMITTED.includes(s.id))
    .map((s) => {
      const n = Number(s.id.slice(3));
      const lateDays = s.id === "sv-2" ? 1 : 0;
      const day = 20 + (n % 3);
      const hour = 9 + (n % 11);
      return {
        id: `sub-bt03-${s.id}`,
        studentId: s.id,
        ai: s.id === "sv-2" ? 7.0 : Math.min(10, Math.round((5.75 + ((n * 13) % 35) / 10) * 4) / 4),
        flag: FLAGS[s.id],
        flagKind: FLAG_KIND[s.id],
        approved: PRE_APPROVED.includes(s.id),
        submittedAt:
          s.id === "sv-2"
            ? BT03_SUBMITTED_AT
            : new Date(`2026-10-${day}T${String(hour).padStart(2, "0")}:${String((n * 7) % 60).padStart(2, "0")}:00+07:00`),
        source: s.id === "sv-18" ? "Nộp qua email lớp" : "Tải lên trên EduPilot",
        lateDays,
      } satisfies Submission;
    });
}
