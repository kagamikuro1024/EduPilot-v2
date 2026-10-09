import { mkdirSync, writeFileSync } from "node:fs";
import { resolve } from "node:path";
import { expect, test, type Page } from "@playwright/test";
import { loadAudit, runAudit } from "./support/audit";
import { settleGoto } from "./support/hydrate";
import { asDemo } from "./support/session";

// US-UI-01 (sprint 5.5): ba phương án độ nổi của panel ở /dev/panels. Tệp TẠM — xoá cùng trang dev ở AC8 sau khi chủ dự án chọn (D59).
// Chạy trên bản build có /dev/*: `pnpm build:gate` rồi `playwright test panels-variants.spec.ts`.

test.beforeEach(({ page }) => settleGoto(page));

const SURFACES = ["a", "b", "c"] as const;
const ROLES = ["teacher", "student"] as const;
const DELTA_L = { a: 5, b: 7, c: 3 } as const;

async function open(page: Page, surface: string, role: string) {
  await asDemo(page.context(), role as "teacher" | "student");
  await page.goto(`/dev/panels?surface=${surface}&role=${role}`);
  await page.locator("[data-ep-panel]").first().waitFor();
  await expect(page.locator("html")).toHaveAttribute("data-surface", surface);
}

/** Đo trong trang: màu của token → sRGB 8 bit; tương phản WCAG; L của OKLab. */
const MEASURE = `(() => {
  const probe = document.createElement("span");
  document.body.appendChild(probe);
  const canvas = document.createElement("canvas");
  canvas.width = canvas.height = 1;
  const ctx = canvas.getContext("2d", { willReadFrequently: true });
  const rgb = (css) => {
    ctx.clearRect(0, 0, 1, 1);
    ctx.fillStyle = "#000";
    ctx.fillStyle = css;
    ctx.fillRect(0, 0, 1, 1);
    const d = ctx.getImageData(0, 0, 1, 1).data;
    return [d[0], d[1], d[2]];
  };
  const token = (name) => {
    probe.style.color = "var(" + name + ")";
    return rgb(getComputedStyle(probe).color);
  };
  const lin = (v) => { const c = v / 255; return c <= 0.04045 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4); };
  const lum = ([r, g, b]) => 0.2126 * lin(r) + 0.7152 * lin(g) + 0.0722 * lin(b);
  const ratio = (a, b) => { const x = lum(a), y = lum(b); return (Math.max(x, y) + 0.05) / (Math.min(x, y) + 0.05); };
  const oklabL = ([r, g, b]) => {
    const R = lin(r), G = lin(g), B = lin(b);
    const l = Math.cbrt(0.4122214708 * R + 0.5363325363 * G + 0.0514459929 * B);
    const m = Math.cbrt(0.2119034982 * R + 0.6806995451 * G + 0.1073969566 * B);
    const s = Math.cbrt(0.0883024619 * R + 0.2817188376 * G + 0.6299787005 * B);
    return 0.2104542553 * l + 0.793617785 * m - 0.0040720468 * s;
  };
  const texts = ["--ep-ink", "--ep-ink-2", "--ep-ink-3", "--ep-red", "--ep-green", "--ep-blue"];
  const grounds = ["--ep-canvas", "--ep-surface", "--ep-surface-strong"];
  const pairs = [];
  for (const g of grounds) for (const t of texts) pairs.push({ text: t, ground: g, ratio: ratio(token(t), token(g)) });
  const out = {
    dL: (oklabL(token("--ep-surface")) - oklabL(token("--ep-canvas"))) * 100,
    pairs,
    elevation: getComputedStyle(document.documentElement).getPropertyValue("--ep-elevation-1").trim(),
  };
  probe.remove();
  return out;
})()`;

test.describe("tokens", () => {
  for (const surface of SURFACES) {
    test(`phương án (${surface}): ΔL, tương phản ≥ 4,5, bóng, viền, bán kính`, async ({ page }) => {
      await open(page, surface, "teacher");
      const m = (await page.evaluate(MEASURE)) as { dL: number; pairs: Array<{ text: string; ground: string; ratio: number }>; elevation: string };
      expect(m.dL, `ΔL của (${surface})`).toBeGreaterThanOrEqual(DELTA_L[surface] - 0.4); // 8 bit làm tròn ≤ 0,4 điểm L
      const low = m.pairs.filter((p) => p.ratio < 4.5);
      expect(low, `cặp dưới 4,5 : 1 ở (${surface})`).toEqual([]);
      if (process.env.PANEL_SHOTS) {
        const out = resolve(__dirname, "../../docs/sprints/5.5/handoff/ui-01");
        mkdirSync(out, { recursive: true });
        writeFileSync(resolve(out, `contrast-${surface}.json`), JSON.stringify({ surface, dL: +m.dL.toFixed(2), pairs: m.pairs.map((p) => ({ ...p, ratio: +p.ratio.toFixed(2) })) }, null, 1));
      }
      if (surface === "a") expect(m.elevation).not.toBe("none");
      else expect(m.elevation).toBe("none");
      const panel = page.locator("[data-ep-panel]").first();
      const css = await panel.evaluate((el) => {
        const c = getComputedStyle(el);
        return { border: c.borderTopColor, width: c.borderTopWidth, radius: parseFloat(c.borderTopLeftRadius), shadow: c.boxShadow };
      });
      expect(css.radius).toBeGreaterThanOrEqual(12);
      expect(css.radius).toBeLessThanOrEqual(16);
      if (surface === "b") expect(css.border).toBe("rgba(0, 0, 0, 0)");
      else expect(css.border).not.toBe("rgba(0, 0, 0, 0)");
      if (surface === "a") expect(css.shadow).not.toBe("none");
      else expect(css.shadow).toBe("none");
      // viền đúng bảng 5.2: (a) = --ep-rule; (c) = oklch(80% …) đậm hơn --ep-rule
      const rule = await page.evaluate(`(() => { const p = document.createElement("span"); document.body.appendChild(p); p.style.color = "var(--ep-rule)"; const v = getComputedStyle(p).color; p.remove(); return v; })()`);
      if (surface === "a") expect(css.border).toBe(rule);
      if (surface === "c") expect(css.border).not.toBe(rule);
    });
  }
});

type Probe = {
  panels: number;
  h2: string[];
  headingInPanel: number;
  nested: number;
  strongMax: number;
  strongTotal: number;
  primary: number;
  primaryInFirst: boolean;
  listRows: number[];
  wall: number;
  redBg: string[];
  redArea: number;
  left: number[];
  right: number[];
  vw: number;
  sw: number;
  firstPanelText: string;
  timeline: number;
};

test.describe("sample rules", () => {
  for (const surface of SURFACES) {
    for (const role of ROLES) {
      test(`(${surface}) ${role}: một vùng = một panel, tiêu đề ngoài, không lồng, không tường thẻ, đỏ là tín hiệu`, async ({ page }, info) => {
        await open(page, surface, role);
        const p = (await page.evaluate(() => {
          const redish = (c: string) => {
            const m = c.match(/rgba?\((\d+), (\d+), (\d+)(?:, ([\d.]+))?\)/);
            if (!m || (m[4] !== undefined && Number(m[4]) === 0)) return false;
            return Number(m[1]) > 150 && Number(m[2]) < 110 && Number(m[3]) < 110;
          };
          const panels = [...document.querySelectorAll<HTMLElement>("main [data-ep-panel]")];
          const rects = panels.map((e) => e.getBoundingClientRect());
          // tường thẻ: ≥ 3 panel anh em cùng rộng cùng hàng trong 800 px đầu trang
          let wall = 0;
          for (const r of rects) {
            if (r.top > 800) continue;
            const row = rects.filter((q) => q.top <= 800 && Math.abs(q.top - r.top) < 8 && Math.abs(q.width - r.width) < 4);
            wall = Math.max(wall, row.length);
          }
          const vw = window.innerWidth;
          const vh = window.innerHeight;
          const redBg: string[] = [];
          let redArea = 0;
          for (const el of document.querySelectorAll<HTMLElement>("body *")) {
            const bg = getComputedStyle(el).backgroundColor;
            if (!redish(bg)) continue;
            const r = el.getBoundingClientRect();
            const w = Math.max(0, Math.min(r.right, vw) - Math.max(r.left, 0));
            const h = Math.max(0, Math.min(r.bottom, vh) - Math.max(r.top, 0));
            redArea += w * h;
            if (el.matches("[data-ep-panel], [data-tone=strong], main, body, aside, header") || el.closest("[data-ep-panel]") === el) redBg.push(el.tagName + ":" + bg);
          }
          for (const el of [document.documentElement, document.body, ...panels, ...document.querySelectorAll<HTMLElement>("[data-tone=strong]")]) {
            if (redish(getComputedStyle(el).backgroundColor)) redBg.push(el.tagName + ":canvas/panel/strong đỏ");
          }
          const prim = [...document.querySelectorAll<HTMLElement>("main [data-variant=primary]")];
          return {
            panels: panels.length,
            h2: [...document.querySelectorAll("main h2")].map((h) => (h.textContent ?? "").trim()),
            headingInPanel: document.querySelectorAll("main [data-ep-panel] h1, main [data-ep-panel] h2").length,
            nested: document.querySelectorAll("[data-ep-panel] [data-ep-panel]").length,
            strongMax: Math.max(0, ...panels.map((e) => e.querySelectorAll("[data-tone=strong]").length)),
            strongTotal: document.querySelectorAll("main [data-tone=strong]").length,
            primary: prim.length,
            primaryInFirst: prim.length > 0 && !!panels[0]?.contains(prim[0]),
            listRows: panels.map((e) => e.querySelectorAll("li").length),
            wall,
            redBg,
            redArea: redArea / (vw * vh),
            left: rects.map((r) => Math.round(r.left * 100) / 100),
            right: rects.map((r) => Math.round((vw - r.right) * 100) / 100),
            vw,
            sw: document.documentElement.scrollWidth,
            firstPanelText: panels[0]?.textContent ?? "",
            timeline: document.querySelectorAll('main ol[aria-label] li').length,
          };
        })) as Probe;

        if (role === "teacher") {
          expect(p.h2).toEqual(["Việc cần xử lý hôm nay", "Lớp cần chú ý", "Sắp tới"]);
          expect(p.panels).toBe(3);
          expect(p.listRows[0]).toBeGreaterThanOrEqual(4);
        } else {
          expect(p.h2).toEqual(["Việc nên làm tiếp", "Hôm nay"]);
          expect(p.panels).toBe(2);
          expect(p.firstPanelText).toContain("≈ 15 phút");
          expect(p.firstPanelText).toMatch(/Bạn đã làm|trước giờ đóng/); // lý do
          expect(p.timeline).toBeGreaterThanOrEqual(3);
        }
        expect(p.headingInPanel, "TITLE = 0").toBe(0);
        expect(p.nested, "NEST = 0").toBe(0);
        expect(p.strongMax, "STRONG ≤ 3 mỗi panel").toBeLessThanOrEqual(3);
        expect(p.strongTotal).toBeGreaterThanOrEqual(1);
        expect(p.wall, "WALL = 0 (không có 3 panel anh em cùng hàng)").toBeLessThan(3);
        expect(p.primary, "một nút chính").toBe(1);
        expect(p.primaryInFirst).toBe(true);
        expect(p.redBg, "đỏ không tô nền canvas / panel / ô nhấn").toEqual([]);
        expect(p.redArea, "diện tích đỏ < 8 % khung hình").toBeLessThan(0.08);

        const { AUDIT_SRC, TOUCH_SRC } = await loadAudit();
        const mobile = info.project.name === "mobile";
        for (const w of mobile ? [375, 390] : [1440, 1024]) {
          await page.setViewportSize({ width: w, height: 900 });
          const a = await runAudit(page, AUDIT_SRC);
          expect({ ox: a.ox, cut: a.cut }, `AUDIT ${surface}/${role}@${w}`).toEqual({ ox: 0, cut: [] });
        }
        if (mobile) {
          await page.setViewportSize({ width: 375, height: 900 });
          const m = await page.evaluate(() => {
            const panels = [...document.querySelectorAll<HTMLElement>("main [data-ep-panel]")];
            return { left: panels.map((e) => e.getBoundingClientRect().left), right: panels.map((e) => window.innerWidth - e.getBoundingClientRect().right), sw: document.documentElement.scrollWidth, vw: window.innerWidth };
          });
          expect(m.sw, "không tràn ngang ở 375 px").toBeLessThanOrEqual(m.vw);
          expect(new Set(m.left.map((x) => Math.round(x))).size, "một cột").toBe(1);
          for (const x of m.left) expect(x).toBeCloseTo(12, 0); // lề 12 px (SRS 4.2 điều 8)
          for (const x of m.right) expect(x).toBeCloseTo(12, 0);
          if (role === "student") expect(await page.evaluate(TOUCH_SRC), `vùng chạm ${surface}/${role}`).toEqual([]);
        }
      });
    }
  }
});

// Ảnh để chủ dự án chọn (AC4): 3 phương án × 2 vai × 3 bề rộng. Chỉ chạy khi PANEL_SHOTS=1 (dự án desktop), ghi vào handoff của sprint 5.5.
test.describe("shots", () => {
  test.skip(!process.env.PANEL_SHOTS, "đặt PANEL_SHOTS=1 để chụp 18 ảnh");
  const dir = resolve(__dirname, "../../docs/sprints/5.5/handoff/ui-01");
  for (const surface of SURFACES) {
    for (const role of ROLES) {
      test(`ảnh (${surface}) ${role}`, async ({ page }, info) => {
        test.skip(info.project.name !== "desktop", "chụp một lần ở dự án desktop");
        mkdirSync(dir, { recursive: true });
        await open(page, surface, role);
        for (const width of [1440, 1024, 375]) {
          await page.setViewportSize({ width, height: 900 });
          await page.waitForTimeout(150);
          await page.screenshot({ path: resolve(dir, `${surface}-${role}-${width}.png`), fullPage: false });
        }
      });
    }
  }
});
