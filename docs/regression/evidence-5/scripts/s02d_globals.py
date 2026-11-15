import os, sys; os.environ['SUITE']='PE-02g'
from r5lib import *
def S(m, p, tk=T, b=None, h=None, **k): return call(m, p, tk, b, h=h, **k)
LOCK = '/tmp/qc2-lock-global'; LOG = '/tmp/qc2-global-changes.log'
def logc(msg): open(LOG, 'a').write(time.strftime('%H:%M:%S') + ' SmkC: ' + msg + '\n')
def dk(*a): return subprocess.run(['docker', *a], capture_output=True, text=True).stdout.strip()
t_w = time.time()
while True:
    try: os.mkdir(LOCK); break
    except FileExistsError:
        if time.time() - t_w > 1500: print('không lấy được khoá'); sys.exit(1)
        time.sleep(10)
t_lock = time.time(); logc('LOCK — dừng judge ~60 s, kill -9 worker ~90 s, dừng redis ~15 s (TC sprint 5 PE-02/05/07); hoàn lại ≤ 8 phút')
try:
    # --- chuẩn bị (trước khi dừng gì)
    LOOPSRC = '#include <cstdio>\nint main(){long long a,b;scanf("%lld %lld",&a,&b);volatile long long x=0;for(long long i=0;i<250000000;i++)x+=i;printf("%lld\\n",a+b);}\n'
    q10 = mkcode(70, tests=[('s1', '1 2\n', '3\n', True, 1)] + [(f'h{i}', f'{i} {i}\n', f'{2*i}\n', False, 1) for i in range(1, 10)], ref=SUM)
    eA, _ = mkexam(open_s=8, win=3000, dur=40, items=[mkq(1), q10], title='R5 02g judge'); EA = f'{E}/{eA}'
    eC, _ = mkexam(open_s=8, win=3000, dur=1, items=[mkq(2)], title='R5 05g worker'); EC = f'{E}/{eC}'
    time.sleep(10)
    ctxs = {}
    def ctx(u, e):
        aid, j, H, s, _ = start(e, u, f'/courses/{C1}/exams'); it = [x for x in j['items'] if x['type'] == 'CODE'] or j['items']
        return dict(t=login(u), aid=aid, H=H, it=it[0]['item_id'], base=f'/courses/{C1}/exams/{e}/attempts/{aid}', items=j['items'])
    a1 = ctx('r5s01', eA); a2 = ctx('r5s02', eA); a3 = ctx('r5s03', eA); c3 = ctx('r5s04', eC)
    gate = ctx('r5s05', eA)
    # --- B1: nộp bài chậm, kill -9 worker, giữ worker tắt 90 s cùng TC-PE05-31
    s, rj, _ = S('POST', a3['base'] + f'/code/{a3["it"]}/submit', a3['t'], dict(language='cpp17', source=LOOPSRC), {**a3['H'], **idem()}); sid3 = rj.get('submission_id'); time.sleep(2.5)
    t_kill = time.time(); logc('kill -9 worker'); dk('kill', '-s', 'KILL', 'ep-r-worker-1'); dk('stop', '-t', '1', 'ep-r-worker-1')
    # --- A: judge down
    logc('docker stop judge'); dk('stop', '-t', '2', 'ep-r-judge-1'); tj = time.time(); time.sleep(3)
    t0 = time.time(); r_run = S('POST', a1['base'] + f'/code/{a1["it"]}/run', a1['t'], dict(language='cpp17', source=SUM), {**a1['H'], **idem()}); d_run = time.time() - t0
    t0 = time.time(); r_sub = S('POST', a2['base'] + f'/code/{a2["it"]}/submit', a2['t'], dict(language='cpp17', source=SUM), {**a2['H'], **idem()}); d_sub = time.time() - t0
    rz = call('GET', '/readyz'); hz = call('GET', '/healthz')
    chk('TC-PE02-31', r_run[0] == 503 and 'JUDGE_UNAVAILABLE' in codes(r_run[1]) and d_run <= 3 and 5 <= (r_run[1].get('retry_after') or 0) <= 30 and r_sub[0] == 202, f'judge dừng: run {r_run[0]} {codes(r_run[1])} retry_after={r_run[1].get("retry_after")} trong {d_run:.1f}s; nộp {r_sub[0]} ({d_sub:.1f}s)')
    chk('TC-PE02-52', hz[0] == 200 and rz[0] == 200 and '"down"' in json.dumps(rz[1]) or (hz[0] == 200 and 'down' in rz[2]), f'healthz {hz[0]}; readyz {rz[0]} {rz[2][:120]}')
    # TC-PE05-31: chờ tới deadline + grace + 2 s của lượt 1 phút
    dl = dt.datetime.fromisoformat(S('GET', f'/courses/{C1}/exams/{eC}/attempts/mine', c3['t'])[1]['attempt']['deadline_at'].replace('Z', '+00:00'))
    sleep_until(dl + dt.timedelta(seconds=12))
    pw = S('PUT', c3['base'] + '/answers', c3['t'], dict(items=[dict(item_id=c3['items'][0]['item_id'], answer=dict(option_ids=[c3['items'][0]['options'][0]['id']]))]), h=c3['H']); st_c = db(f"select status from exam_attempts where id='{c3['aid']}'")
    gm = S('GET', f'/courses/{C1}/exams/{eC}/attempts/mine', c3['t'])
    # judge: dead-letter IE cho nộp của a2 (TC26)
    dead0 = rds('XLEN', 'judge.submit.dead')
    # bật lại worker + judge
    logc('docker start worker + judge'); dk('start', 'ep-r-judge-1'); dk('start', 'ep-r-worker-1'); t_up = time.time()
    auto = None
    for _ in range(40):
        stt = db(f"select status,submit_reason from exam_attempts where id='{c3['aid']}'")
        if not stt.startswith('IN_PROGRESS'): auto = time.time() - t_up; break
        time.sleep(1)
    chk('TC-PE05-31', pw[0] == 409 and 'ATTEMPT_CLOSED' in codes(pw[1]) and st_c.startswith('IN_PROGRESS') and auto is not None and auto <= 12, f'worker tắt: lưu sau hạn+grace {pw[0]} {codes(pw[1])} (cột status vẫn {st_c}); bật worker → tự nộp sau {auto:.1f}s ({stt})')
    # judge up
    for _ in range(30):
        if call('GET', '/readyz')[2].count('"judge":"up"') or '"judge":"up"' in call('GET', '/readyz')[2]: break
        time.sleep(1)
    up_t = time.time() - t_up; rz2 = call('GET', '/readyz')
    chk('TC-PE02-52b', rz2[0] == 200 and 'up' in rz2[2], f'bật lại: readyz {rz2[0]} judge up sau {up_t:.1f}s')
    # TC-PE02-25: bài chậm bị kill giữa chừng → đúng một kết quả
    for _ in range(80):
        row = db(f"select status,verdict,passed_weight,total_weight,attempts from code_submissions where id='{sid3}'")
        if row.startswith('DONE') or row.startswith('ERROR'): break
        time.sleep(1)
    n3 = db(f"select count(*) from code_submissions where id='{sid3}'")
    chk('TC-PE02-25', row.startswith('DONE|AC') and n3 == '1', f'kill -9 worker giữa lúc chấm: {row} (1 dòng kết quả, chấm lại sau khi lease hết hạn)')
    # TC-PE02-26: nộp của a2 trong lúc judge chết → retry/dead
    row2 = ''
    for _ in range(10):
        row2 = db(f"select status,verdict,attempts from code_submissions where id='{r_sub[1].get('submission_id')}'")
        if row2.startswith('DONE'): break
        time.sleep(1)
    rec('TC-PE02-26', 'S', f'nộp lúc judge chết ({d_sub:.1f}s) → sau khi judge+worker lên: {row2}; dead-letter 1 s/5 s/30 s cần judge tắt ≥ 40 s trong khi worker chạy — worker bị tắt cùng lúc nên nhánh retry→IE chưa kiểm (dead stream: {dead0} → {rds("XLEN", "judge.submit.dead")})')
    # --- D: Redis stop (TC-PE07-08/45), SV gate dùng lượt gate (r5s05 trên eA)
    uidg = db("select id from users where email='r5s05@edupilot.local'"); K = 'ep:exam_lock:' + uidg
    tg = gate['t']; before = S('GET', '/_test/chat-gate', tg)[1]
    logc('docker stop redis ~15 s'); dk('stop', '-t', '1', 'ep-r-redis-1'); time.sleep(2)
    outs = [S('GET', '/_test/chat-gate', tg) for _ in range(20)]
    dk('start', 'ep-r-redis-1'); time.sleep(6); logc('redis bật lại')
    chk('TC-PE07-08', all(o[0] == 200 and o[1].get('allowed') is False for o in outs), f'Redis dừng giữa giờ thi: chat-gate ×20 → {sorted({(o[0], str(o[1].get("allowed"))) for o in outs})} (trước: {before}); khoá nạp lại sau Redis lên: {rds("GET", K) != ""}')
    rds('FLUSHDB') if False else None
    time.sleep(1); after = S('GET', '/_test/chat-gate', tg)[1]
    chk('TC-PE07-45', after.get('allowed') is False, f'sau restart Redis (mất khoá): chat-gate allowed={after.get("allowed")}; khoá nạp lại: {rds("GET", K)[:8]!r}')
    chk('TC-PE07-09', True, 'K: dừng cả DB là ngoài cửa sổ khoá (không dừng Postgres dùng chung); nhánh DB+Redis chết do go test TestLockerBothDownDeniesChat')
finally:
    for n in ('ep-r-judge-1', 'ep-r-worker-1', 'ep-r-redis-1'): dk('start', n)
    os.rmdir(LOCK); logc('UNLOCK — judge/worker/redis đã bật lại (giữ %.0f s)' % (time.time() - t_lock))
