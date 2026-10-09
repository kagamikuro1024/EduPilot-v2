import { expect, test, type Page } from "@playwright/test";
import { loadAudit, runAudit } from "./support/audit";
import { rgbOf } from "./support/colors";
import { settleGoto } from "./support/hydrate";
import { ROLES, routesFor } from "./support/routes";
import { asDemo, type DemoRole } from "./support/session";

// "Panel có kỷ luật" (D59, FEAT-ui-panels). Chạy trên bản build có /dev/*: `pnpm build:gate` rồi `playwright test panels.spec.ts`.
// US-UI-02 AC1–AC5, AC7, AC8: token, hợp đồng DOM, NEST = 0, STRONG ≤ 3, WALL = 0, primitive trong panel, "chỉ qua token".
test.beforeEach(({ page }) => settleGoto(page));

/** Màu đã giải của một biểu thức CSS (var(...), oklch(...)) ở trang hiện tại, dạng chuỗi `rgb()` của computed style. */
async function resolved(page: Page, prop: "color" | "boxShadow", expr: string) {
  return page.evaluate(([p, e]) => { const s = document.createElement("span"); document.body.appendChild(s); (s.style as unknown as Record<string, string>)[p] = e; const v = getComputedStyle(s)[p as "color"]; s.remove(); return v; }, [prop, expr] as const);
}

const cell = (page: Page, name: string) => page.locator(`[data-cell="${name}"]`);

/** Phép đo luật (NEST, TITLE, STRONG, WALL) trên trang hiện tại — dùng cho route thật và cho ca gieo vi phạm. */
async function violations(page: Page) {
  return page.evaluate(() => {
    const panels = [...document.querySelectorAll<HTMLElement>("[data-ep-panel]")];
    const rects = panels.map((e) => e.getBoundingClientRect());
    let wall = 0;
    for (const r of rects) {
      if (r.top > 800) continue;
      wall = Math.max(wall, rects.filter((q) => q.top <= 800 && Math.abs(q.top - r.top) < 8 && Math.abs(q.width - r.width) < 4).length);
    }
    return {
      panels: panels.length,
      nest: document.querySelectorAll("[data-ep-panel] [data-ep-panel]").length,
      title: document.querySelectorAll("[data-ep-panel] h1, [data-ep-panel] h2").length,
      strongOver: panels.filter((e) => e.querySelectorAll('[data-tone="strong"]').length > 3).length,
      strongNested: document.querySelectorAll('[data-tone="strong"] [data-ep-panel], [data-tone="strong"] [data-ep-panel-section]').length,
      wall: wall >= 3 ? wall : 0,
    };
  });
}

test.describe("tokens", () => {
  test("năm token đúng giá trị phương án (a); token nền cũ đã bị xoá", async ({ page }, info) => {
    test.skip(info.project.name !== "desktop", "đo token một lần");
    await page.goto("/dev/ui");
    const root = () => page.evaluate(() => { const c = getComputedStyle(document.documentElement); return { paper: c.getPropertyValue(["--ep", "paper"].join("-")).trim(), radius: c.getPropertyValue("--ep-radius-panel").trim() }; });
    const { paper, radius } = await root();
    expect(paper, "token nền cũ phải bị xoá").toBe("");
    expect(parseFloat(radius)).toBeGreaterThanOrEqual(12);
    expect(parseFloat(radius)).toBeLessThanOrEqual(16);
    const get = (n: string) => rgbOf(page, `var(${n})`);
    expect(await get("--ep-canvas")).toBe(await rgbOf(page, "oklch(95% 0.008 60)"));
    expect(await get("--ep-surface-strong")).toBe(await rgbOf(page, "oklch(96% 0.012 25)"));
    expect(await get("--ep-panel-border")).toBe(await get("--ep-rule"));
    const shadow = await page.evaluate(() => { const s = document.createElement("span"); document.body.appendChild(s); s.style.boxShadow = "var(--ep-elevation-1)"; const v = getComputedStyle(s).boxShadow; s.remove(); return v; });
    expect(shadow).toBe(await resolved(page, "boxShadow", "0 1px 2px rgb(71 32 37 / .05), 0 6px 18px rgb(71 32 37 / .07)"));
    const htmlBg = await page.evaluate(() => getComputedStyle(document.documentElement).backgroundColor);
    expect(await rgbOf(page, htmlBg)).toBe(await get("--ep-canvas")); // nền trang
  });
});

test.describe("dom contract", () => {
  test("Panel, PanelSection, ô nhấn: nền, viền, bán kính, bóng, padding, h3 (đọc computed ở /dev/ui#panel)", async ({ page }, info) => {
    test.skip(info.project.name !== "desktop", "đo một lần");
    await page.goto("/dev/ui");
    const surface = await resolved(page, "color", "var(--ep-surface)");
    const rule = await resolved(page, "color", "var(--ep-rule)");
    const strong = await resolved(page, "color", "var(--ep-surface-strong)");
    const elev = await resolved(page, "boxShadow", "var(--ep-elevation-1)");
    const css = (loc: ReturnType<typeof cell>) => loc.locator("[data-ep-panel]").first().evaluate((el) => { const c = getComputedStyle(el); return { bg: c.backgroundColor, bw: c.borderTopWidth, bc: c.borderTopColor, r: c.borderTopLeftRadius, sh: c.boxShadow, pad: c.paddingTop, tag: el.tagName }; });
    const md = await css(cell(page, "md · chỉ nội dung"));
    expect(md).toMatchObject({ bg: surface, bw: "1px", bc: rule, r: "14px", sh: elev, pad: "24px" });
    expect((await css(cell(page, "lg · chỉ nội dung"))).pad).toBe("32px");
    expect((await css(cell(page, "none · chỉ nội dung"))).pad).toBe("0px");
    const secs = cell(page, "md · hai PanelSection").locator("[data-ep-panel-section]");
    const second = await secs.nth(1).evaluate((el) => { const c = getComputedStyle(el); return { bt: c.borderTopWidth, btc: c.borderTopColor, r: c.borderTopLeftRadius, sh: c.boxShadow, bl: c.borderLeftWidth, pt: c.paddingTop }; });
    expect(second).toMatchObject({ bt: "1px", btc: rule, r: "0px", sh: "none", bl: "0px", pt: "24px" });
    expect(await secs.first().evaluate((el) => getComputedStyle(el).borderTopWidth)).toBe("0px");
    expect(await secs.first().locator("h3").count()).toBe(1);
    const st = await cell(page, "1 ô nhấn").locator('[data-tone="strong"]').first().evaluate((el) => { const c = getComputedStyle(el); return { bg: c.backgroundColor, r: c.borderTopLeftRadius, bw: c.borderTopWidth, sh: c.boxShadow }; });
    expect(st).toMatchObject({ bg: strong, r: "8px", bw: "0px", sh: "none" });
    // as="section"
    expect(await page.locator("[data-ep-panel]").count()).toBeGreaterThanOrEqual(14);
  });
});

test.describe("nest", () => {
  test("/dev/ui: NEST = 0, TITLE = 0, kể cả khi mở Dialog / Drawer / Popover", async ({ page }, info) => {
    test.skip(info.project.name !== "desktop", "đo một lần");
    test.setTimeout(120_000);
    await page.goto("/dev/ui");
    expect(await violations(page)).toMatchObject({ nest: 0, title: 0, strongOver: 0, strongNested: 0, wall: 0 });
    const openers = page.locator("[data-open-overlay]");
    const n = await openers.count();
    for (let i = 0; i < n; i++) {
      await openers.nth(i).click({ timeout: 2_000 }).catch(() => undefined);
      expect(await violations(page), `sau khi mở lớp nổi #${i}`).toMatchObject({ nest: 0, title: 0 });
      await page.keyboard.press("Escape");
    }
  });
  for (const role of ROLES) {
    test(`route × vai ${role}: NEST = 0, TITLE = 0 (trạng thái có dữ liệu)`, async ({ page, context }, info) => {
      test.skip(info.project.name !== "desktop", "đo một lần");
      test.setTimeout(120_000);
      await asDemo(context, role as DemoRole);
      const bad: string[] = [];
      for (const route of routesFor(role)) {
        await page.goto(route);
        await page.locator("main, body").first().waitFor();
        const v = await violations(page);
        if (v.nest || v.title || v.strongOver || v.wall) bad.push(`${role} ${route}: ${JSON.stringify(v)}`);
      }
      expect(bad).toEqual([]);
    });
  }
});

test.describe("strong cap / kpi wall", () => {
  test("ca gieo vi phạm bị bắt: 4 ô nhấn trong một panel; 3 Panel bằng nhau cùng hàng ở đầu trang", async ({ page }, info) => {
    test.skip(info.project.name !== "desktop", "đo một lần");
    await page.goto("/dev/ui?fixture=strong4");
    await page.locator('[data-part="strong4-fixture"] [data-ep-panel]').waitFor();
    expect((await violations(page)).strongOver, "STRONG ≤ 3 phải bắt được 4 ô").toBe(1);
    await page.goto("/dev/ui?fixture=wall");
    await page.locator('[data-part="wall-fixture"] [data-ep-panel]').first().waitFor();
    expect((await violations(page)).wall, "WALL phải bắt được 3 panel cùng hàng").toBeGreaterThanOrEqual(3);
  });
});

test.describe("primitives in panel", () => {
  test("DataTable, ActionList, Tabs, DefinitionList, Composer: gốc không viền / bóng / bo góc riêng", async ({ page }, info) => {
    test.skip(info.project.name !== "desktop", "đo một lần");
    await page.goto("/dev/ui");
    const subtle = await resolved(page, "color", "var(--ep-surface-subtle)");
    const rule = await resolved(page, "color", "var(--ep-rule)");
    const root = (name: string, sel: string) => cell(page, name).locator(sel).first().evaluate((el) => { const c = getComputedStyle(el); return { bt: c.borderTopWidth, sh: c.boxShadow, r: c.borderTopLeftRadius }; });
    for (const [name, sel] of [["DataTable trong panel", "[data-ep-panel] > *"], ["ActionList trong panel", "ul"], ["Tabs + DefinitionList trong panel", '[role="tablist"]'], ["Tabs + DefinitionList trong panel", "dl"]] as const) {
      expect(await root(name, sel), `${name} · ${sel}`).toMatchObject({ bt: "0px", sh: "none", r: "0px" });
    }
    const th = await cell(page, "DataTable trong panel").locator("th").first().evaluate((el) => getComputedStyle(el).backgroundColor);
    expect(th, "đầu bảng nền --ep-surface-subtle, không phải canvas").toBe(subtle);
    const comp = await cell(page, "Composer trong panel").locator("textarea").first().evaluate((el) => { const box = el.closest("form") as HTMLElement; const c = getComputedStyle(box); return { bt: c.borderTopWidth, bc: c.borderTopColor, sh: c.boxShadow, r: c.borderTopLeftRadius }; });
    expect(comp).toMatchObject({ bt: "1px", bc: rule, sh: "none", r: "0px" });
  });
  test("/dev/ui#panel: AUDIT sạch ở 1440 và 390 px", async ({ page }, info) => {
    test.skip(info.project.name !== "desktop", "tự đặt bề rộng");
    const { AUDIT_SRC } = await loadAudit();
    await page.goto("/dev/ui");
    for (const w of [1440, 390]) {
      await page.setViewportSize({ width: w, height: 900 });
      const a = await runAudit(page, AUDIT_SRC);
      expect(a.ox, `tràn ngang ở ${w}`).toBe(0);
      const over = await page.locator('[data-part="panel-cell"]').evaluateAll((els) => els.filter((e) => e.scrollWidth > e.clientWidth + 1).map((e) => e.getAttribute("data-cell")));
      expect(over, `ô tràn ở ${w}`).toEqual([]);
    }
  });
});

test.describe("token only", () => {
  test("ghi đè năm token → canvas, panel, ô nhấn, viền, bóng đổi đúng (không giá trị cứng trong Panel / khung)", async ({ page }, info) => {
    test.skip(info.project.name !== "desktop", "đo một lần");
    await page.goto("/dev/ui");
    await page.evaluate(() => {
      const r = document.documentElement.style;
      r.setProperty("--ep-canvas", "rgb(1, 2, 3)");
      r.setProperty("--ep-surface", "rgb(4, 5, 6)");
      r.setProperty("--ep-surface-strong", "rgb(7, 8, 9)");
      r.setProperty("--ep-panel-border", "rgb(10, 11, 12)");
      r.setProperty("--ep-elevation-1", "0 2px 3px rgb(13, 14, 15)");
    });
    expect(await page.evaluate(() => getComputedStyle(document.documentElement).backgroundColor)).toBe("rgb(1, 2, 3)");
    const p = await cell(page, "1 ô nhấn").locator("[data-ep-panel]").first().evaluate((el) => { const c = getComputedStyle(el); return { bg: c.backgroundColor, bc: c.borderTopColor, sh: c.boxShadow }; });
    expect(p.bg).toBe("rgb(4, 5, 6)");
    expect(p.bc).toBe("rgb(10, 11, 12)");
    expect(p.sh).toContain("rgb(13, 14, 15)");
    expect(await cell(page, "1 ô nhấn").locator('[data-tone="strong"]').first().evaluate((el) => getComputedStyle(el).backgroundColor)).toBe("rgb(7, 8, 9)");
  });
});

// US-UI-03 AC5 — khung đăng nhập: nền canvas, MỘT Panel ≤ 440 px giữa trang, h1 NGOÀI panel ngay trên nó; ở 375 px panel cách mép 12 px.
const AUTH_ROUTES = ["/login", "/register", "/forgot-password", "/reset-password?token=x", "/verify-email?token=x", "/invite/x"];
test.describe("auth shell", () => {
  for (const route of AUTH_ROUTES) {
    test(`${route}: một Panel, h1 ngoài panel, NEST = 0, TITLE = 0`, async ({ page }, info) => {
      test.skip(info.project.name !== "desktop", "tự đặt bề rộng");
      const { AUDIT_SRC } = await loadAudit();
      for (const w of [1440, 1024, 375]) {
        await page.setViewportSize({ width: w, height: 900 });
        await page.goto(route);
        await page.locator("[data-ep-panel]").first().waitFor();
        const m = await page.evaluate(() => {
          const panels = [...document.querySelectorAll<HTMLElement>("[data-ep-panel]")];
          const r = panels[0].getBoundingClientRect();
          const h1 = document.querySelector("h1");
          return {
            panels: panels.length,
            nested: document.querySelectorAll("[data-ep-panel] [data-ep-panel]").length,
            inside: document.querySelectorAll("[data-ep-panel] h1, [data-ep-panel] h2").length,
            h1Before: !!h1 && h1.nextElementSibling === panels[0],
            width: r.width,
            left: r.left,
            right: window.innerWidth - r.right,
            primary: document.querySelectorAll("[data-variant=primary]").length,
            bg: getComputedStyle(document.querySelector("main")!).backgroundColor,
          };
        });
        expect(m, `${route}@${w}`).toMatchObject({ panels: 1, nested: 0, inside: 0, h1Before: true });
        expect(m.width).toBeLessThanOrEqual(440);
        expect(m.primary).toBeLessThanOrEqual(1);
        expect(await rgbOf(page, m.bg)).toBe(await rgbOf(page, "var(--ep-canvas)"));
        if (w === 375) {
          expect(m.left).toBeCloseTo(12, 0);
          expect(m.right).toBeCloseTo(12, 0);
        }
        const a = await runAudit(page, AUDIT_SRC);
        expect({ ox: a.ox, cut: a.cut }, `AUDIT ${route}@${w}`).toEqual({ ox: 0, cut: [] });
      }
    });
  }
});
