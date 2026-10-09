import { expect, test, type Page } from "@playwright/test";
import { loadAudit, runAudit } from "./support/audit";
import { BASE_URL } from "./support/env";
import { rgbOf } from "./support/colors";
import { panelViolations as violations } from "./support/panels";
import { settleGoto } from "./support/hydrate";
import { ROLES, routesFor } from "./support/routes";
import { asDemo, type DemoRole } from "./support/session";
import { mockLlmApi, mockStaffApi, STAFF_DRAFT, STAFF_EXAM } from "./support/staffMock";
import { mockTake, TAKE_COURSE, TAKE_EXAM, type TakeKind } from "./support/takeMock";

// "Panel có kỷ luật" (D59, FEAT-ui-panels). Chạy trên bản build có /dev/*: `pnpm build:gate` rồi `playwright test panels.spec.ts`.
// US-UI-02 AC1–AC5, AC7, AC8: token, hợp đồng DOM, NEST = 0, STRONG ≤ 3, WALL = 0, primitive trong panel, "chỉ qua token".
test.beforeEach(({ page }) => settleGoto(page));

/** Màu đã giải của một biểu thức CSS (var(...), oklch(...)) ở trang hiện tại, dạng chuỗi `rgb()` của computed style. */
async function resolved(page: Page, prop: "color" | "boxShadow", expr: string) {
  return page.evaluate(([p, e]) => { const s = document.createElement("span"); document.body.appendChild(s); (s.style as unknown as Record<string, string>)[p] = e; const v = getComputedStyle(s)[p as "color"]; s.remove(); return v; }, [prop, expr] as const);
}

const cell = (page: Page, name: string) => page.locator(`[data-cell="${name}"]`);

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

// ───────── US-UI-04 — Sinh viên ─────────
const STUDENT_ROUTES = [...new Set([...routesFor("student"), "/settings", "/join/ABC123"])];
const WIDTHS = [1440, 1024, 375] as const;
const heightFor = (w: number) => (w === 375 ? 812 : 900);

test.describe("student home", () => {
  for (const w of WIDTHS) {
    test(`/ @${w}: 2 Panel vùng (Việc nên làm tiếp, Hôm nay), NEST/TITLE/WALL = 0, STRONG ≤ 1, một cột ở 375`, async ({ page, context }, info) => {
      test.skip(info.project.name !== "desktop", "đo cả ba bề rộng trong một ca");
      await asDemo(context, "student");
      const course = { id: "00000000-0000-7000-8000-00000000c001", class_code: "761987" };
      const today = {
        no_course: false, email_verified: true, continue: [],
        recommended: { id: "r1", kind: "EXAM_OPEN", title: "Làm bài thi Tuần 9", reason: "Bài thi đóng lúc 21:00 hôm nay.", urgency: "high", href: "/exams", course: null, estimate_minutes: 45 },
        timeline: [{ at: "2026-10-15T02:00:00Z", ends_at: "2026-10-15T04:30:00Z", title: "Buổi 10 · An ninh mạng", place: "P.302", state: "NOW", course }],
      };
      for (const url of ["**/api/v1/me/today**", "**/api/v1/courses/*/today**"]) await page.route(url, (r) => r.fulfill({ status: 200, contentType: "application/json", headers: { "Access-Control-Allow-Origin": BASE_URL, "Access-Control-Allow-Credentials": "true", Vary: "Origin" }, body: JSON.stringify(today) }));
      await page.setViewportSize({ width: w, height: heightFor(w) });
      await page.goto("/");
      await page.getByRole("heading", { name: "Hôm nay", level: 2 }).waitFor();
      const v = await violations(page);
      expect(v).toMatchObject({ nest: 0, title: 0, wall: 0, strongNested: 0 });
      expect(v.strongMax).toBeLessThanOrEqual(1);
      expect(v.panels, "đúng 2 Panel vùng").toBe(2);
      expect(v.h2).toEqual(["Việc nên làm tiếp", "Hôm nay"]);
      const kpi = await page.locator("main [data-part=kpi], main [data-tone=strong]").count();
      expect(kpi, "không ô số liệu ở đầu trang").toBeLessThanOrEqual(1);
      if (w === 375) {
        const lefts = await page.locator("main > div > [data-ep-panel], main [data-ep-panel]").evaluateAll((els) => els.map((e) => Math.round(e.getBoundingClientRect().left)));
        expect(new Set(lefts).size, "một cột ở 375 px").toBe(1);
      }
    });
  }
});

test.describe("student routes", () => {
  test("16 route Sinh viên: main có ≥ 1 Panel, tiêu đề vùng ngoài panel, NEST/WALL = 0, STRONG ≤ 3, không tràn ngang ở 1440 và 1024", async ({ page, context }, info) => {
    test.skip(info.project.name !== "desktop", "đo một lần");
    test.setTimeout(180_000);
    await asDemo(context, "student");
    await page.route(/\/api\/v1\/courses\/[^/]+\/exams(\?|$)/, (r) => r.fulfill({ status: 200, contentType: "application/json", headers: { "Access-Control-Allow-Origin": BASE_URL, "Access-Control-Allow-Credentials": "true", Vary: "Origin" }, body: JSON.stringify({ items: [], next_cursor: null }) }));
    const bad: string[] = [];
    for (const w of [1440, 1024]) {
      await page.setViewportSize({ width: w, height: 900 });
      for (const route of STUDENT_ROUTES) {
        await page.goto(route);
        await page.locator("main").first().waitFor();
        const v = await violations(page);
        if (v.panels < 1 || v.nest || v.title || v.wall || v.strongOver || v.sw > v.vw) bad.push(`${route}@${w}: ${JSON.stringify({ ...v, left: undefined, right: undefined })}`);
      }
    }
    expect(bad).toEqual([]);
  });
});

test.describe("student mock routes", () => {
  test("/chat, /threads, /threads/[id], /practice/[id], /me, /calendar: panel đúng chỗ; Composer trong panel; bộ lọc ngoài panel", async ({ page, context }, info) => {
    test.skip(info.project.name !== "desktop", "đo một lần");
    test.setTimeout(120_000);
    await asDemo(context, "student");
    await page.goto("/chat");
    await page.locator("[data-part=chat-thread]").waitFor();
    expect(await page.locator("[data-part=chat-thread]").evaluate((e) => !!e.closest("[data-ep-panel]"))).toBe(true);
    expect(await page.locator("main textarea").first().evaluate((e) => !!e.closest("[data-ep-panel]")), "Composer trong panel").toBe(true);
    expect(await page.locator("main [data-ep-panel] [data-ep-panel]").count()).toBe(0);
    await page.goto("/threads");
    await page.getByRole("list", { name: "Thread của lớp" }).waitFor();
    expect(await page.getByRole("list", { name: "Thread của lớp" }).evaluate((e) => !!e.closest("[data-ep-panel]"))).toBe(true);
    expect(await page.getByRole("searchbox", { name: "Tìm thread" }).evaluate((e) => !!e.closest("[data-ep-panel]")), "bộ lọc ngoài panel").toBe(false);
    await page.goto("/threads/t-cbc");
    await page.locator("[data-part=thread-question]").waitFor();
    expect(await page.locator("[data-part=thread-question]").evaluate((e) => !!e.closest("[data-ep-panel]"))).toBe(true);
    await page.goto("/practice/at-symmetric");
    await page.locator("main [data-ep-panel]:visible").first().waitFor();
    expect((await violations(page)).panels).toBeGreaterThanOrEqual(1);
    await page.goto("/me");
    await page.locator("main [data-ep-panel]:visible").first().waitFor();
    const me = await violations(page);
    expect(me.strongMax).toBeLessThanOrEqual(3);
    expect(me.wall).toBe(0);
    await page.goto("/calendar");
    await page.locator("main [data-ep-panel]:visible").first().waitFor();
    expect((await violations(page)).panels).toBeGreaterThanOrEqual(1);
  });
});

test.describe("exams student", () => {
  test("/exams: số Panel = số nhóm có dữ liệu; hàng không có Panel; tiêu đề nhóm ngoài panel; rỗng = một Panel", async ({ page, context }, info) => {
    test.skip(info.project.name !== "desktop", "đo một lần");
    await asDemo(context, "student");
    const row = (id: string, title: string, status: string, my: unknown, score: string | null) => ({ id, title, status, my_attempt: my, my_score: score, instructions: null, kind: "MCQ", duration_minutes: 45, max_score: "10.00", opens_at: "2026-12-01T01:00:00Z", closes_at: "2026-12-01T03:00:00Z" });
    const rows = [
      row("s-1", "Tuần 9", "OPEN", null, null),
      row("s-3", "Tuần 10", "SCHEDULED", null, null),
      row("s-4", "Tuần 8", "PUBLISHED", { id: "a-2", status: "GRADED", deadline_at: "2026-11-24T02:00:00Z", submitted_at: "2026-11-24T01:50:00Z" }, "8.50"),
    ];
    let body = rows;
    await page.route(/\/api\/v1\/courses\/[^/]+\/exams(\?|$)/, (r) => r.fulfill({ status: 200, contentType: "application/json", headers: { "Access-Control-Allow-Origin": BASE_URL, "Access-Control-Allow-Credentials": "true", Vary: "Origin" }, body: JSON.stringify({ items: body, next_cursor: null }) }));
    await page.goto("/exams");
    await page.getByRole("heading", { name: "Đang mở", level: 2 }).waitFor();
    const v = await violations(page);
    expect(v.panels).toBe(3);
    expect(v).toMatchObject({ nest: 0, title: 0, wall: 0 });
    expect(await page.locator("main [data-ep-panel] li [data-ep-panel]").count()).toBe(0);
    for (const g of ["Đang mở", "Sắp tới", "Đã có điểm"]) expect(v.h2).toContain(g);
    body = [];
    await page.goto("/exams");
    await page.getByText("Lớp của bạn chưa có bài thi nào.").waitFor();
    expect((await violations(page)).panels).toBe(1);
  });
});

test.describe("take states", () => {
  const STATES: Array<[TakeKind, string]> = [["intro", "Bắt đầu làm bài"], ["running", "Câu 1/3"], ["submitted", "Điểm sẽ hiện khi bài thi đóng với cả lớp"], ["published", "7,75 / 10,00"]];
  for (const [kind, marker] of STATES) {
    for (const w of WIDTHS) {
      test(`${kind} @${w}: Panel đúng chỗ, NEST/TITLE = 0, thanh trên không phải Panel, AUDIT sạch`, async ({ page }, info) => {
        test.skip(info.project.name !== "desktop", "đo cả ba bề rộng trong một ca");
        await mockTake(page, kind);
        await page.setViewportSize({ width: w, height: heightFor(w) });
        await page.goto(`/exams/${TAKE_EXAM}/take?course=${TAKE_COURSE}`);
        await page.getByText(marker, { exact: false }).first().waitFor();
        const v = await violations(page);
        expect(v, `${kind}@${w}`).toMatchObject({ nest: 0, title: 0, wall: 0, strongNested: 0 });
        expect(v.panels).toBeGreaterThanOrEqual(1);
        expect(v.strongMax).toBeLessThanOrEqual(3);
        if (kind === "running") {
          const bar = await page.locator("[data-part=take-bar], header").filter({ has: page.getByRole("timer") }).first().evaluate((e) => ({ panel: !!e.closest("[data-ep-panel]") || e.hasAttribute("data-ep-panel") }));
          expect(bar.panel, "thanh trên cố định không phải Panel").toBe(false);
          expect(await page.getByRole("radio").first().evaluate((e) => !!e.closest("[data-ep-panel]")), "câu hỏi trong Panel").toBe(true);
        }
        const { AUDIT_SRC } = await loadAudit();
        const a = await runAudit(page, AUDIT_SRC);
        expect({ ox: a.ox, cut: a.cut }, `AUDIT ${kind}@${w}`).toEqual({ ox: 0, cut: [] });
      });
    }
  }
});

test.describe("join|settings", () => {
  for (const route of ["/join", "/join/ABC123", "/settings"]) {
    test(`${route}: Panel chứa biểu mẫu, một nút chính, NEST/TITLE = 0`, async ({ page, context }, info) => {
      test.skip(info.project.name !== "desktop", "đo một lần");
      await asDemo(context, "student");
      await page.goto(route);
      await page.locator("main [data-ep-panel]:visible").first().waitFor();
      const v = await violations(page);
      expect(v).toMatchObject({ nest: 0, title: 0, wall: 0 });
      expect(v.panels).toBeGreaterThanOrEqual(1);
      expect(await page.locator("main [data-variant=primary]").count()).toBeLessThanOrEqual(route === "/settings" ? 3 : 1);
      expect(await page.locator("main form, main input").first().evaluate((e) => !!e.closest("[data-ep-panel]"))).toBe(true);
    });
  }
});

test.describe("mobile gutter", () => {
  for (const w of [375, 390]) {
    test(`mọi Panel con của main cách mép viewport đúng 12 px @${w}; AUDIT + vùng chạm sạch`, async ({ page, context }, info) => {
      test.skip(info.project.name !== "desktop", "đo cả hai bề rộng");
      test.setTimeout(180_000);
      await asDemo(context, "student");
      await page.route(/\/api\/v1\/courses\/[^/]+\/exams(\?|$)/, (r) => r.fulfill({ status: 200, contentType: "application/json", headers: { "Access-Control-Allow-Origin": BASE_URL, "Access-Control-Allow-Credentials": "true", Vary: "Origin" }, body: JSON.stringify({ items: [], next_cursor: null }) }));
      await page.setViewportSize({ width: w, height: 844 });
      const { AUDIT_SRC, TOUCH_SRC } = await loadAudit();
      const bad: string[] = [];
      for (const route of STUDENT_ROUTES) {
        await page.goto(route);
        await page.locator("main [data-ep-panel]:visible").first().waitFor();
        const v = await violations(page);
        const off = v.left.concat(v.right).filter((x) => Math.abs(x - 12) > 0.5);
        if (off.length) bad.push(`${route}@${w} lề: ${JSON.stringify({ left: v.left, right: v.right })}`);
        const a = await runAudit(page, AUDIT_SRC);
        if (a.ox || a.cut.length) bad.push(`${route}@${w} AUDIT ${JSON.stringify({ ox: a.ox, cut: a.cut })}`);
        if (w === 375) {
          const t = await page.evaluate(TOUCH_SRC);
          if (Array.isArray(t) && t.length) bad.push(`${route}@${w} TOUCH ${JSON.stringify(t).slice(0, 200)}`);
        }
      }
      expect(bad).toEqual([]);
    });
  }
});

// ───────── US-UI-05 — Giảng viên / TA ─────────
const STAFF_ROLES = ["teacher", "ta"] as const;
const STAFF_ROUTES = (role: (typeof STAFF_ROLES)[number]) => [...new Set([...routesFor(role), "/students/sv-1", "/grading/s-1", "/settings", "/class/settings", `/exams/${STAFF_EXAM}`, `/exams/${STAFF_EXAM}/results`, ...(role === "teacher" ? [`/exams/${STAFF_EXAM}/similarity`, "/gradebook/scheme"] : [])])];
const okCors = { "Access-Control-Allow-Origin": BASE_URL, "Access-Control-Allow-Credentials": "true", Vary: "Origin" };

async function staffToday(page: Page) {
  const course = { id: "00000000-0000-7000-8000-00000000c001", class_code: "761987" };
  const body = {
    count: 2,
    actions: [
      { id: "a1", kind: "COURSE_SETUP", title: "Thiết lập lớp mới · 761987", reason: "1/4 bước xong: chia sẻ mã lớp → tải quy chế → tạo lịch → tải tài liệu.", urgency: "high", href: "/class/settings", course, steps: [] },
      { id: "a2", kind: "JOIN_REQUESTS", title: "3 yêu cầu vào lớp", reason: "Sinh viên chờ duyệt.", urgency: "normal", href: "/class/members", course },
    ],
    attention: [],
    upcoming: [{ at: "2026-10-15T02:00:00Z", title: "Buổi 10 · An ninh mạng", place: "P.302", course }],
  };
  for (const url of ["**/api/v1/me/today**", "**/api/v1/courses/*/today**"]) await page.route(url, (r) => r.fulfill({ status: 200, contentType: "application/json", headers: okCors, body: JSON.stringify(body) }));
}

test.describe("staff home", () => {
  for (const role of STAFF_ROLES) {
    for (const w of [1440, 1024]) {
      test(`/ ${role} @${w}: Panel "Việc cần xử lý" + "Sắp tới", NEST/TITLE/WALL = 0, STRONG ≤ 1, không ô số liệu`, async ({ page, context }, info) => {
        test.skip(info.project.name !== "desktop", "đo cả hai bề rộng trong một ca");
        await asDemo(context, role);
        await staffToday(page);
        await page.setViewportSize({ width: w, height: 900 });
        await page.goto("/");
        await page.getByRole("heading", { name: "Sắp tới", level: 2 }).waitFor();
        const v = await violations(page);
        expect(v).toMatchObject({ nest: 0, title: 0, wall: 0, strongNested: 0, panels: 2 });
        expect(v.strongMax).toBeLessThanOrEqual(1);
        expect(v.h2).toEqual(["Việc cần xử lý", "Sắp tới"]);
        expect(await page.locator("main [data-part=kpi]").count()).toBe(0);
      });
    }
  }
});

test.describe("members", () => {
  test("/class/members: Tabs ngoài panel; mỗi tab đúng 1 Panel; STRONG ≤ 3; NEST = 0", async ({ page, context }, info) => {
    test.skip(info.project.name !== "desktop", "đo một lần");
    await asDemo(context, "teacher");
    await mockStaffApi(page);
    await page.goto("/class/members");
    for (const tab of ["Thành viên", "Chờ duyệt", "Trợ giảng", "Nhập danh sách"]) {
      await page.getByRole("tab", { name: new RegExp(`^${tab}`) }).click();
      await page.locator("main [data-ep-panel]:visible").first().waitFor();
      const v = await violations(page);
      expect(v, tab).toMatchObject({ panels: 1, nest: 0, title: 0, wall: 0 });
      expect(v.strongMax).toBeLessThanOrEqual(3);
      expect(await page.getByRole("tablist").evaluate((e) => !!e.closest("[data-ep-panel]")), `${tab}: Tabs ngoài panel`).toBe(false);
    }
    expect(await page.locator("main [data-ep-panel-section]").count(), "nạp danh sách: PanelSection").toBeGreaterThanOrEqual(1);
  });
});

test.describe("exams staff", () => {
  test("/exams Staff: số Panel = số nhóm có dữ liệu; một nút chính; h2 ngoài panel; rỗng = 1 Panel", async ({ page, context }, info) => {
    test.skip(info.project.name !== "desktop", "đo một lần");
    await asDemo(context, "teacher");
    await mockStaffApi(page);
    await page.goto("/exams");
    await page.getByRole("heading", { name: "Đang mở", level: 2 }).waitFor();
    const v = await violations(page);
    expect(v).toMatchObject({ nest: 0, title: 0, wall: 0 });
    expect(v.panels).toBe(v.h2.length);
    expect(v.panels).toBeGreaterThanOrEqual(3);
    expect(await page.locator("main [data-ep-panel] li [data-ep-panel]").count()).toBe(0);
    expect(await page.locator("main [data-variant=primary]").count(), "một nút chính").toBe(1);
  });
});

test.describe("exam editor", () => {
  test("/exams/[id]: Tabs ngoài; mỗi tab 1 Panel; NEST = 0 kể cả khi mở Drawer; tổng điểm là 1 ô nhấn; Panel không là hậu duệ của Drawer", async ({ page, context }, info) => {
    test.skip(info.project.name !== "desktop", "đo một lần");
    await asDemo(context, "teacher");
    await mockStaffApi(page);
    await page.goto(`/exams/${STAFF_DRAFT}`);
    for (const [tab, strong] of [["Thông tin", 0], ["Câu hỏi", 1], ["Xem trước", 0]] as const) {
      await page.getByRole("tab", { name: new RegExp(`^${tab}`) }).click();
      await page.locator("main [data-ep-panel]:visible").first().waitFor();
      const v = await violations(page);
      expect(v, tab).toMatchObject({ panels: 1, nest: 0, title: 0, wall: 0, strongMax: strong });
      expect(await page.getByRole("tablist").evaluate((e) => !!e.closest("[data-ep-panel]")), `${tab}: Tabs ngoài panel`).toBe(false);
    }
    await page.getByRole("tab", { name: /^Thông tin/ }).click();
    expect(await page.locator("main [data-ep-panel-section]").count(), "Thông tin: PanelSection").toBeGreaterThanOrEqual(4);
    await page.getByRole("tab", { name: /^Câu hỏi/ }).click();
    await page.getByRole("button", { name: "Thêm câu từ ngân hàng" }).click();
    await page.getByRole("dialog").waitFor();
    const inDrawer = await page.evaluate(() => document.querySelectorAll('[role="dialog"] [data-ep-panel]').length);
    expect(inDrawer, "Drawer không chứa Panel").toBe(0);
    expect((await violations(page)).nest).toBe(0);
  });
});

test.describe("questions", () => {
  test("/questions: Toolbar + bảng trong 1 Panel, 1 nút chính; hàng mở Drawer không chứa Panel", async ({ page, context }, info) => {
    test.skip(info.project.name !== "desktop", "đo một lần");
    await asDemo(context, "teacher");
    await mockStaffApi(page);
    await page.goto("/questions");
    await page.getByRole("table").first().waitFor();
    const v = await violations(page);
    expect(v).toMatchObject({ panels: 1, nest: 0, title: 0, wall: 0 });
    expect(await page.getByRole("table").first().evaluate((e) => !!e.closest("[data-ep-panel]"))).toBe(true);
    expect(await page.getByRole("searchbox").first().evaluate((e) => !!e.closest("[data-ep-panel]")), "Toolbar trong panel").toBe(true);
    expect(await page.locator("main [data-variant=primary]").count()).toBe(1);
  });
});

test.describe("exam results|similarity", () => {
  test("results: 1 Panel, Tabs ngoài, tab Nghi giống nhau chỉ Giảng viên; similarity: bảng cặp 1 Panel, hai khối mã là 2 ô strong ≥ 1024", async ({ page, context }, info) => {
    test.skip(info.project.name !== "desktop", "đo một lần");
    await asDemo(context, "teacher");
    await mockStaffApi(page);
    await page.goto(`/exams/${STAFF_EXAM}/results`);
    await page.getByRole("table").first().waitFor();
    expect(await violations(page)).toMatchObject({ panels: 1, nest: 0, title: 0, wall: 0 });
    expect(await page.getByRole("tablist").evaluate((e) => !!e.closest("[data-ep-panel]"))).toBe(false);
    await expect(page.getByRole("tab", { name: "Nghi giống nhau" })).toBeVisible();
    await page.goto(`/exams/${STAFF_EXAM}/similarity`);
    await page.getByRole("table").first().waitFor();
    expect(await violations(page)).toMatchObject({ panels: 1, nest: 0, title: 0, strongMax: 0 });
    await page.getByRole("row", { name: /Tính tổng|Sinh viên 1/ }).first().click();
    await page.locator("[data-part=pair-view]").waitFor();
    const v = await violations(page);
    expect(v).toMatchObject({ panels: 1, nest: 0, strongMax: 2, strongNested: 0 });
    expect(await page.locator("[data-part=similarity-note]").innerText()).toMatch(/Độ giống chỉ là gợi ý/);
    // TA: không có tab Nghi giống nhau
    const ta = await context.browser()!.newContext({ baseURL: BASE_URL });
    const tp = await ta.newPage();
    await asDemo(ta, "ta");
    await mockStaffApi(tp);
    await tp.goto(`/exams/${STAFF_EXAM}/results`);
    await tp.getByRole("table").first().waitFor();
    await expect(tp.getByRole("tab", { name: "Nghi giống nhau" })).toHaveCount(0);
    await ta.close();
  });
});

test.describe("staff routes", () => {
  for (const role of STAFF_ROLES) {
    test(`bảng 7.2 × vai ${role}: main ≥ 1 Panel, h1/h2 ngoài panel, NEST/WALL = 0, STRONG ≤ 3, không tràn ngang ở 1440 và 1024`, async ({ page, context }, info) => {
      test.skip(info.project.name !== "desktop", "đo một lần");
      test.setTimeout(240_000);
      await asDemo(context, role);
      await mockStaffApi(page);
      await mockLlmApi(page);
      await staffToday(page);
      const bad: string[] = [];
      for (const w of [1440, 1024]) {
        await page.setViewportSize({ width: w, height: 900 });
        for (const route of STAFF_ROUTES(role)) {
          await page.goto(route);
          await page.locator("main").first().waitFor();
          await page.waitForTimeout(250);
          const v = await violations(page);
          const allowNone = route === `/exams/${STAFF_EXAM}/similarity` && role === "ta"; // TA: màn chặn quyền, không có panel
          if ((v.panels < 1 && !allowNone) || v.nest || v.title || v.wall || v.strongOver || v.sw > v.vw) bad.push(`${role} ${route}@${w}: ${JSON.stringify({ ...v, left: undefined, right: undefined })}`);
        }
      }
      expect(bad).toEqual([]);
    });
  }
  test("/attendance và /inbox ở 375: không tràn ngang, không bị cắt, vùng chạm ≥ 44 px, Panel cách mép 12 px", async ({ page, context }, info) => {
    test.skip(info.project.name !== "desktop", "đo một lần");
    await asDemo(context, "teacher");
    await page.setViewportSize({ width: 375, height: 812 });
    const { AUDIT_SRC, TOUCH_SRC } = await loadAudit();
    for (const route of ["/attendance", "/inbox"]) {
      await page.goto(route);
      await page.locator("main [data-ep-panel]:visible").first().waitFor();
      const a = await runAudit(page, AUDIT_SRC);
      expect({ ox: a.ox, cut: a.cut }, `AUDIT ${route}`).toEqual({ ox: 0, cut: [] });
      const t = await page.evaluate(TOUCH_SRC);
      expect(Array.isArray(t) ? t.filter((x: { text?: string }) => !String(x?.text ?? JSON.stringify(x)).includes("Bỏ qua điều hướng")) : t, `TOUCH ${route}`).toEqual([]);
      const v = await violations(page);
      expect(v.left.concat(v.right).filter((x) => Math.abs(x - 12) > 0.5), `lề ${route}`).toEqual([]);
    }
  });
});
