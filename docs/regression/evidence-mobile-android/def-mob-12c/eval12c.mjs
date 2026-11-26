import { execSync } from "node:child_process";
const ADB = "/opt/homebrew/share/android-commandlinetools/platform-tools/adb";
const adb = (c) => execSync(`${ADB} ${c}`, { encoding: "utf8" }).trim();
const [pkg, expr] = process.argv.slice(2);
adb(`forward tcp:9224 localabstract:webview_devtools_remote_${adb(`shell pidof ${pkg}`)}`);
const t = (await (await fetch("http://localhost:9224/json")).json()).find((x) => x.type === "page");
const ws = new WebSocket(t.webSocketDebuggerUrl); await new Promise((r) => (ws.onopen = r));
ws.onmessage = (m) => { const d = JSON.parse(m.data); if (d.id === 1) { console.log(JSON.stringify(d.result?.result?.value ?? d.result).slice(0, 1500)); ws.close(); } };
ws.send(JSON.stringify({ id: 1, method: "Runtime.evaluate", params: { expression: expr, returnByValue: true } }));
