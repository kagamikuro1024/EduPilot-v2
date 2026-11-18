import sys, json, datetime, subprocess
sys.path.insert(0, '.'); from q4lib import *
T_ = L('teacher'); TA = L('ta'); SV = L('sv.gioi'); AD = L('admin'); G2 = L('gv.s8f')
cid, cc, code = new_course('g1'); settings(cid)
psql(f"insert into enrollments(course_id,user_id,role_in_course,status,joined_via) select '{cid}',id,'TA','ACTIVE','ADMIN' from users where email='ta@edupilot.local' on conflict do nothing")
d0 = datetime.date.today() + datetime.timedelta(days=30)
while d0.isoweekday() != 1: d0 += datetime.timedelta(days=1)
d1 = d0 + datetime.timedelta(days=7 * 3 + 6)   # 4 tuần
def gen(tok, key=None, **kw):
    body = dict({'weekdays': [1, 3], 'start_time': '08:00', 'end_time': '10:00', 'room': 'B1', 'from': d0.isoformat(), 'to': d1.isoformat(), 'exclude_dates': []}, **kw)
    return call('POST', f'/courses/{cid}/sessions/generate', tok, body, hdr={'Idempotency-Key': key or str(uuid.uuid4())})
ex = (d0 + datetime.timedelta(days=2)).isoformat()      # thứ 4 tuần đầu
r1 = gen(T_, exclude_dates=[ex]); n1 = psql(f"select count(*) from class_sessions where course_id='{cid}'")
nos = psql(f"select string_agg(session_no::text, ',' order by session_no) from class_sessions where course_id='{cid}'"); has_ex = psql(f"select count(*) from class_sessions where course_id='{cid}' and (starts_at at time zone 'Asia/Ho_Chi_Minh')::date='{ex}'")
r2 = gen(T_, exclude_dates=[ex]); n2 = psql(f"select count(*) from class_sessions where course_id='{cid}'"); dup = psql(f"select count(*) - count(distinct starts_at) from class_sessions where course_id='{cid}'")
big = call('POST', f'/courses/{cid}/sessions/generate', T_, {'weekdays': [1, 2, 3, 4, 5, 6, 7], 'start_time': '08:00', 'end_time': '10:00', 'from': (d0 + datetime.timedelta(days=60)).isoformat(), 'to': (d0 + datetime.timedelta(days=120)).isoformat(), 'exclude_dates': []}, hdr={'Idempotency-Key': str(uuid.uuid4())})
r_ta = gen(TA, weekdays=[5], start_time='14:00', end_time='15:00'); r_sv = gen(SV); r_g2 = gen(G2); r_ad = gen(AD, weekdays=[6], start_time='07:00', end_time='08:00')
C('TC-P212-09', r1[0] == 201 and r1[1].get('created') == 7 and n1 == '7' and has_ex == '0' and r2[1].get('created') == 0 and n2 == '7' and dup == '0' and big[0] == 422 and r_ta[0] == 201 and r_sv[0] == 403 and r_g2[0] == 403,
  f'4 tuần, thứ 2+4, loại trừ 1 ngày: {r1[0]} created {r1[1].get("created")} (session_no {nos}); ngày loại trừ có buổi: {has_ex}; chạy lại: created {r2[1].get("created")}, tổng {n2}, trùng starts_at {dup}; >60 buổi → {big[0]}; TA → {r_ta[0]}; SV → {r_sv[0]}; GV lớp khác → {r_g2[0]}; ADMIN → {r_ad[0]} (ghi hành vi: {"được" if r_ad[0] == 201 else "không"})')
# ---- 19: compose test không ghi đè giới hạn
n19 = subprocess.run("cd ~/Documents/TA_Agent_qc2 && grep -nE 'RATE_LIMIT_IP_PER_MIN|AUTH_(LOGIN|REGISTER|TOKEN)' docker-compose.test.yml | wc -l", shell=True, capture_output=True, text=True).stdout.strip()
n19b = subprocess.run("cd ~/Documents/TA_Agent_qc2 && grep -nE 'RATE_LIMIT_IP_PER_MIN|AUTH_(LOGIN|REGISTER|TOKEN)' docker-compose.local.yml docker-compose.yml 2>/dev/null | head -4", shell=True, capture_output=True, text=True).stdout.strip()
print('19:', n19, n19b)
