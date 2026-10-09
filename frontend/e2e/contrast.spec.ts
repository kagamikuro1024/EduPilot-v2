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
