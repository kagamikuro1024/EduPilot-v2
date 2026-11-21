import { test, type BrowserContext, type Page } from "@playwright/test";
import { mkdirSync, writeFileSync, existsSync, readFileSync } from "node:fs";
import { gradebookApi } from "../../../../frontend/e2e/support/gradebook-fixtures";
import { base, reset, script } from "../../../../frontend/e2e/support/llm-fixtures";
import { acquireFakeApi, releaseFakeApi } from "../../../../frontend/e2e/support/fake-api-lock";
import { asDemo, asJwt, type DemoRole } from "../../../../frontend/e2e/support/session";
// QC2 sprint 3 «tự kiểm» (GATEPU-11/12/13/15, PU02-46/47, PU05-39/40, P105-51/52/56): chụp + đo DOM trên bản gate (dữ liệu giả cố định).
const OUT = process.env.QC2_R3_OUT ?? "../docs/regression/evidence-3/gate-self";
mkdirSync(OUT, { recursive: true });
const FROZEN = new Date("2026-10-29T09:20:00+07:00");
type R = { name: string; path: string; role?: DemoRole; admin?: boolean };
const ROUTES: R[] = [
  { name: "home", path: "/", role: "student" }, { name: "chat", path: "/chat", role: "student" }, { name: "threads", path: "/threads", role: "student" },
  { name: "practice", path: "/practice", role: "student" }, { name: "library", path: "/library", role: "student" }, { name: "calendar", path: "/calendar", role: "student" }, { name: "me", path: "/me", role: "student" },
  { name: "inbox", path: "/inbox", role: "teacher" }, { name: "gradebook", path: "/gradebook", role: "teacher" },
  { name: "settings-llm", path: "/settings/llm", role: "admin", admin: true }, { name: "dev-ui", path: "/dev/ui" },
];
test.describe.configure({ retries: 0 });
test.beforeEach(async ({}, info) => { test.skip(info.project.name !== "desktop", "tự đặt bề rộng"); });
const metrics = () => {
  const vis = (e: Element) => { const r = e.getBoundingClientRect(); const cs = getComputedStyle(e as HTMLElement); return r.width > 0 && r.height > 0 && cs.visibility !== "hidden" && cs.display !== "none"; };
  const num = (s: string) => parseFloat(s) || 0;
  const framed = (e: Element) => { const cs = getComputedStyle(e as HTMLElement); const r = e.getBoundingClientRect(); const bw = num(cs.borderTopWidth) + num(cs.borderLeftWidth); const tag = e.tagName; if (["BUTTON", "INPUT", "SELECT", "TEXTAREA", "TD", "TH", "TR", "LI", "A", "LABEL", "SVG", "IMG", "KBD", "SPAN"].includes(tag)) return false; return vis(e) && r.width > 180 && r.height > 56 && bw > 0 && cs.borderTopStyle !== "none" && cs.borderTopColor !== "rgba(0, 0, 0, 0)" && num(cs.borderTopLeftRadius) >= 6; };
  const main = (document.querySelector("main") ?? document.body) as HTMLElement;
  const frames = [...main.querySelectorAll("*")].filter(framed);
  const nested = frames.filter((e) => frames.some((o) => o !== e && o.contains(e)));
  const isRed = (c: string) => { const m = c.match(/rgba?\((\d+), (\d+), (\d+)(?:, ([\d.]+))?\)/); if (!m) return false; const [r, g, b] = [+m[1], +m[2], +m[3]]; const a = m[4] === undefined ? 1 : +m[4]; return a > 0.6 && r > 150 && g < 80 && b < 90; };
  const solidRed = [...document.querySelectorAll("button, a[role=button], [role=button]")].filter((b) => vis(b) && isRed(getComputedStyle(b).backgroundColor)).map((b) => ((b as HTMLElement).innerText || b.getAttribute("aria-label") || "").trim().slice(0, 40));
  const redText = [...main.querySelectorAll("*")].filter((e) => e.children.length === 0 && vis(e) && (e as HTMLElement).innerText?.trim() && isRed(getComputedStyle(e).color)).map((e) => (e as HTMLElement).innerText.trim().slice(0, 40));
  const h = [...document.querySelectorAll("h1,h2,h3")].map((e) => (e as HTMLElement).innerText.trim()).slice(0, 14);
  return { nestedCards: nested.length, nestedSample: nested.slice(0, 3).map((e) => e.tagName + "." + String((e as HTMLElement).className).split(" ")[0]), framedCount: frames.length, solidRed, redText: redText.slice(0, 12), headings: h, text: (document.body as HTMLElement).innerText.replace(/\s+/g, " ").slice(0, 6000), ox: document.documentElement.scrollWidth - window.innerWidth };
};
const results: Record<string, unknown> = existsSync(`${OUT}/metrics.json`) ? JSON.parse(readFileSync(`${OUT}/metrics.json`, "utf8")) : {};
async function settle(page: Page) { await page.evaluate(() => document.fonts.ready); await page.waitForLoadState("networkidle"); await page.waitForTimeout(400); }
for (const r of ROUTES) for (const w of [1440, 390]) {
  test(`${r.name} @${w}`, async ({ page, context }) => {
    if (r.admin) await acquireFakeApi();
    try {
      await page.setViewportSize({ width: w, height: w === 1440 ? 900 : 844 }); await page.clock.setFixedTime(FROZEN);
      if (r.admin) { await reset(page); await script(page, base()); await asJwt(page, "ADMIN", { email: "admin@ptit.edu.vn", fullName: "Đỗ Hoàng Nam", expIn: 366 * 86400 }); }
      else if (r.role) await asDemo(context as BrowserContext, r.role, { expIn: 366 * 86400 });
      if (r.path === "/gradebook") await gradebookApi(page);
      await page.goto(r.path); await page.locator("main").first().waitFor(); await settle(page);
      const m = await page.evaluate(metrics); results[`${r.name}@${w}`] = m;
      await page.screenshot({ path: `${OUT}/${r.name}-${w}.png`, fullPage: false });
      writeFileSync(`${OUT}/metrics.json`, JSON.stringify(results, null, 1));
    } finally { if (r.admin) releaseFakeApi(); }
  });
}
