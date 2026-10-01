// Dữ liệu MÔ PHỎNG cho bản prototype giao diện (D44): tên, MSSV, email đều là giả.
// Thay bằng API thật từ phase PG trở đi; không file nào ngoài src/mock được tự tạo dữ liệu.

export type Role = "student" | "ta" | "teacher" | "admin";

export const ROLE_LABEL: Record<Role, string> = {
  student: "Sinh viên",
  ta: "Trợ giảng",
  teacher: "Giảng viên",
  admin: "Admin",
};

/** "Bây giờ" của bản mô phỏng: Thứ Năm, tuần 5 của học kỳ. */
export const NOW = new Date("2026-10-01T09:20:00+07:00");
export const TERM = "HK1 2026–2027";
export const SUBJECT = { code: "INT1006", name: "Nhập môn An toàn thông tin" };

export type Course = {
  id: string;
  code: string;
  name: string;
  room: string;
  schedule: string;
  week: number;
  size: number;
  /** lớp 2 là lớp "mới nhận": chưa có quy chế, có yêu cầu chờ duyệt (D34) */
  state: "active" | "new";
  joinCode: string;
  requireApproval: boolean;
};

export const COURSES: Course[] = [
  {
    id: "int1006-1",
    code: "INT1006 1",
    name: SUBJECT.name,
    room: "P.302 – G2",
    schedule: "Thứ Năm 16:30–18:00",
    week: 5,
    size: 30,
    state: "active",
    joinCode: "BX4P9TW",
    requireApproval: false,
  },
  {
    id: "int1006-2",
    code: "INT1006 2",
    name: SUBJECT.name,
    room: "P.207 – G3",
    schedule: "Thứ Ba 07:00–08:30",
    week: 5,
    size: 30,
    state: "new",
    joinCode: "K7MQ2RD",
    requireApproval: true,
  },
];

export type Person = { id: string; name: string; email: string; title?: string };

export const PEOPLE: Record<Role, Person> = {
  student: { id: "u-sv-a", name: "Nguyễn Minh Trung", email: "trung.nm229001@sv.edupilot.test" },
  ta: { id: "u-ta", name: "Phạm Quốc Bảo", email: "bao.pq@edupilot.test", title: "Trợ giảng" },
  teacher: { id: "u-gv", name: "Lê Thu Hà", email: "ha.lt@edupilot.test", title: "TS." },
  admin: { id: "u-admin", name: "Đỗ Hoàng Nam", email: "nam.dh@edupilot.test", title: "Quản trị hệ thống" },
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
  attendance: number; // %
  absences: number;
  grade: number | null; // điểm quá trình hiện tại, thang 10
  gradeTrend: number; // so với 2 tuần trước
  participation: number; // số lần phát biểu
  activityMin: number; // phút học / tuần (chỉ tham khảo, D17)
  risk: Risk;
  riskReason?: string;
};

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
    // 0–29 lớp 1; 27–56 lớp 2 → 3 SV học cả hai lớp (D34)
    const courseIds = [i < 30 ? "int1006-1" : null, i >= 27 ? "int1006-2" : null].filter(Boolean) as string[];
    const absences = Math.floor(r() * r() * 6);
    const attendance = Math.round(((10 - absences) / 10) * 100);
    const base = 4.2 + r() * 5.4;
    const grade = Math.round(base * 10) / 10;
    const gradeTrend = Math.round((r() - 0.55) * 18) / 10;
    const participation = Math.floor(r() * 9);
    const activityMin = Math.floor(20 + r() * 260);
    let risk: Risk = "none";
    let riskReason: string | undefined;
    if (absences >= 4 || grade < 5) {
      risk = "high";
      riskReason = absences >= 4 ? `Vắng ${absences}/10 buổi, sắp chạm ngưỡng cấm thi` : `Điểm quá trình ${fmtScore(grade)}, dưới ngưỡng qua môn`;
    } else if (gradeTrend <= -0.8 || activityMin < 45) {
      risk = "watch";
      riskReason = gradeTrend <= -0.8 ? `Điểm giảm ${fmtScore(-gradeTrend)} so với 2 tuần trước` : "Ít hoạt động học trong 2 tuần gần đây";
    }
    out.push({
      id: `sv-${i + 1}`,
      name,
      code,
      email: preset?.email ?? `sv${code}@sv.edupilot.test`,
      courseIds,
      attendance,
      absences,
      grade,
      gradeTrend,
      participation,
      activityMin,
      risk,
      riskReason,
    });
  }
  // SV A cố định: khá, đang hỏng phần mật mã đối xứng
  Object.assign(out[0], { attendance: 90, absences: 1, grade: 7.4, gradeTrend: -0.3, participation: 4, activityMin: 185, risk: "none", riskReason: undefined });
  return out;
}

export const STUDENTS: Student[] = makeStudents();
export const ME_STUDENT = STUDENTS[0];

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

/** "26 phút trước", "3 giờ trước", "2 ngày trước" tính từ NOW */
export function ago(d: Date) {
  const m = Math.round((NOW.getTime() - d.getTime()) / 60000);
  if (m < 1) return "vừa xong";
  if (m < 60) return `${m} phút trước`;
  const h = Math.round(m / 60);
  if (h < 24) return `${h} giờ trước`;
  return `${Math.round(h / 24)} ngày trước`;
}

/** mốc thời gian tương đối so với NOW, tính bằng phút */
export function at(minutesFromNow: number) {
  return new Date(NOW.getTime() + minutesFromNow * 60000);
}
