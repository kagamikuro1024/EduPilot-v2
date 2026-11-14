import { test, type Browser, type Page, type Request } from "@playwright/test";
import { mkdirSync } from "fs";

test.skip(!process.env.QC_REAL, "cần stack thật ep-qc: QC_REAL=1 playwright test -c playwright.qc-dm.config.ts");
// QC sprint 10.6 L4: «Lưu ý chấm» (ngăn bài tập), hàng chờ bản cũ, màn duyệt, /class/settings trên stack thật; 1440 + 375. Ảnh ghi ra QC_OUT.
const PW = process.env.QC_PW ?? "";
const OUT = process.env.QC_OUT ?? "/tmp/qc-shots";
const RID = process.env.QC_RID ?? "";
const ACT = process.env.QC_ACT === "1";
mkdirSync(OUT, { recursive: true });
const BAD = /\b(AI|RAG|PII|nháp|prompt|LLM)\b/i;

async function login(browser: Browser, user: string, w: number, h: number) {
  const ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: w, height: h }, locale: "vi-VN", hasTouch: w < 500, isMobile: w < 500 });
  const page = await ctx.newPage();
  await page.goto("/login");
  await page.locator("input[type=email]").fill(`${user}@edupilot.local`);
  await page.locator("input[type=password]").fill(PW);
  await page.keyboard.press("Enter");
  await page.waitForURL((u: URL) => !u.pathname.startsWith("/login"));
  await page.locator("[data-part=topbar]").waitFor();
  if (user !== "sv.kha" && (await page.locator("[data-part=topbar]").innerText()).includes("761988")) {
    await page.locator("[data-part=topbar]").getByText("761988").first().click();
    await page.getByText("761987").first().click();
    await page.waitForTimeout(1500);
  }
  return { ctx, page };
}
const overflow = (page: Page) => page.evaluate(() => document.documentElement.scrollWidth > document.documentElement.clientWidth + 1);

for (const [w, h] of [[1440, 900], [375, 812]] as const) {
  test(`teacher: ngăn bài tập @${w}`, async ({ browser }, info) => {
    test.skip(info.project.name !== "desktop");
    test.setTimeout(180_000);
    const { ctx, page } = await login(browser, "teacher", w, h);
    await page.goto("/grading?tab=assignments");
    await page.waitForTimeout(2500);
    const rows = page.locator("[data-part=assignment-row]");
    console.log(`QCINFO @${w} hàng bài tập: ${await rows.count()} ${JSON.stringify((await rows.allInnerTexts()).map((t) => t.replace(/\n/g, " ").slice(0, 40)))}`);
    await page.screenshot({ path: `${OUT}/assignments-list-${w}.png` });
    if (w < 500) { await page.getByText("Bài tập 03 —").first().dispatchEvent("click"); console.log(`QCINFO @${w} url sau chạm: ${page.url()}`); }
    else await rows.filter({ hasText: "Bài tập 03" }).first().click({ timeout: 15000 });
    await page.waitForTimeout(2500);
    const panel = page.locator("[data-part=guidance-panel]");
    const has = await panel.count();
    console.log(`QCINFO @${w} guidance-panel có: ${has}`);
    if (has) {
      const t = await panel.innerText();
      console.log(`QCINFO @${w} panel chữ: ${JSON.stringify(t.slice(0, 700))}`);
      console.log(`QCINFO @${w} từ cấm: ${BAD.test(t)} ${t.match(BAD)?.[0] ?? ""}`);
      console.log(`QCINFO @${w} icon Info trong panel: ${await panel.locator("svg.lucide-info, svg.lucide-circle-help, svg.lucide-circle-question-mark").count()}`);
      const sizes = await panel.locator("button, [role=button], [role=switch]").evaluateAll((els) => els.map((e) => { const r = e.getBoundingClientRect(); return `${(e.textContent || "").trim().slice(0, 14)}:${Math.round(r.width)}x${Math.round(r.height)}`; }));
      console.log(`QCINFO @${w} kích thước nút: ${sizes.join(" | ")}`);
      await panel.scrollIntoViewIfNeeded();
      await panel.screenshot({ path: `${OUT}/guidance-panel-${w}.png` });
    }
    console.log(`QCINFO @${w} tràn ngang: ${await overflow(page)}`);
    await page.screenshot({ path: `${OUT}/assignment-drawer-${w}.png`, fullPage: true });
    if (ACT && has) {
      const items = panel.locator("[data-part=guidance-item]");
      console.log(`QCINFO @${w} số mục: ${await items.count()}`);
      await items.first().getByRole("button", { name: "Duyệt" }).click();
      await page.waitForTimeout(800);
      const undo = await panel.getByText("Hoàn tác").count();
      const toast = await page.getByRole("status").filter({ hasText: "Thành công" }).count();
      console.log(`QCINFO @${w} sau Duyệt: Hoàn tác=${undo} toast Thành công=${toast} hộp xác nhận=${await page.getByRole("dialog").count()}`);
      await panel.screenshot({ path: `${OUT}/guidance-after-approve-${w}.png` });
    }
    if (process.env.QC_APPLY === "1" && has && w > 500) {
      await panel.getByRole("button", { name: /^Áp dụng/ }).click();
      await page.waitForTimeout(800);
      const dlg = page.getByRole("dialog").last();
      console.log(`QCINFO @${w} sau bấm Áp dụng: hộp xác nhận=${await page.getByRole("alertdialog").count()} dialog=${await page.getByRole("dialog").count()} chữ=${JSON.stringify((await dlg.innerText().catch(() => "")).slice(0, 300))}`);
      await page.screenshot({ path: `${OUT}/apply-confirm-${w}.png` });
      const confirm = page.getByRole("alertdialog").getByRole("button", { name: /Áp dụng|Đồng ý|Xác nhận/ }).first();
      if (await confirm.count()) await confirm.click();
      await page.waitForTimeout(1500);
      console.log(`QCINFO @${w} sau xác nhận: ${JSON.stringify((await panel.innerText()).slice(0, 160))}`);
      await panel.screenshot({ path: `${OUT}/guidance-after-apply-${w}.png` });
    }
    await ctx.close();
  });
}

test("teacher: hàng chờ, màn duyệt, cài đặt lớp @1440", async ({ browser }, info) => {
  test.skip(info.project.name !== "desktop");
  test.setTimeout(180_000);
  const { ctx, page } = await login(browser, "teacher", 1440, 900);
  await page.goto("/grading");
  await page.waitForTimeout(3000);
  const txt = await page.locator("main").innerText();
  console.log(`QCINFO hàng chờ: «bản cũ»=${/bản cũ/.test(txt)} Chấm lại=${/Chấm lại/.test(txt)} ${JSON.stringify((txt.match(/[^\n]*bản cũ[^\n]*/) ?? [""])[0])}`);
  await page.screenshot({ path: `${OUT}/grading-queue-1440.png` });
  const sel = page.locator("select").first();
  if (await sel.count()) {
    const opts = await sel.locator("option").allInnerTexts();
    console.log(`QCINFO hàng chờ options: ${JSON.stringify(opts)}`);
    const o3 = opts.findIndex((t) => t.includes("Bài tập 03"));
    if (o3 >= 0) await sel.selectOption({ index: o3 });
    await page.waitForTimeout(2500);
    const t4 = await page.locator("main").innerText();
    console.log(`QCINFO hàng chờ sau chọn BT03: «bản cũ»=${/bản cũ/.test(t4)} ${JSON.stringify((t4.match(/[^\n]*bản cũ[^\n]*/) ?? [""])[0])} Chấm lại=${/Chấm lại/.test(t4)}`);
    await page.screenshot({ path: `${OUT}/grading-queue-bt03-1440.png` });
  } else console.log("QCINFO hàng chờ: không có select");
  if (RID) {
    await page.goto(`/grading/${RID}`);
    await page.waitForTimeout(3000);
    const t2 = await page.locator("main").innerText();
    console.log(`QCINFO màn duyệt: ${JSON.stringify((t2.match(/[^\n]*bản v\d[^\n]*/) ?? [""])[0])} chip «Bản cũ»=${/Bản cũ/.test(t2)}`);
    await page.screenshot({ path: `${OUT}/review-1440.png` });
  }
  await page.goto("/class/settings");
  await page.waitForTimeout(2500);
  const t3 = await page.locator("main").innerText();
  const labels = ["Lưu ý chấm", "Ca kiểm thử", "Gợi ý câu hỏi tương tự", "Gợi ý khi soạn"];
  console.log(`QCINFO cài đặt lớp nhãn: ${labels.map((l) => `${l}=${t3.includes(l)}`).join(", ")}; từ cấm=${BAD.test(t3)}`);
  await page.screenshot({ path: `${OUT}/class-settings-1440.png`, fullPage: true });
  await ctx.close();
});

test("sinh viên: không thấy lưu ý @1440", async ({ browser }, info) => {
  test.skip(info.project.name !== "desktop");
  test.setTimeout(120_000);
  const { ctx, page } = await login(browser, "sv.kha", 1440, 900);
  const reqs: string[] = [];
  page.on("request", (r: Request) => {
    if (/guidance|regrade|reflect/.test(r.url())) reqs.push(r.url());
  });
  await page.goto("/assignments");
  await page.waitForTimeout(3000);
  const t = await page.locator("main").innerText();
  console.log(`QCINFO SV: «Lưu ý chấm»=${/Lưu ý chấm/.test(t)} request guidance=${reqs.length}`);
  await ctx.close();
});
