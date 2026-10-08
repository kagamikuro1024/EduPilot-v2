from q6lib import *
import subprocess
def rds(*a): return subprocess.run(['docker', 'exec', 'edupilot100-redis-1', 'redis-cli', *a], capture_output=True, text=True).stdout.strip()
def lock(tok): s, j, t = call('GET', '/me/exam-lock', tok); return s, j
mk = lambda o, c, d, n: mkexam(o, c, d, nq=n, title='QC7 M%d' % d)
e1, ids1, o1, c1 = mk(62, 3000, 5, 3); e2, ids2, o2, c2 = mk(62, 3000, 9, 3)
eN, idsN, oN, cN = mk(62, 300, 5, 3)   # khung 300 s → deadline = closes_at khi bắt đầu muộn
time.sleep(max(0, (o1 - dt.datetime.now(dt.timezone.utc)).total_seconds()) + 3)
U = 'sv.kha'; tok = login(U); uid = db(f"select id from users where email='{U}@edupilot.local'"); K = 'ep:exam_lock:' + uid
print('06 trước khi thi', lock(tok), '| ADMIN', lock(AD)[0], 'GV', lock(T)[0], 'ẩn danh', call('GET', '/me/exam-lock')[0])
def start(e, tk=tok):
    r = call('POST', E + '/' + e + '/attempts', tk, {}, idem()); return r[1]['attempt']['id'], r[1]['attempt']['deadline_at'], {'X-Exam-Tab': str(uuid.uuid4())}
a1, d1, H1 = start(e1)
print('01 sau bắt đầu M1: key', rds('GET', K) == a1, 'TTL', rds('TTL', K), '(kỳ vọng ≈ 300 + grace)', '| me/exam-lock', lock(tok), '| thân có attempt_id?', a1 in json.dumps(lock(tok)[1]))
a2, d2, H2 = start(e2); print('03 sau bắt đầu M2 (hạn muộn hơn): key =', 'M2' if rds('GET', K) == a2 else 'M1', 'TTL', rds('TTL', K), '(≈ 540 + grace)')
rds('DEL', K); print('07 DEL key: me/exam-lock', lock(tok), '| key nạp lại:', rds('GET', K) in (a1, a2), 'TTL', rds('TTL', K))
t0 = time.time(); call('POST', E + '/' + e2 + f'/attempts/{a2}/submit', tok, {}, {**H2, **idem()}) if False else None
# nộp M2 cần takeover trước
call('POST', E + '/' + e2 + f'/attempts/{a2}/takeover', tok, {}, H2); r = call('POST', E + '/' + e2 + f'/attempts/{a2}/submit', tok, {}, {**H2, **idem()}); print('submit M2', r[0])
for _ in range(14):
    k = rds('GET', K); ttl = rds('TTL', K)
    if k == a1: break
    time.sleep(1)
print('03 sau nộp M2 (%.0fs): key = M1: %s, TTL %s (≈ 300 + grace)' % (time.time() - t0, k == a1, ttl), '| me/exam-lock', lock(tok)[1])
# 08 Redis chết
subprocess.run(['docker', 'stop', 'edupilot100-redis-1'], capture_output=True); time.sleep(2)
res = [lock(tok) for _ in range(20)]; print('08 Redis chết ×20:', {(s, json.dumps(j)[:40]) for s, j in res})
subprocess.run(['docker', 'start', 'edupilot100-redis-1'], capture_output=True); time.sleep(8)
print('08 Redis lên lại: me/exam-lock', lock(tok), '| key', rds('GET', K) == a1, 'TTL', rds('TTL', K))
# 45 restart Redis
subprocess.run(['docker', 'restart', 'edupilot100-redis-1'], capture_output=True); time.sleep(6); print('45 restart Redis:', lock(tok), 'key', rds('GET', K) == a1)
# 02 nộp M1 → gỡ ≤ 10 s
call('POST', E + '/' + e1 + f'/attempts/{a1}/takeover', tok, {}, H1); t0 = time.time(); print('submit M1', call('POST', E + '/' + e1 + f'/attempts/{a1}/submit', tok, {}, {**H1, **idem()})[0])
while rds('EXISTS', K) != '0' and time.time() - t0 < 20: time.sleep(0.5)
print('02 key gỡ sau %.1fs ; me/exam-lock' % (time.time() - t0), lock(tok))
# 04 gia hạn: SV mới bắt đầu muộn (deadline = closes_at) rồi extend
tk2 = login('sv.nguyco'); uid2 = db(f"select id from users where email='sv.nguyco@edupilot.local'"); K2 = 'ep:exam_lock:' + uid2; aN, dN, HN = start(eN, tk2)
ttl0 = int(rds('TTL', K2)); new = (cN + dt.timedelta(minutes=10)).strftime('%Y-%m-%dT%H:%M:%SZ'); print('04 deadline', dN, 'closes', cN.strftime('%H:%M:%S'), 'TTL trước', ttl0); r = call('POST', E + '/' + eN + '/extend', T, dict(closes_at=new)); print('extend', r[0], codes(r[1]))
time.sleep(3); print('04 TTL sau extend:', rds('TTL', K2), '(dài hơn ≈ +600?)', '| deadline mới', call('GET', E + '/' + eN + '/attempts/mine', tk2)[1]['attempt']['deadline_at'])
# ---- events (13..16,19..21) trên lượt aN
EV = E + '/' + eN + f'/attempts/{aN}/events'; B = lambda evs, t=tk2: call('POST', EV, t, dict(events=evs))
print('13', B([dict(type='PASTE', meta=dict(chars=812, clipboard='SECRET', ip='1.2.3.4')), dict(type='TAB_HIDDEN'), dict(type='TAB_VISIBLE', meta=dict(duration_ms=4000)), dict(type='OFFLINE'), dict(type='ONLINE'), dict(type='FOO'), dict(type='TAB_TAKEOVER'), dict(type='PASTE', meta=dict(chars=-5)), dict(type='PASTE', meta=dict(chars='abc')), dict(type='PASTE', meta=dict(chars=1.5))])[0:1])
print('13 DB:', db(f"select type,meta::text from exam_events where attempt_id='{aN}' order by at"))
print('14 51 sự kiện', B([dict(type='TAB_HIDDEN')] * 51)[0], '→ thêm', db(f"select count(*) from exam_events where attempt_id='{aN}'"), '| meta 301 byte', B([dict(type='PASTE', meta=dict(item_id=str(uuid.uuid4()), chars=1, duration_ms=1, x='y' * 300))])[0])
for i in range(12): B([dict(type='TAB_HIDDEN')] * 50)
print('14 600 sự kiện → DB', db(f"select count(*) from exam_events where attempt_id='{aN}'"), 'dropped Redis', rds('GET', 'ep:exam:events:dropped:' + aN))
print('15 người khác', call('POST', EV, tok, dict(events=[dict(type='TAB_HIDDEN')]))[0], '| ẩn danh', call('POST', EV, None, dict(events=[]))[0])
print('19 GV', call('GET', E + '/' + eN + f'/events?attempt={aN}', T)[0], 'TA', call('GET', E + '/' + eN + f'/events?attempt={aN}', TA)[0], 'SV chính chủ', call('GET', E + '/' + eN + f'/events?attempt={aN}', tk2)[0], 'SVB', call('GET', E + '/' + eN + f'/events?attempt={aN}', tok)[0], 'ADMIN', call('GET', E + '/' + eN + f'/events?attempt={aN}', AD)[0])
s, j, t = call('GET', E + '/' + eN + f'/events?attempt={aN}&limit=3', T); print('19 GV thân:', json.dumps(j, ensure_ascii=False)[:420])
print('16 DB cột:', db("select column_name from information_schema.columns where table_name='exam_events' order by ordinal_position").replace('\n', ','), '|', db("select column_name from information_schema.columns where table_name='similarity_reports' order by ordinal_position").replace('\n', ','))
print('16 canary SECRET/1.2.3.4 trong DB:', db("select count(*) from exam_events where meta::text like '%SECRET%' or meta::text like '%1.2.3.4%'"))
subprocess.run('docker logs edupilot100-gateway-1 2>&1 | grep -c "SECRET\\|1.2.3.4"', shell=True)
print('16 log gateway chứa SECRET:', subprocess.run('docker logs edupilot100-gateway-1 2>&1 | grep -c SECRET', shell=True, capture_output=True, text=True).stdout.strip())
print('15 lượt đã nộp bỏ qua:', end=' '); call('POST', E + '/' + e1 + f'/attempts/{a1}/events', tok, dict(events=[dict(type='TAB_HIDDEN')])); print(call('POST', E + '/' + e1 + f'/attempts/{a1}/events', tok, dict(events=[dict(type='TAB_HIDDEN')]))[0], db(f"select count(*) from exam_events where attempt_id='{a1}'"))
