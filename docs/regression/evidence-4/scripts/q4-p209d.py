# P209-14 trên stack qc2i (gw1 :18221, gw2 :18222, redis qc2i-redis) — không đụng ep-r
import sys, json, time, subprocess, uuid
sys.path.insert(0, '.'); import q4lib as q
exec(open('qc2i-r4apia-lib.py').read().split("R={}")[0])
A = admin_tok(); gv_em, gv_id = invite('p209gv'); s, j, _ = call('POST', '/admin/courses', A, {'subject_code': 'INT1006', 'class_code': 'QP209' + TAG[:4].upper(), 'name': 'Lớp P209-14', 'semester': '2026-2027-HK1', 'teacher_id': gv_id}, base=G1)
if s != 201: s, j, _ = call('POST', '/admin/courses', A, {'subject_code': 'INT1006', 'class_code': 'QP209' + TAG[:4].upper(), 'name': 'Lớp P209-14', 'semester': '2026-2027-HK1'}, base=G1)
cid = j['id']; code = psql(f"select join_code from courses where id='{cid}'")
sv = reg('p209sv'); resetrl()
def tok(base): s, j, _ = call('POST', '/auth/login', None, {'email': sv, 'password': PWD}, base=base); assert s == 200, (s, j); return j['access_token']
t1, t2 = tok(G1), tok(G2)
def jn(t, c, base): return call('POST', '/courses/join', t, {'code': c}, base=base, retry=False)
for k in rds('--scan', '--pattern', 'ep:join:fail:*').split(): rds('del', k)
seq = [jn(t1, 'WRONG21', G1)[0], jn(t2, 'WRONG22', G2)[0], jn(t1, 'WRONG23', G1)[0], jn(t2, 'WRONG24', G2)[0], jn(t1, 'WRONG25', G1)[0]]
r6a = jn(t2, 'WRONG26', G2)[0]; r6b = jn(t1, 'WRONG27', G1)[0]
uid = psql(f"select id from users where email='{sv}'"); cnt = rds('zcard', 'ep:join:fail:user:' + uid)
subprocess.run(['docker', 'stop', 'qc2i-redis'], capture_output=True)
try:
    time.sleep(1); t0 = time.time(); down = [call('POST', '/courses/join', t1, {'code': 'REDISDWN%d' % i}, base=G1, retry=False, idem=False)[0] for i in range(7)]; dt = time.time() - t0
    keyed = call('POST', '/courses/join', t1, {'code': 'REDISKEY'}, base=G1, retry=False)
    lg_ = subprocess.run('docker logs qc2i-gw1 2>&1; docker logs qc2i-gw2 2>&1', shell=True, capture_output=True, text=True).stdout
finally:
    subprocess.run(['docker', 'start', 'qc2i-redis'], capture_output=True); time.sleep(3)
tried = ['WRONG2%d' % i for i in range(1, 8)] + ['REDISDWN%d' % i for i in range(7)]
leak = sum(lg_.count(c) for c in tried); errlines = [l for l in lg_.splitlines() if '"level":"ERROR"' in l and ('join' in l.lower() or 'guess' in l.lower() or 'redis' in l.lower())]
print(seq, r6a, r6b, cnt, down, len(errlines), leak, errlines[:1])
q.C('TC-P209-14', seq == [404] * 5 and r6a == 429 and r6b == 429 and cnt == '5' and set(down) == {404} and len(errlines) >= 1 and leak == 0,
  f'2 gateway dùng chung Redis (qc2i-gw1 :18221 / qc2i-gw2 :18222, DB qc2i_auth), xen kẽ 5 mã sai ({seq}) → lần 6 ở gw2 {r6a} và ở gw1 {r6b}; ZCARD ep:join:fail:user = {cnt} (đếm chung); dừng qc2i-redis: 7 lần thử sai (không Idempotency-Key, như UI join) → {sorted(set(down))} (mở cửa, {dt:.1f} s); ghi nhận: cùng lời gọi có Idempotency-Key → {keyed[0]} {keyed[1].get("code")} details {keyed[1].get("details")} (bộ chống trùng yêu cầu cần Redis, đóng cửa theo thiết kế); log: {len(errlines)} dòng ERROR về giới hạn/redis; số lần mã đã thử xuất hiện trong log = {leak}')
