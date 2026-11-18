from lib3 import *
A_=C7['A']; Q=f'/courses/{A_}'; T='qc3t'; E_=C7['E']; QE=f'/courses/{E_}'; TE='qc3t3'
def dk(*a): return subprocess.run(['docker',*a],capture_output=True,text=True)
def worker_up(name='ep-r-worker-1'):
    for _ in range(60):
        st=dk('inspect',name,'--format','{{.State.Status}} {{.State.Health.Status}}').stdout.strip()
        if st=='running healthy': return True
        if st.startswith('exited'): dk('start',name)
        time.sleep(2)
    return False
def aip(): 
    d=A(T,'GET',f'/courses/{A_}/today')[1]; return [a['title'] for a in d.get('actions',[]) if a['kind']=='AI_CONFIRM']
# fixture: 2 bài AI chờ duyệt
pids=[]
for i in (1,2):
    th=A('qc3s27','POST',f'{Q}/threads',{'title':f'Thảo luận QCV44-{i}','body':'Nội dung câu hỏi cho bài AI chờ duyệt.','tags':[]})[1]['thread']['id']
    psql(f"update forum_threads set ai_state='PENDING' where id='{th}'")
    pid=psql(f"insert into forum_posts(course_id,thread_id,kind,body,verification_state,confidence,ai_body) values('{A_}','{th}','AI','Câu trả lời AI QCV44-{i}','PENDING',0.5,'Câu trả lời AI QCV44-{i}') returning id").split('\n')[0]; psql(f"update forum_threads set ai_state='ANSWERED' where id='{th}'"); pids.append(pid)
time.sleep(1)
# seed 3 ticket quá hạn lớp E cho phần hai worker
clearopen(TE,E_); tE=[tick('qc3s%02d'%i,E_)[0] for i in (1,2,3)]
R={}
assert lockg('qc3','Sprint7 cửa sổ worker (TC-P4-06-44 VERIFY khi nhúng chết, TC-P4-05-20 hai worker); ≤ 8 phút')
try:
    R['ai_before']=aip()
    dk('stop','-t','2','ep-r-worker-1'); time.sleep(2)
    sv,dv=A(T,'POST',f'{Q}/posts/{pids[0]}/verify',{'version':1}); R['verify']=(sv,dv.get('verification_state') if isinstance(dv,dict) else dv)
    t0=time.time(); R['ai_after_s']=None
    for _ in range(70):
        a=aip()
        if a and '1 câu' in a[0]: R['ai_after_s']=round(time.time()-t0,1); break
        time.sleep(1)
    R['ai_after']=aip()
    R['kn_down']=psql(f"select count(*)||'/'||count(c.embedding) from content_chunks c join knowledge_items k on k.chunk_id=c.id where k.source_id='{pids[0]}'")
    # hai worker: khởi động bản thứ hai, rồi bản 1
    dk('start','ep-r-worker-1'); worker_up()
    dk('rm','-f','ep-r-worker-qc3b'); r=dk('run','-d','--name','ep-r-worker-qc3b','--network','ep-r_default','--env-file','/tmp/qc3e/wk.env','edupilot-worker-test'); R['w2']=r.stdout[:12]; worker_up('ep-r-worker-qc3b')
    for t in tE: seedage('QC3E',t,100)
    mails0=psql(f"select count(*) from mail_outbox where template='ticket_reminder' and payload->>'course_id'='{E_}'")
    t0=time.time()
    for _ in range(100):
        st=[psql(f"select sla_stage from escalation_tickets where id='{t}'") for t in tE]
        if all(x!='0' for x in st): break
        time.sleep(2)
    R['stage']=st; R['stage_s']=round(time.time()-t0,1)
    time.sleep(20)  # thêm một tick để thấy không nhắc đôi
    R['mails']=psql(f"select to_addr||'='||count(*) from mail_outbox where template='ticket_reminder' and payload->>'course_id'='{E_}' group by to_addr order by 1").replace('\n',', ')
    R['mails_total']=psql(f"select count(*) from mail_outbox where template='ticket_reminder' and payload->>'course_id'='{E_}'")
    R['stale_notif']=psql(f"select count(*) from notifications where type='TICKET_STALE' and course_id='{E_}'")
    R['w1_log']=dk('logs','ep-r-worker-1','--since','3m').stdout.count('ticket.sla'); R['w2_log']=dk('logs','ep-r-worker-qc3b','--since','3m').stdout.count('ticket.sla')
finally:
    dk('rm','-f','ep-r-worker-qc3b'); dk('start','ep-r-worker-1'); worker_up()
    unlockg('qc3','worker phụ đã xoá, worker-1 chạy')
for _ in range(30):
    e=psql(f"select count(*)||'/'||count(c.embedding) from content_chunks c join knowledge_items k on k.chunk_id=c.id where k.source_id='{pids[0]}'")
    if e.endswith('/1'): break
    time.sleep(2)
R['kn_after']=e
print(json.dumps(R,ensure_ascii=False)[:1800]); json.dump(R,open('/tmp/qc3e/w_worker2.json','w'),ensure_ascii=False)
