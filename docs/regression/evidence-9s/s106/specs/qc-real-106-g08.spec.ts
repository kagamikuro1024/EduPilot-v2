import { appendFileSync, readFileSync } from "node:fs";
import { test, type Browser } from "@playwright/test";
import { sql, PW, OUT } from "./qc-real-lib";
const RES = "/Users/kuro/Documents/TA_Agent_qc9s/docs/regression/evidence-9s/s106/results_106.jsonl";
const D = JSON.parse(readFileSync("/Users/kuro/.config/edupilot-team/qc3-work/g08_threads.json", "utf8")) as { tids: string[]; tag: string };
const verdict = (tc: string, cond: boolean, why: string) => { appendFileSync(RES, JSON.stringify({ tc, kq: cond ? "P" : "F", note: why }) + "\n"); console.log("VERDICT", cond ? "P" : "F", tc, why.slice(0, 380)); };
test.skip(!process.env.QC_REAL, "cần stack thật");
test.use({ actionTimeout: 20_000 });
const CID = () => sql(`select id from courses where class_code='QD9G'`)[0][0];
async function pageAs(browser: Browser, user: string, w = 1440) {
  const ctx = await browser.newContext({ baseURL: "https://localhost:943", ignoreHTTPSErrors: true, viewport: { width: w, height: w < 500 ? 812 : 900 }, locale: "vi-VN", hasTouch: w < 500, isMobile: w < 500 });
  await ctx.addInitScript((c) => localStorage.setItem("ep:ui:course", c), CID());
  const page = await ctx.newPage(); await page.addLocatorHandler(page.getByRole("button", { name: "Đã hiểu" }), async (b) => { await b.click(); });
  await page.goto("/login"); await page.locator("input[type=email]").fill(`${user}@edupilot.local`); await page.locator("input[type=password]").fill(PW);
  await page.keyboard.press("Enter"); await page.waitForURL((u) => !u.pathname.startsWith("/login")); await page.locator("[data-part=topbar]").waitFor(); return { ctx, page };
}
const rest0 = () => sql(`select c.text from knowledge_items k join forum_posts p on p.id=k.source_id join content_chunks c on c.id=k.chunk_id where p.thread_id='${D.tids[2]}'`).map((r) => r[0]).slice(1).join("\n");
const title0 = () => sql(`select title from forum_threads where id='${D.tids[2]}'`)[0][0];
test("TC-G08-07 gợi ý khi soạn: debounce, định dạng, rỗng không hiện gì", async ({ browser }) => {
  const { ctx, page } = await pageAs(browser, "sv.kha"); await page.goto("/threads"); await page.waitForTimeout(2500); await page.getByRole("button", { name: "Đặt câu hỏi" }).click(); await page.waitForTimeout(800);
  const reqs: { t: number; url: string }[] = []; page.on("request", (r) => { if (/threads\/(precheck|similar-confirmed)/.test(r.url())) reqs.push({ t: Date.now(), url: r.url().split("/").pop()! }); });
  const ti = page.locator("[data-part=thread-form]").getByLabel("Tiêu đề"); const t0 = Date.now(); await ti.pressSequentially("Câu hỏi: " + title0(), { delay: 25 }); await page.locator("[data-part=thread-form]").getByLabel("Nội dung").fill(rest0()); const tEnd = Date.now(); await page.waitForTimeout(3500);
  const list = page.locator("[data-part=similar-confirmed]"); const n = await list.count(); const lis = n ? await list.locator("li").allInnerTexts() : []; const links = n ? await list.locator("a[href^='/threads/']").count() : 0;
  const form = await page.locator("[data-part=thread-form]").boundingBox(); await page.screenshot({ path: `${OUT}/s106-g0807.png` });
  const firstSim = reqs.find((r) => r.url === "similar-confirmed"); const firstPre = reqs.find((r) => r.url === "precheck"); const nPre = reqs.filter((r) => r.url === "precheck").length;
  // rỗng: gõ chữ không giống gì
  await ti.fill(""); await page.waitForTimeout(500); reqs.length = 0; await ti.pressSequentially("Hôm nay trời đẹp quá phải không", { delay: 20 }); await page.waitForTimeout(3000); const n2 = await list.count(); const txt2 = await page.locator("[data-part=thread-form]").innerText();
  verdict("TC-G08-07", n === 1 && links >= 1 && /Đã xác nhận/.test(lis.join(" ")) && n2 === 0 && !/không tìm thấy/i.test(txt2) && !!firstSim && firstSim.t - tEnd >= 500 && firstSim.t - tEnd <= 800 + 400, `gõ ${title0().length + 10} ký tự (25 ms/phím) → ${nPre} precheck (không mỗi phím); request đầu sau phím cuối: precheck ${firstPre ? firstPre.t - tEnd : "?"} ms, similar-confirmed ${firstSim ? firstSim.t - tEnd : "?"} ms (TC ~600 ms); danh sách: ${n} khối, ${links} liên kết /threads/[id], chữ «${lis.join(" ¦ ").replace(/\n/g, " ").slice(0, 120)}»; chữ không giống gì → khối ${n2}, «không tìm thấy» ${/không tìm thấy/i.test(txt2)}`);
  await ctx.close();
});
test("TC-G08-08 đăng khi có gợi ý: không hộp thoại", async ({ browser }) => {
  const { ctx, page } = await pageAs(browser, "sv.kha"); await page.goto("/threads"); await page.waitForTimeout(2500); await page.getByRole("button", { name: "Đặt câu hỏi" }).click(); await page.waitForTimeout(800);
  await page.locator("[data-part=thread-form]").getByLabel("Tiêu đề").fill("Câu hỏi: " + title0()); await page.locator("[data-part=thread-form]").getByLabel("Nội dung").fill(rest0()); await page.waitForTimeout(2500);
  const sim = await page.locator("[data-part=similar-confirmed]").count(); const posts: number[] = []; page.on("response", (r) => { if (/\/threads$/.test(r.url()) && r.request().method() === "POST") posts.push(r.status()); });
  await page.getByRole("button", { name: "Đăng", exact: true }).click(); await page.waitForTimeout(3000); const dlg = await page.getByRole("dialog").count();
  verdict("TC-G08-08", sim === 1 && dlg === 0 && posts.includes(201), `gợi ý đang hiện (${sim} khối); bấm «Đăng»: POST /threads ${JSON.stringify(posts)}, hộp thoại ${dlg}; E2E cũ xanh: chưa chạy lại ở đây`); await ctx.close();
});
test("TC-G08-18 375 px", async ({ browser }) => {
  const { ctx, page } = await pageAs(browser, "sv.kha", 375); await page.goto("/threads"); await page.waitForTimeout(2500); await page.getByRole("button", { name: "Đặt câu hỏi" }).click(); await page.waitForTimeout(800);
  await page.locator("[data-part=thread-form]").getByLabel("Tiêu đề").fill("Câu hỏi: " + title0()); await page.locator("[data-part=thread-form]").getByLabel("Nội dung").fill(rest0()); await page.waitForTimeout(3500);
  const list = page.locator("[data-part=similar-confirmed]"); const n = await list.count(); const items = n ? await list.locator("li, a").evaluateAll((els) => els.map((e) => { const r = e.getBoundingClientRect(); return { w: Math.round(r.width), h: Math.round(r.height), x: Math.round(r.x) }; })) : [];
  const ov = await page.evaluate(() => document.documentElement.scrollWidth > document.documentElement.clientWidth + 1); await page.screenshot({ path: `${OUT}/s106-g0818.png` });
  const col = items.every((i) => i.x < 200); const small = items.filter((i) => i.h < 44);
  verdict("TC-G08-18", n === 1 && !ov && small.length === 0 && col, `375 px: ${n} khối, ${items.length} phần tử; cao < 44 px: ${JSON.stringify(small)}; tràn ngang ${ov}; một cột (x đầu < 200) ${col}`); await ctx.close();
});
