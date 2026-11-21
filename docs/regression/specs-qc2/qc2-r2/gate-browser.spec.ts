// QC2 22/11 — TC-GATE-25 (SSE cắt mạng 10 s, Last-Event-ID) và TC-GATE-27 (hai tab cùng Idempotency-Key) bằng Playwright Chromium thật.
// Cổng từ sse-browser-cut.mjs / idem-two-tabs.mjs (bản eval `browser` của máy này không có CDP target()). Chạy trên ep-r (gateway bản test có /_test/*), cùng origin qua Caddy.
import { test, expect } from "@playwright/test";
import { execFileSync, execSync } from "node:child_process";

const BASE = process.env.QC_BASE ?? "https://localhost:1143";
const sleep = (ms: number) => new Promise((s) => setTimeout(s, ms));
async function login(): Promise<string> {
  const r = await fetch(`${BASE}/api/v1/auth/login`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ email: "sv.kha@edupilot.local", password: "Edupilot#Seed-2026" }) });
  return ((await r.json()) as { access_token: string }).access_token;
}

test("TC-GATE-25 SSE: mất mạng 10 s, tự nối lại bằng Last-Event-ID, không mất / không trùng", async ({ page, context }) => {
  test.setTimeout(150_000);
  const token = await login();
  await page.goto(BASE + "/");
  const cdp = await context.newCDPSession(page);
  await cdp.send("Network.enable");
  const net = (offline: boolean) => cdp.send("Network.emulateNetworkConditions", { offline, latency: 0, downloadThroughput: -1, uploadThroughput: -1 });
  await page.evaluate(([base, tk]) => {
    const w = window as unknown as { __sse: Record<string, unknown> };
    const S = (w.__sse = { events: [] as { id: string; ev: string; d: { n?: number } | null }[], control: [] as { ev: string }[], status: [] as number[], errors: [] as string[], sentLastIds: [] as string[], lastId: null as string | null, reconnects: 0, stop: false, ctl: null as AbortController | null });
    const parse = (blk: string) => {
      let id: string | null = null, ev = "message"; const data: string[] = [];
      for (const ln of blk.split("\n")) { if (ln.startsWith(":")) continue; if (ln.startsWith("id: ")) id = ln.slice(4); else if (ln.startsWith("event: ")) ev = ln.slice(7); else if (ln.startsWith("data: ")) data.push(ln.slice(6)); }
      if (ev === "ready" || (ev === "message" && !data.length)) return;
      if (["reconnect", "shutdown", "resync"].includes(ev)) { S.control.push({ ev }); return; }
      if (id) { S.lastId = id; let d = null; try { d = JSON.parse(data.join("\n")); } catch { /* bỏ */ } S.events.push({ id, ev, d }); }
    };
    void (async () => {
      while (!S.stop) {
        const ctl = new AbortController(); S.ctl = ctl; const headers: Record<string, string> = { Authorization: "Bearer " + tk };
        if (S.lastId) { headers["Last-Event-ID"] = S.lastId; S.sentLastIds.push(S.lastId); }
        try {
          const r = await fetch(base + "/api/v1/events", { headers, signal: ctl.signal });
          S.status.push(r.status); if (!r.ok) throw new Error("HTTP " + r.status);
          const rd = r.body!.getReader(), dec = new TextDecoder(); let buf = "";
          for (;;) { const { value, done } = await rd.read(); if (done) break; buf += dec.decode(value, { stream: true }); let i; while ((i = buf.indexOf("\n\n")) >= 0) { const blk = buf.slice(0, i); buf = buf.slice(i + 2); parse(blk); } }
        } catch (e) { S.errors.push(String((e as Error)?.message ?? e)); }
        if (S.stop) break; S.reconnects++; await new Promise((r) => setTimeout(r, 3000));
      }
    })();
  }, [BASE, token] as const);
  for (let i = 0; i < 40 && !(await page.evaluate(() => (window as unknown as { __sse: { status: number[] } }).__sse.status.length)); i++) await sleep(250);
  const st0 = await page.evaluate(() => (window as unknown as { __sse: { status: number[] } }).__sse.status[0]);
  const pub = (n: number) => { try { return execFileSync("curl", ["-sk", "--max-time", "10", "-o", "/dev/null", "-w", "%{http_code}", "-X", "POST", "-H", "Authorization: Bearer " + token, "-H", "Content-Type: application/json", "-d", JSON.stringify({ type: "test.ping", data: { n } }), BASE + "/api/v1/_test/events"]).toString(); } catch { return "ERR"; } };
  const total = 30, cutAt = 8, restoreAt = 18; const codes: string[] = [];
  for (let n = 1; n <= total; n++) {
    if (n === cutAt) { await net(true); await page.evaluate(() => (window as unknown as { __sse: { ctl: AbortController | null } }).__sse.ctl?.abort()); }
    if (n === restoreAt) await net(false);
    codes.push(pub(n)); await sleep(1000);
  }
  await net(false); await sleep(7000);
  const S = await page.evaluate(() => { const s = (window as unknown as { __sse: Record<string, unknown> }).__sse; s.stop = true; return JSON.parse(JSON.stringify({ ...s, ctl: null })) as { events: { id: string; ev: string; d: { n?: number } | null }[]; control: { ev: string }[]; status: number[]; errors: string[]; sentLastIds: string[]; reconnects: number } });
  const ns = S.events.filter((e) => e.ev === "test.ping").map((e) => e.d?.n);
  const want = Array.from({ length: total }, (_, i) => i + 1);
  const before = S.events.filter((e) => e.d?.n === cutAt - 1 || e.d?.n === cutAt).map((e) => e.id);
  const rows = [
    ["K1 stream 200", st0 === 200, `status=${st0}`], ["K2 30 POST 2xx", codes.every((c) => /^2/.test(c)), codes.join(",")],
    ["K3 nhận đúng 1..30 thứ tự", JSON.stringify(ns) === JSON.stringify(want), ns.join(" ")], ["K4 không trùng", new Set(ns).size === ns.length, `trùng=${ns.length - new Set(ns).size}`],
    ["K5 ≥ 1 lần nối lại", S.reconnects >= 1, `reconnects=${S.reconnects}`], ["K6 Last-Event-ID = id n=7/8", S.sentLastIds.length >= 1 && before.includes(S.sentLastIds[0]), `gửi=${S.sentLastIds[0]} (n7/8 id=${before.join(",")})`],
    ["K7 ≥ 1 lỗi fetch khi offline", S.errors.length >= 1, S.errors.slice(0, 3).join(" | ")], ["K8 không resync", !S.control.some((c) => c.ev === "resync"), JSON.stringify(S.control)],
    ["K9 mọi lần nối lại HTTP 200", S.status.every((s) => s === 200), `status=${S.status.join(",")}`],
  ] as [string, boolean, string][];
  for (const [id, ok, d] of rows) console.log(`${ok ? "PASS" : "FAIL"} ${id} — ${d}`);
  console.log("=> ok=" + rows.every((r) => r[1]));
  expect(rows.filter((r) => !r[1]).map((r) => r[0])).toEqual([]);
});

test("TC-GATE-27 hai tab cùng Idempotency-Key: 1 bản ghi, 2 phản hồi giống nhau, Idempotent-Replayed đọc được", async ({ browser }) => {
  const token = await login();
  const key = "qc-two-tabs-" + Date.now(), name = "two-tabs-" + Date.now();
  const mk = async () => { const c = await browser.newContext({ ignoreHTTPSErrors: true }); const p = await c.newPage(); await p.goto(BASE + "/"); return p; };
  const [A, B] = [await mk(), await mk()];
  const send = (p: typeof A) => p.evaluate(async ([base, tk, k, nm]) => {
    const r = await fetch(base + "/api/v1/_test/items", { method: "POST", headers: { Authorization: "Bearer " + tk, "Content-Type": "application/json", "Idempotency-Key": k }, body: JSON.stringify({ name: nm }) });
    return { status: r.status, body: await r.text(), replayed: r.headers.get("idempotent-replayed"), ct: r.headers.get("content-type"), retry: r.headers.get("retry-after") };
  }, [BASE, token, key, name] as const);
  let [ra, rb] = await Promise.all([send(A), send(B)]);
  const first = `${ra.status}/${rb.status}`;
  if (ra.status === 409) { await sleep(((+(ra.retry ?? 1)) || 1) * 1000 + 200); ra = await send(A); }
  if (rb.status === 409) { await sleep(((+(rb.retry ?? 1)) || 1) * 1000 + 200); rb = await send(B); }
  const count = execSync(`docker exec ep-r-postgres-1 psql -U edupilot -d edupilot -At -c "select count(*) from _test_items where name='${name}'"`).toString().trim();
  const rows = [
    ["I1 đúng 1 bản ghi", count === "1", `count=${count}`], ["I2 hai status giống và 2xx", ra.status === rb.status && /^2/.test(String(ra.status)), `lần đầu ${first} · cuối ${ra.status}/${rb.status}`],
    ["I3 hai thân giống hệt", ra.body === rb.body && ra.body.length > 0, ra.body.slice(0, 80)], ["I4 Content-Type giống", ra.ct === rb.ct, `${ra.ct} / ${rb.ct}`],
    ["I5 đúng một Idempotent-Replayed: true đọc được từ trình duyệt", [ra.replayed, rb.replayed].filter((x) => x === "true").length === 1, `${ra.replayed} / ${rb.replayed}`],
  ] as [string, boolean, string][];
  for (const [id, ok, d] of rows) console.log(`${ok ? "PASS" : "FAIL"} ${id} — ${d}`);
  console.log("=> ok=" + rows.every((r) => r[1]));
  expect(rows.filter((r) => !r[1]).map((r) => r[0])).toEqual([]);
});
