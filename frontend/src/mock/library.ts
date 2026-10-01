// Thư viện của sinh viên (US-PROTO-01, /library): CHỈ tài liệu được phép hiển thị cho sinh viên.
// Đáp án đề thi không có trong danh sách này và không có đường dẫn nào tới chúng.

export type DocKind = "lecture" | "regulation" | "exam";

export const DOC_KIND_LABEL: Record<DocKind, string> = {
  lecture: "Bài giảng",
  regulation: "Quy chế",
  exam: "Đề cũ",
};

export type Doc = {
  id: string;
  title: string;
  kind: DocKind;
  topic: string;
  /** tuần học gắn với tài liệu; quy chế không gắn tuần */
  week?: number;
  file: string;
  pages: number;
  sizeMb: number;
  minsAgo: number;
  summary: string;
  /** đề cũ luyện được: mở lượt luyện từ đề này */
  practiceAttemptId?: string;
};

export const DOCS: Doc[] = [
  {
    id: "d-ch1",
    title: "Chương 1 — Tổng quan an ninh mạng và mô hình đe doạ",
    kind: "lecture",
    topic: "Tổng quan",
    week: 1,
    file: "an-ninh-mang-ch1.pdf",
    pages: 34,
    sizeMb: 2.1,
    minsAgo: 60 * 24 * 60,
    summary: "Tam giác CIA, phân loại tác nhân tấn công, quy trình dựng mô hình đe doạ STRIDE.",
  },
  {
    id: "d-ch2",
    title: "Chương 2 — Tấn công mạng phổ biến",
    kind: "lecture",
    topic: "Tấn công mạng",
    week: 3,
    file: "an-ninh-mang-ch2.pdf",
    pages: 41,
    sizeMb: 2.8,
    minsAgo: 60 * 24 * 45,
    summary: "Quét cổng, giả mạo ARP, từ chối dịch vụ, chiếm phiên đăng nhập và cách phát hiện.",
  },
  {
    id: "d-ch3",
    title: "Chương 3 — Mật mã đối xứng và chế độ vận hành",
    kind: "lecture",
    topic: "Mật mã đối xứng",
    week: 8,
    file: "an-ninh-mang-ch3.pdf",
    pages: 38,
    sizeMb: 3.4,
    minsAgo: 60 * 24 * 18,
    summary: "AES, các chế độ ECB / CBC / CTR / GCM, véc-tơ khởi tạo, đệm và padding oracle.",
  },
  {
    id: "d-ch4",
    title: "Chương 4 — Hàm băm và chữ ký số",
    kind: "lecture",
    topic: "Hàm băm và chữ ký số",
    week: 9,
    file: "an-ninh-mang-ch4.pdf",
    pages: 29,
    sizeMb: 2.2,
    minsAgo: 60 * 24 * 11,
    summary: "SHA-2, HMAC, muối cho mật khẩu, quy trình ký và kiểm tra chữ ký số.",
  },
  {
    id: "d-ch5",
    title: "Chương 5 — Quản lý khoá và PKI",
    kind: "lecture",
    topic: "PKI",
    week: 10,
    file: "an-ninh-mang-ch5.pdf",
    pages: 33,
    sizeMb: 2.6,
    minsAgo: 60 * 20,
    summary: "Vòng đời khoá, chuỗi chứng thư, thu hồi chứng thư (CRL / OCSP), TLS bắt tay.",
  },
  {
    id: "d-threats",
    title: "Modern Network Security Threats",
    kind: "lecture",
    topic: "Tấn công mạng",
    week: 2,
    file: "Mordern_Network_Security_Threats.pdf",
    pages: 72,
    sizeMb: 3.1,
    minsAgo: 60 * 24 * 52,
    summary: "Tài liệu tham khảo tiếng Anh: xu hướng mã độc tống tiền, tấn công chuỗi cung ứng.",
  },
  {
    id: "d-quyche-truong",
    title: "Quy chế đào tạo của trường",
    kind: "regulation",
    topic: "Quy chế",
    file: "Quyche.pdf",
    pages: 48,
    sizeMb: 1.6,
    minsAgo: 60 * 24 * 90,
    summary: "Điều kiện dự thi, thang điểm, xử lý vi phạm quy chế thi.",
  },
  {
    id: "d-quyche-mon",
    title: "Quy chế môn học An ninh mạng – 761987",
    kind: "regulation",
    topic: "Quy chế",
    file: "quy-che-mon-hoc-761987.pdf",
    pages: 4,
    sizeMb: 0.3,
    minsAgo: 60 * 24 * 62,
    summary: "Quá trình 40% / cuối kỳ 60%; +0,25 mỗi lần phát biểu (trần +1,0); −0,5 mỗi buổi vắng không phép từ buổi thứ 3.",
  },
  {
    id: "d-de-2025",
    title: "Đề thi cuối kỳ An ninh mạng — HK1 2025–2026",
    kind: "exam",
    topic: "Ôn thi",
    file: "de-cuoi-ky-hk1-2025.pdf",
    pages: 6,
    sizeMb: 0.8,
    minsAgo: 60 * 24 * 30,
    summary: "40 câu trắc nghiệm + 2 câu tự luận, thời gian 90 phút.",
    practiceAttemptId: "at-symmetric",
  },
  {
    id: "d-de-2024",
    title: "Đề thi giữa kỳ An ninh mạng — HK1 2024–2025",
    kind: "exam",
    topic: "Ôn thi",
    file: "de-giua-ky-hk1-2024.pdf",
    pages: 4,
    sizeMb: 0.6,
    minsAgo: 60 * 24 * 120,
    summary: "30 câu trắc nghiệm về mật mã đối xứng, hàm băm và tường lửa.",
    practiceAttemptId: "at-symmetric",
  },
];

export function docById(id: string) {
  return DOCS.find((d) => d.id === id);
}
