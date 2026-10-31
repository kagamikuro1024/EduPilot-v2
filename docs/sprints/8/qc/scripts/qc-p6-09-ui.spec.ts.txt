import { expect, test, type Page } from "@playwright/test";
import { COURSE, err, json } from "./support/cal-fixtures";
import { asDemo } from "./support/session";

// US-P5-06: `/me` THẬT của sinh viên. Gateway giả theo hợp đồng thật (openapi.yaml): `…/me/attendance`, `…/me/participation`, `…/me/study-time`.
const C2 = "00000000-0000-7000-8000-00000000c002";
const att = (over: Record<string, unknown> = {}) => ({
  attendance: {
    recorded: 9, present: 6, late: 1, excused: 0, absent: 2,
    sessions: [
      { session_no: 7, starts_at: "2026-10-08T02:00:00Z", status: "ABSENT" },
      { session_no: 5, starts_at: "2026-09-24T02:00:00Z", status: "LATE" },
      { session_no: 3, starts_at: "2026-09-10T02:00:00Z", status: "ABSENT" },
      { session_no: 1, starts_at: "2026-08-27T02:00:00Z", status: "PRESENT" },
    ],
    ...over,
  },
  next_cursor: null,
});
const part = (over: Record<string, unknown> = {}) => ({
  participation: { speak_count: 2, points_total: "0.50", events: [{ date: "2026-10-01", type: "SPEAK", points: "0.25" }, { date: "2026-09-17", type: "SPEAK", points: "0.25" }], ...over },
  next_cursor: null,
});
const weeks = Array.from({ length: 8 }, (_, i) => ({ week_start: new Date(Date.UTC(2026, 7, 17 + i * 7)).toISOString().slice(0, 10), minutes: 120 + i * 10, by_area: { CHAT: 60 } }));
const study = (over: Record<string, unknown> = {}) => ({ weeks, this_week_minutes: 45, tracking: { enabled: true, locked: false }, ...over });

type Handlers = Partial<Record<"attendance" | "participation" | "study-time" | "grades" | "what-if", () => unknown>>;
/** `calls` đếm theo lớp + đường; mỗi handler trả thân JSON, `null` = 500. */
async function api(page: Page, course: string, h: Handlers = {}, calls: string[] = []) {
  await page.route(`**/api/v1/courses/${course}/me/**`, (r) => {
    const req = r.request();
    if (req.method() === "OPTIONS") return r.fulfill({ status: 204, headers: json({}).headers });
    const leaf = new URL(req.url()).pathname.split("/").pop() as keyof Handlers;
    calls.push(`${course}:${leaf}`);
    const fallback = { attendance: att, participation: part, "study-time": study, grades: () => ({ status: "NO_SCHEME" }), "what-if": () => ({ hp: "7", letter: "B" }) }[leaf];
    const body = (h[leaf] ?? fallback)();
    return body === null ? r.fulfill(err(500, "INTERNAL")) : r.fulfill(json(body));
  });
  return calls;
}
import { mkdirSync, writeFileSync } from "node:fs";
const OUT = "../docs/sprints/8/qc/shots"; mkdirSync(OUT, { recursive: true });
const comp = (key: string, name: string, weight: string, score?: string) => ({ key, name, weight, status: score ? "PUBLISHED" : "PENDING", ...(score ? { score } : {}) });
const inProg = () => ({ status: "IN_PROGRESS", components: [comp("midterm", "Giữa kỳ", "20", "8.25"), comp("weekly", "Kiểm tra hằng tuần", "20", "8"), comp("final", "Thi cuối kỳ", "60")], qt: "8.25",
  calculation: [{ label: "Giữa kỳ (20%)", expr: "8,25", value: "8,25" }, { label: "Điểm cộng", expr: "min(0,5; 1,0)", value: "0,5", source_quote: "Mỗi lần phát biểu được cộng 0,25 điểm.", page: 2 }, { label: "Điểm quá trình", expr: "min(10; max(0; 8,25 + 0,5 − 0))", value: "8,25" }],
  official_note: "Điểm chính thức nằm ở hệ thống quản lý đào tạo của trường." });
const fin = () => ({ status: "FINAL", components: [comp("final", "Thi cuối kỳ", "60", "7")], qt: "7", final: { numeric_grade: "7.0", letter: "B", version: 1, locked_at: "2026-10-30T03:00:00Z" }, calculation: [{ label: "Điểm quá trình", expr: "7", value: "7" }], official_note: "Điểm chính thức nằm ở hệ thống quản lý đào tạo của trường." });
test.beforeEach(async ({ context }, i) => { test.skip(i.project.name !== "desktop"); await asDemo(context, "student", { person: "sv-2" }); });
const BAD = /streak|xếp hạng|so với lớp|so với bạn|Tuyệt vời|cố lên|🎉|huy hiệu|vòng tiến độ|hạng \d/i;
for (const [name, grades, vp] of [["inprogress-1440", inProg, 1440], ["inprogress-375", inProg, 375], ["final-1440", fin, 1440], ["noscheme-1440", () => ({ status: "NO_SCHEME" }), 1440]] as const) {
  test(`dump ${name}`, async ({ page }) => {
    await page.setViewportSize({ width: vp, height: vp === 375 ? 812 : 900 });
    await api(page, COURSE, { grades: grades as any });
    await page.goto("/me"); await expect(page.getByRole("heading", { name: "Kết quả của tôi" })).toBeVisible(); await page.waitForTimeout(600);
    const txt = await page.locator("main").innerText();
    const sec = await page.locator("main section, main [data-ep-panel]").first().innerText();
    const dot = (txt.match(/\d+\.\d+/g) || []).join(",");
    writeFileSync(`${OUT}/p609-${name}.txt`, `BAD=${BAD.test(txt)} dotDecimals=[${dot}] scroll=${await page.evaluate(() => [document.documentElement.scrollWidth, innerWidth])} officialNote=${await page.getByText("Điểm chính thức nằm ở hệ thống quản lý đào tạo của trường.").count()} whatIfInput=${await page.getByLabel(/cuối kỳ/i).count()}\n---\n${sec}\n`);
    await page.screenshot({ path: `${OUT}/p609-${name}.png`, fullPage: false });
  });
}
