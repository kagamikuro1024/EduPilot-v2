"""S4X: DEF-R7-02 (TC-P4-02-14) chạy lại trên ep-r. Lớp mới S4X7, GV s4x7t, SV s4x7s01/02. Chạy: python3 p402-14.py (cwd bất kỳ)."""
import sys, os
os.makedirs('/tmp/qc3e', exist_ok=True)
sys.path.insert(0, os.path.expanduser('~/Documents/TA_Agent_qc2/docs/regression/evidence-7/scripts-r3'))
from lib3 import *
mkusers('s4x7t', 'TEACHER', 1); mkusers('s4x7s', 'STUDENT', 2)
cid = mkcourse('S4X7', 's4x7t', [], 'S4X7JJK')
for u in ('s4x7s01', 's4x7s02'):
    if not psql(f"select 1 from enrollments where course_id='{cid}' and user_id='{uid(u)}'"): A(u, 'POST', '/courses/join', {'code': 'S4X7JJK'})
T = 's4x7t'; tu = uid(T); Q = f'/courses/{cid}'
def rows(): return psql(f"select id||'|'||coalesce(read_at::text,'-')||'|'||title from notifications where user_id='{tu}' and type='TICKET_NEW' and course_id='{cid}' order by created_at").split('\n')
def ticket(u):
    _, _, am = mkchat(u, cid); s, d = esc(u, am); assert s == 201, (s, d); return d['ticket']['id']
def clear_open():
    for t in psql(f"select id from escalation_tickets where course_id='{cid}' and status in ('OPEN','CLAIMED')").split('\n'):
        if t: A(T, 'POST', f'{Q}/tickets/{t}/claim'); A(T, 'POST', f'{Q}/tickets/{t}/dismiss', {'reason': 'OUT_OF_SCOPE'})
c = SSEConn(T); time.sleep(1)
t1 = ticket('s4x7s01'); time.sleep(6); r = rows(); print('1) sau ticket 1:', r)
link = psql(f"select link from notifications where id='{r[-1].split('|')[0]}'"); print('read', A(T, 'POST', '/notifications/read', {'link': link})); time.sleep(1)
s0 = rows()[-1]; print('2) đã đọc:', s0)
clear_open(); time.sleep(6); s1 = rows()[-1]; print('3) sau dismiss (OPEN=0):', s1, 'OPEN=', psql(f"select count(*) from escalation_tickets where course_id='{cid}' and status in ('OPEN','CLAIMED')"))
nf0 = len([x for x in c.frames if x[2] == 'notification'])
t2 = ticket('s4x7s02'); time.sleep(8); s2 = rows()[-1]; nf1 = len([x for x in c.frames if x[2] == 'notification']); print('4) sau ticket mới (n=1, chờ 8 s):', s2, 'khung SSE notification mới =', nf1 - nf0)
st2 = psql(f"select status from escalation_tickets where id='{t2}'")
ok = s2.split('|')[1] == '-' and nf1 - nf0 >= 1 and st2 == 'OPEN' and len(rows()) == 1 and s1.split('|')[1] != '-' and s1.split('|')[2].startswith('0 ')
print(('P' if ok else 'F'), 'TC-P4-02-14', f'lớp S4X7 (GV s4x7t, SV s4x7s01/02): dòng gộp đã đọc (n=1) → dismiss (n=0): read_at={s1.split("|")[1][:19]} tiêu đề="{s1.split("|")[2]}"; ticket mới (n=1) sau 8 s: read_at={s2.split("|")[1]} (- = chưa đọc), tiêu đề="{s2.split("|")[2]}", khung SSE notification mới={nf1-nf0}, ticket {st2}, 1 dòng TICKET_NEW')
c.close()
