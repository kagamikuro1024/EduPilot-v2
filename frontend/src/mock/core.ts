// Dữ liệu MÔ PHỎNG cho bản prototype giao diện (D44): tên, MSSV, email đều là giả.
// Thay bằng API thật từ phase PG trở đi; không file nào ngoài src/mock được tự tạo dữ liệu.

export type Role = "student" | "ta" | "teacher" | "admin";

export const ROLE_LABEL: Record<Role, string> = {
  student: "Sinh viên",
  ta: "Trợ giảng",
  teacher: "Giảng viên",
  admin: "Admin",
};

/** "Bây giờ" của bản mô phỏng: Thứ Năm 29/10/2026, tuần 10 của học kỳ — khớp DEMO_SCRIPT.md (proposals #12). */
export const NOW = new Date("2026-10-29T09:20:00+07:00");
export const TERM = "HK1 2026–2027";
export const SUBJECT = { code: "INT1006", name: "An ninh mạng" };
/** Ngày buổi học đầu tiên của lớp 1 (Thứ Năm, tuần 1). Buổi n = ngày này + 7·(n−1). */
export const TERM_START = new Date("2026-08-27T00:00:00+07:00");

export type Course = {
  id: string;
  /** mã lớp hiển thị, ví dụ 761987 */
  code: string;
  name: string;
  /** "An ninh mạng – 761987" */
  label: string;
  room: string;
  schedule: string;
  /** tuần học hiện tại của lớp */
  week: number;
  size: number;
  /** lớp 2 là lớp "mới nhận": chưa xác nhận công thức, có yêu cầu chờ duyệt (D34) */
  state: "active" | "new";
  joinCode: string;
  requireApproval: boolean;
  teacher: string;
};

export const COURSES: Course[] = [
  {
    id: "int1006-1",
    code: "761987",
    name: SUBJECT.name,
    label: `${SUBJECT.name} – 761987`,
    room: "P.302 – G2",
    schedule: "Thứ Năm 09:00–11:30",
    week: 10,
    size: 30,
    state: "active",
    joinCode: "AN7K2MQ",
    requireApproval: false,
    teacher: "TS. Lê Thu Hà",
  },
  {
    id: "int1006-2",
    code: "761988",
    name: SUBJECT.name,
    label: `${SUBJECT.name} – 761988`,
    room: "P.207 – G3",
    schedule: "Thứ Ba 07:00–08:30",
    week: 3,
    size: 30,
    state: "new",
    joinCode: "BX4P9TW",
    requireApproval: true,
    teacher: "TS. Lê Thu Hà",
  },
];
export const COURSE_1 = COURSES[0].id;
export const COURSE_2 = COURSES[1].id;

export type Person = { id: string; name: string; email: string; title?: string; blurb?: string };

/** Người dùng cố định của bản mô phỏng (không phải sinh viên). */
export const STAFF: Record<"ta" | "teacher" | "admin", Person> = {
  ta: { id: "u-ta", name: "Phạm Quốc Bảo", email: "bao.pq@edupilot.test", title: "Trợ giảng" },
  teacher: { id: "u-gv", name: "Lê Thu Hà", email: "ha.lt@edupilot.test", title: "TS." },
  admin: { id: "u-admin", name: "Đỗ Hoàng Nam", email: "nam.dh@edupilot.test", title: "Quản trị hệ thống" },
};

/** Bốn sinh viên chính của kịch bản demo (DEMO_SCRIPT): A, B, C, D = sv-1…sv-4. */
export const DEMO_STUDENT_IDS = ["sv-1", "sv-2", "sv-3", "sv-4"] as const;
export const DEMO_STUDENT_BLURB: Record<string, string> = {
  "sv-1": "Học cả hai lớp 761987 và 761988",
  "sv-2": "Lớp 761987 · vắng 2 buổi, có bài nộp muộn",
  "sv-3": "Lớp 761987 · vắng nhiều, thiếu bài",
  "sv-4": "Chưa vào lớp nào — thử nhập mã tham gia",
};

// ---- sinh viên (sinh tất định để hai lần render giống hệt nhau) ----

function lcg(seed: number) {
  let s = seed >>> 0;
  return () => {
    s = (s * 1664525 + 1013904223) >>> 0;
    return s / 4294967296;
  };
}

const HO = ["Nguyễn", "Trần", "Lê", "Phạm", "Hoàng", "Vũ", "Đặng", "Bùi", "Đỗ", "Hồ", "Ngô", "Dương", "Lý", "Đinh", "Trịnh"];
const DEM = ["Văn", "Thị", "Minh", "Đức", "Thu", "Ngọc", "Quang", "Hải", "Thanh", "Gia", "Khánh", "Bảo", "Phương", "Anh", "Tuấn"];
const TEN = [
  "An", "Bình", "Chi", "Dũng", "Giang", "Hà", "Hiếu", "Hoa", "Huy", "Khoa", "Lan", "Linh", "Long", "Mai", "Nam",
  "Ngân", "Nhung", "Phong", "Phúc", "Quân", "Quỳnh", "Sơn", "Tâm", "Thảo", "Thắng", "Trang", "Tú", "Uyên", "Vy", "Yến",
];

export type Risk = "none" | "watch" | "high";

export type Student = {
  id: string;
  name: string;
  code: string;
  email: string;
  courseIds: string[];
  /** số buổi vắng trong các buổi ĐÃ GHI của lớp 1 (9 buổi trước hôm nay) */
  absences: number;
  /** số lần phát biểu đã ghi */
  speaks: number;
  /** điểm quá trình hiện tại, thang 10; null nếu chưa có */
  grade: number | null;
  gradeTrend: number; // so với 2 tuần trước
  activityMin: number; // phút học / tuần (chỉ tham khảo, D17)
  risk: Risk;
  riskReason?: string;
};

/** Số buổi lớp 1 đã ghi điểm danh trước hôm nay. Buổi 10 (hôm nay) đang diễn ra. */
export const RECORDED_SESSIONS = 9;

/** Chỉ số (trong STUDENTS) của từng nhóm — xem quy ước 57 tài khoản ở ARCHITECTURE §9. */
const IDX_BOTH = [0, 4, 5]; // 3 SV học cả hai lớp (A + 2)
const IDX_ONLY_1 = (i: number) => i !== 3 && i <= 30 && !IDX_BOTH.includes(i);
const IDX_ONLY_2 = (i: number) => i >= 31 && i <= 51; // 21 SV chỉ lớp 2
/** 3 SV đang chờ duyệt vào lớp 2 (chưa là thành viên) */
export const PENDING_STUDENT_IDS = ["sv-53", "sv-54", "sv-55"];

function makeStudents(): Student[] {
  const r = lcg(20229);
  const named: Array<Pick<Student, "name" | "code" | "email">> = [
    { name: "Nguyễn Minh Trung", code: "20229001", email: "trung.nm229001@sv.edupilot.test" },
    { name: "Trần Thu Uyên", code: "20229002", email: "uyen.tt229002@sv.edupilot.test" },
    { name: "Lê Quang Huy", code: "20229003", email: "huy.lq229003@sv.edupilot.test" },
    { name: "Phạm Ngọc Linh", code: "20229004", email: "linh.pn229004@sv.edupilot.test" },
  ];
  const out: Student[] = [];
  for (let i = 0; i < 57; i++) {
    const preset = named[i];
    const name = preset?.name ?? `${HO[Math.floor(r() * HO.length)]} ${DEM[Math.floor(r() * DEM.length)]} ${TEN[Math.floor(r() * TEN.length)]}`;
    const code = preset?.code ?? `2022${(1010 + i * 37).toString().padStart(4, "0")}`;
    const courseIds: string[] = [];
    if (IDX_BOTH.includes(i) || IDX_ONLY_1(i)) courseIds.push(COURSE_1);
    if (IDX_BOTH.includes(i) || IDX_ONLY_2(i)) courseIds.push(COURSE_2);
    const absences = Math.floor(r() * r() * 5);
    const base = 5 + r() * 4.6;
    const grade = Math.round(base * 10) / 10;
    const gradeTrend = Math.round((r() - 0.55) * 18) / 10;
    const speaks = Math.floor(r() * 7);
    const activityMin = Math.floor(30 + r() * 250);
    let risk: Risk = "none";
    let riskReason: string | undefined;
    if (absences >= 3) {
      risk = "watch";
      riskReason = `Vắng ${absences}/${RECORDED_SESSIONS} buổi, gần ngưỡng bị trừ điểm`;
    } else if (gradeTrend <= -1.2 || activityMin < 45) {
      risk = "watch";
      riskReason = gradeTrend <= -1.2 ? `Điểm giảm ${fmtScore(-gradeTrend)} so với 2 tuần trước` : "Ít hoạt động học trong 2 tuần gần đây";
    }
    out.push({
      id: `sv-${i + 1}`,
      name,
      code,
      email: preset?.email ?? `sv${code}@sv.edupilot.test`,
      courseIds,
      absences,
      speaks,
      grade,
      gradeTrend,
      activityMin,
      risk,
      riskReason,
    });
  }
  // A: khá, chuyên cần 90% (vắng 1/10), 4 lần phát biểu, sai 4/7 câu "Mật mã đối xứng" gần nhất.
  Object.assign(out[0], { absences: 1, speaks: 4, grade: 7.6, gradeTrend: -0.3, activityMin: 185, risk: "none", riskReason: undefined });
  // B: vắng buổi 2 (03/09) và buổi 4 (17/09), 3 lần phát biểu (+0,75). Điểm quá trình tính ở grades.ts.
  Object.assign(out[1], { absences: 2, speaks: 3, grade: 8.3, gradeTrend: 0.4, activityMin: 160, risk: "none", riskReason: undefined });
  // C: vắng nhiều, thiếu BT02, ít hoạt động ≥ 3 tuần → "Cần chú ý" (chỉ GV/TA thấy).
  Object.assign(out[2], {
    absences: 5,
    speaks: 0,
    grade: 4.9,
    gradeTrend: -1.6,
    activityMin: 22,
    risk: "high",
    riskReason: "Vắng 5/9 buổi, sắp chạm ngưỡng bị trừ điểm; thiếu Bài tập 02; học dưới ngưỡng trong 3 tuần liên tiếp",
  });
  // D: chưa vào lớp nào.
  Object.assign(out[3], { courseIds: [], absences: 0, speaks: 0, grade: null, gradeTrend: 0, activityMin: 0, risk: "none", riskReason: undefined });
  // Hai SV "cần chú ý" khác ở lớp 1 để dải "Lớp cần chú ý" có chứng cứ.
  Object.assign(out[8], { absences: 4, risk: "high", riskReason: "Vắng 4/9 buổi, sắp chạm ngưỡng bị trừ điểm" });
  Object.assign(out[12], { gradeTrend: -1.9, risk: "watch", riskReason: "Điểm giảm 1,9 so với 2 tuần trước" });
  return out;
}

export const STUDENTS: Student[] = makeStudents();
export const STUDENT_A = STUDENTS[0];
export const STUDENT_B = STUDENTS[1];
export const STUDENT_C = STUDENTS[2];
export const STUDENT_D = STUDENTS[3];

/** Thành viên gốc (chưa tính hệ quả của tương tác) của một lớp. */
export function studentsOf(courseId: string) {
  return STUDENTS.filter((s) => s.courseIds.includes(courseId));
}

export function studentById(id: string) {
  return STUDENTS.find((s) => s.id === id);
}

export function courseById(id: string) {
  return COURSES.find((c) => c.id === id) ?? COURSES[0];
}

// ---- định dạng tiếng Việt ----

/** 7.4 → "7,4" (dấu phẩy thập phân) */
export function fmtScore(n: number | null | undefined, digits = 1) {
  if (n === null || n === undefined) return "—";
  return n.toFixed(digits).replace(".", ",");
}

const WEEKDAY = ["Chủ nhật", "Thứ Hai", "Thứ Ba", "Thứ Tư", "Thứ Năm", "Thứ Sáu", "Thứ Bảy"];

export function fmtLongDate(d: Date) {
  return `${WEEKDAY[d.getDay()]}, ${d.getDate()} tháng ${d.getMonth() + 1}`;
}

export function fmtTime(d: Date) {
  return `${d.getHours().toString().padStart(2, "0")}:${d.getMinutes().toString().padStart(2, "0")}`;
}

export function fmtShortDate(d: Date) {
  return `${d.getDate().toString().padStart(2, "0")}/${(d.getMonth() + 1).toString().padStart(2, "0")}`;
}

/** mốc thời gian tương đối so với NOW, tính bằng phút */
export function at(minutesFromNow: number) {
  return new Date(NOW.getTime() + minutesFromNow * 60000);
}
