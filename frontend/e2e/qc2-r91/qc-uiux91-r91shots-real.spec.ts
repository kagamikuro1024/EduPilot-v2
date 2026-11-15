import { mkdirSync } from "node:fs";
import { expect, test, type Page } from "@playwright/test";
test.skip(!process.env.QC_REAL, "QC_REAL");
// TC MAC (K7 ảnh sau): chụp ở HEAD, 1440 và 375/390, cho các route của từng story. Ảnh: docs/regression/evidence-9.1/shots/
const OUT = process.env.QC_SHOTS ?? "/Users/kuro/Documents/TA_Agent_qc2/docs/regression/evidence-9.1/shots";
mkdirSync(OUT, { recursive: true });
const PW = process.env.QC_PW ?? "Edupilot#Seed-2026";
async function login(page: Page, user: string) {
  await page.goto("/login");
  await page.locator("input[type=email]").fill(`${user}@edupilot.local`);
  await page.locator("input[type=password]").fill(PW);
  await page.keyboard.press("Enter");
  await page.waitForURL((u) => !u.pathname.startsWith("/login"), { timeout: 45_000 });
  await page.locator("[data-part=topbar]").waitFor({ timeout: 20_000 });
}
async function courseId(page: Page, code: string) {
  return page.evaluate(async (c) => {
    const r = await fetch("/api/v1/auth/refresh", { method: "POST", credentials: "include" });
    const { access_token } = await r.json();
    const m = await (await fetch("/api/v1/me/courses?limit=100", { headers: { authorization: `Bearer ${access_token}` } })).json();
    return m.items.find((i: { course: { class_code: string } }) => i.course.class_code === c).course.id as string;
  }, code);
}
const SHOTS: Array<{ tc: string; user: string; path: string; name: string }> = [
  { tc: "UX02-12", user: "teacher", path: "/threads", name: "threads" },
  { tc: "UX02-12", user: "sv.gioi", path: "/chat", name: "chat-sv" },
  { tc: "UX02-12", user: "teacher", path: "/inbox", name: "inbox" },
  { tc: "UX03-18", user: "teacher", path: "/gradebook", name: "gradebook" },
  { tc: "UX03-18", user: "teacher", path: "/dev/ui", name: "dev-ui" },
  { tc: "UX01-19", user: "teacher", path: "/", name: "home-gv" },
  { tc: "UX05-12", user: "teacher", path: "/inbox", name: "inbox" },
  { tc: "UX10-28", user: "teacher", path: "/", name: "hom-nay-gv" },
  { tc: "UX11-10", user: "sv.gioi", path: "/", name: "home-sv" },
  { tc: "UX14-12", user: "teacher", path: "/grading?tab=queue", name: "grading-queue" },
  { tc: "UX15-21", user: "teacher", path: "/grading", name: "grading-review" },
  { tc: "UX16-23", user: "teacher", path: "/gradebook", name: "gradebook" },
  { tc: "UX12-22", user: "teacher", path: "/attendance", name: "attendance" },
  { tc: "UX13-28", user: "sv.gioi", path: "/chat", name: "chat-sv" },
  { tc: "UX19-14", user: "teacher", path: "/threads", name: "threads" },
  { tc: "UX20-11", user: "sv.gioi", path: "/me", name: "me-sv" },
  { tc: "UX18-10", user: "teacher", path: "/documents", name: "documents" },
  { tc: "UX17-14", user: "teacher", path: "/insights", name: "insights" },
  { tc: "UX17-14", user: "teacher", path: "/observability", name: "observability" },
  { tc: "UX07-17", user: "teacher", path: "/settings/llm", name: "settings-llm" },
];
const users = [...new Set(SHOTS.map((s) => s.user))];
for (const user of users) {
  for (const [w, h] of [[1440, 900], [375, 812]] as const) {
    test(`shots ${user} ${w}`, async ({ browser }, info) => {
      test.skip(info.project.name !== "desktop");
      const ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: w, height: h }, hasTouch: w < 500, isMobile: w < 500 });
      const page = await ctx.newPage();
      const errs: string[] = [];
      page.on("pageerror", (e) => errs.push(e.message));
      await login(page, user);
      const staff = user === "teacher";
      const c1 = staff || user.startsWith("sv.") ? await courseId(page, "761987") : "";
      const done = new Set<string>();
      for (const s of SHOTS.filter((x) => x.user === user)) {
        const key = `${s.name}-${w}`;
        if (done.has(key)) continue;
        done.add(key);
        let path = s.path;
        if (s.name === "grading-review") {
          await page.goto(`/grading?tab=queue&course=${c1}`);
          await page.waitForTimeout(2500);
          const row = page.locator("main tbody tr").first();
          if (await row.count()) { await row.click().catch(() => undefined); await page.waitForURL(/\/grading\//, { timeout: 10_000 }).catch(() => undefined); }
        } else {
          path += (path.includes("?") ? "&" : "?") + `course=${c1}`;
          await page.goto(path);
        }
        await page.locator("main").first().waitFor({ state: "attached" });
        await page.waitForTimeout(2500);
        await page.screenshot({ path: `${OUT}/${s.name}-${w}.png`, fullPage: false });
        const noOverflow = await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth);
        console.log(`SHOT ${s.tc} ${key} overflowOK=${noOverflow} url=${page.url().replace(/^https?:\/\/[^/]+/, "")}`);
      }
      expect(errs, "pageerror").toEqual([]);
      await ctx.close();
    });
  }
}
