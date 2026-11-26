// Tải lại một URL N lần trong WebView của app, đếm: #418, ReferenceError/TypeError/SyntaxError, request /api/v1, form đăng nhập vẽ ra. Dùng: node cdp12c2.mjs <package> <url> <N> [chiTomTat]
import { execSync } from "node:child_process";
const ADB = "/opt/homebrew/share/android-commandlinetools/platform-tools/adb";
const adb = (c) => execSync(`${ADB} ${c}`, { encoding: "utf8" }).trim();
const [pkg, url, loops = "1"] = process.argv.slice(2);
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
try { adb("forward --remove tcp:9224"); } catch {}
adb(`forward tcp:9224 localabstract:webview_devtools_remote_${adb(`shell pidof ${pkg}`)}`);
const t = (await (await fetch("http://localhost:9224/json")).json()).find((x) => x.type === "page");
const ws = new WebSocket(t.webSocketDebuggerUrl); await new Promise((r, j) => { ws.onopen = r; ws.onerror = j; });
let id = 0; const ev = []; const pend = new Map();
ws.onmessage = (m) => { const d = JSON.parse(m.data); if (d.id && pend.has(d.id)) { pend.get(d.id)(d); pend.delete(d.id); } else if (d.method) ev.push(d); };
const send = (method, params = {}) => new Promise((res) => { const i = ++id; pend.set(i, res); ws.send(JSON.stringify({ id: i, method, params })); });
const val = async (e) => (await send("Runtime.evaluate", { expression: e, returnByValue: true })).result?.result?.value;
for (const d of ["Runtime", "Log", "Page", "Network"]) await send(d + ".enable");
let tot = { n: 0, e418: 0, jserr: 0, nofetch: 0, noform: 0 };
for (let n = 1; n <= Number(loops); n++) {
  ev.length = 0;
  await send("Page.navigate", { url });
  await sleep(7000);
  const texts = [];
  for (const e of ev) {
    if (e.method === "Runtime.consoleAPICalled" && ["error", "assert"].includes(e.params.type)) texts.push(e.params.args.map((a) => a.description ?? a.value ?? "").join(" "));
    else if (e.method === "Runtime.exceptionThrown") texts.push(e.params.exceptionDetails.exception?.description ?? e.params.exceptionDetails.text);
  }
  const e418 = texts.filter((x) => /#418/.test(x)).length;
  const jserr = texts.filter((x) => /ReferenceError|TypeError|SyntaxError/.test(x));
  const api = ev.filter((e) => e.method === "Network.requestWillBeSent" && /\/api\/v1\//.test(e.params.request.url)).map((e) => e.params.request.url.split("/api/v1")[1].split("?")[0]);
  const state = await val("JSON.stringify({path:location.pathname,inputs:document.querySelectorAll('input').length,glob:typeof globalThis})");
  console.log(`lan ${n}: #418=${e418} js(Reference/Type/Syntax)=${jserr.length} api=[${[...new Set(api)].join(",")}] ${state}${jserr.length ? " ERR:" + jserr[0].slice(0, 200) : ""}`);
  tot.n++; tot.e418 += e418; tot.jserr += jserr.length; if (!api.length) tot.nofetch++;
}
console.log("TONG", JSON.stringify(tot));
ws.close();
