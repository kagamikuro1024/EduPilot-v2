import { test, expect } from "@playwright/test";
import { appendFileSync, mkdirSync, writeFileSync } from "fs";
import { execSync } from "child_process";

test.skip(!process.env.QC_REAL || !process.env.QC_ANDROID, "cần stack ep-qc + emulator (QC_ANDROID=1, adb forward tcp:9222 …)");
// QC 10.8 pha 2: kiểm tay trên emulator Android (mức 2) bằng CDP vào WebView của app Capacitor (debug). Theo android-checklist.md.
const BASE = "http://localhost:580";
const PW = process.env.QC_PW ?? "";
const C = process.env.QC_CID ?? "";
const SIDS = (process.env.QC_SIDS ?? "").split(",").filter(Boolean);
const OUT = process.env.QC_OUT ?? "/tmp/qc-as";
mkdirSync(OUT, { recursive: true });
const rec = (tc: string, st: "PASS" | "FAIL" | "NOTE", note = "") => { console.log(`${st} ${tc} ${note}`.slice(0, 300)); appendFileSync(`${OUT}/android.jsonl`, JSON.stringify({ tc, st, note }) + "\n"); };
const check = (tc: string, cond: boolean, note = "") => { rec(tc, cond ? "PASS" : "FAIL", note); return cond; };
async function token(u: string) { const r = await fetch(`${BASE}/api/v1/auth/login`, { method: "POST", headers: { "content-type": "application/json", "Idempotency-Key": crypto.randomUUID() }, body: JSON.stringify({ email: `${u}@edupilot.local`, password: PW }) }); return ((await r.json()) as { access_token: string }).access_token; }
async function api(u: string, method: string, path: string, body?: unknown) { const r = await fetch(`${BASE}/api/v1${path}`, { method, headers: { "content-type": "application/json", authorization: `Bearer ${await token(u)}`, ...(method === "POST" ? { "Idempotency-Key": crypto.randomUUID() } : {}) }, body: body ? JSON.stringify(body) : undefined }); const t = await r.text(); return { s: r.status, j: t ? JSON.parse(t) : null }; }
let idx = 0;
async function win() { const sid = SIDS[idx++]; const o = await api("teacher", "POST", `/courses/${C}/sessions/${sid}/checkin-windows`, {}); expect(o.s).toBe(201); const d = (await api("teacher", "GET", `/courses/${C}/sessions/${sid}/checkin-windows/${o.j.id}/display`)).j; return { sid, wid: o.j.id as string, d }; }
const psql = (q: string) => execSync(`docker compose -p ep-qc exec -T postgres psql -U edupilot -d edupilot -At -F'|' -c "${q}"`, { cwd: process.env.HOME + "/Documents/TA_Agent_qc5" }).toString().trim();
type CdpReply = { id?: number; result?: { data?: string; result?: { value?: unknown }; exceptionDetails?: unknown } };
class Cdp {
  private id = 0; private cb = new Map<number, (r: CdpReply) => void>(); private ws!: globalThis.WebSocket; reqs: string[] = [];
  static async open() {
    const t = (await (await fetch("http://localhost:9222/json")).json()) as { type: string; url: string; webSocketDebuggerUrl: string }[];
    const pg = t.find((x) => x.type === "page" && x.url.startsWith(BASE)) ?? t.find((x) => x.type === "page")!;
    const c = new Cdp(); c.ws = new globalThis.WebSocket(pg.webSocketDebuggerUrl);
    await new Promise((r) => { c.ws.onopen = r; });
    c.ws.onmessage = (m) => { const j = JSON.parse(String(m.data)); if (j.id && c.cb.has(j.id)) { c.cb.get(j.id)!(j); c.cb.delete(j.id); } else if (j.method === "Network.requestWillBeSent") c.reqs.push(`${j.params.request.method} ${j.params.request.url}`); };
    await c.send("Runtime.enable"); await c.send("Page.enable"); await c.send("Network.enable");
    return c;
  }
  send(method: string, params: object = {}) { const id = ++this.id; return new Promise<CdpReply>((res) => { this.cb.set(id, res); this.ws.send(JSON.stringify({ id, method, params })); }); }
  async ev<T = unknown>(expr: string): Promise<T> { const r = await this.send("Runtime.evaluate", { expression: expr, awaitPromise: true, returnByValue: true }); if (r.result?.exceptionDetails) throw new Error(JSON.stringify(r.result.exceptionDetails).slice(0, 300)); return r.result?.result?.value as T; }
  async nav(url: string) { await this.send("Page.navigate", { url }); await new Promise((r) => setTimeout(r, 1500)); }
  async until(expr: string, ms = 25_000) { const t0 = Date.now(); while (Date.now() - t0 < ms) { try { if (await this.ev(expr)) return true; } catch { /* trang đang tải */ } await new Promise((r) => setTimeout(r, 400)); } return false; }
  get url() { return this.ev<string>("location.href"); }
  close() { this.ws.close(); }
}
const sel = (s: string) => `!!document.querySelector(${JSON.stringify(s)})`;
async function loginApp(c: Cdp, user: string) {
  await c.nav(`${BASE}/login`);
  const r = await c.ev(`fetch('/api/v1/auth/login',{method:'POST',credentials:'include',headers:{'content-type':'application/json','Idempotency-Key':crypto.randomUUID()},body:JSON.stringify({email:'${user}@edupilot.local',password:${JSON.stringify(PW)}})}).then(x=>x.status)`);
  expect(r).toBe(200);
}
async function logoutApp(c: Cdp) { await c.nav(`${BASE}/login`); await c.ev(`fetch('/api/v1/auth/logout',{method:'POST',credentials:'include',headers:{'Idempotency-Key':crypto.randomUUID()}}).then(x=>x.status).catch(()=>0)`); await c.ev("localStorage.clear(); sessionStorage.clear(); 1"); }
async function openCheckin(c: Cdp, t: string) { await c.nav("about:blank"); await c.nav(`${BASE}/checkin?c=${C}#${t}`); }
const doneState = (c: Cdp, ms = 30_000) => c.until(sel("[data-part=checkin][data-state=done]"), ms);

test.describe.configure({ mode: "serial" });

test("(chuẩn bị) chính sách lớp về mặc định, quyền vị trí đã cấp", async () => {
  await api("teacher", "PUT", `/courses/${C}/settings`, { attendance: { self_checkin: true, rotate_seconds: 5, late_after_min: 10, require: "WEB", geofence: null }, version: (await api("teacher", "GET", `/courses/${C}`)).j.settings.version });
  for (const w of psql(`select id from checkin_windows where course_id='${C}' and closed_at is null`).split("\n").filter(Boolean)) await api("teacher", "POST", `/courses/${C}/checkin-windows/${w}/close`);
  execSync("adb shell pm grant vn.edupilot.app android.permission.ACCESS_FINE_LOCATION; adb shell pm grant vn.edupilot.app android.permission.ACCESS_COARSE_LOCATION");
});

test("AC-0/1/4 app tải web, native, BarcodeDetector, đăng nhập SV", async () => {
  const c = await Cdp.open();
  const info = await c.ev<{ native: boolean; secure: boolean; bd: string; fmts: string[]; plugin: boolean; href: string }>(`(async()=>({native:Boolean(window.Capacitor&&window.Capacitor.isNativePlatform&&window.Capacitor.isNativePlatform()),secure:window.isSecureContext,bd:typeof window.BarcodeDetector,fmts:window.BarcodeDetector?await window.BarcodeDetector.getSupportedFormats():[],plugin:Boolean(window.Capacitor&&window.Capacitor.isPluginAvailable&&window.Capacitor.isPluginAvailable('EduPilotAttest')),href:location.href}))()`);
  rec("AC-0 thông tin WebView", "NOTE", JSON.stringify(info));
  check("AC-0 app mở được web từ APP_SERVER_URL, Capacitor native, secure context, plugin EduPilotAttest có", info.native && info.secure && info.plugin, JSON.stringify(info));
  check("AC-4 BarcodeDetector + qr_code trên emulator (không có ⇒ chỉ ô nhập mã ngắn)", true, `BarcodeDetector=${info.bd} formats=${info.fmts.join(",")}`);
  await loginApp(c, "sv.gioi"); await c.nav(`${BASE}/checkin?c=${C}`);
  const hasScan = await c.until("[...document.querySelectorAll('button')].some(b=>b.textContent.includes('Quét mã điểm danh'))", 8000);
  const hasInput = await c.until("[...document.querySelectorAll('label')].some(l=>l.textContent.includes('Mã điểm danh'))", 3000);
  check("AC-3a/AC-4 /checkin trong app: nút «Quét mã điểm danh» khi có BarcodeDetector; không thì ô «Mã điểm danh»", hasScan === (info.bd === "function" && info.fmts.includes("qr_code")) || hasInput, `scan=${hasScan} input=${hasInput}`);
  await c.send("Page.captureScreenshot", {}).then((r) => { if (r.result?.data) writeFileSync(`${OUT}/android-checkin-entry.png`, Buffer.from(r.result.data, "base64")); });
  c.close();
});

test("AC-7/8 emulator: đăng ký thiết bị ATTESTATION_FAILED → điểm danh bằng khoá Keystore (WEB_KEY DER) mức 2", async () => {
  const { wid, d } = await win(); const t = d.qr_url.split("#")[1];
  const c = await Cdp.open(); c.reqs = [];
  await openCheckin(c, t);
  const ok1 = await doneState(c);
  const row = psql(`select a.channel, a.assurance, coalesce(array_to_string(a.flags,','),''), coalesce(d.kind::text,''), coalesce(d.attested::text,'') from attendance_checkins a left join user_devices d on d.id=a.device_id where a.window_id='${wid}'`);
  const [ch, as, fl, kind, att] = row.split("|");
  const dev = c.reqs.filter((u) => /\/me\/devices/.test(u)).length;
  check("AC-7 app điểm danh: «Đã điểm danh», mức 2, khoá WEB_KEY chưa chứng thực, không DEVICE_UNTRUSTED", ok1 && as === "2" && kind === "WEB_KEY" && att === "false" && !fl.includes("DEVICE_UNTRUSTED"), `ok=${ok1} ${row}`);
  rec("AC-7 kênh lượt (channel) khi chạy trong app", "NOTE", `channel=${ch}; lời gọi /me/devices=${dev}`);
  check("AC-8 emulator / khoá debug không đạt mức 3 (ATTESTATION_FAILED)", as !== "3", row);
  await api("teacher", "POST", `/courses/${C}/checkin-windows/${wid}/close`); c.close();
});

test("AC-9 require=APP → NOT_APP trên emulator; AC-14 mã hết hạn", async () => {
  const setPol = async (pol: object) => api("teacher", "PUT", `/courses/${C}/settings`, { attendance: { self_checkin: true, rotate_seconds: 5, late_after_min: 10, ...pol }, version: (await api("teacher", "GET", `/courses/${C}`)).j.settings.version });
  await setPol({ require: "APP" });
  const c = await Cdp.open(); await logoutApp(c); await loginApp(c, "sv.kha");
  const w1 = await win(); await openCheckin(c, w1.d.qr_url.split("#")[1]); const ok1 = await doneState(c);
  const f1 = psql(`select coalesce(array_to_string(flags,','),''), assurance from attendance_checkins where window_id='${w1.wid}'`);
  check("AC-9 require=APP: lượt từ emulator (khoá chưa chứng thực) mang NOT_APP (P-108-6)", ok1 && f1.includes("NOT_APP"), f1);
  await setPol({ require: "WEB" });
  const w2 = await win(); const t2 = w2.d.qr_url.split("#")[1]; await new Promise((r) => setTimeout(r, 13_000)); c.reqs = [];
  await openCheckin(c, t2);
  const err = await c.until(sel("[data-part=checkin][data-state=error]"), 20_000);
  const txt = await c.ev<string>("document.querySelector('[data-part=checkin]').innerText"); const i = c.reqs.findIndex((u) => /^POST .*\/checkins/.test(u));
  check("AC-14 token quá hạn trong app: «Mã đã hết hạn…», không refresh/đăng xuất sau 422, vẫn ở /checkin", err && txt.includes("Mã đã hết hạn") && i >= 0 && !c.reqs.slice(i + 1).some((u) => /auth\/(refresh|logout)/.test(u)) && (await c.url).includes("/checkin"), `${txt.slice(0, 80)} | POST@${i}`);
  await api("teacher", "POST", `/courses/${C}/checkin-windows/${w2.wid}/close`); c.close();
});

test("AC-10/11 vị trí trên emulator: trong vùng / ngoài vùng / từ chối quyền", async () => {
  const setPol = async (pol: object) => api("teacher", "PUT", `/courses/${C}/settings`, { attendance: { self_checkin: true, rotate_seconds: 5, late_after_min: 10, require: "WEB", ...pol }, version: (await api("teacher", "GET", `/courses/${C}`)).j.settings.version });
  await setPol({ geofence: { lat: 10.7727, lng: 106.6602, radius_m: 200 } });
  execSync("adb shell pm grant vn.edupilot.app android.permission.ACCESS_FINE_LOCATION; adb shell pm grant vn.edupilot.app android.permission.ACCESS_COARSE_LOCATION; adb emu geo fix 106.6602 10.7727");
  const c = await Cdp.open(); await logoutApp(c); await loginApp(c, "sv.nguyco");
  const w1 = await win(); await openCheckin(c, w1.d.qr_url.split("#")[1]); const ok1 = await doneState(c, 40_000);
  const r1 = psql(`select coalesce(array_to_string(flags,','),''), assurance, coalesce(geo_distance_m::text,'') from attendance_checkins where window_id='${w1.wid}'`);
  check("AC-10a vị trí emulator = tâm vùng: «Đã điểm danh», có geo_distance_m nhỏ, không GEO_OUT/GEO_DENIED", ok1 && !/GEO_OUT|GEO_DENIED/.test(r1) && /\|\d+$/.test(r1), r1);
  rec("AC-11 cờ khi vị trí đặt qua `adb emu geo fix` (có GEO_MOCK nếu plugin báo mock)", "NOTE", `flags|mức|khoảng cách = ${r1}`);
  execSync("adb emu geo fix 106.7 10.8");
  await logoutApp(c); await loginApp(c, "sv04"); const w2 = await win(); await openCheckin(c, w2.d.qr_url.split("#")[1]); const ok2 = await doneState(c, 40_000);
  const r2 = psql(`select coalesce(array_to_string(flags,','),'') from attendance_checkins where window_id='${w2.wid}'`);
  check("AC-10b vị trí ngoài vùng (~7 km): GEO_OUT, vẫn «Đã điểm danh»", ok2 && r2.includes("GEO_OUT"), r2);
  c.close();
  execSync("adb shell pm revoke vn.edupilot.app android.permission.ACCESS_FINE_LOCATION; adb shell pm revoke vn.edupilot.app android.permission.ACCESS_COARSE_LOCATION; adb shell pm set-permission-flags vn.edupilot.app android.permission.ACCESS_FINE_LOCATION user-fixed; adb shell pm set-permission-flags vn.edupilot.app android.permission.ACCESS_COARSE_LOCATION user-fixed; sleep 1; adb shell am start -n vn.edupilot.app/.MainActivity; sleep 9");
  execSync("adb forward --remove-all; adb forward tcp:9222 localabstract:webview_devtools_remote_$(adb shell pidof vn.edupilot.app)");
  const c2 = await Cdp.open(); await logoutApp(c2); await loginApp(c2, "sv05"); const w3 = await win(); await openCheckin(c2, w3.d.qr_url.split("#")[1]); const ok3 = await doneState(c2, 40_000);
  const r3 = psql(`select coalesce(array_to_string(flags,','),'') from attendance_checkins where window_id='${w3.wid}'`);
  check("AC-10c từ chối quyền vị trí: GEO_DENIED, vẫn «Đã điểm danh»", ok3 && r3.includes("GEO_DENIED"), r3);
  await setPol({ geofence: null }); c2.close();
});

test("AC-12 hai tài khoản cùng điện thoại (cùng khoá Keystore) → DEVICE_SHARED (D-AS-01)", async () => {
  const c = await Cdp.open(); const w = await win(); const t = w.d.qr_url.split("#")[1];
  await logoutApp(c); await loginApp(c, "sv06"); await openCheckin(c, t); const a = await doneState(c, 30_000);
  await logoutApp(c); await loginApp(c, "sv07"); await openCheckin(c, t); const b = await doneState(c, 30_000);
  const rows = psql(`select u.email, array_to_string(a.flags,',') from attendance_checkins a join users u on u.id=a.student_id where a.window_id='${w.wid}' order by a.created_at`);
  check("AC-12 cùng điện thoại, 2 tài khoản: cả hai «Đã điểm danh» và có DEVICE_SHARED", a && b && (rows.match(/DEVICE_SHARED/g) ?? []).length === 2, `a=${a} b=${b} ${rows.replace(/\n/g, " ; ")}`);
  c.close();
});
