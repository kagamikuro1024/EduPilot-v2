from p8lib import *
import datetime as dt,sys
C=json.load(open('/tmp/qc3e/p8c.json')); A1=C['A']; P=f'/courses/{A1}'; uid=psql("select id from users where email='sv.gioi@edupilot.local'")
if sys.argv[1]=='on':
    now=dt.datetime.now(dt.timezone.utc); iso=lambda x:x.strftime('%Y-%m-%dT%H:%M:%SZ')
    s,e=A('teacher','POST',P+'/exams',{'title':'QC3 khoá UI P8','opens_at':iso(now+dt.timedelta(days=3)),'closes_at':iso(now+dt.timedelta(days=3,hours=2)),'duration_minutes':30}); eid=e['id']
    psql(f"update exams set status='OPEN', opens_at=now()-interval '1 minute', closes_at=now()+interval '2 hours' where id='{eid}'")
    s,at=A('sv.gioi','POST',P+f'/exams/{eid}/attempts',{}); print(s); open('/tmp/qc3e/w44.json','w').write(json.dumps({'eid':eid,'aid':at['attempt']['id']}))
else:
    j=json.load(open('/tmp/qc3e/w44.json')); psql(f"delete from exam_events where attempt_id='{j['aid']}'"); psql(f"delete from exam_attempts where id='{j['aid']}'"); psql(f"update exams set status='CLOSED' where id='{j['eid']}'"); rds('del',f'ep:exam_lock:{uid}'); print(A('sv.gioi','GET','/me/exam-lock'))
