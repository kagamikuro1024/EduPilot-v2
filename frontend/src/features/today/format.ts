const TZ = "Asia/Ho_Chi_Minh";
const LONG = new Intl.DateTimeFormat("vi-VN", { timeZone: TZ, weekday: "long", day: "numeric", month: "numeric" });
const HM = new Intl.DateTimeFormat("vi-VN", { timeZone: TZ, hour: "2-digit", minute: "2-digit", hour12: false });
const DAY = new Intl.DateTimeFormat("vi-VN", { timeZone: TZ, weekday: "short", day: "numeric", month: "numeric" });

/** "Thứ Tư, 14/10" theo giờ Việt Nam. */
export const longDate = (d: Date) => LONG.format(d);
export const hhmm = (iso: string) => HM.format(new Date(iso));
export const shortDay = (iso: string) => DAY.format(new Date(iso));

/** Mã lớp ở chế độ "Tất cả lớp": "lớp 761988". */
export const classLabel = (c: { class_code: string } | null) => (c ? `lớp ${c.class_code}` : "");
