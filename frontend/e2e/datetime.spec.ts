import { expect, test } from "@playwright/test";
import { dateInputValue, formatDay, formatTime, formatWhen, longDate } from "../src/shared/lib/datetime";

// FEAT-uiux-91 US-02: một mô-đun ngày giờ, luôn múi giờ Việt Nam. Đồng hồ cố định: 2026-11-08 14:00 ICT (Chủ Nhật) = 07:00Z.
const now = new Date("2026-11-08T07:00:00Z");
const ict = (s: string) => new Date(`${s}+07:00`).toISOString();

test("formatWhen: bảng quy ước AC2", () => {
  expect(formatWhen(ict("2026-11-08T09:20:00"), now)).toBe("Hôm nay, 09:20");
  expect(formatWhen(ict("2026-11-07T16:40:00"), now)).toBe("Hôm qua, 16:40");
  expect(formatWhen(ict("2026-11-02T14:00:00"), now)).toBe("Thứ Hai, 14:00"); // 6 ngày trước
  expect(formatWhen(ict("2026-11-09T09:00:00"), now)).toBe("Ngày mai, 09:00");
  expect(formatWhen(ict("2026-11-10T08:00:00"), now)).toBe("Thứ Ba, 08:00"); // tương lai gần
  expect(formatWhen(ict("2026-09-21T14:00:00"), now)).toBe("Thứ Hai, 21/09 · 14:00");
  expect(formatWhen(ict("2025-09-21T14:00:00"), now)).toBe("Chủ Nhật, 21/09/2025 · 14:00");
  expect(formatWhen(ict("2026-11-01T10:00:00"), now)).toBe("Chủ Nhật, 01/11 · 10:00"); // 7 ngày trước: ra khỏi "cùng tuần"
});

test("formatWhen: ranh giới nửa đêm theo giờ Việt Nam, không theo UTC", () => {
  expect(formatWhen(ict("2026-11-07T23:59:00"), now)).toBe("Hôm qua, 23:59");
  expect(formatWhen(ict("2026-11-08T00:01:00"), now)).toBe("Hôm nay, 00:01");
  expect(formatWhen("2026-11-07T17:30:00Z", now)).toBe("Hôm nay, 00:30"); // 00:30 ICT ngày 08/11 dù UTC vẫn là 07/11
});

test("formatDay / formatTime / dateInputValue / longDate", () => {
  expect(formatDay(ict("2026-09-21T14:00:00"), now)).toBe("21/09");
  expect(formatDay(ict("2025-09-21T14:00:00"), now)).toBe("21/09/2025");
  expect(formatTime(ict("2026-11-08T09:05:00"))).toBe("09:05");
  expect(formatTime(ict("2026-11-08T00:00:00"))).toBe("00:00"); // không "24:00"
  expect(dateInputValue("2026-11-07T20:00:00Z")).toBe("2026-11-08");
  expect(longDate(now)).toBe("Chủ Nhật, 08/11");
});

test("chuỗi sai / null → '—', không ném lỗi", () => {
  for (const v of [null, undefined, "", "không phải ngày"]) {
    expect(formatWhen(v, now)).toBe("—");
    expect(formatDay(v, now)).toBe("—");
    expect(formatTime(v)).toBe("—");
  }
  expect(dateInputValue(null)).toBe("");
});

test("múi giờ của máy khác không đổi kết quả (mọi bộ định dạng đặt timeZone tường minh)", () => {
  const saved = process.env.TZ;
  try {
    for (const tz of ["America/Los_Angeles", "UTC", "Pacific/Auckland"]) {
      process.env.TZ = tz;
      expect(formatWhen(ict("2026-11-08T09:20:00"), now)).toBe("Hôm nay, 09:20");
      expect(formatWhen(ict("2026-11-07T23:59:00"), now)).toBe("Hôm qua, 23:59");
    }
  } finally {
    if (saved === undefined) delete process.env.TZ;
    else process.env.TZ = saved;
  }
});
