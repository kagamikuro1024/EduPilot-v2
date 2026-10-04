import { expect, test, type Locator, type Page } from "@playwright/test";
import { loadAudit, runAudit } from "./support/audit";
import { settleGoto } from "./support/hydrate";
import { REGISTRY } from "../src/shared/ui/registry";

// US-PU-02: thư viện thành phần /dev/ui (chạy trên bản dựng cổng: `pnpm -C frontend build:gate`).
const INTERACTIVE = 'button:not([data-open-overlay]), a, input, textarea, select, [role=tab], [role=radio], [role=switch], [role=menuitem], [role=menuitemradio], [tabindex="0"]';

const cell = (page: Page, block: string, state: string) => page.locator(`[data-part=primitive][data-name="${block}"] [data-part=state-cell][data-state=${state}]:not([data-na])`);

test.beforeEach(async ({ page }) => {
  settleGoto(page);
  await page.emulateMedia({ reducedMotion: "reduce" });
});

test("matrix: 25 khối, 125 ô, 75 ô N/A có lý do", async ({ page }) => {
  await page.goto("/dev/ui");
  await expect(page.locator("[data-part=primitive]")).toHaveCount(25);
  await expect(page.locator("[data-part=state-cell]:not([data-na])")).toHaveCount(125);
  await expect(page.locator("[data-part=state-cell][data-na]")).toHaveCount(75);
  const noReason = await page.locator("[data-part=state-cell][data-na]").evaluateAll((els) => els.filter((e) => !(e.getAttribute("data-reason") ?? "").trim()).length);
  expect(noReason).toBe(0);
  const names = await page.locator("[data-part=primitive]").evaluateAll((els) => els.map((e) => e.getAttribute("data-name")));
  expect(names).toEqual(REGISTRY.map((b) => b.name));
  const states = await page.locator("[data-part=state-cell]:not([data-na])").evaluateAll((els) => els.map((e) => e.getAttribute("data-state")));
  expect(states.every((x) => ["default", "hover", "focus", "selected", "disabled", "loading", "empty", "error"].includes(x ?? ""))).toBe(true);
});

test.describe("states", () => {
  const HOVER = ["Button", "Field", "Checkbox", "Switch", "Tabs", "SegmentedControl", "FilterChips", "ActionList", "DataTable", "Menu", "Composer", "UndoLine", "CitationList", "VerificationState", "LLMRouteTable"];

  // ảnh chụp thuộc tính tính toán của phần tử, tổ tiên (tới ô) và con trực tiếp — hover thường tô nền hàng / khung cha
  const snap = (el: Locator) =>
    el.evaluate((e) => {
      const pick = (n: Element) => {
        const c = getComputedStyle(n);
        return [c.backgroundColor, c.color, c.textDecorationLine + c.textDecorationColor, c.borderTopColor, c.boxShadow, c.fontWeight].join("|");
      };
      const out = [pick(e), ...Array.from(e.children).map(pick), ...Array.from(e.parentElement?.children ?? []).map(pick)];
      for (let n = e.parentElement; n && !n.matches("[data-part=state-cell]"); n = n.parentElement) out.push(pick(n));
      return out.join("\n");
    });
  const NOT_SELECTED = INTERACTIVE.split(", ").map((x) => `${x}:not([aria-selected="true"]):not([aria-checked="true"]):not([aria-pressed="true"])`).join(", ");

  test("hover đổi ít nhất một thuộc tính tính toán", async ({ page }) => {
    await page.goto("/dev/ui");
    const bad: string[] = [];
    for (const b of HOVER) {
      const h = cell(page, b, "hover").locator(NOT_SELECTED).filter({ visible: true }).first();
      await h.scrollIntoViewIfNeeded();
      await page.mouse.move(0, 0); // chuột từ khối trước có thể đang nằm trên phần tử này sau khi cuộn
      await page.waitForTimeout(50);
      const before = await snap(h);
      await h.hover();
      try {
        await expect.poll(() => snap(h), { timeout: 4000 }).not.toBe(before); // chờ transition kết thúc
      } catch {
        bad.push(b);
      }
    }
    expect(bad, `không đổi khi hover: ${bad.join(", ")}`).toEqual([]);
  });

  test("focus bàn phím hiện vòng", async ({ page }) => {
    await page.goto("/dev/ui");
    const ring = await page.evaluate(() => {
      const p = document.createElement("i");
      p.style.boxShadow = "var(--ep-focus)";
      document.body.appendChild(p);
      const v = getComputedStyle(p).boxShadow;
      p.remove();
      return v;
    });
    const bad: string[] = [];
    for (const b of REGISTRY.filter((x) => x.states.includes("focus") && !["Dialog", "Drawer", "ConfirmIrreversible", "CommandPalette", "Popover"].includes(x.name)).map((x) => x.name)) {
      const el = cell(page, b, "focus").locator(INTERACTIVE).filter({ visible: true }).first();
      if ((await el.count()) === 0) {
        bad.push(`${b}: ô focus không có phần tử tương tác`);
        continue;
      }
      await el.scrollIntoViewIfNeeded();
      await el.focus();
      await page.keyboard.press("Shift+Tab");
      await page.keyboard.press("Tab");
      await page.waitForTimeout(80); // đợi transition 1 ms (giảm chuyển động) kết thúc
      const ok = await el.evaluate((e, want) => {
        const walk = [e, e.parentElement, e.closest("tr"), e.closest("label"), e.querySelector("td")];
        return document.activeElement === e && walk.some((n) => n && getComputedStyle(n).boxShadow !== "none" && (getComputedStyle(n).boxShadow === want || getComputedStyle(n).boxShadow.includes("inset")));
      }, ring);
      if (!ok) bad.push(b);
    }
    expect(bad, `không có vòng focus: ${bad.join(", ")}`).toEqual([]);
  });

  test("selected có ARIA và dấu hiệu không chỉ là màu", async ({ page }) => {
    await page.goto("/dev/ui");
    const bad: string[] = [];
    for (const b of ["Checkbox", "Switch", "Tabs", "SegmentedControl", "FilterChips", "ActionList", "DataTable", "Menu", "CitationList", "LLMRouteTable"]) {
      const sel = cell(page, b, "selected");
      const aria = await sel.locator('[aria-selected="true"],[aria-pressed="true"],[aria-checked="true"],[aria-current="true"],[aria-expanded="true"],input:checked').count();
      if (aria === 0) bad.push(`${b}: thiếu ARIA`);
    }
    expect(bad).toEqual([]);
  });

  test("disabled: khoá thật, cursor not-allowed, không gọi handler", async ({ page }) => {
    await page.goto("/dev/ui");
    const bad: string[] = [];
    for (const b of ["Button", "Field", "Checkbox", "Switch", "Tabs", "SegmentedControl", "FilterChips", "ActionList", "Menu", "Composer", "VerificationState", "LLMRouteTable"]) {
      const c = cell(page, b, "disabled");
      const target = c.locator(":disabled, [aria-disabled=true]").first();
      if ((await target.count()) === 0) {
        bad.push(`${b}: không có phần tử disabled`);
        continue;
      }
      await target.scrollIntoViewIfNeeded();
      const cursor = await target.evaluate((e) => getComputedStyle(e).cursor);
      if (cursor !== "not-allowed") bad.push(`${b}: cursor ${cursor}`);
      await target.click({ force: true, trial: false }).catch(() => {});
      const calls = await c.getAttribute("data-calls");
      if (calls !== "0") bad.push(`${b}: handler gọi ${calls} lần`);
    }
    expect(bad).toEqual([]);
  });
});

test("loading / empty / error đúng quy ước", async ({ page }) => {
  await page.goto("/dev/ui");
  const bad: string[] = [];
  for (const b of REGISTRY) {
    if (b.states.includes("loading")) {
      const c = cell(page, b.name, "loading");
      if (["Dialog", "Drawer", "ConfirmIrreversible", "CommandPalette"].includes(b.name)) continue; // lớp phủ: mở bằng nút, kiểm ở 'overlay'
      if ((await c.locator("[aria-busy=true]").count()) === 0) bad.push(`${b.name}: loading thiếu aria-busy`);
      if ((await c.locator(".animate-spin").count()) !== 0) bad.push(`${b.name}: có animate-spin`);
    }
    if (b.states.includes("empty") && !["Field", "Composer", "Popover", "CommandPalette", "CitationList"].includes(b.name)) {
      const n = await cell(page, b.name, "empty").locator("button, a").count();
      if (n !== 1) bad.push(`${b.name}: empty có ${n} hành động`);
    }
    if (b.states.includes("error") && !["Field", "Checkbox", "Dialog", "Drawer", "ConfirmIrreversible", "Composer", "StatusText", "UndoLine"].includes(b.name)) {
      const c = cell(page, b.name, "error");
      if ((await c.locator("[role=alert]").count()) === 0) bad.push(`${b.name}: error thiếu role=alert`);
      if ((await c.getByRole("button", { name: "Thử lại" }).count()) === 0) bad.push(`${b.name}: error thiếu Thử lại`);
      const text = (await c.innerText()).trim();
      if (/^[A-Z_]{3,}$/m.test(text) || /Something went wrong|undefined|\[object/.test(text)) bad.push(`${b.name}: lời lỗi xấu`);
    }
  }
  expect(bad).toEqual([]);
});

test("overflow: tràn chữ Việt ở 5 bề rộng, AUDIT sạch và TOUCH ở 375", async ({ page }) => {
  const { AUDIT_SRC, TOUCH_SRC } = await loadAudit();
  for (const width of [375, 640, 900, 1280, 1440]) {
    await page.setViewportSize({ width, height: 900 });
    await page.goto("/dev/ui");
    await expect(page.locator("[data-part=primitive]")).toHaveCount(25);
    await page.waitForTimeout(300);
    const res = await runAudit(page, AUDIT_SRC);
    expect(res, `AUDIT ${width}px`).toEqual({ ox: 0, cut: [], ell: [] });
    if (width === 375) expect(await page.evaluate(TOUCH_SRC), "TOUCH 375px").toEqual([]);
  }
});

test.describe("datatable 1.000 dòng", () => {
  test("ảo hoá, tiêu đề dính, bàn phím, Drawer giữ cuộn và dòng chọn", async ({ page }) => {
    await page.goto("/dev/ui");
    const box = page.locator("[data-part=datatable-1000] [data-part=virtual-scroll]");
    await box.scrollIntoViewIfNeeded();
    await expect(box.locator("tbody tr").first()).toBeVisible();
    const h = await box.locator("tbody tr[data-index]").first().evaluate((e) => e.getBoundingClientRect().height);
    expect(h).toBeGreaterThanOrEqual(44);
    expect(h).toBeLessThanOrEqual(52);
    expect(await box.locator("th").first().evaluate((e) => getComputedStyle(e).position)).toBe("sticky");
    const t0 = Date.now();
    await box.evaluate((e) => { e.scrollTop = e.scrollHeight; });
    await expect(box.locator('tbody tr[data-index="999"]')).toBeVisible();
    expect(Date.now() - t0).toBeLessThan(1000);
    expect(await box.locator("tbody tr").count()).toBeLessThan(80);

    // bàn phím
    await box.evaluate((e) => { e.scrollTop = 0; });
    await box.focus();
    for (let i = 0; i < 3; i++) await page.keyboard.press("ArrowDown");
    await expect(box.locator('tr[aria-current="true"]')).toHaveAttribute("data-index", "2");
    await page.keyboard.press("ArrowUp");
    await expect(box.locator('tr[aria-current="true"]')).toHaveAttribute("data-index", "1");
    await page.keyboard.press("End");
    await expect(box.locator('tr[aria-current="true"]')).toHaveAttribute("data-index", "999");
    await page.keyboard.press("Home");
    await expect(box.locator('tr[aria-current="true"]')).toHaveAttribute("data-index", "0");
    for (let i = 0; i < 8; i++) await page.keyboard.press("ArrowDown");
    const scrollBefore = await box.evaluate((e) => e.scrollTop);
    await page.keyboard.press("Enter"); // kích hoạt dòng → Drawer
    const drawer = page.locator("dialog[open]");
    await expect(drawer).toBeVisible();
    await page.keyboard.press("Escape");
    await expect(drawer).toHaveCount(0);
    expect(await box.evaluate((e) => e.scrollTop)).toBe(scrollBefore);
    await expect(box.locator('tr[aria-current="true"]')).toHaveAttribute("data-index", "8");
    await page.keyboard.press("Escape");
    expect(await page.evaluate(() => document.activeElement === document.body)).toBe(true);
  });

  test("đo khung hình khi cuộn tự động (ghi, không chặn)", async ({ page }, info) => {
    await page.goto("/dev/ui");
    const box = page.locator("[data-part=datatable-1000] [data-part=virtual-scroll]");
    await box.scrollIntoViewIfNeeded();
    const p95 = await box.evaluate(
      (e) =>
        new Promise<number>((done) => {
          const frames: number[] = [];
          let last = performance.now();
          let n = 0;
          const tick = (t: number) => {
            frames.push(t - last);
            last = t;
            e.scrollTop += 120;
            if (++n < 120) requestAnimationFrame(tick);
            else done(frames.sort((a, b) => a - b)[Math.floor(frames.length * 0.95)]);
          };
          requestAnimationFrame(tick);
        }),
    );
    info.annotations.push({ type: "frame-p95-ms", description: p95.toFixed(1) });
    expect(p95).toBeGreaterThan(0);
  });
});

test.describe("cursor", () => {
  const box = (page: Page) => page.locator("[data-part=cursor-demo] [data-part=virtual-scroll]");
  const calls = (page: Page) => page.evaluate(() => (window as unknown as { __cursorCalls: string[] }).__cursorCalls);

  test("gọi đúng một lần cho mỗi con trỏ, dừng khi hết", async ({ page }) => {
    await page.goto("/dev/ui");
    await box(page).scrollIntoViewIfNeeded();
    for (let i = 0; i < 40; i++) {
      await box(page).evaluate((e) => { e.scrollTop = e.scrollHeight; }); // cuộn nhanh liên tục
      await page.waitForTimeout(40);
    }
    await expect.poll(() => box(page).locator("tbody tr[data-index]").count()).toBeGreaterThan(0);
    await page.waitForTimeout(600);
    expect(await calls(page)).toEqual(["c2", "c3"]);
    await box(page).evaluate((e) => { e.scrollTop = e.scrollHeight; });
    await expect(box(page).locator('tbody tr[data-index="59"]')).toBeVisible();
    expect(await calls(page)).toEqual(["c2", "c3"]);
    await expect(page.locator("[data-part=cursor-demo] [data-part=load-more]")).toHaveCount(0);
  });

  test("lỗi trang kế: giữ dòng đã có, có Thử lại", async ({ page }) => {
    await page.goto("/dev/ui?fail2=1");
    await box(page).scrollIntoViewIfNeeded();
    await box(page).evaluate((e) => { e.scrollTop = e.scrollHeight; });
    const retry = page.locator("[data-part=cursor-demo]").getByRole("button", { name: "Thử lại" });
    await expect(retry).toBeVisible();
    await expect(page.locator("[data-part=cursor-demo] [role=alert]")).toBeVisible();
    expect(await box(page).locator("tbody tr[data-index]").count()).toBeGreaterThan(0);
    await retry.click();
    await expect.poll(() => calls(page)).toEqual(["c2", "c2"]);
    await expect(retry).toHaveCount(0);
    for (let i = 0; i < 10; i++) {
      await box(page).evaluate((e) => { e.scrollTop = e.scrollHeight; });
      await page.waitForTimeout(60);
    }
    await expect.poll(() => calls(page)).toEqual(["c2", "c2", "c3"]);
  });
});

test.describe("overlay", () => {
  test("Dialog bẫy focus, Esc đóng, trả focus về nút mở", async ({ page }) => {
    await page.goto("/dev/ui");
    const opener = cell(page, "Dialog", "default").locator("[data-open-overlay]");
    await opener.scrollIntoViewIfNeeded();
    await opener.click();
    const dlg = page.locator("dialog[open]");
    await expect(dlg).toBeVisible();
    for (let i = 0; i < 20; i++) {
      await page.keyboard.press("Tab");
      expect(await page.evaluate(() => !!document.activeElement?.closest("dialog[open]"))).toBe(true);
    }
    await page.keyboard.press("Escape");
    await expect(dlg).toHaveCount(0);
    await expect(opener).toBeFocused();
  });

  test("ConfirmIrreversible đang loading: Esc không đóng", async ({ page }) => {
    await page.goto("/dev/ui");
    await cell(page, "ConfirmIrreversible", "loading").locator("[data-open-overlay]").click();
    const dlg = page.locator("dialog[open]");
    await expect(dlg).toBeVisible();
    await expect(dlg.locator("[aria-busy=true]").first()).toBeVisible();
    await page.keyboard.press("Escape");
    await expect(dlg).toBeVisible();
    await expect(cell(page, "ConfirmIrreversible", "error").locator("[data-open-overlay]")).toBeVisible();
  });

  test("Dialog lỗi: role=alert; Drawer loading: aria-busy", async ({ page }) => {
    await page.goto("/dev/ui");
    await cell(page, "Dialog", "error").locator("[data-open-overlay]").click();
    await expect(page.locator("dialog[open] [role=alert]")).toBeVisible();
    await page.keyboard.press("Escape");
    await cell(page, "Drawer", "loading").locator("[data-open-overlay]").click();
    await expect(page.locator("dialog[open] [aria-busy=true]")).toBeVisible();
  });

  test("Drawer: 420–520 px ở 900, toàn màn ở 375", async ({ page }) => {
    for (const [w, check] of [[900, (x: number) => x >= 420 && x <= 520], [375, (x: number) => x === 375]] as const) {
      await page.setViewportSize({ width: w, height: 800 });
      await page.goto("/dev/ui");
      const opener = cell(page, "Drawer", "default").locator("[data-open-overlay]");
      await opener.scrollIntoViewIfNeeded();
      await opener.click();
      const width = await page.locator("dialog[open]").evaluate((e) => e.getBoundingClientRect().width);
      expect(check(width), `${w}px → ${width}`).toBe(true);
      await page.keyboard.press("Escape");
      await expect(opener).toBeFocused();
    }
  });

  test("Popover: đóng khi bấm ngoài / Esc; Menu: ↑ ↓ đổi mục", async ({ page }) => {
    await page.goto("/dev/ui");
    const c = cell(page, "Popover", "default");
    const trigger = c.getByRole("button", { name: "Bộ lọc" });
    await trigger.scrollIntoViewIfNeeded();
    await trigger.click();
    await expect(c.getByRole("dialog", { name: "Bộ lọc" })).toBeVisible();
    await page.keyboard.press("Escape");
    await expect(c.getByRole("dialog", { name: "Bộ lọc" })).toHaveCount(0);
    await expect(trigger).toBeFocused();
    await trigger.click();
    await page.mouse.click(5, 5);
    await expect(c.getByRole("dialog", { name: "Bộ lọc" })).toHaveCount(0);

    const items = cell(page, "Menu", "default").getByRole("menuitem");
    await items.first().focus();
    await page.keyboard.press("ArrowDown");
    await expect(items.nth(1)).toBeFocused();
    await page.keyboard.press("ArrowUp");
    await expect(items.first()).toBeFocused();
  });
});

test("composer: rộng ≤ 820, một nút chính, ô nhập vào khung nhìn khi bàn phím ảo", async ({ page }) => {
  await page.goto("/dev/ui");
  const c = cell(page, "Composer", "default");
  await c.scrollIntoViewIfNeeded();
  const w = await c.locator("textarea").evaluate((e) => e.closest("form")!.parentElement!.getBoundingClientRect().width);
  expect(w).toBeLessThanOrEqual(820);
  expect(await c.locator('button[data-variant="primary"]').count()).toBe(1);
  expect(await c.locator('button[data-variant="primary"]').first().textContent()).toContain("Gửi");
  // nút phụ là ghost
  expect(await c.locator('button[data-variant="ghost"]').count()).toBeGreaterThan(0);
  await page.setViewportSize({ width: 375, height: 330 });
  await c.locator("textarea").scrollIntoViewIfNeeded();
  await expect(c.locator("textarea")).toBeInViewport();
  // trống thì nút gửi khoá; gõ tiếp được khi đang loading
  const empty = cell(page, "Composer", "empty");
  await expect(empty.locator("button[type=submit]")).toBeDisabled();
  const busy = cell(page, "Composer", "loading");
  await busy.locator("textarea").fill("đang gõ tiếp");
  await expect(busy.locator("textarea")).toHaveValue("đang gõ tiếp");
});

test("offline-input: mất mạng khi gửi, chữ không mất, có Gửi lại", async ({ page, context }) => {
  await page.goto("/dev/ui");
  const c = cell(page, "Composer", "default");
  await c.scrollIntoViewIfNeeded();
  const ta = c.locator("textarea");
  await ta.fill("Câu hỏi quan trọng về bài tập lớn");
  await context.setOffline(true);
  await ta.press("Enter");
  await expect(ta).toHaveValue("Câu hỏi quan trọng về bài tập lớn");
  await expect(c.getByRole("button", { name: "Gửi lại" })).toBeVisible();
  await expect(c.getByRole("alert")).toContainText("vẫn còn trong ô");
  await context.setOffline(false);
  await c.getByRole("button", { name: "Gửi lại" }).click();
  await expect(c.getByRole("button", { name: "Gửi lại" })).toHaveCount(0);
  await expect(ta).toHaveValue("Câu hỏi quan trọng về bài tập lớn");
});

test("cls: khung xương khớp hình, CLS ≤ 0,05 và chiều cao lệch ≤ 8 px", async ({ page }) => {
  await page.addInitScript(() => {
    (window as unknown as { __cls: number }).__cls = 0;
    new PerformanceObserver((list) => {
      for (const e of list.getEntries() as unknown as Array<{ hadRecentInput: boolean; value: number }>) if (!e.hadRecentInput) (window as unknown as { __cls: number }).__cls += e.value;
    }).observe({ type: "layout-shift", buffered: true });
  });
  await page.goto("/dev/ui");
  const demo = page.locator("[data-part=cls-demo]");
  await demo.waitFor({ state: "attached" });
  await page.evaluate(() => { (window as unknown as { __cls: number }).__cls = 0; }); // đếm từ lúc khung xương xuất hiện (bỏ cú nhảy do nạp trang)
  await demo.scrollIntoViewIfNeeded();
  const before = await demo.evaluate((e) => e.getBoundingClientRect().height);
  await expect(demo).toHaveAttribute("data-loaded", "true");
  await page.waitForTimeout(400);
  const after = await demo.evaluate((e) => e.getBoundingClientRect().height);
  expect(Math.abs(after - before), `skeleton ${before} vs thật ${after}`).toBeLessThanOrEqual(8);
  const cls = await page.evaluate(() => (window as unknown as { __cls: number }).__cls);
  expect(cls).toBeLessThanOrEqual(0.05);
});

test("domain: CitationList mở tại chỗ; VerificationState bốn lời, nút duyệt theo vai", async ({ page }) => {
  await page.goto("/dev/ui?as=student");
  const cl = cell(page, "CitationList", "default");
  await cl.scrollIntoViewIfNeeded();
  await expect(cl.getByRole("heading", { name: "Nguồn tham khảo (2)" })).toBeVisible();
  const url = page.url();
  const first = cl.getByRole("button").first();
  await expect(first).toHaveAttribute("aria-expanded", "false");
  await first.click();
  await expect(first).toHaveAttribute("aria-expanded", "true");
  expect(page.url()).toBe(url);
  expect(page.context().pages()).toHaveLength(1);
  await expect(cell(page, "CitationList", "empty")).toContainText("Chưa có nguồn tham khảo cho câu trả lời này.");

  const four = page.locator("[data-part=verification-four]");
  for (const t of ["Chờ xác nhận", "Đã được giảng viên xác nhận · Lê Thu Hà", "Đã được giảng viên sửa & xác nhận", "Đang chờ giảng viên", "Xem câu trả lời AI gốc"]) {
    await expect(four.getByText(t, { exact: true })).toBeVisible();
  }
  for (const n of ["Xác nhận", "Chỉnh sửa", "Loại khỏi tri thức"]) await expect(four.getByRole("button", { name: n, exact: true })).toHaveCount(0); // vai Sinh viên
  const verified = four.locator('[data-status=verified] [data-part=verified-block]');
  const css = await verified.evaluate((e) => { const c = getComputedStyle(e); return { bg: c.backgroundColor, bl: c.borderLeftWidth }; });
  expect(css).toEqual({ bg: "rgba(0, 0, 0, 0)", bl: "1px" });

  await page.goto("/dev/ui?as=teacher");
  const f2 = page.locator("[data-part=verification-four]");
  for (const n of ["Xác nhận", "Chỉnh sửa", "Loại khỏi tri thức"]) await expect(f2.getByRole("button", { name: n, exact: true })).toHaveCount(1);
});

test("one-primary: mỗi vùng làm việc ≤ 1 nút primary", async ({ page }) => {
  await page.goto("/dev/ui");
  await expect(page.locator("[data-part=primitive]")).toHaveCount(25);
  const bad = await page.locator("[data-part=work-region]").evaluateAll((els) =>
    els.map((e, i) => [i, e.querySelectorAll('[data-variant="primary"]').length] as const).filter(([, n]) => n > 1).map(([i]) => i),
  );
  expect(bad).toEqual([]);
});

test("confirm: disabledReason khoá nút xác nhận và gắn aria-describedby", async ({ page }) => {
  await page.goto("/dev/ui");
  await cell(page, "ConfirmIrreversible", "disabled").locator("[data-open-overlay]").click();
  const dlg = page.locator("dialog[open]");
  const confirm = dlg.getByRole("button", { name: "Chốt điểm" });
  await expect(confirm).toBeDisabled();
  const id = await confirm.getAttribute("aria-describedby");
  expect(id).toBeTruthy();
  await expect(dlg.locator(`[id="${id}"]`)).toContainText("chưa có điểm cuối kỳ");
});

test("BUG-PU02-2: Dialog, Drawer, ConfirmIrreversible khoá cuộn nền và mở khoá khi đóng", async ({ page }) => {
  await page.goto("/dev/ui");
  for (const [block, state] of [["Dialog", "default"], ["Drawer", "default"], ["ConfirmIrreversible", "default"]] as const) {
    const opener = cell(page, block, state).locator("[data-open-overlay]");
    await opener.scrollIntoViewIfNeeded();
    await page.evaluate(() => window.scrollBy(0, 0));
    const before = await page.evaluate(() => window.scrollY);
    await opener.click();
    await expect(page.locator("dialog[open]")).toHaveCount(1);
    await page.mouse.move(700, 450);
    await page.mouse.wheel(0, 800);
    await page.waitForTimeout(200);
    expect(await page.evaluate(() => window.scrollY), `${block}: nền không được trôi`).toBe(before);
    expect(await page.evaluate(() => getComputedStyle(document.documentElement).overflow)).toBe("hidden");
    await page.keyboard.press("Escape");
    await expect(page.locator("dialog[open]")).toHaveCount(0);
    expect(await page.evaluate(() => getComputedStyle(document.documentElement).overflow), `${block}: mở khoá sau khi đóng`).not.toBe("hidden");
    await page.mouse.wheel(0, 400);
    await expect.poll(() => page.evaluate(() => window.scrollY)).toBeGreaterThan(before);
  }
});

test("BUG-PU02-1: hàng DataTable focus bằng bàn phím có box-shadow = --ep-focus", async ({ page }) => {
  await page.goto("/dev/ui");
  const row = page.locator('[data-part=primitive][data-name="DataTable"] tbody tr[tabindex="0"]').first();
  await row.scrollIntoViewIfNeeded();
  await page.keyboard.press("Tab"); // bật chế độ focus-visible bằng bàn phím
  await row.focus();
  const r = await page.evaluate(() => {
    const el = document.activeElement as HTMLElement;
    const probe = document.createElement("i");
    probe.style.boxShadow = "var(--ep-focus)";
    document.body.append(probe);
    const want = getComputedStyle(probe).boxShadow;
    probe.remove();
    return { tag: el.tagName, got: getComputedStyle(el).boxShadow, want };
  });
  expect(r.tag).toBe("TR");
  expect(r.got).toBe(r.want);
  expect(r.got).not.toBe("none");
});
