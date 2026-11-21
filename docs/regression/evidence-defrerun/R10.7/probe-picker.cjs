const { chromium } = require("@playwright/test"); (async()=>{
const b = await chromium.launch(); const ctx = await b.newContext({ ignoreHTTPSError: true, ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 }, locale: "vi-VN" });
const page = await ctx.newPage();
await page.goto("https://localhost:1143/login");
await page.locator("input[type=email]").fill("teacher@edupilot.local"); await page.locator("input[type=password]").fill("Edupilot#Seed-2026");
await page.keyboard.press("Enter"); await page.waitForURL((u) => !u.pathname.startsWith("/login"));
const tb = page.locator("[data-part=topbar]"); await tb.waitFor();
console.log("topbar:", (await tb.innerText()).replace(/\s+/g, " "));
await tb.locator("button").filter({ hasText: /\d{3,}|KC106|QC/ }).first().click();
await page.waitForTimeout(1500);
console.log("QCEMPTYr107 count:", await page.getByText("QCEMPTYr107").count());
console.log("menuitems:", await page.locator("[role=menuitem]").count(), await page.locator("[role=menuitem]").allInnerTexts().then(a=>a.slice(0,5)));
await page.screenshot({ path: "../docs/regression/evidence-defrerun/R10.7/picker.png" });
await b.close();
})();
