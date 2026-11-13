import { longDate as longDateOf } from "@/shared/lib/datetime";

/** "Thứ Tư, 14/10" theo giờ Việt Nam. */
export const longDate = longDateOf;

/** Mã lớp ở chế độ "Tất cả lớp": "lớp 761988". */
export const classLabel = (c: { class_code: string } | null) => (c ? `lớp ${c.class_code}` : "");

/** Một lớp đang chọn thì tiêu đề máy chủ không lặp mã lớp (US-UX91-10 AC3): bỏ " · lớp X", " · X", và "lớp X" giữa câu. */
export const withoutClass = (title: string, c: { class_code: string } | null) =>
  c ? title.replace(` · lớp ${c.class_code}`, "").replace(` · ${c.class_code}`, "").replace(`lớp ${c.class_code} `, "lớp ") : title;
