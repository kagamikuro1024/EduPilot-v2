import sys; sys.path.insert(0,'/tmp/qc3oc')
from routes import *
course=psql("select id from courses where class_code='QCOC1'"); uid=psql("select id from users where email='sv.gioi@edupilot.local'")
res=[]
for i in range(5):
    t0=time.time(); s,d=A('teacher','POST',f'/courses/{course}/students/{uid}/ai-summary',{},timeout=100); dt=round(time.time()-t0,1)
    res.append((s,dt)); say('TC51 ai-summary',i+1,s,dt,'s',(str(d)[:140] if s!=200 else 'keys='+','.join(list(d.keys())[:6])+' len='+str(len(json.dumps(d,ensure_ascii=False)))))
    if s==429: say('  429 giới hạn lượt/giờ:',str(d)[:150]); break
time.sleep(3); say(psql(f"select task,provider,model,status,coalesce(error_kind,''),latency_ms from llm_audit where course_id='{course}' and created_at>now()-interval '3 minutes' order by created_at"))
