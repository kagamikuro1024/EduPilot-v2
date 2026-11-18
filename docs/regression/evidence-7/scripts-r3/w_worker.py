from lib3 import *
A_=C7['A']; Q=f'/courses/{A_}'; T='qc3t'
def dk(*a): return subprocess.run(['docker',*a],capture_output=True,text=True)
def worker_up(name='ep-r-worker-1'):
    for _ in range(60):
        st=dk('inspect',name,'--format','{{.State.Status}} {{.State.Health.Status}}').stdout.strip()
        if st=='running healthy': return True
        if st.startswith('exited'): dk('start',name)
        time.sleep(2)
    return False
clearopen(T,A_)
ta,_=tick('qc3s24',A_,q='Quy định nộp bài muộn QCKB42 của lớp là gì?'); A(T,'POST',f'{Q}/tickets/{ta}/claim'); A(T,'POST',f'{Q}/tickets/{ta}/answer',{'body':'Chào em, nộp bài muộn bị trừ điểm theo quy chế của lớp QCKB42.'})
json.dump(dict(A=A_,tid=ta),open('/tmp/qc3e/fx_kb.json','w'))
sp=A(T,'GET',f'{Q}/tickets/{ta}/knowledge-preview'); pv=sp[1]; print('preview',sp[0],json.dumps(pv,ensure_ascii=False)[:200])
qn=pv.get('question'); an=pv.get('answer'); R={}
assert lockg('qc3','Sprint7 cửa sổ dừng worker (TC-P4-05-42 tri thức chưa nạp; TC-P4-05-20 hai worker); ≤ 6 phút')
try:
    dk('stop','-t','2','ep-r-worker-1'); time.sleep(2)
    sk,dk_=A(T,'POST',f'{Q}/tickets/{ta}/knowledge',{'question':qn,'answer':an}); R['save']=(sk,dk_)
    time.sleep(3)
    R['emb']=psql(f"select count(*)||'/'||count(c.embedding) from content_chunks c join knowledge_items k on k.chunk_id=c.id where k.source_id='{ta}'")
    sd,dd=A(T,'GET',f'{Q}/tickets/{ta}'); R['state']=dd.get('knowledge')
    r=subprocess.run("cd /Users/kuro/Documents/TA_Agent_qc2/frontend && NODE_PATH=$PWD/node_modules NODE_TLS_REJECT_UNAUTHORIZED=0 timeout 120 npx playwright test -c /tmp/qc3e/pw.config.ts kb-p405 -g 'kb42$' --timeout 100000 2>&1 | tail -3",shell=True,capture_output=True,text=True); R['pw']=r.stdout[-200:]
    R['outbox_pending']=psql(f"select count(*) from outbox where topic='knowledge.sync' and dispatched_at is null and dead_at is null")
finally:
    dk('start','ep-r-worker-1'); R['up']=worker_up()
    unlockg('qc3','worker chạy lại')
for _ in range(30):
    e=psql(f"select count(*)||'/'||count(c.embedding) from content_chunks c join knowledge_items k on k.chunk_id=c.id where k.source_id='{ta}'")
    if e.endswith('/1'): break
    time.sleep(2)
R['emb_after']=e; R['state_after']=A(T,'GET',f'{Q}/tickets/{ta}')[1].get('knowledge')
print(json.dumps(R,ensure_ascii=False)[:1500]); json.dump(R,open('/tmp/qc3e/w_worker.json','w'),ensure_ascii=False)
