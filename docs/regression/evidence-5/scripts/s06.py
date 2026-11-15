import os; os.environ['SUITE']='PE-06'
from r5lib import *
import hashlib
def S(m, p, tk=T, b=None, h=None, **k): return call(m, p, tk, b, h=h, **k)
qcode = mkcode(60); qm = mkq(61)
eid, _ = mkexam(open_s=8, win=2400, dur=40, items=[qm, qcode], title='R5 06 code'); EX = f'{E}/{eid}'; time.sleep(10)
def ctx(u):
    aid, j, H, s, _ = start(eid, u); it = [x for x in j['items'] if x['type'] == 'CODE'][0]
    return dict(t=login(u), aid=aid, H=H, it=it['item_id'], j=j, base=f'{EX}/attempts/{aid}', u=u)
def run(c, src, lang='cpp17', key=True, wait=True):
    s, rr, tt = S('POST', c['base'] + f'/code/{c["it"]}/run', c['t'], dict(language=lang, source=src), {**c['H'], **(idem() if key else {})})
    if s != 202 or not wait: return s, rr, None
    for _ in range(60):
        s2, r, t2 = S('GET', c['base'] + '/runs/' + rr['run_id'], c['t'])
        if r.get('status') in ('DONE', 'ERROR'): return s, rr, r
        time.sleep(1)
    return s, rr, r
def submit(c, src, lang='cpp17', key=True): return S('POST', c['base'] + f'/code/{c["it"]}/submit', c['t'], dict(language=lang, source=src), {**c['H'], **(idem() if key else {})})
def wait_sub(c, sid):
    for _ in range(90):
        s, r, t = S('GET', c['base'] + f'/submissions/{sid}', c['t'])
        if r.get('status') in ('DONE', 'ERROR', 'SUPERSEDED'): return r, t
        time.sleep(1)
    return r, t
c1 = ctx('r5s01'); jt = json.dumps(c1['j'])
code = c1['it'] and [x for x in c1['j']['items'] if x['type'] == 'CODE'][0].get('code', {})
chk('TC-PE06-01', HID not in jt and 'QC-REF' not in jt and set(code) <= {'languages', 'time_limit_ms', 'memory_limit_mb', 'starter_code', 'samples'} and all(set(s_) <= {'name', 'input', 'expected'} for s_ in code.get('samples', [])), f'khoá code: {sorted(code)}; samples {len(code.get("samples", []))}; 0 canary test ẩn / lời giải')
# TC16
SRC = {'ok': SUM, 'wa': SUM.replace('a+b', 'a+b+1'), 'ce': 'int main(){ undefined_fn(); }', 'tle': '#include <cstdio>\nint main(){ for(;;); }', 're': '#include <cstdlib>\nint main(){ abort(); }'}
V = {}
for n, src in SRC.items():
    s, rr, r = run(c1, src); v = (r or {}).get('samples') and [x['verdict'] for x in r['samples']] or ('CE' if r and r.get('compile_ok') is False else None); V[n] = (s, v); time.sleep(0.5)
exp = {'ok': ['AC', 'AC'], 'wa': ['WA', 'WA'], 'ce': 'CE', 'tle': ['TLE', 'TLE'], 're': ['RE', 'RE']}
chk('TC-PE06-16', all(V[n][0] == 202 and V[n][1] == exp[n] for n in SRC), str(V) + f'; stdout/hidden: HID trong kết quả: {HID in json.dumps(V)}')
# TC17 giới hạn 10/10 phút: đã 5 lần
res17 = []
for i in range(7): res17.append(run(c1, SUM, wait=False)[0]); time.sleep(0.3)
s11, r11, _ = run(c1, SUM, wait=False)
chk('TC-PE06-17', res17[:5] == [202] * 5 and 429 in res17 and 'retry_after' in json.dumps(r11) or 'retry_after' in str(call.last_headers), f'lần 6–12: {res17}+{s11}; lần thứ 11 → 429 (RATE_LIMITED) retry_after={r11.get("retry_after")}')
# TC18 đầu vào sai
c2 = ctx('r5s02')
e18 = [run(c2, SUM, key=False, wait=False)[0:2], run(c2, '', wait=False)[0:2], run(c2, 'x' * 65537, wait=False)[0:2], run(c2, SUM, lang='java', wait=False)[0:2]]
other = S('POST', c2['base'].replace(c2['aid'], c1['aid']) + f'/code/{c1["it"]}/run', c2['t'], dict(language='cpp17', source=SUM), {**c2['H'], **idem()})[0]
chk('TC-PE06-18', e18[0][0] == 422 and 'IDEMPOTENCY_KEY_REQUIRED' in codes(e18[0][1]) and e18[1][0] == 422 and 'SOURCE_EMPTY' in codes(e18[1][1]) and e18[2][0] == 422 and 'SOURCE_TOO_LARGE' in codes(e18[2][1]) and e18[3][0] == 422 and 'LANGUAGE_NOT_ALLOWED' in codes(e18[3][1]) and other == 404, f'{[(x[0], codes(x[1])) for x in e18]}; chạy vào lượt người khác {other}')
# TC23/24/26/29/30/34
s, rj, t = submit(c2, SRC['wa']); sid1 = rj.get('submission_id'); s2, rj2, _ = submit(c2, SRC['ok'])
r1 = wait_sub(c2, sid1)[0]; sid2 = rj2.get('submission_id')
chk('TC-PE06-23', s == 202 and db(f"select kind,source_sha256='{hashlib.sha256(SRC['wa'].encode()).hexdigest()}' from code_submissions where id='{sid1}'") == 'SUBMIT|t' and int(db(f"select count(*) from outbox where topic='judge.enqueue' and payload::text like '%{sid1}%'")) == 1, f'202; kind/sha256 đúng; outbox judge.enqueue 1')
chk('TC-PE06-24', s2 == 429 and 'RATE_LIMITED' in codes(rj2) and (rj2.get('retry_after') or 99) <= 15, f'lần hai trong 15 s: {s2} {codes(rj2)} retry_after={rj2.get("retry_after")}')
time.sleep(15.5); s3, rj3, _ = submit(c2, SRC['ok']); sid2 = rj3.get('submission_id'); r2, _ = wait_sub(c2, sid2); r1 = wait_sub(c2, sid1)[0]
chk('TC-PE06-26', s3 == 202 and r2['status'] == 'DONE' and r1['status'] in ('DONE', 'SUPERSEDED'), f'sau 15 s nộp ok {s3}; lần 1 {r1["status"]} (lần 1 nộp wa trước khi lần 2 vào hàng: {r1["status"]}); lần 2 {r2["status"]} is_final={r2.get("is_final")}')
keys = set(r2); jt29 = json.dumps(r2)
chk('TC-PE06-29', HID not in jt29 and keys <= {'id', 'status', 'language', 'created_at', 'compile_ok', 'compile_log', 'samples', 'is_final', 'source'} and r2['compile_ok'] is True, f'khoá: {sorted(keys)}; 0 canary test ẩn; ẩn chỉ đếm? hidden-khoá: {[k for k in keys if "hid" in k]}')
time.sleep(15.5); s4, rj4, _ = submit(c2, SRC['ce']); r4, _ = wait_sub(c2, rj4.get('submission_id'))
chk('TC-PE06-30', s4 == 202 and r4.get('compile_ok') is False and r4.get('compile_log') and '/tmp' not in r4['compile_log'] and 'main.cpp' in r4['compile_log'].replace('\\', ''), f'CE: compile_log {r4.get("compile_log", "")[:90]!r}')
# TC34 final = SUBMIT cuối (wa)
time.sleep(15.5); s5, rj5, _ = submit(c2, SRC['wa']); wait_sub(c2, rj5.get('submission_id'))
nl = S('GET', c2['base'] + f'/code/{c2["it"]}/submissions?limit=2', c2['t'])[1]
chk('TC-PE06-32', len(nl['items']) == 2 and nl['items'][0]['created_at'] >= nl['items'][1]['created_at'] and nl.get('next_cursor') and S('GET', c2['base'] + f'/code/{c2["it"]}/submissions', c1['t'])[0] in (403, 404), f'limit=2 → {len(nl["items"])}, mới nhất trước, cursor có; SV khác đọc lịch sử: 404/403')
sbm = submit_exam(eid, c2['aid'], 'r5s02', c2['H']); time.sleep(8)
fin = db(f"select verdict,kind,is_final from (select verdict,kind,false as is_final from code_submissions where attempt_id='{c2['aid']}' and kind='SUBMIT' order by created_at desc limit 1) x")
chk('TC-PE06-34', sbm[0] == 200 and fin.startswith('WA|SUBMIT'), f'nộp bài thi → SUBMIT cuối (wa): verdict {fin}')
# TC35 tự nộp từ nháp
c3 = ctx('r5s03'); d1 = S('PUT', c3['base'] + f'/code/{c3["it"]}/draft', c3['t'], dict(language='cpp17', source=SUM, base_rev=0), c3['H'])
sb = submit_exam(eid, c3['aid'], 'r5s03', c3['H']); time.sleep(12)
ar = db(f"select kind,auto,verdict from code_submissions where attempt_id='{c3['aid']}'")
chk('TC-PE06-35', d1[0] == 200 and 'SUBMIT|t' in ar, f'draft {d1[0]} rev={d1[1].get("rev")}; nộp bài thi → code_submissions: {ar}')
# TC36 không nháp khi có SUBMIT
c4 = ctx('r5s04'); S('PUT', c4['base'] + f'/code/{c4["it"]}/draft', c4['t'], dict(language='cpp17', source=SRC['wa'], base_rev=0), c4['H']); submit(c4, SUM); sb = submit_exam(eid, c4['aid'], 'r5s04', c4['H']); time.sleep(10)
ar = db(f"select count(*) filter (where auto), count(*) from code_submissions where attempt_id='{c4['aid']}'")
chk('TC-PE06-36', ar == '0|1', f'có SUBMIT + nháp khác: số auto/tổng = {ar} (không tạo SUBMIT auto)')
# TC37 không nháp
c5 = ctx('r5s05'); sb = submit_exam(eid, c5['aid'], 'r5s05', c5['H']); time.sleep(6)
ar = db(f"select count(*) from code_submissions where attempt_id='{c5['aid']}'")
chk('TC-PE06-37', ar == '0', f'không nháp, không nộp: {ar} code_submissions (earned=0 khi chấm: xem PE-08)')
# TC11/12 nháp
c6 = ctx('r5s06'); P = c6['base'] + f'/code/{c6["it"]}/draft'
a1 = S('PUT', P, c6['t'], dict(language='cpp17', source='a', base_rev=0), c6['H']); a2 = S('PUT', P, c6['t'], dict(language='cpp17', source='b', base_rev=a1[1]['rev']), c6['H'])
big = S('PUT', P, c6['t'], dict(language='cpp17', source='x' * 65536, base_rev=a2[1]['rev']), c6['H']); big1 = S('PUT', P, c6['t'], dict(language='cpp17', source='x' * 65537, base_rev=big[1].get('rev', 0)), c6['H'])
chk('TC-PE06-11', a1[0] == 200 and a2[1]['rev'] == a1[1]['rev'] + 1 and big[0] == 200 and big1[0] == 422 and 'SOURCE_TOO_LARGE' in codes(big1[1]) and 'source' not in json.dumps(a1[1]), f'rev {a1[1]["rev"]}→{a2[1]["rev"]}; 64KiB {big[0]}; +1 {big1[0]} {codes(big1[1])}')
cf_ = S('PUT', P, c6['t'], dict(language='cpp17', source='c', base_rev=a1[1]['rev']), c6['H'])
chk('TC-PE06-12', cf_[0] == 409 and 'DRAFT_CONFLICT' in codes(cf_[1]) and 'current_rev' in json.dumps(cf_[1]), f'base_rev cũ: {cf_[0]} {codes(cf_[1])}')
# TC09 per-language
c7 = ctx('r5s07'); P7 = c7['base'] + f'/code/{c7["it"]}/draft'
S('PUT', P7, c7['t'], dict(language='c11', source='// c11', base_rev=0), c7['H']); S('PUT', P7, c7['t'], dict(language='cpp17', source='// cpp', base_rev=0), c7['H'])
rows = db(f"select language,source from code_drafts where attempt_id='{c7['aid']}' order by language").replace('\n', ';'); jv = S('PUT', P7, c7['t'], dict(language='java', source='x', base_rev=0), c7['H'])
chk('TC-PE06-09', rows == 'c11|// c11;cpp17|// cpp' and jv[0] == 422 and 'LANGUAGE_NOT_ALLOWED' in codes(jv[1]), f'mỗi ngôn ngữ nháp riêng: {rows}; java {jv[0]} {codes(jv[1])}')
# TC44/45 rò
c8 = ctx('r5s08'); outs = []
for p_ in ['/attempts/mine']: outs.append(S('GET', EX + p_, c8['t'])[2])
r8 = run(c8, SRC['wa'])[2]; outs.append(json.dumps(r8)); outs.append(S('GET', c8['base'] + f'/code/{c8["it"]}/submissions', c8['t'])[2])
chk('TC-PE06-44', all(HID not in o and 'QC-REF' not in o and CANARY not in o for o in outs), f'0 canary test ẩn/lời giải/giải thích ở mine/run/submissions (ở trạng thái đang làm)')
x45 = [S('GET', c1['base'] + '/runs/' + str(uuid.uuid4()), c8['t'])[0], S('GET', c1['base'] + f'/code/{c1["it"]}/submissions', c8['t'])[0], S('GET', c1['base'] + f'/code/{c1["it"]}/submissions', TA)[0], S('GET', c1['base'] + f'/code/{c1["it"]}/submissions', T)[0], S('GET', c1['base'] + f'/code/{c1["it"]}/submissions', AD)[0]]
chk('TC-PE06-45', x45[1] == 404 and x45[2:] == [403, 403, 403], str(x45))
# TC25 trần 30
c9 = ctx('r5s09'); db(f"insert into code_submissions(course_id,exam_id,attempt_id,item_id,problem_id,student_id,kind,language,source,source_sha256,status) select '{C1}','{eid}','{c9['aid']}','{c9['it']}',problem_id,student_id,'SUBMIT','cpp17','x'||g,repeat('0',64),'SUPERSEDED' from code_submissions c, generate_series(1,29) g where false")
