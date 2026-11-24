import sys; sys.path.insert(0,'/tmp/qc3oc')
from routes import *
LK='/tmp/qc-global.lock'
os.mkdir(LK); open(LK+'/owner','w').write('qc3 OpenCode Go grading worker '+datetime.datetime.now().isoformat())
say('LOCK qc3 (run4)',datetime.datetime.now().isoformat())
orig={r['task']:[c['model_id'] for c in r['chain']] for r in routes()['items'] if r['task']!='EMBEDDING'}
course=psql("select id from courses where class_code='QCOC1'"); Q=f'/courses/{course}'
def wait_done(aid,limit=240):
    t0=time.time()
    while time.time()-t0<limit:
        n=psql(f"select count(*) from grading_results where assignment_id='{aid}' and run_state in ('QUEUED','RUNNING')")
        tot=psql(f"select count(*) from grading_results where assignment_id='{aid}'")
        if n=='0' and tot!='0': return round(time.time()-t0,1)
        time.sleep(3)
    return None
try:
    fake=mid('QC fake','fake-chat'); oc=mid('OpenCode Go','deepseek-v4.1-flash')
    aid1=psql(f"select id from assignments where course_id='{course}' and submission_code='OC01'")
    say('wk StartedAt',started('ep-qc-worker-1'),'gw',started('ep-qc-gateway-1'))
    s,d,tp=put('GRADING',[fake]); say('PUT GRADING→QC fake',s); time.sleep(1.5)
    t0=time.time(); s,d=A('teacher','POST',f'{Q}/assignments/{aid1}/grade-all',{}); say('grade-all OC01 (route fake)',s,str(d)[:100]); dt=wait_done(aid1); say('  xong sau',dt,'s')
    time.sleep(2.5); say('AUDIT (worker, route fake):'); say(psql(f"select task,provider,model,status,coalesce(error_kind,'') from llm_audit where course_id='{course}' and task='GRADING' and created_at>=to_timestamp({t0}) order by created_at"))
    say('grading_results OC01:',psql(f"select run_state,count(*) from grading_results where assignment_id='{aid1}' group by 1"))
    s,d,tp=put('GRADING',[oc]); say('PUT GRADING→OpenCode Go',s); time.sleep(1.5)
    # OC02 + 2 bài nộp
    import copy,hashlib
    FX="/Users/kuro/Documents/TA_Agent_qc2/docs/specs/FEAT-assignments/fixtures"
    b=json.load(open(f"{FX}/create-bt03.json")); b.update({"submission_code":"OC02","title":"QCOC bài thử OpenCode Go 2","due_at":time.strftime("%Y-%m-%dT%H:%M:%SZ",time.gmtime(time.time()+3*86400))})
    s,a=A('teacher','POST',f"{Q}/assignments",b); aid2=a['id']; say('create OC02',s); say('publish',A('teacher','POST',f"{Q}/assignments/{aid2}/publish",{})[0])
    import urllib.request
    for u,txt in (('sv.gioi',open('/tmp/qc3oc/subm.py').read().split("'sv.gioi':")[1].split("',\n")[0].lstrip('"')[:0] or None),):
        pass
    TX={'sv.gioi':"# Phân tích vụ tấn công SolarWinds\n\nTác nhân: nhóm APT29 cấy mã độc SUNBURST vào bản cập nhật Orion. Bề mặt tấn công là chuỗi cung ứng phần mềm: mã độc chèn vào quy trình build, bản cập nhật có chữ ký hợp lệ. Hậu quả: hàng nghìn tổ chức bị cài cửa hậu, kẻ tấn công di chuyển ngang và đánh cắp thư điện tử. Biện pháp: kiểm tra toàn vẹn build, phân đoạn mạng, giám sát DNS bất thường, đặc quyền tối thiểu.\n",'sv.kha':"# Bài làm\n\nSolarWinds là một vụ tấn công mạng lớn. Kẻ xấu chèn mã độc vào phần mềm. Nên cập nhật phần mềm thường xuyên.\n"}
    for u,t in TX.items():
        data=t.encode(); sha=hashlib.sha256(data).hexdigest()
        s,pre=A(u,'POST',f"{Q}/assignments/{aid2}/submit/presign",{"filename":"bai-lam.md","size_bytes":len(data),"sha256":sha})
        req=urllib.request.Request(pre['url'],data=data,method=pre['method'],headers=pre['headers']); urllib.request.urlopen(req,context=ctx)
        say('submit',u,A(u,'POST',f"{Q}/assignments/{aid2}/submit",{"upload_id":pre['upload_id']})[0])
    t0=time.time(); s,d=A('teacher','POST',f'{Q}/assignments/{aid2}/grade-all',{}); say('grade-all OC02 (route OpenCode Go)',s,str(d)[:100]); dt=wait_done(aid2,300); say('  xong sau',dt,'s')
    time.sleep(2.5); say('AUDIT (worker, route OpenCode Go):'); say(psql(f"select task,provider,model,status,coalesce(error_kind,''),latency_ms,attempts from llm_audit where course_id='{course}' and task='GRADING' and created_at>=to_timestamp({t0}) order by created_at"))
    say('grading_results OC02:',psql(f"select run_state,count(*) from grading_results where assignment_id='{aid2}' group by 1"))
    say(psql(f"select s.id is not null, gr.run_state, left(gr.rubric_scores::text,220), gr.published_at is not null as published from grading_results gr join submissions s on s.id=gr.submission_id where gr.assignment_id='{aid2}' order by gr.created_at limit 2"))
    say('cột grading_results:',psql("select string_agg(column_name,',') from information_schema.columns where table_name='grading_results'"))
finally:
    for task,ids in orig.items():
        if [c['model_id'] for c in route_of(task)['chain']]!=ids:
            s,d,_=put(task,ids); say('RESTORE',task,s)
    cur={r['task']:[c['model_id'] for c in r['chain']] for r in routes()['items'] if r['task']!='EMBEDDING'}
    say('RESTORED đúng?',cur==orig)
    subprocess.run("docker exec ep-qc-redis-1 sh -c \"redis-cli --scan --pattern 'ep:llm:cb:*' | xargs -r redis-cli del\"",shell=True,capture_output=True)
    os.remove(LK+'/owner'); os.rmdir(LK); say('UNLOCK qc3 (run4)',datetime.datetime.now().isoformat())
