// Tài liệu giảng viên của học phần An ninh mạng (/documents — US-PROTO-03).
// Tên tệp lấy từ seed/documents/ và theo quy ước đặt tên của bộ môn.

export type DocKind = "LECTURE" | "REGULATION" | "EXAM_PAPER" | "ANSWER_KEY";
export type DocStatus = "READY" | "PROCESSING" | "FAILED";

export type Doc = {
  id: string;
  name: string;
  kind: DocKind;
  /** tuần học tài liệu phục vụ; null với quy chế / đề cũ */
  week: number | null;
  topic: string;
  /** dùng làm nguồn cho AI trả lời */
  forAi: boolean;
  /** hiện trong Thư viện của sinh viên */
  forStudents: boolean;
  status: DocStatus;
  /** ngày cập nhật, dạng dd/mm */
  updated: string;
  size: string;
  pages: number;
  failReason?: string;
  failFix?: string;
};

export const DOC_KIND_LABEL: Record<DocKind, string> = {
  LECTURE: "Bài giảng",
  REGULATION: "Quy chế",
  EXAM_PAPER: "Đề cũ",
  ANSWER_KEY: "Đáp án",
};

export const ANSWER_KEY_NOTE = "Không hiển thị cho sinh viên · Không dùng cho AI của sinh viên";

export const DOCUMENTS: Doc[] = [
  { id: "doc-1", name: "Chuong1_Tong_quan_an_ninh_mang.pdf", kind: "LECTURE", week: 1, topic: "Tổng quan an ninh mạng", forAi: true, forStudents: true, status: "READY", updated: "27/08", size: "2,4 MB", pages: 38 },
  { id: "doc-2", name: "Chuong2_Mat_ma_doi_xung.pdf", kind: "LECTURE", week: 3, topic: "Mật mã đối xứng (AES, CBC, ECB)", forAi: true, forStudents: true, status: "READY", updated: "10/09", size: "3,1 MB", pages: 52 },
  { id: "doc-3", name: "Chuong3_Ham_bam_va_chu_ky_so.pdf", kind: "LECTURE", week: 5, topic: "Hàm băm và chữ ký số", forAi: true, forStudents: true, status: "READY", updated: "24/09", size: "2,8 MB", pages: 44 },
  { id: "doc-4", name: "Chuong4_Ha_tang_khoa_cong_khai.pdf", kind: "LECTURE", week: 7, topic: "RSA và hạ tầng khoá công khai", forAi: true, forStudents: true, status: "READY", updated: "08/10", size: "2,2 MB", pages: 40 },
  { id: "doc-5", name: "Chuong5_Tuong_lua_va_VPN.pdf", kind: "LECTURE", week: 9, topic: "Tường lửa, phân đoạn mạng và VPN", forAi: true, forStudents: true, status: "READY", updated: "22/10", size: "2,6 MB", pages: 47 },
  { id: "doc-6", name: "Mordern_Network_Security_Threats.pdf", kind: "LECTURE", week: 10, topic: "Các mối đe doạ mạng hiện đại (tiếng Anh)", forAi: true, forStudents: true, status: "PROCESSING", updated: "29/10", size: "3,1 MB", pages: 96 },
  { id: "doc-7", name: "Quyche.pdf", kind: "REGULATION", week: null, topic: "Quy chế đào tạo của trường", forAi: true, forStudents: true, status: "READY", updated: "15/08", size: "1,6 MB", pages: 24 },
  { id: "doc-8", name: "quy-che-mon-hoc-761987.pdf", kind: "REGULATION", week: null, topic: "Quy chế môn học lớp 761987 (công thức điểm)", forAi: true, forStudents: true, status: "READY", updated: "27/08", size: "320 KB", pages: 5 },
  { id: "doc-9", name: "De_thi_cuoi_ky_2024_An_ninh_mang.pdf", kind: "EXAM_PAPER", week: null, topic: "Đề cuối kỳ HK1 2024–2025", forAi: true, forStudents: true, status: "READY", updated: "02/09", size: "480 KB", pages: 6 },
  { id: "doc-10", name: "De_thi_giua_ky_2025_An_ninh_mang.pdf", kind: "EXAM_PAPER", week: null, topic: "Đề giữa kỳ HK2 2024–2025", forAi: true, forStudents: true, status: "READY", updated: "02/09", size: "410 KB", pages: 4 },
  { id: "doc-11", name: "Dap_an_cuoi_ky_2024.pdf", kind: "ANSWER_KEY", week: null, topic: "Đáp án đề cuối kỳ HK1 2024–2025", forAi: false, forStudents: false, status: "READY", updated: "02/09", size: "260 KB", pages: 5 },
  { id: "doc-12", name: "Dap_an_giua_ky_2025.pdf", kind: "ANSWER_KEY", week: null, topic: "Đáp án đề giữa kỳ HK2 2024–2025", forAi: false, forStudents: false, status: "READY", updated: "02/09", size: "240 KB", pages: 3 },
];

/** Tệp mẫu dùng để minh hoạ nhánh lỗi khi tải lên (SRS 4.5). */
export const FAILING_UPLOAD = {
  name: "scan-khong-co-chu.pdf",
  failReason: "File không có lớp chữ, không đọc được",
  failFix: "Đây là bản scan ảnh. Quét lại có nhận dạng chữ (OCR) hoặc tải lên bản PDF gốc, rồi thử lại.",
};
