// Đồng hồ của bước 16 (thời gian học, at-risk): các lệnh `gateway admin seed-study-time | seed-age | seed-activity | crm-risk-run` tính «tuần» theo `--now`.
// Seed chạy lại ở NGÀY KHÁC thì lịch buổi học vẫn là của ngày seed đầu (DB không trống) → buổi neo (buổi 10 của lớp 761987, buổi 4 của 761988 = «hôm nay» lúc seed đầu)
// không còn là hôm nay. Dời `--now` lùi đúng số ngày lệch, giữ nguyên giờ trong ngày: lượt đầu (lệch 0) không đổi, lượt sau ra cùng kết quả.
const VN = new Intl.DateTimeFormat("en-CA", { timeZone: "Asia/Ho_Chi_Minh", year: "numeric", month: "2-digit", day: "2-digit" });
const dayNo = (ms) => Math.floor(Date.parse(`${VN.format(new Date(ms))}T00:00:00Z`) / 86_400_000);

/** `--now` (RFC 3339) cho lệnh seed: `nowMs` lùi bằng số ngày giữa ngày VN của `nowMs` và ngày VN của buổi neo (không lùi nếu buổi neo chưa tới). */
export function anchorNowIso(anchorStartsAtIso, nowMs) {
  const shift = Math.max(0, dayNo(nowMs) - dayNo(Date.parse(anchorStartsAtIso)));
  return new Date(nowMs - shift * 86_400_000).toISOString();
}
