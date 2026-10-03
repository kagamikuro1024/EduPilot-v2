import { readFileSync } from "node:fs";
import { expect, test } from "@playwright/test";
import { settleGoto } from "./support/hydrate";
import { asDemo } from "./support/session";

// US-PU-01 (token, font, chuẩn hoá nền). Chạy trên bản build: `pnpm -C frontend build && pnpm -C frontend exec playwright test tokens.spec.ts`.

test.describe("font", () => {
  test("Be Vietnam Pro tự lưu, 4 độ đậm, không gọi Google", async ({ page }) => {
    const external: string[] = [];
    page.on("request", (r) => {
      if (/fonts\.(googleapis|gstatic)\.com/.test(r.url())) external.push(r.url());
    });
    await page.goto("/login");
    for (const w of [400, 500, 600, 700]) {
      await page.evaluate((weight) => document.fonts.load(`${weight} 16px "Be Vietnam Pro"`, "Ặ Ế Ộ Ử Ữ Ầ"), w);
      expect(await page.evaluate((weight) => document.fonts.check(`${weight} 16px "Be Vietnam Pro"`), w)).toBe(true);
    }
    expect(await page.evaluate(() => getComputedStyle(document.body).fontFamily)).toMatch(/^"?Be Vietnam Pro"?,/);
    expect(await page.evaluate(() => getComputedStyle(document.documentElement).getPropertyValue("--ep-font"))).toMatch(/^"Be Vietnam Pro", "Noto Sans", ui-sans-serif, system-ui,/);
    expect(external).toEqual([]);
  });

  // Góp ý #26 (PM ACCEPTED): bỏ độ đậm 500 để giảm byte phông (LCP); còn 400 / 600 / 700.
  test("layout.tsx chỉ khai báo 3 độ đậm 400/600/700", () => {
    const src = readFileSync("src/app/layout.tsx", "utf8");
    const lines = src.split("\n").filter((l) => l.includes("weight: ["));
    expect(lines).toHaveLength(1);
    expect(lines[0].match(/"(\d+)"/g)).toEqual(['"400"', '"600"', '"700"']);
    expect(src).toMatch(/display:\s*"swap"/);
    expect(src).toMatch(/subsets:\s*\[[^\]]*"vietnamese"/);
  });
});

test.beforeEach(({ page }) => {
  settleGoto(page);
});

test("tabular: số trong bảng canh theo chữ số bằng nhau (/gradebook, giảng viên)", async ({ page, context }) => {
  await asDemo(context, "teacher");
  await page.goto("/gradebook");
  await page.locator("[data-part=topbar]").waitFor();
  const cells = page.locator("td, th");
  await expect(cells.first()).toBeVisible();
  const n = await cells.count();
  expect(n).toBeGreaterThan(0);
  const bad = await cells.evaluateAll((els) => els.filter((e) => !getComputedStyle(e).fontVariantNumeric.includes("tabular-nums")).length);
  expect(bad).toBe(0);
});

test("diacritics: dấu tiếng Việt không bị cắt ở h1, h2, nút, ô nhập", async ({ page, context }) => {
  await asDemo(context, "teacher");
  await page.goto("/students");
  await page.locator("[data-part=topbar]").waitFor();
  const res = await page.evaluate(() => {
    const text = "Ặ Ế Ộ Ử Ữ Ầ";
    const mk = (tag: string, cls = "") => {
      const el = document.createElement(tag);
      if (cls) el.className = cls;
      el.style.width = "200px";
      document.body.appendChild(el);
      return el as HTMLElement;
    };
    const h1 = mk("h1", "ep-page-title");
    const h2 = mk("h2", "ep-section-title");
    h1.textContent = text;
    h2.textContent = text;
    const btn = document.querySelector("main button, button");
    const input = document.querySelector("input:not([type=checkbox]):not([type=radio]):not([type=hidden])") as HTMLInputElement | null;
    if (btn) btn.textContent = text;
    if (input) input.value = text;
    const out: Record<string, boolean> = {};
    for (const [name, el] of Object.entries({ h1, h2, button: btn, input })) {
      out[name] = !!el && el.scrollHeight <= el.clientHeight + 1;
    }
    out.h1Line = parseFloat(getComputedStyle(h1).lineHeight) >= 1.25 * parseFloat(getComputedStyle(h1).fontSize);
    return out;
  });
  expect(res).toEqual({ h1: true, h2: true, button: true, input: true, h1Line: true });
});

test("focus: vòng --ep-focus khi Tab, không hiện khi bấm chuột", async ({ page }) => {
  await page.goto("/login");
  await page.getByLabel("Email").waitFor(); // form đã hydrate: Tab đầu tiên rơi đúng ô đầu
  await page.keyboard.press("Tab");
  const r = await page.evaluate(() => {
    const ring = (() => {
      const p = document.createElement("i");
      p.style.boxShadow = "var(--ep-focus)";
      document.body.appendChild(p);
      const v = getComputedStyle(p).boxShadow;
      p.remove();
      return v;
    })();
    const a = document.activeElement as HTMLElement;
    return { ring, active: getComputedStyle(a).boxShadow, tag: a.tagName };
  });
  expect(r.ring).not.toBe("none");
  expect(r.active).toBe(r.ring);
  const btn = page.locator("button").first();
  await btn.click();
  expect(await btn.evaluate((el) => getComputedStyle(el).boxShadow)).not.toBe(r.ring);
});

test("reduced: giảm chuyển động → transition ≤ 1 ms", async ({ page }) => {
  await page.emulateMedia({ reducedMotion: "reduce" });
  await page.goto("/login");
  const d = await page.locator("button").first().evaluate((el) => parseFloat(getComputedStyle(el).transitionDuration));
  expect(d).toBeLessThanOrEqual(0.001);
});

test("selection + color-scheme", async ({ page }) => {
  await page.goto("/login");
  const r = await page.evaluate(() => {
    const probe = (prop: string, v: string) => {
      const p = document.createElement("i");
      p.style.setProperty(prop, v);
      document.body.appendChild(p);
      const c = getComputedStyle(p).getPropertyValue(prop);
      p.remove();
      return c;
    };
    return {
      sel: getComputedStyle(document.body, "::selection").backgroundColor,
      soft: probe("background-color", "var(--ep-red-soft)"),
      scheme: getComputedStyle(document.documentElement).colorScheme,
    };
  });
  expect(r.sel).toBe(r.soft);
  expect(r.scheme).toBe("light");
});

test("brand: logo thật, cha trong suốt và không bo góc; favicon 200", async ({ page, context, request }) => {
  await asDemo(context, "teacher");
  await page.goto("/");
  const img = page.locator("[data-part=brand] img").first();
  await expect(img).toHaveAttribute("src", /logo-edupilot(-mark)?\.svg$/);
  const css = await img.evaluate((el) => {
    const s = getComputedStyle(el.parentElement as Element);
    return { radius: s.borderRadius, bg: s.backgroundColor };
  });
  expect(css).toEqual({ radius: "0px", bg: "rgba(0, 0, 0, 0)" });
  expect((await request.get("/brand/favicon.svg")).status()).toBe(200);
  expect(await page.locator('link[rel="icon"]').first().getAttribute("href")).toContain("/brand/favicon.svg");
});
