from lib3 import *
A_=C7['A']; F_=C7['F']; Q=f'/courses/{A_}'
for u in ['qc3s01','qc3s02','qc3s03','qc3s04','qc3s05','qc3s06','qc3s07','qc3t','qc3t4','qc3ta','qc3s12','qc3s13']: login(u)
clearopen('qc3t',A_); clearopen('qc3t4',F_)
# chuẩn bị
msgs=[mkchat(u,A_)[2] for u in ('qc3s01','qc3s02')]
mF=mkchat('qc3s12',F_)[2]; mG=mkchat('qc3s13',F_)[2]
tconf,_=tick('qc3s03',A_); A('qc3t','POST',f'{Q}/tickets/{tconf}/claim'); A('qc3t','POST',f'{Q}/tickets/{tconf}/answer',{'body':'Chào em, đã trả lời cho em.'})
mail0=len([x for x in mpq('to:qc3t4@edupilot.local',50) if x['Subject']=='Có câu hỏi mới cần giảng viên'])
tu=uid('qc3t4'); rds('DEL',f'ep:mail:ticket_new:{tu}:{F_}')
ntf0=int(nrows('qc3t4'))
c=SSEConn('qc3t4'); time.sleep(1)
assert lockg('qc3','Sprint7 cửa sổ Redis chết (P4-01-35, P4-02-24, P4-04-17, P4-05-E1); ≤ 4 phút')
res={}
try:
    subprocess.run(['docker','stop','ep-r-redis-1'],capture_output=True); t0=time.time(); print('redis stopped')
    time.sleep(2)
    n0=nt(msgs[0]); r1=esc('qc3s01',msgs[0]); r1b=A('qc3s02','POST',f'/chat/messages/{msgs[1]}/escalate',idem=str(uuid.uuid4()))
    res['35']=(r1[0],r1[1].get('ticket',{}).get('status') if isinstance(r1[1],dict) else r1[1],r1b[0],nt(msgs[0]),nt(msgs[1]))
    # ticket cho GV có mail + chuông
    rF=esc('qc3s12',mF); rG=esc('qc3s13',mG); res['24']=(rF[0],rG[0])
    time.sleep(8)
    res['24n']=int(nrows('qc3t4'))-ntf0
    res['24notif_api']=A('qc3t4','GET','/notifications')[0]
    res['sse']=[(f[2],(f[3] or '')[:80]) for f in c.frames][-4:]
    res['mail_down']=len([x for x in mpq('to:qc3t4@edupilot.local',50) if x['Subject']=='Có câu hỏi mới cần giảng viên'])-mail0
    res['mail_row']=psql(f"select status||'/'||template from mail_outbox where to_addr='qc3t4@edupilot.local' and template='ticket_new' order by created_at desc limit 2").replace('\n',',')
    # confirm khi Redis chết
    rc=A('qc3s03','POST',f'/chat/tickets/{tconf}/confirm'); res['E1']=(rc[0],psql(f"select status||'/'||coalesce(closed_reason,'-') from escalation_tickets where id='{tconf}'"))
    print('elapsed',time.time()-t0)
finally:
    subprocess.run(['docker','start','ep-r-redis-1'],capture_output=True)
    for _ in range(60):
        if subprocess.run("curl -sk https://localhost:1143/api/v1/readyz -o /dev/null -w '%{http_code}'",shell=True,capture_output=True,text=True).stdout=='200': break
        time.sleep(2)
    unlockg('qc3','Redis đã bật lại, readyz 200')
print(res)
time.sleep(25)
res['mail_after']=len([x for x in mpq('to:qc3t4@edupilot.local',50) if x['Subject']=='Có câu hỏi mới cần giảng viên'])-mail0
res['sse_after']=[(f[2]) for f in c.frames][-5:]
print(res)
json.dump(res,open('/tmp/qc3e/w_redis.json','w'),ensure_ascii=False)
