// qc3 s5 (2026-11-21): PE09-11 (seed demo trên Chrome thật), PE09-25 (cổng UX 5 route), PE09-28 (quét innerText SV). ep-r https://localhost:1143, lớp 761987 (seed gốc).
const L = require('./ui_lib.cjs'); const { chromium, B, SH, rec } = L; const fs = require('fs');
const C = '01a1df27-62b5-7775-a346-5c602a43e2f9', EM = '01a1df28-122f-7580-b4c2-b7eb0c3506e9', EC = '01a1df28-123e-7506-81c1-e18193760173';
const TECH = /\b(sandbox|verdict|go-judge|provider|trace|RAG|PII|QUEUED|RUNNING|IE|MLE|TLE|RE)\b|answer_key|canary/;
const ONLY = process.env.ONLY || 'all'; const out = {};
const txt = (p) => p.locator('body').innerText();
async function open(p, path) { await p.goto(B + path, { waitUntil: 'domcontentloaded' }); await p.waitForTimeout(3500); await L.dismiss(p); }
(async () => { const br = await chromium.launch();
  const S = {}; // văn bản từng màn SV theo vai, cho PE09-28
  // ---- PE09-11: từng tài khoản
  for (const [u, tag] of [['sv.gioi', 'A'], ['sv.kha', 'B'], ['sv.nguyco', 'C'], ['sv.moi', 'D'], ['teacher', 'GV'], ['ta', 'TA']]) {
    const { ctx, p, errs } = await L.login(br, u, { width: 1440, height: 900 }); out[u] = {};
    for (const [nm, path] of [['hom-nay', '/?course=' + C], ['exams', '/exams?course=' + C], ['mcq', `/exams/${EM}/take?course=${C}`], ['mcqres', `/exams/${EM}/results?course=${C}`], ['code', `/exams/${EC}/take?course=${C}`]]) {
      await open(p, path); const t = await txt(p); out[u][nm] = t; await p.screenshot({ path: `${SH}/pe09-11-${tag}-${nm}.png`, fullPage: false });
      if (['A', 'B', 'C', 'D'].includes(tag)) (S[u] ||= {})[nm] = t;
    }
    await ctx.close(); }
  fs.writeFileSync(SH + '/../pe09-11-text.json', JSON.stringify(out, null, 1));
  const A = out['sv.gioi'], Cc = out['sv.nguyco'], D = out['sv.moi'], G = out['teacher'], T = out['ta'];
  const okA = /10,00/.test(A.mcq) && /Đáp án đúng/.test(A.mcq.replace(/\n/g, ' ') + A.mcq) ;
  rec('s5-11-A', okA, `A mcq: ${(A.mcq.match(/[^\n]*10,00[^\n]*/) || ['(không thấy 10,00)'])[0]}; có «Đáp án đúng»=${/Đáp án đúng/.test(A.mcq)}`);
  rec('s5-11-C', /Bạn không làm bài này/.test(Cc.mcq), `C mcq: ${(Cc.mcq.match(/[^\n]*(không làm bài)[^\n]*/) || ['(không thấy)'])[0]}`);
  rec('s5-11-D', !/Kiểm tra tuần 9/.test(D.exams) && !/Kiểm tra tuần 9/.test(D['hom-nay']), `D exams: ${D.exams.replace(/\n/g, ' | ').slice(0, 160)}`);
  rec('s5-11-GV', /Nghi giống nhau/.test(G.mcqres) || /Giống nhau/.test(G.mcqres), `GV results: ${G.mcqres.replace(/\n/g, ' | ').slice(0, 260)}`);
  rec('s5-11-TA', !/Tín hiệu|Công bố điểm|Nghi giống nhau/.test(T.mcqres), `TA results có Tín hiệu/Công bố/Nghi giống nhau=${/Tín hiệu|Công bố điểm|Nghi giống nhau/.test(T.mcqres)}; ${T.mcqres.replace(/\n/g, ' | ').slice(0, 200)}`);
  // ---- PE09-28: quét SV
  const bad = []; for (const [u, o] of Object.entries(S)) for (const [nm, t] of Object.entries(o)) { const m = t.match(new RegExp(TECH.source, 'g')); if (m) bad.push(`${u}/${nm}: ${[...new Set(m)].join(',')}`); }
  rec('s5-28', bad.length === 0, `quét innerText ${Object.keys(S).length} SV × 5 màn (Hôm nay, /exams, làm bài/kết quả MCQ, bài code): từ kỹ thuật = ${bad.length ? bad.join('; ') : 'không có'}`);
  await br.close(); fs.writeFileSync(SH + '/../pe09-11-rec.json', JSON.stringify(L.out, null, 1)); })();
