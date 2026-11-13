// Nguồn DUY NHẤT của chữ ngày giờ hiển thị (FEAT-uiux-91 US-02): luôn múi giờ Việt Nam, bất kể múi giờ của máy; chuỗi sai / null → "—".
// Quy ước: "Hôm nay, 09:20" · "Hôm qua, 16:40" · "Ngày mai, 09:00" · "Thứ Hai, 14:00" (trong 6 ngày, cả tương lai gần) · "Thứ Hai, 21/09 · 14:00" · "Thứ Hai, 21/09/2025 · 14:00" (khác năm).
const TZ = "Asia/Ho_Chi_Minh";
const PARTS = new Intl.DateTimeFormat("en-GB", { timeZone: TZ, year: "numeric", month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit", hourCycle: "h23", weekday: "short" });
const WEEKDAY: Record<string, string> = { Mon: "Thứ Hai", Tue: "Thứ Ba", Wed: "Thứ Tư", Thu: "Thứ Năm", Fri: "Thứ Sáu", Sat: "Thứ Bảy", Sun: "Chủ Nhật" };
const DASH = "—";

type Input = string | Date | null | undefined;
type Parts = { y: string; m: string; d: string; hh: string; mm: string; wd: string };

function toDate(v: Input): Date | null {
  if (v == null || v === "") return null;
  const d = v instanceof Date ? v : new Date(v);
  return Number.isNaN(d.getTime()) ? null : d;
}
function parts(d: Date): Parts {
  const p = Object.fromEntries(PARTS.formatToParts(d).map((x) => [x.type, x.value]));
  return { y: p.year, m: p.month, d: p.day, hh: p.hour, mm: p.minute, wd: WEEKDAY[p.weekday] ?? "" };
}
const ymd = (p: Parts) => Date.UTC(Number(p.y), Number(p.m) - 1, Number(p.d));

/** Số ngày lịch (giờ Việt Nam) từ `a` đến `b`; `b` muộn hơn ⇒ dương. */
export function calendarDays(a: Date, b: Date) {
  return Math.round((ymd(parts(b)) - ymd(parts(a))) / 86_400_000);
}

/** "09:20" */
export function formatTime(v: Input): string {
  const d = toDate(v);
  if (!d) return DASH;
  const p = parts(d);
  return `${p.hh}:${p.mm}`;
}

/** "21/09" (cùng năm với `now`) hoặc "21/09/2025". */
export function formatDay(v: Input, now: Date = new Date()): string {
  const d = toDate(v);
  if (!d) return DASH;
  const p = parts(d);
  return p.y === parts(now).y ? `${p.d}/${p.m}` : `${p.d}/${p.m}/${p.y}`;
}

/** Ngày + giờ theo quy ước ở đầu tệp. */
export function formatWhen(v: Input, now: Date = new Date()): string {
  const d = toDate(v);
  if (!d) return DASH;
  const p = parts(d);
  const days = calendarDays(d, now);
  const hm = `${p.hh}:${p.mm}`;
  if (days === 0) return `Hôm nay, ${hm}`;
  if (days === 1) return `Hôm qua, ${hm}`;
  if (days === -1) return `Ngày mai, ${hm}`;
  if (days >= -6 && days <= 6) return `${p.wd}, ${hm}`;
  const sameYear = p.y === parts(now).y;
  return `${p.wd}, ${p.d}/${p.m}${sameYear ? "" : `/${p.y}`} · ${hm}`;
}

/** "Thứ Tư, 14/10": tiêu đề ngày của "Hôm nay". */
export function longDate(d: Date): string {
  const p = parts(d);
  return `${p.wd}, ${p.d}/${p.m}`;
}

/** "2026-11-08": giá trị cho `<input type="date">` theo ngày lịch Việt Nam. */
export function dateInputValue(v: Input): string {
  const d = toDate(v);
  if (!d) return "";
  const p = parts(d);
  return `${p.y}-${p.m}-${p.d}`;
}
