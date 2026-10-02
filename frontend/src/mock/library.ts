// Thư viện của sinh viên (US-PROTO-01, /library). Dữ liệu tài liệu lấy TỪ bảng N5 duy nhất (`mock/docs.ts`,
// SRS 4.8 N5): ở đây chỉ thêm phần riêng của màn thư viện (chủ đề, tóm tắt, đề luyện được).
// Đáp án đề thi (`forStudents = false`) không bao giờ có trong danh sách này.

import { CH5_UPLOADED_AT } from "./derive";
import { DOCS as N5_DOCS, isDocOfCourse, type Doc, type DocKind } from "./docs";

export { DOC_KIND_LABEL } from "./docs";
export type { DocKind };

type Extra = {
  topic: string;
  summary: string;
  /** đề cũ luyện được: mở lượt luyện từ đề này */
  practiceAttemptId?: string;
};

/** Phần mô tả riêng của màn Thư viện, khoá theo id tài liệu của bảng N5. */
const EXTRA: Record<string, Extra> = {
  "d-ch1": { topic: "Tổng quan", summary: "Tam giác CIA, phân loại tác nhân tấn công, quy trình dựng mô hình đe doạ STRIDE." },
  "d-ch2": { topic: "Tấn công mạng", summary: "Quét cổng, giả mạo ARP, từ chối dịch vụ, chiếm phiên đăng nhập và cách phát hiện." },
  "d-ch3": { topic: "Mật mã đối xứng", summary: "AES, các chế độ ECB / CBC / CTR / GCM, véc-tơ khởi tạo, đệm và padding oracle." },
  "d-ch4": { topic: "Hàm băm và chữ ký số", summary: "SHA-2, HMAC, muối cho mật khẩu, quy trình ký và kiểm tra chữ ký số." },
  "d-ch5": { topic: "PKI", summary: "Vòng đời khoá, chuỗi chứng thư, thu hồi chứng thư (CRL / OCSP), TLS bắt tay." },
  "d-threats": { topic: "Tấn công mạng", summary: "Tài liệu tham khảo tiếng Anh: xu hướng mã độc tống tiền, tấn công chuỗi cung ứng." },
  "d-quyche-truong": { topic: "Quy chế", summary: "Điều kiện dự thi, thang điểm, xử lý vi phạm quy chế thi." },
  "d-quyche-mon": {
    topic: "Quy chế",
    summary: "Quá trình 40% / cuối kỳ 60%; +0,25 mỗi lần phát biểu (trần +1,0); −0,5 mỗi buổi vắng không phép từ buổi thứ 3.",
  },
  "d-de-2025": { topic: "Ôn thi", summary: "40 câu trắc nghiệm + 2 câu tự luận, thời gian 90 phút.", practiceAttemptId: "at-symmetric" },
  "d-de-2024": { topic: "Ôn thi", summary: "30 câu trắc nghiệm về mật mã đối xứng, hàm băm và tường lửa.", practiceAttemptId: "at-symmetric" },
};

export type LibraryDoc = Doc & Extra;

/** 10 tài liệu sinh viên được thấy (bảng N5, `forStudents`). */
export const DOCS: LibraryDoc[] = N5_DOCS.filter((d) => d.forStudents).map((d) => ({ ...d, ...EXTRA[d.id] }));

/** Lớp 2 không có quy chế môn học của lớp 1 → 9 tài liệu (SRS 4.8 N5). */
export function libraryDocs(courseId: string): LibraryDoc[] {
  return DOCS.filter((d) => isDocOfCourse(d, courseId));
}

export function docById(id: string) {
  return DOCS.find((d) => d.id === id);
}

/** Ngày tải lên dạng Date để tính "N ngày trước" (N6); Chương 5 có mốc giờ riêng 28/10 14:00. */
export function uploadedAt(d: Doc): Date {
  if (d.id === "d-ch5") return CH5_UPLOADED_AT;
  const [day, month] = d.uploaded.split("/").map(Number);
  return new Date(2026, month - 1, day, 9, 0);
}
