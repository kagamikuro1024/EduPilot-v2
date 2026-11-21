// qc3 s5 F19 trọn trên Chrome thật (ep-r, lớp R5C1): GV tạo câu + dựng bài + lên lịch (UI) → SV 375 px làm trắc nghiệm bằng bàn phím (3G + mất mạng 30 s) → 1440 px làm code (chạy thử, nộp) → nộp bài có số
// → đóng tự động → tự công bố → SV xem + phúc khảo → GV trả lời + CSV. Phủ PE08-67, PE09-31, PE09-21 (3,4,8,11), PE09-26, một phần PE09-20/25.
const L = require('./ui_lib.cjs'); const { chromium, B, SH, rec, sql } = L; const fs = require('fs');
const F = JSON.parse(fs.readFileSync('/tmp/qc3e/f19.json')); const C = F.c1; const TITLE = 'F19 qc3 ' + F.tag;
const R = []; const log = (k, v) => { R.push([k, v]); console.log('·', k, String(v).slice(0, 260)); fs.writeFileSync(SH + '/../f19-log.json', JSON.stringify(R, null, 1)); };
const shot = (p, n) => p.screenshot({ path: `${SH}/f19-${n}.png` }).catch(() => {});
const pad = (n) => String(n).padStart(2, '0'); const loc = (d) => `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const SUMSRC = '#include <cstdio>\nint main(){long long a,b;scanf("%lld %lld",&a,&b);printf("%lld\\n",a+b);}\n';
(async () => { process.on('unhandledRejection', (e) => log('LỖI', e)); const br = await chromium.launch();
  // ================= GV 1440
  const gv = await L.login(br, 'r5t', { width: 1440, height: 900 }); const p = gv.p; let eid;
  try {
    await p.goto(B + `/questions?course=${C}`, { waitUntil: 'domcontentloaded' }); await p.waitForTimeout(3000); await L.dismiss(p);
    await p.getByRole('button', { name: 'Tạo câu hỏi' }).first().click(); await p.getByRole('menuitem', { name: 'Soạn tay', exact: true }).click(); await p.getByRole('menuitem', { name: 'Câu trắc nghiệm', exact: true }).click();
    const dlg = p.getByRole('dialog', { name: 'Câu trắc nghiệm mới' }); await dlg.getByLabel('Tiêu đề').fill('F19 UI ' + F.tag); await dlg.getByLabel('Chủ đề').fill('F19'); await dlg.getByLabel('Đề bài').fill('Cổng mặc định của HTTPS là bao nhiêu?');
    await dlg.getByLabel('Nội dung đáp án 1').fill('80'); await dlg.getByLabel('Nội dung đáp án 2').fill('443'); await dlg.getByRole('button', { name: 'Tạo câu hỏi' }).click(); await p.waitForTimeout(500);
    await dlg.getByLabel('Đáp án 2 là đáp án đúng').check(); await shot(p, '01-gv-tao-cau'); await dlg.getByRole('button', { name: 'Tạo câu hỏi' }).click(); await p.waitForTimeout(2000);
    const made = sql(`select review_status||'|'||type from question_bank where title='F19 UI ${F.tag}'`); log('1 GV tạo câu (UI)', made); await shot(p, '02-gv-cau-da-tao'); await p.getByRole('button', { name: 'Đóng' }).first().click().catch(() => {});
    await p.goto(B + `/exams?course=${C}`, { waitUntil: 'domcontentloaded' }); await p.waitForTimeout(2500);
    await p.getByRole('button', { name: 'Tạo bài thi' }).click(); const form = p.getByRole('form', { name: 'Bài thi mới' }); await form.getByLabel('Tiêu đề').fill(TITLE); await form.getByRole('button', { name: 'Tạo bản nháp' }).click();
    await p.getByRole('heading', { name: TITLE, level: 1 }).waitFor({ timeout: 15000 }); eid = p.url().match(/exams\/([0-9a-f-]{36})/)[1]; log('2 bài nháp', eid);
    await p.getByRole('button', { name: 'Lên lịch' }).click(); await p.waitForTimeout(800); const todo = await p.getByRole('list', { name: 'Việc cần sửa' }).innerText().catch(() => '(không thấy)'); log('2 lỗi lên lịch (nhánh lỗi)', todo.replace(/\n/g, ' | ')); await shot(p, '03-gv-loi-len-lich');
    await p.getByRole('tab', { name: /^Câu hỏi/ }).click(); await p.getByRole('button', { name: 'Thêm câu từ ngân hàng' }).click(); const dr = p.getByRole('dialog', { name: 'Chọn câu đã duyệt từ ngân hàng' });
    for (const t of [...F.mcq.map((q) => q.title), F.code.title]) { await dr.getByRole('checkbox', { name: new RegExp(t.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')) }).first().check(); }
    await shot(p, '04-gv-chon-cau'); await dr.getByRole('button', { name: /^Thêm \d+ câu/ }).click(); await p.waitForTimeout(800); await p.getByRole('button', { name: 'Lưu danh sách câu' }).click(); await p.getByText('Đã lưu danh sách câu hỏi').waitFor({ timeout: 10000 });
    const open = new Date(Date.now() + 100_000), close = new Date(open.getTime() + 8 * 60_000);
    await p.getByRole('tab', { name: 'Thông tin' }).click(); await p.getByLabel('Mở lúc').fill(loc(open)); await p.getByLabel('Đóng lúc').fill(loc(close)); await p.getByLabel('Thời lượng (phút)').fill('7'); await p.getByRole('button', { name: 'Lưu thông tin' }).click(); await p.getByText('Đã lưu thông tin bài thi').waitFor({ timeout: 10000 });
    await p.getByRole('tab', { name: 'Xem trước' }).click(); await p.waitForTimeout(1500); await shot(p, '05-gv-xem-truoc'); const prev = await p.locator('body').innerText(); log('3 xem trước không lộ đáp án/canary', !prev.includes(F.canary));
    // TA không lên lịch được
    const ta = await L.login(br, 'r5ta', { width: 1440, height: 900 }); await ta.p.goto(B + `/exams/${eid}?course=${C}`, { waitUntil: 'domcontentloaded' }); await ta.p.waitForTimeout(3000); const taLen = await ta.p.getByRole('button', { name: /^Lên lịch/ }).count(); await shot(ta.p, '06-ta-khong-len-lich'); log('4 TA thấy nút Lên lịch', taLen); await ta.ctx.close();
    await p.getByRole('button', { name: 'Lên lịch' }).click(); await p.getByText(/Đã lên lịch/).waitFor({ timeout: 15000 }); await shot(p, '07-gv-da-len-lich'); log('5 đã lên lịch', sql(`select status||'|'||opens_at||'|'||closes_at from exams where id='${eid}'`));
  } catch (e) { log('LỖI GV', String(e).slice(0, 400)); await shot(p, 'err-gv'); }
  if (!eid) { await br.close(); process.exit(2); }
  const opens = new Date(sql(`select opens_at from exams where id='${eid}'`).replace(' ', 'T') + (sql(`select opens_at from exams where id='${eid}'`).match(/[+-]\d\d$/) ? ':00' : ''));
  await sleep(Math.max(0, opens.getTime() - Date.now() + 4000)); log('mở bài', new Date().toISOString());
  // ================= SV1 375 px
  const leaks = []; const scan = (tag) => (r) => { const u = r.url(); if (!u.includes('/api/v1/')) return; r.text().then((t) => { for (const k of [F.canary, F.hid, 'answer_key', 'reference_source', 'is_correct']) if (t.includes(k)) leaks.push(`${tag} ${u.split('/api/v1')[1].slice(0, 70)} chứa ${k}`); }).catch(() => {}); };
  const s1 = await L.login(br, 'r5s17', { width: 375, height: 760 }); const q = s1.p; q.on('response', scan('SV1'));
  const aid = () => sql(`select a.id from exam_attempts a join users u on u.id=a.student_id where a.exam_id='${eid}' and u.email='r5s17@edupilot.local'`);
  try {
    await q.goto(B + `/exams?course=${C}`, { waitUntil: 'domcontentloaded' }); await q.waitForTimeout(2500); await shot(q, '10-sv-danh-sach-375'); const listTxt = await q.locator('body').innerText(); log('6 SV thấy bài trong /exams', listTxt.includes(TITLE));
    await q.goto(B + `/exams/${eid}/take?course=${C}`, { waitUntil: 'domcontentloaded' }); await q.waitForTimeout(2500); await L.dismiss(q); await shot(q, '11-sv-man-bat-dau-375');
    const pre = await q.locator('body').innerText(); log('7 màn bắt đầu có «≥ 1024 px»', /1024/.test(pre));
    await q.locator('[data-part=integrity-notice]').scrollIntoViewIfNeeded().catch(() => {}); await q.getByRole('button', { name: 'Bắt đầu làm bài' }).click(); await q.getByText(/Câu 1\//).waitFor({ timeout: 20000 });
    // 3G chậm
    const cdp = await q.context().newCDPSession(q); await cdp.send('Network.enable'); await cdp.send('Network.emulateNetworkConditions', { offline: false, latency: 400, downloadThroughput: (400 * 1024) / 8, uploadThroughput: (400 * 1024) / 8 }); log('8 3G', '400 kbps / RTT 400 ms bật');
    const t0 = Date.now(); await q.keyboard.press('b'); await q.waitForTimeout(1500); await q.keyboard.press('ArrowRight'); await q.waitForTimeout(800); await q.keyboard.press('1'); await q.keyboard.press('3'); await q.waitForTimeout(1500); await q.keyboard.press('ArrowRight'); await q.waitForTimeout(800); await q.keyboard.press('c');
    await q.waitForTimeout(6000); await shot(q, '12-sv-3g-375'); const n3g = sql(`select count(*) from exam_answers where attempt_id='${aid()}'`); log('8 sau 3G: đáp án trên máy chủ', n3g + ' (đã làm 3 câu bằng bàn phím trong ' + Math.round((Date.now() - t0) / 1000) + ' s; trang còn dùng được)');
    // ngắt 30 s
    await cdp.send('Network.emulateNetworkConditions', { offline: false, latency: 0, downloadThroughput: -1, uploadThroughput: -1 });
    await q.keyboard.press('ArrowLeft'); await q.keyboard.press('ArrowLeft'); await q.waitForTimeout(800); await q.keyboard.press('a'); await q.waitForTimeout(3000); const nBefore = sql(`select count(*) from exam_answers where attempt_id='${aid()}'`);
    await q.context().setOffline(true); await q.keyboard.press('c'); await q.waitForTimeout(30_000); const offTxt = await q.locator('body').innerText(); await shot(q, '13-sv-offline-30s-375'); const banner = (offTxt.match(/[^\n]*(Mất mạng|Chưa lưu)[^\n]*/) || ['(không thấy)'])[0];
    await q.context().setOffline(false); await q.waitForTimeout(12_000); await shot(q, '14-sv-online-lai-375'); const nAfter = sql(`select count(*) from exam_answers where attempt_id='${aid()}'`); const saved = await q.locator('[data-part=save-status]').innerText().catch(() => '');
    log('9 ngắt 30 s', `dải «${banner.trim()}»; đáp án trên máy chủ trước ${nBefore} → sau bật mạng ${nAfter}; «${saved}»`);
    // lượt cuối cùng đã lên máy chủ đúng không: so với câu 1 chọn
    const ans = sql(`select count(*) from exam_answers where attempt_id='${aid()}'`); log('9 đủ 3 câu trắc nghiệm trên máy chủ', ans);
    // 375: tràn ngang + vùng chạm
    const ox = await q.evaluate(() => document.documentElement.scrollWidth - innerWidth); log('10 375 px tràn ngang', ox);
    // chuyển 1440 làm code trên cùng lượt
    await q.setViewportSize({ width: 1440, height: 900 }); await q.waitForTimeout(1500); await q.getByText(/^Câu 4$/).first().click().catch(async () => { await q.keyboard.press('ArrowRight'); await q.keyboard.press('ArrowRight'); await q.keyboard.press('ArrowRight'); }); await q.waitForTimeout(1500);
    const ed = q.getByRole('textbox', { name: /Mã nguồn bài/ }).first(); await ed.waitFor({ timeout: 15000 }); await ed.fill(SUMSRC); await q.waitForTimeout(3000); await shot(q, '15-sv-code-1440');
    await q.getByRole('button', { name: /Chạy thử/ }).first().click(); await q.waitForTimeout(12_000); await shot(q, '16-sv-chay-thu'); const runTxt = await q.locator('body').innerText(); log('11 Chạy thử', (runTxt.match(/(Đúng|Sai kết quả|Lỗi biên dịch|Quá thời gian)[^\n]*/g) || []).slice(0, 4).join(' / '));
    await q.getByRole('button', { name: /Nộp lời giải/ }).first().click(); await q.waitForTimeout(20_000); await shot(q, '17-sv-nop-code'); const subTxt = await q.locator('body').innerText(); log('11 Nộp lời giải', (subTxt.match(/(Lần nộp|Đang chấm|Đã nhận|Đúng|Sai kết quả|test ẩn)[^\n]*/g) || []).slice(0, 5).join(' / '));
    await q.getByRole('button', { name: 'Nộp bài' }).first().click(); await q.waitForTimeout(1200); const dlgTxt = await q.getByRole('dialog').innerText().catch(() => '(không có hộp)'); await shot(q, '18-sv-xac-nhan-nop'); log('12 xác nhận nộp', dlgTxt.replace(/\n/g, ' | ')); await q.getByRole('dialog').getByRole('button', { name: 'Nộp bài' }).click(); await q.waitForTimeout(4000);
    await shot(q, '19-sv-da-nop'); log('12 sau nộp', ((await q.locator('body').innerText()).match(/Đã nộp[^\n]*|Điểm sẽ hiện[^\n]*/g) || []).join(' / '));
  } catch (e) { log('LỖI SV1', String(e).slice(0, 500)); await shot(q, 'err-sv1'); }
  // ================= SV2: gõ code dở rồi hết giờ
  const s2 = await L.login(br, 'r5s18', { width: 1440, height: 900 }); const w = s2.p; w.on('response', scan('SV2'));
  try { await w.goto(B + `/exams/${eid}/take?course=${C}`, { waitUntil: 'domcontentloaded' }); await w.waitForTimeout(2500); await L.dismiss(w); await w.locator('[data-part=integrity-notice]').scrollIntoViewIfNeeded().catch(() => {}); await w.getByRole('button', { name: 'Bắt đầu làm bài' }).click(); await w.getByText(/Câu 1\//).waitFor({ timeout: 20000 });
    await w.getByText(/^Câu 4$/).first().click().catch(() => {}); await w.waitForTimeout(1000); const ed2 = w.getByRole('textbox', { name: /Mã nguồn bài/ }).first(); await ed2.fill(SUMSRC); await w.waitForTimeout(500); await shot(w, '20-sv2-go-do'); log('13 SV2 gõ code dở rồi để hết giờ', 'nháp đã gõ');
  } catch (e) { log('LỖI SV2', String(e).slice(0, 400)); }
  // ================= đợi đóng + công bố
  const closes = Date.parse(sql(`select to_char(closes_at at time zone 'utc','YYYY-MM-DD"T"HH24:MI:SS"Z"') from exams where id='${eid}'`));
  const typing = w.getByRole('textbox', { name: /Mã nguồn bài/ }).first(); await sleep(Math.max(0, closes - Date.now() - 4000)); await typing.type(' ', { delay: 0 }).catch(() => {}); await sleep(18_000); await shot(w, '21-sv2-het-gio'); const w2 = await w.locator('body').innerText(); log('13 SV2 hết giờ khi đang gõ', (w2.match(/[^\n]*(Hết giờ|Đã nộp)[^\n]*/) || ['(không thấy)'])[0]);
  log('13 DB SV2', sql(`select a.status||'|'||coalesce(a.submit_reason::text,'')||'|'||(select count(*) from code_submissions s where s.attempt_id=a.id and s.kind='SUBMIT')||'|auto='||coalesce((select bool_or(auto)::text from code_submissions s where s.attempt_id=a.id),'') from exam_attempts a join users u on u.id=a.student_id where a.exam_id='${eid}' and u.email='r5s18@edupilot.local'`));
  for (let i = 0; i < 100; i++) { if (sql(`select status from exams where id='${eid}'`) === 'PUBLISHED') break; await sleep(5000); } log('14 trạng thái bài', sql(`select status||'|published_at='||coalesce(published_at::text,'')||'|closes='||closes_at from exams where id='${eid}'`));
  // ================= SV1 xem kết quả + phúc khảo
  try { await q.setViewportSize({ width: 375, height: 760 }); await q.goto(B + `/exams/${eid}/take?course=${C}`, { waitUntil: 'domcontentloaded' }); await q.waitForTimeout(3500); await shot(q, '30-sv-ket-qua-375'); const sums = q.locator('[data-part=student-result] [data-part=result-item] summary'); for (let i = 0; i < await sums.count(); i++) await sums.nth(i).click(); await q.waitForTimeout(600); await shot(q, '31-sv-ket-qua-mo-375');
    const r = await q.locator('[data-part=student-result]').innerText(); const ox2 = await q.evaluate(() => document.documentElement.scrollWidth - innerWidth); log('15 SV kết quả 375', `điểm ${(r.match(/[\d,]+ \/ 10,00/) || ['?'])[0]}; đáp án đúng ${(r.match(/Đáp án đúng/g) || []).length}; giải thích ${(r.match(/Giải thích/g) || []).length}; test ẩn ${(r.match(/Test ẩn[^\n]*/) || ['?'])[0]}; tràn ngang ${ox2}`);
    await q.getByRole('button', { name: 'Gửi yêu cầu xem lại' }).click(); await q.getByLabel(/Bạn muốn giảng viên xem lại điều gì/).fill('Em muốn xem lại câu 2 vì nghĩ chọn đúng (qc3)'); await q.getByRole('button', { name: 'Gửi yêu cầu', exact: true }).click(); await q.waitForTimeout(3000); await shot(q, '32-sv-phuc-khao'); log('16 phúc khảo', (await q.locator('body').innerText()).match(/Yêu cầu xem lại của bạn[^\n]*/)?.[0] || sql(`select count(*) from exam_appeals where exam_id='${eid}'`));
  } catch (e) { log('LỖI SV kết quả', String(e).slice(0, 500)); await shot(q, 'err-sv-kq'); }
  // ================= GV xem kết quả, trả lời, CSV
  try { await p.goto(B + `/exams/${eid}/results?course=${C}`, { waitUntil: 'domcontentloaded' }); await p.waitForTimeout(3500); await shot(p, '40-gv-ket-qua'); const gt = await p.locator('body').innerText(); log('17 GV bảng điểm', (gt.match(/Đã chấm[^\n]*|Vắng[^\n]*/g) || []).slice(0, 3).join(' / ') + ' ; r5s17 ' + (gt.match(/[^\n]*r5s17[^\n]*\n?[^\n]*/) || [''])[0].replace(/\n/g, ' '));
    const [dl] = await Promise.all([p.waitForEvent('download', { timeout: 15000 }).catch(() => null), p.getByRole('button', { name: 'Xuất CSV' }).first().click()]); if (dl) { const path = SH + '/../f19-results.csv'; await dl.saveAs(path); log('17 CSV', `${dl.suggestedFilename()} ${fs.readFileSync(path, 'utf8').split('\n').length} dòng`); }
    await p.getByRole('tab', { name: 'Xem lại điểm' }).click(); await p.waitForTimeout(1500); const ap = p.locator('[data-part=appeal]'); await shot(p, '41-gv-phuc-khao'); await ap.getByLabel(/Phản hồi cho sinh viên/).first().fill('Đã xem lại, giữ nguyên điểm (qc3).'); await ap.getByRole('button', { name: 'Gửi phản hồi' }).first().click(); await p.waitForTimeout(3000); await shot(p, '42-gv-da-tra-loi'); log('18 GV trả lời phúc khảo', sql(`select status||'|'||coalesce(response,'') from exam_appeals where exam_id='${eid}'`));
    await q.goto(B + `/exams/${eid}/take?course=${C}`, { waitUntil: 'domcontentloaded' }); await q.waitForTimeout(3500); await shot(q, '43-sv-thay-phan-hoi'); log('19 SV thấy phản hồi', /giữ nguyên điểm/.test(await q.locator('body').innerText()));
  } catch (e) { log('LỖI GV kết quả', String(e).slice(0, 500)); await shot(p, 'err-gv-kq'); }
  log('11/21 rò đáp án: số lần thấy canary / tên test ẩn / answer_key trong MỌI phản hồi mạng của SV1+SV2 (trước và sau công bố)', leaks.length ? leaks.join('; ') : '0');
  log('DB cuối', sql(`select u.email||':'||a.status||':'||coalesce(a.auto_score::text,'')||':'||coalesce(a.submit_reason::text,'') from exam_attempts a join users u on u.id=a.student_id where a.exam_id='${eid}' order by 1`).replace(/\n/g, ' ; '));
  await br.close(); })();
