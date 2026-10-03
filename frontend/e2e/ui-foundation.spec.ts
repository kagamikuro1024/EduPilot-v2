import { execFileSync } from "node:child_process";
import { mkdirSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { expect, test } from "@playwright/test";
import { navFor } from "../src/shared/shell/nav";
import { asDemo, type DemoRole } from "./support/session";

// Quét mọi route × vai của bản dựng cổng (US-PU-02 AC14): một nút primary mỗi vùng làm việc, trang không cuộn ngang.
// (AUDIT đầy đủ — cut / ell sau khi cuộn, chuyển trạng thái — là lượt chạy của QC theo audit-baseline.md; ở đây chỉ chặn tràn ngang.)
const ROLES: DemoRole[] = ["student", "ta", "teacher", "admin"];
const REGION = "main section, [role=dialog], [data-part=work-region], form";
const allow: Array<{ route: string; region: string; reason: string }> = JSON.parse(readFileSync("e2e/primary-allow.json", "utf8"));

const routes = (role: DemoRole) => [...new Set(navFor(role, true).flatMap((g) => g.items.map((i) => i.href)))];

test("primary-allow.json: tối đa 5 mục, mỗi mục đủ route / vùng / lý do", () => {
  expect(allow.length).toBeLessThanOrEqual(5);
  for (const a of allow) expect(a.route && a.region && a.reason.trim()).toBeTruthy();
});

for (const role of ROLES) {
  test(`one-primary sweep: ${role}`, async ({ page, context }) => {
    test.setTimeout(120_000);
    await asDemo(context, role);
    const over: string[] = [];
    const audit: string[] = [];
    for (const href of routes(role)) {
      await page.goto(href);
      await page.locator("main").first().waitFor();
      await page.waitForTimeout(150);
      const regions = await page.locator(REGION).evaluateAll((els) =>
        els.map((e) => ({ n: Array.from(e.querySelectorAll('[data-variant="primary"]')).filter((b) => (b as HTMLElement).offsetParent !== null).length, tag: e.tagName + (e.getAttribute("data-part") ? `[${e.getAttribute("data-part")}]` : "") })).filter((r) => r.n > 1),
      );
      for (const r of regions) if (!allow.some((a) => a.route === href && a.region === r.tag)) over.push(`${href} ${r.tag} có ${r.n} nút primary`);
      const ox = await page.evaluate(() => document.documentElement.scrollWidth - innerWidth);
      if (ox !== 0) audit.push(`${href}: tràn ngang ${ox}px`);
    }
    expect(over, "vùng có > 1 nút primary").toEqual([]);
    expect(audit, "tràn ngang").toEqual([]);
  });
}

test("ConfirmIrreversible: thiếu consequence ⇒ lỗi biên dịch nêu `consequence`", async ({}, info) => {
  test.skip(info.project.name !== "desktop", "ghi tệp tạm dùng chung");
  test.setTimeout(90_000);
  const dir = "src/__tsc__";
  mkdirSync(dir, { recursive: true });
  writeFileSync(`${dir}/bad.tsx`, 'import { ConfirmIrreversible } from "@/shared/ui";\nexport const X = () => <ConfirmIrreversible open onClose={() => {}} onConfirm={() => {}} title="x" confirmLabel="Xoá" />;\n');
  try {
    let out = "";
    try {
      execFileSync("pnpm", ["exec", "tsc", "--noEmit"], { encoding: "utf8", stdio: "pipe" });
    } catch (e) {
      out = String((e as { stdout?: string }).stdout ?? "");
    }
    expect(out).toContain("consequence");
    expect(out).toContain("__tsc__/bad.tsx");
  } finally {
    rmSync(dir, { recursive: true, force: true });
  }
});

// ---- US-PU-05 AC3: Sinh viên đi hết /chat và /threads CHỈ bằng bàn phím; 640 px (≈ zoom 200 % của 1280) vẫn dùng được ----
import type { Locator, Page } from "@playwright/test";
import { loadAudit, runAudit } from "./support/audit";

/** Nhấn Tab cho tới khi phần tử đích nhận focus (không gọi .focus() — đúng nghĩa "chỉ bàn phím"). */
async function tabTo(page: Page, target: Locator, max = 120) {
  await target.first().waitFor({ state: "visible", timeout: 10_000 });
  const handle = await target.elementHandle({ timeout: 5000 });
  for (let i = 0; i < max; i++) {
    if (await page.evaluate((el) => el === document.activeElement, handle)) return;
    await page.keyboard.press("Tab");
  }
  throw new Error(`Tab ${max} lần vẫn chưa tới: ${await target.evaluate((e) => e.outerHTML.slice(0, 80))}`);
}
/** Vòng focus nhìn thấy: phần tử đang focus có `box-shadow` hoặc `outline` khác rỗng. */
async function ringVisible(page: Page) {
  return page.evaluate(() => {
    // vòng focus có thể nằm ở chính phần tử hoặc ở khung bao (composer dùng :focus-within)
    for (let e = document.activeElement as HTMLElement | null, i = 0; e && i < 3; e = e.parentElement, i++) {
      const cs = getComputedStyle(e);
      if (cs.boxShadow !== "none" || (cs.outlineStyle !== "none" && parseFloat(cs.outlineWidth) > 0)) return true;
    }
    return false;
  });
}

test.describe("keyboard-only", () => {
  test.beforeEach(async ({}, info) => {
    test.skip(info.project.name !== "desktop", "tự đặt bề rộng");
  });

  test("/chat: gõ, gửi bằng Enter, Dừng, mở nguồn, lịch sử phiên, quay lại composer", async ({ page, context }) => {
    test.setTimeout(90_000);
    await asDemo(context, "student");
    await page.goto("/chat");
    const composer = page.getByLabel("Câu hỏi của bạn");
    await test.step("gõ câu hỏi và gửi bằng Enter", async () => {
      await tabTo(page, composer);
      expect(await ringVisible(page), "vòng focus ở composer").toBe(true);
      await page.keyboard.type("Tôi nghỉ mấy buổi?");
      await page.keyboard.press("Enter");
      await expect(page.getByText("Tôi nghỉ mấy buổi?").first()).toBeVisible();
    });
    await test.step("Dừng khi đang trả lời", async () => {
      const stop = page.getByRole("button", { name: "Dừng" });
      await expect(stop).toBeVisible({ timeout: 4000 });
      await tabTo(page, stop);
      expect(await ringVisible(page)).toBe(true);
      await page.keyboard.press("Enter");
      await expect(stop).toHaveCount(0);
    });
    await test.step("gửi lại và mở nguồn tham khảo", async () => {
      await tabTo(page, composer);
      await page.keyboard.type("Tôi được cộng bao nhiêu điểm phát biểu?");
      await page.keyboard.press("Enter");
      const src = page.getByRole("button", { name: /^Nguồn tham khảo/ });
      await expect(src.last()).toBeVisible({ timeout: 15_000 });
      await page.waitForTimeout(800); // chờ câu trả lời dựng xong (không còn đang phát) trước khi thao tác
      await tabTo(page, src.last());
      expect(await ringVisible(page)).toBe(true);
      await expect(src.last()).toHaveAttribute("aria-expanded", "true"); // nguồn mở sẵn
      await page.keyboard.press("Enter");
      await expect(src.last()).toHaveAttribute("aria-expanded", "false"); // Enter đóng lại
    });
    await test.step("mở một phiên trong lịch sử rồi quay lại composer", async () => {
      const item = page.locator("[data-part=chat-history] li button, [data-part=chat-history] li a").first();
      await tabTo(page, item);
      expect(await ringVisible(page)).toBe(true);
      await page.keyboard.press("Enter");
      await tabTo(page, composer);
      await expect(composer).toBeFocused();
    });
  });

  test("/threads: danh sách, mở thread, form Đặt câu hỏi, gõ, gửi, đọc trạng thái xác nhận", async ({ page, context }) => {
    test.setTimeout(90_000);
    await asDemo(context, "student");
    await page.goto("/threads");
    await test.step("đi tới danh sách và mở một thread", async () => {
      const first = page.locator('main a[href^="/threads/"]').first();
      await tabTo(page, first);
      expect(await ringVisible(page)).toBe(true);
      await page.keyboard.press("Enter");
      await expect(page).toHaveURL(/\/threads\/.+/);
      await expect(page.locator("main h1")).toBeVisible();
    });
    await test.step("quay lại danh sách, mở form Đặt câu hỏi", async () => {
      await page.goBack();
      const ask = page.getByRole("button", { name: "Đặt câu hỏi" }).first();
      await tabTo(page, ask);
      await page.keyboard.press("Enter");
      await expect(page.locator("[data-part=thread-form]")).toBeVisible();
    });
    await test.step("gõ và gửi", async () => {
      const form = page.locator("[data-part=thread-form]");
      await tabTo(page, form.getByLabel(/Tiêu đề câu hỏi/));
      await page.keyboard.type("Vì sao chế độ CBC cần vectơ khởi tạo?");
      const topic = form.getByLabel(/Chủ đề/);
      await tabTo(page, topic);
      const first = (await topic.locator("option").nth(1).textContent()) ?? "";
      await page.keyboard.type(first.slice(0, 1)); // gõ chữ đầu của mục: cách chọn trong <select> chỉ bằng phím
      await expect(topic).not.toHaveValue("");
      await tabTo(page, form.getByLabel(/Nội dung chi tiết/));
      await page.keyboard.type("Em chưa hiểu vì sao hai khối giống nhau lại cho bản mã khác nhau.");
      const send = form.getByRole("button", { name: "Đăng câu hỏi" });
      await tabTo(page, send);
      await expect(send).toBeEnabled();
      await page.keyboard.press("Enter");
      await expect(page.getByText("Vì sao chế độ CBC cần vectơ khởi tạo?").first()).toBeVisible();
    });
    await test.step("đọc trạng thái xác nhận (VerificationState) trong thread mở", async () => {
      // sau khi đăng, ứng dụng mở luôn thread vừa tạo
      await expect(page.locator("main h1")).toContainText("Vì sao chế độ CBC cần vectơ khởi tạo?");
      await expect(page.locator("main")).toContainText(/Chờ xác nhận|Đang chờ giảng viên|Đã được giảng viên/);
    });
  });

  test("640 px: /chat và /threads sạch AUDIT, composer tới được bằng phím và không bị thanh dưới che", async ({ page, context }) => {
    const { AUDIT_SRC } = await loadAudit();
    await asDemo(context, "student");
    await page.setViewportSize({ width: 640, height: 800 });
    for (const route of ["/chat", "/threads"]) {
      await page.goto(route);
      await page.locator("main").waitFor();
      await page.waitForTimeout(200);
      const a = await runAudit(page, AUDIT_SRC);
      expect(a, `${route} AUDIT`).toMatchObject({ ox: 0, cut: [], ell: [] });
    }
    await page.goto("/chat");
    const hist = page.locator("[data-part=chat-sessions-button]");
    await tabTo(page, hist);
    await page.keyboard.press("Enter");
    await expect(page.locator("dialog[open]")).toHaveCount(1);
    await page.keyboard.press("Escape");
    await expect(hist).toBeFocused();
    const composer = page.getByLabel("Câu hỏi của bạn");
    await tabTo(page, composer);
    await page.evaluate(() => window.scrollTo(0, document.documentElement.scrollHeight));
    const r = await page.evaluate(() => {
      const ta = document.activeElement!.getBoundingClientRect();
      const nav = document.querySelector("[data-part=bottom-nav]")!.getBoundingClientRect();
      return { taBottom: ta.bottom, navTop: nav.top };
    });
    expect(r.taBottom).toBeLessThanOrEqual(r.navTop + 0.5);
  });
});
