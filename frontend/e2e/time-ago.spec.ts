import { expect, test } from "@playwright/test";
import { timeAgo } from "../src/shared/lib/timeAgo";

// US-P2-08 AC11 — thời điểm tương đối theo NGÀY LỊCH giờ Việt Nam (UTC+7), không theo 24 giờ trượt.
const now = new Date("2026-10-03T10:00:00Z"); // 17:00 ICT, 03/10

test("timeAgo: vừa xong, phút, giờ (cùng ngày lịch), hôm qua kèm giờ, dd/MM", () => {
  const at = (ms: number) => new Date(now.getTime() - ms).toISOString();
  expect(timeAgo(at(0), now)).toBe("vừa xong");
  expect(timeAgo(at(59_000), now)).toBe("vừa xong");
  expect(timeAgo(at(60_000), now)).toBe("1 phút trước");
  expect(timeAgo(at(59 * 60_000), now)).toBe("59 phút trước");
  expect(timeAgo(at(60 * 60_000), now)).toBe("1 giờ trước");
  expect(timeAgo(at(9 * 3_600_000), now)).toBe("9 giờ trước"); // 08:00 cùng ngày 03/10
  expect(timeAgo(new Date(now.getTime() + 5 * 60_000).toISOString(), now)).toBe("vừa xong"); // đồng hồ máy lệch
  expect(timeAgo("không phải ngày", now)).toBe("vừa xong");
});

test("timeAgo: qua nửa đêm giờ Việt Nam là 'hôm qua' dù chưa đủ 24 giờ", () => {
  const justAfterMidnight = new Date("2026-10-02T17:30:00Z"); // 00:30 ICT 03/10
  expect(timeAgo("2026-10-02T16:40:00Z", justAfterMidnight)).toBe("hôm qua 23:40"); // chỉ 50 phút trước nhưng khác ngày lịch
  expect(timeAgo("2026-10-02T09:40:00Z", now)).toBe("hôm qua 16:40");
  expect(timeAgo("2026-10-01T09:40:00Z", now)).toBe("01/10");
  expect(timeAgo("2025-12-31T09:40:00Z", now)).toBe("31/12");
});
