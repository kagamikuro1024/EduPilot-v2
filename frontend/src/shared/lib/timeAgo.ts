// Thời điểm tương đối cho thông báo THẬT, tính theo NGÀY LỊCH (múi giờ Việt Nam) từ đồng hồ thật — nguồn duy nhất của các chuỗi
// "N phút trước" / "N giờ trước" (US-P2-08 AC11). Màn mô phỏng dùng đồng hồ giả lập riêng ở mock/derive.ts.
import { calendarDays, formatWhen } from "./datetime";

const MIN = 60_000;

/** "vừa xong" · "N phút trước" · "N giờ trước" (cùng ngày lịch) · sau đó theo quy ước của `formatWhen` ("Hôm qua, 16:40", "Thứ Hai, 14:00", …). */
export function timeAgo(iso: string, now: Date = new Date()): string {
  const t = new Date(iso);
  const diff = now.getTime() - t.getTime();
  if (Number.isNaN(diff) || diff < MIN) return "vừa xong"; // gồm cả đồng hồ máy lệch về phía trước
  if (calendarDays(t, now) <= 0) return diff < 60 * MIN ? `${Math.floor(diff / MIN)} phút trước` : `${Math.floor(diff / (60 * MIN))} giờ trước`;
  return formatWhen(t, now);
}
