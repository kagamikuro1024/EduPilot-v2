import { writeFileSync } from "node:fs";
import { test, type Page, type Browser } from "@playwright/test";

test.skip(!process.env.QC_REAL, "cần stack thật ep-qc: QC_REAL=1 playwright test -c playwright.qc10.config.ts");

// QC sprint 10 (FEAT-practice): màn thật trên stack ep-qc. Ảnh vào docs/sprints/10/qc/shots/.
const PW = process.env.QC_PW ?? "";
const OUT = process.env.QC_OUT ?? "/tmp/qc10-shots";
const BAD = /\b(AI|RAG|citation|model|LLM|prompt|embedding|token|streak|huy hiệu|xếp hạng)\b/i;

async function login(browser: Browser, user: string, w: number) {
  const ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: w, height: w < 500 ? 812 : 900 }, locale: "vi-VN", hasTouch: w < 500, isMobile: w < 500 });
  const page = await ctx.newPage();
  await page.goto("/login");
  await page.locator("input[type=email]").fill(`${user}@edupilot.local`);
  await page.locator("input[type=password]").fill(PW);
  await page.keyboard.press("Enter");
  await page.waitForURL((u) => !u.pathname.startsWith("/login"));
  await page.locator("[data-part=topbar]").waitFor();
  return { ctx, page };
}
async function pick(page: Page) {
  const list = page.getByRole("list", { name: "Các đáp án" });
  if (await list.count()) await list.getByRole("button").first().click();
  else if (await page.getByRole("button", { name: "Đúng", exact: true }).count()) await page.getByRole("button", { name: "Đúng", exact: true }).click();
  else await page.getByRole("textbox").first().fill("đáp án thử");
}
async function layout(page: Page, tag: string, notes: string[]) {
  const m = await page.evaluate(() => {
    const small = [...document.querySelectorAll("button,a[role=button],[role=tab]")].filter((e) => { const r = (e as HTMLElement).getBoundingClientRect(); return r.width > 0 && (r.height < 44 || r.width < 44); }).length;
    return { sw: document.documentElement.scrollWidth, cw: document.documentElement.clientWidth, small, text: document.body.innerText };
  });
  notes.push(`${tag}: scrollWidth ${m.sw}/${m.cw} ${m.sw > m.cw ? "TRÀN" : "ok"}; điều khiển <44px: ${m.small}; chữ cấm: ${(m.text.match(BAD) ?? ["không"])[0]}`);
}

for (const w of [1440, 375]) {
  test(`SV luyện đề ${w}px`, async ({ browser }, info) => {
    test.skip(info.project.name !== "desktop");
    test.setTimeout(240_000);
    const notes: string[] = [];
    const { ctx, page } = await login(browser, "sv.kha", w);
    await page.goto("/practice"); await page.getByRole("heading", { name: "Luyện đề", level: 1 }).waitFor();
    await page.waitForTimeout(800);
    await page.screenshot({ path: `${OUT}/practice-${w}.png` }); await layout(page, "/practice", notes);
    // luyện chủ đề: khối khuyến nghị
    await page.getByRole("button", { name: /^Luyện 10 câu/ }).first().click();
    await page.waitForURL(/\/practice\/[0-9a-f-]{36}/);
    await page.waitForTimeout(800);
    await page.screenshot({ path: `${OUT}/topic-q1-${w}.png` }); await layout(page, "topic câu 1", notes);
    await pick(page);
    await page.getByRole("button", { name: "Kiểm tra" }).click();
    await page.waitForTimeout(800);
    await page.screenshot({ path: `${OUT}/topic-feedback-${w}.png` }); await layout(page, "topic phản hồi", notes);
    const bad = await page.getByRole("button", { name: "Giải thích thêm" }).count();
    notes.push(`nút Giải thích thêm: ${bad}`);
    await page.goto("/practice/history"); await page.waitForTimeout(1200);
    await page.screenshot({ path: `${OUT}/history-${w}.png` }); await layout(page, "/practice/history", notes);
    await page.goto("/"); await page.waitForTimeout(1500);
    await page.screenshot({ path: `${OUT}/home-${w}.png` }); await layout(page, "/ (Hôm nay)", notes);
    // thi thử
    await page.goto("/practice"); await page.getByRole("tab", { name: /Thi thử/ }).click();
    await page.getByRole("button", { name: "Bắt đầu thi thử" }).click();
    await page.waitForURL(/\/practice\/[0-9a-f-]{36}/); await page.waitForTimeout(1200);
    await page.screenshot({ path: `${OUT}/mock-${w}.png` }); await layout(page, "mock câu 1", notes);
    await pick(page); await page.waitForTimeout(1500);
    await page.screenshot({ path: `${OUT}/mock-saved-${w}.png` });
    writeFileSync(`${OUT}/notes-sv-${w}.txt`, notes.join("\n"));
    await ctx.close();
  });
}

test("GV /questions + sổ điểm Forms 1440", async ({ browser }, info) => {
  test.skip(info.project.name !== "desktop");
  test.setTimeout(180_000);
  const notes: string[] = [];
  const { ctx, page } = await login(browser, "teacher", 1440);
  await page.goto("/questions"); await page.getByRole("heading", { name: "Ngân hàng câu hỏi", level: 1 }).waitFor(); await page.waitForTimeout(1000);
  await page.screenshot({ path: `${OUT}/questions-1440.png` });
  await page.getByRole("button", { name: "Tạo câu hỏi" }).first().click(); await page.waitForTimeout(500);
  await page.screenshot({ path: `${OUT}/questions-create-menu-1440.png` });
  notes.push("menu tạo: " + (await page.getByRole("menuitem").allInnerTexts()).join(" | "));
  await page.keyboard.press("Escape");
  await page.goto("/gradebook"); await page.waitForTimeout(1500);
  await page.screenshot({ path: `${OUT}/gradebook-1440.png` });
  writeFileSync(`${OUT}/notes-gv.txt`, notes.join("\n"));
  await ctx.close();
});
