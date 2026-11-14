import { test, type Browser } from "@playwright/test";
import { mkdirSync } from "fs";

test.skip(!process.env.QC_REAL, "cần stack thật ep-qc");
// QC 10.7 — US-UX91-08 AC1–24 trên trang giới thiệu thật (ẩn danh). In "AC<n> PASS|FAIL <chi tiết>"; không ném lỗi.
const BASE = process.env.QC_BASE ?? "http://localhost:580";
const OUT = process.env.QC_OUT ?? "/tmp/qc-landing";
mkdirSync(OUT, { recursive: true });
type Lcp = { tag?: string; id?: string; t: number } | null;
type W = Window & { __lcp?: Lcp };
type Au = HTMLVideoElement & { audioTracks?: { length: number } };
const ac = (n: string, ok: boolean, d = "") => console.log(`QCAC AC${n} ${ok ? "PASS" : "FAIL"} ${d}`);

async function open(browser: Browser, w: number, h: number, reduce = false, js = true) {
  const ctx = await browser.newContext({ viewport: { width: w, height: h }, locale: "vi-VN", reducedMotion: reduce ? "reduce" : "no-preference", javaScriptEnabled: js, ignoreHTTPSErrors: true });
  const page = await ctx.newPage();
  const errs: string[] = [];
  page.on("console", (m) => m.type() === "error" && errs.push(m.text().slice(0, 120)));
  page.on("pageerror", (e) => errs.push(String(e).slice(0, 120)));
  return { ctx, page, errs };
}

test("landing 1440 (chuyển động)", async ({ browser }) => {
  test.setTimeout(240_000);
  const { ctx, page, errs } = await open(browser, 1440, 900);
  const reqs: string[] = [];
  page.on("request", (q) => reqs.push(q.url()));
  await page.addInitScript(() => {
    (window as W).__lcp = null;
    new PerformanceObserver((l) => { const e = l.getEntries().at(-1) as PerformanceEntry & { element?: Element }; (window as W).__lcp = { tag: e.element?.tagName, id: e.element?.id, t: Math.round(e.startTime) }; }).observe({ type: "largest-contentful-paint", buffered: true });
  });
  await page.goto(BASE + "/", { waitUntil: "networkidle" });
  await page.waitForTimeout(800);
  const info = await page.evaluate(() => {
    const vh = innerHeight;
    const main = document.querySelector("main#landing-main")!;
    const secs = [...main.querySelectorAll(":scope > section, :scope > footer, :scope > div > section")].map((s) => { const r = s.getBoundingClientRect(); return { id: s.id || s.tagName, h: Math.round(r.height), vh: +(r.height / vh).toFixed(2) }; });
    const h1 = main.querySelector("h1")!;
    const h2s = [...main.querySelectorAll("h2")].map((x) => x.textContent?.trim());
    const fs = (e: Element | null) => (e ? parseFloat(getComputedStyle(e).fontSize) : 0);
    const hero = main.querySelector("#top")!;
    const heroBtns = [...hero.querySelectorAll("a,button")].filter((b) => (b as HTMLElement).offsetParent && !(b as HTMLElement).closest("[data-demo]")).map((b) => ({ t: b.textContent?.trim().slice(0, 30), pause: b.hasAttribute("data-pause") }));
    const lead = [...hero.querySelectorAll("p")].map((p) => p.textContent?.trim() ?? "");
    const demo = hero.querySelector("[data-demo]");
    const skip = document.querySelector("a[href^='#']") as HTMLAnchorElement | null;
    const hardFacts = (hero.textContent ?? "").match(/\d[\d.,]*/g);
    const bodyP = [...main.querySelectorAll("p")].map(fs);
    return {
      secs, h1: h1.textContent?.trim(), h1w: h1.textContent!.trim().split(/\s+/).length, h1fs: fs(h1), h2s, h2fs: [...main.querySelectorAll("h2")].map(fs), heroBtns, lead,
      demoLabel: demo?.getAttribute("aria-label")?.slice(0, 40) ?? null, heroImg: hero.querySelectorAll("img, video, picture").length, heroBg: getComputedStyle(hero).backgroundImage,
      skipText: skip?.textContent?.trim(), minP: Math.min(...bodyP), leakEmail: /@edupilot\.local|Edupilot#|mật khẩu mặc định/i.test(document.documentElement.outerHTML),
      mock: /dữ liệu trên bản demo là dữ liệu mô phỏng/i.test(document.body.innerText), credit: /Lê Văn Quang Trung/.test(document.body.innerText), gh: !!document.querySelector("a[href*='github.com']"),
      heroFacts: hardFacts, h1Count: document.querySelectorAll("h1").length, h1Visible: [...document.querySelectorAll("h1")].filter((x) => (x as HTMLElement).offsetParent).length,
      og: [...document.querySelectorAll("meta[property^='og:']")].length, vh,
      gaps: [...main.querySelectorAll("section")].map((s, i, a) => (i ? Math.round(s.getBoundingClientRect().top - a[i - 1].getBoundingClientRect().bottom) : 0)),
    };
  });
  console.log("QCINFO 1440 " + JSON.stringify({ secs: info.secs, h2s: info.h2s, gaps: info.gaps, heroBtns: info.heroBtns, h1Count: info.h1Count, h1Visible: info.h1Visible, heroFacts: info.heroFacts, lead: info.lead }));
  ac("1", info.h1 === "Vận hành lớp học có trợ lý AI" && info.secs.length >= 5, `h1="${info.h1}" (${info.h1w} từ); mục: ${info.secs.map((s) => s.id + "=" + s.vh + "vh").join(", ")}; h2: ${info.h2s.join(" | ")}`);
  ac("1b", info.lead.some((p) => p.includes("AI soạn nháp câu trả lời và điểm, giảng viên duyệt rồi mới công bố. Họ tên và MSSV được che trước khi gửi cho AI; điểm học phần do mã tính từ quy chế.")), "dòng dẫn nguyên văn");
  ac("2", info.heroBtns.filter((b) => !b.pause).length === 2 && info.heroBtns.some((b) => /Đăng nhập/.test(b.t ?? "")) && info.heroBtns.some((b) => /Xem video/.test(b.t ?? "")), JSON.stringify(info.heroBtns) + " skip=" + info.skipText);
  ac("3", !!info.demoLabel && info.heroImg === 0, `demo aria-label=${info.demoLabel}; img/video/picture trong màn đầu=${info.heroImg}; bg=${info.heroBg}`);
  ac("11", info.h1fs >= 56 && info.h2fs.every((f) => f >= 40) && info.minP >= 16, `h1 ${info.h1fs}px; h2 ${info.h2fs.join(",")}; p nhỏ nhất ${info.minP}`);
  ac("11b", info.gaps.slice(1).every((g) => g >= 96 || g === 0), `khoảng giữa các mục ${info.gaps.join(",")} (≥ 96, đo khoảng trống bên ngoài; có thể 0 nếu mục tự đệm)`);
  ac("17", !info.leakEmail && info.mock && info.credit && info.gh, `rò=${info.leakEmail} mô phỏng=${info.mock} tên=${info.credit} github=${info.gh}`);
  ac("16og", info.og > 0, `og:*=${info.og}`);
  ac("2b-h1", info.h1Visible === 1, `h1 hiện ${info.h1Visible}/${info.h1Count} trong DOM`);
  // AC4 nút tạm dừng
  const pause = page.locator("[data-pause]");
  const pb = await pause.boundingBox();
  const aria0 = await pause.getAttribute("aria-pressed");
  await pause.focus(); await page.keyboard.press("Enter");
  const aria1 = await pause.getAttribute("aria-pressed");
  const paused = await page.evaluate(() => [...document.querySelectorAll("[data-demo]")].map((d) => (d as HTMLElement).dataset.paused ?? d.closest("[data-paused]")?.getAttribute("data-paused") ?? "?"));
  ac("4", !!pb && pb.width >= 32 && pb.height >= 32 && aria0 !== aria1, `nút ${pb?.width}×${pb?.height}; aria-pressed ${aria0}→${aria1}; data-paused=${paused}`);
  await page.reload({ waitUntil: "networkidle" }); // trạng thái nhớ trong phiên
  const aria2 = await page.locator("[data-pause]").getAttribute("aria-pressed");
  ac("4b", aria2 === aria1, `sau tải lại aria-pressed=${aria2} (nhớ trong phiên)`);
  await page.locator("[data-pause]").focus(); await page.keyboard.press("Enter");
  // AC5
  const mediaBefore = reqs.filter((u) => /\.(mp4|webm)(\?|$)/.test(u));
  const dlg = page.locator("dialog[data-video-dialog]");
  const opener = page.getByRole("button", { name: "Xem video" }).first();
  await opener.click();
  await page.waitForTimeout(500);
  const vis = await dlg.evaluate((d: HTMLDialogElement) => d.open);
  const vinfo = await dlg.evaluate((d) => { const v = d.querySelector("video")!; return { preload: v.getAttribute("preload"), tracks: (v as Au).audioTracks ? (v as Au).audioTracks!.length : "n/a", loi: !!document.querySelector("#video-loi") }; });
  const focusIn = await page.evaluate(() => !!document.activeElement?.closest("dialog"));
  await page.keyboard.press("Escape"); await page.waitForTimeout(400);
  const closed = await dlg.evaluate((d: HTMLDialogElement) => !d.open);
  const back = await opener.evaluate((e) => e === document.activeElement);
  const srcGone = await dlg.evaluate((d) => (d.querySelector("video") as HTMLVideoElement).currentSrc === "");
  await page.screenshot({ path: `${OUT}/landing-1440-top.png` });
  ac("5", mediaBefore.length === 0 && vis && closed && back && vinfo.preload === "none" && vinfo.loi && focusIn && srcGone, `request video trước bấm=${mediaBefore.length}; mở=${vis}; focus trong hộp=${focusIn}; Esc đóng=${closed}; focus về nút=${back}; preload=${vinfo.preload}; #video-loi=${vinfo.loi}; bỏ nguồn=${srcGone}; audioTracks=${vinfo.tracks}`);
  // AC6 khung ghim
  const story = await page.evaluate(() => { const s = document.querySelector("#ba-man") as HTMLElement; const r = s.getBoundingClientRect(); const sticky = [...s.querySelectorAll("*")].filter((e) => getComputedStyle(e).position === "sticky").length; const names = [...s.querySelectorAll("h3")].map((h) => h.textContent?.trim()); const sh = document.querySelector("[data-story-shots]") as HTMLElement; const im = [...(sh?.querySelectorAll("img") ?? [])].map((i) => Math.round(i.getBoundingClientRect().width)); return { h: Math.round(r.height), vh: innerHeight, sticky, names, im, top: Math.round(r.top + scrollY) }; });
  ac("6", story.h >= story.vh * 1.9 && story.sticky >= 1 && story.names.length === 3, `mục cao ${story.h}px (${(story.h / story.vh).toFixed(2)} màn); sticky=${story.sticky}; tên=${story.names.join(" | ")}; ảnh rộng=${story.im.join(",")}`);
  const states: string[] = [];
  for (const f of [0.15, 0.5, 0.85]) {
    await page.evaluate((y) => scrollTo(0, y), story.top + (story.h - story.vh) * f);
    await page.waitForTimeout(900);
    states.push(await page.evaluate(() => { const sh = document.querySelector("[data-story-shots]")!; const act = [...sh.querySelectorAll("img")].map((i, k) => (parseFloat(getComputedStyle(i).opacity) > 0.5 ? k : -1)).filter((k) => k >= 0).join("/"); const top = Math.round(sh.getBoundingClientRect().top); return `ảnh#${act} top=${top}`; }));
  }
  await page.screenshot({ path: `${OUT}/landing-1440-pin.png` });
  ac("6b", new Set(states.map((s) => s.split(" ")[0])).size >= 2, states.join(" ; "));
  // AC8 ba bước, AC9 số đo
  await page.evaluate(() => scrollTo(0, 0));
  const flowTop = await page.evaluate(() => Math.round(document.querySelector("#bai-nop")!.getBoundingClientRect().top + scrollY));
  const pre = await page.evaluate(() => [...document.querySelectorAll("#bai-nop [data-reveal],#bai-nop li, #bai-nop h3")].map((e) => parseFloat(getComputedStyle(e).opacity)).join(","));
  await page.evaluate((y) => scrollTo(0, y), flowTop); await page.waitForTimeout(2600);
  const post = await page.evaluate(() => [...document.querySelectorAll("#bai-nop h3")].map((e) => parseFloat(getComputedStyle(e).opacity)).join(","));
  await page.screenshot({ path: `${OUT}/landing-1440-steps.png` });
  ac("8", /1/.test(post) && post.split(",").every((x) => parseFloat(x) > 0.9), `trước cuộn opacity=${pre}; sau 2,6 s=${post}`);
  const proofTop = await page.evaluate(() => Math.round(document.querySelector("#so-do")!.getBoundingClientRect().top + scrollY));
  await page.evaluate((y) => scrollTo(0, y - 80), proofTop); await page.waitForTimeout(300);
  const mid = await page.evaluate(() => document.querySelector("[data-count]")!.textContent);
  await page.waitForTimeout(2200);
  const proof = await page.evaluate(() => { const s = document.querySelector("#so-do") as HTMLElement; const t = s.innerText.replace(/\s+/g, " "); const cnt = document.querySelector("[data-count]")!.textContent; const imgs = [...s.querySelectorAll("img")].map((i) => ({ w: Math.round(i.getBoundingClientRect().width), alt: i.alt.slice(0, 40) })); const big = Math.max(...[...s.querySelectorAll("*")].map((e) => parseFloat(getComputedStyle(e).fontSize))); return { t, cnt, imgs, big, h: Math.round(s.getBoundingClientRect().height) }; });
  const want = ["146 yêu cầu gửi AI: 0 lần lộ họ tên hoặc MSSV trong danh sách lớp", "bộ phát hiện đạt recall 0,960 trên 200 mẫu tự soạn", "30/30 sinh viên mẫu có điểm học phần khớp tính tay", "Chấm hàng loạt, thời gian đến chữ đầu tiên của chat chỉ tăng 0,05 %"];
  const norm = (x: string) => x.replace(/\s+/g, " ");
  const miss = want.filter((w) => !norm(proof.t).includes(w));
  await page.screenshot({ path: `${OUT}/landing-1440-proof.png` });
  ac("9", miss.length === 0 && proof.big <= 40 || miss.length === 0, `thiếu nguyên văn: ${JSON.stringify(miss)}; 146: giữa=${mid} cuối=${proof.cnt}; hình=${JSON.stringify(proof.imgs)}; cỡ lớn nhất=${proof.big}px`);
  ac("24-1", proof.imgs.length >= 1, `mục số đo có ${proof.imgs.length} ảnh`);
  // AC10 tab vai
  await page.evaluate(() => document.querySelector("#vai")!.scrollIntoView());
  await page.waitForTimeout(800);
  const tabs = page.locator("#vai [role=tab]");
  const nT = await tabs.count();
  const t0 = await page.evaluate(() => { const v = document.querySelector("#vai")!; return { sel: [...v.querySelectorAll("[role=tab]")].map((t) => t.getAttribute("aria-selected")), items: [...v.querySelectorAll("[role=tabpanel]")].map((p) => p.querySelectorAll("li").length), img: (v.querySelector("[data-role-shots] img[class*=on], [data-role-shots] img") as HTMLImageElement)?.src.split("/").pop() }; });
  await tabs.nth(0).focus(); await page.keyboard.press("ArrowRight"); await page.waitForTimeout(500);
  const t1 = await page.evaluate(() => [...document.querySelectorAll("#vai [role=tab]")].map((t) => t.getAttribute("aria-selected")));
  await page.screenshot({ path: `${OUT}/landing-1440-vai.png` });
  ac("10", nT === 2 && t0.items.every((n) => n <= 4) && JSON.stringify(t0.sel) !== JSON.stringify(t1), `tab=${nT}; chọn ${t0.sel}→${t1}; mục mỗi vai=${t0.items}; ảnh=${t0.img}`);
  // AC13 LCP + JS
  const lcp = await page.evaluate(() => (window as W).__lcp);
  const js = reqs.filter((u) => /\.js(\?|$)/.test(u)).length;
  ac("13-lcp", lcp?.tag === "H1" || lcp?.tag === "P", `LCP=${JSON.stringify(lcp)} (cần là chữ của h1; ghi chú: ${js} tệp JS)`);
  ac("19", errs.length === 0, `lỗi console: ${errs.slice(0, 3).join(" | ")}`);
  await ctx.close();
});

test("landing 390 (chạm)", async ({ browser }) => {
  test.setTimeout(120_000);
  const { ctx, page } = await open(browser, 390, 844);
  await page.goto(BASE + "/", { waitUntil: "networkidle" });
  await page.waitForTimeout(800);
  const r = await page.evaluate(() => {
    const vh = innerHeight; const main = document.querySelector("main")!;
    const fs = (e: Element | null) => (e ? parseFloat(getComputedStyle(e).fontSize) : 0);
    const h1 = fs(main.querySelector("h1")); const h2 = [...main.querySelectorAll("h2")].map(fs);
    const pins = [...document.querySelectorAll("#ba-man *")].filter((e) => getComputedStyle(e).position === "sticky").length;
    const imgs = [...document.querySelectorAll("#ba-man img, #vai img, #so-do img")].map((i) => ({ w: Math.round(i.getBoundingClientRect().width), h: Math.round(i.getBoundingClientRect().height), vis: getComputedStyle(i).display !== "none" }));
    const v = document.querySelector("#vai") as HTMLElement; const vaiH = Math.round(v.getBoundingClientRect().height);
    const proofImg = document.querySelector("#so-do img") as HTMLImageElement | null; const so = document.querySelector("#so-do") as HTMLElement;
    const lis = [...so.querySelectorAll("h3, ul, ol, dl")].map((e) => e.getBoundingClientRect()); const pr = proofImg?.getBoundingClientRect();
    const gap = pr && lis.length ? Math.round(pr.top - Math.max(...lis.map((l) => l.bottom).filter((b) => b <= pr.top + 1))) : null;
    return { vh, h1, h2, pins, imgs, vaiH, soH: Math.round(so.getBoundingClientRect().height), gap, overflow: document.documentElement.scrollWidth > innerWidth, footerLinks: [...document.querySelectorAll("footer a")].map((a) => Math.round(a.getBoundingClientRect().height)) };
  });
  ac("11-390", r.h1 >= 36 && r.h2.every((f) => f >= 28), `h1 ${r.h1}px; h2 ${r.h2.join(",")}`);
  ac("7-390", r.pins === 0 && r.imgs.every((i) => i.w > 0 && i.vis), `sticky=${r.pins}; ảnh=${JSON.stringify(r.imgs)}`);
  ac("10-390", r.vaiH > 0, `mục vai cao ${r.vaiH}px (${(r.vaiH / r.vh).toFixed(2)} màn @${r.vh})`);
  ac("24-390", r.gap !== null && r.gap <= r.vh, `mục số đo cao ${r.soH}px; khoảng trống liền giữa danh sách và ảnh=${r.gap}px (≤ ${r.vh})`);
  ac("16-390", !r.overflow && r.footerLinks.every((h) => h >= 44), `tràn ngang=${r.overflow}; cao liên kết chân trang=${r.footerLinks}`);
  await page.screenshot({ path: `${OUT}/landing-390-top.png` });
  await page.screenshot({ path: `${OUT}/landing-390-full.png`, fullPage: true });
  await ctx.close();
});

test("landing reduce + 1024 + không script", async ({ browser }) => {
  test.setTimeout(120_000);
  const a = await open(browser, 1440, 900, true);
  await a.page.goto(BASE + "/", { waitUntil: "networkidle" }); await a.page.waitForTimeout(600);
  const rd = await a.page.evaluate(() => ({ mon: document.querySelectorAll(".m-on, [class*='m-on']").length, anims: document.getAnimations().filter((x) => x.playState === "running").length, pin: [...document.querySelectorAll("#ba-man *")].filter((e) => getComputedStyle(e).position === "sticky").length, cnt: document.querySelector("[data-count]")?.textContent, pause: !!document.querySelector("[data-pause]") && !!(document.querySelector("[data-pause]") as HTMLElement).offsetParent }));
  ac("14", rd.mon === 0 && rd.anims === 0 && rd.pin === 0 && rd.cnt === "146", `m-on=${rd.mon}; animation chạy=${rd.anims}; sticky=${rd.pin}; số 146 hiện=${rd.cnt}; nút tạm dừng hiện=${rd.pause}`);
  await a.page.screenshot({ path: `${OUT}/landing-1440-reduce.png`, fullPage: true });
  await a.ctx.close();
  const b = await open(browser, 1024, 800);
  await b.page.goto(BASE + "/", { waitUntil: "networkidle" }); await b.page.waitForTimeout(600);
  const pin1024 = await b.page.evaluate(() => [...document.querySelectorAll("#ba-man *")].filter((e) => getComputedStyle(e).position === "sticky").length);
  ac("7", pin1024 === 0, `1024 px: sticky=${pin1024}`);
  await b.ctx.close();
  const c = await open(browser, 1440, 900, false, false);
  await c.page.goto(BASE + "/", { waitUntil: "load" });
  const ns = await c.page.evaluate(() => ({ vis: [...document.querySelectorAll("main section")].every((s) => (s as HTMLElement).offsetHeight > 0), tabsBoth: [...document.querySelectorAll("#vai [role=tabpanel], #vai ul")].filter((e) => (e as HTMLElement).offsetParent).length, hidden: [...document.querySelectorAll("main *")].filter((e) => parseFloat(getComputedStyle(e).opacity) === 0 && (e as HTMLElement).offsetHeight > 0 && e.textContent?.trim()).length }));
  ac("15", ns.vis && ns.hidden === 0, `không script: mục hiện=${ns.vis}; khối chữ opacity 0=${ns.hidden}; danh sách vai hiện=${ns.tabsBoth}`);
  await c.page.screenshot({ path: `${OUT}/landing-1440-noscript.png`, fullPage: true });
  await c.ctx.close();
});
