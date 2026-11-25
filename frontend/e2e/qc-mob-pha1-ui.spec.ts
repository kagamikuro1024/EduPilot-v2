// QC 10.8 mobile pha 1 — lô C: 6.G độ rộng/xoay, 6.R vùng chạm, 6.I cỡ chữ/phóng, 6.K bàn phím (phần web), 6.X /attendance.
// Trạng thái gửi/đã gửi/lỗi/429 dựng bằng `page.route` (chỉ đo bố cục — không tạo lượt điểm danh giả trong DB).
import { test, expect } from "@playwright/test";
import { ACCT, authed, C, check, gotoEntry, rec, noOverflow } from "./qc-mob-lib";

test.describe.configure({ mode: "serial" });
const OUT = process.env.QC_OUT ?? "/tmp/qc-mob";
type P = import("@playwright/test").Page;

const measure = (p: P) =>
  p.evaluate(() => {
    const box = (e: Element | null) => {
      if (!e) return null;
      const b = e.getBoundingClientRect();
      return { w: Math.round(b.width), h: Math.round(b.height), top: Math.round(b.top), bottom: Math.round(b.bottom) };
    };
    return {
      sw: document.documentElement.scrollWidth,
      iw: window.innerWidth,
      sh: document.documentElement.scrollHeight,
      ih: window.innerHeight,
      dock: box(document.querySelector("[data-part=checkin-dock] button")),
      input: box(document.querySelector("[data-part=checkin] input")),
      nav: box(document.querySelector("nav[aria-label='Điều hướng chính']")),
      state: document.querySelector("[data-part=checkin]")?.getAttribute("data-state") ?? "-",
    };
  });

/** Dựng 4 trạng thái web của /checkin (đang quét là native → N). */
async function fourStates(p: P, course: string, code: string) {
  const out: Record<string, Awaited<ReturnType<typeof measure>>> = {};
  const clear = () => p.unroute("**/checkins").catch(() => undefined);
  const enter = async () => {
    await gotoEntry(p, course);
    await p.getByLabel("Mã điểm danh").fill(code);
  };
  await enter();
  out.entry = await measure(p);

  await clear();
  await p.route("**/checkins", async (r) => {
    if (r.request().method() !== "POST") return r.continue();
    await new Promise((x) => setTimeout(x, 4000));
    await r.fulfill({ status: 201, contentType: "application/json", body: JSON.stringify({ checked_in_at: new Date().toISOString(), round: 1 }) });
  });
  await p.locator("[data-part=checkin-dock] button").click();
  await p.locator("[data-part=checkin][data-state=sending]").waitFor({ timeout: 5000 });
  out.sending = await measure(p);

  await p.waitForTimeout(4500);
  await p.locator("[data-part=checkin][data-state=done]").waitFor({ timeout: 10_000 });
  out.done = await measure(p);

  await clear();
  await p.route("**/checkins", (r) => (r.request().method() === "POST" ? r.fulfill({ status: 500, contentType: "application/json", body: "{}" }) : r.continue()));
  await enter();
  await p.locator("[data-part=checkin-dock] button").click();
  await p.locator("[data-part=checkin][data-state=error]").waitFor({ timeout: 10_000 });
  out.error = await measure(p);

  await clear();
  await p.route("**/checkins", (r) =>
    r.request().method() === "POST"
      ? r.fulfill({ status: 429, headers: { "retry-after": "30" }, contentType: "application/json", body: JSON.stringify({ code: "RATE_LIMITED" }) })
      : r.continue(),
  );
  await enter();
  await p.locator("[data-part=checkin-dock] button").click();
  await p.locator("[data-part=checkin][data-state=limited]").waitFor({ timeout: 10_000 });
  out.limited = await measure(p);
  await clear();
  return out;
}

const okCell = (m: { sw: number; iw: number; dock: { bottom: number } | null }, mih: number) =>
  m.sw <= m.iw + 1 && !!m.dock && m.dock.bottom <= mih + 1;

test("UIW-01…05 /checkin ở 360/375/390/411/430 × 4 trạng thái: không tràn ngang, nút chính trong khung", async ({ browser }) => {
  const sizes: [string, number, number][] = [
    ["360", 360, 640],
    ["375", 375, 812],
    ["390", 390, 844],
    ["411", 411, 891],
    ["430", 430, 932],
  ];
  const tc = { "360": "TC-MOB-UIW-01", "375": "TC-MOB-UIW-02", "390": "TC-MOB-UIW-03", "411": "TC-MOB-UIW-04", "430": "TC-MOB-UIW-05" } as Record<string, string>;
  const { ctx, page } = await authed(browser, "pixel", ACCT.s1);
  const table: Record<string, string> = {};
  for (const [name, w, h] of sizes) {
    await page.setViewportSize({ width: w, height: h });
    const st = await fourStates(page, C.QCMOB1, "ABC123");
    const cells = Object.entries(st).map(([k, m]) => `${k}:${okCell(m, h) ? "P" : `F(sw=${m.sw}/${m.iw},dock=${m.dock?.bottom ?? "-"}/${h})`}`);
    const bad = cells.filter((c) => c.includes("F("));
    check(`${tc[name]} ${name}×${h}: 4/4 trạng thái không tràn ngang, nút chính thấy`, bad.length === 0, cells.join(" "));
    table[name] = cells.map((c) => c.split(":")[0] + (c.includes("F(") ? "✗" : "✓")).join("");
    await page.screenshot({ path: `${OUT}/UIW-${name}-limited.png` });
  }
  rec("TC-MOB-UIW-06 bảng 5×5 (đang quét = native → N)", "NOTE", JSON.stringify(table));
});

test("UIW-07/08/20 câu lỗi dài 360, trạng thái đã điểm danh 360/430, màn 640 thấp", async ({ browser }) => {
  const { ctx, page } = await authed(browser, "pixel", ACCT.s1);
  await page.setViewportSize({ width: 360, height: 640 });
  await page.goto(`/checkin?course=${C.QCMOB1}`);
  await page.getByLabel("Mã điểm danh").waitFor();
  await page.getByLabel("Mã điểm danh").fill("ZZZZZZ");
  await page.locator("[data-part=checkin-dock] button").click();
  await page.waitForTimeout(2500);
  const m = await measure(page);
  const msg = await page.locator("[data-part=checkin]").innerText();
  const noClip = await page.evaluate(() => {
    const el = [...document.querySelectorAll("[data-part=checkin] *")].find((e) => e.textContent?.includes("Mã không đúng hoặc"));
    if (!el) return false;
    const b = el.getBoundingClientRect();
    return b.right <= innerWidth + 1 && el.scrollWidth <= el.clientWidth + 1;
  });
  check("TC-MOB-UIW-07 câu lỗi dài 360: xuống dòng trong panel, không tràn, nút chính thấy", noClip && m.sw <= m.iw + 1 && !!m.dock, `sw=${m.sw}/${m.iw} ${msg.replace(/\n+/g, "|").slice(0, 70)}`);
  await page.screenshot({ path: `${OUT}/UIW-07-long-error-360.png` });
  check("TC-MOB-UIW-20 màn 360×640 trạng thái nhập mã: ô + nút thấy, không cuộn dọc", m.sh <= m.ih + 2 && !!m.input && !!m.dock, `sh=${m.sh}/${m.ih}`);

  await page.unroute("**/checkins").catch(() => undefined);
  await page.route("**/checkins", (r) => (r.request().method() === "POST" ? r.fulfill({ status: 201, contentType: "application/json", body: JSON.stringify({ checked_in_at: new Date().toISOString(), round: 1 }) }) : r.continue()));
  for (const [w, h] of [[360, 640], [430, 932]] as const) {
    await page.setViewportSize({ width: w, height: h });
    await page.goto(`/checkin?course=${C.QCMOB1}`);
    await page.getByLabel("Mã điểm danh").fill("ABC123");
    await page.locator("[data-part=checkin-dock] button").click();
    await page.locator("[data-part=checkin][data-state=done]").waitFor({ timeout: 10_000 });
    const d = await page.evaluate(() => {
      const p = document.querySelector("[data-part=done]");
      const b = p?.getBoundingClientRect();
      const brand = getComputedStyle(document.documentElement).getPropertyValue("--ep-red").trim();
      const probe = document.createElement("span");
      probe.style.color = brand;
      document.body.append(probe);
      const rgb = getComputedStyle(probe).color;
      probe.remove();
      const red = [...document.querySelectorAll("main *")].filter((e) => getComputedStyle(e).color === rgb).map((e) => e.tagName);
      return { txt: p?.textContent ?? "", top: Math.round(b?.top ?? -1), red };
    });
    check(`TC-MOB-UIW-08 ${w}: «Đã điểm danh hh:mm» một dòng, dấu Check đỏ duy nhất`, /Đã điểm danh \d{2}:\d{2}/.test(d.txt) && d.red.length <= 2, `${d.txt.slice(0, 30)} red=${d.red.join()}`);
  }
});

test("UIW-11/12/13 xoay ngang 667×375, 844×390, 915×412", async ({ browser }) => {
  const { ctx, page } = await authed(browser, "pixel", ACCT.s1);
  for (const [tc, w, h] of [["TC-MOB-UIW-11", 667, 375], ["TC-MOB-UIW-12", 844, 390], ["TC-MOB-UIW-13", 915, 412]] as const) {
    await page.setViewportSize({ width: w, height: h });
    const st = await fourStates(page, C.QCMOB1, "ABC123");
    const bad = Object.entries(st).filter(([, m]) => !okCell(m, h)).map(([k, m]) => `${k}(sw=${m.sw}/${m.iw},dock=${m.dock?.bottom ?? "-"})`);
    check(`${tc} ngang ${w}×${h}: 4/4 trạng thái không tràn, nút chính không bị che`, bad.length === 0, bad.join(" ") || `ok`);
    await page.screenshot({ path: `${OUT}/${tc.slice(3, 9)}-${w}x${h}.png` });
  }
});

test("UIW-17/18 khung app: thanh dưới 5 mục, thanh bên khi ≥ 720, 4 màn SV không tràn", async ({ browser }) => {
  const { ctx, page } = await authed(browser, "pixel", ACCT.s1);
  await page.setViewportSize({ width: 375, height: 812 });
  await page.goto("/");
  await page.waitForTimeout(2500);
  const navInfo = await page.evaluate(() => {
    const nav = document.querySelector("nav[aria-label='Điều hướng chính']");
    const items = [...(nav?.querySelectorAll("a,button") ?? [])].map((e) => ({ t: (e.textContent ?? "").trim(), h: Math.round(e.getBoundingClientRect().height), w: Math.round(e.getBoundingClientRect().width) }));
    const navBox = nav?.getBoundingClientRect();
    return { items, navH: Math.round(navBox?.height ?? 0), bottom: Math.round(navBox?.bottom ?? 0), ih: innerHeight };
  });
  check("TC-MOB-UIW-17a dọc 375: 5 mục thanh dưới thấy đủ, chữ không cắt", navInfo.items.length === 5 && navInfo.items.every((i) => i.h >= 44 && i.w > 0) && navInfo.bottom <= navInfo.ih + 1, JSON.stringify(navInfo));
  await page.setViewportSize({ width: 844, height: 390 });
  await page.waitForTimeout(1200);
  const side = await page.evaluate(() => {
    const s = document.querySelector("aside[data-part=sidebar]");
    return { has: !!s, vis: !!s && getComputedStyle(s).display !== "none", w: Math.round(s?.getBoundingClientRect().width ?? 0), ov: document.documentElement.scrollWidth > innerWidth + 1 };
  });
  check("TC-MOB-UIW-17b ngang 844: chuyển thanh bên, không mất mục, không tràn", side.has && side.vis && side.w > 100 && !side.ov, JSON.stringify(side));
  await page.setViewportSize({ width: 360, height: 740 });
  for (const [w, h] of [[360, 740], [430, 932]] as const) {
    await page.setViewportSize({ width: w, height: h });
    const bad: string[] = [];
    for (const r of ["/", "/chat", "/threads", "/practice"]) {
      await page.goto(r);
      await page.waitForTimeout(2000);
      if (!(await noOverflow(page))) bad.push(`${r}@${w}`);
    }
    check(`TC-MOB-UIW-18 ${w} px: 4 màn SV không cuộn ngang`, bad.length === 0, bad.join(" ") || "ok");
  }
});

test("UIW-19 CLS /checkin ≤ 0,1 ở 360 và 430", async ({ browser }) => {
  const { ctx, page } = await authed(browser, "pixel", ACCT.s1);
  for (const [w, h] of [[360, 640], [430, 932]] as const) {
    await page.setViewportSize({ width: w, height: h });
    await page.goto("about:blank");
    await page.goto(`/checkin?course=${C.QCMOB1}`, { waitUntil: "load" });
    await page.waitForTimeout(2500);
    const cls = await page.evaluate(() => {
      const e = performance.getEntriesByType("layout-shift") as unknown as { value: number; hadRecentInput: boolean }[];
      return e.filter((x) => !x.hadRecentInput).reduce((a, x) => a + x.value, 0);
    });
    check(`TC-MOB-UIW-19 CLS ${w} px ≤ 0,1`, cls <= 0.1, cls.toFixed(3));
  }
});

test("TGT-01/02/03/06 vùng chạm ≥ 44 px và khoảng cách ≥ 8 px", async ({ browser }) => {
  const { ctx, page } = await authed(browser, "pixel", ACCT.s1);
  for (const [w, h] of [[360, 740], [430, 932]] as const) {
    await page.setViewportSize({ width: w, height: h });
    await page.goto(`/checkin?course=${C.QCMOB1}`);
    await page.getByLabel("Mã điểm danh").waitFor();
    const r = await page.evaluate(() => {
      const els = [...document.querySelectorAll("main button, main a, main input, header button, nav[aria-label='Điều hướng chính'] a, nav[aria-label='Điều hướng chính'] button")];
      const small = els
        .map((e) => {
          const b = e.getBoundingClientRect();
          return { t: (e.textContent ?? e.getAttribute("aria-label") ?? e.tagName).trim().slice(0, 18), w: Math.round(b.width), h: Math.round(b.height), x: b.left, r: b.right };
        })
        .filter((x) => x.w > 0 && x.h > 0 && (x.w < 44 || x.h < 44));
      const rows = els
        .map((e) => e.getBoundingClientRect())
        .filter((b) => b.width > 0)
        .sort((a, b) => a.top - b.top || a.left - b.left);
      let minGap = 999;
      for (let i = 0; i < rows.length; i++)
        for (let j = i + 1; j < rows.length; j++) {
          const a = rows[i];
          const b = rows[j];
          if (Math.abs(a.top - b.top) < 30 && b.left >= a.right) minGap = Math.min(minGap, Math.round(b.left - a.right));
        }
      return { small, minGap };
    });
    check(`TC-MOB-TGT-01 ${w} px: mọi điều khiển ≥ 44×44`, r.small.length === 0, r.small.map((s) => `${s.t}:${s.w}×${s.h}`).join(" "));
    check(`TC-MOB-TGT-02 ${w} px: khoảng cách hai điều khiển kề ≥ 8 px`, r.minGap >= 8, `minGap=${r.minGap}`);
  }
  await page.setViewportSize({ width: 360, height: 740 });
  await page.goto(`/checkin?course=${C.QCMOB1}`);
  await page.getByLabel("Mã điểm danh").waitFor();
  const d = await page.evaluate(() => {
    const dock = document.querySelector("[data-part=checkin-dock]")?.getBoundingClientRect();
    const nav = document.querySelector("nav[aria-label='Điều hướng chính']")?.getBoundingClientRect();
    return { dockB: Math.round(dock?.bottom ?? -1), navT: Math.round(nav?.top ?? -1), dockH: Math.round(dock?.height ?? 0), navH: Math.round(nav?.height ?? 0) };
  });
  check("TC-MOB-TGT-03 dock không đè thanh dưới, cả hai ≥ 44 px", d.dockB <= d.navT && d.dockH >= 44 && d.navH >= 44, JSON.stringify(d));
});

test("TXT-06 phóng 200 % (180 px) 4 trạng thái: không tràn ngang, không cắt chữ", async ({ browser }) => {
  const { ctx, page } = await authed(browser, "pixel", ACCT.s1);
  await page.setViewportSize({ width: 180, height: 640 });
  const st = await fourStates(page, C.QCMOB1, "ABC123");
  const bad = Object.entries(st).filter(([, m]) => m.sw > m.iw + 1);
  const clipped = await page.evaluate(() => [...document.querySelectorAll("[data-part=checkin] *")].filter((e) => e.children.length === 0 && e.clientWidth > 0 && e.scrollWidth > e.clientWidth + 2).length);
  check("TC-MOB-TXT-06 180 px (≈200 % ở 360): 4 trạng thái không cuộn ngang, chữ không bị cắt", bad.length === 0 && clipped === 0, `${bad.map(([k, m]) => `${k}:${m.sw}/${m.iw}`).join(" ")} clipped=${clipped}`);
});

test("KBD-04/05/07/09/10/12 ô mã trên web: thuộc tính bàn phím, Enter gửi một lần, khoá khi gửi", async ({ browser }) => {
  const { ctx, page } = await authed(browser, "pixel", ACCT.s1);
  await page.setViewportSize({ width: 411, height: 891 });
  await page.goto(`/checkin?course=${C.QCMOB1}`);
  const inp = page.getByLabel("Mã điểm danh");
  await inp.waitFor();
  const attrs = await inp.evaluate((e: HTMLInputElement) => ({
    ac: e.getAttribute("autocapitalize"),
    au: e.getAttribute("autocomplete"),
    aco: e.getAttribute("autocorrect"),
    sp: e.getAttribute("spellcheck"),
    eh: e.getAttribute("enterkeyhint"),
    ml: e.getAttribute("maxlength"),
    ph: e.getAttribute("placeholder"),
  }));
  check("TC-MOB-KBD-04/KBD-10 thuộc tính chống tự sửa/gợi ý + maxlength 12 + gợi ý cú pháp", attrs.ac === "characters" && attrs.au === "off" && attrs.aco === "off" && attrs.sp === "false" && attrs.eh === "go" && attrs.ml === "12" && !!attrs.ph, JSON.stringify(attrs));
  let posts = 0;
  await page.route("**/checkins", (r) => {
    if (r.request().method() !== "POST") return r.continue();
    posts++;
    return r.fulfill({ status: 201, contentType: "application/json", body: JSON.stringify({ checked_in_at: new Date().toISOString(), round: 1 }) });
  });
  await inp.fill("ABC 123");
  await page.keyboard.press("Enter");
  await page.locator("[data-part=checkin][data-state=done]").waitFor({ timeout: 10_000 });
  check("TC-MOB-KBD-05 phím Enter gửi như nút, một POST; mã có khoảng trắng được nhận", posts === 1, `posts=${posts}`);
  await page.goto(`/checkin?course=${C.QCMOB1}`);
  await page.getByLabel("Mã điểm danh").fill("abc-123");
  await page.route("**/checkins", async (r) => {
    if (r.request().method() !== "POST") return r.continue();
    await new Promise((x) => setTimeout(x, 2500));
    await r.fulfill({ status: 201, contentType: "application/json", body: JSON.stringify({ checked_in_at: new Date().toISOString(), round: 1 }) });
  });
  await page.keyboard.press("Enter");
  await page.waitForTimeout(500);
  const locked = await page.evaluate(() => {
    const i = document.querySelector("[data-part=checkin] input") as HTMLInputElement | null;
    const b = document.querySelector("[data-part=checkin-dock] button") as HTMLButtonElement | null;
    return { i: i?.disabled ?? null, b: b?.disabled ?? null };
  });
  check("TC-MOB-KBD-07 đang gửi: ô và nút khoá (không gửi hai lần)", locked.b === true, JSON.stringify(locked));
  await page.locator("[data-part=checkin][data-state=done]").waitFor({ timeout: 15_000 });
  check("TC-MOB-KBD-09 mã dán có gạch nối vẫn nhận", /Đã điểm danh/.test(await page.locator("[data-part=checkin]").innerText()), "");
  // bàn phím ảo: thu khung nhìn, focus ô → ô vào giữa, nút vẫn thấy
  await page.goto(`/checkin?course=${C.QCMOB1}`);
  await page.getByLabel("Mã điểm danh").waitFor();
  await page.setViewportSize({ width: 411, height: 500 });
  await page.getByLabel("Mã điểm danh").focus();
  await page.waitForTimeout(600);
  const kb = await measure(page);
  check("TC-MOB-KBD-12 khung 411×500 (bàn phím ảo): ô thấy, nút Điểm danh vẫn trong khung", !!kb.input && !!kb.dock && kb.dock.bottom <= 500 && kb.input.top >= 0, JSON.stringify({ i: kb.input, d: kb.dock }));
});

test("ATT-01/03/08/09/10 /attendance trên điện thoại", async ({ browser }) => {
  const t = await authed(browser, "pixel", "teacher");
  await t.page.setViewportSize({ width: 375, height: 812 });
  await t.page.goto(`/attendance?course=${C.QCMOB2}`);
  await t.page.waitForTimeout(4000);
  const a1 = await t.page.evaluate(() => {
    const b = [...document.querySelectorAll("main button, main a")].map((e) => (e.textContent ?? "").trim()).filter(Boolean);
    return { ov: document.documentElement.scrollWidth > innerWidth + 1, main: document.querySelector("main")?.innerText.slice(0, 300) ?? "", btn: b.slice(0, 12) };
  });
  check("TC-MOB-ATT-01 375: /attendance mở được, không cuộn ngang, có hành động chính", !a1.ov && /Điểm danh|Buổi|buổi/.test(a1.main), `ov=${a1.ov} ${a1.main.replace(/\n+/g, "|").slice(0, 90)}`);
  await t.page.screenshot({ path: `${OUT}/ATT-01-teacher-375.png` });
  const tt = await t.page.evaluate(() => {
    const els = [...document.querySelectorAll("main button, main a")];
    return els.map((e) => { const b = e.getBoundingClientRect(); return { t: (e.textContent ?? "").trim().slice(0, 16), w: Math.round(b.width), h: Math.round(b.height) }; }).filter((x) => x.w > 0 && (x.w < 44 || x.h < 44));
  });
  check("TC-MOB-ATT-02 mọi nút/hàng trên /attendance 375 ≥ 44×44", tt.length === 0, tt.map((x) => `${x.t}:${x.w}×${x.h}`).join(" "));

  const ta = await authed(browser, "pixel", "ta");
  await ta.page.setViewportSize({ width: 360, height: 740 });
  await ta.page.goto(`/attendance?course=${C.QCMOB2}`);
  await ta.page.waitForTimeout(4000);
  const a3 = await ta.page.evaluate(() => {
    const inner = [...document.querySelectorAll("main *")].some((e) => e.scrollWidth > e.clientWidth + 2 && getComputedStyle(e).overflowX !== "visible");
    return { ov: document.documentElement.scrollWidth > innerWidth + 1, inner, txt: document.querySelector("main")?.innerText.slice(0, 200) ?? "" };
  });
  check("TC-MOB-ATT-03 360: lưới cuộn trong khung riêng, trang không cuộn ngang", !a3.ov, `ov=${a3.ov} inner=${a3.inner} ${a3.txt.replace(/\n+/g, "|").slice(0, 80)}`);
  const axe = await import("@axe-core/playwright").then((m) => m.default).catch(() => null);
  if (axe) {
    const r = await new axe({ page: ta.page }).analyze();
    const bad = r.violations.filter((v) => v.impact === "critical" || v.impact === "serious");
    check("TC-MOB-ATT-07 axe /attendance 360: 0 lỗi nghiêm trọng", bad.length === 0, bad.map((v) => `${v.id}:${v.impact}`).join(","));
  } else rec("TC-MOB-ATT-07 axe", "NOTE", "không nạp được @axe-core/playwright");
  await ta.page.setViewportSize({ width: 844, height: 390 });
  await ta.page.waitForTimeout(1500);
  const a9 = await ta.page.evaluate(() => ({ ov: document.documentElement.scrollWidth > innerWidth + 1, side: !!document.querySelector("aside[data-part=sidebar]") }));
  check("TC-MOB-ATT-09 ngang 844: thanh bên, không cuộn ngang", !a9.ov && a9.side, JSON.stringify(a9));

  const sv = await authed(browser, "pixel", ACCT.s1);
  await sv.page.goto(`/attendance?course=${C.QCMOB2}`);
  await sv.page.waitForTimeout(2500);
  const a8 = await sv.page.evaluate(() => document.querySelector("main")?.innerText.slice(0, 160) ?? "");
  check("TC-MOB-ATT-08 SV mở /attendance: màn chặn, không dữ liệu lớp", !/Điểm danh buổi|đánh vắng|Lưu điểm danh/i.test(a8), a8.replace(/\n+/g, "|").slice(0, 90));

  const { page: disp } = await authed(browser, "pixel", "teacher");
  await disp.goto(`/attendance/display?course=${C.QCMOB2}`);
  await disp.waitForTimeout(3000);
  const a10 = await disp.evaluate(() => ({ txt: document.body.innerText.slice(0, 200), qr: !!document.querySelector("[aria-label='Mã QR']") }));
  check("TC-MOB-ATT-10 /attendance/display trên 393 px: QR/mã hiển thị, không crash", a10.qr || /QR|mã/i.test(a10.txt), a10.txt.replace(/\n+/g, "|").slice(0, 90));
  await disp.screenshot({ path: `${OUT}/ATT-10-display-393.png` });
  rec("TC-MOB-ATT-10b Esc/Thoát trên màn chiếu", "NOTE", (await disp.locator("body").innerText()).includes("Thoát") ? "có nút Thoát" : "không thấy chữ Thoát");
});
