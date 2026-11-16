"""QC2 smoke sprint 5 (SmkM) — TC-PE08-44: xuất CSV 5.001 / 5.000 dòng, RAM gateway, quyền, log, audit. Chạy: python3 s20i_csv.py"""
import os, sys, threading; os.environ['SUITE'] = 'BULK'
from r5lib import *
import rl
bk = json.load(open('/tmp/qc2L/bulk.json')); e = bk['exam']; C4 = bk['c4']; EX = f'/courses/{C4}/exams/{e}'
def mem(): 
    o = subprocess.run(['docker', 'stats', '--no-stream', '--format', '{{.MemUsage}}', 'ep-r-gateway-1'], capture_output=True, text=True).stdout.strip().split('/')[0].strip(); return o
def mb(s):
    m = re.match(r'([\d.]+)(MiB|GiB|KiB)', s); v = float(m.group(1)); return v * (1024 if m.group(2) == 'GiB' else 1 if m.group(2) == 'MiB' else 1 / 1024)
def fetch(tok):
    t0 = time.time(); s, j, t = call('GET', EX + '/results.csv', tok, timeout=300); return s, j, t, time.time() - t0
n_active = db(f"select count(*) from enrollments where course_id='{C4}' and role_in_course='STUDENT' and status='ACTIVE'")
peak = [mb(mem())]; stop = [False]
def poll():
    while not stop[0]: peak.append(mb(mem())); time.sleep(0.5)
base = mb(mem()); th = threading.Thread(target=poll); th.start()
s1, j1, t1, d1 = fetch(T); stop[0] = True; th.join(); lines1 = [l for l in t1.split('\n') if l.strip()]
print('5001:', s1, codes(j1), len(lines1) - 1, round(d1, 1), 'peak', max(peak), 'base', base)
# giảm còn 5.000 SV hoạt động: đặt 1 SV bulk REMOVED
psql = lambda q: subprocess.run(['docker', 'exec', 'ep-r-postgres-1', 'psql', '-U', 'edupilot', '-d', 'edupilot', '-At', '-c', q], capture_output=True, text=True)
uid = db(f"select user_id from enrollments e join users u on u.id=e.user_id where e.course_id='{C4}' and u.email='qc2m5-bulk-05000@edupilot.local'")
r = psql(f"update enrollments set status='REMOVED', removed_at=now(), previous_status='ACTIVE' where course_id='{C4}' and user_id='{uid}'")
n2 = db(f"select count(*) from enrollments where course_id='{C4}' and role_in_course='STUDENT' and status='ACTIVE'")
peak = [mb(mem())]; stop = [False]; th = threading.Thread(target=poll); th.start()
s2, j2, t2, d2 = fetch(T); stop[0] = True; th.join(); lines2 = [l for l in t2.split('\n') if l.strip()]
print('5000:', s2, codes(j2), len(lines2) - 1, round(d2, 1), 'peak', max(peak))
roles = {k: call('GET', EX + '/results.csv', t)[0] for k, t in (('TA', TA), ('SV', login('r5x')), ('ADMIN', AD))}
names = [l.split(';')[1] for l in lines2[1:4]]
lg = subprocess.run('docker logs --since 15m ep-r-gateway-1 2>&1', shell=True, capture_output=True, text=True).stdout; named = [n for n in names if n in lg]
aud = db("select count(*) from audit_log where created_at > now() - interval '15 minutes' and (action ilike '%csv%' or action ilike '%export%')")
psql(f"update enrollments set status='ACTIVE', removed_at=null, previous_status=null where course_id='{C4}' and user_id='{uid}'")
rl.R('TC-PE08-44', s2 == 200 and len(lines2) - 1 == 5000 and (s1 in (409, 422) or len(lines1) - 1 == 5000) and roles == {'TA': 200, 'SV': 403, 'ADMIN': 403} and not named and aud == '0' and max(peak) - base < 120, f'lớp {n_active} SV hoạt động → xuất CSV: {s1} {codes(j1) or len(lines1) - 1} dòng ({d1:.1f} s); giảm còn {n2} SV → {s2}, {len(lines2) - 1} dòng dữ liệu ({d2:.1f} s); RAM gateway nền {base:.0f} MiB, đỉnh khi xuất 5.000 dòng {max(peak):.0f} MiB (tăng {max(peak) - base:.0f} MiB; stream); GV/TA 200, SV/ADMIN {roles["SV"]}/{roles["ADMIN"]}; log gateway chứa tên SV trong CSV: {named or "không"}; audit_log (đọc) mới: {aud}')
