import { longDate as longDateOf } from "@/shared/lib/datetime";

/** "Thứ Tư, 14/10" theo giờ Việt Nam. */
export const longDate = longDateOf;

/** Mã lớp ở chế độ "Tất cả lớp": "lớp 761988". */
export const classLabel = (c: { class_code: string } | null) => (c ? `lớp ${c.class_code}` : "");
