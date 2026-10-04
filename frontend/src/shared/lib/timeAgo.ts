// Thời điểm tương đối cho thông báo THẬT, tính theo NGÀY LỊCH (múi giờ Việt Nam) từ đồng hồ thật — nguồn duy nhất của các chuỗi
// "N phút trước" / "N giờ trước" (US-P2-08 AC11). Màn mô phỏng dùng đồng hồ giả lập riêng ở mock/derive.ts.
const TZ = "Asia/Ho_Chi_Minh";
const DAY = new Intl.DateTimeFormat("en-CA", { timeZone: TZ, year: "numeric", month: "2-digit", day: "2-digit" }); // 2026-10-03
const HM = new Intl.DateTimeFormat("vi-VN", { timeZone: TZ, hour: "2-digit", minute: "2-digit", hour12: false });
const DM = new Intl.DateTimeFormat("en-GB", { timeZone: TZ, day: "2-digit", month: "2-digit" }); // en-GB: "dd/MM" ổn định (vi-VN tuỳ bản ICU)

const MIN = 60_000;

/** Số ngày lịch từ `a` đến `b` (b muộn hơn ⇒ dương). */
function calendarDays(a: Date, b: Date) {
  const d = (x: Date) => Date.parse(`${DAY.format(x)}T00:00:00Z`);
  return Math.round((d(b) - d(a)) / 86_400_000);
}

/** "vừa xong" · "N phút trước" · "N giờ trước" (cùng ngày lịch) · "hôm qua 16:40" · "dd/MM". */
export function timeAgo(iso: string, now: Date = new Date()): string {
  const t = new Date(iso);
  const diff = now.getTime() - t.getTime();
  if (Number.isNaN(diff) || diff < MIN) return "vừa xong"; // gồm cả đồng hồ máy lệch về phía trước
  const days = calendarDays(t, now);
  if (days <= 0) return diff < 60 * MIN ? `${Math.floor(diff / MIN)} phút trước` : `${Math.floor(diff / (60 * MIN))} giờ trước`;
  if (days === 1) return `hôm qua ${HM.format(t)}`;
  return DM.format(t);
}
