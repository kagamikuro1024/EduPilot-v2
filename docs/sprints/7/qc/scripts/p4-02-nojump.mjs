// QC US-P4-02 TC-27: danh sách chuông đang mở, thông báo mới chèn lên đầu → mục đang đọc không nhảy. Chạy từ frontend; cần /tmp/qc/h.sh (tok, newsess, ask, msgs) và C1.
import { createRequire } from "node:module"; import { execSync } from "node:child_process";
const { chromium } = createRequire(process.cwd() + "/")("@playwright/test");
const sh = (c) => execSync(c, { encoding: "utf8", shell: "/bin/bash" }).trim();
const GW = "https://localhost:543";
const T = sh(`. /tmp/qc/h.sh; echo "Authorization: Bearer $(tok sv27)"`); const S = sh(`. /tmp/qc/h.sh; T="${T}"; newsess "$T"`);
sh(`. /tmp/qc/h.sh; T="${T}"; ask "$T" ${S} "Tuần này em có lịch gì? QCJ$RANDOM" >/dev/null`);
const M = sh(`. /tmp/qc/h.sh; T="${T}"; msgs "$T" ${S} | jq -r '[.items[]|select(.role=="ASSISTANT")][0].id'`);
const b = await chromium.launch(); const ctx = await b.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1280, height: 900 } }); const p = await ctx.newPage();
await p.goto(`${GW}/login`); await p.fill("input:not([type=password])", "teacher@edupilot.local"); await p.fill("input[type=password]", "Edupilot#Seed-2026"); await p.click('button:has-text("Đăng nhập")'); await p.waitForTimeout(4000);
await p.getByRole("button", { name: /Thông báo/ }).first().click(); await p.waitForTimeout(800);
const item = p.locator('[role=dialog] *, [role=menu] *').filter({ hasText: /^QC thông báo 6$/ }).first();
const y0 = (await item.boundingBox()).y;
sh(`. /tmp/qc/h.sh; curl -sk -X POST $GW/api/v1/chat/messages/${M}/escalate -H "${T}" -o /dev/null -w '%{http_code}'`);
await p.waitForTimeout(3500);
const y1 = (await item.boundingBox()).y; const txt = await p.locator('[role=dialog],[role=menu]').first().innerText();
console.log(`${Math.abs(y1 - y0) < 2 ? "PASS" : "FAIL"} TC-P4-02-27 y trước=${y0.toFixed(1)} sau=${y1.toFixed(1)} Δ=${(y1 - y0).toFixed(1)}px; dòng gộp nằm ở: ${txt.indexOf("câu hỏi đang chờ") < txt.indexOf("QC thông báo 12") ? "đầu danh sách" : "ngoài đầu"}`);
await b.close();
