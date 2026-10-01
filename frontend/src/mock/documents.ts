// Tài liệu của giảng viên (/documents — US-PROTO-03). Dữ liệu lấy TỪ bảng N5 duy nhất (`mock/docs.ts`, SRS 4.8 N5):
// ở đây chỉ thêm phần riêng của màn quản lý — trạng thái xử lý, cờ "dùng cho AI" và tệp mẫu lỗi khi tải lên.

import { DOCS as N5_DOCS, type Doc } from "./docs";

export { DOC_KIND_LABEL } from "./docs";
export type { Doc };

export type DocStatus = "READY" | "PROCESSING" | "FAILED";

export type DocRow = Doc & {
  status: DocStatus;
  /** dùng làm nguồn cho AI trả lời; đáp án luôn `false` */
  forAi: boolean;
  failReason?: string;
  failFix?: string;
};

export const ANSWER_KEY_NOTE = "Không hiển thị cho sinh viên · Không dùng cho AI của sinh viên";

/** Seed: mọi dòng của bảng N5 đều `READY`; đáp án không hiện cho SV và không dùng cho AI. */
export const DOCUMENTS: DocRow[] = N5_DOCS.map((d) => ({ ...d, status: "READY", forAi: d.forStudents }));

/** Tệp mẫu dùng để minh hoạ nhánh lỗi khi tải lên (SRS 4.5). */
export const FAILING_UPLOAD = {
  name: "scan-khong-co-chu.pdf",
  failReason: "File không có lớp chữ, không đọc được",
  failFix: "Đây là bản scan ảnh. Quét lại có nhận dạng chữ (OCR) hoặc tải lên bản PDF gốc, rồi thử lại.",
};

/** Tên hiển thị tạm của tệp vừa tải lên: bỏ đuôi, gạch nối / gạch dưới thành dấu cách (E33 — danh sách không có tên tệp thô). */
export function titleFromFile(file: string): string {
  const base = file.replace(/\.[a-z0-9]+$/i, "").replace(/[_-]+/g, " ").trim();
  return base.charAt(0).toUpperCase() + base.slice(1);
}
