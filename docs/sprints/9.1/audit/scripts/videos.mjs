// Sprint 9.1 pha 1 — 3 video ngắn để đánh giá chuyển động (webm, 1280×720):
//   landing        : khách mở trang giới thiệu, cuộn hết trang
//   teacher-grading: giảng viên đi từ Hôm nay → Chấm bài → mở một bài (CHỈ XEM, không bấm Duyệt / Công bố)
//   student-chat   : sinh viên (sv.kha) hỏi một câu ở Chat riêng (tạo 1 phiên chat mới trên demo; không đụng điểm)
//   QC_PW_FILE=... node videos.mjs [landing|teacher-grading|student-chat ...]
import { chromium } from "/opt/homebrew/lib/node_modules/@playwright/cli/node_modules/playwright-core/index.mjs";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const HERE = path.dirname(fileURLToPath(import.meta.url));
const OUT = path.join(HERE, "../video");
fs.mkdirSync(OUT, { recursive: true });
const B = process.env.DEMO_URL ?? "https://v2202609419695521098.happysrv.de";
const PW = fs.readFileSync(process.env.QC_PW_FILE ?? "/tmp/qcaudit/pw.txt", "utf8").trim();
const C1 = "01a19813-a273-7dc9-99a5-a518f46f98ea";
const SUB = "01a19817-4b65-7416-a4ac-55a834f6662f";
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const SIZE = { width: 1280, height: 720 };
const browser = await chromium.launch();

async function login(user) {
  // đăng nhập ở context không quay, rồi dùng lại cookie (refresh token) cho context quay → video không chứa mật khẩu
  const c = await browser.newContext({ viewport: SIZE, bypassCSP: true, locale: "vi-VN" });
  const p = await c.newPage();
  await p.goto(B + "/login");
  await p.fill("input[type=email]", user + "@edupilot.local");
  await p.fill("input[type=password]", PW);
  await p.keyboard.press("Enter");
  await p.waitForURL((u) => !u.pathname.startsWith("/login"), { timeout: 30000 });
  await sleep(1500);
  const state = await c.storageState();
  await c.close();
  return state;
}
async function record(name, user, flow) {
  const state = user ? await login(user) : undefined;
  const ctx = await browser.newContext({ viewport: SIZE, bypassCSP: true, locale: "vi-VN", storageState: state, recordVideo: { dir: path.join(OUT, ".tmp-" + name), size: SIZE } });
  const page = await ctx.newPage();
  try { await flow(page); } catch (e) { console.log(name, "lỗi giữa chừng:", String(e.message).split("\n")[0]); }
  await sleep(1200);
  const v = page.video();
  await ctx.close();
  const file = path.join(OUT, `${name}.webm`);
  await v.saveAs(file);
  fs.rmSync(path.join(OUT, ".tmp-" + name), { recursive: true, force: true });
  console.log(name, Math.round(fs.statSync(file).size / 1024), "KB");
}
const scrollAll = async (page, step = 220, gap = 90) => {
  const H = await page.evaluate(() => document.documentElement.scrollHeight);
  for (let y = 0; y < H; y += step) { await page.mouse.wheel(0, step); await sleep(gap); }
};

const FLOWS = {
  landing: () => record("landing", null, async (page) => {
    await page.goto(B + "/", { waitUntil: "load" });
    await sleep(2500);
    await scrollAll(page);
    await sleep(1500);
  }),
  "teacher-grading": () => record("teacher-grading", "teacher", async (page) => {
    await page.goto(B + `/?course=${C1}`);
    await sleep(2500);
    await page.getByRole("link", { name: "Chấm bài" }).first().click();
    await sleep(2500);
    await page.getByRole("button", { name: /Cần xem kỹ/ }).first().click().catch(() => {});
    await sleep(1500);
    await page.goto(B + `/grading/${SUB}?course=${C1}`);
    await sleep(3000);
    await page.mouse.wheel(0, 500); await sleep(1200);
    await page.locator("main button").filter({ hasText: /Đoạn trích/ }).first().click().catch(() => {});
    await sleep(1500);
    await page.mouse.wheel(0, 600); await sleep(1500);
    await page.mouse.wheel(0, -1100); await sleep(1200);
  }),
  "student-chat": () => record("student-chat", "sv.kha", async (page) => {
    await page.goto(B + `/chat?course=${C1}`);
    await sleep(3000);
    const box = page.locator("textarea").first();
    await box.click();
    await page.keyboard.type("Brute force mật khẩu là gì và phòng chống thế nào?", { delay: 55 });
    await sleep(600);
    await page.keyboard.press("Enter");
    await sleep(25000); // chờ câu trả lời stream xong
  }),
};
for (const n of process.argv.slice(2).length ? process.argv.slice(2) : Object.keys(FLOWS)) await FLOWS[n]();
await browser.close();
