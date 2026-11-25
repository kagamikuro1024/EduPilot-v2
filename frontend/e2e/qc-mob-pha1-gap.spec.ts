// QC 10.8 pha 1 (cặp 3): 6 mã trước để N vì THIẾU DỮ KIỆN — SES-07, SES-14, ERR-09, MSG-06, TGT-02, MSG-15.
import { test } from "@playwright/test";
import { BASE, C, DEV, api, check, closeWindow, gotoEntry, login, openWindow, pwFor, rows, token } from "./qc-mob-lib";

const OUT = process.env.QC_OUT ?? "/tmp/qc-mob";
const dock = (p: import("@playwright/test").Page) => p.locator("[data-part=checkin-dock] button");
const body = (p: import("@playwright/test").Page) => p.locator("[data-part=checkin]");
const state = (p: import("@playwright/test").Page) => body(p).getAttribute("data-state");
const USER2 = "sv.qc-b";
const own = (b: import("@playwright/test").Browser) =>
  b.newContext({ ...DEV.pixel, baseURL: BASE, locale: "vi-VN", timezoneId: "Asia/Ho_Chi_Minh", ignoreHTTPSErrors: true } as never);

async function logout(p: import("@playwright/test").Page) {
  await p.getByRole("button").filter({ hasText: "Tài khoản" }).first().click({ timeout: 15_000 });
  await p.getByRole("menuitem", { name: "Đăng xuất" }).click({ timeout: 10_000 });
  await p.waitForURL((u: URL) => u.pathname.startsWith("/login"), { timeout: 25_000 });
}
// Khoá thiết bị web nằm trong IndexedDB `edupilot.device.v1` (deviceKey.ts), KHÔNG phải localStorage.
const dev = (p: import("@playwright/test").Page) =>
  p.evaluate(async () => {
    const open = () => new Promise<IDBDatabase>((res, rej) => { const r = indexedDB.open("edupilot.device.v1", 1); r.onsuccess = () => res(r.result); r.onerror = () => rej(r.error); });
    try {
      const db = await open();
      const all = await new Promise<Record<string, unknown>[]>((res, rej) => { const r = db.transaction("keys", "readonly").objectStore("keys").getAll(); r.onsuccess = () => res(r.result as Record<string, unknown>[]); r.onerror = () => rej(r.error); });
      db.close();
      return all.map((x) => ({ id: String(x.id), pub: String(x.publicKey).length }));
    } catch { return [] as { id: string; pub: number }[]; }
  });

/** Tài khoản mới (sv.qc-b) mở hộp thoại giới thiệu «Thời gian học trên EduPilot» — đóng trước khi thao tác. */
async function clearWelcome(p: import("@playwright/test").Page) {
  const d = p.locator("dialog[open]");
  for (let i = 0; i < 3; i++) {
    if ((await d.count()) === 0) return;
    const btn = d.locator("button").first();
    if ((await btn.count()) === 0) return;
    await btn.click({ timeout: 5000 }).catch(() => undefined);
    await p.waitForTimeout(400);
  }
}

/** Lấy mã ngắn MỚI NHẤT (xoay 10 s) rồi gửi; chờ nút bật. */
async function submitFresh(p: import("@playwright/test").Page, course: string, w: { sid: string; wid: string; code: string }) {
  await clearWelcome(p);
  await p.waitForTimeout(600);
  await clearWelcome(p);
  const d = await api("teacher", "GET", `/courses/${course}/sessions/${w.sid}/checkin-windows/${w.wid}/display`);
  const code = String(d.j?.short_code ?? w.code).replace(/\s/g, "");
  // Gõ lại tới khi ô giữ đúng mã: router.replace của deep link có thể xoá chữ ngay sau khi gõ.
  for (let i = 0; i < 10; i++) {
    await p.getByLabel("Mã điểm danh").fill(code);
    const v = await p.getByLabel("Mã điểm danh").inputValue();
    if (v === code) break;
    await p.waitForTimeout(400);
  }
  const b = p.locator("[data-part=checkin-dock] button");
  for (let i = 0; i < 20; i++) {
    if (await b.isEnabled()) break;
    await p.waitForTimeout(300);
  }
  if (!(await b.isEnabled())) {
    const dbg = await p.evaluate(() => ({
      v: (document.querySelector("[data-part=checkin] input") as HTMLInputElement | null)?.value ?? null,
      st: document.querySelector("[data-part=checkin]")?.getAttribute("data-state") ?? null,
      t: (document.querySelector("[data-part=checkin]") as HTMLElement | null)?.innerText.replace(/\n+/g, "|").slice(0, 160) ?? null,
    }));
    console.log("DBG disabled", JSON.stringify({ code, ...dbg }));
  }
  await b.click();
  return code;
}

async function loginRetry(p: import("@playwright/test").Page, user: string) {
  for (let i = 0; i < 8; i++) {
    try {
      await login(p, user);
      return;
    } catch (e) {
      if (i === 7) throw e;
      console.log("RETRY login", user, (i + 1) * 30, "s");
      await p.waitForTimeout(30_000); // 429 hạn mức đăng nhập theo IP
      await p.goto("/login").catch(() => undefined);
    }
  }
}

test("SES-07/14 hai SV cùng một trình duyệt: khoá thiết bị thuộc trình duyệt, cả hai ghi DEVICE_SHARED", async ({ browser }) => {
  const ctx = await own(browser);
  let page = await ctx.newPage();
  test.setTimeout(420_000);
  const w = await openWindow(C.QCMOB1, 30);
  await loginRetry(page, "sv.qc");
  await gotoEntry(page, C.QCMOB1);
  await clearWelcome(page);
  await submitFresh(page, C.QCMOB1, w);
  await body(page).and(page.locator("[data-state=done]")).waitFor({ timeout: 25_000 });
  const dev1 = await dev(page);
  await logout(page);
  const devAfter = await dev(page);
  check("TC-MOB-SES-14a đăng xuất KHÔNG xoá khoá thiết bị (IndexedDB edupilot.device.v1)", dev1.length > 0 && JSON.stringify(devAfter) === JSON.stringify(dev1), `keys=${JSON.stringify(dev1)} after=${JSON.stringify(devAfter)}`);

  // Tab mới trong CÙNG context (IndexedDB = cùng khoá thiết bị) — tránh app state cũ sau đăng xuất.
  await page.close();
  page = await ctx.newPage();
  await page.goto("/login");
  await page.locator("input[type=email]").fill(`${USER2}@edupilot.local`);
  await page.locator("input[type=password]").fill(pwFor(USER2));
  for (let i = 0; i < 8; i++) {
    await page.keyboard.press("Enter");
    try {
      await page.waitForURL((u: URL) => !u.pathname.startsWith("/login"), { timeout: 30_000 });
      break;
    } catch (e) {
      if (i === 7) throw e;
      console.log("RETRY login", USER2, (i + 1) * 30, "s");
      await page.waitForTimeout(30_000);
      await page.goto("/login");
      await page.locator("input[type=email]").fill(`${USER2}@edupilot.local`);
      await page.locator("input[type=password]").fill(pwFor(USER2));
    }
  }
  await gotoEntry(page, C.QCMOB1);
  await clearWelcome(page);
  await submitFresh(page, C.QCMOB1, w);
  await body(page).and(page.locator("[data-state=done]")).waitFor({ timeout: 25_000 });
  await page.screenshot({ path: `${OUT}/SES-07-sv2-done.png` });

  const raw = await api("teacher", "GET", `/courses/${C.QCMOB1}/sessions/${w.sid}/attendance`);
  const flat = JSON.stringify(raw.j ?? {});
  const shared = (flat.match(/DEVICE_SHARED/g) ?? []).length;
  check("TC-MOB-SES-07/14b hai lượt cùng cửa sổ đều có cờ DEVICE_SHARED (API GV)", shared >= 2, `shared=${shared} j=${flat.slice(0, 260)}`);
  await closeWindow(C.QCMOB1, w.wid).catch(() => undefined);
  await ctx.close();
  console.log("DBHINT", JSON.stringify({ sid: w.sid, wid: w.wid }));
});

test("ERR-09 + MSG-06 sáu mã sai liên tiếp → «Thử lại sau {n} giây.» đếm ngược, mã đúng vẫn 429", async ({ browser }) => {
  test.setTimeout(600_000);
  const ctx = await own(browser);
  const page = await ctx.newPage();
  const w = await openWindow(C.QCMOB1);
  const st: number[] = [];
  page.on("response", (r) => {
    if (r.request().method() === "POST" && /\/checkins$/.test(new URL(r.url()).pathname)) st.push(r.status());
  });
  await loginRetry(page, USER2);
  await gotoEntry(page, C.QCMOB1);
  await clearWelcome(page);
  for (let i = 1; i <= 6; i++) {
    await page.getByLabel("Mã điểm danh").fill(`AAAAA${i}`);
    await page.keyboard.press("Enter");
    await page.waitForTimeout(1200);
  }
  await page.getByText(/Thử lại sau \d+ giây\./).first().waitFor({ timeout: 20_000 }).catch(() => undefined);
  const t6 = await body(page).innerText();
  const reason = await page.evaluate(() => {
    const b = document.querySelector("[data-part=checkin-dock] button") as HTMLButtonElement | null;
    return { dis: b?.disabled ?? null, al: b?.getAttribute("aria-description") ?? b?.getAttribute("title") ?? null, txt: b?.textContent ?? null };
  });
  const cd = /Thử lại sau (\d+) giây\./.exec(t6);
  check("TC-MOB-ERR-09a lần sai thứ 6: «Thử lại sau {n} giây.» (n≥1), nút Điểm danh khoá kèm lý do", !!cd && Number(cd[1]) >= 1 && reason.dis === true && new RegExp(`Thử lại sau \\d+ giây`).test(`${reason.al ?? ""}${reason.txt ?? ""}`), `n=${cd?.[1]} dis=${reason.dis} al=${reason.al} posts=${st.join(",")} text=${t6.replace(/\n+/g, "|").slice(0, 110)}`);
  await page.screenshot({ path: `${OUT}/ERR-09-429.png` });

  const n0 = Number(cd?.[1] ?? 0);
  await page.waitForTimeout(1500);
  const t7 = await body(page).innerText();
  const n1 = Number(/Thử lại sau (\d+) giây\./.exec(t7)?.[1] ?? -1);
  check("TC-MOB-ERR-09b đếm ngược giảm theo giây", n1 >= 0 && n1 < n0, `${n0} → ${n1}`);

  // mã đúng trong lúc 429 vẫn bị 429 (gõ thẳng vào ô nếu ô còn cho nhập)
  const canType = (await page.getByLabel("Mã điểm danh").count()) > 0;
  let during = "";
  if (canType) {
    await page.getByLabel("Mã điểm danh").fill(w.code);
    await page.keyboard.press("Enter");
    await page.waitForTimeout(1500);
    during = (await body(page).innerText()).replace(/\n+/g, "|");
  }
  check("TC-MOB-ERR-09c mã đúng trong lúc 429 vẫn 429 (không ghi lượt)", canType && /Thử lại sau \d+ giây\./.test(during) && (await rows(C.QCMOB1, w.sid)).length === 0, `type=${canType} during=${during.slice(0, 110)} posts=${st.join(",")}`);

  // hết giờ → quay về ô nhập
  const wait = (n1 > 0 ? n1 : 45) + 6;
  await page.waitForTimeout(Math.min(wait, 90) * 1000);
  const back = await page.evaluate(() => !!document.querySelector("[data-part=checkin] input") && !document.querySelector("[data-part=checkin][data-state=limited]"));
  check("TC-MOB-ERR-09d hết giờ đếm ngược quay về ô nhập", back, `wait=${Math.min(wait, 75)}s t=${(await body(page).innerText()).replace(/\n+/g, "|").slice(0, 90)}`);
  await closeWindow(C.QCMOB1, w.wid).catch(() => undefined);
  await ctx.close();
});

test("TGT-02 360 px: khoảng cách kề giữa hai vùng chạm LÁ ≥ 8 px", async ({ browser }) => {
  test.setTimeout(300_000);
  const ctx = await own(browser);
  const page = await ctx.newPage();
  await page.setViewportSize({ width: 360, height: 780 });
  // PM: chạy lại ở 360 và 375 sau bản sửa DEF-MOB-06
  await loginRetry(page, "teacher");
  const measure = (name: string) =>
    page.evaluate(() => {
      const sel = "a[href],button,input,[role=button],[role=link]";
      const all = [...document.querySelectorAll(sel)] as HTMLElement[];
      const leaf = all.filter((e) => !e.querySelector(sel) && e.getBoundingClientRect().width > 0 && e.getBoundingClientRect().height > 0);
      const rects = leaf.map((e) => e.getBoundingClientRect());
      const label = (e: HTMLElement) => (e.getAttribute("aria-label") ?? e.textContent ?? e.tagName).trim().slice(0, 18) || e.tagName;
      let min = Infinity;
      const hist: Record<number, number> = {};
      const sample: string[] = [];
      for (let i = 0; i < rects.length; i++)
        for (let j = i + 1; j < rects.length; j++) {
          const a = rects[i], b = rects[j];
          const dx = Math.max(a.left - b.right, b.left - a.right);
          const dy = Math.max(a.top - b.bottom, b.top - a.bottom);
          const gap = Math.round(Math.max(dx, dy));
          if (gap <= 0) continue;
          if (gap < min) min = gap;
          hist[gap] = (hist[gap] ?? 0) + 1;
          if (gap < 8 && sample.length < 4)
            sample.push(`${gap}px <${leaf[i].tagName} ${leaf[i].className.split("_").pop()}>[${label(leaf[i])}] ↔ <${leaf[j].tagName} ${leaf[j].className.split("_").pop()}>[${label(leaf[j])}]`);
        }
      const badN = Object.entries(hist).filter(([k]) => Number(k) < 8).reduce((n, [, c]) => n + c, 0);
      return { n: leaf.length, min: Math.round(min * 10) / 10, badN, sample };
    });
  for (const w of [360, 375]) {
    await page.setViewportSize({ width: w, height: 812 });
    for (const [name, url] of [["/checkin", `/checkin?course=${C.QCMOB1}`], ["/attendance", "/attendance"]] as const) {
      await page.goto(url);
      await page.waitForTimeout(4000);
      const r = await measure(name);
      check(`TC-MOB-TGT-02 ${name} ${w}: khoảng cách kề ≥ 8 px`, r.min >= 8, `n=${r.n} min=${r.min} <8=${r.badN} ${r.sample.join(" ; ")}`.slice(0, 280));
      await page.screenshot({ path: `${OUT}/TGT-02${name.replace("/", "-")}-${w}.png` });
    }
  }
  await ctx.close();
});

test("MSG-15 quét chữ 6 trạng thái /checkin: không chuỗi cấm", async ({ browser }) => {
  const ctx = await own(browser);
  const page = await ctx.newPage();
  test.setTimeout(420_000);
  const w = await openWindow(C.QCMOB1);
  await loginRetry(page, "sv.qc");
  await gotoEntry(page, C.QCMOB1);
  const bad = /CHECKIN_|NotAllowedError|NotFoundError|NotReadableError|ERR_|net::|undefined|null|\[object|Error|Failed|Exception|\\$/;
  const grab = async (label: string) => {
    const t = await page.evaluate(() => {
      const root = document.querySelector("[data-part=checkin]") ?? document.body;
      const txt = (root as HTMLElement).innerText;
      const aria = [...root.querySelectorAll("[aria-label],[aria-description],img[alt]")]
        .map((e) => [e.getAttribute("aria-label"), e.getAttribute("aria-description"), e.getAttribute("alt")].filter(Boolean).join(" "))
        .join(" | ");
      const http = /\b(?:4\d\d|5\d\d)\b/.test(txt + " " + aria);
      return { txt, aria, http };
    });
    const hits = `${t.txt} ${t.aria}`.match(new RegExp(bad.source, "g")) ?? [];
    const ok = hits.length === 0 && !t.http;
    check(`TC-MOB-MSG-15 ${label}: không chuỗi cấm, không mã HTTP`, ok, `hits=${JSON.stringify(hits.slice(0, 5))} http=${t.http} t=${t.txt.replace(/\n+/g, "|").slice(0, 100)}`);
  };

  await grab("entry");
  const back = async () => {
    await page.unrouteAll({ behavior: "ignoreErrors" });
    await page.waitForTimeout(1500);
    await page.goto("/");
    await gotoEntry(page, C.QCMOB1);
  };
  // gửi chậm (route giữ 12 s)
  await page.route("**/checkins", async (r) => {
    if (r.request().method() !== "POST") return r.continue();
    await new Promise((x) => setTimeout(x, 12_000));
    await r.abort(); // không ghi lượt: đóng yêu cầu sau khi đã đo trạng thái «đang gửi»
  });
  await page.getByLabel("Mã điểm danh").fill(w.code);
  await page.keyboard.press("Enter");
  await page.waitForTimeout(2000);
  await grab("sending");
  await back();
  // 429 (route)
  await page.route("**/checkins", (r) => (r.request().method() === "POST" ? r.fulfill({ status: 429, headers: { "retry-after": "12" }, contentType: "application/json", body: '{"code":"RATE_LIMITED"}' }) : r.continue()));
  await page.getByLabel("Mã điểm danh").fill(w.code);
  await page.keyboard.press("Enter");
  await page.waitForTimeout(1500);
  await grab("limited");
  await back();
  // 500 (route)
  await page.route("**/checkins", (r) => (r.request().method() === "POST" ? r.fulfill({ status: 500, contentType: "application/json", body: '{"code":"INTERNAL"}' }) : r.continue()));
  await page.getByLabel("Mã điểm danh").fill(w.code);
  await page.keyboard.press("Enter");
  await page.waitForTimeout(1500);
  await grab("error");
  await back();
  // mất mạng
  const cdp = await page.context().newCDPSession(page);
  await cdp.send("Network.enable");
  await cdp.send("Network.emulateNetworkConditions", { offline: true, latency: 0, downloadThroughput: -1, uploadThroughput: -1 });
  await page.getByLabel("Mã điểm danh").fill(w.code);
  await page.keyboard.press("Enter");
  await page.waitForTimeout(2500);
  await grab("offline");
  await cdp.send("Network.emulateNetworkConditions", { offline: false, latency: 0, downloadThroughput: -1, uploadThroughput: -1 });
  await back();
  await page.waitForTimeout(1000);
  // thật — mã ngắn xoay 10 s, lấy mã mới nhất ngay trước khi gửi
  const disp = await api("teacher", "GET", `/courses/${C.QCMOB1}/sessions/${w.sid}/checkin-windows/${w.wid}/display`);
  const fresh = String(disp.j?.short_code ?? w.code).replace(/\s/g, "");
  await page.getByLabel("Mã điểm danh").fill(fresh);
  await page.keyboard.press("Enter");
  await body(page).and(page.locator("[data-state=done]")).waitFor({ timeout: 25_000 });
  await grab("done");
  await page.screenshot({ path: `${OUT}/MSG-15-done.png` });
  await closeWindow(C.QCMOB1, w.wid).catch(() => undefined);
  await ctx.close();
});
