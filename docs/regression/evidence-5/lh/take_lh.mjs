// TC-PE09-27 / TC-PE05-48 (phần Lighthouse): /exams/<id>/take trên stack ep-r thật (https://localhost:1143), mobile, simulate 4x (mặc định).
// 1) navigation N lần (màn bắt đầu, SV r5s0X chưa bắt đầu bài) → LCP / TBT / CLS / JS;  2) user-flow: timespan = bấm Bắt đầu, chọn đáp án, qua câu → INP + CLS.
import fs from 'node:fs'; import {execSync} from 'node:child_process';
const lighthouse = (await import('/Users/kuro/checkouts/pq/frontend/node_modules/.pnpm/lighthouse@12.6.1_bare-buffer@3.7.2_supports-color@7.2.0/node_modules/lighthouse/core/index.js')); const {startFlow} = lighthouse; const puppeteer = (await import('/Users/kuro/checkouts/pq/frontend/node_modules/.pnpm/puppeteer-core@24.43.1_bare-buffer@3.7.2_supports-color@7.2.0/node_modules/puppeteer-core/lib/esm/puppeteer/puppeteer-core.js')).default;
const E = JSON.parse(fs.readFileSync('/tmp/qc2-c5/lhexam.json')); const B = 'https://localhost:1143'; const PW = 'Edupilot#Seed-2026';
const CH = process.env.CHROME_PATH; const OUT = new URL('./take/', import.meta.url).pathname; fs.mkdirSync(OUT, {recursive: true});
const clear = () => { try { execSync("docker exec ep-r-redis-1 sh -c \"redis-cli --scan --pattern 'ep:rl:auth*' | xargs -r redis-cli del\"", {stdio: 'ignore', env: {...process.env, DOCKER_HOST: 'unix://' + process.env.HOME + '/.colima/default/docker.sock'}}); } catch (e) {} };
async function session(user) {
  clear(); const br = await puppeteer.launch({executablePath: CH, headless: 'new', args: ['--no-sandbox', '--ignore-certificate-errors']}); const page = await br.newPage();
  await page.setViewport({width: 412, height: 823, deviceScaleFactor: 1.75, isMobile: true, hasTouch: true});
  await page.goto(B + '/login'); await page.type('input[type=email], input[name=email]', user + '@edupilot.local'); await page.type('input[type=password]', PW);
  await Promise.all([page.waitForFunction(() => !location.pathname.startsWith('/login'), {timeout: 20000}), page.keyboard.press('Enter')]); await new Promise(r => setTimeout(r, 2000));
  for (let i = 0; i < 3; i++) { const d = await page.$('dialog[open] button:last-of-type'); if (!d) break; await d.click().catch(() => {}); await new Promise(r => setTimeout(r, 700)); }
  return {br, page};
}
const url = B + '/exams/' + E.eid + '/take';
const pick = (lhr) => { const a = lhr.audits; const sc = a['network-requests'].details.items.filter(x => x.resourceType === 'Script'); return {lcp: a['largest-contentful-paint']?.numericValue, tbt: a['total-blocking-time']?.numericValue, cls: a['cumulative-layout-shift']?.numericValue, inp: a['interaction-to-next-paint']?.numericValue, jsTransfer: sc.reduce((s, x) => s + x.transferSize, 0), jsResourceBytes: sc.reduce((s, x) => s + x.resourceSize, 0), bi: lhr.environment.benchmarkIndex, mult: lhr.configSettings.throttling.cpuSlowdownMultiplier, lcpEl: a['largest-contentful-paint-element']?.details?.items?.[0]?.items?.[0]?.node?.snippet?.slice(0, 90)}; };
const mode = process.argv[2]; const res = [];
if (mode === 'nav') {
  for (let i = 0; i < Number(process.argv[3] || 3); i++) {
    const {br, page} = await session('r5s0' + (3 + i)); const flow = await startFlow(page, {config: {extends: 'lighthouse:default', settings: {onlyCategories: ['performance'], formFactor: 'mobile', screenEmulation: {mobile: true, width: 412, height: 823, deviceScaleFactor: 1.75, disabled: false}, ...(process.env.MULT ? {throttling: {cpuSlowdownMultiplier: Number(process.env.MULT)}} : {}), ...(process.env.THROT ? {throttlingMethod: process.env.THROT} : {})}}});
    await flow.navigate(url, {name: 'take-' + i}); const r = await flow.createFlowResult(); const x = pick(r.steps[0].lhr); res.push(x); console.log(JSON.stringify(x)); fs.writeFileSync(OUT + 'nav' + (process.env.THROT ? 'd' : process.env.MULT ? 'm' : '') + i + '.json', JSON.stringify(r.steps[0].lhr)); await br.close();
  }
}
if (mode === 'flow') {
  const {br, page} = await session('r5s0' + (process.argv[3] || 8)); await page.goto(url, {waitUntil: 'networkidle2'});
  const flow = await startFlow(page, {config: {extends: 'lighthouse:default', settings: {onlyCategories: ['performance'], formFactor: 'mobile', screenEmulation: {mobile: true, width: 412, height: 823, deviceScaleFactor: 1.75, disabled: false}}}});
  await flow.startTimespan({name: 'làm bài'});
  const click = async (sel) => { const h = await page.$(sel); if (h) { await h.click(); await new Promise(r => setTimeout(r, 1200)); return true; } return false; };
  const byText = async (re) => { const els = await page.$$('button, [role=radio], [role=checkbox], label'); for (const e of els) { const t = await e.evaluate(n => (n.innerText || '').trim()); if (re.test(t)) { await e.click(); await new Promise(r => setTimeout(r, 1200)); return t; } } return null; };
  console.log('start', await byText(/^Bắt đầu làm bài$/));
  for (let i = 0; i < 4; i++) { console.log('opt', await byText(/Đáp án \d của câu/)); console.log('next', await byText(/^Câu sau$/)); }
  await page.screenshot({path: OUT + 'flow.png'});
  await flow.endTimespan(); const r = await flow.createFlowResult(); const x = pick(r.steps[0].lhr); console.log(JSON.stringify(x)); fs.writeFileSync(OUT + 'flow.json', JSON.stringify(r.steps[0].lhr)); await br.close();
}
