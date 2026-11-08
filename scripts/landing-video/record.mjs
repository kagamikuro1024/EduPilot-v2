// Quay video giới thiệu trên bản demo (VPS) bằng page.screencast của Playwright 1.63 (docs/sprints/9.1/techlead.md, TL-1 A).
// Chạy trong container Playwright qua scripts/landing-video/run.sh. Ra: $OUT/frames/<cảnh>/*.jpg, $OUT/frames.ffconcat, $OUT/meta.json.
// CHỈ ĐỌC dữ liệu demo (không gửi chat, không duyệt / công bố / sửa điểm): không ghi gì vào DB, không tốn hạn mức LLM.
import { chromium } from "playwright";
import fs from "node:fs";

const HOST = process.env.EP_HOST;
const PW = process.env.PW;
const OUT = process.env.OUT ?? "/out";
const BASE = `https://${HOST}`;
const VIEW = { width: 1440, height: 900 };

// Phụ đề (in vào hình bằng showOverlay). PHẢI khớp VIDEO_CAPTIONS trong frontend/src/features/landing/Landing.tsx (lời thay cho video không tiếng).
const CAPTIONS = [
  "Sinh viên hỏi riêng về điểm và chuyên cần của chính mình.",
  "Họ tên và MSSV được che trước khi gửi cho AI; câu trả lời kèm nguồn trong tài liệu của lớp.",
  "Câu hỏi AI chưa chắc được chuyển tới hộp thư của giảng viên, kèm lý do.",
  "Giảng viên thấy hàng chờ chấm, AI chấm nháp theo rubric kèm đoạn trích từ bài làm.",
  "Giảng viên sửa điểm, duyệt rồi mới công bố. AI không tự công bố.",
  "Sổ điểm: điểm học phần do mã tính từ quy chế, mỗi điểm có giải trình từng dòng.",
  "Hôm nay: việc cần giảng viên quyết, xếp theo mức khẩn.",
];

fs.rmSync(`${OUT}/frames`, { recursive: true, force: true });
fs.mkdirSync(`${OUT}/frames`, { recursive: true });
const browser = await chromium.launch();
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

// Con trỏ tự vẽ (showActions của Playwright luôn in nhãn "Mouse move" / "Click" lên hình, tắt không được): vòng đỏ theo chuột, thu nhỏ khi nhấn.
const cursor = () => {
  const add = () => {
    const c = document.createElement("div");
    c.style.cssText = "position:fixed;left:0;top:0;width:24px;height:24px;margin:-12px 0 0 -12px;border-radius:50%;background:rgba(200,16,46,.25);border:2px solid #c8102e;z-index:2147483647;pointer-events:none;transition:scale .12s ease-out;translate:-60px -60px";
    document.documentElement.append(c);
    addEventListener("mousemove", (e) => (c.style.translate = `${e.clientX}px ${e.clientY}px`), true);
    addEventListener("mousedown", () => (c.style.scale = ".65"), true);
    addEventListener("mouseup", () => (c.style.scale = "1"), true);
  };
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", add); else add();
};

// ---- đăng nhập không quay; giữ cookie phiên + lớp đang chọn (localStorage) trong storageState
async function login(email, classCode) {
  const ctx = await browser.newContext({ viewport: VIEW, locale: "vi-VN" });
  const p = await ctx.newPage();
  await p.goto(`${BASE}/login`, { waitUntil: "load" });
  await p.getByLabel("Email").fill(email);
  await p.getByLabel("Mật khẩu", { exact: true }).fill(PW);
  await p.getByRole("button", { name: "Đăng nhập" }).click();
  await p.waitForURL((u) => u.pathname !== "/login", { timeout: 20000 });
  await sleep(2000);
  const picker = p.getByRole("button", { name: /\d{6}\s*·/ }).first();
  if (classCode && !(await picker.innerText()).includes(classCode)) {
    await picker.click();
    await p.getByRole("option", { name: new RegExp(classCode) }).or(p.getByRole("menuitem", { name: new RegExp(classCode) })).first().click();
    await sleep(1500);
  }
  const state = await ctx.storageState();
  await ctx.close();
  return state;
}

const frameLines = ["ffconcat version 1.0"];
let elapsed = 0; // giây đã ghi của các cảnh trước
const meta = { poster: 0, scenes: [] };

/** Một cảnh = một context. `run(page, a)`: a.start() bắt đầu ghi (gọi khi trang đã sẵn sàng), a.cap(text|null) đặt / xoá phụ đề, a.move / a.click đi chuột có đường. */
async function scene(name, who, run) {
  const state = who && (await login(...who)); // đăng nhập MỚI cho mỗi cảnh: refresh token xoay vòng, dùng lại phiên cũ sẽ bị thu hồi
  const ctx = await browser.newContext({ viewport: VIEW, locale: "vi-VN", timezoneId: "Asia/Ho_Chi_Minh", storageState: state });
  await ctx.addInitScript(cursor);
  const page = await ctx.newPage();
  const dir = `${OUT}/frames/${name}`;
  fs.mkdirSync(dir, { recursive: true });
  const frames = [];
  let overlay;
  const a = {
    async start() {
      await page.screencast.start({
        size: VIEW, quality: 90,
        onFrame: ({ data, timestamp }) => { const f = `${dir}/${String(frames.length).padStart(5, "0")}.jpg`; fs.writeFileSync(f, data); frames.push([f, timestamp]); },
      });
      // trang tĩnh không phát khung nào: một điểm 1 px đổi sắc 1/255 mỗi 100 ms để có khung đều (độ dài cảnh đúng), mắt không thấy
      await page.evaluate(() => {
        const d = document.createElement("div");
        d.style.cssText = "position:fixed;left:0;top:0;width:1px;height:1px;z-index:2147483647;pointer-events:none";
        document.documentElement.append(d);
        let k = 0;
        setInterval(() => (d.style.background = ++k % 2 ? "#fffffe" : "#ffffff"), 100);
      });
    },
    async cap(text) {
      await overlay?.[Symbol.asyncDispose]?.();
      overlay = undefined;
      if (!text) return;
      const ff = (await page.evaluate(() => getComputedStyle(document.body).fontFamily)).replaceAll('"', "'");
      overlay = await page.screencast.showOverlay(
        `<div style="position:fixed;left:50%;bottom:40px;transform:translateX(-50%);max-width:1240px;padding:14px 28px;border-radius:12px;background:rgba(24,24,27,.92);color:#fff;font-family:${ff};font-weight:600;font-size:30px;line-height:1.35;text-align:center">${text}</div>`,
      );
    },
    posterHere() { meta.poster = elapsed + (frames.length ? (frames.at(-1)[1] - frames[0][1]) / 1000 : 0); },
    async move(loc) { const b = await loc.boundingBox(); await page.mouse.move(b.x + b.width / 2, b.y + b.height / 2, { steps: 30 }); },
    async click(loc) { await a.move(loc); await sleep(250); await loc.click(); },
    page,
  };
  await run(page, a);
  await sleep(500);
  await overlay?.[Symbol.asyncDispose]?.();
  await page.screencast.stop();
  await ctx.close();
  for (let i = 0; i < frames.length; i++) {
    const d = i + 1 < frames.length ? (frames[i + 1][1] - frames[i][1]) / 1000 : 0.04; // timestamp tính bằng ms → chia 1000
    frameLines.push(`file ${frames[i][0].replace(`${OUT}/`, "")}`, `duration ${d.toFixed(4)}`);
  }
  const span = frames.length ? (frames.at(-1)[1] - frames[0][1]) / 1000 : 0;
  elapsed += span;
  meta.scenes.push({ name, frames: frames.length, seconds: +span.toFixed(2) });
  console.log("cảnh", name, frames.length, "khung,", span.toFixed(1), "s");
}

const card = (page, html) => page.evaluate((h) => {
  document.body.innerHTML = `<div style="position:fixed;inset:0;background:#fff;display:grid;place-content:center;gap:28px;text-align:center;font-family:var(--ep-font,sans-serif);color:#18181b">${h}</div>`;
}, html);

const sv = ["sv.kha@edupilot.local", "761987"], gv = ["teacher@edupilot.local", "761987"];

await scene("s0-intro", undefined, async (page, a) => {
  await page.goto(`${BASE}/login`, { waitUntil: "load" });
  await card(page, `<img src="/brand/logo-edupilot.svg" alt="" style="height:64px;margin:0 auto"><p style="font-size:60px;font-weight:700;letter-spacing:-.02em;line-height:1.15;max-width:18ch;margin:0 auto">AI soạn nháp. Giảng viên quyết.</p><p style="font-size:28px;color:#52525b;margin:0">Vận hành lớp học có trợ lý AI</p><div style="width:120px;height:4px;background:#c8102e;margin:0 auto"></div>`);
  await a.start();
  await sleep(4000);
});

await scene("s1-chat", sv, async (page, a) => {
  await page.goto(`${BASE}/chat`, { waitUntil: "load" });
  await page.getByRole("textbox").last().waitFor();
  await sleep(1200);
  await a.start();
  await a.cap(CAPTIONS[0]);
  await sleep(2800);
  // mở một cuộc hỏi đáp đã có (dữ liệu seed), không gửi câu mới
  const session = page.getByRole("button", { name: /Điều kiện để được dự thi/ }).first();
  await a.click(session);
  await sleep(1200);
  await a.cap(CAPTIONS[1]);
  await sleep(5500);
  await page.mouse.wheel(0, 300);
  await sleep(3500);
});

await scene("s1b-inbox", gv, async (page, a) => {
  await page.goto(`${BASE}/inbox`, { waitUntil: "load" });
  await sleep(2200);
  await a.start();
  await a.cap(CAPTIONS[2]);
  await sleep(2500);
  await a.move(page.locator("main li, main tbody tr").first());
  await sleep(4500);
});

await scene("s2-review", gv, async (page, a) => {
  await page.goto(`${BASE}/grading`, { waitUntil: "load" });
  await sleep(1500);
  await a.start();
  await a.cap(CAPTIONS[3]);
  await sleep(2600);
  const row = page.locator("main tbody tr").first();
  await a.click(row);
  await page.waitForURL(/\/grading\//);
  await page.mouse.move(1100, 880, { steps: 8 }); // đưa con trỏ ra khỏi vùng nội dung trước khi lấy poster
  await sleep(1500);
  a.posterHere();
  await sleep(3800);
  await a.cap(CAPTIONS[4]);
  await page.mouse.wheel(0, 380);
  await sleep(5200);
});

await scene("s3-gradebook", gv, async (page, a) => {
  await page.goto(`${BASE}/gradebook`, { waitUntil: "load" });
  await sleep(1800);
  await a.start();
  await a.cap(CAPTIONS[5]);
  await sleep(3200);
  // menu hành động của dòng đầu (chỉ xem): mở "Giải trình" nếu có, không sửa ô nào
  await a.click(page.locator("main tbody tr", { hasText: /\d{8}/ }).first().getByRole("button").last());
  await sleep(1000);
  const explain = page.getByRole("menuitem", { name: /Giải trình/ }).first();
  if (await explain.count()) { await a.click(explain); await sleep(600); }
  await sleep(6000);
});

await scene("s4-today", gv, async (page, a) => {
  await page.goto(`${BASE}/`, { waitUntil: "load" });
  await sleep(2000);
  await a.start();
  await a.cap(CAPTIONS[6]);
  await sleep(5500);
});

await scene("s5-outro", undefined, async (page, a) => {
  await page.goto(`${BASE}/login`, { waitUntil: "load" });
  await card(page, `<img src="/brand/logo-edupilot.svg" alt="" style="height:56px;margin:0 auto"><p style="font-size:48px;font-weight:700;letter-spacing:-.02em;line-height:1.2;max-width:22ch;margin:0 auto">Che danh tính. Điểm tính bằng mã. Người quyết.</p><p style="font-size:26px;color:#52525b;margin:0">Dữ liệu trong video là dữ liệu mô phỏng</p>`);
  await a.start();
  await sleep(4000);
});

fs.writeFileSync(`${OUT}/frames.ffconcat`, frameLines.join("\n"));
meta.total = +elapsed.toFixed(2);
fs.writeFileSync(`${OUT}/meta.json`, JSON.stringify(meta, null, 2));
console.log("tổng", meta.total, "s; poster ở", meta.poster.toFixed(1), "s");
await browser.close();
