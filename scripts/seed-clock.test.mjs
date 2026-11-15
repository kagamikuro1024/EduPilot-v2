// Chạy: node --test scripts/seed-clock.test.mjs
// P-11-48 (2): seed chạy lại sang ngày khác — `--now` của bước 16 luôn rơi vào ngày của buổi neo (đồng hồ lệch +1, +7, +30 ngày), lượt đầu không đổi.
import test from "node:test";
import assert from "node:assert/strict";
import { anchorNowIso } from "./seed-clock.mjs";

const DAY = 86_400_000;
const anchor = "2026-11-16T02:00:00Z"; // 09:00 VN
const vnDay = (iso) => new Intl.DateTimeFormat("en-CA", { timeZone: "Asia/Ho_Chi_Minh" }).format(new Date(iso));

test("lần đầu (cùng ngày): giữ nguyên đồng hồ", () => {
  const now = Date.parse("2026-11-16T05:30:00Z");
  assert.equal(anchorNowIso(anchor, now), new Date(now).toISOString());
});
for (const k of [1, 7, 30]) {
  test(`chạy lại sau ${k} ngày: --now lùi về ngày của buổi neo, giữ giờ trong ngày`, () => {
    const now = Date.parse("2026-11-16T05:30:00Z") + k * DAY;
    const iso = anchorNowIso(anchor, now);
    assert.equal(vnDay(iso), vnDay(anchor));
    assert.equal(Date.parse(iso) % DAY, now % DAY);
  });
}
test("qua nửa đêm VN: 17:30Z = 00:30 VN hôm sau vẫn tính theo ngày VN", () => {
  const now = Date.parse("2026-11-16T17:30:00Z"); // 17/11 00:30 VN
  assert.equal(vnDay(anchorNowIso(anchor, now)), "2026-11-16");
});
test("buổi neo ở tương lai: không lùi", () => {
  const now = Date.parse("2026-11-10T05:00:00Z");
  assert.equal(anchorNowIso(anchor, now), new Date(now).toISOString());
});
