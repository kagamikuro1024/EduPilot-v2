"""TC-PE02-25: kill -9 worker ep-r khi đang chấm bài 10 test (≈1 s/test); bật lại; kiểm đúng MỘT kết quả cuối (= lượt đối chứng không kill). Cần khoá toàn cục (script tự lấy/nhả)."""
import sys, os; sys.path.insert(0, os.path.dirname(__file__))
from r5lib import *
import rl3 as rl
LOG = '/tmp/qc2-global-changes.log'
def glog(m): open(LOG, 'a').write(time.strftime('%H:%M:%S') + ' qc3-s5: ' + m + '\n')
SLOW = '#include <stdio.h>\nint main(){long long a,b;scanf("%lld %lld",&a,&b);volatile long long x=0;for(long long i=0;i<400000000LL;i++)x+=i;printf("%lld\\n",a+b);return 0;}\n'
STU = SLOW.replace('printf("%lld\\n",a+b)', 'if(a<10)printf("%lld\\n",a+b);else printf("%lld\\n",a+b+1)')
tests = [('t%02d' % k, '%d %d\n' % (k, k + 1), '%d\n' % (2 * k + 1), k <= 2, 1) for k in range(1, 11)]
qid = mkcode(250, tests=tests, ref=SLOW, tl=4000)
e, _ = mkexam(open_s=10, win=1500, dur=20, items=[qid], title='R5 PE02-25 kill worker'); time.sleep(12)
def sub(u):
    aid, jj, H, s0, _ = start(e, u); it = [i for i in jj['items'] if i['type'] == 'CODE'][0]; t = login(u)
    s, j, _ = call('POST', f'{E}/{e}/attempts/{aid}/code/{it["item_id"]}/submit', t, dict(language='cpp17', source=STU), {**H, **idem()})
    return aid, H, j.get('submission_id'), s
def row(sid): return db(f"select status||'|'||coalesce(verdict::text,'')||'|'||coalesce(passed_weight::text,'')||'/'||coalesce(total_weight::text,'')||'|fc='||fail_count||'|att='||attempts||'|'||coalesce(jsonb_array_length(results)::text,'0') from code_submissions where id='{sid}'")
# đối chứng
a0, H0, s0, st0 = sub('r5s36'); t0 = time.time(); seq0 = []
for _ in range(200):
    r = row(s0)
    if not seq0 or seq0[-1][1] != r: seq0.append((round(time.time() - t0, 1), r))
    if r.startswith('DONE'): break
    time.sleep(0.5)
ctrl = row(s0); dur0 = time.time() - t0
print('đối chứng', st0, ctrl, round(dur0, 1), seq0)
# kill
a1, H1, s1, st1 = sub('r5s37'); t1 = time.time()
for _ in range(120):
    if row(s1).startswith('RUNNING'): break
    time.sleep(0.2)
time.sleep(3)
lk = '/tmp/qc2-lock-global'
for i in range(60):
    if os.system(f'mkdir {lk} 2>/dev/null') == 0: break
    time.sleep(10)
else: raise SystemExit('không lấy được khoá')
glog('LOCK — PE02-25: docker kill ep-r-worker-1 giữa lúc chấm bài 10 test, rồi docker start (≤5 phút)')
try:
    pre = row(s1); lease0 = db(f"select lease_until from code_submissions where id='{s1}'")
    tk = time.time(); k = subprocess.run(['docker', 'kill', 'ep-r-worker-1'], capture_output=True, text=True).stdout.strip()
    time.sleep(8); mid = row(s1); lease_mid = db(f"select lease_until from code_submissions where id='{s1}'")
    ts = time.time(); subprocess.run(['docker', 'start', 'ep-r-worker-1'], capture_output=True, text=True)
    seq = []
    for _ in range(600):
        r = row(s1)
        if not seq or seq[-1][1] != r: seq.append((round(time.time() - ts, 1), r))
        if r.startswith('DONE'): break
        time.sleep(0.5)
    fin = row(s1); rec_s = time.time() - ts
    time.sleep(35)  # đợi thêm: không có kết quả thứ hai
    fin2 = row(s1)
finally:
    glog('UNLOCK — PE02-25: worker đã docker start, khoá nhả'); os.system(f'rmdir {lk}')
n = db(f"select count(*) from code_submissions where attempt_id='{a1}'"); n0 = db(f"select count(*) from code_submissions where attempt_id='{a0}'")
res_a = db(f"select results::text from code_submissions where id='{s0}'"); res_b = db(f"select results::text from code_submissions where id='{s1}'")
strip = lambda t: [(x.get('name'), x.get('verdict') or x.get('status'), x.get('passed')) for x in json.loads(t)] if t else None
same = strip(res_a) == strip(res_b)
for u, a, H in (('r5s36', a0, H0), ('r5s37', a1, H1)): submit_exam(e, a, u, H, E)
for _ in range(120):
    if db(f"select count(*) from exam_attempts where id in ('{a0}','{a1}') and status='GRADED'") == '2': break
    time.sleep(1)
sc0, sc1 = db(f"select auto_score from exam_attempts where id='{a0}'"), db(f"select auto_score from exam_attempts where id='{a1}'")
wst = subprocess.run(['docker', 'inspect', 'ep-r-worker-1', '--format', '{{.State.Health.Status}}'], capture_output=True, text=True).stdout.strip()
ok = fin.startswith('DONE') and fin == fin2 and fin.split('|')[2:3] == ctrl.split('|')[2:3] and n == '1' and same and sc0 == sc1 and mid.startswith('RUNNING')
msg = (f'bài 1 câu code, 10 test ≈{dur0/10:.1f} s/test (đối chứng chấm trọn {dur0:.0f} s: {ctrl}); lượt thử r5s37: kill -9 worker lúc {pre} (hết thuê {lease0}); sau kill 8 s vẫn {mid} (không ai chấm); docker start → chuỗi (giây, trạng thái) {seq}; '
       f'chấm lại xong sau {rec_s:.0f} s kể từ khi bật worker (thuê 30 s + nhịp tick 5 s; JUDGE_CLAIM_IDLE=10 phút chỉ là ngưỡng đưa lại tín hiệu Stream mất tin, không chặn); kết quả cuối {fin}, sau 35 s {fin2}; số dòng code_submissions của lượt: {n} (đối chứng {n0}); '
       f'kết quả từng test trùng đối chứng: {same}; điểm tự động sau nộp bài thi: đối chứng {sc0} = lượt kill {sc1}; worker health sau: {wst}')
rec('TC-PE02-25', ok, msg); rl.R('TC-PE02-25', 'P' if ok else 'F', msg)
