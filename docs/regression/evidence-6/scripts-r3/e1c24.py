from lib import *
import subprocess,os
REPO='/Users/kuro/Documents/TA_Agent_qc3p'
q="select (select count(*) from forum_threads where course_id='01a1df27-62b5-7775-a346-5c602a43e2f9'),(select count(*) from forum_posts p join forum_threads t on t.id=p.thread_id where t.course_id='01a1df27-62b5-7775-a346-5c602a43e2f9'),(select count(*) from pii_events where course_id='01a1df27-62b5-7775-a346-5c602a43e2f9' and created_at>now()-interval '1 day'*0)"
b=psql(q); t0=psql("select now()")
env={**os.environ,'API_URL':'https://localhost:1143/api/v1','SEED_DEFAULT_PASSWORD':'Edupilot#Seed-2026','PYTHONHTTPSVERIFY':'0'}
r=subprocess.run(['python3','benchmarks/eval_pii.py','--out-dir','/tmp/qc3e/e1/r24'],cwd=REPO,capture_output=True,text=True,env=env)
a=psql(q); pe=psql(f"select count(*) from pii_events where course_id='01a1df27-62b5-7775-a346-5c602a43e2f9' and created_at>'{t0}'")
rec('TC-P3-07-24',r.returncode==0 and a==b and pe=='0',f'chạy đầy đủ 200 mẫu (rc={r.returncode}, {r.stdout.strip().splitlines()[-1]}): (threads,posts,pii_events) lớp 761987 trước={b} sau={a}; pii_events tạo mới trong lúc chạy={pe}; script chỉ gọi POST /threads/precheck')
