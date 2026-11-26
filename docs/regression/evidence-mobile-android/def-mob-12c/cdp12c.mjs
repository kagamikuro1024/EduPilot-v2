// Bắt console/exception/log của trang bằng CDP (WebView của app hoặc webview_shell). Dùng: node cdp12c.mjs <package> <url> [lanTai]
import { execSync } from "node:child_process";
const ADB = "/opt/homebrew/share/android-commandlinetools/platform-tools/adb";
const adb = (c) => execSync(`${ADB} ${c}`, { encoding: "utf8" }).trim();
const [pkg, url, loops = "1"] = process.argv.slice(2);
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const pid = adb(`shell pidof ${pkg}`);
try { adb("forward --remove tcp:9224"); } catch {}
adb(`forward tcp:9224 localabstract:webview_devtools_remote_${pid}`);
const list = await (await fetch("http://localhost:9224/json")).json();
const t = list.find((x) => x.type === "page");
const ws = new WebSocket(t.webSocketDebuggerUrl);
await new Promise((r, j) => { ws.onopen = r; ws.onerror = j; });
let id = 0; const ev = [];
const pend = new Map();
ws.onmessage = (m) => { const d = JSON.parse(m.data); if (d.id && pend.has(d.id)) { pend.get(d.id)(d); pend.delete(d.id); } else if (d.method) ev.push(d); };
const send = (method, params = {}) => new Promise((res) => { const i = ++id; pend.set(i, res); ws.send(JSON.stringify({ id: i, method, params })); });
await send("Runtime.enable"); await send("Log.enable"); await send("Page.enable");
for (let n = 1; n <= Number(loops); n++) {
  ev.length = 0;
  await send("Page.navigate", { url });
  await sleep(9000);
  const out = [];
  for (const e of ev) {
    if (e.method === "Runtime.consoleAPICalled" && ["error", "warning", "assert"].includes(e.params.type)) {
      out.push({ k: "console." + e.params.type, text: e.params.args.map((a) => a.description ?? a.value ?? a.type).join(" ").slice(0, 1200), stack: (e.params.stackTrace?.callFrames ?? []).slice(0, 8).map((f) => `${f.functionName || "(anon)"} ${f.url.split("/").pop()}:${f.lineNumber}:${f.columnNumber}`) });
    } else if (e.method === "Runtime.exceptionThrown") {
      const x = e.params.exceptionDetails; out.push({ k: "exception", text: (x.exception?.description ?? x.text).slice(0, 1200) });
    } else if (e.method === "Log.entryAdded" && ["error", "warning"].includes(e.params.entry.level)) {
      out.push({ k: "log." + e.params.entry.level, text: e.params.entry.text.slice(0, 300), url: e.params.entry.url?.split("/").pop(), line: e.params.entry.lineNumber });
    }
  }
  const has418 = out.some((o) => /#418|error #418|\b418\b/.test(o.text));
  console.log(`=== lan tai ${n}: ${out.length} muc loi/canh bao; co #418: ${has418}`);
  for (const o of out) console.log(JSON.stringify(o));
}
ws.close();
