// Dữ liệu cho /gradebook và /gradebook/scheme (US-PROTO-03).
import { COURSE_1, COURSE_2 } from "./core";
import { SCHEME_1, SCHEME_2_DRAFT, type Scheme } from "./grades";

/** Một mục của công thức điểm, kèm đoạn trích quy chế đã rút ra nó. */
export type SchemeRule = {
  id: string;
  label: string;
  /** giá trị đã rút ra; null = chưa rõ, chặn xác nhận cho tới khi người điền */
  value: string | null;
  /** câu hỏi đặt cho giảng viên khi value = null */
  ask?: string;
  quote: string;
  page: number;
};

export type SchemeDoc = {
  courseId: string;
  /** tệp quy chế nguồn */
  file: string;
  pages: number;
  scheme: Scheme;
  rules: SchemeRule[];
};

export const SCHEME_DOC_2: SchemeDoc = {
  courseId: COURSE_2,
  file: "quy-che-lop2.pdf",
  pages: 6,
  scheme: SCHEME_2_DRAFT,
  rules: [
    {
      id: "weight",
      label: "Trọng số",
      value: "Quá trình 30% · Cuối kỳ 70%",
      quote: "Điểm học phần gồm điểm quá trình chiếm 30% và điểm thi kết thúc học phần chiếm 70%.",
      page: 2,
    },
    {
      id: "qt",
      label: "Điểm quá trình",
      value: "Trung bình các bài tập đã công bố, cộng điểm phát biểu, trừ điểm vắng",
      quote: "Điểm quá trình là trung bình cộng các bài tập đã công bố, có cộng điểm chuyên cần – phát biểu và trừ điểm vắng mặt.",
      page: 2,
    },
    {
      id: "speak",
      label: "Điểm cộng phát biểu",
      value: "+0,2 mỗi lần phát biểu, tối đa +0,6",
      quote: "Mỗi lần phát biểu xây dựng bài được cộng 0,2 điểm vào điểm quá trình, tổng cộng không quá 0,6 điểm.",
      page: 3,
    },
    {
      id: "absence",
      label: "Trừ điểm vắng",
      value: "−0,5 mỗi buổi vắng, tính từ buổi vắng thứ 3",
      quote: "Từ buổi vắng không phép thứ ba trở đi, mỗi buổi trừ 0,5 điểm quá trình.",
      page: 3,
    },
    {
      id: "rounding",
      label: "Quy tắc làm tròn",
      value: null,
      ask: "Quy chế không nói làm tròn đến mấy chữ số. Giảng viên điền để công thức tính được.",
      quote: "Điểm học phần được công bố theo thang 10. (Quy chế không nêu quy tắc làm tròn.)",
      page: 5,
    },
  ],
};

export const SCHEME_DOC_1: SchemeDoc = {
  courseId: COURSE_1,
  file: "quy-che-mon-hoc-761987.pdf",
  pages: 5,
  scheme: SCHEME_1,
  rules: [
    { id: "weight", label: "Trọng số", value: "Quá trình 40% · Cuối kỳ 60%", quote: "Điểm quá trình chiếm 40%, điểm thi kết thúc học phần chiếm 60% điểm học phần.", page: 2 },
    {
      id: "qt",
      label: "Điểm quá trình",
      value: "Trung bình các bài tập đã công bố, cộng điểm phát biểu, trừ điểm vắng",
      quote: "Điểm quá trình là trung bình cộng các bài tập đã công bố của học phần, có cộng – trừ theo chuyên cần.",
      page: 2,
    },
    { id: "speak", label: "Điểm cộng phát biểu", value: "+0,25 mỗi lần phát biểu, tối đa +1,0", quote: "Mỗi lần phát biểu được cộng 0,25 điểm, tổng không quá 1,0 điểm.", page: 3 },
    { id: "absence", label: "Trừ điểm vắng", value: "−0,5 mỗi buổi vắng, tính từ buổi vắng thứ 3", quote: "Từ buổi vắng không phép thứ ba, mỗi buổi trừ 0,5 điểm quá trình.", page: 3 },
    { id: "rounding", label: "Quy tắc làm tròn", value: "Làm tròn đến 0,1", quote: "Điểm quá trình và điểm học phần làm tròn đến một chữ số thập phân.", page: 4 },
  ],
};

/** Ô mẫu có xung đột phiên bản (409 giả lập) khi giảng viên sửa — SRS mục 3. */
export const CONFLICT_CELL = { studentId: "sv-5", column: "bt02", otherValue: "8,0", by: "Lê Thu Hà", at: "09:18" };

export const OFFICIAL_GRADE_NOTE = "Điểm chính thức nằm ở hệ thống quản lý đào tạo của trường; bảng này là bản làm việc của lớp.";

/** Lịch sử sửa một ô điểm (mở từ menu ngữ cảnh của hàng). */
export const CELL_HISTORY = [
  { at: "22/10 16:04", by: "AI chấm nháp", text: "Đề xuất 7,0 cho Bài tập 03" },
  { at: "22/10 16:04", by: "AI chấm nháp (lượt 2)", text: "Tiêu chí 2 chấm 2,5 — lệch 1,5 so với lượt 1, chuyển sang Cần xem kỹ" },
  { at: "05/10 08:40", by: "Lê Thu Hà", text: "Công bố điểm Bài tập 02: 8,0" },
  { at: "20/09 10:12", by: "Lê Thu Hà", text: "Công bố điểm Bài tập 01: 7,0" },
];
