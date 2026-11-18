from lib3 import *
import datetime as dt
A_=C7['A']; H=C7['H']; Q=f'/courses/{H}'
for u in ['qc3s06','qc3t','qc3ta']: login(u)
clearopen('qc3t',A_)
am=mkchat('qc3s06',A_)[2]
S=psql(f"select id from class_sessions where course_id='{H}' and session_no=110"); G=f'{Q}/sessions/{S}/attendance'
g=A('qc3t','GET',G)[1]; rows=[{'student_id':r['id'],'status':'ABSENT' if r['record']['status']=='PRESENT' else 'PRESENT','version':r['record']['version']} for r in g['students'][:5]]
before=psql(f"select count(*),sum(version),md5(string_agg(status::text||version,',' order by student_id)) from attendance_records where session_id='{S}'")
ns0=psql(f"select count(*) from class_sessions where course_id='{H}'")
today=dt.date.today(); B={'weekdays':[1],'start_time':'07:00','end_time':'08:00','from':(today+dt.timedelta(days=60)).isoformat(),'to':(today+dt.timedelta(days=61+0)).isoformat()}
B={'weekdays':[(today+dt.timedelta(days=45)).isoweekday()],'start_time':'07:00','end_time':'08:00','from':(today+dt.timedelta(days=45)).isoformat(),'to':(today+dt.timedelta(days=46)).isoformat()}
n_t0=psql(f"select count(*) from escalation_tickets where message_id='{am}'")
res={}
assert lockg('qc3','Sprint7 cửa sổ Postgres chết (P4-01-35, P5-01-E2, P5-02-12); ≤ 1 phút')
try:
    subprocess.run(['docker','stop','ep-r-postgres-1'],capture_output=True); time.sleep(1)
    t0=time.time()
    res['esc']=esc('qc3s06',am); res['esc_key']=A('qc3s06','POST',f'/chat/messages/{am}/escalate',idem=str(uuid.uuid4()))
    res['att']=A('qc3t','PUT',G,{'records':rows,'complete':False})
    res['gen']=A('qc3t','POST',f'{Q}/sessions/generate',B,idem=str(uuid.uuid4()))
    res['el']=time.time()-t0
finally:
    subprocess.run(['docker','start','ep-r-postgres-1'],capture_output=True)
    for _ in range(90):
        if subprocess.run("curl -sk https://localhost:1143/api/v1/readyz -o /dev/null -w '%{http_code}'",shell=True,capture_output=True,text=True).stdout=='200': break
        time.sleep(2)
    unlockg('qc3','Postgres đã bật lại, readyz 200')
time.sleep(3)
after=psql(f"select count(*),sum(version),md5(string_agg(status::text||version,',' order by student_id)) from attendance_records where session_id='{S}'")
ns1=psql(f"select count(*) from class_sessions where course_id='{H}'"); n_t1=psql(f"select count(*) from escalation_tickets where message_id='{am}'"); ob=psql(f"select count(*) from outbox where payload->>'course_id'='{A_}' and topic='escalation.created' and created_at>now()-interval '2 minutes'")
print(json.dumps({k:(v if k in('el',) else (v[0],str(v[1])[:160])) for k,v in res.items()},ensure_ascii=False))
rt=lambda r: isinstance(r[1],dict) and (r[1].get('code'),json.dumps(r[1].get('details'))[:60])
ok35=all(res[k][0]==503 for k in('esc','esc_key')) and n_t0==n_t1=='0'
rec('TC-P4-01-35',ok35,f'Redis chết (cửa sổ trước): escalate → 201 vẫn tạo ticket (spec #16/AC12: Redis chết vẫn 201, chuông muộn qua đọc bù — đã kiểm 24); Postgres chết: escalate → {res["esc"][0]} {res["esc"][1].get("code")} retry_after={json.dumps(res["esc"][1])[:140]}; có Idempotency-Key → {res["esc_key"][0]}; ticket của tin trước/sau={n_t0}/{n_t1}')
rec('TC-P5-02-12',res['att'][0]==503 and before==after,f'Postgres chết giữa PUT lưới → {res["att"][0]} {res["att"][1].get("code") if isinstance(res["att"][1],dict) else ""}; attendance_records trước/sau bật lại: {before} → {after}')
rec('TC-P5-01-E2',res['gen'][0]==503 and ns0==ns1,f'Postgres chết khi generate → {res["gen"][0]} {res["gen"][1].get("code") if isinstance(res["gen"][1],dict) else ""}; số buổi {ns0}→{ns1} (không buổi nửa vời). Ghi chú E2 nêu Redis/DB: Redis chết → generate không dùng Redis (xem dưới)')
save('w_pg')
