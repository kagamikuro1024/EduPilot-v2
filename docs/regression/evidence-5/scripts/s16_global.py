"""QC2 smoke sprint 5 (SmkL) — cửa sổ khoá toàn cục (mkdir /tmp/qc2-lock-global; ghi /tmp/qc2-global-changes.log; hoàn lại cấu hình mặc định).
 g1: judge dừng (TC-PE02-15/26/31/52/53, PE06-19 phần 503, PE08-06)         ~4 phút
 g2: 2 worker, leader, tắt worker qua hạn (TC-PE04-24/25/26, PE05-31, PE02-25 kill -9)   ~5 phút
 g4: 2 worker + kill -9 leader liên tục + Redis pause khi 20 bài đóng/công bố (TC-PE08-18/65)
 g3: Redis dừng / Redis+Postgres dừng (TC-PE07-08/09/45)                   ~1 phút
Chạy: python3 s16_global.py g1|g2|g3"""
import os, sys; os.environ['SUITE'] = 'PE-GL'
from r5lib import *
import rl
def R(tc, ok, note): rl.R(tc, 'P' if ok is True else 'F' if ok is False else ok, note)
def S(m, p, tk=None, b=None, h=None, **k): return call(m, p, tk or T, b, h=h, **k)
LOCK = '/tmp/qc2-lock-global'; LOG = '/tmp/qc2-global-changes.log'
def logc(msg): open(LOG, 'a').write(time.strftime('%H:%M:%S') + ' SmkL: ' + msg + '\n')
def dk(*a): return subprocess.run(['docker', *a], capture_output=True, text=True).stdout.strip()
WT = os.path.expanduser('~/Documents/TA_Agent_qc2')
def compose(*a):
    env = subprocess.run(['bash', '-c', 'set -a; source /tmp/ep-r.env; set +a; cd %s; docker compose --env-file .env.local -f docker-compose.local.yml -f docker-compose.test.yml -f /tmp/ep-r.seed.override.yml -p ep-r %s' % (WT, ' '.join(a))], capture_output=True, text=True)
    return env.stdout + env.stderr
def img(n): return dk('inspect', n, '--format', '{{.Config.Image}}')
stage = sys.argv[1]
ACT = [x.split('@')[0] for x in db(f"select u.email from enrollments e join users u on u.id=e.user_id where e.course_id='{C1}' and u.email like 'r5s%' and e.status='ACTIVE' order by 1").splitlines() if int(x[3:5]) not in (13, 14)]
def ctxs(eid, users):
    out = {}
    for u in users:
        aid, j, H, s, _ = start(eid, u); assert aid, (u, s, _)
        out[u] = dict(t=login(u), aid=aid, H=H, items=j['items'], base=f'{E}/{eid}/attempts/{aid}')
    return out
def codeitem(c): return [i for i in c['items'] if i['type'] == 'CODE'][0]['item_id']
def readyz(): s, j, t = call('GET', '/readyz'); return s, t
if stage in ('g4', 'g1'):  # Main xếp hàng khoá: nhường tới khi thấy cả hai dòng UNLOCK (rebuild + US-P6-03) SAU lượt g3 của SmkL
    _t = time.time()
    while time.time() - _t < 3000:
        _txt = open(LOG, encoding='utf-8', errors='ignore').read(); _i = _txt.rfind('SmkL: UNLOCK — g3'); _new = _txt[_i:] if _i >= 0 else _txt
        if 'UNLOCK rebuild' in _new and 'UNLOCK — US-P6-03' in _new: break
        time.sleep(15)
t_w = time.time()
while True:
    try: os.mkdir(LOCK); break
    except FileExistsError:
        if time.time() - t_w > 1500: print('không lấy được khoá'); sys.exit(1)
        time.sleep(15)
t_lock = time.time(); logc(f'LOCK — {stage}: ' + {'g1': 'dừng judge ~4 phút (TC sprint 5 PE-02/06/08)', 'g2': 'scale worker=2, kill -9 / dừng worker ~3 phút (TC PE-04/05/02)', 'g3': 'dừng redis ~10 s và redis+postgres ~8 s (TC PE-07)', 'g4': 'scale worker=2, kill -9 leader nhiều lần, docker pause redis ~5 s ×2 (TC PE-08)'}[stage] + '; hoàn lại ≤ 8 phút')
imgs0 = {n: img(n) for n in ('ep-r-gateway-1', 'ep-r-worker-1')}
try:
    if stage == 'g1':
        q10 = mkcode(70, tests=[('s1', '1 2\n', '3\n', True, 1)] + [(f'h{i}', f'{i} {i}\n', f'{2*i}\n', False, 1) for i in range(1, 10)], ref=SUM)
        eA, _ = mkexam(open_s=8, win=200, dur=3, items=[mkq(1), q10], title='R5L g1 judge'); time.sleep(10)
        gv = ACT[:3]; cx = ctxs(eA, gv); a1, a2, a3 = (cx[u] for u in gv)
        t0 = time.time(); logc('docker stop judge'); dk('stop', '-t', '2', 'ep-r-judge-1'); t_down = None
        for _ in range(90):
            rzz = readyz()
            if '"judge":"down"' in rzz[1].replace(' ', ''): t_down = time.time() - t0; break
            time.sleep(1)
        tmark = time.strftime('%Y-%m-%dT%H:%M:%S')
        t1 = time.time(); r_run = S('POST', f'{a1["base"]}/code/{codeitem(a1)}/run', a1['t'], dict(language='cpp17', source=SUM), {**a1['H'], **idem()}); d_run = time.time() - t1
        t1 = time.time(); r_sub = S('POST', f'{a2["base"]}/code/{codeitem(a2)}/submit', a2['t'], dict(language='cpp17', source=SUM), {**a2['H'], **idem()}); d_sub = time.time() - t1; sid = r_sub[1].get('submission_id')
        rz = readyz(); hz = call('GET', '/healthz')
        # tiến trình thử lại tới dead-letter
        prog = []; last = None; dead0 = rds('XLEN', 'judge.submit.dead'); dead_t = None
        for _ in range(110):
            row = db(f"select status,coalesce(verdict::text,''),attempts,fail_count from code_submissions where id='{sid}'")
            if row != last: prog.append((round(time.time() - t0, 1), row)); last = row
            if row.startswith('ERROR'): dead_t = time.time() - t0; break
            time.sleep(1)
        dead1 = rds('XLEN', 'judge.submit.dead'); msg = subprocess.run(['docker', 'exec', 'ep-r-redis-1', 'redis-cli', 'XREVRANGE', 'judge.submit.dead', '+', '-', 'COUNT', '1'], capture_output=True, text=True).stdout
        warn = int(subprocess.run("(docker logs --since %ds ep-r-gateway-1 2>&1; docker logs --since %ds ep-r-worker-1 2>&1) | grep -i 'judge' | grep -ci '\"level\":\"warn\"\\|level=WARN'" % (int(time.time() - t0) + 2, int(time.time() - t0) + 2), shell=True, capture_output=True, text=True).stdout.strip() or 0)
        R('TC-PE02-31', t_down is not None and r_run[0] == 503 and 'JUDGE_UNAVAILABLE' in codes(r_run[1]) and d_run <= 3 and 5 <= (r_run[1].get('retry_after') or 0) <= 30 and r_sub[0] == 202, f'judge dừng (readyz báo judge:down sau {t_down and round(t_down,1)} s): Chạy thử {r_run[0]} {codes(r_run[1])} retry_after={r_run[1].get("retry_after")} sau {d_run:.2f}s (≤3); nộp {r_sub[0]} QUEUED ({d_sub:.2f}s)')
        R('TC-PE06-19', r_run[0] == 503 and d_run <= 3, f'(phần judge chết) run → 503 JUDGE_UNAVAILABLE retry_after={r_run[1].get("retry_after")} trong {d_run:.2f}s; phần hàng đầy + run ≤5s: TC-PE02-27 (s13_judge.py burst)')
        R('TC-PE02-52', hz[0] == 200 and rz[0] == 200 and 'down' in rz[1], f'judge dừng: healthz {hz[0]}; readyz {rz[0]} {rz[1][:100]}; log warn liên quan judge trong {int(time.time()-t0)} s: {warn}')
        sj = json.dumps(prog)
        R('TC-PE02-26', dead_t is not None and dead1 == str(int(dead0) + 1) and 'package main' not in msg and '#include' not in msg, f'judge dừng, worker chạy: tiến trình thử lại (giây, status|verdict|attempts|fail_count) = {sj}; dead-letter sau {dead_t and round(dead_t,1)} s; judge.submit.dead {dead0}→{dead1} (1 tin), tin cuối có mã nguồn: {"#include" in msg}')
        R('TC-PE02-15', dead_t is not None and prog[-1][1].startswith('ERROR|IE'), f'judge lỗi giữa chấm: verdict cuối = {prog[-1][1] if prog else None} (IE, không RE/WA, không điểm 0 — lượt ở GRADING, xem PE08-06)')
        # chờ bài đóng + lượt vào GRADING; kiểm Today
        sleep_until(dt.datetime.fromisoformat(db(f"select to_char(closes_at at time zone 'utc','YYYY-MM-DD\"T\"HH24:MI:SS+00:00') from exams where id='{eA}'")) + dt.timedelta(seconds=25))
        st2 = db(f"select status,submit_reason from exam_attempts where id='{a2['aid']}'"); tg = S('GET', '/me/today', T)[2]; tta = S('GET', '/me/today', TA)[2]
        has = lambda t: 'EXAM_GRADE_ERROR' in t
        R('TC-PE02-53', has(tg) and has(tta), f'judge down > 60 s, có bài chờ chấm (IE): Today GV EXAM_GRADE_ERROR={has(tg)}, TA={has(tta)}')
        time.sleep(40); ex_st = db(f"select status,published_at is not null from exams where id='{eA}'"); st2b = db(f"select status from exam_attempts where id='{a2['aid']}'"); nn = db(f"select count(*) from notifications where link like '%{eA}%'")
        R('TC-PE08-20', st2b == 'GRADING' and ex_st.startswith('CLOSED|f'), f'1 lượt kẹt GRADING (IE, judge dừng): sau đóng +65 s bài ở {ex_st} (CLOSED, published_at null — không công bố khi còn GRADING), lượt {st2b}; thông báo công bố đã gửi: {nn}')
        t_up = time.time(); logc('docker start judge'); dk('start', 'ep-r-judge-1')
        for _ in range(60):
            if '"judge":"up"' in readyz()[1] or 'judge' in readyz()[1] and 'up' in readyz()[1]: break
            time.sleep(1)
        up_t = time.time() - t_up; rz2 = readyz()
        R('TC-PE02-52', rz2[0] == 200 and up_t <= 10 + 10, f'judge dừng → healthz 200 độc lập, readyz 200 judge:"down" (log warn {warn} dòng trong {int(dead_t or 0)}+ s); bật lại: readyz judge=up sau {up_t:.1f}s (TC ≤10 s cộng nhịp kiểm 10 s của máy chủ)')
        rg = S('POST', f'{E}/{eA}/regrade', b=dict(scope='errors', reason='judge lên lại'), h=idem()); time.sleep(1)
        for _ in range(90):
            st3 = db(f"select status,auto_score from exam_attempts where id='{a2['aid']}'"); sr = db(f"select status,verdict from code_submissions where id='{sid}'")
            if st3.startswith('GRADED'): break
            time.sleep(2)
        R('TC-PE08-06', st2.startswith('GRADING') and has(tg) and rg[0] == 202 and st3.startswith('GRADED'), f'dead-letter IE: lượt ở {st2} (không GRADED, không 0 điểm); việc EXAM_GRADE_ERROR cho GV+TA; judge lên + regrade scope=errors → {rg[0]}; lượt {st3}, bài nộp {sr}')
    elif stage == 'g2':
        logc('scale worker=2'); out = compose('up -d --no-recreate --scale worker=2 worker'); time.sleep(8)
        names = [n for n in dk('ps', '--filter', 'name=ep-r-worker', '--format', '{{.Names}}').split('\n') if n]
        leader = rds('GET', 'ep:exam:tick:leader'); ttl = rds('PTTL', 'ep:exam:tick:leader')
        eB, _ = mkexam(open_s=10, win=110, dur=1, items=[mkq(2)], title='R5L g2 close'); eC, _ = mkexam(open_s=75, win=75, dur=1, items=[mkq(3)], title='R5L g2 sched'); time.sleep(12)
        n_open = db(f"select count(*) from outbox where topic='exam.opened' and payload::text like '%{eB}%'")
        a3 = ctxs(eB, [ACT[3]])[ACT[3]]
        R('TC-PE04-25', len(names) == 2 and leader != '' , f'2 worker ({names}); ep:exam:tick:leader={leader} (PTTL {ttl} ms ≤ 15000): 1 người giữ; outbox exam.opened của bài mở: {n_open} dòng (1)')
        # kill -9 leader
        lid_host = {n: dk('inspect', n, '--format', '{{.Config.Hostname}}') for n in names}; lname = [n for n, h in lid_host.items() if leader and (leader.startswith(h) or h in leader)] or [names[0]]
        tk = time.time(); logc('kill -9 leader ' + lname[0]); dk('kill', '-s', 'KILL', lname[0])
        new = leader
        for _ in range(40):
            new = rds('GET', 'ep:exam:tick:leader')
            if new and new != leader: break
            time.sleep(1)
        took = time.time() - tk
        R('TC-PE04-25', new not in ('', leader) and took <= 25, f'2 worker ({names}); leader={leader} (PTTL {ttl} ms ≤ 15 s), outbox exam.opened = {n_open} dòng (1); kill -9 leader {lname[0]} → leader mới {new} sau {took:.1f}s (≤ TTL 15 s + nhịp 5 s)')
        # tắt hết worker trước closes_at của eB
        for n in names: dk('kill', '-s', 'KILL', n)
        logc('mọi worker dừng (kill -9) để thử bộ lập lịch chết'); t_dead = time.time()
        d = dt.datetime.fromisoformat(db(f"select to_char(closes_at at time zone 'utc','YYYY-MM-DD\"T\"HH24:MI:SS+00:00') from exams where id='{eB}'"))
        dl = dt.datetime.fromisoformat(S('GET', f'{E}/{eB}/attempts/mine', a3['t'])[1]['attempt']['deadline_at'].replace('Z', '+00:00'))
        sleep_until(max(d, dl) + dt.timedelta(seconds=13))
        pw = S('PUT', a3['base'] + '/answers', a3['t'], dict(items=[dict(item_id=a3['items'][0]['item_id'], answer=dict(option_ids=[a3['items'][0]['options'][0]['id']]))]), h=a3['H']); sb = submit_exam(eB, a3['aid'], ACT[3], a3['H'])
        ex = S('GET', f'{E}/{eB}', T)[1]; col = db(f"select status from exams where id='{eB}'"); st_att = db(f"select status from exam_attempts where id='{a3['aid']}'")
        R('TC-PE04-24', ex.get('effective_status') == 'CLOSED' and col == 'OPEN' and pw[0] == 409 and sb[0] == 409, f'worker tắt qua closes_at+grace: GET exam effective_status={ex.get("effective_status")} (cột status={col}); lưu {pw[0]} {codes(pw[1])}; nộp {sb[0]} {codes(sb[1])}')
        # eC: SCHEDULED qua cả closes_at khi worker tắt
        sleep_until(dt.datetime.fromisoformat(db(f"select to_char(closes_at at time zone 'utc','YYYY-MM-DD\"T\"HH24:MI:SS+00:00') from exams where id='{eC}'")) + dt.timedelta(seconds=3))
        sC0 = db(f"select status from exams where id='{eC}'")
        logc('bật lại worker-1'); dk('start', 'ep-r-worker-1'); t_up = time.time()
        auto = None
        for _ in range(60):
            if not db(f"select status from exam_attempts where id='{a3['aid']}'").startswith('IN_PROGRESS'): auto = time.time() - t_up; break
            time.sleep(1)
        for _ in range(30):
            sC = db(f"select status from exams where id='{eC}'")
            if sC in ('CLOSED', 'PUBLISHED'): break
            time.sleep(2)
        opened = db(f"select count(*) from outbox where topic='exam.opened' and payload::text like '%{eC}%'"); closed = db(f"select count(*) from outbox where topic='exam.closed' and payload::text like '%{eC}%'")
        row = db(f"select status,submit_reason from exam_attempts where id='{a3['aid']}'")
        R('TC-PE05-31', pw[0] == 409 and 'ATTEMPT_CLOSED' in codes(pw[1]) and sC0 != '' and auto is not None and auto <= 12 and row.split('|')[1] in ('CLOSED', 'TIMEOUT'), f'worker tắt qua hạn: lưu {pw[0]} {codes(pw[1])} (cột status vẫn {st_att}); bật worker → lượt tự nộp sau {auto and round(auto,1)}s: {row}')
        R('TC-PE04-26', sC0 == 'SCHEDULED' and sC in ('CLOSED', 'PUBLISHED') and opened == '0' and closed == '1', f'bài SCHEDULED (opens_at đã qua? không — worker tắt từ trước giờ mở, closes_at đã qua): trước bật worker {sC0}; sau: {sC}; outbox exam.opened={opened} (chuyển thẳng, bỏ qua OPEN), exam.closed={closed}')
    elif stage == 'g3':
        eA, _ = mkexam(open_s=8, win=3000, dur=40, items=[mkq(1)], title='R5L g3 redis'); time.sleep(10)
        c = ctxs(eA, [ACT[4]])[ACT[4]]; uidg = db(f"select id from users where email='{ACT[4]}@edupilot.local'"); K = 'ep:exam_lock:' + uidg; tg = c['t']
        before = S('GET', '/_test/chat-gate', tg)[1]
        logc('docker stop redis ~10 s'); dk('stop', '-t', '1', 'ep-r-redis-1'); time.sleep(2)
        outs = [S('GET', '/_test/chat-gate', tg) for _ in range(20)]
        dk('start', 'ep-r-redis-1'); time.sleep(8); logc('redis bật lại')
        R('TC-PE07-08', all(o[0] == 200 and o[1].get('allowed') is False for o in outs), f'Redis dừng giữa giờ thi: chat-gate ×20 → {sorted({(o[0], str(o[1].get("allowed"))) for o in outs})} (trước: {before}); sau Redis lên khoá nạp lại: key={rds("EXISTS", K)}')
        after = S('GET', '/_test/chat-gate', tg)[1]; time.sleep(2)
        R('TC-PE07-45', after.get('allowed') is False and rds('EXISTS', K) in ('0', '1'), f'restart Redis (mất khoá): chat-gate allowed={after.get("allowed")} (nhờ DB); khoá nạp lại ở yêu cầu kế: EXISTS={rds("EXISTS", K)}')
        logc('docker stop redis + postgres ~8 s'); dk('stop', '-t', '1', 'ep-r-redis-1'); dk('stop', '-t', '1', 'ep-r-postgres-1'); time.sleep(2)
        both = [S('GET', '/_test/chat-gate', tg) for _ in range(3)]
        dk('start', 'ep-r-postgres-1'); dk('start', 'ep-r-redis-1'); logc('postgres + redis bật lại'); time.sleep(12)
        R('TC-PE07-09', all(o[0] >= 500 or o[0] in (401, 503) for o in both) and not any(o[1].get('allowed') is True for o in both), f'dừng Redis + Postgres: chat-gate ×3 → {[(o[0], str(o[1].get("allowed")), codes(o[1])) for o in both]} (không có allowed:true)')
    elif stage == 'g4':
        import random, threading
        logc('scale worker=2'); compose('up -d --no-recreate --scale worker=2 worker'); time.sleep(8)
        names = [n for n in dk('ps', '--filter', 'name=ep-r-worker', '--format', '{{.Names}}').split('\n') if n]
        qs = [mkq(700), mkq(701)]; random.seed(18); exs = []
        for i in range(20):
            e, _ = mkexam(open_s=12, win=100 + random.randint(0, 50), dur=1, items=qs, title=f'R5L g4 #{i}'); exs.append(e)
        time.sleep(14)
        att = {}
        def run_ex(e, users):
            for u in users:
                aid, j, H, s_, _ = start(e, u)
                if aid: att.setdefault(e, []).append((u, aid))
        ths = [threading.Thread(target=run_ex, args=(e, ACT[(i * 3) % 27:(i * 3) % 27 + 3])) for i, e in enumerate(exs)]
        for t in ths: t.start()
        for t in ths: t.join()
        starts = sum(len(v) for v in att.values()); t0 = time.time(); kills = 0; pauses = 0
        last_close = max(dt.datetime.fromisoformat(db(f"select to_char(max(closes_at) at time zone 'utc','YYYY-MM-DD\"T\"HH24:MI:SS+00:00') from exams where id in ('" + "','".join(exs) + "')")) , dt.datetime.now(dt.timezone.utc))
        extras = []
        while dt.datetime.now(dt.timezone.utc) < last_close + dt.timedelta(seconds=70):
            time.sleep(14 + random.random() * 6)
            lead = rds('GET', 'ep:exam:tick:leader'); alive = [n for n in dk('ps', '--filter', 'name=ep-r-worker', '--format', '{{.Names}}').split('\n') if n]
            vic = [n for n in alive if lead and dk('inspect', n, '--format', '{{.Config.Hostname}}') in lead] or alive[:1]
            if vic: logc('kill -9 leader ' + vic[0]); dk('kill', '-s', 'KILL', vic[0]); kills += 1; time.sleep(2); dk('start', vic[0])
            if kills in (2, 4) and pauses < 2: logc('docker pause redis 5 s'); dk('pause', 'ep-r-redis-1'); time.sleep(5); dk('unpause', 'ep-r-redis-1'); pauses += 1
            # thao tác đồng thời trên bài đã đóng: chấm lại + sửa điểm + hold
            for e in exs[:6]:
                if db(f"select status from exams where id='{e}'") in ('PUBLISHED', 'CLOSED'):
                    g = S('GET', f'{E}/{e}')[1]; S('POST', f'{E}/{e}/regrade', b=dict(scope='ALL', reason='đua'), h=idem()); extras.append('regrade')
        time.sleep(15)
        for _ in range(60):
            if all(db(f"select status from exams where id='{e}'") == 'PUBLISHED' for e in exs): break
            time.sleep(3)
        il = "','".join(exs)
        stat = db(f"select status,count(*) from exams where id in ('{il}') group by 1").replace(chr(10), '; ')
        pub = db(f"select count(*) from exams where id in ('{il}') and published_at is not null"); stuck = db(f"select count(*) from exam_attempts where exam_id in ('{il}') and status<>'GRADED'")
        outp = db(f"select payload->>'exam_id', count(*) from outbox where topic='exam.published' and payload::text ~ '({'|'.join(exs)})' group by 1 having count(*)>1")
        dup_n = db(f"select count(*) from (select user_id, link, type, count(*) c from notifications where link ~ '({'|'.join(exs)})' and type ilike '%PUBLISH%' group by 1,2,3 having count(*)>1) x")
        n_tot = db(f"select count(*) from notifications where link ~ '({'|'.join(exs)})' and type ilike '%PUBLISH%'")
        rg_dup = db(f"select count(*) from (select user_id, link, type, count(*) c from notifications where link ~ '({'|'.join(exs)})' and type ilike '%REGRAD%' group by 1,2,3 having count(*)>1) x")
        R('TC-PE08-18', stat.startswith('PUBLISHED|20') and dup_n == '0' and outp == '', f'20 bài (khung 100–150 s), 2 worker, kill -9 leader {kills} lần (bật lại ngay): trạng thái {stat}; published_at đặt {pub}/20; lượt kẹt GRADING {stuck}; outbox exam.published trùng: {outp or "không"}; thông báo công bố {n_tot} (trùng (SV,bài): {dup_n})')
        R('TC-PE08-65', stat.startswith('PUBLISHED|20') and dup_n == '0' and rg_dup == '0' and stuck == '0', f'đồng thời: tick đóng + công bố + regrade ({len(extras)} lần gọi) + kill -9 leader ×{kills} + docker pause redis 5 s ×{pauses}: cuối cùng {stat}; published_at 1 lần; thông báo trùng (SV,bài): công bố {dup_n}, chấm lại {rg_dup}; lượt kẹt GRADING {stuck}')
finally:
    dk('unpause', 'ep-r-redis-1')
    for n in ('ep-r-postgres-1', 'ep-r-redis-1', 'ep-r-judge-1', 'ep-r-worker-1'): dk('start', n)
    if stage in ('g2', 'g4'): compose('up -d --no-recreate --scale worker=1 worker')
    time.sleep(5)
    now = {n: img(n) for n in ('ep-r-gateway-1', 'ep-r-worker-1')}
    os.rmdir(LOCK); logc(f'UNLOCK — {stage}: judge/worker/redis/postgres đã bật; image gateway/worker {now == imgs0 and "giữ nguyên" or now}; giữ {time.time()-t_lock:.0f}s')
