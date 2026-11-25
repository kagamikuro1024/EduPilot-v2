import sys,json,datetime as dt; sys.path.insert(0,'/Users/kuro/Documents/TA_Agent_qc3/docs/regression/evidence-oc/scripts')
import oc
C=oc.psql("select id from courses where class_code='QCOC1'")
T='teacher'
def q(i):
    s,d=oc.A(T,'POST',f'/courses/{C}/questions',{'type':'MCQ_SINGLE','title':f'QC3LT câu {i}','topic':'QC3','difficulty':'EASY','stem':f'Chọn B ({i})','options':[{'body':'A'},{'body':'B'},{'body':'C'},{'body':'D'}],'correct':[1]})
    assert s==201,(s,d)
    s2,d2=oc.A(T,'PUT',f'/courses/{C}/questions/{d["id"]}/review',{'decision':'APPROVE','version':d['version']}); assert s2==200,(s2,d2)
    return d['id']
qs=[q(1),q(2)]
now=dt.datetime.now(dt.timezone.utc); f=lambda h:(now+dt.timedelta(hours=h)).strftime('%Y-%m-%dT%H:%M:%SZ')
out={}
for name,st in (('QC3 LT nháp','DRAFT'),('QC3 LT đang mở','OPEN'),('QC3 LT đã đóng','CLOSED')):
    s,d=oc.A(T,'POST',f'/courses/{C}/exams',{'title':name,'opens_at':f(2),'closes_at':f(4),'duration_minutes':45}); assert s==201,(s,d)
    e=d['id']
    s,d=oc.A(T,'PUT',f'/courses/{C}/exams/{e}/items',{'items':[{'question_id':x,'points':1} for x in qs],'version':d['version']}); assert s==200,(s,d)
    if st!='DRAFT':
        s,d=oc.A(T,'POST',f'/courses/{C}/exams/{e}/schedule'); assert s==200,(s,d)
        if st=='OPEN': oc.psql(f"update exams set status='OPEN', opens_at=now()-interval '10 minutes', closes_at=now()+interval '3 hours' where id='{e}'")
        else: oc.psql(f"update exams set status='CLOSED', opens_at=now()-interval '3 hours', closes_at=now()-interval '1 hour' where id='{e}'")
    out[st]=e; print(st,e,oc.psql(f"select status from exams where id='{e}'"))
json.dump({'course':C,**out},open('/tmp/qc3lt/exams.json','w'))
