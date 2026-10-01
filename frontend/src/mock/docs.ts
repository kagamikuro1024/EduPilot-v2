// N5 (SRS 4.8): MỘT bảng tài liệu cho /documents, /library, `Nguồn tham khảo` của AI, chuông và "Hỏi AI về tài liệu".
// Tên hiển thị (`title`) là tên duy nhất người dùng thấy ở mọi nơi; tên tệp gốc (`file`) chỉ ở Drawer chi tiết và khi tải.
// Ngày tải lên của bài giảng tuần w = 27/08 + 7·(w − 1) ngày, trừ Chương 5 (tải hôm qua 28/10 14:00).
import type { Citation } from "./chat";

export type DocKind = "lecture" | "regulation" | "exam" | "answer";

export const DOC_KIND_LABEL: Record<DocKind, string> = {
  lecture: "Bài giảng",
  regulation: "Quy chế",
  exam: "Đề cũ",
  answer: "Đáp án",
};

export type Doc = {
  id: string;
  /** tên hiển thị */
  title: string;
  /** tên tệp gốc (giữ nguyên lỗi chính tả của nguồn: `Mordern_…`) */
  file: string;
  kind: DocKind;
  /** tuần học gắn với tài liệu; quy chế, đề, đáp án không gắn tuần */
  week: number | null;
  /** ngày tải lên dd/mm */
  uploaded: string;
  pages: number;
  size: string;
  /** sinh viên thấy trong Thư viện và được AI dùng làm nguồn; đáp án = false cả hai */
  forStudents: boolean;
};

export const DOCS: Doc[] = [
  { id: "d-ch1", title: "Chương 1 — Tổng quan an ninh mạng và mô hình đe doạ", file: "an-ninh-mang-ch1.pdf", kind: "lecture", week: 1, uploaded: "27/08", pages: 34, size: "2,1 MB", forStudents: true },
  { id: "d-ch2", title: "Chương 2 — Tấn công mạng phổ biến", file: "an-ninh-mang-ch2.pdf", kind: "lecture", week: 3, uploaded: "10/09", pages: 41, size: "2,8 MB", forStudents: true },
  { id: "d-ch3", title: "Chương 3 — Mật mã đối xứng và chế độ vận hành", file: "an-ninh-mang-ch3.pdf", kind: "lecture", week: 8, uploaded: "15/10", pages: 38, size: "3,4 MB", forStudents: true },
  { id: "d-ch4", title: "Chương 4 — Hàm băm và chữ ký số", file: "an-ninh-mang-ch4.pdf", kind: "lecture", week: 9, uploaded: "22/10", pages: 29, size: "2,2 MB", forStudents: true },
  { id: "d-ch5", title: "Chương 5 — Quản lý khoá và PKI", file: "an-ninh-mang-ch5.pdf", kind: "lecture", week: 10, uploaded: "28/10", pages: 33, size: "2,6 MB", forStudents: true },
  { id: "d-threats", title: "Modern Network Security Threats", file: "Mordern_Network_Security_Threats.pdf", kind: "lecture", week: 2, uploaded: "03/09", pages: 72, size: "3,1 MB", forStudents: true },
  { id: "d-quyche-truong", title: "Quy chế đào tạo của trường", file: "Quyche.pdf", kind: "regulation", week: null, uploaded: "15/08", pages: 48, size: "1,6 MB", forStudents: true },
  { id: "d-quyche-mon", title: "Quy chế môn học An ninh mạng – 761987", file: "quy-che-mon-hoc-761987.pdf", kind: "regulation", week: null, uploaded: "27/08", pages: 4, size: "300 KB", forStudents: true },
  { id: "d-de-2025", title: "Đề thi cuối kỳ An ninh mạng — HK1 2025–2026", file: "de-cuoi-ky-hk1-2025.pdf", kind: "exam", week: null, uploaded: "02/09", pages: 6, size: "800 KB", forStudents: true },
  { id: "d-de-2024", title: "Đề thi giữa kỳ An ninh mạng — HK1 2024–2025", file: "de-giua-ky-hk1-2024.pdf", kind: "exam", week: null, uploaded: "02/09", pages: 4, size: "600 KB", forStudents: true },
  { id: "k-de-2025", title: "Đáp án đề cuối kỳ HK1 2025–2026", file: "dap-an-cuoi-ky-hk1-2025.pdf", kind: "answer", week: null, uploaded: "02/09", pages: 5, size: "260 KB", forStudents: false },
  { id: "k-de-2024", title: "Đáp án đề giữa kỳ HK1 2024–2025", file: "dap-an-giua-ky-hk1-2024.pdf", kind: "answer", week: null, uploaded: "02/09", pages: 3, size: "240 KB", forStudents: false },
];

export const docById = (id: string) => DOCS.find((d) => d.id === id);

/** Trích dẫn của AI luôn dùng đúng tên hiển thị của bảng này; `locator` ví dụ "trang 14–17" hoặc "mục 2.4". */
export function docCite(id: string, locator: string): Citation {
  const d = docById(id);
  if (!d) throw new Error(`Tài liệu không có trong bảng N5: ${id}`);
  return { title: d.title, locator, href: "/library" };
}
