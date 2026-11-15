import type { BrowserContext, Page, Route } from "@playwright/test";
import { COURSE, err, json } from "./att-fixtures";
import { asDemo, type DemoRole } from "./session";

// Gateway giả cho /checkin (US-AS-09). Mỗi lần gọi `POST …/checkins` được ghi lại (thân + URL + header) để test đếm / kiểm token không lộ.
export { COURSE, err, json };
export const SUB = "00000000-0000-7000-8000-0000000000a0"; // `sub` của JWT giả (support/session.ts)
export const TOKEN = "v1.AAAAAAAAAAAAAAAAAAAAAA.BBBBBBBBBBBBBBBBBBBBBB"; // dạng khác gì cũng được: máy chủ giả không kiểm
export const CHECKED_IN_AT = "2026-10-07T00:03:12.000Z"; // 07:03 giờ Việt Nam
export const item = (id: string, code: string, role = "STUDENT") => ({ course: { id, class_code: code, subject_code: "INT1006", name: "An ninh mạng", semester: "2026-2027-HK1", status: "ACTIVE" }, role_in_course: role, enrollment_status: "ACTIVE" });

export type Post = { url: string; headers: Record<string, string>; body: Record<string, unknown> };
type Handler = (r: Route, n: number) => Promise<void> | void;

export async function checkinApi(page: Page, context: BrowserContext, o: { role?: DemoRole; geoCheck?: boolean; selfCheckin?: boolean; post?: Handler } = {}) {
  await asDemo(context, o.role ?? "student");
  await page.route("**/api/v1/me/courses**", (r) => r.fulfill(json({ items: [item(COURSE, "761988")], next_cursor: null })));
  const log = { posts: [] as Post[], urls: [] as string[], headers: [] as string[], course: 0 };
  page.on("request", (r) => {
    if (!r.url().includes("/api/v1/")) return;
    log.urls.push(r.url());
    log.headers.push(JSON.stringify(r.headers()));
  });
  await page.route(`**/api/v1/courses/${COURSE}`, (r) => {
    log.course++;
    return r.fulfill(json({ id: COURSE, class_code: "761988", attendance: { self_checkin: o.selfCheckin ?? true, require: "WEB", geo_check: o.geoCheck ?? false } }));
  });
  await page.route(`**/api/v1/courses/${COURSE}/checkins`, (r, ) => {
    const req = r.request();
    log.posts.push({ url: req.url(), headers: req.headers(), body: JSON.parse(req.postData() ?? "{}") as Record<string, unknown> });
    if (o.post) return o.post(r, log.posts.length);
    return r.fulfill(json({ checked_in_at: CHECKED_IN_AT, round: 1 }, 201));
  });
  return log;
}
