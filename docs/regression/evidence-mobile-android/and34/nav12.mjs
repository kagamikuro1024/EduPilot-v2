import { execSync } from "node:child_process";
const ADB="/opt/homebrew/share/android-commandlinetools/platform-tools/adb";const a=c=>execSync(ADB+" "+c,{encoding:"utf8"}).trim();
try{a("forward --remove tcp:9224")}catch{}
a("forward tcp:9224 localabstract:webview_devtools_remote_"+a("shell pidof vn.edupilot.app"));
const t=(await (await fetch("http://localhost:9224/json")).json()).find(x=>x.type==="page");const ws=new WebSocket(t.webSocketDebuggerUrl);await new Promise(r=>ws.onopen=r);
let id=0;const p=new Map();ws.onmessage=m=>{const d=JSON.parse(m.data);if(p.has(d.id)){p.get(d.id)(d);p.delete(d.id)}};
const send=(method,params={})=>new Promise(r=>{const i=++id;p.set(i,r);ws.send(JSON.stringify({id:i,method,params}))});
const r=await send("Page.navigate",{url:process.argv[2]});console.log("navigate:",JSON.stringify(r.result).slice(0,160));
await new Promise(r=>setTimeout(r,4000));
const u=await send("Runtime.evaluate",{expression:"location.href+' | '+document.body.innerText.slice(0,160).replace(/\\n/g,' / ')",returnByValue:true});console.log("sau 4s:",u.result?.result?.value);ws.close();
