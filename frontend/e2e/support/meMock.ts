import type { Page } from "@playwright/test";
import { json } from "./cal-fixtures";

/** Gateway giả cho `/me` thật (US-P5-06; proposals #32 B): ba API của chính sinh viên theo hợp đồng thật, cùng thân với `me.spec.ts`. */
export async function mockMe(page: Page) {
  const weeks = Array.from({ length: 8 }, (_, i) => ({ week_start: new Date(Date.UTC(2026, 7, 17 + i * 7)).toISOString().slice(0, 10), minutes: 120 + i * 10, by_area: { CHAT: 60 } }));
  const bodies: Record<string, unknown> = {
    attendance: {
      attendance: { recorded: 9, present: 6, late: 1, excused: 0, absent: 2, sessions: [{ session_no: 7, starts_at: "2026-10-08T02:00:00Z", status: "ABSENT" }, { session_no: 3, starts_at: "2026-09-10T02:00:00Z", status: "ABSENT" }] },
      next_cursor: null,
    },
    participation: { participation: { speak_count: 2, points_total: "0.50", events: [{ date: "2026-10-01", type: "SPEAK", points: "0.25" }, { date: "2026-09-17", type: "SPEAK", points: "0.25" }] }, next_cursor: null },
    "study-time": { weeks, this_week_minutes: 45, tracking: { enabled: true, locked: false } },
  };
  await page.route("**/api/v1/courses/*/me/**", (r) => {
    if (r.request().method() === "OPTIONS") return r.fulfill({ status: 204, headers: json({}).headers });
    const leaf = new URL(r.request().url()).pathname.split("/").pop() ?? "";
    return r.fulfill(json(bodies[leaf] ?? {}));
  });
}
