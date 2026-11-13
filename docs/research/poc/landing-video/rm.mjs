import { chromium } from "playwright-core";
const css = (guard) => `
@media (prefers-reduced-motion: reduce){*,*::before,*::after{animation-duration:1ms!important;transition-duration:1ms!important}}
body{margin:0} .sp{height:1500px} .box{height:200px;background:#c00}
${guard ? "@media (prefers-reduced-motion: no-preference){" : ""}
@supports (animation-timeline: view()){ .box{animation:rv linear both;animation-timeline:view();animation-range:entry 0% cover 40%} }
${guard ? "}" : ""}
@keyframes rv{from{opacity:0;transform:translateY(24px)}to{opacity:1;transform:none}}`;
const b = await chromium.launch();
for (const rm of ["no-preference", "reduce"]) for (const guard of [false, true]) {
  const p = await b.newPage({ viewport: { width: 1440, height: 900 }, reducedMotion: rm });
  await p.setContent(`<style>${css(guard)}</style><div class=sp></div><div class=box id=b></div><div class=sp></div>`);
  await p.evaluate(() => window.scrollTo(0, 1500 - 900 + 60)); await p.waitForTimeout(200);
  const o = await p.$eval("#b", e => getComputedStyle(e).opacity);
  console.log(`reducedMotion=${rm} guard=${guard} supports=${await p.evaluate(() => CSS.supports("animation-timeline: view()"))} opacity_at_entry_15pct=${o}`);
  await p.close();
}
console.log("chromium", b.version()); await b.close();
