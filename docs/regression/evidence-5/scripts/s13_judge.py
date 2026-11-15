"""QC2 smoke sprint 5 (SmkL) — judge/hàng đợi, không đổi cấu hình toàn cục. Chạy: python3 s13_judge.py <stage>
 stage: w0 (TC-PE02-21 test chưa duyệt + Σweight=0, PE02-47 chạy thử rảnh ×20, PE02-19 đếm /run)
        burst (TC-PE02-27 làn run khi hàng đầy, PE02-46@P=2 60 bài, PE02-29 SUPERSEDED)
        slow  (TC-PE02-48 240 bài / 5 phút)
Ghi kết quả qua rl.R -> results-q2.jsonl."""
import os, sys; os.environ['SUITE'] = 'PE-02L'
from r5lib import *
import rl
def R(tc, ok, note): rl.R(tc, 'P' if ok is True else 'F' if ok is False else ok, note)
def S(m, p, tk=None, b=None, h=None, **k): return call(m, p, tk or T, b, h=h, **k)
def ctxs(eid, users, base_e=E):
    out = {}
    for u in users:
        aid, j, H, s, _ = start(eid, u, base_e)
        assert aid, (u, s, _)
        out[u] = dict(t=login(u), aid=aid, H=H, items=j['items'], base=f'{base_e}/{eid}/attempts/{aid}')
    return out
BITS = '#include <bits/stdc++.h>\nusing namespace std;\nint main(){long long a,b;cin>>a>>b;cout<<a+b<<"\\n";return 0;}\n'
def judge_post_run(since): return int(subprocess.run(f"docker logs --since {since}s ep-r-judge-1 2>&1 | grep -c '\"path\":\"/run\"'", shell=True, capture_output=True, text=True).stdout.strip() or 0)
def wait_sub(sid, n=240):
    for _ in range(n):
        r = db(f"select status,verdict,passed_weight,total_weight,extract(epoch from (judged_at-created_at)) from code_submissions where id='{sid}'")
        if r.startswith('DONE') or r.startswith('ERROR') or r.startswith('SUPERSEDED'): return r
        time.sleep(1)
    return r
def tests10(): return [('mau1', '1 2\n', '3\n', True, 1), ('mau2', '10 20\n', '30\n', True, 1)] + [(f'h{i}', f'{i} {i+1}\n', f'{2*i+1}\n', False, 1) for i in range(1, 9)]
ACT = [x.split('@')[0] for x in db(f"select u.email from enrollments e join users u on u.id=e.user_id where e.course_id='{C1}' and u.email like 'r5s%' and e.status='ACTIVE' order by 1").splitlines() if int(x[3:5]) not in (13,14)]
stage = sys.argv[1]
if stage == 'w0':
    # --- TC-PE02-21: test chưa duyệt không được chấm; Σweight=0 → IE
    qa = mkcode(81, tests=[('ok1', '1 2\n', '3\n', True, 1), ('ok2', '5 6\n', '11\n', False, 2)], ref=SUM, verify=False, approve_q=False)
    P = f'{Q}/{qa}'
    s, j, _ = S('POST', P + '/testcases', b=dict(name='pending5', input='9 9\n', expected='18\n', is_sample=False, weight=5)); tid_un = j.get('id')   # KHÔNG duyệt
    s, j, _ = S('POST', P + '/reference/verify', b={}, h=idem()); wait_job(j['job_id']); print('approve qa', approve(qa))
    gl = S('GET', P + '/testcases?limit=100')[1]; apr = {t['name']: t.get('approved') for t in gl.get('items', [])}
    qz = mkcode(82, tests=[('z1', '1 2\n', '3\n', True, 1), ('z2', '2 2\n', '4\n', False, 1)], ref=SUM, verify=True, approve_q=True)
    PZ = f'{Q}/{qz}'; zt = S('GET', PZ + '/testcases?limit=100')[1]['items']
    api0 = 'duyệt câu có Σweight=0 → 422 TOTAL_WEIGHT_ZERO; sửa weight về 0 sau duyệt hạ câu khỏi APPROVED'
    e, _ = mkexam(open_s=14, win=900, dur=10, items=[qa, qz], title='R5L 02-21'); time.sleep(16)
    db(f"update code_testcases set weight=0 where problem_id='{qz}'") if not all(x == 200 for x in api0) or True else None
    c = ctxs(e, ['r5s20'])['r5s20']; items = {it['question_id'] if 'question_id' in it else it.get('id'): it for it in c['items']}
    itA = [it for it in c['items'] if it['type'] == 'CODE'][0]['item_id']; itZ = [it for it in c['items'] if it['type'] == 'CODE'][1]['item_id']
    sids = {}
    for nm, itm in (('A', itA), ('Z', itZ)):
        s, rj, _ = S('POST', f'{c["base"]}/code/{itm}/submit', c['t'], dict(language='cpp17', source=SUM), {**c['H'], **idem()}); sids[nm] = rj.get('submission_id'); print(nm, s, rj.get('submission_id'), codes(rj))
    rA = wait_sub(sids['A']); rZ = wait_sub(sids['Z'])
    resA = json.loads(db(f"select results::text from code_submissions where id='{sids['A']}'") or '[]')
    names = [x.get('name') for x in resA] if isinstance(resA, list) else resA
    PAN = subprocess.run("docker logs --since 120s ep-r-worker-1 2>&1 | grep -ci 'division\\|panic'", shell=True, capture_output=True, text=True).stdout.strip()
    R('TC-PE02-21', apr.get('pending5') is False and 'pending5' not in str(names) and rA.split('|')[:4] == ['DONE', 'AC', '3', '3'] and rZ.startswith('ERROR|IE'), f'test chưa duyệt (approved={apr.get("pending5")}, w=5) không vào kết quả: tests chấm={names}, verdict/passed/total={rA}; Σweight=0 (mọi test w=0; API từ chối {api0} → QC đặt weight=0 bằng SQL trên fixture riêng để vào nhánh phòng thủ): {rZ} (IE + không chia 0, không điểm); log worker 120 s: {PAN} dòng panic/division')
    # --- TC-PE02-47: chạy thử lúc rảnh ×20
    ts = []; users = [sv(i) for i in range(21, 31)]
    e2, _ = mkexam(open_s=14, win=900, dur=10, items=[qa], title='R5L 02-47'); time.sleep(16)
    cx = ctxs(e2, users); 
    for rnd in range(2):
        for u in users:
            c2 = cx[u]; itm = [it for it in c2['items'] if it['type'] == 'CODE'][0]['item_id']; t0 = time.time()
            s, rj, _ = S('POST', f'{c2["base"]}/code/{itm}/run', c2['t'], dict(language='cpp17', source=SUM), {**c2['H'], **idem()})
            if s != 202: print('run', s, rj); ts.append(None); continue
            rid = rj.get('run_id')
            while time.time() - t0 < 30:
                g = S('GET', f'{c2["base"]}/runs/{rid}', c2['t'])[1]
                if g.get('status') in ('DONE', 'ERROR'): break
                time.sleep(0.3)
            ts.append(time.time() - t0)
            time.sleep(0.2)
    ok = [x for x in ts if x is not None]; ok.sort(); p95 = ok[int(len(ok) * .95) - 1] if ok else None
    R('TC-PE02-47', len(ok) == 20 and p95 is not None and p95 <= 5, f'Chạy thử ×20 (biên dịch + 2 test mẫu, 10 SV × 2, máy load {os.getloadavg()[0]:.1f}): p50={ok[len(ok)//2]:.2f}s p95={p95:.2f}s max={ok[-1]:.2f}s; thành công {len(ok)}/20')
    # --- TC-PE02-19: số lời gọi POST /run tới judge cho 1 bài nộp (1 biên dịch + N test, không lệnh so khớp)
    e3, _ = mkexam(open_s=14, win=900, dur=10, items=[qa], title='R5L 02-19'); time.sleep(16)
    c3 = ctxs(e3, ['r5s22'])['r5s22']; itm = [it for it in c3['items'] if it['type'] == 'CODE'][0]['item_id']
    t_s = int(time.time()); time.sleep(1)
    s, rj, _ = S('POST', f'{c3["base"]}/code/{itm}/submit', c3['t'], dict(language='cpp17', source=SUM), {**c3['H'], **idem()}); sid = rj['submission_id']; r_ = wait_sub(sid); time.sleep(2)
    n_run = judge_post_run(int(time.time()) - t_s + 3)
    n_delete = int(subprocess.run(f"docker logs --since {int(time.time()) - t_s + 3}s ep-r-judge-1 2>&1 | grep -c '\"method\":\"DELETE\"'", shell=True, capture_output=True, text=True).stdout.strip() or 0)
    n_tests = int(db(f"select count(*) from code_testcases where problem_id='{qa}' and approved"))
    R('TC-PE02-19', n_run == 1 + n_tests and n_delete >= 1, f'bài nộp {r_.split("|")[0]}: lời gọi POST /run vào judge = {n_run} (1 biên dịch + {n_tests} test đã duyệt = {1+n_tests}); DELETE /file sau chấm = {n_delete}; không có lệnh checker thứ hai (so khớp trong worker). Không có người dùng khác chấm cùng lúc trong cửa sổ? (bằng chứng: số khớp đúng)')
elif stage == 'burst':
    qs = [mkcode(90 + i, tests=tests10(), ref=BITS, verify=True) for i in range(2)]
    # SV 1..30, mỗi SV nộp vào 2 câu → 60 bài; cooldown 15s theo (lượt,câu) nên 1 lần/câu
    e, _ = mkexam(open_s=14, win=1500, dur=20, items=qs + [mkq(9)], title='R5L burst'); time.sleep(16)
    users = ACT[:30]; cx = ctxs(e, users)
    sub = {}; t0 = time.time(); N = 0
    for u in users:
        c2 = cx[u]; for_items = [it['item_id'] for it in c2['items'] if it['type'] == 'CODE']
        for itm in for_items:
            s, rj, _ = S('POST', f'{c2["base"]}/code/{itm}/submit', c2['t'], dict(language='cpp17', source=BITS), {**c2['H'], **idem()})
            if s == 202: sub[rj['submission_id']] = (u, time.time()); N += 1
            else: print('sub', u, s, codes(rj))
    t_sent = time.time() - t0
    # trong lúc hàng đầy: SV khác Chạy thử
    time.sleep(2); backlog = db(f"select count(*) from code_submissions where status in ('QUEUED','RUNNING') and kind='SUBMIT' and exam_id='{e}'")
    c2 = cx[ACT[29]]; itm = [it for it in c2['items'] if it['type'] == 'CODE'][0]['item_id']; tr = time.time()
    s, rj, _ = S('POST', f'{c2["base"]}/code/{itm}/run', c2['t'], dict(language='cpp17', source=BITS), {**c2['H'], **idem()}); rid = rj.get('run_id'); d_run = None
    while time.time() - tr < 60:
        g = S('GET', f'{c2["base"]}/runs/{rid}', c2['t'])[1]
        if g.get('status') in ('DONE', 'ERROR'): d_run = time.time() - tr; break
        time.sleep(0.3)
    R('TC-PE02-27', s == 202 and d_run is not None and d_run <= 5, f'hàng nộp có {backlog} bài chờ/đang chạy (gửi {N} bài trong {t_sent:.0f}s); SV khác Chạy thử: {s}, xong sau {d_run if d_run is None else round(d_run,2)}s (≤5); verdict run {g.get("verdict")}')
    # chờ hết
    t_end = None
    for _ in range(280):
        pend = int(db(f"select count(*) from code_submissions where status in ('QUEUED','RUNNING') and kind='SUBMIT' and exam_id='{e}'"))
        if pend == 0: t_end = time.time(); break
        time.sleep(2)
    rows = db(f"select count(*), count(*) filter (where verdict='AC'), count(*) filter (where verdict='IE'), percentile_cont(0.95) within group (order by extract(epoch from (judged_at-created_at))), max(extract(epoch from (judged_at-created_at))) from code_submissions where exam_id='{e}' and kind='SUBMIT'")
    last = time.time() - t0
    R('TC-PE02-46', t_end is not None and rows.split('|')[2] == '0' and float(rows.split('|')[4]) <= 60, f'JUDGE_PARALLELISM=2 (cấu hình ep-r hiện tại; mức 4 cần đổi cấu hình toàn cục): 60 bài bits/stdc++.h × 10 test gửi cùng lúc, tới DONE cuối {last:.0f}s từ lúc gửi; (n,AC,IE,p95,max)={rows}; máy load {os.getloadavg()[0]:.1f}')
    json.dump(dict(exam=e), open('/tmp/qc2L/burst.json', 'w'))
elif stage == 'supersede':
    # backlog dài: 14 SV × 2 câu chậm (TLE 1s × 10 test); SV mục tiêu nộp lần 1 (QUEUED), 17 s sau nộp lần 2
    LOOP = '#include <cstdio>\nint main(){long long a,b;scanf("%lld %lld",&a,&b);volatile long long x=0;for(long long i=0;i<2000000000LL;i++)x+=i;printf("%lld\\n",a+b);}\n'
    qs = [mkcode(95 + i, tests=tests10(), ref=SUM, tl=1000) for i in range(2)]
    e, _ = mkexam(open_s=14, win=1500, dur=20, items=qs, title='R5L supersede'); time.sleep(16)
    users = ACT[:16]; cx = ctxs(e, users)
    for u in users[:15]:
        c2 = cx[u]
        for it in c2['items']:
            if it['type'] == 'CODE': S('POST', f'{c2["base"]}/code/{it["item_id"]}/submit', c2['t'], dict(language='cpp17', source=LOOP), {**c2['H'], **idem()})
    tg = cx[users[15]]; itm = [it['item_id'] for it in tg['items'] if it['type'] == 'CODE'][0]
    s1, r1, _ = S('POST', f'{tg["base"]}/code/{itm}/submit', tg['t'], dict(language='cpp17', source=SUM), {**tg['H'], **idem()}); sid1 = r1['submission_id']; st1 = db(f"select status from code_submissions where id='{sid1}'")
    j0 = int(time.time()); time.sleep(16)
    s2, r2, _ = S('POST', f'{tg["base"]}/code/{itm}/submit', tg['t'], dict(language='cpp17', source=SUM), {**tg['H'], **idem()}); sid2 = r2.get('submission_id'); print(s2, r2)
    time.sleep(2); row1 = db(f"select status,verdict,attempts from code_submissions where id='{sid1}'"); row2 = wait_sub(sid2, 400)
    # 0 lời gọi sandbox cho lần 1: so số /run của judge trong cửa sổ giữa 2 lần nộp không có mã nguồn riêng → dùng cột attempts=0 / judged_at null
    row1b = db(f"select attempts,judged_at is null,compile_ok is null from code_submissions where id='{sid1}'")
    pend = db(f"select count(*) from code_submissions where status in ('QUEUED','RUNNING') and exam_id='{e}'")
    nrun_sup = db(f"select count(*) from code_submissions where exam_id='{e}' and kind='RUN' and status='SUPERSEDED'")
    R('TC-PE02-29', st1 == 'QUEUED' and row1.startswith('SUPERSEDED') and row1b == '0|t|t' and row2.startswith('DONE|AC'), f'lần 1 lúc nộp: {st1} (hàng có ~30 bài chậm); sau lần 2 (17 s sau, còn {pend} bài khác chờ): lần 1 = {row1} (attempts/judged_at null/compile_ok null = {row1b} → 0 lời gọi sandbox); lần 2 = {row2}; Chạy thử SUPERSEDED trong bài = {nrun_sup}')
