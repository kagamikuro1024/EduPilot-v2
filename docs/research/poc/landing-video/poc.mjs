import { chromium } from "playwright-core";
import fs from "node:fs";
const URL = "https://v2202609419695521098.happysrv.de/";
const b = await chromium.launch();
async function run(mode) {
  const ctx = await b.newContext({ viewport: { width: 1440, height: 900 }, locale: "vi-VN" });
  const page = await ctx.newPage();
  fs.mkdirSync(`frames-${mode}`, { recursive: true });
  const ts = [];
  let n = 0;
  if (mode === "webm") await page.screencast.start({ path: "pw.webm", size: { width: 1440, height: 900 } });
  else await page.screencast.start({ size: { width: 1440, height: 900 }, quality: 90, onFrame: ({ data, timestamp }) => { const f = `frames-${mode}/${String(n++).padStart(5, "0")}.jpg`; fs.writeFileSync(f, data); ts.push([f, timestamp]); } });
  const act = await page.screencast.showActions({ cursor: "pointer", duration: 600 });
  await page.goto(URL, { waitUntil: "networkidle" });
  const cap = await page.screencast.showOverlay(`<div style="position:fixed;left:50%;bottom:48px;transform:translateX(-50%);padding:10px 18px;border-radius:8px;background:rgba(20,20,20,.82);color:#fff;font:600 26px/1.4 Be Vietnam Pro,system-ui">Sinh viên hỏi riêng — tên và MSSV được che trước khi gửi AI</div>`);
  for (let y = 0; y < 2400; y += 40) { await page.mouse.wheel(0, 40); await page.waitForTimeout(60); }
  await page.mouse.move(700, 400); await page.waitForTimeout(800);
  await cap[Symbol.asyncDispose]?.(); await act[Symbol.asyncDispose]?.();
  await page.screencast.showChapter("Giảng viên duyệt bài", { description: "AI chấm nháp, giảng viên quyết", duration: 1500 });
  await page.waitForTimeout(1600);
  await page.screencast.stop();
  await ctx.close();
  if (mode === "jpg") { const lines = ["ffconcat version 1.0"]; for (let i = 0; i < ts.length; i++) { const d = i + 1 < ts.length ? (ts[i + 1][1] - ts[i][1]) / 1000 : 0.04; lines.push(`file ${ts[i][0]}`, `duration ${d.toFixed(4)}`); } fs.writeFileSync("frames.ffconcat", lines.join("\n")); console.log("jpg frames", ts.length, "span ms", ts.at(-1)?.[1] - ts[0]?.[1]); }
}
await run("webm"); await run("jpg"); await b.close();
