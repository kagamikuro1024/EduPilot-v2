import { expect, test } from "@playwright/test";
import { COLOR_TOOLS } from "./support/colors";
import { settleGoto } from "./support/hydrate";

// US-UI-02 AC6 — tương phản chữ × nền của "panel có kỷ luật" (D59). Mọi ô chữ ≥ 4,5 : 1; amber (viền / chấm / biểu tượng, không phải chữ) ≥ 3 : 1 trên panel.
// Tính từ màu đã giải ở /dev/ui (bản build có /dev/*). Bảng số in ra bằng test.info().attach để dán vào handoff.
test.beforeEach(({ page }) => settleGoto(page));

const TEXTS = ["--ep-ink", "--ep-ink-2", "--ep-ink-3", "--ep-red", "--ep-green", "--ep-blue"];
const GROUNDS = ["--ep-canvas", "--ep-surface", "--ep-surface-strong", "--ep-surface-subtle", "--ep-red-soft"]; // sidebar / thanh trên = --ep-surface

test("ma trận chữ × nền ≥ 4,5 : 1; amber ≥ 3 : 1 trên panel", async ({ page }, info) => {
  test.skip(info.project.name !== "desktop", "đo token một lần");
  await page.goto("/dev/ui");
  const m = (await page.evaluate(`(() => { const t = ${COLOR_TOOLS}; const out = { pairs: [], amber: 0 };
    for (const g of ${JSON.stringify(GROUNDS)}) for (const x of ${JSON.stringify(TEXTS)}) out.pairs.push({ text: x, ground: g, ratio: t.ratio(t.token(x), t.token(g)) });
    out.amber = t.ratio(t.token("--ep-amber"), t.token("--ep-surface")); t.done(); return out; })()`)) as { pairs: Array<{ text: string; ground: string; ratio: number }>; amber: number };
  const table = ["| Chữ \\ Nền | " + GROUNDS.join(" | ") + " |", "|---|" + GROUNDS.map(() => "---|").join("")]
    .concat(TEXTS.map((x) => `| ${x} | ` + GROUNDS.map((g) => m.pairs.find((p) => p.text === x && p.ground === g)!.ratio.toFixed(2)).join(" | ") + " |"))
    .concat([`| amber trên --ep-surface | ${m.amber.toFixed(2)} |`]);
  await info.attach("contrast-table.md", { body: table.join("\n"), contentType: "text/markdown" });
  console.log(table.join("\n"));
  expect(m.pairs.filter((p) => p.ratio < 4.5), "ô chữ dưới 4,5 : 1").toEqual([]);
  expect(m.amber, "amber ≥ 3 : 1 trên panel").toBeGreaterThanOrEqual(3);
});

// US-UI-03 AC2 — chữ trong sidebar / thanh trên (nền --ep-surface): nhãn nhóm (--ep-ink-3), mục nav (--ep-ink-2), mục đang chọn (--ep-ink) trên nền surface-subtle.
test("sidebar: chữ nav và nhãn nhóm ≥ 4,5 : 1 trên nền của chính chúng", async ({ page, context }, info) => {
  test.skip(info.project.name !== "desktop", "đo một lần");
  const { asDemo } = await import("./support/session");
  await asDemo(context, "teacher");
  await page.setViewportSize({ width: 1440, height: 900 });
  await page.goto("/");
  await page.locator("[data-part=sidebar] nav").waitFor();
  const rows = (await page.evaluate(`(() => { const t = ${COLOR_TOOLS}; const out = [];
    const side = getComputedStyle(document.querySelector("[data-part=sidebar]")).backgroundColor;
    const pick = (el, ground, name) => { const g = getComputedStyle(el).backgroundColor; out.push({ name, ratio: t.ratio(t.rgb(getComputedStyle(el).color), t.rgb(g === "rgba(0, 0, 0, 0)" ? ground : g)) }); };
    const nav = document.querySelector("[data-part=sidebar] nav");
    pick(nav.querySelector("[class*=groupLabel]"), side, "nhãn nhóm");
    pick(nav.querySelector("a:not([aria-current])"), side, "mục nav");
    pick(nav.querySelector("a[aria-current=page]"), side, "mục đang chọn");
    t.done(); return out; })()`)) as Array<{ name: string; ratio: number }>;
  console.log(rows.map((r) => `${r.name}: ${r.ratio.toFixed(2)}`).join(" · "));
  for (const r of rows) expect(r.ratio, r.name).toBeGreaterThanOrEqual(4.5);
});
