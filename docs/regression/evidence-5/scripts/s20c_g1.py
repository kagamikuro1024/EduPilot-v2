"""QC2 smoke sprint 5 (SmkL) — cửa sổ khoá toàn cục (mkdir /tmp/qc2-lock-global; ghi /tmp/qc2-global-changes.log; hoàn lại cấu hình mặc định).
 g1: judge dừng (TC-PE02-15/26/31/52/53, PE06-19 phần 503, PE08-06)         ~4 phút
 g2: 2 worker, leader, tắt worker qua hạn (TC-PE04-24/25/26, PE05-31, PE02-25 kill -9)   ~5 phút
 g4: 2 worker + kill -9 leader liên tục + Redis pause khi 20 bài đóng/công bố (TC-PE08-18/65)
 g3: Redis dừng / Redis+Postgres dừng (TC-PE07-08/09/45)                   ~1 phút
s20c_g1: bản chạy lại g1 (SmkM): readyz đọc ở /readyz CỦA WORKER :8081 (judge up|down|off), không phải gateway."""
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
stage = 'g1'
ACT = [x.split('@')[0] for x in db(f"select u.email from enrollments e join users u on u.id=e.user_id where e.course_id='{C1}' and u.email like 'r5s%' and e.status='ACTIVE' order by 1").splitlines() if int(x[3:5]) not in (13, 14)]
def ctxs(eid, users):
    out = {}
    for u in users:
        aid, j, H, s, _ = start(eid, u); assert aid, (u, s, _)
        out[u] = dict(t=login(u), aid=aid, H=H, items=j['items'], base=f'{E}/{eid}/attempts/{aid}')
    return out
def codeitem(c): return [i for i in c['items'] if i['type'] == 'CODE'][0]['item_id']
def readyz():
    r = subprocess.run(['docker', 'exec', 'qc2-s5-curl', 'curl', '-s', '-m', '3', '-w', ' HTTP%{http_code}', 'http://ep-r-worker-1:8081/readyz'], capture_output=True, text=True).stdout
    return (200 if r.endswith('HTTP200') else 0), r
def hzw():
    r = subprocess.run(['docker', 'exec', 'qc2-s5-curl', 'curl', '-s', '-m', '3', '-o', '/dev/null', '-w', '%{http_code}', 'http://ep-r-worker-1:8081/healthz'], capture_output=True, text=True).stdout
    return int(r or 0)
t_w = time.time()
while True:
    try: os.mkdir(LOCK); break
    except FileExistsError:
        if time.time() - t_w > 1500: print('không lấy được khoá'); sys.exit(1)
        time.sleep(15)
t_lock = time.time(); logc(f'LOCK — {stage}: ' + {'g1': 'dừng judge ~4 phút (TC sprint 5 PE-02/06/08)', 'g2': 'scale worker=2, kill -9 / dừng worker ~3 phút (TC PE-04/05/02)', 'g3': 'dừng redis ~10 s và redis+postgres ~8 s (TC PE-07)', 'g4': 'scale worker=2, kill -9 leader nhiều lần, docker pause redis ~5 s ×2 (TC PE-08)'}[stage] + '; hoàn lại ≤ 8 phút')
imgs0 = {n: img(n) for n in ('ep-r-gateway-1', 'ep-r-worker-1')}
subprocess.run(['docker', 'rm', '-f', 'qc2-s5-curl'], capture_output=True); dk('run', '-d', '--rm', '--name', 'qc2-s5-curl', '--network', 'ep-r_default', '--entrypoint', 'sleep', 'curlimages/curl', '1800')
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
        rz = readyz(); hz = (hzw(), ''); hzg = call('GET', '/healthz')[0]
        # tiến trình thử lại tới dead-letter
        prog = []; last = None; dead0 = rds('XLEN', 'judge.submit.dead'); dead_t = None
        for _ in range(110):
            row = db(f"select status,coalesce(verdict::text,''),attempts,fail_count from code_submissions where id='{sid}'")
            if row != last: prog.append((round(time.time() - t0, 1), row)); last = row
            if row.startswith('ERROR'): dead_t = time.time() - t0; break
            time.sleep(1)
        dead1 = rds('XLEN', 'judge.submit.dead'); msg = subprocess.run(['docker', 'exec', 'ep-r-redis-1', 'redis-cli', 'XREVRANGE', 'judge.submit.dead', '+', '-', 'COUNT', '1'], capture_output=True, text=True).stdout
        warn = int(subprocess.run("docker logs --since %ds ep-r-worker-1 2>&1 | grep -c 'máy chấm không phản hồi quá 60 giây'" % (int(time.time() - t0) + 2), shell=True, capture_output=True, text=True).stdout.strip() or 0)
        R('TC-PE02-31', t_down is not None and r_run[0] == 503 and 'JUDGE_UNAVAILABLE' in codes(r_run[1]) and d_run <= 3 and 5 <= (r_run[1].get('retry_after') or 0) <= 30 and r_sub[0] == 202, f'judge dừng (readyz báo judge:down sau {t_down and round(t_down,1)} s): Chạy thử {r_run[0]} {codes(r_run[1])} retry_after={r_run[1].get("retry_after")} sau {d_run:.2f}s (≤3); nộp {r_sub[0]} QUEUED ({d_sub:.2f}s)')
        R('TC-PE06-19', r_run[0] == 503 and d_run <= 3, f'(phần judge chết) run → 503 JUDGE_UNAVAILABLE retry_after={r_run[1].get("retry_after")} trong {d_run:.2f}s; phần hàng đầy + run ≤5s: TC-PE02-27 (s13_judge.py burst)')
        ok52a = hz[0] == 200 and hzg == 200 and rz[0] == 200 and '"judge":"down"' in rz[1].replace(' ', '') and warn == 1; note52a = f'judge dừng: worker healthz {hz[0]} (gateway healthz {hzg}); worker readyz HTTP {rz[0]} {rz[1].strip()[:80]} (judge báo down sau {t_down and round(t_down,1)} s); log warn "máy chấm không phản hồi quá 60 giây" {warn} dòng trong {int(time.time()-t0)} s (TC: đúng 1)'
        sj = json.dumps(prog)
        R('TC-PE02-26', dead_t is not None and dead1 == str(int(dead0) + 1) and 'package main' not in msg and '#include' not in msg, f'judge dừng, worker chạy: tiến trình thử lại (giây, status|verdict|attempts|fail_count) = {sj}; dead-letter sau {dead_t and round(dead_t,1)} s; judge.submit.dead {dead0}→{dead1} (1 tin), tin cuối có mã nguồn: {"#include" in msg}')
        R('TC-PE02-15', dead_t is not None and prog[-1][1].startswith('ERROR|IE'), f'judge lỗi giữa chấm: verdict cuối = {prog[-1][1] if prog else None} (IE, không RE/WA, không điểm 0 — lượt ở GRADING, xem PE08-06)')
        # chờ bài đóng + lượt vào GRADING; kiểm Today
        sleep_until(dt.datetime.fromisoformat(db(f"select to_char(closes_at at time zone 'utc','YYYY-MM-DD\"T\"HH24:MI:SS+00:00') from exams where id='{eA}'")) + dt.timedelta(seconds=25))
        st2 = db(f"select status,submit_reason from exam_attempts where id='{a2['aid']}'"); has = lambda t: 'EXAM_GRADE_ERROR' in t and eA in t
        for _ in range(60):
            tg = S('GET', '/me/today', T)[2]; tta = S('GET', '/me/today', TA)[2]
            if has(tg) and has(tta): break
            time.sleep(2)
        t_today = round(time.time() - t0)
        R('TC-PE02-53', has(tg) and has(tta), f'judge down > 60 s, có bài chờ chấm (IE): Today GV EXAM_GRADE_ERROR của bài này={has(tg)}, TA={has(tta)} (chờ tối đa 120 s vì bộ gửi outbox có thể trễ khi máy tải)')
        time.sleep(40); ex_st = db(f"select status,published_at is not null from exams where id='{eA}'"); st2b = db(f"select status from exam_attempts where id='{a2['aid']}'"); nn = db(f"select count(*) from notifications where link like '%{eA}%'")
        R('TC-PE08-20', st2b == 'GRADING' and ex_st.startswith('CLOSED|f'), f'1 lượt kẹt GRADING (IE, judge dừng): sau đóng +65 s bài ở {ex_st} (CLOSED, published_at null — không công bố khi còn GRADING), lượt {st2b}; thông báo công bố đã gửi: {nn}')
        t_up = time.time(); logc('docker start judge'); dk('start', 'ep-r-judge-1')
        for _ in range(60):
            if '"judge":"up"' in readyz()[1].replace(' ', ''): break
            time.sleep(1)
        up_t = time.time() - t_up; rz2 = readyz()
        R('TC-PE02-52', ok52a and rz2[0] == 200 and up_t <= 10 + 10 and '"judge":"up"' in rz2[1].replace(' ', ''), note52a + f'; bật lại judge: worker readyz judge=up sau {up_t:.1f}s (TC ≤10 s cộng nhịp kiểm 10 s của máy chủ)')
        rg = S('POST', f'{E}/{eA}/regrade', b=dict(scope='errors', reason='judge lên lại'), h=idem()); time.sleep(1)
        for _ in range(90):
            st3 = db(f"select status,auto_score from exam_attempts where id='{a2['aid']}'"); sr = db(f"select status,verdict from code_submissions where id='{sid}'")
            if st3.startswith('GRADED'): break
            time.sleep(2)
        R('TC-PE08-06', st2.startswith('GRADING') and has(tg) and rg[0] == 202 and st3.startswith('GRADED'), f'dead-letter IE: lượt ở {st2} (không GRADED, không 0 điểm); việc EXAM_GRADE_ERROR cho GV+TA; judge lên + regrade scope=errors → {rg[0]}; lượt {st3}, bài nộp {sr}')
finally:
    dk('unpause', 'ep-r-redis-1')
    for n in ('ep-r-postgres-1', 'ep-r-redis-1', 'ep-r-judge-1', 'ep-r-worker-1'): dk('start', n)
    dk('rm', '-f', 'qc2-s5-curl')
    time.sleep(5)
    now = {n: img(n) for n in ('ep-r-gateway-1', 'ep-r-worker-1')}
    os.rmdir(LOCK); logc(f'UNLOCK — {stage}: judge/worker/redis/postgres đã bật; image gateway/worker {now == imgs0 and "giữ nguyên" or now}; giữ {time.time()-t_lock:.0f}s')
